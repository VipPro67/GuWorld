package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

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
	if player.PrimevalStones < float64(stoneCost) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch (cần %d)", stoneCost)})
		return
	}

	// Deduct cost
	player.CurrentThought -= thoughtCost
	player.PrimevalStones -= float64(stoneCost)

	// 2b. Gu instant-finish: one roll per deduce, best equipped Gu wins.
	// Chance scales with (guRank - targetRank); cost is always deducted.
	bestInstantChance := 0.0
	bestInstantGu := ""
	for _, gu := range models.FilterEquippedActive(player.GuInventory) {
		if gu.Hunger <= 0 {
			continue
		}
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok {
			continue
		}
		mech, ok := models.GuNativeMechanic(spec, "DEDUCTION_INSTANT_FINISH")
		if !ok || mech.Trigger != gameplayeffects.TriggerAlwaysOn {
			continue
		}
		base, _ := mech.Values["base_chance"].(float64)
		step, _ := mech.Values["per_rank_step"].(float64)
		minC, _ := mech.Values["min_chance"].(float64)
		maxC, _ := mech.Values["max_chance"].(float64)
		if base == 0 && step == 0 && minC == 0 && maxC == 0 {
			base, step, maxC = 0.05, 0.02, 0.5
		}
		if maxC <= 0 {
			maxC = 1.0
		}
		chance := gamefx.InstantDeductionChance(gu.Rank, targetRank, base, step, minC, maxC)
		if chance > bestInstantChance {
			bestInstantChance = chance
			bestInstantGu = spec.Name
		}
	}
	guInstantFinish := bestInstantChance > 0 && rand.Float64() < bestInstantChance

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

	realmMultiplier := gamefx.DeductionRealmMultiplier(wisdomRealm, refineRealm, targetRealm)

	wisdomDaoMarks := int64(0)
	if player.DaoMarks != nil {
		wisdomDaoMarks = player.DaoMarks["wisdom"]
	}
	daoMultiplier := 1.0 + math.Sqrt(float64(wisdomDaoMarks))

	rankPenalty := gamefx.RankScale(targetRank)

	// Base improvement for Rank 1 is 10% - 25%
	baseImprovement := 10.0 + rand.Float64()*15.0
	improvement := (baseImprovement * realmMultiplier * daoMultiplier) / rankPenalty

	// Flash of Inspiration chance scales with Wisdom
	inspirationFlashChance := (0.03 * realmMultiplier) / rankPenalty
	inspirationFlash := rand.Float64() < inspirationFlashChance

	var newRate float64
	var message string

	if guInstantFinish {
		improvement = 100.0 - currentRate
		newRate = 100.0
		RecipeSvc.Upsert(tx, &services.RecipeProgress{
			PlayerID:       player.ID.String(),
			RecipeID:       req.RecipeID,
			CompletionRate: 100.0,
			IsUnlocked:     true,
		})
		message = fmt.Sprintf("Cổ trùng [%s] cộng hưởng! Đạo hữu lập tức hoàn thiện 100%% công thức!", bestInstantGu)
	} else if inspirationFlash {
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

	qiGained := 0.0

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

	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, ItemInventoryDomain, RecipesDomain)

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "recipe.deduce",
		EntityType:     "refining_recipe",
		EntityID:       req.RecipeID,
		RequestSummary: fmt.Sprintf("Thôi diễn bí phương %s: %s (Hoàn thiện: %.1f%%)", req.RecipeID, message, newRate),
		Metadata: map[string]interface{}{
			"recipe_id":         req.RecipeID,
			"improvement":       improvement,
			"completion_rate":   newRate,
			"inspiration":       inspirationFlash,
			"gu_instant_finish": guInstantFinish,
			"proc_gu":           bestInstantGu,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message":           message + fmt.Sprintf(" Nhân Khí +%.0f.", qiGained),
		"inspiration":       inspirationFlash,
		"gu_instant_finish": guInstantFinish,
		"proc_gu":           bestInstantGu,
		"improvement":       improvement,
		"completion_rate":   newRate,
	})
}

func calculateRefinementPathGain(targetRank int, refinementRealm int) int {
	return gamefx.CalculateRefinementPathGain(targetRank, refinementRealm)
}
