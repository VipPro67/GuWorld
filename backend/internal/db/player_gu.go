package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// IsImmortalConflict reports whether err is a unique-violation from the
// Tiên Cổ cross-rank indexes (uq_player_gu_immortal / uq_sect_vault_gu_immortal):
// hai bên cùng tạo/rút cùng gu_id rank>=6, bên thua nhận conflict thay vì 500.
func IsImmortalConflict(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return pqErr.Constraint == "uq_player_gu_immortal" ||
			pqErr.Constraint == "uq_sect_vault_gu_immortal"
	}
	return false
}

// Queryer is satisfied by both *sql.DB and *sql.Tx for read queries.
type Queryer interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Execer is satisfied by both *sql.DB and *sql.Tx for single write statements.
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// QueryExecer is satisfied by both *sql.DB and *sql.Tx for read+write flows.
type QueryExecer interface {
	Queryer
	Execer
}

// MaxEquippedGus caps how many Gu instances a player may have equipped at once.
// Mirrors the aperture capacity enforced by the EquipGu handler.
const MaxEquippedGus = 60

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
// When the new instance is equipped, it first counts the player's already-equipped
// Gus and refuses to equip beyond MaxEquippedGus to avoid over-equipping.
func AddPlayerGu(tx QueryExecer, playerID string, gu models.GuInstance) error {
	if gu.Equipped {
		var equippedCount int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM player_gu WHERE player_id = $1 AND equipped AND id <> $2`, playerID, gu.InstanceID).Scan(&equippedCount); err != nil {
			return err
		}
		if equippedCount >= MaxEquippedGus {
			return fmt.Errorf("aperture capacity full: cannot equip more than %d Gu bugs", MaxEquippedGus)
		}
	}
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

// RemovePlayerGuForPlayer deletes one Gu instance only when it belongs to the player.
func RemovePlayerGuForPlayer(tx Execer, playerID, instanceID string) error {
	result, err := tx.Exec(`DELETE FROM player_gu WHERE player_id = $1 AND id = $2`, playerID, instanceID)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err != nil {
		return err
	} else if affected != 1 {
		return fmt.Errorf("player_gu instance %q does not belong to player %q", instanceID, playerID)
	}
	return nil
}

// RemovePlayerGus deletes multiple Gu instances by their instance ids in a single query.
func RemovePlayerGus(tx Execer, instanceIDs []string) error {
	if len(instanceIDs) == 0 {
		return nil
	}
	_, err := tx.Exec(`DELETE FROM player_gu WHERE id = ANY($1::text[])`, pq.Array(instanceIDs))
	return err
}

// RemovePlayerGusForPlayer deletes Gu instances only when they belong to the player.
func RemovePlayerGusForPlayer(tx Execer, playerID string, instanceIDs []string) error {
	if len(instanceIDs) == 0 {
		return nil
	}
	result, err := tx.Exec(`DELETE FROM player_gu WHERE player_id = $1 AND id = ANY($2::text[])`, playerID, pq.Array(instanceIDs))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	uniqueIDs := uniqueStrings(instanceIDs)
	if affected != int64(len(uniqueIDs)) {
		return fmt.Errorf("deleted %d of %d player_gu instances for player %q", affected, len(uniqueIDs), playerID)
	}
	return nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// UpdatePlayerGusBulk updates the mutable fields of multiple Gu instances in a
// single query, keyed by instance id.
func UpdatePlayerGusBulk(tx Execer, gus models.GuInventory) error {
	if len(gus) == 0 {
		return nil
	}

	n := len(gus)
	ids := make([]string, n)
	guIDs := make([]string, n)
	ranks := make([]int, n)
	hunger := make([]int, n)
	states := make([]string, n)
	equipped := make([]bool, n)
	vital := make([]bool, n)
	intTimes := make([]float64, n)
	growths := make([]float64, n)
	equippedAts := make([]string, n)

	for i, gu := range gus {
		ids[i] = gu.InstanceID
		guIDs[i] = gu.GuID
		ranks[i] = gu.Rank
		hunger[i] = gu.Hunger
		states[i] = gu.Status
		equipped[i] = gu.Equipped
		vital[i] = gu.IsVital
		intTimes[i] = gu.IntegrationTime
		growths[i] = gu.GrowthHours
		if gu.EquippedAt != nil {
			equippedAts[i] = gu.EquippedAt.Format("2006-01-02T15:04:05Z")
		}
	}

	query := `UPDATE player_gu AS pg SET
		gu_id = u.gu_id,
		gu_rank = u.gu_rank,
		rank = u.rank,
		hunger = u.hunger,
		state = u.state,
		equipped = u.equipped,
		is_vital = u.is_vital,
		integration_time = u.integration_time,
		equipped_at = u.equipped_at,
		growth_hours = u.growth_hours,
		updated_at = NOW()
	FROM (
		SELECT unnest($1::text[]) AS id,
		       unnest($2::text[]) AS gu_id,
		       unnest($3::int[]) AS gu_rank,
		       unnest($4::int[]) AS rank,
		       unnest($5::int[]) AS hunger,
		       unnest($6::text[]) AS state,
		       unnest($7::bool[]) AS equipped,
		       unnest($8::bool[]) AS is_vital,
		       unnest($9::float8[]) AS integration_time,
		       NULLIF(unnest($10::text[]), '')::timestamptz AS equipped_at,
		       unnest($11::float8[]) AS growth_hours
	) AS u(id, gu_id, gu_rank, rank, hunger, state, equipped, is_vital, integration_time, equipped_at, growth_hours)
	WHERE pg.id = u.id`

	_, err := tx.Exec(query,
		pq.Array(ids), pq.Array(guIDs), pq.Array(ranks), pq.Array(ranks), pq.Array(hunger), pq.Array(states),
		pq.Array(equipped), pq.Array(vital), pq.Array(intTimes), pq.Array(equippedAts), pq.Array(growths))
	return err
}

// SyncPlayerGus persists the difference between two snapshots of a player's Gu
// inventory. before must be a complete snapshot of the player's player_gu rows
// (e.g. the inventory loaded at the start of a request) and after the desired
// post-state. Only changed rows are written: removed instances are deleted, new
// instances inserted, and instances whose mutable fields changed are updated.
//
// The change set is always applied incrementally (delete removed ids, bulk
// update changed rows, bulk insert new rows). We deliberately avoid a full
// delete-and-reinsert rewrite: `before` is loaded outside the players row lock
// (PlayerService.Load reads player_gu on a pool connection in a goroutine), so
// a full rewrite would race the hunger/growth workers that UPDATE/DELETE
// player_gu directly, silently losing or resurrecting rows whenever a large
// batch triggers the destructive path.
func SyncPlayerGus(tx *sql.Tx, playerID string, before models.GuInventory, after models.GuInventory) error {
	beforeIndex := make(map[string]models.GuInstance, len(before))
	for _, gu := range before {
		beforeIndex[gu.InstanceID] = gu
	}
	afterSet := make(map[string]struct{}, len(after))
	for _, gu := range after {
		afterSet[gu.InstanceID] = struct{}{}
	}

	var removed []string
	for id := range beforeIndex {
		if _, ok := afterSet[id]; !ok {
			removed = append(removed, id)
		}
	}
	var added models.GuInventory
	var updated models.GuInventory
	dropped := 0
	for _, gu := range after {
		prev, ok := beforeIndex[gu.InstanceID]
		if !ok {
			// Drop instances whose (gu_id, gu_rank) no longer exist in the
			// dictionary to avoid FK violations.
			if _, valid := models.GetGu(gu.GuID, gu.Rank); valid {
				added = append(added, gu)
			} else {
				dropped++
			}
			continue
		}
		if !sameGuInstance(prev, gu) {
			updated = append(updated, gu)
		}
	}

	if dropped > 0 {
		logger.S.Warnf("SyncPlayerGus player=%s before=%d after=%d dropped=%d (invalid gu_id/rank) removed=%d updated=%d added=%d",
			playerID, len(before), len(after), dropped, len(removed), len(updated), len(added))
	}

	// ── SAFETY: NEVER DELETE through diff ─────────────────────────────
	// SyncPlayerGus only ADDS and UPDATES. If a handler needs to remove
	// Gus, it must call RemovePlayerGu/RemovePlayerGus explicitly BEFORE
	// SyncPlayerGus. This prevents silent Gu loss from handler bugs.
	if len(removed) > 0 {
		// Explicit deletes happen before SyncPlayerGus in consumption flows
		// such as refining and feeding. Only warn for rows that still exist,
		// which indicates that the handler forgot the explicit delete.
		rows, err := tx.Query(`
			SELECT id
			FROM player_gu
			WHERE player_id = $1 AND id = ANY($2::text[])`, playerID, pq.Array(removed))
		if err != nil {
			logger.S.Warnf("SyncPlayerGus player=%s could not verify removed Gus: %v", playerID, err)
		} else {
			defer rows.Close()
			remaining := make(map[string]struct{})
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err != nil {
					logger.S.Warnf("SyncPlayerGus player=%s could not read removed Gu verification: %v", playerID, err)
					remaining = nil
					break
				}
				remaining[id] = struct{}{}
			}
			if err := rows.Err(); err != nil {
				logger.S.Warnf("SyncPlayerGus player=%s could not finish removed Gu verification: %v", playerID, err)
				remaining = nil
			}
			if remaining != nil {
				for _, id := range removed {
					if _, exists := remaining[id]; !exists {
						continue
					}
					gu := beforeIndex[id]
					logger.S.Errorf("SyncPlayerGus player=%s SKIP_DELETE id=%s gu=%s(rank=%d,hunger=%d,state=%s,vital=%v,equipped=%v) — handler should call RemovePlayerGu explicitly!",
						playerID, id, gu.GuID, gu.Rank, gu.Hunger, gu.Status, gu.IsVital, gu.Equipped)
				}
			}
		}
	}
	// ── END SAFETY ────────────────────────────────────────────────────

	if len(added) == 0 && len(updated) == 0 {
		return nil
	}

	if len(updated) > 0 {
		if err := UpdatePlayerGusBulk(tx, updated); err != nil {
			return err
		}
	}
	if len(added) > 0 {
		if err := AddPlayerGusBulk(tx, playerID, added); err != nil {
			return err
		}
	}
	return nil
}

func sameGuInstance(a, b models.GuInstance) bool {
	if a.GuID != b.GuID || a.Rank != b.Rank || a.Hunger != b.Hunger ||
		a.Status != b.Status || a.Equipped != b.Equipped || a.IsVital != b.IsVital ||
		a.IntegrationTime != b.IntegrationTime || a.GrowthHours != b.GrowthHours {
		return false
	}
	if a.EquippedAt == nil && b.EquippedAt == nil {
		return true
	}
	if a.EquippedAt == nil || b.EquippedAt == nil {
		return false
	}
	return a.EquippedAt.Equal(*b.EquippedAt)
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

// AddPlayerGusBulk inserts multiple Gu instances for a player in a single query.
// It does not delete or modify existing Gu instances.
func AddPlayerGusBulk(tx Execer, playerID string, gus []models.GuInstance) error {
	if len(gus) == 0 {
		return nil
	}

	n := len(gus)
	ids := make([]string, n)
	guIDs := make([]string, n)
	ranks := make([]int, n)
	hunger := make([]int, n)
	states := make([]string, n)
	equipped := make([]bool, n)
	vital := make([]bool, n)
	intTimes := make([]float64, n)
	growths := make([]float64, n)
	equippedAts := make([]string, n)

	for i, gu := range gus {
		ids[i] = gu.InstanceID
		guIDs[i] = gu.GuID
		ranks[i] = gu.Rank
		hunger[i] = gu.Hunger
		states[i] = gu.Status
		equipped[i] = gu.Equipped
		vital[i] = gu.IsVital
		intTimes[i] = gu.IntegrationTime
		growths[i] = gu.GrowthHours
		if gu.EquippedAt != nil {
			equippedAts[i] = gu.EquippedAt.Format("2006-01-02T15:04:05Z")
		} else {
			equippedAts[i] = ""
		}
	}

	query := `INSERT INTO player_gu (id, player_id, gu_id, gu_rank, rank, level, hunger, state, location, is_locked, equipped, is_vital, integration_time, equipped_at, growth_hours)
		SELECT unnest($1::text[]), $2, unnest($3::text[]), unnest($4::int[]), unnest($4::int[]), 1, unnest($5::int[]), unnest($6::text[]), 'INVENTORY', false, unnest($7::bool[]), unnest($8::bool[]), unnest($9::float[]), NULLIF(unnest($10::text[]), '')::timestamptz, unnest($11::float[])`

	_, err := tx.Exec(query,
		pq.Array(ids), playerID,
		pq.Array(guIDs), pq.Array(ranks), pq.Array(hunger), pq.Array(states),
		pq.Array(equipped), pq.Array(vital), pq.Array(intTimes),
		pq.Array(equippedAts), pq.Array(growths))
	return err
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

// GuInventoryAggregateEquipped is like GuInventoryAggregate but only includes
// equipped ACTIVE Gus — the subset used for combat stats and PVP snapshots.
func GuInventoryAggregateEquipped(playerRef string) string {
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
	FROM player_gu pg WHERE pg.player_id = ` + playerRef + ` AND pg.equipped = true AND pg.state = 'ACTIVE')`
}

// EquippedTimeSpeedBonus sums the passive time_speed_multiplier contributed by
// a player's equipped ACTIVE Gus. It queries player_gu and looks up the memory
// registry, avoiding complex JSONB SQL queries that trigger JIT compilation.
func EquippedTimeSpeedBonus(q Queryer, playerID string) (float64, error) {
	rows, err := q.Query(`
		SELECT gu_id, gu_rank 
		FROM player_gu 
		WHERE player_id = $1 AND equipped = true AND state = 'ACTIVE'`, playerID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var total float64
	for rows.Next() {
		var guID string
		var rank int
		if err := rows.Scan(&guID, &rank); err == nil {
			if spec, ok := models.GetGu(guID, rank); ok {
				total += models.GuStatValue(spec, "TIME_SPEED")
			}
		}
	}
	return total, nil
}

// TransferGuOwnership transfers a Gu instance from one player to another.
// It sets equipped=false, is_vital=false, equipped_at=nil and updates player_id.
func TransferGuOwnership(tx Execer, instanceID, fromPlayerID, toPlayerID string) error {
	res, err := tx.Exec(`
		UPDATE player_gu 
		SET player_id = $1, equipped = false, is_vital = false, equipped_at = NULL, updated_at = NOW()
		WHERE id = $2 AND player_id = $3`,
		toPlayerID, instanceID, fromPlayerID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("gu instance %s not found or not owned by %s", instanceID, fromPlayerID)
	}
	return nil
}
