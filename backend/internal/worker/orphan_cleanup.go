package worker

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartOrphanedKeyCleanup periodically removes cache entries that are no longer
// in the pending queue. This prevents stale data from accumulating in Redis
// indefinitely (e.g., when a payload was queued but later superseded by a newer
// update that didn't remove the old entry).
func StartOrphanedKeyCleanup(ctx context.Context, interval time.Duration) {
	RunEvery(ctx, interval, "Orphaned Key Cleanup", func(ctx context.Context) {
		cleanupOrphanedKeys(ctx)
	})
}

func cleanupOrphanedKeys(ctx context.Context) {
	if cache.Client == nil {
		return
	}

	for entityType := range batchHandlers {
		ids, err := cache.GetPendingIDs(entityType)
		if err != nil {
			continue
		}

		// Build a set of pending IDs for fast lookup
		pending := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			pending[id] = struct{}{}
		}

		// Scan all keys for this entity type
		pattern := entityType + ":*"
		var cursor uint64
		var orphanCount int64

		for {
			keys, nextCursor, err := cache.Client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				break
			}

			for _, key := range keys {
				// Extract entity ID from key: "entityType:entityID"
				entityID := key[len(entityType)+1:]
				if _, exists := pending[entityID]; !exists {
					_ = cache.Client.Del(ctx, key)
					orphanCount++
				}
			}

			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}

		if orphanCount > 0 {
			logger.S.Infof("Cleaned up %d orphaned keys for %s", orphanCount, entityType)
			atomic.AddInt64(&Metrics.ItemsDropped, orphanCount)
		}
	}
}
