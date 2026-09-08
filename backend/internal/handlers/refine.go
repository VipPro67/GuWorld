package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// Points of Cảnh Giới Lưu Phái earned by a successful Luyện Cổ (refine).
const (
	refinePathMasteryGain         = 2
	refinePathMasteryGainImmortal = 10
)

type RefineOutcome string
const (
	OutcomeSuccess      RefineOutcome = "SUCCESS"
	OutcomeFailSafe     RefineOutcome = "FAIL_SAFE"
	OutcomeFailNormal   RefineOutcome = "FAIL_NORMAL"
	OutcomeFailBackfire RefineOutcome = "FAIL_BACKFIRE"
)

type RefineRateBreakdown struct {
	BaseRate           float64 `json:"base_rate"`
	CompletionPenalty  float64 `json:"completion_penalty"`
	ComboAccuracy      bool    `json:"combo_accuracy"`
	MaterialQuality    float64 `json:"material_quality"`
	RefinementDaoMarks float64 `json:"refinement_dao_marks"`
	RefinementMastery  float64 `json:"refinement_mastery"`
	TargetPathMastery  float64 `json:"target_path_mastery"`
	LuckBonus          float64 `json:"luck_bonus"`
	SupportBonus       float64 `json:"support_bonus"`
	StatusPenalty      float64 `json:"status_penalty"`
	EnvBonus           float64 `json:"env_bonus"`
	FinalRate          float64 `json:"final_rate"`
	RefinementRealm    int     `json:"refinement_realm"`
}

func determineRefineOutcome(roll, finalRate float64, refinementRealm, targetRank int) RefineOutcome {
	if roll <= finalRate {
		return OutcomeSuccess
	}
	safeZone := (1.0 - finalRate) * 0.25 * float64(refinementRealm) / 4.0
	if roll <= finalRate+safeZone {
		return OutcomeFailSafe
	}
	backfireThreshold := 0.90
	if targetRank >= 6 {
		backfireThreshold = 0.75
	}
	if roll >= backfireThreshold {
		return OutcomeFailBackfire
	}
	return OutcomeFailNormal
}

func collectRefineSupportBonus(player *models.Player) float64 {
	totalBonus := 0.0
	bestRank := map[string]int{}
	for _, gu := range player.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		if r, ok := bestRank[gu.GuID]; !ok || gu.Rank > r {
			bestRank[gu.GuID] = gu.Rank
		}
	}

	applied := map[string]bool{}
	for _, gu := range player.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		if gu.Rank != bestRank[gu.GuID] || applied[gu.GuID] {
			continue
		}
		applied[gu.GuID] = true

		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok || spec.Effects.Passive == nil {
			continue
		}
		totalBonus += spec.Effects.Passive.SuccessRateBonus
	}
	return totalBonus / 100.0
}

// ParseRecipeID parses recipe_GU_ID_RANK_VARIANT into components.
// E.g., recipe_GU_BIG_BELLY_TOAD_1_1 -> target_gu_id = "GU_BIG_BELLY_TOAD", target_rank = 1, variant = 1
func ParseRecipeID(recipeID string) (string, int, int, error) {
	s := strings.TrimPrefix(recipeID, "recipe_")
	parts := strings.Split(s, "_")
	if len(parts) < 3 {
		return "", 0, 0, fmt.Errorf("invalid recipe id format: %s", recipeID)
	}
	variantStr := parts[len(parts)-1]
	rankStr := parts[len(parts)-2]
	targetGuID := strings.Join(parts[:len(parts)-2], "_")

	var rank, variant int
	if _, err := fmt.Sscanf(rankStr, "%d", &rank); err != nil {
		return "", 0, 0, fmt.Errorf("invalid rank in recipe id: %s", recipeID)
	}
	if _, err := fmt.Sscanf(variantStr, "%d", &variant); err != nil {
		return "", 0, 0, fmt.Errorf("invalid variant in recipe id: %s", recipeID)
	}
	return targetGuID, rank, variant, nil
}

