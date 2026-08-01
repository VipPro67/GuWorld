package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartPoolWatchdog periodically logs the database connection pool state so
// pool exhaustion (a common cause of the whole server hanging) is visible.
// It also probes PostgreSQL pg_stat_activity for queries that have been
// running longer than slowQueryThreshold, which points at stuck locks.
func StartPoolWatchdog(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting DB Pool Watchdog with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("DB Pool Watchdog stopped")
				return
			case <-ticker.C:
				checkDBHealth(db)
			}
		}
	}()
}

func checkDBHealth(db *sql.DB) {
	stats := db.Stats()
	logger.S.Infof("DB Pool stats: open=%d inUse=%d idle=%d max=%d waitCount=%d waitDuration=%s",
		stats.OpenConnections, stats.InUse, stats.Idle, stats.MaxOpenConnections,
		stats.WaitCount, stats.WaitDuration)

	if stats.InUse >= stats.MaxOpenConnections {
		logger.S.Warnf("DB Pool exhausted: all %d connections in use", stats.MaxOpenConnections)
	}

	logSlowQueries(db)
}

const slowQueryThreshold = 5 * time.Second

// logSlowQueries lists long-running / blocked queries in pg_stat_activity.
func logSlowQueries(db *sql.DB) {
	rows, err := db.Query(`
		SELECT pid, state, wait_event_type, wait_event, now() - query_start AS age, left(query, 200)
		FROM pg_stat_activity
		WHERE state <> 'idle' AND query_start < now() - ($1::float8 || ' seconds')::interval
		ORDER BY age DESC`, slowQueryThreshold.Seconds())
	if err != nil {
		logger.S.Debugf("Pool Watchdog could not query pg_stat_activity: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var pid int
		var state, waitEventType, waitEvent, query string
		var age time.Duration
		if err := rows.Scan(&pid, &state, &waitEventType, &waitEvent, &age, &query); err != nil {
			continue
		}
		logger.S.Warnf("SLOW/STUCK DB query pid=%d state=%s waitEvent=%s/%s age=%s query=%q",
			pid, state, waitEventType, waitEvent, age, query)
	}
}
