package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"golang.org/x/crypto/bcrypt"
)

// RegisterPlayer handles registering a new player, hashing password, and rolling their aptitude
func RegisterPlayer(c *gin.Context) {
	var req CreatePlayerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	if len(req.Username) < 3 || len(req.Username) > 30 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu phải từ 3 đến 30 ký tự!"})
		return
	}
	if !isValidUsername(req.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hiệu chỉ được chứa chữ cái, dấu gạch ngang và khoảng trắng!"})
		return
	}

	validRegions := map[string]bool{
		"hub_nam_cuong":  true,
		"hub_trung_chau": true,
		"hub_bac_nguyen": true,
		"hub_tay_mac":    true,
		"hub_dong_hai":   true,
	}
	if req.StartingRegion == "" {
		req.StartingRegion = "hub_nam_cuong"
	}
	if !validRegions[req.StartingRegion] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Starting region không hợp lệ"})
		return
	}

	// Chủng tộc luôn mặc định Nhân Tộc khi tạo tài khoản —
	// các chủng tộc khác nhận được qua hoạt động/sự kiện trong game.
	req.Race = "human"
	if req.Gender == "" {
		req.Gender = "male"
	}
	if !gamefx.ValidGenders()[req.Gender] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Giới tính không hợp lệ"})
		return
	}

	// Email format verification and normalization
	req.Email = normalizeEmail(req.Email)
	if !isValidEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng Email không hợp lệ!"})
		return
	}
	if isDisposableEmail(req.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email tạm thời không được chấp nhận. Vui lòng dùng email thật!"})
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

	roll := rand.Intn(100)

	var aptitudeType string
	var aptitudePercent int

	switch {
	case roll < 5:
		aptitudeType = "Giáp"
		aptitudePercent = 80 + rand.Intn(21)
	case roll < 20:
		aptitudeType = "Ất"
		aptitudePercent = 60 + rand.Intn(20)
	case roll < 50:
		aptitudeType = "Bính"
		aptitudePercent = 50 + rand.Intn(10)
	default:
		aptitudeType = "Đinh"
		aptitudePercent = 30 + rand.Intn(10)
	}

	var constitution string = ""
	if aptitudePercent == 100 {
		defs := gamefx.AllConstitutions()
		if len(defs) > 0 {
			idx := rand.Intn(len(defs))
			constitution = defs[idx].Key
		}
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

	switch req.VitalGuChoice {
	case "ATTACK":
		isAttackVital = true
	case "DEFENSE":
		isDefenseVital = true
	case "NONE":
		// both false
	default:
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
			Rank:       1,
			Hunger:     100,
			Status:     "ACTIVE",
			Equipped:   true,
			IsVital:    isAttackVital,
		},
		{
			InstanceID: defenseGuInstanceID,
			GuID:       req.DefenseGuID,
			Rank:       1,
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
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE LOWER(username) = LOWER($1))", req.Username).Scan(&usernameExists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra đạo hiệu"})
		return
	}
	if usernameExists {
		c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu này đã tồn tại trong Cổ Giới!"})
		return
	}

	// Verify device fingerprint registration limit (max 2 accounts)
	var fingerprintCount int
	err = tx.QueryRow("SELECT COUNT(1) FROM players WHERE register_fingerprint = $1", strings.TrimSpace(req.Fingerprint)).Scan(&fingerprintCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra giới hạn thiết bị đăng ký"})
		return
	}
	if fingerprintCount >= 2 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Thiết bị này đã đạt giới hạn đăng ký tối đa (2 tài khoản)!"})
		return
	}

	if blocked, errBlock := deviceBlocked(tx, c.ClientIP(), strings.TrimSpace(req.Fingerprint)); errBlock != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra thiết bị"})
		return
	} else if blocked {
		c.JSON(http.StatusForbidden, gin.H{"error": "Thiết bị này đã bị chặn. Vui lòng liên hệ bộ phận hỗ trợ."})
		return
	}

	// Generate unique referral code
	refCode := "REF-" + strings.ToUpper(uuid.New().String()[:8])

	var player models.Player
	query := `
		INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, wall_progress, human_qi, next_tribulation_time, hp, base_hp, body_mods, x, y, equipped_gus, cleared_cells, visited_subzones, current_region, path_masteries, active_combat, lifespan, time_speed_multiplier, email, explore_count, last_explore_reset, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, avatar_url, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, ref_code, needs_onboarding, race, gender, constitution, register_fingerprint, register_ip)
		VALUES ($1, $2, 0, 1, $3, $4, $5, $6, $7, 20, 0, 0.0, 0.0, NULL, 100.0, 100.0, '[]'::jsonb, $14, $15, $11, '{}'::jsonb, '[]'::jsonb, $8, '{}'::jsonb, NULL, 100.0, 1.0, $9, 0, NOW(), 10, 10, 'NHAN_HON', false, 'Hãy viết gì đó về bản thân...', '', 1, 1, '', false, $10, true, $12, $13, $16, $17, $18)
		RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, ` + db.GuInventoryAggregate("players.id") + `, wall_progress, human_qi, next_tribulation_time, hp, base_hp, body_mods, x, y, equipped_gus, cleared_cells, visited_subzones, current_region, path_masteries, active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier, party_id, arena_rank, email, google_id, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, ref_code, avatar_url, needs_onboarding, race, gender, constitution, statuses
	`

	var activeCombatBytes []byte
	hx, hy := hubEntryPoint()
	err = tx.QueryRow(query, req.Username, passwordHash, aptitudeType, aptitudePercent, aptitudeGrade, currentMana, maxMana, req.StartingRegion, req.Email, refCode, equippedGus, req.Race, req.Gender, hx, hy, constitution, strings.TrimSpace(req.Fingerprint), c.ClientIP()).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.AptitudeType, &player.AptitudePercent, &player.AptitudeGrade, &player.CurrentMana,
		&player.MaxMana, &player.PrimevalStones, &player.ImmortalStones, &player.GuInventory,
		&player.WallProgress, &player.HumanQi, &player.NextTribulationTime,
		&player.HP, &player.BaseHP, &player.BodyMods, &player.X, &player.Y,
		&player.EquippedGus, &player.ClearedCells, &player.VisitedSubZones,
		&player.CurrentRegion, &player.PathMasteries, &activeCombatBytes,
		&player.LastManaUpdate, &player.CreatedAt,
		&player.Lifespan, &player.TimeSpeedMultiplier, &player.PartyID, &player.ArenaRank,
		&player.Email, &player.GoogleID, &player.ExploreCount, &player.LastExploreReset,
		&player.ApertureStress, &player.LastCultivateTime,
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.AvatarURL, &player.NeedsOnboarding, &player.Race, &player.Gender, &player.Constitution, &player.Statuses,
	)
	if err != nil {
		if isUsernameUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "Đạo hiệu này đã tồn tại (không phân biệt chữ hoa/thường)!"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create player: " + err.Error()})
		return
	}

	// Persist starter Gus in the player_gu table (the players.gu_inventory column is gone).
	if err := db.AddPlayerGusBulk(tx, player.ID.String(), starterGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to seed starter Gus: " + err.Error()})
		return
	}
	player.GuInventory = starterGu

	// Seed starter materials
	InventorySvc.AddItem(tx, player.ID.String(), "moon_orchid_petal", "materials", 5)
	InventorySvc.AddItem(tx, player.ID.String(), "green_bamboo_wine", "materials", 2)
	InventorySvc.AddItem(tx, player.ID.String(), "jade_stone", "materials", 1)

	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if json.Unmarshal(activeCombatBytes, &cs) == nil {
			player.ActiveCombat = &cs
		}
	}

	// 3.5 Handle referral code logic
	var referralBanTarget uuid.UUID
	var referralBanCount int
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

				if isSuspicious {
					var suspiciousCount int
					if err := tx.QueryRow("SELECT COUNT(*) FROM referrals WHERE referrer_id = $1 AND is_suspicious", referrerID).Scan(&suspiciousCount); err == nil {
						referralBanTarget = referrerID
						referralBanCount = suspiciousCount
					}
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player record: " + err.Error()})
		return
	}

	if referralBanTarget != uuid.Nil {
		autoBanForReferral(c.Request.Context(), referralBanTarget, referralBanCount)
	}

	// 4. Generate token
	sessionID := uuid.New().String()
	_, err = db.DB.Exec(`UPDATE players SET active_session_id = $1 WHERE id = $2`, sessionID, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	token, err := auth.GenerateToken(player.ID, player.Username, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create access token"})
		return
	}
	setAuthCookie(c, token)

	c.JSON(http.StatusCreated, CreatePlayerResponse{
		Player: player,
		Token:  token,
	})
}

