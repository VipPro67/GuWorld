package handlers

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type lockedGuState struct {
	ownerID                   uuid.UUID
	guID                      string
	rank                      int
	isVital, equipped, locked bool
	state                     string
}

func lockGuInstances(tx *sql.Tx, instanceIDs []string) (map[string]lockedGuState, error) {
	states := make(map[string]lockedGuState, len(instanceIDs))
	if len(instanceIDs) == 0 {
		return states, nil
	}

	rows, err := tx.Query(`
		SELECT id, player_id, gu_id, gu_rank, is_vital, equipped, is_locked, state
		FROM player_gu
		WHERE id = ANY($1)
		ORDER BY id
		FOR UPDATE
	`, pq.Array(instanceIDs))
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var instanceID string
		var state lockedGuState
		if err := rows.Scan(
			&instanceID, &state.ownerID, &state.guID, &state.rank,
			&state.isVital, &state.equipped, &state.locked, &state.state,
		); err != nil {
			rows.Close()
			return nil, err
		}
		states[instanceID] = state
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return states, nil
}

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

func loadActiveProductGuReservationsForInstances(tx *sql.Tx, instanceIDs []string) (map[string]bool, error) {
	reserved := make(map[string]bool, len(instanceIDs))
	if len(instanceIDs) == 0 {
		return reserved, nil
	}
	rows, err := tx.Query(`
		SELECT gu_instance_id FROM player_killer_move_gu_reservations WHERE gu_instance_id = ANY($1)
		UNION
		SELECT gu_instance_id FROM player_gu_house_core_reservations WHERE gu_instance_id = ANY($1)
		UNION
		SELECT gu_instance_id FROM teleport_array_gu_reservations WHERE gu_instance_id = ANY($1)
	`, pq.Array(instanceIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
