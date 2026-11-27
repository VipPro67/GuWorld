package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// ---------- Request types ----------

type AssignWorkerRequest struct {
	VariantHumanID     string `json:"variant_human_id" binding:"required"`
	AssignedPopulation int    `json:"assigned_population" binding:"required,min=0"`
}

type CreateRefinementJobRequest struct {
	RecipeID       string `json:"recipe_id" binding:"required"`
	Mode           string `json:"mode" binding:"required,oneof=FIXED_COUNT UNTIL_SUCCESS UNTIL_MATERIAL_EMPTY MAINTAIN_STOCK"`
	TargetQuantity int    `json:"target_quantity" binding:"required,min=1"`
	Priority       int    `json:"priority"`
}

// ---------- Refinement Queue ----------

// ListRefinementQueue returns all refinement jobs for the player's aperture.
func ListRefinementQueue(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT rq.id, rq.aperture_id, rq.recipe_id, rq.mode, rq.target_quantity,
		       rq.current_quantity, rq.priority, rq.enabled, rq.last_error,
		       rq.last_processed_at, rq.created_at, rq.updated_at
		FROM aperture_refinement_queue rq
		JOIN immortal_apertures a ON a.id = rq.aperture_id
		WHERE a.player_id = $1
		ORDER BY rq.priority DESC, rq.created_at ASC
	`, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer rows.Close()

	var list []models.ApertureRefinementQueue
	for rows.Next() {
		var j models.ApertureRefinementQueue
		if err := rows.Scan(&j.ID, &j.ApertureID, &j.RecipeID, &j.Mode, &j.TargetQuantity,
			&j.CurrentQuantity, &j.Priority, &j.Enabled, &j.LastError, &j.LastProcessedAt,
			&j.CreatedAt, &j.UpdatedAt); err != nil {
			continue
		}
		list = append(list, j)
	}
	if list == nil {
		list = []models.ApertureRefinementQueue{}
	}
	c.JSON(http.StatusOK, gin.H{"queue": list})
}

// CreateRefinementJob creates a new auto-refinement job.
func CreateRefinementJob(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateRefinementJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	// Get player's aperture ID.
	var apertureID string
	err := db.DB.QueryRow(`
		SELECT id FROM immortal_apertures WHERE player_id = $1
	`, playerID).Scan(&apertureID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chưa khai mở Tiên Khiếu"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	var hairyManPopulation int
	if err := db.DB.QueryRow(`
		SELECT COALESCE(population, 0) FROM aperture_variant_humans
		WHERE aperture_id = $1 AND race = 'HAIRY_MAN'
	`, apertureID).Scan(&hairyManPopulation); err == sql.ErrNoRows || hairyManPopulation <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần có Mao Dân trong Tiên Khiếu để tự động luyện Cổ"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	// Validate mode.
	switch req.Mode {
	case models.RefineModeFixedCount, models.RefineModeUntilSuccess,
		models.RefineModeUntilMaterial, models.RefineModeMaintainStock:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mode không hợp lệ"})
		return
	}

	// 1. Kiểm tra bí phương hoàn thiện của người chơi (player_recipe_progress)
	var canonicalRecipeID string
	var isUnlocked bool
	var completionRate float64
	err = db.DB.QueryRow(`
		SELECT recipe_id, is_unlocked, completion_rate
		FROM player_recipe_progress
		WHERE player_id = $1 AND (
			recipe_id = $2 OR
			recipe_id = 'recipe_' || $2 OR
			recipe_id LIKE 'recipe_' || $2 || '_%' OR
			recipe_id LIKE $2 || '_%' OR
			('recipe_' || REPLACE($2, 'recipe_', '')) = recipe_id
		)
		ORDER BY completion_rate DESC, is_unlocked DESC
		LIMIT 1
	`, playerID, req.RecipeID).Scan(&canonicalRecipeID, &isUnlocked, &completionRate)
	if err == sql.ErrNoRows || (!isUnlocked && completionRate < 100.0) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa sở hữu bí phương hoàn thiện (100%) cho Cổ trùng này!"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	actualRecipeID := canonicalRecipeID
	if actualRecipeID == "" {
		actualRecipeID = req.RecipeID
	}

	// 2. Kiểm tra nguyên liệu tương ứng trong kho/hành trang người chơi
	var exactRecipeID string
	var reqMatsJSON, reqGusJSON sql.NullString
	err = db.DB.QueryRow(`
		SELECT ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text),
		       required_materials::text, required_gus::text
		FROM refining_recipes
		WHERE ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) = $1
		   OR target_gu_id = $1
		   OR ('recipe_' || target_gu_id) = $1
		   OR ('recipe_' || target_gu_id || '_') LIKE $1 || '%'
		   OR ($1 LIKE 'recipe_%' AND ('recipe_' || target_gu_id || '_') LIKE REPLACE($1, 'recipe_', '') || '%')
		LIMIT 1
	`, actualRecipeID).Scan(&exactRecipeID, &reqMatsJSON, &reqGusJSON)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bí phương không tồn tại"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	actualRecipeID = exactRecipeID
	if reqGusJSON.Valid && reqGusJSON.String != "" && reqGusJSON.String != "{}" && reqGusJSON.String != "[]" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tự động luyện hiện chưa hỗ trợ bí phương cần Cổ trùng làm nguyên liệu"})
		return
	}

	var matSummary string
	if reqMatsJSON.Valid && reqMatsJSON.String != "" && reqMatsJSON.String != "{}" {
		var reqMats map[string]int
		if err := json.Unmarshal([]byte(reqMatsJSON.String), &reqMats); err != nil {
			httperr.Respond(c, httperr.Internal("Dữ liệu bí phương không hợp lệ", err))
			return
		} else if len(reqMats) > 0 {
			var parts []string
			for matID, reqQty := range reqMats {
				var currentQty int
				_ = db.DB.QueryRow(`
					SELECT quantity FROM player_inventory_items
					WHERE player_id = $1 AND item_id = $2
				`, playerID, matID).Scan(&currentQty)

				if currentQty < reqQty {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s trong hành trang (cần %d, hiện có %d)!", matID, reqQty, currentQty)})
					return
				}
				parts = append(parts, fmt.Sprintf("%s x%d", matID, reqQty*req.TargetQuantity))
			}
			matSummary = strings.Join(parts, ", ")
		}
	}
	if matSummary == "" {
		matSummary = "Nguyên liệu theo công thức bí phương"
	}

	id := uuid.New().String()
	_, err = db.DB.Exec(`
		INSERT INTO aperture_refinement_queue (id, aperture_id, recipe_id, mode, target_quantity, priority)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, apertureID, actualRecipeID, req.Mode, req.TargetQuantity, req.Priority)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	modeLabels := map[string]string{
		"FIXED_COUNT":          "Số lượng cố định",
		"MAINTAIN_STOCK":       "Duy trì trong kho",
		"UNTIL_SUCCESS":        "Luyện đến khi thành công",
		"UNTIL_MATERIAL_EMPTY": "Luyện hết nguyên liệu trong kho",
	}

	notify.SendBestEffort(db.DB, playerID, "refine_job_created", "🔨 Tiếp Nhận Lệnh Luyện Cổ Tự Động", fmt.Sprintf(
		"Tộc Mao Dân đã tiếp nhận lệnh luyện Cổ [%s]!\n\n📋 Chi tiết công việc:\n• Chế độ luyện: %s\n• Mục tiêu: %d con\n• Khấu trừ dự kiến: %s\n• Nơi lưu trữ: Kho Tiên Khiếu",
		actualRecipeID, modeLabels[req.Mode], req.TargetQuantity, matSummary,
	), c.Request.Context())

	c.JSON(http.StatusCreated, gin.H{"message": "Đã tạo lệnh luyện cổ tự động", "id": id})
}

