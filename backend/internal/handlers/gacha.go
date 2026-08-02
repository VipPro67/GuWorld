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

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
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
		CostAmount:  2500,
		Rewards: []GachaRewardItem{
			{Type: "primeval_stones", Min: 500, Max: 1500, Weight: 40.0, Label: "500~1.500 Nguyên Thạch"},
			{Type: "item", Min: 1, Max: 1, Weight: 25.0, Label: "Vật liệu cấp thấp"},
			{Type: "gu", Min: 1, Max: 1, Weight: 20.0, Label: "Cổ Trùng R1~R3"},
			{Type: "primeval_stones", Min: 2000, Max: 5000, Weight: 10.0, Label: "2.000~5.000 Nguyên Thạch"},
			{Type: "gu_rank", ID: "4,5", Min: 1, Max: 1, Weight: 4.0, Label: "Cổ Trùng R4~R5"},
			{Type: "gu_rank", ID: "6,7", Min: 1, Max: 1, Weight: 0.004, Label: "Cổ Trùng R6+ Cao Cấp"},
			{Type: "gu_rank", ID: "8,9", Min: 1, Max: 1, Weight: 0.00004, Label: "Cổ Trùng R8+ Siêu Cấp"},
		},
	},
	// {
	// 	ID:          "tho_nguyen_co_tu",
	// 	Name:        "Thọ Nguyên Cổ Từ",
	// 	Description: "Đánh đổi thọ nguyên để nhận phúc duyên! Càng đánh cược thì càng có thể gia tăng thọ nguyên vĩnh viễn và đột phá tu vi!",
	// 	CostType:    "lifespan",
	// 	CostAmount:  50,
	// 	Rewards: []GachaRewardItem{
	// 		{Type: "lifespan", Min: 5, Max: 10, Weight: 50.0, Label: "+5~10 năm Thọ Nguyên"},
	// 		{Type: "wall_progress", Min: 0.05, Max: 0.15, Weight: 25.0, Label: "+5%~15% Phá Vách"},
	// 		{Type: "human_qi", Min: 500, Max: 2000, Weight: 15.0, Label: "+500~2.000 Nhân Khí"},
	// 		{Type: "lifespan", Min: 10, Max: 30, Weight: 10.0, Label: "+10~30 năm Thọ Nguyên"},
	// 	},
	// },
	// {
	// 	ID:          "van_menh_co_tu",
	// 	Name:        "Vận Mệnh Cổ Từ",
	// 	Description: "Truyền thuyết về Cổ Từ Vận Mệnh chứa đựng những Tiên Cổ siêu phàm! Mở ra để thử vận may đột phá Tiên Đạo!",
	// 	CostType:    "primeval_stones",
	// 	CostAmount:  5000,
	// 	Rewards: []GachaRewardItem{
	// 		{Type: "gu_rank", ID: "3,4", Min: 1, Max: 1, Weight: 30.0, Label: "Cổ Trùng R3~R4"},
	// 		{Type: "gu_rank", ID: "5", Min: 1, Max: 1, Weight: 35.0, Label: "Cổ Trùng R5"},
	// 		{Type: "gu_rank", ID: "6", Min: 1, Max: 1, Weight: 20.0, Label: "Cổ Trùng R6"},
	// 		{Type: "gu_rank", ID: "7", Min: 1, Max: 1, Weight: 10.0, Label: "Cổ Trùng R7"},
	// 		{Type: "gu_rank", ID: "8", Min: 1, Max: 1, Weight: 4.0, Label: "Cổ Trùng R8"},
	// 		{Type: "gu_rank", ID: "9", Min: 1, Max: 1, Weight: 1.0, Label: "Cổ Trùng R9"},
	// 	},
	// },
}

func GetGachaPools(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"pools": gachaPools,
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
}

type weightedMaterialEntry struct {
	id     string
	weight float64
}

type precomputedPools struct {
	guByRank   map[string][]weightedGuEntry // "1,2,3" → pool
	materials  []weightedMaterialEntry
	totalByRank map[string]float64
	totalMat    float64
}

