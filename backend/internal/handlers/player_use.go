package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// HasInventorySpace checks if the player has enough space in their bag
func HasInventorySpace(player *models.Player, incomingCount int) bool {

	return true
	// stats := CalculatePlayerStats(player)
	// maxSlots := int(stats["storage"])
	// if player.Rank >= 6 {
	// 	maxSlots = 99999999
	// }

	// currentSize := len(player.GuInventory)
	// invItems, _ := InventorySvc.ListByPlayer(player.ID.String())
	// seen := map[string]bool{}
	// for _, inv := range invItems {
	// 	if !seen[inv.ItemID] {
	// 		seen[inv.ItemID] = true
	// 		currentSize++
	// 	}
	// }
	//return (currentSize + incomingCount) <= maxSlots
}

// UseItemRequest defines the request body for using an item
type UseItemRequest struct {
	ItemID string `json:"item_id" binding:"required"`
}

// UseItem consumes a material to restore HP or mana
func UseItem(c *gin.Context) {
	var req UseItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	item, ok := models.ItemRegistry[req.ItemID]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong ItemRegistry"})
		return
	}

	if item.Type != "Consumable" && item.Type != "SoulResource" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm này không thể sử dụng trực tiếp"})
		return
	}

	if player.IsDead {
		_, hasLifespanBoost := item.Effects["lifespan_boost"]
		_, hasRaceChange := item.Effects["change_race"]
		if !hasLifespanBoost && !hasRaceChange {
			c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Chỉ có thể sử dụng Thọ Nguyên Cổ để hồi sinh!"})
			return
		}
	}

	// Determine which category the item is in (single query)
	qty, itemCategory, err := InventorySvc.GetQuantityInAnyCategory(tx, player.ID.String(), req.ItemID, "consumables", "materials")
	if err != nil || qty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm này trong túi đồ"})
		return
	}

	messages := []string{}

	// 1. Validation phase
	for effectKey := range item.Effects {
		if valHandler, ok := ValidateItemEffects[effectKey]; ok {
			_, err := valHandler(tx, &player, item)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	// 2. Application phase
	for effectKey := range item.Effects {
		if appHandler, ok := ApplyItemEffects[effectKey]; ok {
			msg, err := appHandler(tx, &player, item)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if msg != "" {
				messages = append(messages, msg)
			}
		}
	}

	if len(messages) == 0 {
		messages = append(messages, fmt.Sprintf("Đạo hữu sử dụng thành công %s.", item.Name))
	}

	// Deduct item from the correct category
	if err := InventorySvc.RemoveItem(tx, player.ID.String(), req.ItemID, itemCategory, 1); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
		return
	}

	// No Gu inventory changes — skip SavePlayerGus
	_, err = tx.Exec(`
		UPDATE players 
		SET current_mana = $1, max_mana = $2, hp = $3, base_hp = $4,
		    primeval_stones = $5, lifespan = $6, human_qi = $7,
		    race = $8, constitution = $9, soul_hp = $10, max_soul = $11, is_dead = $12
		WHERE id = $13`,
		player.CurrentMana, player.MaxMana, player.HP, player.BaseHP,
		player.PrimevalStones, player.Lifespan, player.HumanQi,
		player.Race, player.Constitution, player.SoulHP, player.MaxSoul, player.IsDead,
		player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state: " + err.Error()})
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

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "item.use",
		EntityType:     "item",
		EntityID:       req.ItemID,
		RequestSummary: fmt.Sprintf("Sử dụng 1x %s: %s", item.Name, strings.Join(messages, " ")),
		Metadata: map[string]interface{}{
			"item_id":   req.ItemID,
			"item_name": item.Name,
			"category":  itemCategory,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	player.MaterialsInventory = loadMaterialsInventory(player.ID.String())
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": strings.Join(messages, " ")})
}

type UseItemsRequest struct {
	ItemIDs []string `json:"item_ids" binding:"required,min=1"`
}

func UseItems(c *gin.Context) {
	var req UseItemsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.ItemIDs) > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chỉ có thể sử dụng tối đa 100 vật phẩm một lúc"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}
	initialHP := player.HP
	initialMana := player.CurrentMana
	initialLifespan := player.Lifespan
	initialHumanQi := player.HumanQi
	initialStones := player.PrimevalStones
	initialSoul := int64(0)
	if player.DaoMarks != nil {
		initialSoul = player.DaoMarks["soul"]
	}

	messages := []string{}

	for _, itemID := range req.ItemIDs {
		item, ok := models.ItemRegistry[itemID]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm %s không tồn tại trong ItemRegistry", itemID)})
			return
		}

		if item.Type != "Consumable" && item.Type != "SoulResource" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm %s này không thể sử dụng trực tiếp", item.Name)})
			return
		}

		// Determine which category the item is in (single query)
		qty, itemCategory, err := InventorySvc.GetQuantityInAnyCategory(tx, player.ID.String(), itemID, "consumables", "materials")
		if err != nil || qty <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không có vật phẩm %s này trong túi đồ", item.Name)})
			return
		}

		// 1. Validation phase
		for effectKey := range item.Effects {
			if valHandler, ok := ValidateItemEffects[effectKey]; ok {
				_, err := valHandler(tx, &player, item)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm %s: %s", item.Name, err.Error())})
					return
				}
			}
		}

		// 2. Application phase
		itemMsg := []string{}
		for effectKey := range item.Effects {
			if appHandler, ok := ApplyItemEffects[effectKey]; ok {
				msg, err := appHandler(tx, &player, item)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Vật phẩm %s: %s", item.Name, err.Error())})
					return
				}
				if msg != "" {
					itemMsg = append(itemMsg, msg)
				}
			}
		}

		if len(itemMsg) == 0 {
			itemMsg = append(itemMsg, fmt.Sprintf("Đạo hữu sử dụng thành công %s.", item.Name))
		}
		messages = append(messages, itemMsg...)

		// Deduct item from the correct category
		if err := InventorySvc.RemoveItem(tx, player.ID.String(), itemID, itemCategory, 1); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể xóa vật phẩm %s khỏi túi đồ", item.Name)})
			return
		}
	}

	// No Gu inventory changes — skip SavePlayerGus
	_, err = tx.Exec(`
		UPDATE players 
		SET current_mana = $1, max_mana = $2, hp = $3, base_hp = $4,
		    primeval_stones = $5, lifespan = $6, human_qi = $7,
		    race = $8, constitution = $9, soul_hp = $10, max_soul = $11, is_dead = $12
		WHERE id = $13`,
		player.CurrentMana, player.MaxMana, player.HP, player.BaseHP,
		player.PrimevalStones, player.Lifespan, player.HumanQi,
		player.Race, player.Constitution, player.SoulHP, player.MaxSoul, player.IsDead,
		player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state: " + err.Error()})
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

	// Group/consolidate stats change and messages
	usedCounts := make(map[string]int)
	itemNames := make(map[string]string)
	for _, itemID := range req.ItemIDs {
		if spec, ok := models.ItemRegistry[itemID]; ok {
			usedCounts[itemID]++
			itemNames[itemID] = spec.Name
		}
	}

	var usageDetails []string
	for itemID, count := range usedCounts {
		usageDetails = append(usageDetails, fmt.Sprintf("%dx %s", count, itemNames[itemID]))
	}
	summaryMsg := fmt.Sprintf("Đạo hữu đã sử dụng thành công: %s.", strings.Join(usageDetails, ", "))

	var statChanges []string
	if diffHP := player.HP - initialHP; diffHP > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.0f HP", diffHP))
	}
	if diffMana := player.CurrentMana - initialMana; diffMana > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.0f Chân Nguyên", diffMana))
	}
	if diffLifespan := player.Lifespan - initialLifespan; diffLifespan > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.0f năm Thọ Nguyên", diffLifespan))
	}
	if diffHumanQi := player.HumanQi - initialHumanQi; diffHumanQi > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.0f Nhân Khí", diffHumanQi))
	}
	if diffStones := player.PrimevalStones - initialStones; diffStones > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%.3f Nguyên Thạch", diffStones))
	}
	finalSoul := int64(0)
	if player.DaoMarks != nil {
		finalSoul = player.DaoMarks["soul"]
	}
	if diffSoul := finalSoul - initialSoul; diffSoul > 0 {
		statChanges = append(statChanges, fmt.Sprintf("+%d Hồn Lực", diffSoul))
	}

	if len(statChanges) > 0 {
		summaryMsg += fmt.Sprintf(" Kết quả: %s.", strings.Join(statChanges, ", "))
	}

	var specialMessages []string
	for _, m := range messages {
		isStandard := false
		lowerM := strings.ToLower(m)
		if strings.Contains(lowerM, "hồi phục") && strings.Contains(lowerM, "hp") {
			isStandard = true
		} else if strings.Contains(lowerM, "phục hồi") && strings.Contains(lowerM, "chân nguyên") {
			isStandard = true
		} else if strings.Contains(lowerM, "thọ nguyên") {
			isStandard = true
		} else if strings.Contains(lowerM, "nhân khí") {
			isStandard = true
		} else if strings.Contains(lowerM, "hồn lực") || strings.Contains(lowerM, "hồn phách") {
			isStandard = true
		} else if strings.Contains(lowerM, "tiên khiếu") && strings.Contains(lowerM, "khôi phục") {
			isStandard = true
		} else if strings.Contains(lowerM, "sử dụng thành công") {
			isStandard = true
		}

		if !isStandard && m != "" {
			specialMessages = append(specialMessages, m)
		}
	}

	finalMessage := summaryMsg
	if len(specialMessages) > 0 {
		aggregatedSpecial := aggregateChestSpecialMessages(specialMessages)
		finalMessage += "\n" + strings.Join(aggregatedSpecial, "\n")
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "item.use_batch",
		EntityType:     "item",
		RequestSummary: fmt.Sprintf("Sử dụng hàng loạt %d vật phẩm: %s", len(req.ItemIDs), strings.Join(usageDetails, ", ")),
		Metadata: map[string]interface{}{
			"item_counts":  usedCounts,
			"total_count":  len(req.ItemIDs),
			"stat_changes": statChanges,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	player.MaterialsInventory = loadMaterialsInventory(player.ID.String())
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": finalMessage})
}

// SetVitalGuRequest represents the payload to select a vital Gu
type SetVitalGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// SetVitalGu sets a selected Gu as the player's Vital Gu
func SetVitalGu(c *gin.Context) {
	var req SetVitalGuRequest
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

	targetIndex := -1
	for idx, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			targetIndex = idx
			break
		}
	}

	if targetIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng trong túi đồ"})
		return
	}

	targetGu := &player.GuInventory[targetIndex]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể thiết lập Cổ trùng ĐÃ CHẾT làm Bản Mệnh Cổ"})
		return
	}

	if targetGu.Rank > player.Rank {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể thiết lập Cổ trùng Rank %d làm Bản Mệnh Cổ do vượt quá tu vi hiện tại của đạo hữu (Rank %d)!", targetGu.Rank, player.Rank)})
		return
	}

	dictGu, ok := models.GetGu(targetGu.GuID, targetGu.Rank)
	if ok && (dictGu.Type == "Consumable" || dictGu.Type == "consumable") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể thiết lập Cổ trùng tiêu hao (Consumable) làm Bản Mệnh Cổ"})
		return
	}

	if targetGu.IsVital {
		respondActionGuOnly(c, http.StatusOK, player, gin.H{
			"message": "Cổ trùng này đã là Bản Mệnh Cổ của đạo hữu rồi.",
		})
		return
	}

	// Vital Gu is always equipped; refuse if equipping it would exceed the aperture capacity
	if !targetGu.Equipped {
		if models.CountEquippedActive(player.GuInventory) >= db.MaxEquippedGus {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Aperture capacity full! Cannot equip more than %d Gu bugs.", db.MaxEquippedGus)})
			return
		}
	}

	// Check if they already have an existing Vital Gu
	_, _, hadVital := models.FindFirstMatchingGu(player.GuInventory, func(gu models.GuInstance) bool {
		return gu.IsVital
	})

	backlashMessage := ""
	if hadVital {
		// Backlash: apply severely_wounded status for 1 hour
		player.Statuses.Add("severely_wounded", "set_vital_gu", 1*time.Hour, 0)
		backlashMessage = " Đạo hữu bị phản phệ rơi vào trạng thái Trọng Thương do thay đổi Bản Mệnh Cổ!"
	}

	// Update is_vital flags; vital Gu is always ACTIVE and equipped
	for idx := range player.GuInventory {
		if idx == targetIndex {
			player.GuInventory[idx].IsVital = true
			player.GuInventory[idx].Status = "ACTIVE"
			player.GuInventory[idx].Equipped = true
		} else {
			player.GuInventory[idx].IsVital = false
		}
	}

	// Persist with two targeted statements: clear the old vital flag if one existed,
	// then mark the target, without locking or rewriting the whole inventory.
	if _, err := tx.Exec(`UPDATE player_gu SET is_vital = false, updated_at = NOW() WHERE player_id = $1 AND is_vital = true AND id <> $2`, player.ID, player.GuInventory[targetIndex].InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player vital Gu status"})
		return
	}
	if _, err := tx.Exec(`UPDATE player_gu SET is_vital = true, state = 'ACTIVE', equipped = true, updated_at = NOW() WHERE id = $1`, player.GuInventory[targetIndex].InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player vital Gu status"})
		return
	}

	// Recalculate equipped_gus
	player.EquippedGus = []string{}
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}

	_, err = tx.Exec(`
		UPDATE players 
		SET hp = $1, equipped_gus = $2, active_combat = $3, statuses = $4
		WHERE id = $5`,
		player.HP, player.EquippedGus, player.ActiveCombat, player.Statuses, player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player vital Gu status"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "gu.set_vital", EntityType: "gu_instance",
		EntityID:       req.GuInstanceID,
		RequestSummary: fmt.Sprintf("Thiết lập %s thành Bản Mệnh Cổ | hadVital=%v", targetGu.GuID, hadVital),
		IPAddress:      c.ClientIP(), Success: true,
	})
	respondActionGuOnly(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Thiết lập %s thành Bản Mệnh Cổ thành công!%s", targetGu.GuID, backlashMessage),
	})
}

