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
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type UseGuOnMapRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
	TargetX      int    `json:"target_x"`
	TargetY      int    `json:"target_y"`
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

	cost := gamefx.EnergyCost(&player, &specs)
	if specs.Effects.Active != nil && (specs.Effects.Active.PortalCreate || specs.Effects.Active.RandomPortal) {
		cost = GetPortalManaCost(&player, &specs)
	}
	logs := []string{}
	var cdDuration time.Duration = 10 * time.Second // default CD
	applied := false
	effectType := "utility"
	effectValue := 0.0

	// Apply effects from Active (Combat mapping to World Map)
	if specs.Effects.Active != nil {
		active := specs.Effects.Active

		if active.HealFlat > 0 {
			maxHP := gamefx.EffectiveMaxHP(&player)
			healVal := active.HealFlat
			player.HP += healVal
			if player.HP > maxHP {
				player.HP = maxHP
			}
			logs = append(logs, fmt.Sprintf("%s điều trị thương thế, phục hồi %.0f Sinh Lực.", specs.Name, healVal))
			applied = true
			cdDuration = 45 * time.Second
			effectType = "heal"
			effectValue = healVal
		}
		if active.ManaRegen > 0 {
			player.CurrentMana += active.ManaRegen
			if player.CurrentMana > player.MaxMana {
				player.CurrentMana = player.MaxMana
			}
			logs = append(logs, fmt.Sprintf("%s hồi phục %.1f Chân Nguyên.", specs.Name, active.ManaRegen))
			applied = true
			cdDuration = 30 * time.Second
			effectType = "mana_regen"
			effectValue = active.ManaRegen
		}
		if active.PortalCreate {
			// We can teleport to TargetX, TargetY if within bounds
			if inWorld(req.TargetX, req.TargetY) {
				player.X = req.TargetX
				player.Y = req.TargetY
				player.CurrentRegion = DeriveRegionAt(player.X, player.Y)
				logs = append(logs, fmt.Sprintf("%s xé rách không gian, dịch chuyển đến (%d, %d).", specs.Name, player.X, player.Y))
				applied = true
				cdDuration = 120 * time.Second
				effectType = "teleport"
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ không hợp lệ!"})
				return
			}
		}
		if active.RandomPortal {
			player.X = rand.Intn(WorldWidth)
			player.Y = rand.Intn(WorldHeight)
			player.CurrentRegion = DeriveRegionAt(player.X, player.Y)
			logs = append(logs, fmt.Sprintf("%s xé rách không gian hỗn loạn, dịch chuyển ngẫu nhiên đến (%d, %d).", specs.Name, player.X, player.Y))
			applied = true
			cdDuration = 120 * time.Second
			effectType = "teleport"
		}
		// HOT tick -> heal
		if active.ApplyStatus == "REGEN" {
			healVal := active.StatusTickDamage * float64(active.StatusDurationTicks)
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
		if active.ApplyStatus == "ATK_UP" {
			buffValue := active.StatusTickDamage
			if buffValue == 0 {
				buffValue = 50 // fallback
			}
			duration := time.Duration(active.StatusDurationTicks) * time.Minute
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
		if active.ApplyStatus == "DEF_UP" {
			buffValue := active.StatusTickDamage
			if buffValue == 0 {
				buffValue = 50 // fallback
			}
			duration := time.Duration(active.StatusDurationTicks) * time.Minute
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
		if active.ApplyStatus == "SPEED_UP" {
			buffValue := active.StatusTickDamage
			if buffValue == 0 {
				buffValue = 20
			}
			duration := time.Duration(active.StatusDurationTicks) * time.Minute
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

	// Apply Other Mechanics for World Map
	if specs.Effects.Other != nil {
		for _, mechanic := range specs.Effects.Other {
			if mechanic.EventTrigger == "ON_WORLD_MAP" {
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
		}
	}

	// Fallback for general Active Gu effects if no map-specific condition matched
	if !applied && specs.Effects.Active != nil {
		stats := CalculatePlayerStats(&player)
		ctx := &CombatContext{
			Attacker: &models.Combatant{
				MaxHP:      gamefx.EffectiveMaxHP(&player),
				BaseAttack: stats["attack"],
			},
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

	player.CurrentMana -= cost
	player.Statuses.Add(cdID, "cooldown", cdDuration, 0)

	statusesJSON, _ := json.Marshal(player.Statuses)
	_, err = tx.Exec(`UPDATE players SET current_mana = $1, hp = $2, x = $3, y = $4, current_region = $5, statuses = $6 WHERE id = $7`,
		player.CurrentMana, player.HP, player.X, player.Y, player.CurrentRegion, statusesJSON, player.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu trạng thái"})
		return
	}
	
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message":          strings.Join(logs, " "),
		"logs":             logs,
		"effect_type":      effectType,
		"effect_value":     effectValue,
		"cooldown_seconds": cdDuration.Seconds(),
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

	if player.CurrentMana < preview.TotalManaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}
	player.CurrentMana -= preview.TotalManaCost

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
					player.X = req.TargetX
					player.Y = req.TargetY
					player.CurrentRegion = DeriveRegionAt(player.X, player.Y)
					logs = append(logs, fmt.Sprintf("Sát chiêu xé rách không gian, dịch chuyển đến (%d, %d).", player.X, player.Y))
					applied = true
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ không hợp lệ!"})
					return
				}
			}
			if k == "random_portal" {
				player.X = rand.Intn(WorldWidth)
				player.Y = rand.Intn(WorldHeight)
				player.CurrentRegion = DeriveRegionAt(player.X, player.Y)
				logs = append(logs, fmt.Sprintf("Sát chiêu xé rách không gian hỗn loạn, dịch chuyển ngẫu nhiên đến (%d, %d).", player.X, player.Y))
				applied = true
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

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": strings.Join(logs, " "),
		"logs":    logs,
	})
}