func precomputeGachaPools(excludedImmortal map[string]bool) precomputedPools {
	pc := precomputedPools{
		guByRank:    make(map[string][]weightedGuEntry),
		totalByRank: make(map[string]float64),
	}

	// Build material pool once
	for id, spec := range models.ItemRegistry {
		if spec.Type != "Material" {
			continue
		}
		w := rankDropMultiplier(spec.Rank)
		if w > 0 {
			pc.materials = append(pc.materials, weightedMaterialEntry{id: id, weight: w})
			pc.totalMat += w
		}
	}

	// Build Gu pools for all rank combinations used in gacha
	rankFilters := []string{"1", "2", "3", "4,5", "6,7", "8,9"}
	for _, rf := range rankFilters {
		var pool []weightedGuEntry
		var total float64
		rankSet := make(map[int]bool)
		for _, rs := range strings.Split(rf, ",") {
			r := 0
			fmt.Sscanf(rs, "%d", &r)
			rankSet[r] = true
		}
		for id, ranks := range models.GuRegistry {
			if excludedImmortal[id] {
				continue
			}
			for _, spec := range ranks {
				if rankSet[spec.Rank] {
					w := guRarityWeightByRank(spec.Rank)
					if w > 0 {
						pool = append(pool, weightedGuEntry{id: id, weight: w})
						total += w
					}
				}
			}
		}
		pc.guByRank[rf] = pool
		pc.totalByRank[rf] = total
	}

	return pc
}