// DeleteRefinementJob deletes a refinement job.
func DeleteRefinementJob(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	jobID := c.Param("job_id")

	result, err := db.DB.Exec(`
		DELETE FROM aperture_refinement_queue rq
		USING immortal_apertures a
		WHERE rq.aperture_id = a.id AND rq.id = $1 AND a.player_id = $2
	`, jobID, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy lệnh luyện cổ"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa lệnh luyện cổ"})
}

// ---------- Settlement ----------

// GetSettlements returns all settlements for the player's aperture.
func GetSettlements(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT s.id, s.aperture_id, s.race, COALESCE(s.name, ''), s.settlement_type,
		       s.population_capacity, s.housing, s.food_storage, s.security,
		       COALESCE(s.pos_x, 0), COALESCE(s.pos_y, 0),
		       s.created_at, s.updated_at
		FROM aperture_variant_settlements s
		JOIN immortal_apertures a ON a.id = s.aperture_id
		WHERE a.player_id = $1
		ORDER BY s.race
	`, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer rows.Close()

	var list []models.ApertureVariantSettlement
	for rows.Next() {
		var s models.ApertureVariantSettlement
		if err := rows.Scan(&s.ID, &s.ApertureID, &s.Race, &s.Name, &s.SettlementType,
			&s.PopulationCapacity, &s.Housing, &s.FoodStorage, &s.Security,
			&s.PosX, &s.PosY,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		list = append(list, s)
	}
	if list == nil {
		list = []models.ApertureVariantSettlement{}
	}
	c.JSON(http.StatusOK, gin.H{"settlements": list})
}

// BuildSettlement creates a new settlement for a variant race in the player's aperture.
type CreateSettlementRequest struct {
	Race           string `json:"race" binding:"required"`
	Name           string `json:"name"`
	SettlementType string `json:"settlement_type"` // Always starts at VILLAGE
	PosX           int    `json:"pos_x"`
	PosY           int    `json:"pos_y"`
}

func isSettlementCellOccupied(tx *sql.Tx, apertureID string, x, y int) (bool, error) {
	var hit int
	err := tx.QueryRow(`
		SELECT 1 FROM (
			SELECT x, y FROM aperture_resource_points WHERE aperture_id = $1 AND x = $2 AND y = $3
			UNION ALL
			SELECT x, y FROM aperture_resource_node_cells WHERE aperture_id = $1 AND x = $2 AND y = $3
			UNION ALL
			SELECT pos_x AS x, pos_y AS y FROM aperture_variant_settlements WHERE aperture_id = $1 AND pos_x = $2 AND pos_y = $3
		) t LIMIT 1
	`, apertureID, x, y).Scan(&hit)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func findFirstFreeApertureCell(tx *sql.Tx, apertureID string) (int, int, bool, error) {
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			occ, err := isSettlementCellOccupied(tx, apertureID, x, y)
			if err != nil {
				return 0, 0, false, err
			}
			if !occ {
				return x, y, true, nil
			}
		}
	}
	return 0, 0, false, nil
}

func BuildSettlement(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CreateSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}
	req.Race = normalizeVariantEnum(req.Race)

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	// Get player's aperture ID and state
	var apertureID, apertureState string
	err = tx.QueryRow(`SELECT id, COALESCE(state, 'STABLE') FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apertureID, &apertureState)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy Tiên Khiếu"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	if apertureState == "DESTROYED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Khiếu đã bị phá hủy, không thể xây Thành Trì"})
		return
	}

	// Check if a settlement already exists for this race
	var existingID string
	err = tx.QueryRow(`SELECT id FROM aperture_variant_settlements WHERE aperture_id = $1 AND race = $2`, apertureID, req.Race).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bộ tộc này đã có cơ sở định cư/thành trì trong Tiên Khiếu!"})
		return
	}

	// Sequential start: always build VILLAGE first, costing 500 Immortal Stones.
	const villageBuildCost = 500
	var stones float64
	if err := tx.QueryRow(`SELECT immortal_stones FROM players WHERE id = $1 FOR UPDATE`, playerID).Scan(&stones); err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	if stones < villageBuildCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần %d Tiên Nguyên Thạch để xây Thành Trì VILLAGE", villageBuildCost)})
		return
	}

	posX, posY := req.PosX, req.PosY
	if posX < 0 || posX >= 20 || posY < 0 || posY >= 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tọa độ đặt Thành Trì phải nằm trong lưới 0..19"})
		return
	}
	occupied, err := isSettlementCellOccupied(tx, apertureID, posX, posY)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	if occupied {
		fx, fy, found, fErr := findFirstFreeApertureCell(tx, apertureID)
		if fErr != nil || !found {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Ô bản đồ đã có công trình/điểm tài nguyên và không còn ô trống"})
			return
		}
		posX, posY = fx, fy
	}

	settlementType := "VILLAGE"
	name := req.Name
	if name == "" {
		raceNames := map[string]string{
			"HAIRY_MAN":   "Bản Làng Mao Dân",
			"ROCKMAN":     "Thạch Sấm Động",
			"FEATHERMAN":  "Lông Vũ Trại",
			"SNOWMAN":     "Tuyết Nhân Thôn",
			"INKMAN":      "Mặc Thư Viện",
			"MINIMAN":     "Tiểu Nhân Điền",
			"MERMAN":      "Giao Nhân Thủy Trái",
			"MUSHROOMMAN": "Nấm Nhân Động",
			"DRAGONMAN":   "Long Nhân Thành",
		}
		if defaultName, ok := raceNames[req.Race]; ok {
			name = defaultName
		} else {
			name = "Bản Làng Dị Nhân"
		}
	}

	const capacity = 10000
	const housing = 2000
	const foodStorage = 5000
	const security = 50.0

	if _, err := tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, villageBuildCost, playerID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể khấu trừ Tiên Nguyên Thạch", err))
		return
	}

	var newID string
	err = tx.QueryRow(`
		INSERT INTO aperture_variant_settlements (aperture_id, race, name, settlement_type, population_capacity, housing, food_storage, security, pos_x, pos_y)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, apertureID, req.Race, name, settlementType, capacity, housing, foodStorage, security, posX, posY).Scan(&newID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tạo thành trì", err))
		return
	}

	// Building a settlement creates capacity with 0 free settlers; population is acquired via world/market recruitment.
	_, err = tx.Exec(`
		INSERT INTO aperture_variant_humans (aperture_id, race, population, happiness, oppression, labor_intensity, prosperity, food_stored)
		VALUES ($1, $2, 0, 70.0, 0.0, 'NORMAL', 50.0, 0)
		ON CONFLICT (aperture_id, race) DO NOTHING
	`, apertureID, req.Race)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	notify.SendBestEffort(db.DB, playerID, "settlement_built", "🏰 Xây Dựng Thành Trì Mới Thành Công", fmt.Sprintf(
		"Đã xây dựng [%s] (%s) cho bộ tộc [%s] tại (%d, %d) (tốn %d Tiên Nguyên Thạch). Sức chứa: %d dân. Hãy chiêu mộ/nhập dân vào thành trì!",
		name, settlementType, req.Race, posX, posY, villageBuildCost, capacity,
	), c.Request.Context())

	c.JSON(http.StatusOK, gin.H{
		"message": "Xây dựng cơ sở định cư thành công!",
		"id":      newID,
		"pos_x":   posX,
		"pos_y":   posY,
	})
}

type UpgradeSettlementRequest struct {
	Name               string  `json:"name"`
	SettlementType     string  `json:"settlement_type"` // VILLAGE, TOWN, CITY, FORTRESS
	PopulationCapacity int     `json:"population_capacity" binding:"omitempty,min=0"`
	Housing            int     `json:"housing" binding:"omitempty,min=0"`
	FoodStorage        int     `json:"food_storage" binding:"omitempty,min=0"`
	Security           float64 `json:"security" binding:"omitempty,min=0"`
}

// UpgradeSettlement upgrades a settlement's capacity and type sequentially with rank, area, and stone costs.
func UpgradeSettlement(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	settlementID := c.Param("settlement_id")

	var req UpgradeSettlementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	var currentType, currentName string
	var currentCap, currentHousing, currentFoodStorage int
	var currentSecurity float64
	var areaSize int64
	var playerRank int
	var stones float64
	err = tx.QueryRow(`
		SELECT s.settlement_type, COALESCE(s.name, ''), s.population_capacity, s.housing, s.food_storage, s.security,
		       COALESCE(a.area_size, 0), p.rank, p.immortal_stones
		FROM aperture_variant_settlements s
		JOIN immortal_apertures a ON a.id = s.aperture_id
		JOIN players p ON p.id = a.player_id
		WHERE s.id = $1 AND a.player_id = $2
		FOR UPDATE OF p
	`, settlementID, playerID).Scan(&currentType, &currentName, &currentCap, &currentHousing, &currentFoodStorage, &currentSecurity, &areaSize, &playerRank, &stones)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy settlement"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	var newType string
	switch currentType {
	case "VILLAGE":
		newType = "TOWN"
	case "TOWN":
		newType = "CITY"
	case "CITY":
		newType = "FORTRESS"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thành trì đã đạt cấp tối đa (FORTRESS)"})
		return
	}
	if req.SettlementType != "" && req.SettlementType != newType {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Phải nâng cấp tuần tự từ %s lên %s", currentType, newType)})
		return
	}

	var requiredRank int
	var requiredArea int64
	var upgradeCost float64
	var newCap, newHousing, newFoodStorage int
	var newSecurity float64
	switch newType {
	case "TOWN":
		requiredRank, requiredArea, upgradeCost = 6, 500, 2000
		newCap, newHousing, newFoodStorage, newSecurity = 50000, 10000, 25000, 70.0
	case "CITY":
		requiredRank, requiredArea, upgradeCost = 7, 2000, 8000
		newCap, newHousing, newFoodStorage, newSecurity = 250000, 50000, 100000, 85.0
	case "FORTRESS":
		requiredRank, requiredArea, upgradeCost = 8, 8000, 25000
		newCap, newHousing, newFoodStorage, newSecurity = 1000000, 200000, 500000, 95.0
	}

	if playerRank < requiredRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần đạt Rank %d để nâng cấp lên %s", requiredRank, newType)})
		return
	}
	if areaSize < requiredArea {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Diện tích Tiên Khiếu cần tối thiểu %d mẫu (hiện có %d) để nâng cấp lên %s", requiredArea, areaSize, newType)})
		return
	}
	if stones < upgradeCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần %.0f Tiên Nguyên Thạch để nâng cấp lên %s", upgradeCost, newType)})
		return
	}

	if _, err := tx.Exec(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2`, upgradeCost, playerID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể khấu trừ Tiên Nguyên Thạch", err))
		return
	}

	newName := currentName
	if req.Name != "" {
		newName = req.Name
	}

	_, err = tx.Exec(`
		UPDATE aperture_variant_settlements
		SET settlement_type = $2, name = $3, population_capacity = $4, housing = $5, food_storage = $6, security = $7, updated_at = NOW()
		WHERE id = $1
	`, settlementID, newType, newName, newCap, newHousing, newFoodStorage, newSecurity)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	notify.SendBestEffort(db.DB, playerID, "settlement_upgraded", "🏰 Hoàn Thành Nâng Cấp Thành Trì", fmt.Sprintf(
		"Cơ sở định cư [%s] đã được nâng cấp từ [%s] lên [%s] (tốn %.0f Tiên Nguyên Thạch)!\n• Sức chứa: %d\n• Kho lương thực: %d",
		newName, currentType, newType, upgradeCost, newCap, newFoodStorage,
	), c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật và nâng cấp thành trì thành công"})
}

