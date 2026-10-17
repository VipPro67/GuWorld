package worker

import (
	"context"
	"database/sql"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func StartDailyMissionRolloverWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		logger.S.Errorf("Daily mission worker: load timezone: %v", err)
		return
	}
	dayKey := func() string { return time.Now().In(loc).Format("2006-01-02") }

	go SafeRun("Daily Mission Rollover Worker", ctx, func(ctx context.Context) {
		key := dayKey()
		rolloverDailyMissionCycle(ctx, db, key)
		cleanExpiredDailyMissionData(ctx, db, key)
	})
	RunEvery(ctx, interval, "Daily Mission Rollover Worker", func(ctx context.Context) {
		rolloverDailyMissionCycle(ctx, db, dayKey())
	})
}

func rolloverDailyMissionCycle(ctx context.Context, db *sql.DB, dayKey string) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.S.Errorf("Daily mission worker: begin rollover: %v", err)
		return
	}
	defer tx.Rollback()
	var previous string
	err = tx.QueryRowContext(ctx, `SELECT day_key::text FROM daily_mission_cycle_state WHERE id=TRUE FOR UPDATE`).Scan(&previous)
	if err == sql.ErrNoRows {
		_, err = tx.ExecContext(ctx, `INSERT INTO daily_mission_cycle_state(id,day_key) VALUES(TRUE,$1::date)`, dayKey)
		previous = ""
	} else if err != nil {
		logger.S.Errorf("Daily mission worker: read cycle state: %v", err)
		return
	}
	if err != nil {
		logger.S.Errorf("Daily mission worker: initialize cycle state: %v", err)
		return
	}
	dayChanged := previous != dayKey
	if dayChanged {
		if _, err := tx.ExecContext(ctx, `INSERT INTO player_daily_mission_rank_snapshots(player_id,day_key,rank_snapshot)
			SELECT id,$1::date,rank FROM players ON CONFLICT DO NOTHING`, dayKey); err != nil {
			logger.S.Errorf("Daily mission worker: snapshot ranks: %v", err)
			return
		}
		if _, err := tx.ExecContext(ctx, `UPDATE daily_mission_cycle_state SET day_key=$1::date WHERE id=TRUE`, dayKey); err != nil {
			logger.S.Errorf("Daily mission worker: update cycle state: %v", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Daily mission worker: commit rollover: %v", err)
		return
	}

	if dayChanged {
		cleanExpiredDailyMissionData(ctx, db, dayKey)
	}
}

func cleanExpiredDailyMissionData(ctx context.Context, db *sql.DB, dayKey string) {
	for _, table := range []string{"player_daily_missions", "player_daily_mission_rank_snapshots"} {
		if _, err := db.ExecContext(ctx, `DELETE FROM `+table+` WHERE day_key < $1::date`, dayKey); err != nil {
			logger.S.Errorf("Daily mission worker: clear expired %s: %v", table, err)
		}
	}

	// For player_daily_mission_events, delete in batches of 5000 rows to prevent
	// statement timeouts, transaction lock escalation, and huge WAL spikes.
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res, err := db.ExecContext(ctx, `
			DELETE FROM player_daily_mission_events
			WHERE ctid IN (
				SELECT ctid FROM player_daily_mission_events
				WHERE day_key < $1::date
				LIMIT 5000
			)
		`, dayKey)
		if err != nil {
			logger.S.Errorf("Daily mission worker: clear expired player_daily_mission_events: %v", err)
			return
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil || rowsAffected == 0 {
			break
		}
		if rowsAffected < 5000 {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
}
