package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// CreatePlayerRequest represents the incoming registration payload
type CreatePlayerRequest struct {
	Username       string `json:"username" binding:"required,min=3,max=30"`
	Password       string `json:"password" binding:"required,min=6,max=50"`
	StartingRegion string `json:"starting_region" binding:"required"`
	SetVitalNow    bool   `json:"set_vital_now"`
	Email          string `json:"email" binding:"required,email"`
	ReferralCode   string `json:"referral_code"`
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

func authCookieSecure(c *gin.Context, isSameSiteNone bool) bool {
	secureEnv := os.Getenv("AUTH_COOKIE_SECURE")
	if secureEnv != "" {
		return strings.EqualFold(secureEnv, "true")
	}

	if isSameSiteNone {
		return true // SameSite=None strictly requires Secure=true in modern browsers
	}

	return c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https"
}

func setAuthCookie(c *gin.Context, token string) {
	sameSite := http.SameSiteLaxMode

	// Auto-detect cross-origin requirements
	origin := c.Request.Header.Get("Origin")
	corsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	if (origin != "" && !strings.Contains(origin, "localhost") && !strings.Contains(origin, c.Request.Host)) ||
		(corsEnv != "" && !strings.Contains(corsEnv, "localhost") && corsEnv != "*") {
		sameSite = http.SameSiteNoneMode
	}

	switch strings.ToLower(os.Getenv("AUTH_COOKIE_SAMESITE")) {
	case "strict":
		sameSite = http.SameSiteStrictMode
	case "none":
		sameSite = http.SameSiteNoneMode
	case "lax":
		sameSite = http.SameSiteLaxMode
	}

	c.SetSameSite(sameSite)
	c.SetCookie(authCookieName, token, authCookieMaxAge, "/", authCookieDomain(), authCookieSecure(c, sameSite == http.SameSiteNoneMode), true)
}

func clearAuthCookie(c *gin.Context) {
	sameSite := http.SameSiteLaxMode
	origin := c.Request.Header.Get("Origin")
	corsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	if (origin != "" && !strings.Contains(origin, "localhost") && !strings.Contains(origin, c.Request.Host)) ||
		(corsEnv != "" && !strings.Contains(corsEnv, "localhost") && corsEnv != "*") {
		sameSite = http.SameSiteNoneMode
	}

	c.SetSameSite(sameSite)
	c.SetCookie(authCookieName, "", -1, "/", authCookieDomain(), authCookieSecure(c, sameSite == http.SameSiteNoneMode), true)
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

	// Email format verification
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng Email không hợp lệ!"})
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

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Verify email uniqueness inside tx
	var emailExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE email = $1)", req.Email).Scan(&emailExists)
	if err == nil && emailExists {
		c.JSON(http.StatusConflict, gin.H{"error": "Email này đã được đăng ký trong Cổ Giới!"})
		return
	}

	// Verify username uniqueness inside tx
	var usernameExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE username = $1)", req.Username).Scan(&usernameExists)
	if err == nil && usernameExists {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu này đã tồn tại trong Cổ Giới!"})
		return
	}

	// Generate unique referral code
	refCode := "REF-" + strings.ToUpper(uuid.New().String()[:8])

	var player models.Player
	query := `
		INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, lifespan, max_lifespan, time_speed_multiplier, email, completed_quests, temp_loot, explore_count, last_explore_reset, soul_hp, max_soul, soul_tier, needs_soul_refine, unlocked_recipes, bio, avatar_url, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, ref_code)
		VALUES ($1, $2, 1, 1, $3, $4, $5, $6, $7, 0, 0, $8, $9, 0.0, 0.0, NULL, 0, 100.0, 100.0, '[]'::jsonb, 50, 50, '["50,50"]'::jsonb, '[]'::jsonb, '{}'::jsonb, $10, '{"strength": 0}'::jsonb, '{}'::jsonb, NULL, 100.0, 100.0, 1.0, $11, '[]'::jsonb, '{"gu_inventory": [], "materials_inventory": {}, "primeval_stones": 0}'::jsonb, 0, NOW(), 10, 10, 'NHAN_HON', false, '[{"recipe_id": "recipe_moon_raiment", "completion_rate": 100.0}, {"recipe_id": "recipe_white_jade", "completion_rate": 50.0}, {"recipe_id": "recipe_four_flavors_liquor", "completion_rate": 30.0}]'::jsonb, 'Hãy viết gì đó về bản thân...', '', 1, 1, '', false, $12)
		RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, max_lifespan, time_speed_multiplier, party_id, arena_rank, email, google_id, completed_quests, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, unlocked_recipes, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, ref_code, avatar_url
	`

	var activeCombatBytes []byte
	err = tx.QueryRow(query, req.Username, passwordHash, aptitudeType, aptitudePercent, aptitudeGrade, currentMana, maxMana, starterGu, starterMaterials, req.StartingRegion, req.Email, refCode).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.MaxLifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.GoogleID, &player.CompletedQuests, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.UnlockedRecipes, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.AvatarURL,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create player: " + err.Error()})
		return
	}

	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	// 3.5 Handle referral code logic
	trimmedRef := strings.TrimSpace(req.ReferralCode)
	if trimmedRef != "" {
		var referrerID uuid.UUID
		err = tx.QueryRow("SELECT id FROM players WHERE ref_code = $1", trimmedRef).Scan(&referrerID)
		if err == nil {
			// Do not allow self referral
			if referrerID != player.ID {
				clientIP := c.ClientIP()
				userAgent := c.Request.UserAgent()
				var count int
				// Flag as suspicious if the same client IP and device is already referred
				_ = tx.QueryRow("SELECT COUNT(*) FROM referrals WHERE referee_ip = $1 AND referee_device = $2", clientIP, userAgent).Scan(&count)
				isSuspicious := count > 0

				_, err = tx.Exec(`
					INSERT INTO referrals (referrer_id, referee_id, referee_ip, referee_device, is_suspicious)
					VALUES ($1, $2, $3, $4, $5)
				`, referrerID, player.ID, clientIP, userAgent, isSuspicious)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record referral: " + err.Error()})
					return
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player record: " + err.Error()})
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

// LoginRequest represents credentials login payload.
// Email is the canonical login identifier; Username is accepted only as a
// backwards-compatible JSON field for older clients that already send an email
// value under "username".
type LoginRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginPlayer validates cultivator credentials and signs a JWT token
func LoginPlayer(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email và mật khẩu là bắt buộc"})
		return
	}

	loginEmail := strings.ToLower(strings.TrimSpace(req.Email))
	if loginEmail == "" {
		loginEmail = strings.ToLower(strings.TrimSpace(req.Username))
	}
	if loginEmail == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email và mật khẩu là bắt buộc"})
		return
	}
	if !isValidEmail(loginEmail) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng đăng nhập bằng email hợp lệ"})
		return
	}

	var player models.Player
	var passwordHash string

	query := `
		SELECT id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, max_lifespan, time_speed_multiplier, party_id, arena_rank, COALESCE(email, ''), google_id, completed_quests, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, unlocked_recipes, COALESCE(bio, ''), COALESCE(fake_rank, 1), COALESCE(fake_sub_rank, 1), COALESCE(fake_vital_gu, ''), COALESCE(is_disguised, false), COALESCE(is_admin, false), COALESCE(avatar_url, '')
		FROM players
		WHERE LOWER(email) = $1
	`
	var activeCombatBytes []byte
	err := db.DB.QueryRow(query, loginEmail).Scan(
		&player.ID, &player.Username, &passwordHash, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.MaxLifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.GoogleID, &player.CompletedQuests, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.UnlockedRecipes, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.AvatarURL,
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

// GetPlayerStatus retrieves a projected player status for UI refresh without persisting lazy state.
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

	player, err := projectPlayerStatusTx(db.DB, playerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to project player status: " + err.Error()})
		}
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
		if player.ImmortalStones < float64(req.Stones) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Tiên Nguyên Thạch."})
			return
		}

		neededMana := player.MaxMana - player.CurrentMana
		if neededMana <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tiên Nguyên Hải đã đầy."})
			return
		}

		stonesToUse := int(math.Min(float64(req.Stones), math.Ceil(neededMana)))
		if stonesToUse <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng Tiên Nguyên Thạch nạp không hợp lệ."})
			return
		}

		player.ImmortalStones -= float64(stonesToUse)
		player.CurrentMana += float64(stonesToUse)
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}

		_, err = tx.Exec(`
			UPDATE players 
			SET current_mana = $1, immortal_stones = $2, last_mana_update = NOW()
			WHERE id = $3
		`, player.CurrentMana, player.ImmortalStones, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Tiên Nguyên"})
			return
		}

		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2`, int(player.CurrentMana), player.ID)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Luyện hóa thành công %d Tiên Nguyên Thạch thành %d hạt Tiên Nguyên!", stonesToUse, stonesToUse),
			"player":  player,
		})
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

	// Roll 5% chance to find a recipe (Tàn Phương)
	rollRecipe := rand.Intn(100)
	if rollRecipe < 5 {
		// Fetch all recipe IDs from refining_recipes
		var allRecipeIDs []string
		rows, err := tx.Query(`SELECT id FROM refining_recipes`)
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
			unlockedMap := make(map[string]float64)
			for _, ur := range player.UnlockedRecipes {
				unlockedMap[ur.RecipeID] = ur.CompletionRate
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
				found := false
				for i, ur := range player.UnlockedRecipes {
					if ur.RecipeID == chosenCandidate {
						if ur.CompletionRate < randomRate {
							player.UnlockedRecipes[i].CompletionRate = randomRate
						}
						found = true
						break
					}
				}
				if !found {
					player.UnlockedRecipes = append(player.UnlockedRecipes, models.UnlockedRecipe{
						RecipeID:       chosenCandidate,
						CompletionRate: randomRate,
					})
				}

				// Find target Gu name
				var targetGuName string
				_ = tx.QueryRow(`
					SELECT name FROM gu_dictionary 
					WHERE id = (SELECT target_gu_id FROM refining_recipes WHERE id = $1)
				`, chosenCandidate).Scan(&targetGuName)

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
	updateQuery := `
		UPDATE players
		SET current_mana = $1, primeval_stones = $2, materials_inventory = $3, gu_inventory = $4, last_mana_update = $5, unlocked_recipes = $6, explored_cells = $7
		WHERE id = $8
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.MaterialsInventory, player.GuInventory, player.LastManaUpdate, player.UnlockedRecipes, player.ExploredCells, player.ID)
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

	foodName := chosenFood
	if itemSpec, ok := models.ItemRegistry[chosenFood]; ok {
		foodName = itemSpec.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": fmt.Sprintf("Cho %s ăn thành công! Tiêu hao 1 %s.", specs.Name, foodName),
	})
}

