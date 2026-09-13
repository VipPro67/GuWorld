package handlers

import (
	"database/sql"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
)

var (
	wsConnectedMu sync.RWMutex
	wsConnectedFn func(playerID uuid.UUID) bool
)

// SetWSConnectedProvider sets the provider function for checking live WebSocket connections.
func SetWSConnectedProvider(fn func(playerID uuid.UUID) bool) {
	wsConnectedMu.Lock()
	defer wsConnectedMu.Unlock()
	wsConnectedFn = fn
}

func isWSConnected(playerID uuid.UUID) bool {
	wsConnectedMu.RLock()
	fn := wsConnectedFn
	wsConnectedMu.RUnlock()
	if fn == nil {
		return false
	}
	return fn(playerID)
}

// isPlayerOnline reports whether the player has been active within the online window.
// It checks 3 tiers:
// 1. Live WebSocket connection in ws.Hub (instant O(1), zero DB query).
// 2. In-memory recent activity from WebSocket pings/messages (lastActivityStore.recent).
// 3. Persistent players.last_activity in PostgreSQL (fallback).
func isPlayerOnline(playerID string) bool {
	uid, err := uuid.Parse(playerID)
	if err == nil {
		// Tier 1: Real-time WebSocket connection
		if isWSConnected(uid) {
			return true
		}
		// Tier 2: Recent activity in memory
		if IsPlayerActiveRecently(uid, tribulationOnlineWindow) {
			return true
		}
	}

	// Tier 3: Fallback to PostgreSQL
	if db.DB == nil {
		return false
	}
	var la sql.NullTime
	err = db.DB.QueryRow(`SELECT last_activity FROM players WHERE id = $1`, playerID).Scan(&la)
	if err != nil || !la.Valid {
		return false
	}
	return time.Since(la.Time) <= tribulationOnlineWindow
}
