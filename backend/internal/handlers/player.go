package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
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
	Email          string `json:"email" binding:"required,email"`
	PhoneNumber    string `json:"phone_number" binding:"required"`
}

// CreatePlayerResponse represents the output payload for registration/login
type CreatePlayerResponse struct {
	Player models.Player `json:"player"`
}

const authCookieName = "gu_auth_token"
const authCookieMaxAge = 7 * 24 * 60 * 60

func authCookieDomain() string {
	return os.Getenv("AUTH_COOKIE_DOMAIN")
}

func authCookieSecure(c *gin.Context) bool {
	secureEnv := os.Getenv("AUTH_COOKIE_SECURE")
	if secureEnv != "" {
		return strings.EqualFold(secureEnv, "true")
	}

	return c.Request.TLS != nil
}

func setAuthCookie(c *gin.Context, token string) {
	sameSite := http.SameSiteLaxMode
	switch strings.ToLower(os.Getenv("AUTH_COOKIE_SAMESITE")) {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	}

	c.SetSameSite(sameSite)
	c.SetCookie(authCookieName, token, authCookieMaxAge, "/", authCookieDomain(), authCookieSecure(c), true)
}

func clearAuthCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(authCookieName, "", -1, "/", authCookieDomain(), authCookieSecure(c), true)
}

// RegisterPlayer handles registering a new player, hashing password, and rolling their aptitude
func RegisterPlayer(c *gin.Context) {
	var req CreatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	validRegions := map[string]bool{
		"Southern_Border":   true,
		"Central_Continent": true,
		"Northern_Plains":   true,
		"Western_Desert":    true,
		"Eastern_Sea":       true,
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

	// Map aptitude type to numeric grade (1 = A/Giáp .. 4 = D/Đinh)
	aptitudeGrade := 4
	switch aptitudeType {
	case "Giáp":
		aptitudeGrade = 1
	case "Ất":
		aptitudeGrade = 2
	case "Bính":
		aptitudeGrade = 3
	case "Đinh":
		aptitudeGrade = 4
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
			Equipped:   true,
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
		INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, lifespan, max_lifespan, time_speed_multiplier, email, phone_number)
		VALUES ($1, $2, 1, 1, $3, $4, $5, $6, $7, 0, 0, $8, $9, 0.0, 0.0, NULL, 0, 100.0, 100.0, '[]'::jsonb, 50, 50, '["50,50"]'::jsonb, '[]'::jsonb, '{}'::jsonb, $10, '{"strength": 0}'::jsonb, '{}'::jsonb, NULL, 100.0, 100.0, 1.0, $11, $12)
		RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, max_lifespan, time_speed_multiplier, party_id, arena_rank, email, phone_number
	`

	var activeCombatBytes []byte
	err = db.DB.QueryRow(query, req.Username, passwordHash, aptitudeType, aptitudePercent, aptitudeGrade, currentMana, maxMana, starterGu, starterMaterials, req.StartingRegion, req.Email, req.PhoneNumber).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.MaxLifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.PhoneNumber,
	)
	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu, Email hoặc Số điện thoại đã tồn tại trong Cổ Giới!"})
		return
	}

	// 4. Generate token
	token, err := auth.GenerateToken(player.ID, player.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access token"})
		return
	}
	setAuthCookie(c, token)

	c.JSON(http.StatusCreated, CreatePlayerResponse{
		Player: player,
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
		SELECT id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, max_lifespan, time_speed_multiplier, party_id, arena_rank, COALESCE(email, ''), COALESCE(phone_number, '')
		FROM players
		WHERE username = $1 OR email = $1 OR phone_number = $1
	`
	var activeCombatBytes []byte
	err := db.DB.QueryRow(query, req.Username).Scan(
		&player.ID, &player.Username, &passwordHash, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.MaxLifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.PhoneNumber,
	)
	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thông tin đăng nhập không hợp lệ!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu: " + err.Error()})
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
	setAuthCookie(c, token)

	c.JSON(http.StatusOK, CreatePlayerResponse{
		Player: player,
	})
}

// LogoutPlayer clears the auth cookie for the current browser session.
func LogoutPlayer(c *gin.Context) {
	clearAuthCookie(c)
	c.Status(http.StatusNoContent)
}

// GetCurrentPlayer returns the authenticated player from the httpOnly cookie session.
func GetCurrentPlayer(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid auth session"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to persist mana update"})
		return
	}

	c.JSON(http.StatusOK, player)
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

