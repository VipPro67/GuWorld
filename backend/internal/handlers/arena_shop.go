package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type ArenaShopItem struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Price int64  `json:"price"` // in ARENA_COIN
	Qty   int64  `json:"qty"`
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
		LEFT JOIN refining_recipes r ON a.id = ('recipe_' || r.target_gu_id || '_' || r.target_rank::text || '_' || r.variant::text)
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
	var items []ArenaShopItem
	if err := cache.GetJSON("arena_shop_items", &items); err == nil && len(items) > 0 {
		c.JSON(http.StatusOK, gin.H{"items": items})
		return
	}

	items, err := loadArenaShopItems()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải Shop Đấu Trường"})
		return
	}

	_ = cache.SetJSON("arena_shop_items", items, 24*time.Hour)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

type BuyArenaShopRequest struct {
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity"`
}

func BuyArenaShopItem(c *gin.Context) {
	var req BuyArenaShopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã vật phẩm"})
		return
	}

	quantity := req.Quantity
	if quantity <= 0 {
		quantity = 1
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Look up item in DB
	var targetPrice, targetQty int64
	var targetStock int
	var targetName string
	err := db.DB.QueryRow(`
		SELECT a.price, a.qty, a.stock, COALESCE(i.name, g.name, 'Công thức ' || rg.name, a.id)
		FROM arena_shop_items a
		LEFT JOIN item_dictionary i ON a.id = i.id
		LEFT JOIN gu_dictionary g ON a.id = g.id
		LEFT JOIN refining_recipes r ON a.id = ('recipe_' || r.target_gu_id || '_' || r.target_rank::text || '_' || r.variant::text)
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		WHERE a.id = $1
	`, req.ItemID).Scan(&targetPrice, &targetQty, &targetStock, &targetName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong Shop Đấu Trường"})
		return
	}

	totalCost := targetPrice * quantity
	totalQty := targetQty * quantity

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Deduct coins (atomic FOR UPDATE + check + deduct)
	if err := InventorySvc.RemoveItem(tx, authID.String(), "arena_coin", "materials", totalCost); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Xu Đấu Trường!"})
		return
	}

	// Check stock
	if targetStock > 0 {
		var soldCount int
		_ = tx.QueryRow("SELECT COALESCE(SUM((il->>'qty')::int), 0) FROM audit_logs, jsonb_array_elements(details->'items') il WHERE action='arena_shop_purchase' AND details->>'item_id'=$1 AND player_id=$2", req.ItemID, authID).Scan(&soldCount)
		_ = soldCount // placeholder for future per-player stock tracking
	}

	// Give item
	var giveErr error
	if req.ItemID == "primeval_stone" {
		_, giveErr = tx.Exec("UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2", totalQty, authID)
	} else {
		giveErr = InventorySvc.AddItem(tx, authID.String(), req.ItemID, "materials", totalQty)
	}

	if giveErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thêm vật phẩm"})
		return
	}

	tx.Commit()

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "arena_shop_purchase", EntityType: "arena_shop_item",
		EntityID:       req.ItemID,
		RequestSummary: fmt.Sprintf("%s | qty=%d | giá=%d Xu Đấu Trường", targetName, totalQty, totalCost),
		Metadata:       map[string]interface{}{"item_id": req.ItemID, "qty": totalQty, "quantity": quantity},
		IPAddress:      c.ClientIP(), Success: true,
	})

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Đổi %dx %s thành công! Nhân Khí +%d.", totalQty, targetName, 5*quantity), "cost": totalCost})
}
