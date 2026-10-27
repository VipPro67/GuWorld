// Package persistence stores durable realtime world checkpoints. Zone ticks
// never call it; join/leave and a coarse periodic worker are the only writers.
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
)

type State struct {
	PlayerID            uuid.UUID
	RegionID            string
	ZoneID              string
	InstanceID          string
	X                   float64
	Y                   float64
	HP                  float64
	Shield              float64
	SafeX               float64
	SafeY               float64
	MapVersion          string
	ContentVersion      string
	LegacyX             int
	LegacyY             int
	SyncLegacy          bool
	LocationRevision    int64
	HasLocationRevision bool
	// Combat is the restorable zone combat state (cooldowns, statuses, hidden
	// proc counters). Nil means "no durable combat state to restore".
	Combat *CombatState
}

var ErrInvalidState = errors.New("invalid realtime world state")

func (s State) Validate() error {
	if s.PlayerID == uuid.Nil || s.ZoneID == "" || s.InstanceID == "" {
		return ErrInvalidState
	}
	for _, value := range []float64{s.X, s.Y, s.HP, s.Shield, s.SafeX, s.SafeY} {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return ErrInvalidState
		}
	}
	if s.HP < 0 || s.Shield < 0 {
		return ErrInvalidState
	}
	if s.HasLocationRevision && s.LocationRevision < 0 {
		return ErrInvalidState
	}
	return nil
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Load(ctx context.Context, playerID uuid.UUID) (State, bool, error) {
	var state State
	var combatJSON []byte
	err := s.db.QueryRowContext(ctx, `
		SELECT p.id, p.current_region, p.world_zone_id, p.world_instance_id,
		       p.world_x, p.world_y, p.world_safe_x, p.world_safe_y,
		       COALESCE(c.hp, 0), COALESCE(c.shield, 0),
		       COALESCE(c.map_version, ''), COALESCE(c.content_version, ''),
		       p.world_location_revision,
		       COALESCE(cc.combat_state, '{}'::jsonb)
		FROM players p
		LEFT JOIN player_world_state_v2 c ON c.player_id = p.id
		LEFT JOIN player_world_state_combat cc ON cc.player_id = p.id
		WHERE p.id = $1 AND p.world_zone_id <> ''
	`, playerID).Scan(
		&state.PlayerID, &state.RegionID, &state.ZoneID, &state.InstanceID,
		&state.X, &state.Y, &state.SafeX, &state.SafeY,
		&state.HP, &state.Shield, &state.MapVersion, &state.ContentVersion,
		&state.LocationRevision, &combatJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}, false, nil
	}
	if err != nil {
		return State{}, false, err
	}
	state.HasLocationRevision = true
	if len(combatJSON) > 0 {
		var combat CombatState
		if err := json.Unmarshal(combatJSON, &combat); err != nil {
			return State{}, false, fmt.Errorf("decode realtime combat state: %w", err)
		}
		if combat.Version == CombatStateVersion {
			state.Combat = &combat
		}
	}
	return state, true, nil
}

func (s *Service) Save(ctx context.Context, state State) error {
	_, err := s.SaveRevision(ctx, state)
	return err
}

