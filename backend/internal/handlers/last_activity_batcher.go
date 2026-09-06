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

	ids := make([]uuid.UUID, 0, len(snapshot))
	for id := range snapshot {
		ids = append(ids, id)
	}

	_, err := db.DB.Exec(
		`UPDATE players SET last_activity = NOW() WHERE id = ANY($1)`,
		pq.Array(ids),
	)
	if err != nil {
		logger.S.Errorf("Last-activity batch flush failed (%d players): %v", len(ids), err)
	}
}

// StartLastActivityFlusher starts the background batch flusher for SSE heartbeat
// last_activity updates.  Called from main.go during startup.
func StartLastActivityFlusher(ctx context.Context) {
	lastActivity.StartFlusher(ctx, 30*time.Second)
}
