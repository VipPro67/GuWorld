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

// GuSkillProfile is the optional skill presentation and targeting metadata
// authored inline on a Gu row in Gu.csv. Effects remain in the Gu effects data.
type GuSkillProfile struct {
	CastMode        SkillCastMode `json:"cast_mode"`
	Shape           SkillShape    `json:"shape,omitempty"`
	Radius          float64       `json:"radius,omitempty"`
	Length          float64       `json:"length,omitempty"`
	Width           float64       `json:"width,omitempty"`
	AngleDeg        float64       `json:"angle_deg,omitempty"`
	CastRange       float64       `json:"cast_range,omitempty"`
	CastTimeMs      int           `json:"cast_time_ms,omitempty"`
	RecoveryMs      int           `json:"recovery_ms,omitempty"`
	CooldownMs      int           `json:"cooldown_ms,omitempty"`
	ProjectileSpeed float64       `json:"projectile_speed,omitempty"`
	Pierce          bool          `json:"pierce,omitempty"`
	MaxTargets      int           `json:"max_targets,omitempty"`
	MoveDistance    float64       `json:"move_distance,omitempty"`
	KnockbackDist   float64       `json:"knockback_dist,omitempty"`
	PlayerAtkRatio  float64       `json:"player_atk_ratio,omitempty"`
	GuPowerRatio    float64       `json:"gu_power_ratio,omitempty"`
}
