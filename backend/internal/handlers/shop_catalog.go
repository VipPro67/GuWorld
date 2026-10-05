package handlers

import (
	"database/sql"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
	"sync"
)

type ShopItem struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Type        string  `json:"type"` // "Gu", "Material", "Recipe" or "Consumable"
	Price       float64 `json:"price"`
	Rank        int     `json:"rank"`
	Currency    string  `json:"currency"` // "primeval" (rank < 6) | "immortal" (rank >= 6)
	Stock       int     `json:"stock"`    // -1 for unlimited
	IsAuto      bool    `json:"-"`        // true when sourced from the virtual random pool (daily stock enforced)
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
func markedUpPrice(basePrice float64) float64 {
	p := pricing.Quantize(basePrice * randomPoolMarkup)
	if p < 0.001 {
		return 0.001
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
		var basePrice float64
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
		var basePrice float64
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
	// Fast path: check active caravan catalog cache to avoid heavy joins
	var cachedItems []ShopItem
	if err := cache.GetJSON(shopCatalogCacheKey(), &cachedItems); err == nil && len(cachedItems) > 0 {
		for _, item := range cachedItems {
			if item.ID == itemID && (rank == 0 || item.Rank == rank) {
				cp := item
				return &cp, nil
			}
		}
	}

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