// FeedGroupRequest represents payload for feeding all hungry Gu bugs of the same type
type FeedGroupRequest struct {
	GuID string `json:"gu_id" binding:"required"`
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

	// 2. Find static specs in Dictionary
	specs, found := models.GuRegistry[req.GuID]
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gu definition not found in registry"})
		return
	}

	foodOptions := specs.FoodTypes
	if len(foodOptions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This Gu bug does not require any food or has no food defined"})
		return
	}

	// Helper to find food from materials inventory
	getAvailableFood := func() string {
		for _, food := range foodOptions {
			if player.MaterialsInventory[food] > 0 {
				itemSpec, ok := models.ItemRegistry[food]
				if !ok {
					continue
				}
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
					return food
				}
			}
		}
		return ""
	}

	fedCount := 0
	foodConsumed := make(map[string]int)

	// 3. Iterate over the player's Gu inventory and feed hungry ones of matching type
	for i := range player.GuInventory {
		gu := &player.GuInventory[i]
		if gu.GuID == req.GuID && gu.Status != "DEAD" && gu.Status != "SEALED" && gu.Hunger < specs.MaxHunger {
			chosenFood := getAvailableFood()
			if chosenFood == "" {
				break // No more food available
			}

			// Consume 1 food
			player.MaterialsInventory[chosenFood]--
			if player.MaterialsInventory[chosenFood] <= 0 {
				delete(player.MaterialsInventory, chosenFood)
			}
			foodConsumed[chosenFood]++

			// Restore hunger
			gu.Hunger = specs.MaxHunger
			gu.Status = "ACTIVE" // Wakes up from sleep (hibernation)
			if gu.IsVital {
				gu.Equipped = true
			}
			fedCount++
		}
	}

	if fedCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có Cổ trùng nào cần cho ăn hoặc không có đủ thức ăn phù hợp"})
		return
	}

	// 4. Update database
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

	// Build success message
	var foodDetails []string
	for foodID, qty := range foodConsumed {
		foodName := foodID
		if itemSpec, ok := models.ItemRegistry[foodID]; ok {
			foodName = itemSpec.Name
		}
		foodDetails = append(foodDetails, fmt.Sprintf("%dx %s", qty, foodName))
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": fmt.Sprintf("Đã cho ăn thành công %d con %s! Tiêu hao: %s.", fedCount, specs.Name, strings.Join(foodDetails, ", ")),
	})
}

