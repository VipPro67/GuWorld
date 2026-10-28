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

// RealtimeTeleportBridge suspends the authoritative zone checkpoint for one
// legacy HTTP teleport, then either releases the lease after the destination is
// persisted or resumes normal checkpoints when the teleport is refused.
type RealtimeTeleportBridge struct {
	Begin  func(uuid.UUID) (bool, error)
	Abort  func(uuid.UUID)
	Finish func(uuid.UUID)
}

var (
	realtimeOwnershipChecker atomic.Pointer[RealtimeOwnershipChecker]
	realtimeVitals           atomic.Pointer[realtimeVitalsBridge]
	realtimeTeleport         atomic.Pointer[RealtimeTeleportBridge]
	realtimeLocation         atomic.Pointer[RealtimeLocationReader]
)

// RealtimeLocationReader reports the live region and legacy-projected position
// of a character owned by a zone session.
type RealtimeLocationReader func(uuid.UUID) (regionID string, x, y int, ok bool)

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

// SetRealtimeTeleportBridge installs the lease handoff used by legacy HTTP
// teleport handlers. Passing nil clears it.
func SetRealtimeTeleportBridge(bridge *RealtimeTeleportBridge) {
	if bridge == nil || bridge.Begin == nil || bridge.Abort == nil || bridge.Finish == nil {
		realtimeTeleport.Store(nil)
		return
	}
	realtimeTeleport.Store(bridge)
}

// SetRealtimeLocationReader installs the live position lookup used by legacy
// handlers that must read the character's MMO cell instead of the last
// checkpoint. Passing nil clears it.
func SetRealtimeLocationReader(reader RealtimeLocationReader) {
	if reader == nil {
		realtimeLocation.Store(nil)
		return
	}
	realtimeLocation.Store(&reader)
}

// RealtimePlayerLocation returns the live region and legacy-projected position.
// ok is false when the character has no active zone session.
func RealtimePlayerLocation(playerID uuid.UUID) (regionID string, x, y int, ok bool) {
	reader := realtimeLocation.Load()
	if reader == nil {
		return "", 0, 0, false
	}
	return (*reader)(playerID)
}

// BeginRealtimeTeleport suspends zone checkpoints and flushes the live position
// so a legacy teleport reads the character's real coordinates. It reports
// whether a live session was handed off.
func BeginRealtimeTeleport(playerID uuid.UUID) (bool, error) {
	bridge := realtimeTeleport.Load()
	if bridge == nil {
		return false, nil
	}
	return bridge.Begin(playerID)
}

// AbortRealtimeTeleport resumes checkpoints after a refused teleport.
func AbortRealtimeTeleport(playerID uuid.UUID) {
	if bridge := realtimeTeleport.Load(); bridge != nil {
		bridge.Abort(playerID)
	}
}

// FinishRealtimeTeleport releases the suspended lease after the teleport has
// been persisted. The client is expected to rejoin the destination zone.
func FinishRealtimeTeleport(playerID uuid.UUID) {
	if bridge := realtimeTeleport.Load(); bridge != nil {
		bridge.Finish(playerID)
	}
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
