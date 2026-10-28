package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// RealtimeEquippedGu is one immutable equipped-Gu entry used to build the
// zone hotbar. Instance IDs are stable; the zone never writes them back itself.
type RealtimeEquippedGu struct {
	InstanceID  string
	GuID        string
	Rank        int
	Hunger      float64
	ManaCost    float64
	DamageScale float64
	ShieldScale float64
	BacklashPct float64
	Path        string
	Usable      bool
}

type RealtimeKillerMoveComponent struct {
	InstanceID string
	GuID       string
	Rank       int
	Hunger     float64
}

type RealtimeKillerMove struct {
	ID                   string
	Name                 string
	Rank                 int
	Path                 string
	Components           []RealtimeKillerMoveComponent
	ExpectedComponents   int
	Effects              []gameplayeffects.Block
	ManaCost             float64
	SuccessChance        float64
	BackfireHPPercent    float64
	BackfireStatus       string
	Complexity           float64
	DamageScale          float64
	RealtimeWindupMS     uint64
	RealtimeCooldownMS   uint64
	CreationMasteryLevel int
	MasteryXP            int
}

// RealtimeJoinState is loaded from the primary database for admission. It is
// deliberately separate from the cached UI snapshot used by the legacy client.
type RealtimeJoinState struct {
	Username                 string
	MapID                    string
	RegionID                 string
	MapInstanceID            string
	X                        int
	Y                        int
	WorldX                   float64
	WorldY                   float64
	WorldSafeX               float64
	WorldSafeY               float64
	WorldInstanceID          string
	WorldLocationRevision    int64
	HasWorldLocation         bool
	HasWorldLocationRevision bool
	HP                       float64
	BaseHP                   float64
	CurrentMana              float64
	MaxMana                  float64
	ManaRegenPerSecond       float64
	BaseAttack               float64
	Defense                  float64
	MovementSpeed            float64
	DaoProfile               *gamefx.DaoCombatProfile
	PathResistance           map[string]float64
	Rank                     int
	NeedsOnboarding          bool
	PartyID                  string
	EquippedGu               []RealtimeEquippedGu
	KillerMoves              []RealtimeKillerMove
}

