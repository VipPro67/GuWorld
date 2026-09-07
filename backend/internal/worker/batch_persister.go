package worker

import (
	"context"
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
				flushAll(ctx)
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

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = entityType + ":" + id
	}

	payloads, err := cache.GetManyJSON(keys)
	if err != nil {
		return err
	}

	var validPayloads []string
	var validIDs []string
	for i, payload := range payloads {
		if payload != "" {
			validPayloads = append(validPayloads, payload)
			validIDs = append(validIDs, ids[i])
		}
	}

	if len(validPayloads) == 0 {
		return cache.RemovePendingIDs(entityType, ids...)
	}

	if err := handler(ctx, validPayloads); err != nil {
		logger.S.Errorf("Batch persist failed for %s: %v", entityType, err)
		return err 
	}

	return cache.RemovePendingIDs(entityType, validIDs...)
}
