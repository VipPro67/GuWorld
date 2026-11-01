package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
	"github.com/vippro67/GuWorld/backend/pkg/pricing"
)

// ---------------------------------------------------------------------------
// Merchant model + pure pricing
// ---------------------------------------------------------------------------

type merchantBuyTags struct {
	Types   []string `json:"types,omitempty"`
	MinRank int      `json:"min_rank,omitempty"`
	MaxRank int      `json:"max_rank,omitempty"`
	ItemIDs []string `json:"item_ids,omitempty"`
}

type merchantProfile struct {
	ID            string
	Name          string
	Description   string
	Kind          string
	BuyRatio      float64
	BuyFloorRatio float64
	BuyTags       merchantBuyTags
	QuotaPerItem  int
	QuotaPeriod   string
	SellMarkup    float64
}

// accepts reports whether the merchant will buy the given entity. An explicit
// item_ids allow-list wins; otherwise type + rank range gate the purchase.
func (t merchantBuyTags) accepts(itemType string, rank int, itemID string) bool {
	if len(t.ItemIDs) > 0 {
		for _, id := range t.ItemIDs {
			if id == itemID {
				return true
			}
		}
		return false
	}
	if len(t.Types) > 0 {
		ok := false
		for _, ty := range t.Types {
			if strings.EqualFold(ty, itemType) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if t.MinRank > 0 && rank < t.MinRank {
		return false
	}
	if t.MaxRank > 0 && rank > t.MaxRank {
		return false
	}
	return true
}

// merchantBuyRatio is the dynamic supply/demand price factor: it starts at
// `start` when nothing has been bought this period and decays linearly to
// `floor` once the per-item quota is filled.
func merchantBuyRatio(bought, quota int, start, floor float64) float64 {
	r := start
	if quota > 0 {
		f := float64(bought) / float64(quota)
		if f < 0 {
			f = 0
		} else if f > 1 {
			f = 1
		}
		r = start - (start-floor)*f
	}
	if r < 0 {
		r = 0
	}
	return r
}

// merchantPeriodStart returns the start of the current quota window in UTC.
func merchantPeriodStart(period string, now time.Time) time.Time {
	switch strings.ToUpper(period) {
	case "HOUR":
		return now.UTC().Truncate(time.Hour)
	default:
		t := now.UTC()
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}
}

func loadMerchantProfile(q queryer, id string) (merchantProfile, error) {
	var p merchantProfile
	var tagsRaw []byte
	err := q.QueryRow(`
		SELECT id, name, description, kind, buy_ratio, buy_floor_ratio, buy_tags, quota_per_item, quota_period, sell_markup
		FROM merchant_profiles WHERE id = $1 AND enabled`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.Kind, &p.BuyRatio, &p.BuyFloorRatio, &tagsRaw, &p.QuotaPerItem, &p.QuotaPeriod, &p.SellMarkup)
	if err != nil {
		return p, err
	}
	_ = json.Unmarshal(tagsRaw, &p.BuyTags)
	return p, nil
}

// loadMerchantEffective loads a merchant's profile then overlays any per-instance
// overrides (buy_tags / buy_ratio / buy_floor_ratio / quota_per_item) so each
// roaming trader can own its own buy pool.
func loadMerchantEffective(q queryer, merchantID string) (merchantProfile, string, error) {
	var p merchantProfile
	var regionID string
	var tagsRaw, oTagsRaw []byte
	var oRatio, oFloor sql.NullFloat64
	var oQuota sql.NullInt64
	err := q.QueryRow(`
		SELECT p.id, p.name, p.description, p.kind, p.buy_ratio, p.buy_floor_ratio, p.buy_tags,
		       p.quota_per_item, p.quota_period, p.sell_markup,
		       m.region_id, m.buy_tags, m.buy_ratio, m.buy_floor_ratio, m.quota_per_item
		FROM merchants m
		JOIN merchant_profiles p ON p.id = m.profile_id
		WHERE m.id = $1 AND m.is_active AND p.enabled
		  AND (m.expires_at IS NULL OR m.expires_at > NOW())`, merchantID).
		Scan(&p.ID, &p.Name, &p.Description, &p.Kind, &p.BuyRatio, &p.BuyFloorRatio, &tagsRaw,
			&p.QuotaPerItem, &p.QuotaPeriod, &p.SellMarkup,
			&regionID, &oTagsRaw, &oRatio, &oFloor, &oQuota)
	if err != nil {
		return p, "", err
	}
	_ = json.Unmarshal(tagsRaw, &p.BuyTags)
	if len(oTagsRaw) > 0 {
		var t merchantBuyTags
		if json.Unmarshal(oTagsRaw, &t) == nil {
			p.BuyTags = t
		}
	}
	if oRatio.Valid {
		p.BuyRatio = oRatio.Float64
	}
	if oFloor.Valid {
		p.BuyFloorRatio = oFloor.Float64
	}
	if oQuota.Valid {
		p.QuotaPerItem = int(oQuota.Int64)
	}
	return p, regionID, nil
}

// isRecipeItem reports whether itemID is a refining recipe key.
func isRecipeItem(q queryer, itemID string) bool {
	var exists bool
	_ = q.QueryRow(`SELECT EXISTS(SELECT 1 FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1)`, itemID).Scan(&exists)
	return exists
}

// recipeCompletionRate reads a player's completion (0-100) for a recipe key.
// ok=false means the player does not own that recipe (no progress row).
func recipeCompletionRate(q queryer, playerID, recipeID string) (float64, bool) {
	var rate float64
	err := q.QueryRow(`SELECT completion_rate FROM player_recipe_progress WHERE player_id=$1 AND recipe_id=$2`, playerID, recipeID).Scan(&rate)
	if err != nil {
		return 0, false
	}
	return rate, true
}

// merchantResolveBase resolves the dictionary base price and the effective rank
// for a polymorphic entity (Gu / Recipe / item).
func merchantResolveBase(q queryer, itemID, itemType string, rank int) (float64, int, error) {
	switch itemType {
	case "Gu":
		var basePrice float64
		var r int
		if rank > 0 {
			_ = q.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 AND rank=$2`, itemID, rank).Scan(&basePrice, &r)
		}
		if r == 0 {
			_ = q.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM gu_dictionary WHERE id=$1 ORDER BY rank ASC LIMIT 1`, itemID).Scan(&basePrice, &r)
		}
		if r == 0 {
			return 0, 0, fmt.Errorf("Không tìm thấy Cổ trùng %s", itemID)
		}
		return pricing.GuBasePrice(basePrice, r), r, nil
	case "Recipe":
		var targetRank int
		_ = q.QueryRow(`SELECT target_rank FROM refining_recipes WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1`, itemID).Scan(&targetRank)
		if targetRank <= 0 {
			targetRank = 1
		}
		return pricing.RecipeBasePrice(targetRank), targetRank, nil
	default: // Material, Consumable, ...
		var basePrice float64
		var r int
		_ = q.QueryRow(`SELECT COALESCE(base_price,0), COALESCE(rank,1) FROM item_dictionary WHERE id=$1`, itemID).Scan(&basePrice, &r)
		if r == 0 {
			r = 1
		}
		return pricing.ItemBasePrice(basePrice, r), r, nil
	}
}

// ---------------------------------------------------------------------------
// Placement
// ---------------------------------------------------------------------------

// ensureRegionMerchants lazily creates the fixed village merchant for a region
// that qualifies (HUB / SAFE_ZONE). Idempotent via deterministic ids.
func ensureRegionMerchants(regionID string) error {
	if regionID == "" {
		return nil
	}
	var zoneType string
	if err := db.DB.QueryRow(`SELECT zone_type FROM regions WHERE id=$1`, regionID).Scan(&zoneType); err != nil {
		return err
	}
	if zoneType != "HUB" && zoneType != "SAFE_ZONE" {
		return nil
	}
	if _, err := db.DB.Exec(`INSERT INTO merchants (id, profile_id, region_id, is_active)
		VALUES ('village_'||$1, 'village_general', $1, TRUE) ON CONFLICT (id) DO NOTHING`, regionID); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Read endpoints
// ---------------------------------------------------------------------------

type merchantStockItem struct {
	ItemID string  `json:"item_id"`
	Name   string  `json:"name"`
	Type   string  `json:"type"`
	Rank   int     `json:"rank"`
	Price  float64 `json:"price"`
	Stock  int     `json:"stock"`
}

type merchantDTO struct {
	ID           string              `json:"id"`
	RegionID     string              `json:"region_id"`
	X            int                 `json:"x"`
	Y            int                 `json:"y"`
	Distance     int                 `json:"distance"`
	ProfileID    string              `json:"profile_id"`
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	Kind         string              `json:"kind"`
	BuyRatio     float64             `json:"buy_ratio"`
	BuyFloor     float64             `json:"buy_floor_ratio"`
	BuyTags      merchantBuyTags     `json:"buy_tags"`
	QuotaPerItem int                 `json:"quota_per_item"`
	QuotaPeriod  string              `json:"quota_period"`
	Stock        []merchantStockItem `json:"stock"`
}

// roamingTradeRadius is how close a player must be to a roaming merchant to
// trade with it (adjacent, not just "in view").
const roamingTradeRadius = 2

func loadMerchantStock(merchantID string) []merchantStockItem {
	rows, err := db.DB.Query(`
		SELECT s.item_id, COALESCE(i.name, g.name, s.item_id),
		       CASE WHEN g.id IS NOT NULL THEN 'Gu' ELSE COALESCE(i.item_type,'Material') END,
		       s.rank, s.price, s.stock
		FROM merchant_stock s
		LEFT JOIN item_dictionary i ON i.id = s.item_id
		LEFT JOIN gu_dictionary g ON g.id = s.item_id AND g.rank = (SELECT MIN(g2.rank) FROM gu_dictionary g2 WHERE g2.id = s.item_id)
		WHERE s.merchant_id = $1
		ORDER BY s.sort_order, s.item_id`, merchantID)
	if err != nil {
		return []merchantStockItem{}
	}
	defer rows.Close()
	out := []merchantStockItem{}
	for rows.Next() {
		var it merchantStockItem
		if err := rows.Scan(&it.ItemID, &it.Name, &it.Type, &it.Rank, &it.Price, &it.Stock); err != nil {
			return out
		}
		if it.Stock == 0 {
			continue
		}
		out = append(out, it)
	}
	return out
}

// GetMerchants returns the roaming merchants the player is currently standing
// next to (within roamingTradeRadius) plus region/at_village flags so the UI can
// gate itself. The fixed village shop is the Caravan / Thương Hội UI.
func GetMerchants(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var region string
	var px, py int
	if err := db.DB.QueryRow(`SELECT current_region, x, y FROM players WHERE id=$1`, playerID).Scan(&region, &px, &py); err != nil || region == "" {
		c.JSON(http.StatusOK, gin.H{"region": region, "zone_type": "", "at_village": false, "merchants": []merchantDTO{}})
		return
	}
	entry, err := loadRegionMerchantCache(region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thương nhân"})
		return
	}
	atVillage := entry.zoneType == "HUB" || entry.zoneType == "SAFE_ZONE"
	out := []merchantDTO{}
	for i := range entry.rows {
		if m, ok := entry.rows[i].toDTO(px, py); ok {
			out = append(out, m)
		}
	}
	c.JSON(http.StatusOK, gin.H{"region": region, "zone_type": entry.zoneType, "at_village": atVillage, "merchants": out})
}

// merchantRow is the shared column set for merchant list/map queries.
const merchantRowColumns = `
	m.id, m.region_id, m.x, m.y, p.id, p.name, p.description, p.kind,
	p.buy_ratio, p.buy_floor_ratio, p.buy_tags, p.quota_per_item, p.quota_period,
	m.buy_tags, m.buy_ratio, m.buy_floor_ratio, m.quota_per_item`

// merchantRowData is one merchant row (profile fields with per-instance
// overrides already applied) without any player-specific distance or stock.
type merchantRowData struct {
	ID           string
	RegionID     string
	X            int
	Y            int
	ProfileID    string
	Name         string
	Description  string
	Kind         string
	BuyRatio     float64
	BuyFloor     float64
	BuyTags      merchantBuyTags
	QuotaPerItem int
	QuotaPeriod  string
}

// toDTO builds the per-player view, applying the trade-radius gate and loading
// stock only for merchants the player can actually trade with. Returns ok=false
// when the merchant is out of trade range.
func (r merchantRowData) toDTO(px, py int) (merchantDTO, bool) {
	dx, dy := r.X-px, r.Y-py
	sq := dx*dx + dy*dy
	if sq > roamingTradeRadius*roamingTradeRadius {
		return merchantDTO{}, false
	}
	m := merchantDTO{
		ID: r.ID, RegionID: r.RegionID, X: r.X, Y: r.Y,
		ProfileID: r.ProfileID, Name: r.Name, Description: r.Description, Kind: r.Kind,
		BuyRatio: r.BuyRatio, BuyFloor: r.BuyFloor, BuyTags: r.BuyTags,
		QuotaPerItem: r.QuotaPerItem, QuotaPeriod: r.QuotaPeriod,
		Distance: int(math.Sqrt(float64(sq))),
		Stock:    loadMerchantStock(r.ID),
	}
	return m, true
}

// The merchant set of a region is global (not per-player), so /merchant serves
// from the same short-TTL cache keyed by region instead of re-running the join.
type merchantRegionCacheEntry struct {
	rows     []merchantRowData
	zoneType string
	cachedAt time.Time
}

const merchantRegionCacheTTL = 10 * time.Second

var (
	merchantRegionCacheMu sync.RWMutex
	merchantRegionCache   = map[string]merchantRegionCacheEntry{}
)

// invalidateMerchantRegionCache drops a region's cached merchant rows. An empty
// regionID clears the whole cache (used after roaming spawn/expire batches).
func invalidateMerchantRegionCache(regionID string) {
	merchantRegionCacheMu.Lock()
	if regionID == "" {
		merchantRegionCache = map[string]merchantRegionCacheEntry{}
	} else {
		delete(merchantRegionCache, regionID)
	}
	merchantRegionCacheMu.Unlock()
}

// loadRegionMerchantCache returns a region's active roaming merchants (with
// overrides applied) and its zone type, cached for merchantRegionCacheTTL.
func loadRegionMerchantCache(region string) (merchantRegionCacheEntry, error) {
	merchantRegionCacheMu.RLock()
	entry, ok := merchantRegionCache[region]
	merchantRegionCacheMu.RUnlock()
	if ok && time.Since(entry.cachedAt) < merchantRegionCacheTTL {
		return entry, nil
	}

	var zoneType string
	_ = db.DB.QueryRow(`SELECT zone_type FROM regions WHERE id=$1`, region).Scan(&zoneType)

	rows, err := db.DB.Query(`
		SELECT `+merchantRowColumns+`
		FROM merchants m
		JOIN merchant_profiles p ON p.id = m.profile_id
		WHERE m.region_id = $1 AND m.is_active AND p.enabled AND p.kind = 'ROAMING'
		  AND (m.expires_at IS NULL OR m.expires_at > NOW())
		ORDER BY m.id`, region)
	if err != nil {
		return merchantRegionCacheEntry{}, err
	}
	defer rows.Close()
	out := []merchantRowData{}
	for rows.Next() {
		var r merchantRowData
		var tagsRaw, oTagsRaw []byte
		var oRatio, oFloor sql.NullFloat64
		var oQuota sql.NullInt64
		if err := rows.Scan(&r.ID, &r.RegionID, &r.X, &r.Y, &r.ProfileID, &r.Name, &r.Description, &r.Kind,
			&r.BuyRatio, &r.BuyFloor, &tagsRaw, &r.QuotaPerItem, &r.QuotaPeriod,
			&oTagsRaw, &oRatio, &oFloor, &oQuota); err != nil {
			continue
		}
		_ = json.Unmarshal(tagsRaw, &r.BuyTags)
		if len(oTagsRaw) > 0 {
			var t merchantBuyTags
			if json.Unmarshal(oTagsRaw, &t) == nil {
				r.BuyTags = t
			}
		}
		if oRatio.Valid {
			r.BuyRatio = oRatio.Float64
		}
		if oFloor.Valid {
			r.BuyFloor = oFloor.Float64
		}
		if oQuota.Valid {
			r.QuotaPerItem = int(oQuota.Int64)
		}
		out = append(out, r)
	}

	entry = merchantRegionCacheEntry{rows: out, zoneType: zoneType, cachedAt: time.Now()}
	merchantRegionCacheMu.Lock()
	merchantRegionCache[region] = entry
	merchantRegionCacheMu.Unlock()
	return entry, nil
}

// ---------------------------------------------------------------------------
// Sell to merchant (dynamic supply/demand pricing)
// ---------------------------------------------------------------------------

type merchantSellLine struct {
	ItemID   string `json:"item_id"`
	ItemType string `json:"item_type"`
	Quantity int64  `json:"quantity"`
	Rank     int    `json:"rank"`
}

type merchantQuoteRequest struct {
	MerchantID string             `json:"merchant_id" binding:"required"`
	Items      []merchantSellLine `json:"items" binding:"required,dive"`
}

type merchantQuoteItem struct {
	ItemID    string  `json:"item_id"`
	Rank      int     `json:"rank"`
	Accepted  bool    `json:"accepted"`
	UnitPrice float64 `json:"unit_price"`
	QuotaLeft int     `json:"quota_left"`
}

// QuoteMerchant reports the current unit price and remaining demand for a set of
// items without executing anything.
func QuoteMerchant(c *gin.Context) {
	var req merchantQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	playerID := authID.String()
	profile, _, err := loadMerchantEffective(db.DB, req.MerchantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thương nhân không tồn tại"})
		return
	}
	periodStart := merchantPeriodStart(profile.QuotaPeriod, time.Now())
	out := make([]merchantQuoteItem, 0, len(req.Items))
	for _, line := range req.Items {
		qi := merchantQuoteItem{ItemID: line.ItemID, Accepted: true}
		base, sellRank, berr := merchantResolveBase(db.DB, line.ItemID, line.ItemType, line.Rank)
		qi.Rank = sellRank
		if berr != nil || !profile.BuyTags.accepts(line.ItemType, sellRank, line.ItemID) {
			qi.Accepted = false
			out = append(out, qi)
			continue
		}
		if line.ItemType == "Recipe" {
			rate, owned := recipeCompletionRate(db.DB, playerID, line.ItemID)
			if !owned {
				qi.Accepted = false
				out = append(out, qi)
				continue
			}
			base *= pricing.RecipeCompletionFactor(rate)
		}
		var bought int
		_ = db.DB.QueryRow(`SELECT bought FROM merchant_demand WHERE merchant_id=$1 AND item_id=$2 AND period_start=$3`,
			req.MerchantID, line.ItemID, periodStart).Scan(&bought)
		left := profile.QuotaPerItem - bought
		if profile.QuotaPerItem <= 0 {
			left = -1 // unlimited
		}
		qi.QuotaLeft = left
		ratio := merchantBuyRatio(bought, profile.QuotaPerItem, profile.BuyRatio, profile.BuyFloorRatio)
		unit := pricing.Quantize(base * ratio)
		if unit < 0.001 {
			unit = 0.001
		}
		qi.UnitPrice = unit
		if left == 0 {
			qi.Accepted = false
		}
		out = append(out, qi)
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}

// executeMerchantSale removes the sold entities from the player's inventory,
// credits nothing (caller adds stones), and records demand. It is shared by
// /api/merchant/sell and the legacy /api/shop/sell delegation.
func executeMerchantSale(tx *sql.Tx, player *models.Player, merchantID string, profile *merchantProfile, lines []merchantSellLine) (float64, float64, []string, []string, bool, error) {
	now := time.Now().UTC()
	periodStart := merchantPeriodStart(profile.QuotaPeriod, now)
	totalPrimeval, totalImmortal := 0.0, 0.0
	summaries := []string{}
	removedGuIDs := []string{}
	guSold := false

	for _, line := range lines {
		if line.Quantity <= 0 {
			continue
		}
		base, sellRank, err := merchantResolveBase(tx, line.ItemID, line.ItemType, line.Rank)
		if err != nil {
			return 0, 0, nil, nil, false, err
		}
		if !profile.BuyTags.accepts(line.ItemType, sellRank, line.ItemID) {
			return 0, 0, nil, nil, false, fmt.Errorf("Thương nhân không thu mua %s", line.ItemID)
		}
		if line.ItemType == "Recipe" {
			rate, owned := recipeCompletionRate(tx, player.ID.String(), line.ItemID)
			if !owned {
				return 0, 0, nil, nil, false, fmt.Errorf("Đạo hữu không sở hữu bí phương %s", line.ItemID)
			}
			base *= pricing.RecipeCompletionFactor(rate)
		}
		var bought int
		_ = tx.QueryRow(`SELECT bought FROM merchant_demand WHERE merchant_id=$1 AND item_id=$2 AND period_start=$3`,
			merchantID, line.ItemID, periodStart).Scan(&bought)
		if profile.QuotaPerItem > 0 {
			remaining := int64(profile.QuotaPerItem - bought)
			if remaining <= 0 {
				return 0, 0, nil, nil, false, fmt.Errorf("Thương nhân đã hết nhu cầu %s trong kỳ này", line.ItemID)
			}
			if line.Quantity > remaining {
				return 0, 0, nil, nil, false, fmt.Errorf("Thương nhân chỉ còn nhận %d %s trong kỳ này", remaining, line.ItemID)
			}
		}

		// Remove the entity from the player's inventory.
		switch line.ItemType {
		case "Material", "Consumable":
			if err := removeInventoryItem(tx, player, line.ItemID, line.ItemType, line.Quantity); err != nil {
				return 0, 0, nil, nil, false, err
			}
		case "Gu":
			removed := int64(0)
			newInv := models.GuInventory{}
			for _, gu := range player.GuInventory {
				if gu.GuID == line.ItemID && !gu.IsVital && !gu.Equipped && gu.Status != "DEAD" && removed < line.Quantity {
					removed++
					removedGuIDs = append(removedGuIDs, gu.InstanceID)
					continue
				}
				newInv = append(newInv, gu)
			}
			if removed < line.Quantity {
				return 0, 0, nil, nil, false, fmt.Errorf("Không đủ Cổ trùng %s để bán hoặc Cổ là Bản Mệnh/Đang dùng/Đã chết", line.ItemID)
			}
			player.GuInventory = newInv
			guSold = true
		case "Recipe":
			if RecipeSvc != nil {
				_ = RecipeSvc.Delete(tx, player.ID.String(), line.ItemID)
			}
			line.Quantity = 1
		default:
			return 0, 0, nil, nil, false, fmt.Errorf("Loại vật phẩm không hợp lệ")
		}

		ratio := merchantBuyRatio(bought, profile.QuotaPerItem, profile.BuyRatio, profile.BuyFloorRatio)
		unit := pricing.Quantize(base * ratio)
		if unit < 0.001 {
			unit = 0.001
		}
		gain := pricing.Quantize(unit * float64(line.Quantity))
		if pricing.StoneTierByRank(sellRank) == "immortal" {
			totalImmortal += gain
		} else {
			totalPrimeval += gain
		}
		if _, err := tx.Exec(`
			INSERT INTO merchant_demand (merchant_id, item_id, period_start, bought)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (merchant_id, item_id, period_start)
			DO UPDATE SET bought = merchant_demand.bought + EXCLUDED.bought`,
			merchantID, line.ItemID, periodStart, line.Quantity); err != nil {
			return 0, 0, nil, nil, false, err
		}
		name := line.ItemID
		if line.ItemType == "Gu" {
			if spec, ok := models.GetGuAny(line.ItemID); ok {
				name = spec.Name
			}
		} else {
			_ = tx.QueryRow(`SELECT name FROM item_dictionary WHERE id=$1`, line.ItemID).Scan(&name)
		}
		summaries = append(summaries, fmt.Sprintf("%s x%d (+%.3f %s)", name, line.Quantity, gain, currencyLabel(sellRank)))
	}
	return totalPrimeval, totalImmortal, summaries, removedGuIDs, guSold, nil
}

// removeInventoryItem deducts a Material/Consumable quantity across categories.
func removeInventoryItem(tx *sql.Tx, player *models.Player, itemID, itemType string, qty int64) error {
	preferredCat := "materials"
	if itemType == "Consumable" {
		preferredCat = "consumables"
	}
	rows, err := tx.Query(`SELECT category, quantity FROM player_inventory_items WHERE player_id=$1 AND item_id=$2`, player.ID.String(), itemID)
	if err != nil {
		return fmt.Errorf("Không kiểm tra được túi đồ")
	}
	type invItem struct {
		category string
		quantity int64
	}
	var invs []invItem
	total := int64(0)
	for rows.Next() {
		var iv invItem
		if err := rows.Scan(&iv.category, &iv.quantity); err != nil {
			rows.Close()
			return fmt.Errorf("Không đọc được túi đồ")
		}
		invs = append(invs, iv)
		total += iv.quantity
	}
	rows.Close()
	if total < qty {
		return fmt.Errorf("Không đủ vật phẩm để bán")
	}
	remaining := qty
	deduct := func(cat string, preferred bool) error {
		for _, iv := range invs {
			if (iv.category == preferredCat) != preferred {
				continue
			}
			take := iv.quantity
			if take > remaining {
				take = remaining
			}
			if take <= 0 {
				continue
			}
			if err := InventorySvc.RemoveItem(tx, player.ID.String(), itemID, iv.category, take); err != nil {
				return fmt.Errorf("Không thể xóa vật phẩm khỏi túi đồ")
			}
			remaining -= take
			if remaining <= 0 {
				return nil
			}
		}
		return nil
	}
	if err := deduct(preferredCat, true); err != nil {
		return err
	}
	return deduct(preferredCat, false)
}

type merchantSellRequest struct {
	MerchantID string             `json:"merchant_id" binding:"required"`
	Items      []merchantSellLine `json:"items" binding:"required,dive"`
}

// SellToMerchant executes a sale against a merchant with dynamic pricing and a
// server-wide per-item quota.
func SellToMerchant(c *gin.Context) {
	var req merchantSellRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết, không thể giao dịch!"})
		return
	}
	profile, regionID, err := loadMerchantEffective(tx, req.MerchantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thương nhân không tồn tại"})
		return
	}
	if regionID != player.CurrentRegion {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không ở cùng khu vực với thương nhân này"})
		return
	}
	primeval, immortal, summaries, removedGuIDs, guSold, err := executeMerchantSale(tx, &player, req.MerchantID, &profile, req.Items)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	player.PrimevalStones += primeval
	player.ImmortalStones += immortal
	// Preserve the saved hotbar order in players.equipped_gus: prune invalid
	// entries and append newly equipped actives instead of rebuilding from
	// inventory order (which would reset the realtime hotbar assignment).
	RebuildEquippedGus(&player)
	if guSold {
		if err := db.RemovePlayerGus(tx, removedGuIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lưu được thay đổi Cổ trùng"})
			return
		}
	}
	if _, err := tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2, equipped_gus=$3 WHERE id=$4`,
		player.PrimevalStones, player.ImmortalStones, player.EquippedGus, player.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lưu được giao dịch"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	gainSummary := formatGainSummary(primeval, immortal)
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "merchant.sell",
		EntityType:     "merchant",
		EntityID:       req.MerchantID,
		RequestSummary: fmt.Sprintf("Bán cho %s: %s | Tổng nhận: %s", profile.Name, strings.Join(summaries, ", "), gainSummary),
		Metadata:       map[string]interface{}{"merchant_id": req.MerchantID, "profile_id": profile.ID, "lines": len(req.Items)},
		IPAddress:      c.ClientIP(),
		Success:        true,
	})
	notify.Send(db.DB, authID, "merchant_sell", "Bán Cho Thương Nhân",
		fmt.Sprintf("Đã bán cho %s, thu về %s.", profile.Name, gainSummary))
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": fmt.Sprintf("Bán thành công cho %s, nhận %s!", profile.Name, gainSummary)})
}

func formatGainSummary(primeval, immortal float64) string {
	if immortal > 0 && primeval > 0 {
		return fmt.Sprintf("+%.3f Nguyên Thạch và +%.3f Tiên Nguyên Thạch", primeval, immortal)
	}
	if immortal > 0 {
		return fmt.Sprintf("+%.3f Tiên Nguyên Thạch", immortal)
	}
	return fmt.Sprintf("+%.3f Nguyên Thạch", primeval)
}

// ---------------------------------------------------------------------------
// Buy from merchant
// ---------------------------------------------------------------------------

type merchantBuyLine struct {
	ItemID   string `json:"item_id" binding:"required"`
	Rank     int    `json:"rank"`
	Quantity int64  `json:"quantity" binding:"required,min=1"`
}

type merchantBuyRequest struct {
	MerchantID string            `json:"merchant_id" binding:"required"`
	Items      []merchantBuyLine `json:"items" binding:"required,dive"`
}

// BuyFromMerchant purchases items from a merchant's stock.
func BuyFromMerchant(c *gin.Context) {
	var req merchantBuyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết, không thể giao dịch!"})
		return
	}
	var regionID string
	if err := tx.QueryRow(`SELECT region_id FROM merchants WHERE id=$1 AND is_active`, req.MerchantID).Scan(&regionID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Thương nhân không tồn tại"})
		return
	}
	if regionID != player.CurrentRegion {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không ở cùng khu vực với thương nhân này"})
		return
	}

	before := append(models.GuInventory(nil), player.GuInventory...)
	beforePrimeval, beforeImmortal := player.PrimevalStones, player.ImmortalStones
	guBought := false
	for _, line := range req.Items {
		var price float64
		var stock, rank int
		if err := tx.QueryRow(`SELECT price, stock, rank FROM merchant_stock WHERE merchant_id=$1 AND item_id=$2 AND rank IN ($3,0) ORDER BY CASE WHEN rank=$3 THEN 0 ELSE 1 END LIMIT 1 FOR UPDATE`,
			req.MerchantID, line.ItemID, line.Rank).Scan(&price, &stock, &rank); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Thương nhân không bán %s", line.ItemID)})
			return
		}
		if stock >= 0 && int64(stock) < line.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Thương nhân không đủ %s trong kho", line.ItemID)})
			return
		}
		totalCost := pricing.Quantize(price * float64(line.Quantity))
		tier := pricing.StoneTierByRank(rank)
		if tier == "immortal" {
			if player.ImmortalStones < totalCost {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Tiên Nguyên Thạch"})
				return
			}
			player.ImmortalStones -= totalCost
		} else {
			if player.PrimevalStones < totalCost {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch"})
				return
			}
			player.PrimevalStones -= totalCost
		}
		// Determine entity kind and grant it.
		if spec, ok := models.GetGu(line.ItemID, rank); ok {
			for i := int64(0); i < line.Quantity; i++ {
				player.GuInventory = append(player.GuInventory, models.GuInstance{
					InstanceID: uuid.New().String(), GuID: spec.ID, Rank: spec.Rank,
					Hunger: 100, Status: "ACTIVE", Equipped: false,
				})
			}
			guBought = true
		} else if isRecipeItem(tx, line.ItemID) {
			if RecipeSvc == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở khóa bí phương"})
				return
			}
			if err := RecipeSvc.Upsert(tx, &services.RecipeProgress{
				PlayerID:       player.ID.String(),
				RecipeID:       line.ItemID,
				CompletionRate: 0,
				IsUnlocked:     true,
			}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi mở khóa bí phương"})
				return
			}
		} else {
			var itemType string
			_ = tx.QueryRow(`SELECT COALESCE(item_type,'Material') FROM item_dictionary WHERE id=$1`, line.ItemID).Scan(&itemType)
			category := "materials"
			switch itemType {
			case "Consumable":
				category = "consumables"
			case "Treasure":
				category = "treasures"
			case "ImmortalMaterial":
				category = "immortal_materials"
			}
			if err := InventorySvc.AddItem(tx, player.ID.String(), line.ItemID, category, line.Quantity); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thêm vật phẩm vào túi đồ"})
				return
			}
		}
		if stock >= 0 {
			if _, err := tx.Exec(`UPDATE merchant_stock SET stock = stock - $1 WHERE merchant_id=$2 AND item_id=$3 AND rank=$4`,
				line.Quantity, req.MerchantID, line.ItemID, rank); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không cập nhật được kho thương nhân"})
				return
			}
		}
	}
	if guBought {
		if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lưu được Cổ trùng mới"})
			return
		}
	}
	if _, err := tx.Exec(`UPDATE players SET primeval_stones=$1, immortal_stones=$2 WHERE id=$3`,
		player.PrimevalStones, player.ImmortalStones, player.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lưu được giao dịch"})
		return
	}
	spentPrimeval := beforePrimeval - player.PrimevalStones
	spentImmortal := beforeImmortal - player.ImmortalStones
	summaryLines := make([]string, 0, len(req.Items))
	for _, line := range req.Items {
		summaryLines = append(summaryLines, fmt.Sprintf("%s x%d", line.ItemID, line.Quantity))
	}
	if err := auditlog.LogChecked(tx, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "merchant.buy",
		EntityType:     "merchant",
		EntityID:       req.MerchantID,
		RequestSummary: fmt.Sprintf("Mua từ thương nhân %s: %s | Tổng chi: %s", req.MerchantID, strings.Join(summaryLines, ", "), formatGainSummary(-spentPrimeval, -spentImmortal)),
		Metadata: map[string]interface{}{
			"merchant_id":    req.MerchantID,
			"lines":          len(req.Items),
			"spent_primeval": spentPrimeval,
			"spent_immortal": spentImmortal,
			"gu_bought":      guBought,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi audit"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": "Mua hàng thành công!"})
}
