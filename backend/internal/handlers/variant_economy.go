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

// ---------- Node Worker Assignment ----------

// GetApertureNodeWorkers returns all worker assignments for a given node.
func GetApertureNodeWorkers(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	nodeID := c.Param("id")

	// Verify node belongs to player's aperture.
	var nodeIDCheck string
	err := db.DB.QueryRow(`
		SELECT n.id FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON a.id = n.aperture_id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&nodeIDCheck)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy resource node"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT aw.id, aw.node_id, aw.variant_human_id, vh.race, vh.population, aw.assigned_population, aw.created_at, aw.updated_at
		FROM aperture_node_workers aw
		JOIN aperture_variant_humans vh ON vh.id = aw.variant_human_id
		WHERE aw.node_id = $1
		ORDER BY vh.race
	`, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	type WorkerInfo struct {
		ID                 string `json:"id"`
		NodeID             string `json:"node_id"`
		VariantHumanID     string `json:"variant_human_id"`
		Race               string `json:"race"`
		Population         int    `json:"population"`
		AssignedPopulation int    `json:"assigned_population"`
		CreatedAt          string `json:"created_at"`
		UpdatedAt          string `json:"updated_at"`
	}

	var list []WorkerInfo
	for rows.Next() {
		var w WorkerInfo
		if err := rows.Scan(&w.ID, &w.NodeID, &w.VariantHumanID, &w.Race, &w.Population, &w.AssignedPopulation, &w.CreatedAt, &w.UpdatedAt); err != nil {
			continue
		}
		list = append(list, w)
	}
	if list == nil {
		list = []WorkerInfo{}
	}
	c.JSON(http.StatusOK, gin.H{"workers": list})
}

// AssignWorkerToNode assigns (or updates) a variant human worker to a node.
func AssignWorkerToNode(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	nodeID := c.Param("id")

	var req AssignWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Verify node belongs to player's aperture.
	var apertureID string
	err = tx.QueryRow(`
		SELECT n.aperture_id FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON a.id = n.aperture_id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&apertureID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy resource node"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Verify variant human belongs to same aperture.
	var vhRace string
	var vhPop int
	err = tx.QueryRow(`
		SELECT race, population FROM aperture_variant_humans
		WHERE id = $1 AND aperture_id = $2 FOR UPDATE
	`, req.VariantHumanID, apertureID).Scan(&vhRace, &vhPop)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy dị nhân"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// A population can only work at one node at a time. Exclude this node so an
	// existing assignment can be resized without counting itself twice.
	var assignedElsewhere int
	if err = tx.QueryRow(`
		SELECT COALESCE(SUM(assigned_population), 0)
		FROM aperture_node_workers
		WHERE variant_human_id = $1 AND node_id <> $2
	`, req.VariantHumanID, nodeID).Scan(&assignedElsewhere); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if assignedElsewhere+req.AssignedPopulation > vhPop {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng lao động vượt quá dân số"})
		return
	}

	// Upsert worker assignment.
	_, err = tx.Exec(`
		INSERT INTO aperture_node_workers (node_id, variant_human_id, assigned_population)
		VALUES ($1, $2, $3)
		ON CONFLICT (node_id, variant_human_id)
		DO UPDATE SET assigned_population = $3, updated_at = NOW()
	`, nodeID, req.VariantHumanID, req.AssignedPopulation)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	_ = tx.Commit()

	notify.Send(db.DB, playerID, "worker_assigned", "⛏️ Phân Công Lao Động Dị Nhân", fmt.Sprintf(
		"Đã phân công thành công %d %s vào khai thác/sản xuất tại Node [%s]!\n\n📈 Tác động:\n• Lao động phân công: %d người\n• Chủng tộc: %s",
		req.AssignedPopulation, vhRace, nodeID, req.AssignedPopulation, vhRace,
	))

	c.JSON(http.StatusOK, gin.H{"message": "Phân công lao động thành công"})
}

// RemoveWorkerFromNode removes a worker assignment from a node.
func RemoveWorkerFromNode(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	nodeID := c.Param("id")
	workerID := c.Param("worker_id")

	// Verify node belongs to player's aperture.
	err := db.DB.QueryRow(`
		SELECT n.id FROM aperture_resource_nodes n
		JOIN immortal_apertures a ON a.id = n.aperture_id
		WHERE n.id = $1 AND a.player_id = $2
	`, nodeID, playerID).Scan(&nodeID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy resource node"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	result, err := db.DB.Exec(`
		DELETE FROM aperture_node_workers WHERE id = $1 AND node_id = $2
	`, workerID, nodeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy phân công lao động"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa phân công lao động"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Dữ liệu bí phương không hợp lệ"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	modeLabels := map[string]string{
		"FIXED_COUNT":          "Số lượng cố định",
		"MAINTAIN_STOCK":       "Duy trì trong kho",
		"UNTIL_SUCCESS":        "Luyện đến khi thành công",
		"UNTIL_MATERIAL_EMPTY": "Luyện hết nguyên liệu trong kho",
	}

	notify.Send(db.DB, playerID, "refine_job_created", "🔨 Tiếp Nhận Lệnh Luyện Cổ Tự Động", fmt.Sprintf(
		"Tộc Mao Dân đã tiếp nhận lệnh luyện Cổ [%s]!\n\n📋 Chi tiết công việc:\n• Chế độ luyện: %s\n• Mục tiêu: %d con\n• Khấu trừ dự kiến: %s\n• Nơi lưu trữ: Kho Tiên Khiếu",
		actualRecipeID, modeLabels[req.Mode], req.TargetQuantity, matSummary,
	))

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
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
		       s.created_at, s.updated_at
		FROM aperture_variant_settlements s
		JOIN immortal_apertures a ON a.id = s.aperture_id
		WHERE a.player_id = $1
		ORDER BY s.race
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var list []models.ApertureVariantSettlement
	for rows.Next() {
		var s models.ApertureVariantSettlement
		if err := rows.Scan(&s.ID, &s.ApertureID, &s.Race, &s.Name, &s.SettlementType,
			&s.PopulationCapacity, &s.Housing, &s.FoodStorage, &s.Security,
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
	SettlementType string `json:"settlement_type"` // VILLAGE, TOWN, CITY, FORTRESS
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

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Get player's aperture ID
	var apertureID string
	err = tx.QueryRow(`SELECT id FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apertureID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy Tiên Khiếu"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if a settlement already exists for this race
	var existingID string
	err = tx.QueryRow(`SELECT id FROM aperture_variant_settlements WHERE aperture_id = $1 AND race = $2`, apertureID, req.Race).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bộ tộc này đã có cơ sở định cư/thành trì trong Tiên Khiếu!"})
		return
	}

	settlementType := req.SettlementType
	if settlementType == "" {
		settlementType = "VILLAGE"
	}

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

	capacity := 10000
	housing := 2000
	foodStorage := 5000
	security := 50.0

	switch settlementType {
	case "TOWN":
		capacity = 50000
		housing = 10000
		foodStorage = 25000
		security = 70.0
	case "CITY":
		capacity = 250000
		housing = 50000
		foodStorage = 100000
		security = 85.0
	case "FORTRESS":
		capacity = 1000000
		housing = 200000
		foodStorage = 500000
		security = 95.0
	}

	var newID string
	err = tx.QueryRow(`
		INSERT INTO aperture_variant_settlements (aperture_id, race, name, settlement_type, population_capacity, housing, food_storage, security)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, apertureID, req.Race, name, settlementType, capacity, housing, foodStorage, security).Scan(&newID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo thành trì"})
		return
	}

	// Ensure the settlement always brings an initial group of settlers, including
	// when an empty race row survived an earlier disaster.
	_, err = tx.Exec(`
		INSERT INTO aperture_variant_humans (aperture_id, race, population, happiness, oppression, labor_intensity, prosperity, food_stored)
		VALUES ($1, $2, 500, 70.0, 0.0, 'NORMAL', 50.0, $3)
		ON CONFLICT (aperture_id, race) DO UPDATE
		SET population = GREATEST(aperture_variant_humans.population, 500),
		    food_stored = GREATEST(aperture_variant_humans.food_stored, $3), updated_at = NOW()
	`, apertureID, req.Race, foodStorage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	notify.Send(db.DB, playerID, "settlement_built", "🏰 Xây Dựng Thành Trì Mới Thành Công", fmt.Sprintf(
		"Đã xây dựng thành công pháo đài/định cư [%s] cho bộ tộc [%s]!\n\n📈 Thông số gia tăng:\n• Sức chứa dân số: +%d người\n• Mức độ nhà ở (Housing): +%d điểm\n• Dung lượng kho lương: +%d\n• Chỉ số an ninh: %.0f%%",
		name, req.Race, capacity, housing, foodStorage, security,
	))

	c.JSON(http.StatusOK, gin.H{
		"message": "Xây dựng cơ sở định cư thành công!",
		"id":      newID,
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

// UpgradeSettlement upgrades a settlement's capacity and type.
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Verify settlement belongs to player's aperture.
	var currentType, currentName string
	var currentCap, currentHousing, currentFoodStorage int
	var currentSecurity float64
	err = tx.QueryRow(`
		SELECT s.settlement_type, COALESCE(s.name, ''), s.population_capacity, s.housing, s.food_storage, s.security
		FROM aperture_variant_settlements s
		JOIN immortal_apertures a ON a.id = s.aperture_id
		WHERE s.id = $1 AND a.player_id = $2
	`, settlementID, playerID).Scan(&currentType, &currentName, &currentCap, &currentHousing, &currentFoodStorage, &currentSecurity)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy settlement"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	newType := currentType
	if req.SettlementType != "" {
		newType = req.SettlementType
	} else {
		// Default level upgrade if not provided
		switch currentType {
		case "VILLAGE":
			newType = "TOWN"
		case "TOWN":
			newType = "CITY"
		case "CITY":
			newType = "FORTRESS"
		}
	}

	newCap := currentCap
	newHousing := currentHousing
	newFoodStorage := currentFoodStorage
	newSecurity := currentSecurity

	switch newType {
	case "TOWN":
		newCap = 50000
		newHousing = 10000
		newFoodStorage = 25000
		newSecurity = 70.0
	case "CITY":
		newCap = 250000
		newHousing = 50000
		newFoodStorage = 100000
		newSecurity = 85.0
	case "FORTRESS":
		newCap = 1000000
		newHousing = 200000
		newFoodStorage = 500000
		newSecurity = 95.0
	}

	if req.PopulationCapacity > newCap {
		newCap = req.PopulationCapacity
	}
	if req.Housing > newHousing {
		newHousing = req.Housing
	}
	if req.FoodStorage > newFoodStorage {
		newFoodStorage = req.FoodStorage
	}
	if req.Security > newSecurity {
		newSecurity = req.Security
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	_ = tx.Commit()

	notify.Send(db.DB, playerID, "settlement_upgraded", "🏰 Hoàn Thành Nâng Cấp Thành Trì", fmt.Sprintf(
		"Cơ sở định cư [%s] của bộ tộc đã được nâng cấp từ [%s] lên [%s]!\n\n📈 Chỉ số cập nhật:\n• Tổng sức chứa dân số: %d người (+%d)\n• Tổng nhà ở: %d điểm (+%d)\n• Tổng kho lương thực: %d (+%d)\n• Độ an ninh: %.0f%%",
		newName, currentType, newType,
		newCap, newCap-currentCap,
		newHousing, newHousing-currentHousing,
		newFoodStorage, newFoodStorage-currentFoodStorage,
		newSecurity,
	))

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
		       n.path, n.role, n.loyalty, n.talent, n.status,
		       n.created_at, n.updated_at
		FROM aperture_variant_npcs n
		JOIN immortal_apertures a ON a.id = n.aperture_id
		WHERE a.player_id = $1
		ORDER BY n.race, n.rank DESC, n.name
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer rows.Close()

	var list []models.ApertureVariantNPC
	for rows.Next() {
		var n models.ApertureVariantNPC
		if err := rows.Scan(&n.ID, &n.ApertureID, &n.Race, &n.Name, &n.Rank, &n.SubRank,
			&n.Path, &n.Role, &n.Loyalty, &n.Talent, &n.Status,
			&n.CreatedAt, &n.UpdatedAt); err != nil {
			continue
		}
		list = append(list, n)
	}
	if list == nil {
		list = []models.ApertureVariantNPC{}
	}
	c.JSON(http.StatusOK, gin.H{"npcs": list})
}

type AssignNPCRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=NONE REFINER MINER FARMER SCHOLAR GENERAL GOVERNOR"`
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
		UPDATE aperture_variant_npcs n
		SET role = $3, status = CASE WHEN $3 = 'NONE' THEN 'IDLE' ELSE 'ASSIGNED' END, updated_at = NOW()
		FROM immortal_apertures a
		WHERE n.aperture_id = a.id AND n.id = $1 AND a.player_id = $2
	`, npcID, playerID, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Get NPC info before delete.
	var apertureID, race string
	err = tx.QueryRow(`
		SELECT n.aperture_id, n.race FROM aperture_variant_npcs n
		JOIN immortal_apertures a ON a.id = n.aperture_id
		WHERE n.id = $1 AND a.player_id = $2
	`, npcID, playerID).Scan(&apertureID, &race)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy NPC"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	_, err = tx.Exec(`DELETE FROM aperture_variant_npcs WHERE id = $1`, npcID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	_, err = tx.Exec(`
		UPDATE aperture_variant_humans
		SET gu_master_count = GREATEST(0, gu_master_count - 1), updated_at = NOW()
		WHERE aperture_id = $1 AND race = $2
	`, apertureID, race)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	_ = tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Đã sa thải NPC"})
}
