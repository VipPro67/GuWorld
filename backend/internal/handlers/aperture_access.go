package handlers

import (
	"database/sql"

	"github.com/google/uuid"
)

// apertureInstanceForPlayer returns the aperture instance the player is
// currently inside, if they are its owner or a registered guest.
func apertureInstanceForPlayer(tx *sql.Tx, playerID uuid.UUID) (string, bool, error) {
	var apertureID string
	var isOwner bool
	err := tx.QueryRow(`
		SELECT a.id::text, a.player_id = p.id
		FROM players p
		JOIN immortal_apertures a ON a.id = p.map_instance_id
		WHERE p.id = $1 AND p.current_region = 'aperture_interior'
		  AND (a.player_id = p.id OR EXISTS (
			SELECT 1 FROM aperture_guest_roster g
			WHERE g.aperture_id = a.id AND g.player_id = p.id
		  ))
	`, playerID).Scan(&apertureID, &isOwner)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return apertureID, isOwner, nil
}
