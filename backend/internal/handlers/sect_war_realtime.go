package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const sectWarGuardianTemplateID = "sect_war_guardian"

type sectWarSessionMetadata struct {
	AttackerID         string  `json:"attacker_id"`
	AttackerName       string  `json:"attacker_name"`
	TargetSectID       string  `json:"target_sect_id"`
	TargetSectName     string  `json:"target_sect_name"`
	TargetRegion       string  `json:"target_region"`
	GuardianTemplateID string  `json:"guardian_template_id"`
	GuardianName       string  `json:"guardian_name"`
	GuardianRank       int     `json:"guardian_rank"`
	GuardianSubRank    int     `json:"guardian_sub_rank"`
	GuardianHP         float64 `json:"guardian_hp"`
	GuardianAttack     float64 `json:"guardian_attack"`
	GuardianDefense    float64 `json:"guardian_defense"`
	GuardianDead       bool    `json:"guardian_dead"`
}

type sectWarParticipantPayload struct {
	RealtimeCombatInstanceVitals
	Side string `json:"side"`
}

func startRealtimeSectWar(ctx context.Context, tx *sql.Tx, attackerID uuid.UUID, attackerName, targetSectID, targetSectName, targetRegion string, targetLevel int, arrayLevel int, defenderIDs []uuid.UUID) (uuid.UUID, []uuid.UUID, error) {
	if reason := RealtimeRuntimeError(); reason != "" {
		return uuid.Nil, nil, fmt.Errorf("realtime combat runtime unavailable: %s", reason)
	}
	if !middleware.IsRealtimeWorldOwned(attackerID) {
		return uuid.Nil, nil, errors.New("attacker must be in the realtime world")
	}
	playerIDs := []uuid.UUID{attackerID}
	// Party members participate only when online in the same region as the
	// attacker. Offline members remain outside this server-authoritative match.
	var partyID sql.NullString
	_ = tx.QueryRowContext(ctx, `SELECT party_id::text FROM players WHERE id=$1`, attackerID).Scan(&partyID)
	if partyID.Valid && partyID.String != "" {
		var raw []byte
		if err := tx.QueryRowContext(ctx, `SELECT members FROM parties WHERE id=$1`, partyID.String).Scan(&raw); err == nil {
			var members []PartyMember
			if json.Unmarshal(raw, &members) == nil {
				for _, member := range members {
					id, parseErr := uuid.Parse(member.ID)
					if parseErr == nil && id != attackerID && middleware.IsRealtimeWorldOwned(id) {
						playerIDs = append(playerIDs, id)
					}
				}
			}
		}
	}
	// Defenders are a live roster: an offline registered defender is not
	// teleported into a fight they cannot control.
	for _, id := range defenderIDs {
		if id != attackerID && middleware.IsRealtimeWorldOwned(id) {
			playerIDs = append(playerIDs, id)
		}
	}
	playerIDs = uniqueUUIDs(playerIDs)
	handedOff, err := beginTournamentHandoffs(playerIDs)
	if err != nil {
		return uuid.Nil, nil, err
	}
	committed := false
	defer func() {
		if !committed {
			abortTournamentHandoffs(handedOff)
		}
	}()

	instanceID := uuid.New()
	attackerTeam := "sectwar:" + instanceID.String() + ":attack"
	defenderTeam := "sectwar:" + instanceID.String() + ":defend"
	guardian := sectGuardianProfileFor(targetSectName, targetLevel, arrayLevel)
	metadata := sectWarSessionMetadata{
		AttackerID: attackerID.String(), AttackerName: attackerName,
		TargetSectID: targetSectID, TargetSectName: targetSectName, TargetRegion: targetRegion,
		GuardianTemplateID: sectWarGuardianTemplateID, GuardianName: guardian.Name,
		GuardianRank: guardian.Rank, GuardianSubRank: guardian.SubRank,
		GuardianHP: guardian.MaxHP, GuardianAttack: guardian.BaseAttack, GuardianDefense: guardian.BaseDefense,
	}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return uuid.Nil, nil, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO realtime_combat_sessions (id,mode,zone_id,origin_zone_id,payload) VALUES ($1,'sect_war',$2,$3,$4)`, instanceID, practiceRealtimeMapID, practiceRealtimeMapID, metaJSON); err != nil {
		return uuid.Nil, nil, fmt.Errorf("create sect war realtime session: %w", err)
	}
	defenderSet := make(map[uuid.UUID]bool, len(defenderIDs))
	for _, id := range defenderIDs {
		defenderSet[id] = true
	}
	attackerRegion := ""
	if err := tx.QueryRowContext(ctx, `SELECT current_region FROM players WHERE id=$1`, attackerID).Scan(&attackerRegion); err != nil {
		return uuid.Nil, nil, fmt.Errorf("load sect war attacker region: %w", err)
	}
	for i, id := range playerIDs {
		member, err := evaluateManaTxEquippedReadonly(tx, id)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("load sect war participant %s: %w", id, err)
		}
		if member.CurrentRegion != "" && member.CurrentRegion != attackerRegion {
			return uuid.Nil, nil, fmt.Errorf("sect war participant %s is not at the battle region", id)
		}
		origin, err := tournamentOriginZone(ctx, tx, id)
		if err != nil {
			return uuid.Nil, nil, err
		}
		maxHP := gamefx.EffectiveMaxHP(&member)
		if maxHP <= 0 {
			maxHP = member.BaseHP
		}
		if maxHP <= 0 {
			return uuid.Nil, nil, fmt.Errorf("sect war participant %s has invalid max HP", id)
		}
		hp := member.HP
		if hp <= 0 || hp > maxHP {
			hp = maxHP
		}
		maxMana := gamefx.EffectiveMaxMana(&member)
		mana := math.Max(0, math.Min(maxMana, member.CurrentMana))
		side := "attack"
		team := attackerTeam
		x := 18.0 + float64(i%3)*2
		if defenderSet[id] {
			side, team = "defend", defenderTeam
			x = 46.0 - float64(i%3)*2
		}
		payload := sectWarParticipantPayload{
			RealtimeCombatInstanceVitals: RealtimeCombatInstanceVitals{CurrentHP: hp, CurrentMana: mana, X: x, Y: 22.0 + float64(i/3)*2, CombatTeamID: team, OriginZoneID: origin},
			Side:                         side,
		}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return uuid.Nil, nil, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO realtime_combat_session_participants (session_id,player_id,payload) VALUES ($1,$2,$3)`, instanceID, id, payloadJSON); err != nil {
			return uuid.Nil, nil, fmt.Errorf("register sect war participant %s: %w", id, err)
		}
		result, err := tx.ExecContext(ctx, `UPDATE players SET world_zone_id=$2,world_instance_id=$3,world_location_revision=world_location_revision+1,human_qi=CASE WHEN id=$4 THEN LEAST(human_qi+5,$5) ELSE human_qi END WHERE id=$1`, id, practiceRealtimeMapID, "combat:"+instanceID.String(), attackerID, MaxHumanQi)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("route sect war participant %s: %w", id, err)
		}
		if changed, _ := result.RowsAffected(); changed != 1 {
			return uuid.Nil, nil, fmt.Errorf("sect war participant %s could not be routed", id)
		}
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, nil, fmt.Errorf("commit sect war realtime session: %w", err)
	}
	committed = true
	finishTournamentHandoffs(handedOff)
	for _, id := range playerIDs {
		if !containsUUID(handedOff, id) {
			PublishPlayerRealtimeEvent(id, PlayerEventUpdated)
		}
	}
	return instanceID, playerIDs, nil
}

func uniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(ids))
	out := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func containsUUID(ids []uuid.UUID, target uuid.UUID) bool {
	for _, id := range ids {
		if id == target {
			return true
		}
	}
	return false
}

// settleRealtimeSectWarDeath handles both player defeats and the guardian's
// death. It returns the participant IDs whose normal-world routes were restored.
func settleRealtimeSectWarDeath(ctx context.Context, tx *sql.Tx, sessionID uuid.UUID, rawMetadata []byte, death realtimeworld.DeathEvent) ([]uuid.UUID, error) {
	var metadata sectWarSessionMetadata
	if err := json.Unmarshal(rawMetadata, &metadata); err != nil {
		return nil, fmt.Errorf("decode sect war session: %w", err)
	}
	playerDeath := death.TargetKind == string(sim.EntityPlayer)
	if playerDeath {
		playerID, err := uuid.Parse(string(death.TargetID))
		if err != nil {
			return nil, nil
		}
		var raw []byte
		var status string
		if err := tx.QueryRowContext(ctx, `SELECT payload,status FROM realtime_combat_session_participants WHERE session_id=$1 AND player_id=$2 FOR UPDATE`, sessionID, playerID).Scan(&raw, &status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, nil
			}
			return nil, err
		}
		if status != "active" {
			return nil, nil
		}
		var payload sectWarParticipantPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, err
		}
		payload.CurrentHP = 0
		deadJSON, _ := json.Marshal(payload)
		if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET status='eliminated',payload=$3 WHERE session_id=$1 AND player_id=$2 AND status='active'`, sessionID, playerID, deadJSON); err != nil {
			return nil, err
		}
	} else if death.TemplateID == metadata.GuardianTemplateID || strings.HasPrefix(death.TemplateID, "sect_guardian_") {
		metadata.GuardianDead = true
		updated, _ := json.Marshal(metadata)
		if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_sessions SET payload=$2 WHERE id=$1 AND status='active'`, sessionID, updated); err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	var attackers, defenders int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE payload->>'side'='attack'),COUNT(*) FILTER (WHERE payload->>'side'='defend') FROM realtime_combat_session_participants WHERE session_id=$1 AND status='active'`, sessionID).Scan(&attackers, &defenders); err != nil {
		return nil, err
	}
	if playerDeath && attackers > 0 && (defenders > 0 || !metadata.GuardianDead) {
		playerID, _ := uuid.Parse(string(death.TargetID))
		ids, err := restoreSectWarParticipants(ctx, tx, sessionID, playerID)
		return ids, err
	}
	victory := attackers > 0 && defenders == 0 && metadata.GuardianDead
	defeat := attackers == 0
	if !victory && !defeat {
		return nil, nil
	}
	if victory {
		attackerID, err := uuid.Parse(metadata.AttackerID)
		if err != nil {
			return nil, err
		}
		attacker, err := evaluateManaTx(tx, attackerID)
		if err != nil {
			return nil, fmt.Errorf("load sect war attacker for settlement: %w", err)
		}
		combat := &sectWarSettlement{TargetSectID: metadata.TargetSectID}
		rows, err := tx.QueryContext(ctx, `SELECT player_id FROM realtime_combat_session_participants WHERE session_id=$1 AND status='active' AND payload->>'side'='attack' ORDER BY player_id`, sessionID)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err == nil && id != attackerID {
				combat.Participants = append(combat.Participants, sectWarParticipant{PlayerID: id})
			}
		}
		rows.Close()
		ApplySectWarVictory(tx, &attacker, combat)
	} else {
		attacker, err := evaluateManaTx(tx, uuid.MustParse(metadata.AttackerID))
		if err != nil {
			return nil, fmt.Errorf("load sect war attacker for defeat settlement: %w", err)
		}
		ApplySectWarDefeat(tx, &attacker, &sectWarSettlement{TargetSectID: metadata.TargetSectID})
	}
	if err := saveSectWarRealtimeHistory(ctx, tx, sessionID, metadata, victory); err != nil {
		return nil, fmt.Errorf("save sect war realtime history: %w", err)
	}
	ids, err := restoreSectWarParticipants(ctx, tx, sessionID, uuid.Nil)
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET status='completed' WHERE session_id=$1 AND status IN ('active','eliminated')`, sessionID); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE realtime_combat_sessions SET status='completed',ended_at=NOW() WHERE id=$1 AND status='active'`, sessionID); err != nil {
		return nil, err
	}
	return ids, nil
}

func saveSectWarRealtimeHistory(ctx context.Context, tx *sql.Tx, sessionID uuid.UUID, metadata sectWarSessionMetadata, attackerWon bool) error {
	winner := "Phe Địch"
	if attackerWon {
		winner = "Phe Ta"
	}
	logs, err := realtimeCombatHistoryJSON(ctx, tx, sessionID)
	if err != nil {
		return err
	}
	if string(logs) == "[]" {
		logs, _ = json.Marshal([]map[string]string{{
			"action": "SECT_WAR_REALTIME_RESULT",
			"detail": fmt.Sprintf("Chiến trận realtime tại Sơn Môn %s kết thúc. Kết quả phe tấn công: %s.", metadata.TargetSectName, winner),
		}})
	}
	summary, _ := json.Marshal(map[string]any{"sect_war_won": attackerWon, "realtime": true})
	combatantList := []map[string]any{{"id": metadata.GuardianTemplateID, "name": metadata.GuardianName, "kind": "monster", "side": "defend"}}
	rows, err := tx.QueryContext(ctx, `
		SELECT p.player_id::text,pl.username,p.payload->>'side'
		FROM realtime_combat_session_participants p JOIN players pl ON pl.id=p.player_id
		WHERE p.session_id=$1 AND p.status IN ('active','eliminated') ORDER BY p.player_id
	`, sessionID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id, name, side string
		if err := rows.Scan(&id, &name, &side); err != nil {
			rows.Close()
			return err
		}
		combatantList = append(combatantList, map[string]any{"id": id, "name": name, "kind": "player", "side": side})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	combatants, _ := json.Marshal(combatantList)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO combat_history (player_id,winner,logs,summary,combatants,enemy_name,enemy_type,region)
		SELECT p.player_id,
			CASE WHEN (p.payload->>'side'='attack')=$2 THEN 'Phe Ta' ELSE 'Phe Địch' END,
			$3::jsonb,$4::jsonb,$5::jsonb,
			CASE WHEN p.payload->>'side'='attack' THEN $6 ELSE $7 END,
			'SECT_WAR',$8
		FROM realtime_combat_session_participants p
		WHERE p.session_id=$1 AND p.status IN ('active','eliminated')
	`, sessionID, attackerWon, string(logs), string(summary), string(combatants), metadata.GuardianName, metadata.AttackerName, metadata.TargetRegion)
	return err
}

