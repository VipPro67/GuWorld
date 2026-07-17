package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type ArenaShopItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price"` // in ARENA_COIN
	Qty   int    `json:"qty"`
}

var ArenaShopItems = []ArenaShopItem{
	{ID: "primeval_stone", Name: "Nguyên Thạch x100", Price: 50, Qty: 100},
	{ID: "water", Name: "Nước Sạch", Price: 10, Qty: 1},
	{ID: "ration", Name: "Lương Khô", Price: 10, Qty: 1},
	{ID: "wine", Name: "Rượu Ngon", Price: 20, Qty: 1},
	{ID: "boar_meat", Name: "Thịt Lợn Rừng", Price: 15, Qty: 1},
}

func GetArenaShop(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": ArenaShopItems})
}

type BuyArenaShopRequest struct {
	ItemID string `json:"item_id" binding:"required"`
}

func BuyArenaShopItem(c *gin.Context) {
	var req BuyArenaShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã vật phẩm"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var targetItem *ArenaShopItem
	for _, item := range ArenaShopItems {
		if item.ID == req.ItemID {
			targetItem = &item
			break
		}
	}

	if targetItem == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong Shop Đấu Trường"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var materialsJSON []byte
	var primevalStones int
	err = tx.QueryRow("SELECT materials_inventory, primeval_stones FROM players WHERE id = $1 FOR UPDATE", authID).Scan(&materialsJSON, &primevalStones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy dữ liệu người chơi"})
		return
	}

	if len(materialsJSON) > 0 {
		importJson := true // dummy logic, we already import encoding/json
		_ = importJson
	}
	// Let's do it safer with SQL
	var arenaCoins int
	err = tx.QueryRow("SELECT COALESCE((materials_inventory->>'arena_coin')::int, 0) FROM players WHERE id = $1", authID).Scan(&arenaCoins)
	if err != nil || arenaCoins < targetItem.Price {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Xu Đấu Trường!"})
		return
	}

	// Deduct coins
	_, err = tx.Exec("UPDATE players SET materials_inventory = jsonb_set(materials_inventory, '{arena_coin}', (($1)::text)::jsonb) WHERE id = $2", arenaCoins-targetItem.Price, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi trừ xu"})
		return
	}

	// Give item
	if targetItem.ID == "primeval_stone" {
		_, err = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", targetItem.Qty, authID)
	} else {
		_, err = tx.Exec(`
			UPDATE players 
			SET materials_inventory = COALESCE(materials_inventory, '{}'::jsonb) || jsonb_build_object($1::text, COALESCE((materials_inventory->>$1)::int, 0) + $2)
			WHERE id = $3
		`, targetItem.ID, targetItem.Qty, authID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thêm vật phẩm"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Đổi vật phẩm thành công!", "cost": targetItem.Price})
}
