package handlers

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// ---------- Request types ----------

type AssignWorkerRequest struct {
	VariantHumanID    string `json:"variant_human_id" binding:"required"`
	AssignedPopulation int   `json:"assigned_population" binding:"required,min=0"`
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
		ID                string  `json:"id"`
		NodeID            string  `json:"node_id"`
		VariantHumanID    string  `json:"variant_human_id"`
		Race              string  `json:"race"`
		Population        int     `json:"population"`
		AssignedPopulation int    `json:"assigned_population"`
		CreatedAt         string  `json:"created_at"`
		UpdatedAt         string  `json:"updated_at"`
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
		WHERE id = $1 AND aperture_id = $2
	`, req.VariantHumanID, apertureID).Scan(&vhRace, &vhPop)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy dị nhân"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Validate assigned population doesn't exceed total population.
	if req.AssignedPopulation > vhPop {
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
		       rq.current_quantity, rq.priority, rq.enabled, rq.created_at, rq.updated_at
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
			&j.CurrentQuantity, &j.Priority, &j.Enabled, &j.CreatedAt, &j.UpdatedAt); err != nil {
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

	// Validate mode.
	switch req.Mode {
	case models.RefineModeFixedCount, models.RefineModeUntilSuccess,
		models.RefineModeUntilMaterial, models.RefineModeMaintainStock:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mode không hợp lệ"})
		return
	}

	// Validate recipe exists in registry.
	if _, ok := models.ItemRegistry[req.RecipeID]; !ok {
		// Recipe may also be a Gu ID — skip validation if registry doesn't have it.
		// Workers will skip invalid recipes at tick time.
	}

	id := uuid.New().String()
	_, err = db.DB.Exec(`
		INSERT INTO aperture_refinement_queue (id, aperture_id, recipe_id, mode, target_quantity, priority)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, id, apertureID, req.RecipeID, req.Mode, req.TargetQuantity, req.Priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

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

type UpgradeSettlementRequest struct {
	Name               string `json:"name"`
	PopulationCapacity int    `json:"population_capacity" binding:"omitempty,min=0"`
	Housing            int    `json:"housing" binding:"omitempty,min=0"`
	FoodStorage        int    `json:"food_storage" binding:"omitempty,min=0"`
}

// UpgradeSettlement upgrades a settlement's capacity.
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
	var currentCap, currentHousing, currentFoodStorage int
	err = tx.QueryRow(`
		SELECT s.population_capacity, s.housing, s.food_storage
		FROM aperture_variant_settlements s
		JOIN immortal_apertures a ON a.id = s.aperture_id
		WHERE s.id = $1 AND a.player_id = $2
	`, settlementID, playerID).Scan(&currentCap, &currentHousing, &currentFoodStorage)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy settlement"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Apply upgrades (only increase, never decrease).
	newCap := currentCap
	if req.PopulationCapacity > currentCap {
		newCap = req.PopulationCapacity
	}
	newHousing := currentHousing
	if req.Housing > currentHousing {
		newHousing = req.Housing
	}
	newFoodStorage := currentFoodStorage
	if req.FoodStorage > currentFoodStorage {
		newFoodStorage = req.FoodStorage
	}

	nameUpdate := ""
	if req.Name != "" {
		nameUpdate = req.Name
	}

	if nameUpdate != "" {
		_, err = tx.Exec(`
			UPDATE aperture_variant_settlements
			SET name = $3, population_capacity = $4, housing = $5, food_storage = $6, updated_at = NOW()
			WHERE id = $1
		`, settlementID, playerID, nameUpdate, newCap, newHousing, newFoodStorage)
	} else {
		_, err = tx.Exec(`
			UPDATE aperture_variant_settlements
			SET population_capacity = $3, housing = $4, food_storage = $5, updated_at = NOW()
			WHERE id = $1
		`, settlementID, newCap, newHousing, newFoodStorage)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	_ = tx.Commit()
	c.JSON(http.StatusOK, gin.H{"message": "Cập nhật settlement thành công"})
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
