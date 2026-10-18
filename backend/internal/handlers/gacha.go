package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
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
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type GachaRewardItem struct {
	Type   string  `json:"type"`
	ID     string  `json:"id,omitempty"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
	Weight float64 `json:"weight"`
	Label  string  `json:"label"`
}

type GachaPool struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	CostType    string            `json:"cost_type"`
	CostAmount  float64           `json:"cost_amount"`
	Rewards     []GachaRewardItem `json:"rewards"`
}

var gachaPools = []GachaPool{
	{
		ID:          "da_thuong",
		Name:        "Đá thường",
		Description: "Mở Đá kỳ bí, có cơ hội nhận được Nguyên Thạch, vật liệu quý hiếm và Cổ trùng huyền bí!",
		CostType:    "primeval_stones",
		CostAmount:  2000,
		Rewards: []GachaRewardItem{
			{Type: "item", Min: 1, Max: 1, Weight: 25.0, Label: "Vật liệu cấp thấp"},
			{Type: "primeval_stones", Min: 2000, Max: 3000, Weight: 10.0, Label: "2.000~5.000 Nguyên Thạch"},
			{Type: "gu_rank", ID: "1", Min: 1, Max: 1, Weight: 12.0, Label: "Cổ Trùng R1"},
			{Type: "gu_rank", ID: "2", Min: 1, Max: 1, Weight: 6.0, Label: "Cổ Trùng R2"},
			{Type: "gu_rank", ID: "3", Min: 1, Max: 1, Weight: 2.0, Label: "Cổ Trùng R3"},
			{Type: "gu_rank", ID: "4", Min: 1, Max: 1, Weight: 3.0, Label: "Cổ Trùng R4"},
			{Type: "gu_rank", ID: "5", Min: 1, Max: 1, Weight: 1.0, Label: "Cổ Trùng R5"},
		},
	},
	{
		ID:          "da_vip",
		Name:        "Đá VIP",
		Description: "Mở Đá kỳ bí, có cơ hội nhận được Tiên Nguyên Thạch, vật liệu quý hiếm và Cổ trùng huyền bí!",
		CostType:    "immortal_stones",
		CostAmount:  1000,
		Rewards: []GachaRewardItem{
			{Type: "item", Min: 1, Max: 1, Weight: 25.0, Label: "Vật liệu cấp thấp"},
			{Type: "immortal_stones", Min: 500, Max: 1500, Weight: 10.0, Label: "2.000~5.000 Tiên Nguyên Thạch"},
			{Type: "gu_rank", ID: "3", Min: 1, Max: 1, Weight: 2.0, Label: "Cổ Trùng R3"},
			{Type: "gu_rank", ID: "4", Min: 1, Max: 1, Weight: 3.0, Label: "Cổ Trùng R4"},
			{Type: "gu_rank", ID: "5", Min: 1, Max: 1, Weight: 1.0, Label: "Cổ Trùng R5"},
			{Type: "gu_rank", ID: "6", Min: 1, Max: 1, Weight: 0.00001, Label: "Cổ Trùng R6"},
			{Type: "gu_rank", ID: "7", Min: 1, Max: 1, Weight: 0.0000001, Label: "Cổ Trùng R7"},
		},
	},
	{
		ID:          "da_thuong_pham",
		Name:        "Đá Thượng Phẩm",
		Description: "Mở Đá Thượng Phẩm, tỉ lệ ra nguyên liệu và Cổ trùng R4-R5 cao hơn hẳn. Không có bí phương, Thọ Nguyên.",
		CostType:    "primeval_stones",
		CostAmount:  20000,
		Rewards: []GachaRewardItem{
			{Type: "item", Min: 1, Max: 1, Weight: 30.0, Label: "Vật liệu"},
			{Type: "food", Min: 1, Max: 1, Weight: 25.0, Label: "Cổ lương"},
			{Type: "primeval_stones", Min: 5000, Max: 20000, Weight: 8.0, Label: "5.000~20.000 Nguyên Thạch"},
			{Type: "gu_rank", ID: "1", Min: 1, Max: 1, Weight: 8.0, Label: "Cổ Trùng R1"},
			{Type: "gu_rank", ID: "2", Min: 1, Max: 1, Weight: 7.0, Label: "Cổ Trùng R2"},
			{Type: "gu_rank", ID: "3", Min: 1, Max: 1, Weight: 5.0, Label: "Cổ Trùng R3"},
			{Type: "gu_rank", ID: "4", Min: 1, Max: 1, Weight: 4.0, Label: "Cổ Trùng R4"},
			{Type: "gu_rank", ID: "5", Min: 1, Max: 1, Weight: 2.0, Label: "Cổ Trùng R5"},
			{Type: "gu_rank", ID: "6", Min: 1, Max: 1, Weight: 0.00005, Label: "Cổ Trùng R6"},
		},
	},
	{
		ID:          "da_cuc_pham",
		Name:        "Đá Cực Phẩm",
		Description: "Mở Đá Cực Phẩm, tỉ lệ ra nguyên liệu dồi dào cùng cơ hội chạm tới Cổ trùng R6-R7. Không có bí phương, Thọ Nguyên.",
		CostType:    "primeval_stones",
		CostAmount:  100000,
		Rewards: []GachaRewardItem{
			{Type: "item", Min: 1, Max: 1, Weight: 32.0, Label: "Vật liệu"},
			{Type: "food", Min: 1, Max: 1, Weight: 28.0, Label: "Cổ lương"},
			{Type: "primeval_stones", Min: 20000, Max: 80000, Weight: 7.0, Label: "20.000~80.000 Nguyên Thạch"},
			{Type: "gu_rank", ID: "2", Min: 1, Max: 1, Weight: 6.0, Label: "Cổ Trùng R2"},
			{Type: "gu_rank", ID: "3", Min: 1, Max: 1, Weight: 6.0, Label: "Cổ Trùng R3"},
			{Type: "gu_rank", ID: "4", Min: 1, Max: 1, Weight: 5.0, Label: "Cổ Trùng R4"},
			{Type: "gu_rank", ID: "5", Min: 1, Max: 1, Weight: 3.0, Label: "Cổ Trùng R5"},
			{Type: "gu_rank", ID: "6", Min: 1, Max: 1, Weight: 0.0002, Label: "Cổ Trùng R6"},
			{Type: "gu_rank", ID: "7", Min: 1, Max: 1, Weight: 0.000002, Label: "Cổ Trùng R7"},
		},
	},
}

func GetGachaPools(c *gin.Context) {
	// Only the pool metadata the UI renders; reward tables (exact drop
	// weights) stay server-side.
	type gachaPoolView struct {
		ID          string  `json:"id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		CostType    string  `json:"cost_type"`
		CostAmount  float64 `json:"cost_amount"`
	}
	pools := make([]gachaPoolView, 0, len(gachaPools))
	for _, p := range gachaPools {
		pools = append(pools, gachaPoolView{
			ID: p.ID, Name: p.Name, Description: p.Description,
			CostType: p.CostType, CostAmount: p.CostAmount,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"pools": pools,
	})
}

