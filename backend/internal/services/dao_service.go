package services

import (
	"database/sql"
	"time"
)

// DaoPath represents a player's progress on a single cultivation path.
type DaoPath struct {
	PlayerID  string    `json:"player_id"`
	Path      string    `json:"path"`
	Mastery   string    `json:"mastery"` // Ordinary, Master, Grandmaster, Great Grandmaster, Supreme Grandmaster
	DaoMarks  int       `json:"dao_marks"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DaoService handles player dao path persistence.
type DaoService struct {
	db *sql.DB
}

func NewDaoService(db *sql.DB) *DaoService {
	return &DaoService{db: db}
}

// ListByPlayer returns all dao paths for a player.
func (s *DaoService) ListByPlayer(playerID string) ([]DaoPath, error) {
	rows, err := s.db.Query(`
		SELECT player_id, path, mastery, dao_marks, created_at, updated_at
		FROM player_dao_path
		WHERE player_id = $1
		ORDER BY path
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DaoPath
	for rows.Next() {
		var d DaoPath
		if err := rows.Scan(&d.PlayerID, &d.Path, &d.Mastery, &d.DaoMarks, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, d)
	}
	return results, rows.Err()
}

// Get returns a single dao path for a player.
func (s *DaoService) Get(q QueryRower, playerID, path string) (*DaoPath, error) {
	var d DaoPath
	err := q.QueryRow(`
		SELECT player_id, path, mastery, dao_marks, created_at, updated_at
		FROM player_dao_path
		WHERE player_id = $1 AND path = $2
	`, playerID, path).Scan(&d.PlayerID, &d.Path, &d.Mastery, &d.DaoMarks, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

// Upsert inserts or updates a dao path row.
func (s *DaoService) Upsert(tx *sql.Tx, d *DaoPath) error {
	_, err := tx.Exec(`
		INSERT INTO player_dao_path (player_id, path, mastery, dao_marks, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (player_id, path) DO UPDATE SET
			mastery = EXCLUDED.mastery,
			dao_marks = EXCLUDED.dao_marks,
			updated_at = NOW()
	`, d.PlayerID, d.Path, d.Mastery, d.DaoMarks)
	return err
}

// AddDaoMarks increments dao marks for a path (creates row if not exists).
func (s *DaoService) AddDaoMarks(tx *sql.Tx, playerID, path string, amount int) error {
	_, err := tx.Exec(`
		INSERT INTO player_dao_path (player_id, path, mastery, dao_marks, updated_at)
		VALUES ($1, $2, 'Ordinary', $3, NOW())
		ON CONFLICT (player_id, path) DO UPDATE SET
			dao_marks = player_dao_path.dao_marks + $3,
			updated_at = NOW()
	`, playerID, path, amount)
	return err
}

// SetMastery updates the mastery level for a path.
func (s *DaoService) SetMastery(tx *sql.Tx, playerID, path, mastery string) error {
	_, err := tx.Exec(`
		INSERT INTO player_dao_path (player_id, path, mastery, dao_marks, updated_at)
		VALUES ($1, $2, $3, 0, NOW())
		ON CONFLICT (player_id, path) DO UPDATE SET
			mastery = EXCLUDED.mastery,
			updated_at = NOW()
	`, playerID, path, mastery)
	return err
}

// Delete removes a dao path row.
func (s *DaoService) Delete(tx *sql.Tx, playerID, path string) error {
	_, err := tx.Exec(`DELETE FROM player_dao_path WHERE player_id = $1 AND path = $2`, playerID, path)
	return err
}
