package handlers

import (
	"context"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// worldIndexSyncInterval is how often the position index is reconciled against
// the database. The database is the durable source of truth; the index is a
// realtime overlay updated eagerly on movement. Reconcile bootstraps the index
// at startup and self-heals any stale/missed entry (retreats, teleports, crash
// recovery) within one interval.
const worldIndexSyncInterval = 30 * time.Second

var lastEntryCount int

// StartWorldIndexSyncWorker periodically reloads active players into the
// in-memory position index. It runs the initial reconcile immediately so the
// fast path is warm on the very first nearby-players query.
func StartWorldIndexSyncWorker(ctx context.Context) {
	reconcile := func(ctx context.Context) {
		entries := loadActiveWorldPositions()
		if len(entries) == 0 && lastEntryCount == 0 {
			return
		}
		worldPosIndex.Reconcile(entries)
		lastEntryCount = len(entries)
	}

	// Warm the fast path immediately (off the startup path), then reconcile on
	// every tick.
	go worker.SafeRun("World Index Sync", ctx, reconcile)
	worker.RunEvery(ctx, worldIndexSyncInterval, "World Index Sync", reconcile)
}

// loadActiveWorldPositions loads the position snapshot of every active player
// (matching the existing 15-minute active window) into a slice for Reconcile.
func loadActiveWorldPositions() []WorldPosEntry {
	rows, err := db.DB.Query(`
		SELECT p.id, p.username, COALESCE(p.avatar_url, ''), p.x, p.y, COALESCE(p.current_region, ''),
		       COALESCE(rwm.world_map_id,'world_map'), COALESCE(p.map_instance_id::text,''), p.last_mana_update
		FROM players p
		LEFT JOIN region_world_maps rwm ON rwm.region_id=p.current_region
		WHERE (last_activity >= NOW() - INTERVAL '5 minutes' OR last_mana_update >= NOW() - INTERVAL '5 minutes')
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
		if err := rows.Scan(&e.ID, &e.Username, &e.AvatarURL, &e.X, &e.Y, &e.Region, &e.MapID, &e.InstanceID, &ts); err != nil {
			continue
		}
		e.TS = ts.UnixMilli()
		entries = append(entries, e)
	}
	return entries
}