type GachaOpenRequest struct {
	PoolID   string `json:"pool_id" binding:"required"`
	Quantity int    `json:"quantity"`
}

// ── Pre-computed weighted pools (built once per gacha open) ──────────

type weightedGuEntry struct {
	id     string
	weight float64
	rank   int
}

type weightedMaterialEntry struct {
	id     string
	weight float64
	rank   int // effectiveDropLevel, dùng cho boost rương Tiên Thạch
}

type weightedRecipeEntry struct {
	id         string
	targetGuID string
	targetRank int
	weight     float64
}

type precomputedPools struct {
	guByRank      map[string][]weightedGuEntry // "1,2,3" → pool
	materials     []weightedMaterialEntry
	foods         []weightedMaterialEntry
	drops         []weightedMaterialEntry          // Material + Food (rương mỏ)
	recipesByRank map[string][]weightedRecipeEntry // "all" → pool (mine chest)
	totalMat      float64
	totalFood     float64
	totalDrops    float64
}

var globalPrecomputedPools *precomputedPools
var gachaPoolsMu sync.RWMutex
var gachaPoolsLastUpdate time.Time

func getPrecomputedPools() *precomputedPools {
	gachaPoolsMu.RLock()
	if globalPrecomputedPools != nil && time.Since(gachaPoolsLastUpdate) < 1*time.Hour {
		defer gachaPoolsMu.RUnlock()
		return globalPrecomputedPools
	}
	gachaPoolsMu.RUnlock()

	gachaPoolsMu.Lock()
	defer gachaPoolsMu.Unlock()

	// Double check
	if globalPrecomputedPools != nil && time.Since(gachaPoolsLastUpdate) < 1*time.Hour {
		return globalPrecomputedPools
	}

	pc := &precomputedPools{
		guByRank:      make(map[string][]weightedGuEntry),
		recipesByRank: make(map[string][]weightedRecipeEntry),
	}

	// Build material and food pools once
	for id, spec := range models.ItemRegistry {
		lvl := effectiveDropLevel(spec.Rank, spec.Rarity)
		w := rankDropMultiplier(lvl)
		if w <= 0 {
			continue
		}
		switch spec.Type {
		case "Material":
			pc.materials = append(pc.materials, weightedMaterialEntry{id: id, weight: w, rank: lvl})
			pc.totalMat += w
			pc.drops = append(pc.drops, weightedMaterialEntry{id: id, weight: w, rank: lvl})
			pc.totalDrops += w
		case "Food":
			pc.foods = append(pc.foods, weightedMaterialEntry{id: id, weight: w, rank: lvl})
			pc.totalFood += w
			pc.drops = append(pc.drops, weightedMaterialEntry{id: id, weight: w, rank: lvl})
			pc.totalDrops += w
		}
	}

	// Build Gu pools for all rank combinations used in gacha
	rankFilters := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "1,2,3", "3,4", "4,5", "6,7", "8,9"}
	for _, rf := range rankFilters {
		var pool []weightedGuEntry
		rankSet := make(map[int]bool)
		for _, rs := range strings.Split(rf, ",") {
			r := 0
			fmt.Sscanf(rs, "%d", &r)
			rankSet[r] = true
		}
		for id, ranks := range models.GuRegistry {
			for _, spec := range ranks {
				if rankSet[spec.Rank] {
					w := guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))
					if w > 0 {
						pool = append(pool, weightedGuEntry{id: id, weight: w, rank: spec.Rank})
					}
				}
			}
		}
		pc.guByRank[rf] = pool
	}

	// Build the recipe pool for open_mine_chest (item effect).
	rows, err := db.DB.Query(`SELECT target_gu_id, target_rank, variant FROM refining_recipes`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var targetGuID string
			var targetRank, variant int
			if err := rows.Scan(&targetGuID, &targetRank, &variant); err == nil {
				// Rương mỏ: mọi rank đều có thể ra, rank cao hiếm hơn.
				weight := recipeDropMultiplier(targetRank)
				if weight > 0 {
					pc.recipesByRank["all"] = append(pc.recipesByRank["all"], weightedRecipeEntry{
						id:         fmt.Sprintf("recipe_%s_%d_%d", targetGuID, targetRank, variant),
						targetGuID: targetGuID,
						targetRank: targetRank,
						weight:     weight,
					})
				}
			}
		}
	}

	globalPrecomputedPools = pc
	gachaPoolsLastUpdate = time.Now()
	return pc
}

