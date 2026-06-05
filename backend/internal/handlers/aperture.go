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

	// Sync player.DaoMarks JSONB to dao_marks table if the table has no marks for this player
	var hasMarks bool
	_ = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM dao_marks WHERE player_id = $1)`, playerID).Scan(&hasMarks)
	if !hasMarks && player.DaoMarks != nil {
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
		SELECT id, player_id, grade, time_rate, aperture_hp, max_aperture_hp, area_size, immortal_essence, essence_production_per_year, internal_years_passed, next_tribulation_year, tribulation_count, state, created_at, updated_at
		FROM immortal_apertures WHERE player_id = $1
	`, playerID).Scan(&ap.ID, &ap.PlayerID, &ap.Grade, &ap.TimeRate, &ap.ApertureHP, &ap.MaxApertureHP, &ap.AreaSize, &ap.ImmortalEssence, &ap.EssenceProductionPerYear, &ap.InternalYearsPassed, &ap.NextTribulationYear, &ap.TribulationCount, &ap.State, &createdAt, &updatedAt)

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
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn Tiên Khiếu"})
		return
	} else {
		// Sync current mana to database
		ap.ImmortalEssence = int(player.CurrentMana)
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE id = $2`, ap.ImmortalEssence, ap.ID)
	}

	ap.CreatedAt = createdAt
	ap.UpdatedAt = updatedAt

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failure"})
		return
	}

	c.JSON(http.StatusOK, ap)
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
		SELECT n.id, n.aperture_id, n.node_type, n.name, n.production_type, n.production_amount, n.production_interval_hours, n.last_collected_at, n.is_destroyed, a.area_size, a.time_rate
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.NodeType, &n.Name, &n.ProductionType, &n.ProductionAmount, &n.ProductionIntervalHours, &lastCol, &n.IsDestroyed, &areaSize, &timeRate)

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
	baseReward := n.ProductionAmount * multiples
	
	// Apply area size production multiplier
	areaMultiplier := 1.0 + float64(areaSize)/10000000.0
	totalReward := int(float64(baseReward) * areaMultiplier)

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

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save collection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": rewardMsg,
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
		SELECT n.id, n.aperture_id, n.name
		FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON n.aperture_id = a.id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&n.ID, &n.ApertureID, &n.Name)

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

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete node"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("San phẳng thành công %s. Vị trí đã được giải phóng.", n.Name),
		"node_id": nodeIDStr,
	})
}

