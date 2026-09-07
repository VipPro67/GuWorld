package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartStatusExpiryWorker periodically purges expired entries from player statuses.
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

func processStatusExpiryTick(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT id, statuses 
		FROM players 
		WHERE statuses != '[]'::jsonb AND statuses::text LIKE '%expires_at%'
		LIMIT 200
	`)
	if err != nil {
		logger.S.Errorf("Status Expiry Worker Error fetching players with expiring statuses: %v", err)
		return
	}
	defer rows.Close()

	now := time.Now()
	cleanedCount := 0

	type playerStatusPurge struct {
		id       string
		statuses models.StatusList
	}
	var purges []playerStatusPurge

	for rows.Next() {
		var id string
		var rawStatuses []byte
		if err := rows.Scan(&id, &rawStatuses); err != nil {
			continue
		}
		var list models.StatusList
		if err := json.Unmarshal(rawStatuses, &list); err != nil {
			continue
		}
		if list.Expire(now) {
			purges = append(purges, playerStatusPurge{id: id, statuses: list})
		}
	}
	_ = rows.Close()

	for _, p := range purges {
		b, err := json.Marshal(p.statuses)
		if err != nil {
			continue
		}
		_, err = db.ExecContext(ctx, `UPDATE players SET statuses = $1 WHERE id = $2`, string(b), p.id)
		if err != nil {
			logger.S.Warnf("Status Expiry Worker: failed to update player %s: %v", p.id, err)
		} else {
			cleanedCount++
		}
	}

	if cleanedCount > 0 {
		logger.S.Infof("Status Expiry Worker: cleaned expired statuses for %d player(s)", cleanedCount)
	}
}
