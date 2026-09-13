package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

type StockVaultRequest struct {
	ItemType string `json:"item_type" binding:"required"` // gu, material, recipe
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
}

// stockItemUnitCost computes the sect-funds unit cost for stocking an item and
// enforces the rank gates (Gu: rank 1-5 & <= sect level; material: rank <= sect level).
// Returns (unitCost, httpStatusOK, "") on success; otherwise (0, status, errorMessage).
func stockItemUnitCost(tx *sql.Tx, sectLevel int, itemType, itemID string) (int64, int, string) {
	itemType = strings.ToLower(itemType)
	if itemType == "gu" {
		return 0, http.StatusForbidden, "Không thể nhập Cổ trùng vào bảo khố."
	}

	if itemType == "material" {
		// calculate max allowed item level based on sect level (10 sect levels = 1 item level, max 5)
		maxLevel := sectLevel/10 + 1
		if maxLevel > 5 {
			maxLevel = 5
		}
		var matRank, matRarity int
		err := tx.QueryRow(`SELECT COALESCE(rank, 1), COALESCE(rarity, 0) FROM item_dictionary WHERE id = $1`, itemID).Scan(&matRank, &matRarity)
		if err != nil {
			matRank = 1
			matRarity = 0
		}
		effectiveLevel := matRank
		if matRarity > effectiveLevel {
			effectiveLevel = matRarity
		}

		if effectiveLevel >= 6 {
			return 0, http.StatusForbidden, "Không thể nhập Tiên phẩm Linh tài vào bảo khố tông môn (chỉ hỗ trợ phàm phẩm cấp 1-5)."
		}

		if effectiveLevel > maxLevel {
			return 0, http.StatusForbidden, fmt.Sprintf("Tông môn chưa đạt cấp %d để nhập Linh tài hạng %d! (Hiện tại: Cấp %d)", (effectiveLevel-1)*10, effectiveLevel, sectLevel)
		}

		return int64(effectiveLevel * 50), http.StatusOK, ""
	}

	if itemType == "recipe" {
		var targetGuRank int = 1
		var targetGuRarity int = 0
		err := tx.QueryRow(`
			SELECT COALESCE(gd.rank, 1), COALESCE(gd.rarity, 0)
			FROM refining_recipes rr
			JOIN gu_dictionary gd ON gd.id = rr.target_gu_id AND gd.rank = rr.target_rank
			WHERE ('recipe_' || rr.target_gu_id || '_' || rr.target_rank::text || '_' || rr.variant::text) = $1
		`, itemID).Scan(&targetGuRank, &targetGuRarity)
		if err != nil {
			return 0, http.StatusBadRequest, "Bí phương không tồn tại trong tàng kinh các"
		}

		effectiveLevel := targetGuRank
		if targetGuRarity > effectiveLevel {
			effectiveLevel = targetGuRarity
		}

		if effectiveLevel >= 6 {
			return 0, http.StatusForbidden, "Không thể nhập Tiên cấp Bí phương vào bảo khố tông môn (chỉ hỗ trợ phàm cấp 1-5)."
		}

		// calculate max allowed item level based on sect level (10 sect levels = 1 item level, max 5)
		maxLevel := sectLevel/10 + 1
		if maxLevel > 5 {
			maxLevel = 5
		}
		if effectiveLevel > maxLevel {
			return 0, http.StatusForbidden, fmt.Sprintf("Tông môn chưa đạt cấp %d để nhập Bí phương hạng %d! (Hiện tại: Cấp %d)", (effectiveLevel-1)*10, effectiveLevel, sectLevel)
		}
		return pricing.RecipeCostByGuRank(effectiveLevel), http.StatusOK, ""
	}

	return 0, http.StatusBadRequest, "Loại vật phẩm không hợp lệ (chỉ hỗ trợ gu, material hoặc recipe)"
}