type playerQueryRower interface {
	QueryRow(query string, args ...any) *sql.Row
}

// loadPlayerForStatus loads the complete player row used by mana/status projection.
// Pass forUpdate=true only for authoritative mutation flows that must lock before writing.
func loadPlayerForStatus(q playerQueryRower, id uuid.UUID, forUpdate bool) (models.Player, error) {
	var player models.Player

	query := `
		SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, max_lifespan, time_speed_multiplier, party_id, arena_rank, COALESCE(email, ''), google_id, completed_quests, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, unlocked_recipes, COALESCE(bio, ''), COALESCE(fake_rank, 1), COALESCE(fake_sub_rank, 1), COALESCE(fake_vital_gu, ''), COALESCE(is_disguised, false), COALESCE(is_admin, false), COALESCE(ref_code, ''), COALESCE(avatar_url, '')
		FROM players
		WHERE id = $1
	`
	if forUpdate {
		query += "\t\tFOR UPDATE\n"
	}
	var activeCombatBytes []byte
	err := q.QueryRow(query, id).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.MaxLifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.GoogleID, &player.CompletedQuests, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.UnlockedRecipes, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.AvatarURL,
	)
	if err != nil {
		return player, err
	}
	if len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	return player, nil
}

func setDefaultPlayerStatusFields(player *models.Player) {
	// Initialize default unlocked recipes if empty. Projection callers keep this in memory only.
	if len(player.UnlockedRecipes) == 0 {
		player.UnlockedRecipes = models.UnlockedRecipesList{
			{RecipeID: "recipe_moon_raiment", CompletionRate: 100.0},
			{RecipeID: "recipe_white_jade", CompletionRate: 50.0},
			{RecipeID: "recipe_four_flavors_liquor", CompletionRate: 30.0},
		}
	}

	// Initialize defaults for new fields if not set.
	if player.MaxLifespan <= 0 {
		player.MaxLifespan = 100.0
	}
	if player.Lifespan <= 0 && player.MaxLifespan > 0 {
		// Only initialize lifespan if it was never set (i.e. <= 0).
		player.Lifespan = player.MaxLifespan
	}
	if player.TimeSpeedMultiplier <= 0 {
		if player.Rank >= 6 {
			player.TimeSpeedMultiplier = 10.0
		} else {
			player.TimeSpeedMultiplier = 1.0
		}
	}
}

