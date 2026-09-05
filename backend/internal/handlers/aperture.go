package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type ResourceNodeTemplate struct {
	NodeType          string
	Name              string
	Path              string
	CostStones        int
	CostImmortal      int
	ProdType          string // "Material" | "Stones"
	ProdKey           string // e.g. "moon_orchid_petal"
	ProdAmount        int
	ProdIntervalHours int
	NodeCategory      string   // "RESOURCE" | "DEFENSE"
	HPMax             int      // max HP for node (default 1000)
	DefensePower      int      // damage per tick (WATCHTOWER)
	AttackRange       int      // radius in cells (Manhattan distance)
	HPBonus           int      // bonus HP to nearby nodes (GUARDIAN_STATUE)
	DamageReduction   int      // % damage reduction (BARRACKS, FORCE_FIELD)
	BlocksMovement    bool     // wall blocks enemy movement
	CostGu            []string // required Gu IDs
	CostGuRank        int      // minimum rank for each required Gu
	FoodPath          string   // path of the food item produced (if any)
	ItemTier          string   // tier of the item (T1-T9)
	ZoneType          string   // zone type (khu_trong, khu_nuoi, khu_khai_thac, khu_che_bien, khu_phong_thu, khu_thuc_an)
}

var ResourceNodeTemplates = map[string]ResourceNodeTemplate{
	// ---- GENERIC CUSTOMIZABLE ZONES ----
	"FARMING_ZONE": {
		NodeType:          "FARMING_ZONE",
		Name:              "Khu Trồng Trọt",
		Path:              "Wood Path",
		CostStones:        10,
		ProdType:          "Material",
		ProdAmount:        10,
		ProdIntervalHours: 10,
		ZoneType:          "khu_trong",
	},
	"BREEDING_ZONE": {
		NodeType:          "BREEDING_ZONE",
		Name:              "Khu Chăn Nuôi",
		Path:              "Food Path",
		CostStones:        24,
		ProdType:          "Material",
		ProdAmount:        5,
		ProdIntervalHours: 10,
		ZoneType:          "khu_nuoi",
	},
	"MINING_ZONE": {
		NodeType:          "MINING_ZONE",
		Name:              "Khu Khai Thác",
		Path:              "Earth Path",
		CostStones:        20,
		ProdType:          "Material",
		ProdAmount:        3,
		ProdIntervalHours: 10,
		ZoneType:          "khu_khai_thac",
	},
	// ---- DEFENSE STRUCTURES ----
	"WALL": {
		NodeType:       "WALL",
		Name:           "Tường Thành Tiên Khiếu",
		Path:           "Earth Path",
		CostStones:     16,
		NodeCategory:   "DEFENSE",
		HPMax:          3000,
		BlocksMovement: true,
		CostGu:         []string{"GU_BEAST_SKIN", "GU_STONE_SKIN"},
		CostGuRank:     1,
		ZoneType:       "khu_phong_thu",
	},
	"WATCHTOWER": {
		NodeType:     "WATCHTOWER",
		Name:         "Tháp Canh Tiên Khiếu",
		Path:         "Metal Path",
		CostStones:   24,
		NodeCategory: "DEFENSE",
		HPMax:        2000,
		DefensePower: 5,
		AttackRange:  3,
		CostGu:       []string{"GU_IRON_SKIN", "GU_HALF_STONE"},
		CostGuRank:   2,
		ZoneType:     "khu_phong_thu",
	},
	"GUARDIAN_STATUE": {
		NodeType:     "GUARDIAN_STATUE",
		Name:         "Tượng Hộ Mệnh Tiên Khiếu",
		Path:         "Earth Path",
		CostStones:   30,
		NodeCategory: "DEFENSE",
		HPMax:        2500,
		HPBonus:      500,
		AttackRange:  2,
		CostGu:       []string{"GU_WHITE_JADE", "GU_MOON_RAIMENT"},
		CostGuRank:   2,
		ZoneType:     "khu_phong_thu",
	},
	"BARRACKS": {
		NodeType:        "BARRACKS",
		Name:            "Doanh Trại Tiên Khiếu",
		Path:            "Human Path",
		CostStones:      40,
		NodeCategory:    "DEFENSE",
		HPMax:           4000,
		DamageReduction: 15,
		AttackRange:     0, // 0 = affects all nodes in aperture
		CostGu:          []string{"GU_GOLDEN_BELL_SHIELD", "GU_IVORY_ARMOR"},
		CostGuRank:      4,
		ZoneType:        "khu_phong_thu",
	},
	"FORCE_FIELD": {
		NodeType:        "FORCE_FIELD",
		Name:            "Trường Lực Tiên Khiếu",
		Path:            "Metal Path",
		CostStones:      50,
		NodeCategory:    "DEFENSE",
		HPMax:           1500,
		DamageReduction: 30,
		AttackRange:     2,
		CostGu:          []string{"GU_SUPREME_SHIELD", "GU_GOLDEN_BELL_SHIELD"},
		CostGuRank:      4,
		ZoneType:        "khu_phong_thu",
	},
}

func resolveNodeProdKey(n models.ApertureResourceNode) string {
	if n.ProductionKey != "" {
		return n.ProductionKey
	}
	if template, ok := ResourceNodeTemplates[n.NodeType]; ok && template.ProdKey != "" {
		return template.ProdKey
	}
	if strings.Contains(n.Name, "(") && strings.Contains(n.Name, ")") {
		start := strings.Index(n.Name, "(")
		end := strings.Index(n.Name, ")")
		if start < end {
			itemName := strings.TrimSpace(n.Name[start+1 : end])
			for id, spec := range models.ItemRegistry {
				if strings.EqualFold(spec.Name, itemName) {
					return id
				}
			}
		}
	}
	return ""
}

func getVariantHumanForNode(nodeType string) string {
	switch nodeType {
	case "HAIRY_MAN_SETTLEMENT":
		return "HAIRY_MAN"
	case "ROCKMAN_TRIBE", "JADE_STONE_MINE", "PRIMEVAL_MINE", "MINING_ZONE":
		return "ROCKMAN"
	case "FEATHERMAN_CANYON":
		return "FEATHERMAN"
	default:
		return ""
	}
}

