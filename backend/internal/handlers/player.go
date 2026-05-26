package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// CreatePlayerRequest represents the incoming registration payload
type CreatePlayerRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=6,max=50"`
}

// CreatePlayerResponse represents the output payload for registration/login
type CreatePlayerResponse struct {
	Player models.Player `json:"player"`
	Token  string        `json:"token"`
}

// RegisterPlayer handles registering a new player, hashing password, and rolling their aptitude
func RegisterPlayer(c *gin.Context) {
	var req CreatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Hash password with bcrypt
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to secure password"})
		return
	}
	passwordHash := string(hashedBytes)

	// 2. Gacha rolling for aptitude
	roll := rand.Intn(100)
	var aptitudeType string
	var aptitudePercent int

	if roll < 10 {
		aptitudeType = "Giáp"
		aptitudePercent = 80 + rand.Intn(20) // 80 - 99
	} else if roll < 30 {
		aptitudeType = "Ất"
		aptitudePercent = 60 + rand.Intn(20) // 60 - 79
	} else if roll < 60 {
		aptitudeType = "Bính"
		aptitudePercent = 40 + rand.Intn(20) // 40 - 59
	} else {
		aptitudeType = "Đinh"
		aptitudePercent = 20 + rand.Intn(20) // 20 - 39
	}

	maxMana := float64(aptitudePercent)
	currentMana := maxMana

	// 3. Initialize starter inventory for Phase 2
	starterGu := models.GuInventory{
		{
			InstanceID: uuid.New().String(),
			GuID:       "GU_MOONLIGHT",
			Hunger:     100,
			Status:     "ACTIVE",
		},
	}
	starterMaterials := models.MaterialsInventory{
		"moon_orchid_petal": 5,
		"green_bamboo_wine": 2,
		"jade_stone":        1,
	}

	var player models.Player
	query := `
		INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, gu_inventory, materials_inventory, hp, current_region, dao_marks, active_combat)
		VALUES ($1, $2, 1, 1, $3, $4, $5, $6, 0, $7, $8, 100.0, 'Southern_Border', '{"strength": 0}'::jsonb, NULL)
		RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, gu_inventory, materials_inventory, hp, current_region, dao_marks, active_combat, last_mana_update, created_at
	`

	var activeCombatBytes []byte
	err = db.DB.QueryRow(query, req.Username, passwordHash, aptitudeType, aptitudePercent, currentMana, maxMana, starterGu, starterMaterials).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.HP, &player.CurrentRegion, &player.DaoMarks, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
	)
	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists in the Gu World"})
		return
	}

	// 4. Generate token
	token, err := auth.GenerateToken(player.ID, player.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access token"})
		return
	}

	c.JSON(http.StatusCreated, CreatePlayerResponse{
		Player: player,
		Token:  token,
	})
}

// LoginRequest represents credentials login payload
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginPlayer validates cultivator credentials and signs a JWT token
func LoginPlayer(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username and password are required"})
		return
	}

	var player models.Player
	var passwordHash string

	query := `
		SELECT id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, gu_inventory, materials_inventory, hp, current_region, dao_marks, active_combat, last_mana_update, created_at
		FROM players
		WHERE username = $1
	`
	var activeCombatBytes []byte
	err := db.DB.QueryRow(query, req.Username).Scan(
		&player.ID, &player.Username, &passwordHash, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.HP, &player.CurrentRegion, &player.DaoMarks, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
	)
	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		}
		return
	}

	// Verify hashed password
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Lazy evaluate mana regen upon login for accurate stats immediately
	tx, err := db.DB.Begin()
	if err == nil {
		defer tx.Rollback()
		if updatedPlayer, err := evaluateManaTx(tx, player.ID); err == nil {
			if err := tx.Commit(); err == nil {
				player = updatedPlayer
			}
		}
	}

	token, err := auth.GenerateToken(player.ID, player.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign access token"})
		return
	}

	c.JSON(http.StatusOK, CreatePlayerResponse{
		Player: player,
		Token:  token,
	})
}

// GetPlayerStatus retrieves the player and updates their mana via lazy evaluation
func GetPlayerStatus(c *gin.Context) {
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

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to evaluate player status: " + err.Error()})
		}
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player status"})
		return
	}

	c.JSON(http.StatusOK, player)
}

