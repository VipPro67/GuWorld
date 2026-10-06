package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	combatHistoryCleanupBatchSize  = 1000
	combatHistoryCleanupMaxBatches = 10
)

// StartCombatHistoryCleanupWorker removes legacy PVE entries from
// combat_history while preserving every PVP entry, including the per-player
// copy written for each participant.
func StartCombatHistoryCleanupWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		logger.S.Infof("Starting Combat History Cleanup Worker with tick interval of %v", interval)
		cleanupCombatHistory(ctx, db)
		for {
			select {
			case <-ctx.Done():
				logger.S.Info("Combat History Cleanup Worker stopped")
				return
			case <-ticker.C:
				cleanupCombatHistory(ctx, db)
			}
		}
	}()
}

func cleanupCombatHistory(parent context.Context, db *sql.DB) {
	if db == nil {
		return
	}
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	var totalDeleted int64
	for i := 0; i < combatHistoryCleanupMaxBatches; i++ {
		result, err := db.ExecContext(ctx, `
			WITH pve AS (
				SELECT id
				FROM combat_history
				WHERE UPPER(enemy_type) <> 'PVP'
				ORDER BY created_at, id
				LIMIT $1
				FOR UPDATE SKIP LOCKED
			)
			DELETE FROM combat_history h
			USING pve
			WHERE h.id = pve.id
		`, combatHistoryCleanupBatchSize)
		if err != nil {
			logger.S.Errorf("Combat History Cleanup Worker failed: %v", err)
			return
		}
		deleted, err := result.RowsAffected()
		if err != nil {
			logger.S.Errorf("Combat History Cleanup Worker could not read deleted row count: %v", err)
			return
		}
		totalDeleted += deleted
		if deleted < combatHistoryCleanupBatchSize {
			break
		}
	}
	if totalDeleted > 0 {
		logger.S.Infof("Combat History Cleanup Worker deleted %d PVE entries", totalDeleted)
	}
}
