package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// GetRegionalWallDebuff calculates the debuff percentage (negative float64) and duration
// based on player rank and sub-rank when crossing between different NGU_VUC regions.
// Higher rank and sub-rank result in harsher stat suppression ("rank càng cao giảm càng nhiều").
func GetRegionalWallDebuff(rank int, subRank int) (float64, time.Duration) {
	return gamefx.RegionalWallDebuff(rank, subRank)
}

// CheckAndApplyRegionalWallDebuff checks if a player moved from one NGU_VUC to a different NGU_VUC.
// If so, it applies the "regional_wall_suppression" debuff status to the player's Statuses list.
func CheckAndApplyRegionalWallDebuff(player *models.Player, fromX, fromY, toX, toY int, q ...notify.Querier) string {
	if player == nil || (fromX == 0 && fromY == 0) {
		return ""
	}
	return applyRegionalWallDebuff(player, FindLargeZoneAt(fromX, fromY), FindLargeZoneAt(toX, toY), q...)
}

// CheckAndApplyRegionalWallDebuffBetween resolves the containing large zones
// from region ancestry. Use it when callers know both region IDs: stacked maps
// can reuse coordinates, so coordinate-only lookup may miss or misidentify a
// transition between world-map layers.
func CheckAndApplyRegionalWallDebuffBetween(player *models.Player, fromRegion, toRegion string, q ...notify.Querier) string {
	if player == nil || fromRegion == "" || toRegion == "" || fromRegion == toRegion {
		return ""
	}
	return applyRegionalWallDebuff(player, largeZoneForRegion(fromRegion), largeZoneForRegion(toRegion), q...)
}

func largeZoneForRegion(regionID string) *models.Region {
	if regionID == "" {
		return nil
	}
	ensureZonesLoaded()
	seen := make(map[string]struct{}, 8)
	for curr := regionID; curr != ""; {
		if _, dup := seen[curr]; dup {
			break
		}
		seen[curr] = struct{}{}
		zone := ZoneByID(curr)
		if zone == nil {
			break
		}
		if zone.ZoneType == ZoneTypeLarge {
			return zone
		}
		curr = zone.ParentID
	}
	zone := ZoneByID(regionID)
	if zone == nil {
		return nil
	}
	if zone.ZoneType == ZoneTypeLarge {
		return zone
	}
	if strings.EqualFold(zone.ZoneType, ZoneTypeSecret) || strings.EqualFold(zone.ZoneType, ZoneTypeDungeon) {
		return nil
	}
	refX, refY := zone.AtlasX+zone.EntryX, zone.AtlasY+zone.EntryY
	if refX == 0 && refY == 0 {
		refX = zone.AtlasX + (zone.MinX+zone.MaxX)/2
		refY = zone.AtlasY + (zone.MinY+zone.MaxY)/2
	}
	if refX != 0 || refY != 0 {
		zoneReg.mu.RLock()
		defer zoneReg.mu.RUnlock()
		for i := range zoneReg.zones {
			z := &zoneReg.zones[i]
			if z.ZoneType == ZoneTypeLarge && zoneContains(z, refX, refY) {
				return z
			}
		}
	}
	return nil
}

// RegionalWallSuppressionEnabled reports whether crossing a Great Domain (NGU_VUC)
// boundary applies the regional_wall_suppression penalty. Defaults to enabled;
// set REGIONAL_WALL_SUPPRESSION_ENABLED to false/0/off/no to disable it.
func RegionalWallSuppressionEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("REGIONAL_WALL_SUPPRESSION_ENABLED"))) {
	case "false", "0", "off", "no":
		return false
	}
	return true
}

