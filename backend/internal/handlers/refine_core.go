package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
	InsightBonus       float64 `json:"insight_bonus"`
	InsightStacks      int     `json:"insight_stacks"`
	FinalRate          float64 `json:"final_rate"`
	RefinementRealm    int     `json:"refinement_realm"`
}

// RefineInsightBonusPerStack is the pity bonus per failed attempt (+1%/stack, no cap).
const RefineInsightBonusPerStack = 0.01

// refineInsightStacks returns active pity stacks for the exact target rank.
func refineInsightStacks(player *models.Player, targetRank int) int {
	if player == nil || targetRank < 1 || targetRank > 9 {
		return 0
	}
	key := gamefx.RefineInsightKey(targetRank)
	for _, st := range player.Statuses.Active() {
		if st.ID == key {
			if st.Value < 0 {
				return 0
			}
			return int(st.Value)
		}
	}
	return 0
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

// collectRefineSupportBonus đã bị loại bỏ cùng bonus Cổ phụ trợ.
// Giữ stub trả về 0 để tương thích ngược (không còn được gọi trong RefineGu).
func collectRefineSupportBonus(player *models.Player) float64 {
	return 0.0
}

// ParseRecipeID parses recipe_GU_ID_RANK_VARIANT into components.
// E.g., recipe_GU_BIG_BELLY_TOAD_1_1 -> target_gu_id = "GU_BIG_BELLY_TOAD", target_rank = 1, variant = 1
func ParseRecipeID(recipeID string) (string, int, int, error) {
	return models.ParseRecipeID(recipeID)
}

// RefineRequest struct stays above; helpers below are shared by the instant
// quick path (POST /api/refine) and the timed refine session flow.

type RefineRequest struct {
	RecipeID              string   `json:"recipe_id" binding:"required"`
	SelectedGuInstanceIDs []string `json:"selected_gu_instance_ids"`
}

// RefineRateSnapshot freezes the success-rate computation at the moment a
// refine session is created (WF-REF-05 §4.1). Resolution later uses these
// values; cultivation/build changes made while waiting do not alter the rate.
type RefineRateSnapshot struct {
	Breakdown            RefineRateBreakdown `json:"breakdown"`
	RawPositiveRate      float64             `json:"raw_positive_rate"`
	StatusPenalty        float64             `json:"status_penalty"`
	RefinementRealm      int                 `json:"refinement_realm"`
	InsightStacks        int                 `json:"insight_stacks"`
	ImmortalPenalty      float64             `json:"immortal_penalty"`
	AccuracyMaxBonus     float64             `json:"accuracy_max_bonus"`
	DeathPenalty         float64             `json:"death_penalty"`
	MasteryMultiplierMax float64             `json:"mastery_multiplier_max"`
	PityEligible         bool                `json:"pity_eligible"`
}

// refineRateInput carries the selection-dependent factors into computeRefineRate.
type refineRateInput struct {
	materialQualityBonus float64
	pityEligible         bool
	requirementsNotMet   bool
}

// Default stage config when refine_stage_templates lacks the rank (WF-REF-06 baseline).
const (
	refineBaselineAccuracyMaxBonus     = 0.08
	refineBaselineDeathPenalty         = 0.25
	refineBaselineMasteryMultiplierMax = 2.0
)

// refineQuerier accepts *sql.Tx, *sql.DB and any read-only wrapper exposing
// QueryRow — used by the session flow so preview can run without a tx.
type refineQuerier interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

// computeRefineRate ports the 10-factor rate formula shared by the quick
// path, session creation and preview. stage may be nil → WF-REF-06 baseline
// values are used. It reads (but never writes) only tx-scoped reference data.
func computeRefineRate(q refineQuerier, player *models.Player, recipe models.RefiningRecipe, stage *models.RefineStageTemplate, input refineRateInput) (RefineRateSnapshot, error) {
	snap := RefineRateSnapshot{
		AccuracyMaxBonus:     0.08,
		DeathPenalty:         0.25,
		MasteryMultiplierMax: 2.0,
	}
	if stage != nil {
		snap.AccuracyMaxBonus = stage.AccuracyMaxBonus
		snap.DeathPenalty = stage.DeathPenalty
		snap.MasteryMultiplierMax = stage.MasteryMultiplierMax
	}

	var pathRaw []byte
	if err := q.QueryRow(`SELECT path FROM gu_dictionary WHERE id = $1 AND rank = $2`, recipe.TargetGuID, recipe.TargetRank).Scan(&pathRaw); err != nil {
		return snap, fmt.Errorf("query target Gu path: %w", err)
	}
	var paths []string
	if err := json.Unmarshal(pathRaw, &paths); err != nil {
		paths = []string{string(pathRaw)}
	}

	// Target path attainment bonus & points
	targetPathMastery := 0.0
	targetPathPoints := 0
	if player.PathMasteries != nil {
		for _, path := range paths {
			cleanPath := strings.TrimSuffix(strings.ToLower(path), " path")
			if points, ok := player.PathMasteries[cleanPath]; ok {
				if points > targetPathPoints {
					targetPathPoints = points
				}
				bonus := gamefx.TargetPathMasteryBonus(gamefx.PathMasteryLevel(points))
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
	refinementDaoBonus := gamefx.RefinementDaoBonus(refinementMarks, targetPathPoints, recipe.TargetRank)
	refinementRealm := gamefx.PathMasteryLevel(refinementPoints)
	refinementMasteryBonus := gamefx.RefinementMasteryBonus(refinementRealm)

	// Luck Bonus
	luckMarks := int64(0)
	if player.DaoMarks != nil {
		luckMarks = player.DaoMarks["luck"]
	}
	fx := gamefx.Get(player)
	luckBonus := gamefx.RefinementLuckBonus(luckMarks, fx.LuckBonus)

	// Environment Bonus
	var regionPathsRaw []byte
	if err := q.QueryRow(`SELECT COALESCE(paths, '[]') FROM regions WHERE id = $1`, player.CurrentRegion).Scan(&regionPathsRaw); err != nil {
		logger.S.Warnf("computeRefineRate: failed to query region paths for region %s: %v", player.CurrentRegion, err)
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

	// Support Bonus: chỉ từ chủng tộc/thể chất dương (bonus Cổ phụ trợ đã bị loại bỏ).
	posRefineBonus := 0.0
	negRefinePenalty := 0.0
	if fx.RefineBonus > 0 {
		posRefineBonus = fx.RefineBonus
	} else if fx.RefineBonus < 0 {
		negRefinePenalty = -fx.RefineBonus
	}
	supportBonus := posRefineBonus

	// Per-rank pity: fail rank R stacks only buff rank R (+1%/stack, no cap).
	insightStacks := refineInsightStacks(player, recipe.TargetRank)
	insightBonus := 0.0
	if input.pityEligible {
		insightBonus = float64(insightStacks) * RefineInsightBonusPerStack
	}
	materialQualityBonus := input.materialQualityBonus

	// Apply Immortal Penalty to flat bonuses for Rank 6+
	immortalPenalty := gamefx.RefinementImmortalPenalty(recipe.TargetRank)
	if immortalPenalty > 1.0 {
		refinementMasteryBonus /= immortalPenalty
		targetPathMastery /= immortalPenalty
		luckBonus /= immortalPenalty
		supportBonus /= immortalPenalty
		envBonus /= immortalPenalty
		materialQualityBonus /= immortalPenalty
		insightBonus /= immortalPenalty
	}

	baseRate := recipe.SuccessRate
	sumPositiveBonus := refinementDaoBonus + refinementMasteryBonus + targetPathMastery + luckBonus + supportBonus + envBonus + materialQualityBonus + insightBonus

	rawPositiveRate := baseRate + sumPositiveBonus
	completionPenalty := 1.0

	// Cap positive rate at 1.0 (100%) BEFORE applying negative debuff penalties
	cappedPositiveRate := rawPositiveRate
	if cappedPositiveRate > 1.0 {
		cappedPositiveRate = 1.0
	}
	finalSuccessRate := cappedPositiveRate - negRefinePenalty
	if finalSuccessRate < 0.0 {
		finalSuccessRate = 0.0
	}
	if input.requirementsNotMet {
		finalSuccessRate = 0.0
	}

	snap.Breakdown = RefineRateBreakdown{
		BaseRate:           baseRate,
		CompletionPenalty:  completionPenalty,
		ComboAccuracy:      !input.requirementsNotMet,
		MaterialQuality:    materialQualityBonus,
		RefinementDaoMarks: refinementDaoBonus,
		RefinementMastery:  refinementMasteryBonus,
		TargetPathMastery:  targetPathMastery,
		LuckBonus:          luckBonus,
		SupportBonus:       supportBonus,
		StatusPenalty:      negRefinePenalty,
		EnvBonus:           envBonus,
		InsightBonus:       insightBonus,
		InsightStacks:      insightStacks,
		FinalRate:          finalSuccessRate,
		RefinementRealm:    refinementRealm,
	}
	snap.RawPositiveRate = rawPositiveRate
	snap.StatusPenalty = negRefinePenalty
	snap.RefinementRealm = refinementRealm
	snap.InsightStacks = insightStacks
	snap.ImmortalPenalty = immortalPenalty
	snap.PityEligible = input.pityEligible && !input.requirementsNotMet
	return snap, nil
}


// fetchRefiningRecipe loads a recipe by its derived recipe_id — memory cache
// first, DB fallback. Used by both the quick path and refine sessions.
func fetchRefiningRecipe(q refineQuerier, recipeID string) (models.RefiningRecipe, error) {
	targetGuID, targetRank, variant, err := ParseRecipeID(recipeID)
	if err != nil {
		return models.RefiningRecipe{}, fmt.Errorf("ID công thức không hợp lệ")
	}
	var recipe models.RefiningRecipe
	if rec, found := models.GetRecipeByTarget(targetGuID, targetRank, variant); found {
		recipe = rec
	} else {
		err = q.QueryRow(`
			SELECT target_gu_id, target_rank, variant, required_gus, required_materials, success_rate, primeval_stones_cost, immortal_essence_stones_cost
			FROM refining_recipes
			WHERE target_gu_id = $1 AND target_rank = $2 AND variant = $3`, targetGuID, targetRank, variant).Scan(
			&recipe.TargetGuID, &recipe.TargetRank, &recipe.Variant, &recipe.RequiredGus,
			&recipe.RequiredMaterials, &recipe.SuccessRate, &recipe.PrimevalStonesCost,
			&recipe.ImmortalEssenceStonesCost,
		)
		if err != nil {
			return models.RefiningRecipe{}, err
		}
	}
	recipe.ID = recipeID
	return recipe, nil
}

// validateRefineSelection checks the selected Gu instance ids against a
// recipe: correct count, no duplicates, owned, not DEAD, matching ranks.
// Returns the per-GuID satisfaction counts, the material-quality bonus, and
// whether the selection fails to fulfil the recipe requirement.
func validateRefineSelection(player *models.Player, recipe *models.RefiningRecipe, selected []string) (map[string]int, float64, bool, error) {
	totalReqGusCount := 0
	for _, reqGu := range recipe.RequiredGus {
		totalReqGusCount += reqGu.Qty
	}
	if len(selected) != totalReqGusCount {
		return nil, 0, false, fmt.Errorf("Incorrect number of Gu bugs selected for refining")
	}

	selectedGusCounts := make(map[string]int)
	seenInstances := make(map[string]bool)
	materialQualityBonus := 0.0

	for _, instID := range selected {
		instID = strings.TrimSpace(instID)
		if seenInstances[instID] {
			return nil, 0, false, fmt.Errorf("Duplicate Gu instance selected")
		}
		seenInstances[instID] = true

		item, _, found := models.FindGuByInstanceID(player.GuInventory, instID)
		if !found {
			return nil, 0, false, fmt.Errorf("Gu bug not found in your inventory: %s", instID)
		}
		if item.Status == "DEAD" {
			return nil, 0, false, fmt.Errorf("Cannot use a DEAD Gu bug for refining")
		}
		if reqGu, ok := recipe.RequiredGus[item.GuID]; ok && (reqGu.Rank == 0 || item.Rank >= reqGu.Rank) {
			selectedGusCounts[item.GuID]++
			if reqGu.Rank > 0 && item.Rank > reqGu.Rank {
				materialQualityBonus += 0.02 * float64(item.Rank-reqGu.Rank)
			}
		}
	}

	guRequirementsNotMet := false
	for guID, reqGu := range recipe.RequiredGus {
		if selectedGusCounts[guID] != reqGu.Qty {
			guRequirementsNotMet = true
			break
		}
	}
	return selectedGusCounts, materialQualityBonus, guRequirementsNotMet, nil
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
	recipeProgresses, err := RecipeSvc.ListByPlayerContext(c.Request.Context(), db.DB, playerID.String())
	if err != nil {
		logger.S.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player recipes"})
		return
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
