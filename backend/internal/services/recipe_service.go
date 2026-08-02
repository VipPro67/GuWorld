package services

import (
	"database/sql"
	"time"
)

// RecipeProgress represents a player's progress on a single refining recipe.
type RecipeProgress struct {
	PlayerID       string     `json:"player_id"`
	RecipeID       string     `json:"recipe_id"`
	CompletionRate float64    `json:"completion_rate"`
	IsUnlocked     bool       `json:"is_unlocked"`
	DeduceCount    int        `json:"deduce_count"`
	LastDeduceAt   *time.Time `json:"last_deduce_at,omitempty"`
	Favorite       bool       `json:"favorite"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// RecipeService handles player recipe progress persistence.
type RecipeService struct {
	db *sql.DB
}

func NewRecipeService(db *sql.DB) *RecipeService {
	return &RecipeService{db: db}
}

// ListByPlayer returns all recipe progress rows for a player.
func (s *RecipeService) ListByPlayer(playerID string) ([]RecipeProgress, error) {
	rows, err := s.db.Query(`
		SELECT player_id, recipe_id, completion_rate, is_unlocked, deduce_count,
		       last_deduce_at, favorite, created_at, updated_at
		FROM player_recipe_progress
		WHERE player_id = $1
		ORDER BY recipe_id
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RecipeProgress
	for rows.Next() {
		var r RecipeProgress
		if err := rows.Scan(
			&r.PlayerID, &r.RecipeID, &r.CompletionRate, &r.IsUnlocked,
			&r.DeduceCount, &r.LastDeduceAt, &r.Favorite, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// Get returns a single recipe progress row.
func (s *RecipeService) Get(q QueryRower, playerID, recipeID string) (*RecipeProgress, error) {
	var r RecipeProgress
	err := q.QueryRow(`
		SELECT player_id, recipe_id, completion_rate, is_unlocked, deduce_count,
		       last_deduce_at, favorite, created_at, updated_at
		FROM player_recipe_progress
		WHERE player_id = $1 AND recipe_id = $2
	`, playerID, recipeID).Scan(
		&r.PlayerID, &r.RecipeID, &r.CompletionRate, &r.IsUnlocked,
		&r.DeduceCount, &r.LastDeduceAt, &r.Favorite, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *RecipeService) exec(tx *sql.Tx, query string, args ...interface{}) (sql.Result, error) {
	if tx != nil {
		return tx.Exec(query, args...)
	}
	return s.db.Exec(query, args...)
}

// Upsert inserts or updates a recipe progress row.
func (s *RecipeService) Upsert(tx *sql.Tx, r *RecipeProgress) error {
	_, err := s.exec(tx, `
		INSERT INTO player_recipe_progress (player_id, recipe_id, completion_rate, is_unlocked, deduce_count, last_deduce_at, favorite, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (player_id, recipe_id) DO UPDATE SET
			completion_rate = EXCLUDED.completion_rate,
			is_unlocked = EXCLUDED.is_unlocked,
			deduce_count = EXCLUDED.deduce_count,
			last_deduce_at = EXCLUDED.last_deduce_at,
			favorite = EXCLUDED.favorite,
			updated_at = NOW()
	`, r.PlayerID, r.RecipeID, r.CompletionRate, r.IsUnlocked,
		r.DeduceCount, r.LastDeduceAt, r.Favorite)
	return err
}

// Delete removes a recipe progress row.
func (s *RecipeService) Delete(tx *sql.Tx, playerID, recipeID string) error {
	_, err := s.exec(tx, `DELETE FROM player_recipe_progress WHERE player_id = $1 AND recipe_id = $2`, playerID, recipeID)
	return err
}
