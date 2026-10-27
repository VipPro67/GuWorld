package effects

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

const ProfileSchemaVersion = 1

const (
	MaxProfileFileBytes = 16 << 20
	MaxWindupMS         = 5_000
	MaxRecoveryMS       = 5_000
	MaxCooldownMS       = 120_000
	MaxStatusDurationMS = 30_000
	MinStatusDurationMS = 100
)

type ProfileCatalog struct {
	SchemaVersion int         `json:"schema_version"`
	Profiles      []GuProfile `json:"profiles"`
}

type GuProfile struct {
	GUID           string           `json:"gu_id"`
	Rank           int              `json:"rank"`
	Slot           int              `json:"slot,omitempty"`
	Active         *ActiveProfile   `json:"active"`
	Passives       []PassiveProfile `json:"passives,omitempty"`
	HiddenPassives []PassiveProfile `json:"hidden_passives,omitempty"`
	Visuals        VisualProfile    `json:"visuals"`
}

type ActiveProfile struct {
	Mode              combat.Mode        `json:"mode"`
	Shape             combat.Shape       `json:"shape,omitempty"`
	AngleDeg          float64            `json:"angle_deg,omitempty"`
	LengthUnits       float64            `json:"length_units,omitempty"`
	WidthUnits        float64            `json:"width_units,omitempty"`
	InnerRadius       float64            `json:"inner_radius,omitempty"`
	WindupMS          uint64             `json:"windup_ms"`
	RecoveryMS        uint64             `json:"recovery_ms"`
	CooldownMS        uint64             `json:"cooldown_ms"`
	RangeUnits        float64            `json:"range_units"`
	ManaCost          float64            `json:"mana_cost"`
	MovementPolicy    string             `json:"movement_policy"`
	RadiusUnits       float64            `json:"radius_units,omitempty"`
	MaxTargets        int                `json:"max_targets,omitempty"`
	IsToggle          bool               `json:"is_toggle,omitempty"`
	MoveDistance      float64            `json:"move_distance,omitempty"`
	KnockbackDistance float64            `json:"knockback_distance,omitempty"`
	PlayerAtkRatio    float64            `json:"player_atk_ratio,omitempty"`
	GuPowerRatio      float64            `json:"gu_power_ratio,omitempty"`
	Projectile        *ProjectileProfile `json:"projectile,omitempty"`
	Effects           []EffectProfile    `json:"effects"`
}

type ProjectileProfile struct {
	SpeedUnitsPerSec float64 `json:"speed_units_per_sec"`
	RadiusUnits      float64 `json:"radius_units"`
	LifetimeTicks    uint64  `json:"lifetime_ticks"`
	PierceCount      int     `json:"pierce_count,omitempty"`
	SpreadCount      int     `json:"spread_count,omitempty"`
	SpreadAngleDeg   float64 `json:"spread_angle_deg,omitempty"`
}

type PassiveProfile struct {
	Trigger string          `json:"trigger"`
	Effects []EffectProfile `json:"effects"`
}

type EffectProfile struct {
	Type               string   `json:"type"`
	Value              float64  `json:"value,omitempty"`
	Stat               string   `json:"stat,omitempty"`
	StatusID           string   `json:"status_id,omitempty"`
	DurationMS         uint64   `json:"duration_ms,omitempty"`
	PeriodMS           uint64   `json:"period_ms,omitempty"`
	TickValue          float64  `json:"tick_value,omitempty"`
	MoveMultiplier     *float64 `json:"move_multiplier,omitempty"`
	MaxStacks          int      `json:"max_stacks,omitempty"`
	StackPolicy        string   `json:"stack_policy,omitempty"`
	Debuff             *bool    `json:"debuff,omitempty"`
	PvPDrCategory      string   `json:"pvp_dr_category,omitempty"`
	ManaDrainPerSec    float64  `json:"mana_drain_per_sec,omitempty"`
	ThoughtDrainPerSec float64  `json:"thought_drain_per_sec,omitempty"`
	IsToggle           bool     `json:"is_toggle,omitempty"`
}

