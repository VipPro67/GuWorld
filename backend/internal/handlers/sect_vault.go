package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

type VaultGuRow struct {
	ID         string
	GuID       string
	Rank       int
	Cost       int64
	Quantity   int64
	InstanceID string
}

type VaultGuDeductionPlan struct {
	ID     string
	GuID   string
	Rank   int
	Cost   int64
	Deduct int64
	Del    bool
	InstID string
}

// rejectVaultMutationIfShielded seals the sect vault while the Hộ Sơn Đại
// Trận is up: no stocking, donating, buying or destroying items. Returns true
// when the caller must stop (response already written).
func rejectVaultMutationIfShielded(c *gin.Context, tx *sql.Tx, sectID string) bool {
	var active bool
	if err := tx.QueryRow(`SELECT COALESCE(shield_until > NOW(), false) FROM sects WHERE id = $1`, sectID).Scan(&active); err != nil {
		return false
	}
	if !active {
		return false
	}
	c.JSON(http.StatusConflict, gin.H{"error": "Hộ Sơn Đại Trận đang kích hoạt — bảo khố tông môn bị phong tỏa, không thể nhập hoặc rút vật phẩm!"})
	return true
}

func planVaultGuDeductions(matchedRows []VaultGuRow, quantity int64) ([]VaultGuDeductionPlan, int64, error) {
	if quantity <= 0 {
		return nil, 0, fmt.Errorf("Số lượng phải lớn hơn 0")
	}
	var totalAvailable int64
	for _, r := range matchedRows {
		totalAvailable += r.Quantity
	}
	if totalAvailable < quantity || len(matchedRows) == 0 {
		return nil, 0, fmt.Errorf("Cổ trùng này không đủ số lượng hoặc không có sẵn trong Bảo khố!")
	}
	if matchedRows[0].Rank >= 6 && quantity > 1 {
		return nil, 0, fmt.Errorf("Tiên Cổ (rank>=6) mỗi lần chỉ rút 1 instance")
	}
	remaining := quantity
	var totalCost int64
	var plans []VaultGuDeductionPlan
	for _, r := range matchedRows {
		if remaining <= 0 {
			break
		}
		toTake := r.Quantity
		if toTake > remaining {
			toTake = remaining
		}
		totalCost += r.Cost * toTake
		plans = append(plans, VaultGuDeductionPlan{
			ID:     r.ID,
			GuID:   r.GuID,
			Rank:   r.Rank,
			Cost:   r.Cost,
			Deduct: toTake,
			Del:    toTake >= r.Quantity,
			InstID: r.InstanceID,
		})
		remaining -= toTake
	}
	return plans, totalCost, nil
}

