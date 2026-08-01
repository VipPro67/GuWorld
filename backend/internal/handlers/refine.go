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
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
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

	// 1. Fetch player's unlocked recipes via RecipeSvc
	recipeProgresses, err := RecipeSvc.ListByPlayer(playerID.String())
	if err != nil {
		logger.S.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player recipes"})
		return
	}

	// If empty, initialize with defaults
	if len(recipeProgresses) == 0 {
		defaults := []services.RecipeProgress{
			{PlayerID: playerID.String(), RecipeID: "recipe_moon_raiment", CompletionRate: 100.0, IsUnlocked: true},
			{PlayerID: playerID.String(), RecipeID: "recipe_white_jade", CompletionRate: 50.0, IsUnlocked: true},
			{PlayerID: playerID.String(), RecipeID: "recipe_four_flavors_liquor", CompletionRate: 30.0, IsUnlocked: true},
		}
		for i := range defaults {
			RecipeSvc.Upsert(nil, &defaults[i])
		}
		recipeProgresses = defaults
	}

	// Create map for lookup
	unlockedMap := make(map[string]float64)
	for _, rp := range recipeProgresses {
		if rp.IsUnlocked {
			unlockedMap[rp.RecipeID] = rp.CompletionRate
		}
	}

	// 2. Query global database or cache
	var masterRecipes []models.RefiningRecipe
	if err := cache.GetJSON("master_refining_recipes", &masterRecipes); err != nil {
		rows, err := db.DB.QueryContext(c.Request.Context(), `
			SELECT id, target_gu_id, target_rank, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost 
			FROM refining_recipes
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query recipes catalog"})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var recipe models.RefiningRecipe
			err = rows.Scan(
				&recipe.ID, &recipe.TargetGuID, &recipe.TargetRank, &recipe.RequiredGus,
				&recipe.RequiredMaterials, &recipe.SuccessRate, &recipe.PrimevalStonesCost,
				&recipe.ImmortalEssenceStonesCost,
			)
			if err == nil {
				masterRecipes = append(masterRecipes, recipe)
			}
		}
		_ = cache.SetJSON("master_refining_recipes", masterRecipes, 1*time.Hour)
	}

	recipes := []models.RefiningRecipe{}
	for _, recipe := range masterRecipes {
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
	recipeProgress, err := RecipeSvc.Get(tx, player.ID.String(), req.RecipeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở công thức này!"})
		return
	}
	unlockedRate = recipeProgress.CompletionRate
	isForcedRefine := unlockedRate < 100.0

	// 3. Fetch Recipe
	var recipe models.RefiningRecipe
	err = tx.QueryRow(`
		SELECT id, target_gu_id, target_rank, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost 
		FROM refining_recipes 
		WHERE id = $1`, req.RecipeID).Scan(
		&recipe.ID, &recipe.TargetGuID, &recipe.TargetRank, &recipe.RequiredGus,
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
		currentQty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), matID, "materials")
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
		InventorySvc.RemoveItem(tx, player.ID.String(), matID, "materials", reqQty)
	}

	// 8. RNG Roll
	var pathRaw []byte
	err = tx.QueryRow(`SELECT path FROM gu_dictionary WHERE id = $1 AND rank = $2`, recipe.TargetGuID, recipe.TargetRank).Scan(&pathRaw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query target Gu path: " + err.Error()})
		return
	}

	var paths []string
	if err := json.Unmarshal(pathRaw, &paths); err != nil {
		paths = []string{string(pathRaw)}
	}

	attainmentBonus := 0.0
	if player.PathMasteries != nil {
		for _, path := range paths {
			cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
			if mLevel, ok := player.PathMasteries[cleanPath]; ok {
				bonus := 0.0
				switch mLevel {
				case "Ordinary":
					bonus = 0.0
				case "Master":
					bonus = 0.05
				case "Grandmaster":
					bonus = 0.10
				case "Great Grandmaster":
					bonus = 0.15
				case "Supreme Grandmaster":
					bonus = 0.20
				}
				if bonus > attainmentBonus {
					attainmentBonus = bonus
				}
			}
		}
	}

	finalSuccessRate := recipe.SuccessRate + attainmentBonus
	hasWisdomGu := false
	for _, gu := range player.GuInventory {
		if gu.GuID == "GU_WISDOM_GU" && gu.Equipped && gu.Status == "ACTIVE" {
			hasWisdomGu = true
			break
		}
	}
	if hasWisdomGu {
		finalSuccessRate += 0.30
	}
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
		// Tiên cổ duy nhất check
		spec, ok := models.GetGu(recipe.TargetGuID, recipe.TargetRank)
		if ok && spec.Rank >= 6 {
			exists, _ := db.CheckImmortalGuExists(tx, recipe.TargetGuID, recipe.TargetRank, req.SelectedGuInstanceIDs)
			if exists {
				success = false
				statusStr = "failed"
				statusMsg = "Tiên cổ là duy nhất, đã tồn tại trong thiên hạ, luyện chế thất bại!"
			}
		}
	}

	if success {
		statusStr = "success"
		statusMsg = "Hợp luyện thành công! Đạo hữu luyện chế được Cổ trùng mới."

		// Check if any of the consumed ingredients was a Vital Gu & fire ON_REFINE_MATERIAL
		hasVitalIngredient := false
		newInventory := models.GuInventory{}
		var refineEventMsg string
		for _, item := range player.GuInventory {
			if seenInstances[item.InstanceID] {
				if item.IsVital {
					hasVitalIngredient = true
				}
				if spec, ok := models.GetGuAny(item.GuID); ok {
					if msg, err := HandleGuEvent(&player, spec, item, "ON_REFINE_MATERIAL"); err == nil && msg != "" {
						refineEventMsg = msg
					}
				}
			} else {
				newInventory = append(newInventory, item)
			}
		}
		if refineEventMsg != "" {
			statusMsg += " " + refineEventMsg
		}

		newGuInstance = models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       recipe.TargetGuID,
			Rank:       recipe.TargetRank,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   false,
			IsVital:    hasVitalIngredient,
		}

		newInventory = append(newInventory, newGuInstance)
		player.GuInventory = newInventory
		if newGuInstance.Rank >= 6 {
			message := fmt.Sprintf("Đạo hữu %s đã luyện chế thành công Cổ Trùng [%s] R%d!", player.Username, newGuInstance.GuID, newGuInstance.Rank)
			_ = SendSystemMessage("world", sql.NullString{}, message)
		}

	} else {
		statusStr = "failed"
		isUniqueImmortalFail := statusMsg != ""
		if !isUniqueImmortalFail {
			statusMsg = "Hợp luyện thất bại! Các nguyên liệu thường đã hóa thành tro bụi."
		}

		hasVitalSurvived := false
		newInventory := models.GuInventory{}
		var refineEventMsg string
		for _, item := range player.GuInventory {
			if seenInstances[item.InstanceID] {
				if item.IsVital {
					item.Hunger = 1
					item.Status = "DYING"
					item.Equipped = false
					newInventory = append(newInventory, item)
					hasVitalSurvived = true
				}
				if spec, ok := models.GetGuAny(item.GuID); ok {
					if msg, err := HandleGuEvent(&player, spec, item, "ON_REFINE_MATERIAL"); err == nil && msg != "" {
						refineEventMsg = msg
					}
				}
			} else {
				newInventory = append(newInventory, item)
			}
		}
		if refineEventMsg != "" {
			statusMsg += " " + refineEventMsg
		}

		if !isUniqueImmortalFail {
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
	}

	// Recalculate equipped_gus list
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	qiGained := AddHumanQi(&player, 10.0)

	// 9. Update player status in database
	if err := db.SavePlayerGus(tx, player.ID.String(), player.GuInventory); err != nil {
		logger.S.Error("DEBUG SavePlayerGus failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu túi đồ: " + err.Error()})
		return
	}
	updateQuery := `
		UPDATE players
		SET primeval_stones = $1, immortal_stones = $2, equipped_gus = $3, hp = $4, max_mana = $5, current_mana = $6, human_qi = $7
		WHERE id = $8
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.ImmortalStones, player.EquippedGus, player.HP, player.MaxMana, player.CurrentMana, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refining results"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit refining transaction"})
		return
	}

	targetName := recipe.TargetGuID
	if targetSpec, ok := models.GetGuAny(recipe.TargetGuID); ok {
		targetName = targetSpec.Name
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "refine.gu", EntityType: "refining_recipe",
		EntityID:       req.RecipeID,
		RequestSummary: fmt.Sprintf("Luyện Cổ: công thức %s -> %s | Thành công: %v | Ép luyện: %v", req.RecipeID, targetName, success, isForcedRefine),
		Metadata: map[string]interface{}{
			"recipe_id":      req.RecipeID,
			"target_gu_id":   recipe.TargetGuID,
			"target_gu_name": targetName,
			"success":        success,
			"is_forced":      isForcedRefine,
		},
		IPAddress: c.ClientIP(), Success: success,
	})
	// Return response
	if success {
		responseWithPlayer(c, http.StatusOK, player, gin.H{
			"status":  statusStr,
			"message": statusMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
			"new_gu":  newGuInstance,
		})
	} else {
		responseWithPlayer(c, http.StatusOK, player, gin.H{
			"status":  statusStr,
			"message": statusMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
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
	recipeProgress, err := RecipeSvc.Get(tx, player.ID.String(), req.RecipeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa sở hữu Tàn Phương của công thức này!"})
		return
	}

	currentRate := recipeProgress.CompletionRate
	if currentRate >= 100.0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Công thức này đã hoàn chỉnh 100%!"})
		return
	}

	// Cost verification — Deduce tiêu hao Thought thay vì Mana
	thoughtCost := 15.0
	stoneCost := 10

	hasWisdomGu := false
	for _, gu := range player.GuInventory {
		if gu.GuID == "GU_WISDOM_GU" && gu.Equipped && gu.Status == "ACTIVE" {
			hasWisdomGu = true
			break
		}
	}
	if hasWisdomGu {
		thoughtCost = 7.5
		stoneCost = 5
	}

	if player.CurrentThought < thoughtCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Niệm Lực (cần %.1f)", thoughtCost)})
		return
	}
	if player.PrimevalStones < stoneCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch (cần %d)", stoneCost)})
		return
	}

	// Deduct cost
	player.CurrentThought -= thoughtCost
	player.PrimevalStones -= stoneCost

	// Roll for deduction outcome
	inspirationFlashChance := 0.03
	if hasWisdomGu {
		inspirationFlashChance += 0.30
	}
	inspirationFlash := rand.Float64() < inspirationFlashChance
	var improvement float64
	var newRate float64
	var message string

	if inspirationFlash {
		improvement = 100.0 - currentRate
		RecipeSvc.Upsert(tx, &services.RecipeProgress{
			PlayerID:       player.ID.String(),
			RecipeID:       req.RecipeID,
			CompletionRate: 100.0,
			IsUnlocked:     true,
		})
		message = "Linh quang chớp lóe! Đạo hữu đột nhiên thông suốt, lập tức hoàn thiện 100% công thức!"
	} else {
		// improve by 10% - 25%
		improvement = 10.0 + rand.Float64()*15.0
		if hasWisdomGu {
			improvement *= 1.3
		}
		// round to 1 decimal place for neatness
		improvement = float64(int(improvement*10)) / 10.0

		newRate = currentRate + improvement
		if newRate >= 100.0 {
			newRate = 100.0
			message = fmt.Sprintf("Thôi diễn thành công! Tăng thêm %.1f%% độ hoàn thiện. Công thức đã được hoàn thiện 100%%!", improvement)
		} else {
			message = fmt.Sprintf("Thôi diễn thành công! Tăng thêm %.1f%% độ hoàn thiện.", improvement)
		}
		RecipeSvc.Upsert(tx, &services.RecipeProgress{
			PlayerID:       player.ID.String(),
			RecipeID:       req.RecipeID,
			CompletionRate: newRate,
			IsUnlocked:     true,
		})
	}

	qiGained := AddHumanQi(&player, 5.0)

	// Save back to database
	updateQuery := `
		UPDATE players
		SET current_thought = $1, primeval_stones = $2, human_qi = $3
		WHERE id = $4
	`
	_, err = tx.Exec(updateQuery, player.CurrentThought, player.PrimevalStones, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save deduction results"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "refine.deduce_recipe", EntityType: "refining_recipe",
		EntityID: req.RecipeID, RequestSummary: fmt.Sprintf("improvement=%.1f%% inspiration=%v", improvement, inspirationFlash),
		IPAddress: c.ClientIP(), Success: true,
	})
	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message":         message + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
		"inspiration":     inspirationFlash,
		"improvement":     improvement,
		"completion_rate": newRate,
	})
}
