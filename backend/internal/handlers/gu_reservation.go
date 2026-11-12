package handlers

import (
	"database/sql"

	"github.com/google/uuid"
)

// guHasActiveProductReservation reports whether an equipped product currently
// owns this Gu as a component. The database constraints remain the final guard;
// callers use this query to return a domain conflict before attempting a
// consume, transfer, or refinement mutation.
func guHasActiveProductReservation(tx *sql.Tx, instanceID string) (bool, error) {
	var reserved bool
	err := tx.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM player_killer_move_gu_reservations WHERE gu_instance_id=$1
		UNION ALL
		SELECT 1 FROM player_gu_house_core_reservations WHERE gu_instance_id=$1
		UNION ALL
		SELECT 1 FROM teleport_array_gu_reservations WHERE gu_instance_id=$1
	)`, instanceID).Scan(&reserved)
	return reserved, err
}

func loadActiveProductGuReservations(tx *sql.Tx, playerID uuid.UUID) (map[string]bool, error) {
	rows, err := tx.Query(`SELECT gu_instance_id FROM player_killer_move_gu_reservations WHERE player_id=$1
		UNION SELECT gu_instance_id FROM player_gu_house_core_reservations WHERE player_id=$1
		UNION SELECT gu_instance_id FROM teleport_array_gu_reservations WHERE owner_id=$1`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reserved := make(map[string]bool)
	for rows.Next() {
		var instanceID string
		if err := rows.Scan(&instanceID); err != nil {
			return nil, err
		}
		reserved[instanceID] = true
	}
	return reserved, rows.Err()
}

// lockGuProductOwner serializes death-drop selection with product equip flows,
// which use the same owner-row lock before creating or releasing reservations.
func lockGuProductOwner(tx *sql.Tx, playerID uuid.UUID) error {
	var lockedID uuid.UUID
	return tx.QueryRow(`SELECT id FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&lockedID)
}