func (pc *precomputedPools) rollGu(rankKey string, excludedImmortal map[string]bool) (string, int) {
	pool := pc.guByRank[rankKey]
	if len(pool) == 0 {
		return "", 0
	}

	// Recalculate eligible weight so excluded immortal Gu can never be selected
	// by the fallback path and the remaining entries keep their relative odds.
	var total float64
	for _, g := range pool {
		if excludedImmortal == nil || !excludedImmortal[g.id] {
			total += g.weight
		}
	}
	if total <= 0 {
		return "", 0
	}
	r := rand.Float64() * total
	for _, g := range pool {
		if excludedImmortal != nil && excludedImmortal[g.id] {
			continue
		}
		r -= g.weight
		if r <= 0 {
			return g.id, g.rank
		}
	}
	return "", 0
}

func (pc *precomputedPools) rollMaterial() string {
	if len(pc.materials) == 0 || pc.totalMat <= 0 {
		return ""
	}
	r := rand.Float64() * pc.totalMat
	for _, m := range pc.materials {
		r -= m.weight
		if r <= 0 {
			return m.id
		}
	}
	return pc.materials[0].id
}

func (pc *precomputedPools) rollFood() string {
	if len(pc.foods) == 0 || pc.totalFood <= 0 {
		return ""
	}
	r := rand.Float64() * pc.totalFood
	for _, f := range pc.foods {
		r -= f.weight
		if r <= 0 {
			return f.id
		}
	}
	return pc.foods[0].id
}

