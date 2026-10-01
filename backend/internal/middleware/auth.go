package middleware

import (
	"database/sql"
	"errors"
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/authz"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

// PlayerHasAdminAccess checks RBAC first and preserves players.is_admin as a
// rollout fallback for legacy administrators.
func PlayerHasAdminAccess(ctx context.Context, playerID uuid.UUID) (bool, error) {
	var hasRBACRole, isLegacyAdmin bool
	err := db.DB.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM player_admin_roles par
			JOIN admin_roles ar ON ar.id = par.role_id
			JOIN admin_role_permissions arp ON arp.role_id = ar.id
			WHERE par.player_id = $1 AND arp.permission_key = 'admin.*'
		), COALESCE(is_admin, false)
		FROM players
		WHERE id = $1`, playerID).Scan(&hasRBACRole, &isLegacyAdmin)
	return hasRBACRole || isLegacyAdmin, err
}

const (
	// SessionKickedError is returned when the session has been invalidated by a newer login
	SessionKickedError = "Phiên đăng nhập đã hết hạn"
)

func csrfProtection(c *gin.Context) {
	var csrfCookies []string
	for _, cookie := range c.Request.Cookies() {
		if cookie.Name == "gu_csrf_token" && cookie.Value != "" {
			csrfCookies = append(csrfCookies, cookie.Value)
		}
	}

	if len(csrfCookies) == 0 {
		// Nếu là request an toàn (GET/HEAD/OPTIONS), tự sinh mới cookie nếu thiếu
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			token, genErr := GenerateCSRFToken()
			if genErr == nil {
				SetCSRFCookie(c, token)
			}
			return
		}
		// Với request thay đổi dữ liệu, thiếu cookie là lỗi bảo mật
		c.JSON(http.StatusForbidden, gin.H{"error": "Missing CSRF token"})
		c.Abort()
		return
	}

	if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
		return
	}

	csrfHeader := c.GetHeader("X-CSRF-Token")
	if csrfHeader == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid CSRF token"})
		c.Abort()
		return
	}

	matched := false
	for _, cookieVal := range csrfCookies {
		if csrfHeader == cookieVal {
			matched = true
			break
		}
	}

	if !matched {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid CSRF token"})
		c.Abort()
		return
	}
}

// AuthRequired is a Gin middleware that enforces authentication
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		usedCookie := false
		tokenStr := ""
		var err error
		// Prefer the Authorization header: the frontend always sends the
		// current token from localStorage there, while duplicate/stale cookies
		// (e.g. after a DB reset) could otherwise shadow it via c.Cookie().
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be: Bearer <token>"})
				c.Abort()
				return
			}
			tokenStr = parts[1]
		} else {
			// Try query param (for EventSource which cannot send headers)
			tokenStr = c.Query("token")
			if tokenStr == "" {
				tokenStr, err = c.Cookie("gu_auth_token")
				if err != nil || tokenStr == "" {
					c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication cookie or Authorization header required"})
					c.Abort()
					return
				}
				usedCookie = true
			}
		}
		claims, err := auth.ValidateToken(tokenStr)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired authorization token"})
			c.Abort()
			return
		}

		playerID, err := uuid.Parse(claims.PlayerID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid player identity in token"})
			c.Abort()
			return
		}
		if db.DB == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Authentication database unavailable"})
			c.Abort()
			return
		}

		// Verify player actually exists in the database
		var exists bool
		var dbSessionID string
		err = db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM players WHERE id=$1), COALESCE(active_session_id, '') FROM players WHERE id=$1", playerID).Scan(&exists, &dbSessionID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Authentication database unavailable"})
			c.Abort()
			return
		}
		if errors.Is(err, sql.ErrNoRows) || !exists {
			// If usedCookie is true, we should ideally clear it, but the client handling 401 will often clear it anyway or redirect to login.
			// Let's just return 401. The frontend interceptor will call logout().
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tài khoản không tồn tại hoặc đã bị xóa."})
			c.Abort()
			return
		}

		// Validate session_id matches the active session in database
		if claims.SessionID != dbSessionID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": SessionKickedError})
			c.Abort()
			return
		}

		// Store variables in context
		c.Set("player_id", playerID)
		c.Set("username", claims.Username)

		if usedCookie {
			csrfProtection(c)
			if c.IsAborted() {
				return
			}
		}

		c.Next()
	}
}

// GetPlayerID retrieves the authenticated player ID from context
func GetPlayerID(c *gin.Context) (uuid.UUID, bool) {
	val, exists := c.Get("player_id")
	if !exists {
		return uuid.Nil, false
	}
	id, ok := val.(uuid.UUID)
	return id, ok
}

// AdminRequired is a compatibility alias. New routes should name the required
// permission explicitly with RequirePermission.
func AdminRequired() gin.HandlerFunc {
	return RequirePermission(authz.AdminAccessPermission)
}