// SweepDisconnectedSectWars enforces the same 30 second reconnect grace used
// by virtual tournaments, so an abandoned sect war cannot hold the target
// sect's battle slot indefinitely.
func SweepDisconnectedSectWars(ctx context.Context) {
	rows, err := db.DB.QueryContext(ctx, `
		SELECT s.id,p.player_id,p.disconnected_at
		FROM realtime_combat_sessions s
		JOIN realtime_combat_session_participants p ON p.session_id=s.id AND p.status='active'
		WHERE s.mode='sect_war' AND s.status='active'
	`)
	if err != nil {
		return
	}
	type disconnected struct {
		sessionID, playerID uuid.UUID
		at                  sql.NullTime
	}
	var pending []disconnected
	for rows.Next() {
		var item disconnected
		if rows.Scan(&item.sessionID, &item.playerID, &item.at) == nil {
			pending = append(pending, item)
		}
	}
	rows.Close()
	for _, item := range pending {
		if middleware.IsRealtimeWorldOwned(item.playerID) {
			_, _ = db.DB.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET disconnected_at=NULL WHERE session_id=$1 AND player_id=$2 AND status='active'`, item.sessionID, item.playerID)
			continue
		}
		if !item.at.Valid {
			_, _ = db.DB.ExecContext(ctx, `UPDATE realtime_combat_session_participants SET disconnected_at=NOW() WHERE session_id=$1 AND player_id=$2 AND status='active' AND disconnected_at IS NULL`, item.sessionID, item.playerID)
			continue
		}
		if time.Since(item.at.Time) < tournamentReconnectGrace {
			continue
		}
		if err := settleDisconnectedSectWarPlayer(ctx, item.sessionID, item.playerID); err != nil {
			logger.S.Warnf("settle disconnected sect war participant %s: %v", item.playerID, err)
		}
	}
}

func settleDisconnectedSectWarPlayer(ctx context.Context, sessionID, playerID uuid.UUID) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var rawMetadata []byte
	if err := tx.QueryRowContext(ctx, `SELECT payload FROM realtime_combat_sessions WHERE id=$1 AND mode='sect_war' AND status='active' FOR UPDATE`, sessionID).Scan(&rawMetadata); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT player_id FROM realtime_combat_session_participants WHERE session_id=$1 AND status IN ('active','eliminated') ORDER BY player_id`, sessionID)
	if err != nil {
		return err
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	var handed []uuid.UUID
	for _, id := range ids {
		ok, handoffErr := middleware.BeginRealtimeHandoff(id)
		if handoffErr != nil {
			for _, prev := range handed {
				middleware.AbortRealtimeHandoff(prev)
			}
			return handoffErr
		}
		if ok {
			handed = append(handed, id)
		}
	}
	death := realtimeworld.DeathEvent{ZoneKey: sim.ZoneKey{InstanceID: sessionID.String()}, TargetID: sim.EntityID(playerID.String()), TargetKind: string(sim.EntityPlayer)}
	restored, err := settleRealtimeSectWarDeath(ctx, tx, sessionID, rawMetadata, death)
	if err != nil {
		for _, id := range handed {
			middleware.AbortRealtimeHandoff(id)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		for _, id := range handed {
			middleware.AbortRealtimeHandoff(id)
		}
		return err
	}
	restoredSet := make(map[uuid.UUID]struct{}, len(restored))
	for _, id := range restored {
		restoredSet[id] = struct{}{}
		PublishPlayerRealtimeEvent(id, PlayerEventUpdated)
	}
	for _, id := range handed {
		if _, ok := restoredSet[id]; ok {
			middleware.FinishRealtimeHandoff(id)
		} else {
			middleware.AbortRealtimeHandoff(id)
		}
	}
	return nil
}

func restoreSectWarParticipants(ctx context.Context, tx *sql.Tx, sessionID, only uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT player_id,payload,status FROM realtime_combat_session_participants WHERE session_id=$1 AND status IN ('active','eliminated')`
	args := []any{sessionID}
	if only != uuid.Nil {
		query += ` AND player_id=$2`
		args = append(args, only)
	}
	query += ` FOR UPDATE`
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	type entry struct {
		id     uuid.UUID
		state  sectWarParticipantPayload
		status string
	}
	var entries []entry
	for rows.Next() {
		var item entry
		var raw []byte
		if err := rows.Scan(&item.id, &raw, &item.status); err != nil {
			rows.Close()
			return nil, err
		}
		if err := json.Unmarshal(raw, &item.state); err != nil {
			rows.Close()
			return nil, err
		}
		entries = append(entries, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	ids := make([]uuid.UUID, 0, len(entries))
	for _, item := range entries {
		hp := item.state.CurrentHP
		if item.status == "eliminated" {
			hp = 1
		}
		if hp <= 0 {
			hp = 1
		}
		if _, err := tx.ExecContext(ctx, `UPDATE players SET hp=$1,current_mana=$2,world_zone_id=$3,world_instance_id='',world_location_revision=world_location_revision+1 WHERE id=$4`, hp, item.state.CurrentMana, item.state.OriginZoneID, item.id); err != nil {
			return nil, err
		}
		ids = append(ids, item.id)
	}
	return ids, nil
}
