package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// lastActivityStore collects last_activity timestamps in memory and flushes
// them to the database in a single batch UPDATE every flushInterval.
// This replaces the previous pattern where every SSE heartbeat ran an
// individual UPDATE players SET last_activity = NOW() query.
type lastActivityStore struct {
	mu    sync.Mutex
	dirty map[uuid.UUID]time.Time
}

var lastActivity = &lastActivityStore{
	dirty: make(map[uuid.UUID]time.Time),
}

// Touch records a player as recently active.  Called from SSE heartbeat
// loops – must be cheap (memory-only, no DB round-trip).
func (s *lastActivityStore) Touch(id uuid.UUID) {
	s.mu.Lock()
	s.dirty[id] = time.Now()
	s.mu.Unlock()
}

// StartFlusher starts a background goroutine that periodically writes all
// buffered last_activity updates to the database in one batch query.
func (s *lastActivityStore) StartFlusher(ctx context.Context, flushInterval time.Duration) {
	go func() {
		ticker := time.NewTicker(flushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				s.flush()
				logger.S.Info("Last-activity batcher stopped")
				return
			case <-ticker.C:
				s.flush()
			}
		}
	}()
	logger.S.Infof("Last-activity batcher started (flush every %v)", flushInterval)
}

func (s *lastActivityStore) flush() {
	s.mu.Lock()
	if len(s.dirty) == 0 {
		s.mu.Unlock()
		return
	}
	// Swap out the map so the lock is held for minimum time.
	snapshot := s.dirty
	s.dirty = make(map[uuid.UUID]time.Time, len(snapshot))
	s.mu.Unlock()

	if len(snapshot) == 0 {
		return
	}

	// Batch UPDATE using unnest for all dirty players
	ids := make([]string, len(snapshot))
	times := make([]time.Time, len(snapshot))
	i := 0
	for id, lastTime := range snapshot {
		ids[i] = id.String()
		times[i] = lastTime
		i++
	}

	_, err := db.DB.Exec(`
		WITH updates(id, last_activity) AS (
			SELECT unnest($1::text[]), unnest($2::timestamptz[])
		)
		UPDATE players p SET last_activity = u.last_activity
		FROM updates u WHERE p.id = u.id::uuid
	`, pq.Array(ids), pq.Array(times))
	if err != nil {
		logger.S.Warnf("Last-activity flush failed for %d players: %v", len(snapshot), err)
		// Re-add failed players back to dirty set
		s.mu.Lock()
		for id, lastTime := range snapshot {
			if _, exists := s.dirty[id]; !exists {
				s.dirty[id] = lastTime
			}
		}
		s.mu.Unlock()
		return
	}

	logger.S.Debugf("Last-activity flush: updated %d player(s)", len(snapshot))
}

// StartLastActivityFlusher starts the background batch flusher for SSE heartbeat
// last_activity updates.  Called from main.go during startup.
func StartLastActivityFlusher(ctx context.Context) {
	lastActivity.StartFlusher(ctx, 30*time.Second)
}
