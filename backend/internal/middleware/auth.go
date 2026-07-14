package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
)

func csrfProtection(c *gin.Context) {
	csrfCookie, err := c.Cookie("gu_csrf_token")
	if err != nil || csrfCookie == "" {
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
	if csrfHeader == "" || csrfHeader != csrfCookie {
		c.JSON(http.StatusForbidden, gin.H{"error": "Invalid CSRF token"})
		c.Abort()
		return
	}
}

// AuthRequired is a Gin middleware that enforces authentication
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr, err := c.Cookie("gu_auth_token")
		if err != nil || tokenStr == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication cookie or Authorization header required"})
				c.Abort()
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be: Bearer <token>"})
				c.Abort()
				return
			}

			tokenStr = parts[1]
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

		// Store variables in context
		c.Set("player_id", playerID)
		c.Set("username", claims.Username)

		csrfProtection(c)
		if c.IsAborted() {
			return
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
