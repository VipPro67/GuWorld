package models

import (
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// KillerMoveCombatProfile defines the unified tactical combat profile of a killer move.
// It is deterministically compiled from a Core Gu (hạch tâm) and auxiliary Gu (phụ trợ).
type KillerMoveCombatProfile struct {
	ProfileVersion    int                 `json:"profile_version,omitempty"`
	EffectLabel       string              `json:"effect_label,omitempty"`
	ID                string              `json:"id"`
	Name              string              `json:"name"`
	Rank              int                 `json:"rank"`
	MainPath          string              `json:"main_path"`
	AllPaths          []string            `json:"all_paths"`
	TargetType        string              `json:"target_type"` // SINGLE | ROW | COL | CROSS | ALL_ENEMIES
	Bypass            bool                `json:"bypass"`      // Bypasses frontline / shields
	CooldownTurns     int                 `json:"cooldown_turns,omitempty"`
	WarmupTurns       int                 `json:"warmup_turns,omitempty"`
	Priority          int                 `json:"priority"`
	ManaCost          float64             `json:"mana_cost"`
	HungerCost        float64             `json:"hunger_cost"`
	BaseDamage        float64             `json:"base_damage"`
	DamageMult        float64             `json:"damage_mult"`
	BaseShield        float64             `json:"base_shield"`
	BaseHealing       float64             `json:"base_healing"`
	LifeStealPct      float64             `json:"lifesteal_pct"`
	CritBonus         float64             `json:"crit_bonus"`
	StatusOps         []statusfx.StatusOp `json:"status_ops,omitempty"`
	SuccessChance     float64             `json:"success_chance"`
	BackfireHPPercent float64             `json:"backfire_hp_pct"`
	BackfireStatus    string              `json:"backfire_status"`
}

// GuHouseCombatProfile represents an Immortal Gu House (Tiên Cổ Ốc) in combat.
// It acts as a materialized immortal killer move entity / vehicle stance.
type GuHouseCombatProfile struct {
	ProfileVersion     int                       `json:"profile_version,omitempty"`
	PreviewFingerprint string                    `json:"preview_fingerprint,omitempty"`
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	Rank               int                       `json:"rank"`
	CoreGuIDs          []string                  `json:"core_gu_ids"`
	AuxGuCount         int                       `json:"aux_gu_count"`
	MaxHouseHP         float64                   `json:"max_house_hp"`
	CurrentHouseHP     float64                   `json:"current_house_hp"`
	HouseDefense       float64                   `json:"house_defense"`
	DamageReduction    float64                   `json:"damage_reduction"` // % damage reduction (0..1)
	ImmuneDebuffs      bool                      `json:"immune_debuffs"`
	ZeroBackfire       bool                      `json:"zero_backfire"`
	Actions            []KillerMoveCombatProfile `json:"actions"`
}
