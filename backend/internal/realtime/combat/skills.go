// Package combat defines the wire-independent combat contract used by the
// authoritative zone. The preview skills are placeholders until the Gu
// realtime compiler (PR-18) supplies per-loadout definitions.
package combat

import "math"

type Mode string

const (
	ModeTarget    Mode = "TARGET"
	ModeDirection Mode = "DIRECTION"
	ModeGround    Mode = "GROUND"
	ModeSelf      Mode = "SELF"
)

type Shape string

const (
	ShapeCircle    Shape = "CIRCLE"
	ShapeCone      Shape = "CONE"
	ShapeRectangle Shape = "RECTANGLE"
	ShapeDonut     Shape = "DONUT"
	ShapeCross     Shape = "CROSS"
	ShapeWall      Shape = "WALL"
)

// MovementPolicy controls how fast the caster may move during windup and
// recovery. The authored profile decides the policy; the zone never infers it
// from the skill mode.
const (
	MovementFree = "FREE"
	MovementSlow = "SLOW"
	MovementRoot = "ROOT"
)

// SlowMovementMultiplier is the ground speed factor for SLOW casts. It is a
// gameplay balance value, not a conversion from turn-based turns.
const SlowMovementMultiplier = 0.5

// DefaultGuHungerCostPerCast is the amount of hunger deducted from the required Gu on each skill cast attempt.
const DefaultGuHungerCostPerCast = 0.1

// DefaultGuDurabilityCostPerCast is the amount of durability deducted from the required Gu on each skill cast attempt.
const DefaultGuDurabilityCostPerCast = 0.02

const (
	PreviewBoltID            = "PREVIEW_BOLT"
	PreviewWaterID           = "PREVIEW_WATER_ARROW"
	PreviewGroundID          = "PREVIEW_GROUND_BURST"
	PreviewShieldID          = "PREVIEW_IRON_SHELL"
	PreviewHealID            = "PREVIEW_MENDING"
	PreviewHeavyID           = "PREVIEW_HEAVY_STRIKE"
	PreviewFrostID           = "PREVIEW_FROST_LOCK"
	DefaultGCDTicks          = 8
	DefaultManaMax           = 100
	DefaultManaRegen         = 2
	ManaRegenInterval        = 20
	MaxCastRangeUnits        = 64
	MaxProjectileTicks       = 120
	BasicAttackRange         = 2.2
	AimedHitPadding          = 0.35
	BasicAttackDamage        = 12
	BasicAttackCooldownTicks = 10
	MaxWindupTicks           = 100 // 5s at 20 Hz
	MaxRecoveryTicks         = 100 // 5s at 20 Hz
)

// Skill is one active slot for the preview loadout. Numeric values are balance
// placeholders; they are not derived from Gu.csv yet.
type Skill struct {
	ID                   string
	GuInstanceID         string   `json:"-"` // owned instance charged for live casts
	GuHunger             float64  `json:"-"` // admission snapshot; zone decrements per attempt
	GuComponentIDs       []string `json:"-"` // all owned Gu instances charged by a composite killer move
	KillerMoveID         string   `json:"-"` // persistent mastery target for successful Sát Chiêu casts
	Slot                 int
	Mode                 Mode
	Name                 string
	ManaCost             float64
	BacklashPct          float64 `json:"-"` // admission-only legacy combat projection
	SuccessChance        float64 `json:"-"`
	BackfireHPPercent    float64 `json:"-"`
	Path                 string  `json:"-"` // damage path charged for Chân Ý settlement
	Rank                 int     `json:"-"` // Gu rank charged for Chân Ý settlement
	Range                float64
	WindupTicks          uint64
	RecoveryTicks        uint64
	MovementPolicy       string
	CooldownTicks        uint64
	Damage               float64
	Shield               float64
	Heal                 float64
	Radius               float64
	ProjectileSpeed      float64 // world units per second
	ProjectileRadius     float64
	ProjectileTicks      uint64
	MaxTargets           int
	RequiresEnemyTarget  bool
	StatusID             string
	StatusDurationTicks  uint64
	StatusPeriodTicks    uint64
	StatusDamagePerTick  float64
	StatusHealPerTick    float64
	StatusMoveMultiplier float64
	StatusMaxStacks      int
	StatusStackPolicy    string
	StatusDebuff         bool
	PvPDrCategory        string
	Cleanse              bool
	Shape                Shape         `json:"shape,omitempty"`
	AngleDeg             float64       `json:"angle_deg,omitempty"`
	LengthUnits          float64       `json:"length_units,omitempty"`
	WidthUnits           float64       `json:"width_units,omitempty"`
	InnerRadius          float64       `json:"inner_radius,omitempty"`
	PierceCount          int           `json:"pierce_count,omitempty"`
	SpreadCount          int           `json:"spread_count,omitempty"`
	SpreadAngleDeg       float64       `json:"spread_angle_deg,omitempty"`
	ManaDrainPerTick     float64       `json:"mana_drain_per_tick,omitempty"`
	ThoughtDrainPerTick  float64       `json:"thought_drain_per_tick,omitempty"`
	IsToggle             bool          `json:"is_toggle,omitempty"`
	FormActivation       bool          `json:"form_activation,omitempty"`
	FormAbility          bool          `json:"form_ability,omitempty"`
	FormManaDrainPerTick float64       `json:"form_mana_drain_per_tick,omitempty"`
	FormDefenseBonus     float64       `json:"form_defense_bonus,omitempty"`
	MoveDistance         float64       `json:"move_distance,omitempty"`
	MoveDurationTicks    uint64        `json:"move_duration_ticks,omitempty"`
	MoveBehavior         string        `json:"move_behavior,omitempty"`
	KnockbackDistance    float64       `json:"knockback_distance,omitempty"`
	PlayerAtkRatio       float64       `json:"player_atk_ratio,omitempty"`
	GuPowerRatio         float64       `json:"gu_power_ratio,omitempty"`
	Visuals              VisualProfile `json:"-"` // presentation-only; never affects simulation rules
}

