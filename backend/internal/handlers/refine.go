package handlers

import (
	"database/sql"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type RefineRequest struct {
	RecipeID              string   `json:"recipe_id" binding:"required"`
	SelectedGuInstanceIDs []string `json:"selected_gu_instance_ids"`
}

// GetRecipes returns list of all seeded refining formulas
func GetRecipes(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT id, target_gu_id, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost 
		FROM refining_recipes
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query recipes catalog"})
		return
	}
	defer rows.Close()

	recipes := []models.RefiningRecipe{}
	for rows.Next() {
		var recipe models.RefiningRecipe
		err = rows.Scan(
			&recipe.ID, &recipe.TargetGuID, &recipe.RequiredGus,
			&recipe.RequiredMaterials, &recipe.SuccessRate, &recipe.PrimevalStonesCost,
			&recipe.ImmortalEssenceStonesCost,
		)
		if err != nil {
			continue
		}
		recipes = append(recipes, recipe)
	}

	c.JSON(http.StatusOK, recipes)
}

// RefineGu handles the secure RNG-based craft logic
func RefineGu(c *gin.Context) {
	var req RefineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipe_id is required"})
		return
	}

	// 1. Get auth player ID
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	// 2. Fetch and lock player details
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player status"})
		}
		return
	}

	// 3. Fetch Recipe
	var recipe models.RefiningRecipe
	err = tx.QueryRow(`
		SELECT id, target_gu_id, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost 
		FROM refining_recipes 
		WHERE id = $1`, req.RecipeID).Scan(
		&recipe.ID, &recipe.TargetGuID, &recipe.RequiredGus,
		&recipe.RequiredMaterials, &recipe.SuccessRate, &recipe.PrimevalStonesCost,
		&recipe.ImmortalEssenceStonesCost,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Refining recipe not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch recipe"})
		}
		return
	}

	// 4. Validate Costs (Primeval Stones)
	if player.PrimevalStones < recipe.PrimevalStonesCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough Primeval Stones"})
		return
	}

	// 4b. Validate Costs (Immortal Essence Stones)
	if recipe.ImmortalEssenceStonesCost > 0 && player.ImmortalStones < recipe.ImmortalEssenceStonesCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough Immortal Essence Stones (Tiên Tinh Thạch)"})
		return
	}

	// 5. Validate Costs (Required Materials)
	for matID, reqQty := range recipe.RequiredMaterials {
		currentQty := player.MaterialsInventory[matID]
		if currentQty < reqQty {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough required material items: " + matID})
			return
		}
	}

	// 6. Validate Selected Gu Instance IDs
	totalReqGusCount := 0
	for _, reqQty := range recipe.RequiredGus {
		totalReqGusCount += reqQty
	}
	if len(req.SelectedGuInstanceIDs) != totalReqGusCount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect number of Gu bugs selected for refining"})
		return
	}

	// Count Gu types and verify ownership
	selectedGusCounts := make(map[string]int)
	seenInstances := make(map[string]bool)

	for _, instID := range req.SelectedGuInstanceIDs {
		if seenInstances[instID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate Gu instance selected"})
			return
		}
		seenInstances[instID] = true

		found := false
		for _, item := range player.GuInventory {
			if item.InstanceID == instID {
				if item.Status == "DEAD" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot use a DEAD Gu bug for refining"})
					return
				}
				selectedGusCounts[item.GuID]++
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "Gu bug not found in your inventory: " + instID})
			return
		}
	}

	// Check if selection matches recipe required Gus exactly
	for guID, reqQty := range recipe.RequiredGus {
		if selectedGusCounts[guID] != reqQty {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect types of Gu bugs selected for this recipe"})
			return
		}
	}

	// 7. Consume resources (Stones & Materials & Gus)
	player.PrimevalStones -= recipe.PrimevalStonesCost
	if recipe.ImmortalEssenceStonesCost > 0 {
		player.ImmortalStones -= recipe.ImmortalEssenceStonesCost
	}

	for matID, reqQty := range recipe.RequiredMaterials {
		player.MaterialsInventory[matID] -= reqQty
		if player.MaterialsInventory[matID] <= 0 {
			delete(player.MaterialsInventory, matID)
		}
	}

	// 8. RNG Roll
	roll := rand.Float64()
	success := roll <= recipe.SuccessRate

	var newGuInstance models.GuInstance
	var statusMsg string
	var statusStr string

	if success {
		statusStr = "success"
		statusMsg = "Hợp luyện thành công! Đạo hữu luyện chế được Cổ trùng mới."
		
		// Check if any of the consumed ingredients was a Vital Gu
		hasVitalIngredient := false
		newInventory := models.GuInventory{}
		for _, item := range player.GuInventory {
			if seenInstances[item.InstanceID] {
				if item.IsVital {
					hasVitalIngredient = true
				}
			} else {
				newInventory = append(newInventory, item)
			}
		}
		
		newGuInstance = models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       recipe.TargetGuID,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
			IsVital:    hasVitalIngredient,
		}
		
		newInventory = append(newInventory, newGuInstance)
		player.GuInventory = newInventory
	} else {
		statusStr = "failed"
		statusMsg = "Hợp luyện thất bại! Các nguyên liệu thường đã hóa thành tro bụi."
		
		hasVitalSurvived := false
		newInventory := models.GuInventory{}
		for _, item := range player.GuInventory {
			if seenInstances[item.InstanceID] {
				if item.IsVital {
					// Vital Gu survives in DYING state
					item.Hunger = 1
					item.Status = "DYING"
					item.Equipped = false
					newInventory = append(newInventory, item)
					hasVitalSurvived = true
				}
				// Regular Gus are not appended (perished)
			} else {
				newInventory = append(newInventory, item)
			}
		}
		player.GuInventory = newInventory
		
		if hasVitalSurvived {
			statusMsg = "Hợp luyện thất bại! Các nguyên liệu thường đã hóa thành tro bụi, riêng Cổ Bản Mệnh của đạo hữu vẫn sống sót nhưng rơi vào trạng thái Hấp Hối!"
		}
	}

	// Recalculate equipped_gus list
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	// 9. Update player status in database
	updateQuery := `
		UPDATE players
		SET primeval_stones = $1, immortal_stones = $2, materials_inventory = $3, gu_inventory = $4, equipped_gus = $5
		WHERE id = $6
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.ImmortalStones, player.MaterialsInventory, player.GuInventory, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refining results"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit refining transaction"})
		return
	}

	// Return response
	if success {
		c.JSON(http.StatusOK, gin.H{
			"status":  statusStr,
			"message": statusMsg,
			"player":  player,
			"new_gu":  newGuInstance,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"status":  statusStr,
			"message": statusMsg,
			"player":  player,
		})
	}
}
