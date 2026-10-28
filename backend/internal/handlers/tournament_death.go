package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
)

// settleRealtimeTournamentLoss handles a combatant who remained disconnected
// beyond the tournament reconnect grace period.
func settleRealtimeTournamentLoss(ctx context.Context, sessionID, playerID uuid.UUID, reason string) (bool, error) {
	var rawMetadata []byte
	err := db.DB.QueryRowContext(ctx, `SELECT payload FROM realtime_combat_sessions WHERE id=$1 AND mode='tournament' AND status='active'`, sessionID).Scan(&rawMetadata)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	death := realtimeworld.DeathEvent{ZoneKey: sim.ZoneKey{InstanceID: sessionID.String()}, TargetID: sim.EntityID(playerID.String()), TargetKind: string(sim.EntityPlayer)}
	ids, err := settleRealtimeTournamentDeath(ctx, tx, sessionID, rawMetadata, death)
	if err != nil {
		return true, err
	}
	if len(ids) == 0 {
		return false, nil
	}
	var handoffs []uuid.UUID
	for _, id := range ids {
		handed, handoffErr := middleware.BeginRealtimeHandoff(id)
		if handoffErr != nil {
			for _, previous := range handoffs {
				middleware.AbortRealtimeHandoff(previous)
			}
			return true, fmt.Errorf("checkpoint disconnected tournament participant %s: %w", id, handoffErr)
		}
		if handed {
			handoffs = append(handoffs, id)
		}
	}
	if err := tx.Commit(); err != nil {
		for _, id := range handoffs {
			middleware.AbortRealtimeHandoff(id)
		}
		return true, err
	}
	for _, id := range handoffs {
		middleware.FinishRealtimeHandoff(id)
	}
	for _, id := range ids {
		PublishPlayerRealtimeEvent(id, PlayerEventUpdated)
	}
	_ = reason // recorded by the tournament status as an elimination, without combat log text.
	return true, nil
}