// DropGuRequest removes an owned Gu from inventory intentionally.
type DropGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

func DropGu(c *gin.Context) {
	var req DropGuRequest
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

	idx := -1
	for i, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng trong túi đồ"})
		return
	}
	if player.GuInventory[idx].IsVital {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể vứt Bản Mệnh Cổ"})
		return
	}

	droppedInstanceID := player.GuInventory[idx].InstanceID
	droppedGu := player.GuInventory[idx]
	newInv := models.GuInventory{}
	for i, gu := range player.GuInventory {
		if i != idx {
			newInv = append(newInv, gu)
		}
	}
	player.GuInventory = newInv
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}
	if err := db.RemovePlayerGuForPlayer(tx, player.ID.String(), droppedInstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop Gu"})
		return
	}
	// Terminal log cho Tiên Cổ: thả = end lifecycle instance_id này.
	_ = db.LogGuHistory(tx, droppedGu.InstanceID, droppedGu.GuID, droppedGu.Rank, "DROPPED_RELEASED", &player.ID, nil,
		fmt.Sprintf("Cổ Sư [%s] thả Cổ trùng [%s] R%d, kết thúc vòng đời instance %s.", player.Username, droppedGu.GuID, droppedGu.Rank, droppedInstanceID),
		map[string]interface{}{"terminal": true, "reason": "drop"})
	_, err = tx.Exec(`UPDATE players SET equipped_gus = $1 WHERE id = $2`, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop Gu"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "gu.drop", EntityType: "gu_instance",
		EntityID: req.GuInstanceID, RequestSummary: "Vứt bỏ Cổ trùng khỏi túi đồ",
		IPAddress: c.ClientIP(), Success: true,
	})
	respondActionGuOnly(c, http.StatusOK, player, gin.H{"message": "Đã vứt bỏ Cổ trùng khỏi túi đồ."})
}