func applyRegionalWallDebuff(player *models.Player, oldZone, newZone *models.Region, q ...notify.Querier) string {
	if !RegionalWallSuppressionEnabled() {
		return ""
	}
	if oldZone == nil || newZone == nil {
		return ""
	}
	if oldZone.ID == newZone.ID {
		return ""
	}
	if oldZone.ZoneType != ZoneTypeLarge || newZone.ZoneType != ZoneTypeLarge {
		return ""
	}

	debuffPct, duration := GetRegionalWallDebuff(player.Rank, player.SubRank)
	sourceLabel := fmt.Sprintf("Giới Vực Bích (%s ➔ %s)", oldZone.Name, newZone.Name)
	debuffInt := int(math.Abs(debuffPct * 100))
	durMinutes := int(duration.Minutes())

	type rowQuerier interface {
		notify.Querier
		QueryRow(string, ...interface{}) *sql.Row
	}
	var rq rowQuerier
	var querier notify.Querier
	if len(q) > 0 && q[0] != nil {
		querier = q[0]
		if r, ok := q[0].(rowQuerier); ok {
			rq = r
		}
	} else if db.DB != nil {
		querier = db.DB
		rq = db.DB
	}

	// Check if player is piloting an active equipped Immortal Gu House
	var houseID, houseName string
	var currentDurability float64
	var auxGuCount int
	if rq != nil && player.ID != uuid.Nil {
		_ = rq.QueryRow(`
			SELECT id::text, name, current_durability, aux_gu_count 
			FROM player_gu_houses 
			WHERE player_id = $1 AND is_equipped = true 
			LIMIT 1
		`, player.ID).Scan(&houseID, &houseName, &currentDurability, &auxGuCount)
	}

	if houseID != "" && currentDurability > 0 {
		// Tiên Cổ Ốc chịu áp lực khổng lồ thay cho chủ nhân!
		durLoss := 25.0
		newDur := math.Max(0, currentDurability-durLoss)
		guLoss := int(float64(auxGuCount) * 0.15)
		newAux := int(math.Max(0, float64(auxGuCount-guLoss)))

		if newDur <= 0 {
			// Cổ Ốc vỡ vụn ngay trong Giới Bích!
			_, _ = querier.Exec(`UPDATE player_gu_houses SET current_durability = 0, aux_gu_count = $1, is_equipped = false, updated_at = NOW() WHERE id = $2`, newAux, houseID)
			msg := fmt.Sprintf("💥 NGUY CẤP! Tiên Cổ Ốc [%s] không chịu nổi áp lực cuồng bạo của Giới Vực Bích (%s ➔ %s) và đã nổ tung! Đạo hữu văng ra ngoài và bị [Áp Chế Giới Vực] (Công/Thủ/Tốc giảm %d%% trong %d phút).",
				houseName, oldZone.Name, newZone.Name, debuffInt, durMinutes)
			player.Statuses.Add("regional_wall_suppression", sourceLabel, duration, debuffPct)
			if player.ID != uuid.Nil {
				_ = notify.Send(querier, player.ID, "regional_wall_suppression", "Tiên Cổ Ốc Sụp Đổ", msg)
			}
			return msg
		}

		// Cổ Ốc che chở cho nhân vật, nhưng bản thân bị bào mòn độ bền và chấn tử phàm cổ
		_, _ = querier.Exec(`UPDATE player_gu_houses SET current_durability = $1, aux_gu_count = $2, updated_at = NOW() WHERE id = $3`, newDur, newAux, houseID)
		msg := fmt.Sprintf("🛡️ Tiên Cổ Ốc [%s] gượng ép xé rách Giới Vực Bích (%s ➔ %s)! Áp lực thiên địa đạo ngân xung đột làm tổn hại %.0f%% độ bền (còn %.0f%%) và chấn tử %d phàm cổ phụ trợ!",
			houseName, oldZone.Name, newZone.Name, durLoss, newDur, guLoss)
		if player.ID != uuid.Nil {
			_ = notify.Send(querier, player.ID, "regional_wall_suppression", "Tiên Cổ Ốc Chấn Động", msg)
		}
		return msg
	}

	player.Statuses.Add("regional_wall_suppression", sourceLabel, duration, debuffPct)

	var msg string
	if durMinutes >= 60 {
		msg = fmt.Sprintf("⚠️ Đạo hữu vừa vượt qua Giới Vực Bích từ %s sang %s! Quy tắc địa giới phản phệ gây ra [Áp Chế Giới Vực] (Công/Thủ/Tốc giảm %d%% trong %d giờ).",
			oldZone.Name, newZone.Name, debuffInt, durMinutes/60)
	} else {
		msg = fmt.Sprintf("⚠️ Đạo hữu vừa vượt qua Giới Vực Bích từ %s sang %s! Quy tắc địa giới phản phệ gây ra [Áp Chế Giới Vực] (Công/Thủ/Tốc giảm %d%% trong %d phút).",
			oldZone.Name, newZone.Name, debuffInt, durMinutes)
	}

	logger.S.Warnf("[Áp Chế Giới Vực] Người chơi %s (%s) bị áp chế khi vượt Giới Vực Bích từ %s sang %s (giảm %d%%, thời gian %v)",
		player.Username, player.ID, oldZone.Name, newZone.Name, debuffInt, duration)

	if querier != nil && player.ID != uuid.Nil {
		_ = notify.Send(querier, player.ID, "regional_wall_suppression", "Áp Chế Giới Vực", msg)
	}

	return msg
}

func resolveCellArrival(tx *sql.Tx, player *models.Player, x, y int, attackTargetID string, allowAmbush bool) (returnedMessage string, startedCombat bool, returnedErr error) {
	return resolveCellArrivalWithInstance(tx, player, x, y, attackTargetID, allowAmbush, nil)
}

