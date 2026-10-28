package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const tournamentReconnectGrace = 30 * time.Second

type realtimeTournamentSessionMetadata struct {
	Competition  string `json:"competition"`
	TournamentID string `json:"tournament_id"`
	MatchID      string `json:"match_id"`
	TeamAID      string `json:"team_a_id,omitempty"`
	TeamBID      string `json:"team_b_id,omitempty"`
	CombatantAID string `json:"combatant_a_id,omitempty"`
	CombatantBID string `json:"combatant_b_id,omitempty"`
}

type realtimeTournamentParticipantPayload struct {
	RealtimeCombatInstanceVitals
	Side         string `json:"side"`
	CompetitorID string `json:"competitor_id"`
}

type realtimeTournamentParticipant struct {
	PlayerID uuid.UUID
	Payload  realtimeTournamentParticipantPayload
}

func tournamentOriginZone(ctx context.Context, tx *sql.Tx, playerID uuid.UUID) (string, error) {
	var zone, region, zoneType, worldMap, instance string
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(p.world_zone_id,''),COALESCE(p.current_region,''),COALESCE(r.zone_type,''),
			COALESCE(rwm.world_map_id,''),COALESCE(p.world_instance_id,'')
		FROM players p LEFT JOIN regions r ON r.id=p.current_region
		LEFT JOIN region_world_maps rwm ON rwm.region_id=p.current_region WHERE p.id=$1
	`, playerID).Scan(&zone, &region, &zoneType, &worldMap, &instance); err != nil {
		return "", err
	}
	if instance != "" {
		return "", fmt.Errorf("player %s is already assigned to realtime instance %s", playerID, instance)
	}
	if zone == "" {
		if worldMap == "" {
			worldMap = "world_map"
		}
		zone = realtimeMapID(region, zoneType, worldMap)
	}
	if zone == "" {
		return "", fmt.Errorf("player %s has no realtime origin zone", playerID)
	}
	return zone, nil
}

func beginTournamentHandoffs(playerIDs []uuid.UUID) ([]uuid.UUID, error) {
	ordered := append([]uuid.UUID(nil), playerIDs...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].String() < ordered[j].String() })
	handedOff := make([]uuid.UUID, 0, len(ordered))
	for _, playerID := range ordered {
		handed, err := middleware.BeginRealtimeHandoff(playerID)
		if err != nil {
			for _, previous := range handedOff {
				middleware.AbortRealtimeHandoff(previous)
			}
			return nil, fmt.Errorf("checkpoint tournament combatant %s: %w", playerID, err)
		}
		if handed {
			handedOff = append(handedOff, playerID)
		}
	}
	return handedOff, nil
}

func finishTournamentHandoffs(playerIDs []uuid.UUID) {
	for _, playerID := range playerIDs {
		middleware.FinishRealtimeHandoff(playerID)
		PublishPlayerRealtimeEvent(playerID, PlayerEventUpdated)
	}
}

func abortTournamentHandoffs(playerIDs []uuid.UUID) {
	for _, playerID := range playerIDs {
		middleware.AbortRealtimeHandoff(playerID)
	}
}

func startRealtimeDeathArenaMatch(tournamentID, matchID, teamAID, teamBID string, teamA, teamB []*teamMemberCombatant) error {
	if reason := RealtimeRuntimeError(); reason != "" {
		return fmt.Errorf("realtime combat runtime unavailable: %s", reason)
	}
	players := make([]uuid.UUID, 0, len(teamA)+len(teamB))
	for _, member := range append(append([]*teamMemberCombatant(nil), teamA...), teamB...) {
		id, err := uuid.Parse(member.PlayerID)
		if err != nil {
			return fmt.Errorf("invalid death arena participant id %q: %w", member.PlayerID, err)
		}
		players = append(players, id)
	}
	handedOff, err := beginTournamentHandoffs(players)
	if err != nil {
		return err
	}
	defer abortTournamentHandoffs(handedOff)

	tx, err := db.DB.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin death arena realtime session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	instanceID := uuid.New()
	metadata := realtimeTournamentSessionMetadata{Competition: "death_arena", TournamentID: tournamentID, MatchID: matchID, TeamAID: teamAID, TeamBID: teamBID}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode death arena session metadata: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO realtime_combat_sessions (id,mode,zone_id,origin_zone_id,payload) VALUES ($1,'tournament',$2,$3,$4)`, instanceID, practiceRealtimeMapID, practiceRealtimeMapID, metaJSON); err != nil {
		return fmt.Errorf("create death arena realtime session: %w", err)
	}
	participants := make([]realtimeTournamentParticipant, 0, len(players))
	for side, team := range []struct {
		id      string
		members []*teamMemberCombatant
	}{{teamAID, teamA}, {teamBID, teamB}} {
		teamKey := "a"
		if side == 1 {
			teamKey = "b"
		}
		for i, member := range team.members {
			playerID, parseErr := uuid.Parse(member.PlayerID)
			if parseErr != nil {
				return fmt.Errorf("parse death arena participant id: %w", parseErr)
			}
			var locked uuid.UUID
			if err := tx.QueryRow(`SELECT id FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&locked); err != nil {
				return fmt.Errorf("lock death arena participant %s: %w", playerID, err)
			}
			origin, originErr := tournamentOriginZone(context.Background(), tx, playerID)
			if originErr != nil {
				return originErr
			}
			maxHP := member.MaxHP
			if maxHP <= 0 {
				maxHP = member.BaseHP
			}
			currentHP := member.CurrentHP
			if currentHP <= 0 || currentHP > maxHP {
				currentHP = maxHP
			}
			maxMana := member.MaxMana
			currentMana := member.CurrentMana
			if currentMana < 0 {
				currentMana = 0
			}
			if currentMana > maxMana {
				currentMana = maxMana
			}
			x, y := 17.0+float64(i%3)*2, 20.0+float64(i/3)*2
			if side == 1 {
				x = 47.0 - float64(i%3)*2
			}
			payload := realtimeTournamentParticipantPayload{
				RealtimeCombatInstanceVitals: RealtimeCombatInstanceVitals{CurrentHP: currentHP, CurrentMana: currentMana, X: x, Y: y, CombatTeamID: "tournament:" + instanceID.String() + ":" + teamKey, OriginZoneID: origin},
				Side:                         teamKey, CompetitorID: team.id,
			}
			payloadJSON, marshalErr := json.Marshal(payload)
			if marshalErr != nil {
				return fmt.Errorf("encode death arena participant: %w", marshalErr)
			}
			if _, err := tx.Exec(`INSERT INTO realtime_combat_session_participants (session_id,player_id,payload) VALUES ($1,$2,$3)`, instanceID, playerID, payloadJSON); err != nil {
				return fmt.Errorf("register death arena participant %s: %w", playerID, err)
			}
			if _, err := tx.Exec(`UPDATE players SET world_zone_id=$2,world_instance_id=$3,world_location_revision=world_location_revision+1 WHERE id=$1`, playerID, practiceRealtimeMapID, "combat:"+instanceID.String()); err != nil {
				return fmt.Errorf("route death arena participant %s: %w", playerID, err)
			}
			participants = append(participants, realtimeTournamentParticipant{PlayerID: playerID, Payload: payload})
		}
	}
	if _, err := tx.Exec(`UPDATE death_arena_matches SET status='IN_PROGRESS' WHERE id=$1 AND status='IN_PROGRESS'`, matchID); err != nil {
		return fmt.Errorf("confirm death arena match session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit death arena realtime session: %w", err)
	}
	finishTournamentHandoffs(players)
	return nil
}

func startRealtimeMartialArenaMatch(tournamentID, matchID, combatantAID, combatantBID string, memberA, memberB *teamMemberCombatant) error {
	if reason := RealtimeRuntimeError(); reason != "" {
		return fmt.Errorf("realtime combat runtime unavailable: %s", reason)
	}
	playerA, err := uuid.Parse(memberA.PlayerID)
	if err != nil {
		return fmt.Errorf("invalid martial arena participant A: %w", err)
	}
	playerB, err := uuid.Parse(memberB.PlayerID)
	if err != nil {
		return fmt.Errorf("invalid martial arena participant B: %w", err)
	}
	players := []uuid.UUID{playerA, playerB}
	handedOff, err := beginTournamentHandoffs(players)
	if err != nil {
		return err
	}
	defer abortTournamentHandoffs(handedOff)

	tx, err := db.DB.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin martial arena realtime session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	instanceID := uuid.New()
	metadata := realtimeTournamentSessionMetadata{Competition: "martial_arena", TournamentID: tournamentID, MatchID: matchID, CombatantAID: combatantAID, CombatantBID: combatantBID}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encode martial arena session metadata: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO realtime_combat_sessions (id,mode,zone_id,origin_zone_id,payload) VALUES ($1,'tournament',$2,$3,$4)`, instanceID, practiceRealtimeMapID, practiceRealtimeMapID, metaJSON); err != nil {
		return fmt.Errorf("create martial arena realtime session: %w", err)
	}
	for side, pair := range []struct {
		playerID    uuid.UUID
		combatantID string
		member      *teamMemberCombatant
		x           float64
	}{{playerA, combatantAID, memberA, 18}, {playerB, combatantBID, memberB, 46}} {
		var locked uuid.UUID
		if err := tx.QueryRow(`SELECT id FROM players WHERE id=$1 FOR UPDATE`, pair.playerID).Scan(&locked); err != nil {
			return fmt.Errorf("lock martial arena participant %s: %w", pair.playerID, err)
		}
		origin, originErr := tournamentOriginZone(context.Background(), tx, pair.playerID)
		if originErr != nil {
			return originErr
		}
		maxHP := pair.member.MaxHP
		if maxHP <= 0 {
			maxHP = pair.member.BaseHP
		}
		if maxHP <= 0 {
			return fmt.Errorf("martial arena participant %s has invalid max HP", pair.playerID)
		}
		maxMana := pair.member.MaxMana
		currentMana := maxMana
		if currentMana < 0 {
			currentMana = 0
		}
		teamKey := "a"
		if side == 1 {
			teamKey = "b"
		}
		payload := realtimeTournamentParticipantPayload{
			RealtimeCombatInstanceVitals: RealtimeCombatInstanceVitals{CurrentHP: maxHP, CurrentMana: currentMana, X: pair.x, Y: 24, CombatTeamID: "tournament:" + instanceID.String() + ":" + teamKey, OriginZoneID: origin},
			Side:                         teamKey, CompetitorID: pair.combatantID,
		}
		payloadJSON, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return fmt.Errorf("encode martial arena participant: %w", marshalErr)
		}
		if _, err := tx.Exec(`INSERT INTO realtime_combat_session_participants (session_id,player_id,payload) VALUES ($1,$2,$3)`, instanceID, pair.playerID, payloadJSON); err != nil {
			return fmt.Errorf("register martial arena participant %s: %w", pair.playerID, err)
		}
		if _, err := tx.Exec(`UPDATE players SET world_zone_id=$2,world_instance_id=$3,world_location_revision=world_location_revision+1 WHERE id=$1`, pair.playerID, practiceRealtimeMapID, "combat:"+instanceID.String()); err != nil {
			return fmt.Errorf("route martial arena participant %s: %w", pair.playerID, err)
		}
	}
	if _, err := tx.Exec(`UPDATE martial_matches SET status='IN_PROGRESS' WHERE id=$1 AND status='IN_PROGRESS'`, matchID); err != nil {
		return fmt.Errorf("confirm martial arena match session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit martial arena realtime session: %w", err)
	}
	finishTournamentHandoffs(players)
	return nil
}

