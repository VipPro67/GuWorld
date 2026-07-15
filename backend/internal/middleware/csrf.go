package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"strings"

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

	domain := os.Getenv("AUTH_COOKIE_DOMAIN")

	isSecure := false
	secureEnv := os.Getenv("AUTH_COOKIE_SECURE")
	if secureEnv != "" {
		isSecure = strings.EqualFold(secureEnv, "true")
	} else if sameSite == http.SameSiteNoneMode {
		isSecure = true
	} else {
		isSecure = c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https"
	}

	c.SetSameSite(sameSite)
	c.SetCookie(csrfCookieName, token, 86400*7, "/", domain, isSecure, false) // non-HttpOnly so JS can read it
}

func ClearCSRFCookie(c *gin.Context) {
	sameSite := http.SameSiteLaxMode
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

	domain := os.Getenv("AUTH_COOKIE_DOMAIN")

	isSecure := false
	secureEnv := os.Getenv("AUTH_COOKIE_SECURE")
	if secureEnv != "" {
		isSecure = strings.EqualFold(secureEnv, "true")
	} else if sameSite == http.SameSiteNoneMode {
		isSecure = true
	} else {
		isSecure = c.Request.TLS != nil || c.Request.Header.Get("X-Forwarded-Proto") == "https"
	}

	c.SetSameSite(sameSite)
	c.SetCookie(csrfCookieName, "", -1, "/", domain, isSecure, false)
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
