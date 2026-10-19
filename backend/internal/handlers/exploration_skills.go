package handlers

import (
	"context"
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
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type WorldCastTarget struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	RegionID string `json:"region_id,omitempty"`
}

type UseGuOnMapRequest struct {
	CommandID    string           `json:"command_id"`
	GuInstanceID string           `json:"gu_instance_id" binding:"required"`
	Target       *WorldCastTarget `json:"target,omitempty"`
	TargetX      int              `json:"target_x"`
	TargetY      int              `json:"target_y"`
}

// chebyshevDistance returns the grid distance used by world movement and map
// skills (diagonal steps count as one).
func chebyshevDistance(x1, y1, x2, y2 int) int {
	dx, dy := x2-x1, y2-y1
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// randomTeleportTarget picks a random destination. A positive limit keeps the
// jump within that many cells of the caster; 0 means anywhere on the map.
func randomTeleportTarget(cx, cy, limit int) (int, int) {
	if limit <= 0 {
		return rand.Intn(WorldWidth), rand.Intn(WorldHeight)
	}
	span := 2*limit + 1
	x := cx + rand.Intn(span) - limit
	y := cy + rand.Intn(span) - limit
	if x < 0 {
		x = 0
	} else if x >= WorldWidth {
		x = WorldWidth - 1
	}
	if y < 0 {
		y = 0
	} else if y >= WorldHeight {
		y = WorldHeight - 1
	}
	return x, y
}

// persistGuHungerTx writes a Gu's post-cast hunger with a short bounded retry.
//
// Every mutation flow that calls EvaluateAndPersistMana runs with a 2s
// transaction lock_timeout (setFastFailTimeouts) and lazy hunger decay
// (EvaluatePlayerHungerTx) also updates the same player_gu rows. A cast can
// therefore lose a lock race and abort the statement with 55P03, which used to
// surface as the misleading "Không thể cập nhật độ no Cổ trùng" error. Retrying
// the whole cast is unsafe (position, cooldown and monster side effects would be
// replayed), so only this single write is retried inside a SAVEPOINT:
// PostgreSQL allows ROLLBACK TO SAVEPOINT on an aborted transaction and then
// lets the statement run again.
//
// The attempt count stays deliberately small: the caller already holds the
// players row FOR UPDATE, so every extra wait also blocks concurrent requests
// for the same player.
func persistGuHungerTx(ctx context.Context, tx *sql.Tx, playerID uuid.UUID, gu *models.GuInstance) error {
	const attempts = 2
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if _, err := tx.ExecContext(ctx, `SAVEPOINT sp_gu_hunger`); err != nil {
			return err
		}
		_, err := tx.ExecContext(ctx,
			`UPDATE player_gu SET hunger = $1, updated_at = NOW() WHERE id = $2 AND player_id = $3`,
			gu.Hunger, gu.InstanceID, playerID)
		if err == nil {
			_, err = tx.ExecContext(ctx, `RELEASE SAVEPOINT sp_gu_hunger`)
			return err
		}
		if _, rbErr := tx.ExecContext(ctx, `ROLLBACK TO SAVEPOINT sp_gu_hunger`); rbErr != nil {
			// The transaction is unusable now; surface that instead of the
			// transient error so the caller does not advertise a retry.
			return rbErr
		}
		lastErr = err
		if !worker.IsHungerTransientError(err) || attempt == attempts-1 {
			break
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 150 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return err
		case <-timer.C:
		}
	}
	return lastErr
}