func loadRealtimeKillerMoves(ctx context.Context, tx *sql.Tx, playerID uuid.UUID) ([]RealtimeKillerMove, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT id::text,name,main_gu_rank,path,components,compiled_effects,
			COALESCE((combat_profile->>'mana_cost')::double precision,0),
			success_chance,complexity,COALESCE(combat_profile->>'backfire_hp_pct','0'),
			COALESCE(combat_profile->>'backfire_status',''),
			COALESCE((combat_profile->>'realtime_windup_ms')::bigint,0),
			COALESCE((combat_profile->>'realtime_cooldown_ms')::bigint,0),
			creation_path_mastery_level,mastery_xp
		FROM player_killer_moves WHERE player_id=$1 AND is_equipped=true
		ORDER BY updated_at DESC,id LIMIT 4
	`, playerID)
	if err != nil {
		return nil, fmt.Errorf("load equipped realtime killer moves: %w", err)
	}
	type componentRef struct {
		InstanceID string `json:"instance_id"`
		GuID       string `json:"gu_id"`
		Rank       int    `json:"rank"`
	}
	type pendingComponents struct {
		moveIndex  int
		components []componentRef
	}
	moves := make([]RealtimeKillerMove, 0, 4)
	componentRefs := make([]pendingComponents, 0, 4)
	for rows.Next() {
		var move RealtimeKillerMove
		var rawComponents, rawEffects []byte
		if err := rows.Scan(&move.ID, &move.Name, &move.Rank, &move.Path, &rawComponents, &rawEffects,
			&move.ManaCost, &move.SuccessChance, &move.Complexity, &move.BackfireHPPercent, &move.BackfireStatus,
			&move.RealtimeWindupMS, &move.RealtimeCooldownMS, &move.CreationMasteryLevel, &move.MasteryXP); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan equipped realtime killer move: %w", err)
		}
		var components []componentRef
		if err := json.Unmarshal(rawComponents, &components); err != nil {
			rows.Close()
			return nil, fmt.Errorf("decode realtime killer move components: %w", err)
		}
		move.ExpectedComponents = len(components)
		componentRefs = append(componentRefs, pendingComponents{moveIndex: len(moves), components: components})
		if move.RealtimeWindupMS == 0 || move.RealtimeCooldownMS == 0 {
			move.RealtimeWindupMS, move.RealtimeCooldownMS = models.KillerMoveRealtimeTiming(move.Rank, move.Complexity)
		}
		if err := json.Unmarshal(rawEffects, &move.Effects); err != nil {
			rows.Close()
			return nil, fmt.Errorf("decode realtime killer move effects: %w", err)
		}
		moves = append(moves, move)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("read equipped realtime killer moves: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close equipped realtime killer moves: %w", err)
	}
	// database/sql transactions use one connection. Resolve component rows only
	// after closing the loadout cursor so the follow-up queries cannot wait on it.
	for _, pending := range componentRefs {
		move := &moves[pending.moveIndex]
		for _, component := range pending.components {
			var hunger float64
			var state string
			componentErr := tx.QueryRowContext(ctx, `
				SELECT hunger,state FROM player_gu WHERE id=$1 AND player_id=$2 AND gu_id=$3 AND gu_rank=$4
			`, component.InstanceID, playerID, component.GuID, component.Rank).Scan(&hunger, &state)
			if errors.Is(componentErr, sql.ErrNoRows) || strings.EqualFold(state, "DEAD") {
				move.Components = nil
				break
			}
			if componentErr != nil {
				return nil, fmt.Errorf("load realtime killer move component %s: %w", component.InstanceID, componentErr)
			}
			move.Components = append(move.Components, RealtimeKillerMoveComponent{
				InstanceID: component.InstanceID, GuID: component.GuID, Rank: component.Rank, Hunger: hunger,
			})
		}
	}
	return moves, nil
}

func LoadRealtimeJoinState(ctx context.Context, playerID uuid.UUID) (RealtimeJoinState, error) {
	if db.DB == nil {
		return RealtimeJoinState{}, errors.New("database not initialized")
	}
	var state RealtimeJoinState
	var equippedOrder models.EquippedGusList
	var regionType string
	err := db.DB.QueryRowContext(ctx, `
		SELECT COALESCE(NULLIF(p.world_zone_id, ''), rwm.world_map_id, 'world_map'), COALESCE(p.current_region, ''),
			COALESCE(r.zone_type, ''),
			COALESCE(p.map_instance_id::text, ''), p.x, p.y,
			p.world_x, p.world_y, p.world_safe_x, p.world_safe_y,
			p.world_instance_id, p.world_location_revision, p.world_zone_id <> '', p.hp, p.base_hp,
			p.needs_onboarding, COALESCE(p.party_id::text, ''), COALESCE(p.equipped_gus, '[]'::jsonb),
			COALESCE(p.username, '')
		FROM players p
		LEFT JOIN region_world_maps rwm ON rwm.region_id = p.current_region
		LEFT JOIN regions r ON r.id = p.current_region
		WHERE p.id = $1
	`, playerID).Scan(
		&state.MapID, &state.RegionID, &regionType, &state.MapInstanceID, &state.X, &state.Y,
		&state.WorldX, &state.WorldY, &state.WorldSafeX, &state.WorldSafeY,
		&state.WorldInstanceID, &state.WorldLocationRevision, &state.HasWorldLocation,
		&state.HP, &state.BaseHP, &state.NeedsOnboarding, &state.PartyID, &equippedOrder,
		&state.Username,
	)
	if err != nil {
		return RealtimeJoinState{}, err
	}
	state.HasWorldLocationRevision = true
	if !state.HasWorldLocation || state.RegionID == "aperture_tribulation" {
		state.MapID = realtimeMapID(state.RegionID, regionType, state.MapID)
	}
	// Project the player resources and passive combat stats used by realtime
	// admission. The equipped-only service keeps large inventories out of
	// realtime admission while preserving mana and Gu effect rules.
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return RealtimeJoinState{}, err
	}
	defer func() { _ = tx.Rollback() }()
	player, err := PlayerSvc.EvaluateAndPersistManaEquipped(tx, playerID)
	if err != nil {
		return RealtimeJoinState{}, err
	}
	equipped, err := loadRealtimeEquipped(ctx, tx, playerID, equippedOrder)
	if err != nil {
		return RealtimeJoinState{}, err
	}
	state.EquippedGu = equipped
	killerMoves, err := loadRealtimeKillerMoves(ctx, tx, playerID)
	if err != nil {
		return RealtimeJoinState{}, err
	}
	state.KillerMoves = killerMoves
	if err := tx.Commit(); err != nil {
		return RealtimeJoinState{}, err
	}
	state.CurrentMana = player.CurrentMana
	state.MaxMana = gamefx.EffectiveMaxMana(&player)
	// Legacy combat does not grant passive world-time recovery during actions.
	// PlayerSvc projects elapsed recovery on admission instead; the world pool
	// spends mana until the character next leaves and re-enters.
	state.ManaRegenPerSecond = 0
	state.Rank = player.Rank
	stats := CalculatePlayerStats(&player)
	state.BaseAttack = stats["attack"]
	state.Defense = stats["defense"]
	state.MovementSpeed = RealtimeMovementSpeed(stats["movement"])
	fx := gamefx.Get(&player)
	dao := gamefx.BuildDaoCombatProfileWithReduction(player.DaoMarks, player.PathMasteries, fx.NonConflictingDaoMarks, fx.DaoConflictReductionPct, fx.DaoNganBonus)
	for i := range state.KillerMoves {
		move := &state.KillerMoves[i]
		currentMastery := gamefx.PathMasteryLevel(player.PathMasteries[gamefx.CleanPath(move.Path)])
		move.SuccessChance = killerMoveAdjustedChance(move.SuccessChance, move.CreationMasteryLevel, currentMastery, move.MasteryXP)
		amp, efficiency, conflict := dao.GetOffensiveFactors(move.Path)
		move.DamageScale = (1 + (amp-1)*efficiency) * conflict * gamefx.PathDamageMultiplierForPaths([]string{move.Path}, fx.PathDamageBonus) * gamefx.PathMasteryMultiplier(player.PathMasteries, []string{move.Path})
		if move.DamageScale <= 0 {
			move.DamageScale = 1
		}
	}
	basicAmp, basicEfficiency, basicConflict := dao.GetOffensiveFactors("strength")
	state.BaseAttack *= (1 + (basicAmp-1)*basicEfficiency) * basicConflict * gamefx.PathDamageMultiplierForPaths(nil, fx.PathDamageBonus)
	state.DaoProfile = dao
	state.PathResistance = make(map[string]float64, len(fx.PathResistanceBonus))
	for path, value := range fx.PathResistanceBonus {
		state.PathResistance[path] = value
	}
	for _, instance := range models.FilterEquippedActive(player.GuInventory) {
		if spec, ok := models.GetGu(instance.GuID, instance.Rank); ok {
			for path, value := range models.GuPathResistance(spec) {
				state.PathResistance[path] += value
			}
		}
	}
	for i := range state.EquippedGu {
		entry := &state.EquippedGu[i]
		spec, ok := models.GetGu(entry.GuID, entry.Rank)
		if !ok {
			spec, ok = models.GetGuAny(entry.GuID)
		}
		if !ok || entry.Hunger < 1 || !gamefx.CanUseGuWithMana(player.Rank, &spec) {
			continue
		}
		entry.Usable = true
		path := "strength"
		damagePaths := spec.Path
		if len(spec.Path) > 0 {
			path = spec.Path[0]
		}
		if spec.Effects.Active != nil && spec.Effects.Active.DamagePath != "" {
			path = spec.Effects.Active.DamagePath
			damagePaths = []string{path}
		} else {
			for _, block := range spec.Effects.Blocks {
				if block.Trigger != gameplayeffects.TriggerOnAction {
					continue
				}
				for _, op := range block.Operations {
					if op.Op == gameplayeffects.OpDamage && op.Path != "" {
						path = op.Path
						damagePaths = []string{path}
					}
				}
			}
		}
		entry.Path = path
		amp, efficiency, conflict := dao.GetOffensiveFactors(path)
		entry.ManaCost = gamefx.EnergyCost(&player, &spec) * gamefx.ManaTaxMultiplier(conflict)
		entry.BacklashPct = gamefx.HigherRankHPCost(entry.Rank, player.Rank, 1)
		skillMultiplier := gamefx.PathMasteryMultiplier(player.PathMasteries, spec.Path)
		if spec.Effects.Active != nil && spec.Effects.Active.DamageMult > 0 {
			skillMultiplier += spec.Effects.Active.DamageMult - 1
		} else {
			for _, block := range spec.Effects.Blocks {
				if block.Trigger != gameplayeffects.TriggerOnAction {
					continue
				}
				for _, op := range block.Operations {
					if op.Op == gameplayeffects.OpDamage && strings.EqualFold(op.SubType, "MULTIPLIER") && op.Value > 0 {
						skillMultiplier += op.Value - 1
					}
				}
			}
		}
		entry.DamageScale = skillMultiplier *
			(1 + (amp-1)*efficiency) * conflict * gamefx.PathDamageMultiplierForPaths(damagePaths, fx.PathDamageBonus)
		if entry.DamageScale <= 0 {
			entry.DamageScale = 1
		}
		// Apply the existing self-shield scaling: path attainment and the
		// conflict-adjusted Dao bonus. Conflict cannot reduce the base shield;
		// it only reduces the additional Dao amplification. Path damage bonuses
		// remain damage-only.
		shieldDaoMult := (1 + (amp-1)*efficiency) * conflict
		if shieldDaoMult < 1 {
			shieldDaoMult = 1
		}
		entry.ShieldScale = gamefx.PathMasteryMultiplier(player.PathMasteries, spec.Path) * shieldDaoMult
	}
	return state, nil
}

// RealtimeMovementSpeed converts the passive movement stat into authoritative
// world units per second. Movement has a baseline of 1; each point above it
// adds five percent to the normal realtime player speed.
func RealtimeMovementSpeed(movement float64) float64 {
	bonus := movement - 1
	if bonus < 0 {
		bonus = 0
	}
	return 4 * (1 + bonus*0.05)
}

// realtimeMapID makes detailed KHU_VUC/SAFE_ZONE/TRIBULATION maps authoritative for MMO
// admission even while an older region_world_maps assignment still points at
// the outer NGU_VUC world map. Parent and special regions keep their explicit
// map assignment.
func realtimeMapID(regionID, zoneType, assignedMapID string) string {
	switch zoneType {
	case "KHU_VUC", "SAFE_ZONE", "TRIBULATION":
		if regionID != "" {
			return regionID
		}
	}
	if assignedMapID != "" {
		return assignedMapID
	}
	return "world_map"
}

func loadRealtimeEquipped(ctx context.Context, queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, playerID uuid.UUID, order models.EquippedGusList) ([]RealtimeEquippedGu, error) {
	rows, err := queryer.QueryContext(ctx, `
			SELECT id, gu_id, COALESCE(NULLIF(gu_rank, 0), rank, 1), hunger
		FROM player_gu
		WHERE player_id = $1 AND equipped = true AND state = 'ACTIVE'
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[string]RealtimeEquippedGu{}
	for rows.Next() {
		var entry RealtimeEquippedGu
		if err := rows.Scan(&entry.InstanceID, &entry.GuID, &entry.Rank, &entry.Hunger); err != nil {
			return nil, err
		}
		byID[entry.InstanceID] = entry
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	ordered := make([]RealtimeEquippedGu, 0, len(byID))
	seen := map[string]bool{}
	for _, instanceID := range order {
		if entry, ok := byID[instanceID]; ok && !seen[instanceID] {
			ordered = append(ordered, entry)
			seen[instanceID] = true
		}
	}
	remaining := make([]RealtimeEquippedGu, 0, len(byID)-len(ordered))
	for instanceID, entry := range byID {
		if !seen[instanceID] {
			remaining = append(remaining, entry)
		}
	}
	sort.Slice(remaining, func(i, j int) bool { return remaining[i].InstanceID < remaining[j].InstanceID })
	ordered = append(ordered, remaining...)
	if len(ordered) > 6 {
		ordered = ordered[:6]
	}
	return ordered, nil
}