// GetImmortalAperture returns the player's immortal aperture, creating one if not exists (Rank >= 6)
func GetImmortalAperture(c *gin.Context) {
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

	// 1. Fetch player
	var player models.Player
	err = tx.QueryRow(`
		SELECT id, rank, aptitude_percent, human_qi, primeval_stones, immortal_stones, current_mana, time_speed_multiplier
		FROM players WHERE id = $1
	`, playerID).Scan(&player.ID, &player.Rank, &player.AptitudePercent, &player.HumanQi, &player.PrimevalStones, &player.ImmortalStones, &player.CurrentMana, &player.TimeSpeedMultiplier)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cổ sư không tồn tại"})
		return
	}

	if player.Rank < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa thăng tiên (Rank 6), chưa có Tiên Khiếu."})
		return
	}

	// 2. Fetch aperture
	var ap models.ImmortalAperture
	var createdAt, updatedAt time.Time
	err = tx.QueryRow(`
		SELECT id, player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, tribulation_count, rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, state, dao_marks, created_at, updated_at
		FROM immortal_apertures WHERE player_id = $1
	`, playerID).Scan(&ap.ID, &ap.PlayerID, &ap.Grade, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.AreaSize, &ap.ImmortalEssence, &ap.EssenceProductionPerYear, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.TribulationCount, &ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.State, &ap.DaoMarks, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		// Auto-create aperture (fallback for legacy players) — grade rolled from current Nhân Khí
		ap, err = createAperture(tx, playerID.String(), player.HumanQi, player.CurrentMana)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khai mở Tiên Khiếu: " + err.Error()})
			return
		}
		createdAt = ap.CreatedAt
		updatedAt = ap.UpdatedAt
	} else if err != nil {
		logger.S.Errorf("Error fetching aperture: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn Tiên Khiếu"})
		return
	} else {
		// Sync current mana to database
		ap.ImmortalEssence = int(player.CurrentMana)
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE id = $2`, ap.ImmortalEssence, ap.ID)
	}

	// Verify state consistency: aperture can ONLY be STABLE if there are no active tribulation events
	// and no living tribulation enemies.
	hasActiveTrib, errActive := db.HasActiveTribulationOrEnemies(tx, ap.ID)
	if errActive == nil {
		if hasActiveTrib && ap.State == "STABLE" {
			ap.State = "TRIBULATION_ACTIVE"
			_, _ = tx.Exec(`UPDATE immortal_apertures SET state = 'TRIBULATION_ACTIVE' WHERE id = $1`, ap.ID)
		} else if !hasActiveTrib && ap.State == "TRIBULATION_ACTIVE" && ap.ApertureHP >= ap.MaxApertureHP {
			ap.State = "STABLE"
			_, _ = tx.Exec(`UPDATE immortal_apertures SET state = 'STABLE' WHERE id = $1`, ap.ID)
		}
	}

	ap.CreatedAt = createdAt
	ap.UpdatedAt = updatedAt

	// Project internal_years_passed forward based on real time elapsed since updated_at.
	// This ensures the frontend always gets accurate time values even if other handlers
	// (e.g. SetApertureTimeRate) update updated_at without projecting internal years.
	if ap.State == "STABLE" || ap.State == "DAMAGED" {
		now := time.Now()
		elapsedSec := now.Sub(updatedAt).Seconds()
		if elapsedSec > 0 && ap.TimeRate > 0 {
			tsm := player.TimeSpeedMultiplier
			if tsm <= 0 {
				tsm = 1.0
			}
			if bonus, err := db.EquippedTimeSpeedBonus(tx, playerID.String()); err == nil {
				tsm += bonus
			}
			elapsedInternalYears := elapsedSec * ap.TimeRate * tsm / 36000.0
			ap.InternalYearsPassed += elapsedInternalYears
			ap.UpdatedAt = now
			_, _ = tx.Exec(`
				UPDATE immortal_apertures
				SET internal_years_passed = $1, updated_at = $2
				WHERE id = $3
			`, ap.InternalYearsPassed, now, ap.ID)
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failure"})
		return
	}

	c.JSON(http.StatusOK, ap)
}

func seedInitialApertureNodes(tx *sql.Tx, apertureID string, grade string) {
	type StarterConfig struct {
		NodeType     string
		Name         string
		Path         string
		ProdKey      string
		ZoneType     string
		ProdType     string
		ProdAmount   int
		ProdInterval int
	}

	starterConfigs := []StarterConfig{
		{NodeType: "FARMING_ZONE", Name: "Khu Trồng Trọt (Rượu Thanh Trúc)", Path: "Wood Path", ProdKey: "green_bamboo_wine", ZoneType: "khu_trong", ProdType: "Material", ProdAmount: 10, ProdInterval: 10},
		{NodeType: "BREEDING_ZONE", Name: "Khu Chăn Nuôi (Thịt Hươu Linh)", Path: "Food Path", ProdKey: "spirit_deer_meat", ZoneType: "khu_nuoi", ProdType: "Material", ProdAmount: 5, ProdInterval: 10},
		{NodeType: "MINING_ZONE", Name: "Khu Khai Thác (Ngọc Thạch)", Path: "Earth Path", ProdKey: "jade_stone", ZoneType: "khu_khai_thac", ProdType: "Material", ProdAmount: 3, ProdInterval: 10},
	}

	count := 2
	switch grade {
	case "MID":
		count = 2
	case "HIGH", "EXTREME":
		count = 3
	}

	used := make(map[string]bool)
	for i := 0; i < count && i < len(starterConfigs); i++ {
		cfg := starterConfigs[i]

		x, y := rand.Intn(20), rand.Intn(20)
		key := fmt.Sprintf("%d,%d", x, y)
		for used[key] {
			x, y = rand.Intn(20), rand.Intn(20)
			key = fmt.Sprintf("%d,%d", x, y)
		}
		used[key] = true

		_, _ = tx.Exec(`
			INSERT INTO aperture_resource_nodes
				(aperture_id, node_type, name, path, level, hp, max_hp, x, y,
				 production_type, production_amount, production_interval_hours, last_collected_at,
				 node_category, defense_power, attack_range, hp_bonus, damage_reduction, blocks_movement,
				 required_gu_id, required_gu_rank, zone_type, food_path, item_tier, production_key)
			VALUES ($1, $2, $3, $4, 1, 1000, 1000, $5, $6, $7, $8, $9, NOW(),
				'RESOURCE', 0, 0, 0, 0, false,
				'', 0, $10, '', 'T1', $11)
		`, apertureID, cfg.NodeType, cfg.Name, cfg.Path, x, y,
			cfg.ProdType, cfg.ProdAmount, cfg.ProdInterval,
			cfg.ZoneType, cfg.ProdKey)
	}
}

// rollApertureGrade rolls the Tiên Khiếu grade based on Nhân Khí ratio q (0..1).
// Higher Nhân Khí increases the chance of higher grades (weights sum to a constant 21).
func rollApertureGrade(q float64) string {
	wExtreme := 1.0 + 9.0*q
	wHigh := 3.0 + 3.0*q
	wMid := 5.0
	roll := rand.Float64() * 21.0
	switch {
	case roll < wExtreme:
		return "EXTREME"
	case roll < wExtreme+wHigh:
		return "HIGH"
	case roll < wExtreme+wHigh+wMid:
		return "MID"
	default:
		return "LOW"
	}
}

// createAperture inserts a brand new Tiên Khiếu for a newly ascended player,
// rolls its grade from the Nhân Khí they had at ascension, and seeds starter nodes.
func createAperture(tx *sql.Tx, playerID string, humanQi, currentMana float64) (models.ImmortalAperture, error) {
	ap := buildApertureByGrade(playerID, rollApertureGrade(humanQi/MaxHumanQi))
	ap.ImmortalEssence = int(currentMana)
	err := tx.QueryRow(`
		INSERT INTO immortal_apertures (player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, state)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`, ap.PlayerID, ap.Grade, ap.TimeRate, ap.ApertureHP, ap.MaxApertureHP, ap.AreaSize, ap.ImmortalEssence, ap.EssenceProductionPerYear, ap.InternalYearsPassed, ap.NextTribulationYear, ap.State).Scan(&ap.ID, &ap.CreatedAt, &ap.UpdatedAt)
	if err != nil {
		return models.ImmortalAperture{}, err
	}
	// Seed a few random initial resource nodes for the new aperture
	seedInitialApertureNodes(tx, ap.ID, ap.Grade)
	return ap, nil
}

func buildApertureByGrade(playerID string, grade string) models.ImmortalAperture {
	switch grade {
	case "EXTREME":
		timeRate := 40.0 + float64(rand.Intn(16)) // 40 to 55
		return models.ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 timeRate,
			ApertureHP:               50000,
			MaxApertureHP:            50000,
			AreaSize:                 10000000,
			EssenceProductionPerYear: 50,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	case "HIGH":
		timeRate := 30.0 + float64(rand.Intn(11)) // 30 to 40
		return models.ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 timeRate,
			ApertureHP:               30000,
			MaxApertureHP:            30000,
			AreaSize:                 9000000,
			EssenceProductionPerYear: 30,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	case "MID":
		timeRate := 20.0 + float64(rand.Intn(9)) // 20 to 28
		return models.ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 timeRate,
			ApertureHP:               20000,
			MaxApertureHP:            20000,
			AreaSize:                 6000000,
			EssenceProductionPerYear: 20,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	default: // "LOW"
		timeRate := 10.0 + float64(rand.Intn(6)) // 10 to 15
		return models.ImmortalAperture{
			PlayerID:                 playerID,
			Grade:                    grade,
			TimeRate:                 timeRate,
			ApertureHP:               10000,
			MaxApertureHP:            10000,
			AreaSize:                 3000000,
			EssenceProductionPerYear: 10,
			NextTribulationYear:      10,
			State:                    "STABLE",
		}
	}
}

// GetDaoMarks returns player's Dao Marks
func GetDaoMarks(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var daoJSON []byte
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&daoJSON)
	if err == sql.ErrNoRows {
		daoJSON = []byte("{}")
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load Dao Marks"})
		return
	}

	daoMap := make(map[string]int)
	if len(daoJSON) > 0 {
		_ = json.Unmarshal(daoJSON, &daoMap)
	}

	marks := make([]models.DaoMark, 0, len(daoMap))
	for path, amount := range daoMap {
		if amount <= 0 {
			continue
		}
		marks = append(marks, models.DaoMark{
			ID:       playerID.String() + ":" + path,
			PlayerID: playerID.String(),
			Path:     path,
			Amount:   amount,
		})
	}
	sort.Slice(marks, func(i, j int) bool {
		return marks[i].Amount > marks[j].Amount
	})

	c.JSON(http.StatusOK, marks)
}

// GetApertureNodes returns resource nodes in player's aperture
func GetApertureNodes(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.path, n.level, n.hp, n.max_hp, n.x, n.y,
			n.production_type, n.production_amount, n.production_interval_hours, n.last_collected_at,
			n.is_destroyed, n.created_at, n.updated_at,
			COALESCE(n.node_category, 'RESOURCE'), COALESCE(n.defense_power, 0), COALESCE(n.attack_range, 0),
			COALESCE(n.hp_bonus, 0), COALESCE(n.damage_reduction, 0), COALESCE(n.blocks_movement, false),
			COALESCE(n.required_gu_id, ''), COALESCE(n.required_gu_rank, 0),
			COALESCE(n.zone_type, ''), COALESCE(n.food_path, ''), COALESCE(n.item_tier, '')
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE a.player_id = $1 AND n.is_destroyed = false
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy danh sách điểm tài nguyên: " + err.Error()})
		return
	}
	defer rows.Close()

	nodes := []models.ApertureResourceNode{}
	for rows.Next() {
		var n models.ApertureResourceNode
		var lastCol sql.NullTime
		err = rows.Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.Path, &n.Level, &n.HP, &n.MaxHP, &n.X, &n.Y,
			&n.ProductionType, &n.ProductionAmount, &n.ProductionIntervalHours, &lastCol,
			&n.IsDestroyed, &n.CreatedAt, &n.UpdatedAt,
			&n.NodeCategory, &n.DefensePower, &n.AttackRange,
			&n.HPBonus, &n.DamageReduction, &n.BlocksMovement,
			&n.RequiredGuID, &n.RequiredGuRank,
			&n.ZoneType, &n.FoodPath, &n.ItemTier)
		if err == nil {
			if lastCol.Valid {
				n.LastCollectedAt = &lastCol.Time
			}
			nodes = append(nodes, n)
		}
	}

	c.JSON(http.StatusOK, nodes)
}

type PlaceNodeRequest struct {
	NodeType string `json:"node_type" binding:"required"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Path     string `json:"path"`
	ProdKey  string `json:"prod_key"`
}

