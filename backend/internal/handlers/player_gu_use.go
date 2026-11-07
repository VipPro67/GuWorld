package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type UseGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// UseConsumableGu activates a consumable Gu bug\x27s one-time effect and removes it
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

	var logMsg string
	var specs models.GuDictionaryItem
	var reduceHunger bool
	var realtimeBaseHP float64
	var realtimeActive bool

	player, err := MutatePlayer(authID, PlayerMutationOptions{
		Scope:        playerstate.GuFull,
		EventDomains: []PlayerEventDomain{PlayerDomain, GuInventoryDomain, CombatStatsDomain},
	}, func(tx *sql.Tx, player *models.Player) error {
		realtimeBaseHP, realtimeActive = seedRealtimeHP(authID, player)

		targetIdx := -1
		for idx, gu := range player.GuInventory {
			if gu.InstanceID == req.GuInstanceID {
				targetIdx = idx
				break
			}
		}

		if targetIdx == -1 {
			return NewMutationError(http.StatusNotFound, "Không tìm thấy Cổ trùng này trong túi đồ")
		}

		// Capture all needed values from targetGu BEFORE any slice mutation.
		instanceIDToDelete := player.GuInventory[targetIdx].InstanceID
		guRank := player.GuInventory[targetIdx].Rank
		guID := player.GuInventory[targetIdx].GuID
		guHunger := player.GuInventory[targetIdx].Hunger

		s, found := models.GetGu(guID, guRank)
		if !found {
			return NewMutationError(http.StatusInternalServerError, "Không tìm thấy thông tin Cổ trùng trong thư viện")
		}
		specs = s

		if err := checkRegionAction(player.CurrentRegion, "gu_use"); err != nil {
			return NewMutationError(http.StatusForbidden, err.Error())
		}
		if err := checkRegionGu(player.CurrentRegion, specs.ID, specs.Rank, specs.Path); err != nil {
			return NewMutationError(http.StatusForbidden, err.Error())
		}

		if specs.Type != "Consumable" {
			return NewMutationError(http.StatusBadRequest, "Cổ trùng này không thuộc loại Tiêu Hao, không thể sử dụng trực tiếp!")
		}

		targetGuCopy := player.GuInventory[targetIdx]
		if err := ValidateGuUsable(player, &targetGuCopy, &specs, false); err != nil {
			return NewMutationError(http.StatusBadRequest, err.Error())
		}

		cost := gamefx.EnergyCost(player, &specs)
		player.CurrentMana -= cost

		var logs []string
		for _, trigger := range []string{"ON_CONSUME", "ON_USE"} {
			consumeMsg, err := HandleGuEventTx(tx, player, specs, targetGuCopy, trigger)
			if err != nil {
				return NewMutationError(http.StatusBadRequest, err.Error())
			}
			if consumeMsg != "" {
				logs = append(logs, consumeMsg)
			}
		}

		if len(logs) == 0 {
			return NewMutationError(http.StatusBadRequest, "Hiệu ứng của Cổ trùng tiêu hao này chưa được định nghĩa hoặc không kích hoạt thành công!")
		}

		logMsg = strings.Join(logs, " ")

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

		var hungerCost int = 20
		_, reduceHunger = models.GuNativeMechanic(specs, "SACRIFICE_APTITUDE")

		if reduceHunger {
			if guHunger < hungerCost {
				return NewMutationError(http.StatusBadRequest, "Cổ trùng quá đói, không đủ độ no để hiến tế!")
			}
			targetGuCopy.Hunger -= hungerCost
			if err := db.UpdatePlayerGu(tx, targetGuCopy); err != nil {
				return fmt.Errorf("failed to update Gu hunger: %w", err)
			}
			player.GuInventory[targetIdx].Hunger = targetGuCopy.Hunger
			logMsg += fmt.Sprintf(" Cổ trùng bị suy yếu, độ no giảm %d.", hungerCost)
		} else {
			RemoveGuFromInventory(player, instanceIDToDelete)
			if err := db.RemovePlayerGuForPlayer(tx, player.ID.String(), instanceIDToDelete); err != nil {
				return fmt.Errorf("failed to remove consumed Gu: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		HandleMutationError(c, err)
		return
	}

	applyRealtimeHPDelta(authID, realtimeBaseHP, realtimeActive, player)

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

	respondActionGuOnly(c, http.StatusOK, *player, gin.H{
		"message": logMsg + " Nhân Khí +0.",
	})
}

// UseSupportGuRequest represents request payload to use a support Gu out of combat
type UseSupportGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// UseSupportGu activates an out-of-combat support Gu bug\x27s effect
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

	var logMsg string
	var realtimeBaseHP float64
	var realtimeActive bool

	player, err := MutatePlayer(authID, PlayerMutationOptions{
		Scope:        playerstate.GuFull,
		EventDomains: []PlayerEventDomain{PlayerDomain, GuInventoryDomain, CombatStatsDomain},
	}, func(tx *sql.Tx, player *models.Player) error {
		realtimeBaseHP, realtimeActive = seedRealtimeHP(authID, player)

		targetIdx := -1
		for idx, gu := range player.GuInventory {
			if gu.InstanceID == req.GuInstanceID {
				targetIdx = idx
				break
			}
		}

		if targetIdx == -1 {
			return NewMutationError(http.StatusNotFound, "Không tìm thấy Cổ trùng này trong túi đồ")
		}

		targetGuCopy := player.GuInventory[targetIdx]
		guID := targetGuCopy.GuID
		guRank := targetGuCopy.Rank

		specs, found := models.GetGu(guID, guRank)
		if !found {
			return NewMutationError(http.StatusInternalServerError, "Không tìm thấy thông tin Cổ trùng trong thư viện")
		}

		if err := checkRegionAction(player.CurrentRegion, "gu_use"); err != nil {
			return NewMutationError(http.StatusForbidden, err.Error())
		}
		if err := checkRegionGu(player.CurrentRegion, specs.ID, specs.Rank, specs.Path); err != nil {
			return NewMutationError(http.StatusForbidden, err.Error())
		}

		if specs.Type == "Consumable" {
			return NewMutationError(http.StatusBadRequest, "Cổ trùng Tiêu Hao phải dùng qua chức năng Sử Dụng!")
		}

		if err := ValidateGuUsable(player, &targetGuCopy, &specs, false); err != nil {
			return NewMutationError(http.StatusBadRequest, err.Error())
		}

		cost := gamefx.EnergyCost(player, &specs)
		player.CurrentMana -= cost

		var logs []string
		for _, trigger := range []string{"ON_CONSUME", "ON_USE"} {
			supportMsg, err := HandleGuEvent(player, specs, targetGuCopy, trigger)
			if err != nil {
				return NewMutationError(http.StatusBadRequest, err.Error())
			}
			if supportMsg != "" {
				logs = append(logs, supportMsg)
			}
		}

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
			if maxHP := gamefx.EffectiveMaxHP(player); player.HP > maxHP {
				player.HP = maxHP
			}
			if healed := player.HP - before; healed > 0 {
				logs = append(logs, fmt.Sprintf("%s tấu lên khúc nhạc chữa lành, hồi phục %.0f HP!", specs.Name, healed))
			}
		}

		if len(logs) == 0 {
			return NewMutationError(http.StatusBadRequest, "Cổ trùng này không có hiệu ứng hồi phục để dùng ngoài chiến đấu!")
		}

		targetGuCopy.Hunger -= 1
		if targetGuCopy.Hunger < 0 {
			targetGuCopy.Hunger = 0
		}
		if _, errGu := tx.Exec(`UPDATE player_gu SET hunger = $1, updated_at = NOW() WHERE id = $2`, targetGuCopy.Hunger, targetGuCopy.InstanceID); errGu != nil {
			return fmt.Errorf("không thể cập nhật độ no Cổ trùng: %w", errGu)
		}
		player.GuInventory[targetIdx].Hunger = targetGuCopy.Hunger

		logMsg = strings.Join(logs, " ")
		return nil
	})
	if err != nil {
		HandleMutationError(c, err)
		return
	}

	applyRealtimeHPDelta(authID, realtimeBaseHP, realtimeActive, player)
	respondActionGuOnly(c, http.StatusOK, *player, gin.H{
		"message": logMsg + " Nhân Khí +0.",
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
			// Skip the "Mở <Item> thành công!" or "Mở <Item> thất bại" line as it\x27s redundant
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
