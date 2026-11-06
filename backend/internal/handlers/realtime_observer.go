package handlers

import (
	"sync"

	"github.com/google/uuid"
)

// RealtimeObserver defines the read-only seam between gameplay handlers and the
// realtime world/transport subsystem (WT-311, WT-316). It replaces loose, mutable
// function pointers across package boundaries with an explicit interface contract.
type RealtimeObserver interface {
	IsConnected(playerID uuid.UUID) bool
	GetEventStats() (clients int, channels int)
	GetZoneStats() any
}

var (
	activeRealtimeObserverMu sync.RWMutex
	activeRealtimeObserver   RealtimeObserver
)

// SetRealtimeObserver installs the primary realtime observer implementation at server bootstrap.
func SetRealtimeObserver(obs RealtimeObserver) {
	activeRealtimeObserverMu.Lock()
	defer activeRealtimeObserverMu.Unlock()
	activeRealtimeObserver = obs
}

// GetRealtimeObserver returns the installed observer, if any.
func GetRealtimeObserver() RealtimeObserver {
	activeRealtimeObserverMu.RLock()
	defer activeRealtimeObserverMu.RUnlock()
	return activeRealtimeObserver
}
