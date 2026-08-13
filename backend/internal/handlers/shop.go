package handlers

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

// ShopItem represents a product in the shop
type ShopItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "Gu", "Material", or "Consumable"
	Price       int    `json:"price"`
	Stock       int    `json:"stock"` // -1 for unlimited
}

func loadCaravanShopCatalog() ([]ShopItem, error) {
	rows, err := db.DB.Query(`
		SELECT c.id, 
		       COALESCE(i.name, g.name, 'Công thức ' || rg.name, c.id) AS name, 
		       COALESCE(i.description, g.description, '') AS description,
		       CASE WHEN r.target_gu_id IS NOT NULL THEN 'Recipe' ELSE COALESCE(i.item_type, 'Gu') END AS type, 
		       c.price, c.stock, c.is_default
		FROM caravan_shop_items c
		LEFT JOIN item_dictionary i ON c.id = i.id
		LEFT JOIN gu_dictionary g ON c.id = g.id
		LEFT JOIN refining_recipes r ON c.id = ('recipe_' || r.target_gu_id || '_' || r.target_rank::text || '_' || r.variant::text)
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		ORDER BY c.sort_order ASC, c.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ShopItem{}
	for rows.Next() {
		var item ShopItem
		var isDefault bool
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Type, &item.Price, &item.Stock, &isDefault); err != nil {
			return nil, err
		}

		if isDefault {
			items = append(items, item)
		} else {
			// 30% chance to appear
			if rand.Float32() < 0.3 {
				items = append(items, item)
			}
		}
	}
	return items, rows.Err()
}

func findCaravanShopItem(itemID string) (*ShopItem, error) {
	var item ShopItem
	err := db.DB.QueryRow(`
		SELECT c.id, 
		       COALESCE(i.name, g.name, 'Công thức ' || rg.name, c.id) AS name, 
		       COALESCE(i.description, g.description, '') AS description,
		       CASE WHEN r.target_gu_id IS NOT NULL THEN 'Recipe' ELSE COALESCE(i.item_type, 'Gu') END AS type, 
		       c.price, c.stock
		FROM caravan_shop_items c
		LEFT JOIN item_dictionary i ON c.id = i.id
		LEFT JOIN gu_dictionary g ON c.id = g.id
		LEFT JOIN refining_recipes r ON c.id = ('recipe_' || r.target_gu_id || '_' || r.target_rank::text || '_' || r.variant::text)
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		WHERE c.id = $1
	`, itemID).Scan(&item.ID, &item.Name, &item.Description, &item.Type, &item.Price, &item.Stock)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// GetShopList returns the database-backed shop inventory, cached in Redis.
func GetShopList(c *gin.Context) {
	var items []ShopItem
	cacheKey := "caravan_shop_catalog"

	if err := cache.GetJSON(cacheKey, &items); err == nil && len(items) > 0 {
		c.JSON(http.StatusOK, items)
		return
	}

	items, err := loadCaravanShopCatalog()
	if err != nil {
		logger.S.Errorf("Error loading shop catalog: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách Thương Đội"})
		return
	}

	_ = cache.SetJSON(cacheKey, items, 10*time.Minute)
	c.JSON(http.StatusOK, items)
}

// BuyShopRequest defines the request body for buying items
type BuyShopRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
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
	cacheKey := "caravan_shop_catalog"
	if err := cache.GetJSON(cacheKey, &items); err != nil || len(items) == 0 {
		// If cache missing, load it to establish the current catalog
		items, _ = loadCaravanShopCatalog()
		if len(items) > 0 {
			_ = cache.SetJSON(cacheKey, items, 10*time.Minute)
		}
	}

	foundInCatalog := false
	for _, item := range items {
		if item.ID == req.ItemID {
			foundInCatalog = true
			break
		}
	}

	if !foundInCatalog {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vật phẩm này hiện không được bán trong đợt này. Hãy quay lại sau!"})
		return
	}

	// 2. Find product in database to get exact price and stock
	selectedItem, err := findCaravanShopItem(req.ItemID)
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
	before := append(models.GuInventory(nil), player.GuInventory...)

	// 4. Validate Funds
	totalCost := float64(selectedItem.Price * req.Quantity)
	if player.PrimevalStones < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch để giao dịch."})
		return
	}

	// Validate Inventory space
	incomingCount := 0
	if selectedItem.Type == "Gu" {
		incomingCount = req.Quantity
	} else {
		qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), selectedItem.ID, "materials")
		if qty == 0 {
			incomingCount = 1
		}
	}

	if !HasInventorySpace(&player, incomingCount) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bách Bảo Nang đã đầy! Hãy trang bị thêm Thố Ti Hoa Cổ hoặc thăng cấp lên Lục Chuyển."})
		return
	}

	// Deduct currency
	player.PrimevalStones -= totalCost

	// 5. Add items to player inventory
	switch selectedItem.Type {
	case "Gu":
		logger.S.Infof("Adding %d x %s to player %s inventory", req.Quantity, selectedItem.Name, player.ID)
		spec, ok := models.GetGuAny(selectedItem.ID)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy định nghĩa của Cổ trùng trong Dictionary"})
			return
		}

		for i := 0; i < req.Quantity; i++ {
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
	qiGained := AddHumanQi(&player, 5.0)
	if selectedItem.Type == "Gu" {
		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu Cổ trùng mới vào CSDL"})
			return
		}
	}
	updateQuery := `
		UPDATE players
		SET primeval_stones = $1, human_qi = $2
		WHERE id = $3
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.HumanQi, player.ID)
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
		PlayerID: &authID, Action: "shop.buy", EntityType: "shop_item",
		EntityID:       selectedItem.ID,
		RequestSummary: fmt.Sprintf("%s x%d | %.0f Nguyên Thạch", selectedItem.Name, req.Quantity, totalCost),
		IPAddress:      c.ClientIP(), Success: true,
	})
	player.MaterialsInventory = loadMaterialsInventory(player.ID.String())
	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Giao dịch thành công! Nhân Khí +%.0f.", qiGained)})
}

type SellShopRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
	ItemType string `json:"item_type" binding:"required"`
}

func SellShopItem(c *gin.Context) {
	var req SellShopRequest
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

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}

	totalGain := 0.0
	var removedIDs []string
	if req.ItemType == "Material" || req.ItemType == "Consumable" {
		cat := "materials"
		if req.ItemType == "Consumable" {
			cat = "consumables"
		}
		qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), req.ItemID, cat)
		if qty < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ vật phẩm để bán"})
			return
		}
		InventorySvc.RemoveItem(tx, player.ID.String(), req.ItemID, cat, req.Quantity)
		// Resolve sell price: prefer shop catalog price, fallback to item_dictionary rank
		var unitSellPrice int
		if shopItem, err := findCaravanShopItem(req.ItemID); err == nil {
			unitSellPrice = pricing.SellPrice(shopItem.Price)
		} else {
			var dbBasePrice, rank int
			_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, req.ItemID).Scan(&dbBasePrice, &rank)
			unitSellPrice = pricing.SellPrice(pricing.ItemBasePrice(dbBasePrice, rank))
		}
		totalGain = float64(unitSellPrice * req.Quantity)
	} else if req.ItemType == "Gu" {
		removed := 0
		newInv := models.GuInventory{}
		for _, gu := range player.GuInventory {
			if gu.GuID == req.ItemID && !gu.IsVital && removed < req.Quantity {
				removed++
				removedIDs = append(removedIDs, gu.InstanceID)
				continue
			}
			newInv = append(newInv, gu)
		}
		if removed < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Cổ trùng để bán hoặc Cổ là Bản Mệnh"})
			return
		}
		player.GuInventory = newInv
		// Resolve sell price: prefer shop catalog price, fallback to gu_dictionary rank
		var unitSellPrice int
		if shopItem, err := findCaravanShopItem(req.ItemID); err == nil {
			unitSellPrice = pricing.SellPrice(shopItem.Price)
		} else {
			var dbBasePrice, rank int
			_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1`, req.ItemID).Scan(&dbBasePrice, &rank)
			unitSellPrice = pricing.SellPrice(pricing.GuBasePrice(dbBasePrice, rank))
		}
		totalGain = float64(unitSellPrice * req.Quantity)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_type phải là Gu hoặc Material"})
		return
	}

	player.PrimevalStones += totalGain
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}
	qiGained := AddHumanQi(&player, 5.0)
	if req.ItemType == "Gu" {
		if err := db.RemovePlayerGus(tx, removedIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory changes"})
			return
		}
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones=$1, equipped_gus=$2, human_qi=$3 WHERE id=$4`,
		player.PrimevalStones, player.EquippedGus, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save sell transaction"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
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
		PlayerID: &authID, Action: "shop.sell", EntityType: "shop_item",
		EntityID:       req.ItemID,
		RequestSummary: fmt.Sprintf("%s x%d | nhận %.0f Nguyên Thạch", itemName, req.Quantity, totalGain),
		IPAddress:      c.ClientIP(), Success: true,
	})
	player.MaterialsInventory = loadMaterialsInventory(player.ID.String())
	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Bán thành công, nhận Nguyên Thạch! Nhân Khí +%.0f.", qiGained)})
}

type SellShopItemInfo struct {
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int    `json:"quantity" binding:"required,min=1"`
	ItemType string `json:"item_type" binding:"required"`
}

type SellShopBulkRequest struct {
	PlayerID string             `json:"player_id" binding:"required"`
	Items    []SellShopItemInfo `json:"items" binding:"required,dive"`
}

