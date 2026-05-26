package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// ShopItem represents a product in the shop
type ShopItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "Gu" or "Material"
	Price       int    `json:"price"`
	Stock       int    `json:"stock"` // -1 for unlimited
}

// CaravanShopCatalog hardcodes starter items in the Caravan
var CaravanShopCatalog = []ShopItem{
	{ID: "grass_puppet", Name: "Hình nộm người cỏ", Description: "Hình nộm cỏ để tập luyện chiến đấu.", Type: "Material", Price: 3, Stock: 99},
	{ID: "moon_orchid_petal", Name: "Cánh hoa Nguyệt Lan", Description: "Cánh hoa lan phát sáng vào ban đêm. Thức ăn của Nguyệt Quang Cổ.", Type: "Material", Price: 5, Stock: 99},
	{ID: "boar_meat", Name: "Thịt heo rừng", Description: "Nguồn thức ăn giàu năng lượng. Thức ăn của Bạch Thỉ Cổ và Hắc Thỉ Cổ.", Type: "Material", Price: 10, Stock: 99},
	{ID: "GU_MOONLIGHT", Name: "Nguyệt Quang Cổ", Description: "Cổ trùng Rank 1 Nhất Chuyển, phóng ra nguyệt nhận sắc bén tấn công.", Type: "Gu", Price: 50, Stock: 5},
	{ID: "GU_BLACK_BOAR", Name: "Hắc Thỉ Cổ", Description: "Cổ trùng Rank 1 Nhất Chuyển, cải tạo thể chất tăng lực lực lượng vĩnh viễn.", Type: "Gu", Price: 100, Stock: 2},
	{ID: "GU_CLEANSING_WATER", Name: "Tịnh Thủy Cổ", Description: "Cổ trùng hỗ trợ tẩy rửa cặn bã Không khiếu và thanh trừ ý chí lạ.", Type: "Gu", Price: 650, Stock: 1},
}

// GetShopList returns the hardcoded shop inventory
func GetShopList(c *gin.Context) {
	c.JSON(http.StatusOK, CaravanShopCatalog)
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

	// 1. Find product in catalog
	var selectedItem *ShopItem
	for _, item := range CaravanShopCatalog {
		if item.ID == req.ItemID {
			selectedItem = &item
			break
		}
	}

	if selectedItem == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy vật phẩm này trong Thương Đội."})
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

	// 4. Validate Funds
	totalCost := selectedItem.Price * req.Quantity
	if player.PrimevalStones < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch để giao dịch."})
		return
	}

	// Deduct currency
	player.PrimevalStones -= totalCost

	// 5. Add items to player inventory
	if selectedItem.Type == "Gu" {
		// Verify if the Gu ID is in static GuRegistry
		spec, ok := models.GuRegistry[selectedItem.ID]
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy định nghĩa của Cổ trùng trong Dictionary"})
			return
		}

		for i := 0; i < req.Quantity; i++ {
			newInstance := models.GuInstance{
				InstanceID: uuid.New().String(),
				GuID:       spec.ID,
				Hunger:     100,
				Status:     "ACTIVE",
			}
			player.GuInventory = append(player.GuInventory, newInstance)
		}
	} else if selectedItem.Type == "Material" {
		if player.MaterialsInventory == nil {
			player.MaterialsInventory = make(models.MaterialsInventory)
		}
		player.MaterialsInventory[selectedItem.ID] += req.Quantity
	}

	// 6. Save player state
	updateQuery := `
		UPDATE players
		SET primeval_stones = $1, gu_inventory = $2, materials_inventory = $3
		WHERE id = $4
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.GuInventory, player.MaterialsInventory, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player inventory updates"})
		return
	}

	// Commit Transaction
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": "Giao dịch thành công!",
	})
}
