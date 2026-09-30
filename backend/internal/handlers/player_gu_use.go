package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type UseGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// UseConsumableGu activates a consumable Gu bug's one-time effect and removes it
func UseConsumableGu(c *gin.Context) {
	var req UseGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	targetIdx := -1
	for idx, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			targetIdx = idx
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này trong túi đồ"})
		return
	}

	// Capture all needed values from targetGu BEFORE any slice mutation.
	// RemoveGuFromInventory uses append(inventory[:i], inventory[i+1:]...) which
	// shifts elements in-place, invalidating the &player.GuInventory[targetIdx]
	// pointer. Reading targetGu.InstanceID after that would return the wrong ID.
	instanceIDToDelete := player.GuInventory[targetIdx].InstanceID
	guRank := player.GuInventory[targetIdx].Rank
	guID := player.GuInventory[targetIdx].GuID
	guHunger := player.GuInventory[targetIdx].Hunger

	specs, found := models.GetGu(guID, guRank)
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thông tin Cổ trùng trong thư viện"})
		return
	}

	if specs.Type != "Consumable" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này không thuộc loại Tiêu Hao, không thể sử dụng trực tiếp!"})
		return
	}

	// Use a value copy for validation and effect resolution to avoid stale-pointer issues.
	targetGuCopy := player.GuInventory[targetIdx]
	if err := ValidateGuUsable(&player, &targetGuCopy, &specs, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cost := gamefx.EnergyCost(&player, &specs)
	player.CurrentMana -= cost

	// Single dispatcher: every ON_CONSUME mechanic shares HandleGuEvent.
	// ON_USE is also accepted so info/record consumables (RECORD_MESSAGE)
	// work through this endpoint instead of 400ing.
	var logs []string
	for _, trigger := range []string{"ON_CONSUME", "ON_USE"} {
		consumeMsg, err := HandleGuEvent(&player, specs, targetGuCopy, trigger)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if consumeMsg != "" {
			logs = append(logs, consumeMsg)
		}
	}

	if len(logs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hiệu ứng của Cổ trùng tiêu hao này chưa được định nghĩa hoặc không kích hoạt thành công!"})
		return
	}

	logMsg := strings.Join(logs, " ")

	qiGained := 0.0

	// Cổ trùng tiêu hao: cộng Path Mastery (Chân Ý)
	if player.PathMasteries == nil {
		player.PathMasteries = make(models.PathMasteriesMap)
	}
	if len(specs.Path) > 0 {
		cleanPath := strings.TrimSuffix(strings.ToLower(specs.Path[0]), " path")
		currentLevel := gamefx.PathMasteryLevel(player.PathMasteries[cleanPath])
		pathGain := gamefx.CalculatePathMasteryGain(guRank, currentLevel, gamefx.PathMasteryGuUseBaseGain(guRank))
		if pathGain > 0 {
			player.PathMasteries[cleanPath] += pathGain
			newLevel := gamefx.PathMasteryLevel(player.PathMasteries[cleanPath])
			logMsg += fmt.Sprintf(" Ngộ Đạo tăng tiến! +%d điểm Chân Ý lưu phái %s (Cảnh giới: %s).", pathGain, strings.Title(cleanPath), gamefx.PathMasteryLevelName(newLevel))
		}
	}
	// SACRIFICE_APTITUDE doubles as the keep-the-Gu flag: a sacrifice drains
	// hunger instead of consuming the instance. Any other ON_CONSUME deletes it.
	var reduceHunger bool
	var hungerCost int = 20
	_, reduceHunger = models.GuNativeMechanic(specs, "SACRIFICE_APTITUDE")

	if reduceHunger {
		if guHunger < hungerCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng quá đói, không đủ độ no để hiến tế!"})
			return
		}
		targetGuCopy.Hunger -= hungerCost
		if err := db.UpdatePlayerGu(tx, targetGuCopy); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Gu hunger"})
			return
		}
		// Sync in-memory inventory so the SSE snapshot reflects the updated hunger.
		player.GuInventory[targetIdx].Hunger = targetGuCopy.Hunger
		logMsg += fmt.Sprintf(" Cổ trùng bị suy yếu, độ no giảm %d.", hungerCost)
	} else {
		// Delete from inventory — use instanceIDToDelete (captured before slice mutation).
		// RemoveGuFromInventory shifts the backing array, so any pointer into
		// player.GuInventory taken before this call is no longer valid.
		RemoveGuFromInventory(&player, instanceIDToDelete)

		if err := db.RemovePlayerGuForPlayer(tx, player.ID.String(), instanceIDToDelete); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state after using Gu"})
			fmt.Println("Failed to update player state after using Gu:", err)
			return
		}
	}

	_, err = tx.Exec(`
		UPDATE players 
		SET sub_rank = $1, rank = $2, primeval_stones = $3, max_mana = $4, current_mana = $5, equipped_gus = $6, wall_progress = $7, hp = $8, lifespan = $9, aptitude_percent = $10, immortal_stones = $11, aperture_stress = $12, human_qi = $13, max_soul = $14, current_thought = $15, path_masteries = $16, time_speed_multiplier = $17, aptitude_type = $18, soul_hp = $19, max_thought = $20
		WHERE id = $21`,
		player.SubRank, player.Rank, player.PrimevalStones, player.MaxMana, player.CurrentMana, player.EquippedGus, player.WallProgress, player.HP, player.Lifespan, player.AptitudePercent, player.ImmortalStones, player.ApertureStress, player.HumanQi, player.MaxSoul, player.CurrentThought, player.PathMasteries, player.TimeSpeedMultiplier, player.AptitudeType, player.SoulHP, player.MaxThought, player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state after using Gu"})
		fmt.Println("Failed to update player state after using Gu:", err)
		return
	}

	if player.Rank >= 6 {
		_, err = tx.Exec(`
			UPDATE immortal_apertures 
			SET dao_marks = $1 
			WHERE player_id = $2`,
			player.DaoMarks, player.ID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update aperture Dao Marks: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, CombatStatsDomain)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "gu.use_consumable",
		EntityType:     "gu_instance",
		EntityID:       req.GuInstanceID,
		RequestSummary: fmt.Sprintf("Sử dụng Cổ trùng tiêu hao %s (Rank %d): %s", specs.Name, specs.Rank, logMsg),
		Metadata: map[string]interface{}{
			"gu_instance_id": req.GuInstanceID,
			"gu_id":          specs.ID,
			"gu_name":        specs.Name,
			"rank":           specs.Rank,
			"reduce_hunger":  reduceHunger,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionGuOnly(c, http.StatusOK, player, gin.H{
		"message": logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

// UseSupportGuRequest represents request payload to use a support Gu out of combat
type UseSupportGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// UseSupportGu activates an out-of-combat support Gu bug's effect
func UseSupportGu(c *gin.Context) {
	var req UseSupportGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	targetIdx := -1
	for idx, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			targetIdx = idx
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này trong túi đồ"})
		return
	}

	// Capture scalar values from the target slot before any possible slice mutation.
	// UseSupportGu does not currently remove the Gu from the slice, but using value
	// copies is defensive and consistent with UseConsumableGu.
	targetGuCopy := player.GuInventory[targetIdx]
	guID := targetGuCopy.GuID
	guRank := targetGuCopy.Rank

	specs, found := models.GetGu(guID, guRank)
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thông tin Cổ trùng trong thư viện"})
		return
	}

	if specs.Type != "Support" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này không thuộc loại Hỗ Trợ, không thể sử dụng trực tiếp ngoài combat!"})
		return
	}

	if err := ValidateGuUsable(&player, &targetGuCopy, &specs, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cost := gamefx.EnergyCost(&player, &specs)
	player.CurrentMana -= cost

	// Support endpoint dispatches ON_CONSUME and ON_USE (e.g. RECORD_MESSAGE);
	// ON_TICK Gus must not be used here — they accrue via workers/lazy regen.
	var logs []string
	for _, trigger := range []string{"ON_CONSUME", "ON_USE"} {
		supportMsg, err := HandleGuEvent(&player, specs, targetGuCopy, trigger)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if supportMsg != "" {
			logs = append(logs, supportMsg)
		}
	}
	// support.heal_flat is gone: out-of-combat healing reads the same
	// active.heal_flat the combat engine uses, capped at EffectiveMaxHP.
	heal := 0.0
	if compiled := models.CompiledGuEffects(specs); compiled != nil {
		for _, operation := range compiled.CompileCombatAction().Operations {
			if operation.Type == gameplayeffects.OpHeal && operation.SubType == "FLAT" {
				heal += operation.Value
			}
		}
	}
	if heal > 0 {
		before := player.HP
		player.HP += heal
		if maxHP := gamefx.EffectiveMaxHP(&player); player.HP > maxHP {
			player.HP = maxHP
		}
		if healed := player.HP - before; healed > 0 {
			logs = append(logs, fmt.Sprintf("%s tấu lên khúc nhạc chữa lành, hồi phục %.0f HP!", specs.Name, healed))
		}
	}

	if len(logs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng hỗ trợ này không có hiệu ứng ngoài chiến đấu!"})
		return
	}

	targetGuCopy.Hunger -= 1
	if targetGuCopy.Hunger < 0 {
		targetGuCopy.Hunger = 0
	}
	if _, errGu := tx.Exec(`UPDATE player_gu SET hunger = $1, updated_at = NOW() WHERE id = $2`, targetGuCopy.Hunger, targetGuCopy.InstanceID); errGu != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật độ no Cổ trùng"})
		return
	}
	// Sync in-memory inventory so the SSE snapshot reflects the updated hunger.
	player.GuInventory[targetIdx].Hunger = targetGuCopy.Hunger

	qiGained := 0.0

	logMsg := strings.Join(logs, " ")
	// Save to DB — support Gu stays in inventory, no need to rewrite inventory.
	// Column list mirrors UseConsumableGu so no effect write is silently lost.
	_, err = tx.Exec(`
		UPDATE players 
		SET sub_rank = $1, rank = $2, primeval_stones = $3, max_mana = $4, current_mana = $5, equipped_gus = $6, wall_progress = $7, hp = $8, lifespan = $9, aptitude_percent = $10, immortal_stones = $11, aperture_stress = $12, human_qi = $13, max_soul = $14, current_thought = $15, path_masteries = $16, time_speed_multiplier = $17, aptitude_type = $18, soul_hp = $19, max_thought = $20
		WHERE id = $21`,
		player.SubRank, player.Rank, player.PrimevalStones, player.MaxMana, player.CurrentMana, player.EquippedGus, player.WallProgress, player.HP, player.Lifespan, player.AptitudePercent, player.ImmortalStones, player.ApertureStress, player.HumanQi, player.MaxSoul, player.CurrentThought, player.PathMasteries, player.TimeSpeedMultiplier, player.AptitudeType, player.SoulHP, player.MaxThought, player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state after using support Gu"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	respondActionGuOnly(c, http.StatusOK, player, gin.H{
		"message": logMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
	})
}