// PlaceApertureNode constructs a new resource node on the grid
func PlaceApertureNode(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req PlaceNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	if req.X < 0 || req.X >= 20 || req.Y < 0 || req.Y >= 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ nằm ngoài ranh giới Tiên Khiếu (0-19)."})
		return
	}

	template, valid := ResourceNodeTemplates[req.NodeType]
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại tài nguyên xây dựng không tồn tại"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch player & aperture
	var ap models.ImmortalAperture
	err = tx.QueryRow(`
		SELECT id, state, area_size FROM immortal_apertures WHERE player_id = $1
	`, playerID).Scan(&ap.ID, &ap.State, &ap.AreaSize)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra Tiên Khiếu"})
		return
	}

	if ap.State == "TRIBULATION_ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Khiếu đang có tai kiếp càn quét, không thể xây dựng!"})
		return
	}

	// Limit maximum resource nodes count based on area_size
	var currentNodesCount int
	err = tx.QueryRow(`
		SELECT COUNT(1) FROM aperture_resource_nodes 
		WHERE aperture_id = $1 AND is_destroyed = false
	`, ap.ID).Scan(&currentNodesCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra số lượng mỏ hiện tại"})
		return
	}

	maxNodes := int(ap.AreaSize / 100000)
	if maxNodes > 200 {
		maxNodes = 200
	}
	if currentNodesCount >= maxNodes {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Quy mô Tiên Khiếu không đủ rộng lớn. Số lượng điểm tài nguyên tối đa là %d mỏ (Hiện tại: %d).", maxNodes, currentNodesCount)})
		return
	}

	var immortalStones int64
	err = tx.QueryRow(`
		SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1 FOR UPDATE
	`, playerID).Scan(&immortalStones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
		return
	}

	// 2. Check collision
	var collision bool
	err = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM aperture_resource_nodes 
			WHERE aperture_id = $1 AND x = $2 AND y = $3 AND is_destroyed = false
		)
	`, ap.ID, req.X, req.Y).Scan(&collision)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra trùng tọa độ"})
		return
	}

	if collision {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ này đã có điểm tài nguyên khác xây dựng rồi."})
		return
	}

	// 2b. Limit unique defense structures (Barracks: max 1)
	if req.NodeType == "BARRACKS" {
		var barrackCount int
		err = tx.QueryRow(`
			SELECT COUNT(1) FROM aperture_resource_nodes
			WHERE aperture_id = $1 AND node_type = 'BARRACKS' AND is_destroyed = false
		`, ap.ID).Scan(&barrackCount)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra Doanh Trại"})
			return
		}
		if barrackCount >= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mỗi Tiên Khiếu chỉ được xây dựng tối đa 1 Doanh Trại."})
			return
		}
	}

	// 3. Deduct cost or item
	nodeCategory := template.NodeCategory
	if nodeCategory == "" {
		nodeCategory = "RESOURCE"
	}
	nodeHPMax := template.HPMax
	if nodeHPMax <= 0 {
		nodeHPMax = 1000
	}

	nodeItemID := "node_" + strings.ToLower(req.NodeType)
	qty, err := InventorySvc.GetQuantity(tx, playerID.String(), nodeItemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn túi đồ"})
		return
	}
	hasItem := qty > 0

	if hasItem {
		if err := InventorySvc.RemoveItem(tx, playerID.String(), nodeItemID, "materials", 1); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ vật phẩm cơ sở"})
			return
		}
	} else {
		if immortalStones < int64(template.CostStones) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Nguyên Thạch không đủ để xây dựng mỏ này (Yêu cầu: %d, Hiện có: %d).", template.CostStones, immortalStones)})
			return
		}
		_, err = tx.Exec(`
			UPDATE players 
			SET immortal_stones = immortal_stones - $1
			WHERE id = $2
		`, template.CostStones, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ chi phí"})
			return
		}
	}

	// 3b. Consume required Gu items for defense structures
	if len(template.CostGu) > 0 {
		for _, requiredGuID := range template.CostGu {
			var guInstanceID string
			err = tx.QueryRow(`
				SELECT id FROM player_gu 
				WHERE player_id = $1 AND gu_id = $2 AND gu_rank >= $3 AND state = 'ACTIVE'
				LIMIT 1
			`, playerID, requiredGuID, template.CostGuRank).Scan(&guInstanceID)
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần sở hữu Cổ trùng %s (rank %d+) để xây dựng %s", requiredGuID, template.CostGuRank, template.Name)})
				return
			} else if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn cổ trùng"})
				return
			}
			// Consume the Gu (destroy it)
			_, err = tx.Exec(`DELETE FROM player_gu WHERE id = $1`, guInstanceID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tiêu hao cổ trùng"})
				return
			}
		}
	}

	nodePath := template.Path
	if req.Path != "" {
		nodePath = req.Path
	}

	nodeProdKey := template.ProdKey
	if req.ProdKey != "" {
		nodeProdKey = req.ProdKey
	}

	nodeName := template.Name
	if nodeProdKey != "" {
		if itemSpec, exists := models.ItemRegistry[nodeProdKey]; exists {
			nodeName = template.Name + " (" + itemSpec.Name + ")"
		}
	}

	// 4. Insert node
	var node models.ApertureResourceNode
	var createdAt time.Time
	err = tx.QueryRow(`
		INSERT INTO aperture_resource_nodes (aperture_id, node_type, name, path, level, hp, max_hp, x, y,
			production_type, production_amount, production_interval_hours, last_collected_at,
			node_category, defense_power, attack_range, hp_bonus, damage_reduction, blocks_movement,
			required_gu_id, required_gu_rank, zone_type, food_path, item_tier, production_key)
		VALUES ($1, $2, $3, $4, 1, $5, $5, $6, $7, $8, $9, $10, NOW(),
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
		RETURNING id, created_at
	`, ap.ID, template.NodeType, nodeName, nodePath, nodeHPMax, req.X, req.Y,
		template.ProdType, template.ProdAmount, template.ProdIntervalHours,
		nodeCategory, template.DefensePower, template.AttackRange, template.HPBonus,
		template.DamageReduction, template.BlocksMovement,
		strings.Join(template.CostGu, ","), template.CostGuRank,
		template.ZoneType, template.FoodPath, template.ItemTier, nodeProdKey).Scan(&node.ID, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu điểm tài nguyên: " + err.Error()})
		return
	}

	err = updateVariantHumanOnNodeChange(tx, ap.ID, template.NodeType, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật số lượng dị nhân: " + err.Error()})
		return
	}

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 5, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit"})
		return
	}

	node.ApertureID = ap.ID
	node.NodeType = template.NodeType
	node.Name = template.Name
	node.Path = template.Path
	node.Level = 1
	node.HP = nodeHPMax
	node.MaxHP = nodeHPMax
	node.X = req.X
	node.Y = req.Y
	node.ProductionType = template.ProdType
	node.ProductionAmount = template.ProdAmount
	node.ProductionIntervalHours = template.ProdIntervalHours
	node.NodeCategory = nodeCategory
	node.DefensePower = template.DefensePower
	node.AttackRange = template.AttackRange
	node.HPBonus = template.HPBonus
	node.DamageReduction = template.DamageReduction
	node.BlocksMovement = template.BlocksMovement
	node.RequiredGuID = strings.Join(template.CostGu, ",")
	node.RequiredGuRank = template.CostGuRank
	node.ZoneType = template.ZoneType
	node.FoodPath = template.FoodPath
	node.ItemTier = template.ItemTier
	node.CreatedAt = createdAt
	node.UpdatedAt = createdAt

	buildMsg := fmt.Sprintf("Xây dựng thành công %s!", template.Name)
	notify.Send(db.DB, playerID, "aperture_build", "Xây Dựng Tiên Khiếu", buildMsg)

	c.JSON(http.StatusOK, gin.H{
		"node":    node,
		"message": buildMsg,
	})
}

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
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.production_type, n.production_amount, n.production_interval_hours, n.last_collected_at, n.is_destroyed, a.area_size, a.time_rate, n.path, n.x, n.y, COALESCE(n.production_key, ''), COALESCE(n.level, 1)
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.ProductionType, &n.ProductionAmount, &n.ProductionIntervalHours, &lastCol, &n.IsDestroyed, &areaSize, &timeRate, &n.Path, &n.X, &n.Y, &n.ProductionKey, &n.Level)

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

	// 2. Validate production timer
	lastTime := time.Now().Add(-24 * time.Hour) // default
	if lastCol.Valid {
		lastTime = lastCol.Time
	}

	elapsed := time.Since(lastTime)
	elapsedInternal := time.Duration(float64(elapsed) * timeRate)
	requiredHoursInternal := time.Duration(n.ProductionIntervalHours) * time.Hour
	if elapsedInternal < requiredHoursInternal {
		remainingInternal := requiredHoursInternal - elapsedInternal
		remainingReal := time.Duration(float64(remainingInternal) / timeRate)
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Mỏ đang tích lũy tài nguyên, vui lòng đợi thêm %d phút.", int(remainingReal.Minutes()))})
		return
	}

	// Calculate multiples of production if player waited longer (max 3x stock)
	multiples := int(elapsedInternal / requiredHoursInternal)
	if multiples > 3 {
		multiples = 3
	}

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

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 2, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save collection"})
		return
	}

	fullRewardMsg := rewardMsg + " Nhân Khí +2."
	notify.Send(db.DB, playerID, "aperture_harvest", "Thu Hoạch Tiên Khiếu", fullRewardMsg)

	c.JSON(http.StatusOK, gin.H{
		"message": fullRewardMsg,
	})
}

// applyBulkReward applies a combined multiplier and an optional bulk discount to a base reward,
// clamping the result to at least 1 whenever the base reward is positive.
func applyBulkReward(baseReward int, combinedMultiplier float64, bulkMultiplier float64) int {
	totalReward := int(float64(baseReward) * combinedMultiplier * bulkMultiplier)
	if totalReward < 1 && baseReward > 0 {
		totalReward = 1
	}
	return totalReward
}

// computeBulkRepairCost returns the repair cost for a node after applying a bulk fee multiplier
// (rounded up). Destroyed nodes cost max_hp*2; damaged nodes cost max_hp - hp.
func computeBulkRepairCost(hp, maxHP int, isDestroyed bool, feeMultiplier float64) (baseCost, totalCost int) {
	if isDestroyed {
		baseCost = int(math.Ceil(float64(maxHP*2) / 50.0))
	} else {
		baseCost = int(math.Ceil(float64(maxHP-hp) / 50.0))
	}
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
	return 1.0 + float64(bestMarks)*0.01
}

// computeApertureNodeReward computes the total reward for harvesting a single resource node,
// applying all multipliers (Dao marks, conflicts, ecosystem, labor, suitability, area size)
// and an optional bulkMultiplier (used by collect-all, e.g. 0.8 for a -20% bulk penalty).
// It returns the reward amount and a suitability warning note (empty when production is optimal).
func computeApertureNodeReward(tx *sql.Tx, playerID uuid.UUID, n models.ApertureResourceNode, multiples int, areaSize int64, allNodes []models.ApertureResourceNode, bulkMultiplier float64) (int, string, error) {
	// 1. Dao Marks multiplier
	var nodeDaoJSON []byte
	if err := tx.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&nodeDaoJSON); err != nil {
		return 0, "", err
	}
	nodeDaoMap := make(map[string]int)
	if len(nodeDaoJSON) > 0 {
		_ = json.Unmarshal(nodeDaoJSON, &nodeDaoMap)
	}
	daoMultiplier := getPathDaoMultiplier(n.Path, nodeDaoMap)

	// Check if player has Tiên Thai Chí Tôn (ignore Dao conflicts)
	hasSovereignFetus := false
	var guJSON []byte
	errGu := tx.QueryRow(`SELECT `+db.GuInventoryAggregate("players.id")+` FROM players WHERE id = $1`, playerID).Scan(&guJSON)
	if errGu == nil {
		var guInv models.GuInventory
		if json.Unmarshal(guJSON, &guInv) == nil {
			for _, gu := range guInv {
				if gu.GuID == "GU_SOVEREIGN_FETUS" && gu.Equipped && gu.Status == "ACTIVE" {
					hasSovereignFetus = true
					break
				}
			}
		}
	}

	// 2. Conflict multiplier
	conflictMultiplier := 1.0
	if CheckNodeConflict(n, allNodes) && !hasSovereignFetus {
		conflictMultiplier = 0.5
	}

	// 3. Ecosystem / Food Chain multiplier
	_, herbivores, _, overgrazing, starvationHerb, starvationCarn := EvaluateEcosystem(allNodes)
	ecoMultiplier := 1.0
	switch n.NodeType {
	case "MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS", "FARMING_ZONE":
		if overgrazing {
			ecoMultiplier = 0.5
		} else if herbivores > 0 {
			ecoMultiplier = 1.25 // Fertilization bonus
		}
	case "SPIRIT_DEER_PASTURE", "BREEDING_ZONE":
		if starvationHerb {
			ecoMultiplier = 0.2
		}
	case "SILVER_WOLF_DEN":
		if starvationCarn {
			ecoMultiplier = 0.2
		}
	}

	// 4. Variant Human labor intensity multiplier
	laborMultiplier := 1.0
	variantBonus := 1.0
	race := getVariantHumanForNode(n.NodeType)
	if race != "" {
		var pop int
		_ = tx.QueryRow(`
			SELECT count FROM aperture_variant_humans 
			WHERE aperture_id = $1 AND race = $2
		`, n.ApertureID, race).Scan(&pop)
		if pop > 0 {
			variantBonus = 1.0 + float64(pop)/1000.0
		}

		var intensity string
		err := tx.QueryRow(`
			SELECT labor_intensity FROM aperture_variant_humans 
			WHERE aperture_id = $1 AND race = $2
		`, n.ApertureID, race).Scan(&intensity)
		if err == nil {
			switch intensity {
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

	// 5. Node Level / Rank multiplier (medium scaling: +20% per level above 1)
	nodeLevel := n.Level
	if nodeLevel < 1 { nodeLevel = 1 }
	nodeLevelMultiplier := 1.0 + float64(nodeLevel-1)*0.20

	// 6. Item Rank / Rarity divisor (scaled by standard rank power table P)
	var itemFoodPathMultiplier float64 = 1.0
	var itemRankDivisor float64 = 1.0
	if n.ProductionType == "Material" {
		prodKey := resolveNodeProdKey(n)
		if prodKey != "" {
			if item, exists := models.ItemRegistry[prodKey]; exists {
				if item.Path != "" {
					itemFoodPathMultiplier = getPathDaoMultiplier(item.Path, nodeDaoMap)
				}
				effectiveRank := item.Rank
				if item.Rarity > effectiveRank { effectiveRank = item.Rarity }
				if effectiveRank >= 1 { itemRankDivisor = getManaVal(effectiveRank) }
			}
		}
	}

	baseReward := n.ProductionAmount * multiples

	// Apply area size production multiplier, node level, item rank divisor + optional bulk discount
	areaMultiplier := 1.0 + float64(areaSize)/10000000.0
	combinedMultiplier := (areaMultiplier * daoMultiplier * conflictMultiplier * ecoMultiplier * variantBonus * laborMultiplier * suitabilityMultiplier * itemFoodPathMultiplier * nodeLevelMultiplier) / itemRankDivisor

	totalReward := applyBulkReward(baseReward, combinedMultiplier, bulkMultiplier)
	if n.ProductionType == "Material" && totalReward > 10000000 {
		totalReward = 10000000
	}

	note := ""
	return totalReward, note, nil
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
	err = tx.QueryRow(`
		SELECT id, area_size, COALESCE(time_rate, 1.0) FROM immortal_apertures WHERE player_id = $1
	`, playerID).Scan(&apertureID, &areaSize, &timeRate)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Tiên Khiếu."})
		return
	}
	if timeRate <= 0 { timeRate = 1.0 }

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
			if lastCol.Valid { n.LastCollectedAt = &lastCol.Time }
			allNodes = append(allNodes, n)
		}
	}
	rows.Close()

	// 3. Determine ready resource nodes and compute rewards
	type collectResult struct {
		NodeID string `json:"node_id"`
		Name   string `json:"name"`
		Reward int    `json:"reward"`
	}

	var readyCount int
	var totalStones int
	materialTotals := make(map[string]int)
	var results []collectResult

	now := time.Now()

	for _, n := range allNodes {
		if n.NodeCategory != "RESOURCE" || n.ProductionAmount <= 0 {
			continue
		}

		intervalHours := n.ProductionIntervalHours
		if intervalHours <= 0 { intervalHours = 1 }
		intervalDuration := time.Duration(float64(intervalHours*3600)/timeRate) * time.Second

		var elapsed time.Duration
		if n.LastCollectedAt != nil {
			elapsed = now.Sub(*n.LastCollectedAt)
		} else {
			elapsed = now.Sub(n.CreatedAt)
		}

		multiples := int(elapsed / intervalDuration)
		if multiples < 1 { continue }

		// Calculate reward with 0.8 bulk discount (-20%)
		reward, _, err := computeApertureNodeReward(tx, playerID, n, multiples, areaSize, allNodes, 0.8)
		if err != nil || reward <= 0 { continue }

		if n.ProductionType == "Stones" {
			totalStones += reward
		} else if n.ProductionType == "Material" {
			prodKey := n.ProductionKey
			if prodKey == "" {
				if template, ok := ResourceNodeTemplates[n.NodeType]; ok {
					prodKey = template.ProdKey
				} else {
					prodKey = "unknown"
				}
			}
			materialTotals[prodKey] += reward
		}

		results = append(results, collectResult{NodeID: n.ID, Name: n.Name, Reward: reward})
		readyCount++
	}

	if readyCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có mỏ nào sẵn sàng để thu hoạch."})
		return
	}

	// 4. Grant rewards
	if totalStones > 0 {
		if _, err := tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, totalStones, playerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng Nguyên Thạch"})
			return
		}
	}
	for prodKey, amount := range materialTotals {
		if err := InventorySvc.AddItem(tx, playerID.String(), prodKey, "materials", amount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng vật phẩm thu hoạch"})
			return
		}
	}

	// 5. Reset timers + Human Qi for harvested nodes
	for _, r := range results {
		if _, err := tx.Exec(`UPDATE aperture_resource_nodes SET last_collected_at = NOW() WHERE id = $1`, r.NodeID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật thời gian mỏ"})
			return
		}
	}
	if _, err := tx.Exec(`UPDATE players SET human_qi = LEAST(human_qi + $1, $2) WHERE id = $3`, float64(readyCount), MaxHumanQi, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
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
		"message":      bulkMsg,
		"count":        readyCount,
		"total_stones": totalStones,
		"materials":    materialTotals,
		"collected":    results,
	})
}

// RepairAllApertureNodes repairs every damaged or destroyed node in the player's aperture at
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

	// 2. Load damaged/destroyed nodes
	rows, err := tx.Query(`
		SELECT id, name, hp, max_hp, is_destroyed
		FROM aperture_resource_nodes
		WHERE aperture_id = $1 AND (is_destroyed = true OR hp < max_hp)
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
		var destroyed bool
		if err := rows.Scan(&id, &name, &hp, &maxHP, &destroyed); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc dữ liệu công trình"})
			return
		}
		_, cost := computeBulkRepairCost(hp, maxHP, destroyed, 1.3)
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
	if _, err := tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, totalCost, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên Thạch"})
		return
	}

	for _, r := range results {
		if _, err := tx.Exec(`
			UPDATE aperture_resource_nodes
			SET hp = max_hp, is_destroyed = false, updated_at = NOW()
			WHERE id = $1
		`, r.NodeID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái sửa chữa"})
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
		"message":        repairAllMsg,
		"repaired_count": len(results),
		"total_cost":     totalCost,
		"nodes":          results,
	})
}

