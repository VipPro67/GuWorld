package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
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
func CheckAndApplyRegionalWallDebuff(player *models.Player, fromX, fromY, toX, toY int) string {
	if player == nil || (fromX == 0 && fromY == 0) {
		return ""
	}
	oldZone := FindLargeZoneAt(fromX, fromY)
	newZone := FindLargeZoneAt(toX, toY)
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
	player.Statuses.Add("regional_wall_suppression", "Giới Vực Bích", duration, debuffPct)

	debuffInt := int(math.Abs(debuffPct * 100))
	durMinutes := int(duration.Minutes())
	if durMinutes >= 60 {
		return fmt.Sprintf("⚠️ Đạo hữu vừa vượt qua Giới Vực Bích từ %s sang %s! Quy tắc địa giới phản phệ gây ra [Áp Chế Giới Vực] (Công/Thủ/Tốc giảm %d%% trong %d giờ).",
			oldZone.Name, newZone.Name, debuffInt, durMinutes/60)
	}
	return fmt.Sprintf("⚠️ Đạo hữu vừa vượt qua Giới Vực Bích từ %s sang %s! Quy tắc địa giới phản phệ gây ra [Áp Chế Giới Vực] (Công/Thủ/Tốc giảm %d%% trong %d phút).",
		oldZone.Name, newZone.Name, debuffInt, durMinutes)
}