func SellShopItemsBulk(c *gin.Context) {
	var req SellShopBulkRequest
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

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm nào được chọn"})
		return
	}
	if len(req.Items) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi lần thanh lý tối đa 50 loại vật phẩm!"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player Aperture"})
		return
	}

	totalGain := 0.0
	guSold := false
	var removedIDs []string
	var auditSummaries []string

	for _, item := range req.Items {
		if item.ItemType == "Material" || item.ItemType == "Consumable" {
			cat := "materials"
			if item.ItemType == "Consumable" {
				cat = "consumables"
			}
			qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), item.ItemID, cat)
			if qty < item.Quantity {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ vật phẩm %s để bán", item.ItemID)})
				return
			}
			InventorySvc.RemoveItem(tx, player.ID.String(), item.ItemID, cat, item.Quantity)

			var unitSellPrice int
			if shopItem, err := findCaravanShopItem(item.ItemID); err == nil {
				unitSellPrice = pricing.SellPrice(shopItem.Price)
			} else {
				var dbBasePrice, rank int
				_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, item.ItemID).Scan(&dbBasePrice, &rank)
				unitSellPrice = pricing.SellPrice(pricing.ItemBasePrice(dbBasePrice, rank))
			}
			gain := float64(unitSellPrice * item.Quantity)
			totalGain += gain

			var name string
			_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, item.ItemID).Scan(&name)
			if name == "" {
				name = item.ItemID
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d (+%.0f)", name, item.Quantity, gain))

		} else if item.ItemType == "Gu" {
			removed := 0
			var newInv models.GuInventory
			for _, gu := range player.GuInventory {
				if gu.GuID == item.ItemID && !gu.IsVital && !gu.Equipped && gu.Status != "DEAD" && removed < item.Quantity {
					removed++
					removedIDs = append(removedIDs, gu.InstanceID)
					continue
				}
				newInv = append(newInv, gu)
			}
			if removed < item.Quantity {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Cổ trùng %s để bán hoặc Cổ là Bản Mệnh/Đang thiết bị/Đã chết", item.ItemID)})
				return
			}
			player.GuInventory = newInv
			guSold = true

			var unitSellPrice int
			if shopItem, err := findCaravanShopItem(item.ItemID); err == nil {
				unitSellPrice = pricing.SellPrice(shopItem.Price)
			} else {
				var dbBasePrice, rank int
				_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1`, item.ItemID).Scan(&dbBasePrice, &rank)
				unitSellPrice = pricing.SellPrice(pricing.GuBasePrice(dbBasePrice, rank))
			}
			gain := float64(unitSellPrice * item.Quantity)
			totalGain += gain

			var name string
			if spec, ok := models.GetGuAny(item.ItemID); ok {
				name = spec.Name
			}
			if name == "" {
				name = item.ItemID
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d (+%.0f)", name, item.Quantity, gain))
		} else if item.ItemType == "Recipe" {
			recipeToSell, err := RecipeSvc.Get(tx, player.ID.String(), item.ItemID)
			if err != nil || recipeToSell == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không tìm thấy Bí phương %s trong túi đồ", item.ItemID)})
				return
			}
			// Remove recipe from seller's unlocked list
			RecipeSvc.Delete(tx, player.ID.String(), item.ItemID)
			item.Quantity = 1 // Recipes are always singular

			var targetRank int
			_ = db.DB.QueryRow(`SELECT target_rank FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1`, item.ItemID).Scan(&targetRank)
			if targetRank <= 0 {
				targetRank = 1
			}

			unitSellPrice := pricing.RecipeSellPrice(targetRank)
			gain := float64(unitSellPrice * item.Quantity)
			totalGain += gain

			name := fmt.Sprintf("Bí phương (%s)", item.ItemID)
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d (+%.0f)", name, item.Quantity, gain))
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "item_type không hợp lệ (phải là Gu, Material, hoặc Recipe)"})
			return
		}
	}

	player.PrimevalStones += totalGain

	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	qiGained := AddHumanQi(&player, float64(len(req.Items))*5.0)

	if guSold {
		if err := db.RemovePlayerGus(tx, removedIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory changes"})
			return
		}
	}

	_, err = tx.Exec(`UPDATE players SET primeval_stones=$1, equipped_gus=$2, human_qi=$3 WHERE id=$4`,
		player.PrimevalStones, player.EquippedGus, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save sell transaction"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	summaryStr := strings.Join(auditSummaries, ", ")
	if len(summaryStr) > 250 {
		summaryStr = summaryStr[:247] + "..."
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "shop.sell_bulk", EntityType: "shop_item",
		EntityID:       "bulk",
		RequestSummary: fmt.Sprintf("Bán nhiều: %s | nhận %.0f Nguyên Thạch", summaryStr, totalGain),
		IPAddress:      c.ClientIP(), Success: true,
	})

	player.MaterialsInventory = loadMaterialsInventory(player.ID.String())
	responseWithPlayer(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Bán thành công %d loại vật phẩm, nhận +%.0f Nguyên Thạch! Nhân Khí +%.0f.", len(req.Items), totalGain, qiGained)})
}
