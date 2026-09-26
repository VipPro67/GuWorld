package models

import (
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

// CompileLegacyToBlocks converts legacy ActiveEffect, PassiveEffect, and SpecialMechanic
// into a standardized slice of ComposableEffectBlock.
func CompileLegacyToBlocks(active *ActiveEffect, passive *PassiveEffect, other []SpecialMechanic) []effects.Block {
	var blocks []effects.Block

	// 1. Passive Stats -> ALWAYS_ON Block
	if passive != nil {
		var passOps []effects.Operation
		if passive.StatBaseAttack != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "ATTACK", SubType: "ADD_FLAT", Value: passive.StatBaseAttack})
		}
		if passive.StatBaseDefense != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "DEFENSE", SubType: "ADD_FLAT", Value: passive.StatBaseDefense})
		}
		if passive.StatBaseHP != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "HP", SubType: "ADD_FLAT", Value: passive.StatBaseHP})
		}
		if passive.StatBaseSpeed != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "SPEED", SubType: "ADD_FLAT", Value: passive.StatBaseSpeed})
		}
		if passive.DamageMultiplierOnHit != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "DAMAGE_DEALT", SubType: "MULTIPLY_PERCENT", Value: passive.DamageMultiplierOnHit})
		}
		if passive.Accuracy != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "ACCURACY", SubType: "ADD_FLAT", Value: passive.Accuracy})
		}
		if passive.Evasion != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "EVASION", SubType: "ADD_FLAT", Value: passive.Evasion})
		}
		if passive.Tenacity != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "TENACITY", SubType: "ADD_FLAT", Value: passive.Tenacity})
		}
		if passive.CCResist != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "CC_RESIST", SubType: "ADD_FLAT", Value: passive.CCResist})
		}
		if passive.Luck != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "LUCK", SubType: "ADD_FLAT", Value: passive.Luck})
		}
		if len(passOps) > 0 {
			blocks = append(blocks, effects.Block{
				ID:         "legacy_passive_stats",
				Trigger:    effects.TriggerAlwaysOn,
				Target:     effects.Target{Selector: effects.TargetSelf},
				Operations: passOps,
			})
		}
	}

	// 2. Active Effect -> ON_ACTION Block
	if active != nil {
		var actOps []effects.Operation
		selector := effects.TargetSingle
		switch strings.ToUpper(strings.TrimSpace(active.TargetSelector)) {
		case "ALL":
			selector = effects.TargetAllEnemies
		case "ROW":
			selector = effects.TargetRow
		case "COL", "COLUMN":
			selector = effects.TargetCol
		}

		if active.DamageFlat > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpDamage,
				SubType: "FLAT",
				Value:   active.DamageFlat,
				Path:    active.DamagePath,
			})
		}
		if active.DamageMult > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpDamage,
				SubType: "MULTIPLIER",
				Value:   active.DamageMult,
			})
		}
		if active.ShieldFlat > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpShield,
				SubType: "FLAT",
				Target:  &effects.Target{Selector: effects.TargetSelf},
				Value:   active.ShieldFlat,
			})
		}
		if active.ShieldMult > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpShield,
				SubType: "MULTIPLIER",
				Target:  &effects.Target{Selector: effects.TargetSelf},
				Value:   active.ShieldMult,
			})
		}
		if active.HealFlat > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpHeal,
				SubType: "FLAT",
				Target:  &effects.Target{Selector: effects.TargetSelf},
				Value:   active.HealFlat,
			})
		}
		if active.ManaRegen > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpHeal,
				SubType: "MANA",
				Target:  &effects.Target{Selector: effects.TargetSelf},
				Value:   active.ManaRegen,
			})
		}
		if active.LifeStealRate > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpHeal,
				SubType: "LIFESTEAL",
				Value:   active.LifeStealRate,
			})
		} else if active.BloodDrain {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpHeal,
				SubType: "LIFESTEAL",
				Value:   0.5,
			})
		}
		if active.DamageHPPercent > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpDirectHP,
				SubType: "MAX_HP_PERCENT",
				Value:   active.DamageHPPercent,
			})
		}
		if active.ShieldFlatReducePercent > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpDirectHP,
				SubType: "SHIELD_STRIP",
				Value:   active.ShieldFlatReducePercent,
			})
		}
		if active.DisableGuNextTurn {
			actOps = append(actOps, effects.Operation{
				Op:       effects.OpStatus,
				StatusID: "GU_SEAL",
				Turns:    1,
				Chance:   1.0,
			})
		}
		if active.SpeedBoostFlat > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpStatMod,
				Stat:    "SPEED",
				SubType: "ADD_FLAT",
				Target:  &effects.Target{Selector: effects.TargetSelf},
				Value:   active.SpeedBoostFlat,
			})
		}
		if active.CritRateBoost > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpStatMod,
				Stat:    "CRIT_RATE",
				SubType: "ADD_FLAT",
				Target:  &effects.Target{Selector: effects.TargetSelf},
				Value:   active.CritRateBoost,
			})
		}
		if active.SelfEvasionBoostPercent > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpStatMod,
				Stat:    "EVASION",
				SubType: "ADD_FLAT",
				Target:  &effects.Target{Selector: effects.TargetSelf},
				Value:   active.SelfEvasionBoostPercent,
			})
		}
		if active.AllStatsBuffPct > 0 {
			actOps = append(actOps, effects.Operation{
				Op:      effects.OpStatMod,
				Stat:    "ALL",
				SubType: "MULTIPLY_PERCENT",
				Target:  &effects.Target{Selector: effects.TargetSelf},
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
			actOps = append(actOps, effects.Operation{
				Op:       effects.OpStatus,
				StatusID: sa.ID,
				Value:    val,
				Turns:    turns,
				Chance:   chance,
			})
		}

		if len(actOps) > 0 {
			blocks = append(blocks, effects.Block{
				ID:         "legacy_active_action",
				Trigger:    effects.TriggerOnAction,
				Target:     effects.Target{Selector: selector, BypassFrontline: active.CurveAttack},
				Operations: actOps,
			})
		}
	}

	// 3. Special Mechanics -> Respective Trigger Blocks
	for _, sm := range other {
		trigger := effects.Trigger(strings.ToUpper(strings.TrimSpace(sm.EventTrigger)))
		blocks = append(blocks, effects.Block{
			ID:      sm.ActionCode,
			Trigger: trigger,
			Target:  effects.Target{Selector: effects.TargetSelf},
			Operations: []effects.Operation{
				{
					Op:     effects.OpUnique,
					Code:   sm.ActionCode,
					Params: sm.Params,
				},
			},
		})
	}

	return blocks
}
