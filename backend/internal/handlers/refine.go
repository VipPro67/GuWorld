package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type RefineRequest struct {
	RecipeID              string   `json:"recipe_id" binding:"required"`
	SelectedGuInstanceIDs []string `json:"selected_gu_instance_ids"`
}

// GetRecipes returns list of all seeded refining formulas unlocked by the player
func GetRecipes(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		tokenStr, err := c.Cookie("gu_auth_token")
		if err == nil && tokenStr != "" {
			claims, err := auth.ValidateToken(tokenStr)
			if err == nil {
				if parsedID, err := uuid.Parse(claims.PlayerID); err == nil {
					playerID = parsedID
					exists = true
				}
			}
		} else {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
					claims, err := auth.ValidateToken(parts[1])
					if err == nil {
						if parsedID, err := uuid.Parse(claims.PlayerID); err == nil {
							playerID = parsedID
							exists = true
						}
					}
				}
			}
		}
	}

	if !exists {
		// Return empty list of recipes for unauthenticated users, preventing 401
		c.JSON(http.StatusOK, []models.RefiningRecipe{})
		return
	}

	// 1. Fetch player's unlocked recipes
	var unlockedJSON []byte
	err := db.DB.QueryRow(`SELECT unlocked_recipes FROM players WHERE id = $1`, playerID).Scan(&unlockedJSON)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player recipes"})
		return
	}

	var unlockedList models.UnlockedRecipesList
	if len(unlockedJSON) > 0 {
		_ = json.Unmarshal(unlockedJSON, &unlockedList)
	}

	// If empty, initialize with defaults
	if len(unlockedList) == 0 {
		unlockedList = models.UnlockedRecipesList{
			{RecipeID: "recipe_moon_raiment", CompletionRate: 100.0},
			{RecipeID: "recipe_white_jade", CompletionRate: 50.0},
			{RecipeID: "recipe_four_flavors_liquor", CompletionRate: 30.0},
		}
		// Save to DB
		_, _ = db.DB.Exec(`UPDATE players SET unlocked_recipes = $1 WHERE id = $2`, unlockedList, playerID)
	}

	// Create map for lookup
	unlockedMap := make(map[string]float64)
	for _, ur := range unlockedList {
		unlockedMap[ur.RecipeID] = ur.CompletionRate
	}

	// 2. Query global database
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

		// Only include if unlocked by the player
		if rate, ok := unlockedMap[recipe.ID]; ok {
			recipe.CompletionRate = rate
			recipes = append(recipes, recipe)
		}
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
	}

	// 2b. Verify player has unlocked the recipe
	var unlockedRate float64 = 0.0
	unlockedFound := false
	for _, ur := range player.UnlockedRecipes {
		if ur.RecipeID == req.RecipeID {
			unlockedRate = ur.CompletionRate
			unlockedFound = true
			break
		}
	}
	if !unlockedFound {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở công thức này!"})
		return
	}
	isForcedRefine := unlockedRate < 100.0

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
	var targetGuPath string
	err = tx.QueryRow(`SELECT path FROM gu_dictionary WHERE id = $1`, recipe.TargetGuID).Scan(&targetGuPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query target Gu path: " + err.Error()})
		return
	}

	cleanPath := strings.TrimSuffix(strings.ToLower(targetGuPath), " path")
	attainmentBonus := 0.0
	if player.PathMasteries != nil {
		if mLevel, ok := player.PathMasteries[cleanPath]; ok {
			switch mLevel {
			case "Ordinary":
				attainmentBonus = 0.0
			case "Master":
				attainmentBonus = 0.05
			case "Grandmaster":
				attainmentBonus = 0.10
			case "Great Grandmaster":
				attainmentBonus = 0.15
			case "Supreme Grandmaster":
				attainmentBonus = 0.25
			}
		}
	}

	finalSuccessRate := recipe.SuccessRate + attainmentBonus
	if isForcedRefine {
		finalSuccessRate = finalSuccessRate * (unlockedRate / 100.0)
	}
	if finalSuccessRate > 1.0 {
		finalSuccessRate = 1.0
	}

	roll := rand.Float64()
	success := roll <= finalSuccessRate

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
		
		if isForcedRefine {
			player.HP -= 30.0
			if player.HP < 10.0 {
				player.HP = 10.0
			}
			nolee := rand.Float64() < 0.20
			if nolee {
				player.MaxMana = player.MaxMana * 0.95
				if player.CurrentMana > player.MaxMana {
					player.CurrentMana = player.MaxMana
				}
				statusMsg = "Ép Luyện thất bại! Đạo hữu bị phản phệ dữ dội nổ Không Khiếu, tối đa Chân Nguyên bị suy giảm vĩnh viễn 5%!"
			} else {
				statusMsg = "Ép Luyện thất bại! Phản phệ chấn thương kinh mạch, tổn thất 30 HP."
			}
			if hasVitalSurvived {
				statusMsg += " Riêng Cổ Bản Mệnh vẫn sống sót nhưng rơi vào trạng thái Hấp Hối!"
			}
		} else if hasVitalSurvived {
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
		SET primeval_stones = $1, immortal_stones = $2, materials_inventory = $3, gu_inventory = $4, equipped_gus = $5, hp = $6, max_mana = $7, current_mana = $8, unlocked_recipes = $9
		WHERE id = $10
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.ImmortalStones, player.MaterialsInventory, player.GuInventory, player.EquippedGus, player.HP, player.MaxMana, player.CurrentMana, player.UnlockedRecipes, player.ID)
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

