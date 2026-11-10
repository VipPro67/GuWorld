package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func combatOwnedByPlayer(state *models.CombatState, playerID string) bool {
	return state == nil || state.CombatOwnerID == "" || state.CombatOwnerID == playerID
}

func completeWorldMonsterCombat(ctx context.Context, state *models.CombatState, winner, playerID string) {
	clearWorldMonsterPartyCombat(state, playerID)
}

func clearWorldMonsterPartyCombat(state *models.CombatState, playerID string) {
	if state == nil || state.WorldMonsterEncounterID == "" || db.DB == nil {
		return
	}
	for _, ally := range state.Allies {
		if (ally.Kind != "Ally" && ally.Kind != "Player") || ally.ID == playerID {
			continue
		}
		allyID, err := uuid.Parse(ally.ID)
		if err != nil {
			continue
		}
		result, err := db.DB.Exec(`UPDATE players SET active_combat=NULL,version=version+1,state_revision=COALESCE(state_revision,0)+1
			WHERE id=$1 AND (active_combat IS NULL OR
				(active_combat->>'combat_owner_id'=$2 AND active_combat->>'world_monster_encounter_id'=$3))`,
			allyID, playerID, state.WorldMonsterEncounterID)
		if err != nil {
			logger.S.Errorf("clear world monster combat lock for ally=%s: %v", ally.ID, err)
			continue
		}
		if rows, _ := result.RowsAffected(); rows > 0 {
			ClearPendingPlayerQueues(ally.ID)
			PublishPlayerCombatDelta(allyID, CombatEventEnded, nil)
		}
	}
}

func savePlayerMovement(tx *sql.Tx, player *models.Player, combat any) error {
	var active any
	switch value := combat.(type) {
	case nil:
		active = nil
	case string:
		active = value
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode active combat: %w", err)
		}
		active = encoded
	}
	_, err := tx.Exec(`UPDATE players SET current_mana=$1,x=$2,y=$3,current_region=$4,cleared_cells=$5,active_combat=$6,last_mana_update=NOW(),primeval_stones=$7,hp=$8,human_qi=$9,visited_subzones=$10,statuses=$11,version=version+1,state_revision=COALESCE(state_revision,0)+1 WHERE id=$12`,
		player.CurrentMana, player.X, player.Y, player.CurrentRegion, player.ClearedCells, active,
		player.PrimevalStones, player.HP, player.HumanQi, player.VisitedSubZones, player.Statuses, player.ID)
	if err == nil {
		player.Version++
		player.StateRevision++
	}
	return err
}
