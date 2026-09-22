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
	"github.com/vippro67/GuWorld/backend/internal/middleware"
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
	case 2:
		rankPenalty = 2.0
	case 3:
		rankPenalty = 5.0
	case 4:
		rankPenalty = 20.0
	case 5:
		rankPenalty = 100.0
	case 6:
		rankPenalty = 10000.0
	case 7:
		rankPenalty = 1000000.0
	case 8:
		rankPenalty = 100000000.0
	case 9:
		rankPenalty = 10000000000.0
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
			"recipe_id":       req.RecipeID,
			"improvement":     improvement,
			"completion_rate": newRate,
			"inspiration":     inspirationFlash,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
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