// rollDrop picks from the combined Material + Food pool (rương mỏ).
func (pc *precomputedPools) rollDrop() string {
	if len(pc.drops) == 0 || pc.totalDrops <= 0 {
		return ""
	}
	r := rand.Float64() * pc.totalDrops
	for _, d := range pc.drops {
		r -= d.weight
		if r <= 0 {
			return d.id
		}
	}
	return pc.drops[0].id
}

// rollDropBoosted như rollDrop nhưng nhân trọng số các mục rank >= minRank lên
// factor lần. Rương Mỏ Tiên Thạch dùng để đẩy tỉ lệ vật phẩm Chuyển 6-9.
func (pc *precomputedPools) rollDropBoosted(minRank int, factor float64) string {
	if len(pc.drops) == 0 {
		return ""
	}
	weightOf := func(d weightedMaterialEntry) float64 {
		if d.rank >= minRank {
			return d.weight * factor
		}
		return d.weight
	}
	var total float64
	for _, d := range pc.drops {
		total += weightOf(d)
	}
	if total <= 0 {
		return pc.rollDrop()
	}
	r := rand.Float64() * total
	for _, d := range pc.drops {
		r -= weightOf(d)
		if r <= 0 {
			return d.id
		}
	}
	return pc.drops[0].id
}

// ── Immortal Gu cache (5-minute TTL) ────────────────────────────────

type immortalCache struct {
	mu       sync.RWMutex
	data     map[string]bool
	loadedAt int64 // unix timestamp
	ttlSec   int64
}

func getCachedImmortalGuIDs(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) (map[string]bool, error) {
	return db.GetExistingImmortalGuIDs(ctx, q)
}

func timeNowUnix() int64 {
	return time.Now().Unix()
}

func OpenGacha(c *gin.Context) {
	runGachaOpen(c, 0)
}

// OpenGacha10 kept for backward compatibility — forces a 10x open.
func OpenGacha10(c *gin.Context) {
	runGachaOpen(c, 10)
}

