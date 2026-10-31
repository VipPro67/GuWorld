package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
)

type realtimeCombatHistoryEvent struct {
	EventID        uint64  `json:"event_id"`
	Turn           uint64  `json:"turn"`
	ActorID        string  `json:"actor_id"`
	ActorName      string  `json:"actor_name"`
	Action         string  `json:"action"`
	TargetID       string  `json:"target_id,omitempty"`
	TargetName     string  `json:"target_name,omitempty"`
	Damage         float64 `json:"damage,omitempty"`
	ShieldAbsorbed float64 `json:"shield_absorbed,omitempty"`
	HPDamage       float64 `json:"hp_damage,omitempty"`
	Detail         string  `json:"detail,omitempty"`
}

// PersistRealtimeCombatEvents stores authoritative events for private matches.
// Open-world snapshots are intentionally not retained as match logs.
func PersistRealtimeCombatEvents(ctx context.Context, snapshot sim.Snapshot) error {
	if len(snapshot.Events) == 0 || snapshot.ZoneKey.InstanceID == "" {
		return nil
	}
	sessionID, err := uuid.Parse(snapshot.ZoneKey.InstanceID)
	if err != nil {
		return nil
	}
	names := make(map[string]string, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		if entity.ID != "" {
			names[string(entity.ID)] = entity.Name
		}
	}
	entries := make([]realtimeCombatHistoryEvent, 0, len(snapshot.Events))
	for _, event := range snapshot.Events {
		actorName := names[event.SourceID]
		if actorName == "" {
			actorName = event.SourceID
		}
		targetName := names[event.TargetID]
		if targetName == "" && event.TemplateID != "" {
			targetName = event.TemplateID
		}
		detail := event.Reason
		if event.SkillID != "" {
			if detail != "" {
				detail = event.SkillID + ": " + detail
			} else {
				detail = event.SkillID
			}
		}
		entries = append(entries, realtimeCombatHistoryEvent{
			EventID: event.ID, Turn: event.Tick, ActorID: event.SourceID, ActorName: actorName,
			Action: string(event.Type), TargetID: event.TargetID, TargetName: targetName,
			Damage: event.Amount, ShieldAbsorbed: event.ShieldAbsorbed, HPDamage: event.HPDamage, Detail: detail,
		})
	}
	raw, err := json.Marshal(entries)
	if err != nil {
		return fmt.Errorf("encode realtime combat history events: %w", err)
	}
	states := make([]struct {
		PlayerID    string  `json:"player_id"`
		CurrentHP   float64 `json:"current_hp"`
		CurrentMana float64 `json:"current_mana"`
		X           float64 `json:"x"`
		Y           float64 `json:"y"`
	}, 0)
	for _, entity := range snapshot.Entities {
		if entity.Kind != sim.EntityPlayer || entity.CharacterID == "" {
			continue
		}
		states = append(states, struct {
			PlayerID    string  `json:"player_id"`
			CurrentHP   float64 `json:"current_hp"`
			CurrentMana float64 `json:"current_mana"`
			X           float64 `json:"x"`
			Y           float64 `json:"y"`
		}{string(entity.CharacterID), entity.Vitals.HP, entity.Combat.Mana, entity.Transform.Position.X, entity.Transform.Position.Y})
	}
	stateJSON, err := json.Marshal(states)
	if err != nil {
		return fmt.Errorf("encode realtime combat state checkpoint: %w", err)
	}
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin realtime combat event checkpoint: %w", err)
	}
	defer tx.Rollback()
	var mode string
	if err := tx.QueryRowContext(ctx, `SELECT mode FROM realtime_combat_sessions WHERE id=$1 AND status='active' FOR UPDATE`, sessionID).Scan(&mode); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("load realtime combat event session: %w", err)
	}
	if mode != "arena" && mode != "tournament" && mode != "sect_war" {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE realtime_combat_session_participants p
		SET payload=p.payload || jsonb_build_object('current_hp',s.current_hp,'current_mana',s.current_mana,'x',s.x,'y',s.y)
		FROM jsonb_to_recordset($2::jsonb) AS s(player_id uuid,current_hp double precision,current_mana double precision,x double precision,y double precision)
		WHERE p.session_id=$1 AND p.player_id=s.player_id AND p.status='active'
	`, sessionID, stateJSON); err != nil {
		return fmt.Errorf("checkpoint realtime combat participants: %w", err)
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO realtime_combat_session_events (session_id,event_id,payload)
		SELECT $1,(event->>'event_id')::bigint,event
		FROM jsonb_array_elements($2::jsonb) AS events(event)
		JOIN realtime_combat_sessions s ON s.id=$1 AND s.status='active'
		WHERE s.mode IN ('arena','tournament','sect_war')
		  AND (SELECT COUNT(*) FROM realtime_combat_session_events WHERE session_id=$1) < 10000
	`, sessionID, raw)
	if err != nil {
		return fmt.Errorf("persist realtime combat history events: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit realtime combat event checkpoint: %w", err)
	}
	return nil
}

