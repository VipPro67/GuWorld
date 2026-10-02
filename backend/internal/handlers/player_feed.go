package handlers

import (
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func GetPlayerInventoryGus(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if tx, errTx := db.DB.Begin(); errTx == nil {
		if errH := worker.EvaluatePlayerHungerTx(tx, authID); errH == nil {
			_ = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
	}

	guInv, err := db.LoadPlayerGus(db.DB, authID.String(), false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load Gu inventory: " + err.Error()})
		return
	}

	equippedGus := make([]string, 0)
	for _, gu := range guInv {
		if gu.Equipped && gu.Status == "ACTIVE" {
			equippedGus = append(equippedGus, gu.InstanceID)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"gu_inventory": guInv,
		"equipped_gus": equippedGus,
	})
}

type FeedRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

var guManaTable = gamefx.RankScaleTable

const (
	feedBloodID    = "blood"
	feedThoughtID  = "thought"
	feedSoulID     = "soul"
	feedLifespanID = "lifespan"

	feedBloodCost    = 10.0
	feedThoughtCost  = 10.0
	feedSoulCost     = 10
	feedLifespanCost = 1.0
)

func isFeedResource(foodID string) bool {
	switch foodID {
	case "primeval_stone", "immortal_stone", feedBloodID, feedThoughtID, feedSoulID, feedLifespanID:
		return true
	default:
		return false
	}
}

// canConsumeFeedResource reports whether one serving can be paid. Blood, soul,
// and lifespan always leave the cultivator with at least one point.
func canConsumeFeedResource(player models.Player, foodID string, guRank int) bool {
	switch foodID {
	case "primeval_stone":
		return player.PrimevalStones >= math.Pow(2, float64(guRank))
	case "immortal_stone":
		return player.ImmortalStones >= float64(guRank)
	case feedBloodID:
		return player.HP > feedBloodCost
	case feedThoughtID:
		return player.CurrentThought >= feedThoughtCost
	case feedSoulID:
		return player.SoulHP > feedSoulCost
	case feedLifespanID:
		return player.Lifespan > feedLifespanCost
	default:
		return false
	}
}

func consumeFeedResource(player *models.Player, foodID string, guRank int) bool {
	if !canConsumeFeedResource(*player, foodID, guRank) {
		return false
	}
	switch foodID {
	case "primeval_stone":
		player.PrimevalStones -= math.Pow(2, float64(guRank))
	case "immortal_stone":
		player.ImmortalStones -= float64(guRank)
	case feedBloodID:
		player.HP -= feedBloodCost
	case feedThoughtID:
		player.CurrentThought -= feedThoughtCost
	case feedSoulID:
		player.SoulHP -= feedSoulCost
	case feedLifespanID:
		player.Lifespan -= feedLifespanCost
	}
	return true
}

func feedResourceName(foodID string) string {
	switch foodID {
	case "primeval_stone":
		return "Nguyên Thạch"
	case "immortal_stone":
		return "Tiên Nguyên Thạch"
	case feedBloodID:
		return "Máu"
	case feedThoughtID:
		return "Niệm Lực"
	case feedSoulID:
		return "Linh Hồn"
	case feedLifespanID:
		return "Tuổi Thọ"
	default:
		return foodID
	}
}

func getManaVal(rank int) float64 {
	return gamefx.RankScale(rank)
}

// computeFoodNutrition calculates hunger restoration based on the Gu's rank and the food's rank
// using the standard rank mana table scaling (base 100 for same rank, no minimum floor limit).
func computeFoodNutrition(foodID string, guRank int) int {
	foodRank := 1
	found := false
	if spec, ok := models.GetGuAny(foodID); ok {
		foodRank = spec.Rank
		found = true
	} else if spec, ok := models.ItemRegistry[foodID]; ok {
		foodRank = spec.Rank
		found = true
	}

	if !found {
		// Default fallback if food rank is unknown.
		return 100
	}

	if foodRank >= guRank {
		return 100
	}

	guMana := getManaVal(guRank)
	foodMana := getManaVal(foodRank)
	return int(math.Floor((foodMana / guMana) * 100.0))
}

func chooseFeedCandidate(player models.Player, targetGu models.GuInstance, foodOptions []string, targetInstanceID string, materialQty map[string]int64) (foodID string, consumedGuIndex int) {
	for _, food := range foodOptions {
		if isFeedResource(food) {
			if canConsumeFeedResource(player, food, targetGu.Rank) {
				return food, -1
			}
			continue
		}
		if strings.HasPrefix(food, "GU_") {
			for i, invGu := range player.GuInventory {
				if invGu.GuID == food && invGu.InstanceID != targetInstanceID && invGu.Status != "DEAD" && invGu.Status != "SEALED" && !invGu.IsVital && !invGu.Equipped {
					return food, i
				}
			}
			continue
		}
		if materialQty != nil && materialQty[food] > 0 {
			return food, -1
		}
	}
	return "", -1
}

func FeedGu(c *gin.Context) {
	var req FeedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
		return
	}

	// Retrieve auth player ID
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

	// 1. Fetch player details with FOR UPDATE lock
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// 2. Find target Gu instance
	guIndex := -1
	for i, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			guIndex = i
			break
		}
	}

	if guIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gu bug not found in your inventory"})
		return
	}

	targetGu := &player.GuInventory[guIndex]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This Gu bug is DEAD and cannot be fed"})
		return
	}

	// 3. Find static specs in Dictionary
	specs, found := models.GetGu(targetGu.GuID, targetGu.Rank)
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gu definition not found in registry"})
		return
	}

	// 4. Loop consuming food until full
	foodOptions := specs.FoodTypes
	foodConsumed := make(map[string]int)
	var eatenGuIDs []string
	var feedEventMsg string

	if targetGu.Hunger >= specs.MaxHunger {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đã no, không cần ăn thêm."})
		return
	}

	materialQty := make(map[string]int64)
	for _, food := range foodOptions {
		if isFeedResource(food) || strings.HasPrefix(food, "GU_") {
			continue
		}
		qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), food)
		materialQty[food] = qty
	}

	for targetGu.Hunger < specs.MaxHunger {
		chosenFoodItem, consumedGuIndex := chooseFeedCandidate(player, *targetGu, foodOptions, targetGu.InstanceID, materialQty)
		if chosenFoodItem == "" && consumedGuIndex == -1 {
			break // No more food
		}

		foodID := ""
		if chosenFoodItem != "" {
			if !isFeedResource(chosenFoodItem) {
				if err := InventorySvc.RemoveItem(tx, player.ID.String(), chosenFoodItem, "materials", 1); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ thức ăn phù hợp để cho Cổ trùng ăn."})
					return
				}
				if materialQty[chosenFoodItem] > 0 {
					materialQty[chosenFoodItem]--
				}
			}
			foodID = chosenFoodItem
		} else {
			consumedGu := player.GuInventory[consumedGuIndex]
			foodID = consumedGu.GuID
			eatenGuIDs = append(eatenGuIDs, consumedGu.InstanceID)

			// Fire ON_FEED trigger for the consumed Gu
			if cSpec, cOk := models.GetGu(consumedGu.GuID, consumedGu.Rank); cOk {
				if msg, err := HandleGuEvent(&player, cSpec, consumedGu, "ON_FEED"); err == nil && msg != "" {
					feedEventMsg = msg
				}
			}

			player.GuInventory = append(player.GuInventory[:consumedGuIndex], player.GuInventory[consumedGuIndex+1:]...)

			// Re-find targetGu since slice changed
			guIndex = -1
			for i, gu := range player.GuInventory {
				if gu.InstanceID == req.GuInstanceID {
					guIndex = i
					break
				}
			}
			if guIndex == -1 {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Target Gu instance was removed while feeding"})
				return
			}
			targetGu = &player.GuInventory[guIndex]
		}

		if isFeedResource(foodID) && !consumeFeedResource(&player, foodID, targetGu.Rank) {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ %s để cho Cổ trùng ăn.", feedResourceName(foodID))})
			return
		}

		foodConsumed[foodID]++
		nutrition := computeFoodNutrition(foodID, targetGu.Rank)
		targetGu.Hunger += nutrition
		if targetGu.Hunger > specs.MaxHunger {
			targetGu.Hunger = specs.MaxHunger
		}
	}

	if len(foodConsumed) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không có thức ăn phù hợp cho %s. Cần một trong: %v", specs.Name, foodOptions)})
		return
	}

	targetGu.Status = "ACTIVE" // Wakes up from sleep (hibernation)
	if targetGu.IsVital {
		targetGu.Equipped = true
	}

	qiGained := 0.0

	// 6. Update database
	if len(eatenGuIDs) > 0 {
		if err := db.RemovePlayerGus(tx, eatenGuIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove consumed Gu bugs"})
			return
		}
	}
	if err := db.UpdatePlayerGu(tx, *targetGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}
	if _, err := tx.Exec(`UPDATE player_gu SET last_fed_at = NOW() WHERE id = $1`, targetGu.InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, immortal_stones = $2, hp = $3, max_mana = $4, current_mana = $5, lifespan = $6, human_qi = $7, current_thought = $8, soul_hp = $9 WHERE id = $10`,
		player.PrimevalStones, player.ImmortalStones, player.HP, player.MaxMana, player.CurrentMana, player.Lifespan, player.HumanQi, player.CurrentThought, player.SoulHP, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	var foodDetails []string
	for fID, qty := range foodConsumed {
		fName := feedResourceName(fID)
		if itemSpec, ok := models.ItemRegistry[fID]; ok {
			fName = itemSpec.Name
		} else if guSpec, ok := models.GetGuAny(fID); ok {
			fName = guSpec.Name
		}
		foodDetails = append(foodDetails, fmt.Sprintf("%dx %s", qty, fName))
	}
	foodStr := strings.Join(foodDetails, ", ")

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "gu.feed",
		EntityType:     "gu_instance",
		EntityID:       targetGu.InstanceID,
		RequestSummary: fmt.Sprintf("Cho %s ăn: %s | Độ no: %d/%d", specs.Name, foodStr, targetGu.Hunger, specs.MaxHunger),
		Metadata: map[string]interface{}{
			"gu_instance_id": targetGu.InstanceID,
			"gu_id":          targetGu.GuID,
			"gu_name":        specs.Name,
			"rank":           targetGu.Rank,
			"hunger":         targetGu.Hunger,
			"max_hunger":     specs.MaxHunger,
			"food_consumed":  foodConsumed,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	msg := fmt.Sprintf("Cho %s ăn thành công! Đã khôi phục tới %d độ no. Tiêu hao: %s. Nhân Khí +%.0f.", specs.Name, targetGu.Hunger, foodStr, qiGained)
	if feedEventMsg != "" {
		msg += " " + feedEventMsg
	}
	respondActionGuOnly(c, http.StatusOK, player, gin.H{
		"message": msg,
	})
}

// FeedGroupRequest represents payload for feeding all hungry Gu bugs of the same type
type FeedGroupRequest struct {
	GuID string `json:"gu_id" binding:"required"`
	Rank *int   `json:"rank,omitempty"`
}

// FeedGroup feeds all hungry Gu bugs of the same type (gu_id) using available food
func FeedGroup(c *gin.Context) {
	var req FeedGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_id is required"})
		return
	}

	// Retrieve auth player ID
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

	// 1. Fetch player details with FOR UPDATE lock
	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	// 2. Find static specs in Dictionary (match both ID and rank when rank provided)
	specs, found := models.GetGuAny(req.GuID)
	if req.Rank != nil {
		if spec, ok := models.GetGu(req.GuID, *req.Rank); ok {
			specs = spec
			found = true
		}
	}
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gu definition not found in registry"})
		return
	}

	foodOptions := specs.FoodTypes
	if len(foodOptions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This Gu bug does not require any food or has no food defined"})
		return
	}

	getAvailableFood := func(targetRank int) string {
		for _, food := range foodOptions {
			if isFeedResource(food) {
				if canConsumeFeedResource(player, food, targetRank) {
					return food
				}
			} else if !strings.HasPrefix(food, "GU_") {
				qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), food)
				if qty > 0 {
					return food
				}
			}
		}
		return ""
	}

	getAvailableGuFood := func(skipInstanceID string) int {
		for _, food := range foodOptions {
			if strings.HasPrefix(food, "GU_") {
				for i, invGu := range player.GuInventory {
					if invGu.GuID == food && invGu.InstanceID != skipInstanceID && invGu.Status != "DEAD" && !invGu.IsVital && !invGu.Equipped {
						return i
					}
				}
			}
		}
		return -1
	}

	fedCount := 0
	foodConsumed := make(map[string]int)
	consumedGuIDs := []string{}  // Track Gu IDs consumed as food for explicit delete.
	fedInstanceIDs := []string{} // Track instances actually fed, to stamp last_fed_at.

	// 3. Iterate over the player's Gu inventory and feed hungry ones of matching type
	for i := 0; i < len(player.GuInventory); i++ {
		gu := &player.GuInventory[i]
		if gu.GuID == req.GuID && (req.Rank == nil || gu.Rank == *req.Rank) && gu.Status != "DEAD" && gu.Status != "SEALED" && gu.Hunger < specs.MaxHunger {
			fedThisGu := false
			for gu.Hunger < specs.MaxHunger {
				chosenFood := getAvailableFood(gu.Rank)
				consumedGuIndex := -1
				if chosenFood == "" {
					consumedGuIndex = getAvailableGuFood(gu.InstanceID)
				}

				if chosenFood == "" && consumedGuIndex == -1 {
					break // No more food available
				}

				foodID := ""
				// Consume food
				if chosenFood != "" {
					if isFeedResource(chosenFood) {
						if !consumeFeedResource(&player, chosenFood, gu.Rank) {
							break
						}
					} else {
						InventorySvc.RemoveItem(tx, player.ID.String(), chosenFood, "materials", 1)
					}
					foodID = chosenFood
					foodConsumed[chosenFood]++
				} else {
					consumedGu := player.GuInventory[consumedGuIndex]
					foodID = consumedGu.GuID
					foodConsumed[consumedGu.GuID]++
					consumedGuIDs = append(consumedGuIDs, consumedGu.InstanceID)
					player.GuInventory = append(player.GuInventory[:consumedGuIndex], player.GuInventory[consumedGuIndex+1:]...)
					if consumedGuIndex <= i {
						i--
					}
					gu = &player.GuInventory[i] // refetch pointer
				}

				nutrition := computeFoodNutrition(foodID, gu.Rank)
				gu.Hunger += nutrition
				if gu.Hunger > specs.MaxHunger {
					gu.Hunger = specs.MaxHunger
				}
				fedThisGu = true
			}

			if fedThisGu {
				gu.Status = "ACTIVE"
				if gu.IsVital {
					gu.Equipped = true
				}
				fedCount++
				fedInstanceIDs = append(fedInstanceIDs, gu.InstanceID)
			}
		}
	}

	if fedCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có Cổ trùng nào cần cho ăn hoặc không có đủ thức ăn phù hợp"})
		return
	}

	// 4. Update database
	// Explicit delete — SyncPlayerGus no longer deletes.
	if len(consumedGuIDs) > 0 {
		if err := db.RemovePlayerGus(tx, consumedGuIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to consume Gu as food"})
			return
		}
	}
	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}
	if len(fedInstanceIDs) > 0 {
		if _, err := tx.Exec(`UPDATE player_gu SET last_fed_at = NOW() WHERE id = ANY($1::text[])`, pq.Array(fedInstanceIDs)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
			return
		}
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, immortal_stones = $2, hp = $3, current_thought = $4, soul_hp = $5, lifespan = $6 WHERE id = $7`,
		player.PrimevalStones, player.ImmortalStones, player.HP, player.CurrentThought, player.SoulHP, player.Lifespan, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	// Build success message
	var foodDetails []string
	for foodID, qty := range foodConsumed {
		foodName := feedResourceName(foodID)
		if itemSpec, ok := models.ItemRegistry[foodID]; ok {
			foodName = itemSpec.Name
		} else if guSpec, ok := models.GetGuAny(foodID); ok {
			foodName = guSpec.Name
		}
		foodDetails = append(foodDetails, fmt.Sprintf("%dx %s", qty, foodName))
	}

	foodStr := strings.Join(foodDetails, ", ")

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "gu.feed_group",
		EntityType:     "gu_definition",
		EntityID:       req.GuID,
		RequestSummary: fmt.Sprintf("Cho %d Cổ trùng %s ăn | Tiêu hao: %s", fedCount, specs.Name, foodStr),
		Metadata: map[string]interface{}{
			"gu_id":         req.GuID,
			"gu_name":       specs.Name,
			"fed_count":     fedCount,
			"food_consumed": foodConsumed,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionGuOnly(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Đã cho ăn thành công %d con %s! Tiêu hao: %s.", fedCount, specs.Name, foodStr),
	})
}

