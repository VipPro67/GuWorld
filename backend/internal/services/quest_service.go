package services

import (
	"database/sql"
	"encoding/json"
	"time"
)

// QuestProgress represents a player's progress on a quest.
type QuestProgress struct {
	PlayerID     string          `json:"player_id"`
	QuestID      string          `json:"quest_id"`
	Status       string          `json:"status"` // ACTIVE, COMPLETED, FAILED
	ProgressData json.RawMessage `json:"progress_data"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// QuestService handles player quest progress persistence.
type QuestService struct {
	db *sql.DB
}

func NewQuestService(db *sql.DB) *QuestService {
	return &QuestService{db: db}
}

// ListByPlayer returns all quest progress rows for a player.
func (s *QuestService) ListByPlayer(playerID string) ([]QuestProgress, error) {
	rows, err := s.db.Query(`
		SELECT player_id, quest_id, status, progress_data, completed_at, created_at, updated_at
		FROM player_quest_progress
		WHERE player_id = $1
		ORDER BY quest_id
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []QuestProgress
	for rows.Next() {
		var q QuestProgress
		if err := rows.Scan(&q.PlayerID, &q.QuestID, &q.Status, &q.ProgressData, &q.CompletedAt, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, q)
	}
	return results, rows.Err()
}

// ListActive returns all active quests for a player.
func (s *QuestService) ListActive(playerID string) ([]QuestProgress, error) {
	rows, err := s.db.Query(`
		SELECT player_id, quest_id, status, progress_data, completed_at, created_at, updated_at
		FROM player_quest_progress
		WHERE player_id = $1 AND status = 'ACTIVE'
		ORDER BY quest_id
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []QuestProgress
	for rows.Next() {
		var q QuestProgress
		if err := rows.Scan(&q.PlayerID, &q.QuestID, &q.Status, &q.ProgressData, &q.CompletedAt, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, q)
	}
	return results, rows.Err()
}

// Get returns a single quest progress row.
func (s *QuestService) Get(q QueryRower, playerID, questID string) (*QuestProgress, error) {
	var qp QuestProgress
	err := q.QueryRow(`
		SELECT player_id, quest_id, status, progress_data, completed_at, created_at, updated_at
		FROM player_quest_progress
		WHERE player_id = $1 AND quest_id = $2
	`, playerID, questID).Scan(
		&qp.PlayerID, &qp.QuestID, &qp.Status, &qp.ProgressData, &qp.CompletedAt, &qp.CreatedAt, &qp.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &qp, nil
}

// Upsert inserts or updates quest progress.
func (s *QuestService) Upsert(tx *sql.Tx, q *QuestProgress) error {
	_, err := tx.Exec(`
		INSERT INTO player_quest_progress (player_id, quest_id, status, progress_data, completed_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (player_id, quest_id)
		DO UPDATE SET status = EXCLUDED.status, progress_data = EXCLUDED.progress_data,
		              completed_at = EXCLUDED.completed_at, updated_at = NOW()
	`, q.PlayerID, q.QuestID, q.Status, q.ProgressData, q.CompletedAt)
	return err
}

// Complete marks a quest as completed.
func (s *QuestService) Complete(tx *sql.Tx, playerID, questID string) error {
	now := time.Now()
	_, err := tx.Exec(`
		UPDATE player_quest_progress SET status = 'COMPLETED', completed_at = $1, updated_at = NOW()
		WHERE player_id = $2 AND quest_id = $3
	`, now, playerID, questID)
	return err
}

// Delete removes a quest progress row.
func (s *QuestService) Delete(tx *sql.Tx, playerID, questID string) error {
	_, err := tx.Exec(`DELETE FROM player_quest_progress WHERE player_id = $1 AND quest_id = $2`, playerID, questID)
	return err
}

// DeleteAll removes all quest progress for a player.
func (s *QuestService) DeleteAll(tx *sql.Tx, playerID string) error {
	_, err := tx.Exec(`DELETE FROM player_quest_progress WHERE player_id = $1`, playerID)
	return err
}