// SaveRevision commits the player location only if no newer owner has written
// it. The returned revision becomes the caller's next compare-and-swap token.
func (s *Service) SaveRevision(ctx context.Context, state State) (int64, error) {
	if err := state.Validate(); err != nil {
		return 0, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	revision, err := upsert(ctx, tx, state)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return revision, nil
}

// SaveAndApplyManaDelta commits position/HP and the zone's newly spent mana
// together. A failed write leaves both unchanged for the next checkpoint.
func (s *Service) SaveAndApplyManaDelta(ctx context.Context, state State, delta float64) error {
	_, err := s.SaveAndApplyManaDeltaRevision(ctx, state, delta)
	return err
}

func (s *Service) SaveAndApplyManaDeltaRevision(ctx context.Context, state State, delta float64) (int64, error) {
	return s.SaveAndApplyCombatDeltaRevision(ctx, state, delta, nil)
}

// SaveAndApplyCombatDeltaRevision atomically settles the zone's mana and Gu
// usage with the location/vitals checkpoint. Usage is additive so feeding or
// passive hunger decay during the session cannot be overwritten by a snapshot.
func (s *Service) SaveAndApplyCombatDeltaRevision(ctx context.Context, state State, delta float64, guUses map[string]uint64) (int64, error) {
	if err := state.Validate(); err != nil {
		return 0, err
	}
	if !finiteManaDelta(delta) {
		return 0, ErrInvalidState
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	revision, err := upsert(ctx, tx, state)
	if err != nil {
		return 0, err
	}
	if delta != 0 {
		if err := applyManaDelta(ctx, tx, state.PlayerID, delta); err != nil {
			return 0, err
		}
	}
	for instanceID, uses := range guUses {
		if instanceID == "" || uses == 0 || uses > 1_000_000 {
			return 0, ErrInvalidState
		}
		result, err := tx.ExecContext(ctx, `
			UPDATE player_gu SET hunger = GREATEST(0, hunger - $1), updated_at = NOW()
			WHERE id = $2 AND player_id = $3
		`, uses, instanceID, state.PlayerID)
		if err != nil {
			return 0, fmt.Errorf("settle Gu hunger for %s: %w", instanceID, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		if rows != 1 {
			return 0, fmt.Errorf("settle Gu hunger: instance %s not found for player %s", instanceID, state.PlayerID)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return revision, nil
}

// ApplyManaDelta settles the zone-owned mana change against the player row.
// The additive update preserves other character writes that happened during
// the session (for example a consumable) instead of overwriting them.
func (s *Service) ApplyManaDelta(ctx context.Context, playerID uuid.UUID, delta float64) error {
	if !finiteManaDelta(delta) {
		return ErrInvalidState
	}
	if delta == 0 {
		return nil
	}
	return applyManaDelta(ctx, s.db, playerID, delta)
}

func applyManaDelta(ctx context.Context, exec execer, playerID uuid.UUID, delta float64) error {
	result, err := exec.ExecContext(ctx, `UPDATE players SET current_mana = GREATEST(0, current_mana + $1) WHERE id = $2`, delta, playerID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("settle mana: player %s not found", playerID)
	}
	return nil
}

func finiteManaDelta(delta float64) bool { return !math.IsNaN(delta) && !math.IsInf(delta, 0) }

// SaveBatch writes checkpoints in one transaction. Invalid states are rejected
// before the transaction begins so one bad actor cannot block the zone batch.
func (s *Service) SaveBatch(ctx context.Context, states []State) error {
	if len(states) == 0 {
		return nil
	}
	for _, state := range states {
		if err := state.Validate(); err != nil {
			return err
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, state := range states {
		if _, err := upsert(ctx, tx, state); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func upsert(ctx context.Context, exec interface {
	execer
	queryRower
}, state State) (int64, error) {
	var revision int64
	query := `
		UPDATE players SET current_region = COALESCE(NULLIF($1, ''), current_region),
			world_zone_id = $2, world_instance_id = $3, world_x = $4, world_y = $5,
			world_safe_x = $6, world_safe_y = $7,
			x = CASE WHEN $8 THEN $9 ELSE x END, y = CASE WHEN $8 THEN $10 ELSE y END,
			world_location_revision = world_location_revision + 1
		WHERE id = $11 AND world_location_revision = $12
		RETURNING world_location_revision
	`
	if !state.HasLocationRevision {
		query = `
			UPDATE players SET current_region = COALESCE(NULLIF($1, ''), current_region),
				world_zone_id = $2, world_instance_id = $3, world_x = $4, world_y = $5,
				world_safe_x = $6, world_safe_y = $7,
				x = CASE WHEN $8 THEN $9 ELSE x END, y = CASE WHEN $8 THEN $10 ELSE y END,
				world_location_revision = world_location_revision + 1
			WHERE id = $11 AND world_zone_id = '' AND world_location_revision = 0
			RETURNING world_location_revision
		`
	}
	args := []any{
		state.RegionID, state.ZoneID, state.InstanceID, state.X, state.Y,
		state.SafeX, state.SafeY, state.SyncLegacy, state.LegacyX, state.LegacyY,
		state.PlayerID,
	}
	if state.HasLocationRevision {
		args = append(args, state.LocationRevision)
	}
	err := exec.QueryRowContext(ctx, query, args...).Scan(&revision)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("save realtime world state: location revision is stale or missing for player %s", state.PlayerID)
		}
		return 0, fmt.Errorf("save realtime player location: %w", err)
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO player_world_state_v2 (player_id, hp, shield, map_version, content_version, updated_at)
		VALUES ($1,$2,$3,$4,$5,NOW())
		ON CONFLICT (player_id) DO UPDATE SET hp=EXCLUDED.hp, shield=EXCLUDED.shield,
			map_version=EXCLUDED.map_version, content_version=EXCLUDED.content_version, updated_at=NOW()
	`, state.PlayerID, state.HP, state.Shield, state.MapVersion, state.ContentVersion)
	if err != nil {
		return 0, fmt.Errorf("save realtime vitals checkpoint: %w", err)
	}
	combatJSON := []byte("{}")
	if state.Combat != nil {
		encoded, err := json.Marshal(state.Combat)
		if err != nil {
			return 0, fmt.Errorf("encode realtime combat state: %w", err)
		}
		combatJSON = encoded
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO player_world_state_combat (player_id, zone_id, instance_id, combat_state, updated_at)
		VALUES ($1,$2,$3,$4::jsonb,NOW())
		ON CONFLICT (player_id) DO UPDATE SET zone_id=EXCLUDED.zone_id,
			instance_id=EXCLUDED.instance_id, combat_state=EXCLUDED.combat_state, updated_at=NOW()
	`, state.PlayerID, state.ZoneID, state.InstanceID, combatJSON)
	if err != nil {
		return 0, fmt.Errorf("save realtime combat checkpoint: %w", err)
	}
	return revision, nil
}