// UninstallApertureNode uninstalls a node, deleting it and refunding 1x node_<type> to materials_inventory
func UninstallApertureNode(c *gin.Context) {
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
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.is_destroyed
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.IsDestroyed)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}

	if n.IsDestroyed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ tài nguyên đã bị phá hủy, không thể tháo dỡ thu hồi!"})
		return
	}

	// 2. Add refund item to materials_inventory
	nodeItemID := "node_" + strings.ToLower(n.NodeType)
	err = InventorySvc.AddItem(tx, playerID.String(), nodeItemID, "materials", 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hoàn trả vật phẩm khi thu hồi mỏ"})
		return
	}

	// 3. Delete node from database completely
	_, err = tx.Exec(`DELETE FROM aperture_resource_nodes WHERE id = $1`, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xóa điểm tài nguyên trong cơ sở dữ liệu"})
		return
	}

	err = updateVariantHumanOnNodeChange(tx, n.ApertureID, n.NodeType, -50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật số lượng dị nhân: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to uninstall node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Tháo dỡ thành công %s, đã thu hồi lại mô hình mỏ vào túi đồ.", n.Name),
		"node_id": nodeIDStr,
	})
}

// DeleteApertureNode completely deletes a node (even if destroyed) without refunding the item
func DeleteApertureNode(c *gin.Context) {
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

	// 1. Fetch node details to verify ownership
	var n models.ApertureResourceNode
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.name, n.node_type
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.Name, &n.NodeType)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}

	// 2. Delete node from database
	_, err = tx.Exec(`DELETE FROM aperture_resource_nodes WHERE id = $1`, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xóa điểm tài nguyên trong cơ sở dữ liệu"})
		return
	}

	err = updateVariantHumanOnNodeChange(tx, n.ApertureID, n.NodeType, -50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật số lượng dị nhân: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("San phẳng thành công %s. Vị trí đã được giải phóng.", n.Name),
		"node_id": nodeIDStr,
	})
}

