package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// CollectApertureResource claims the produced resources from a node
func CollectApertureResource(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	nodeIDStr := c.Param("id")
	nodeID, err := uuid.Parse(nodeIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã tài nguyên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch node details
	var n models.ApertureResourceNode
	var lastCol sql.NullTime
	var areaSize int64
	var timeRate float64
	var pTSM float64
	var pRace string
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.production_type, n.production_amount, n.production_interval_hours, n.last_collected_at, n.is_destroyed, a.area_size, a.time_rate, n.path, n.x, n.y, COALESCE(n.production_key, ''), COALESCE(n.level, 1), p.time_speed_multiplier, COALESCE(p.race, '')
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		JOIN players p ON a.player_id = p.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.ProductionType, &n.ProductionAmount, &n.ProductionIntervalHours, &lastCol, &n.IsDestroyed, &areaSize, &timeRate, &n.Path, &n.X, &n.Y, &n.ProductionKey, &n.Level, &pTSM, &pRace)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}

	if n.IsDestroyed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ tài nguyên này đã bị tàn phá không thể thu hoạch"})
		return
	}

	if timeRate <= 0 {
		timeRate = 1.0
	}
	tsm := 1.0
	dummyP := models.Player{}
	dummyP.Race = pRace
	if !gamefx.Get(&dummyP).Immortal {
		tsm = pTSM
		if tsm <= 0 {
			tsm = 1.0
		}
		if bonus, err := db.EquippedTimeSpeedBonus(tx, playerID.String()); err == nil {
			tsm += bonus
		}
	}
	effectiveTimeRate := timeRate * tsm
	if effectiveTimeRate <= 0 {
		effectiveTimeRate = 1.0
	}

	// 2. Validate production timer
	lastTime := time.Now().Add(-24 * time.Hour) // default
	if lastCol.Valid {
		lastTime = lastCol.Time
	}

	elapsed := time.Since(lastTime)
	elapsedInternal := time.Duration(float64(elapsed) * effectiveTimeRate)
	requiredHoursInternal := time.Duration(n.ProductionIntervalHours) * time.Hour
	if elapsedInternal < requiredHoursInternal {
		remainingInternal := requiredHoursInternal - elapsedInternal
		remainingReal := time.Duration(float64(remainingInternal) / effectiveTimeRate)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Mỏ đang tích lũy tài nguyên, vui lòng đợi thêm %d phút.", int(remainingReal.Minutes()))})
		return
	}

	// Single collection always harvests 1 production interval
	multiples := 1

	// Retrieve all active nodes in the aperture to evaluate conflicts and food chain
	rows, err := tx.Query(`
		SELECT id, node_type, path, x, y, is_destroyed 
		FROM aperture_resource_nodes 
		WHERE aperture_id = $1 AND is_destroyed = false
	`, n.ApertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải bản đồ sinh thái"})
		return
	}
	var allNodes []models.ApertureResourceNode
	for rows.Next() {
		var other models.ApertureResourceNode
		if err := rows.Scan(&other.ID, &other.NodeType, &other.Path, &other.X, &other.Y, &other.IsDestroyed); err == nil {
			allNodes = append(allNodes, other)
		}
	}
	rows.Close()

	totalReward, suitabilityNote, err := computeApertureNodeReward(tx, playerID, n, multiples, areaSize, allNodes, 1.0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tính sản lượng mỏ"})
		return
	}
	if bonus := equippedTitleUtilityBonus(tx, playerID.String()); bonus > 0 {
		totalReward = int64(math.Floor(float64(totalReward) * (1 + bonus/100)))
	}

	// 3. Grant rewards
	var rewardMsg string
	if n.ProductionType == "Stones" {
		_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, totalReward, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng Nguyên Thạch"})
			return
		}
		rewardMsg = fmt.Sprintf("Thu hoạch được %d Nguyên Thạch!", totalReward)
	} else if n.ProductionType == "Material" {
		prodKey := resolveNodeProdKey(n)
		if prodKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy thông tin vật phẩm sản xuất của mỏ này."})
			return
		}

		if totalReward > 10000000 {
			totalReward = 10000000
		}

		err = InventorySvc.AddItem(tx, playerID.String(), prodKey, "materials", totalReward)
		if err != nil {
			logger.S.Errorf("CollectApertureResource AddItem failed: player=%s, prodKey=%s, totalReward=%d, err=%v", playerID.String(), prodKey, totalReward, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng vật phẩm thu hoạch"})
			return
		}

		// Translate item name for nice display
		translatedName := prodKey
		if itemSpec, exists := models.ItemRegistry[prodKey]; exists {
			translatedName = itemSpec.Name
		}
		rewardMsg = fmt.Sprintf("Thu hoạch được %dx %s!", totalReward, translatedName)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Loại sản phẩm không hợp lệ"})
		return
	}

	rewardMsg += suitabilityNote

	// 4. Update node timer
	_, err = tx.Exec(`
		UPDATE aperture_resource_nodes 
		SET last_collected_at = NOW() 
		WHERE id = $1
	`, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật thời gian mỏ"})
		return
	}
	var rank int
	if err := tx.QueryRow(`SELECT rank FROM players WHERE id=$1`, playerID).Scan(&rank); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tải được cảnh giới"})
		return
	}
	if err := TrackDailyMissionProgress(tx, playerID, rank, "aperture-collect:"+nodeID.String()+":"+lastTime.UTC().Format(time.RFC3339Nano), "aperture_collect", "", 1); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật nhiệm vụ hằng ngày"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save collection"})
		return
	}

	fullRewardMsg := rewardMsg + " Nhân Khí +2."
	notify.Send(db.DB, playerID, "aperture_harvest", "Thu Hoạch Tiên Khiếu", fullRewardMsg)

	resp := gin.H{
		"message": fullRewardMsg,
	}
	if n.ProductionType == "Stones" {
		var stones float64
		_ = db.DB.QueryRow(`SELECT primeval_stones FROM players WHERE id = $1`, playerID).Scan(&stones)
		resp["primeval_stones"] = stones
	} else if n.ProductionType == "Material" {
		resp["materials_inventory"] = loadMaterialsInventory(playerID.String())
	}

	c.JSON(http.StatusOK, resp)
}