type RefineRequest struct {
	RecipeID              string   `json:"recipe_id" binding:"required"`
	SelectedGuInstanceIDs []string `json:"selected_gu_instance_ids"`
}

// GetRecipes returns list of all seeded refining formulas unlocked by the player
func GetRecipes(c *gin.Context) {
	if c.Query("all") == "true" {
		var masterRecipes []models.RefiningRecipe
		if err := cache.GetJSON("master_refining_recipes", &masterRecipes); err != nil {
			rows, err := db.DB.QueryContext(c.Request.Context(), `
				SELECT target_gu_id, target_rank, variant, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost 
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
					&recipe.TargetGuID, &recipe.TargetRank, &recipe.Variant, &recipe.RequiredGus,
					&recipe.RequiredMaterials, &recipe.SuccessRate, &recipe.PrimevalStonesCost,
					&recipe.ImmortalEssenceStonesCost,
				)
				if err == nil {
					recipe.ID = fmt.Sprintf("recipe_%s_%d_%d", recipe.TargetGuID, recipe.TargetRank, recipe.Variant)
					masterRecipes = append(masterRecipes, recipe)
				}
			}
			_ = cache.SetJSON("master_refining_recipes", masterRecipes, 1*time.Hour)
		}
		c.JSON(http.StatusOK, masterRecipes)
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		// Prefer the Authorization header (current token from localStorage);
		// fall back to the cookie for clients that only send cookies.
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
		} else if tokenStr, err := c.Cookie("gu_auth_token"); err == nil && tokenStr != "" {
			claims, err := auth.ValidateToken(tokenStr)
			if err == nil {
				if parsedID, err := uuid.Parse(claims.PlayerID); err == nil {
					playerID = parsedID
					exists = true
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
			{PlayerID: playerID.String(), RecipeID: "recipe_GU_MOON_RAIMENT_2_1", CompletionRate: 100.0, IsUnlocked: true},
			{PlayerID: playerID.String(), RecipeID: "recipe_GU_WHITE_JADE_2_1", CompletionRate: 100.0, IsUnlocked: true},
			{PlayerID: playerID.String(), RecipeID: "recipe_GU_FOUR_FLAVORS_LIQUOR_2_1", CompletionRate: 100.0, IsUnlocked: true},
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
			SELECT target_gu_id, target_rank, variant, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost 
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
				&recipe.TargetGuID, &recipe.TargetRank, &recipe.Variant, &recipe.RequiredGus,
				&recipe.RequiredMaterials, &recipe.SuccessRate, &recipe.PrimevalStonesCost,
				&recipe.ImmortalEssenceStonesCost,
			)
			if err == nil {
				recipe.ID = fmt.Sprintf("recipe_%s_%d_%d", recipe.TargetGuID, recipe.TargetRank, recipe.Variant)
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
// calculateRefineManaCost determines the amount of Mana/Immortal Essence required based on rank difference
func calculateRefineManaCost(playerRank, targetRank int) (float64, error) {
	if playerRank < 6 && targetRank >= 6 {
		return 0, fmt.Errorf("Phàm nhân không có Tiên Nguyên, không thể luyện Tiên Cổ!")
	}
	if playerRank >= 6 && targetRank < 6 {
		return 0, nil // Immortals refine Mortal Gu for effectively 0 cost
	}
	
	if playerRank < 6 && targetRank < 6 {
		baseCost := 20.0
		if playerRank > targetRank {
			return baseCost / math.Pow(10.0, float64(playerRank-targetRank)), nil
		} else if playerRank < targetRank {
			return baseCost * gamefx.HigherRankManaMultiplier(targetRank, playerRank), nil
		}
		return baseCost, nil
	}
	
	// Immortal refining Immortal Gu
	baseCost := 10.0
	if playerRank > targetRank {
		return baseCost / math.Pow(100.0, float64(playerRank-targetRank)), nil
	} else if playerRank < targetRank {
		return baseCost * gamefx.HigherRankManaMultiplier(targetRank, playerRank), nil
	}
	return baseCost, nil
}

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
	before := append(models.GuInventory(nil), player.GuInventory...)

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
	targetGuID, targetRank, variant, err := ParseRecipeID(req.RecipeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID công thức không hợp lệ"})
		return
	}

	var recipe models.RefiningRecipe
	err = tx.QueryRow(`
		SELECT target_gu_id, target_rank, variant, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost 
		FROM refining_recipes 
		WHERE target_gu_id = $1 AND target_rank = $2 AND variant = $3`, targetGuID, targetRank, variant).Scan(
		&recipe.TargetGuID, &recipe.TargetRank, &recipe.Variant, &recipe.RequiredGus,
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
	recipe.ID = req.RecipeID

	// 3b. Calculate Mana Cost
	manaCost, err := calculateRefineManaCost(player.Rank, recipe.TargetRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 4. Validate Costs (Primeval Stones & Mana)
	if player.CurrentMana < manaCost {
		unit := "Chân Nguyên"
		if player.Rank >= 6 {
			unit = "Tiên Nguyên"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ %s! Cần %.1f", unit, manaCost)})
		return
	}
	
	if player.PrimevalStones < recipe.PrimevalStonesCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough Primeval Stones"})
		return
	}

	// 4b. Validate Costs (Immortal Essence Stones)
	if recipe.ImmortalEssenceStonesCost > 0 && player.ImmortalStones < recipe.ImmortalEssenceStonesCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough Immortal Essence Stones"})
		return
	}

	// 5. Validate Costs (Required Materials)
	for matID, reqQty := range recipe.RequiredMaterials {
		currentQty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), matID)
		if currentQty < reqQty {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough required material items: " + matID})
			return
		}
	}

	// 6. Validate Selected Gu Instance IDs
	totalReqGusCount := 0
	for _, req := range recipe.RequiredGus {
		totalReqGusCount += req.Qty
	}
	if len(req.SelectedGuInstanceIDs) != totalReqGusCount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Incorrect number of Gu bugs selected for refining"})
		return
	}

	// Count Gu types (matching the required rank or higher) and verify ownership
	selectedGusCounts := make(map[string]int)
	seenInstances := make(map[string]bool)
	materialQualityBonus := 0.0

	for _, instID := range req.SelectedGuInstanceIDs {
		if seenInstances[instID] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate Gu instance selected"})
			return
		}
		seenInstances[instID] = true

		item, _, found := models.FindGuByInstanceID(player.GuInventory, instID)
		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "Gu bug not found in your inventory: " + instID})
			return
		}
		if item.Status == "DEAD" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot use a DEAD Gu bug for refining"})
			return
		}
		// Count instances that satisfy the recipe requirement (equal or higher rank).
		if reqGu, ok := recipe.RequiredGus[item.GuID]; ok && (reqGu.Rank == 0 || item.Rank >= reqGu.Rank) {
			selectedGusCounts[item.GuID]++
			if reqGu.Rank > 0 && item.Rank > reqGu.Rank {
				materialQualityBonus += 0.02 * float64(item.Rank-reqGu.Rank)
			}
		}
	}

	// Check if selection matches recipe required Gus.
	guRequirementsNotMet := false
	for guID, reqGu := range recipe.RequiredGus {
		if selectedGusCounts[guID] != reqGu.Qty {
			guRequirementsNotMet = true
			break
		}
	}

	// 7. Consume resources (Stones, Mana, Materials & Gus)
	player.CurrentMana -= manaCost
	player.PrimevalStones -= recipe.PrimevalStonesCost
	if recipe.ImmortalEssenceStonesCost > 0 {
		player.ImmortalStones -= recipe.ImmortalEssenceStonesCost
	}

	// 8. RNG Roll & 10 Factors Calculation
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

	// Calculate target path attainment bonus & points
	targetPathMastery := 0.0
	targetPathPoints := 0
	if player.PathMasteries != nil {
		for _, path := range paths {
			cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
			if points, ok := player.PathMasteries[cleanPath]; ok {
				if points > targetPathPoints {
					targetPathPoints = points
				}
				bonus := 0.0
				switch gamefx.PathMasteryLevel(points) {
				case 1:
					bonus = 0.02
				case 2:
					bonus = 0.05
				case 3:
					bonus = 0.08
				case 4:
					bonus = 0.10
				}
				if bonus > targetPathMastery {
					targetPathMastery = bonus
				}
			}
		}
	}

	// Refinement Dao Marks & Mastery
	refinementMarks := int64(0)
	refinementPoints := 0
	if player.DaoMarks != nil {
		refinementMarks = player.DaoMarks["refinement"]
	}
	if player.PathMasteries != nil {
		refinementPoints = player.PathMasteries["refinement"]
	}
	
	rankScale := 1.0
	switch recipe.TargetRank {
	case 1:
		rankScale = 1.0
	case 2:
		rankScale = 2.0
	case 3:
		rankScale = 5.0
	case 4:
		rankScale = 20.0
	case 5:
		rankScale = 100.0
	case 6:
		rankScale = 10000.0
	case 7:
		rankScale = 1000000.0
	case 8:
		rankScale = 100000000.0
	case 9:
		rankScale = 10000000000.0
	}

	masteryMultiplier := 1.0 + float64(targetPathPoints)
	refinementDaoBonus := (0.00632 * math.Sqrt(float64(refinementMarks)) * masteryMultiplier) / rankScale
	
	refinementRealm := gamefx.PathMasteryLevel(refinementPoints)
	refinementMasteryBonus := 0.0
	switch refinementRealm {
	case 1: refinementMasteryBonus = 0.05
	case 2: refinementMasteryBonus = 0.10
	case 3: refinementMasteryBonus = 0.18
	case 4: refinementMasteryBonus = 0.25
	}

	// Luck Bonus
	luckMarks := int64(0)
	if player.DaoMarks != nil {
		luckMarks = player.DaoMarks["luck"]
	}
	fx := gamefx.Get(&player)
	luckBonus := math.Min(0.10, (float64(luckMarks)*0.001)+fx.LuckBonus/100.0)

	// Environment Bonus
	var regionPathsRaw []byte
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(paths, '[]') FROM regions WHERE id = $1`, player.CurrentRegion).Scan(&regionPathsRaw); err != nil {
		logger.S.Warnf("RefineGu: failed to query region paths for region %s: %v", player.CurrentRegion, err)
	}
	var regionPaths []string
	if len(regionPathsRaw) > 0 {
		_ = json.Unmarshal(regionPathsRaw, &regionPaths)
	}
	envBonus := 0.0
	for _, rp := range regionPaths {
		cleanRP := strings.TrimSuffix(strings.ToLower(rp), " path")
		for _, tp := range paths {
			cleanTP := strings.TrimSuffix(strings.ToLower(tp), " path")
			if cleanRP == cleanTP {
				envBonus = 0.05
				break
			}
		}
		if envBonus > 0 {
			break
		}
	}

	// Support Bonus & Status/Race Refine Effects
	supportGuBonus := collectRefineSupportBonus(&player)
	posRefineBonus := 0.0
	negRefinePenalty := 0.0
	if fx.RefineBonus > 0 {
		posRefineBonus = fx.RefineBonus
	} else if fx.RefineBonus < 0 {
		negRefinePenalty = -fx.RefineBonus
	}
	supportBonus := supportGuBonus + posRefineBonus

	// Apply Immortal Penalty to flat bonuses for Rank 6+
	immortalPenalty := 1.0
	switch recipe.TargetRank {
	case 6: immortalPenalty = 2.0
	case 7: immortalPenalty = 5.0
	case 8: immortalPenalty = 20.0
	case 9: immortalPenalty = 100.0
	}

	if immortalPenalty > 1.0 {
		refinementMasteryBonus /= immortalPenalty
		targetPathMastery /= immortalPenalty
		luckBonus /= immortalPenalty
		supportBonus /= immortalPenalty
		envBonus /= immortalPenalty
		materialQualityBonus /= immortalPenalty
	}

	// Final Rate Calculation
	baseRate := recipe.SuccessRate
	sumPositiveBonus := refinementDaoBonus + refinementMasteryBonus + targetPathMastery + luckBonus + supportBonus + envBonus + materialQualityBonus
	
	rawPositiveRate := baseRate + sumPositiveBonus
	completionPenalty := 1.0
	if isForcedRefine {
		completionPenalty = unlockedRate / 100.0
		rawPositiveRate = rawPositiveRate * completionPenalty
	}
	
	// Cap positive rate at 1.0 (100%) BEFORE applying negative debuff penalties
	cappedPositiveRate := rawPositiveRate
	if cappedPositiveRate > 1.0 {
		cappedPositiveRate = 1.0
	}

	// Subtract debuff penalties (e.g., refine_backfire, refine_fatigue) after capping
	finalSuccessRate := cappedPositiveRate - negRefinePenalty
	if finalSuccessRate < 0.0 {
		finalSuccessRate = 0.0
	}
	if guRequirementsNotMet {
		finalSuccessRate = 0.0
	}

	roll := rand.Float64()
	outcome := determineRefineOutcome(roll, finalSuccessRate, refinementRealm, recipe.TargetRank)
	
	breakdown := RefineRateBreakdown{
		BaseRate:           baseRate,
		CompletionPenalty:  completionPenalty,
		ComboAccuracy:      !guRequirementsNotMet,
		MaterialQuality:    materialQualityBonus,
		RefinementDaoMarks: refinementDaoBonus,
		RefinementMastery:  refinementMasteryBonus,
		TargetPathMastery:  targetPathMastery,
		LuckBonus:          luckBonus,
		SupportBonus:       supportBonus,
		StatusPenalty:      negRefinePenalty,
		EnvBonus:           envBonus,
		FinalRate:          finalSuccessRate,
		RefinementRealm:    refinementRealm,
	}

	var newGuInstance models.GuInstance
	var statusMsg string
	var statusStr string
	var isUniqueImmortalFail bool

	// Tiên cổ duy nhất check
	if recipe.TargetRank >= 6 {
		exists, _ := db.CheckImmortalGuExists(tx, recipe.TargetGuID, recipe.TargetRank, req.SelectedGuInstanceIDs)
		if exists {
			isUniqueImmortalFail = true
			// Tiên cổ duy nhất đã tồn tại: không thể luyện thành công,
			// và cũng không bị phản phệ (backfire) vì thất bại này có nguyên nhân khách quan.
			if outcome != OutcomeFailNormal {
				outcome = OutcomeFailNormal
			}
		}
	}

	// Consume Materials based on outcome
	if outcome == OutcomeFailSafe {
		for matID, reqQty := range recipe.RequiredMaterials {
			deductQty := int64(math.Ceil(float64(reqQty) * 0.5))
			if deductQty > 0 {
				_, cat, _ := InventorySvc.GetQuantityInAnyCategory(tx, player.ID.String(), matID, "materials", "consumables", "other")
				if cat == "" {
					cat = "materials"
				}
				if err := InventorySvc.RemoveItem(tx, player.ID.String(), matID, cat, deductQty); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s để thực hiện hợp luyện", matID)})
					return
				}
			}
		}
	} else {
		for matID, reqQty := range recipe.RequiredMaterials {
			_, cat, _ := InventorySvc.GetQuantityInAnyCategory(tx, player.ID.String(), matID, "materials", "consumables", "other")
			if cat == "" {
				cat = "materials"
			}
			if err := InventorySvc.RemoveItem(tx, player.ID.String(), matID, cat, reqQty); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ nguyên liệu %s để thực hiện hợp luyện", matID)})
				return
			}
		}
	}

	hasVitalIngredient := false
	hasVitalSurvived := false
	var refineEventMsg string

	if outcome == OutcomeSuccess {
		statusStr = "success"
		statusMsg = "Hợp luyện thành công! Đạo hữu luyện chế được Cổ trùng mới."
		newInventory := models.GuInventory{}
		for _, item := range player.GuInventory {
			if seenInstances[item.InstanceID] {
				if item.IsVital {
					hasVitalIngredient = true
				}
				if spec, ok := models.GetGu(item.GuID, item.Rank); ok {
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
		
		// Add refinement path mastery (Luyện Đạo)
		refineGain := calculateRefinementPathGain(recipe.TargetRank, refinementRealm)
		if refineGain > 0 {
			if player.PathMasteries == nil {
				player.PathMasteries = make(models.PathMasteriesMap)
			}
			player.PathMasteries["refinement"] += refineGain
			newRefineLevel := gamefx.PathMasteryLevel(player.PathMasteries["refinement"])
			statusMsg += fmt.Sprintf(" Ngộ Đạo tăng tiến! +%d điểm Chân Ý lưu phái Luyện Đạo (Cảnh giới: %s).", refineGain, gamefx.PathMasteryLevelName(newRefineLevel))
		}
		if newGuInstance.Rank >= 6 {
			guName := newGuInstance.GuID
			if targetSpec, ok := models.GetGu(newGuInstance.GuID, newGuInstance.Rank); ok {
				guName = targetSpec.Name
			}
			message := fmt.Sprintf("Đạo hữu %s đã luyện chế thành công Cổ Trùng [%s] R%d!", player.Username, guName, newGuInstance.Rank)
			_ = SendSystemMessage("world", sql.NullString{}, message)
		}
	} else if outcome == OutcomeFailSafe {
		statusStr = "failed"
		statusMsg = "Luyện cổ thất bại nhưng Cảnh Giới đủ cao giúp bảo toàn Cổ nguyên liệu! Chỉ mất 50% nguyên liệu phụ."
	} else {
		statusStr = "failed"
		if isUniqueImmortalFail {
			statusMsg = "Tiên cổ là duy nhất, đã tồn tại trong thiên hạ, luyện chế thất bại!"
		} else if guRequirementsNotMet {
			statusMsg = "Cổ nguyên liệu không khớp công thức! Các nguyên liệu đã hóa thành tro bụi."
		} else {
			statusMsg = "Hợp luyện thất bại! Các nguyên liệu thường đã hóa thành tro bụi."
		}
		
		newInventory := models.GuInventory{}
		for _, item := range player.GuInventory {
			if seenInstances[item.InstanceID] {
				if item.IsVital {
					item.Hunger = 1
					item.Status = "DYING"
					item.Equipped = false
					newInventory = append(newInventory, item)
					hasVitalSurvived = true
				}
				if spec, ok := models.GetGu(item.GuID, item.Rank); ok {
					if msg, err := HandleGuEvent(&player, spec, item, "ON_REFINE_MATERIAL"); err == nil && msg != "" {
						refineEventMsg = msg
					}
				}
			} else {
				newInventory = append(newInventory, item)
			}
		}
		player.GuInventory = newInventory
		if refineEventMsg != "" {
			statusMsg += " " + refineEventMsg
		}
		
		if outcome == OutcomeFailBackfire {
			hpLoss := gamefx.EffectiveMaxHP(&player) * 0.50
			player.HP -= hpLoss
			if player.HP < 10.0 {
				player.HP = 10.0
			}
			soulLoss := int(math.Ceil(float64(player.MaxSoul) * 0.30))
			player.SoulHP -= soulLoss
			if player.SoulHP < 1 {
				player.SoulHP = 1
			}
			statusMsg += fmt.Sprintf(" Luyện cổ CHÁY NỔ (Backfire)! Phản phệ dữ dội, tổn thất %d Hồn Phách và %.0f HP.", soulLoss, hpLoss)
		} else if isForcedRefine {
			player.HP -= 30.0
			if player.HP < 10.0 {
				player.HP = 10.0
			}
			statusMsg += " Ép Luyện thất bại! Phản phệ chấn thương kinh mạch, tổn thất 30 HP."
		}
		
		if hasVitalSurvived {
			statusMsg += " Riêng Cổ Bản Mệnh vẫn sống sót nhưng rơi vào trạng thái Hấp Hối!"
		}
	}

	statusMsg = fmt.Sprintf("%s (Tỉ lệ: %.1f%%)", statusMsg, finalSuccessRate*100.0)

	// Recalculate equipped_gus list
	player.EquippedGus = []string{}
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
	}

	qiGained := AddHumanQi(&player, 10.0)

	// Apply / escalate refine fatigue and backfire statuses
	// Only trigger fatigue when refining Immortal Gu (Rank >= 6) or Gu with Rank >= player's Rank.
	isTiringRefine := recipe.TargetRank >= 6 || recipe.TargetRank >= player.Rank
	activeStatuses := player.Statuses.Active()
	hasFatigue := activeStatuses.Has("refine_fatigue")
	hasBackfire := activeStatuses.Has("refine_backfire")

	if isTiringRefine || outcome == OutcomeFailBackfire {
		if hasFatigue {
			player.Statuses.Remove("refine_fatigue")
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			statusMsg += " [Luyện Cổ Phản Phệ]: Tỉ lệ Luyện Cổ giảm 50% trong 20 phút do cố luyện khi tâm thần mệt mỏi!"
		} else if hasBackfire {
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			statusMsg += " Trạng thái [Luyện Cổ Phản Phệ] (giảm 50% tỉ lệ) đã được gia hạn thêm 20 phút."
		} else if outcome == OutcomeFailBackfire {
			player.Statuses.Add("refine_backfire", "refine", 20*time.Minute, 0)
			statusMsg += " [Luyện Cổ Phản Phệ]: Thất bại cháy nổ khiến bạn bị Phản Phệ! Tỉ lệ Luyện Cổ giảm 50% trong 20 phút."
		} else {
			player.Statuses.Add("refine_fatigue", "refine", 10*time.Minute, 0)
			statusMsg += " [Tâm Thần Mệt Mỏi]: Tỉ lệ Luyện Cổ lượt tiếp theo giảm 25% trong 10 phút."
		}
	}

	statusesRaw, err := player.Statuses.Value()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize player statuses"})
		return
	}

	// 9. Update player status in database
	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		logger.S.Error("DEBUG SyncPlayerGus failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu túi đồ: " + err.Error()})
		return
	}
	updateQuery := `
		UPDATE players
		SET primeval_stones = $1, immortal_stones = $2, equipped_gus = $3, hp = $4, max_mana = $5, current_mana = $6, human_qi = $7, path_masteries = $8, soul_hp = $9, statuses = $10
		WHERE id = $11
	`
	_, err = tx.Exec(updateQuery, player.PrimevalStones, player.ImmortalStones, player.EquippedGus, player.HP, player.MaxMana, player.CurrentMana, player.HumanQi, player.PathMasteries, player.SoulHP, statusesRaw, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save refining results"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit refining transaction"})
		return
	}

	targetName := recipe.TargetGuID
	if targetSpec, ok := models.GetGu(recipe.TargetGuID, recipe.TargetRank); ok {
		targetName = targetSpec.Name
	}
	
	isSuccess := outcome == OutcomeSuccess

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "refine.gu", EntityType: "refining_recipe",
		EntityID:       req.RecipeID,
		RequestSummary: fmt.Sprintf("Luyện Cổ: công thức %s -> %s | Thành công: %v | Ép luyện: %v | Outcome: %s", req.RecipeID, targetName, isSuccess, isForcedRefine, outcome),
		Metadata: map[string]interface{}{
			"recipe_id":      req.RecipeID,
			"target_gu_id":   recipe.TargetGuID,
			"target_gu_name": targetName,
			"success":        isSuccess,
			"outcome":        outcome,
			"is_forced":      isForcedRefine,
		},
		IPAddress: c.ClientIP(), Success: isSuccess,
	})

	notifyTitle := "Luyện Cổ Thất Bại"
	if isSuccess {
		notifyTitle = "Luyện Cổ Thành Công"
	}
	notify.Send(db.DB, player.ID, "SYSTEM", notifyTitle, statusMsg)

	// Return response
	if isSuccess {
		responseWithPlayer(c, http.StatusOK, player, gin.H{
			"status":         statusStr,
			"message":        statusMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
			"new_gu":         newGuInstance,
			"rate_breakdown": breakdown,
		})
	} else {
		responseWithPlayer(c, http.StatusOK, player, gin.H{
			"status":         statusStr,
			"message":        statusMsg + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
			"rate_breakdown": breakdown,
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

	// 1. Get Gu Rank and Path
	targetGuID, targetRank, _, err := ParseRecipeID(req.RecipeID)
	var targetPath string
	if err == nil {
		_ = tx.QueryRow(`SELECT path FROM gu_dictionary WHERE id = $1 AND rank = $2`, targetGuID, targetRank).Scan(&targetPath)
	} else {
		targetRank = 1 // Fallback just in case
	}

	// 2. Cost verification
	thoughtCost := 15.0
	stoneCost := int64(10)

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

	// 3. Deduction Calculation (Wisdom + Refinement + Target Path)
	getPathPoints := func(p string) int {
		if player.PathMasteries == nil {
			return 0
		}
		clean := strings.TrimSuffix(strings.ToLower(strings.Trim(p, " []\"")), " path")
		if pts, ok := player.PathMasteries[clean]; ok {
			return pts
		}
		if pts, ok := player.PathMasteries[p]; ok {
			return pts
		}
		return 0
	}

	wisdomPts := getPathPoints("wisdom")
	refinePts := getPathPoints("refinement")
	targetPts := getPathPoints(targetPath)

	wisdomRealm := gamefx.PathMasteryLevel(wisdomPts)
	refineRealm := gamefx.PathMasteryLevel(refinePts)
	targetRealm := gamefx.PathMasteryLevel(targetPts)

	realmMultiplier := 1.0 + float64(wisdomRealm)*3.0 + float64(refineRealm)*2.0 + float64(targetRealm)*1.0
	
	wisdomDaoMarks := int64(0)
	if player.DaoMarks != nil {
		wisdomDaoMarks = player.DaoMarks["wisdom"]
	}
	daoMultiplier := 1.0 + math.Sqrt(float64(wisdomDaoMarks))

	rankPenalty := 1.0
	switch targetRank {
	case 2: rankPenalty = 2.0
	case 3: rankPenalty = 5.0
	case 4: rankPenalty = 20.0
	case 5: rankPenalty = 100.0
	case 6: rankPenalty = 10000.0
	case 7: rankPenalty = 1000000.0
	case 8: rankPenalty = 100000000.0
	case 9: rankPenalty = 10000000000.0
	}

	// Base improvement for Rank 1 is 10% - 25%
	baseImprovement := 10.0 + rand.Float64()*15.0
	improvement := (baseImprovement * realmMultiplier * daoMultiplier) / rankPenalty

	// Flash of Inspiration chance scales with Wisdom
	inspirationFlashChance := (0.03 * realmMultiplier) / rankPenalty
	inspirationFlash := rand.Float64() < inspirationFlashChance

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
		// round to 4 decimal places for precision at high ranks
		improvement = math.Round(improvement*10000) / 10000.0

		newRate = currentRate + improvement
		if newRate >= 100.0 {
			newRate = 100.0
			message = fmt.Sprintf("Thôi diễn thành công! Tăng thêm %g%% độ hoàn thiện. Công thức đã được hoàn thiện 100%%!", improvement)
		} else {
			message = fmt.Sprintf("Thôi diễn thành công! Tăng thêm %g%% độ hoàn thiện.", improvement)
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

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message":         message + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
		"inspiration":     inspirationFlash,
		"improvement":     improvement,
		"completion_rate": newRate,
	})
}

func calculateRefinementPathGain(targetRank int, refinementRealm int) int {
	baseGain := 0
	switch targetRank {
	case 1:
		baseGain = 2
	case 2:
		baseGain = 5
	case 3:
		baseGain = 15
	case 4:
		baseGain = 40
	case 5:
		baseGain = 100
	case 6:
		baseGain = 500
	case 7:
		baseGain = 2500
	case 8:
		baseGain = 15000
	case 9:
		baseGain = 50000
	default:
		baseGain = 0
	}

	return gamefx.CalculatePathMasteryGain(targetRank, refinementRealm, baseGain)
}