// resolveCellArrivalWithInstance lets a multi-cell move reuse the locked
// player's map-instance value instead of querying the same row on every step.
func resolveCellArrivalWithInstance(tx *sql.Tx, player *models.Player, x, y int, attackTargetID string, allowAmbush bool, loadedInstanceID *sql.NullString) (returnedMessage string, startedCombat bool, returnedErr error) {
	regionWarning := ""
	defer func() {
		if regionWarning != "" {
			returnedMessage += " " + regionWarning
		}
	}()
	oldX := player.X
	oldY := player.Y
	oldRegion := player.CurrentRegion
	mapID := worldMapIDForRegion(player.CurrentRegion)
	ignoreMonsters := PlayerIgnoresMonsters(player)
	var instanceID sql.NullString
	if loadedInstanceID != nil {
		instanceID = *loadedInstanceID
	} else if err := tx.QueryRow(`SELECT map_instance_id::text FROM players WHERE id=$1`, player.ID).Scan(&instanceID); err != nil {
		return "", false, err
	}
	if instanceID.Valid {
		region := ZoneByID(player.CurrentRegion)
		if region == nil || region.ZoneType != ZoneTypeSecret || !zoneContains(region, x, y) {
			return "Ranh giới bí cảnh không cho phép di chuyển tới tọa độ này.", false, nil
		}
	}

	destinationRegion := player.CurrentRegion
	if !instanceID.Valid {
		currentZone := ZoneByID(player.CurrentRegion)
		if currentZone != nil && zoneContains(currentZone, x, y) {
			destinationRegion = player.CurrentRegion
		} else {
			derived := DeriveRegionAtOnMap(x, y, mapID)
			if derived != "" && derived != HubRegionID {
				destinationRegion = derived
			}
		}
	}
	if destinationRegion != player.CurrentRegion {
		if err := checkRegionEntry(destinationRegion, player.Rank, player.GuInventory); err != nil {
			return err.Error(), false, nil
		}
	}
	player.X = x
	player.Y = y
	player.CurrentRegion = destinationRegion
	visitedSubZone := SubZoneKey(player.X, player.Y)
	recordVisitedSubZone(player, visitedSubZone)
	if err := TrackDailyMissionProgress(tx, player.ID, player.Rank, "explore:"+player.CurrentRegion+":"+visitedSubZone, "explore", "", 1); err != nil {
		return "", false, err
	}

	coordKey := fmt.Sprintf("%d,%d", player.X, player.Y)
	isCleared := isCellClearedRecently(player.CurrentRegion, player.X, player.Y, clearedCellCooldown)
	cell := ResolveCellInRegion(player.CurrentRegion, player.X, player.Y)
	if attackTargetID != "" {
		if err := checkRegionAction(player.CurrentRegion, "pvp"); err != nil {
			return err.Error(), false, nil
		}
		if err := checkRegionAction(player.CurrentRegion, "combat"); err != nil {
			return err.Error(), false, nil
		}
		if err := checkRegionEquippedGus(player.CurrentRegion, player.GuInventory); err != nil {
			return err.Error(), false, nil
		}
	}
	logMsg := fmt.Sprintf("Di chuyển thành công đến (%d, %d).", player.X, player.Y)
	if oldRegion != "" && destinationRegion != "" {
		regionWarning = CheckAndApplyRegionalWallDebuffBetween(player, oldRegion, destinationRegion, tx)
	} else if oldRegion == "" && (oldX != 0 || oldY != 0) {
		regionWarning = CheckAndApplyRegionalWallDebuff(player, oldX, oldY, player.X, player.Y, tx)
	}
	combatTriggered := false
	pvpTriggered := false

	playerTerritory := CheckPlayerSectTerritory(tx, player.ID, player.CurrentRegion, player.X, player.Y)

	// Player-initiated PVP: attack target player within attack range (default <= 2 tiles, plus stats bonus)
	if attackTargetID != "" {
		targetUUID, errUUID := uuid.Parse(attackTargetID)
		if errUUID == nil {
			var targetPlayer models.Player
			queryStr := `
				SELECT p.id
				FROM players p
				WHERE p.id = $1 AND p.id != $2
				  AND p.active_combat IS NULL
				  AND (p.party_id IS NULL OR p.party_id IS DISTINCT FROM (SELECT party_id FROM players WHERE id=$2))
				  AND (p.last_activity >= NOW() - INTERVAL '5 minutes' OR p.last_mana_update >= NOW() - INTERVAL '5 minutes')
				  AND p.map_instance_id IS NOT DISTINCT FROM (SELECT map_instance_id FROM players WHERE id=$2)
				  AND COALESCE((SELECT world_map_id FROM region_world_maps WHERE region_id=p.current_region),'world_map') =
				      COALESCE((SELECT world_map_id FROM region_world_maps WHERE region_id=(SELECT current_region FROM players WHERE id=$2)),'world_map')
			`
			var errTarget error
			var matchedID uuid.UUID
			queryStr += ` FOR UPDATE OF p`
			errTarget = tx.QueryRow(queryStr, targetUUID, player.ID).Scan(&matchedID)

			if errTarget != nil {
				if errTarget != sql.ErrNoRows {
					return "", false, errTarget
				}
				var exists bool
				_ = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM players WHERE id = $1)`, targetUUID).Scan(&exists)
				if exists {
					return "Cổ Sư này đang bận, trong tổ đội của đạo hữu hoặc hiện quy ẩn; không thể khai chiến!", false, nil
				}
			}

			if errTarget == nil {
				var errLoad error
				targetPlayer, errLoad = evaluateManaTxEquippedReadonly(tx, targetUUID)
				if errLoad != nil {
					return "Không thể tải thông tin đối thủ!", false, nil
				}
				if err := checkRegionAction(targetPlayer.CurrentRegion, "pvp"); err != nil {
					return err.Error(), false, nil
				}
				if instanceID.Valid && instanceID.String != "" {
					if instUUID, errUUID := uuid.Parse(instanceID.String); errUUID == nil {
						if instRules, errRules := GetSpatialRealmRules(instUUID); errRules == nil && instRules != nil {
							if instRules.PvPMode == "SAFE" {
								return "Khu vực bí cảnh này cấm giao tranh giữa các Cổ sư (SAFE mode)!", false, nil
							}
						}
					}
				}
				if err := checkRegionAction(targetPlayer.CurrentRegion, "combat"); err != nil {
					return err.Error(), false, nil
				}
				if err := checkRegionEquippedGus(targetPlayer.CurrentRegion, targetPlayer.GuInventory); err != nil {
					return err.Error(), false, nil
				}
				// Target PvP immunity check (after death/respawn)
				if targetPlayer.Statuses.Has("pvp_immunity") {
					return fmt.Sprintf("Cổ Sư %s đang trong thời gian Miễn Chiến sau khi hồi sinh, không thể tấn công!", targetPlayer.Username), false, nil
				}

				// Distance check: Chebyshev distance max(|x1-x2|, |y1-y2|)
				dx := math.Abs(float64(targetPlayer.X - player.X))
				dy := math.Abs(float64(targetPlayer.Y - player.Y))
				dist := int(math.Max(dx, dy))

				// Range requirement: base 2 tiles + bonus stats (attack_range / combat_range)
				stats := calculatePlayerStatsWithTerritory(player, playerTerritory)
				bonusRange := int(stats["attack_range"]) + int(stats["combat_range"])
				maxPvPRange := 2 + bonusRange

				// HUB and SAFE_ZONE protect both sides, regardless of Sát Khí.
				if IsSafeZone(player.CurrentRegion, player.X, player.Y) || IsSafeZone(targetPlayer.CurrentRegion, targetPlayer.X, targetPlayer.Y) {
					return "Không thể tấn công trong Vùng An Toàn!", false, nil
				}

				targetTerritory := CheckPlayerSectTerritory(tx, targetPlayer.ID, targetPlayer.CurrentRegion, targetPlayer.X, targetPlayer.Y)
				if targetTerritory.InTerritory && !targetTerritory.IsShielded {
					return fmt.Sprintf("Cổ Sư %s đang ở trong phạm vi lãnh địa Tông Môn %s, được trận pháp Hộ Sơn che chở không thể tấn công!", targetPlayer.Username, targetTerritory.SectName), false, nil
				}
				if playerTerritory.SectID != "" && playerTerritory.SectID == targetTerritory.SectID {
					return "Đồng môn tương tàn là điều đại kỵ trong Tông Môn!", false, nil
				}

				if dist <= maxPvPRange {
					// Immunity is lost only when the attack actually starts.
					if player.Statuses.Has("pvp_immunity") {
						RemovePvpImmunity(tx, player.ID)
						player.Statuses.Remove("pvp_immunity")
						logMsg += " [Chủ động khai chiến, trạng thái Miễn Chiến bị phá vỡ!]"
					}
					combatTriggered = true
					pvpTriggered = true
					targetStats := calculatePlayerStatsWithTerritory(&targetPlayer, targetTerritory)

					enemyHP := targetPlayer.HP
					if enemyHP <= 0 {
						enemyHP = targetPlayer.BaseHP
					}
					if enemyHP <= 0 {
						enemyHP = 100.0
					}
					enemyMaxHP := gamefx.EffectiveMaxHP(&targetPlayer)
					if enemyMaxHP <= 0 {
						enemyMaxHP = 100.0
					}
					combatStartedAt := time.Now().UTC()

					pvpCombat := models.CombatState{
						CombatOwnerID:       player.ID.String(),
						CombatStartedAt:     combatStartedAt,
						EnemyTemplateID:     targetPlayer.ID.String(),
						EnemyName:           targetPlayer.Username,
						EnemyAvatarURL:      targetPlayer.AvatarURL,
						EnemyType:           "PVP",
						CombatMode:          "real",
						EnemyHP:             enemyHP,
						EnemyMaxHP:          enemyMaxHP,
						EnemyMana:           targetPlayer.CurrentMana,
						EnemyMaxMana:        targetPlayer.MaxMana,
						EnemyDamage:         targetStats["attack"],
						EnemyDefense:        targetStats["defense"],
						EnemySpeed:          targetStats["speed"],
						EnemyRank:           targetPlayer.Rank,
						EnemySubRank:        targetPlayer.SubRank,
						EnemyGuInventory:    targetPlayer.GuInventory,
						EnemyDaoMarks:       targetPlayer.DaoMarks,
						EnemyPathMasteries:  targetPlayer.PathMasteries,
						EnemyCombatSettings: &targetPlayer.CombatSettings,
						TargetPlayerID:      targetPlayer.ID.String(),
					}
					gamefx.ApplyEnemyEffects(&pvpCombat, &targetPlayer)
					player.ActiveCombat = &pvpCombat
					attackerStats := CalculatePlayerStats(player)
					defenderCombat := models.CombatState{
						CombatOwnerID: player.ID.String(), CombatStartedAt: combatStartedAt,
						EnemyTemplateID: player.ID.String(), EnemyName: player.Username,
						EnemyAvatarURL: player.AvatarURL, EnemyType: "PVP", CombatMode: "real",
						EnemyHP: player.HP, EnemyMaxHP: gamefx.EffectiveMaxHP(player), EnemyMana: player.CurrentMana,
						EnemyMaxMana: player.MaxMana, EnemyDamage: attackerStats["attack"], EnemyDefense: attackerStats["defense"],
						EnemySpeed: attackerStats["speed"], EnemyRank: player.Rank, EnemySubRank: player.SubRank,
						EnemyGuInventory: player.GuInventory, EnemyDaoMarks: player.DaoMarks,
						EnemyPathMasteries: player.PathMasteries, EnemyCombatSettings: &player.CombatSettings,
						TargetPlayerID: player.ID.String(),
					}
					gamefx.ApplyEnemyEffects(&defenderCombat, player)
					defenderJSON, marshalErr := json.Marshal(&defenderCombat)
					if marshalErr != nil {
						return "", false, marshalErr
					}
					updated, updateErr := tx.Exec(`UPDATE players SET active_combat=$1, version=version+1, state_revision=state_revision+1 WHERE id=$2 AND active_combat IS NULL`, string(defenderJSON), targetPlayer.ID)
					if updateErr != nil {
						return "", false, updateErr
					}
					if rows, _ := updated.RowsAffected(); rows != 1 {
						player.ActiveCombat = nil
						return "Cổ Sư này vừa vào một trận chiến khác.", false, nil
					}
					logMsg = fmt.Sprintf("Khai chiến! Đạo hữu tấn công Cổ Sư %s tại khoảng cách %d ô!", targetPlayer.Username, dist)
				} else {
					return fmt.Sprintf("Khoảng cách quá xa (%d ô)! Cần tiếp cận đối phương trong vòng %d ô để khai chiến.", dist, maxPvPRange), false, nil
				}
			}
		}
	}

	if isCleared {
		logMsg = fmt.Sprintf("Di chuyển thành công đến (%d, %d). Vùng đất trống trải hoang tàn.", player.X, player.Y)
	} else if !pvpTriggered {
		switch cell.Type {
		case "Resource":
			if err := checkRegionAction(player.CurrentRegion, "resource_gathering"); err != nil {
				logMsg = err.Error()
				break
			}
			itemName := cell.Content
			if item, ok := models.ItemRegistry[cell.Content]; ok {
				itemName = item.Name
			}
			if HasExplorationInventorySpace(player, 1, playerTerritory) {
				if err := InventorySvc.AddItem(tx, player.ID.String(), cell.Content, "materials", 1); err != nil {
					logger.S.Errorf("Failed to add resource %s to player %s: %v", cell.Content, player.ID, err)
					// Propagate: a failed write poisons the tx (25P02 on next statement).
					return "", false, err
				}
				if err := TrackDailyMissionProgress(tx, player.ID, player.Rank, fmt.Sprintf("gather:%s:%d:%d", player.CurrentRegion, player.X, player.Y), "collect", "", 1); err != nil {
					return "", false, err
				}
				if player.MaterialsInventory == nil {
					player.MaterialsInventory = make(models.MaterialsInventory)
				}
				player.MaterialsInventory[cell.Content]++
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Đạo hữu tìm thấy nguyên liệu hoang dã: 1x %s!", player.X, player.Y, itemName)

				clearTTL := clearedCellCooldown
				if pc, found := persistentCellForRegion(player.CurrentRegion, player.X, player.Y); found && pc.RespawnSeconds > 0 {
					// Shared resource nodes carry their own respawn window.
					clearTTL = time.Duration(pc.RespawnSeconds) * time.Second
				}
				if err := cache.SetCellCleared(player.CurrentRegion, player.X, player.Y, player.ID.String(), clearTTL); err != nil {
					logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, player.X, player.Y, err)
					return "", false, err
				}
				// Persist the shared respawn state for DB-backed nodes (no-op for
				// procedural cells). Without this map_cells.last_cleared_at never
				// advances, so the node ignores respawn_seconds and the atlas view
				// keeps reporting it as available.
				if err := MarkPersistentCellCleared(tx, player.ID, player.CurrentRegion, player.X, player.Y); err != nil {
					logger.S.Errorf("Failed to persist cleared resource cell %s (%d,%d) for player %s: %v", player.CurrentRegion, player.X, player.Y, player.ID, err)
					return "", false, err
				}
				broadcastWorldCell(cellEventCleared, WorldCellEvent{RegionID: player.CurrentRegion, X: player.X, Y: player.Y, CellType: cell.Type, ContentID: cell.Content})

				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			} else {
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Phát hiện 1x %s, nhưng Bách Bảo Nang đã đầy!", player.X, player.Y, itemName)
			}
		case "WildGu":
			if err := checkRegionAction(player.CurrentRegion, "gu_capture"); err != nil {
				logMsg = err.Error()
				break
			}
			var guSpec models.GuDictionaryItem
			var ok bool
			if cell.Rank > 0 {
				guSpec, ok = models.GetGu(cell.Content, cell.Rank)
			}
			if !ok {
				guSpec, _ = models.GetGuAny(cell.Content)
			}
			guName := cell.Content
			if guSpec.Name != "" {
				guName = guSpec.Name
			}
			if HasExplorationInventorySpace(player, 1, playerTerritory) {
				newGu := models.GuInstance{
					InstanceID: uuid.New().String(),
					GuID:       cell.Content,
					Rank:       guSpec.Rank,
					Hunger:     100,
					Status:     "ACTIVE",
					Equipped:   false,
				}
				player.GuInventory = append(player.GuInventory, newGu)
				logMsg = fmt.Sprintf("Cơ duyên! Đạo hữu bắt gặp và thu phục thành công Cổ trùng hoang dã: %s!", guName)

				clearTTL := clearedCellCooldown
				if pc, found := persistentCellForRegion(player.CurrentRegion, player.X, player.Y); found && pc.RespawnSeconds > 0 {
					clearTTL = time.Duration(pc.RespawnSeconds) * time.Second
				}
				if err := cache.SetCellCleared(player.CurrentRegion, player.X, player.Y, player.ID.String(), clearTTL); err != nil {
					logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, player.X, player.Y, err)
					return "", false, err
				}
				if err := MarkPersistentCellCleared(tx, player.ID, player.CurrentRegion, player.X, player.Y); err != nil {
					logger.S.Errorf("Failed to persist cleared Gu cell %s (%d,%d) for player %s: %v", player.CurrentRegion, player.X, player.Y, player.ID, err)
					return "", false, err
				}
				cache.BroadcastWildGuCaptured(player.X, player.Y, cell.Content)

				if player.ClearedCells == nil {
					player.ClearedCells = make(models.ClearedCellsMap)
				}
				player.ClearedCells[coordKey] = time.Now().Format(time.RFC3339)
			} else {
				logMsg = fmt.Sprintf("Đạo hữu phát hiện thấy Cổ trùng hoang dã %s, tiếc là Bách Bảo Nang đã đầy nên để nó trốn thoát!", guName)
			}
		case "Monster", "Boss":
			if ignoreMonsters {
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Đạo hữu phi thân trên không, lướt qua quái thú mà không bị chạm trán.", player.X, player.Y)
				break
			}
			if err := checkRegionAction(player.CurrentRegion, "combat"); err != nil {
				return err.Error(), false, nil
			}
			if err := checkRegionAction(player.CurrentRegion, "monster_combat"); err != nil {
				return err.Error(), false, nil
			}
			if err := checkRegionEquippedGus(player.CurrentRegion, player.GuInventory); err != nil {
				return err.Error(), false, nil
			}
			combatTriggered = true

			mon, ok := models.MonsterRegistry[cell.Content]
			if !ok {
				// Fall back to the DB (registry may not have loaded this id).
				var err error
				mon, err = db.LoadMonsterTemplate(tx, cell.Content)
				if err != nil {
					return "", false, err
				}
			}

			rank := 1
			if cell.Rank > 0 {
				rank = cell.Rank
			}
			subRank := 1
			if cell.SubRank > 0 {
				subRank = cell.SubRank
			}

			combatState := models.BuildCombatStateFromMonsterWithRegion(mon, rank, subRank, regionMonsterGuPool(player.CurrentRegion))
			combatState.WorldMonsterRegionID = player.CurrentRegion
			player.ActiveCombat = &combatState
			logMsg = fmt.Sprintf("Nguy hiểm! Đạo hữu chạm trán %s tại tọa độ (%d, %d)!", mon.Name, player.X, player.Y)

			hasHeavenSecret := models.InventoryHasGuCapability(player.GuInventory, "LOOT_PREVIEW", true)
			if hasHeavenSecret && mon.LootTableID != "" {
				logMsg += heavenSecretLootPreview(tx, mon.LootTableID, rank, player.CurrentRegion, mon.Type)
			}
		case "PVP":
			if IsSafeZone(player.CurrentRegion, player.X, player.Y) || playerTerritory.InTerritory {
				if playerTerritory.InTerritory {
					logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Lãnh địa Tông Môn %s thanh tịnh, không ai dám quấy nhiễu.", player.X, player.Y, playerTerritory.SectName)
				} else {
					logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Vùng An Toàn yên bình, phong cảnh thanh bình.", player.X, player.Y)
				}
				break
			}
			var targetUUID uuid.UUID
			err := tx.QueryRow(`
				SELECT p.id FROM players p
				WHERE p.id != $1
				ORDER BY p.last_login_at DESC NULLS LAST
				LIMIT 1 OFFSET floor(random() * 50)::int`, player.ID,
			).Scan(&targetUUID)

			var targetPlayer models.Player
			var errLoad error
			if err == nil {
				// Lock-free snapshot: PvP preview must not FOR UPDATE a random
				// victim row inside the mover Tx (cross-player deadlock source).
				targetPlayer, errLoad = PlayerSvc.ProjectStatusEquipped(tx, targetUUID)
			}

			combatTriggered = true
			if err != nil || errLoad != nil {
				player.ActiveCombat = &models.CombatState{
					EnemyName:      "Cổ Sư Lang Thang (Wandering Cultivator)",
					EnemyHP:        100.0,
					EnemyMaxHP:     100.0,
					EnemyDamage:    15.0,
					EnemyDefense:   5.0,
					TargetPlayerID: "dummy",
				}
				logMsg = "Đột ngột một Cổ Sư Lang Thang xuất hiện khiêu chiến đạo hữu!"
			} else {
				targetStats := CalculatePlayerStats(&targetPlayer)
				player.ActiveCombat = &models.CombatState{
					EnemyName:           targetPlayer.Username,
					EnemyType:           "PVP",
					CombatMode:          "real",
					EnemyHP:             100.0,
					EnemyMaxHP:          100.0,
					EnemyDamage:         targetStats["attack"],
					EnemyDefense:        targetStats["defense"],
					EnemySpeed:          targetStats["speed"],
					EnemyRank:           targetPlayer.Rank,
					EnemySubRank:        targetPlayer.SubRank,
					EnemyGuInventory:    targetPlayer.GuInventory,
					EnemyDaoMarks:       targetPlayer.DaoMarks,
					EnemyPathMasteries:  targetPlayer.PathMasteries,
					EnemyCombatSettings: &targetPlayer.CombatSettings,
					TargetPlayerID:      targetPlayer.ID.String(),
				}
				gamefx.ApplyEnemyEffects(player.ActiveCombat, &targetPlayer)
				logMsg = fmt.Sprintf("Đạo hữu bắt gặp một Cổ Sư khác đang du ngoạn: %s! Lập tức giao đấu!", targetPlayer.Username)
			}
		}
	}

	if !combatTriggered && allowAmbush && !playerTerritory.InTerritory && !ignoreMonsters {
		for ax := -1; ax <= 1 && !combatTriggered; ax++ {
			for ay := -1; ay <= 1 && !combatTriggered; ay++ {
				if ax == 0 && ay == 0 {
					continue
				}
				adjX := player.X + ax
				adjY := player.Y + ay

				// check if already cleared
				if isCellClearedRecently(player.CurrentRegion, adjX, adjY, clearedCellCooldown) {
					continue
				}

				adj := ResolveCellInRegion(player.CurrentRegion, adjX, adjY)

				rank := 1
				if adj.Rank > 0 {
					rank = adj.Rank
				}
				ambushChance := 50 + (rank * 2) // Base 50% + 2% per rank (min 50%)
				// DISGUISE lowers the chance of being ambushed by adjacent monsters.
				ambushChance -= int(PlayerDisguiseLevel(player))
				if ambushChance < 5 {
					ambushChance = 5
				}
				if ambushChance > 95 {
					ambushChance = 95
				}

				combatDenied := checkRegionAction(player.CurrentRegion, "combat")
				monsterCombatDenied := checkRegionAction(player.CurrentRegion, "monster_combat")
				if (adj.Type == "Monster" || adj.Type == "Boss") && combatDenied == nil && monsterCombatDenied == nil && rand.Intn(100) < ambushChance {
					combatTriggered = true

					mon, ok := models.MonsterRegistry[adj.Content]
					if !ok {
						var errLoadMon error
						mon, errLoadMon = db.LoadMonsterTemplate(tx, adj.Content)
						if errLoadMon != nil {
							return "", false, errLoadMon
						}
					}
					if mon.ID != "" {
						subRank := 1
						if adj.SubRank > 0 {
							subRank = adj.SubRank
						}
						combatState := models.BuildCombatStateFromMonsterWithRegion(mon, rank, subRank, regionMonsterGuPool(player.CurrentRegion))
						combatState.WorldMonsterRegionID = player.CurrentRegion
						player.ActiveCombat = &combatState
						logMsg = fmt.Sprintf("Phục kích! Quái thú %s lân cận (%d, %d) chủ động lao ra tấn công đạo hữu!", mon.Name, adjX, adjY)
					}
					// Mark the original monster cell as cleared
					if err := cache.SetCellCleared(player.CurrentRegion, adjX, adjY, player.ID.String(), clearedCellCooldown); err != nil {
						logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, adjX, adjY, err)
					}
					cache.BroadcastMonsterKilled(adjX, adjY, mon.Name, player.CurrentRegion)
					if player.ClearedCells == nil {
						player.ClearedCells = make(models.ClearedCellsMap)
					}
					coordKeyAdj := fmt.Sprintf("%d,%d", adjX, adjY)
					player.ClearedCells[coordKeyAdj] = time.Now().Format(time.RFC3339)
				}
			}
		}
	}

	if player.ActiveCombat != nil {
		player.ActiveCombat = StartCombat(tx, player.ActiveCombat, player)
	}

	// Fire ON_EXPLORE_MOVE on equipped Gus
	var exploreEventMsg string
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				if msg, err := HandleGuEvent(player, spec, gu, "ON_EXPLORE_MOVE"); err == nil && msg != "" {
					exploreEventMsg = msg
				}
			}
		}
	}
	if exploreEventMsg != "" {
		logMsg += " " + exploreEventMsg
	}

	return logMsg, combatTriggered, nil
}

// MoveRequest specifies target position
type MoveRequest struct {
	X              int    `json:"x"`
	Y              int    `json:"y"`
	AttackTargetID string `json:"attack_target_id,omitempty"`
}

// hpTooLowForExplore reports whether a player with HP ≤ 20% max HP is trying
// to move into the wild. Movement is still allowed towards the safe hub so a
// wounded player can always regroup and heal.
func hpTooLowForExplore(player *models.Player, destX, destY int) bool {
	if player == nil || player.HP <= 0 {
		return true
	}
	maxHP := gamefx.EffectiveMaxHP(player)
	if maxHP <= 0 {
		maxHP = 1.0
	}
	if player.HP > 0.20*maxHP {
		return false
	}
	return DeriveRegionAt(destX, destY) != HubRegionID
}

// lineCells returns the cells on a straight line from (x0,y0) to (x1,y1)
// using Bresenham's algorithm, including the target cell and excluding the
// starting cell.
func lineCells(x0, y0, x1, y1 int) [][2]int {
	dx := x1 - x0
	dy := y1 - y0
	sx := 1
	if dx < 0 {
		sx = -1
	}
	sy := 1
	if dy < 0 {
		sy = -1
	}
	adx := dx
	if adx < 0 {
		adx = -adx
	}
	ady := dy
	if ady < 0 {
		ady = -ady
	}
	err := adx - ady
	x, y := x0, y0
	var cells [][2]int
	for {
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -ady {
			err -= ady
			x += sx
		}
		if e2 < adx {
			err += adx
			y += sy
		}
		cells = append(cells, [2]int{x, y})
	}
	return cells
}

// MovePathPlayerUpdate is a lightweight player snapshot for movement responses,
// avoiding sending huge inventories, recipes, and quest histories on high-frequency moves.
type MovePathPlayerUpdate struct {
	ID                 uuid.UUID                 `json:"id"`
	X                  int                       `json:"x"`
	Y                  int                       `json:"y"`
	CurrentRegion      string                    `json:"current_region"`
	HP                 float64                   `json:"hp"`
	BaseHP             float64                   `json:"base_hp"`
	CurrentMana        float64                   `json:"current_mana"`
	MaxMana            float64                   `json:"max_mana"`
	HumanQi            float64                   `json:"human_qi"`
	PrimevalStones     float64                   `json:"primeval_stones"`
	ImmortalStones     float64                   `json:"immortal_stones"`
	MaterialsInventory models.MaterialsInventory `json:"materials_inventory,omitempty"`
	VisitedSubzones    []string                  `json:"visited_subzones,omitempty"`
	ActiveCombat       *models.CombatState       `json:"active_combat"`
	GuInventory        models.GuInventory        `json:"gu_inventory,omitempty"`
}

func buildMovePathPlayerUpdate(player *models.Player, beforeGus models.GuInventory) MovePathPlayerUpdate {
	update := MovePathPlayerUpdate{
		ID:                 player.ID,
		X:                  player.X,
		Y:                  player.Y,
		CurrentRegion:      player.CurrentRegion,
		HP:                 player.HP,
		BaseHP:             gamefx.EffectiveMaxHP(player),
		CurrentMana:        player.CurrentMana,
		MaxMana:            gamefx.EffectiveMaxMana(player),
		HumanQi:            player.HumanQi,
		PrimevalStones:     player.PrimevalStones,
		ImmortalStones:     player.ImmortalStones,
		MaterialsInventory: player.MaterialsInventory,
		VisitedSubzones:    player.VisitedSubZones,
		ActiveCombat:       player.ActiveCombat,
	}
	if len(player.GuInventory) != len(beforeGus) {
		update.GuInventory = player.GuInventory
	}
	return update
}

// ExecuteMovePlayerPath moves the player along a straight path from its current cell
// to the requested cell, decoupled from HTTP gin.Context.
func ExecuteMovePlayerPath(authID uuid.UUID, req MoveRequest) (int, gin.H) {
	spec := movementSpec{kind: movementWalk, x: req.X, y: req.Y, hasTarget: true}
	if err := validateMovementTarget(spec); err != nil {
		status, payload, _ := movementErrorPayload(err)
		return status, payload
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return http.StatusInternalServerError, gin.H{"error": "Database transaction failure"}
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		logger.S.Errorf("MovePlayerPath: failed to evaluate mana for player=%s: %v", authID, err)
		return http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load player Aperture: %v", err)}
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if errMove := checkMovement(tx, &player, spec); errMove != nil {
		if status, payload, ok := movementErrorPayload(errMove); ok {
			return status, payload
		}
		return http.StatusInternalServerError, gin.H{"error": errMove.Error()}
	}

	cleanupExpiredClearedCells(&player, clearedCellCooldown)

	// Calculate Chebyshev distance
	dx := math.Abs(float64(req.X - player.X))
	dy := math.Abs(float64(req.Y - player.Y))
	dist := int(math.Max(dx, dy))

	if dist == 0 {
		return http.StatusBadRequest, gin.H{"error": "Bạn đang ở vị trí này rồi."}
	}

	// Free movement: the player may click anywhere in the world. The path is
	// walked cell by cell (each traversed cell triggers its action), stopping
	// early on the first combat. maxPathCells bounds the work of a single
	// request so the client chains requests for very long journeys. A smaller
	// chunk keeps server-side position commits fine-grained so other clients
	// see near-real-time movement and walking can be redirected sooner.
	const maxPathCells = 5
	path := lineCells(player.X, player.Y, req.X, req.Y)
	if len(path) > maxPathCells {
		path = path[:maxPathCells]
	}
	var mapInstanceID sql.NullString
	if err := tx.QueryRow(`SELECT map_instance_id::text FROM players WHERE id=$1`, player.ID).Scan(&mapInstanceID); err != nil {
		return http.StatusInternalServerError, gin.H{"error": "Không thể tải định nghĩa quái vật từ cơ sở dữ liệu"}
	}
	reached := false
	var eventLogs []string
	combatTriggered := false
	var lastStepLog string
	for _, step := range path {
		stepLog, triggered, errArrival := resolveCellArrivalWithInstance(tx, &player, step[0], step[1], "", true, &mapInstanceID)
		if errArrival != nil {
			return http.StatusInternalServerError, gin.H{"error": "Không thể tải định nghĩa quái vật từ cơ sở dữ liệu"}
		}
		lastStepLog = stepLog
		// Filter out generic movement logs; only keep noteworthy events (encounters, discoveries, gathering, traps, pvp, warnings)
		if !strings.HasPrefix(stepLog, "Di chuyển thành công đến") || strings.Contains(stepLog, "⚠️") {
			eventLogs = append(eventLogs, stepLog)
		}
		if triggered {
			combatTriggered = true
			break
		}
	}
	if !combatTriggered && player.X == req.X && player.Y == req.Y {
		reached = true
	}

	qiGained := 0.0

	// Auto-resolve combat if skip_battle is enabled
	var pendingNotifications []notify.Notification
	if combatTriggered && player.CombatSettings.SkipBattle && player.ActiveCombat != nil {
		res, err := ResolveCombat(tx, &player, nil, nil)
		if err != nil {
			logger.S.Errorf("Auto-resolve combat failed during path move: %v", err)
			_ = tx.Rollback()
			return http.StatusInternalServerError, gin.H{"error": "Tự động chiến đấu thất bại: " + err.Error()}
		} else {
			winner, skipSummary := res.Winner, res.Summary
			afterCombatCommit := res.AfterCommit
			pendingNotifications = res.Notifications
			combatTriggered = false // Combat was auto-resolved
			enemyName := "kẻ thù"
			if e, ok := skipSummary["enemy_name"].(string); ok {
				enemyName = e
			}
			var autoLog string
			if winner == "Phe Ta" {
				if v, ok := skipSummary["human_qi_gained"].(float64); ok {
					qiGained = v
				}
				autoLog = fmt.Sprintf("Tự động chiến thắng %s tại (%d, %d)!", enemyName, player.X, player.Y)
			} else {
				autoLog = fmt.Sprintf("Tự động chiến đấu với %s tại (%d, %d) thất bại! Đạo hữu bị truyền tống về an toàn.", enemyName, player.X, player.Y)
			}
			if skipSummary != nil {
				if loot, ok := skipSummary["loot"].([]string); ok && len(loot) > 0 {
					autoLog += " " + strings.Join(loot, " ")
				}
			}
			eventLogs = append(eventLogs, autoLog)
			if err := tx.Commit(); err != nil {
				logger.S.Errorf("MovePlayerPath: tx.Commit auto-combat failed for player=%s: %v", player.ID, err)
				return http.StatusInternalServerError, gin.H{"error": "Transaction commit failed: " + err.Error()}
			}
			if afterCombatCommit != nil {
				afterCombatCommit()
			}
			for _, notification := range pendingNotifications {
				notify.Publish(notification)
			}
			syncPlayerFlying(&player)
			broadcastPlayerWorldPos(&player)
			PublishPlayerSnapshot(&player, PlayerEventUpdated)

			finalLog := strings.Join(eventLogs, "\n")
			if qiGained > 0 {
				finalLog += fmt.Sprintf(" Nhân Khí +%.0f.", qiGained)
			}
			return http.StatusOK, gin.H{
				"player":           buildMovePathPlayerUpdate(&player, before),
				"log":              finalLog,
				"combat_triggered": false,
				"skip_result":      skipSummary,
				"reached":          reached,
			}
		}
	}

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		logger.S.Errorf("MovePlayerPath: failed to sync Gu inventory for player=%s: %v", player.ID, err)
		return http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"}
	}
	var combatJSON any
	if player.ActiveCombat != nil {
		b, err := json.Marshal(player.ActiveCombat)
		if err != nil {
			logger.S.Errorf("MovePlayerPath: failed to marshal active combat for player=%s: %v", player.ID, err)
			return http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"}
		}
		combatJSON = string(b)
	}
	if err := savePlayerMovement(tx, &player, combatJSON); err != nil {
		logger.S.Errorf("MovePlayerPath: failed to update player=%s at (%d,%d), region=%s, combat=%t: %v", player.ID, player.X, player.Y, player.CurrentRegion, player.ActiveCombat != nil, err)
		return http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"}
	}

	syncedMembers := SyncPartyMembersLocation(tx, &player)

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("MovePlayerPath: tx.Commit failed for player=%s: %v", player.ID, err)
		return http.StatusInternalServerError, gin.H{"error": "Transaction commit failure: " + err.Error()}
	}

	syncPlayerFlying(&player)
	broadcastPlayerWorldPos(&player)
	publishPartyMovement(syncedMembers, publishPartyPositionDelta)

	PublishPlayerSnapshot(&player, PlayerEventUpdated)
	if combatTriggered {
		PublishPlayerCombatDeltaWithPos(&player, CombatEventStarted, player.ActiveCombat)
		if player.ActiveCombat != nil && player.ActiveCombat.EnemyType == "PVP" {
			if defenderID, parseErr := uuid.Parse(player.ActiveCombat.TargetPlayerID); parseErr == nil {
				var defenderCombat models.CombatState
				if loadErr := db.DB.QueryRow(`SELECT active_combat FROM players WHERE id=$1`, defenderID).Scan(&defenderCombat); loadErr == nil {
					PublishPlayerCombatDelta(defenderID, CombatEventStarted, &defenderCombat)
				}
			}
		}
	}

	var finalLog string
	if len(eventLogs) > 0 {
		finalLog = strings.Join(eventLogs, "\n")
		if !combatTriggered {
			finalLog += fmt.Sprintf("\nĐã di chuyển đến (%d, %d).", player.X, player.Y)
		}
	} else {
		if lastStepLog != "" {
			finalLog = lastStepLog
		} else {
			finalLog = fmt.Sprintf("Đã di chuyển đến (%d, %d).", player.X, player.Y)
		}
	}
	if qiGained > 0 {
		finalLog += fmt.Sprintf(" Nhân Khí +%.0f.", qiGained)
	}

	return http.StatusOK, gin.H{
		"player":               buildMovePathPlayerUpdate(&player, before),
		"synced_party_members": syncedMembers,
		"log":                  finalLog,
		"combat_triggered":     combatTriggered,
		"reached":              reached,
	}
}

// MovePlayer handles POST /api/explore/move
func MovePlayer(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	status, payload := ExecuteMovePlayerPath(authID, req)
	c.JSON(status, payload)
}
