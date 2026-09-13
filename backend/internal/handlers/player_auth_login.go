package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents credentials login payload.
// Email is the canonical login identifier; Username is accepted only as a
// backwards-compatible JSON field for older clients that already send an email
// value under "username".
type LoginRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	Password    string `json:"password"`
	Fingerprint string `json:"fingerprint"`
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

	loginEmail = normalizeEmail(loginEmail)

	var player models.Player
	var passwordHash string

	query := `
		SELECT id, username, password_hash, rank, sub_rank, aptitude_type, aptitude_percent, aptitude_grade, current_mana, max_mana, primeval_stones, immortal_stones, ` + db.GuInventoryAggregateEquipped("players.id") + `, wall_progress, human_qi, next_tribulation_time, hp, base_hp, body_mods, x, y, equipped_gus, cleared_cells, visited_subzones, current_region, path_masteries, active_combat, last_mana_update, created_at, lifespan, time_speed_multiplier, 		       party_id, arena_rank, COALESCE(email, ''), google_id, explore_count, last_explore_reset, aperture_stress, last_cultivate_time, soul_hp, max_soul, soul_tier, needs_soul_refine, COALESCE(bio, ''), COALESCE(fake_rank, 1), COALESCE(fake_sub_rank, 1), COALESCE(fake_vital_gu, ''), COALESCE(is_disguised, false), COALESCE(is_admin, false), 		COALESCE(avatar_url, ''), COALESCE(race, ''), COALESCE(gender, ''), COALESCE(constitution, ''), COALESCE(is_dead, false), statuses, growth_data
		FROM players
		WHERE LOWER(email) = $1
	`
	var activeCombatBytes []byte
	err := db.DB.QueryRow(query, loginEmail).Scan(
		&player.ID, &player.Username, &passwordHash, &player.Rank, &player.SubRank,
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
		&player.SoulHP, &player.MaxSoul, &player.SoulTier, &player.NeedsSoulRefine, &player.Bio, &player.FakeRank, &player.FakeSubRank, &player.FakeVitalGu, &player.IsDisguised, &player.IsAdmin, &player.AvatarURL,
		&player.Race, &player.Gender, &player.Constitution, &player.IsDead, &player.Statuses,
		&player.GrowthData,
	)
	if err == nil {
		if player.Rank >= 6 {
			var daoMarks models.DaoMarksMap
			errDb := db.DB.QueryRow(`SELECT dao_marks FROM immortal_apertures WHERE player_id = $1`, player.ID).Scan(&daoMarks)
			if errDb == nil {
				player.DaoMarks = daoMarks
			} else {
				player.DaoMarks = models.DaoMarksMap{"strength": 0}
			}
		} else {
			player.DaoMarks = models.DaoMarksMap{"strength": 0}
		}
	}
	if err == nil && len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if errU := json.Unmarshal(activeCombatBytes, &cs); errU == nil {
			if !cs.IsEmpty() {
				player.ActiveCombat = &cs
			}
		} else {
			logger.S.Warnf("Login: failed to unmarshal active_combat for player=%s: %v", player.ID, errU)
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

	// Enforce fingerprint multi-login rotation limit
	fingerprint := strings.TrimSpace(req.Fingerprint)
	if fingerprint == "" {
		fingerprint = "ip-" + c.ClientIP()
	}

	var otherAccountsCount int
	err = db.DB.QueryRow(`
		SELECT COUNT(DISTINCT id) FROM players
		WHERE last_login_fingerprint = $1
		  AND last_login_time > NOW() - INTERVAL '24 hours'
		  AND id != $2`,
		fingerprint, player.ID).Scan(&otherAccountsCount)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi kiểm tra thiết bị đăng nhập"})
		return
	}

	if otherAccountsCount >= 2 {
		// Log to Audit Log for Admin
		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID:       &player.ID,
			Action:         "security.multi_login_blocked",
			RequestSummary: fmt.Sprintf("Blocked login for %s due to multi-login fingerprint limit. Fingerprint=%s IP=%s", player.Username, fingerprint, c.ClientIP()),
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
						fmt.Sprintf("Hệ thống đã chặn yêu cầu đăng nhập tài khoản thứ 3 (%s) trong ngày từ thiết bị của đạo hữu.", player.Username))
				}
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Thiết bị này đã đạt giới hạn đăng nhập tối đa 2 tài khoản trong vòng 24 giờ!"})
		return
	}

	// Update last login info
	_, err = db.DB.Exec(`
		UPDATE players
		SET last_login_fingerprint = $1, last_login_ip = $2, last_login_time = NOW()
		WHERE id = $3`,
		fingerprint, c.ClientIP(), player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật phiên đăng nhập"})
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

	sessionID := uuid.New().String()
	_, err = db.DB.Exec(`UPDATE players SET active_session_id = $1 WHERE id = $2`, sessionID, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	token, err := auth.GenerateToken(player.ID, player.Username, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign access token"})
		return
	}
	setAuthCookie(c, token)

	c.JSON(http.StatusOK, CreatePlayerResponse{
		Player: player,
		Token:  token,
	})
}

// LogoutPlayer clears the auth cookie for the current browser session.
func LogoutPlayer(c *gin.Context) {
	if playerID, exists := middleware.GetPlayerID(c); exists {
		_ = playerID
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

	player, err := evaluateManaTxEquipped(tx, playerID)
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
	player.CompletedQuests = loadCompletedQuests(playerID.String())
	player.UnlockedRecipes = loadUnlockedRecipes(playerID.String())
	c.JSON(http.StatusOK, player)
}