func runGachaOpen(c *gin.Context, forced int) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req GachaOpenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	quantity := forced
	if quantity <= 0 {
		quantity = req.Quantity
	}
	if quantity <= 0 {
		quantity = 1
	}
	if quantity > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi lần chỉ được mở tối đa 100 lần!"})
		return
	}

	var pool *GachaPool
	for i := range gachaPools {
		if gachaPools[i].ID == req.PoolID {
			pool = &gachaPools[i]
			break
		}
	}
	if pool == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gói quà không tồn tại"})
		return
	}

	// Populate the global catalog before reserving a transaction connection.
	// A cold cache queries refining_recipes through db.DB; doing that while a
	// transaction is open can exhaust the pool under concurrent gacha traffic.
	pc := getPrecomputedPools()
	excludedImmortal, err := getCachedImmortalGuIDs(c.Request.Context(), db.DB)
	if err != nil {
		logger.S.Errorf("gacha: load existing immortal Gu failed for player %s: %v", playerID, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Database đang bận, vui lòng thử lại"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()

	player, err := loadPlayerForStatus(tx, playerID, true)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết. Không thể rút quẻ!"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	// 10% discount when opening 10 or more times in one go
	totalCost := pool.CostAmount * float64(quantity)
	discounted := false
	if quantity >= 10 {
		totalCost *= 0.9
		discounted = true
	}
	if err := deductGachaCostAmount(&player, pool.CostType, totalCost); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Roll all rewards, batch items, apply non-item rewards in-memory
	rewards := make([]GachaResultItem, quantity)
	itemQtyMap := make(map[string]int64)
	for i := 0; i < quantity; i++ {
		rewards[i] = rollGachaRewardPrecomputed(pool, pc, excludedImmortal)
		switch rewards[i].Type {
		case "item":
			itemQtyMap[rewards[i].ID] += int64(rewards[i].Amount)
		default:
			applyGachaRewardInMemory(&player, rewards[i])
		}
		if rewards[i].Type == "gu" || rewards[i].Type == "gu_rank" {
			if models.HasImmortalRank(rewards[i].ID) {
				excludedImmortal[rewards[i].ID] = true
				// Pool doesn't need to be rebuilt anymore! rollGu skips excludedImmortal dynamically!
			}
		}
	}

	// Bulk upsert all item rewards in 1 query
	if len(itemQtyMap) > 0 {
		if err := InventorySvc.BulkAddItems(tx, player.ID.String(), "materials", itemQtyMap); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật vật phẩm"})
			return
		}
	}

	qiGained := 0.0

	updateQuery, updateArgs := buildGachaUpdateQuery(&player)
	_, err = tx.Exec(updateQuery, updateArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật người chơi"})
		fmt.Println("Error updating player:", err)
		return
	}
	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		if db.IsImmortalConflict(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Tiên Cổ này đã tồn tại trong thiên hạ, quay thưởng thất bại"})
			fmt.Println("Gacha immortal conflict:", err)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật cổ trùng"})
		fmt.Println("Error saving player Gu inventory:", err)
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit"})
		fmt.Println("Error committing transaction:", err)
		return
	}

	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, ItemInventoryDomain, CombatStatsDomain)

	// Batch rare Gu system messages into a single message
	var rareDrops []string
	for _, r := range rewards {
		if r.Type == "gu" && r.Rank >= 6 {
			if spec, ok := models.GetGuAny(r.ID); ok {
				rareDrops = append(rareDrops, fmt.Sprintf("%s (R%d)", spec.Name, r.Rank))
			}
		}
	}
	if len(rareDrops) > 0 {
		dropList := strings.Join(rareDrops, ", ")
		message := fmt.Sprintf("Đạo hữu %s đã mở được Cổ Trùng %s từ Mở Thạch %s!", player.Username, dropList, pool.Name)
		SendSystemMessage("world", sql.NullString{Valid: false}, message)
	}

	if quantity == 1 {
		respondActionWithInventory(c, http.StatusOK, player, gin.H{
			"reward":  rewards[0],
			"rewards": rewards,
			"message": buildGachaRewardLabel(rewards[0]) + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
		})
	} else {
		announcement := aggregateGachaRewards(rewards)
		discountMsg := ""
		if discounted {
			discountMsg = " Đã giảm 10% chi phí mở."
		}
		respondActionWithInventory(c, http.StatusOK, player, gin.H{
			"rewards": rewards,
			"message": fmt.Sprintf("Mở %d lần thành công! Nhân Khí +%.0f.%s\n", quantity, qiGained, discountMsg) + announcement,
		})
	}
}