// CompleteTutorial is invoked when the player finishes the lv0 (Phàm Nhân)
// onboarding-guidance phase. Rank advances from 0 → 1; needs_onboarding stays
// true so the existing OnboardingModal (region/Gu/vital selection) shows next,
// and all gameplay mutations remain locked behind RequireOnboarded until
// complete-onboarding flips needs_onboarding=false.
func CompleteTutorial(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
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

	if player.Rank >= 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Giai đoạn Phàm Nhân đã kết thúc"})
		return
	}

	if _, err := tx.Exec(`UPDATE players SET rank = 1, sub_rank = GREATEST(sub_rank, 1) WHERE id = $1`, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save tutorial completion"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save"})
		return
	}

	player.Rank = 1
	if player.SubRank < 1 {
		player.SubRank = 1
	}
	// The players trigger bumps state_revision by 1 on the UPDATE above; mirror
	// it so the client's revision-gated merge accepts rank 0 -> 1 immediately.
	player.StateRevision++
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "player.tutorial_complete",
		RequestSummary: "lv0 complete -> rank 1", Success: true,
	})
	c.JSON(http.StatusOK, gin.H{"player": player})
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
		"hub_nam_cuong":  true,
		"hub_trung_chau": true,
		"hub_bac_nguyen": true,
		"hub_tay_mac":    true,
		"hub_dong_hai":   true,
	}
	if !validRegions[req.StartingRegion] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Starting region không hợp lệ"})
		return
	}

	startingHubID := req.StartingRegion
	if startingHubID == "" {
		startingHubID = "hub_nam_cuong"
	}
	spawnX, spawnY := hubEntryPoint()
	if hub := ZoneByID(startingHubID); hub != nil && hub.EntryX != 0 && hub.EntryY != 0 {
		spawnX, spawnY = hub.EntryX, hub.EntryY
	} else {
		startingHubID = HubRegionID
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
		{InstanceID: attackGuInstanceID, GuID: req.AttackGuID, Rank: 1, Hunger: 100, Status: "ACTIVE", Equipped: true, IsVital: isAttackVital},
		{InstanceID: defenseGuInstanceID, GuID: req.DefenseGuID, Rank: 1, Hunger: 100, Status: "ACTIVE", Equipped: true, IsVital: isDefenseVital},
	}
	equippedGus := models.EquippedGusList{attackGuInstanceID, defenseGuInstanceID}

	// Award 20x food of the selected Gus
	gusToProcess := []string{req.AttackGuID, req.DefenseGuID}
	for _, guID := range gusToProcess {
		if guID == "" {
			continue
		}
		spec, ok := models.GetGuAny(guID)
		if !ok {
			continue
		}
		for _, food := range spec.FoodTypes {
			if food == "primeval_stone" {
				player.PrimevalStones += 20
			} else if food == "immortal_stone" {
				player.ImmortalStones += 20.0
			} else {
				// Regular item
				if err := InventorySvc.AddItem(tx, playerID.String(), food, "materials", 20); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add food items: " + err.Error()})
					return
				}
			}
		}
	}

	_, err = tx.Exec(`
		UPDATE players
		SET needs_onboarding = false, current_region = $1, x = $2, y = $3, equipped_gus = $4, primeval_stones = $5, immortal_stones = $6
		WHERE id = $7
	`, startingHubID, spawnX, spawnY, equippedGus, player.PrimevalStones, player.ImmortalStones, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save onboarding: " + err.Error()})
		return
	}
	if err := db.AddPlayerGusBulk(tx, playerID.String(), newGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save starter Gus: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save"})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "player.onboarding",
		RequestSummary: fmt.Sprintf("region=%s attack=%s defense=%s vital=%s", req.StartingRegion, req.AttackGuID, req.DefenseGuID, req.VitalGuChoice),
		IPAddress:      c.ClientIP(), Success: true,
	})

	player.NeedsOnboarding = false
	player.CurrentRegion = startingHubID
	player.X = spawnX
	player.Y = spawnY
	player.GuInventory = newGu
	player.EquippedGus = equippedGus
	player.MaterialsInventory = loadMaterialsInventory(playerID.String())
	player.MaxMana = gamefx.EffectiveMaxMana(&player)
	setWorldPosFromPlayer(&player)
	// Mirror the trigger's state_revision bump so the client applies the
	// needs_onboarding=false transition immediately.
	player.StateRevision++
	c.JSON(http.StatusOK, gin.H{"player": player})
}
