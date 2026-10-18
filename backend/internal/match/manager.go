package match

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/worker"
)

// reapGraceTicks keeps a finished match around briefly so the final frames and
// the outcome reach the client before it is dropped.
const reapGraceTicks = 40

// Manager owns every live match. It is a plain in-memory map: a server restart
// loses live matches, which is acceptable for the demo and avoids replaying
// per-tick state through the database.
//
// ponytail: in-memory only, single process. Move to Redis-backed state if
// matches must survive a restart or span instances.
type Manager struct {
	mu      sync.RWMutex
	matches map[uuid.UUID]*Match
}

// Global is the process-wide match manager.
var Global = &Manager{matches: map[uuid.UUID]*Match{}}

func (m *Manager) Add(match *Match) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.matches[match.ID] = match
}

func (m *Manager) Get(id uuid.UUID) (*Match, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	match, ok := m.matches[id]
	return match, ok
}

func (m *Manager) Remove(id uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.matches, id)
}

// ForPlayer finds the match a player is currently in, preferring a live one so
// a rematch is never shadowed by the previous result while it is still being
// reaped.
func (m *Manager) ForPlayer(playerID uuid.UUID) (*Match, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var ended *Match
	for _, match := range m.matches {
		if !match.HasPlayer(playerID) {
			continue
		}
		if !match.IsEnded() {
			return match, true
		}
		ended = match
	}
	if ended != nil {
		return ended, true
	}
	return nil, false
}

// LiveMatchFor reports only matches still in progress.
func (m *Manager) LiveMatchFor(playerID uuid.UUID) (*Match, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, match := range m.matches {
		if match.HasPlayer(playerID) && !match.IsEnded() {
			return match, true
		}
	}
	return nil, false
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.matches)
}

// TickAll advances every live match once. Matches are copied out under the read
// lock and stepped outside it so a slow match cannot block new match creation.
func (m *Manager) TickAll() {
	m.mu.RLock()
	all := make([]*Match, 0, len(m.matches))
	for _, match := range m.matches {
		all = append(all, match)
	}
	m.mu.RUnlock()

	for _, match := range all {
		match.Step()
	}

	m.mu.Lock()
	for id, match := range m.matches {
		if match.ShouldReap() {
			delete(m.matches, id)
		}
	}
	m.mu.Unlock()
}

// StartTickLoop drives every match from a single ticker. One ticker for all
// matches keeps goroutine count bounded.
//
// ponytail: single ticker, one lock per match. Shard the loop if the number of
// concurrent matches ever makes a 50ms budget tight.
func StartTickLoop(ctx context.Context) {
	worker.RunEvery(ctx, TickInterval, "PvP Match Tick", func(context.Context) {
		Global.TickAll()
	})
}
