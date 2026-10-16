package worker

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type BatchHandler func(ctx context.Context, payloads []string) error

var batchHandlers = map[string]BatchHandler{}

// RegisterBatchHandler registers a function to handle persisting batches for a specific entity type
func RegisterBatchHandler(entityType string, handler BatchHandler) {
	batchHandlers[entityType] = handler
}

// StartBatchPersister begins periodically flushing queued Redis changes to the database.
func StartBatchPersister(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		logger.S.Info("Batch persister worker started")

		for {
			select {
			case <-ctx.Done():
				logger.S.Info("Batch persister worker stopping, performing final flush...")
				// Force a final flush before stopping
				flushAll(context.Background())
				return
			case <-ticker.C:
				SafeRun("Batch Persister", ctx, flushAll)
			}
		}
	}()
}

func flushAll(ctx context.Context) {
	for entityType, handler := range batchHandlers {
		err := flushEntityType(ctx, entityType, handler)
		if err != nil {
			logger.S.Errorf("Error flushing entity type %s: %v", entityType, err)
		}
	}
}

func flushEntityType(ctx context.Context, entityType string, handler BatchHandler) error {
	ids, err := cache.GetPendingIDs(entityType)
	if err != nil || len(ids) == 0 {
		return nil
	}

	atomic.StoreInt64(&Metrics.QueueDepth, int64(len(ids)))
	if len(ids) > 100 {
		logger.S.Warnf("High queue depth for %s: %d items", entityType, len(ids))
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = entityType + ":" + id
	}

	payloads, err := cache.GetManyJSON(keys)
	if err != nil {
		return err
	}

	validPayloads, missingIDs := partitionPendingPayloads(ids, payloads)
	for _, id := range missingIDs {
		if _, err := cache.AcknowledgeMissingPendingJSON(entityType, id); err != nil {
			logger.S.Warnf("Failed to clean missing pending %s %s: %v", entityType, id, err)
		}
	}

	if len(validPayloads) == 0 {
		Metrics.RecordFlush(entityType, 0, true)
		return nil
	}

	if err := handler(ctx, validPayloads); err != nil {
		logger.S.Errorf("Batch persist failed for %s: %v", entityType, err)
		Metrics.RecordFlush(entityType, len(validPayloads), false)
		return err
	}

	Metrics.RecordFlush(entityType, len(validPayloads), true)
	return nil
}

func partitionPendingPayloads(ids, payloads []string) (validPayloads, missingIDs []string) {
	for i, id := range ids {
		if i >= len(payloads) || payloads[i] == "" {
			missingIDs = append(missingIDs, id)
			continue
		}
		validPayloads = append(validPayloads, payloads[i])
	}
	return validPayloads, missingIDs
}
