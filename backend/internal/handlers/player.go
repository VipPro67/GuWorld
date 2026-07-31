package handlers

import (
	crypto_rand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
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
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"golang.org/x/crypto/bcrypt"
)

// PlayerSvc is the package-level player service for handler use.
var PlayerSvc *services.PlayerService

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// CreatePlayerRequest represents the incoming registration payload
type CreatePlayerRequest struct {
	Username       string `json:"username" binding:"required,min=3,max=30"`
	Password       string `json:"password" binding:"required,min=6,max=50"`
	StartingRegion string `json:"starting_region"`
	SetVitalNow    bool   `json:"set_vital_now"`
	Email          string `json:"email" binding:"required,email"`
	ReferralCode   string `json:"referral_code"`
	AttackGuID     string `json:"attack_gu_id"`
	DefenseGuID    string `json:"defense_gu_id"`
	VitalGuChoice  string `json:"vital_gu_choice"`
	OTP            string `json:"otp" binding:"required"`
}

// CreatePlayerResponse represents the output payload for registration/login
type CreatePlayerResponse struct {
	Player models.Player `json:"player"`
	Token  string        `json:"token,omitempty"`
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

	// Tự động phát hiện yêu cầu cross-origin/cross-site
	origin := c.Request.Header.Get("Origin")
	corsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")

	originHost := ""
	if origin != "" {
		if u, err := url.Parse(origin); err == nil {
			originHost = u.Hostname()
		}
	} else if corsEnv != "" {
		firstCors := corsEnv
		if strings.Contains(corsEnv, ",") {
			firstCors = strings.Split(corsEnv, ",")[0]
		}
		if u, err := url.Parse(firstCors); err == nil {
			originHost = u.Hostname()
		}
	}

	requestHost := c.Request.Host
	if strings.Contains(requestHost, ":") {
		requestHost = strings.Split(requestHost, ":")[0]
	}

	// Nếu host của frontend khác host của backend, chúng là cross-site
	// Ngoại lệ: Nếu cả hai đều chạy trên localhost, SameSite Lax vẫn được chấp nhận
	if originHost != "" && originHost != requestHost {
		if originHost == "localhost" && requestHost == "localhost" {
			sameSite = http.SameSiteLaxMode
		} else {
			sameSite = http.SameSiteNoneMode
		}
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

	// Set CSRF token cookie (non-HttpOnly so JS can read it for header)
	csrfToken, err := middleware.GenerateCSRFToken()
	if err == nil {
		middleware.SetCSRFCookie(c, csrfToken)
	}
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
	middleware.ClearCSRFCookie(c)
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
	if req.StartingRegion == "" {
		req.StartingRegion = "Southern_Border"
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

	// Validate selected starting Gu
	if req.AttackGuID == "" {
		req.AttackGuID = "GU_MOONLIGHT"
	}
	if req.DefenseGuID == "" {
		req.DefenseGuID = "GU_STONE_SKIN"
	}

	validAttackGus := map[string]bool{
		"GU_MOONLIGHT":  true,
		"GU_STREAM":     true,
		"GU_BONE_SPEAR": true,
	}
	validDefenseGus := map[string]bool{
		"GU_GREEN_SILK": true,
		"GU_STONE_SKIN": true,
		"GU_IRON_SKIN":  true,
		"GU_BEAST_SKIN": true,
		"GU_FISH_SCALE": true,
	}

	if !validAttackGus[req.AttackGuID] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng tấn công không hợp lệ"})
		return
	}
	if !validDefenseGus[req.DefenseGuID] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng phòng thủ không hợp lệ"})
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

	isAttackVital := false
	isDefenseVital := false

	if req.VitalGuChoice == "ATTACK" {
		isAttackVital = true
	} else if req.VitalGuChoice == "DEFENSE" {
		isDefenseVital = true
	} else if req.VitalGuChoice == "NONE" {
		// both false
	} else {
		// fallback to SetVitalNow
		if req.SetVitalNow {
			isAttackVital = true // default to attack
		}
	}

	attackGuInstanceID := uuid.New().String()
	defenseGuInstanceID := uuid.New().String()

	// 3. Initialize starter inventory for Phase 2
	starterGu := models.GuInventory{
		{
			InstanceID: attackGuInstanceID,
			GuID:       req.AttackGuID,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   true,
			IsVital:    isAttackVital,
		},
		{
			InstanceID: defenseGuInstanceID,
			GuID:       req.DefenseGuID,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   true,
			IsVital:    isDefenseVital,
		},
	}
	equippedGus := models.EquippedGusList{attackGuInstanceID, defenseGuInstanceID}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Verify email OTP inside tx
	var dbOTP string
	var expiresAt time.Time
	err = tx.QueryRow("SELECT otp, expires_at FROM email_otps WHERE email = $1", req.Email).Scan(&dbOTP, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Mã xác thực OTP chưa được gửi hoặc đã bị xóa!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi xác thực OTP"})
		}
		return
	}

	if time.Now().After(expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã OTP đã hết hạn! Vui lòng yêu cầu mã mới."})
		return
	}

	if dbOTP != req.OTP {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã OTP không chính xác!"})
		return
	}

	// Delete OTP to prevent reuse
	_, _ = tx.Exec("DELETE FROM email_otps WHERE email = $1", req.Email)

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
		INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier, party_id, arena_rank, email, google_id, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, ref_code)
		VALUES ($1, $2, 1, 1, $3, $4, $5, $6, $7, 20, 0, $8, 0.0, 0.0, NULL, 0, 100.0, 100.0, '[]'::jsonb, 50, 50, '["50,50"]'::jsonb, $12, '{}'::jsonb, $9, '{"strength": 0}'::jsonb, '{}'::jsonb, NULL, $10, NOW(), 100.0, 1.0, NULL, 0, $11, NULL, NULL, 0, 0.0, 0.0, NULL, NULL, '', 0, 0, '', false, false, $13)
		RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier, party_id, arena_rank, email, google_id, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, ref_code
	`

	var activeCombatBytes []byte
	err = tx.QueryRow(query, req.Username, passwordHash, aptitudeType, aptitudePercent, aptitudeGrade, currentMana, maxMana, starterGu, req.StartingRegion, req.Email, refCode, equippedGus).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.GoogleID, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.Ref[...]
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create player: " + err.Error()})
		return
	}

	// Seed starter materials
	InventorySvc.AddItem(tx, player.ID.String(), "moon_orchid_petal", "materials", 5)
	InventorySvc.AddItem(tx, player.ID.String(), "green_bamboo_wine", "materials", 2)
	InventorySvc.AddItem(tx, player.ID.String(), "jade_stone", "materials", 1)
	// Seed starter recipes
	RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_moon_raiment", CompletionRate: 100, IsUnlocked: true})
	RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_white_jade", CompletionRate: 50, IsUnlocked: true})
	RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_four_flavors_liquor", CompletionRate: 30, IsUnlocked: true})

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

	auditlog.Logf(db.DB, &player.ID, "player.register",
		"username=%s email=%s region=%s", req.Username, req.Email, req.StartingRegion)

	c.JSON(http.StatusCreated, CreatePlayerResponse{
		Player: player,
		Token:  token,
	})
}

type CompleteOnboardingRequest struct {
	StartingRegion string `json:"starting_region" binding:"required"`
	AttackGuID     string `json:"attack_gu_id"`
	DefenseGuID    string `json:"defense_gu_id"`
	VitalGuChoice  string `json:"vital_gu_choice"`
}

func CompleteOnboarding(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req CompleteOnboardingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validRegions := map[string]bool{
		"Southern_Border": true, "Central_Continent": true,
		"Northern_Plains": true, "Western_Desert": true, "Eastern_Sea": true,
	}
	if !validRegions[req.StartingRegion] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Starting region không hợp lệ"})
		return
	}

	if req.AttackGuID == "" {
		req.AttackGuID = "GU_MOONLIGHT"
	}
	if req.DefenseGuID == "" {
		req.DefenseGuID = "GU_STONE_SKIN"
	}

	validAttackGus := map[string]bool{
		"GU_MOONLIGHT": true, "GU_STREAM": true, "GU_BONE_SPEAR": true,
	}
	validDefenseGus := map[string]bool{
		"GU_GREEN_SILK": true, "GU_STONE_SKIN": true, "GU_IRON_SKIN": true,
		"GU_BEAST_SKIN": true, "GU_FISH_SCALE": true,
	}
	if !validAttackGus[req.AttackGuID] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng tấn công không hợp lệ"})
		return
	}
	if !validDefenseGus[req.DefenseGuID] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng phòng thủ không hợp lệ"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	if !player.NeedsOnboarding {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Onboarding đã hoàn tất"})
		return
	}

	attackGuInstanceID := uuid.New().String()
	defenseGuInstanceID := uuid.New().String()

	isAttackVital := req.VitalGuChoice == "ATTACK"
	isDefenseVital := req.VitalGuChoice == "DEFENSE"

	newGu := models.GuInventory{
		{InstanceID: attackGuInstanceID, GuID: req.AttackGuID, Hunger: 100, Status: "ACTIVE", Equipped: true, IsVital: isAttackVital},
		{InstanceID: defenseGuInstanceID, GuID: req.DefenseGuID, Hunger: 100, Status: "ACTIVE", Equipped: true, IsVital: isDefenseVital},
	}
	equippedGus := models.EquippedGusList{attackGuInstanceID, defenseGuInstanceID}

	_, err = tx.Exec(`
		UPDATE players
		SET needs_onboarding = false, current_region = $1, gu_inventory = $2, equipped_gus = $3
		WHERE id = $4
	`, req.StartingRegion, newGu, equippedGus, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save onboarding: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save"})
		return
	}

	player.NeedsOnboarding = false
	player.CurrentRegion = req.StartingRegion
	player.GuInventory = newGu
	player.EquippedGus = equippedGus

	c.JSON(http.StatusOK, player)
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
		SELECT id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, gu_inventory, wall_progress, human_qi, next_tribulation_time, tribulation_count, hp, base_hp, body_mods, x, y, explored_cells, equipped_gus, cleared_cells, current_region, dao_marks, path_masteries, active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier, party_id, arena_rank, email, google_id, temp_loot, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, avatar_url
		FROM players
		WHERE LOWER(email) = $1
	`
	var activeCombatBytes []byte
	err := db.DB.QueryRow(query, loginEmail).Scan(
		&player.ID, &player.Username, &passwordHash, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime, &player.TribulationCount,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y, &player.ExploredCells,
		&player.EquippedGus, &player.ClearedCells,
		&player.CurrentRegion, &player.DaoMarks, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.GoogleID, &player.TempLoot, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.Avat[...]
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

	auditlog.Logf(db.DB, &player.ID, "player.login",
		"username=%s email=%s", player.Username, loginEmail)

	c.JSON(http.StatusOK, CreatePlayerResponse{
		Player: player,
		Token:  token,
	})
}

// LogoutPlayer clears the auth cookie for the current browser session.
func LogoutPlayer(c *gin.Context) {
	if playerID, exists := middleware.GetPlayerID(c); exists {
		auditlog.Logf(db.DB, &playerID, "player.logout", "")
	}
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

	player.MaterialsInventory = loadMaterialsInventory(playerID.String())
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

	player.MaterialsInventory = loadMaterialsInventory(playerID.String())
	c.JSON(http.StatusOK, player)
}

// Note: the rest of the file remains unchanged from the original version