func applyProjectedStatus(player *models.Player, essenceProd int, now time.Time) (bool, bool) {
	poolChanged := normalizeEnergyPool(player)
	stressChanged := false

	var regenRatePerHour float64
	if player.Rank >= 6 {
		subRank := player.SubRank
		if subRank < 1 {
			subRank = 1
		}
		if subRank > 4 {
			subRank = 4
		}
		subRankFactor := 1.0 + float64(subRank-1)*0.15
		regenRatePerHour = (float64(essenceProd) / 120.0) * subRankFactor
	} else {
		regenRatePerHour = energyRegenRatePerHour(player)
	}

	// Lão hóa: Giảm 50% tốc độ hồi mana nếu thọ nguyên cạn kiệt.
	if player.Lifespan <= 0 {
		regenRatePerHour = regenRatePerHour * 0.5
	}

	secondsPassed := now.Sub(player.LastManaUpdate).Seconds()
	if secondsPassed > 0 {
		// Subjective time speed passed.
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

		// HP Regeneration: 5% of BaseHP per hour + healing stat
		if player.HP > 0 && player.HP < player.BaseHP {
			stats := CalculatePlayerStats(player)
			healingStat := stats["healing"]
			regenHPPerHour := (player.BaseHP * 0.05) + healingStat
			earnedHP := (subjectiveSeconds / 3600.0) * regenHPPerHour
			newHP := player.HP + earnedHP
			if newHP > player.BaseHP {
				newHP = player.BaseHP
			}
			if newHP != player.HP {
				player.HP = newHP
				poolChanged = true
			}
		}

		// Lifespan decay (1 game year = 5 days = 120 subjective hours = 432000 subjective seconds).
		yearsPassed := subjectiveSeconds / 432000.0
		player.Lifespan -= yearsPassed
		if player.Lifespan < 0 {
			player.Lifespan = 0
		}
		poolChanged = true
	}

	// Real-time Aperture Stress Decay: 10.0% per hour in real time.
	if player.ApertureStress > 0 && !player.LastCultivateTime.IsZero() {
		elapsedHours := now.Sub(player.LastCultivateTime).Hours()
		if elapsedHours > 0 {
			decay := elapsedHours * 10.0
			newStress := player.ApertureStress - decay
			if newStress < 0 {
				newStress = 0
			}
			if newStress != player.ApertureStress {
				player.ApertureStress = newStress
				poolChanged = true
				stressChanged = true
			}
		}
	}

	return poolChanged, stressChanged
}

func applyImmortalApertureProjection(q playerQueryRower, player *models.Player) (int, bool) {
	essenceProd := 10
	if player.Rank < 6 {
		return essenceProd, false
	}

	var timeRate float64
	if err := q.QueryRow(`SELECT time_rate, essence_production_per_year FROM immortal_apertures WHERE player_id = $1`, player.ID).Scan(&timeRate, &essenceProd); err == nil && timeRate > 0 {
		if player.TimeSpeedMultiplier != timeRate {
			player.TimeSpeedMultiplier = timeRate
			return essenceProd, true
		}
	}

	return essenceProd, false
}

