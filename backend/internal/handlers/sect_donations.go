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
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

// DonateStones converts player primeval stones to contribution points
func DonateStones(c *gin.Context) {
	var req DonateStonesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	if player.PrimevalStones < float64(req.Amount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Nguyên Thạch!"})
		return
	}

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

	player.PrimevalStones -= float64(req.Amount)
	m.ContributionPoints += int64(req.Amount)

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, req.Amount, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	qiGained := 0.0
	res, err := tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2 AND primeval_stones >= $1`, req.Amount, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct stones"})
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch"})
		return
	}

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to award contribution"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp %d Nguyên Thạch cho tông môn.", player.Username, req.Amount))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_stones", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%d Nguyên Thạch → %d điểm cống hiến", req.Amount, req.Amount),
		IPAddress: c.ClientIP(), Success: true,
	})
	respondActionOnly(c, http.StatusOK, player, gin.H{
		"message":             fmt.Sprintf("Quyên góp thành công! Nhân Khí +%.0f.", qiGained),
		"primeval_stones":     player.PrimevalStones,
		"contribution_points": m.ContributionPoints,
	})
}

// DonateMaterials converts player materials to contribution points
func DonateMaterials(c *gin.Context) {
	var req DonateMaterialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	rows, err := tx.Query(`SELECT category, quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2`, playerID.String(), req.MaterialID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking inventory"})
		return
	}

	type invItem struct {
		category string
		quantity int64
	}
	var invs []invItem
	totalQty := int64(0)
	for rows.Next() {
		var iv invItem
		if err := rows.Scan(&iv.category, &iv.quantity); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error scanning inventory"})
			return
		}
		invs = append(invs, iv)
		totalQty += iv.quantity
	}
	rows.Close()

	if totalQty < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có đủ linh tài yêu cầu!"})
		return
	}

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

	// Calculate value dynamically from item_dictionary.base_price + rank (via pricing package)
	var dbBasePrice float64
	var matRank int
	var itemType string
	if err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1), COALESCE(item_type,'') FROM item_dictionary WHERE id=$1`, req.MaterialID).Scan(&dbBasePrice, &matRank, &itemType); err != nil {
		matRank = 1
		dbBasePrice = 0
		itemType = ""
	}
	pointsPerItem := pricing.DonatePoints(pricing.ItemBasePrice(dbBasePrice, matRank))

	gainedContribution := int64(pointsPerItem) * req.Amount

	preferredCat := "materials"
	if itemType == "Consumable" {
		preferredCat = "consumables"
	}

	remainingDeduct := req.Amount
	for _, iv := range invs {
		if iv.category == preferredCat {
			deduct := iv.quantity
			if deduct > remainingDeduct {
				deduct = remainingDeduct
			}
			if err := InventorySvc.RemoveItem(tx, playerID.String(), req.MaterialID, iv.category, deduct); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
				return
			}
			remainingDeduct -= deduct
			if remainingDeduct <= 0 {
				break
			}
		}
	}
	if remainingDeduct > 0 {
		for _, iv := range invs {
			if iv.category != preferredCat {
				deduct := iv.quantity
				if deduct > remainingDeduct {
					deduct = remainingDeduct
				}
				if err := InventorySvc.RemoveItem(tx, playerID.String(), req.MaterialID, iv.category, deduct); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
					return
				}
				remainingDeduct -= deduct
				if remainingDeduct <= 0 {
					break
				}
			}
		}
	}

	m.ContributionPoints += gainedContribution
	qiGained := 0.0

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, gainedContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	// Add to sect vault
	var vaultMaterialsRaw json.RawMessage
	err = tx.QueryRow(`SELECT vault_materials FROM sects WHERE id = $1 FOR UPDATE`, m.SectID).Scan(&vaultMaterialsRaw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault"})
		return
	}

	vaultMats := make(map[string]int64)
	if len(vaultMaterialsRaw) > 0 {
		_ = json.Unmarshal(vaultMaterialsRaw, &vaultMats)
	}

	vaultMats[req.MaterialID] += req.Amount
	newVaultMats, _ := json.Marshal(vaultMats)
	_, err = tx.Exec(`UPDATE sects SET vault_materials = $1 WHERE id = $2`, newVaultMats, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault materials"})
		return
	}

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

	materialName := req.MaterialID
	_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, req.MaterialID).Scan(&materialName)
	if materialName == "" {
		materialName = req.MaterialID
	}

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp %d %s cho tông môn.", player.Username, req.Amount, materialName))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_materials", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%s x%d → %d điểm", materialName, req.Amount, gainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})
	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message":             fmt.Sprintf("Quyên góp linh tài thành công! Nhân Khí +%.0f.", qiGained),
		"contribution_points": m.ContributionPoints,
	})
}