type VisualProfile struct {
	Icon          string `json:"icon,omitempty"`
	CastVFX       string `json:"cast_vfx,omitempty"`
	ProjectileVFX string `json:"projectile_vfx,omitempty"`
	ImpactVFX     string `json:"impact_vfx,omitempty"`
}

type CompiledActive struct {
	Skill          combat.Skill
	RecoveryTicks  uint64
	MovementPolicy string
	Effects        []EffectProfile
}

type CompiledPassive struct {
	Trigger string
	Effects []EffectProfile
}

type CompiledProfile struct {
	GUID           string
	Rank           int
	Slot           int
	Active         *CompiledActive
	Passives       []CompiledPassive
	HiddenPassives []CompiledPassive
	Visuals        VisualProfile
	SourceHash     string
}

type CompiledCatalog struct {
	SchemaVersion int
	Profiles      []CompiledProfile
	DefaultSkills []combat.Skill
	Hash          string
}

func (c *CompiledCatalog) Profile(guID string, rank int) (CompiledProfile, bool) {
	for _, profile := range c.Profiles {
		if strings.EqualFold(profile.GUID, guID) && profile.Rank == rank {
			return profile, true
		}
	}
	return CompiledProfile{}, false
}

// LoadGuCSV loads the canonical entries from Gu.csv so the realtime compiler
// can validate profiles directly against the primary CSV without depending on DB state.
func LoadGuCSV(path string) ([]GuEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	columns := map[string]int{}
	for index, name := range header {
		columns[name] = index
	}
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	entries := make([]GuEntry, 0, len(rows))
	for _, row := range rows {
		rank, _ := strconv.Atoi(row[columns["rank"]])
		mana, _ := strconv.ParseFloat(row[columns["base_mana_cost"]], 64)
		entries = append(entries, GuEntry{
			ID: row[columns["id"]], Name: row[columns["name"]], Rank: rank, Type: row[columns["gu_type"]],
			PrimaryRole: row[columns["primary_role"]], ManaCost: mana,
			Effects: json.RawMessage(row[columns["effects"]]),
		})
	}
	return entries, nil
}

