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
	BasicAttackDamage        = 12
	BasicAttackCooldownTicks = 10
)

// Skill is one active slot for the preview loadout. Numeric values are balance
// placeholders; they are not derived from Gu.csv yet.
type Skill struct {
	ID                   string
	Slot                 int
	Mode                 Mode
	Name                 string
	ManaCost             float64
	BacklashPct          float64 `json:"-"` // admission-only legacy combat projection
	Path                 string  `json:"-"` // admission-only legacy combat projection
	Range                float64
	WindupTicks          uint64
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
	Shape                Shape   `json:"shape,omitempty"`
	AngleDeg             float64 `json:"angle_deg,omitempty"`
	LengthUnits          float64 `json:"length_units,omitempty"`
	WidthUnits           float64 `json:"width_units,omitempty"`
	InnerRadius          float64 `json:"inner_radius,omitempty"`
	PierceCount          int     `json:"pierce_count,omitempty"`
	SpreadCount          int     `json:"spread_count,omitempty"`
	SpreadAngleDeg       float64 `json:"spread_angle_deg,omitempty"`
	ManaDrainPerTick     float64 `json:"mana_drain_per_tick,omitempty"`
	ThoughtDrainPerTick  float64 `json:"thought_drain_per_tick,omitempty"`
	IsToggle             bool    `json:"is_toggle,omitempty"`
	MoveDistance         float64 `json:"move_distance,omitempty"`
	KnockbackDistance    float64 `json:"knockback_distance,omitempty"`
	PlayerAtkRatio       float64 `json:"player_atk_ratio,omitempty"`
	GuPowerRatio         float64 `json:"gu_power_ratio,omitempty"`
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
	if skill.ID == "" || skill.Slot < 1 || skill.Slot > 6 || !finite(skill.ManaCost) || skill.ManaCost < 0 || !finite(skill.BacklashPct) || skill.BacklashPct < 0 {
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
	switch skill.Mode {
	case ModeTarget:
		return skill.Range > 0 && skill.Damage >= 0
	case ModeDirection:
		return skill.Range > 0 && skill.ProjectileSpeed > 0 && skill.ProjectileRadius > 0 && skill.ProjectileTicks > 0
	case ModeGround:
		return skill.Range > 0 && skill.Radius > 0 && skill.Damage >= 0
	case ModeSelf:
		return skill.Shield >= 0 && skill.Heal >= 0 && (skill.Shield > 0 || skill.Heal > 0 || skill.StatusID != "" || skill.Cleanse)
	default:
		return false
	}
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
