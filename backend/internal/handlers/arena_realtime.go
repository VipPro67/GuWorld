package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
)

// SweepDisconnectedArenas gives arena participants the same 30-second reconnect
// grace as realtime tournaments before recording a forfeit.
func SweepDisconnectedArenas(ctx context.Context) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT s.id,p.player_id,p.disconnected_at
		FROM realtime_combat_sessions s
		JOIN realtime_combat_session_participants p ON p.session_id=s.id AND p.status='active'
		WHERE s.mode='arena' AND s.status='active'
	`)
	if err != nil {
		loggerWarnf("list disconnected arena participants: %v", err)
		return
	}
	type candidate struct {
		sessionID      uuid.UUID
		playerID       uuid.UUID
		disconnectedAt sql.NullTime
	}
	var pending []candidate
	for rows.Next() {
		var item candidate
		if rows.Scan(&item.sessionID, &item.playerID, &item.disconnectedAt) == nil {
			pending = append(pending, item)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		loggerWarnf("read disconnected arena participants: %v", err)
		return
	}
	rows.Close()
	for _, item := range pending {
		if middleware.IsRealtimeWorldOwned(item.playerID) {
			_, _ = db.DB.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET disconnected_at=NULL WHERE session_id=$1 AND player_id=$2 AND status='active'`, item.sessionID, item.playerID)
			continue
		}
		if !item.disconnectedAt.Valid {
			_, _ = db.DB.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET disconnected_at=NOW() WHERE session_id=$1 AND player_id=$2 AND status='active' AND disconnected_at IS NULL`, item.sessionID, item.playerID)
			continue
		}
		if time.Since(item.disconnectedAt.Time) < tournamentReconnectGrace {
			continue
		}
		if err := settleDisconnectedArena(ctx, item.sessionID, item.playerID, item.disconnectedAt.Time); err != nil {
			loggerWarnf("settle disconnected arena participant %s: %v", item.playerID, err)
		}
	}
}

func settleDisconnectedArena(ctx context.Context, sessionID, disconnectedID uuid.UUID, disconnectedAt time.Time) error {
	var metadataRaw []byte
	var otherID uuid.UUID
	err := db.DB.QueryRowContext(ctx, `
		SELECT s.payload,other.player_id
		FROM realtime_combat_sessions s
		JOIN realtime_combat_session_participants gone ON gone.session_id=s.id AND gone.player_id=$2 AND gone.status='active'
		JOIN realtime_combat_session_participants other ON other.session_id=s.id AND other.player_id<>gone.player_id AND other.status='active'
		WHERE s.id=$1 AND s.mode='arena' AND s.status='active'
		AND gone.disconnected_at=$3 AND gone.disconnected_at<=NOW()-$4::interval
	`, sessionID, disconnectedID, disconnectedAt, tournamentReconnectGrace.String()).Scan(&metadataRaw, &otherID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load expired arena disconnect: %w", err)
	}
	if middleware.IsRealtimeWorldOwned(disconnectedID) {
		_, _ = db.DB.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET disconnected_at=NULL WHERE session_id=$1 AND player_id=$2 AND status='active'`, sessionID, disconnectedID)
		return nil
	}

	// Lock both character handoffs while restoring their persistent locations.
	handoffs, err := beginTournamentHandoffs([]uuid.UUID{disconnectedID, otherID})
	if err != nil {
		return err
	}
	if middleware.IsRealtimeWorldOwned(disconnectedID) {
		abortTournamentHandoffs(handoffs)
		_, _ = db.DB.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET disconnected_at=NULL WHERE session_id=$1 AND player_id=$2 AND status='active'`, sessionID, disconnectedID)
		return nil
	}
	committed := false
	defer func() {
		if !committed {
			abortTournamentHandoffs(handoffs)
		}
	}()

	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin arena forfeit settlement: %w", err)
	}
	defer tx.Rollback()
	var payload []byte
	if err := tx.QueryRowContext(ctx, `SELECT payload FROM realtime_combat_sessions WHERE id=$1 AND mode='arena' AND status='active' FOR UPDATE`, sessionID).Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("lock arena session: %w", err)
	}
	var currentDisconnected sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT disconnected_at FROM realtime_combat_session_participants WHERE session_id=$1 AND player_id=$2 AND status='active' FOR UPDATE`, sessionID, disconnectedID).Scan(&currentDisconnected); err != nil {
		return fmt.Errorf("lock disconnected arena participant: %w", err)
	}
	if !currentDisconnected.Valid || !currentDisconnected.Time.Equal(disconnectedAt) || time.Since(currentDisconnected.Time) < tournamentReconnectGrace {
		return nil
	}
	death := realtimeworld.DeathEvent{
		ZoneKey:  sim.ZoneKey{InstanceID: sessionID.String()},
		SourceID: sim.EntityID(otherID.String()), TargetID: sim.EntityID(disconnectedID.String()),
		TargetKind: string(sim.EntityPlayer),
	}
	winnerID, loserID, handled, err := finalizeRealtimeArenaDeath(ctx, tx, sessionID, payload, death)
	if err != nil {
		return fmt.Errorf("finalize arena forfeit: %w", err)
	}
	if !handled {
		return nil
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit arena forfeit: %w", err)
	}
	committed = true
	finishTournamentHandoffs(handoffs)
	for _, id := range []uuid.UUID{winnerID, loserID} {
		PublishPlayerRealtimeEvent(id, PlayerEventUpdated)
	}
	return nil
}
