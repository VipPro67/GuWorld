package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

// ShopItem represents a product in the shop
type ShopItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // "Gu", "Material", "Recipe" or "Consumable"
	Price       int64  `json:"price"`
	Rank        int    `json:"rank"`
	Currency    string `json:"currency"` // "primeval" (rank < 6) | "immortal" (rank >= 6)
	Stock       int    `json:"stock"`    // -1 for unlimited
	IsAuto      bool   `json:"-"`        // true when sourced from the virtual random pool (daily stock enforced)
}

// shopCatalogRankExpr resolves a single rank for a polymorphic shop item:
// a pinned rank (c.rank > 0) wins; otherwise items use item_dictionary.rank,
// Gus use their lowest-rank variant (consistent with models.GetGuAny), and
// recipes use refining_recipes.target_rank.
const shopCatalogRankExpr = `CASE WHEN c.rank > 0 THEN c.rank ELSE COALESCE(i.rank, g.rank, r.target_rank, 1) END`

// shopCatalogGuJoin constrains the Gu join: a pinned rank matches that exact
// variant, while an auto (rank = 0) entry matches the lowest-rank variant so
// multi-rank Gus never produce duplicate catalog rows.
const shopCatalogGuJoin = `AND (
	(c.rank > 0 AND g.rank = c.rank) OR
	(c.rank = 0 AND g.rank = (SELECT MIN(g2.rank) FROM gu_dictionary g2 WHERE g2.id = g.id))
)`

// currencyLabel returns the Vietnamese name for the stone currency tier of an
// entity with the given rank.
func currencyLabel(rank int) string {
	if pricing.StoneTierByRank(rank) == "immortal" {
		return "Tiên Nguyên Thạch"
	}
	return "Nguyên Thạch"
}

// Daily random rotation targets: Gu and item (food/material/recipe) entries are
// balanced so the random list favours items over Gus (roughly 40/60 split).
// Default entries are always shown on top of these.
const (
	maxDailyRandomGus   = 40
	maxDailyRandomItems = 60
)

// randomPoolMarkup is the price multiplier applied to auto random-pool items
// on top of their dictionary base_price.
const randomPoolMarkup = 1.5

// randomPoolStock returns the per-day stock for an auto random-pool item of the
// given rank. Higher ranks are scarcer.
func randomPoolStock(rank int) int {
	switch {
	case rank >= 6:
		return 0
	case rank == 5:
		return 2
	case rank == 4:
		return 5
	case rank == 3:
		return 20
	case rank == 2:
		return 50
	default:
		return 100
	}
}

// markedUpPrice applies the random-pool markup to a base price (minimum 1).
func markedUpPrice(basePrice int64) int64 {
	p := int64(float64(basePrice) * randomPoolMarkup)
	if p < 1 {
		return 1
	}
	return p
}

// shopCatalogCacheKey scopes the Redis catalog cache to the current day so the
// daily random rotation refreshes automatically after midnight.
func shopCatalogCacheKey() string {
	return "caravan_shop_catalog:" + time.Now().Format("2006-01-02")
}

// invalidateShopCatalogCache drops the Redis catalog cache and the in-memory
// auto-pool cache so the next /api/shop/list reflects the latest remaining
// stock after a purchase.
func invalidateShopCatalogCache() {
	_ = cache.Delete(shopCatalogCacheKey())
	autoPoolCacheMu.Lock()
	autoPoolCache = nil
	autoPoolCacheAt = time.Time{}
	autoPoolCacheMu.Unlock()
}

// dailyShopSeed derives a stable integer seed from the current UTC date so the
// random subset is identical for every request on the same day.
func dailyShopSeed() int64 {
	now := time.Now().UTC()
	return int64(now.Year()*10000 + int(now.Month())*100 + now.Day())
}

// dailyShopRandomSubset deterministically shuffles a pool using the date as a
// seed and returns at most limit entries. The result is stable within a day and
// rotates daily.
func dailyShopRandomSubset(pool []ShopItem, limit int) []ShopItem {
	if len(pool) == 0 {
		return pool
	}
	shuffled := make([]ShopItem, len(pool))
	copy(shuffled, pool)
	rng := rand.New(rand.NewSource(dailyShopSeed()))
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	if len(shuffled) > limit {
		shuffled = shuffled[:limit]
	}
	return shuffled
}

