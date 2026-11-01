package models

// SkillCastMode describes how an authored Gu skill is aimed.
type SkillCastMode string

const (
	SkillCastTarget    SkillCastMode = "TARGET"
	SkillCastDirection SkillCastMode = "DIRECTION"
	SkillCastGround    SkillCastMode = "GROUND"
	SkillCastSelf      SkillCastMode = "SELF"
)

// SkillShape is the hit area a cast resolves against once it completes.
type SkillShape string

const (
	SkillShapeSingle    SkillShape = "SINGLE"
	SkillShapeLine      SkillShape = "LINE"
	SkillShapeCircle    SkillShape = "CIRCLE"
	SkillShapeCone      SkillShape = "CONE"
	SkillShapeRectangle SkillShape = "RECTANGLE"
	SkillShapeDonut     SkillShape = "DONUT"
	SkillShapeCross     SkillShape = "CROSS"
	SkillShapeWall      SkillShape = "WALL"
)

// GuSkillProfile is the unified skill presentation, resource, combat geometry,
// timing, status effect, and visual/audio metadata authored inline on a Gu row in Gu.csv.
type GuSkillProfile struct {
	// 1. Presentation Override
	SkillName string `json:"skill_name,omitempty"`
	SkillIcon string `json:"skill_icon,omitempty"`
	Slot      int    `json:"slot,omitempty"`

	// 2. Resource Costs
	ManaCost    float64 `json:"mana_cost,omitempty"`
	ThoughtCost float64 `json:"thought_cost,omitempty"`
	HpCost      float64 `json:"hp_cost,omitempty"`
	HungerCost  float64 `json:"hunger_cost,omitempty"`

	// 3. Path & Damage Category
	Path          string  `json:"path,omitempty"`
	ElementType   string  `json:"element_type,omitempty"`
	DamageType    string  `json:"damage_type,omitempty"` // "PATH", "TRUE", "DIRECT_HP"
	IsCritAllowed bool    `json:"is_crit_allowed,omitempty"`
	CritRateBonus float64 `json:"crit_rate_bonus,omitempty"`
	CritDmgBonus  float64 `json:"crit_damage_bonus,omitempty"`

	// 4. Cast & Geometry
	CastMode   SkillCastMode `json:"cast_mode"`
	Shape      SkillShape    `json:"shape,omitempty"`
	Radius     float64       `json:"radius,omitempty"`
	Length     float64       `json:"length,omitempty"`
	Width      float64       `json:"width,omitempty"`
	AngleDeg   float64       `json:"angle_deg,omitempty"`
	CastRange  float64       `json:"cast_range,omitempty"`
	MinRange   float64       `json:"min_range,omitempty"`
	MaxTargets int           `json:"max_targets,omitempty"`

	// 5. Timing (ms)
	CastTimeMs int `json:"cast_time_ms,omitempty"`
	RecoveryMs int `json:"recovery_ms,omitempty"`
	CooldownMs int `json:"cooldown_ms,omitempty"`

	// 6. Projectile & Mobility
	ProjectileSpeed float64 `json:"projectile_speed,omitempty"`
	Pierce          bool    `json:"pierce,omitempty"`
	SpreadCount     int     `json:"spread_count,omitempty"`
	SpreadAngleDeg  float64 `json:"spread_angle_deg,omitempty"`
	MoveDistance    float64 `json:"move_distance,omitempty"`
	MoveDurationMs  int     `json:"move_duration_ms,omitempty"`
	MoveBehavior    string  `json:"move_behavior,omitempty"`
	KnockbackDist   float64 `json:"knockback_dist,omitempty"`

	// 7. Base Payload & Scaling
	DamageBase     float64 `json:"damage_base,omitempty"`
	DamageValue    float64 `json:"damage_value,omitempty"` // backward-compatible alias
	PlayerAtkRatio float64 `json:"player_atk_ratio,omitempty"`
	GuPowerRatio   float64 `json:"gu_power_ratio,omitempty"`

	// 8. Status Effects & CC
	StatusID             string   `json:"status_id,omitempty"`
	EffectChance         float64  `json:"effect_chance,omitempty"`
	DurationMs           int      `json:"duration_ms,omitempty"`
	StatusDurationMs     int      `json:"status_duration_ms,omitempty"` // backward-compatible alias
	DotPeriodMs          int      `json:"dot_period_ms,omitempty"`
	StatusPeriodMs       int      `json:"status_period_ms,omitempty"` // backward-compatible alias
	DotTickValue         float64  `json:"dot_tick_value,omitempty"`
	StatusTickValue      float64  `json:"status_tick_value,omitempty"` // backward-compatible alias
	StatusMoveMultiplier *float64 `json:"status_move_multiplier,omitempty"`
	StatusMoveMult       *float64 `json:"status_move_mult,omitempty"` // alias
	StatusMaxStacks      int      `json:"status_max_stacks,omitempty"`
	StatusStackPolicy    string   `json:"status_stack_policy,omitempty"`
	StatusDebuff         *bool    `json:"status_debuff,omitempty"`
	PvPDrCategory        string   `json:"pvp_dr_category,omitempty"`

	// 9. Support & Defense
	ShieldBase    float64 `json:"shield_base,omitempty"`
	ShieldValue   float64 `json:"shield_value,omitempty"` // backward-compatible alias
	ShieldRatio   float64 `json:"shield_ratio,omitempty"`
	HealBase      float64 `json:"heal_base,omitempty"`
	HealValue     float64 `json:"heal_value,omitempty"` // backward-compatible alias
	HealRatio     float64 `json:"heal_ratio,omitempty"`
	CleanseDebuff bool    `json:"cleanse_debuff,omitempty"`

	// 10. VFX & SFX Keys (Asset Identifiers)
	CastVFX       string `json:"cast_vfx,omitempty"`
	ProjectileVFX string `json:"projectile_vfx,omitempty"`
	ImpactVFX     string `json:"impact_vfx,omitempty"`
	PersistentVFX string `json:"persistent_vfx,omitempty"`
	CastSFX       string `json:"cast_sfx,omitempty"`
	HitSFX        string `json:"hit_sfx,omitempty"`

	// 11. Combat Juice & Feedback
	CameraShakeIntensity float64 `json:"camera_shake_intensity,omitempty"`
	HitStopDurationMs    int     `json:"hit_stop_duration_ms,omitempty"`
}
