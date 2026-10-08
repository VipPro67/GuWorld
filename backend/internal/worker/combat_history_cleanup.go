package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// combatHistoryRetentionDays is how long PVE defeat replays are kept. PVP rows
// (enemy_type = 'PVP') are retained indefinitely.
const combatHistoryRetentionDays = 30

// StartCombatHistoryCleanup periodically deletes PVE combat history rows older
// than the retention window so the table does not grow without bound. PVP rows
// are never touched. Uses idx_combat_history_pve_cleanup.
func StartCombatHistoryCleanup(ctx context.Context, database *sql.DB, interval time.Duration) {
	if database == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		logger.S.Info("Combat history cleanup worker started")
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupCombatHistory(ctx, database)
			}
		}
	}()
}

func cleanupCombatHistory(ctx context.Context, database *sql.DB) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	res, err := database.ExecContext(ctx, `
		DELETE FROM combat_history
		WHERE UPPER(enemy_type) <> 'PVP' AND created_at < NOW() - make_interval(days => $1)`,
		combatHistoryRetentionDays)
	if err != nil {
		logger.S.Errorf("Combat history cleanup failed: %v", err)
		return
	}
	if n, _ := res.RowsAffected(); n > 0 {
		logger.S.Infof("Combat history cleanup removed %d PVE rows", n)
	}
}