// autoShopPool builds the virtual random pool sourced from the dictionaries:
// every rank < 5 Gu variant (pinned to its specific rank) plus the food items
// those Gus eat. Ids already listed in caravan_shop_items are excluded so
// manually managed entries (including defaults) take precedence. These entries
// live outside caravan_shop_items, so they never pollute the admin list or CSV
// export.
func autoShopPool() ([]ShopItem, error) {
	items := []ShopItem{}

	// Rank < 5 Gus not already listed in the shop, with today's remaining stock
	// (falls back to the per-rank daily limit when no row exists yet).
	guRows, err := db.DB.Query(`
		SELECT g.id, g.rank, COALESCE(g.name, g.id), COALESCE(g.description, ''), COALESCE(g.base_price, 0), ds.remaining
		FROM gu_dictionary g
		LEFT JOIN caravan_shop_daily_stock ds
		       ON ds.day = CURRENT_DATE AND ds.id = g.id AND ds.rank = g.rank
		WHERE g.rank < 5
		  AND NOT EXISTS (SELECT 1 FROM caravan_shop_items c WHERE c.id = g.id)
		ORDER BY g.id, g.rank
	`)
	if err != nil {
		return nil, err
	}
	for guRows.Next() {
		var id, name, desc string
		var rank int
		var basePrice int64
		var remaining sql.NullInt64
		if err := guRows.Scan(&id, &rank, &name, &desc, &basePrice, &remaining); err != nil {
			guRows.Close()
			return nil, err
		}
		stock := randomPoolStock(rank)
		if remaining.Valid {
			stock = int(remaining.Int64)
		}
		items = append(items, ShopItem{
			ID: id, Name: name, Description: desc, Type: "Gu",
			Price: markedUpPrice(pricing.GuBasePrice(basePrice, rank)), Rank: rank,
			Currency: pricing.StoneTierByRank(rank), Stock: stock, IsAuto: true,
		})
	}
	if err := guRows.Err(); err != nil {
		guRows.Close()
		return nil, err
	}
	guRows.Close()

	// Food items referenced by any rank < 5 Gu. The special stone markers
	// ("primeval_stone", "immortal_stone") are not item_dictionary rows so the
	// join naturally drops them. Deduplicated via DISTINCT.
	foodRows, err := db.DB.Query(`
		SELECT DISTINCT i.id, COALESCE(i.name, i.id), COALESCE(i.description, ''), COALESCE(i.item_type, 'Food'), COALESCE(i.rank, 1), COALESCE(i.base_price, 0), ds.remaining
		FROM gu_dictionary g
		JOIN LATERAL jsonb_array_elements_text(g.food_types) AS f(fid) ON TRUE
		JOIN item_dictionary i ON i.id = f.fid
		LEFT JOIN caravan_shop_daily_stock ds
		       ON ds.day = CURRENT_DATE AND ds.id = i.id AND ds.rank = COALESCE(i.rank, 1)
		WHERE g.rank < 5
		  AND NOT EXISTS (SELECT 1 FROM caravan_shop_items c WHERE c.id = i.id)
		ORDER BY i.id
	`)
	if err != nil {
		return nil, err
	}
	for foodRows.Next() {
		var id, name, desc, itemType string
		var rank int
		var basePrice int64
		var remaining sql.NullInt64
		if err := foodRows.Scan(&id, &name, &desc, &itemType, &rank, &basePrice, &remaining); err != nil {
			foodRows.Close()
			return nil, err
		}
		stock := randomPoolStock(rank)
		if remaining.Valid {
			stock = int(remaining.Int64)
		}
		items = append(items, ShopItem{
			ID: id, Name: name, Description: desc, Type: itemType,
			Price: markedUpPrice(pricing.ItemBasePrice(basePrice, rank)), Rank: rank,
			Currency: pricing.StoneTierByRank(rank), Stock: stock, IsAuto: true,
		})
	}
	if err := foodRows.Err(); err != nil {
		foodRows.Close()
		return nil, err
	}
	foodRows.Close()

	return items, nil
}

