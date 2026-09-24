package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

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
	safeZone := (1.0 - finalRate) * gamefx.RefineSafeZoneFactor * float64(refinementRealm) / 4.0
	if roll <= finalRate+safeZone {
		return OutcomeFailSafe
	}
	backfireThreshold := gamefx.RefineBackfireThresholdMortal
	if targetRank >= 6 {
		backfireThreshold = gamefx.RefineBackfireThresholdImmortal
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
		masterRecipes, err := loadMasterRecipes()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query recipes catalog"})
			return
		}
		writeRecipeCatalogResponse(c, masterRecipes)
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
		writeRecipeCatalogResponse(c, []models.RefiningRecipe{})
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
	masterRecipes, err := loadMasterRecipes()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query recipes catalog"})
		return
	}

	recipes := []models.RefiningRecipe{}
	for _, recipe := range masterRecipes {
		// Only include if unlocked by the player
		if rate, ok := unlockedMap[recipe.ID]; ok {
			recipe.CompletionRate = rate
			recipes = append(recipes, recipe)
		}
	}

	writeRecipeCatalogResponse(c, recipes)
}

func writeRecipeCatalogResponse(c *gin.Context, recipes []models.RefiningRecipe) {
	revision := currentRecipeCatalogRevision()
	writeRecipeCatalogResponseAtRevision(c, recipes, revision)
}

func writeRecipeCatalogResponseAtRevision(c *gin.Context, recipes []models.RefiningRecipe, revision uint64) {
	etag := recipeResponseETag(recipes, revision)
	c.Header("ETag", etag)
	c.Header("X-Catalog-Revision", fmt.Sprint(revision))
	c.Header("Cache-Control", "private, no-cache")
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
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
