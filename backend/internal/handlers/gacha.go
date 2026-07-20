package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
		CostAmount:  1000,
		Rewards: []GachaRewardItem{
			{Type: "primeval_stones", Min: 500, Max: 1500, Weight: 40.0, Label: "500~1.500 Nguyên Thạch"},
			{Type: "item", Min: 1, Max: 1, Weight: 25.0, Label: "Vật liệu cấp thấp"},
			{Type: "gu", Min: 1, Max: 1, Weight: 20.0, Label: "Cổ Trùng R1~R3"},
			{Type: "primeval_stones", Min: 2000, Max: 5000, Weight: 10.0, Label: "2.000~5.000 Nguyên Thạch"},
			{Type: "gu_rank", ID: "4,5", Min: 1, Max: 1, Weight: 4.0, Label: "Cổ Trùng R4~R5"},
			{Type: "gu_rank", ID: "6,7,8,9", Min: 1, Max: 1, Weight: 1.0, Label: "Cổ Trùng R6+ Cao Cấp"},
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
	PoolID string `json:"pool_id" binding:"required"`
}

func OpenGacha(c *gin.Context) {
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

	if err := deductGachaCost(&player, pool); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reward := rollGachaReward(pool)

	if err := applyGachaReward(tx, &player, reward); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi áp dụng phần thưởng"})
		return
	}

	updateQuery, updateArgs := buildGachaUpdateQuery(&player)
	_, err = tx.Exec(updateQuery, updateArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật người chơi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"reward":  reward,
		"message": buildGachaRewardLabel(reward),
	})
}

func OpenGacha10(c *gin.Context) {
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

	// 10x cost with 10% discount
	totalCost := pool.CostAmount * 10 * 0.9
	if err := deductGachaCostAmount(&player, pool.CostType, totalCost); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rewards := make([]GachaResultItem, 10)
	for i := 0; i < 10; i++ {
		rewards[i] = rollGachaReward(pool)
		if err := applyGachaReward(tx, &player, rewards[i]); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi áp dụng phần thưởng"})
			return
		}
	}

	updateQuery, updateArgs := buildGachaUpdateQuery(&player)
	_, err = tx.Exec(updateQuery, updateArgs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật người chơi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit"})
		return
	}

	announcement := ""
	for _, r := range rewards {
		announcement += buildGachaRewardLabel(r) + "\n"
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"rewards": rewards,
		"message": "Mở 10 lần thành công!\n" + announcement,
	})
}

type GachaResultItem struct {
	Type   string  `json:"type"`
	ID     string  `json:"id,omitempty"`
	Amount float64 `json:"amount"`
	Label  string  `json:"label"`
	IsRare bool    `json:"is_rare"`
}

func rollGachaReward(pool *GachaPool) GachaResultItem {
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
		itemID := rollWeightedMaterial()
		if itemID == "" {
			return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
		}
		itemName := itemID
		if spec, ok := models.ItemRegistry[itemID]; ok {
			itemName = spec.Name
		}
		return GachaResultItem{Type: "item", ID: itemID, Amount: 1, Label: fmt.Sprintf("1x %s", itemName), IsRare: isRare}

	case "gu":
		guID := rollWeightedGu(nil)
		if guID == "" {
			return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
		}
		guName := guID
		if spec, ok := models.GuRegistry[guID]; ok {
			guName = spec.Name
		}
		return GachaResultItem{Type: "gu", ID: guID, Amount: 1, Label: fmt.Sprintf("1x %s", guName), IsRare: isRare}

	case "gu_rank":
		rankList := strings.Split(chosen.ID, ",")
		guID := rollWeightedGu(rankList)
		if guID == "" {
			return GachaResultItem{Type: "primeval_stones", Amount: 100, Label: "100 Nguyên Thạch"}
		}
		guName := guID
		if spec, ok := models.GuRegistry[guID]; ok {
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

func rollWeightedGu(rankFilter []string) string {
	type weighted struct {
		id     string
		weight float64
	}
	var pool []weighted
	var totalWeight float64

	for id, spec := range models.GuRegistry {
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

func deductGachaCost(player *models.Player, pool *GachaPool) error {
	return deductGachaCostAmount(player, pool.CostType, pool.CostAmount)
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

func applyGachaReward(tx *sql.Tx, player *models.Player, reward GachaResultItem) error {
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
	case "item":
		if player.MaterialsInventory == nil {
			player.MaterialsInventory = make(models.MaterialsInventory)
		}
		player.MaterialsInventory[reward.ID] += int(reward.Amount)
	case "gu":
		newGu := models.GuInstance{
			InstanceID: "gu-" + uuid.New().String(),
			GuID:       reward.ID,
			Hunger:     rand.Intn(21),
			Status:     "ACTIVE",
			Equipped:   false,
		}
		player.GuInventory = append(player.GuInventory, newGu)
	}
	return nil
}

func buildGachaUpdateQuery(player *models.Player) (string, []interface{}) {
	baseFields := []string{"primeval_stones", "lifespan", "human_qi", "wall_progress"}
	baseVals := []interface{}{player.PrimevalStones, player.Lifespan, player.HumanQi, player.WallProgress}
	extraFields := []string{}
	extraVals := []interface{}{}

	if len(player.MaterialsInventory) > 0 {
		extraFields = append(extraFields, "materials_inventory")
		extraVals = append(extraVals, player.MaterialsInventory)
	}
	if len(player.GuInventory) > 0 {
		extraFields = append(extraFields, "gu_inventory")
		extraVals = append(extraVals, player.GuInventory)
	}

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
