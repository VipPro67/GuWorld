package middleware

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RealtimeOwnershipChecker func(uuid.UUID) bool

var realtimeOwnershipChecker atomic.Pointer[RealtimeOwnershipChecker]

// SetRealtimeOwnershipChecker installs the process-local ownership lookup.
// Passing nil clears it; the server sets this once during startup.
func SetRealtimeOwnershipChecker(checker RealtimeOwnershipChecker) {
	if checker == nil {
		realtimeOwnershipChecker.Store(nil)
		return
	}
	realtimeOwnershipChecker.Store(&checker)
}

func IsRealtimeWorldOwned(playerID uuid.UUID) bool {
	checker := realtimeOwnershipChecker.Load()
	return checker != nil && (*checker)(playerID)
}

// RequireLegacyWorldMode blocks legacy world/combat mutations for a character
// whose authoritative controller lease belongs to the realtime runtime.
func RequireLegacyWorldMode() gin.HandlerFunc {
	return func(c *gin.Context) {
		playerID, ok := GetPlayerID(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			return
		}
		if IsRealtimeWorldOwned(playerID) {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{
				"code":  "realtime_world_owns_character",
				"error": "Character is controlled by the realtime world",
			})
			return
		}
		c.Next()
	}
}