// ArePathsConflicting returns true if two paths conflict
func ArePathsConflicting(path1, path2 string) bool {
	p1 := strings.ToLower(strings.TrimSpace(path1))
	p2 := strings.ToLower(strings.TrimSpace(path2))
	if p1 == p2 {
		return false
	}
	if p1 > p2 {
		p1, p2 = p2, p1
	}
	// e.g. "earth path:wood path"
	conflicts := map[string]bool{
		"fire path:water path": true,
		"metal path:wood path": true,
		"earth path:wood path": true, // Wood roots break earth
		"fire path:ice path":   true,
	}
	return conflicts[p1+":"+p2]
}

// ChebyshevDistance returns Chebyshev distance between two coordinates
func ChebyshevDistance(x1, y1, x2, y2 int) int {
	dx := x1 - x2
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y2
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

// CheckNodeConflict returns true if a node is in conflict with any other node
func CheckNodeConflict(node models.ApertureResourceNode, allNodes []models.ApertureResourceNode) bool {
	for _, other := range allNodes {
		if other.ID == node.ID || other.IsDestroyed {
			continue
		}
		if ChebyshevDistance(node.X, node.Y, other.X, other.Y) <= 2 {
			if ArePathsConflicting(node.Path, other.Path) {
				return true
			}
		}
	}
	return false
}

// EvaluateEcosystem computes plant, herbivore, and carnivore metrics
func EvaluateEcosystem(allNodes []models.ApertureResourceNode) (plants, herbivores, carnivores int, overgrazing bool, starvationHerb bool, starvationCarn bool) {
	for _, n := range allNodes {
		if n.IsDestroyed {
			continue
		}
		switch n.NodeType {
		case "MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS",
			"TRI_FARM", "TRI_FARM_ADV", "HOA_FARM", "TIN_FARM", "AM_FARM", "DOC_FARM",
			"BAC_FARM", "QUANG_FARM", "KIEM_FARM", "THUY_FARM", "CHUA_FARM", "CAM_FARM",
			"KIM_FARM", "VANG_FARM", "HUYET_FARM", "THAT_FARM", "THAY_THUAT_FARM", "LUAT_FARM",
			"THUAT_FARM", "THOI_THUAT_FARM", "LUC_FARM", "THOI_FARM", "PHONG_FARM",
			"NO_FARM", "NO_FARM_ADV":
			plants++
		case "SPIRIT_DEER_PASTURE":
			herbivores++
		case "SILVER_WOLF_DEN", "FEATHERMAN_CANYON", "HAIRY_MAN_SETTLEMENT", "ROCKMAN_TRIBE":
			carnivores++
		}
	}

	starvationHerb = herbivores > 0 && plants == 0
	starvationCarn = carnivores > 0 && herbivores == 0
	overgrazing = herbivores > plants && carnivores == 0 && plants > 0
	return
}

func updateVariantHumanOnNodeChange(tx *sql.Tx, apertureID string, nodeType string, change int) error {
	var race string
	switch nodeType {
	case "HAIRY_MAN_SETTLEMENT":
		race = "HAIRY_MAN"
	case "ROCKMAN_TRIBE":
		race = "ROCKMAN"
	case "FEATHERMAN_CANYON":
		race = "FEATHERMAN"
	default:
		return nil
	}

	// Update population
	_, err := tx.Exec(`
		INSERT INTO aperture_variant_humans (aperture_id, race, population, happiness, oppression, rebellion_rate, labor_intensity)
		VALUES ($1, $2, GREATEST(0, $3), 70.0, 0.0, 0.0, 'NORMAL')
		ON CONFLICT (aperture_id, race) 
		DO UPDATE SET population = GREATEST(0, aperture_variant_humans.population + $3), updated_at = NOW()
	`, apertureID, race, change)
	return err
}

type SetTimeRateRequest struct {
	TimeRate float64 `json:"time_rate" binding:"required"`
}

func SetApertureTimeRate(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SetTimeRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	if req.TimeRate < 1.0 || req.TimeRate > 100.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tốc độ dòng thời gian chỉ được điều chỉnh từ 1.0x đến 100.0x"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi giao dịch"})
		return
	}
	defer tx.Rollback()

	// Project internal_years_passed forward using the OLD rate before changing it.
	// This prevents the frontend from miscalculating the arrival time when
	// updated_at is set to NOW() but internal_years_passed is stale.
	now := time.Now()
	res, err := tx.Exec(`
		UPDATE immortal_apertures ia
		SET internal_years_passed = ia.internal_years_passed + GREATEST(0, EXTRACT(EPOCH FROM (NOW() - ia.updated_at))) * ia.time_rate * (p.time_speed_multiplier + COALESCE((SELECT SUM((gd.effects->'passive'->>'time_speed_multiplier')::double precision) FROM player_gu pg JOIN gu_dictionary gd ON gd.id = pg.gu_id AND gd.rank = pg.gu_rank WHERE pg.player_id = p.id AND pg.equipped = true AND pg.state = 'ACTIVE'), 0)) / 36000.0,
			time_rate = $1, updated_at = $2
		FROM players p
		WHERE ia.player_id = p.id AND ia.player_id = $3
	`, req.TimeRate, now, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật dòng thời gian"})
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi ghi dữ liệu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Điều chỉnh dòng thời gian Tiên Khiếu thành công: %.1fx", req.TimeRate)})
}

