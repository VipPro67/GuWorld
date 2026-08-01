package db

import (
	"database/sql"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// Queryer is satisfied by both *sql.DB and *sql.Tx for read queries.
type Queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Execer is satisfied by both *sql.DB and *sql.Tx for single write statements.
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

const playerGuColumns = `id, gu_id, gu_rank, hunger, state, equipped, is_vital, integration_time, equipped_at, growth_hours`

func scanGuRow(rows interface{ Scan(...any) error }) (models.GuInstance, error) {
	var (
		gu              models.GuInstance
		hunger          float64
		integrationTime float64
		equippedAt      sql.NullTime
		growthHours     float64
	)
	if err := rows.Scan(&gu.InstanceID, &gu.GuID, &gu.Rank, &hunger, &gu.Status,
		&gu.Equipped, &gu.IsVital, &integrationTime, &equippedAt, &growthHours); err != nil {
		return gu, err
	}
	gu.Hunger = int(hunger)
	gu.IntegrationTime = integrationTime
	if equippedAt.Valid {
		t := equippedAt.Time
		gu.EquippedAt = &t
	}
	gu.GrowthHours = growthHours
	return gu, nil
}

// LoadPlayerGus loads a player's Gu instances from the player_gu table.
// When equippedOnly is true, only equipped or vital Gus are returned (used for
// the lightweight player status snapshot). Otherwise the full inventory is returned.
func LoadPlayerGus(q Queryer, playerID string, equippedOnly bool) (models.GuInventory, error) {
	query := `SELECT ` + playerGuColumns + ` FROM player_gu WHERE player_id = $1`
	if equippedOnly {
		query += ` AND (equipped OR is_vital)`
	}
	query += ` ORDER BY created_at ASC, id ASC`

	rows, err := q.Query(query, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	inventory := models.GuInventory{}
	for rows.Next() {
		gu, err := scanGuRow(rows)
		if err != nil {
			return nil, err
		}
		inventory = append(inventory, gu)
	}
	return inventory, rows.Err()
}

// AddPlayerGu inserts a single Gu instance owned by a player, upserting on id conflict.
func AddPlayerGu(tx Execer, playerID string, gu models.GuInstance) error {
	var equippedAt any
	if gu.EquippedAt != nil {
		equippedAt = *gu.EquippedAt
	}
	_, err := tx.Exec(`
		INSERT INTO player_gu (id, player_id, gu_id, gu_rank, rank, level, hunger, state, location, is_locked, equipped, is_vital, integration_time, equipped_at, growth_hours)
		VALUES ($1,$2,$3,$4,$4,1,$5,$6,'INVENTORY',false,$7,$8,$9,$10,$11)
		ON CONFLICT (id) DO UPDATE SET
			player_id = EXCLUDED.player_id,
			gu_id = EXCLUDED.gu_id,
			gu_rank = EXCLUDED.gu_rank,
			rank = EXCLUDED.rank,
			hunger = EXCLUDED.hunger,
			state = EXCLUDED.state,
			equipped = EXCLUDED.equipped,
			is_vital = EXCLUDED.is_vital,
			integration_time = EXCLUDED.integration_time,
			equipped_at = EXCLUDED.equipped_at,
			growth_hours = EXCLUDED.growth_hours,
			updated_at = NOW()`,
		gu.InstanceID, playerID, gu.GuID, gu.Rank, gu.Hunger, gu.Status,
		gu.Equipped, gu.IsVital, gu.IntegrationTime, equippedAt, gu.GrowthHours)
	return err
}

// RemovePlayerGu deletes a single Gu instance by its instance id.
func RemovePlayerGu(tx Execer, instanceID string) error {
	_, err := tx.Exec(`DELETE FROM player_gu WHERE id = $1`, instanceID)
	return err
}

// UpdatePlayerGu updates an existing Gu instance row, keyed by instance id.
func UpdatePlayerGu(tx Execer, gu models.GuInstance) error {
	var equippedAt any
	if gu.EquippedAt != nil {
		equippedAt = *gu.EquippedAt
	}
	_, err := tx.Exec(`
		UPDATE player_gu SET
			gu_id = $2, gu_rank = $3, rank = $3, hunger = $4, state = $5,
			equipped = $6, is_vital = $7, integration_time = $8, equipped_at = $9, growth_hours = $10,
			updated_at = NOW()
		WHERE id = $1`,
		gu.InstanceID, gu.GuID, gu.Rank, gu.Hunger, gu.Status,
		gu.Equipped, gu.IsVital, gu.IntegrationTime, equippedAt, gu.GrowthHours)
	return err
}

// SavePlayerGus replaces a player's full Gu inventory in the player_gu table.
func SavePlayerGus(tx *sql.Tx, playerID string, inv models.GuInventory) error {
	if _, err := tx.Exec(`DELETE FROM player_gu WHERE player_id = $1`, playerID); err != nil {
		return err
	}
	for _, gu := range inv {
		if err := AddPlayerGu(tx, playerID, gu); err != nil {
			return err
		}
	}
	return nil
}

// CountPlayerGusByGuID returns how many Gu instances of a given gu_id exist across all players.
func CountPlayerGusByGuID(q Queryer, guID string) (int, error) {
	var count int
	err := q.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE gu_id = $1`, guID).Scan(&count)
	return count, err
}

// GuInventoryAggregate returns a scalar subquery that reconstructs a player's
// Gu inventory as a JSONB array from the player_gu table. playerRef must be a
// valid SQL reference to the player's id column (e.g. "players.id" or "p.id").
// It keeps legacy SELECT/RETURNING scans of the gu_inventory column working
// after the column is dropped from the players table.
func GuInventoryAggregate(playerRef string) string {
	return `(SELECT COALESCE(jsonb_agg(jsonb_build_object(
		'instance_id', pg.id,
		'gu_id', pg.gu_id,
		'rank', pg.gu_rank,
		'hunger', pg.hunger::int,
		'status', pg.state,
		'equipped', pg.equipped,
		'is_vital', pg.is_vital,
		'integration_time', pg.integration_time,
		'equipped_at', pg.equipped_at,
		'growth_hours', pg.growth_hours
	) ORDER BY pg.created_at, pg.id), '[]'::jsonb)
	FROM player_gu pg WHERE pg.player_id = ` + playerRef + `)`
}