// autoPoolCache memoizes autoShopPool() briefly since the pool only changes
// when an admin imports dictionary data (which also refreshes the Redis
// catalog cache).
var (
	autoPoolCacheMu sync.Mutex
	autoPoolCache   []ShopItem
	autoPoolCacheAt time.Time
)

func cachedAutoShopPool() ([]ShopItem, error) {
	autoPoolCacheMu.Lock()
	defer autoPoolCacheMu.Unlock()
	if autoPoolCache != nil && time.Since(autoPoolCacheAt) < 5*time.Minute {
		return autoPoolCache, nil
	}
	pool, err := autoShopPool()
	if err != nil {
		return nil, err
	}
	autoPoolCache = pool
	autoPoolCacheAt = time.Now()
	return pool, nil
}

func loadCaravanShopCatalog() ([]ShopItem, error) {
	rows, err := db.DB.Query(`
		SELECT c.id, 
		       COALESCE(i.name, g.name, 'Công thức ' || rg.name, c.id) AS name, 
		       COALESCE(i.description, g.description, '') AS description,
		       CASE WHEN r.target_gu_id IS NOT NULL THEN 'Recipe' ELSE COALESCE(i.item_type, 'Gu') END AS type, 
		       c.price, ` + shopCatalogRankExpr + ` AS rank, c.stock, c.is_default
		FROM caravan_shop_items c
		LEFT JOIN item_dictionary i ON c.id = i.id
		LEFT JOIN gu_dictionary g ON c.id = g.id ` + shopCatalogGuJoin + `
		LEFT JOIN refining_recipes r ON c.id = ('recipe_' || r.target_gu_id || '_' || r.target_rank::text || '_' || r.variant::text)
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		ORDER BY c.sort_order ASC, c.id ASC, c.rank ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	defaults := []ShopItem{}
	randomPool := []ShopItem{}
	for rows.Next() {
		var item ShopItem
		var isDefault bool
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Type, &item.Price, &item.Rank, &item.Stock, &isDefault); err != nil {
			return nil, err
		}
		item.Currency = pricing.StoneTierByRank(item.Rank)

		if isDefault {
			defaults = append(defaults, item)
		} else {
			randomPool = append(randomPool, item)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Append the auto-generated random pool (rank<5 Gus + their food) that is
	// derived from the dictionaries and never persisted to caravan_shop_items.
	auto, err := cachedAutoShopPool()
	if err != nil {
		return nil, err
	}
	randomPool = append(randomPool, auto...)
	// Drop sold-out auto items (remaining == 0) so they don't take a daily slot.
	available := randomPool[:0]
	for _, it := range randomPool {
		if it.IsAuto && it.Stock <= 0 {
			continue
		}
		available = append(available, it)
	}
	randomPool = available
	// Balance the daily rotation: pick a fixed number of Gus and items.
	var guPool, itemPool []ShopItem
	for _, it := range randomPool {
		if it.Type == "Gu" {
			guPool = append(guPool, it)
		} else {
			itemPool = append(itemPool, it)
		}
	}
	items := append(defaults, dailyShopRandomSubset(guPool, maxDailyRandomGus)...)
	items = append(items, dailyShopRandomSubset(itemPool, maxDailyRandomItems)...)
	return items, nil
}

func findCaravanShopItem(itemID string, rank int) (*ShopItem, error) {
	query := `
		SELECT c.id, 
		       COALESCE(i.name, g.name, 'Công thức ' || rg.name, c.id) AS name, 
		       COALESCE(i.description, g.description, '') AS description,
		       CASE WHEN r.target_gu_id IS NOT NULL THEN 'Recipe' ELSE COALESCE(i.item_type, 'Gu') END AS type, 
		       c.price, ` + shopCatalogRankExpr + ` AS rank, c.stock
		FROM caravan_shop_items c
		LEFT JOIN item_dictionary i ON c.id = i.id
		LEFT JOIN gu_dictionary g ON c.id = g.id ` + shopCatalogGuJoin + `
		LEFT JOIN refining_recipes r ON c.id = ('recipe_' || r.target_gu_id || '_' || r.target_rank::text || '_' || r.variant::text)
		LEFT JOIN gu_dictionary rg ON r.target_gu_id = rg.id
		WHERE c.id = $1 AND c.rank = $2
	`
	var item ShopItem
	// Exact (id, rank) match first, falling back to the auto (rank = 0) entry.
	err := db.DB.QueryRow(query, itemID, rank).Scan(&item.ID, &item.Name, &item.Description, &item.Type, &item.Price, &item.Rank, &item.Stock)
	if err != nil && rank != 0 {
		err = db.DB.QueryRow(query, itemID, 0).Scan(&item.ID, &item.Name, &item.Description, &item.Type, &item.Price, &item.Rank, &item.Stock)
	}
	if err != nil {
		// Fall back to the virtual auto pool (rank<5 Gus + their food).
		if auto, aerr := cachedAutoShopPool(); aerr == nil {
			for _, a := range auto {
				if a.ID == itemID && a.Rank == rank {
					return &a, nil
				}
			}
		}
		return nil, err
	}
	item.Currency = pricing.StoneTierByRank(item.Rank)
	return &item, nil
}

// GetShopList returns the database-backed shop inventory, cached in Redis.
func GetShopList(c *gin.Context) {
	var items []ShopItem
	cacheKey := shopCatalogCacheKey()

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

	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Giao dịch thành công! Nhân Khí +%.0f.", qiGained)})
}

type SellShopRequest struct {
	PlayerID string `json:"player_id" binding:"required"`
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
	ItemType string `json:"item_type" binding:"required"`
	Rank     int    `json:"rank"` // 0 = auto; >0 = the specific rank being sold
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

	totalGain := int64(0)
	sellRank := 1
	var removedIDs []string
	if req.ItemType == "Material" || req.ItemType == "Consumable" {
		preferredCat := "materials"
		if req.ItemType == "Consumable" {
			preferredCat = "consumables"
		}

		rows, err := tx.Query(`SELECT category, quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2`, player.ID.String(), req.ItemID)
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

		if totalQty < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ vật phẩm để bán"})
			return
		}

		remainingDeduct := req.Quantity
		for _, iv := range invs {
			if iv.category == preferredCat {
				deduct := iv.quantity
				if deduct > remainingDeduct {
					deduct = remainingDeduct
				}
				if err := InventorySvc.RemoveItem(tx, player.ID.String(), req.ItemID, iv.category, deduct); err != nil {
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
					if err := InventorySvc.RemoveItem(tx, player.ID.String(), req.ItemID, iv.category, deduct); err != nil {
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

		// Resolve sell price: prefer shop catalog price, fallback to item_dictionary rank
		var unitSellPrice int64
		if shopItem, err := findCaravanShopItem(req.ItemID, req.Rank); err == nil {
			unitSellPrice = pricing.SellPrice(shopItem.Price)
			sellRank = shopItem.Rank
		} else {
			var rank int
			var dbBasePrice int64
			_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, req.ItemID).Scan(&dbBasePrice, &rank)
			sellRank = rank
			unitSellPrice = pricing.SellPrice(pricing.ItemBasePrice(dbBasePrice, rank))
		}
		totalGain = unitSellPrice * req.Quantity
	} else if req.ItemType == "Gu" {
		var removed int64 = 0
		newInv := models.GuInventory{}
		for _, gu := range player.GuInventory {
			if gu.GuID == req.ItemID && !gu.IsVital && !gu.Equipped && removed < req.Quantity {
				removed++
				removedIDs = append(removedIDs, gu.InstanceID)
				continue
			}
			newInv = append(newInv, gu)
		}
		if removed < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Cổ trùng để bán hoặc Cổ là Bản Mệnh/Trang bị"})
			return
		}
		player.GuInventory = newInv
		// Resolve sell price: prefer shop catalog price, fallback to gu_dictionary rank
		var unitSellPrice int64
		if shopItem, err := findCaravanShopItem(req.ItemID, req.Rank); err == nil {
			unitSellPrice = pricing.SellPrice(shopItem.Price)
			sellRank = shopItem.Rank
		} else {
			var rank int
			var dbBasePrice int64
			if req.Rank > 0 {
				_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 AND rank=$2`, req.ItemID, req.Rank).Scan(&dbBasePrice, &rank)
			}
			if rank == 0 {
				_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 ORDER BY rank ASC LIMIT 1`, req.ItemID).Scan(&dbBasePrice, &rank)
			}
			sellRank = rank
			unitSellPrice = pricing.SellPrice(pricing.GuBasePrice(dbBasePrice, rank))
		}
		totalGain = unitSellPrice * req.Quantity
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "item_type phải là Gu hoặc Material"})
		return
	}

	// Pay out in the currency tier matching the item's rank
	if pricing.StoneTierByRank(sellRank) == "immortal" {
		player.ImmortalStones += totalGain
	} else {
		player.PrimevalStones += totalGain
	}
	player.EquippedGus = []string{}
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}
	qiGained := 0.0
	if req.ItemType == "Gu" {
		if err := db.RemovePlayerGus(tx, removedIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory changes"})
			return
		}
	}
	sellCurrencyName := "Nguyên Thạch"
	if pricing.StoneTierByRank(sellRank) == "immortal" {
		sellCurrencyName = "Tiên Nguyên Thạch"
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2, equipped_gus=$3, human_qi=$4 WHERE id=$5`,
		player.PrimevalStones, player.ImmortalStones, player.EquippedGus, player.HumanQi, player.ID)
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

	action := "shop.sell_item"
	entityType := "item"
	if req.ItemType == "Gu" {
		action = "shop.sell_gu"
		entityType = "gu_instance"
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         action,
		EntityType:     entityType,
		EntityID:       req.ItemID,
		RequestSummary: fmt.Sprintf("Bán %dx %s (%s) cho Tiệm Buôn | Nhận %d %s", req.Quantity, itemName, req.ItemID, totalGain, sellCurrencyName),
		Metadata: map[string]interface{}{
			"item_id":       req.ItemID,
			"item_name":     itemName,
			"item_type":     req.ItemType,
			"quantity":      req.Quantity,
			"total_gain":    totalGain,
			"currency_type": sellCurrencyName,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Bán thành công, nhận %s! Nhân Khí +%.0f.", sellCurrencyName, qiGained)})
}

type SellShopItemInfo struct {
	ItemID   string `json:"item_id" binding:"required"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
	ItemType string `json:"item_type" binding:"required"`
	Rank     int    `json:"rank"` // 0 = auto; >0 = the specific rank being sold
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

	totalGainPrimeval := int64(0)
	totalGainImmortal := int64(0)
	guSold := false
	var removedIDs []string
	var auditSummaries []string

	addGain := func(sellRank int, gain int64) {
		if pricing.StoneTierByRank(sellRank) == "immortal" {
			totalGainImmortal += gain
		} else {
			totalGainPrimeval += gain
		}
	}

	for _, item := range req.Items {
		if item.ItemType == "Material" || item.ItemType == "Consumable" {
			preferredCat := "materials"
			if item.ItemType == "Consumable" {
				preferredCat = "consumables"
			}

			rows, err := tx.Query(`SELECT category, quantity FROM player_inventory_items WHERE player_id = $1 AND item_id = $2`, player.ID.String(), item.ItemID)
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

			if totalQty < item.Quantity {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ vật phẩm %s để bán", item.ItemID)})
				return
			}

			remainingDeduct := item.Quantity
			for _, iv := range invs {
				if iv.category == preferredCat {
					deduct := iv.quantity
					if deduct > remainingDeduct {
						deduct = remainingDeduct
					}
					if err := InventorySvc.RemoveItem(tx, player.ID.String(), item.ItemID, iv.category, deduct); err != nil {
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
						if err := InventorySvc.RemoveItem(tx, player.ID.String(), item.ItemID, iv.category, deduct); err != nil {
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

			var unitSellPrice int64
			sellRank := 1
			if shopItem, err := findCaravanShopItem(item.ItemID, item.Rank); err == nil {
				unitSellPrice = pricing.SellPrice(shopItem.Price)
				sellRank = shopItem.Rank
			} else {
				var rank int
				var dbBasePrice int64
				_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, item.ItemID).Scan(&dbBasePrice, &rank)
				sellRank = rank
				unitSellPrice = pricing.SellPrice(pricing.ItemBasePrice(dbBasePrice, rank))
			}
			gain := unitSellPrice * item.Quantity
			addGain(sellRank, gain)

			var name string
			_ = db.DB.QueryRow(`SELECT name FROM item_dictionary WHERE id = $1`, item.ItemID).Scan(&name)
			if name == "" {
				name = item.ItemID
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d (+%d %s)", name, item.Quantity, gain, currencyLabel(sellRank)))

		} else if item.ItemType == "Gu" {
			var removed int64 = 0
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

			var unitSellPrice int64
			sellRank := 1
			if shopItem, err := findCaravanShopItem(item.ItemID, item.Rank); err == nil {
				unitSellPrice = pricing.SellPrice(shopItem.Price)
				sellRank = shopItem.Rank
			} else {
				var rank int
				var dbBasePrice int64
				if item.Rank > 0 {
					_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 AND rank=$2`, item.ItemID, item.Rank).Scan(&dbBasePrice, &rank)
				}
				if rank == 0 {
					_ = db.DB.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 ORDER BY rank ASC LIMIT 1`, item.ItemID).Scan(&dbBasePrice, &rank)
				}
				sellRank = rank
				unitSellPrice = pricing.SellPrice(pricing.GuBasePrice(dbBasePrice, rank))
			}
			gain := unitSellPrice * item.Quantity
			addGain(sellRank, gain)

			var name string
			if spec, ok := models.GetGuAny(item.ItemID); ok {
				name = spec.Name
			}
			if name == "" {
				name = item.ItemID
			}
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d (+%d %s)", name, item.Quantity, gain, currencyLabel(sellRank)))
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
			gain := unitSellPrice * item.Quantity
			addGain(targetRank, gain)

			name := fmt.Sprintf("Bí phương (%s)", item.ItemID)
			auditSummaries = append(auditSummaries, fmt.Sprintf("%s x%d (+%d %s)", name, item.Quantity, gain, currencyLabel(targetRank)))
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "item_type không hợp lệ (phải là Gu, Material, hoặc Recipe)"})
			return
		}
	}

	player.PrimevalStones += totalGainPrimeval
	player.ImmortalStones += totalGainImmortal

	player.EquippedGus = []string{}
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}

	qiGained := 0.0

	if guSold {
		if err := db.RemovePlayerGus(tx, removedIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Gu inventory changes"})
			return
		}
	}

	_, err = tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2, equipped_gus=$3, human_qi=$4 WHERE id=$5`,
		player.PrimevalStones, player.ImmortalStones, player.EquippedGus, player.HumanQi, player.ID)
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

	gainSummary := fmt.Sprintf("+%d Nguyên Thạch", totalGainPrimeval)
	if totalGainImmortal > 0 {
		if totalGainPrimeval > 0 {
			gainSummary = fmt.Sprintf("+%d Nguyên Thạch và +%d Tiên Nguyên Thạch", totalGainPrimeval, totalGainImmortal)
		} else {
			gainSummary = fmt.Sprintf("+%d Tiên Nguyên Thạch", totalGainImmortal)
		}
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "shop.sell_bulk",
		EntityType:     "shop",
		RequestSummary: fmt.Sprintf("Bán sỉ %d loại cho Tiệm Buôn: %s | Tổng nhận: %s", len(req.Items), summaryStr, gainSummary),
		Metadata: map[string]interface{}{
			"item_count":          len(req.Items),
			"total_gain_primeval": totalGainPrimeval,
			"total_gain_immortal": totalGainImmortal,
			"summaries":           auditSummaries,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	notify.Send(db.DB, authID, "shop_sell", "Bán Hàng Tiệm Buôn",
		fmt.Sprintf("Đã bán sỉ %d loại vật phẩm/Cổ trùng, thu về %s.", len(req.Items), gainSummary))

	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Bán thành công %d loại vật phẩm, nhận %s! Nhân Khí +%.0f.", len(req.Items), gainSummary, qiGained)})
}