type SetLaborRequest struct {
	Race           string `json:"race" binding:"required"`
	LaborIntensity string `json:"labor_intensity" binding:"required"`
}

func SetApertureLabor(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SetLaborRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	intensity := strings.ToUpper(req.LaborIntensity)
	if intensity != "LOW" && intensity != "NORMAL" && intensity != "HIGH" && intensity != "EXPLOIT" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cường độ lao động không hợp lệ (LOW, NORMAL, HIGH, EXPLOIT)"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var apID string
	err = tx.QueryRow(`SELECT id FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	}

	_, err = tx.Exec(`
		UPDATE aperture_variant_humans
		SET labor_intensity = $1, updated_at = NOW()
		WHERE aperture_id = $2 AND race = $3
	`, intensity, apID, req.Race)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật cường độ lao động"})
		return
	}

	_ = tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Cập nhật cường độ lao động của %s thành %s thành công!", req.Race, intensity)})
}

func RepairApertureNode(c *gin.Context) {
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

	// Fetch node
	var n models.ApertureResourceNode
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.name, n.hp, n.max_hp, n.is_destroyed
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.Name, &n.HP, &n.MaxHP, &n.IsDestroyed)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}

	cost := int(math.Ceil(float64(n.MaxHP-n.HP) / 50.0))
	if n.IsDestroyed {
		cost = int(math.Ceil(float64(n.MaxHP*2) / 50.0))
	}

	if cost <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ tài nguyên đang ở trạng thái nguyên vẹn tốt nhất."})
		return
	}

	// Check player stones
	var stones int
	err = tx.QueryRow(`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1`, playerID).Scan(&stones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
		return
	}

	if stones < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Nguyên Thạch không đủ để sửa chữa mỏ này (Yêu cầu: %d, Hiện có: %d).", cost, stones)})
		return
	}

	// Deduct stones
	_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, cost, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên Thạch"})
		return
	}

	// Repair node
	_, err = tx.Exec(`
		UPDATE aperture_resource_nodes
		SET hp = max_hp, is_destroyed = false, updated_at = NOW()
		WHERE id = $1
	`, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái sửa chữa"})
		return
	}

	_ = tx.Commit()

	repairMsg := fmt.Sprintf("Sửa chữa thành công %s! Tiêu tốn %d Tiên Nguyên Thạch.", n.Name, cost)
	notify.Send(db.DB, playerID, "aperture_repair", "Sửa Chữa Tiên Khiếu", repairMsg)

	c.JSON(http.StatusOK, gin.H{"message": repairMsg})
}

const MaxNodeLevel = 5

// UpgradeApertureNode upgrades a resource/defense node to the next level
func UpgradeApertureNode(c *gin.Context) {
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

	// Fetch node
	var n models.ApertureResourceNode
	err = tx.QueryRow(`
		SELECT n.id, n.aperture_id, n.name, n.node_type, n.level, n.max_hp, n.is_destroyed,
			COALESCE(n.node_category, 'RESOURCE'), COALESCE(n.defense_power, 0), COALESCE(n.attack_range, 0),
			COALESCE(n.hp_bonus, 0), COALESCE(n.damage_reduction, 0), COALESCE(n.production_amount, 0)
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.Name, &n.NodeType, &n.Level, &n.MaxHP, &n.IsDestroyed,
		&n.NodeCategory, &n.DefensePower, &n.AttackRange,
		&n.HPBonus, &n.DamageReduction, &n.ProductionAmount)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mỏ tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}

	if n.IsDestroyed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể nâng cấp công trình đã bị phá hủy!"})
		return
	}

	if n.Level >= MaxNodeLevel {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đã đạt cấp tối đa (%d). Không thể nâng cấp thêm.", MaxNodeLevel)})
		return
	}

	// Calculate upgrade cost: base cost * level * 1.5
	template, ok := ResourceNodeTemplates[n.NodeType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại node không hợp lệ"})
		return
	}
	upgradeCost := int(float64(template.CostStones) * float64(n.Level) * 1.5)
	if upgradeCost < 5 {
		upgradeCost = 5
	}

	// Check player stones
	var stones int
	err = tx.QueryRow(`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1`, playerID).Scan(&stones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
		return
	}

	if stones < upgradeCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Nguyên Thạch không đủ để nâng cấp (Yêu cầu: %d, Hiện có: %d).", upgradeCost, stones)})
		return
	}

	// Deduct stones
	_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, upgradeCost, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên Thạch"})
		return
	}

	// Calculate new stats
	newLevel := n.Level + 1
	newMaxHP := n.MaxHP + 500
	hpGain := newMaxHP - n.MaxHP // HP increased by this amount

	newProductionAmount := n.ProductionAmount
	newDefensePower := n.DefensePower
	newHPBonus := n.HPBonus

	if n.NodeCategory == "RESOURCE" {
		// Resource nodes: +50% production per level
		newProductionAmount = int(float64(n.ProductionAmount) * 1.5)
		if newProductionAmount == n.ProductionAmount {
			newProductionAmount = n.ProductionAmount + 1
		}
	} else {
		// Defense nodes: +25% defense stats per level
		if n.DefensePower > 0 {
			newDefensePower = int(float64(n.DefensePower) * 1.25)
			if newDefensePower == n.DefensePower {
				newDefensePower = n.DefensePower + 1
			}
		}
		if n.HPBonus > 0 {
			newHPBonus = int(float64(n.HPBonus) * 1.25)
			if newHPBonus == n.HPBonus {
				newHPBonus = n.HPBonus + 1
			}
		}
	}

	// Apply upgrade
	_, err = tx.Exec(`
		UPDATE aperture_resource_nodes
		SET level = $1, max_hp = $2, hp = hp + $3,
			production_amount = $4, defense_power = $5, hp_bonus = $6,
			updated_at = NOW()
		WHERE id = $7
	`, newLevel, newMaxHP, hpGain, newProductionAmount, newDefensePower, newHPBonus, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi nâng cấp: " + err.Error()})
		return
	}

	_ = tx.Commit()

	upgradeMsg := fmt.Sprintf("Nâng cấp %s lên cấp %d thành công! Tiêu tốn %d Tiên Nguyên Thạch.", n.Name, newLevel, upgradeCost)
	notify.Send(db.DB, playerID, "aperture_upgrade", "Nâng Cấp Tiên Khiếu", upgradeMsg)

	c.JSON(http.StatusOK, gin.H{
		"message":    upgradeMsg,
		"new_level":  newLevel,
		"new_max_hp": newMaxHP,
		"cost":       upgradeCost,
	})
}

func GetApertureVariantHumans(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT v.id, v.aperture_id, v.race, v.population, v.happiness, v.oppression, v.rebellion_rate, v.labor_intensity, v.created_at, v.updated_at
		FROM aperture_variant_humans v
		JOIN immortal_apertures a ON v.aperture_id = a.id
		WHERE a.player_id = $1
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin dị nhân"})
		return
	}
	defer rows.Close()

	list := []models.ApertureVariantHuman{}
	for rows.Next() {
		var v models.ApertureVariantHuman
		err = rows.Scan(&v.ID, &v.ApertureID, &v.Race, &v.Population, &v.Happiness, &v.Oppression, &v.RebellionRate, &v.LaborIntensity, &v.CreatedAt, &v.UpdatedAt)
		if err == nil {
			list = append(list, v)
		}
	}

	c.JSON(http.StatusOK, list)
}