// applyBulkReward applies a combined multiplier and an optional bulk discount to a base reward,
// clamping the result to at least 1 whenever the base reward is positive.
func applyBulkReward(baseReward int64, combinedMultiplier float64, bulkMultiplier float64) int64 {
	totalReward := int64(float64(baseReward) * combinedMultiplier * bulkMultiplier)
	if totalReward < 1 && baseReward > 0 {
		totalReward = 1
	}
	return totalReward
}

// computeBulkRepairCost returns the repair cost for a node after applying a bulk fee multiplier
// (rounded up). Destroyed nodes cannot be repaired because they are deleted.
func computeBulkRepairCost(hp, maxHP int, isDestroyed bool, feeMultiplier float64) (baseCost, totalCost int) {
	if isDestroyed {
		return 0, 0
	}
	baseCost = int(math.Ceil(float64(maxHP-hp) / 50.0))
	if baseCost <= 0 {
		return 0, 0
	}
	totalCost = int(math.Ceil(float64(baseCost) * feeMultiplier))
	return baseCost, totalCost
}

func parseItemPaths(pathRaw string) []string {
	pathRaw = strings.TrimSpace(pathRaw)
	if pathRaw == "" {
		return nil
	}
	var paths []string
	if strings.HasPrefix(pathRaw, "[") && strings.HasSuffix(pathRaw, "]") {
		_ = json.Unmarshal([]byte(pathRaw), &paths)
	}
	if len(paths) == 0 {
		paths = []string{pathRaw}
	}
	return paths
}

