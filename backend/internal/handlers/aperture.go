package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
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
}

var ResourceNodeTemplates = map[string]ResourceNodeTemplate{
	"MOON_ORCHID_FIELD": {
		NodeType:          "MOON_ORCHID_FIELD",
		Name:              "Cánh Đồng Hoa Nguyệt Lan",
		Path:              "Moon Path",
		CostStones:        500,
		ProdType:          "Material",
		ProdKey:           "moon_orchid_petal",
		ProdAmount:        10,
		ProdIntervalHours: 4,
	},
	"GREEN_BAMBOO_GROVE": {
		NodeType:          "GREEN_BAMBOO_GROVE",
		Name:              "Rừng Trúc Thanh Trúc",
		Path:              "Wood Path",
		CostStones:        600,
		ProdType:          "Material",
		ProdKey:           "green_bamboo_wine",
		ProdAmount:        5,
		ProdIntervalHours: 6,
	},
	"JADE_STONE_MINE": {
		NodeType:          "JADE_STONE_MINE",
		Name:              "Mỏ Ngọc Thạch Bản Địa",
		Path:              "Earth Path",
		CostStones:        1000,
		ProdType:          "Material",
		ProdKey:           "jade_stone",
		ProdAmount:        3,
		ProdIntervalHours: 8,
	},
	"PRIMEVAL_MINE": {
		NodeType:          "PRIMEVAL_MINE",
		Name:              "Mạch Nguyên Thạch Tiên Khiếu",
		Path:              "Rule Path",
		CostStones:        1500,
		ProdType:          "Stones",
		ProdAmount:        200,
		ProdIntervalHours: 12,
	},
	"SWORD_LEAF_GRASS": {
		NodeType:          "SWORD_LEAF_GRASS",
		Name:              "Đồng Cỏ Kiếm Diệp",
		Path:              "Sword Path",
		CostStones:        800,
		ProdType:          "Material",
		ProdKey:           "sword_leaf_grass",
		ProdAmount:        8,
		ProdIntervalHours: 6,
	},
	"SPIRIT_DEER_PASTURE": {
		NodeType:          "SPIRIT_DEER_PASTURE",
		Name:              "Bãi Chăn Thả Hươu Linh",
		Path:              "Food Path",
		CostStones:        1200,
		ProdType:          "Material",
		ProdKey:           "spirit_deer_meat",
		ProdAmount:        5,
		ProdIntervalHours: 8,
	},
	"SILVER_WOLF_DEN": {
		NodeType:          "SILVER_WOLF_DEN",
		Name:              "Hang Sói Bạc",
		Path:              "Food Path",
		CostStones:        2000,
		ProdType:          "Material",
		ProdKey:           "silver_wolf_pelt",
		ProdAmount:        3,
		ProdIntervalHours: 12,
	},
	"HAIRY_MAN_SETTLEMENT": {
		NodeType:          "HAIRY_MAN_SETTLEMENT",
		Name:              "Bộ Lạc Người Lông",
		Path:              "Human Path",
		CostStones:        3000,
		ProdType:          "Material",
		ProdKey:           "hairy_man_slaves",
		ProdAmount:        1,
		ProdIntervalHours: 24,
	},
	"ROCKMAN_TRIBE": {
		NodeType:          "ROCKMAN_TRIBE",
		Name:              "Bản Doanh Người Đá",
		Path:              "Human Path",
		CostStones:        3000,
		ProdType:          "Material",
		ProdKey:           "jade_stone",
		ProdAmount:        2,
		ProdIntervalHours: 24,
	},
	"FEATHERMAN_CANYON": {
		NodeType:          "FEATHERMAN_CANYON",
		Name:              "Hẻm Núi Người Lông Vũ",
		Path:              "Human Path",
		CostStones:        3000,
		ProdType:          "Material",
		ProdKey:           "featherman_feather",
		ProdAmount:        4,
		ProdIntervalHours: 18,
	},
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
		SELECT id, rank, aptitude_percent, human_qi, primeval_stones, immortal_stones, dao_marks, current_mana
		FROM players WHERE id = $1
	`, playerID).Scan(&player.ID, &player.Rank, &player.AptitudePercent, &player.HumanQi, &player.PrimevalStones, &player.ImmortalStones, &player.DaoMarks, &player.CurrentMana)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Cổ sư không tồn tại"})
		return
	}

	if player.Rank < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa thăng tiên (Rank 6), chưa có Tiên Khiếu."})
		return
	}

	// Sync player.DaoMarks JSONB to dao_marks table if API table exists
	if player.DaoMarks != nil {
		for pathName, amount := range player.DaoMarks {
			if amount > 0 {
				cleanPath := strings.TrimSuffix(strings.ToLower(pathName), " path")
				_, _ = tx.Exec(`
					INSERT INTO dao_marks (player_id, path, amount)
					VALUES ($1, $2, $3)
					ON CONFLICT (player_id, path) DO UPDATE SET amount = EXCLUDED.amount
				`, playerID, cleanPath, amount)
			}
		}
	}

	// 2. Fetch aperture
	var ap models.ImmortalAperture
	var createdAt, updatedAt time.Time
	err = tx.QueryRow(`
		SELECT id, player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, tribulation_count, rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, state, created_at, updated_at
		FROM immortal_apertures WHERE player_id = $1
	`, playerID).Scan(&ap.ID, &ap.PlayerID, &ap.Grade, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.AreaSize, &ap.ImmortalEssence, &ap.EssenceProductionPerYear, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.TribulationCount, &ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.State, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		// Auto-create aperture based on stats
		score := int(player.HumanQi) + player.AptitudePercent + 250 // Base boost
		grade := "LOW"
		if score >= 900 {
			grade = "EXTREME"
		} else if score >= 700 {
			grade = "HIGH"
		} else if score >= 450 {
			grade = "MID"
		}

		ap = buildApertureByGrade(playerID.String(), grade)
		ap.ImmortalEssence = int(player.CurrentMana)
		err = tx.QueryRow(`
			INSERT INTO immortal_apertures (player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, state)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id, created_at, updated_at
		`, ap.PlayerID, ap.Grade, ap.TimeRate, ap.ApertureHP, ap.MaxApertureHP, ap.AreaSize, ap.ImmortalEssence, ap.EssenceProductionPerYear, ap.InternalYearsPassed, ap.NextTribulationYear, ap.State).Scan(&ap.ID, &createdAt, &updatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khai mở Tiên Khiếu: " + err.Error()})
			return
		}
		// Sync time_speed_multiplier to player table
		_, _ = tx.Exec(`UPDATE players SET time_speed_multiplier = $1 WHERE id = $2`, ap.TimeRate, playerID)

		// Seed a few random initial resource nodes for the new aperture
		seedInitialApertureNodes(tx, ap.ID, grade)
	} else if err != nil {
		logger.S.Errorf("Error fetching aperture: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn Tiên Khiếu"})
		return
	} else {
		// Sync current mana to database
		ap.ImmortalEssence = int(player.CurrentMana)
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE id = $2`, ap.ImmortalEssence, ap.ID)
	}

	ap.CreatedAt = createdAt
	ap.UpdatedAt = updatedAt
	FillApertureEnvironment(&ap, player.DaoMarks)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failure"})
		return
	}

	c.JSON(http.StatusOK, ap)
}