// ExploreResponse represents the action of exploring
type ExploreResponse struct {
	Player       models.Player `json:"player"`
	RewardStones int           `json:"reward_stones"`
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

	// 2. Check mana cost (10.0 mana)
	manaCost := 10.0
	if player.CurrentMana < manaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Not enough Primeval Essence. Rest to recover."})
		return
	}

	// 3. Deduct mana and roll reward stones
	player.CurrentMana -= manaCost
	rewardStones := 1 + rand.Intn(3) // 1 to 3 primeval stones
	player.PrimevalStones += rewardStones
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

	if droppedMaterial != "" {
		if player.MaterialsInventory == nil {
			player.MaterialsInventory = make(models.MaterialsInventory)
		}
		player.MaterialsInventory[droppedMaterial]++
	}

	// Roll 10% chance to capture wild Gu bugs
	rollGu := rand.Intn(100)
	var capturedGuID string
	var capturedGuName string
	if rollGu < 10 { // 10% chance overall
		subRoll := rand.Intn(10) // 0 to 9
		if subRoll < 5 { // 5% absolute chance
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
		newGu := models.GuInstance{
			InstanceID: uuid.New().String(),
			GuID:       capturedGuID,
			Hunger:     100,
			Status:     "ACTIVE",
		}
		player.GuInventory = append(player.GuInventory, newGu)
	}

	// 5. Persist changes
	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, materials_inventory = $3, gu_inventory = $4, last_mana_update = $5
		WHERE id = $6
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.MaterialsInventory, player.GuInventory, player.LastManaUpdate, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update explore progress"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	// Cultivation logs
	exploreLogs := []string{
		"You explored the bamboo forests of Qing Mao Mountain and picked up %d Primeval Stones!",
		"You defeated a wild mountain boar and extracted %d Primeval Stones from its body.",
		"You scaled the rocky cliffs and discovered %d Primeval Stones hidden in a small crevice.",
		"You sat by a quiet stream, filtered the sediments, and recovered %d Primeval Stones.",
	}
	selectedLogTemplate := exploreLogs[rand.Intn(len(exploreLogs))]
	logMsg := fmt.Sprintf(selectedLogTemplate, rewardStones)

	if droppedMaterial != "" {
		logMsg = fmt.Sprintf("%s Additionally, you found 1x %s!", logMsg, droppedMaterialName)
	}

	if capturedGuID != "" {
		logMsg = fmt.Sprintf("%s And you successfully captured a wild %s!", logMsg, capturedGuName)
	}

	c.JSON(http.StatusOK, ExploreResponse{
		Player:       player,
		RewardStones: rewardStones,
		Log:          logMsg,
	})
}

// FeedRequest represents payload for feeding Gu
type FeedRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// FeedGu consumes a matching material and resets a Gu instance's hunger status
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
	specs, found := models.GuRegistry[targetGu.GuID]
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gu definition not found in registry"})
		return
	}

	// 4. Check food availability
	foodCount, hasFood := player.MaterialsInventory[specs.FoodType]
	if !hasFood || foodCount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("You need at least 1 %s to feed %s.", specs.FoodType, specs.Name)})
		return
	}

	// 5. Consume 1 food and restore hunger
	player.MaterialsInventory[specs.FoodType]--
	targetGu.Hunger = specs.MaxHunger
	targetGu.Status = "ACTIVE" // Wakes up from sleep (hibernation)

	// 6. Update database
	updateQuery := `
		UPDATE players
		SET gu_inventory = $1, materials_inventory = $2
		WHERE id = $3
	`
	_, err = tx.Exec(updateQuery, player.GuInventory, player.MaterialsInventory, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save feeding progress"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": fmt.Sprintf("Fed %s successfully! Consumed 1 %s.", specs.Name, specs.FoodType),
	})
}

// evaluateManaTx calculates and updates player mana within an active transaction
func evaluateManaTx(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	var player models.Player

	query := `
		SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, gu_inventory, materials_inventory, hp, current_region, dao_marks, active_combat, last_mana_update, created_at
		FROM players
		WHERE id = $1
		FOR UPDATE
	`
	var activeCombatBytes []byte
	err := tx.QueryRow(query, id).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.HP, &player.CurrentRegion, &player.DaoMarks, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
	)
	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}
	if err != nil {
		return player, err
	}

	var regenRatePerHour float64
	switch player.AptitudeType {
	case "Giáp":
		regenRatePerHour = 8.0
	case "Ất":
		regenRatePerHour = 6.0
	case "Bính":
		regenRatePerHour = 4.0
	case "Đinh":
		regenRatePerHour = 2.0
	default:
		regenRatePerHour = 2.0
	}

	now := time.Now()
	secondsPassed := now.Sub(player.LastManaUpdate).Seconds()

	if secondsPassed > 0 {
		earnedMana := (secondsPassed / 3600.0) * regenRatePerHour
		newMana := player.CurrentMana + earnedMana
		if newMana > player.MaxMana {
			newMana = player.MaxMana
		}

		if newMana != player.CurrentMana {
			player.CurrentMana = newMana
			player.LastManaUpdate = now

			updateQuery := `
				UPDATE players
				SET current_mana = $1, last_mana_update = $2
				WHERE id = $3
			`
			_, err = tx.Exec(updateQuery, player.CurrentMana, player.LastManaUpdate, player.ID)
			if err != nil {
				return player, err
			}
		}
	}

	return player, nil
}
