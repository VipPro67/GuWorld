package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

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
