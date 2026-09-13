package handlers

import (
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func normalizeEmail(email string) string {
	email = strings.ToLower(strings.TrimSpace(email))
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	local := parts[0]
	domain := parts[1]

	// Handle Gmail-specific normalization
	if domain == "gmail.com" || domain == "googlemail.com" {
		// Remove everything after '+'
		if idx := strings.Index(local, "+"); idx != -1 {
			local = local[:idx]
		}
		// Remove dots '.'
		local = strings.ReplaceAll(local, ".", "")
	} else {
		// For non-gmail, still remove plus aliases as it's common (RFC 5233)
		if idx := strings.Index(local, "+"); idx != -1 {
			local = local[:idx]
		}
	}
	return local + "@" + domain
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
	Race           string `json:"race"`
	Gender         string `json:"gender"`
	Fingerprint    string `json:"fingerprint" binding:"required"`
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
