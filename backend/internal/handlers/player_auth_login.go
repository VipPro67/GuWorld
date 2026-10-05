package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
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

const loginEvaluationAttempts = 3

// evaluateLoginPlayer retries the complete status transaction when lazy hunger
// evaluation loses a short-lived PostgreSQL lock race. A lock timeout aborts
// the transaction, so retrying only the hunger statement is not safe.
func evaluateLoginPlayer(c *gin.Context, playerID uuid.UUID) (models.Player, error) {
	return retryLoginEvaluation(c.Request.Context(), func() (models.Player, error) {
		tx, err := db.DB.BeginTx(c.Request.Context(), nil)
		if err != nil {
			return models.Player{}, err
		}

		// Login should retry quickly rather than spending the connection-wide
		// ten-second lock timeout on every attempt.
		if _, err = tx.ExecContext(c.Request.Context(), `SET LOCAL lock_timeout = '1500ms'`); err == nil {
			var player models.Player
			player, err = evaluateManaTxEquipped(tx, playerID)
			if err == nil {
				err = tx.Commit()
				if err == nil {
					return player, nil
				}
			}
		}
		_ = tx.Rollback()
		return models.Player{}, err
	})
}

func retryLoginEvaluation(ctx context.Context, evaluate func() (models.Player, error)) (models.Player, error) {
	var lastErr error
	for attempt := 0; attempt < loginEvaluationAttempts; attempt++ {
		player, err := evaluate()
		if err == nil {
			return player, nil
		}
		lastErr = err
		if !worker.IsHungerTransientError(err) || attempt == loginEvaluationAttempts-1 {
			break
		}

		timer := time.NewTimer(time.Duration(attempt+1) * 100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return models.Player{}, ctx.Err()
		case <-timer.C:
		}
	}
	return models.Player{}, lastErr
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

	var targetPlayerID uuid.UUID
	var passwordHash string
	err := db.DB.QueryRow(`SELECT id, password_hash FROM players WHERE LOWER(email) = $1`, loginEmail).Scan(&targetPlayerID, &passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thông tin đăng nhập không hợp lệ!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu: " + err.Error()})
		}
		return
	}

	// Verify hashed password before running heavy aggregation queries
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Thông tin đăng nhập không hợp lệ!"})
		return
	}

	player, err := evaluateLoginPlayer(c, targetPlayerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thông tin đăng nhập không hợp lệ!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu: " + err.Error()})
		}
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

	// Update last login info + rotate session in a single write to avoid two
	// consecutive row versions (and trigger bumps) on the hot player row.
	sessionID := uuid.New().String()
	_, err = db.DB.Exec(`
		UPDATE players
		SET last_login_fingerprint = $1, last_login_ip = $2, last_login_time = NOW(),
		    active_session_id = $3
		WHERE id = $4`,
		fingerprint, c.ClientIP(), sessionID, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật phiên đăng nhập"})
		return
	}
	middleware.SetSessionCache(player.ID, sessionID)

	token, err := auth.GenerateToken(player.ID, player.Username, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sign access token"})
		return
	}
	// Invalidate any stale admin-access cache entry so the fresh value is
	// fetched and cached for subsequent /auth/me polls.
	_ = cache.Delete("admin_access:" + player.ID.String())
	player.CanAccessAdmin, err = cachedPlayerHasAdminAccess(c.Request.Context(), player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve admin access"})
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
		middleware.InvalidateSessionCache(playerID)
	}
	clearAuthCookie(c)
	c.Status(http.StatusNoContent)
}

// adminAccessCacheTTL is how long we cache the admin access result per player.
// Admin role changes are rare; 60s staleness is acceptable.
const adminAccessCacheTTL = 60 * time.Second

// cachedPlayerHasAdminAccess checks player admin access with a Redis cache
// to avoid a 3-table JOIN on every /auth/me poll request.
func cachedPlayerHasAdminAccess(ctx context.Context, playerID uuid.UUID) (bool, error) {
	cacheKey := "admin_access:" + playerID.String()
	var cached bool
	if err := cache.GetJSON(cacheKey, &cached); err == nil {
		return cached, nil
	}
	result, err := middleware.PlayerHasAdminAccess(ctx, playerID)
	if err != nil {
		return false, err
	}
	_ = cache.SetJSON(cacheKey, result, adminAccessCacheTTL)
	return result, nil
}

// GetCurrentPlayer returns the authenticated player from the httpOnly cookie session.
func GetCurrentPlayer(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid auth session"})
		return
	}

	// Lock-free polling path: project without FOR UPDATE, persist at most
	// once per 45s in a short Tx. Contended persists are skipped so reads
	// never queue behind statement_timeout.
	player, err := PlayerSvc.ProjectAndPersistBestEffort(playerID, true)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to evaluate player status: " + err.Error()})
		}
		return
	}

	// player.MaterialsInventory is already populated by loadRow() inside ProjectAndPersistBestEffort.
	// No need to call loadMaterialsInventory() again here.
	player.CompletedQuests = loadCompletedQuests(playerID.String())
	player.UnlockedRecipes = loadUnlockedRecipes(playerID.String())
	player.CanAccessAdmin, err = cachedPlayerHasAdminAccess(c.Request.Context(), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve admin access"})
		return
	}
	c.JSON(http.StatusOK, player)
}