// settleRealtimeTournamentDeath removes a defeated combatant from a virtual
// tournament map. It returns all players whose world routing was restored.
func settleRealtimeTournamentDeath(ctx context.Context, tx *sql.Tx, sessionID uuid.UUID, rawMetadata []byte, death realtimeworld.DeathEvent) ([]uuid.UUID, error) {
	playerID, err := uuid.Parse(string(death.TargetID))
	if err != nil || death.TargetKind != string(sim.EntityPlayer) {
		return nil, nil
	}
	var metadata realtimeTournamentSessionMetadata
	if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
		return nil, fmt.Errorf("decode tournament session metadata: %w", err)
	}
	if metadata.Competition != "death_arena" && metadata.Competition != "martial_arena" {
		return nil, nil
	}
	var rawPayload []byte
	var participantStatus string
	err = tx.QueryRowContext(ctx, `SELECT payload,status FROM realtime_combat_session_participants WHERE session_id=$1 AND player_id=$2 FOR UPDATE`, sessionID, playerID).Scan(&rawPayload, &participantStatus)
	if err == sql.ErrNoRows || participantStatus != "active" {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load tournament participant: %w", err)
	}
	var participant realtimeTournamentParticipantPayload
	if err := json.Unmarshal(rawPayload, &participant); err != nil {
		return nil, fmt.Errorf("decode tournament participant state: %w", err)
	}
	participant.CurrentHP = 0
	if participant.CurrentMana < 0 {
		participant.CurrentMana = 0
	}
	deadPayload, _ := json.Marshal(participant)
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET status='eliminated',payload=$3 WHERE session_id=$1 AND player_id=$2 AND status='active'`, sessionID, playerID, deadPayload); err != nil {
		return nil, fmt.Errorf("eliminate tournament participant: %w", err)
	}
	if metadata.Competition == "death_arena" {
		teamID := metadata.TeamAID
		if participant.Side == "b" {
			teamID = metadata.TeamBID
		}
		if _, err := tx.ExecContext(ctx, `UPDATE death_arena_team_members SET current_hp=0,current_mana=$1,eliminated=true,eliminated_at_round=(SELECT current_round FROM death_arena_tournaments WHERE id=$2) WHERE team_id=$3 AND player_id=$4`, participant.CurrentMana, metadata.TournamentID, teamID, playerID); err != nil {
			return nil, fmt.Errorf("eliminate death arena member: %w", err)
		}
	} else {
		combatantID := metadata.CombatantAID
		if participant.Side == "b" {
			combatantID = metadata.CombatantBID
		}
		if _, err := tx.ExecContext(ctx, `UPDATE martial_combatants SET eliminated=true,eliminated_at_round=(SELECT current_round FROM martial_tournaments WHERE id=$1) WHERE id=$2`, metadata.TournamentID, combatantID); err != nil {
			return nil, fmt.Errorf("eliminate martial arena combatant: %w", err)
		}
	}

	var activeA, activeB int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE payload->>'side'='a'),COUNT(*) FILTER (WHERE payload->>'side'='b') FROM realtime_combat_session_participants WHERE session_id=$1 AND status='active'`, sessionID).Scan(&activeA, &activeB); err != nil {
		return nil, fmt.Errorf("count tournament survivors: %w", err)
	}
	if activeA > 0 && activeB > 0 {
		return restoreTournamentPlayers(ctx, tx, sessionID, playerID)
	}
	winnerSide := "a"
	if activeA == 0 && activeB > 0 {
		winnerSide = "b"
	}
	winnerID, loserID := metadata.TeamAID, metadata.TeamBID
	if metadata.Competition == "martial_arena" {
		winnerID, loserID = metadata.CombatantAID, metadata.CombatantBID
	}
	if winnerSide == "b" {
		winnerID, loserID = loserID, winnerID
	}
	combatLog, err := realtimeCombatHistoryJSON(ctx, tx, sessionID)
	if err != nil {
		return nil, err
	}
	if metadata.Competition == "death_arena" {
		// Persist the latest realtime checkpoints before the bracket advances;
		// surviving members carry their remaining HP and mana into the next round.
		if _, err := tx.ExecContext(ctx, `UPDATE death_arena_team_members m
			SET current_hp=CASE WHEN p.status='active' THEN COALESCE((p.payload->>'current_hp')::double precision,m.current_hp) ELSE 0 END,
				current_mana=COALESCE((p.payload->>'current_mana')::double precision,m.current_mana),
				eliminated=(p.status<>'active'),
				eliminated_at_round=CASE WHEN p.status='active' THEN m.eliminated_at_round ELSE (SELECT current_round FROM death_arena_tournaments WHERE id=$1) END
			FROM realtime_combat_session_participants p
			WHERE p.session_id=$2 AND p.player_id=m.player_id AND m.team_id IN ($3,$4)`, metadata.TournamentID, sessionID, metadata.TeamAID, metadata.TeamBID); err != nil {
			return nil, fmt.Errorf("persist death arena realtime checkpoints: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE death_arena_matches SET status='FINISHED',winner_team_id=$1,combat_log=$2,finished_at=NOW() WHERE id=$3 AND status='IN_PROGRESS'`, winnerID, combatLog, metadata.MatchID); err != nil {
			return nil, fmt.Errorf("finish death arena match: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE death_arena_teams SET eliminated=true,eliminated_at_round=(SELECT current_round FROM death_arena_tournaments WHERE id=$1) WHERE id=$2`, metadata.TournamentID, loserID); err != nil {
			return nil, fmt.Errorf("eliminate death arena team: %w", err)
		}
		var reward int
		_ = tx.QueryRowContext(ctx, `SELECT reward_stones_per_member FROM death_arena_matches WHERE id=$1`, metadata.MatchID).Scan(&reward)
		if reward > 0 {
			_, _ = tx.ExecContext(ctx, `UPDATE players SET primeval_stones=primeval_stones+$1 WHERE id IN (SELECT player_id FROM death_arena_team_members WHERE team_id=$2 AND eliminated=false)`, reward, winnerID)
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE martial_matches SET status='FINISHED',winner_combatant_id=$1,combat_log=$2,finished_at=NOW() WHERE id=$3 AND status='IN_PROGRESS'`, winnerID, combatLog, metadata.MatchID); err != nil {
			return nil, fmt.Errorf("finish martial arena match: %w", err)
		}
	}
	ids, err := restoreTournamentPlayers(ctx, tx, sessionID, uuid.Nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET status='completed' WHERE session_id=$1 AND status IN ('active','eliminated')`, sessionID); err != nil {
		return nil, fmt.Errorf("complete tournament participants: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_sessions SET status='completed',ended_at=NOW() WHERE id=$1 AND status='active'`, sessionID); err != nil {
		return nil, fmt.Errorf("complete tournament session: %w", err)
	}
	return ids, nil
}

// restoreTournamentPlayers writes each persisted checkpoint back to the
// player's normal zone. uuid.Nil means restore every session participant.
func restoreTournamentPlayers(ctx context.Context, tx *sql.Tx, sessionID, only uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT player_id,payload FROM realtime_combat_session_participants WHERE session_id=$1 AND status IN ('active','eliminated')`
	args := []any{sessionID}
	if only != uuid.Nil {
		query += ` AND player_id=$2`
		args = append(args, only)
	}
	query += ` FOR UPDATE`
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load tournament route checkpoints: %w", err)
	}
	type checkpoint struct {
		id    uuid.UUID
		state realtimeTournamentParticipantPayload
	}
	var checkpoints []checkpoint
	for rows.Next() {
		var c checkpoint
		var raw []byte
		if err := rows.Scan(&c.id, &raw); err != nil {
			rows.Close()
			return nil, err
		}
		if err := json.Unmarshal(raw, &c.state); err != nil {
			rows.Close()
			return nil, err
		}
		checkpoints = append(checkpoints, c)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	ids := make([]uuid.UUID, 0, len(checkpoints))
	for _, c := range checkpoints {
		hp := c.state.CurrentHP
		if hp <= 0 {
			hp = 1
		}
		_, err := tx.ExecContext(ctx, `UPDATE players SET hp=$1,current_mana=$2,world_zone_id=$3,world_instance_id='',world_location_revision=world_location_revision+1 WHERE id=$4`, hp, c.state.CurrentMana, c.state.OriginZoneID, c.id)
		if err != nil {
			return nil, fmt.Errorf("restore tournament player %s: %w", c.id, err)
		}
		ids = append(ids, c.id)
	}
	return ids, nil
}
