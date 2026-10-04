package models

import (
	"encoding/json"
	"strings"
)

// EffectTrigger defines when a ComposableEffectBlock is executed.
type EffectTrigger string

const (
	TriggerAlwaysOn          EffectTrigger = "ALWAYS_ON"
	TriggerOnAction          EffectTrigger = "ON_ACTION"
	TriggerOnHit             EffectTrigger = "ON_HIT"
	TriggerBeforeHit         EffectTrigger = "BEFORE_HIT"
	TriggerOnTurnStart       EffectTrigger = "ON_TURN_START"
	TriggerOnDeath           EffectTrigger = "ON_DEATH"
	TriggerOnTick            EffectTrigger = "ON_TICK"
	TriggerCapability        EffectTrigger = "CAPABILITY"
	TriggerCombatPassive     EffectTrigger = "COMBAT_PASSIVE"
	TriggerPassive           EffectTrigger = "PASSIVE"
	TriggerOnBattleStart     EffectTrigger = "ON_BATTLE_START"
	TriggerBeforeEnemyAttack EffectTrigger = "BEFORE_ENEMY_ATTACK"
	TriggerOnCast            EffectTrigger = "ON_CAST"
	TriggerOnConsume         EffectTrigger = "ON_CONSUME"
	TriggerOnUse             EffectTrigger = "ON_USE"
)

// TargetSelector defines the targeting scope for an effect block or operation.
type TargetSelector string

const (
	TargetSelf       TargetSelector = "SELF"
	TargetSingle     TargetSelector = "TARGET"
	TargetRow        TargetSelector = "ROW"
	TargetCol        TargetSelector = "COL"
	TargetAllEnemies TargetSelector = "ALL_ENEMIES"
	TargetAllAllies  TargetSelector = "ALL_ALLIES"
	TargetAll        TargetSelector = "ALL"
)

// TargetScope defines target filtering and grid positioning attributes.
type TargetScope struct {
	Selector        TargetSelector `json:"selector"`
	BypassFrontline bool           `json:"bypass_frontline,omitempty"`
}

// EffectCondition represents conditional gating on effect execution.
type EffectCondition struct {
	Field    string  `json:"field"`
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
}

// AtomicOpType defines the 9 standardized atomic operation classes.
type AtomicOpType string

const (
	OpDamage   AtomicOpType = "DAMAGE"
	OpShield   AtomicOpType = "SHIELD"
	OpHeal     AtomicOpType = "HEAL"
	OpDirectHP AtomicOpType = "DIRECT_HP"
	OpStatMod  AtomicOpType = "STAT_MOD"
	OpStatus   AtomicOpType = "STATUS"
	OpBarrier  AtomicOpType = "BARRIER"
	OpCleanse  AtomicOpType = "CLEANSE"
	OpUnique   AtomicOpType = "UNIQUE"
)

// AtomicOperation is a single granular action within a ComposableEffectBlock.
type AtomicOperation struct {
	Op          AtomicOpType    `json:"op"`
	SubType     string          `json:"sub_type,omitempty"`
	Target      *TargetScope    `json:"target,omitempty"`
	Value       float64         `json:"value,omitempty"`
	Unit        string          `json:"unit,omitempty"`
	Turns       int             `json:"turns,omitempty"`
	Chance      float64         `json:"chance,omitempty"`
	Path        string          `json:"path,omitempty"`
	StatusID    string          `json:"status_id,omitempty"`
	Stat        string          `json:"stat,omitempty"`
	Resource    string          `json:"resource,omitempty"`
	StackPolicy string          `json:"stack_policy,omitempty"`
	Code        string          `json:"code,omitempty"`
	Params      json.RawMessage `json:"params,omitempty"`
}

// ComposableEffectBlock bundles an Event-Condition-Action (ECA) unit.
type ComposableEffectBlock struct {
	ID         string            `json:"id,omitempty"`
	Trigger    EffectTrigger     `json:"trigger"`
	Target     TargetScope       `json:"target"`
	Conditions []EffectCondition `json:"conditions,omitempty"`
	Operations []AtomicOperation `json:"operations"`
}