// StockSectVault allows leaders/elders to buy vault items using sect funds
func StockSectVault(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req StockVaultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// Fetch caller's sect and rank
	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền mua thêm vật phẩm cho bảo khố!"})
		return
	}

	// Fetch sect details
	var s Sect
	err = tx.QueryRow(`
		SELECT id, vault_gu, vault_materials, vault_recipes, funds, level
		FROM sects
		WHERE id = $1
	`, sectID).Scan(&s.ID, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details: " + err.Error()})
		return
	}

	unitCost, code, msg := stockItemUnitCost(tx, s.Level, req.ItemType, req.ItemID)
	if code != http.StatusOK {
		c.JSON(code, gin.H{"error": msg})
		return
	}
	itemType := strings.ToLower(req.ItemType)

	totalCost := int64(unitCost) * int64(req.Quantity)
	if s.Funds < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ! Cần %d quỹ tông môn", totalCost)})
		return
	}

	// Deduct funds
	s.Funds -= totalCost

	// Update vault json
	if itemType == "gu" {
		var vaultGus []SectVaultGuItem
		if len(s.VaultGu) > 0 {
			_ = json.Unmarshal(s.VaultGu, &vaultGus)
		}

		// Recalculate cost for this Gu using pricing package
		var stockGuRank int
		var stockGuBasePrice int64
		_ = tx.QueryRow(`SELECT COALESCE(rank,1), COALESCE(base_price,0) FROM gu_dictionary WHERE id = $1 ORDER BY rank DESC LIMIT 1`, req.ItemID).Scan(&stockGuRank, &stockGuBasePrice)
		if stockGuRank == 0 {
			stockGuRank = 1
		}
		stockGuCost := int64(pricing.VaultCost(pricing.GuBasePrice(stockGuBasePrice, stockGuRank)))

		found := false
		for i, v := range vaultGus {
			if v.ID == req.ItemID {
				vaultGus[i].Quantity += req.Quantity
				vaultGus[i].Cost = stockGuCost // refresh cost to current pricing
				vaultGus[i].Hunger = vaultGuMaxHunger(req.ItemID, stockGuRank)
				found = true
				break
			}
		}

		if !found {
			vaultGus = append(vaultGus, SectVaultGuItem{
				ID:       req.ItemID,
				Rank:     stockGuRank,
				Cost:     stockGuCost,
				Quantity: req.Quantity,
				Hunger:   vaultGuMaxHunger(req.ItemID, stockGuRank),
			})
		}

		newVaultGu, _ := json.Marshal(vaultGus)
		_, err = tx.Exec(`UPDATE sects SET vault_gu = $1, funds = $2 WHERE id = $3`, newVaultGu, s.Funds, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault: " + err.Error()})
			return
		}
	} else if itemType == "recipe" {
		var vaultRecipes []models.SectVaultRecipe
		if len(s.VaultRecipes) > 0 {
			_ = json.Unmarshal(s.VaultRecipes, &vaultRecipes)
		}
		if vaultRecipes == nil {
			vaultRecipes = []models.SectVaultRecipe{}
		}

		found := false
		for i := range vaultRecipes {
			if vaultRecipes[i].RecipeID == req.ItemID {
				if vaultRecipes[i].CompletionRate < 100.0 {
					vaultRecipes[i].CompletionRate = 100.0
				}
				found = true
				break
			}
		}
		if !found {
			vaultRecipes = append(vaultRecipes, models.SectVaultRecipe{
				RecipeID:       req.ItemID,
				CompletionRate: 100.0,
			})
		}

		newVaultRecipes, _ := json.Marshal(vaultRecipes)
		_, err = tx.Exec(`UPDATE sects SET vault_recipes = $1, funds = $2 WHERE id = $3`, newVaultRecipes, s.Funds, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes: " + err.Error()})
			return
		}
	} else {
		vaultMats := make(map[string]int64)
		if len(s.VaultMaterials) > 0 {
			_ = json.Unmarshal(s.VaultMaterials, &vaultMats)
		}

		vaultMats[req.ItemID] += req.Quantity

		newVaultMats, _ := json.Marshal(vaultMats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials = $1, funds = $2 WHERE id = $3`, newVaultMats, s.Funds, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Trigger System Chat Notification on Sect channel
	rankName := "Chưởng Môn"
	if callerRank == "TRUONG_LAO" {
		rankName = "Trưởng Lão"
	} else if callerRank == "HO_PHAP" {
		rankName = "Hộ Pháp"
	}
	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	itemName := req.ItemID
	if itemType == "gu" {
		if spec, ok := models.GetGuAny(req.ItemID); ok {
			itemName = spec.Name
		}
	} else if itemType == "material" {
		if spec, ok := models.ItemRegistry[req.ItemID]; ok {
			itemName = spec.Name
		}
	} else if itemType == "recipe" {
		if targetGuID, _, _, err := ParseRecipeID(req.ItemID); err == nil {
			if spec, ok := models.GetGuAny(targetGuID); ok {
				itemName = spec.Name
			}
		}
		itemName = "Bí phương " + itemName
	}
	if username != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("%s %s đã nhập thêm %d %s vào bảo khố tông môn.", rankName, username, req.Quantity, itemName))
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Nhập thêm vật phẩm vào bảo khố thành công!",
		"new_funds": s.Funds,
	})
}

type StockVaultBulkItem struct {
	ItemType string `json:"item_type" binding:"required"` // gu, material, recipe
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
}

type StockVaultBulkRequest struct {
	Items []StockVaultBulkItem `json:"items" binding:"required,min=1"`
}

// StockSectVaultBulk stocks multiple vault items in a single atomic transaction.
func StockSectVaultBulk(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req StockVaultBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Danh sách vật phẩm rỗng!"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Trưởng Lão hoặc Chưởng Môn mới có quyền mua thêm vật phẩm cho bảo khố!"})
		return
	}

	var s Sect
	err = tx.QueryRow(`SELECT id, vault_gu, vault_materials, vault_recipes, funds, level FROM sects WHERE id = $1`, sectID).Scan(&s.ID, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes, &s.Funds, &s.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details: " + err.Error()})
		return
	}

	type stockLine struct {
		itemType string
		itemID   string
		quantity int64
		unitCost int64
	}
	lines := make([]stockLine, 0, len(req.Items))
	var totalCost int64
	for _, it := range req.Items {
		unitCost, code, msg := stockItemUnitCost(tx, s.Level, it.ItemType, it.ItemID)
		if code != http.StatusOK {
			c.JSON(code, gin.H{"error": msg})
			return
		}
		lines = append(lines, stockLine{strings.ToLower(it.ItemType), it.ItemID, it.Quantity, unitCost})
		totalCost += unitCost * it.Quantity
	}

	if s.Funds < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ! Cần %d quỹ tông môn", totalCost)})
		return
	}
	s.Funds -= totalCost

	var vaultGus []SectVaultGuItem
	if len(s.VaultGu) > 0 {
		_ = json.Unmarshal(s.VaultGu, &vaultGus)
	}
	vaultMats := make(map[string]int64)
	if len(s.VaultMaterials) > 0 {
		_ = json.Unmarshal(s.VaultMaterials, &vaultMats)
	}
	var vaultRecipes []models.SectVaultRecipe
	if len(s.VaultRecipes) > 0 {
		_ = json.Unmarshal(s.VaultRecipes, &vaultRecipes)
	}
	if vaultRecipes == nil {
		vaultRecipes = []models.SectVaultRecipe{}
	}

	for _, line := range lines {
		if line.itemType == "gu" {
			// Recalculate cost for this Gu using pricing package
			var stockGuRank int
			var stockGuBasePrice int64
			_ = tx.QueryRow(`SELECT COALESCE(rank,1), COALESCE(base_price,0) FROM gu_dictionary WHERE id = $1 ORDER BY rank DESC LIMIT 1`, line.itemID).Scan(&stockGuRank, &stockGuBasePrice)
			if stockGuRank == 0 {
				stockGuRank = 1
			}
			stockGuCost := int64(pricing.VaultCost(pricing.GuBasePrice(stockGuBasePrice, stockGuRank)))

			found := false
			for i := range vaultGus {
				if vaultGus[i].ID == line.itemID {
					vaultGus[i].Quantity += line.quantity
					vaultGus[i].Cost = stockGuCost
					vaultGus[i].Hunger = vaultGuMaxHunger(line.itemID, stockGuRank)
					found = true
					break
				}
			}
			if !found {
				vaultGus = append(vaultGus, SectVaultGuItem{ID: line.itemID, Rank: stockGuRank, Cost: stockGuCost, Quantity: line.quantity, Hunger: vaultGuMaxHunger(line.itemID, stockGuRank)})
			}
		} else if line.itemType == "recipe" {
			found := false
			for i := range vaultRecipes {
				if vaultRecipes[i].RecipeID == line.itemID {
					if vaultRecipes[i].CompletionRate < 100.0 {
						vaultRecipes[i].CompletionRate = 100.0
					}
					found = true
					break
				}
			}
			if !found {
				vaultRecipes = append(vaultRecipes, models.SectVaultRecipe{
					RecipeID:       line.itemID,
					CompletionRate: 100.0,
				})
			}
		} else {
			vaultMats[line.itemID] += line.quantity
		}
	}

	newVaultGu, _ := json.Marshal(vaultGus)
	newVaultMats, _ := json.Marshal(vaultMats)
	newVaultRecipes, _ := json.Marshal(vaultRecipes)
	_, err = tx.Exec(`UPDATE sects SET vault_gu = $1, vault_materials = $2, vault_recipes = $3, funds = $4 WHERE id = $5`, newVaultGu, newVaultMats, newVaultRecipes, s.Funds, s.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Trigger System Chat Notification on Sect channel
	rankName := "Chưởng Môn"
	if callerRank == "TRUONG_LAO" {
		rankName = "Trưởng Lão"
	} else if callerRank == "HO_PHAP" {
		rankName = "Hộ Pháp"
	}
	var username string
	_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, playerID).Scan(&username)
	if username != "" {
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("%s %s đã nhập thêm %d mặt hàng vào bảo khố tông môn.", rankName, username, len(lines)))
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Nhập thêm vật phẩm vào bảo khố thành công!",
		"new_funds": s.Funds,
	})
}

func BuyFromSectVault(c *gin.Context) {
	var req VaultBuyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	if quantity > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi lần chỉ được trao đổi tối đa 100 đơn vị!"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
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

	_ = worker.EvaluatePlayerHungerTx(tx, playerID)

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	var m SectMember
	err = tx.QueryRow(`
		SELECT id, sect_id, contribution_points
		FROM sect_members
		WHERE player_id = $1
	`, playerID).Scan(&m.ID, &m.SectID, &m.ContributionPoints)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}
	var s Sect
	err = tx.QueryRow(`
		SELECT id, vault_gu, vault_materials, vault_recipes
		FROM sects
		WHERE id = $1
		FOR UPDATE
	`, m.SectID).Scan(&s.ID, &s.VaultGu, &s.VaultMaterials, &s.VaultRecipes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details"})
		return
	}

	var cost int64
	if strings.ToLower(req.ItemType) == "gu" {
		var vaultGus []SectVaultGuItem
		if err := json.Unmarshal(s.VaultGu, &vaultGus); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse vault Gu list"})
			return
		}

		targetIdx := -1
		for idx, item := range vaultGus {
			if item.ID == req.ItemID && (req.Rank == 0 || item.Rank == req.Rank) {
				targetIdx = idx
				break
			}
		}

		if targetIdx == -1 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Cổ trùng này không có sẵn trong Bảo khố!"})
			return
		}

		if vaultGus[targetIdx].Quantity < quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bảo khố chỉ còn %d con Cổ trùng này!", vaultGus[targetIdx].Quantity)})
			return
		}

		cost = int64(vaultGus[targetIdx].Cost * quantity)
		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost
		vaultGus[targetIdx].Quantity -= quantity

		guSpec, _ := models.GetGuAny(req.ItemID)
		actualRank := vaultGus[targetIdx].Rank
		if actualRank == 0 {
			actualRank = guSpec.Rank
		}
		withdrawnInstID := vaultGus[targetIdx].InstanceID
		if vaultGus[targetIdx].Quantity <= 0 {
			vaultGus = append(vaultGus[:targetIdx], vaultGus[targetIdx+1:]...)
		}
		for i := int64(0); i < quantity; i++ {
			newGu := createVaultWithdrawnGu(req.ItemID, actualRank, guSpec)
			if withdrawnInstID != "" {
				newGu.InstanceID = withdrawnInstID
			}
			player.GuInventory = append(player.GuInventory, newGu)

			guName := req.ItemID
			if guSpec.Name != "" {
				guName = guSpec.Name
			}
			withdrawSummary := fmt.Sprintf("Cổ Sư [%s] rút/đổi Cổ trùng [%s] R%d từ Kho Tông Môn.", player.Username, guName, actualRank)
			_ = db.LogGuHistory(tx, newGu.InstanceID, newGu.GuID, newGu.Rank, "SECT_VAULT_WITHDRAWN", nil, &player.ID, withdrawSummary, map[string]interface{}{
				"sect_id": s.ID,
				"cost":    cost,
				"gu_name": guName,
			})
		}

		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory"})
			return
		}

		newVaultGu, err := json.Marshal(vaultGus)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize updated Gu list"})
			return
		}
		_, err = tx.Exec(`UPDATE sects SET vault_gu = $1 WHERE id = $2`, newVaultGu, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu list"})
			return
		}

	} else if strings.ToLower(req.ItemType) == "material" {
		var vaultMats map[string]int64
		if err := json.Unmarshal(s.VaultMaterials, &vaultMats); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse vault materials"})
			return
		}

		qty, hasMat := vaultMats[req.ItemID]
		if !hasMat || qty < quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Linh tài này không đủ số lượng trong Bảo khố!"})
			return
		}

		// Dynamic cost from item_dictionary base_price + rank via pricing package
		{
			var matRank int
			var matBasePrice int64
			_ = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, req.ItemID).Scan(&matBasePrice, &matRank)
			cost = int64(pricing.VaultCost(pricing.ItemBasePrice(matBasePrice, matRank))) * quantity
		}

		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost
		vaultMats[req.ItemID] -= quantity
		if vaultMats[req.ItemID] <= 0 {
			delete(vaultMats, req.ItemID)
		}

		InventorySvc.AddItem(tx, player.ID.String(), req.ItemID, "materials", quantity)

		newVaultMats, _ := json.Marshal(vaultMats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials = $1 WHERE id = $2`, newVaultMats, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vault materials"})
			return
		}
	} else if strings.ToLower(req.ItemType) == "recipe" {
		var vaultRecipes []models.SectVaultRecipe
		if err := json.Unmarshal(s.VaultRecipes, &vaultRecipes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse recipes list"})
			return
		}

		var recipeEntry *models.SectVaultRecipe
		for i := range vaultRecipes {
			if vaultRecipes[i].RecipeID == req.ItemID {
				recipeEntry = &vaultRecipes[i]
				break
			}
		}

		if recipeEntry == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Công thức này không có sẵn trong Bảo khố!"})
			return
		}

		if quantity > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Công thức mỗi lần chỉ có thể học một lần!"})
			return
		}

		// Dynamic recipe cost based on target Gu rank via pricing package
		{
			var targetGuRank int = 1
			if _, tr, _, err := ParseRecipeID(req.ItemID); err == nil {
				targetGuRank = tr
			} else {
				_ = tx.QueryRow(`SELECT COALESCE(gd.rank,1) FROM refining_recipes rr JOIN gu_dictionary gd ON gd.id=rr.target_gu_id AND gd.rank=rr.target_rank WHERE ('recipe_' || rr.target_gu_id || '_' || rr.target_rank::text || '_' || rr.variant::text)=$1`, req.ItemID).Scan(&targetGuRank)
			}
			cost = int64(pricing.RecipeCostByGuRank(targetGuRank))
		}
		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost

		// Remove recipe from vault
		for i := range vaultRecipes {
			if vaultRecipes[i].RecipeID == req.ItemID {
				vaultRecipes = append(vaultRecipes[:i], vaultRecipes[i+1:]...)
				break
			}
		}
		newVaultRecipes, err := json.Marshal(vaultRecipes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize updated recipes list"})
			return
		}
		_, err = tx.Exec(`UPDATE sects SET vault_recipes = $1 WHERE id = $2`, newVaultRecipes, s.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes list"})
			return
		}

		// Grant the recipe to the buyer, keeping the higher completion rate.
		grantedRate := recipeEntry.CompletionRate
		if existing, err := RecipeSvc.Get(tx, player.ID.String(), req.ItemID); err == nil && existing != nil {
			if existing.CompletionRate > grantedRate {
				grantedRate = existing.CompletionRate
			}
		}
		if grantedRate > 100.0 {
			grantedRate = 100.0
		}
		if err := RecipeSvc.Upsert(tx, &services.RecipeProgress{
			PlayerID:       player.ID.String(),
			RecipeID:       req.ItemID,
			CompletionRate: grantedRate,
			IsUnlocked:     true,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể học công thức từ bảo khố"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm trao đổi không hợp lệ!"})
		return
	}

	qiGained := 0.0

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	_, err = tx.Exec(`UPDATE players SET human_qi = $1 WHERE id = $2`, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update human qi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	itemName := req.ItemID
	if req.ItemType == "Gu" {
		if spec, ok := models.GetGuAny(req.ItemID); ok {
			itemName = spec.Name
		}
	} else {
		_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, req.ItemID).Scan(&itemName)
	}
	if itemName == "" {
		itemName = req.ItemID
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.buy_vault", EntityType: "sect",
		EntityID:       m.SectID,
		RequestSummary: fmt.Sprintf("%s | %s x%d | giá %d điểm cống hiến", req.ItemType, itemName, quantity, cost),
		Metadata: map[string]interface{}{
			"item_type":           req.ItemType,
			"item_id":             req.ItemID,
			"quantity":            quantity,
			"cost_contribution":   cost,
			"contribution_change": -cost,
		},
		IPAddress: c.ClientIP(), Success: true,
	})
	msg := fmt.Sprintf("Đổi thành công %d %s! Nhân Khí +%.0f.", quantity, itemName, qiGained)
	if strings.ToLower(req.ItemType) == "gu" {
		if spec, ok := models.GetGuAny(req.ItemID); ok && spec.HungerRate > 0 {
			msg += " Cổ đang đói lả (Độ no = 1), hãy cho ăn sớm để tránh chết đói!"
		}
	}
	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": msg,
		"member":  m,
	})
}
