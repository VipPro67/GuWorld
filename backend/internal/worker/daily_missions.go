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

	SafeRun("Daily Mission Rollover Worker", ctx, func(ctx context.Context) {
		rolloverDailyMissionCycle(ctx, db, dayKey())
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
	if previous != dayKey {
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
	for _, table := range []string{"player_daily_missions", "player_daily_mission_events", "player_daily_mission_rank_snapshots"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE day_key < $1::date`, dayKey); err != nil {
			logger.S.Errorf("Daily mission worker: clear expired %s: %v", table, err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Daily mission worker: commit rollover: %v", err)
	}
}
