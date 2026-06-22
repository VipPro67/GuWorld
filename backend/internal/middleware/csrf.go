package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	csrfCookieName = "gu_csrf_token"
	csrfHeaderName = "X-CSRF-Token"
)

func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func SetCSRFCookie(c *gin.Context, token string) {
	sameSite := http.SameSiteLaxMode
	origin := c.Request.Header.Get("Origin")
	corsEnv := c.GetHeader("Origin") // we just use a basic check
	_ = corsEnv
	if origin != "" && origin != c.Request.Host {
		sameSite = http.SameSiteNoneMode
	}
	c.SetSameSite(sameSite)
	c.SetCookie(csrfCookieName, token, 86400*7, "/", "", false, false) // non-HttpOnly so JS can read it
}

func ClearCSRFCookie(c *gin.Context) {
	c.SetCookie(csrfCookieName, "", -1, "/", "", false, false)
}

func CSRFRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

		csrfCookie, err := c.Cookie(csrfCookieName)
		if err != nil || csrfCookie == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Missing CSRF token"})
			c.Abort()
			return
		}

		csrfHeader := c.GetHeader(csrfHeaderName)
		if csrfHeader == "" || csrfHeader != csrfCookie {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid CSRF token"})
			c.Abort()
			return
		}

		c.Next()
	}
}
