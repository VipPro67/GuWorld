package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type WorldCastTarget struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type UseGuOnMapRequest struct {
	CommandID    string           `json:"command_id"`
	GuInstanceID string           `json:"gu_instance_id" binding:"required"`
	Target       *WorldCastTarget `json:"target,omitempty"`
	TargetX      int              `json:"target_x"`
	TargetY      int              `json:"target_y"`
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
	if _, hasIgnoreRange := models.GuSpecCapability(specs, "IGNORE_MOVEMENT_RANGE"); hasIgnoreRange {
		hasPortalCreate = true
	}
	if specs.Effects.Active != nil {
		if specs.Effects.Active.PortalCreate {
			hasPortalCreate = true
		}
		if specs.Effects.Active.RandomPortal {
			hasRandomPortal = true
		}
	}

	cost := gamefx.EnergyCost(&player, &specs)
	if hasPortalCreate || hasRandomPortal {
		cost = GetPortalManaCost(&player, &specs)
	}
	logs := []string{}
	var cdDuration time.Duration = 10 * time.Second // default CD
	applied := false
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
		// We can teleport to TargetX, TargetY if within bounds
		if inWorld(req.TargetX, req.TargetY) {
			oldX := player.X
			oldY := player.Y
			oldRegion := player.CurrentRegion
			player.X = req.TargetX
			player.Y = req.TargetY
			player.CurrentRegion = DeriveRegionAt(player.X, player.Y)
			teleportMsg := fmt.Sprintf("🌀 Cổ trùng %s xé rách không gian, đưa %s dịch chuyển từ (%d, %d) [%s] đến (%d, %d) [%s].", specs.Name, player.Username, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion)
			if debuffMsg := CheckAndApplyRegionalWallDebuff(&player, oldX, oldY, player.X, player.Y); debuffMsg != "" {
				teleportMsg += " " + debuffMsg
			}
			logs = append(logs, teleportMsg)
			applied = true
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
		player.X = rand.Intn(WorldWidth)
		player.Y = rand.Intn(WorldHeight)
		player.CurrentRegion = DeriveRegionAt(player.X, player.Y)
		teleportMsg := fmt.Sprintf("🌀 Cổ trùng %s xé rách không gian hỗn loạn, đưa %s dịch chuyển ngẫu nhiên từ (%d, %d) [%s] đến (%d, %d) [%s].", specs.Name, player.Username, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion)
		if debuffMsg := CheckAndApplyRegionalWallDebuff(&player, oldX, oldY, player.X, player.Y); debuffMsg != "" {
			teleportMsg += " " + debuffMsg
		}
		logs = append(logs, teleportMsg)
		applied = true
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

	// Fallback for general Active Gu effects if no map-specific condition matched
	if !applied && specs.Effects.Active != nil {
		stats := CalculatePlayerStats(&player)
		attacker := models.CombatantFromPlayer(&player, stats)
		ctx := &CombatContext{
			Attacker: &attacker,
		}
		act, err := BuildCombatAction(ctx, specs)
		if err == nil {
			applied = true
			cdDuration = 30 * time.Second
			effectType = "utility"
			if act.Dmg > 0 {
				effectValue = act.Dmg
				logs = append(logs, fmt.Sprintf("Kích hoạt %s thôi động sát chiêu (Uy lực %.0f)!", specs.Name, act.Dmg))
			} else if act.Def > 0 {
				effectValue = act.Def
				player.Statuses.Add("virtual_shield", "world_map", 5*time.Minute, act.Def)
				logs = append(logs, fmt.Sprintf("Kích hoạt %s tạo lớp ảo thuẫn hộ thể (%.0f) duy trì 5 phút.", specs.Name, act.Def))
			}
			if len(act.Logs) > 0 {
				logs = append(logs, act.Logs...)
			}
			if len(logs) == 0 {
				logs = append(logs, fmt.Sprintf("Kích hoạt %s thành công!", specs.Name))
			}
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
		monster, exists, getErr := cache.GetWorldMonster(c.Request.Context(), player.CurrentRegion, req.Target.ID)
		if getErr != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Monster runtime unavailable"})
			return
		}
		if !exists || monster.State == MonsterStateDefeated {
			c.JSON(http.StatusConflict, gin.H{"error": "Monster is no longer active"})
			return
		}
		if monster.RegionID != player.CurrentRegion {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Target is in another region"})
			return
		}
		if math.Hypot(float64(player.X)-monster.X, float64(player.Y)-monster.Y) > 12 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Target is out of cast range"})
			return
		}
		rawDmg := effectValue
		if rawDmg <= 0 {
			rawDmg = math.Max(20, CalculatePlayerStats(&player)["attack"]*0.8)
		}
		damageDealt = simulator.ResolveBaseDamage(rawDmg, monster.BaseDefense)
		outcome, updated, hitErr := cache.HitWorldMonster(c.Request.Context(), monster.RegionID, monster.InstanceID, player.ID.String()+":"+req.CommandID, monster.Revision, damageDealt, time.Now().UTC(), time.Duration(monster.RespawnSeconds)*time.Second, monsterTTL(monster.RespawnSeconds))
		if hitErr != nil {
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
	if _, errGu := tx.Exec(`UPDATE player_gu SET hunger = $1, updated_at = NOW() WHERE id = $2`, targetGu.Hunger, targetGu.InstanceID); errGu != nil {
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

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message":          strings.Join(logs, " "),
		"logs":             logs,
		"effect_type":      effectType,
		"effect_value":     effectValue,
		"cooldown_seconds": cdDuration.Seconds(),
		"damage_dealt":     damageDealt,
		"target_defeated":  targetDefeated,
		"target_hp":        targetHP,
		"target_max_hp":    targetMaxHP,
		"target_name":      targetName,
		"target_x":         req.TargetX,
		"target_y":         req.TargetY,
	})
}

type UseKillerMoveOnMapRequest struct {
	KillerMoveID string `json:"killer_move_id" binding:"required"`
	TargetX      int    `json:"target_x"`
	TargetY      int    `json:"target_y"`
}

func UseKillerMoveOnMap(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req UseKillerMoveOnMapRequest
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

	move, slots, err := loadKillerMoveTx(tx, playerID, req.KillerMoveID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy sát chiêu"})
		return
	}

	// Calculate stats
	supportIDs := []string{}
	for _, slot := range slots {
		if slot.SlotType == models.KillerMoveSlotSupport {
			supportIDs = append(supportIDs, slot.PlayerGuID)
		}
	}
	preview, err := calculateKillerMoveStatsForPlayer(CalculateKillerMoveRequest{CorePlayerGuID: move.CorePlayerGuID, SupportPlayerGuIDs: supportIDs}, &player, tx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cdID := "cd_km_" + req.KillerMoveID
	if player.Statuses.Has(cdID) {
		activeStatuses := player.Statuses.Active()
		for _, st := range activeStatuses {
			if st.ID == cdID {
				remains := time.Until(*st.ExpiresAt).Seconds()
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Sát chiêu đang trong thời gian hồi (%.0f giây)", remains)})
				return
			}
		}
	}

	allGuIDs := append([]string{move.CorePlayerGuID}, supportIDs...)
	for _, gID := range allGuIDs {
		for _, gu := range player.GuInventory {
			if gu.InstanceID == gID {
				if gu.Hunger <= 0 {
					spec, _ := models.GetGu(gu.GuID, gu.Rank)
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cổ trùng %s trong sát chiêu đã kiệt sức (độ no = 0)!", spec.Name)})
					return
				}
			}
		}
	}

	if player.CurrentMana < preview.TotalManaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}
	player.CurrentMana -= preview.TotalManaCost

	for _, gID := range allGuIDs {
		for idx := range player.GuInventory {
			if player.GuInventory[idx].InstanceID == gID {
				player.GuInventory[idx].Hunger -= 1
				if player.GuInventory[idx].Hunger < 0 {
					player.GuInventory[idx].Hunger = 0
				}
				break
			}
		}
		if _, errExec := tx.Exec(`UPDATE player_gu SET hunger = GREATEST(hunger - 1, 0), updated_at = NOW() WHERE id = $1`, gID); errExec != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật độ no Cổ trùng"})
			return
		}
	}

	succeeded := math.Max(0.0, float64(preview.SuccessRate)) >= (100.0 * 0.5)
	logs := []string{}

	if !succeeded {
		backlashDamage := preview.TotalManaCost * 0.5
		player.HP -= backlashDamage
		logs = append(logs, fmt.Sprintf("Sát chiêu thất bại, phản phệ %.1f sát thương.", backlashDamage))
	} else {
		applied := false
		var cdDuration time.Duration = time.Duration(preview.Cooldown) * 10 * time.Second

		// Interpret the effects
		for k, v := range preview.Effects {
			if k == "mana_regen" {
				player.CurrentMana += v
				if player.CurrentMana > player.MaxMana {
					player.CurrentMana = player.MaxMana
				}
				logs = append(logs, fmt.Sprintf("Sát chiêu hồi phục %.1f Chân Nguyên.", v))
				applied = true
			}
			if k == "heal_hp" || k == "regen" {
				player.HP += v
				if player.HP > gamefx.EffectiveMaxHP(&player) {
					player.HP = gamefx.EffectiveMaxHP(&player)
				}
				logs = append(logs, fmt.Sprintf("Sát chiêu hồi phục %.1f HP.", v))
				applied = true
			}
			if k == "portal_create" {
				if inWorld(req.TargetX, req.TargetY) {
					oldX := player.X
					oldY := player.Y
					oldRegion := player.CurrentRegion
					player.X = req.TargetX
					player.Y = req.TargetY
					player.CurrentRegion = DeriveRegionAt(player.X, player.Y)
					teleportMsg := fmt.Sprintf("🌀 Sát chiêu %s xé rách không gian, đưa %s dịch chuyển từ (%d, %d) [%s] đến (%d, %d) [%s].", move.Name, player.Username, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion)
					if debuffMsg := CheckAndApplyRegionalWallDebuff(&player, oldX, oldY, player.X, player.Y); debuffMsg != "" {
						teleportMsg += " " + debuffMsg
					}
					logs = append(logs, teleportMsg)
					applied = true
					auditlog.Log(tx, auditlog.Entry{
						PlayerID:       &player.ID,
						Action:         "killer_move.portal_create",
						EntityType:     "killer_move",
						EntityID:       move.ID,
						RequestSummary: fmt.Sprintf("Dịch chuyển sát chiêu %s từ (%d, %d) [%s] đến (%d, %d) [%s]", move.Name, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion),
						Metadata: map[string]interface{}{
							"move_id":     move.ID,
							"move_name":   move.Name,
							"from_x":      oldX,
							"from_y":      oldY,
							"from_region": oldRegion,
							"to_x":        player.X,
							"to_y":        player.Y,
							"to_region":   player.CurrentRegion,
						},
						IPAddress: c.ClientIP(),
						Success:   true,
					})
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ không hợp lệ!"})
					return
				}
			}
			if k == "random_portal" {
				oldX := player.X
				oldY := player.Y
				oldRegion := player.CurrentRegion
				player.X = rand.Intn(WorldWidth)
				player.Y = rand.Intn(WorldHeight)
				player.CurrentRegion = DeriveRegionAt(player.X, player.Y)
				teleportMsg := fmt.Sprintf("🌀 Sát chiêu %s xé rách không gian hỗn loạn, đưa %s dịch chuyển ngẫu nhiên từ (%d, %d) [%s] đến (%d, %d) [%s].", move.Name, player.Username, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion)
				if debuffMsg := CheckAndApplyRegionalWallDebuff(&player, oldX, oldY, player.X, player.Y); debuffMsg != "" {
					teleportMsg += " " + debuffMsg
				}
				logs = append(logs, teleportMsg)
				applied = true
				auditlog.Log(tx, auditlog.Entry{
					PlayerID:       &player.ID,
					Action:         "killer_move.random_portal",
					EntityType:     "killer_move",
					EntityID:       move.ID,
					RequestSummary: fmt.Sprintf("Dịch chuyển ngẫu nhiên sát chiêu %s từ (%d, %d) [%s] đến (%d, %d) [%s]", move.Name, oldX, oldY, oldRegion, player.X, player.Y, player.CurrentRegion),
					Metadata: map[string]interface{}{
						"move_id":     move.ID,
						"move_name":   move.Name,
						"from_x":      oldX,
						"from_y":      oldY,
						"from_region": oldRegion,
						"to_x":        player.X,
						"to_y":        player.Y,
						"to_region":   player.CurrentRegion,
					},
					IPAddress: c.ClientIP(),
					Success:   true,
				})
			}
		}

		if !applied {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Sát chiêu này không có hiệu ứng áp dụng trên World Map!"})
			return
		}
		player.Statuses.Add(cdID, "cooldown", cdDuration, 0)
	}

	// encoding/json missing? Handled.
	statusesJSON, _ := json.Marshal(player.Statuses)
	_, err = tx.Exec(`UPDATE players SET current_mana = $1, hp = $2, x = $3, y = $4, current_region = $5, statuses = $6 WHERE id = $7`,
		player.CurrentMana, player.HP, player.X, player.Y, player.CurrentRegion, statusesJSON, player.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu trạng thái"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction error"})
		return
	}

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": strings.Join(logs, " "),
		"logs":    logs,
	})
}
