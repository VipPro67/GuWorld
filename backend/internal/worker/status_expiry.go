package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartStatusExpiryWorker periodically purges expired entries from every
// player's statuses JSONB array. Expiry is wall-clock based; effect resolution
// also filters expired entries lazily, so this worker only keeps the stored
// data tidy (and makes admin views accurate).
func StartStatusExpiryWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Status Expiry Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Status Expiry Worker stopped")
				return
			case <-ticker.C:
				processStatusExpiryTick(db)
			}
		}
	}()
}

// processStatusExpiryTick rewrites the statuses array for every player that
// has at least one entry carrying an expires_at, dropping elements whose
// expires_at has passed. Permanent entries (no expires_at) are kept as-is.
func processStatusExpiryTick(db *sql.DB) {
	tx, err := db.Begin()
	if err != nil {
		logger.S.Errorf("Status Expiry Worker Error starting transaction: %v", err)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`SET LOCAL statement_timeout = '30000'`); err != nil {
		logger.S.Errorf("Status Expiry Worker Error setting statement_timeout: %v", err)
		return
	}
	if _, err := tx.Exec(`SET LOCAL lock_timeout = '30000'`); err != nil {
		logger.S.Errorf("Status Expiry Worker Error setting lock_timeout: %v", err)
		return
	}

	res, err := tx.Exec(`
		UPDATE players p
		SET statuses = COALESCE((
			SELECT jsonb_agg(s ORDER BY idx)
			FROM jsonb_array_elements(p.statuses) WITH ORDINALITY AS t(s, idx)
			WHERE (s->>'expires_at') IS NULL
			   OR (s->>'expires_at')::timestamptz > now()
		), '[]'::jsonb)
		WHERE jsonb_path_exists(p.statuses, '$[*] ? (@.expires_at != null)')
		  AND EXISTS (
			SELECT 1
			FROM jsonb_array_elements(p.statuses) s
			WHERE (s->>'expires_at') IS NOT NULL
			  AND (s->>'expires_at')::timestamptz <= now()
		  )
	`)
	if err != nil {
		logger.S.Errorf("Status Expiry Worker Error purging expired statuses: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Status Expiry Worker Error committing transaction: %v", err)
		return
	}

	if n, _ := res.RowsAffected(); n > 0 {
		logger.S.Infof("Status Expiry Worker: cleaned expired statuses for %d player(s)", n)
	}
}
