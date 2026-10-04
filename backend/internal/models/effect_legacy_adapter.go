package models

import (
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

// CompileLegacyToBlocks converts legacy ActiveEffect, PassiveEffect, and SpecialMechanic
// into a standardized slice of ComposableEffectBlock.
func CompileLegacyToBlocks(active *ActiveEffect, passive *PassiveEffect, other []SpecialMechanic) []effects.Block {
	return CompileLegacyToBlocksWithGrowth(active, passive, nil, other)
}

// CompileLegacyToBlocksWithGrowth is the compatibility compiler used while
// legacy catalogs are being shadowed against Block v2.
func CompileLegacyToBlocksWithGrowth(active *ActiveEffect, passive *PassiveEffect, growth *GrowthEffect, other []SpecialMechanic) []effects.Block {
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
		if passive.Tenacity != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "TENACITY", SubType: "ADD_FLAT", Value: passive.Tenacity})
		}
		if passive.CCResist != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "CC_RESIST", SubType: "ADD_FLAT", Value: passive.CCResist})
		}
		if passive.Luck != 0 {
			passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "LUCK", SubType: "ADD_FLAT", Value: passive.Luck})
		}
		legacyStats := []struct {
			stat  string
			value float64
		}{
			{"MOVEMENT", passive.Movement}, {"INVESTIGATION", passive.Investigation},
			{"HEARING", passive.Hearing}, {"DISGUISE", passive.DisguiseLevel},
			{"TIME_SPEED", passive.TimeSpeedMultiplier}, {"REFINE_SUCCESS", passive.SuccessRateBonus},
			{"REFINE_MANA_TIER", passive.RefineManaTier}, {"STORAGE", passive.StorageCapacity},
			{"SOUL_REFINEMENT", passive.SoulRefinement}, {"CLEANSING_POWER", passive.CleansingPower},
			{"PATH_CONFLICT_REDUCTION", passive.PathConflictReduction},
		}
		for _, entry := range legacyStats {
			if entry.value != 0 {
				passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: entry.stat, SubType: "ADD_FLAT", Value: entry.value})
			}
		}
		// Legacy anti-path counters map to per-path PATH_RESISTANCE so the
		// import-time bridge preserves them as Block v2.
		for _, entry := range []struct {
			path  string
			value float64
		}{
			{"heaven", passive.AntiHeavenPct}, {"earth", passive.AntiEarthPct}, {"dream", passive.AntiDreamPct},
		} {
			if entry.value != 0 {
				passOps = append(passOps, effects.Operation{Op: effects.OpStatMod, Stat: "PATH_RESISTANCE", SubType: "ADD_FLAT", Path: entry.path, Value: entry.value})
			}
		}
		if passive.ImmuneDreamStatus {
			passOps = append(passOps, effects.Operation{Op: effects.OpUnique, Code: "IMMUNE_DREAM_STATUS"})
		}
		if len(passOps) > 0 {
			blocks = append(blocks, effects.Block{
				ID:         "LEGACY_PASSIVE_STATS",
				Trigger:    effects.TriggerAlwaysOn,
				Target:     effects.Target{Selector: effects.TargetSelf},
				Operations: passOps,
			})
		}
	}

	if growth != nil {
		var ops []effects.Operation
		for _, entry := range []struct {
			stat  string
			value float64
		}{{"ATTACK", growth.MaxAttack}, {"DEFENSE", growth.MaxDefense}, {"HP", growth.MaxHP}, {"SPEED", growth.MaxSpeed}} {
			if entry.value != 0 {
				ops = append(ops, effects.Operation{Op: effects.OpStatMod, Stat: entry.stat, SubType: "GROWTH_CAP", Value: entry.value})
			}
		}
		if growth.HoursToMax > 0 {
			ops = append(ops, effects.Operation{Op: effects.OpUnique, Code: "GROWTH_DURATION_HOURS", Params: []byte(fmt.Sprintf(`{"value":%g}`, growth.HoursToMax))})
		}
		if len(ops) > 0 {
			blocks = append(blocks, effects.Block{ID: "LEGACY_GROWTH", Trigger: effects.TriggerAlwaysOn, Target: effects.Target{Selector: effects.TargetSelf}, Operations: ops})
		}
	}

	if active != nil {
		var capabilities []effects.Operation
		if active.PortalCreate {
			capabilities = append(capabilities, effects.Operation{Op: effects.OpUnique, Code: "PORTAL_CREATE"})
		}
		if active.RandomPortal {
			capabilities = append(capabilities, effects.Operation{Op: effects.OpUnique, Code: "RANDOM_PORTAL"})
		}
		if active.CaptureBonus != 0 {
			capabilities = append(capabilities, effects.Operation{Op: effects.OpUnique, Code: "CAPTURE_BONUS", Params: []byte(fmt.Sprintf(`{"value":%g}`, active.CaptureBonus))})
		}
		if len(capabilities) > 0 {
			blocks = append(blocks, effects.Block{ID: "LEGACY_CAPABILITIES", Trigger: effects.TriggerCapability, Target: effects.Target{Selector: effects.TargetSelf}, Operations: capabilities})
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
				Op:       effects.OpHeal,
				SubType:  "RESOURCE",
				Resource: "MANA",
				Target:   &effects.Target{Selector: effects.TargetSelf},
				Value:    active.ManaRegen,
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
				ID:         "LEGACY_ACTIVE_ACTION",
				Trigger:    effects.TriggerOnAction,
				Target:     effects.Target{Selector: selector, BypassFrontline: active.CurveAttack},
				Operations: actOps,
			})
		}
	}

	// 3. Special Mechanics -> Respective Trigger Blocks
	for _, sm := range other {
		trigger := effects.Trigger(strings.ToUpper(strings.TrimSpace(sm.EventTrigger)))
		code := strings.ToUpper(strings.TrimSpace(sm.ActionCode))
		params := sm.Params
		if string(params) == "{}" || len(params) == 0 {
			switch code {
			case "MANA_REGEN_TICK":
				params = []byte(`{"value":0.5}`)
			case "THOUGHT_REGEN_TICK", "SOUL_REGEN_TICK":
				params = []byte(`{"value":1}`)
			}
		}
		blocks = append(blocks, effects.Block{
			ID:      code,
			Trigger: trigger,
			Target:  effects.Target{Selector: effects.TargetSelf},
			Operations: []effects.Operation{
				{
					Op:     effects.OpUnique,
					Code:   code,
					Params: params,
				},
			},
		})
	}

	return blocks
}
