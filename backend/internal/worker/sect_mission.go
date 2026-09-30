package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartSectMissionWorker periodically expires sect missions: player missions
// whose expires_at has passed are marked EXPIRED, then board missions that
// expired are deleted (player rows cascade via FK).
func StartSectMissionWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Sect Mission Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Sect Mission Worker stopped")
				return
			case <-ticker.C:
				processSectMissionTick(db)
			}
		}
	}()
}

func processSectMissionTick(db *sql.DB) {
	// Batched by ctid so each statement locks at most 500 rows instead of
	// scanning and locking the whole table under one statement_timeout.
	expired, err := db.Exec(`
		UPDATE player_sect_missions SET status = 'EXPIRED'
		WHERE ctid IN (
			SELECT ctid FROM player_sect_missions
			WHERE status = 'IN_PROGRESS' AND expires_at IS NOT NULL AND expires_at <= now()
			LIMIT 500
		)
	`)
	if err != nil {
		logger.S.Errorf("Sect Mission Worker Error expiring player missions: %v", err)
	} else if n, _ := expired.RowsAffected(); n > 0 {
		logger.S.Infof("Sect Mission Worker: marked %d player mission(s) EXPIRED", n)
	}

	deleted, err := db.Exec(`
		DELETE FROM sect_missions
		WHERE ctid IN (
			SELECT ctid FROM sect_missions
			WHERE expires_at IS NOT NULL AND expires_at <= now()
			LIMIT 200
		)
	`)
	if err != nil {
		logger.S.Errorf("Sect Mission Worker Error deleting expired missions: %v", err)
	} else if n, _ := deleted.RowsAffected(); n > 0 {
		logger.S.Infof("Sect Mission Worker: removed %d expired board mission(s)", n)
	}
}
