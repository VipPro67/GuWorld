package services

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// CombatSession represents an active combat session stored separately from the player row.
type CombatSession struct {
	ID        string          `json:"id"`
	PlayerID  string          `json:"player_id"`
	State     json.RawMessage `json:"state"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// CombatSessionService handles combat session persistence.
type CombatSessionService struct {
	db *sql.DB
}

func NewCombatSessionService(db *sql.DB) *CombatSessionService {
	return &CombatSessionService{db: db}
}

// Get retrieves the active combat session for a player.
func (s *CombatSessionService) Get(q QueryRower, playerID string) (*models.CombatState, error) {
	var rawJSON []byte
	err := q.QueryRow(`
		SELECT state FROM combat_sessions
		WHERE player_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, playerID).Scan(&rawJSON)
	if err != nil {
		return nil, err
	}

	var cs models.CombatState
	if len(rawJSON) == 0 || string(rawJSON) == "null" {
		return nil, nil
	}
	if err := json.Unmarshal(rawJSON, &cs); err != nil {
		return nil, err
	}
	return &cs, nil
}

// Save inserts or updates a combat session.
func (s *CombatSessionService) Save(tx *sql.Tx, playerID string, state *models.CombatState) error {
	if state == nil {
		return s.Delete(tx, playerID)
	}

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO combat_sessions (player_id, state, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (player_id) DO UPDATE SET
			state = EXCLUDED.state,
			updated_at = NOW()
	`, playerID, data)
	return err
}

// Delete removes the combat session for a player.
func (s *CombatSessionService) Delete(tx *sql.Tx, playerID string) error {
	_, err := tx.Exec(`DELETE FROM combat_sessions WHERE player_id = $1`, playerID)
	return err
}

// MigrateFromPlayer migrates active_combat from the players JSONB column to combat_sessions table.
func (s *CombatSessionService) MigrateFromPlayer(tx *sql.Tx, playerID uuid.UUID, state *models.CombatState) error {
	if state == nil {
		return nil
	}
	return s.Save(tx, playerID.String(), state)
}
