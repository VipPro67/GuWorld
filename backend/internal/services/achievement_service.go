package services

import (
	"database/sql"
	"time"
)

// AchievementEntry represents a player's unlocked achievement.
type AchievementEntry struct {
	PlayerID      string    `json:"player_id"`
	AchievementID string    `json:"achievement_id"`
	SelectedTitle string    `json:"selected_title"`
	SelectedFrame string    `json:"selected_frame"`
	UnlockedAt    time.Time `json:"unlocked_at"`
}

// AchievementService handles player achievement persistence.
type AchievementService struct {
	db *sql.DB
}

func NewAchievementService(db *sql.DB) *AchievementService {
	return &AchievementService{db: db}
}

// ListByPlayer returns all achievements for a player.
func (s *AchievementService) ListByPlayer(playerID string) ([]AchievementEntry, error) {
	rows, err := s.db.Query(`
		SELECT player_id, achievement_id, selected_title, selected_frame, unlocked_at
		FROM player_achievements
		WHERE player_id = $1
		ORDER BY unlocked_at DESC
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []AchievementEntry
	for rows.Next() {
		var a AchievementEntry
		if err := rows.Scan(&a.PlayerID, &a.AchievementID, &a.SelectedTitle, &a.SelectedFrame, &a.UnlockedAt); err != nil {
			return nil, err
		}
		results = append(results, a)
	}
	return results, rows.Err()
}

// HasUnlocked checks if a player has a specific achievement.
func (s *AchievementService) HasUnlocked(q QueryRower, playerID, achievementID string) (bool, error) {
	var exists bool
	err := q.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM player_achievements WHERE player_id = $1 AND achievement_id = $2)
	`, playerID, achievementID).Scan(&exists)
	return exists, err
}

// Unlock adds an achievement for a player (no-op if already unlocked).
func (s *AchievementService) Unlock(tx *sql.Tx, playerID, achievementID string) error {
	_, err := tx.Exec(`
		INSERT INTO player_achievements (player_id, achievement_id, unlocked_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (player_id, achievement_id) DO NOTHING
	`, playerID, achievementID)
	return err
}

// UpdateDisplay sets the selected title and frame for display.
func (s *AchievementService) UpdateDisplay(tx *sql.Tx, playerID, achievementID, title, frame string) error {
	_, err := tx.Exec(`
		UPDATE player_achievements SET selected_title = $1, selected_frame = $2
		WHERE player_id = $3 AND achievement_id = $4
	`, title, frame, playerID, achievementID)
	return err
}

// Delete removes an achievement entry.
func (s *AchievementService) Delete(tx *sql.Tx, playerID, achievementID string) error {
	_, err := tx.Exec(`DELETE FROM player_achievements WHERE player_id = $1 AND achievement_id = $2`, playerID, achievementID)
	return err
}

// DeleteAll removes all achievements for a player.
func (s *AchievementService) DeleteAll(tx *sql.Tx, playerID string) error {
	_, err := tx.Exec(`DELETE FROM player_achievements WHERE player_id = $1`, playerID)
	return err
}