// DonateGu converts player Gu bug to contribution points
func DonateGu(c *gin.Context) {
	var req DonateGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	targetIdx := -1
	for i, gi := range player.GuInventory {
		if gi.InstanceID == req.GuInstanceID {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cổ trùng không tồn tại trong túi đồ!"})
		return
	}

	guInstance := player.GuInventory[targetIdx]
	if guInstance.IsVital {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể quyên góp Bản Mệnh Cổ!"})
		return
	}
	if guInstance.Equipped {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng tháo Cổ trùng ra trước khi quyên góp!"})
		return
	}

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

	// Fetch Gu base_price + rank for contribution calculation
	var guRank int
	var guBasePrice float64
	err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1 AND rank = $2`, guInstance.GuID, guInstance.Rank).Scan(&guBasePrice, &guRank)
	if err != nil {
		guRank = 1
		guBasePrice = 0
	}

	effectiveBase := pricing.GuBasePrice(guBasePrice, guRank)
	gainedContribution := pricing.DonatePoints(effectiveBase)
	vaultItemCost := int64(pricing.VaultCost(effectiveBase))

	// Add to sect vault

	var vaultGuRaw json.RawMessage
	err = tx.QueryRow(`SELECT vault_gu FROM sects WHERE id = $1 FOR UPDATE`, m.SectID).Scan(&vaultGuRaw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault"})
		return
	}

	var vaultGus []SectVaultGuItem
	if len(vaultGuRaw) > 0 {
		_ = json.Unmarshal(vaultGuRaw, &vaultGus)
	}

	// Increment quantity if Gu type already exists in vault, otherwise add new
	// entry. Raise the group hunger to the donated Gu's hunger when fresher.
	donatedGuFound := false
	if guInstance.Rank < 6 {
		for i, v := range vaultGus {
			if v.ID == guInstance.GuID && v.Rank == guInstance.Rank && v.InstanceID == "" {
				vaultGus[i].Quantity++
				vaultGus[i].Hunger = mergeVaultGuHunger(guInstance.GuID, guInstance.Rank, v.Hunger, float64(guInstance.Hunger))
				donatedGuFound = true
				break
			}
		}
	}
	if !donatedGuFound {
		itemInstID := ""
		if guInstance.Rank >= 6 {
			itemInstID = guInstance.InstanceID
		}
		vaultGus = append(vaultGus, SectVaultGuItem{
			ID:         guInstance.GuID,
			InstanceID: itemInstID,
			Rank:       guInstance.Rank,
			Cost:       vaultItemCost,
			Quantity:   1,
			Hunger:     normalizeVaultGuHunger(guInstance.GuID, guInstance.Rank, float64(guInstance.Hunger)),
		})
	}

	newVaultGus, _ := json.Marshal(vaultGus)
	_, err = tx.Exec(`UPDATE sects SET vault_gu = $1 WHERE id = $2`, newVaultGus, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu list"})
		return
	}

	// Remove Gu
	player.GuInventory = append(player.GuInventory[:targetIdx], player.GuInventory[targetIdx+1:]...)
	m.ContributionPoints += int64(gainedContribution)
	qiGained := 0.0

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, gainedContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	if err := db.RemovePlayerGu(tx, guInstance.InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gu inventory"})
		return
	}

	guName := guInstance.GuID
	if spec, ok := models.GetGu(guInstance.GuID, guInstance.Rank); ok && spec.Name != "" {
		guName = spec.Name
	}
	sectSummary := fmt.Sprintf("Cổ Sư [%s] cống hiến Cổ trùng [%s] R%d vào Kho Tông Môn.", player.Username, guName, guInstance.Rank)
	_ = db.LogGuHistory(tx, guInstance.InstanceID, guInstance.GuID, guInstance.Rank, "SECT_VAULT_DEPOSITED", &player.ID, nil, sectSummary, map[string]interface{}{
		"sect_id": m.SectID,
		"gu_name": guName,
	})
	_, err = tx.Exec(`UPDATE players SET human_qi = $1 WHERE id = $2`, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gu inventory"})
		return
	}

	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	guName = guInstance.GuID
	if spec, ok := models.GetGu(guInstance.GuID, guInstance.Rank); ok {
		guName = spec.Name
	}

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp Cổ trùng %s cho tông môn.", player.Username, guName))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_gu", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%s → %d điểm", guName, gainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})
	respondActionGuOnly(c, http.StatusOK, player, gin.H{
		"message":             fmt.Sprintf("Quyên góp Cổ trùng thành công! Nhân Khí +%.0f.", qiGained),
		"contribution_points": m.ContributionPoints,
	})
}

// DonateRecipe converts a player recipe to contribution points and adds it to
// the sect vault so other members can learn it.
func DonateRecipe(c *gin.Context) {
	var req DonateRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player"})
		return
	}

	progress, err := RecipeSvc.Get(tx, playerID.String(), req.RecipeID)
	if err != nil || progress == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bí phương không tồn tại trong túi đồ!"})
		return
	}

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

	// Resolve target Gu rank and compute contribution points scaled by completion rate.
	var targetGuRank int = 1
	if _, tr, _, err := ParseRecipeID(req.RecipeID); err == nil {
		targetGuRank = tr
	} else {
		_ = tx.QueryRow(`
			SELECT COALESCE(gd.rank,1)
			FROM refining_recipes rr
			JOIN gu_dictionary gd ON gd.id=rr.target_gu_id AND gd.rank=rr.target_rank
			WHERE ('recipe_' || rr.target_gu_id || '_' || rr.target_rank::text || '_' || rr.variant::text)=$1
		`, req.RecipeID).Scan(&targetGuRank)
	}
	basePoints := pricing.RecipeDonatePoints(targetGuRank)
	gainedContribution := int(float64(basePoints) * progress.CompletionRate / 100.0)
	if gainedContribution < 1 {
		gainedContribution = 1
	}

	if err := RecipeSvc.Delete(tx, playerID.String(), req.RecipeID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa Bí phương khỏi túi đồ"})
		return
	}

	m.ContributionPoints += int64(gainedContribution)
	qiGained := 0.0

	// Increase sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, gainedContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	// Add recipe to sect vault, keeping the higher completion rate.
	var vaultRecipes []models.SectVaultRecipe
	_ = tx.QueryRow(`SELECT vault_recipes FROM sects WHERE id = $1 FOR UPDATE`, m.SectID).Scan(&vaultRecipes)
	if vaultRecipes == nil {
		vaultRecipes = []models.SectVaultRecipe{}
	}
	found := false
	for i := range vaultRecipes {
		if vaultRecipes[i].RecipeID == req.RecipeID {
			if progress.CompletionRate > vaultRecipes[i].CompletionRate {
				vaultRecipes[i].CompletionRate = progress.CompletionRate
			}
			found = true
			break
		}
	}
	if !found {
		vaultRecipes = append(vaultRecipes, models.SectVaultRecipe{
			RecipeID:       req.RecipeID,
			CompletionRate: progress.CompletionRate,
		})
	}
	newVaultRecipes, _ := json.Marshal(vaultRecipes)
	_, err = tx.Exec(`UPDATE sects SET vault_recipes = $1 WHERE id = $2`, newVaultRecipes, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes"})
		return
	}

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

	itemName := req.RecipeID
	if spec, ok := models.GetGuAny(req.RecipeID); ok {
		itemName = spec.Name
	}

	// Trigger System Chat Notification on Sect channel
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp Bí phương %s cho tông môn.", player.Username, itemName))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_recipe", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("%s (%.0f%%) → %d điểm", itemName, progress.CompletionRate, gainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})
	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message":             fmt.Sprintf("Quyên góp Bí phương thành công! Nhân Khí +%.0f.", qiGained),
		"contribution_points": m.ContributionPoints,
	})
}

// BulkDonateRequest represents a collection of materials, Gus and recipes to donate
type BulkDonateRequest struct {
	Materials []DonateMaterialsRequest `json:"materials"`
	Gus       []DonateGuRequest        `json:"gus"`
	Recipes   []DonateRecipeRequest    `json:"recipes"`
}

// DonateBulk converts multiple player materials and Gu bugs to contribution points in a single transaction
func DonateBulk(c *gin.Context) {
	var req BulkDonateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

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

	totalGainedContribution := int64(0)
	var auditSummaries []string

	// 1. Process Materials
	if len(req.Materials) > 0 {
		// Fetch sect vault materials
		var vaultMaterialsRaw json.RawMessage
		err = tx.QueryRow(`SELECT vault_materials FROM sects WHERE id = $1 FOR UPDATE`, m.SectID).Scan(&vaultMaterialsRaw)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault materials"})
			return
		}
		vaultMats := make(map[string]int64)
		if len(vaultMaterialsRaw) > 0 {
			_ = json.Unmarshal(vaultMaterialsRaw, &vaultMats)
		}

		for _, matReq := range req.Materials {
			rows, err := tx.Query(`SELECT category, quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2`, playerID.String(), matReq.MaterialID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking inventory"})
				return
			}

			type invItem struct {
				category string
				quantity int64
			}
			var invs []invItem
			totalQty := int64(0)
			for rows.Next() {
				var iv invItem
				if err := rows.Scan(&iv.category, &iv.quantity); err != nil {
					rows.Close()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error scanning inventory"})
					return
				}
				invs = append(invs, iv)
				totalQty += iv.quantity
			}
			rows.Close()

			if totalQty < matReq.Amount {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đạo hữu không có đủ linh tài: %s", matReq.MaterialID)})
				return
			}

			// Calculate contribution points dynamically using in-memory registry
			var dbBasePrice float64
			var matRank int = 1
			var itemType string
			materialName := matReq.MaterialID
			if item, exists := models.ItemRegistry[matReq.MaterialID]; exists {
				dbBasePrice = item.BasePrice
				matRank = item.Rank
				itemType = item.Type
				if item.Name != "" {
					materialName = item.Name
				}
			} else if err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1), COALESCE(item_type,''), COALESCE(name, '') FROM item_dictionary WHERE id=$1`, matReq.MaterialID).Scan(&dbBasePrice, &matRank, &itemType, &materialName); err != nil {
				matRank = 1
				dbBasePrice = 0
				itemType = ""
			}
			pointsPerItem := pricing.DonatePoints(pricing.ItemBasePrice(dbBasePrice, matRank))
			gained := int64(pointsPerItem) * matReq.Amount
			totalGainedContribution += gained

			preferredCat := "materials"
			if itemType == "Consumable" {
				preferredCat = "consumables"
			}

			// Update Inventory
			remainingDeduct := matReq.Amount
			for _, iv := range invs {
				if iv.category == preferredCat {
					deduct := iv.quantity
					if deduct > remainingDeduct {
						deduct = remainingDeduct
					}
					if err := InventorySvc.RemoveItem(tx, playerID.String(), matReq.MaterialID, iv.category, deduct); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
						return
					}
					remainingDeduct -= deduct
					if remainingDeduct <= 0 {
						break
					}
				}
			}
			if remainingDeduct > 0 {
				for _, iv := range invs {
					if iv.category != preferredCat {
						deduct := iv.quantity
						if deduct > remainingDeduct {
							deduct = remainingDeduct
						}
						if err := InventorySvc.RemoveItem(tx, playerID.String(), matReq.MaterialID, iv.category, deduct); err != nil {
							c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa vật phẩm khỏi túi đồ"})
							return
						}
						remainingDeduct -= deduct
						if remainingDeduct <= 0 {
							break
						}
					}
				}
			}

			// Update Sect Vault map
			vaultMats[matReq.MaterialID] += matReq.Amount
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d", materialName, matReq.Amount))
		}

		// Save updated materials vault
		newVaultMats, _ := json.Marshal(vaultMats)
		_, err = tx.Exec(`UPDATE sects SET vault_materials = $1 WHERE id = $2`, newVaultMats, m.SectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault materials"})
			return
		}
	}

	// 2. Process Gu Instances
	if len(req.Gus) > 0 {

		// Fetch sect vault Gu
		var vaultGuRaw json.RawMessage
		err = tx.QueryRow(`SELECT vault_gu FROM sects WHERE id = $1 FOR UPDATE`, m.SectID).Scan(&vaultGuRaw)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault gu"})
			return
		}
		var vaultGus []SectVaultGuItem
		if len(vaultGuRaw) > 0 {
			_ = json.Unmarshal(vaultGuRaw, &vaultGus)
		}

		// Keep track of instance IDs we want to remove
		gusToRemove := make(map[string]bool)
		removedIDs := make([]string, 0, len(req.Gus))
		for _, guReq := range req.Gus {
			gusToRemove[guReq.GuInstanceID] = true
			removedIDs = append(removedIDs, guReq.GuInstanceID)
		}

		// Process each target Gu instance
		var newGuInventory []models.GuInstance
		for _, gi := range player.GuInventory {
			if gusToRemove[gi.InstanceID] {
				if gi.IsVital {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể quyên góp Bản Mệnh Cổ!"})
					return
				}
				if gi.Equipped {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng tháo Cổ trùng ra trước khi quyên góp!"})
					return
				}

				// Fetch base price using in-memory Gu registry
				var guRank int = gi.Rank
				var guBasePrice float64
				guName := gi.GuID
				if spec, ok := models.GetGu(gi.GuID, gi.Rank); ok {
					guRank = spec.Rank
					guBasePrice = spec.BasePrice
					if spec.Name != "" {
						guName = spec.Name
					}
				} else {
					err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1 AND rank = $2`, gi.GuID, gi.Rank).Scan(&guBasePrice, &guRank)
					if err != nil {
						guRank = 1
						guBasePrice = 0
					}
				}
				effectiveBase := pricing.GuBasePrice(guBasePrice, guRank)
				gained := int64(pricing.DonatePoints(effectiveBase))
				vaultItemCost := int64(pricing.VaultCost(effectiveBase))

				totalGainedContribution += gained

				// Update Vault Gu count, raising group hunger when the donated
				// Gu is fresher.
				donatedGuFound := false
				for idx, v := range vaultGus {
					if v.ID == gi.GuID && v.Rank == gi.Rank {
						vaultGus[idx].Quantity++
						vaultGus[idx].Hunger = mergeVaultGuHunger(gi.GuID, gi.Rank, v.Hunger, float64(gi.Hunger))
						donatedGuFound = true
						break
					}
				}
				if !donatedGuFound {
					vaultGus = append(vaultGus, SectVaultGuItem{
						ID:       gi.GuID,
						Rank:     gi.Rank,
						Cost:     vaultItemCost,
						Quantity: 1,
						Hunger:   normalizeVaultGuHunger(gi.GuID, gi.Rank, float64(gi.Hunger)),
					})
				}

				auditSummaries = append(auditSummaries, guName)
			} else {
				newGuInventory = append(newGuInventory, gi)
			}
		}

		// Verify that all requested Gus were actually found and processed
		if len(player.GuInventory)-len(newGuInventory) != len(req.Gus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Một hoặc nhiều Cổ trùng không tồn tại trong túi đồ!"})
			return
		}

		player.GuInventory = newGuInventory

		// Save player Gu inventory
		if err := db.RemovePlayerGus(tx, removedIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update gu inventory"})
			return
		}

		// Save updated Gu vault
		newVaultGus, _ := json.Marshal(vaultGus)
		_, err = tx.Exec(`UPDATE sects SET vault_gu = $1 WHERE id = $2`, newVaultGus, m.SectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu list"})
			return
		}
	}

	// 3. Process Recipes
	if len(req.Recipes) > 0 {
		var vaultRecipes []models.SectVaultRecipe
		var vaultRecipesRaw json.RawMessage
		err = tx.QueryRow(`SELECT vault_recipes FROM sects WHERE id = $1 FOR UPDATE`, m.SectID).Scan(&vaultRecipesRaw)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sect vault recipes"})
			return
		}
		if len(vaultRecipesRaw) > 0 {
			_ = json.Unmarshal(vaultRecipesRaw, &vaultRecipes)
		}
		if vaultRecipes == nil {
			vaultRecipes = []models.SectVaultRecipe{}
		}

		donatedRecipeIDs := make(map[string]bool)
		for _, r := range req.Recipes {
			donatedRecipeIDs[r.RecipeID] = true
		}

		for _, recipeReq := range req.Recipes {
			progress, err := RecipeSvc.Get(tx, playerID.String(), recipeReq.RecipeID)
			if err != nil || progress == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bí phương không tồn tại trong túi đồ: %s", recipeReq.RecipeID)})
				return
			}

			var targetGuRank int = 1
			if _, tr, _, err := ParseRecipeID(recipeReq.RecipeID); err == nil {
				targetGuRank = tr
			} else {
				_ = tx.QueryRow(`
					SELECT COALESCE(gd.rank,1)
					FROM refining_recipes rr
					JOIN gu_dictionary gd ON gd.id=rr.target_gu_id AND gd.rank=rr.target_rank
					WHERE ('recipe_' || rr.target_gu_id || '_' || rr.target_rank::text || '_' || rr.variant::text)=$1
				`, recipeReq.RecipeID).Scan(&targetGuRank)
			}
			basePoints := pricing.RecipeDonatePoints(targetGuRank)
			gained := int(float64(basePoints) * progress.CompletionRate / 100.0)
			if gained < 1 {
				gained = 1
			}
			totalGainedContribution += int64(gained)

			if err := RecipeSvc.Delete(tx, playerID.String(), recipeReq.RecipeID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Không thể xóa Bí phương %s", recipeReq.RecipeID)})
				return
			}

			found := false
			for i := range vaultRecipes {
				if vaultRecipes[i].RecipeID == recipeReq.RecipeID {
					if progress.CompletionRate > vaultRecipes[i].CompletionRate {
						vaultRecipes[i].CompletionRate = progress.CompletionRate
					}
					found = true
					break
				}
			}
			if !found {
				vaultRecipes = append(vaultRecipes, models.SectVaultRecipe{
					RecipeID:       recipeReq.RecipeID,
					CompletionRate: progress.CompletionRate,
				})
			}

			recipeName := recipeReq.RecipeID
			if spec, ok := models.GetGuAny(recipeReq.RecipeID); ok {
				recipeName = spec.Name
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s (%.0f%%)", recipeName, progress.CompletionRate))
		}

		newVaultRecipes, _ := json.Marshal(vaultRecipes)
		_, err = tx.Exec(`UPDATE sects SET vault_recipes = $1 WHERE id = $2`, newVaultRecipes, m.SectID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes"})
			return
		}
	}

	if totalGainedContribution <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm nào được quyên góp."})
		return
	}

	qiGained := 0.0

	m.ContributionPoints += int64(totalGainedContribution)

	// Update Sect funds
	_, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, totalGainedContribution, m.SectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds"})
		return
	}

	// Update player points
	_, err = tx.Exec(`UPDATE sect_members SET contribution_points = $1 WHERE player_id = $2`, m.ContributionPoints, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update contribution"})
		return
	}

	// Update player Human Qi
	_, err = tx.Exec(`UPDATE players SET human_qi = $1 WHERE id = $2`, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update human qi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	// Trigger System Chat Notification on Sect channel
	summaryStr := strings.Join(auditSummaries, ", ")
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("Đạo hữu %s đã quyên góp hàng loạt vật phẩm: %s cho tông môn.", player.Username, summaryStr))

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.donate_bulk", EntityType: "sect",
		EntityID: m.SectID, RequestSummary: fmt.Sprintf("Bulk: %s → %d điểm", summaryStr, totalGainedContribution),
		IPAddress: c.ClientIP(), Success: true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message":             fmt.Sprintf("Quyên góp hàng loạt thành công! Nhân Khí +%.0f.", qiGained),
		"contribution_points": m.ContributionPoints,
	})
}