func realtimeCombatHistoryJSON(ctx context.Context, tx *sql.Tx, sessionID uuid.UUID) ([]byte, error) {
	var logs []byte
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(jsonb_agg(payload ORDER BY event_id,id),'[]'::jsonb)
		FROM realtime_combat_session_events WHERE session_id=$1
	`, sessionID).Scan(&logs); err != nil {
		return nil, fmt.Errorf("load realtime combat history events: %w", err)
	}
	if len(logs) == 0 {
		return []byte("[]"), nil
	}
	return logs, nil
}

// RecordRealtimeMonsterKill adds a compact durable history row for ordinary
// open-world realtime PvE, whose combat events are not a private session.
func RecordRealtimeMonsterKill(ctx context.Context, playerID uuid.UUID, death realtimeworld.DeathEvent, monsterName string) error {
	var playerName string
	if err := db.DB.QueryRowContext(ctx, `SELECT username FROM players WHERE id=$1`, playerID).Scan(&playerName); err != nil {
		return fmt.Errorf("load realtime combat history player: %w", err)
	}
	if monsterName == "" {
		monsterName = death.TemplateID
	}
	logs, _ := json.Marshal([]map[string]any{{
		"turn": death.Tick, "actor_id": playerID.String(), "actor_name": playerName,
		"action": "MONSTER_DEFEATED", "target_id": string(death.TargetID),
		"target_name": monsterName, "detail": "Hạ gục quái trên bản đồ realtime.",
	}})
	combatants, _ := json.Marshal([]map[string]any{
		{"id": playerID.String(), "name": playerName, "kind": "ally"},
		{"id": string(death.TargetID), "name": monsterName, "kind": "enemy"},
	})
	deathID := fmt.Sprintf("%s:%d:%s", death.Generation, death.EventID, death.TargetID)
	summary, _ := json.Marshal(map[string]any{"realtime": true, "monster_defeated": death.TemplateID, "tick": death.Tick, "death_id": deathID})
	if _, err := db.DB.ExecContext(ctx, `
		INSERT INTO combat_history (player_id,winner,logs,summary,combatants,enemy_name,enemy_type,region)
		VALUES ($1,'Phe Ta',$2,$3,$4,$5,'PVE',$6)
		ON CONFLICT (player_id,(summary->>'death_id')) WHERE summary ? 'death_id' DO NOTHING
	`, playerID, logs, summary, combatants, monsterName, death.ZoneKey.MapID); err != nil {
		return fmt.Errorf("persist realtime monster combat history: %w", err)
	}
	return nil
}

func recordRealtimePlayerCombatHistory(ctx context.Context, tx *sql.Tx, winnerID, loserID uuid.UUID, winnerName, loserName, region string, tick uint64) error {
	logs, _ := json.Marshal([]map[string]any{{
		"turn": tick, "actor_id": winnerID.String(), "actor_name": winnerName,
		"action": "PLAYER_DEFEATED", "target_id": loserID.String(), "target_name": loserName,
		"detail": "Trận PvP realtime kết thúc khi một Cổ Sư bị hạ gục.",
	}})
	winnerCombatants, _ := json.Marshal([]map[string]any{
		{"id": winnerID.String(), "name": winnerName, "kind": "ally"},
		{"id": loserID.String(), "name": loserName, "kind": "enemy"},
	})
	loserCombatants, _ := json.Marshal([]map[string]any{
		{"id": loserID.String(), "name": loserName, "kind": "ally"},
		{"id": winnerID.String(), "name": winnerName, "kind": "enemy"},
	})
	summary, _ := json.Marshal(map[string]any{"realtime": true, "winner_id": winnerID.String(), "loser_id": loserID.String(), "tick": tick})
	for _, row := range []struct {
		id            uuid.UUID
		winner, enemy string
		combatants    []byte
	}{
		{winnerID, "Phe Ta", loserName, winnerCombatants},
		{loserID, "Phe Địch", winnerName, loserCombatants},
	} {
		if _, err := tx.ExecContext(ctx, `INSERT INTO combat_history (player_id,winner,logs,summary,combatants,enemy_name,enemy_type,region) VALUES ($1,$2,$3,$4,$5,$6,'PVP',$7)`, row.id, row.winner, logs, summary, row.combatants, row.enemy, region); err != nil {
			return fmt.Errorf("persist realtime PvP combat history for %s: %w", row.id, err)
		}
	}
	return nil
}
