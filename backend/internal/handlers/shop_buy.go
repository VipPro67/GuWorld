package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// BuyShopRequest defines the request body for buying items
type BuyShopRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
	Rank     int    `json:"rank"` // 0 = auto (lowest/legacy); >0 = pinned rank variant
}

// BuyShopItem executes purchasing shop goods and updates player inventory
func BuyShopItem(c *gin.Context) {
	var req BuyShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	playerUUID, err := uuid.Parse(req.PlayerID)
	if err != nil || playerUUID != authID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Aperture mismatch"})
		return
	}

	// 1. Verify item is in current cached shop catalog
	var items []ShopItem
	cacheKey := shopCatalogCacheKey()
	if err := cache.GetJSON(cacheKey, &items); err != nil || len(items) == 0 {
		// If cache missing, load it to establish the current catalog
		items, _ = loadCaravanShopCatalog()
		if len(items) > 0 {
			_ = cache.SetJSON(cacheKey, items, 10*time.Minute)
		}
	}

	foundInCatalog := false
	for _, item := range items {
		if item.ID == req.ItemID && (req.Rank == 0 || item.Rank == req.Rank) {
			foundInCatalog = true
			break
		}
	}

	if !foundInCatalog {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vật phẩm này hiện không được bán trong đợt này. Hãy quay lại sau!"})
		return
	}

	// 2. Find product in database to get exact price and stock
	selectedItem, err := findCaravanShopItem(req.ItemID, req.Rank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy vật phẩm này trong CSDL."})
		return
	}

	// 2. Begin DB Transaction
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// 3. Lock & Load Player row
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}
	if player.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể mua sắm!"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	// 3.1 Enforce daily stock for virtual random-pool items (rank<5 Gus + food).
	if selectedItem.IsAuto {
		limit := randomPoolStock(selectedItem.Rank)
		// Ensure a row exists for today (full daily limit) when never sold yet.
		if _, err := tx.Exec(`
			INSERT INTO caravan_shop_daily_stock (day, id, rank, remaining)
			VALUES (CURRENT_DATE, $1, $2, $3)
			ON CONFLICT (day, id, rank) DO NOTHING
		`, selectedItem.ID, selectedItem.Rank, limit); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tồn kho."})
			return
		}
		// Atomically decrement only if enough stock remains (race-safe).
		res, err := tx.Exec(`
			UPDATE caravan_shop_daily_stock
			SET remaining = remaining - $3
			WHERE day = CURRENT_DATE AND id = $1 AND rank = $2 AND remaining >= $3
		`, selectedItem.ID, selectedItem.Rank, req.Quantity)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tồn kho."})
			return
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Hàng này hôm nay đã hết, hãy quay lại vào ngày mai!"})
			return
		}
	}

	// 4. Validate Funds
	currency := selectedItem.Currency
	if currency != "immortal" {
		currency = "primeval"
	}
	totalCost := selectedItem.Price * req.Quantity
	currencyName := "Nguyên Thạch"
	if currency == "immortal" {
		currencyName = "Tiên Nguyên Thạch"
		if player.ImmortalStones < totalCost {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ " + currencyName + " để giao dịch."})
			return
		}
	} else if player.PrimevalStones < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ " + currencyName + " để giao dịch."})
		return
	}

	// Validate Inventory space
	incomingCount := 0
	if selectedItem.Type == "Gu" {
		incomingCount = int(req.Quantity)
	} else {
		qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), selectedItem.ID)
		if qty == 0 {
			incomingCount = 1
		}
	}

	if !HasInventorySpace(&player, incomingCount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bách Bảo Nang đã đầy! Hãy trang bị thêm Thố Ti Hoa Cổ hoặc thăng cấp lên Lục Chuyển."})
		return
	}

	// Deduct currency
	if currency == "immortal" {
		player.ImmortalStones -= totalCost
	} else {
		player.PrimevalStones -= totalCost
	}

	// 5. Add items to player inventory
	switch selectedItem.Type {
	case "Gu":
		logger.S.Infof("Adding %d x %s to player %s inventory", req.Quantity, selectedItem.Name, player.ID)
		spec, ok := models.GetGu(selectedItem.ID, selectedItem.Rank)
		if !ok {
			spec, ok = models.GetGuAny(selectedItem.ID)
		}
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy định nghĩa của Cổ trùng trong Dictionary"})
			return
		}

		for i := int64(0); i < req.Quantity; i++ {
			newInstance := models.GuInstance{
				InstanceID: uuid.New().String(),
				GuID:       spec.ID,
				Rank:       spec.Rank,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   false,
			}
			player.GuInventory = append(player.GuInventory, newInstance)
		}
	case "Material", "Recipe", "RefiningRecipe", "Food":
		if err := InventorySvc.AddItem(tx, player.ID.String(), selectedItem.ID, "materials", req.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thêm vật phẩm vào túi đồ"})
			return
		}
	case "Consumable":
		if err := InventorySvc.AddItem(tx, player.ID.String(), selectedItem.ID, "consumables", req.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thêm vật phẩm tiêu hao vào túi đồ"})
			return
		}
	case "Treasure":
		if err := InventorySvc.AddItem(tx, player.ID.String(), selectedItem.ID, "treasures", req.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thêm bảo vật vào túi đồ"})
			return
		}
	case "ImmortalMaterial":
		if err := InventorySvc.AddItem(tx, player.ID.String(), selectedItem.ID, "immortal_materials", req.Quantity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thêm tiên tài vào túi đồ"})
			return
		}
	}

	// 6. Save player state
	qiGained := 0.0
	if selectedItem.Type == "Gu" {
		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu Cổ trùng mới vào CSDL"})
			return
		}
	}
	updateQuery := `
		UPDATE players
		SET primeval_stones = $1, immortal_stones = $2, human_qi = $3
		WHERE id = $4
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.ImmortalStones, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state"})
		return
	}

	// Commit Transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "shop.buy_item",
		EntityType:     selectedItem.Type,
		EntityID:       selectedItem.ID,
		RequestSummary: fmt.Sprintf("Mua %dx %s | Giá: %d %s", req.Quantity, selectedItem.Name, totalCost, currencyName),
		Metadata: map[string]interface{}{
			"item_id":       selectedItem.ID,
			"item_name":     selectedItem.Name,
			"item_type":     selectedItem.Type,
			"rank":          selectedItem.Rank,
			"quantity":      req.Quantity,
			"total_cost":    totalCost,
			"currency_type": currency,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Giao dịch thành công! Nhân Khí +%.0f.", qiGained)})
}