func normalizePathKey(path string) string {
	p := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(path)), " path")
	p = strings.TrimSuffix(p, "_path")
	switch p {
	case "vận", "vận đạo", "luck":
		return "luck"
	case "mộc", "mộc đạo", "wood":
		return "wood"
	case "hỏa", "hỏa đạo", "fire":
		return "fire"
	case "thủy", "thủy đạo", "water":
		return "water"
	case "kim", "kim đạo", "metal":
		return "metal"
	case "thổ", "thổ đạo", "earth":
		return "earth"
	case "lôi", "lôi đạo", "lightning":
		return "lightning"
	case "băng", "băng đạo", "ice":
		return "ice"
	case "phong", "phong đạo", "wind":
		return "wind"
	case "quang", "quang đạo", "light":
		return "light"
	case "ám", "ám đạo", "dark":
		return "dark"
	case "kiếm", "kiếm đạo", "sword":
		return "sword"
	case "đao", "đao đạo", "blade":
		return "blade"
	case "thực", "thực đạo", "food":
		return "food"
	case "biến hóa", "transformation":
		return "transformation"
	case "nô", "enslavement":
		return "enslavement"
	case "tín", "information":
		return "information"
	case "huyết", "blood":
		return "blood"
	case "độc", "poison":
		return "poison"
	case "nhân", "human":
		return "human"
	case "mộng", "dream":
		return "dream"
	case "huyễn", "phantom":
		return "phantom"
	case "sát", "killing":
		return "killing"
	case "lực", "strength":
		return "strength"
	case "trí", "wisdom":
		return "wisdom"
	default:
		return p
	}
}

func getPathDaoMultiplier(pathRaw string, nodeDaoMap map[string]int) float64 {
	paths := parseItemPaths(pathRaw)
	if len(paths) == 0 {
		return 1.0
	}
	bestMarks := 0
	for _, p := range paths {
		key := normalizePathKey(p)
		if marks, ok := nodeDaoMap[key]; ok && marks > bestMarks {
			bestMarks = marks
		}
	}
	return 1.0 + float64(bestMarks)*0.0001
}

// nodeDaoPurity returns Dao purity (dominant / total) for an aperture's mark map.
// Returns 0 for an empty map, which callers treat as "no penalty".
func nodeDaoPurity(nodeDaoMap map[string]int) float64 {
	var total, dominant int64
	for _, v := range nodeDaoMap {
		if v <= 0 {
			continue
		}
		total += int64(v)
		if int64(v) > dominant {
			dominant = int64(v)
		}
	}
	if total == 0 {
		return 0
	}
	return float64(dominant) / float64(total)
}

type apertureRewardContext struct {
	nodeDaoMap              map[string]int
	daoConflictReductionPct float64
	overgrazing             bool
	herbivores              int
	starvationHerb          bool
	starvationCarn          bool
	variantHumans           map[string]struct {
		population     int
		laborIntensity string
	}
}

func loadApertureRewardContext(tx *sql.Tx, playerID uuid.UUID, apertureID any, allNodes []models.ApertureResourceNode) (*apertureRewardContext, error) {
	ctx := &apertureRewardContext{
		nodeDaoMap: make(map[string]int),
		variantHumans: make(map[string]struct {
			population     int
			laborIntensity string
		}),
	}

	var nodeDaoJSON []byte
	if err := tx.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&nodeDaoJSON); err == nil && len(nodeDaoJSON) > 0 {
		_ = json.Unmarshal(nodeDaoJSON, &ctx.nodeDaoMap)
	}

	var bodyMods models.BodyModsList
	var guInventory models.GuInventory
	var race, constitution string
	if err := tx.QueryRow(`
		SELECT COALESCE(p.body_mods, '[]'::jsonb), COALESCE(p.race, 'human'), COALESCE(p.constitution, ''), `+db.GuInventoryAggregate("p.id")+`
		FROM players p WHERE p.id = $1
	`, playerID).Scan(&bodyMods, &race, &constitution, &guInventory); err == nil {
		player := &models.Player{
			CharacterState: models.CharacterState{Race: race, Constitution: constitution},
			BodyState:      models.BodyState{BodyMods: bodyMods},
			GuInventory:    guInventory,
		}
		ctx.daoConflictReductionPct = gamefx.Get(player).DaoConflictReductionPct
	}

	_, ctx.herbivores, _, ctx.overgrazing, ctx.starvationHerb, ctx.starvationCarn = EvaluateEcosystem(allNodes)

	vhRows, err := tx.Query(`SELECT race, population, labor_intensity FROM aperture_variant_humans WHERE aperture_id = $1`, apertureID)
	if err == nil {
		defer vhRows.Close()
		for vhRows.Next() {
			var race, intensity string
			var pop int
			if err := vhRows.Scan(&race, &pop, &intensity); err == nil {
				ctx.variantHumans[race] = struct {
					population     int
					laborIntensity string
				}{population: pop, laborIntensity: intensity}
			}
		}
	}

	return ctx, nil
}