func isTournamentSessionActive(ctx context.Context, tournamentID, matchID string) bool {
	var exists bool
	err := db.DB.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM realtime_combat_sessions
		WHERE mode='tournament' AND status='active' AND payload->>'tournament_id'=$1 AND payload->>'match_id'=$2)
	`, tournamentID, matchID).Scan(&exists)
	return err == nil && exists
}

func settleExpiredTournamentDisconnects(ctx context.Context, tournamentID, competition string) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT s.id,p.player_id,p.disconnected_at
		FROM realtime_combat_sessions s
		JOIN realtime_combat_session_participants p ON p.session_id=s.id AND p.status='active'
		WHERE s.mode='tournament' AND s.status='active' AND s.payload->>'tournament_id'=$1 AND s.payload->>'competition'=$2
	`, tournamentID, competition)
	if err != nil {
		loggerWarnf("list disconnected tournament participants: %v", err)
		return
	}
	type disconnectedParticipant struct {
		sessionID uuid.UUID
		playerID  uuid.UUID
		at        sql.NullTime
	}
	var participants []disconnectedParticipant
	for rows.Next() {
		var p disconnectedParticipant
		if err := rows.Scan(&p.sessionID, &p.playerID, &p.at); err == nil {
			participants = append(participants, p)
		}
	}
	rows.Close()
	for _, participant := range participants {
		if middleware.IsRealtimeWorldOwned(participant.playerID) {
			_, _ = db.DB.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET disconnected_at=NULL WHERE session_id=$1 AND player_id=$2 AND status='active'`, participant.sessionID, participant.playerID)
			continue
		}
		if !participant.at.Valid {
			_, _ = db.DB.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET disconnected_at=NOW() WHERE session_id=$1 AND player_id=$2 AND status='active' AND disconnected_at IS NULL`, participant.sessionID, participant.playerID)
			continue
		}
		if time.Since(participant.at.Time) >= tournamentReconnectGrace {
			if _, err := settleRealtimeTournamentLoss(ctx, participant.sessionID, participant.playerID, "DISCONNECT"); err != nil {
				loggerWarnf("settle disconnected tournament combatant %s: %v", participant.playerID, err)
			}
		}
	}
}

func loggerWarnf(format string, args ...any) {
	// Kept local so the tournament worker can report transient retry conditions.
	if logger.S != nil {
		logger.S.Warnf(format, args...)
	}
}

func validateTournamentParticipantPayload(raw []byte) (realtimeTournamentParticipantPayload, error) {
	var payload realtimeTournamentParticipantPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return payload, err
	}
	if payload.Side != "a" && payload.Side != "b" {
		return payload, errors.New("invalid tournament participant side")
	}
	return payload, nil
}