func (pc *precomputedPools) rollGu(rankKey string) string {
	pool := pc.guByRank[rankKey]
	total := pc.totalByRank[rankKey]
	if len(pool) == 0 || total <= 0 {
		return ""
	}
	r := rand.Float64() * total
	for _, g := range pool {
		r -= g.weight
		if r <= 0 {
			return g.id
		}
	}
	return pool[0].id
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

// ── Immortal Gu cache (5-minute TTL) ────────────────────────────────

type immortalCache struct {
	mu        sync.RWMutex
	data      map[string]bool
	loadedAt  int64 // unix timestamp
	ttlSec    int64
}

var globalImmortalCache = &immortalCache{ttlSec: 300}

func getCachedImmortalGuIDs(tx *sql.Tx) (map[string]bool, error) {
	c := globalImmortalCache
	c.mu.RLock()
	if c.data != nil && timeNowUnix()-c.loadedAt < c.ttlSec {
		result := make(map[string]bool, len(c.data))
		for k, v := range c.data {
			result[k] = v
		}
		c.mu.RUnlock()
		return result, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	// Double-check after acquiring write lock
	if c.data != nil && timeNowUnix()-c.loadedAt < c.ttlSec {
		result := make(map[string]bool, len(c.data))
		for k, v := range c.data {
			result[k] = v
		}
		return result, nil
	}

	data, err := db.GetExistingImmortalGuIDs(tx)
	if err != nil {
		return nil, err
	}
	c.data = data
	c.loadedAt = timeNowUnix()

	result := make(map[string]bool, len(data))
	for k, v := range data {
		result[k] = v
	}
	return result, nil
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

	tx, err := db.DB.Begin()
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

	excludedImmortal, err := getCachedImmortalGuIDs(tx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}

	// Pre-compute weighted pools once
	pc := precomputeGachaPools(excludedImmortal)

	// Roll all rewards, batch items, apply non-item rewards in-memory
	rewards := make([]GachaResultItem, quantity)
	itemQtyMap := make(map[string]int)
	for i := 0; i < quantity; i++ {
		rewards[i] = rollGachaRewardPrecomputed(pool, pc, excludedImmortal)
		switch rewards[i].Type {
		case "item":
			itemQtyMap[rewards[i].ID] += int(rewards[i].Amount)
		default:
			applyGachaRewardInMemory(&player, rewards[i])
		}
		if rewards[i].Type == "gu" {
			if spec, ok := models.GetGuAny(rewards[i].ID); ok && spec.Rank >= 6 {
				excludedImmortal[rewards[i].ID] = true
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

	qiGained := AddHumanQi(&player, float64(quantity))

	updateQuery, updateArgs := buildGachaUpdateQuery(&player)
	_, err = tx.Exec(updateQuery, updateArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật người chơi"})
		return
	}
	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật cổ trùng"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit"})
		return
	}

	action := "gacha.open"
	if quantity > 1 {
		action = "gacha.open_bulk"
	}
	summary := fmt.Sprintf("pool=%s | %dx | cost=%.0f %s", pool.Name, quantity, totalCost, pool.CostType)
	if discounted {
		summary += " (đã giảm 10%)"
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         action,
		EntityType:     "gacha_pool",
		EntityID:       pool.ID,
		RequestSummary: summary,
		Metadata: map[string]interface{}{
			"pool_id":     pool.ID,
			"cost_type":   pool.CostType,
			"cost_amount": totalCost,
			"quantity":    quantity,
			"rewards":     rewards,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

	if quantity == 1 {
		responseWithPlayerData(c, http.StatusOK, player, player.MaterialsInventory, loadCompletedQuests(player.ID.String()), gin.H{
			"reward":  rewards[0],
			"message": buildGachaRewardLabel(rewards[0]) + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
		})
	} else {
		announcement := aggregateGachaRewards(rewards)
		discountMsg := ""
		if discounted {
			discountMsg = " Đã giảm 10% chi phí mở."
		}
		responseWithPlayerData(c, http.StatusOK, player, player.MaterialsInventory, loadCompletedQuests(player.ID.String()), gin.H{
			"rewards": rewards,
			"message": fmt.Sprintf("Mở %d lần thành công! Nhân Khí +%.0f.%s\n", quantity, qiGained, discountMsg) + announcement,
		})
	}

	// Batch rare Gu system messages into a single message
	var rareDrops []string
	for _, r := range rewards {
		if r.Type == "gu" {
			if spec, ok := models.GetGuAny(r.ID); ok && spec.Rank >= 6 {
				rareDrops = append(rareDrops, fmt.Sprintf("%s (R%d)", spec.Name, spec.Rank))
			}
		}
	}
	if len(rareDrops) > 0 {
		dropList := strings.Join(rareDrops, ", ")
		message := fmt.Sprintf("Đạo hữu %s đã mở được Cổ Trùng %s từ Mở Thạch %s!", player.Username, dropList, pool.Name)
		SendSystemMessage("world", sql.NullString{Valid: false}, message)
	}
}

type GachaResultItem struct {
	Type   string  `json:"type"`
	ID     string  `json:"id,omitempty"`
	Amount float64 `json:"amount"`
	Label  string  `json:"label"`
	IsRare bool    `json:"is_rare"`
}

func rollGachaReward(pool *GachaPool, excludedImmortal map[string]bool) GachaResultItem {
	pc := precomputeGachaPools(excludedImmortal)
	return rollGachaRewardPrecomputed(pool, pc, excludedImmortal)
}

func rollGachaRewardPrecomputed(pool *GachaPool, pc precomputedPools, excludedImmortal map[string]bool) GachaResultItem {
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

	case "gu":
		guID := pc.rollGu("1,2,3")
		if guID == "" {
			return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
		}
		guName := guID
		if spec, ok := models.GetGuAny(guID); ok {
			guName = spec.Name
		}
		return GachaResultItem{Type: "gu", ID: guID, Amount: 1, Label: fmt.Sprintf("1x %s", guName), IsRare: isRare}

	case "gu_rank":
		guID := pc.rollGu(chosen.ID)
		if guID == "" {
			return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
		}
		guName := guID
		if spec, ok := models.GetGuAny(guID); ok {
			guName = spec.Name
		}
		return GachaResultItem{Type: "gu", ID: guID, Amount: 1, Label: fmt.Sprintf("1x %s", guName), IsRare: isRare}

	case "primeval_stones":
		return GachaResultItem{Type: "primeval_stones", Amount: amount, Label: fmt.Sprintf("%.0f Nguyên Thạch", amount), IsRare: isRare}

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
			w := guRarityWeightByRank(spec.Rank)
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
		w := rankDropMultiplier(spec.Rank)
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
		if float64(player.PrimevalStones) < amount {
			return fmt.Errorf("Không đủ Nguyên Thạch. Cần %.0f Nguyên Thạch", amount)
		}
		player.PrimevalStones -= int(amount)
	case "lifespan":
		if player.Lifespan < amount {
			return fmt.Errorf("Không đủ Thọ Nguyên. Cần %.0f năm Thọ Nguyên", amount)
		}
		player.Lifespan -= amount
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
		player.PrimevalStones += int(reward.Amount)
	case "lifespan":
		player.Lifespan += reward.Amount
	case "human_qi":
		player.HumanQi += reward.Amount
		if player.HumanQi > 100000.0 {
			player.HumanQi = 100000.0
		}
	case "wall_progress":
		player.WallProgress += reward.Amount
		if player.WallProgress > 1.0 {
			player.WallProgress = 1.0
		}
	case "gu":
		guSpec, _ := models.GetGuAny(reward.ID)
		newGu := models.GuInstance{
			InstanceID: "gu-" + uuid.New().String(),
			GuID:       reward.ID,
			Rank:       guSpec.Rank,
			Hunger:     rand.Intn(21),
			Status:     "ACTIVE",
			Equipped:   false,
		}
		player.GuInventory = append(player.GuInventory, newGu)
	}
}

func buildGachaUpdateQuery(player *models.Player) (string, []interface{}) {
	baseFields := []string{"primeval_stones", "lifespan", "human_qi", "wall_progress"}
	baseVals := []interface{}{player.PrimevalStones, player.Lifespan, player.HumanQi, player.WallProgress}
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
				amount: 0,
				label:  name,
				isRare: r.IsRare,
			}
		}
		aggMap[key].amount += r.Amount
	}

	var lines []string
	for _, key := range order {
		val := aggMap[key]
		var line string
		if strings.HasPrefix(key, "item:") || strings.HasPrefix(key, "gu:") {
			line = fmt.Sprintf("%.0fx %s", val.amount, val.label)
		} else if key == "primeval_stones" {
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

