package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

// onboardedExemptPaths allow the onboarding endpoints themselves through.
var onboardedExemptPaths = map[string]bool{
	"/api/player/complete-tutorial":   true,
	"/api/player/complete-onboarding": true,
}

// RequireOnboarded blocks POST/PUT/DELETE mutations until the authenticated
// player has completed onboarding. GET/HEAD reads are always allowed so the
// client can render lv0/Onboarding progress from /player/status and friends.
//
// ponytail: one indexed needs_onboarding lookup per mutating request. If this
// shows up in profiles, cache the flag on the session (auth:session:<id>).
func RequireOnboarded() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}
		if onboardedExemptPaths[c.Request.URL.Path] {
			c.Next()
			return
		}
		playerID, ok := GetPlayerID(c)
		if !ok {
			// Auth middleware (or a public endpoint) must handle unauthenticated.
			c.Next()
			return
		}
		var needsOnboarding bool
		if err := db.DB.QueryRowContext(c.Request.Context(),
			"SELECT needs_onboarding FROM players WHERE id = $1", playerID,
		).Scan(&needsOnboarding); err != nil {
			// Fail-open: let downstream handlers surface DB errors.
			c.Next()
			return
		}
		if needsOnboarding {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Đạo hữu cần hoàn thành giai đoạn Tân Thủ trước khi hành động này.",
			})
			return
		}
		c.Next()
	}
}
