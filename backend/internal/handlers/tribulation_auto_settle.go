package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// AutoSettleCompletedTribulations reconciles every active tribulation event and
// settles it once no enemy is alive. It is the safety net for a lost death
// event, a restarted realtime zone or a death handler that failed after the
// final kill: the durable event must not stay ACTIVE forever.
func AutoSettleCompletedTribulations(ctx context.Context) error {
	if db.DB == nil {
		return nil
	}
	rows, err := db.DB.QueryContext(ctx, `
		SELECT id, player_id
		FROM tribulation_events
		WHERE status = 'ACTIVE'
	`)
	if err != nil {
		return fmt.Errorf("list active tribulations: %w", err)
	}
	type activeEvent struct {
		eventID  uuid.UUID
		playerID uuid.UUID
	}
	var events []activeEvent
	for rows.Next() {
		var ev activeEvent
		if err := rows.Scan(&ev.eventID, &ev.playerID); err != nil {
			rows.Close()
			return fmt.Errorf("scan active tribulation: %w", err)
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read active tribulations: %w", err)
	}
	rows.Close()

	for _, ev := range events {
		if err := autoSettleTribulationEventIfCleared(ctx, ev.eventID, ev.playerID); err != nil {
			logger.S.Errorf("Auto-settle tribulation %s for player %s failed: %v", ev.eventID, ev.playerID, err)
		}
	}
	return nil
}

func autoSettleTribulationEventIfCleared(ctx context.Context, eventID, playerID uuid.UUID) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin auto-settle check: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := reconcileTribulationEnemiesFromLiveState(ctx, tx, playerID, eventID); err != nil {
		return fmt.Errorf("reconcile live tribulation enemies: %w", err)
	}
	var alive int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM tribulation_enemies WHERE event_id = $1 AND is_dead = false
	`, eventID).Scan(&alive); err != nil {
		return fmt.Errorf("count living tribulation enemies: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit auto-settle check: %w", err)
	}
	if alive > 0 {
		return nil
	}

	finishHandoff, abortHandoff, _, handoffStatus, _, _ := beginTribulationHandoff(playerID)
	if handoffStatus != 0 {
		// The player is in the middle of another handoff. The next tick retries.
		return nil
	}
	defer abortHandoff()

	result, status, settleErr := settleRealtimeTribulationVictory(ctx, playerID, eventID, "auto-reconcile")
	if settleErr != nil {
		if status == http.StatusNotFound {
			finishHandoff()
			return nil
		}
		return fmt.Errorf("settle cleared tribulation (%d): %w", status, settleErr)
	}
	finishHandoff()
	PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated)
	notify.Send(db.DB, playerID, "tribulation_victory", "Vượt Kiếp Thành Công", result.Message)
	logger.S.Infof("Auto-settled cleared tribulation %s for %s: %s", eventID, playerID, result.Message)
	return nil
}

// StartTribulationAutoSettleWorker periodically closes events whose enemies are
// all dead. Five seconds keeps the manual "Hoàn Tất" button a fallback rather
// than a requirement.
func StartTribulationAutoSettleWorker(ctx context.Context) {
	worker.RunEvery(ctx, 5*time.Second, "Tribulation Auto Settle", func(ctx context.Context) {
		if err := AutoSettleCompletedTribulations(ctx); err != nil {
			logger.S.Errorf("Tribulation auto-settle worker error: %v", err)
		}
	})
}