// stockItemUnitCost computes the sect-funds unit cost for stocking an item and
// enforces the rank gates (10 sect levels = 1 rank level, up to rank 9).
// Returns (unitCost, effectiveRank, httpStatusOK, "") on success; otherwise (0, 0, status, errorMessage).
func stockItemUnitCost(tx *sql.Tx, sectLevel int, itemType, itemID string) (int64, int, int, string) {
	itemType = strings.ToLower(itemType)
	if itemType == "consumable" || itemType == "currency" {
		return 0, 0, http.StatusForbidden, "Không thể nhập vật phẩm loại Tiêu hao (Consumable) hoặc Tiền tệ (Currency) vào bảo khố tông môn."
	}
	if itemType == "gu" {
		return 0, 0, http.StatusForbidden, "Không thể nhập Cổ trùng vào bảo khố."
	}

	maxLevel := sectLevel/10 + 1
	if maxLevel > 9 {
		maxLevel = 9
	}

	if itemType == "material" {
		var matRank, matRarity int
		var matBasePrice float64
		var dbItemType string
		var err error = sql.ErrNoRows
		if tx != nil {
			err = tx.QueryRow(`SELECT COALESCE(rank, 1), COALESCE(rarity, 0), COALESCE(base_price, 0), COALESCE(item_type, '') FROM item_dictionary WHERE id = $1`, itemID).Scan(&matRank, &matRarity, &matBasePrice, &dbItemType)
		}
		if err != nil {
			if spec, ok := models.ItemRegistry[itemID]; ok {
				matRank = spec.Rank
				matRarity = spec.Rarity
				matBasePrice = spec.BasePrice
				dbItemType = spec.Type
				if dbItemType == "" {
					dbItemType = spec.ItemType
				}
			} else {
				return 0, 0, http.StatusBadRequest, "Vật phẩm không tồn tại trong từ điển"
			}
		}

		if strings.EqualFold(dbItemType, "Consumable") || strings.EqualFold(dbItemType, "Currency") {
			return 0, 0, http.StatusForbidden, "Không thể nhập vật phẩm thuộc loại Tiêu hao (Consumable) hoặc Tiền tệ (Currency) vào bảo khố tông môn."
		}

		effectiveLevel := matRank
		if matRarity > effectiveLevel {
			effectiveLevel = matRarity
		}

		if effectiveLevel > maxLevel {
			return 0, 0, http.StatusForbidden, fmt.Sprintf("Tông môn chưa đạt cấp %d để nhập Linh tài hạng %d! (Hiện tại: Cấp %d)", (effectiveLevel-1)*10, effectiveLevel, sectLevel)
		}

		baseCost := pricing.ItemBasePrice(matBasePrice, effectiveLevel)
		if effectiveLevel >= 6 {
			baseCost *= 1000.0
		}
		unitCost := int64(math.Round(baseCost * 1.5))
		if unitCost < 1 {
			unitCost = 1
		}
		return unitCost, effectiveLevel, http.StatusOK, ""
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
			return 0, 0, http.StatusBadRequest, "Bí phương không tồn tại trong tàng kinh các"
		}

		effectiveLevel := targetGuRank
		if targetGuRarity > effectiveLevel {
			effectiveLevel = targetGuRarity
		}

		if effectiveLevel > maxLevel {
			return 0, 0, http.StatusForbidden, fmt.Sprintf("Tông môn chưa đạt cấp %d để nhập Bí phương hạng %d! (Hiện tại: Cấp %d)", (effectiveLevel-1)*10, effectiveLevel, sectLevel)
		}

		baseCost := float64(pricing.RecipeCostByGuRank(effectiveLevel))
		if effectiveLevel >= 6 {
			baseCost *= 1000.0
		}
		unitCost := int64(math.Round(baseCost * 1.5))
		if unitCost < 1 {
			unitCost = 1
		}
		return unitCost, effectiveLevel, http.StatusOK, ""
	}

	return 0, 0, http.StatusBadRequest, "Loại vật phẩm không hợp lệ (chỉ hỗ trợ gu, material hoặc recipe)"
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
		SELECT id, funds, level
		FROM sects
		WHERE id = $1
		FOR UPDATE
	`, sectID).Scan(&s.ID, &s.Funds, &s.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details: " + err.Error()})
		return
	}

	if rejectVaultMutationIfShielded(c, tx, sectID) {
		return
	}

	unitCost, effectiveRank, code, msg := stockItemUnitCost(tx, s.Level, req.ItemType, req.ItemID)
	if code != http.StatusOK {
		c.JSON(code, gin.H{"error": msg})
		return
	}
	itemType := strings.ToLower(req.ItemType)

	if effectiveRank >= 6 {
		if req.Quantity > 5 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi lần chỉ được nhập tối đa 5 Tiên phẩm cùng loại!"})
			return
		}
		if itemType == "material" {
			var currentInVault int64
			_ = tx.QueryRow(`SELECT COALESCE(quantity, 0) FROM sect_vault_materials WHERE sect_id = $1 AND item_id = $2`, s.ID, req.ItemID).Scan(&currentInVault)
			if currentInVault+req.Quantity > 200 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bảo khố chỉ có thể lưu trữ tối đa 200 đơn vị Tiên phẩm cùng loại (hiện có: %d)!", currentInVault)})
				return
			}
		}
	}

	totalCost := int64(unitCost) * int64(req.Quantity)
	if s.Funds < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ! Cần %d quỹ tông môn", totalCost)})
		return
	}

	// Deduct funds
	s.Funds -= totalCost
	_, err = tx.Exec(`UPDATE sects SET funds = $1 WHERE id = $2`, s.Funds, s.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds: " + err.Error()})
		return
	}

	// Update vault relational tables
	if itemType == "gu" {
		var stockGuRank int
		var stockGuBasePrice float64
		_ = tx.QueryRow(`SELECT COALESCE(rank,1), COALESCE(base_price,0) FROM gu_dictionary WHERE id = $1 ORDER BY rank DESC LIMIT 1`, req.ItemID).Scan(&stockGuRank, &stockGuBasePrice)
		if stockGuRank == 0 {
			stockGuRank = 1
		}
		var stockInstID string
		var rowID string
		if stockGuRank >= 6 {
			stockInstID = uuid.New().String()
			rowID = fmt.Sprintf("%s:%s", s.ID, stockInstID)
		} else {
			rowID = fmt.Sprintf("%s:%s:%d", s.ID, req.ItemID, stockGuRank)
		}

		_, err = tx.Exec(`
			INSERT INTO sect_vault_gu (id, sect_id, gu_id, rank, quantity, instance_id)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				quantity = sect_vault_gu.quantity + EXCLUDED.quantity,
				updated_at = NOW()
		`, rowID, s.ID, req.ItemID, stockGuRank, req.Quantity, stockInstID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu: " + err.Error()})
			return
		}
	} else if itemType == "recipe" {
		_, err = tx.Exec(`
			INSERT INTO sect_vault_recipes (sect_id, recipe_id, completion_rate)
			VALUES ($1, $2, 100.0)
			ON CONFLICT (sect_id, recipe_id) DO UPDATE SET
				completion_rate = 100.0, unlocked_at = NOW()
		`, s.ID, req.ItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes: " + err.Error()})
			return
		}
	} else {
		_, err = tx.Exec(`
			INSERT INTO sect_vault_materials (sect_id, item_id, quantity)
			VALUES ($1, $2, $3)
			ON CONFLICT (sect_id, item_id) DO UPDATE SET
				quantity = sect_vault_materials.quantity + EXCLUDED.quantity,
				updated_at = NOW()
		`, s.ID, req.ItemID, req.Quantity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault materials: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
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
		itemName := req.ItemID
		if itemType == "gu" {
			if spec, ok := models.GetGuAny(req.ItemID); ok {
				itemName = spec.Name
			}
		} else if itemType == "material" {
			_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, req.ItemID).Scan(&itemName)
		}
		if itemName == "" {
			itemName = req.ItemID
		}
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("%s %s đã mua thêm %d %s vào bảo khố tông môn. Tiêu hao %d Quỹ, còn lại %d Quỹ.", rankName, username, req.Quantity, itemName, totalCost, s.Funds))
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.stock_vault", EntityType: "sect",
		EntityID:       sectID,
		RequestSummary: fmt.Sprintf("%s | %s x%d | tổng giá %d quỹ tông môn", req.ItemType, req.ItemID, req.Quantity, totalCost),
		Metadata: map[string]interface{}{
			"item_type":    req.ItemType,
			"item_id":      req.ItemID,
			"quantity":     req.Quantity,
			"total_cost":   totalCost,
			"funds_change": -totalCost,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{
		"message":   "Nhập thêm vật phẩm vào bảo khố thành công!",
		"new_funds": s.Funds,
	})
}

type DestroyVaultRequest struct {
	ItemType string `json:"item_type" binding:"required"` // gu, material, recipe
	ItemID   string `json:"item_id" binding:"required"`
	Rank     int    `json:"rank"` // required for gu
}

// DestroySectVaultItem permanently deletes a whole vault stack: a Gu by
// gu_id+rank, a material item, or a recipe. Restricted to Chưởng Môn / Hộ Pháp.
// There is no refund of sect funds or contribution points.
func DestroySectVaultItem(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req DestroyVaultRequest
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

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}
	if !canDestroyVaultItem(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ Chưởng Môn hoặc Hộ Pháp mới có quyền tiêu huỷ vật phẩm trong bảo khố!"})
		return
	}

	if rejectVaultMutationIfShielded(c, tx, sectID) {
		return
	}

	var affected int64
	switch strings.ToLower(req.ItemType) {
	case "gu":
		if req.Rank <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu phẩm cấp Cổ trùng cần tiêu huỷ!"})
			return
		}
		res, err := tx.Exec(`DELETE FROM sect_vault_gu WHERE sect_id = $1 AND gu_id = $2 AND rank = $3`, sectID, req.ItemID, req.Rank)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tiêu huỷ: " + err.Error()})
			return
		}
		affected, _ = res.RowsAffected()
	case "material":
		res, err := tx.Exec(`DELETE FROM sect_vault_materials WHERE sect_id = $1 AND item_id = $2`, sectID, req.ItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tiêu huỷ: " + err.Error()})
			return
		}
		affected, _ = res.RowsAffected()
	case "recipe":
		res, err := tx.Exec(`DELETE FROM sect_vault_recipes WHERE sect_id = $1 AND recipe_id = $2`, sectID, req.ItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tiêu huỷ: " + err.Error()})
			return
		}
		affected, _ = res.RowsAffected()
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại vật phẩm không hợp lệ (gu, material, recipe)"})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy vật phẩm trong bảo khố!"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.destroy_vault_item", EntityType: "sect",
		EntityID:       sectID,
		RequestSummary: fmt.Sprintf("%s | %s rank=%d", req.ItemType, req.ItemID, req.Rank),
		Metadata: map[string]interface{}{
			"item_type": req.ItemType,
			"item_id":   req.ItemID,
			"rank":      req.Rank,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã tiêu huỷ vật phẩm khỏi bảo khố tông môn!"})
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
	err = tx.QueryRow(`SELECT id, funds, level FROM sects WHERE id = $1 FOR UPDATE`, sectID).Scan(&s.ID, &s.Funds, &s.Level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load sect details: " + err.Error()})
		return
	}

	if rejectVaultMutationIfShielded(c, tx, sectID) {
		return
	}

	type stockLine struct {
		itemType      string
		itemID        string
		quantity      int64
		unitCost      int64
		effectiveRank int
	}
	lines := make([]stockLine, 0, len(req.Items))
	var totalCost int64
	plannedImmortalQty := make(map[string]int64)
	for _, it := range req.Items {
		unitCost, effectiveRank, code, msg := stockItemUnitCost(tx, s.Level, it.ItemType, it.ItemID)
		if code != http.StatusOK {
			c.JSON(code, gin.H{"error": msg})
			return
		}
		if effectiveRank >= 6 {
			if it.Quantity > 5 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Mỗi lần chỉ được nhập tối đa 5 Tiên phẩm cùng loại: %s", it.ItemID)})
				return
			}
			if strings.ToLower(it.ItemType) == "material" {
				var currentInVault int64
				_ = tx.QueryRow(`SELECT COALESCE(quantity, 0) FROM sect_vault_materials WHERE sect_id = $1 AND item_id = $2`, s.ID, it.ItemID).Scan(&currentInVault)
				if currentInVault+plannedImmortalQty[it.ItemID]+it.Quantity > 200 {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Bảo khố chỉ có thể lưu trữ tối đa 200 đơn vị Tiên phẩm cùng loại (%s hiện có: %d)!", it.ItemID, currentInVault+plannedImmortalQty[it.ItemID])})
					return
				}
				plannedImmortalQty[it.ItemID] += it.Quantity
			}
		}
		lines = append(lines, stockLine{strings.ToLower(it.ItemType), it.ItemID, it.Quantity, unitCost, effectiveRank})
		totalCost += unitCost * it.Quantity
	}

	if s.Funds < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quỹ tông môn không đủ! Cần %d quỹ tông môn", totalCost)})
		return
	}
	s.Funds -= totalCost
	_, err = tx.Exec(`UPDATE sects SET funds = $1 WHERE id = $2`, s.Funds, s.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect funds: " + err.Error()})
		return
	}

	for _, line := range lines {
		if line.itemType == "gu" {
			var stockGuRank int
			var stockGuBasePrice float64
			_ = tx.QueryRow(`SELECT COALESCE(rank,1), COALESCE(base_price,0) FROM gu_dictionary WHERE id = $1 ORDER BY rank DESC LIMIT 1`, line.itemID).Scan(&stockGuRank, &stockGuBasePrice)
			if stockGuRank == 0 {
				stockGuRank = 1
			}
			var stockInstID string
			var rowID string
			if stockGuRank >= 6 {
				stockInstID = uuid.New().String()
				rowID = fmt.Sprintf("%s:%s", s.ID, stockInstID)
			} else {
				rowID = fmt.Sprintf("%s:%s:%d", s.ID, line.itemID, stockGuRank)
			}

			_, err = tx.Exec(`
				INSERT INTO sect_vault_gu (id, sect_id, gu_id, rank, quantity, instance_id)
				VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (id) DO UPDATE SET
					quantity = sect_vault_gu.quantity + EXCLUDED.quantity,
					updated_at = NOW()
			`, rowID, s.ID, line.itemID, stockGuRank, line.quantity, stockInstID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault Gu: " + err.Error()})
				return
			}
		} else if line.itemType == "recipe" {
			_, err = tx.Exec(`
				INSERT INTO sect_vault_recipes (sect_id, recipe_id, completion_rate)
				VALUES ($1, $2, 100.0)
				ON CONFLICT (sect_id, recipe_id) DO UPDATE SET
					completion_rate = 100.0, unlocked_at = NOW()
			`, s.ID, line.itemID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes: " + err.Error()})
				return
			}
		} else {
			_, err = tx.Exec(`
				INSERT INTO sect_vault_materials (sect_id, item_id, quantity)
				VALUES ($1, $2, $3)
				ON CONFLICT (sect_id, item_id) DO UPDATE SET
					quantity = sect_vault_materials.quantity + EXCLUDED.quantity,
					updated_at = NOW()
			`, s.ID, line.itemID, line.quantity)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault materials: " + err.Error()})
				return
			}
		}
	}

	itemSummaries := make([]string, 0, len(lines))
	for _, line := range lines {
		itemSummaries = append(itemSummaries, fmt.Sprintf("%s:%s x%d", line.itemType, line.itemID, line.quantity))
	}
	auditlog.Log(tx, auditlog.Entry{
		PlayerID: &playerID, Action: "sect.stock_vault_bulk", EntityType: "sect",
		EntityID:       sectID,
		RequestSummary: fmt.Sprintf("%d mặt hàng | tổng giá %d quỹ tông môn | %s", len(lines), totalCost, strings.Join(itemSummaries, ", ")),
		Metadata: map[string]interface{}{
			"item_count":   len(lines),
			"items":        itemSummaries,
			"total_cost":   totalCost,
			"funds_change": -totalCost,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

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
		_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("%s %s đã nhập thêm %d mặt hàng vào bảo khố tông môn. Tiêu hao %d Quỹ, còn lại %d Quỹ.", rankName, username, len(lines), totalCost, s.Funds))
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

	if rejectVaultMutationIfShielded(c, tx, m.SectID) {
		return
	}

	var cost int64
	learnedRecipe := false
	if strings.ToLower(req.ItemType) == "gu" {
		if req.Rank >= 6 && quantity > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Cổ (rank>=6) mỗi lần chỉ rút 1 instance"})
			return
		}

		rows, err := tx.Query(`
			SELECT id, gu_id, rank, quantity, COALESCE(instance_id, '')
			FROM sect_vault_gu
			WHERE sect_id = $1 
			  AND (gu_id = $2 OR LOWER(gu_id) = LOWER($2) OR id = $2 OR instance_id = $2)
			  AND ($3 = 0 OR rank = $3)
			  AND quantity > 0
			ORDER BY rank ASC, id ASC
			FOR UPDATE
		`, m.SectID, req.ItemID, req.Rank)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn bảo khố: " + err.Error()})
			return
		}

		var matchedRows []VaultGuRow
		for rows.Next() {
			var r VaultGuRow
			if err := rows.Scan(&r.ID, &r.GuID, &r.Rank, &r.Quantity, &r.InstanceID); err != nil {
				rows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc dữ liệu bảo khố: " + err.Error()})
				return
			}
			r.Cost = sectVaultGuCost(r.GuID, r.Rank)
			matchedRows = append(matchedRows, r)
		}
		rows.Close()

		plans, totalCost, planErr := planVaultGuDeductions(matchedRows, quantity)
		if planErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": planErr.Error()})
			return
		}
		cost = totalCost

		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost
		for _, p := range plans {
			if p.Del {
				_, err = tx.Exec(`DELETE FROM sect_vault_gu WHERE id = $1`, p.ID)
			} else {
				_, err = tx.Exec(`UPDATE sect_vault_gu SET quantity = quantity - $1, updated_at = NOW() WHERE id = $2`, p.Deduct, p.ID)
			}
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct Gu from vault: " + err.Error()})
				return
			}
		}

		for _, p := range plans {
			guSpec, ok := models.GetGu(p.GuID, p.Rank)
			if !ok {
				guSpec, _ = models.GetGuAny(p.GuID)
			}
			actualRank := p.Rank
			if actualRank == 0 {
				actualRank = guSpec.Rank
			}
			for i := int64(0); i < p.Deduct; i++ {
				newGu := createVaultWithdrawnGu(p.GuID, actualRank, guSpec)
				if p.InstID != "" && p.Rank >= 6 {
					newGu.InstanceID = p.InstID
				}
				player.GuInventory = append(player.GuInventory, newGu)

				if p.Rank >= 6 {
					guName := p.GuID
					if guSpec.Name != "" {
						guName = guSpec.Name
					}
					withdrawSummary := fmt.Sprintf("Cổ Sư [%s] rút/đổi Cổ trùng [%s] R%d từ Kho Tông Môn.", player.Username, guName, actualRank)
					_ = db.LogGuHistory(tx, newGu.InstanceID, newGu.GuID, newGu.Rank, "SECT_VAULT_WITHDRAWN", nil, &player.ID, withdrawSummary, map[string]interface{}{
						"sect_id": m.SectID,
						"cost":    cost,
						"gu_name": guName,
					})
				}
			}
		}

		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory"})
			return
		}

	} else if strings.ToLower(req.ItemType) == "material" {
		var matRank int
		var matBasePrice float64
		_ = tx.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, req.ItemID).Scan(&matBasePrice, &matRank)
		baseCost := pricing.VaultCost(pricing.ItemBasePrice(matBasePrice, matRank))
		if matRank >= 6 {
			baseCost *= 1000
		}
		cost = baseCost * quantity

		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		res, err := tx.Exec(`
			UPDATE sect_vault_materials 
			SET quantity = quantity - $1, updated_at = NOW() 
			WHERE sect_id = $2 AND item_id = $3 AND quantity >= $1
		`, quantity, m.SectID, req.ItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update vault materials: " + err.Error()})
			return
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Linh tài này không đủ số lượng trong Bảo khố!"})
			return
		}
		_, _ = tx.Exec(`DELETE FROM sect_vault_materials WHERE sect_id = $1 AND item_id = $2 AND quantity <= 0`, m.SectID, req.ItemID)

		m.ContributionPoints -= cost
		InventorySvc.AddItem(tx, player.ID.String(), req.ItemID, "materials", quantity)

	} else if strings.ToLower(req.ItemType) == "recipe" {
		var vrRecipeID string
		var vrCompletionRate float64
		err = tx.QueryRow(`
			SELECT recipe_id, completion_rate 
			FROM sect_vault_recipes 
			WHERE sect_id = $1 AND recipe_id = $2 
			FOR UPDATE
		`, m.SectID, req.ItemID).Scan(&vrRecipeID, &vrCompletionRate)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Công thức này không có sẵn trong Bảo khố!"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}

		if quantity > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Công thức mỗi lần chỉ có thể học một lần!"})
			return
		}

		// Rank from the catalog tuple; parsing the id is the legacy fallback.
		targetGuRank := recipeTargetGuRank(tx, req.ItemID)
		baseRecipeCost := pricing.RecipeCostByGuRank(targetGuRank)
		if targetGuRank >= 6 {
			baseRecipeCost *= 1000
		}
		cost = int64(baseRecipeCost)

		if m.ContributionPoints < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Điểm cống hiến!"})
			return
		}

		m.ContributionPoints -= cost

		_, err = tx.Exec(`DELETE FROM sect_vault_recipes WHERE sect_id = $1 AND recipe_id = $2`, m.SectID, req.ItemID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sect vault recipes: " + err.Error()})
			return
		}

		grantedRate := vrCompletionRate
		var existingProgress *services.RecipeProgress
		if existing, getErr := RecipeSvc.Get(tx, player.ID.String(), req.ItemID); getErr == nil && existing != nil {
			existingProgress = existing
			if existing.CompletionRate > grantedRate {
				grantedRate = existing.CompletionRate
			}
		}
		if grantedRate > 100.0 {
			grantedRate = 100.0
		}
		newProgress := &services.RecipeProgress{
			PlayerID:       player.ID.String(),
			RecipeID:       req.ItemID,
			CompletionRate: grantedRate,
			IsUnlocked:     true,
		}
		// Keep the player's deduce history/favourite instead of resetting it
		// when the vault copy only improves the completion rate.
		if existingProgress != nil {
			newProgress.DeduceCount = existingProgress.DeduceCount
			newProgress.LastDeduceAt = existingProgress.LastDeduceAt
			newProgress.Favorite = existingProgress.Favorite
		}
		if err := RecipeSvc.Upsert(tx, newProgress); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể học công thức từ bảo khố"})
			return
		}
		learnedRecipe = true
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
	switch strings.ToLower(req.ItemType) {
	case "gu":
		if spec, ok := models.GetGu(req.ItemID, req.Rank); ok && spec.Name != "" {
			itemName = spec.Name
		} else if spec, ok := models.GetGuAny(req.ItemID); ok && spec.Name != "" {
			itemName = spec.Name
		}
	case "recipe":
		if _, _, _, _, catalogName, ok := recipeCatalogInfo(db.DB, req.ItemID); ok && catalogName != "" {
			itemName = catalogName
		}
	default:
		_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, req.ItemID).Scan(&itemName)
	}
	if itemName == "" {
		itemName = req.ItemID
	}
	_ = SendSystemMessage("sect", sql.NullString{String: m.SectID, Valid: true}, fmt.Sprintf("[Bảo Khố] Đạo hữu %s đã đổi %s (x%d) từ Bảo khố (tiêu hao %d Điểm cống hiến).", player.Username, itemName, quantity, cost))
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
	resp := gin.H{
		"message": msg,
		"member":  m,
	}
	if learnedRecipe {
		// Read after commit so the freshly learned row is visible, and tell
		// other sessions/tabs to refresh their cached recipe list.
		resp["unlocked_recipes"] = loadUnlockedRecipes(playerID.String())
		respondActionWithInventoryDomains(c, http.StatusOK, player, []PlayerEventDomain{RecipesDomain}, resp)
		return
	}
	respondActionWithInventory(c, http.StatusOK, player, resp)
}
