package middleware

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RealtimeOwnershipChecker func(uuid.UUID) bool

// RealtimeHPReader returns the authoritative zone HP for an online realtime
// character. ok is false when the character has no live zone entity.
type RealtimeHPReader func(uuid.UUID) (hp, maxHP float64, ok bool)

// RealtimeHPApplier applies a signed HP delta to the authoritative zone entity.
// It returns false when the character is not currently owned by a zone.
type RealtimeHPApplier func(uuid.UUID, float64) bool

type realtimeVitalsBridge struct {
	read  RealtimeHPReader
	apply RealtimeHPApplier
}

var (
	realtimeOwnershipChecker atomic.Pointer[RealtimeOwnershipChecker]
	realtimeVitals           atomic.Pointer[realtimeVitalsBridge]
)

// SetRealtimeOwnershipChecker installs the process-local ownership lookup.
// Passing nil clears it; the server sets this once during startup.
func SetRealtimeOwnershipChecker(checker RealtimeOwnershipChecker) {
	if checker == nil {
		realtimeOwnershipChecker.Store(nil)
		return
	}
	realtimeOwnershipChecker.Store(&checker)
}

// SetRealtimeVitalsBridge installs the live zone HP bridge used by legacy HTTP
// inventory actions (items and support Gu) while a character is in MMO mode.
// Passing nil for either callback clears the bridge.
func SetRealtimeVitalsBridge(read RealtimeHPReader, apply RealtimeHPApplier) {
	if read == nil || apply == nil {
		realtimeVitals.Store(nil)
		return
	}
	realtimeVitals.Store(&realtimeVitalsBridge{read: read, apply: apply})
}

func IsRealtimeWorldOwned(playerID uuid.UUID) bool {
	checker := realtimeOwnershipChecker.Load()
	return checker != nil && (*checker)(playerID)
}

// RealtimePlayerHP reads the live zone HP. It is false when the player is not
// in the MMO world, the zone entity is gone, or the character is dead.
func RealtimePlayerHP(playerID uuid.UUID) (hp, maxHP float64, ok bool) {
	bridge := realtimeVitals.Load()
	if bridge == nil {
		return 0, 0, false
	}
	hp, maxHP, ok = bridge.read(playerID)
	if !ok || hp <= 0 {
		return 0, 0, false
	}
	return hp, maxHP, true
}

// ApplyRealtimeHPDelta pushes an HP change made by a legacy HTTP action back to
// the live zone so the MMO HUD and checkpoint see the same result.
func ApplyRealtimeHPDelta(playerID uuid.UUID, delta float64) bool {
	bridge := realtimeVitals.Load()
	if bridge == nil || delta == 0 {
		return false
	}
	return bridge.apply(playerID, delta)
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
