package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartStatusExpiryWorker periodically purges expired entries from player statuses.
func StartStatusExpiryWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	RunEvery(ctx, interval, "Status Expiry Worker", func(ctx context.Context) {
		processStatusExpiryTick(db)
	})
}

func processStatusExpiryTick(db *sql.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT id, statuses
		FROM players
		WHERE statuses @? '$[*] ? (exists(@.expires_at))'
		LIMIT 200
	`)
	if err != nil {
		logger.S.Errorf("Status Expiry Worker Error fetching players with expiring statuses: %v", err)
		return
	}
	defer rows.Close()

	now := time.Now()

	type playerStatusPurge struct {
		id       string
		original []byte
		statuses []byte
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
			if b, err := json.Marshal(list); err == nil {
				purges = append(purges, playerStatusPurge{id: id, original: rawStatuses, statuses: b})
			}
		}
	}
	if err := rows.Err(); err != nil {
		logger.S.Errorf("Status Expiry Worker Error reading players with expiring statuses: %v", err)
		return
	}

	if len(purges) == 0 {
		return
	}

	// Batch UPDATE using unnest for all affected players in one query
	ids := make([]string, len(purges))
	originalStatuses := make([]string, len(purges))
	statuses := make([]string, len(purges))
	for i, p := range purges {
		ids[i] = p.id
		originalStatuses[i] = string(p.original)
		statuses[i] = string(p.statuses)
	}

	_, err = db.ExecContext(ctx, `
		WITH updates(id, original_statuses, statuses) AS (
			SELECT unnest($1::text[]), unnest($2::text[]), unnest($3::text[])
		)
		UPDATE players p SET statuses = u.statuses::jsonb
		FROM updates u
		WHERE p.id = u.id::uuid AND p.statuses = u.original_statuses::jsonb
	`, pq.Array(ids), pq.Array(originalStatuses), pq.Array(statuses))
	if err != nil {
		logger.S.Errorf("Status Expiry Worker Error batch updating: %v", err)
		return
	}

	if len(purges) > 0 {
		logger.S.Infof("Status Expiry Worker: cleaned expired statuses for %d player(s)", len(purges))
	}
}
