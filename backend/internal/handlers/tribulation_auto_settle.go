package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// AutoSettleCompletedTribulations reconciles every active tribulation event and
// settles it once no enemy is alive.
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

	var alive int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM tribulation_enemies WHERE event_id = $1 AND is_dead = false
	`, eventID).Scan(&alive); err != nil {
		return fmt.Errorf("count living tribulation enemies: %w", err)
	}
	if alive > 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE tribulation_events SET status = 'RESOLVED', completed_at = NOW() WHERE id = $1 AND status = 'ACTIVE'
	`, eventID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated, PlayerDomain, TribulationDomain)
	return nil
}

// StartTribulationAutoSettleWorker periodically closes events whose enemies are all dead.
func StartTribulationAutoSettleWorker(ctx context.Context) {
	worker.RunEvery(ctx, 15*time.Second, "Tribulation Auto Settle", func(ctx context.Context) {
		if err := AutoSettleCompletedTribulations(ctx); err != nil {
			logger.S.Errorf("Tribulation auto-settle worker error: %v", err)
		}
	})
}
