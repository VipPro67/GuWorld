package handlers

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
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
	HouseStrike          bool
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
	PvPEnabled               bool
	EquippedGu               []RealtimeEquippedGu
	KillerMoves              []RealtimeKillerMove
	Companions               []RealtimeCompanion
}

type RealtimeCompanion struct {
	ID        string
	ActorKind string
	Name      string
	Rank      int
	HP        float64
	MaxHP     float64
	Attack    float64
	Defense   float64
	Speed     float64
	Loyalty   float64
}

// ProjectRealtimeEffects converts the character loadout row projection into
// the realtime effects engine input. Realtime hotbar/loadout compilation and
// gateway admission share this single mapping so neither side can drift from
// the other's slot semantics.
func (move RealtimeKillerMove) ProjectRealtimeEffects() realtimeeffects.KillerMove {
	components := make([]realtimeeffects.KillerMoveComponent, 0, len(move.Components))
	for _, component := range move.Components {
		components = append(components, realtimeeffects.KillerMoveComponent{InstanceID: component.InstanceID, Hunger: component.Hunger})
	}
	return realtimeeffects.KillerMove{
		ID: move.ID, Name: move.Name, Rank: move.Rank, Components: components,
		ExpectedComponents: move.ExpectedComponents, Effects: move.Effects, ManaCost: move.ManaCost,
		SuccessChance: move.SuccessChance, BackfireHPPercent: move.BackfireHPPercent,
		BackfireStatus: move.BackfireStatus, Complexity: move.Complexity, DamageScale: move.DamageScale,
		WindupMS: move.RealtimeWindupMS, CooldownMS: move.RealtimeCooldownMS, HouseStrike: move.HouseStrike,
	}
}