// UpdateProfileSettingsRequest represents settings data to update
func aggregateChestSpecialMessages(specialMessages []string) []string {
	var stones int
	materials := make(map[string]int)
	var others []string
	var chestNames []string

	for _, msg := range specialMessages {
		lines := strings.Split(msg, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// Skip the "Mở <Item> thành công!" or "Mở <Item> thất bại" line as it's redundant
			if strings.HasPrefix(line, "Mở ") && strings.HasSuffix(line, "thành công!") {
				chestName := strings.TrimSuffix(strings.TrimPrefix(line, "Mở "), " thành công!")
				chestNames = append(chestNames, chestName)
				continue
			}

			// 1. Stones pattern: "- 💎 Nhận được %d Nguyên Thạch"
			var amt int
			if _, err := fmt.Sscanf(line, "- 💎 Nhận được %d Nguyên Thạch", &amt); err == nil {
				stones += amt
				continue
			}

			// 2. Material/item pattern: "- Nhận được %dx %s"
			if strings.HasPrefix(line, "- Nhận được ") {
				rest := strings.TrimPrefix(line, "- Nhận được ")
				if idx := strings.Index(rest, "x "); idx != -1 {
					qtyStr := rest[:idx]
					name := rest[idx+2:]
					var q int
					if _, err := fmt.Sscanf(qtyStr, "%d", &q); err == nil {
						materials[name] += q
						continue
					}
				}
			}

			// If not matched, keep it as is
			others = append(others, line)
		}
	}

	var result []string
	if len(chestNames) > 0 {
		counts := make(map[string]int)
		for _, name := range chestNames {
			counts[name]++
		}
		var headers []string
		for name, count := range counts {
			headers = append(headers, fmt.Sprintf("Mở %d lần %s thành công!", count, name))
		}
		result = append(result, headers...)
	}

	if stones > 0 {
		result = append(result, fmt.Sprintf("- 💎 Nhận được %d Nguyên Thạch", stones))
	}

	for name, qty := range materials {
		result = append(result, fmt.Sprintf("- Nhận được %dx %s", qty, name))
	}

	result = append(result, others...)

	return result
}
