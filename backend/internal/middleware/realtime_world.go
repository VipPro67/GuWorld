package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RealtimeOwnershipChecker func(uuid.UUID) bool
type RealtimeHPReader func(uuid.UUID) (hp, maxHP float64, ok bool)
type RealtimeHPApplier func(uuid.UUID, float64) bool

type RealtimeHandoffBridge struct {
	Begin  func(uuid.UUID) (bool, error)
	Abort  func(uuid.UUID)
	Finish func(uuid.UUID)
}

type RealtimeTribulationEnemyReader func(uuid.UUID) (alive, dead map[string]bool, ok bool)
type RealtimeLocationReader func(uuid.UUID) (regionID string, x, y int, ok bool)

func SetRealtimeOwnershipChecker(_ RealtimeOwnershipChecker) {}
func SetRealtimeVitalsBridge(_ RealtimeHPReader, _ RealtimeHPApplier) {}
func IsRealtimeWorldOwned(_ uuid.UUID) bool { return false }
func SetRealtimeHandoffBridge(_ *RealtimeHandoffBridge) {}
func SetRealtimeLocationReader(_ RealtimeLocationReader) {}
func SetRealtimeTribulationEnemyReader(_ RealtimeTribulationEnemyReader) {}
func RealtimeTribulationEnemies(_ uuid.UUID) (alive, dead map[string]bool, ok bool) { return nil, nil, false }
func RealtimePlayerLocation(_ uuid.UUID) (regionID string, x, y int, ok bool) { return "", 0, 0, false }
func BeginRealtimeHandoff(_ uuid.UUID) (bool, error) { return false, nil }
func AbortRealtimeHandoff(_ uuid.UUID) {}
func FinishRealtimeHandoff(_ uuid.UUID) {}
func RealtimePlayerHP(_ uuid.UUID) (hp, maxHP float64, ok bool) { return 0, 0, false }
func ApplyRealtimeHPDelta(_ uuid.UUID, _ float64) bool { return false }

// RequireLegacyWorldMode is kept for route middleware compatibility and passes through.
func RequireLegacyWorldMode() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