func UseGuOnMap(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req UseGuOnMapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot load player"})
		return
	}

	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể sử dụng kỹ năng World Map khi đang trong chiến đấu!"})
		return
	}

	var targetGu *models.GuInstance
	for i := range player.GuInventory {
		if player.GuInventory[i].InstanceID == req.GuInstanceID {
			targetGu = &player.GuInventory[i]
			break
		}
	}
	if targetGu == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này."})
		return
	}

	specs, found := models.GetGu(targetGu.GuID, targetGu.Rank)
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Cổ trùng không tồn tại trong từ điển."})
		return
	}
	if err := checkRegionAction(player.CurrentRegion, "gu_use"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	if err := checkRegionGu(player.CurrentRegion, specs.ID, specs.Rank, specs.Path); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Validate usability
	if err := ValidateGuUsable(&player, targetGu, &specs, true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check Cooldown
	cdID := "cd_gu_" + req.GuInstanceID
	if player.Statuses.Has(cdID) {
		activeStatuses := player.Statuses.Active()
		for _, st := range activeStatuses {
			if st.ID == cdID {
				remains := time.Until(*st.ExpiresAt).Seconds()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cổ trùng đang trong thời gian hồi (%.0f giây)", remains)})
				return
			}
		}
	}

	_, hasPortalCreate := models.GuSpecCapability(specs, "PORTAL_CREATE")
	_, hasRandomPortal := models.GuSpecCapability(specs, "RANDOM_PORTAL")
	if specs.Effects.Active != nil {
		if specs.Effects.Active.PortalCreate {
			hasPortalCreate = true
		}
		if specs.Effects.Active.RandomPortal {
			hasRandomPortal = true
		}
	}
	// IGNORE_MOVEMENT_RANGE also enables targeted teleport, but a random-portal
	// Cổ must stay random (the client sends no coordinates for it).
	if _, hasIgnoreRange := models.GuSpecCapability(specs, "IGNORE_MOVEMENT_RANGE"); hasIgnoreRange && !hasPortalCreate && !hasRandomPortal {
		hasPortalCreate = true
	}
	teleportRange := GuTeleportRange(&specs)

	cost := gamefx.EnergyCost(&player, &specs)
	if hasPortalCreate || hasRandomPortal {
		if err := checkRegionAction(player.CurrentRegion, "teleport"); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		var inSpatialInstance bool
		if err := tx.QueryRow(`SELECT map_instance_id IS NOT NULL FROM players WHERE id=$1`, player.ID).Scan(&inSpatialInstance); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra phiên bản đồ hiện tại"})
			return
		}
		if inSpatialInstance {
			c.JSON(http.StatusForbidden, gin.H{"error": "Không thể dịch chuyển trong bí cảnh; hãy dùng cổng thoát"})
			return
		}
		if err := checkPartyTeleportRankGate(tx, &player, targetGu.Rank); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		cost = GetPortalManaCost(&player, &specs)
	}
	logs := []string{}
	var cdDuration time.Duration = 10 * time.Second // default CD
	applied := false
	teleported := false
	effectType := "utility"
	effectValue := 0.0

	// Apply Portal / Teleport effects
	if hasPortalCreate {
		if targetGu.Rank < player.Rank {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Không thể thực hiện dịch chuyển: Cổ trùng dịch chuyển (%s Rank %d) thấp hơn tu vi bản thân (Rank %d)!", specs.Name, targetGu.Rank, player.Rank),
			})
			return
		}
		if teleportRange > 0 {
			dist := chebyshevDistance(player.X, player.Y, req.TargetX, req.TargetY)
			if dist > teleportRange {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": fmt.Sprintf("Cổ trùng %s chỉ dịch chuyển tối đa %d ô (mục tiêu cách %d ô).", specs.Name, teleportRange, dist),
				})
				return
			}
		}
		// We can teleport to TargetX, TargetY if within bounds
		if inWorld(req.TargetX, req.TargetY) {
			oldX := player.X
			oldY := player.Y
			oldRegion := player.CurrentRegion
			targetRegion := DeriveRegionAtOnMap(req.TargetX, req.TargetY, worldMapIDForRegion(player.CurrentRegion))
			if targetRegion != oldRegion {
				if err := checkRegionEntry(targetRegion, player.Rank, player.GuInventory); err != nil {
					c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
					return
				}
			}
			player.X = req.TargetX
			player.Y = req.TargetY
			player.CurrentRegion = targetRegion
			teleportMsg := fmt.Sprintf("🌀 Cổ trùng %s xé rách không gian, đưa %s dịch chuyển từ (%d, %d) [%s] đến (%d, %d) [%s].", specs.Name, player.Username, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion)
			debuffMsg := CheckAndApplyRegionalWallDebuffBetween(&player, oldRegion, targetRegion, tx)
			if debuffMsg != "" {
				teleportMsg += " " + debuffMsg
			}
			logs = append(logs, teleportMsg)
			applied = true
			teleported = true
			cdDuration = 120 * time.Second
			effectType = "teleport"
			logger.S.Infof("[Dịch Chuyển Cổ] Người chơi %s (ID: %s) kích hoạt %s (Rank %d) dịch chuyển từ (%d, %d) [%s] đến (%d, %d) [%s]", player.Username, player.ID, specs.Name, targetGu.Rank, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ không hợp lệ!"})
			return
		}
	} else if hasRandomPortal {
		if targetGu.Rank < player.Rank {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Không thể thực hiện dịch chuyển: Cổ trùng dịch chuyển (%s Rank %d) thấp hơn tu vi bản thân (Rank %d)!", specs.Name, targetGu.Rank, player.Rank),
			})
			return
		}
		oldX := player.X
		oldY := player.Y
		oldRegion := player.CurrentRegion
		targetX, targetY := randomTeleportTarget(player.X, player.Y, teleportRange)
		targetRegion := DeriveRegionAtOnMap(targetX, targetY, worldMapIDForRegion(player.CurrentRegion))
		if targetRegion != oldRegion {
			if err := checkRegionEntry(targetRegion, player.Rank, player.GuInventory); err != nil {
				c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}
		}
		player.X = targetX
		player.Y = targetY
		player.CurrentRegion = targetRegion
		teleportMsg := fmt.Sprintf("🌀 Cổ trùng %s xé rách không gian hỗn loạn, đưa %s dịch chuyển ngẫu nhiên từ (%d, %d) [%s] đến (%d, %d) [%s].", specs.Name, player.Username, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion)
		debuffMsg := CheckAndApplyRegionalWallDebuffBetween(&player, oldRegion, targetRegion, tx)
		if debuffMsg != "" {
			teleportMsg += " " + debuffMsg
		}
		logs = append(logs, teleportMsg)
		applied = true
		teleported = true
		cdDuration = 120 * time.Second
		effectType = "teleport"
		logger.S.Infof("[Dịch Chuyển Cổ] Người chơi %s (ID: %s) kích hoạt %s (Rank %d) dịch chuyển ngẫu nhiên từ (%d, %d) [%s] đến (%d, %d) [%s]", player.Username, player.ID, specs.Name, targetGu.Rank, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion)
	}

	// Apply effects from Active (Combat mapping to World Map)
	if specs.Effects.Active != nil {
		active := specs.Effects.Active
		// HOT tick -> heal
		if op, ok := active.StatusOp("REGEN"); ok {
			healVal := op.Value * float64(op.Turns)
			if healVal <= 0 {
				healVal = 100.0
			}
			player.HP += healVal
			if player.HP > gamefx.EffectiveMaxHP(&player) {
				player.HP = gamefx.EffectiveMaxHP(&player)
			}
			logs = append(logs, fmt.Sprintf("%s kích hoạt Hồi Sinh, phục hồi %.1f HP.", specs.Name, healVal))
			applied = true
			cdDuration = 60 * time.Second
			effectType = "heal"
			effectValue = healVal
		}
		// Shield -> Virtual Shield
		if active.ShieldFlat > 0 {
			player.Statuses.Add("virtual_shield", "world_map", 5*time.Minute, active.ShieldFlat)
			logs = append(logs, fmt.Sprintf("%s tạo một lớp ảo thuẫn hộ thể (%.1f) duy trì 5 phút.", specs.Name, active.ShieldFlat))
			applied = true
			cdDuration = 60 * time.Second
			effectType = "shield"
			effectValue = active.ShieldFlat
		}
		// Stat Buffs
		if op, ok := active.StatusOp("ATK_UP"); ok {
			buffValue := op.Value
			if buffValue == 0 {
				buffValue = 50 // fallback
			}
			duration := time.Duration(op.Turns) * time.Minute
			if duration == 0 {
				duration = 5 * time.Minute
			}
			player.Statuses.Add("buff_atk", specs.Name, duration, buffValue)
			logs = append(logs, fmt.Sprintf("%s tăng %.0f Công Kích trong %.0f phút.", specs.Name, buffValue, duration.Minutes()))
			applied = true
			cdDuration = duration + (30 * time.Second)
			effectType = "buff_atk"
			effectValue = buffValue
		}
		if op, ok := active.StatusOp("DEF_UP"); ok {
			buffValue := op.Value
			if buffValue == 0 {
				buffValue = 50 // fallback
			}
			duration := time.Duration(op.Turns) * time.Minute
			if duration == 0 {
				duration = 5 * time.Minute
			}
			player.Statuses.Add("buff_def", specs.Name, duration, buffValue)
			logs = append(logs, fmt.Sprintf("%s tăng %.0f Phòng Ngự trong %.0f phút.", specs.Name, buffValue, duration.Minutes()))
			applied = true
			cdDuration = duration + (30 * time.Second)
			effectType = "buff_def"
			effectValue = buffValue
		}
		if op, ok := active.StatusOp("SPEED_UP"); ok {
			buffValue := op.Value
			if buffValue == 0 {
				buffValue = 20
			}
			duration := time.Duration(op.Turns) * time.Minute
			if duration == 0 {
				duration = 5 * time.Minute
			}
			player.Statuses.Add("buff_speed", specs.Name, duration, buffValue)
			logs = append(logs, fmt.Sprintf("%s gia tăng %.0f Tốc Độ trong %.0f phút.", specs.Name, buffValue, duration.Minutes()))
			applied = true
			cdDuration = duration + (30 * time.Second)
			effectType = "buff_speed"
			effectValue = buffValue
		}
	}

	// Apply Other Mechanics for World Map (Block v2 only: compiled native
	// mechanics with the ON_WORLD_MAP trigger; legacy Other is never read).
	hasWorldMapMechanic := false
	if compiled := models.CompiledGuEffects(specs); compiled != nil {
		for _, op := range compiled.NativeMechanics() {
			if string(op.Trigger) == "ON_WORLD_MAP" {
				hasWorldMapMechanic = true
				break
			}
		}
	}
	if hasWorldMapMechanic {
		msg, err := HandleGuEvent(&player, specs, *targetGu, "ON_WORLD_MAP")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if msg != "" {
			logs = append(logs, msg)
			applied = true
			if effectType == "utility" {
				effectType = "other"
			}
		}
	}

	// Fallback for general active Gu effects if no map-specific condition matched.
	// Use the simulator projection here because Block v2 intentionally leaves the
	// legacy Effects.Active field nil.
	if !applied && hasCombatActiveEffects(&specs) {
		stats := CalculatePlayerStats(&player)
		attacker := models.CombatantFromPlayer(&player, stats)
		act, err := buildSimActionFromGu(&attacker, &specs, "GU_SKILL")
		if err == nil {
			cdDuration = 30 * time.Second
			effectType = "utility"
			if act.Damage > 0 {
				applied = true
				effectValue = act.Damage
				logs = append(logs, fmt.Sprintf("Kích hoạt %s thôi động sát chiêu (Uy lực %.0f)!", specs.Name, act.Damage))
			}
			if act.Defense > 0 {
				applied = true
				effectValue = act.Defense
				player.Statuses.Add("virtual_shield", "world_map", 5*time.Minute, act.Defense)
				logs = append(logs, fmt.Sprintf("Kích hoạt %s tạo lớp ảo thuẫn hộ thể (%.0f) duy trì 5 phút.", specs.Name, act.Defense))
			}
			if act.Healing > 0 {
				applied = true
				heal := math.Min(act.Healing, gamefx.EffectiveMaxHP(&player)-player.HP)
				player.HP += heal
				effectType = "heal"
				effectValue = heal
				logs = append(logs, fmt.Sprintf("%s phục hồi %.1f HP.", specs.Name, heal))
			}
			for _, block := range specs.Effects.Blocks {
				if block.Trigger != gameplayeffects.TriggerOnAction {
					continue
				}
				for _, op := range block.Operations {
					if op.Op != gameplayeffects.OpStatus {
						continue
					}
					duration := time.Duration(max(op.Turns, 1)) * time.Minute
					statusID, statusType := strings.ToUpper(op.StatusID), ""
					switch statusID {
					case "ATK_UP":
						statusType, effectType = "buff_atk", "buff_atk"
					case "DEF_UP":
						statusType, effectType = "buff_def", "buff_def"
					case "SPEED_UP", "HASTE":
						statusType, effectType = "buff_speed", "buff_speed"
					}
					if statusType != "" {
						player.Statuses.Add(statusType, specs.Name, duration, op.Value)
						effectValue = op.Value
						applied = true
						logs = append(logs, fmt.Sprintf("%s kích hoạt %s trong %.0f phút.", specs.Name, statusID, duration.Minutes()))
					}
				}
			}
			if applied && len(logs) == 0 {
				logs = append(logs, fmt.Sprintf("Kích hoạt %s thành công!", specs.Name))
			}
		}
	}

	// Flying (phi hành): toggle a timed status that hides the player from ground
	// monsters. Runs after the combat-action fallback so a Gu that also buffs or
	// damages still applies those effects; a pure-utility flying Gu (no damage)
	// works too.
	for _, block := range specs.Effects.Blocks {
		if block.Trigger != gameplayeffects.TriggerOnAction {
			continue
		}
		for _, op := range block.Operations {
			if op.Op != gameplayeffects.OpStatus {
				continue
			}
			statusID := strings.ToUpper(op.StatusID)
			if statusID != "FLYING" && statusID != "PHI_HANH" {
				continue
			}
			msg, flyingOn := toggleFlyingStatus(&player, specs.Name, targetGu.Rank, op.Turns)
			if flyingOn {
				effectType = "fly_on"
			} else {
				effectType = "fly_off"
			}
			logs = append(logs, msg)
			applied = true
			cdDuration = 2 * time.Second
		}
	}

	if !applied {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này không có kỹ năng chủ động (Active) để kích hoạt!"})
		return
	}

	// Monster casts use a stable runtime identity. Coordinate targeting remains
	// available only for utility/portal Gu for backwards compatibility.
	var targetDefeated bool
	var targetHP, targetMaxHP float64
	var targetName string
	var damageDealt float64

	if req.Target != nil && req.Target.Kind == "monster" {
		if req.CommandID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "command_id is required for monster casts"})
			return
		}
		targetRegion := player.CurrentRegion
		if req.Target.RegionID != "" {
			targetRegion = req.Target.RegionID
		}
		monster, exists, getErr := cache.GetWorldMonster(c.Request.Context(), targetRegion, req.Target.ID)
		if (!exists || getErr != nil) && targetRegion == player.CurrentRegion {
			for _, scopeID := range regionScopeIDs(player.CurrentRegion) {
				if scopeID == targetRegion {
					continue
				}
				if m, ok, _ := cache.GetWorldMonster(c.Request.Context(), scopeID, req.Target.ID); ok {
					monster, exists, getErr = m, ok, nil
					break
				}
			}
		}
		if getErr != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Monster runtime unavailable"})
			return
		}
		if !exists || monster.State == MonsterStateDefeated {
			c.JSON(http.StatusConflict, gin.H{"error": "Monster is no longer active"})
			return
		}
		if !isRegionWithinScope(monster.RegionID, player.CurrentRegion) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Target is in another region"})
			return
		}
		if math.Hypot(float64(player.X)-monster.X, float64(player.Y)-monster.Y) > 12 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Target is out of cast range"})
			return
		}
		stats := CalculatePlayerStats(&player)
		attacker := models.CombatantFromPlayer(&player, stats)
		attackAction, buildErr := buildSimActionFromGu(&attacker, &specs, "GU_SKILL")
		if buildErr != nil || attackAction.Damage <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này không có kỹ năng sát thương để tấn công quái vật."})
			return
		}
		rawDmg := attackAction.Damage
		damageDealt = simulator.ResolveBaseDamage(rawDmg, monster.BaseDefense)
		outcome, updated, hitErr := cache.HitWorldMonster(c.Request.Context(), monster.RegionID, monster.InstanceID, player.ID.String()+":"+req.CommandID, monster.Revision, damageDealt, time.Now().UTC(), monsterRespawnDelay(monster.RespawnSeconds), monsterTTL(monster.RespawnSeconds))
		if hitErr != nil {
			logger.S.Errorf("[WorldGuCast] failed to update monster %s in region %s: %v", monster.InstanceID, monster.RegionID, hitErr)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Cannot update monster state"})
			return
		}
		if outcome == cache.WorldMonsterRevisionConflict {
			worldMonsterMetrics.conflicts.Add(1)
			c.JSON(http.StatusConflict, gin.H{"error": "Monster state changed; retry with the latest snapshot"})
			return
		}
		if outcome == cache.WorldMonsterAlreadyDefeated {
			worldMonsterMetrics.preventedDoubleKills.Add(1)
			c.JSON(http.StatusConflict, gin.H{"error": "Command already processed or monster defeated"})
			return
		}
		targetHP, targetMaxHP = updated.HP, updated.MaxHP
		targetName = models.MonsterRegistry[updated.MonsterTemplateID].Name
		targetDefeated = outcome == cache.WorldMonsterFirstKill
		effectType = "damage"
		if targetDefeated {
			worldMonsterMetrics.kills.Add(1)
			stonesGain := int64(30)
			player.PrimevalStones += float64(stonesGain)
			logs = append(logs, fmt.Sprintf("💥 %s gây %.0f sát thương, tiêu diệt %s! Nhận +%d Nguyên Thạch.", specs.Name, damageDealt, targetName, stonesGain))
			broadcastWorldCombat(updated, "monster_defeated", map[string]any{"killer_id": player.ID.String(), "damage": damageDealt})
			if region, x, y, procedural := proceduralMonsterCell(updated.SpawnID); procedural {
				_ = cache.SetCellCleared(region, x, y, player.ID.String(), clearedCellCooldown)
				_ = cache.BroadcastMonsterKilled(x, y, targetName, region)
			} else if region, x, y, boss := mapCellBossSpawn(updated.SpawnID); boss {
				if err := MarkPersistentCellCleared(tx, player.ID, region, x, y); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Cannot clear defeated boss cell"})
					return
				}
			} else {
				// Database-backed world spawns can also overlap a static generated
				// cell. Clear and broadcast that representation so it does not
				// reappear as soon as the runtime actor is removed.
				clearDefeatedWorldMonsterCell(updated, player.ID.String())
			}
		} else {
			logs = append(logs, fmt.Sprintf("⚡ %s gây %.0f sát thương lên %s (HP %.0f/%.0f).", specs.Name, damageDealt, targetName, targetHP, targetMaxHP))
			broadcastWorldCombat(updated, "entity_damaged", map[string]any{"source_id": player.ID.String(), "damage": damageDealt, "hp": targetHP, "max_hp": targetMaxHP})
		}
	}

	player.CurrentMana -= cost
	player.Statuses.Add(cdID, "cooldown", cdDuration, 0)

	targetGu.Hunger -= 1
	if targetGu.Hunger < 0 {
		targetGu.Hunger = 0
	}
	if errGu := persistGuHungerTx(c.Request.Context(), tx, player.ID, targetGu); errGu != nil {
		logger.S.Errorf("[UseGuOnMap] failed to persist hunger for player=%s gu=%s: %v", player.ID, targetGu.InstanceID, errGu)
		if worker.IsHungerTransientError(errGu) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Hệ thống đang bận, đạo hữu vui lòng thử lại sau ít giây."})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật độ no Cổ trùng"})
		return
	}

	statusesJSON, _ := json.Marshal(player.Statuses)
	clearedJSON, _ := json.Marshal(player.ClearedCells)

	_, err = tx.Exec(`UPDATE players SET current_mana = $1, hp = $2, x = $3, y = $4, current_region = $5, statuses = $6, cleared_cells = $7, primeval_stones = $8 WHERE id = $9`,
		player.CurrentMana, player.HP, player.X, player.Y, player.CurrentRegion, statusesJSON, clearedJSON, player.PrimevalStones, player.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu trạng thái"})
		return
	}

	// A successful teleport drags the party along (leader only). Non-teleport
	// Gu casts must never relocate party members to the caster's cell.
	var syncedMembers []SyncedPartyMember
	if teleported {
		syncedMembers = SyncPartyMembersLocation(tx, &player)
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	syncPlayerFlying(&player)

	if teleported {
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
	}

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"player":               player,
		"message":              strings.Join(logs, " "),
		"logs":                 logs,
		"effect_type":          effectType,
		"effect_value":         effectValue,
		"cooldown_seconds":     cdDuration.Seconds(),
		"damage_dealt":         damageDealt,
		"target_defeated":      targetDefeated,
		"target_hp":            targetHP,
		"target_max_hp":        targetMaxHP,
		"target_name":          targetName,
		"target_x":             req.TargetX,
		"target_y":             req.TargetY,
		"synced_party_members": syncedMembers,
	})
}