// VisualProfile carries symbolic client VFX template keys with a cast skill.
type VisualProfile struct {
	Icon          string `json:"icon,omitempty"`
	CastVFX       string `json:"cast_vfx,omitempty"`
	ProjectileVFX string `json:"projectile_vfx,omitempty"`
	ImpactVFX     string `json:"impact_vfx,omitempty"`
}

// DefaultLoadout returns the six preview slots. Replacing this with a compiled
// per-player loadout is PR-18/PR-23.
func DefaultLoadout() []Skill {
	return []Skill{
		{
			ID: PreviewBoltID, Slot: 1, Mode: ModeTarget, Name: "Bolt",
			ManaCost: 4, Range: 9, WindupTicks: 3, CooldownTicks: 30, Damage: 18,
			MaxTargets: 1, RequiresEnemyTarget: true,
		},
		{
			ID: PreviewWaterID, Slot: 2, Mode: ModeDirection, Name: "Water Arrow",
			ManaCost: 5, Range: 16, WindupTicks: 3, CooldownTicks: 40, Damage: 14,
			Radius: 0.35, ProjectileSpeed: 18, ProjectileRadius: 0.35, ProjectileTicks: 40,
			MaxTargets: 1, RequiresEnemyTarget: true,
		},
		{
			ID: PreviewGroundID, Slot: 3, Mode: ModeGround, Name: "Ground Burst",
			ManaCost: 7, Range: 10, WindupTicks: 6, CooldownTicks: 60, Damage: 12,
			Radius: 2.5, MaxTargets: 5, RequiresEnemyTarget: true,
		},
		{
			ID: PreviewShieldID, Slot: 4, Mode: ModeSelf, Name: "Iron Shell",
			ManaCost: 6, Range: 0, WindupTicks: 2, CooldownTicks: 120, Shield: 30,
			MaxTargets: 1,
		},
		{
			ID: PreviewHealID, Slot: 5, Mode: ModeSelf, Name: "Mending",
			ManaCost: 8, Range: 0, WindupTicks: 2, CooldownTicks: 150, Heal: 25,
			MaxTargets: 1,
		},
		{
			ID: PreviewFrostID, Slot: 6, Mode: ModeTarget, Name: "Frost Lock",
			ManaCost: 9, Range: 9, WindupTicks: 4, CooldownTicks: 120,
			StatusID: "FREEZE", StatusDurationTicks: 30, StatusMoveMultiplier: 0,
			StatusMaxStacks: 1, StatusStackPolicy: StackRefresh, StatusDebuff: true,
			PvPDrCategory: "HARD_CC", MaxTargets: 1, RequiresEnemyTarget: true,
		},
	}
}

func SkillBySlot(slot int) (Skill, bool) {
	for _, skill := range DefaultLoadout() {
		if skill.Slot == slot {
			return skill, true
		}
	}
	return Skill{}, false
}

