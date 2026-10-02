package handlers

import (
	"database/sql"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func raceHasDeadAperture(race string) bool {
	player := models.Player{}
	player.Race = race
	return gamefx.Get(&player).Immortal
}

// suppressTribulationsForDeadAperture closes any stale event that predates a
// player's transformation into an immortal race. A dead aperture never faces
// tribulations, so the cancellation carries no reward or penalty.
func suppressTribulationsForDeadAperture(tx *sql.Tx, playerID string) (bool, error) {
	var race string
	if err := tx.QueryRow(`SELECT COALESCE(race, '') FROM players WHERE id = $1`, playerID).Scan(&race); err != nil {
		return false, err
	}
	if !raceHasDeadAperture(race) {
		return false, nil
	}

	if _, err := tx.Exec(`
		UPDATE tribulation_events
		SET status = 'CANCELLED', ended_at = COALESCE(ended_at, NOW()), reward_dao_marks = 0
		WHERE player_id = $1 AND status = 'ACTIVE'
	`, playerID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`
		UPDATE tribulation_enemies
		SET is_dead = true, updated_at = NOW()
		WHERE event_id IN (SELECT id FROM tribulation_events WHERE player_id = $1 AND status = 'CANCELLED')
		  AND is_dead = false
	`, playerID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`
		UPDATE immortal_apertures
		SET state = 'STABLE', tribulation_pending_at = NULL, updated_at = NOW()
		WHERE player_id = $1 AND state IN ('TRIBULATION_ACTIVE', 'TRIBULATION_PENDING')
	`, playerID); err != nil {
		return false, err
	}
	if _, err := tx.Exec(`
		UPDATE players
		SET next_tribulation_time = NULL,
		    active_combat = CASE
		        WHEN active_combat->>'tribulation_event_id' <> ''
		          OR active_combat->>'enemy_template_id' = 'monster_heavenly_tribulation'
		        THEN NULL ELSE active_combat END
		WHERE id = $1
	`, playerID); err != nil {
		return false, err
	}
	return true, nil
}
