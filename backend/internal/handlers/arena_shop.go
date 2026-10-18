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
	"github.com/vippro67/GuWorld/backend/pkg/logger"
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

// Immortal-grade ("tiên cấp") arena resources.
//
// Instead of enumerating hundreds of rank >= 6 Food/Material rows by hand, the
// shop advertises a deterministic rotating subset sourced from item_dictionary.
// It mirrors the caravan shop's virtual auto-pool (shop_catalog.go): rotation is
// derived from the hourly shopRotationBucket seed and the entries are never
// persisted to arena_shop_items.
const (
	maxArenaRandomResources  = 30
	arenaRandomResourcePrice = 99 // flat Xu Đấu Trường (đồng giá)
)

// arenaRandomResourcePool returns every rank >= 6 Food/Material dictionary item
// not already listed explicitly in arena_shop_items.
func arenaRandomResourcePool() ([]ShopItem, error) {
	rows, err := db.DB.Query(`
		SELECT i.id, COALESCE(i.name, i.id), COALESCE(i.item_type, 'Material'), COALESCE(i.rank, 1)
		FROM item_dictionary i
		WHERE i.item_type IN ('Food', 'Material')
		  AND COALESCE(i.rank, 1) >= 6
		  AND NOT EXISTS (SELECT 1 FROM arena_shop_items a WHERE a.id = i.id)
		ORDER BY i.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pool := []ShopItem{}
	for rows.Next() {
		var it ShopItem
		if err := rows.Scan(&it.ID, &it.Name, &it.Type, &it.Rank); err != nil {
			return nil, err
		}
		it.Price = arenaRandomResourcePrice
		it.Stock = -1
		it.IsAuto = true
		pool = append(pool, it)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return pool, nil
}

// selectArenaRandomResources picks the stable-per-rotation showcase subset,
// reusing the caravan shop's deterministic shuffle/seed.
func selectArenaRandomResources(pool []ShopItem) []ShopItem {
	return dailyShopRandomSubset(pool, maxArenaRandomResources)
}

// arenaRandomResourceByID resolves a purchase against the full immortal
// resource pool (not just the current showcase) so a rotation boundary mid-modal
// can't reject an item the player just saw.
func arenaRandomResourceByID(itemID string) (*ShopItem, bool) {
	pool, err := arenaRandomResourcePool()
	if err != nil {
		return nil, false
	}
	for i := range pool {
		if pool[i].ID == itemID {
			return &pool[i], true
		}
	}
	return nil, false
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

	// Append the rotating immortal-grade resource showcase (rank >= 6 Food /
	// Material). Auto items live outside arena_shop_items so they never pollute
	// the admin list or CSV export.
	if pool, perr := arenaRandomResourcePool(); perr == nil {
		for _, it := range selectArenaRandomResources(pool) {
			items = append(items, ArenaShopItem{ID: it.ID, Name: it.Name, Price: int64(it.Price), Qty: 1, Stock: -1})
		}
	} else {
		logger.S.Warnf("arena random resource pool: %v", perr)
	}

	// Short TTL (not 24h) so the rotating subset refreshes when the rotation
	// window changes; admin edits still invalidate via cache.Delete("arena_shop_items").
	_ = cache.SetJSON("arena_shop_items", items, 10*time.Minute)
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
		// Fall back to the rotating immortal-grade resource pool (Food/Material,
		// rank >= 6) which is advertised by GetArenaShop but not persisted.
		auto, ok := arenaRandomResourceByID(req.ItemID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong Shop Đấu Trường"})
			return
		}
		targetPrice, targetQty, targetStock, targetName = int64(auto.Price), 1, -1, auto.Name
	}

	totalCost := targetPrice * quantity
	totalQty := targetQty * quantity

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

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

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu giao dịch"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "arena_shop_purchase", EntityType: "arena_shop_item",
		EntityID:       req.ItemID,
		RequestSummary: fmt.Sprintf("%s | qty=%d | giá=%d Xu Đấu Trường", targetName, totalQty, totalCost),
		Metadata:       map[string]interface{}{"item_id": req.ItemID, "qty": totalQty, "quantity": quantity},
		IPAddress:      c.ClientIP(), Success: true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Đổi %dx %s thành công! Nhân Khí +%d.", totalQty, targetName, 5*quantity),
		"cost":    totalCost,
	})
}