// DropGusRequest represents request payload to drop multiple Gu instances.
type DropGusRequest struct {
	GuInstanceIDs []string `json:"gu_instance_ids" binding:"required"`
}

// DropGus removes multiple owned Gus from inventory.
func DropGus(c *gin.Context) {
	var req DropGusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_ids is required"})
		return
	}
	if len(req.GuInstanceIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách gu_instance_ids trống"})
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

	dropMap := make(map[string]bool)
	for _, id := range req.GuInstanceIDs {
		dropMap[id] = true
	}

	foundCount := 0
	for _, gu := range player.GuInventory {
		if dropMap[gu.InstanceID] {
			foundCount++
			if gu.IsVital {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không thể vứt Bản Mệnh Cổ (%s)", gu.InstanceID)})
				return
			}
		}
	}

	if foundCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng nào trong danh sách yêu cầu"})
		return
	}

	newInv := models.GuInventory{}
	droppedGus := []models.GuInstance{}
	for _, gu := range player.GuInventory {
		if !dropMap[gu.InstanceID] {
			newInv = append(newInv, gu)
		} else {
			droppedGus = append(droppedGus, gu)
		}
	}
	player.GuInventory = newInv
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}
	if err := db.RemovePlayerGusForPlayer(tx, player.ID.String(), req.GuInstanceIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop Gus"})
		return
	}
	for _, dg := range droppedGus {
		_ = db.LogGuHistory(tx, dg.InstanceID, dg.GuID, dg.Rank, "DROPPED_RELEASED", &player.ID, nil,
			fmt.Sprintf("Cổ Sư [%s] thả Cổ trùng [%s] R%d, kết thúc vòng đời instance %s.", player.Username, dg.GuID, dg.Rank, dg.InstanceID),
			map[string]interface{}{"terminal": true, "reason": "drop_batch"})
	}
	_, err = tx.Exec(`UPDATE players SET equipped_gus = $1 WHERE id = $2`, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop Gus"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "gu.drop_batch", EntityType: "gu_instance",
		RequestSummary: fmt.Sprintf("Vứt bỏ hàng loạt %d Cổ trùng khỏi túi đồ", len(req.GuInstanceIDs)),
		IPAddress:      c.ClientIP(), Success: true,
	})
	respondActionGuOnly(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Đã vứt bỏ %d Cổ trùng khỏi túi đồ.", foundCount)})
}