func (skill Skill) Valid() bool {
	if skill.ID == "" || skill.Slot < 1 || skill.Slot > 12 || !finite(skill.ManaCost) || skill.ManaCost < 0 || !finite(skill.BacklashPct) || skill.BacklashPct < 0 || !finite(skill.SuccessChance) || skill.SuccessChance < 0 || skill.SuccessChance > 1 || !finite(skill.BackfireHPPercent) || skill.BackfireHPPercent < 0 {
		return false
	}
	if skill.FormActivation {
		return skill.Slot == 7 && skill.Mode == ModeSelf && skill.WindupTicks <= MaxWindupTicks &&
			finite(skill.FormManaDrainPerTick) && skill.FormManaDrainPerTick > 0 && finite(skill.FormDefenseBonus) && skill.FormDefenseBonus >= 0
	}
	if skill.FormAbility && skill.Slot != 8 {
		return false
	}
	if !ValidMovementPolicy(skill.MovementPolicy) {
		return false
	}
	if skill.WindupTicks > MaxWindupTicks || skill.RecoveryTicks > MaxRecoveryTicks {
		return false
	}
	if !finite(skill.Range) || skill.Range < 0 || skill.Range > MaxCastRangeUnits {
		return false
	}
	if !finite(skill.Damage) || !finite(skill.Shield) || !finite(skill.Heal) || !finite(skill.Radius) ||
		!finite(skill.ProjectileSpeed) || !finite(skill.ProjectileRadius) ||
		!finite(skill.AngleDeg) || skill.AngleDeg < 0 || skill.AngleDeg > 360 ||
		!finite(skill.LengthUnits) || skill.LengthUnits < 0 ||
		!finite(skill.WidthUnits) || skill.WidthUnits < 0 ||
		!finite(skill.InnerRadius) || skill.InnerRadius < 0 ||
		!finite(skill.SpreadAngleDeg) || skill.SpreadAngleDeg < 0 || skill.SpreadAngleDeg > 360 ||
		!finite(skill.ManaDrainPerTick) || skill.ManaDrainPerTick < 0 ||
		!finite(skill.ThoughtDrainPerTick) || skill.ThoughtDrainPerTick < 0 {
		return false
	}
	if skill.PierceCount < 0 || skill.PierceCount > 20 || skill.SpreadCount < 0 || skill.SpreadCount > 20 {
		return false
	}
	if skill.ProjectileTicks > MaxProjectileTicks || skill.MaxTargets < 0 || skill.MaxTargets > 100 {
		return false
	}
	if !finite(skill.MoveDistance) || skill.MoveDistance < 0 || skill.MoveDistance > MaxCastRangeUnits || (skill.MoveDistance > 0 && (skill.MoveDurationTicks == 0 || (skill.MoveBehavior != "MOVE_ONLY" && skill.MoveBehavior != "IMPACT"))) || (skill.MoveDistance == 0 && (skill.MoveDurationTicks != 0 || skill.MoveBehavior != "")) || (skill.MoveBehavior == "IMPACT" && skill.Damage <= 0) || (skill.MoveBehavior == "MOVE_ONLY" && (skill.Damage != 0 || skill.StatusID != "")) {
		return false
	}
	switch skill.Mode {
	case ModeTarget:
		return skill.Range > 0 && skill.Damage >= 0
	case ModeDirection:
		return skill.Range > 0 && ((skill.ProjectileSpeed > 0 && skill.ProjectileRadius > 0 && skill.ProjectileTicks > 0) || skill.MoveDistance > 0)
	case ModeGround:
		return skill.Range > 0 && (skill.Radius > 0 || skill.MoveDistance > 0) && skill.Damage >= 0
	case ModeSelf:
		return skill.Shield >= 0 && skill.Heal >= 0 && (skill.Shield > 0 || skill.Heal > 0 || skill.StatusID != "" || skill.Cleanse || skill.MoveDistance > 0)
	default:
		return false
	}
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// ValidMovementPolicy accepts the authored spell-cast movement policies plus
// the empty legacy default. An empty policy is treated as FREE so skills
// authored before the field existed keep their previous unrestricted
// behaviour instead of silently becoming roots.
func ValidMovementPolicy(policy string) bool {
	switch policy {
	case "", MovementFree, MovementSlow, MovementRoot:
		return true
	default:
		return false
	}
}

// MovementMultiplier is the ground-speed factor applied during windup and
// recovery. ROOT stops ground movement; SLOW halves it; FREE and the empty
// legacy default leave it unchanged.
func (skill Skill) MovementMultiplier() float64 {
	switch skill.MovementPolicy {
	case MovementRoot:
		return 0
	case MovementSlow:
		return SlowMovementMultiplier
	default:
		return 1
	}
}