func computeApertureNodeRewardWithCtx(ctx *apertureRewardContext, n models.ApertureResourceNode, multiples int, areaSize int64, allNodes []models.ApertureResourceNode, bulkMultiplier float64) int64 {
	// 1. Dao Marks multiplier
	daoMultiplier := getPathDaoMultiplier(n.Path, ctx.nodeDaoMap)

	// 2. Conflict multiplier
	conflictMultiplier := 1.0
	if CheckNodeConflict(n, allNodes) {
		conflictPenalty := 0.5 * (1.0 - ctx.daoConflictReductionPct/100.0)
		conflictMultiplier = 1.0 - conflictPenalty
	}

	// 2b. Ecology purity factor (spec §5.1): tạp tu làm cạn kiệt linh khí/khoáng mạch.
	// Dao conflict reduction also attenuates Dao-mark purity loss; the permanent
	// sovereign body sets this to 100%, while ordinary Gu reductions stay below 100%.
	ecologyMultiplier := 1.0
	if p := nodeDaoPurity(ctx.nodeDaoMap); p > 0 {
		purityMultiplier := gamefx.EcologyPurityFactor(p)
		ecologyPenalty := (1.0 - purityMultiplier) * (1.0 - ctx.daoConflictReductionPct/100.0)
		ecologyMultiplier = 1.0 - ecologyPenalty
	}

	// 3. Ecosystem multiplier
	ecoMultiplier := 1.0
	switch n.NodeType {
	case "MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS", "FARMING_ZONE":
		if ctx.overgrazing {
			ecoMultiplier = 0.5
		} else if ctx.herbivores > 0 {
			ecoMultiplier = 1.25
		}
	case "SPIRIT_DEER_PASTURE", "BREEDING_ZONE":
		if ctx.starvationHerb {
			ecoMultiplier = 0.2
		}
	case "SILVER_WOLF_DEN":
		if ctx.starvationCarn {
			ecoMultiplier = 0.2
		}
	}

	// 4. Variant Human labor intensity multiplier
	laborMultiplier := 1.0
	variantBonus := 1.0
	race := getVariantHumanForNode(n.NodeType)
	if race != "" {
		if vh, ok := ctx.variantHumans[race]; ok {
			if vh.population > 0 {
				variantBonus = 1.0 + float64(vh.population)/1000.0
			}
			switch vh.laborIntensity {
			case "LOW":
				laborMultiplier = 0.7
			case "NORMAL":
				laborMultiplier = 1.0
			case "HIGH":
				laborMultiplier = 1.5
			case "EXPLOIT":
				laborMultiplier = 2.0
			}
		}
	}

	suitabilityMultiplier := 1.0

	// 5. Node Level
	nodeLevel := n.Level
	if nodeLevel < 1 {
		nodeLevel = 1
	}
	nodeLevelMultiplier := 1.0 + float64(nodeLevel-1)*0.20

	// 6. Item Rank / Rarity divisor
	var itemFoodPathMultiplier float64 = 1.0
	var itemRankDivisor float64 = 1.0
	if n.ProductionType == "Material" {
		prodKey := resolveNodeProdKey(n)
		if prodKey != "" {
			if item, exists := models.ItemRegistry[prodKey]; exists {
				if item.Path != "" {
					itemFoodPathMultiplier = getPathDaoMultiplier(item.Path, ctx.nodeDaoMap)
				}
				effectiveRank := item.Rank
				if item.Rarity > effectiveRank {
					effectiveRank = item.Rarity
				}
				if effectiveRank >= 1 {
					itemRankDivisor = getManaVal(effectiveRank)
				}
			}
		}
	}

	baseReward := n.ProductionAmount * int64(multiples)
	areaMultiplier := 1.0 + float64(areaSize)/10000000.0
	combinedMultiplier := (areaMultiplier * daoMultiplier * conflictMultiplier * ecologyMultiplier * ecoMultiplier * variantBonus * laborMultiplier * suitabilityMultiplier * itemFoodPathMultiplier * nodeLevelMultiplier) / itemRankDivisor

	totalReward := applyBulkReward(baseReward, combinedMultiplier, bulkMultiplier)
	if n.ProductionType == "Material" && totalReward > 10000000 {
		totalReward = 10000000
	}
	return totalReward
}

