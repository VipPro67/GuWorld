package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
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

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu: " + err.Error()})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTxEquipped(tx, targetPlayerID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thông tin đăng nhập không hợp lệ!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu: " + err.Error()})
		}
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu: " + err.Error()})
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