type GachaResultItem struct {
	Type   string  `json:"type"`
	ID     string  `json:"id,omitempty"`
	Rank   int     `json:"rank,omitempty"`
	Amount float64 `json:"amount"`
	Label  string  `json:"label"`
	IsRare bool    `json:"is_rare"`
}

func rollGachaReward(pool *GachaPool, excludedImmortal map[string]bool) GachaResultItem {
	pc := getPrecomputedPools()
	return rollGachaRewardPrecomputed(pool, pc, excludedImmortal)
}

func rollGachaRewardPrecomputed(pool *GachaPool, pc *precomputedPools, excludedImmortal map[string]bool) GachaResultItem {
	var totalWeight float64
	for _, r := range pool.Rewards {
		totalWeight += r.Weight
	}

	roll := rand.Float64() * totalWeight
	var chosen GachaRewardItem
	for _, r := range pool.Rewards {
		roll -= r.Weight
		if roll <= 0 {
			chosen = r
			break
		}
	}

	amount := chosen.Min
	if chosen.Max > chosen.Min {
		amount = chosen.Min + rand.Float64()*(chosen.Max-chosen.Min)
	}

	isRare := chosen.Weight < 10.0

	switch chosen.Type {
	case "item":
		itemID := pc.rollMaterial()
		if itemID == "" {
			return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
		}
		itemName := itemID
		if spec, ok := models.ItemRegistry[itemID]; ok {
			itemName = spec.Name
		}
		return GachaResultItem{Type: "item", ID: itemID, Amount: 1, Label: fmt.Sprintf("1x %s", itemName), IsRare: isRare}

	case "food":
		foodID := pc.rollFood()
		if foodID == "" {
			return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
		}
		foodName := foodID
		if spec, ok := models.ItemRegistry[foodID]; ok {
			foodName = spec.Name
		}
		return GachaResultItem{Type: "item", ID: foodID, Amount: 1, Label: fmt.Sprintf("1x %s", foodName), IsRare: isRare}

	case "gu":
		guID, rank := pc.rollGu("1,2,3", excludedImmortal)
		if guID == "" {
			return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
		}
		guName := guID
		if spec, ok := models.GetGuAny(guID); ok {
			guName = spec.Name
		}
		return GachaResultItem{Type: "gu", ID: guID, Rank: rank, Amount: 1, Label: fmt.Sprintf("1x %s", guName), IsRare: isRare}

	case "gu_rank":
		guID, rank := pc.rollGu(chosen.ID, excludedImmortal)
		if guID == "" {
			return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
		}
		guName := guID
		if spec, ok := models.GetGuAny(guID); ok {
			guName = spec.Name
		}
		return GachaResultItem{Type: "gu", ID: guID, Rank: rank, Amount: 1, Label: fmt.Sprintf("1x %s", guName), IsRare: isRare}

	case "primeval_stones":
		return GachaResultItem{Type: "primeval_stones", Amount: amount, Label: fmt.Sprintf("%.0f Nguyên Thạch", amount), IsRare: isRare}

	case "immortal_stones":
		return GachaResultItem{Type: "immortal_stones", Amount: amount, Label: fmt.Sprintf("%.0f Tiên Nguyên Thạch", amount), IsRare: isRare}

	case "lifespan":
		return GachaResultItem{Type: "lifespan", Amount: amount, Label: fmt.Sprintf("+%.0f năm Thọ Nguyên", amount), IsRare: isRare}

	case "wall_progress":
		return GachaResultItem{Type: "wall_progress", Amount: amount, Label: fmt.Sprintf("+%.0f%% Phá Vách", amount*100), IsRare: isRare}

	case "human_qi":
		return GachaResultItem{Type: "human_qi", Amount: amount, Label: fmt.Sprintf("+%.0f Nhân Khí", amount), IsRare: isRare}

	default:
		return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
	}
}