func LoadRealtimeJoinState(ctx context.Context, playerID uuid.UUID) (RealtimeJoinState, error) {
	if db.DB == nil {
		return RealtimeJoinState{}, errors.New("database not initialized")
	}
	var state RealtimeJoinState
	var equippedOrder models.EquippedGusList
	var hotbarSetting models.RealtimeHotbarList
	var combatSettings models.AutoCombatConfig
	var regionType string
	err := db.DB.QueryRowContext(ctx, `
		SELECT COALESCE(NULLIF(p.world_zone_id, ''), rwm.world_map_id, 'world_map'), COALESCE(p.current_region, ''),
			COALESCE(r.zone_type, ''),
			COALESCE(p.map_instance_id::text, ''), p.x, p.y,
			p.world_x, p.world_y, p.world_safe_x, p.world_safe_y,
			p.world_instance_id, p.world_location_revision, p.world_zone_id <> '', p.hp, p.base_hp,
			p.needs_onboarding, COALESCE(p.party_id::text, ''), COALESCE(p.equipped_gus, '[]'::jsonb),
			COALESCE(p.realtime_hotbar, '[]'::jsonb),
			COALESCE(p.username, ''), COALESCE(p.combat_settings, '{}'::jsonb)
		FROM players p
		LEFT JOIN region_world_maps rwm ON rwm.region_id = p.current_region
		LEFT JOIN regions r ON r.id = p.current_region
		WHERE p.id = $1
	`, playerID).Scan(
		&state.MapID, &state.RegionID, &regionType, &state.MapInstanceID, &state.X, &state.Y,
		&state.WorldX, &state.WorldY, &state.WorldSafeX, &state.WorldSafeY,
		&state.WorldInstanceID, &state.WorldLocationRevision, &state.HasWorldLocation,
		&state.HP, &state.BaseHP, &state.NeedsOnboarding, &state.PartyID, &equippedOrder,
		&hotbarSetting,
		&state.Username, &combatSettings,
	)
	if err != nil {
		return RealtimeJoinState{}, err
	}
	state.PvPEnabled = combatSettings.PvPEnabled
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
	loadout, err := LoadCombatLoadout(ctx, tx, playerID, equippedOrder, hotbarSetting)
	if err != nil {
		return RealtimeJoinState{}, err
	}
	state.EquippedGu = loadout.EquippedGu
	state.KillerMoves = loadout.KillerMoves
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

	compRows, compErr := db.DB.QueryContext(ctx, `
		SELECT id, actor_kind, name, rank, hp, max_hp, base_attack, base_defense, speed, loyalty
		FROM non_player_actors
		WHERE owner_player_id = $1
		  AND actor_kind IN ('COMPANION_BEAST', 'VARIANT_NPC')
		  AND lifecycle_state = 'ACTIVE'
		  AND execution_state != 'INJURED'
		  AND formation_slot > 0
		ORDER BY formation_slot ASC
		LIMIT 3
	`, playerID)
	if compErr == nil {
		defer compRows.Close()
		for compRows.Next() {
			var comp RealtimeCompanion
			if err := compRows.Scan(
				&comp.ID, &comp.ActorKind, &comp.Name, &comp.Rank,
				&comp.HP, &comp.MaxHP, &comp.Attack, &comp.Defense, &comp.Speed, &comp.Loyalty,
			); err == nil {
				state.Companions = append(state.Companions, comp)
			}
		}
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

// realtimeGuRow is one player_gu row relevant to hotbar composition.
type realtimeGuRow struct {
	entry RealtimeEquippedGu
	// valid means equipped + ACTIVE + alive durability: this Gu may occupy a
	// hotbar slot right now.
	valid bool
}

// HotbarSlots is the number of realtime hotbar slots (1..6). Aliased to the
// effects compiler constant so REST validation, loadout composition and the
// gateway share one limit.
const HotbarSlots = realtimeeffects.MaxHotbarSlots

// orderRealtimeHotbar composes the hotbar slot entries with the explicit
// realtime_hotbar assignment:
//   - a pinned slot keeps its static position. A pin on an owned Gu keeps the
//     slot reserved while the Gu is invalid (sleeping, damaged, unequipped) so
//     that the Gu reclaims its slot automatically; a pin on a Gu the player no
//     longer owns is released and the slot becomes fillable;
//   - Gu not pinned anywhere (equipped + ACTIVE + valid) fill the remaining
//     slots in hotbarOrder order, then any leftover valid ids by instance id;
//   - the result never exceeds HotbarSlots entries and trailing unoccupied
//     slots are trimmed. Interior unoccupied slots stay as zero placeholders
//     so later pinned slots are never renumbered.
//
// Invalid entries (reserved pins) are carried as ID-only placeholders (empty
// GU) so the compiler neither resolves a profile nor leaks form skills from a
// Gu that currently cannot act.
func orderRealtimeHotbar(hotbar models.RealtimeHotbarList, order models.EquippedGusList, gus map[string]realtimeGuRow) []RealtimeEquippedGu {
	slots := make([]RealtimeEquippedGu, HotbarSlots)
	occupied := make([]bool, HotbarSlots)
	consumed := make(map[string]bool, len(gus))
	for i := 0; i < len(hotbar) && i < HotbarSlots; i++ {
		id := hotbar[i]
		if id == "" || consumed[id] {
			continue
		}
		row, ok := gus[id]
		if !ok {
			// Gu no longer belongs to the player: drop the pin.
			continue
		}
		consumed[id] = true
		occupied[i] = true
		if row.valid {
			slots[i] = row.entry
		} else {
			// Reserved but currently unable to act: keep the slot static with
			// an ID-only placeholder the resolver will treat as unsupported.
			slots[i] = RealtimeEquippedGu{InstanceID: id}
		}
	}

	remaining := make([]RealtimeEquippedGu, 0, len(gus))
	for _, id := range order {
		if len(remaining) >= HotbarSlots {
			break
		}
		if consumed[id] {
			continue
		}
		row, ok := gus[id]
		if !ok || !row.valid {
			continue
		}
		consumed[id] = true
		remaining = append(remaining, row.entry)
	}
	leftover := make([]string, 0, len(gus))
	for id, row := range gus {
		if !consumed[id] && row.valid {
			leftover = append(leftover, id)
		}
	}
	sort.Strings(leftover)
	for _, id := range leftover {
		if len(remaining) >= HotbarSlots {
			break
		}
		remaining = append(remaining, gus[id].entry)
	}

	next := 0
	for i := range slots {
		if occupied[i] {
			continue
		}
		if next >= len(remaining) {
			break
		}
		slots[i] = remaining[next]
		occupied[i] = true
		next++
	}

	// Trim only trailing unoccupied slots so interior holes never renumber the
	// pinned slots that follow them.
	last := -1
	for i := range slots {
		if occupied[i] {
			last = i
		}
	}
	return slots[:last+1]
}

func loadRealtimeEquipped(ctx context.Context, queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, playerID uuid.UUID, order models.EquippedGusList, hotbar models.RealtimeHotbarList) ([]RealtimeEquippedGu, error) {
	rows, err := queryer.QueryContext(ctx, `
		SELECT id, gu_id, COALESCE(NULLIF(gu_rank, 0), rank, 1), hunger, equipped, state, durability
		FROM player_gu
		WHERE player_id = $1
	`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	gus := make(map[string]realtimeGuRow)
	for rows.Next() {
		var entry RealtimeEquippedGu
		var equipped bool
		var state string
		var durability float64
		if err := rows.Scan(&entry.InstanceID, &entry.GuID, &entry.Rank, &entry.Hunger, &equipped, &state, &durability); err != nil {
			return nil, err
		}
		gus[entry.InstanceID] = realtimeGuRow{
			entry: entry,
			valid: equipped && state == "ACTIVE" && durability > 0,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orderRealtimeHotbar(hotbar, order, gus), nil
}
