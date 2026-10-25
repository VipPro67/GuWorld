// Package persistence stores durable realtime world checkpoints. Zone ticks
// never call it; join/leave and a coarse periodic worker are the only writers.
package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
)

type State struct {
	PlayerID       uuid.UUID
	ZoneID         string
	InstanceID     string
	X              float64
	Y              float64
	HP             float64
	Shield         float64
	SafeX          float64
	SafeY          float64
	MapVersion     string
	ContentVersion string
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
	err := s.db.QueryRowContext(ctx, `
		SELECT player_id, zone_id, instance_id, x, y, hp, shield, safe_x, safe_y,
		       map_version, content_version
		FROM player_world_state_v2
		WHERE player_id = $1
	`, playerID).Scan(
		&state.PlayerID, &state.ZoneID, &state.InstanceID, &state.X, &state.Y,
		&state.HP, &state.Shield, &state.SafeX, &state.SafeY,
		&state.MapVersion, &state.ContentVersion,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return State{}, false, nil
	}
	if err != nil {
		return State{}, false, err
	}
	return state, true, nil
}

func (s *Service) Save(ctx context.Context, state State) error {
	if err := state.Validate(); err != nil {
		return err
	}
	return upsert(ctx, s.db, state)
}

// SaveAndApplyManaDelta commits position/HP and the zone's newly spent mana
// together. A failed write leaves both unchanged for the next checkpoint.
func (s *Service) SaveAndApplyManaDelta(ctx context.Context, state State, delta float64) error {
	if err := state.Validate(); err != nil {
		return err
	}
	if !finiteManaDelta(delta) {
		return ErrInvalidState
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := upsert(ctx, tx, state); err != nil {
		return err
	}
	if delta != 0 {
		if err := applyManaDelta(ctx, tx, state.PlayerID, delta); err != nil {
			return err
		}
	}
	return tx.Commit()
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
		if err := upsert(ctx, tx, state); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func upsert(ctx context.Context, exec execer, state State) error {
	_, err := exec.ExecContext(ctx, `
		INSERT INTO player_world_state_v2
			(player_id, zone_id, instance_id, x, y, hp, shield, safe_x, safe_y, map_version, content_version, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NOW())
		ON CONFLICT (player_id) DO UPDATE SET
			zone_id=EXCLUDED.zone_id, instance_id=EXCLUDED.instance_id,
			x=EXCLUDED.x, y=EXCLUDED.y, hp=EXCLUDED.hp, shield=EXCLUDED.shield,
			safe_x=EXCLUDED.safe_x, safe_y=EXCLUDED.safe_y,
			map_version=EXCLUDED.map_version, content_version=EXCLUDED.content_version,
			updated_at=NOW()
	`, state.PlayerID, state.ZoneID, state.InstanceID, state.X, state.Y, state.HP, state.Shield,
		state.SafeX, state.SafeY, state.MapVersion, state.ContentVersion)
	if err != nil {
		return fmt.Errorf("save realtime world state: %w", err)
	}
	return nil
}