func rollWeightedGu(rankFilter []string, excludedImmortal map[string]bool) string {
	type weighted struct {
		id     string
		weight float64
	}
	var pool []weighted
	var totalWeight float64

	for id, ranks := range models.GuRegistry {
		if excludedImmortal[id] {
			continue
		}
		for _, spec := range ranks {
			if len(rankFilter) > 0 {
				matched := false
				for _, rs := range rankFilter {
					r := 0
					fmt.Sscanf(rs, "%d", &r)
					if spec.Rank == r {
						matched = true
						break
					}
				}
				if !matched {
					continue
				}
			}
			w := guRarityWeightByRank(effectiveDropLevel(spec.Rank, spec.Rarity))
			if w > 0 {
				pool = append(pool, weighted{id: id, weight: w})
				totalWeight += w
			}
		}
	}

	if len(pool) == 0 || totalWeight <= 0 {
		return ""
	}

	r := rand.Float64() * totalWeight
	for _, g := range pool {
		r -= g.weight
		if r <= 0 {
			return g.id
		}
	}
	return pool[0].id
}

func rollWeightedMaterial() string {
	type weighted struct {
		id     string
		weight float64
	}
	var pool []weighted
	var totalWeight float64

	for id, spec := range models.ItemRegistry {
		if spec.Type != "Material" {
			continue
		}
		w := rankDropMultiplier(effectiveDropLevel(spec.Rank, spec.Rarity))
		if w > 0 {
			pool = append(pool, weighted{id: id, weight: w})
			totalWeight += w
		}
	}

	if len(pool) == 0 || totalWeight <= 0 {
		return ""
	}

	r := rand.Float64() * totalWeight
	for _, m := range pool {
		r -= m.weight
		if r <= 0 {
			return m.id
		}
	}
	return pool[0].id
}

func deductGachaCostAmount(player *models.Player, costType string, amount float64) error {
	switch costType {
	case "primeval_stones":
		if player.PrimevalStones < amount {
			return fmt.Errorf("Không đủ Nguyên Thạch. Cần %d Nguyên Thạch", int64(amount))
		}
		player.PrimevalStones -= float64(amount)
	case "lifespan":
		if player.Lifespan < amount {
			return fmt.Errorf("Không đủ Thọ Nguyên. Cần %.0f năm Thọ Nguyên", amount)
		}
		player.Lifespan -= amount
	case "immortal_stones":
		if player.ImmortalStones < amount {
			return fmt.Errorf("Không đủ Tiên Nguyên Thạch. Cần %d Tiên Nguyên Thạch", int64(amount))
		}
		player.ImmortalStones -= float64(amount)
	default:
		return fmt.Errorf("Loại chi phí không hợp lệ: %s", costType)
	}
	return nil
}

// applyGachaRewardInMemory modifies player state in-memory only (no DB calls).
// Item rewards are handled separately via BulkAddItems.
func applyGachaRewardInMemory(player *models.Player, reward GachaResultItem) {
	switch reward.Type {
	case "primeval_stones":
		player.PrimevalStones += float64(reward.Amount)
	case "immortal_stones":
		player.ImmortalStones += float64(reward.Amount)
	case "lifespan":
		player.Lifespan += reward.Amount
	case "human_qi":
		player.HumanQi += reward.Amount
		if player.HumanQi > MaxHumanQi {
			player.HumanQi = MaxHumanQi
		}
	case "wall_progress":
		player.WallProgress += reward.Amount
		if player.WallProgress > 1.0 {
			player.WallProgress = 1.0
		}
	case "gu":
		guSpec, _ := models.GetGuAny(reward.ID)
		rank := reward.Rank
		if rank == 0 {
			rank = guSpec.Rank
		}
		newGu := models.GuInstance{
			InstanceID: "gu-" + uuid.New().String(),
			GuID:       reward.ID,
			Rank:       rank,
			Hunger:     rand.Intn(21),
			Status:     "ACTIVE",
			Equipped:   false,
		}
		player.GuInventory = append(player.GuInventory, newGu)
	}
}