type FeedAllRequest struct {
	Scope    string `json:"scope"`              // "all", "equipped", "unequipped" / "rest"
	Equipped *bool  `json:"equipped,omitempty"` // true: equipped, false: unequipped
}

// FeedAll feeds all hungry Gu bugs in the player's inventory using available food
func FeedAll(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req FeedAllRequest
	_ = c.ShouldBindJSON(&req)
	if req.Scope == "" {
		req.Scope = c.Query("scope")
	}
	if req.Scope == "rest" {
		req.Scope = "unequipped"
	}

	filterEquipped := req.Scope == "equipped" || (req.Equipped != nil && *req.Equipped)
	filterUnequipped := req.Scope == "unequipped" || (req.Equipped != nil && !*req.Equipped)

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}
	before := append(models.GuInventory(nil), player.GuInventory...)

	fedCount := 0
	foodConsumed := make(map[string]int64)
	materialQty := make(map[string]int64)

	getAvailableFood := func(specs models.GuDictionaryItem, targetRank int) string {
		for _, food := range specs.FoodTypes {
			if isFeedResource(food) {
				if canConsumeFeedResource(player, food, targetRank) {
					return food
				}
			} else if !strings.HasPrefix(food, "GU_") {
				if _, cached := materialQty[food]; !cached {
					qty, _ := InventorySvc.GetQuantity(tx, player.ID.String(), food)
					materialQty[food] = qty
				}
				if materialQty[food] > 0 {
					return food
				}
			}
		}
		return ""
	}

	getAvailableGuFood := func(specs models.GuDictionaryItem, skipInstanceID string) int {
		for _, food := range specs.FoodTypes {
			if strings.HasPrefix(food, "GU_") {
				for i, invGu := range player.GuInventory {
					if invGu.GuID == food && invGu.InstanceID != skipInstanceID && invGu.Status != "DEAD" && invGu.Status != "SEALED" && !invGu.IsVital && !invGu.Equipped {
						return i
					}
				}
			}
		}
		return -1
	}

	consumedGuIDs := []string{}  // Track Gu IDs consumed as food for explicit delete.
	fedInstanceIDs := []string{} // Track instances actually fed, to stamp last_fed_at.

	for i := 0; i < len(player.GuInventory); i++ {
		gu := &player.GuInventory[i]
		if gu.Status == "DEAD" || gu.Status == "SEALED" {
			continue
		}

		if filterEquipped && !gu.Equipped {
			continue
		}
		if filterUnequipped && gu.Equipped {
			continue
		}

		specs, found := models.GetGu(gu.GuID, gu.Rank)
		if !found {
			if specAny, ok := models.GetGuAny(gu.GuID); ok {
				specs = specAny
				found = true
			}
		}
		if !found || len(specs.FoodTypes) == 0 {
			continue
		}

		if gu.Hunger >= specs.MaxHunger {
			continue
		}

		fedThisGu := false
		for gu.Hunger < specs.MaxHunger {
			chosenFood := getAvailableFood(specs, gu.Rank)
			consumedGuIndex := -1
			if chosenFood == "" {
				consumedGuIndex = getAvailableGuFood(specs, gu.InstanceID)
			}

			if chosenFood == "" && consumedGuIndex == -1 {
				break // No more food for this Gu
			}

			foodID := ""
			if chosenFood != "" {
				if isFeedResource(chosenFood) {
					if !consumeFeedResource(&player, chosenFood, gu.Rank) {
						break
					}
				} else {
					if err := InventorySvc.RemoveItem(tx, player.ID.String(), chosenFood, "materials", 1); err == nil {
						materialQty[chosenFood]--
					} else {
						materialQty[chosenFood] = 0
						continue
					}
				}
				foodID = chosenFood
				foodConsumed[chosenFood]++
			} else {
				consumedGu := player.GuInventory[consumedGuIndex]
				foodID = consumedGu.GuID
				foodConsumed[consumedGu.GuID]++
				consumedGuIDs = append(consumedGuIDs, consumedGu.InstanceID)
				player.GuInventory = append(player.GuInventory[:consumedGuIndex], player.GuInventory[consumedGuIndex+1:]...)
				if consumedGuIndex <= i {
					i--
				}
				gu = &player.GuInventory[i]
			}

			nutrition := computeFoodNutrition(foodID, gu.Rank)
			gu.Hunger += nutrition
			if gu.Hunger > specs.MaxHunger {
				gu.Hunger = specs.MaxHunger
			}
			fedThisGu = true
		}

		if fedThisGu {
			gu.Status = "ACTIVE"
			if gu.IsVital {
				gu.Equipped = true
			}
			fedCount++
			fedInstanceIDs = append(fedInstanceIDs, gu.InstanceID)
		}
	}

	if fedCount == 0 {
		errMsg := "Không có Cổ trùng nào cần cho ăn hoặc không có đủ thức ăn phù hợp"
		if filterEquipped {
			errMsg = "Không có Cổ trùng đang xuất trận nào cần cho ăn hoặc không có đủ thức ăn phù hợp"
		} else if filterUnequipped {
			errMsg = "Không có Cổ trùng trong túi nào cần cho ăn hoặc không có đủ thức ăn phù hợp"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}

	// Explicit delete — SyncPlayerGus no longer deletes.
	if len(consumedGuIDs) > 0 {
		if err := db.RemovePlayerGus(tx, consumedGuIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to consume Gu as food"})
			return
		}
	}
	if err := db.SyncPlayerGus(tx, player.ID.String(), before, player.GuInventory); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}
	if len(fedInstanceIDs) > 0 {
		if _, err := tx.Exec(`UPDATE player_gu SET last_fed_at = NOW() WHERE id = ANY($1::text[])`, pq.Array(fedInstanceIDs)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
			return
		}
	}
	_, err = tx.Exec(`UPDATE players SET primeval_stones = $1, immortal_stones = $2, hp = $3, current_thought = $4, soul_hp = $5, lifespan = $6 WHERE id = $7`,
		player.PrimevalStones, player.ImmortalStones, player.HP, player.CurrentThought, player.SoulHP, player.Lifespan, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	// Build success message
	var foodDetails []string
	for foodID, qty := range foodConsumed {
		foodName := feedResourceName(foodID)
		if itemSpec, ok := models.ItemRegistry[foodID]; ok {
			foodName = itemSpec.Name
		} else if guSpec, ok := models.GetGuAny(foodID); ok {
			foodName = guSpec.Name
		}
		foodDetails = append(foodDetails, fmt.Sprintf("%dx %s", qty, foodName))
	}

	foodStr := strings.Join(foodDetails, ", ")

	scopeDesc := "toàn bộ"
	scopeName := "tất cả Cổ trùng"
	if filterEquipped {
		scopeDesc = "toàn bộ Cổ trùng xuất trận"
		scopeName = "Cổ trùng xuất trận"
	} else if filterUnequipped {
		scopeDesc = "toàn bộ Cổ trùng trong túi"
		scopeName = "Cổ trùng trong túi"
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &authID,
		Action:         "gu.feed_all",
		EntityType:     "player",
		EntityID:       player.ID.String(),
		RequestSummary: fmt.Sprintf("Cho %s %d Cổ trùng ăn | Tiêu hao: %s", scopeDesc, fedCount, foodStr),
		Metadata: map[string]interface{}{
			"fed_count":     fedCount,
			"food_consumed": foodConsumed,
			"scope":         req.Scope,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	respondActionGuOnly(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Cho %s ăn thành công! Đã khôi phục cho %d Cổ trùng. Tiêu hao: %s.", scopeName, fedCount, foodStr),
	})
}
