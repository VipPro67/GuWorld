package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	_, err = tx.Exec(`
		INSERT INTO sect_vault_materials (sect_id, item_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (sect_id, item_id) DO UPDATE SET
			quantity = sect_vault_materials.quantity + EXCLUDED.quantity,
			updated_at = NOW()
	`, m.SectID, req.MaterialID, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault materials: " + err.Error()})
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
		PlayerID: &playerID, Action: "sect.donate_material", EntityType: "sect",
		EntityID:       m.SectID,
		RequestSummary: fmt.Sprintf("%s x%d → %d điểm cống hiến", materialName, req.Amount, gainedContribution),
		Metadata: map[string]interface{}{
			"material_id":         req.MaterialID,
			"material_name":       materialName,
			"amount":              req.Amount,
			"gained_contribution": gainedContribution,
			"contribution_change": gainedContribution,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Quyên góp thành công %d %s! +%d Điểm cống hiến, Nhân Khí +%.0f.", req.Amount, materialName, gainedContribution, qiGained),
		"member":  m,
	})
}

// DonateGu deposits player Gu into sect vault in exchange for contribution points
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

	// Find the Gu in player inventory
	targetIdx := -1
	var guInstance models.GuInstance
	for i, g := range player.GuInventory {
		if g.InstanceID == req.GuInstanceID {
			targetIdx = i
			guInstance = g
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng không tồn tại trong túi đồ!"})
		return
	}

	if guInstance.IsVital {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể quyên góp Bản Mệnh Cổ!"})
		return
	}

	if guInstance.Equipped {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng tháo Cổ trùng ra trước khi quyên góp!"})
		return
	}

	// Calculate base price & points from gu_dictionary (or in-memory registry)
	var guRank int = guInstance.Rank
	var guBasePrice float64
	guName := guInstance.GuID
	if spec, ok := models.GetGu(guInstance.GuID, guInstance.Rank); ok {
		guRank = spec.Rank
		guBasePrice = spec.BasePrice
		if spec.Name != "" {
			guName = spec.Name
		}
	} else {
		err = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id = $1 AND rank = $2`, guInstance.GuID, guInstance.Rank).Scan(&guBasePrice, &guRank)
		if err != nil {
			guRank = 1
			guBasePrice = 0
		}
	}
	effectiveBase := pricing.GuBasePrice(guBasePrice, guRank)
	gainedContribution := int(pricing.DonatePoints(effectiveBase))
	vaultItemCost := int64(pricing.VaultCost(effectiveBase))

	// Add to sect vault
	itemInstID := guInstance.InstanceID
	if itemInstID == "" {
		itemInstID = uuid.New().String()
	}
	donatedHunger := normalizeVaultGuHunger(guInstance.GuID, guInstance.Rank, float64(guInstance.Hunger))

	// Tiên Cổ rank>=6 giữ instance_id, không gộp quantity: 1 gu_id chỉ 1 row.
	// Kiểm tra cross-rank (loại trừ chính instance đang donate) trước khi nhập kho.
	if guInstance.Rank >= 6 {
		rowID := fmt.Sprintf("%s:%s", m.SectID, itemInstID)
		if exists, err := db.CheckImmortalGuExists(tx, guInstance.GuID, guInstance.Rank, []string{guInstance.InstanceID}); err == nil && exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Tiên Cổ này đã tồn tại ở nơi khác (kho/đấu giá/túi đồ)"})
			return
		}
		var dup int
		_ = tx.QueryRow(`SELECT COUNT(*) FROM sect_vault_gu WHERE gu_id = $1 AND rank >= 6 AND quantity > 0 AND instance_id <> $2`, guInstance.GuID, guInstance.InstanceID).Scan(&dup)
		if dup > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Tiên Cổ này đã tồn tại trong kho tông môn"})
			return
		}
		_, err = tx.Exec(`
			INSERT INTO sect_vault_gu (id, sect_id, gu_id, rank, quantity, cost, hunger, instance_id)
			VALUES ($1, $2, $3, $4, 1, $5, $6, $7)
			ON CONFLICT (id) DO NOTHING
		`, rowID, m.SectID, guInstance.GuID, guInstance.Rank, vaultItemCost, donatedHunger, itemInstID)
	} else {
		rowID := fmt.Sprintf("%s:%s:%d", m.SectID, guInstance.GuID, guInstance.Rank)
		_, err = tx.Exec(`
		INSERT INTO sect_vault_gu (id, sect_id, gu_id, rank, quantity, cost, hunger, instance_id)
		VALUES ($1, $2, $3, $4, 1, $5, $6, '')
		ON CONFLICT (id) DO UPDATE SET
			quantity = sect_vault_gu.quantity + 1,
			cost = EXCLUDED.cost,
			hunger = GREATEST(sect_vault_gu.hunger, EXCLUDED.hunger),
			updated_at = NOW()
	`, rowID, m.SectID, guInstance.GuID, guInstance.Rank, vaultItemCost, donatedHunger)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu: " + err.Error()})
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

	guName = guInstance.GuID
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
	_, err = tx.Exec(`
		INSERT INTO sect_vault_recipes (sect_id, recipe_id, completion_rate)
		VALUES ($1, $2, $3)
		ON CONFLICT (sect_id, recipe_id) DO UPDATE SET
			completion_rate = GREATEST(sect_vault_recipes.completion_rate, EXCLUDED.completion_rate),
			unlocked_at = NOW()
	`, m.SectID, req.RecipeID, progress.CompletionRate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes: " + err.Error()})
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

			// Insert/update into sect_vault_materials
			_, err = tx.Exec(`
				INSERT INTO sect_vault_materials (sect_id, item_id, quantity)
				VALUES ($1, $2, $3)
				ON CONFLICT (sect_id, item_id) DO UPDATE SET
					quantity = sect_vault_materials.quantity + EXCLUDED.quantity,
					updated_at = NOW()
			`, m.SectID, matReq.MaterialID, matReq.Amount)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault materials: " + err.Error()})
				return
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d", materialName, matReq.Amount))
		}
	}

	// 2. Process Gu Instances
	if len(req.Gus) > 0 {
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

				itemInstID := gi.InstanceID
				if itemInstID == "" {
					itemInstID = uuid.New().String()
				}
				rowID := fmt.Sprintf("%s:%s", m.SectID, itemInstID)
				instIDToStore := itemInstID
				if gi.Rank < 6 {
					rowID = fmt.Sprintf("%s:%s:%d", m.SectID, gi.GuID, gi.Rank)
					instIDToStore = ""
				}
				hunger := normalizeVaultGuHunger(gi.GuID, gi.Rank, float64(gi.Hunger))

				_, err = tx.Exec(`
					INSERT INTO sect_vault_gu (id, sect_id, gu_id, rank, quantity, cost, hunger, instance_id)
					VALUES ($1, $2, $3, $4, 1, $5, $6, $7)
					ON CONFLICT (id) DO UPDATE SET
						quantity = sect_vault_gu.quantity + 1,
						cost = EXCLUDED.cost,
						hunger = GREATEST(sect_vault_gu.hunger, EXCLUDED.hunger),
						updated_at = NOW()
				`, rowID, m.SectID, gi.GuID, gi.Rank, vaultItemCost, hunger, instIDToStore)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu: " + err.Error()})
					return
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
	}

	// 3. Process Recipes
	if len(req.Recipes) > 0 {
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

			_, err = tx.Exec(`
				INSERT INTO sect_vault_recipes (sect_id, recipe_id, completion_rate)
				VALUES ($1, $2, $3)
				ON CONFLICT (sect_id, recipe_id) DO UPDATE SET
					completion_rate = GREATEST(sect_vault_recipes.completion_rate, EXCLUDED.completion_rate),
					unlocked_at = NOW()
			`, m.SectID, recipeReq.RecipeID, progress.CompletionRate)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes: " + err.Error()})
				return
			}

			recipeName := recipeReq.RecipeID
			if spec, ok := models.GetGuAny(recipeReq.RecipeID); ok {
				recipeName = spec.Name
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s (%.0f%%)", recipeName, progress.CompletionRate))
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