type RechargeWithStoneRequest struct {
	Stones int `json:"stones" binding:"required,min=1"`
}

// RechargeWithStones converts primeval stones into current mana for mortals
func RechargeWithStones(c *gin.Context) {
	var req RechargeWithStoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
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

	// Cổ Tiên (Rank 6+) sử dụng Tiên Nguyên Thạch, không nạp bằng Nguyên Thạch phàm tục
	if player.Rank >= 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ Tiên sử dụng Tiên Nguyên Thạch, không thể nạp bằng Nguyên Thạch phàm tục."})
		return
	}

	if player.CurrentMana >= player.MaxMana {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chân nguyên đã đầy, không cần nạp thêm."})
		return
	}

	// --- BẮT ĐẦU CHỈNH SỬA LOGIC MATCH VỚI TRUYỆN ---
	// Mốc cơ bản: 1 viên nguyên thạch hồi 10% Chân nguyên cho Cổ sư Nhất chuyển Sơ giai (Rank 1, SubRank 1)
	baseManaPerStone := 20.0

	// 1. Hệ số nén của Tiểu cảnh giới (Sơ, Trung, Cao, Đỉnh)
	// Nguyên tác: 2 thành Sơ giai luyện thành 1 thành Trung giai -> Hệ số x2 mỗi tiểu cảnh giới.
	subRankMultiplier := math.Pow(2.0, float64(player.SubRank-1))

	// 2. Hệ số nén của Đại cảnh giới (Nhất chuyển lên Nhị chuyển...)
	// Nguyên tác: Chênh lệch chất lượng chân nguyên giữa các Đại cảnh giới cực lớn, lấy mốc x10 lần.
	rankMultiplier := math.Pow(10.0, float64(player.Rank-1))

	// Lượng % chân nguyên thực tế mà 1 viên nguyên thạch hồi được ở Level hiện tại của Player
	manaPerStone := baseManaPerStone / (rankMultiplier * subRankMultiplier)
	// ------------------------------------------------

	neededMana := player.MaxMana - player.CurrentMana

	// Tính số Nguyên thạch TỐI ĐA cần dùng để sạc đầy Không khiếu
	maxUsableStones := int(math.Ceil(neededMana / manaPerStone))

	stonesToUse := req.Stones
	if stonesToUse > maxUsableStones {
		stonesToUse = maxUsableStones
	}

	if stonesToUse > player.PrimevalStones {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên thạch trong túi."})
		return
	}

	// Tính toán số chân nguyên cộng vào
	addedMana := float64(stonesToUse) * manaPerStone

	// Tránh việc nạp bị lố MaxMana do làm tròn math.Ceil
	if player.CurrentMana+addedMana > player.MaxMana {
		addedMana = player.MaxMana - player.CurrentMana
		player.CurrentMana = player.MaxMana
	} else {
		player.CurrentMana += addedMana
	}

	player.PrimevalStones -= stonesToUse
	player.LastManaUpdate = time.Now()

	updateQuery := `
		UPDATE players SET current_mana = $1, primeval_stones = $2, last_mana_update = $3
		WHERE id = $4
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.LastManaUpdate, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save recharge"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": fmt.Sprintf("Nạp %d nguyên thạch, khôi phục %.4f%% chân nguyên.", stonesToUse, addedMana),
	})
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

	// 2. Check energy cost. Mortals spend Chân Nguyên; Immortals spend a smaller Tiên Nguyên travel upkeep.
	energyCost := 10.0
	if isImmortalCultivator(&player) {
		energyCost = 0.5
	}
	if player.CurrentMana < energyCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}

	// 3. Deduct energy and roll reward stones
	player.CurrentMana -= energyCost
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
	chosenFood := ""
	for _, food := range foodOptions {
		if player.MaterialsInventory[food] > 0 {
			itemSpec, ok := models.ItemRegistry[food]
			if !ok {
				continue
			}
			// Check if item is food
			isFood := itemSpec.Type == "Food"
			if !isFood {
				for _, t := range itemSpec.Tags {
					if t == "food" {
						isFood = true
						break
					}
				}
			}
			if isFood {
				chosenFood = food
				break
			}
		}
	}
	if chosenFood == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không có thức ăn phù hợp cho %s. Cần một trong: %v", specs.Name, foodOptions)})
		return
	}

	// 5. Consume 1 food and restore hunger
	player.MaterialsInventory[chosenFood]--
	if player.MaterialsInventory[chosenFood] <= 0 {
		delete(player.MaterialsInventory, chosenFood)
	}
	targetGu.Hunger = specs.MaxHunger
	targetGu.Status = "ACTIVE" // Wakes up from sleep (hibernation)

	// Vital Gu must always be equipped and active
	if targetGu.IsVital {
		targetGu.Equipped = true
	}

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
		SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, max_lifespan, time_speed_multiplier, party_id, arena_rank, COALESCE(email, ''), COALESCE(phone_number, '')
		FROM players
		WHERE id = $1
		FOR UPDATE
	`
	var activeCombatBytes []byte
	err := tx.QueryRow(query, id).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.MaxLifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.PhoneNumber,
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

	// Initialize defaults for new fields if not set
	if player.MaxLifespan <= 0 {
		player.MaxLifespan = 100.0
	}
	if player.Lifespan <= 0 && player.MaxLifespan > 0 {
		// Only initialize lifespan if it was never set (i.e. <= 0)
		player.Lifespan = player.MaxLifespan
	}
	if player.TimeSpeedMultiplier <= 0 {
		if player.Rank >= 6 {
			player.TimeSpeedMultiplier = 10.0
		} else {
			player.TimeSpeedMultiplier = 1.0
		}
	}

	poolChanged := normalizeEnergyPool(&player)
	regenRatePerHour := energyRegenRatePerHour(&player)

	// Lão hóa: Giảm 50% tốc độ hồi mana nếu thọ nguyên cạn kiệt
	if player.Lifespan <= 0 {
		regenRatePerHour = regenRatePerHour * 0.5
	}

	now := time.Now()
	secondsPassed := now.Sub(player.LastManaUpdate).Seconds()

	if secondsPassed > 0 {
		// Subjective time speed passed
		subjectiveSeconds := secondsPassed * player.TimeSpeedMultiplier
		earnedMana := (subjectiveSeconds / 3600.0) * regenRatePerHour
		newMana := player.CurrentMana + earnedMana
		if newMana > player.MaxMana {
			newMana = player.MaxMana
		}

		if newMana != player.CurrentMana {
			player.CurrentMana = newMana
			poolChanged = true
		}

		// Lifespan decay (1 game year = 3600 subjective seconds)
		yearsPassed := subjectiveSeconds / 3600.0
		player.Lifespan -= yearsPassed
		if player.Lifespan < 0 {
			player.Lifespan = 0
		}
		poolChanged = true
	}

	if poolChanged {
		player.LastManaUpdate = now

		updateQuery := `
			UPDATE players
			SET current_mana = $1, max_mana = $2, last_mana_update = $3, lifespan = $4, max_lifespan = $5, time_speed_multiplier = $6
			WHERE id = $7
		`
		_, err = tx.Exec(updateQuery, player.CurrentMana, player.MaxMana, player.LastManaUpdate, player.Lifespan, player.MaxLifespan, player.TimeSpeedMultiplier, player.ID)
		if err != nil {
			return player, err
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

			// Ràng buộc Bí cảnh (Secret Realm): Chỉ dùng Moon Path hoặc Earth Path
			if player.CurrentRegion == "three_kings_inheritance" {
				if spec.Path != "Moon Path" && spec.Path != "Earth Path" {
					continue
				}
			}

			for effectKey := range spec.Effects {
				if handler, ok := StatEffectRegistry[effectKey]; ok {
					handler(player, &spec, stats)
				}
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

	// Vital Gu is always equipped and active — cannot be unequipped
	if targetGu.IsVital && !req.Equip {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bản Mệnh Cổ luôn luôn được trang bị, không thể tháo ra."})
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

	// Vital Gu is always ACTIVE — cannot be put to sleep or sealed
	if targetGu.IsVital && req.Status != "ACTIVE" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bản Mệnh Cổ luôn luôn ở trạng thái ACTIVE, không thể thay đổi."})
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

	item, ok := models.ItemRegistry[req.ItemID]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm không tồn tại trong ItemRegistry"})
		return
	}

	if item.Type != "Consumable" && item.Type != "SoulResource" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vật phẩm này không thể sử dụng trực tiếp"})
		return
	}

	messages := []string{}

	// 1. Validation phase
	for effectKey := range item.Effects {
		if valHandler, ok := ValidateItemEffects[effectKey]; ok {
			_, err := valHandler(&player, item)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
		}
	}

	// 2. Application phase
	for effectKey := range item.Effects {
		if appHandler, ok := ApplyItemEffects[effectKey]; ok {
			msg, err := appHandler(&player, item)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if msg != "" {
				messages = append(messages, msg)
			}
		}
	}

	if len(messages) == 0 {
		messages = append(messages, fmt.Sprintf("Đạo hữu sử dụng thành công %s.", item.Name))
	}

	player.MaterialsInventory[req.ItemID]--
	if player.MaterialsInventory[req.ItemID] <= 0 {
		delete(player.MaterialsInventory, req.ItemID)
	}

	_, err = tx.Exec(`
		UPDATE players 
		SET materials_inventory = $1, current_mana = $2, max_mana = $3, hp = $4, base_hp = $5, dao_marks = $6
		WHERE id = $7`,
		player.MaterialsInventory, player.CurrentMana, player.MaxMana, player.HP, player.BaseHP, player.DaoMarks, player.ID,
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
		"message": strings.Join(messages, " "),
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

	// Update is_vital flags; vital Gu is always ACTIVE and equipped
	for idx := range player.GuInventory {
		if idx == targetIndex {
			player.GuInventory[idx].IsVital = true
			player.GuInventory[idx].Status = "ACTIVE"
			player.GuInventory[idx].Equipped = true
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
	specs, found := models.GuRegistry[targetGu.GuID]
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thông tin Cổ trùng trong thư viện"})
		return
	}

	if specs.Type != "Consumable" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này không thuộc loại Tiêu Hao, không thể sử dụng trực tiếp!"})
		return
	}

	if err := ValidateGuUsable(&player, targetGu, &specs, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cost := calculateEnergyCost(&player, &specs)
	player.CurrentMana -= cost

	var logs []string
	for effectKey := range specs.Effects {
		if handler, ok := ConsumableEffectRegistry[effectKey]; ok {
			msg, err := handler(&player, *targetGu, specs)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if msg != "" {
				logs = append(logs, msg)
			}
		}
	}

	if len(logs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hiệu ứng của Cổ trùng tiêu hao này chưa được định nghĩa hoặc không kích hoạt thành công!"})
		return
	}

	logMsg := strings.Join(logs, " ")

	// Delete from inventory
	RemoveGuFromInventory(&player, targetGu.InstanceID)

	_, err = tx.Exec(`
		UPDATE players 
		SET gu_inventory = $1, sub_rank = $2, rank = $3, primeval_stones = $4, max_mana = $5, current_mana = $6, equipped_gus = $7, dao_marks = $8, wall_progress = $9, hp = $10
		WHERE id = $11`,
		player.GuInventory, player.SubRank, player.Rank, player.PrimevalStones, player.MaxMana, player.CurrentMana, player.EquippedGus, player.DaoMarks, player.WallProgress, player.HP, player.ID,
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

// UseSupportGuRequest represents request payload to use a support Gu out of combat
type UseSupportGuRequest struct {
	GuInstanceID string `json:"gu_instance_id" binding:"required"`
}

// UseSupportGu activates an out-of-combat support Gu bug's effect
func UseSupportGu(c *gin.Context) {
	var req UseSupportGuRequest
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
	specs, found := models.GuRegistry[targetGu.GuID]
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không tìm thấy thông tin Cổ trùng trong thư viện"})
		return
	}

	if specs.Type != "Support" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này không thuộc loại Hỗ Trợ, không thể sử dụng trực tiếp ngoài combat!"})
		return
	}

	if err := ValidateGuUsable(&player, targetGu, &specs, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cost := calculateEnergyCost(&player, &specs)
	player.CurrentMana -= cost

	var logs []string
	for effectKey := range specs.Effects {
		if handler, ok := OutCombatSupportRegistry[effectKey]; ok {
			msg, err := handler(&player, specs)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if msg != "" {
				logs = append(logs, msg)
			}
		}
	}

	if len(logs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng hỗ trợ này không có hiệu ứng ngoài chiến đấu!"})
		return
	}

	logMsg := strings.Join(logs, " ")

	// Save to DB
	_, err = tx.Exec(`
		UPDATE players 
		SET gu_inventory = $1, max_mana = $2, current_mana = $3, aptitude_percent = $4, hp = $5
		WHERE id = $6`,
		player.GuInventory, player.MaxMana, player.CurrentMana, player.AptitudePercent, player.HP, player.ID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player state after using support Gu"})
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
