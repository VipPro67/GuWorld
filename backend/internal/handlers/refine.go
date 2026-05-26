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
		SELECT id, target_gu_id, required_gus, required_materials, success_rate, primeval_stones_cost 
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
		SELECT id, target_gu_id, required_gus, required_materials, success_rate, primeval_stones_cost 
		FROM refining_recipes 
		WHERE id = $1`, req.RecipeID).Scan(
		&recipe.ID, &recipe.TargetGuID, &recipe.RequiredGus,
		&recipe.RequiredMaterials, &recipe.SuccessRate, &recipe.PrimevalStonesCost,
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

	for matID, reqQty := range recipe.RequiredMaterials {
		player.MaterialsInventory[matID] -= reqQty
		if player.MaterialsInventory[matID] <= 0 {
			delete(player.MaterialsInventory, matID)
		}
	}

	newInventory := models.GuInventory{}
	for _, item := range player.GuInventory {
		if !seenInstances[item.InstanceID] {
			newInventory = append(newInventory, item)
		}
	}
	player.GuInventory = newInventory

	// 8. RNG Roll
	roll := rand.Float64()
	success := roll <= recipe.SuccessRate

	var newGuInstance models.GuInstance
	var statusMsg string
	var statusStr string

	if success {
		statusStr = "success"
		statusMsg = "Refining successful! You obtained a new Gu bug."
		newGuInstance = models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       recipe.TargetGuID,
			Hunger:     100,
			Status:     "ACTIVE",
		}
		player.GuInventory = append(player.GuInventory, newGuInstance)
	} else {
		statusStr = "failed"
		statusMsg = "Refining failed. The Gu bugs perished in the fire."
	}

	// 9. Update player status in database
	updateQuery := `
		UPDATE players
		SET primeval_stones = $1, materials_inventory = $2, gu_inventory = $3
		WHERE id = $4
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.MaterialsInventory, player.GuInventory, player.ID)
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
