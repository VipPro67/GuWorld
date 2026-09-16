package handlers

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/internal/util"
)

type ExploreResponse struct {
	Player       models.Player `json:"player"`
	RewardStones int64         `json:"reward_stones"`
	Log          string        `json:"log"`
}

// Explore handles deducting mana, giving player primeval stones, and drops random food materials
func Explore(c *gin.Context) {
	idStr := c.Param("id")
	playerID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid player UUID"})
		return
	}

	// AUTH SECURITY CHECK: Verify token matches the requested resource
	authID, exists := middleware.GetPlayerID(c)
	if !exists || authID != playerID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: You do not own this cultivation Aperture"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Evaluate latest mana status first
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		}
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	// Statuses that block exploration (e.g. Trọng Thương).
	if gamefx.HasBlockedAction(&player, gamefx.ActionExplore) {
		c.JSON(http.StatusForbidden, gin.H{"error": statusBlockedMessage(&player, gamefx.ActionExplore)})
		return
	}

	// 2. Check energy cost. Mortals spend Chân Nguyên; Immortals spend a smaller Tiên Nguyên travel upkeep.
	energyCost := 10.0
	if util.IsImmortalPlayer(&player) {
		energyCost = 0.5
	}
	if player.CurrentMana < energyCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}

	// 3. Deduct energy and roll reward stones
	player.CurrentMana -= energyCost
	rewardStones := int64(1 + rand.Intn(3)) // 1 to 3 primeval stones
	player.PrimevalStones += float64(rewardStones)
	player.LastManaUpdate = time.Now()

	// 4. Roll random material drop for Phase 2 & 3
	rollMat := rand.Intn(100)
	var droppedMaterial string
	var droppedMaterialName string

	if rollMat < 20 {
		droppedMaterial = "moon_orchid_petal"
		droppedMaterialName = "Moon Orchid Petal (Cánh hoa Nguyệt Lan)"
	} else if rollMat < 35 {
		droppedMaterial = "green_bamboo_wine"
		droppedMaterialName = "Green Bamboo Wine (Rượu Thanh Trúc)"
	} else if rollMat < 50 {
		droppedMaterial = "jade_stone"
		droppedMaterialName = "Jade Stone (Ngọc Thạch)"
	} else if rollMat < 65 {
		droppedMaterial = "boar_meat"
		droppedMaterialName = "Boar Meat (Thịt Lợn Rừng)"
	} else if rollMat < 80 {
		droppedMaterial = "wild_boar_tusk"
		droppedMaterialName = "Wild Boar Tusk (Nanh Lợn Rừng)"
	} else {
		// Roll a random wine type
		wines := []string{"sour_wine", "sweet_wine", "bitter_wine", "spicy_wine"}
		wineNames := []string{"Sour Wine (Rượu Chua)", "Sweet Wine (Rượu Ngọt)", "Bitter Wine (Rượu Đắng)", "Spicy Wine (Rượu Cay)"}
		idx := rand.Intn(len(wines))
		droppedMaterial = wines[idx]
		droppedMaterialName = wineNames[idx]
	}

	inventoryFullMsg := ""

	if droppedMaterial != "" {
		if HasInventorySpace(&player, 1) {
			InventorySvc.AddItem(tx, player.ID.String(), droppedMaterial, "materials", 1)
		} else {
			droppedMaterial = ""
			inventoryFullMsg = " [Hành lý đầy, vật phẩm bị rơi mất!]"
		}
	}

	// Roll 10% chance to capture wild Gu bugs
	rollGu := rand.Intn(100)
	var capturedGuID string
	var capturedGuName string
	if rollGu < 10 { // 10% chance overall
		subRoll := rand.Intn(10) // 0 to 9
		if subRoll < 5 {         // 5% absolute chance
			capturedGuID = "GU_JADE_SKIN"
			capturedGuName = "Ngọc Bì Cổ (Jade Skin Gu)"
		} else if subRoll < 8 { // 3% absolute chance
			capturedGuID = "GU_LIQUOR_WORM"
			capturedGuName = "Tửu Trùng (Liquor Worm)"
		} else { // 2% absolute chance
			capturedGuID = "GU_WHITE_BOAR"
			capturedGuName = "Bạch Thỉ Cổ (White Boar Gu)"
		}
	}

	if capturedGuID != "" {
		if HasInventorySpace(&player, 1) {
			guSpec, _ := models.GetGuAny(capturedGuID)
			newGu := models.GuInstance{
				InstanceID: uuid.New().String(),
				GuID:       capturedGuID,
				Rank:       guSpec.Rank,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   false,
			}
			player.GuInventory = append(player.GuInventory, newGu)
		} else {
			capturedGuID = ""
			inventoryFullMsg = " [Hành lý đầy, Cổ trùng hoang dã trốn thoát!]"
		}
	}

	// Roll 5% chance to find a recipe (Tàn Phương)
	rollRecipe := rand.Intn(100)
	if rollRecipe < 5 {
		// Fetch all recipe IDs from refining_recipes
		var allRecipeIDs []string
		rows, err := tx.Query(`SELECT ('recipe_' || target_gu_id || '_' || target_rank::text || '_' || variant::text) FROM refining_recipes`)
		if err == nil {
			for rows.Next() {
				var rID string
				if rows.Scan(&rID) == nil {
					allRecipeIDs = append(allRecipeIDs, rID)
				}
			}
			rows.Close()
		}

		if len(allRecipeIDs) > 0 {
			// Find which ones player hasn't unlocked or isn't 100%
			recipes, _ := RecipeSvc.ListByPlayer(player.ID.String())
			unlockedMap := make(map[string]float64)
			for _, r := range recipes {
				unlockedMap[r.RecipeID] = r.CompletionRate
			}

			var candidates []string
			for _, rID := range allRecipeIDs {
				if rate, ok := unlockedMap[rID]; !ok || rate < 100.0 {
					candidates = append(candidates, rID)
				}
			}

			if len(candidates) > 0 {
				chosenCandidate := candidates[rand.Intn(len(candidates))]
				randomRate := float64(30 + rand.Intn(30)) // 30% to 60%

				// Add or update
				RecipeSvc.Upsert(tx, &services.RecipeProgress{
					PlayerID:       player.ID.String(),
					RecipeID:       chosenCandidate,
					CompletionRate: randomRate,
					IsUnlocked:     true,
				})

				// Find target Gu name
				var targetGuName string
				var targetGuID string
				if tg, _, _, err := ParseRecipeID(chosenCandidate); err == nil {
					targetGuID = tg
				}
				_ = tx.QueryRow(`
					SELECT name FROM gu_dictionary 
					WHERE id = $1
				`, targetGuID).Scan(&targetGuName)

				if targetGuName == "" {
					targetGuName = chosenCandidate
				}

				// Set capturedGuID as non-empty so we can append to logMsg below
				capturedGuName = targetGuName              // reuse variable to communicate name
				capturedGuID = "RECIPE:" + chosenCandidate // mark as recipe
			}
		}
	}

	// 5. Persist changes
	qiGained := 0.0
	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, last_mana_update = $3, human_qi = $4
		WHERE id = $5
	`
	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update explore progress: " + err.Error()})
		return
	}
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.LastManaUpdate, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update explore progress: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	// Cultivation logs
	exploreLogs := []string{
		"Đạo hữu khám phá rừng trúc Thanh Mao Sơn và nhặt được %d Nguyên Thạch!",
		"Đạo hữu đánh bại dã trư rừng và thu hoạch %d Nguyên Thạch từ thi thể nó.",
		"Đạo hữu leo lên vách đá gồ ghề và phát hiện %d Nguyên Thạch giấu trong khe đá.",
		"Đạo hữu ngồi bên dòng suối nhỏ yên bình đãi cát tìm thấy %d Nguyên Thạch.",
	}
	selectedLogTemplate := exploreLogs[rand.Intn(len(exploreLogs))]
	logMsg := fmt.Sprintf(selectedLogTemplate, rewardStones)

	if droppedMaterial != "" {
		logMsg = fmt.Sprintf("%s Ngoài ra, đạo hữu còn tìm thấy 1x %s!", logMsg, droppedMaterialName)
	}

	if capturedGuID != "" {
		if strings.HasPrefix(capturedGuID, "RECIPE:") {
			logMsg = fmt.Sprintf("%s Ngoài ra, đạo hữu phát hiện một tờ Tàn Phương của %s!", logMsg, capturedGuName)
		} else {
			logMsg = fmt.Sprintf("%s Và đạo hữu đã thu phục thành công một con %s hoang dã!", logMsg, capturedGuName)
		}
	}

	if inventoryFullMsg != "" {
		logMsg = logMsg + inventoryFullMsg
	}

	logMsg += fmt.Sprintf(" Nhân Khí +%.0f.", qiGained)

	c.JSON(http.StatusOK, ExploreResponse{
		Player:       player,
		RewardStones: rewardStones,
		Log:          logMsg,
	})
}

// FeedRequest represents payload for feeding Gu
