package handlers

import (
	"fmt"
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
	Stock int    `json:"stock"`
}

func loadArenaShopItems() ([]ArenaShopItem, error) {
	rows, err := db.DB.Query(`
		SELECT a.id,
			COALESCE(i.name, g.name, 'Công thức ' || rg.name, a.id) AS name,
			a.price, a.qty, a.stock
		FROM arena_shop_items a
		LEFT JOIN item_dictionary i ON a.id = i.id
		LEFT JOIN gu_dictionary g ON a.id = g.id
		LEFT JOIN refining_recipes r ON a.id = r.id
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		ORDER BY a.sort_order ASC, a.id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query arena shop: %w", err)
	}
	defer rows.Close()

	var items []ArenaShopItem
	for rows.Next() {
		var item ArenaShopItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Price, &item.Qty, &item.Stock); err != nil {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func GetArenaShop(c *gin.Context) {
	items, err := loadArenaShopItems()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải Shop Đấu Trường"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
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

	// Look up item in DB
	var targetPrice, targetQty, targetStock int
	var targetName string
	err := db.DB.QueryRow(`
		SELECT a.price, a.qty, a.stock, COALESCE(i.name, g.name, 'Công thức ' || rg.name, a.id)
		FROM arena_shop_items a
		LEFT JOIN item_dictionary i ON a.id = i.id
		LEFT JOIN gu_dictionary g ON a.id = g.id
		LEFT JOIN refining_recipes r ON a.id = r.id
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		WHERE a.id = $1
	`, req.ItemID).Scan(&targetPrice, &targetQty, &targetStock, &targetName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong Shop Đấu Trường"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var arenaCoins int
	err = tx.QueryRow("SELECT COALESCE((materials_inventory->>'arena_coin')::int, 0) FROM players WHERE id = $1 FOR UPDATE", authID).Scan(&arenaCoins)
	if err != nil || arenaCoins < targetPrice {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Xu Đấu Trường!"})
		return
	}

	// Check stock
	if targetStock > 0 {
		var soldCount int
		_ = tx.QueryRow("SELECT COALESCE(SUM((il->>'qty')::int), 0) FROM audit_logs, jsonb_array_elements(details->'items') il WHERE action='arena_shop_purchase' AND details->>'item_id'=$1 AND player_id=$2", req.ItemID, authID).Scan(&soldCount)
		_ = soldCount // placeholder for future per-player stock tracking
	}

	// Deduct coins
	_, err = tx.Exec("UPDATE players SET materials_inventory = jsonb_set(materials_inventory, '{arena_coin}', (($1)::text)::jsonb) WHERE id = $2", arenaCoins-targetPrice, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi trừ xu"})
		return
	}

	// Give item
	if req.ItemID == "primeval_stone" {
		_, err = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", targetQty, authID)
	} else {
		_, err = tx.Exec(`
			UPDATE players 
			SET materials_inventory = COALESCE(materials_inventory, '{}'::jsonb) || jsonb_build_object($1::text, COALESCE((materials_inventory->>$1)::int, 0) + $2)
			WHERE id = $3
		`, req.ItemID, targetQty, authID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thêm vật phẩm"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Đổi vật phẩm thành công!", "cost": targetPrice})
}