func seedInitialApertureNodes(tx *sql.Tx, apertureID string, grade string) {
	starterKeys := []string{
		"MOON_ORCHID_FIELD",
		"GREEN_BAMBOO_GROVE",
		"JADE_STONE_MINE",
		"PRIMEVAL_MINE",
		"SWORD_LEAF_GRASS",
		"SPIRIT_DEER_PASTURE",
	}

	count := 2
	switch grade {
	case "MID":
		count = 3
	case "HIGH", "EXTREME":
		count = 4
	}

	rand.Shuffle(len(starterKeys), func(i, j int) {
		starterKeys[i], starterKeys[j] = starterKeys[j], starterKeys[i]
	})

	used := make(map[string]bool)
	for i := 0; i < count && i < len(starterKeys); i++ {
		tmpl := ResourceNodeTemplates[starterKeys[i]]

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
				 production_type, production_amount, production_interval_hours, last_collected_at)
			VALUES ($1, $2, $3, $4, 1, 1000, 1000, $5, $6, $7, $8, $9, NOW())
		`, apertureID, tmpl.NodeType, tmpl.Name, tmpl.Path, x, y, tmpl.ProdType, tmpl.ProdAmount, tmpl.ProdIntervalHours)
	}
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

	rows, err := db.DB.Query(`
		SELECT id, player_id, path, amount, created_at, updated_at
		FROM dao_marks WHERE player_id = $1
		ORDER BY amount DESC
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load Dao Marks"})
		return
	}
	defer rows.Close()

	marks := []models.DaoMark{}
	for rows.Next() {
		var m models.DaoMark
		err = rows.Scan(&m.ID, &m.PlayerID, &m.Path, &m.Amount, &m.CreatedAt, &m.UpdatedAt)
		if err == nil {
			marks = append(marks, m)
		}
	}

	c.JSON(http.StatusOK, marks)
}

