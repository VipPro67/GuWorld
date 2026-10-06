package handlers

import (
	crypto_rand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

var googleOAuthHTTPClient = &http.Client{Timeout: 5 * time.Second}

// GoogleLogin redirects user to Google OAuth2 consent page
func GoogleLogin(c *gin.Context) {
	clientID := os.Getenv("GOOGLE_CLIENT_ID")
	if clientID == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Google Client ID is not configured"})
		return
	}

	redirectURI := os.Getenv("GOOGLE_REDIRECT_URI")
	if redirectURI == "" {
		redirectURI = "http://localhost:8080/api/auth/google/callback"
	}

	stateToken := make([]byte, 16)
	if _, err := crypto_rand.Read(stateToken); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate security state"})
		return
	}
	stateStr := hex.EncodeToString(stateToken)

	// Set state cookie valid for 5 minutes
	c.SetCookie("google_oauth_state", stateStr, 300, "/api/auth/google", "", authCookieSecure(c, false), true)

	refCode := c.Query("ref")
	fingerprint := c.Query("fingerprint")
	state := stateStr + ":" + refCode + ":" + fingerprint
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
	if state == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Security state parameter not found"})
		return
	}

	stateCookie, err := c.Cookie("google_oauth_state")
	if err != nil || stateCookie == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Security state cookie not found or expired"})
		return
	}

	// Xóa cookie ngay lập tức
	c.SetCookie("google_oauth_state", "", -1, "/api/auth/google", "", authCookieSecure(c, false), true)

	parts := strings.Split(state, ":")
	if len(parts) == 0 || parts[0] != stateCookie {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid security state mismatch"})
		return
	}

	var referralCode string
	if len(parts) >= 2 {
		referralCode = parts[1]
	}
	var fingerprint string
	if len(parts) >= 3 {
		fingerprint = parts[2]
	}

	clientID := os.Getenv("GOOGLE_CLIENT_ID")
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

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Google token request"})
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := googleOAuthHTTPClient.Do(req)
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
	userinfoURL := "https://www.googleapis.com/oauth2/v2/userinfo"
	userinfoReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, userinfoURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Google user info request"})
		return
	}
	userinfoReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	userinfoResp, err := googleOAuthHTTPClient.Do(userinfoReq)
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

	if db.DB == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Authentication database unavailable"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	_, _ = tx.ExecContext(c.Request.Context(), "SET LOCAL lock_timeout = '3000ms'")

	var player models.Player
	var hasPlayer bool
	var existingPlayerID uuid.UUID

	// 1. Try to search by google_id
	err = tx.QueryRowContext(c.Request.Context(), `SELECT id FROM players WHERE google_id = $1`, googleUser.ID).Scan(&existingPlayerID)
	if err == nil {
		hasPlayer = true
	} else if err != sql.ErrNoRows {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking google_id: " + err.Error()})
		return
	}

	if !hasPlayer {
		// 2. Try to search by email (Merge Account!)
		err = tx.QueryRowContext(c.Request.Context(), `SELECT id FROM players WHERE email = $1`, googleUser.Email).Scan(&existingPlayerID)
		if err == nil {
			// Merge Account: update players table to link Google account ID
			_, err = tx.ExecContext(c.Request.Context(), "UPDATE players SET google_id = $1 WHERE id = $2", googleUser.ID, existingPlayerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to merge account: " + err.Error()})
				return
			}
			hasPlayer = true
		} else if err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking email: " + err.Error()})
			return
		}
	}

	if hasPlayer {
		player, err = PlayerSvc.ProjectStatusEquipped(tx, existingPlayerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player: " + err.Error()})
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
		err = tx.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM players WHERE username = $1)", username).Scan(&exists)
		if err == nil && exists {
			username = fmt.Sprintf("%s_%d", username, rand.Intn(9000)+1000)
		}

		var prod_env = os.Getenv("PROD_ENV") == "test"
		var roll int
		if prod_env {
			roll = rand.Intn(30) // 0-29
		} else {
			roll = rand.Intn(100) // 0-99
		}
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

		attackGuInstanceID := uuid.New().String()
		defenseGuInstanceID := uuid.New().String()

		starterGu := models.GuInventory{
			{
				InstanceID: attackGuInstanceID,
				GuID:       "GU_MOONLIGHT",
				Rank:       1,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   true,
				IsVital:    true,
			},
			{
				InstanceID: defenseGuInstanceID,
				GuID:       "GU_STONE_SKIN",
				Rank:       1,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   true,
				IsVital:    false,
			},
		}
		equippedGus := models.EquippedGusList{attackGuInstanceID, defenseGuInstanceID}

		// Generate unique referral code
		refCode := "REF-" + strings.ToUpper(uuid.New().String()[:8])

		queryInsert := `
			INSERT INTO players (username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, wall_progress, human_qi, next_tribulation_time, hp, base_hp, body_mods, x, y, equipped_gus, cleared_cells, visited_subzones, current_region, path_masteries, active_combat, lifespan, time_speed_multiplier, email, google_id, explore_count, last_explore_reset, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, ref_code, needs_onboarding, race, gender)
			VALUES ($1, '', 1, 1, $2, $3, $4, $5, $6, 20, 0, 0.0, 0.0, NULL, 100.0, 100.0, '[]'::jsonb, $11, $12, $10, '{}'::jsonb, '[]'::jsonb, 'hub_nam_cuong', '{}'::jsonb, NULL, 100.0, 1.0, $7, $8, 0, NOW(), 10, 10, 'NHAN_HON', false, 'Hãy viết gì đó về bản thân...', 1, 1, '', false, $9, true, 'human', 'male')
			RETURNING id, username, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, ` + db.GuInventoryAggregate("players.id") + `, wall_progress, human_qi, next_tribulation_time, hp, base_hp, body_mods, x, y, equipped_gus, cleared_cells, visited_subzones, current_region, path_masteries, active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier, party_id, arena_rank, email, google_id, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, bio, fake_rank, fake_sub_rank, fake_vital_gu, is_disguised, is_admin, ref_code, needs_onboarding, race, gender, constitution, statuses
		`

		var activeCombatBytes []byte
		hx, hy := hubEntryPoint()
		err = tx.QueryRowContext(c.Request.Context(), queryInsert, username, aptitudeType, aptitudePercent, aptitudeGrade, currentMana, maxMana, googleUser.Email, googleUser.ID, refCode, equippedGus, hx, hy).Scan(
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
			&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.RefCode, &player.NeedsOnboarding, &player.Race, &player.Gender, &player.Constitution, &player.Statuses,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new player record: " + err.Error()})
			return
		}

		if len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
			var cs models.CombatState
			if json.Unmarshal(activeCombatBytes, &cs) == nil {
				player.ActiveCombat = &cs
			}
		}

		// Persist starter Gus in the player_gu table (the gu_inventory column is gone).
		if err := db.AddPlayerGusBulk(tx, player.ID.String(), starterGu); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to seed starter Gus: " + err.Error()})
			return
		}
		player.GuInventory = starterGu

		// Seed starter materials
		InventorySvc.AddItem(tx, player.ID.String(), "moon_orchid_petal", "materials", 5)
		InventorySvc.AddItem(tx, player.ID.String(), "green_bamboo_wine", "materials", 2)
		InventorySvc.AddItem(tx, player.ID.String(), "jade_stone", "materials", 1)
		// Seed starter recipes
		RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_moon_raiment", CompletionRate: 100, IsUnlocked: true})
		RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_white_jade", CompletionRate: 50, IsUnlocked: true})
		RecipeSvc.Upsert(tx, &services.RecipeProgress{PlayerID: player.ID.String(), RecipeID: "recipe_four_flavors_liquor", CompletionRate: 30, IsUnlocked: true})

		// Handle referral code logic
		if referralCode != "" {
			var referrerID uuid.UUID
			err = tx.QueryRowContext(c.Request.Context(), "SELECT id FROM players WHERE ref_code = $1", referralCode).Scan(&referrerID)
			if err == nil {
				if referrerID != player.ID {
					clientIP := c.ClientIP()
					userAgent := c.Request.UserAgent()
					var count int
					_ = tx.QueryRowContext(c.Request.Context(), "SELECT COUNT(*) FROM referrals WHERE referee_ip = $1 AND referee_device = $2", clientIP, userAgent).Scan(&count)
					isSuspicious := count > 0

					_, err = tx.ExecContext(c.Request.Context(), `
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

	// Enforce fingerprint multi-login rotation limit for Google OAuth
	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint == "" {
		fingerprint = "ip-" + c.ClientIP()
	}

	var otherAccountsCount int
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT COUNT(DISTINCT id) FROM players
		WHERE last_login_fingerprint = $1
		  AND last_login_time > NOW() - INTERVAL '24 hours'
		  AND id != $2`,
		fingerprint, player.ID).Scan(&otherAccountsCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi kiểm tra thiết bị"})
		return
	}

	if otherAccountsCount >= 2 {
		_ = tx.Rollback()

		// Log to Audit Log for Admin
		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID:       &player.ID,
			Action:         "security.multi_login_blocked",
			RequestSummary: fmt.Sprintf("Blocked Google login for %s due to multi-login fingerprint limit. Fingerprint=%s IP=%s", player.Username, fingerprint, c.ClientIP()),
			IPAddress:      c.ClientIP(),
			Success:        false,
			ErrorMessage:   "Thiết bị đăng nhập quá nhiều tài khoản trong 24 giờ qua",
		})

		// Send real-time SSE notifications to the other active accounts from this fingerprint
		rows, err := db.DB.Query(`
			SELECT id FROM players
			WHERE last_login_fingerprint = $1
			  AND last_login_time > NOW() - INTERVAL '24 hours'
			  AND id != $2`,
			fingerprint, player.ID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var otherPID uuid.UUID
				if err := rows.Scan(&otherPID); err == nil {
					notify.Send(db.DB, otherPID, "security_alert",
						"Cảnh báo bảo mật",
						fmt.Sprintf("Hệ thống đã chặn yêu cầu đăng nhập tài khoản Google thứ 3 (%s) trong ngày từ thiết bị của đạo hữu.", player.Username))
				}
			}
		}

		// Redirect back to frontend with error query parameter
		frontendURL := os.Getenv("CORS_ALLOWED_ORIGINS")
		if frontendURL == "" {
			frontendURL = "http://localhost:3000"
		}
		if strings.Contains(frontendURL, ",") {
			frontendURL = strings.Split(frontendURL, ",")[0]
		}
		redirectURL := frontendURL + "?error=" + url.QueryEscape("Thiết bị này đã đạt giới hạn đăng nhập tối đa 2 tài khoản trong vòng 24 giờ!")
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	// Update last login info and rotate session in a single write
	sessionID := uuid.New().String()
	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players
		SET last_login_fingerprint = $1, last_login_ip = $2, last_login_time = NOW(),
		    active_session_id = $3
		WHERE id = $4`,
		fingerprint, c.ClientIP(), sessionID, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Google login details"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Google login state"})
		return
	}

	// Generate JWT session token
	token, err := auth.GenerateToken(player.ID, player.Username, sessionID)
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

	redirectURL := frontendURL
	if strings.Contains(redirectURL, "?") {
		redirectURL += "&token=" + token
	} else {
		redirectURL += "?token=" + token
	}

	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}