func LoadProfiles(path string) (ProfileCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProfileCatalog{}, err
	}
	if len(data) == 0 || len(data) > MaxProfileFileBytes {
		return ProfileCatalog{}, fmt.Errorf("profile file must be 1..%d bytes", MaxProfileFileBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var catalog ProfileCatalog
	if err := decoder.Decode(&catalog); err != nil {
		return ProfileCatalog{}, fmt.Errorf("decode profiles: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return ProfileCatalog{}, fmt.Errorf("decode profiles: multiple JSON values")
		}
		return ProfileCatalog{}, fmt.Errorf("decode profiles: %w", err)
	}
	return catalog, nil
}

// LoadProfileCatalogs loads one or more comma-separated profile catalogs and
// merges them in order. Used by startup/CLI for MVP + Alpha content packs.
func LoadProfileCatalogs(paths string) (ProfileCatalog, error) {
	parts := strings.Split(paths, ",")
	merged := ProfileCatalog{SchemaVersion: ProfileSchemaVersion}
	for _, raw := range parts {
		path := strings.TrimSpace(raw)
		if path == "" {
			continue
		}
		catalog, err := LoadProfiles(path)
		if err != nil {
			return ProfileCatalog{}, err
		}
		if catalog.SchemaVersion != merged.SchemaVersion {
			return ProfileCatalog{}, fmt.Errorf("profile catalog %s has schema_version %d, want %d", path, catalog.SchemaVersion, ProfileSchemaVersion)
		}
		merged.Profiles = append(merged.Profiles, catalog.Profiles...)
	}
	if len(merged.Profiles) == 0 {
		return ProfileCatalog{}, fmt.Errorf("no realtime profile catalogs loaded")
	}
	return merged, nil
}

// CompileCatalog validates realtime profiles against the Gu catalog and
// produces immutable skills. requiredMVP entries must all have a profile.
func CompileCatalog(catalog ProfileCatalog, entries []GuEntry, requiredMVP []EntryRef) (*CompiledCatalog, error) {
	if catalog.SchemaVersion != ProfileSchemaVersion {
		return nil, fmt.Errorf("unsupported realtime profile schema_version %d", catalog.SchemaVersion)
	}
	if len(catalog.Profiles) == 0 {
		return nil, fmt.Errorf("realtime profile catalog is empty")
	}
	entryIndex := map[string]GuEntry{}
	for _, entry := range entries {
		entryIndex[refKey(entry.ID, entry.Rank)] = entry
	}
	mvpSet := map[string]bool{}
	for _, ref := range requiredMVP {
		mvpSet[refKey(ref.ID, ref.Rank)] = true
	}
	provided := map[string]bool{}
	slots := map[int]string{}
	compiled := &CompiledCatalog{SchemaVersion: ProfileSchemaVersion}
	seen := map[string]bool{}
	for index, profile := range catalog.Profiles {
		path := fmt.Sprintf("profiles[%d]", index)
		key := refKey(profile.GUID, profile.Rank)
		if profile.GUID == "" || profile.Rank < 1 || profile.Rank > 10 {
			return nil, fmt.Errorf("%s: gu_id and rank 1..10 are required", path)
		}
		if seen[key] {
			return nil, fmt.Errorf("%s: duplicate profile %s rank %d", path, profile.GUID, profile.Rank)
		}
		seen[key] = true
		provided[key] = true
		entry, exists := entryIndex[key]
		if len(entries) > 0 && !exists {
			return nil, fmt.Errorf("%s: %s rank %d does not exist in loaded Gu registry (database gu_dictionary or Gu.csv)", path, profile.GUID, profile.Rank)
		}
		if profile.Slot != 0 {
			if profile.Slot < 1 || profile.Slot > 6 {
				return nil, fmt.Errorf("%s: slot must be 1..6", path)
			}
			if existing, ok := slots[profile.Slot]; ok && profile.Active != nil {
				return nil, fmt.Errorf("%s: slot %d is already used by %s", path, profile.Slot, existing)
			}
			slots[profile.Slot] = profile.GUID
		}
		compiledProfile := CompiledProfile{
			GUID: profile.GUID, Rank: profile.Rank, Slot: profile.Slot, Visuals: profile.Visuals,
		}
		if len(entry.Effects) > 0 {
			hash := sha256.Sum256(entry.Effects)
			compiledProfile.SourceHash = hex.EncodeToString(hash[:])
		}
		if profile.Active != nil {
			active, err := compileActive(path+".active", *profile.Active, profile.Slot, profile.GUID, entry)
			if err != nil {
				return nil, err
			}
			compiledProfile.Active = active
			if profile.Slot > 0 {
				compiled.DefaultSkills = append(compiled.DefaultSkills, active.Skill)
			}
		} else if profile.Slot != 0 {
			return nil, fmt.Errorf("%s: slot requires an active block", path)
		}
		for passiveIndex, passive := range profile.Passives {
			compiledPassive, err := compilePassive(fmt.Sprintf("%s.passives[%d]", path, passiveIndex), passive)
			if err != nil {
				return nil, err
			}
			compiledProfile.Passives = append(compiledProfile.Passives, compiledPassive)
		}
		for passiveIndex, passive := range profile.HiddenPassives {
			compiledPassive, err := compilePassive(fmt.Sprintf("%s.hidden_passives[%d]", path, passiveIndex), passive)
			if err != nil {
				return nil, err
			}
			compiledProfile.HiddenPassives = append(compiledProfile.HiddenPassives, compiledPassive)
		}
		compiled.Profiles = append(compiled.Profiles, compiledProfile)
	}
	for key := range mvpSet {
		if !provided[key] {
			return nil, fmt.Errorf("MVP Gu %s has no realtime profile", strings.ReplaceAll(key, "\x00", " rank "))
		}
	}
	sort.Slice(compiled.Profiles, func(i, j int) bool {
		if compiled.Profiles[i].GUID != compiled.Profiles[j].GUID {
			return compiled.Profiles[i].GUID < compiled.Profiles[j].GUID
		}
		return compiled.Profiles[i].Rank < compiled.Profiles[j].Rank
	})
	sort.Slice(compiled.DefaultSkills, func(i, j int) bool { return compiled.DefaultSkills[i].Slot < compiled.DefaultSkills[j].Slot })
	serialized, err := json.Marshal(compiled.Profiles)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(serialized)
	compiled.Hash = hex.EncodeToString(hash[:])
	return compiled, nil
}

func compileActive(path string, active ActiveProfile, slot int, guID string, entry GuEntry) (*CompiledActive, error) {
	if slot < 0 || slot > 6 {
		return nil, fmt.Errorf("%s: active profile slot must be 0 (catalog-only) or 1..6", path)
	}
	skillSlot := slot
	if skillSlot == 0 {
		// Catalog-only profiles are re-slotted from the owned loadout in PR-23.
		skillSlot = 1
	}
	switch active.Mode {
	case combat.ModeTarget, combat.ModeDirection, combat.ModeGround, combat.ModeSelf:
	default:
		return nil, fmt.Errorf("%s: unsupported mode %q", path, active.Mode)
	}
	if active.WindupMS > MaxWindupMS || active.RecoveryMS > MaxRecoveryMS || active.CooldownMS > MaxCooldownMS {
		return nil, fmt.Errorf("%s: timing exceeds windup/recovery/cooldown limits", path)
	}
	if active.ManaCost < 0 || !finiteFloat(active.ManaCost) || !finiteFloat(active.RangeUnits) {
		return nil, fmt.Errorf("%s: mana/range must be finite and non-negative", path)
	}
	if active.RangeUnits > combat.MaxCastRangeUnits {
		return nil, fmt.Errorf("%s: range exceeds %v world units", path, combat.MaxCastRangeUnits)
	}
	if active.Mode != combat.ModeSelf && active.RangeUnits <= 0 {
		return nil, fmt.Errorf("%s: range_units is required for %s", path, active.Mode)
	}
	if len(active.Effects) == 0 {
		return nil, fmt.Errorf("%s: at least one effect is required", path)
	}
	skill := combat.Skill{
		ID: guID, Slot: skillSlot, Mode: active.Mode, Name: entry.Name,
		Shape: active.Shape, AngleDeg: active.AngleDeg, LengthUnits: active.LengthUnits,
		WidthUnits: active.WidthUnits, InnerRadius: active.InnerRadius,
		ManaCost: active.ManaCost, Range: active.RangeUnits,
		WindupTicks: msToTicks(active.WindupMS), CooldownTicks: msToTicks(active.CooldownMS),
		MoveDistance: active.MoveDistance, KnockbackDistance: active.KnockbackDistance,
		PlayerAtkRatio: active.PlayerAtkRatio, GuPowerRatio: active.GuPowerRatio,
	}
	compiled := &CompiledActive{
		RecoveryTicks: msToTicks(active.RecoveryMS), MovementPolicy: active.MovementPolicy,
	}
	if skill.Name == "" {
		skill.Name = guID
	}
	for effectIndex, effect := range active.Effects {
		effectPath := fmt.Sprintf("%s.effects[%d]", path, effectIndex)
		if err := validateEffect(effectPath, effect, true); err != nil {
			return nil, err
		}
		switch effect.Type {
		case "DAMAGE":
			skill.Damage += effect.Value
		case "SHIELD":
			skill.Shield += effect.Value
		case "HEAL":
			skill.Heal += effect.Value
		case "STATUS":
			if skill.StatusID != "" {
				return nil, fmt.Errorf("%s: only one status effect per preview active is supported", effectPath)
			}
			skill.StatusID = effect.StatusID
			skill.StatusDurationTicks = msToTicks(effect.DurationMS)
			skill.StatusPeriodTicks = msToTicks(effect.PeriodMS)
			if effect.TickValue > 0 {
				if effect.Debuff == nil || *effect.Debuff {
					skill.StatusDamagePerTick = effect.TickValue
				} else {
					skill.StatusHealPerTick = effect.TickValue
				}
			}
			skill.StatusMoveMultiplier = 1
			if effect.MoveMultiplier != nil {
				skill.StatusMoveMultiplier = *effect.MoveMultiplier
			}
			skill.StatusMaxStacks = 1
			if effect.MaxStacks > 0 {
				skill.StatusMaxStacks = effect.MaxStacks
			}
			skill.StatusStackPolicy = combat.StackRefresh
			if effect.StackPolicy != "" {
				skill.StatusStackPolicy = effect.StackPolicy
			}
			skill.StatusDebuff = effect.Debuff == nil || *effect.Debuff
			skill.PvPDrCategory = effect.PvPDrCategory
			if effect.ManaDrainPerSec > 0 {
				ticks := skill.StatusPeriodTicks
				if ticks == 0 {
					ticks = 20
					skill.StatusPeriodTicks = 20
				}
				skill.ManaDrainPerTick = effect.ManaDrainPerSec * (float64(ticks) / 20.0)
			}
			if effect.ThoughtDrainPerSec > 0 {
				ticks := skill.StatusPeriodTicks
				if ticks == 0 {
					ticks = 20
					skill.StatusPeriodTicks = 20
				}
				skill.ThoughtDrainPerTick = effect.ThoughtDrainPerSec * (float64(ticks) / 20.0)
			}
			if active.IsToggle || effect.IsToggle {
				skill.IsToggle = true
			}
		case "CLEANSE":
			skill.Cleanse = true
		case "STAT_MOD":
			// Active stat mods are authored but not applied by the PR-16 preview
			// runtime; the compiler still keeps the payload for PR-19.
		default:
			return nil, fmt.Errorf("%s: effect type %q is not supported by the realtime compiler", effectPath, effect.Type)
		}
		compiled.Effects = append(compiled.Effects, effect)
	}
	switch active.Mode {
	case combat.ModeDirection:
		if active.Projectile == nil {
			return nil, fmt.Errorf("%s: DIRECTION requires a projectile block", path)
		}
		if active.Projectile.SpeedUnitsPerSec <= 0 || active.Projectile.RadiusUnits <= 0 ||
			active.Projectile.LifetimeTicks == 0 || active.Projectile.LifetimeTicks > combat.MaxProjectileTicks {
			return nil, fmt.Errorf("%s: invalid projectile speed/radius/lifetime", path)
		}
		skill.ProjectileSpeed = active.Projectile.SpeedUnitsPerSec
		skill.ProjectileRadius = active.Projectile.RadiusUnits
		skill.ProjectileTicks = active.Projectile.LifetimeTicks
		skill.PierceCount = active.Projectile.PierceCount
		skill.SpreadCount = active.Projectile.SpreadCount
		skill.SpreadAngleDeg = active.Projectile.SpreadAngleDeg
		skill.MaxTargets = 1
	case combat.ModeGround:
		if active.RadiusUnits <= 0 || active.MaxTargets <= 0 || active.MaxTargets > 100 {
			return nil, fmt.Errorf("%s: GROUND requires radius_units and max_targets 1..100", path)
		}
		skill.Radius = active.RadiusUnits
		skill.MaxTargets = active.MaxTargets
	case combat.ModeTarget:
		skill.MaxTargets = 1
	case combat.ModeSelf:
		skill.MaxTargets = 1
	}
	skill.RequiresEnemyTarget = skill.Damage > 0 || skill.StatusID != ""
	if !skill.Valid() {
		return nil, fmt.Errorf("%s: compiled skill failed combat validation", path)
	}
	compiled.Skill = skill
	return compiled, nil
}

func compilePassive(path string, passive PassiveProfile) (CompiledPassive, error) {
	switch passive.Trigger {
	case "ALWAYS_ON", "PASSIVE", "COMBAT_PASSIVE":
	default:
		return CompiledPassive{}, fmt.Errorf("%s: unsupported passive trigger %q", path, passive.Trigger)
	}
	if len(passive.Effects) == 0 {
		return CompiledPassive{}, fmt.Errorf("%s: passive needs at least one effect", path)
	}
	for index, effect := range passive.Effects {
		effectPath := fmt.Sprintf("%s.effects[%d]", path, index)
		if err := validateEffect(effectPath, effect, false); err != nil {
			return CompiledPassive{}, err
		}
		if effect.Type != "STAT_MOD" {
			return CompiledPassive{}, fmt.Errorf("%s: only STAT_MOD is supported for compiled passives", effectPath)
		}
	}
	return CompiledPassive{Trigger: passive.Trigger, Effects: passive.Effects}, nil
}

func validateEffect(path string, effect EffectProfile, active bool) error {
	switch effect.Type {
	case "DAMAGE", "SHIELD", "HEAL":
		if !finiteFloat(effect.Value) || effect.Value <= 0 {
			return fmt.Errorf("%s: %s needs a positive value", path, effect.Type)
		}
	case "STATUS":
		if effect.StatusID == "" || !strings.EqualFold(effect.StatusID, strings.TrimSpace(effect.StatusID)) {
			return fmt.Errorf("%s: STATUS needs a canonical status_id", path)
		}
		if effect.DurationMS < MinStatusDurationMS || effect.DurationMS > MaxStatusDurationMS {
			return fmt.Errorf("%s: status duration_ms must be %d..%d", path, MinStatusDurationMS, MaxStatusDurationMS)
		}
		if effect.PvPDrCategory == "" {
			return fmt.Errorf("%s: STATUS needs pvp_dr_category", path)
		}
		if effect.TickValue < 0 || !finiteFloat(effect.TickValue) {
			return fmt.Errorf("%s: status tick_value must be non-negative", path)
		}
		if effect.TickValue > 0 && (effect.PeriodMS < 100 || effect.PeriodMS > 10_000) {
			return fmt.Errorf("%s: status period_ms must be 100..10000 when tick_value is set", path)
		}
		if effect.PeriodMS > 0 && effect.TickValue == 0 && effect.ManaDrainPerSec == 0 && effect.ThoughtDrainPerSec == 0 {
			return fmt.Errorf("%s: status period_ms requires tick_value or drain_per_sec", path)
		}
		if effect.MoveMultiplier != nil && (*effect.MoveMultiplier < 0 || *effect.MoveMultiplier > 3.0 || !finiteFloat(*effect.MoveMultiplier)) {
			return fmt.Errorf("%s: status move_multiplier must be 0..3", path)
		}
		if effect.ManaDrainPerSec < 0 || !finiteFloat(effect.ManaDrainPerSec) {
			return fmt.Errorf("%s: mana_drain_per_sec must be non-negative", path)
		}
		if effect.ThoughtDrainPerSec < 0 || !finiteFloat(effect.ThoughtDrainPerSec) {
			return fmt.Errorf("%s: thought_drain_per_sec must be non-negative", path)
		}
		if effect.MaxStacks < 0 || effect.MaxStacks > 20 {
			return fmt.Errorf("%s: status max_stacks must be 0..20", path)
		}
		if effect.StackPolicy != "" && !combat.ValidStackPolicy(effect.StackPolicy) {
			return fmt.Errorf("%s: unknown status stack_policy %q", path, effect.StackPolicy)
		}
	case "STAT_MOD":
		if effect.Stat == "" || !finiteFloat(effect.Value) {
			return fmt.Errorf("%s: STAT_MOD needs stat and value", path)
		}
	case "CLEANSE":
		if active != true {
			return fmt.Errorf("%s: CLEANSE is only supported as an active effect", path)
		}
	default:
		return fmt.Errorf("%s: unsupported effect type %q", path, effect.Type)
	}
	return nil
}

func msToTicks(ms uint64) uint64 {
	if ms == 0 {
		return 0
	}
	return (ms*sim.TickRate + 999) / 1000
}

func finiteFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
