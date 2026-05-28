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
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

// CreatePlayerRequest represents the incoming registration payload
type CreatePlayerRequest struct {
	Username       string `json:"username" binding:"required,min=3,max=30"`
	Password       string `json:"password" binding:"required,min=6,max=50"`
	StartingRegion string `json:"starting_region" binding:"required"`
	SetVitalNow    bool   `json:"set_vital_now"`
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
	validRegions := map[string]bool{
		"Southern_Border": true,
		"Central_Continent": true,
		"Northern_Plains": true,
		"Western_Desert": true,
		"Eastern_Sea": true,
	}
	if !validRegions[req.StartingRegion] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Starting region không hợp lệ"})
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
			IsVital:    req.SetVitalNow,
		},
	}
	starterMaterials := models.MaterialsInventory{
		"moon_orchid_petal": 5,
		"green_bamboo_wine": 2,
		"jade_stone":        1,
	}

	var player models.Player
	query := `
		INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, gu_inventory, materials_inventory, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat)
		VALUES ($1, $2, 1, 1, $3, $4, $5, $6, 0, $7, $8, 100.0, 100.0, '[]'::jsonb, 50, 50, '["50,50"]'::jsonb, '[]'::jsonb, '{}'::jsonb, $9, '{"strength": 0}'::jsonb, '{}'::jsonb, NULL)
		RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, gu_inventory, materials_inventory, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at
	`

	var activeCombatBytes []byte
	err = db.DB.QueryRow(query, req.Username, passwordHash, aptitudeType, aptitudePercent, currentMana, maxMana, starterGu, starterMaterials, req.StartingRegion).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
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
		SELECT id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, gu_inventory, materials_inventory, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at
		FROM players
		WHERE username = $1
	`
	var activeCombatBytes []byte
	err := db.DB.QueryRow(query, req.Username).Scan(
		&player.ID, &player.Username, &passwordHash, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
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

	inventoryFullMsg := ""

	if droppedMaterial != "" {
		if HasInventorySpace(&player, 1) {
			if player.MaterialsInventory == nil {
				player.MaterialsInventory = make(models.MaterialsInventory)
			}
			player.MaterialsInventory[droppedMaterial]++
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
			newGu := models.GuInstance{
				InstanceID: uuid.New().String(),
				GuID:       capturedGuID,
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

	if inventoryFullMsg != "" {
		logMsg = logMsg + inventoryFullMsg
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
	foodOptions := specs.FoodTypes
	if len(foodOptions) == 0 && specs.FoodType != "" {
		foodOptions = []string{specs.FoodType}
	}
	chosenFood := ""
	for _, food := range foodOptions {
		if player.MaterialsInventory[food] > 0 {
			chosenFood = food
			break
		}
	}
	if chosenFood == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không có thức ăn phù hợp cho %s. Cần một trong: %v", specs.Name, foodOptions)})
		return
	}

	// 5. Consume 1 food and restore hunger
	player.MaterialsInventory[chosenFood]--
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
		"message": fmt.Sprintf("Fed %s successfully! Consumed 1 %s.", specs.Name, chosenFood),
	})
}

// evaluateManaTx calculates and updates player mana within an active transaction
func evaluateManaTx(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	var player models.Player

	query := `
		SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, current_mana, max_mana, primeval_stones, gu_inventory, materials_inventory, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at
		FROM players
		WHERE id = $1
		FOR UPDATE
	`
	var activeCombatBytes []byte
	err := tx.QueryRow(query, id).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
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

// EquipGuRequest defines request to equip/unequip Gu bugs
type EquipGuRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Equip      bool   `json:"equip"`
}

// CalculatePlayerStats computes dynamic stats based on equipped active Gu bugs and passive mods
func CalculatePlayerStats(player *models.Player) map[string]float64 {
	strengthMarks := 0
	if player.DaoMarks != nil {
		strengthMarks = player.DaoMarks["strength"]
	}
	stats := map[string]float64{
		"attack":        10.0 + float64(player.Rank-1)*10.0 + float64(player.SubRank-1)*2.0 + float64(strengthMarks)/10.0,
		"defense":       5.0 + float64(player.Rank-1)*5.0 + float64(player.SubRank-1)*1.0,
		"healing":       0.0,
		"movement":      1.0,
		"investigation": 1.0,
		"storage":       20.0,
	}

	for _, mod := range player.BodyMods {
		switch mod {
		case "Iron Bone":
			stats["defense"] += 15.0
			stats["attack"] += 5.0
		case "Jade Skin Mod":
			stats["defense"] += 25.0
		case "Wind Walk":
			stats["movement"] += 1.0
		}
	}

	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			spec, ok := models.GuRegistry[gu.GuID]
			if !ok {
				continue
			}

			masteryMultiplier := 1.0
			if player.PathMasteries != nil {
				if mLevel, ok := player.PathMasteries[spec.Path]; ok {
					switch mLevel {
					case "Ordinary":
						masteryMultiplier = 1.0
					case "Master":
						masteryMultiplier = 1.1
					case "Grandmaster":
						masteryMultiplier = 1.25
					case "Supreme Grandmaster":
						masteryMultiplier = 1.5
					}
				}
			}

			if dmg, ok := spec.Effects["damage"]; ok {
				stats["attack"] += dmg * masteryMultiplier
			}
			if dmg, ok := spec.Effects["add_base_attack"]; ok {
				stats["attack"] += dmg * masteryMultiplier
			}

			if def, ok := spec.Effects["defense"]; ok {
				stats["defense"] += def * masteryMultiplier
			}
			if def, ok := spec.Effects["add_base_defense"]; ok {
				stats["defense"] += def * masteryMultiplier
			}

			if heal, ok := spec.Effects["refine_mana_tier"]; ok {
				stats["healing"] += heal * 10.0
			}
			if heal, ok := spec.Effects["cleansing_power"]; ok {
				stats["healing"] += heal * 15.0
			}

			if mv, ok := spec.Effects["movement"]; ok {
				stats["movement"] += mv
			}

			if inv, ok := spec.Effects["investigation"]; ok {
				stats["investigation"] += inv
			}

			if st, ok := spec.Effects["storage"]; ok {
				stats["storage"] += st
			}
		}
	}

	return stats
}

// EquipGu toggles whether a Gu instance is equipped
func EquipGu(c *gin.Context) {
	var req EquipGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	targetIdx := -1
	equippedCount := 0
	for i, gu := range player.GuInventory {
		if gu.InstanceID == req.InstanceID {
			targetIdx = i
		}
		if gu.Equipped && gu.Status == "ACTIVE" {
			equippedCount++
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gu bug not found in inventory"})
		return
	}

	targetGu := &player.GuInventory[targetIdx]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot equip a DEAD Gu bug"})
		return
	}

	if req.Equip {
		if targetGu.Equipped {
			c.JSON(http.StatusOK, gin.H{"message": "Gu bug is already equipped", "player": player})
			return
		}
		if equippedCount >= 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Aperture capacity full! Cannot equip more than 6 Gu bugs."})
			return
		}
		targetGu.Equipped = true
	} else {
		targetGu.Equipped = false
	}

	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	_, err = tx.Exec("UPDATE players SET gu_inventory = $1, equipped_gus = $2 WHERE id = $3", player.GuInventory, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update equipped Gu inventory"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	statusMsg := "Unequipped"
	if req.Equip {
		statusMsg = "Equipped"
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": fmt.Sprintf("%s Gu successfully!", statusMsg),
	})
}

// UpdateGuStatusRequest defines body to change status of owned Gu instance
type UpdateGuStatusRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
	Status     string `json:"status" binding:"required"` // "ACTIVE", "SLEEPING", "SEALED"
}

// UpdateGuStatus changes Gu status between ACTIVE, SLEEPING, SEALED
func UpdateGuStatus(c *gin.Context) {
	var req UpdateGuStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Status != "ACTIVE" && req.Status != "SLEEPING" && req.Status != "SEALED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái không hợp lệ. Chỉ chấp nhận ACTIVE, SLEEPING hoặc SEALED."})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	targetIdx := -1
	for idx, inst := range player.GuInventory {
		if inst.InstanceID == req.InstanceID {
			targetIdx = idx
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Gu bug not found in inventory"})
		return
	}

	targetGu := &player.GuInventory[targetIdx]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể thay đổi trạng thái của Cổ trùng ĐÃ CHẾT."})
		return
	}

	// If transitioning away from ACTIVE (e.g. to SLEEPING or SEALED), force unequip if equipped
	if req.Status != "ACTIVE" && targetGu.Equipped {
		targetGu.Equipped = false
	}

	targetGu.Status = req.Status

	// Recalculate equipped_gus list
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	_, err = tx.Exec("UPDATE players SET gu_inventory = $1, equipped_gus = $2 WHERE id = $3", player.GuInventory, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Gu inventory status"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": fmt.Sprintf("Cập nhật trạng thái Cổ trùng thành %s thành công!", req.Status),
	})
}

// GetPlayerStatsHandler returns the calculated stats for the current player
func GetPlayerStatsHandler(c *gin.Context) {
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	stats := CalculatePlayerStats(&player)
	c.JSON(http.StatusOK, gin.H{
		"player": player,
		"stats":  stats,
	})
}

// HasInventorySpace checks if the player has enough space in their bag
func HasInventorySpace(player *models.Player, incomingCount int) bool {
	stats := CalculatePlayerStats(player)
	maxSlots := int(stats["storage"])
	if player.Rank >= 6 {
		maxSlots = 999
	}

	currentSize := len(player.GuInventory)
	for _, qty := range player.MaterialsInventory {
		if qty > 0 {
			currentSize++
		}
	}

	return (currentSize + incomingCount) <= maxSlots
}

// UseItemRequest defines the request body for using an item
type UseItemRequest struct {
	ItemID string `json:"item_id" binding:"required"`
}

// UseItem consumes a material to restore HP or mana
func UseItem(c *gin.Context) {
	var req UseItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		return
	}

	qty, ok := player.MaterialsInventory[req.ItemID]
	if !ok || qty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có vật phẩm này trong túi đồ"})
		return
	}

	logMsg := ""
	switch req.ItemID {
	case "green_bamboo_wine":
		player.CurrentMana += 20.0
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}
		logMsg = "Đạo hữu uống Rượu Thanh Trúc, phục hồi 20 Chân Nguyên!"
	case "moon_orchid_petal":
		player.CurrentMana += 10.0
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}
		logMsg = "Đạo hữu luyện hóa Cánh hoa Nguyệt Lan, phục hồi 10 Chân Nguyên!"
	case "jade_stone":
		player.HP += 20.0
		if player.HP > 100.0 {
			player.HP = 100.0
		}
		logMsg = "Đạo hữu mài da Ngọc Thạch đắp vết thương, hồi phục 20 HP!"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm này không thể sử dụng trực tiếp"})
		return
	}

	player.MaterialsInventory[req.ItemID]--
	if player.MaterialsInventory[req.ItemID] <= 0 {
		delete(player.MaterialsInventory, req.ItemID)
	}

	_, err = tx.Exec(`
		UPDATE players 
		SET materials_inventory = $1, current_mana = $2, hp = $3
		WHERE id = $4`,
		player.MaterialsInventory, player.CurrentMana, player.HP, player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": logMsg,
	})
}

// SetVitalGuRequest represents the payload to select a vital Gu
type SetVitalGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// SetVitalGu sets a selected Gu as the player's Vital Gu
func SetVitalGu(c *gin.Context) {
	var req SetVitalGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	targetIndex := -1
	for idx, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			targetIndex = idx
			break
		}
	}

	if targetIndex == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng trong túi đồ"})
		return
	}

	targetGu := &player.GuInventory[targetIndex]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể thiết lập Cổ trùng ĐÃ CHẾT làm Bản Mệnh Cổ"})
		return
	}

	if targetGu.IsVital {
		c.JSON(http.StatusOK, gin.H{
			"player":  player,
			"message": "Cổ trùng này đã là Bản Mệnh Cổ của đạo hữu rồi.",
		})
		return
	}

	// Check if they already have an existing Vital Gu
	hadVital := false
	for _, gu := range player.GuInventory {
		if gu.IsVital {
			hadVital = true
			break
		}
	}

	backlashMessage := ""
	if hadVital {
		// Backlash: cost 50 HP
		player.HP -= 50.0
		backlashMessage = " Đạo hữu bị phản phệ mất 50 HP do thay đổi Bản Mệnh Cổ!"
		if player.HP <= 0 {
			player.HP = 0
			player.ActiveCombat = nil // reset combat
			backlashMessage += " Đạo hữu khí huyết cạn kiệt, bất tĩnh nhân sự!"
		}
	}

	// Update is_vital flags
	for idx := range player.GuInventory {
		if idx == targetIndex {
			player.GuInventory[idx].IsVital = true
		} else {
			player.GuInventory[idx].IsVital = false
		}
	}

	// Recalculate equipped_gus
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	_, err = tx.Exec(`
		UPDATE players 
		SET gu_inventory = $1, hp = $2, equipped_gus = $3, active_combat = $4
		WHERE id = $5`,
		player.GuInventory, player.HP, player.EquippedGus, player.ActiveCombat, player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player vital Gu status"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": fmt.Sprintf("Thiết lập %s thành Bản Mệnh Cổ thành công!%s", targetGu.GuID, backlashMessage),
	})
}

// DropGuRequest removes an owned Gu from inventory intentionally.
type DropGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

func DropGu(c *gin.Context) {
	var req DropGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	idx := -1
	for i, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			idx = i
			break
		}
	}
	if idx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng trong túi đồ"})
		return
	}
	if player.GuInventory[idx].IsVital {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể vứt Bản Mệnh Cổ"})
		return
	}

	newInv := models.GuInventory{}
	for i, gu := range player.GuInventory {
		if i != idx {
			newInv = append(newInv, gu)
		}
	}
	player.GuInventory = newInv
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}
	_, err = tx.Exec(`UPDATE players SET gu_inventory = $1, equipped_gus = $2 WHERE id = $3`, player.GuInventory, player.EquippedGus, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to drop Gu"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"player": player, "message": "Đã vứt bỏ Cổ trùng khỏi túi đồ."})
}

// UseGuRequest represents request payload to use a consumable Gu
type UseGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// UseConsumableGu activates a consumable Gu bug's one-time effect and removes it
func UseConsumableGu(c *gin.Context) {
	var req UseGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "gu_instance_id is required"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	targetIdx := -1
	for idx, gu := range player.GuInventory {
		if gu.InstanceID == req.GuInstanceID {
			targetIdx = idx
			break
		}
	}

	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này trong túi đồ"})
		return
	}

	targetGu := &player.GuInventory[targetIdx]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đã CHẾT, không thể sử dụng!"})
		return
	}

	// Fetch specs from dictionary
	specs, found := models.GuRegistry[targetGu.GuID]
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thông tin Cổ trùng trong thư viện"})
		return
	}

	if specs.Type != "Consumable" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này không thuộc loại Tiêu Hao, không thể sử dụng trực tiếp!"})
		return
	}

	logMsg := ""
	switch targetGu.GuID {
	case "GU_RELIC":
		if player.SubRank >= 4 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Xá Lợi Cổ không thể dùng để đột phá đại cảnh giới. Hãy hợp luyện đột phá!"})
			return
		}
		player.SubRank++

		stageNames := map[int]string{1: "Sơ Giai", 2: "Trung Giai", 3: "Cao Giai", 4: "Đỉnh Phong"}
		logMsg = fmt.Sprintf("Đột phá! Đạo hữu luyện hóa Xá Lợi Cổ, tu vi nâng cao lên %s %s!", getRankName(player.Rank), stageNames[player.SubRank])

	case "GU_STONE_APERTURE":
		player.PrimevalStones += 500
		player.MaxMana -= 15.0
		if player.MaxMana < 20.0 {
			player.MaxMana = 20.0
		}
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}
		logMsg = "Đạo hữu kích hoạt Thạch Khiếu Cổ, nhận ngay 500 Nguyên Thạch nhưng Không khiếu bị hóa đá một phần (giảm 15 Chân Nguyên tối đa)!"

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hiệu ứng của Cổ trùng tiêu hao này chưa được định nghĩa!"})
		return
	}

	// Delete from inventory
	newInventory := models.GuInventory{}
	for idx, item := range player.GuInventory {
		if idx != targetIdx {
			newInventory = append(newInventory, item)
		}
	}
	player.GuInventory = newInventory

	// Recalculate equipped_gus
	player.EquippedGus = []string{}
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			player.EquippedGus = append(player.EquippedGus, gu.InstanceID)
		}
	}

	_, err = tx.Exec(`
		UPDATE players 
		SET gu_inventory = $1, sub_rank = $2, primeval_stones = $3, max_mana = $4, current_mana = $5, equipped_gus = $6
		WHERE id = $7`,
		player.GuInventory, player.SubRank, player.PrimevalStones, player.MaxMana, player.CurrentMana, player.EquippedGus, player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state after using Gu"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": logMsg,
	})
}

// Helper function to get rank name in Vietnamese
func getRankName(rank int) string {
	names := map[int]string{
		1: "Nhất Chuyển",
		2: "Nhị Chuyển",
		3: "Tam Chuyển",
		4: "Tứ Chuyển",
		5: "Ngũ Chuyển",
		6: "Lục Chuyển",
		7: "Thất Chuyển",
		8: "Bát Chuyển",
		9: "Cửu Chuyển",
	}
	if name, ok := names[rank]; ok {
		return name
	}
	return fmt.Sprintf("%d Chuyển", rank)
}