// CompileLegacyToBlocks converts legacy ActiveEffect, PassiveEffect, and SpecialMechanic
// into a standardized slice of ComposableEffectBlock.
// compileLegacyPrototype is retained only for the original prototype tests.
// Production conversion uses effect_legacy_adapter.go and the typed gameplay
// effects package so there is exactly one public legacy adapter.
func compileLegacyPrototype(active *ActiveEffect, passive *PassiveEffect, other []SpecialMechanic) []ComposableEffectBlock {
	var blocks []ComposableEffectBlock

	// 1. Passive Stats -> ALWAYS_ON Block
	if passive != nil {
		var passOps []AtomicOperation
		if passive.StatBaseAttack != 0 {
			passOps = append(passOps, AtomicOperation{Op: OpStatMod, Stat: "ATTACK", SubType: "ADD_FLAT", Value: passive.StatBaseAttack})
		}
		if passive.StatBaseDefense != 0 {
			passOps = append(passOps, AtomicOperation{Op: OpStatMod, Stat: "DEFENSE", SubType: "ADD_FLAT", Value: passive.StatBaseDefense})
		}
		if passive.StatBaseHP != 0 {
			passOps = append(passOps, AtomicOperation{Op: OpStatMod, Stat: "HP", SubType: "ADD_FLAT", Value: passive.StatBaseHP})
		}
		if passive.StatBaseSpeed != 0 {
			passOps = append(passOps, AtomicOperation{Op: OpStatMod, Stat: "SPEED", SubType: "ADD_FLAT", Value: passive.StatBaseSpeed})
		}
		if passive.DamageMultiplierOnHit != 0 {
			passOps = append(passOps, AtomicOperation{Op: OpStatMod, Stat: "DAMAGE_DEALT", SubType: "MULTIPLY_PERCENT", Value: passive.DamageMultiplierOnHit})
		}
		if passive.Tenacity != 0 {
			passOps = append(passOps, AtomicOperation{Op: OpStatMod, Stat: "TENACITY", SubType: "ADD_FLAT", Value: passive.Tenacity})
		}
		if passive.CCResist != 0 {
			passOps = append(passOps, AtomicOperation{Op: OpStatMod, Stat: "CC_RESIST", SubType: "ADD_FLAT", Value: passive.CCResist})
		}
		if passive.Luck != 0 {
			passOps = append(passOps, AtomicOperation{Op: OpStatMod, Stat: "LUCK", SubType: "ADD_FLAT", Value: passive.Luck})
		}
		if len(passOps) > 0 {
			blocks = append(blocks, ComposableEffectBlock{
				ID:         "LEGACY_PASSIVE_STATS",
				Trigger:    TriggerAlwaysOn,
				Target:     TargetScope{Selector: TargetSelf},
				Operations: passOps,
			})
		}
	}

	// 2. Active Effect -> ON_ACTION Block
	if active != nil {
		var actOps []AtomicOperation
		selector := TargetSingle
		switch strings.ToUpper(strings.TrimSpace(active.TargetSelector)) {
		case "ALL":
			selector = TargetAllEnemies
		case "ROW":
			selector = TargetRow
		case "COL", "COLUMN":
			selector = TargetCol
		}

		if active.DamageFlat > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpDamage,
				SubType: "FLAT",
				Value:   active.DamageFlat,
				Path:    active.DamagePath,
			})
		}
		if active.DamageMult > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpDamage,
				SubType: "MULTIPLIER",
				Value:   active.DamageMult,
			})
		}
		if active.ShieldFlat > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpShield,
				SubType: "FLAT",
				Target:  &TargetScope{Selector: TargetSelf},
				Value:   active.ShieldFlat,
			})
		}
		if active.ShieldMult > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpShield,
				SubType: "MULTIPLIER",
				Target:  &TargetScope{Selector: TargetSelf},
				Value:   active.ShieldMult,
			})
		}
		if active.HealFlat > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpHeal,
				SubType: "FLAT",
				Target:  &TargetScope{Selector: TargetSelf},
				Value:   active.HealFlat,
			})
		}
		if active.ManaRegen > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpHeal,
				SubType: "MANA",
				Target:  &TargetScope{Selector: TargetSelf},
				Value:   active.ManaRegen,
			})
		}
		if active.LifeStealRate > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpHeal,
				SubType: "LIFESTEAL",
				Value:   active.LifeStealRate,
			})
		} else if active.BloodDrain {
			actOps = append(actOps, AtomicOperation{
				Op:      OpHeal,
				SubType: "LIFESTEAL",
				Value:   0.5,
			})
		}
		if active.DamageHPPercent > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpDirectHP,
				SubType: "MAX_HP_PERCENT",
				Value:   active.DamageHPPercent,
			})
		}
		if active.ShieldFlatReducePercent > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpDirectHP,
				SubType: "SHIELD_STRIP",
				Value:   active.ShieldFlatReducePercent,
			})
		}
		if active.DisableGuNextTurn {
			actOps = append(actOps, AtomicOperation{
				Op:       OpStatus,
				StatusID: "GU_SEAL",
				Turns:    1,
				Chance:   1.0,
			})
		}
		if active.SpeedBoostFlat > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpStatMod,
				Stat:    "SPEED",
				SubType: "ADD_FLAT",
				Target:  &TargetScope{Selector: TargetSelf},
				Value:   active.SpeedBoostFlat,
			})
		}
		if active.CritRateBoost > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpStatMod,
				Stat:    "CRIT_RATE",
				SubType: "ADD_FLAT",
				Target:  &TargetScope{Selector: TargetSelf},
				Value:   active.CritRateBoost,
			})
		}
		if active.AllStatsBuffPct > 0 {
			actOps = append(actOps, AtomicOperation{
				Op:      OpStatMod,
				Stat:    "ALL",
				SubType: "MULTIPLY_PERCENT",
				Target:  &TargetScope{Selector: TargetSelf},
				Value:   active.AllStatsBuffPct,
			})
		}

		// Status Applications
		for _, sa := range active.ApplyStatus {
			turns := sa.Turns
			if turns <= 0 {
				turns = active.StatusDurationTicks
			}
			val := sa.Value
			if val <= 0 {
				val = active.StatusTickDamage
			}
			chance := sa.Chance
			if chance <= 0 {
				chance = 1.0
			}
			actOps = append(actOps, AtomicOperation{
				Op:       OpStatus,
				StatusID: sa.ID,
				Value:    val,
				Turns:    turns,
				Chance:   chance,
			})
		}

		if len(actOps) > 0 {
			blocks = append(blocks, ComposableEffectBlock{
				ID:         "LEGACY_ACTIVE_ACTION",
				Trigger:    TriggerOnAction,
				Target:     TargetScope{Selector: selector, BypassFrontline: active.CurveAttack},
				Operations: actOps,
			})
		}
	}

	// 3. Special Mechanics -> Respective Trigger Blocks
	for _, sm := range other {
		trigger := EffectTrigger(strings.ToUpper(strings.TrimSpace(sm.EventTrigger)))
		blocks = append(blocks, ComposableEffectBlock{
			ID:      sm.ActionCode,
			Trigger: trigger,
			Target:  TargetScope{Selector: TargetSelf},
			Operations: []AtomicOperation{
				{
					Op:     OpUnique,
					Code:   sm.ActionCode,
					Params: sm.Params,
				},
			},
		})
	}

	return blocks
}
