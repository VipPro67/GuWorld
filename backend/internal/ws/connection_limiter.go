package ws

import (
	"sync"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ConnectionLimiter enforces limits on WebSocket connections per player and per IP.
type ConnectionLimiter struct {
	mu                sync.Mutex
	playerConnections map[uuid.UUID]int
	ipConnections     map[string]int
	maxPerPlayer      int
	maxPerIP          int
}

var globalLimiter *ConnectionLimiter

// InitConnectionLimiter initializes the global connection limiter.
func InitConnectionLimiter(maxPerPlayer, maxPerIP int) {
	globalLimiter = &ConnectionLimiter{
		playerConnections: make(map[uuid.UUID]int),
		ipConnections:     make(map[string]int),
		maxPerPlayer:      maxPerPlayer,
		maxPerIP:          maxPerIP,
	}
	logger.S.Infof("WebSocket Connection Limiter initialized: max %d per player, %d per IP", maxPerPlayer, maxPerIP)
}

// IsConnectionAllowed checks if a new WebSocket connection is allowed.
func IsConnectionAllowed(playerID uuid.UUID, clientIP string) bool {
	if globalLimiter == nil {
		return true
	}
	return globalLimiter.CheckConnection(playerID, clientIP)
}

// CheckConnection validates if a new WebSocket connection is allowed.
func (lim *ConnectionLimiter) CheckConnection(playerID uuid.UUID, clientIP string) bool {
	lim.mu.Lock()
	defer lim.mu.Unlock()

	if playerID != uuid.Nil {
		if lim.playerConnections[playerID] >= lim.maxPerPlayer {
			logger.S.Warnf("WebSocket rate limit exceeded for player %s: %d/%d connections", playerID, lim.playerConnections[playerID], lim.maxPerPlayer)
			return false
		}
		lim.playerConnections[playerID]++
	}

	if lim.ipConnections[clientIP] >= lim.maxPerIP {
		logger.S.Warnf("WebSocket rate limit exceeded for IP %s: %d/%d connections", clientIP, lim.ipConnections[clientIP], lim.maxPerIP)
		return false
	}
	lim.ipConnections[clientIP]++
	return true
}

// ReleaseConnection decrements connection counts when a client disconnects.
func ReleaseConnection(playerID uuid.UUID, clientIP string) {
	if globalLimiter == nil {
		return
	}
	globalLimiter.ReleaseConnection(playerID, clientIP)
}

// ReleaseConnection decrements connection counts when a client disconnects.
func (lim *ConnectionLimiter) ReleaseConnection(playerID uuid.UUID, clientIP string) {
	lim.mu.Lock()
	defer lim.mu.Unlock()

	if playerID != uuid.Nil {
		if lim.playerConnections[playerID] > 0 {
			lim.playerConnections[playerID]--
			if lim.playerConnections[playerID] == 0 {
				delete(lim.playerConnections, playerID)
			}
		}
	}
	if lim.ipConnections[clientIP] > 0 {
		lim.ipConnections[clientIP]--
		if lim.ipConnections[clientIP] == 0 {
			delete(lim.ipConnections, clientIP)
		}
	}
}

// GetConnectionCount returns the number of connections for a player.
func GetConnectionCount(playerID uuid.UUID) int {
	if globalLimiter == nil {
		return 0
	}
	globalLimiter.mu.Lock()
	defer globalLimiter.mu.Unlock()
	return globalLimiter.playerConnections[playerID]
}
