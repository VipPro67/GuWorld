package services

import (
	"database/sql"
	"time"
)

// MapCellProgress represents exploration data for a single map cell.
type MapCellProgress struct {
	PlayerID   string    `json:"player_id"`
	RegionID   string    `json:"region_id"`
	CellX      int       `json:"cell_x"`
	CellY      int       `json:"cell_y"`
	Explored   bool      `json:"explored"`
	Cleared    bool      `json:"cleared"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// MapService handles player map progress persistence.
type MapService struct {
	db *sql.DB
}

func NewMapService(db *sql.DB) *MapService {
	return &MapService{db: db}
}

// ListExplored returns all explored cells for a player.
func (s *MapService) ListExplored(playerID, regionID string) ([]MapCellProgress, error) {
	rows, err := s.db.Query(`
		SELECT player_id, region_id, cell_x, cell_y, explored, cleared, updated_at
		FROM player_map_progress
		WHERE player_id = $1 AND region_id = $2 AND explored = true
		ORDER BY cell_x, cell_y
	`, playerID, regionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMapRows(rows)
}

// ListCleared returns all cleared cells for a player.
func (s *MapService) ListCleared(playerID string) ([]MapCellProgress, error) {
	rows, err := s.db.Query(`
		SELECT player_id, region_id, cell_x, cell_y, explored, cleared, updated_at
		FROM player_map_progress
		WHERE player_id = $1 AND cleared = true
		ORDER BY region_id, cell_x, cell_y
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMapRows(rows)
}

func scanMapRows(rows *sql.Rows) ([]MapCellProgress, error) {
	var results []MapCellProgress
	for rows.Next() {
		var m MapCellProgress
		if err := rows.Scan(&m.PlayerID, &m.RegionID, &m.CellX, &m.CellY, &m.Explored, &m.Cleared, &m.UpdatedAt); err != nil {
			return nil, err
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

// UpsertCell inserts or updates a single map cell progress.
func (s *MapService) UpsertCell(tx *sql.Tx, m *MapCellProgress) error {
	_, err := tx.Exec(`
		INSERT INTO player_map_progress (player_id, region_id, cell_x, cell_y, explored, cleared, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (player_id, region_id, cell_x, cell_y) DO UPDATE SET
			explored = EXCLUDED.explored,
			cleared = EXCLUDED.cleared,
			updated_at = NOW()
	`, m.PlayerID, m.RegionID, m.CellX, m.CellY, m.Explored, m.Cleared)
	return err
}

// MarkExplored sets a cell as explored.
func (s *MapService) MarkExplored(tx *sql.Tx, playerID, regionID string, x, y int) error {
	_, err := tx.Exec(`
		INSERT INTO player_map_progress (player_id, region_id, cell_x, cell_y, explored, cleared, updated_at)
		VALUES ($1, $2, $3, $4, true, false, NOW())
		ON CONFLICT (player_id, region_id, cell_x, cell_y) DO UPDATE SET
			explored = true, updated_at = NOW()
	`, playerID, regionID, x, y)
	return err
}

// MarkCleared sets a cell as cleared.
func (s *MapService) MarkCleared(tx *sql.Tx, playerID, regionID string, x, y int) error {
	_, err := tx.Exec(`
		INSERT INTO player_map_progress (player_id, region_id, cell_x, cell_y, explored, cleared, updated_at)
		VALUES ($1, $2, $3, $4, true, true, NOW())
		ON CONFLICT (player_id, region_id, cell_x, cell_y) DO UPDATE SET
			cleared = true, updated_at = NOW()
	`, playerID, regionID, x, y)
	return err
}

// DeletePlayerCells removes all map progress for a player.
func (s *MapService) DeletePlayerCells(tx *sql.Tx, playerID string) error {
	_, err := tx.Exec(`DELETE FROM player_map_progress WHERE player_id = $1`, playerID)
	return err
}