func buildGachaUpdateQuery(player *models.Player) (string, []interface{}) {
	baseFields := []string{"primeval_stones", "immortal_stones", "lifespan", "human_qi", "wall_progress"}
	baseVals := []interface{}{player.PrimevalStones, player.ImmortalStones, player.Lifespan, player.HumanQi, player.WallProgress}
	extraFields := []string{}
	extraVals := []interface{}{}

	// materials_inventory no longer stored on players table

	allFields := append(baseFields, extraFields...)
	allVals := append(baseVals, extraVals...)

	placeholders := make([]string, len(allFields))
	for i := range allFields {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	setClauses := make([]string, len(allFields))
	for i, f := range allFields {
		setClauses[i] = fmt.Sprintf("%s = %s", f, placeholders[i])
	}

	query := fmt.Sprintf("UPDATE players SET %s WHERE id = $%d",
		strings.Join(setClauses, ", "),
		len(allFields)+1,
	)
	allVals = append(allVals, player.ID)

	return query, allVals
}

func buildGachaRewardLabel(r GachaResultItem) string {
	if r.Label != "" {
		return r.Label
	}
	return fmt.Sprintf("%.0f %s", r.Amount, r.Type)
}

func guRarityWeightByRank(rank int) float64 {
	switch {
	case rank <= 1:
		return 100.0
	case rank == 2:
		return 60.0
	case rank == 3:
		return 30.0
	case rank == 4:
		return 12.0
	case rank == 5:
		return 5.0
	case rank == 6:
		return 1.5
	case rank == 7:
		return 0.5
	case rank == 8:
		return 0.15
	default:
		return 0.05
	}
}

func aggregateGachaRewards(rewards []GachaResultItem) string {
	type aggVal struct {
		amount float64
		label  string // base name without amount prefix
		isRare bool
	}

	var order []string
	aggMap := make(map[string]*aggVal)

	for _, r := range rewards {
		var key string
		var name string

		switch r.Type {
		case "primeval_stones":
			key = "primeval_stones"
			name = "Nguyên Thạch"
		case "immortal_stones":
			key = "immortal_stones"
			name = "Tiên Nguyên Thạch"
		case "lifespan":
			key = "lifespan"
			name = "năm Thọ Nguyên"
		case "wall_progress":
			key = "wall_progress"
			name = "% Phá Vách"
		case "human_qi":
			key = "human_qi"
			name = "Nhân Khí"
		case "item":
			key = "item:" + r.ID
			name = r.ID
			if spec, ok := models.ItemRegistry[r.ID]; ok {
				name = spec.Name
			}
		case "gu":
			key = "gu:" + r.ID
			name = r.ID
			if spec, ok := models.GetGuAny(r.ID); ok {
				name = spec.Name
			}
		default:
			key = r.Type + ":" + r.ID
			name = r.Label
		}

		if _, exists := aggMap[key]; !exists {
			order = append(order, key)
			aggMap[key] = &aggVal{
				amount: r.Amount,
				label:  name,
				isRare: r.IsRare,
			}
		} else {
			aggMap[key].amount += r.Amount
		}
	}

	var lines []string
	for _, key := range order {
		val := aggMap[key]
		var line string
		if strings.HasPrefix(key, "item:") || strings.HasPrefix(key, "gu:") {
			line = fmt.Sprintf("%.0fx %s", val.amount, val.label)
		} else if key == "primeval_stones" {
			line = fmt.Sprintf("%.0f %s", val.amount, val.label)
		} else if key == "immortal_stones" {
			line = fmt.Sprintf("%.0f %s", val.amount, val.label)
		} else if key == "lifespan" {
			line = fmt.Sprintf("+%.0f %s", val.amount, val.label)
		} else if key == "wall_progress" {
			line = fmt.Sprintf("+%.0f%% Phá Vách", val.amount*100)
		} else if key == "human_qi" {
			line = fmt.Sprintf("+%.0f %s", val.amount, val.label)
		} else {
			line = fmt.Sprintf("%.0f %s", val.amount, val.label)
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}