// computeApertureNodeReward computes the total reward for harvesting a single resource node,
// applying all multipliers (Dao marks, conflicts, ecosystem, labor, suitability, area size)
// and an optional bulkMultiplier (used by collect-all, e.g. 0.8 for a -20% bulk penalty).
// It returns the reward amount and a suitability warning note (empty when production is optimal).
func computeApertureNodeReward(tx *sql.Tx, playerID uuid.UUID, n models.ApertureResourceNode, multiples int, areaSize int64, allNodes []models.ApertureResourceNode, bulkMultiplier float64) (int64, string, error) {
	ctx, err := loadApertureRewardContext(tx, playerID, n.ApertureID, allNodes)
	if err != nil {
		return 0, "", err
	}
	totalReward := computeApertureNodeRewardWithCtx(ctx, n, multiples, areaSize, allNodes, bulkMultiplier)
	return totalReward, "", nil
}

// CollectAllApertureResources harvests every resource node whose production timer is ready, in a
// single transaction. As a convenience cost, yield is reduced by 20% and Human Qi gain is halved.
func CollectAllApertureResources(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	// 1. Get aperture details
	var apertureID uuid.UUID
	var areaSize int64
	var timeRate float64
	var pTSM float64
	var pRace string
	err = tx.QueryRow(`
		SELECT a.id, a.area_size, COALESCE(a.time_rate, 1.0), p.time_speed_multiplier, COALESCE(p.race, '')
		FROM immortal_apertures a
		JOIN players p ON a.player_id = p.id
		WHERE a.player_id = $1
	`, playerID).Scan(&apertureID, &areaSize, &timeRate, &pTSM, &pRace)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Tiên Khiếu."})
		return
	}
	if timeRate <= 0 {
		timeRate = 1.0
	}
	tsm := 1.0
	dummyP := models.Player{}
	dummyP.Race = pRace
	if !gamefx.Get(&dummyP).Immortal {
		tsm = pTSM
		if tsm <= 0 {
			tsm = 1.0
		}
		if bonus, err := db.EquippedTimeSpeedBonus(tx, playerID.String()); err == nil {
			tsm += bonus
		}
	}
	effectiveTimeRate := timeRate * tsm
	if effectiveTimeRate <= 0 {
		effectiveTimeRate = 1.0
	}

	// 2. Load all active nodes
	rows, err := tx.Query(`
		SELECT id, aperture_id, node_type, name, path, level, hp, max_hp, x, y,
			production_type, production_amount, production_interval_hours, last_collected_at,
			is_destroyed, created_at, updated_at,
			COALESCE(node_category, 'RESOURCE'), COALESCE(production_key, '')
		FROM aperture_resource_nodes
		WHERE aperture_id = $1 AND is_destroyed = false
	`, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải danh sách mỏ"})
		return
	}

	var allNodes []models.ApertureResourceNode
	for rows.Next() {
		var n models.ApertureResourceNode
		var lastCol sql.NullTime
		if err := rows.Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.Path, &n.Level, &n.HP, &n.MaxHP, &n.X, &n.Y,
			&n.ProductionType, &n.ProductionAmount, &n.ProductionIntervalHours, &lastCol,
			&n.IsDestroyed, &n.CreatedAt, &n.UpdatedAt,
			&n.NodeCategory, &n.ProductionKey); err == nil {
			if lastCol.Valid {
				n.LastCollectedAt = &lastCol.Time
			}
			allNodes = append(allNodes, n)
		}
	}
	rows.Close()

	// 3. Determine ready resource nodes and compute rewards
	type collectResult struct {
		NodeID string `json:"node_id"`
		Name   string `json:"name"`
		Reward int64  `json:"reward"`
	}

	var readyCount int
	var totalStones int64
	materialTotals := make(map[string]int64)
	var results []collectResult

	now := time.Now()

	// Pre-load reward context once for all nodes to avoid N+1 queries
	rewardCtx, err := loadApertureRewardContext(tx, playerID, apertureID, allNodes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải dữ liệu Tiên Khiếu"})
		return
	}

	for _, n := range allNodes {
		if n.NodeCategory != "RESOURCE" || n.ProductionAmount <= 0 {
			continue
		}

		intervalHours := n.ProductionIntervalHours
		if intervalHours <= 0 {
			intervalHours = 1
		}
		intervalDuration := time.Duration(float64(intervalHours*3600)/effectiveTimeRate) * time.Second

		var elapsed time.Duration
		if n.LastCollectedAt != nil {
			elapsed = now.Sub(*n.LastCollectedAt)
		} else {
			elapsed = now.Sub(n.CreatedAt)
		}

		if elapsed < intervalDuration {
			continue
		}
		multiples := 1

		// Calculate reward with 0.8 bulk discount (-20%)
		reward := computeApertureNodeRewardWithCtx(rewardCtx, n, multiples, areaSize, allNodes, 0.8)
		if reward <= 0 {
			continue
		}

		if n.ProductionType == "Stones" {
			totalStones += reward
		} else if n.ProductionType == "Material" {
			prodKey := resolveNodeProdKey(n)
			if prodKey != "" {
				rewardToGrant := reward
				if rewardToGrant > 10000000 {
					rewardToGrant = 10000000
				}
				materialTotals[prodKey] += rewardToGrant
			}
		}

		results = append(results, collectResult{NodeID: n.ID, Name: n.Name, Reward: reward})
		readyCount++
	}

	if readyCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có mỏ nào sẵn sàng để thu hoạch."})
		return
	}

	// 4. Grant rewards
	var updatedStones float64
	if totalStones > 0 {
		if err := tx.QueryRow(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2 RETURNING primeval_stones`, totalStones, playerID).Scan(&updatedStones); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng Nguyên Thạch"})
			return
		}
	} else {
		_ = tx.QueryRow(`SELECT primeval_stones FROM players WHERE id = $1`, playerID).Scan(&updatedStones)
	}
	if err := InventorySvc.BulkAddItems(tx, playerID.String(), "materials", materialTotals); err != nil {
		logger.S.Errorf("CollectAllApertureResources BulkAddItems failed: player=%s, err=%v", playerID.String(), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng vật phẩm thu hoạch"})
		return
	}

	// 5. Reset timers in a single batch statement
	harvestedIDs := make([]string, len(results))
	for idx, r := range results {
		harvestedIDs[idx] = r.NodeID
	}
	if len(harvestedIDs) > 0 {
		if _, err := tx.Exec(`UPDATE aperture_resource_nodes SET last_collected_at = NOW() WHERE id = ANY($1::uuid[])`, pq.Array(harvestedIDs)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật thời gian mỏ"})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save collection"})
		return
	}

	var harvestDetails []string
	if totalStones > 0 {
		harvestDetails = append(harvestDetails, fmt.Sprintf("%d Nguyên Thạch", totalStones))
	}
	for prodKey, amount := range materialTotals {
		itemName := prodKey
		if itemSpec, exists := models.ItemRegistry[prodKey]; exists {
			itemName = itemSpec.Name
		}
		harvestDetails = append(harvestDetails, fmt.Sprintf("%dx %s", amount, itemName))
	}

	detailStr := ""
	if len(harvestDetails) > 0 {
		detailStr = " Nhận được: " + strings.Join(harvestDetails, ", ") + "."
	}

	bulkMsg := fmt.Sprintf("Thu hoạch hàng loạt %d mỏ thành công!%s Nhân Khí +%d (Sản lượng -20%%).", readyCount, detailStr, readyCount)
	notify.Send(db.DB, playerID, "aperture_harvest_bulk", "Thu Hoạch Hàng Loạt Tiên Khiếu", bulkMsg)

	c.JSON(http.StatusOK, gin.H{
		"message":             bulkMsg,
		"count":               readyCount,
		"total_stones":        totalStones,
		"primeval_stones":     updatedStones,
		"materials":           materialTotals,
		"materials_inventory": loadMaterialsInventory(playerID.String()),
		"collected":           results,
	})
}

// RepairAllApertureNodes repairs every damaged node in the player's aperture at
// once. As a convenience cost, each node's repair fee is increased by 30% (rounded up).
func RepairAllApertureNodes(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch aperture
	var apertureID string
	err = tx.QueryRow(`SELECT id FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apertureID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra Tiên Khiếu"})
		return
	}

	// 2. Load damaged nodes. Fully destroyed nodes are deleted and cannot be rebuilt.
	rows, err := tx.Query(`
		SELECT id, name, hp, max_hp
		FROM aperture_resource_nodes
		WHERE aperture_id = $1 AND is_destroyed = false AND hp < max_hp
		FOR UPDATE
	`, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải danh sách công trình"})
		return
	}

	type repairResult struct {
		NodeID string `json:"node_id"`
		Name   string `json:"name"`
		Cost   int    `json:"cost"`
	}
	var results []repairResult
	totalCost := 0

	for rows.Next() {
		var id, name string
		var hp, maxHP int
		if err := rows.Scan(&id, &name, &hp, &maxHP); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc dữ liệu công trình"})
			return
		}
		_, cost := computeBulkRepairCost(hp, maxHP, false, 1.3)
		if cost <= 0 {
			continue
		}
		totalCost += cost
		results = append(results, repairResult{NodeID: id, Name: name, Cost: cost})
	}
	rows.Close()

	if len(results) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có công trình nào cần sửa chữa."})
		return
	}

	// 3. Check player stones
	var stones int
	err = tx.QueryRow(`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1`, playerID).Scan(&stones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
		return
	}

	if stones < totalCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Nguyên Thạch không đủ để sửa chữa toàn bộ (Yêu cầu: %d, Hiện có: %d).", totalCost, stones)})
		return
	}

	// 4. Deduct + repair
	var newStones float64
	if err := tx.QueryRow(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2 RETURNING immortal_stones`, totalCost, playerID).Scan(&newStones); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên Thạch"})
		return
	}

	nodeIDs := make([]string, len(results))
	for idx, r := range results {
		nodeIDs[idx] = r.NodeID
	}
	if len(nodeIDs) > 0 {
		res, err := tx.Exec(`
			UPDATE aperture_resource_nodes
			SET hp = max_hp, updated_at = NOW()
			WHERE id = ANY($1::uuid[]) AND is_destroyed = false
		`, pq.Array(nodeIDs))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái sửa chữa"})
			return
		}
		if repaired, rowsErr := res.RowsAffected(); rowsErr != nil || repaired != int64(len(nodeIDs)) {
			c.JSON(http.StatusConflict, gin.H{"error": "Một công trình đã bị phá hủy trong lúc sửa chữa; giao dịch đã được hoàn lại"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save repair"})
		return
	}

	repairAllMsg := fmt.Sprintf("Sửa chữa hàng loạt %d công trình thành công! Tiêu tốn tổng cộng %d Tiên Nguyên Thạch (Phí +30%%).", len(results), totalCost)
	notify.Send(db.DB, playerID, "aperture_repair_bulk", "Sửa Chữa Hàng Loạt Tiên Khiếu", repairAllMsg)

	c.JSON(http.StatusOK, gin.H{
		"message":         repairAllMsg,
		"repaired_count":  len(results),
		"total_cost":      totalCost,
		"nodes":           results,
		"immortal_stones": newStones,
	})
}