// ---------- Variant NPC ----------

// GetApertureNPCs returns all variant NPCs for the player's aperture.
func GetApertureNPCs(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT n.id, n.aperture_id, n.race, n.name, n.rank, n.sub_rank,
		       n.path, n.role, n.loyalty, n.talent, n.experience, n.hp, n.max_hp,
		       n.formation_slot, COALESCE(n.dao_marks, 0), COALESCE(n.defense_stationed, false), n.version,
		       CASE n.execution_state WHEN 'TRAINING' THEN 'TRAINING' WHEN 'STORED' THEN 'IDLE' ELSE 'ASSIGNED' END,
		       n.created_at, n.updated_at
		FROM non_player_actors n
		JOIN immortal_apertures a ON a.id = n.aperture_id
		WHERE a.player_id = $1 AND n.actor_kind = 'VARIANT_NPC' AND n.lifecycle_state = 'ACTIVE'
		ORDER BY n.race, n.rank DESC, n.name
	`, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer rows.Close()

	var list []models.ApertureVariantNPC
	for rows.Next() {
		var n models.ApertureVariantNPC
		if err := rows.Scan(&n.ID, &n.ApertureID, &n.Race, &n.Name, &n.Rank, &n.SubRank,
			&n.Path, &n.Role, &n.Loyalty, &n.Talent, &n.Experience, &n.HP, &n.MaxHP,
			&n.FormationSlot, &n.DaoMarks, &n.DefenseStationed, &n.Version, &n.Status,
			&n.CreatedAt, &n.UpdatedAt); err != nil {
			continue
		}
		n.EquippedGu, _ = loadActorGu(db.DB, n.ID)
		list = append(list, n)
	}
	if list == nil {
		list = []models.ApertureVariantNPC{}
	}
	c.JSON(http.StatusOK, gin.H{"npcs": list})
}

type AssignNPCRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=NONE REFINER MINER FARMER STEWARD"`
}