// GetApertureNodes returns resource nodes in player's aperture
func GetApertureNodes(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.path, n.level, n.hp, n.max_hp, n.x, n.y, n.production_type, n.production_amount, n.production_interval_hours, n.last_collected_at, n.is_destroyed, n.created_at, n.updated_at
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
		err = rows.Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.Path, &n.Level, &n.HP, &n.MaxHP, &n.X, &n.Y, &n.ProductionType, &n.ProductionAmount, &n.ProductionIntervalHours, &lastCol, &n.IsDestroyed, &n.CreatedAt, &n.UpdatedAt)
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

	var stones int
	var immortalStones float64
	var matsJSON []byte
	err = tx.QueryRow(`
		SELECT primeval_stones, immortal_stones, materials_inventory FROM players WHERE id = $1
	`, playerID).Scan(&stones, &immortalStones, &matsJSON)
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

	// 3. Deduct cost or item
	var materials map[string]int
	_ = json.Unmarshal(matsJSON, &materials)
	if materials == nil {
		materials = make(map[string]int)
	}

	nodeItemID := "node_" + strings.ToLower(req.NodeType)
	hasItem := materials[nodeItemID] > 0

	if hasItem {
		materials[nodeItemID]--
		if materials[nodeItemID] <= 0 {
			delete(materials, nodeItemID)
		}
		updatedMats, _ := json.Marshal(materials)
		_, err = tx.Exec(`UPDATE players SET materials_inventory = $1 WHERE id = $2`, updatedMats, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ vật phẩm cơ sở"})
			return
		}
	} else {
		if stones < template.CostStones || int(immortalStones) < template.CostImmortal {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nguyên thạch hoặc Tiên nguyên không đủ để xây dựng mỏ này."})
			return
		}
		_, err = tx.Exec(`
			UPDATE players 
			SET primeval_stones = primeval_stones - $1, immortal_stones = immortal_stones - $2
			WHERE id = $3
		`, template.CostStones, template.CostImmortal, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ chi phí"})
			return
		}
	}

	// 4. Insert node
	var node models.ApertureResourceNode
	var createdAt time.Time
	err = tx.QueryRow(`
		INSERT INTO aperture_resource_nodes (aperture_id, node_type, name, path, level, hp, max_hp, x, y, production_type, production_amount, production_interval_hours, last_collected_at)
		VALUES ($1, $2, $3, $4, 1, 1000, 1000, $5, $6, $7, $8, $9, NOW())
		RETURNING id, created_at
	`, ap.ID, template.NodeType, template.Name, template.Path, req.X, req.Y, template.ProdType, template.ProdAmount, template.ProdIntervalHours).Scan(&node.ID, &createdAt)
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
	node.HP = 1000
	node.MaxHP = 1000
	node.X = req.X
	node.Y = req.Y
	node.ProductionType = template.ProdType
	node.ProductionAmount = template.ProdAmount
	node.ProductionIntervalHours = template.ProdIntervalHours
	node.CreatedAt = createdAt
	node.UpdatedAt = createdAt

	c.JSON(http.StatusOK, gin.H{
		"node":    node,
		"message": fmt.Sprintf("Xây dựng thành công %s!", template.Name),
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
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.production_type, n.production_amount, n.production_interval_hours, n.last_collected_at, n.is_destroyed, a.area_size, a.time_rate, n.path, n.x, n.y
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.ProductionType, &n.ProductionAmount, &n.ProductionIntervalHours, &lastCol, &n.IsDestroyed, &areaSize, &timeRate, &n.Path, &n.X, &n.Y)

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

	// 1. Dao Marks multiplier
	var daoMarks int
	_ = tx.QueryRow(`SELECT amount FROM dao_marks WHERE player_id = $1 AND path = $2`, playerID, n.Path).Scan(&daoMarks)
	daoMultiplier := 1.0 + float64(daoMarks)*0.005

	// Check if player has Tiên Thai Chí Tôn (ignore Dao conflicts)
	hasSovereignFetus := false
	var guJSON []byte
	errGu := tx.QueryRow(`SELECT gu_inventory FROM players WHERE id = $1`, playerID).Scan(&guJSON)
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
	case "MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS":
		if overgrazing {
			ecoMultiplier = 0.5
		} else if herbivores > 0 {
			ecoMultiplier = 1.25 // Fertilization bonus
		}
	case "SPIRIT_DEER_PASTURE":
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

	// Check if Rockman labor boosts mine nodes
	if n.NodeType == "JADE_STONE_MINE" || n.NodeType == "PRIMEVAL_MINE" {
		var pop int
		var intensity string
		err = tx.QueryRow(`
			SELECT population, labor_intensity FROM aperture_variant_humans 
			WHERE aperture_id = $1 AND race = 'ROCKMAN'
		`, n.ApertureID).Scan(&pop, &intensity)
		if err == nil && pop > 0 {
			variantBonus = 1.0 + float64(pop)*0.001
			if variantBonus > 1.5 {
				variantBonus = 1.5
			}
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
	} else if n.NodeType == "HAIRY_MAN_SETTLEMENT" || n.NodeType == "ROCKMAN_TRIBE" || n.NodeType == "FEATHERMAN_CANYON" {
		// Settlement production is boosted by its own labor intensity setting
		var race string
		if n.NodeType == "HAIRY_MAN_SETTLEMENT" {
			race = "HAIRY_MAN"
		} else if n.NodeType == "ROCKMAN_TRIBE" {
			race = "ROCKMAN"
		} else {
			race = "FEATHERMAN"
		}
		var pop int
		var intensity string
		err = tx.QueryRow(`
			SELECT population, labor_intensity FROM aperture_variant_humans 
			WHERE aperture_id = $1 AND race = $2
		`, n.ApertureID, race).Scan(&pop, &intensity)
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

	// Calculate dynamic environment parameters from player's Dao Marks
	var daoJSON []byte
	var daoMap models.DaoMarksMap
	_ = tx.QueryRow(`SELECT dao_marks FROM players WHERE id = $1`, playerID).Scan(&daoJSON)
	if daoJSON != nil {
		_ = json.Unmarshal(daoJSON, &daoMap)
	}
	if daoMap == nil {
		daoMap = make(models.DaoMarksMap)
	}

	tempFire := daoMap["fire"]
	tempIce := daoMap["ice"]
	tempMoon := daoMap["moon"]
	temperature := 25.0 + float64(tempFire)/500.0 - float64(tempIce+tempMoon)/500.0

	tempWater := daoMap["water"]
	humidity := 50.0 + float64(tempWater)/200.0 - float64(tempFire)/200.0
	if humidity < 0 {
		humidity = 0
	}
	if humidity > 100 {
		humidity = 100
	}

	suitabilityMultiplier := 1.0
	cleanNodePath := strings.TrimSuffix(strings.ToLower(n.Path), " path")
	switch cleanNodePath {
	case "moon":
		if temperature > 40.0 || temperature < 10.0 {
			suitabilityMultiplier *= 0.6
		}
		if humidity < 30.0 {
			suitabilityMultiplier *= 0.7
		}
	case "wood":
		if temperature < 5.0 {
			suitabilityMultiplier *= 0.5
		}
		if humidity < 25.0 {
			suitabilityMultiplier *= 0.6
		}
	case "fire":
		if temperature < 20.0 {
			suitabilityMultiplier *= 0.5
		}
	case "water":
		if humidity < 40.0 {
			suitabilityMultiplier *= 0.5
		}
	}

	baseReward := n.ProductionAmount * multiples

	// Apply area size production multiplier
	areaMultiplier := 1.0 + float64(areaSize)/10000000.0
	totalReward := int(float64(baseReward) * areaMultiplier * daoMultiplier * conflictMultiplier * ecoMultiplier * variantBonus * laborMultiplier * suitabilityMultiplier)
	if totalReward < 1 && baseReward > 0 {
		totalReward = 1
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
		template, ok := ResourceNodeTemplates[n.NodeType]
		prodKey := "unknown"
		if ok {
			prodKey = template.ProdKey
		}

		// Fetch current player materials
		var matsJSON []byte
		err = tx.QueryRow(`SELECT materials_inventory FROM players WHERE id = $1`, playerID).Scan(&matsJSON)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy túi đồ"})
			return
		}

		var materials map[string]int
		_ = json.Unmarshal(matsJSON, &materials)
		if materials == nil {
			materials = make(map[string]int)
		}

		materials[prodKey] += totalReward
		updatedMats, _ := json.Marshal(materials)

		_, err = tx.Exec(`UPDATE players SET materials_inventory = $1 WHERE id = $2`, updatedMats, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật túi đồ"})
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

	if suitabilityMultiplier < 1.0 {
		rewardMsg += " (Sản lượng bị giảm do sinh cảnh Tiên Khiếu không tương thích!)"
	}

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

	c.JSON(http.StatusOK, gin.H{
		"message": rewardMsg + " Nhân Khí +2.",
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
	var matsJSON []byte
	err = tx.QueryRow(`SELECT materials_inventory FROM players WHERE id = $1`, playerID).Scan(&matsJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy túi đồ"})
		return
	}

	var materials map[string]int
	_ = json.Unmarshal(matsJSON, &materials)
	if materials == nil {
		materials = make(map[string]int)
	}

	nodeItemID := "node_" + strings.ToLower(n.NodeType)
	materials[nodeItemID]++
	updatedMats, _ := json.Marshal(materials)

	_, err = tx.Exec(`UPDATE players SET materials_inventory = $1 WHERE id = $2`, updatedMats, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật túi đồ khi thu hồi mỏ"})
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
		case "MOON_ORCHID_FIELD", "GREEN_BAMBOO_GROVE", "SWORD_LEAF_GRASS":
			plants++
		case "SPIRIT_DEER_PASTURE":
			herbivores++
		case "SILVER_WOLF_DEN":
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

	res, err := tx.Exec(`
		UPDATE immortal_apertures
		SET time_rate = $1, updated_at = NOW()
		WHERE player_id = $2
	`, req.TimeRate, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật dòng thời gian"})
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	}

	_, _ = tx.Exec(`UPDATE players SET time_speed_multiplier = $1 WHERE id = $2`, req.TimeRate, playerID)

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

	cost := n.MaxHP - n.HP
	if n.IsDestroyed {
		cost = n.MaxHP * 2
	}

	if cost <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mỏ tài nguyên đang ở trạng thái nguyên vẹn tốt nhất."})
		return
	}

	// Check player stones
	var stones int
	err = tx.QueryRow(`SELECT primeval_stones FROM players WHERE id = $1`, playerID).Scan(&stones)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
		return
	}

	if stones < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Nguyên thạch không đủ để sửa chữa mỏ này (Yêu cầu: %d, Hiện có: %d).", cost, stones)})
		return
	}

	// Deduct stones
	_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2`, cost, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Nguyên thạch"})
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

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Sửa chữa thành công %s! Tiêu tốn %d Nguyên Thạch.", n.Name, cost)})
}

func GetApertureVariantHumans(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.Query(`
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

func FillApertureEnvironment(ap *models.ImmortalAperture, daoMarks models.DaoMarksMap) {
	if daoMarks == nil {
		daoMarks = make(models.DaoMarksMap)
	}
	fire := daoMarks["fire"]
	ice := daoMarks["ice"]
	moon := daoMarks["moon"]
	ap.Temperature = 25.0 + float64(fire)/500.0 - float64(ice+moon)/500.0

	water := daoMarks["water"]
	humidity := 50.0 + float64(water)/200.0 - float64(fire)/200.0
	if humidity < 0 {
		humidity = 0
	}
	if humidity > 100 {
		humidity = 100
	}
	ap.Humidity = humidity

	qi := daoMarks["qi"]
	rule := daoMarks["rule"]
	ap.Qi = 100.0 + float64(qi+rule)/100.0

	maxAmount := 0
	dominantPath := ""
	for path, amount := range daoMarks {
		if amount > maxAmount {
			maxAmount = amount
			dominantPath = path
		}
	}

	if maxAmount > 100000 {
		ap.Terrain = fmt.Sprintf("Bí Cảnh %s Đạo", strings.Title(dominantPath))
	} else if maxAmount > 20000 {
		ap.Terrain = fmt.Sprintf("Địa Mạch %s Đạo", strings.Title(dominantPath))
	} else if maxAmount > 5000 {
		switch dominantPath {
		case "fire":
			ap.Terrain = "Vùng Núi Lửa Hoạt Động"
		case "water":
			ap.Terrain = "Vùng Biển Sâu Tiên Khiếu"
		case "earth":
			ap.Terrain = "Vùng Núi Non Trùng Điệp"
		case "wood":
			ap.Terrain = "Linh Hải Đại Sâm Lâm"
		case "wind":
			ap.Terrain = "Phong Nha Cốc Tiên Khiếu"
		case "lightning":
			ap.Terrain = "Lôi Điện Hải Vực"
		default:
			ap.Terrain = fmt.Sprintf("Vùng Đặc Thù %s Đạo", strings.Title(dominantPath))
		}
	} else {
		ap.Terrain = "Thảo Nguyên Trù Phú"
	}
}