// projectPlayerStatusTx computes a read-only projected status for UI refreshes.
// It intentionally uses SELECT without FOR UPDATE and never persists current_mana,
// lifespan, aperture_stress, last_cultivate_time, or players.last_mana_update.
func projectPlayerStatusTx(q playerQueryRower, id uuid.UUID) (models.Player, error) {
	player, err := loadPlayerForStatus(q, id, false)
	if err != nil {
		return player, err
	}

	setDefaultPlayerStatusFields(&player)
	essenceProd, _ := applyImmortalApertureProjection(q, &player)
	applyProjectedStatus(&player, essenceProd, time.Now())

	return player, nil
}

// evaluateManaTx is the authoritative status evaluator for mutation flows.
// Keep this for actions that need a locked, authoritative status before mutating DB
// (cultivate, recharge, combat, explore, refine, login/logout, or periodic sync).
func evaluateManaTx(tx *sql.Tx, id uuid.UUID) (models.Player, error) {
	player, err := loadPlayerForStatus(tx, id, true)
	if err != nil {
		return player, err
	}

	needsRecipePersist := len(player.UnlockedRecipes) == 0
	setDefaultPlayerStatusFields(&player)
	if needsRecipePersist {
		_, _ = tx.Exec(`UPDATE players SET unlocked_recipes = $1 WHERE id = $2`, player.UnlockedRecipes, player.ID)
	}

	essenceProd, poolChanged := applyImmortalApertureProjection(tx, &player)
	now := time.Now()
	projectedChanged, stressChanged := applyProjectedStatus(&player, essenceProd, now)
	if projectedChanged {
		poolChanged = true
		player.LastManaUpdate = now
	}
	if stressChanged {
		player.LastCultivateTime = now
	}

	if poolChanged {
		updateQuery := `
			UPDATE players
			SET current_mana = $1, max_mana = $2, last_mana_update = $3, lifespan = $4, max_lifespan = $5, time_speed_multiplier = $6, aperture_stress = $7, last_cultivate_time = $8, hp = $9
			WHERE id = $10
		`
		_, err = tx.Exec(updateQuery, player.CurrentMana, player.MaxMana, player.LastManaUpdate, player.Lifespan, player.MaxLifespan, player.TimeSpeedMultiplier, player.ApertureStress, player.LastCultivateTime, player.HP, player.ID)
		if err != nil {
			return player, err
		}
	}

	if player.Rank >= 6 {
		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2`, int(player.CurrentMana), player.ID)
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
			stats["defense"] += 100.0
		case "Jade Skin Mod":
			stats["defense"] += 40.0
		case "Copper Skin Mod":
			stats["defense"] += 20.0
		case "White Boar Strength":
			stats["attack"] += 15.0
		case "Black Boar Strength":
			stats["attack"] += 15.0
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

			for effectKey := range models.GetGuEffectSection(spec, "passive") {
				if handler, ok := StatEffectRegistry[effectKey]; ok {
					handler(player, &spec, stats)
				}
			}
		}
	}

	// Add storage capacity bonus based on dynamic strength/attack
	stats["storage"] += math.Floor(stats["attack"] * 0.5)

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

	guName := targetGu.GuID
	if spec, ok := models.GuRegistry[targetGu.GuID]; ok {
		guName = spec.Name
	}

	statusMsg := fmt.Sprintf("Tháo %s thành công!", guName)
	if req.Equip {
		statusMsg = fmt.Sprintf("Trang bị %s thành công!", guName)
	}

	c.JSON(http.StatusOK, gin.H{
		"player":  player,
		"message": statusMsg,
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

			// If it's a regen item, set Redis cache
			if effectKey == "regen_hp_out_of_combat" {
				redisKey := fmt.Sprintf("regen_buff:%s", player.ID.String())
				cache.SetJSON(redisKey, item.ID, 10*time.Minute)
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
	for effectKey := range models.GetGuEffectSection(specs, "active") {
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
	for effectKey := range models.GetGuEffectSection(specs, "support") {
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

// UpdateProfileSettingsRequest represents settings data to update
type UpdateProfileSettingsRequest struct {
	Username    string `json:"username"`
	Bio         string `json:"bio"`
	NewPassword string `json:"new_password"`
	FakeRank    int    `json:"fake_rank"`
	FakeSubRank int    `json:"fake_sub_rank"`
	FakeVitalGu string `json:"fake_vital_gu"`
	IsDisguised bool   `json:"is_disguised"`
}

// UpdateProfileSettings updates the current user's profile settings (email, bio, password, fake profile)
func UpdateProfileSettings(c *gin.Context) {
	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateProfileSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// Validate Disguise requirements if player wants to activate or configure disguise
	hasConcealment := false
	hasAttitude := false
	hasFormChange := false
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			switch gu.GuID {
			case "GU_SHADOW", "GU_SHADOW_STEPS", "GU_STEALTH_SCALES":
				hasConcealment = true
			case "GU_CHANGE_FORM":
				hasConcealment = true
				hasFormChange = true
			case "GU_ATTITUDE":
				hasConcealment = true
				hasAttitude = true
			}
		}
	}

	if req.IsDisguised {
		if !hasConcealment {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không trang bị cổ trùng ngụy trang nào hoạt động! Cần trang bị các cổ trùng như Ẩn Lân Cổ, Biến Hóa Cổ, Thái Độ Cổ..."})
			return
		}

		if req.FakeRank != player.Rank {
			if !hasAttitude {
				// Standard stealth only allows hiding/lowering rank ("giả yếu")
				if req.FakeRank > player.Rank || req.FakeRank < 1 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng hiện tại chỉ hỗ trợ giấu khí tức làm yếu đi, không thể giả làm cường giả mạnh hơn tu vi bản thân! Chỉ có Thái Độ Cổ mới đạt tới mức ngụy trang vô ngại."})
					return
				}
			} else {
				if req.FakeRank < 1 || req.FakeRank > 9 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Tu vi giả chỉ có thể từ Nhất Chuyển (1) đến Cửu Chuyển (9)!"})
					return
				}
			}
		}

		if req.FakeVitalGu != "" {
			if !hasAttitude && !hasFormChange {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Cần trang bị Biến Hóa Cổ hoặc Thái Độ Cổ để tạo dựng Bản Mệnh Cổ giả!"})
				return
			}
			// Optional: verify they own this Gu or it is a valid Gu ID in registry
			found := false
			for _, gu := range player.GuInventory {
				if gu.GuID == req.FakeVitalGu {
					found = true
					break
				}
			}
			if req.FakeVitalGu != "GU_SHADOW" && !found {
				// Allow standard fakes or anything they actually own
				c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không sở hữu cổ trùng " + req.FakeVitalGu + " để thiết lập ngụy trang!"})
				return
			}
		}
	}

	username := strings.TrimSpace(req.Username)
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu không được để trống!"})
		return
	}

	if len(username) < 3 || len(username) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu phải từ 3 đến 30 ký tự!"})
		return
	}

	for _, r := range username {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) && !unicode.IsDigit(r) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu chỉ được chứa chữ cái, số và khoảng trắng, không chứa ký tự đặc biệt!"})
			return
		}
	}

	if username != player.Username {
		var exists bool
		err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM players WHERE username = $1 AND id <> $2)`, username, player.ID).Scan(&exists)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu này đã được sử dụng bởi cổ sư khác!"})
			return
		}
		player.Username = username
	}

	player.Bio = req.Bio
	player.FakeRank = req.FakeRank
	player.FakeSubRank = req.FakeSubRank
	player.FakeVitalGu = req.FakeVitalGu
	player.IsDisguised = req.IsDisguised

	if req.NewPassword != "" {
		if len(req.NewPassword) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mật khẩu mới phải có ít nhất 6 ký tự"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi mã hóa mật khẩu"})
			return
		}
		_, err = tx.Exec(`UPDATE players SET username = $1, bio = $2, fake_rank = $3, fake_sub_rank = $4, fake_vital_gu = $5, is_disguised = $6, password_hash = $7 WHERE id = $8`, player.Username, player.Bio, player.FakeRank, player.FakeSubRank, player.FakeVitalGu, player.IsDisguised, string(hash), player.ID)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu đã được sử dụng bởi cổ sư khác!"})
			return
		}
	} else {
		_, err = tx.Exec(`UPDATE players SET username = $1, bio = $2, fake_rank = $3, fake_sub_rank = $4, fake_vital_gu = $5, is_disguised = $6 WHERE id = $7`, player.Username, player.Bio, player.FakeRank, player.FakeSubRank, player.FakeVitalGu, player.IsDisguised, player.ID)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu đã được sử dụng bởi cổ sư khác!"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save profile settings"})
		return
	}

	// Update token cookie to keep session claim synchronized with the new username
	newToken, err := auth.GenerateToken(player.ID, player.Username)
	if err == nil {
		setAuthCookie(c, newToken)
	}

	c.JSON(http.StatusOK, gin.H{"player": player, "message": "Cập nhật hồ sơ thành công!"})
}