type DeduceRecipeRequest struct {
	RecipeID string `json:"recipe_id" binding:"required"`
}

// DeduceRecipe allows players to spend Mana (Chân Nguyên) and Primeval Stones to improve recipe completion rate
func DeduceRecipe(c *gin.Context) {
	var req DeduceRecipeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipe_id is required"})
		return
	}

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

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lock player status"})
		}
		return
	}

	// Find the recipe in player's unlocked list
	foundIdx := -1
	for i, ur := range player.UnlockedRecipes {
		if ur.RecipeID == req.RecipeID {
			foundIdx = i
			break
		}
	}

	if foundIdx == -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa sở hữu Tàn Phương của công thức này!"})
		return
	}

	currentRate := player.UnlockedRecipes[foundIdx].CompletionRate
	if currentRate >= 100.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Công thức này đã hoàn chỉnh 100%!"})
		return
	}

	// Cost verification
	manaCost := 15.0
	stoneCost := 10

	if player.CurrentMana < manaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Chân Nguyên (cần 15)"})
		return
	}
	if player.PrimevalStones < stoneCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch (cần 10)"})
		return
	}

	// Deduct cost
	player.CurrentMana -= manaCost
	player.PrimevalStones -= stoneCost

	// Roll for deduction outcome
	inspirationFlash := rand.Float64() < 0.03
	var improvement float64
	var message string

	if inspirationFlash {
		improvement = 100.0 - currentRate
		player.UnlockedRecipes[foundIdx].CompletionRate = 100.0
		message = "Linh quang chớp lóe! Đạo hữu đột nhiên thông suốt, lập tức hoàn thiện 100% công thức!"
	} else {
		// improve by 10% - 25%
		improvement = 10.0 + rand.Float64()*15.0
		// round to 1 decimal place for neatness
		improvement = float64(int(improvement*10)) / 10.0

		newRate := currentRate + improvement
		if newRate >= 100.0 {
			newRate = 100.0
			message = fmt.Sprintf("Thôi diễn thành công! Tăng thêm %.1f%% độ hoàn thiện. Công thức đã được hoàn thiện 100%%!", improvement)
		} else {
			message = fmt.Sprintf("Thôi diễn thành công! Tăng thêm %.1f%% độ hoàn thiện.", improvement)
		}
		player.UnlockedRecipes[foundIdx].CompletionRate = newRate
	}

	// Save back to database
	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, unlocked_recipes = $3
		WHERE id = $4
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.UnlockedRecipes, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save deduction results"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         message,
		"inspiration":     inspirationFlash,
		"improvement":     improvement,
		"completion_rate": player.UnlockedRecipes[foundIdx].CompletionRate,
		"player":          player,
	})
}