// AssignNPCRole assigns a role to a variant NPC.
func AssignNPCRole(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	npcID := c.Param("npc_id")

	var req AssignNPCRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	// Verify NPC belongs to player's aperture.
	result, err := db.DB.Exec(`
		UPDATE non_player_actors n
		SET role = $3, formation_slot = 0,
		    execution_state = CASE WHEN $3 = 'NONE' THEN 'STORED' ELSE 'WORKING' END,
		    updated_at = NOW(), version = version + 1
		FROM immortal_apertures a
		WHERE n.aperture_id = a.id AND n.id = $1 AND a.player_id = $2
		  AND n.actor_kind = 'VARIANT_NPC' AND n.lifecycle_state = 'ACTIVE'
		  AND n.execution_state IN ('STORED','WORKING')
	`, npcID, playerID, req.Role)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy NPC"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Phân công vai trò thành công"})
}

// DismissNPC dismisses (removes) a variant NPC.
func DismissNPC(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	npcID := c.Param("npc_id")

	tx, err := db.DB.Begin()
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	// Lock the active actor so concurrent dismissals decrement the aggregate once.
	var apertureID, race string
	err = tx.QueryRow(`
		SELECT n.aperture_id, n.race FROM non_player_actors n
		JOIN immortal_apertures a ON a.id = n.aperture_id
		WHERE n.id = $1 AND a.player_id = $2 AND n.actor_kind = 'VARIANT_NPC'
		  AND n.lifecycle_state = 'ACTIVE' FOR UPDATE
	`, npcID, playerID).Scan(&apertureID, &race)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy NPC"})
		return
	}
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	_, err = tx.Exec(`UPDATE non_player_actors
		SET lifecycle_state = 'DISMISSED', execution_state = 'DISMISSED', updated_at = NOW(), version = version + 1
		WHERE id = $1`, npcID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	err = releaseActorGu(tx, npcID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	_, err = tx.Exec(`
		UPDATE aperture_variant_humans
		SET gu_master_count = GREATEST(0, gu_master_count - 1), updated_at = NOW()
		WHERE aperture_id = $1 AND race = $2
	`, apertureID, race)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}

	_ = tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Đã sa thải NPC"})
}
