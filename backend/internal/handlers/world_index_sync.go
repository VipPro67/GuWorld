package handlers

import (
	"context"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// worldIndexSyncInterval is how often the position index is reconciled against
// the database. The database is the durable source of truth; the index is a
// realtime overlay updated eagerly on movement. Reconcile bootstraps the index
// at startup and self-heals any stale/missed entry (retreats, teleports, crash
// recovery) within one interval.
const worldIndexSyncInterval = 30 * time.Second

// StartWorldIndexSyncWorker periodically reloads active players into the
// in-memory position index. It runs the initial reconcile immediately so the
// fast path is warm on the very first nearby-players query.
func StartWorldIndexSyncWorker(ctx context.Context) {
	reconcile := func() {
		entries := loadActiveWorldPositions()
		worldPosIndex.Reconcile(entries)
	}

	go func() {
		reconcile()
		ticker := time.NewTicker(worldIndexSyncInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				reconcile()
			}
		}
	}()
}

// loadActiveWorldPositions loads the position snapshot of every active player
// (matching the existing 15-minute active window) into a slice for Reconcile.
func loadActiveWorldPositions() []WorldPosEntry {
	rows, err := db.DB.Query(`
		SELECT id, username, COALESCE(avatar_url, ''), x, y, COALESCE(current_region, ''), last_mana_update
		FROM players
		WHERE last_mana_update >= NOW() - INTERVAL '16 minutes'
	`)
	if err != nil {
		logger.S.Warnf("world index sync: query failed: %v", err)
		return nil
	}
	defer rows.Close()

	entries := make([]WorldPosEntry, 0, 512)
	for rows.Next() {
		var e WorldPosEntry
		var ts time.Time
		if err := rows.Scan(&e.ID, &e.Username, &e.AvatarURL, &e.X, &e.Y, &e.Region, &ts); err != nil {
			continue
		}
		e.TS = ts.UnixMilli()
		entries = append(entries, e)
	}
	return entries
}