func getClientIDFromGoogleJSON() string {
	candidates := []string{
		"google.json",
		"backend/google.json",
		"../google.json",
		"../../google.json",
	}
	var data []byte
	var err error
	for _, p := range candidates {
		data, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}
	if len(data) == 0 {
		return ""
	}
	var config struct {
		ClientID string `json:"client_id"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return ""
	}
	return config.ClientID
}

// GoogleLogin redirects user to Google OAuth2 consent page
func GoogleLogin(c *gin.Context) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		clientID = getClientIDFromGoogleJSON()
	}
	if clientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Client ID is not configured"})
		return
	}

	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:8080/api/auth/google/callback"
	}

	state := "google_oauth_state_secret"
	refCode := c.Query("ref")
	if refCode != "" {
		state = "ref_" + refCode
	}
	googleURL := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+profile+email&state=%s",
		clientID, url.QueryEscape(redirectURI), state)

	c.Redirect(http.StatusTemporaryRedirect, googleURL)
}

// GoogleCallback handles authorization callback from Google
func GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Authorization code not found"})
		return
	}

	state := c.Query("state")
	var referralCode string
	if strings.HasPrefix(state, "ref_") {
		referralCode = strings.TrimPrefix(state, "ref_")
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		clientID = getClientIDFromGoogleJSON()
	}
	if clientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Client ID is not configured"})
		return
	}

	clientSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:8080/api/auth/google/callback"
	}

	// Exchange authorization code for token
	tokenURL := "https://oauth2.googleapis.com/token"
	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to exchange authorization code: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errData map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errData)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google token exchange failed", "details": errData})
		return
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse token response"})
		return
	}

	// Fetch user profile using access token
	userinfoURL := "https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + tokenResp.AccessToken
	userinfoResp, err := http.Get(userinfoURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info: " + err.Error()})
		return
	}
	defer userinfoResp.Body.Close()

	if userinfoResp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to retrieve user profile from Google"})
		return
	}

	var googleUser struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(userinfoResp.Body).Decode(&googleUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user info"})
		return
	}

	if googleUser.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Google account is missing email details"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var player models.Player
	var hasPlayer bool

	// 1. Try to search by google_id
	err = tx.QueryRow(`
		SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, last_mana_update, created_at, lifespan, max_lifespan, time_speed_multiplier, email, google_id, completed_quests, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, unlocked_recipes, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, COALESCE(ref_code, '')
		FROM players
		WHERE google_id = $1
	`, googleUser.ID).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory, &player.MaterialsInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.MaxLifespan, &player.TimeSpeedMultiplier,
		&player.Email, &player.GoogleID, &player.CompletedQuests, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.UnlockedRecipes, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode,
	)

	if err == nil {
		hasPlayer = true
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking google_id: " + err.Error()})
		return
	}

	if !hasPlayer {
		// 2. Try to search by email (Merge Account!)
		err = tx.QueryRow(`
			SELECT id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, last_mana_update, created_at, lifespan, max_lifespan, time_speed_multiplier, email, google_id, completed_quests, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, unlocked_recipes, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, COALESCE(ref_code, '')
			FROM players
			WHERE email = $1
		`, googleUser.Email).Scan(
			&player.ID, &player.Username, &player.Rank, &player.SubRank,
			&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
			&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory, &player.MaterialsInventory,
			&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
			&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
			&player.EquippedGus, &player.ClearedCells,
			&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries,
			&player.LastManaUpdate, &player.CreatedAt,
			&player.Lifespan, &player.MaxLifespan, &player.TimeSpeedMultiplier,
			&player.Email, &player.GoogleID, &player.CompletedQuests, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
			&player.ApertureStress, &player.LastCultivateTime,
			&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.UnlockedRecipes, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode,
		)

		if err == nil {
			// Merge Account: update players table to link Google account ID
			_, err = tx.Exec("UPDATE players SET google_id = $1 WHERE id = $2", googleUser.ID, player.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to merge account: " + err.Error()})
				return
			}
			gID := googleUser.ID
			player.GoogleID = &gID
			hasPlayer = true
		} else if err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking email: " + err.Error()})
			return
		}
	}

	if !hasPlayer {
		// 3. Create a brand new player
		username := googleUser.Name
		if username == "" {
			parts := strings.Split(googleUser.Email, "@")
			username = parts[0]
		}
		username = strings.TrimSpace(username)
		if len(username) < 3 {
			username = "cultivator_" + strings.ToLower(uuid.New().String()[:6])
		} else if len(username) > 30 {
			username = username[:30]
		}

		// Check duplicate username and resolve
		var exists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE username = $1)", username).Scan(&exists)
		if err == nil && exists {
			username = fmt.Sprintf("%s_%d", username, rand.Intn(9000)+1000)
		}

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

		starterGu := models.GuInventory{
			{
				InstanceID: uuid.New().String(),
				GuID:       "GU_MOONLIGHT",
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   true,
				IsVital:    true,
			},
		}
		starterMaterials := models.MaterialsInventory{
			"moon_orchid_petal": 5,
			"green_bamboo_wine": 2,
			"jade_stone":        1,
		}

		// Generate unique referral code
		refCode := "REF-" + strings.ToUpper(uuid.New().String()[:8])

		queryInsert := `
			INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, lifespan, max_lifespan, time_speed_multiplier, email, google_id, completed_quests, temp_loot, explore_count, last_explore_reset, soul_hp, max_soul, soul_tier, needs_soul_refine, unlocked_recipes, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, ref_code)
			VALUES ($1, '', 1, 1, $2, $3, $4, $5, $6, 0, 0, $7, $8, 0.0, 0.0, NULL, 0, 100.0, 100.0, '[]'::jsonb, 50, 50, '["50,50"]'::jsonb, '[]'::jsonb, '{}'::jsonb, 'Southern_Border', '{"strength": 0}'::jsonb, '{}'::jsonb, NULL, 100.0, 100.0, 1.0, $9, $10, '[]'::jsonb, '{"gu_inventory": [], "materials_inventory": {}, "primeval_stones": 0}'::jsonb, 0, NOW(), 10, 10, 'NHAN_HON', false, '[{"recipe_id": "recipe_moon_raiment", "completion_rate": 100.0}, {"recipe_id": "recipe_white_jade", "completion_rate": 50.0}, {"recipe_id": "recipe_four_flavors_liquor", "completion_rate": 30.0}]'::jsonb, 'Hãy viết gì đó về bản thân...', '', 1, 1, '', false, $11)
			RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, materials_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, max_lifespan, time_speed_multiplier, party_id, arena_rank, email, google_id, completed_quests, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, unlocked_recipes, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, ref_code
		`

		err = tx.QueryRow(queryInsert, username, aptitudeType, aptitudePercent, aptitudeGrade, currentMana, maxMana, starterGu, starterMaterials, googleUser.Email, googleUser.ID, refCode).Scan(
			&player.ID, &player.Username, &player.Rank, &player.SubRank,
			&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
			&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory, &player.MaterialsInventory,
			&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
			&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
			&player.EquippedGus, &player.ClearedCells,
			&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &player.ActiveCombat,
			&player.LastManaUpdate, &player.CreatedAt,
			&player.Lifespan, &player.MaxLifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
			&player.Email, &player.GoogleID, &player.CompletedQuests, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
			&player.ApertureStress, &player.LastCultivateTime,
			&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.UnlockedRecipes, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new player record: " + err.Error()})
			return
		}

		// Handle referral code logic
		if referralCode != "" {
			var referrerID uuid.UUID
			err = tx.QueryRow("SELECT id FROM players WHERE ref_code = $1", referralCode).Scan(&referrerID)
			if err == nil {
				if referrerID != player.ID {
					clientIP := c.ClientIP()
					userAgent := c.Request.UserAgent()
					var count int
					_ = tx.QueryRow("SELECT COUNT(*) FROM referrals WHERE referee_ip = $1 AND referee_device = $2", clientIP, userAgent).Scan(&count)
					isSuspicious := count > 0

					_, err = tx.Exec(`
						INSERT INTO referrals (referrer_id, referee_id, referee_ip, referee_device, is_suspicious)
						VALUES ($1, $2, $3, $4, $5)
					`, referrerID, player.ID, clientIP, userAgent, isSuspicious)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record referral: " + err.Error()})
						return
					}
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Google login state"})
		return
	}

	// Generate JWT session token
	token, err := auth.GenerateToken(player.ID, player.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign access token"})
		return
	}
	setAuthCookie(c, token)

	// Redirect to frontend
	frontendURL := os.Getenv("CORS_ALLOWED_ORIGINS")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	// Parse the first origin if multiple are defined
	if strings.Contains(frontendURL, ",") {
		frontendURL = strings.Split(frontendURL, ",")[0]
	}

	c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}

// TickRegen checks Redis for active regeneration buffs and heals the player
func TickRegen(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	redisKey := fmt.Sprintf("regen_buff:%s", authID.String())
	var itemID string
	err := cache.GetJSON(redisKey, &itemID)
	if err != nil {
		// Key not found or expired
		c.JSON(http.StatusOK, gin.H{"regen_active": false})
		return
	}

	itemSpec, ok := models.ItemRegistry[itemID]
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid item in regen buff"})
		return
	}

	healVal := itemSpec.Effects["regen_hp_out_of_combat"]
	// Divide by 10 since it's meant for a 10-minute duration with 1 tick per minute
	tickHeal := healVal / 10.0
	if tickHeal < 1.0 {
		tickHeal = 1.0
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	if player.HP >= player.BaseHP {
		c.JSON(http.StatusOK, gin.H{"regen_active": true, "message": "HP đã đầy."})
		return
	}

	player.HP += tickHeal
	if player.HP > player.BaseHP {
		player.HP = player.BaseHP
	}

	_, err = tx.Exec("UPDATE players SET hp = $1 WHERE id = $2", player.HP, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save HP"})
		return
	}

	tx.Commit()
	c.JSON(http.StatusOK, gin.H{
		"regen_active": true,
		"healed":       tickHeal,
		"current_hp":   player.HP,
		"max_hp":       player.BaseHP,
	})
}
