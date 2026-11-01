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
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// SkillUseKey identifies one regular Gu cast source for Chân Ý settlement.
// Path is the damage path resolved at admission, Rank is the Gu instance rank.
type SkillUseKey struct {
	Path string
	Rank int
}

// CombatUsage is the delta of combat actions accumulated since the previous
// durable checkpoint. Gu settles hunger, KillerMoves settle per-move mastery,
// and Skills/BasicAttacks settle path mastery (Chân Ý).
type CombatUsage struct {
	Gu           map[string]uint64
	KillerMoves  map[string]uint64
	Skills       map[SkillUseKey]uint64
	BasicAttacks uint64
}

type killerMoveGain struct {
	path string
	rank int
	uses uint64
}

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
	// UpdatedAt is when the vitals checkpoint was written. Reconnect uses it to
	// age the restored shield; zero means the caller has no timestamp (tests,
	// synthetic states) and the shield is restored as-is.
	UpdatedAt time.Time
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
		       COALESCE(c.updated_at, NOW()),
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
		&state.UpdatedAt, &state.LocationRevision, &combatJSON,
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
	return s.SaveAndApplyCombatDeltaRevision(ctx, state, delta, CombatUsage{})
}

// SaveAndApplyCombatDeltaRevision atomically settles the zone's mana and combat
// usage with the location/vitals checkpoint. Usage is additive so feeding or
// passive hunger decay during the session cannot be overwritten by a snapshot.
func (s *Service) SaveAndApplyCombatDeltaRevision(ctx context.Context, state State, delta float64, usage CombatUsage) (int64, error) {
	if err := state.Validate(); err != nil {
		return 0, err
	}
	if !finiteManaDelta(delta) {
		return 0, ErrInvalidState
	}
	for _, uses := range usage.Skills {
		if uses == 0 || uses > 1_000_000 {
			return 0, ErrInvalidState
		}
	}
	if usage.BasicAttacks > 1_000_000 {
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
	for instanceID, uses := range usage.Gu {
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
	killerGains := make([]killerMoveGain, 0, len(usage.KillerMoves))
	for moveID, uses := range usage.KillerMoves {
		if moveID == "" || uses == 0 || uses > 1_000_000 {
			return 0, ErrInvalidState
		}
		if _, parseErr := uuid.Parse(moveID); parseErr != nil {
			// Trận Kích / other non-killer-move ids never accrue mastery XP.
			continue
		}
		var movePath string
		var moveRank int
		err := tx.QueryRowContext(ctx, `
			UPDATE player_killer_moves
			SET mastery_xp = LEAST(5000, mastery_xp + $1), updated_at = NOW()
			WHERE id = $2::uuid AND player_id = $3
			RETURNING COALESCE(path, ''), COALESCE(main_gu_rank, 1)
		`, uses, moveID, state.PlayerID).Scan(&movePath, &moveRank)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("settle Sát Chiêu mastery for %s: %w", moveID, err)
		}
		killerGains = append(killerGains, killerMoveGain{path: movePath, rank: moveRank, uses: uses})
	}
	if err := settlePathMastery(ctx, tx, state.PlayerID, usage.Skills, usage.BasicAttacks, killerGains); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return revision, nil
}

// settlePathMastery accumulates Chân Ý from regular Gu casts, basic attacks and
// successful killer moves. It locks the player row so concurrent writers (for
// example an out-of-combat consumable) cannot lose points.
func settlePathMastery(ctx context.Context, tx *sql.Tx, playerID uuid.UUID, skills map[SkillUseKey]uint64, basicAttacks uint64, killerGains []killerMoveGain) error {
	if len(skills) == 0 && basicAttacks == 0 && len(killerGains) == 0 {
		return nil
	}
	merged := make(map[SkillUseKey]uint64, len(skills)+len(killerGains)+1)
	addUse := func(path string, rank int, uses uint64) {
		if uses == 0 {
			return
		}
		key := normalizeSkillUseKey(path, rank)
		merged[key] += uses
	}
	for _, gain := range killerGains {
		addUse(gain.path, gain.rank, gain.uses)
	}
	for key, uses := range skills {
		addUse(key.Path, key.Rank, uses)
	}
	if basicAttacks > 0 {
		addUse("strength", 1, basicAttacks)
	}
	if len(merged) == 0 {
		return nil
	}
	// Apply deterministic order: multiple sources on one path can land on
	// different ranks, and level efficiency makes the total order-dependent.
	keys := make([]SkillUseKey, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].Path != keys[j].Path {
			return keys[i].Path < keys[j].Path
		}
		return keys[i].Rank < keys[j].Rank
	})
	var raw []byte
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(path_masteries,'{}'::jsonb) FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&raw); err != nil {
		return fmt.Errorf("load path masteries: %w", err)
	}
	masteries := models.PathMasteriesMap{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &masteries); err != nil {
			return fmt.Errorf("decode path masteries: %w", err)
		}
	}
	if masteries == nil {
		masteries = models.PathMasteriesMap{}
	}
	changed := false
	for _, key := range keys {
		if applyPathMasteryUses(masteries, key.Path, key.Rank, merged[key]) {
			changed = true
		}
	}
	if !changed {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE players SET path_masteries=$1 WHERE id=$2`, masteries, playerID); err != nil {
		return fmt.Errorf("settle path masteries: %w", err)
	}
	return nil
}

func normalizeSkillUseKey(path string, rank int) SkillUseKey {
	cleanPath := gamefx.CleanPath(path)
	if cleanPath == "" {
		cleanPath = "strength"
	}
	if rank < 1 {
		rank = 1
	}
	return SkillUseKey{Path: cleanPath, Rank: rank}
}

// applyPathMasteryUses applies the current rank-based Chân Ý table once per
// use. Sequential application lets a long checkpoint chain cross a realm
// milestone with the correct efficiency, and stops as soon as the rank cap
// makes further gains impossible.
func applyPathMasteryUses(masteries models.PathMasteriesMap, path string, rank int, uses uint64) bool {
	if uses == 0 {
		return false
	}
	key := normalizeSkillUseKey(path, rank)
	baseGain := gamefx.PathMasteryGuUseBaseGain(key.Rank)
	if baseGain <= 0 {
		return false
	}
	changed := false
	for i := uint64(0); i < uses; i++ {
		points := masteries[key.Path]
		gain := gamefx.CalculatePathMasteryGain(key.Rank, gamefx.PathMasteryLevel(points), baseGain)
		if gain <= 0 {
			break
		}
		masteries[key.Path] = points + gain
		changed = true
	}
	return changed
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