// returns the log message plus whether a fight was started. attackTargetID is
// only used to explicitly attack another player standing on the destination.
func resolveCellArrival(tx *sql.Tx, player *models.Player, x, y int, attackTargetID string, allowAmbush bool) (string, bool, error) {
	oldX := player.X
	oldY := player.Y
	mapID := worldMapIDForRegion(player.CurrentRegion)
	var instanceID sql.NullString
	if err := tx.QueryRow(`SELECT map_instance_id::text FROM players WHERE id=$1`, player.ID).Scan(&instanceID); err != nil {
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
		destinationRegion = DeriveRegionAtOnMap(x, y, mapID)
	}
	if destinationRegion != player.CurrentRegion {
		if err := checkRegionEntry(destinationRegion, player.Rank, player.GuInventory); err != nil {
			return err.Error(), false, nil
		}
	}
	player.X = x
	player.Y = y
	player.CurrentRegion = destinationRegion
	recordVisitedSubZone(player, SubZoneKey(player.X, player.Y))

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
	if debuffMsg := CheckAndApplyRegionalWallDebuff(player, oldX, oldY, player.X, player.Y); debuffMsg != "" {
		logMsg += " " + debuffMsg
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
				  AND (p.last_activity >= NOW() - INTERVAL '5 minutes' OR p.last_mana_update >= NOW() - INTERVAL '5 minutes')
				  AND p.map_instance_id IS NOT DISTINCT FROM (SELECT map_instance_id FROM players WHERE id=$2)
				  AND COALESCE((SELECT world_map_id FROM region_world_maps WHERE region_id=p.current_region),'world_map') =
				      COALESCE((SELECT world_map_id FROM region_world_maps WHERE region_id=(SELECT current_region FROM players WHERE id=$2)),'world_map')
			`
			var errTarget error
			var matchedID uuid.UUID
			if player.PartyID != nil {
				queryStr += ` AND (p.party_id IS NULL OR p.party_id != $3)`
				errTarget = tx.QueryRow(queryStr, targetUUID, player.ID, player.PartyID).Scan(&matchedID)
			} else {
				errTarget = tx.QueryRow(queryStr, targetUUID, player.ID).Scan(&matchedID)
			}

			if errTarget != nil {
				if errTarget != sql.ErrNoRows {
					return "", false, errTarget
				}
				var exists bool
				_ = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM players WHERE id = $1)`, targetUUID).Scan(&exists)
				if exists {
					return "Cổ Sư này hiện đang quy ẩn (Offline), không thể khai chiến!", false, nil
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
				if err := checkRegionAction(targetPlayer.CurrentRegion, "combat"); err != nil {
					return err.Error(), false, nil
				}
				if err := checkRegionEquippedGus(targetPlayer.CurrentRegion, targetPlayer.GuInventory); err != nil {
					return err.Error(), false, nil
				}
				// If attacker has PvP immunity, attacking someone breaks their immunity
				if player.Statuses.Has("pvp_immunity") {
					RemovePvpImmunity(tx, player.ID)
					player.Statuses.Remove("pvp_immunity")
					logMsg += " [Chủ động khai chiến, trạng thái Miễn Chiến bị phá vỡ!]"
				}

				// Check target Sát Khí level
				_, targetSatKhiLevel := getSatKhiMultiplier(tx, targetUUID)

				// Target PvP immunity check (after death/respawn)
				if targetPlayer.Statuses.Has("pvp_immunity") {
					return fmt.Sprintf("Cổ Sư %s đang trong thời gian Miễn Chiến sau khi hồi sinh, không thể tấn công!", targetPlayer.Username), false, nil
				}

				// Distance check: Chebyshev distance max(|x1-x2|, |y1-y2|)
				dx := math.Abs(float64(targetPlayer.X - player.X))
				dy := math.Abs(float64(targetPlayer.Y - player.Y))
				dist := int(math.Max(dx, dy))

				// Range requirement: base 2 tiles + bonus stats (attack_range / combat_range)
				stats := CalculatePlayerStats(player)
				bonusRange := int(stats["attack_range"]) + int(stats["combat_range"])
				maxPvPRange := 2 + bonusRange

				// Safe zone check (Wanted criminal with Sát Khí > 10 can be attacked in safe zones)
				isSafe1 := IsSafeZone(player.CurrentRegion, player.X, player.Y)
				isSafe2 := IsSafeZone(targetPlayer.CurrentRegion, targetPlayer.X, targetPlayer.Y)

				if targetSatKhiLevel > 10 {
					// Target's safe zone immunity is stripped due to excessive killing intent
					isSafe2 = false
					if isSafe1 {
						isSafe1 = false // Attacker can strike the wanted criminal from safe zone
					}
				}

				if isSafe1 || isSafe2 {
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
					combatTriggered = true
					pvpTriggered = true
					targetStats := CalculatePlayerStats(&targetPlayer)

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

					pvpCombat := models.CombatState{
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
			if HasExplorationInventorySpace(player, 1) {
				if err := InventorySvc.AddItem(tx, player.ID.String(), cell.Content, "materials", 1); err != nil {
					logger.S.Errorf("Failed to add resource %s to player %s: %v", cell.Content, player.ID, err)
					// Propagate: a failed write poisons the tx (25P02 on next statement).
					return "", false, err
				}
				logMsg = fmt.Sprintf("Di chuyển đến (%d, %d). Đạo hữu tìm thấy nguyên liệu hoang dã: 1x %s!", player.X, player.Y, itemName)

				if err := cache.SetCellCleared(player.CurrentRegion, player.X, player.Y, player.ID.String(), clearedCellCooldown); err != nil {
					logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, player.X, player.Y, err)
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
			if HasExplorationInventorySpace(player, 1) {
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

				if err := cache.SetCellCleared(player.CurrentRegion, player.X, player.Y, player.ID.String(), clearedCellCooldown); err != nil {
					logger.S.Errorf("Failed to mark cell cleared in Redis for player %s at (%d,%d): %v", player.ID, player.X, player.Y, err)
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

			combatState := models.BuildCombatStateFromMonster(mon, rank, subRank)
			player.ActiveCombat = &combatState
			logMsg = fmt.Sprintf("Nguy hiểm! Đạo hữu chạm trán %s tại tọa độ (%d, %d)!", mon.Name, player.X, player.Y)

			hasHeavenSecret := models.InventoryHasGuCapability(player.GuInventory, "LOOT_PREVIEW", true)
			if hasHeavenSecret && mon.LootTableID != "" {
				lt, errLt := db.LoadLootTable(tx, mon.LootTableID)
				if errLt == nil {
					chanceMul := lootChanceMultiplier(rank)
					qtyMul := lootQuantityMultiplier(rank)
					extraLootMsg := fmt.Sprintf(" [Thiên Cơ hiển lộ (Rank %d x%.1f) - Tỉ lệ rơi đồ: ", rank, qtyMul)
					first := true
					for _, entry := range lt.Entries {
						if !first {
							extraLootMsg += ", "
						}
						scaledChance := math.Min(entry.Chance*chanceMul, 1.0)
						scaledMin := int(math.Round(float64(entry.MinQty) * qtyMul))
						scaledMax := int(math.Round(float64(entry.MaxQty) * qtyMul))
						if itemID := entry.ItemID; itemID != "" {
							extraLootMsg += fmt.Sprintf("%s x%d-%d (%.1f%%)", itemID, scaledMin, scaledMax, scaledChance*100)
						} else if entry.GuID != "" {
							extraLootMsg += fmt.Sprintf("%s (%.1f%%)", entry.GuID, scaledChance*100)
						}
						first = false
					}
					extraLootMsg += "]"
					logMsg += extraLootMsg
				}
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

	if !combatTriggered && allowAmbush && !playerTerritory.InTerritory {
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
						combatState := models.BuildCombatStateFromMonster(mon, rank, subRank)
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

// MovePlayer moves the player on the shared world grid and triggers cell actions
func MovePlayer(c *gin.Context) {
	var req MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.X < 0 || req.X >= WorldWidth || req.Y < 0 || req.Y >= WorldHeight {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ vượt ra ngoài ranh giới bản đồ."})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		logger.S.Errorf("MovePlayer: failed to evaluate mana for player=%s: %v", authID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load player Aperture: %v", err)})
		return
	}
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể di chuyển!"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errLeader.Error()})
		return
	}

	cleanupExpiredClearedCells(&player, clearedCellCooldown)

	if player.HP <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong trạng thái trọng thương, không thể tiếp tục di chuyển!"})
		return
	}

	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đang trong trạng thái chiến đấu! Hãy kết thúc trận chiến trước."})
		return
	}

	stats := CalculatePlayerStats(&player)
	// Toc chay map = MOVEMENT (so o). SPEED chi dung thu tu danh combat, khong anh huong di chuyen.
	movementRange := int(stats["movement"])
	if movementRange < 1 {
		movementRange = 1
	}

	// Calculate Chebyshev distance
	dx := math.Abs(float64(req.X - player.X))
	dy := math.Abs(float64(req.Y - player.Y))
	dist := int(math.Max(dx, dy))

	if dist == 0 && req.AttackTargetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đang ở vị trí này rồi."})
		return
	}

	hasFixedImmortalTravel := models.InventoryHasGuCapability(player.GuInventory, "IGNORE_MOVEMENT_RANGE", true)

	if dist > movementRange && !hasFixedImmortalTravel {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vị trí quá xa! Thân pháp hiện tại chỉ di chuyển tối đa %d ô.", movementRange)})
		return
	}

	// Movement no longer consumes mana
	logMsg, combatTriggered, errArrival := resolveCellArrival(tx, &player, req.X, req.Y, req.AttackTargetID, true)
	if errArrival != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải định nghĩa quái vật từ cơ sở dữ liệu"})
		return
	}

	// Auto-resolve combat if skip_battle is enabled
	var pendingNotifications []notify.Notification
	if combatTriggered && player.CombatSettings.SkipBattle && player.ActiveCombat != nil {
		winner, _, skipSummary, notifications, err := AutoResolveCombat(tx, &player)
		if err != nil {
			logger.S.Errorf("Auto-resolve combat failed during move: %v", err)
			_ = tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Tự động chiến đấu thất bại: " + err.Error()})
			return
		} else {
			pendingNotifications = notifications
			combatTriggered = false // Combat was auto-resolved
			enemyName := "kẻ thù"
			if e, ok := skipSummary["enemy_name"].(string); ok {
				enemyName = e
			}
			if winner == "Phe Ta" {
				logMsg = fmt.Sprintf("Tự động chiến thắng %s tại (%d, %d)! Nhân Khí +3.", enemyName, player.X, player.Y)
			} else {
				logMsg = fmt.Sprintf("Tự động chiến đấu với %s tại (%d, %d) thất bại! Đạo hữu bị truyền tống về an toàn.", enemyName, player.X, player.Y)
			}
			if skipSummary != nil {
				if loot, ok := skipSummary["loot"].([]string); ok && len(loot) > 0 {
					logMsg += " " + strings.Join(loot, " ")
				}
				if mineOccupied, ok := skipSummary["mine_occupied"].(int); ok && mineOccupied > 0 {
					logMsg += fmt.Sprintf(" Đã chiếm mỏ %d!", mineOccupied)
				}
			}
			if err := tx.Commit(); err != nil {
				logger.S.Errorf("MovePlayer: tx.Commit auto-combat failed for player=%s: %v", player.ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed: " + err.Error()})
				return
			}
			for _, notification := range pendingNotifications {
				notify.Publish(notification)
			}
			ClearPendingPlayerQueues(player.ID.String())
			PublishPlayerCombatDelta(player.ID, CombatEventEnded, nil)
			// Include skip summary in response
			respondActionWithInventory(c, http.StatusOK, player, gin.H{
				"log":              logMsg,
				"combat_triggered": false,
				"skip_result":      skipSummary,
				"stats":            CalculatePlayerStats(&player),
			})
			return
		}
	}

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		logger.S.Errorf("MovePlayer: failed to sync Gu inventory for player=%s: %v", player.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}
	var combatJSON any
	if player.ActiveCombat != nil {
		b, err := json.Marshal(player.ActiveCombat)
		if err != nil {
			logger.S.Errorf("MovePlayer: failed to marshal active combat for player=%s: %v", player.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
			return
		}
		combatJSON = string(b)
	}
	if err := savePlayerMovement(tx, &player, combatJSON); err != nil {
		logger.S.Errorf("MovePlayer: failed to update player=%s at (%d,%d), region=%s, combat=%t: %v", player.ID, player.X, player.Y, player.CurrentRegion, player.ActiveCombat != nil, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}

	syncedMembers := SyncPartyMembersLocation(tx, &player)

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("MovePlayer: tx.Commit failed for player=%s: %v", player.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure: " + err.Error()})
		return
	}

	broadcastWorldPos(WorldPosEvent{
		ID:        player.ID.String(),
		Username:  player.Username,
		AvatarURL: player.AvatarURL,
		X:         player.X,
		Y:         player.Y,
		Region:    player.CurrentRegion,
		TS:        time.Now().UnixMilli(),
	})
	for _, sm := range syncedMembers {
		broadcastWorldPos(WorldPosEvent{
			ID:        sm.ID.String(),
			Username:  sm.Username,
			AvatarURL: sm.AvatarURL,
			X:         sm.X,
			Y:         sm.Y,
			Region:    sm.Region,
			TS:        time.Now().UnixMilli(),
		})
		PublishPlayerPositionDelta(sm.ID, sm.X, sm.Y)
	}

	if combatTriggered {
		PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
	}

	// Calculate fresh stats
	newStats := CalculatePlayerStats(&player)

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"log":                  logMsg,
		"combat_triggered":     combatTriggered,
		"stats":                newStats,
		"synced_party_members": syncedMembers,
	})
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
	ID              uuid.UUID           `json:"id"`
	X               int                 `json:"x"`
	Y               int                 `json:"y"`
	CurrentRegion   string              `json:"current_region"`
	HP              float64             `json:"hp"`
	BaseHP          float64             `json:"base_hp"`
	CurrentMana     float64             `json:"current_mana"`
	MaxMana         float64             `json:"max_mana"`
	HumanQi         float64             `json:"human_qi"`
	PrimevalStones  float64             `json:"primeval_stones"`
	ImmortalStones  float64             `json:"immortal_stones"`
	VisitedSubzones []string            `json:"visited_subzones,omitempty"`
	ActiveCombat    *models.CombatState `json:"active_combat"`
	GuInventory     models.GuInventory  `json:"gu_inventory,omitempty"`
}

func buildMovePathPlayerUpdate(player *models.Player, beforeGus models.GuInventory) MovePathPlayerUpdate {
	update := MovePathPlayerUpdate{
		ID:              player.ID,
		X:               player.X,
		Y:               player.Y,
		CurrentRegion:   player.CurrentRegion,
		HP:              player.HP,
		BaseHP:          gamefx.EffectiveMaxHP(player),
		CurrentMana:     player.CurrentMana,
		MaxMana:         gamefx.EffectiveMaxMana(player),
		HumanQi:         player.HumanQi,
		PrimevalStones:  player.PrimevalStones,
		ImmortalStones:  player.ImmortalStones,
		VisitedSubzones: player.VisitedSubZones,
		ActiveCombat:    player.ActiveCombat,
	}
	if len(player.GuInventory) != len(beforeGus) {
		update.GuInventory = player.GuInventory
	}
	return update
}

// MovePlayerPath moves the player along a straight path from its current cell
// to the requested cell, triggering the normal cell action on every traversed
// cell. Movement is still bounded by the player's movement stat (base +
// passive movement). The walk stops early if a fight is triggered on the way.
func MovePlayerPath(c *gin.Context) {
	var req MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.X < 0 || req.X >= WorldWidth || req.Y < 0 || req.Y >= WorldHeight {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ vượt ra ngoài ranh giới bản đồ."})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		logger.S.Errorf("MovePlayerPath: failed to evaluate mana for player=%s: %v", authID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load player Aperture: %v", err)})
		return
	}
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể di chuyển!"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	if errLeader := checkPartyLeaderOrSolo(tx, &player); errLeader != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": errLeader.Error()})
		return
	}

	cleanupExpiredClearedCells(&player, clearedCellCooldown)

	if player.HP <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đang trong trạng thái trọng thương, không thể tiếp tục di chuyển!"})
		return
	}

	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đang trong trạng thái chiến đấu! Hãy kết thúc trận chiến trước."})
		return
	}

	stats := CalculatePlayerStats(&player)
	// Toc chay map = MOVEMENT (so o). SPEED chi dung thu tu danh combat, khong anh huong di chuyen.
	movementRange := int(stats["movement"])
	if movementRange < 1 {
		movementRange = 1
	}

	// Calculate Chebyshev distance
	dx := math.Abs(float64(req.X - player.X))
	dy := math.Abs(float64(req.Y - player.Y))
	dist := int(math.Max(dx, dy))

	if dist == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bạn đang ở vị trí này rồi."})
		return
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
	reached := false
	var eventLogs []string
	combatTriggered := false
	var lastStepLog string
	for _, step := range path {
		stepLog, triggered, errArrival := resolveCellArrival(tx, &player, step[0], step[1], "", true)
		if errArrival != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải định nghĩa quái vật từ cơ sở dữ liệu"})
			return
		}
		lastStepLog = stepLog
		// Filter out generic movement logs; only keep noteworthy events (encounters, discoveries, gathering, traps, pvp)
		if !strings.HasPrefix(stepLog, "Di chuyển thành công đến") {
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
		winner, _, skipSummary, notifications, err := AutoResolveCombat(tx, &player)
		if err != nil {
			logger.S.Errorf("Auto-resolve combat failed during path move: %v", err)
			_ = tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Tự động chiến đấu thất bại: " + err.Error()})
			return
		} else {
			pendingNotifications = notifications
			combatTriggered = false // Combat was auto-resolved
			enemyName := "kẻ thù"
			if e, ok := skipSummary["enemy_name"].(string); ok {
				enemyName = e
			}
			var autoLog string
			if winner == "Phe Ta" {
				autoLog = fmt.Sprintf("Tự động chiến thắng %s tại (%d, %d)! Nhân Khí +3.", enemyName, player.X, player.Y)
			} else {
				autoLog = fmt.Sprintf("Tự động chiến đấu với %s tại (%d, %d) thất bại! Đạo hữu bị truyền tống về an toàn.", enemyName, player.X, player.Y)
			}
			if skipSummary != nil {
				if loot, ok := skipSummary["loot"].([]string); ok && len(loot) > 0 {
					autoLog += " " + strings.Join(loot, " ")
				}
				if mineOccupied, ok := skipSummary["mine_occupied"].(int); ok && mineOccupied > 0 {
					autoLog += fmt.Sprintf(" Đã chiếm mỏ %d!", mineOccupied)
				}
			}
			eventLogs = append(eventLogs, autoLog)
			if err := tx.Commit(); err != nil {
				logger.S.Errorf("MovePlayerPath: tx.Commit auto-combat failed for player=%s: %v", player.ID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed: " + err.Error()})
				return
			}
			for _, notification := range pendingNotifications {
				notify.Publish(notification)
			}
			finalLog := strings.Join(eventLogs, "\n")
			if qiGained > 0 {
				finalLog += fmt.Sprintf(" Nhân Khí +%.0f.", qiGained)
			}
			c.JSON(http.StatusOK, gin.H{
				"player":           buildMovePathPlayerUpdate(&player, before),
				"log":              finalLog,
				"combat_triggered": false,
				"skip_result":      skipSummary,
				"reached":          reached,
			})
			return
		}
	}

	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		logger.S.Errorf("MovePlayerPath: failed to sync Gu inventory for player=%s: %v", player.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}
	var combatJSON any
	if player.ActiveCombat != nil {
		b, err := json.Marshal(player.ActiveCombat)
		if err != nil {
			logger.S.Errorf("MovePlayerPath: failed to marshal active combat for player=%s: %v", player.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
			return
		}
		combatJSON = string(b)
	}
	if err := savePlayerMovement(tx, &player, combatJSON); err != nil {
		logger.S.Errorf("MovePlayerPath: failed to update player=%s at (%d,%d), region=%s, combat=%t: %v", player.ID, player.X, player.Y, player.CurrentRegion, player.ActiveCombat != nil, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update movement coordinates"})
		return
	}

	syncedMembers := SyncPartyMembersLocation(tx, &player)

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("MovePlayerPath: tx.Commit failed for player=%s: %v", player.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure: " + err.Error()})
		return
	}

	broadcastWorldPos(WorldPosEvent{
		ID:        player.ID.String(),
		Username:  player.Username,
		AvatarURL: player.AvatarURL,
		X:         player.X,
		Y:         player.Y,
		Region:    player.CurrentRegion,
		TS:        time.Now().UnixMilli(),
	})
	for _, sm := range syncedMembers {
		broadcastWorldPos(WorldPosEvent{
			ID:        sm.ID.String(),
			Username:  sm.Username,
			AvatarURL: sm.AvatarURL,
			X:         sm.X,
			Y:         sm.Y,
			Region:    sm.Region,
			TS:        time.Now().UnixMilli(),
		})
		PublishPlayerPositionDelta(sm.ID, sm.X, sm.Y)
	}

	if combatTriggered {
		PublishPlayerCombatDelta(player.ID, CombatEventStarted, player.ActiveCombat)
	} else {
		PublishPlayerSnapshot(&player, PlayerEventUpdated)
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

	c.JSON(http.StatusOK, gin.H{
		"player":               buildMovePathPlayerUpdate(&player, before),
		"synced_party_members": syncedMembers,
		"log":                  finalLog,
		"combat_triggered":     combatTriggered,
		"reached":              reached,
	})
}
