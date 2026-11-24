package handlers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

type combatOperationGroup struct {
	conditions []gameplayeffects.Condition
}

// combatPassiveDamageModifiers projects data-authored combat passives that
// modify every action, separately from the action operation list.
func combatPassiveDamageModifiers(spec models.GuDictionaryItem) []simulator.DamageModifier {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.DamageModifier
	for _, op := range compiled.Operations(gameplayeffects.TriggerCombatPassive) {
		if op.Type != gameplayeffects.OpDamageModifier {
			continue
		}
		if len(op.Conditions) > 0 || op.Phase != "" {
			continue
		}
		result = append(result, simulator.DamageModifier{
			SubType: op.SubType, BaseMultiplier: op.Value, PerMissingHP: op.PerMissingHP,
			PerTargetDebuff: op.PerTargetDebuff, PerMinion: op.PerMinion, CounterKey: op.Key, PerCounter: op.PerStack,
			StatusID: op.StatusID, Phase: op.Phase, LogAction: op.LogAction, Message: op.Message,
		})
	}
	return result
}

// combatDamageMitigations projects BEFORE_HIT operations into the incoming
// strike pipeline for every equipped Gu that can affect its owner.
func combatDamageMitigations(spec models.GuDictionaryItem) []simulator.DamageMitigation {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.DamageMitigation
	for _, op := range compiled.Operations(gameplayeffects.TriggerBeforeHit) {
		if op.Type != gameplayeffects.OpMitigateDamage {
			continue
		}
		result = append(result, simulator.DamageMitigation{
			BlockID: op.BlockID, Ratio: op.Value, ConvertToShield: op.SubType == "CONVERT_TO_SHIELD",
			LogAction: op.LogAction, Message: op.Message,
		})
	}
	return result
}

func combatPassiveOperationEffects(spec models.GuDictionaryItem) []simulator.SimActionOperation {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.SimActionOperation
	for _, op := range compiled.Operations(gameplayeffects.TriggerCombatPassive) {
		switch op.Type {
		case gameplayeffects.OpDamageModifier:
			if len(op.Conditions) == 0 && op.Phase == "" {
				continue // Existing unconditional modifiers use the fast passive slice.
			}
		case gameplayeffects.OpCleanse:
			if op.SubType != "STATUS" || op.Phase == "" {
				continue
			}
		case gameplayeffects.OpHeal:
			if op.SubType != "COUNTER_PERCENT" || op.Phase == "" {
				continue
			}
		case gameplayeffects.OpNotify:
			if op.Phase != "FINAL_RIDER" {
				continue // Validation rejects passive notifications without an executable phase.
			}
		case gameplayeffects.OpTenacityModifier, gameplayeffects.OpStatusImmunity, gameplayeffects.OpEffectImmunity:
			// Recomputed from the compiled passive contract with derived combat stats.
		default:
			continue
		}
		child := &simulator.SimAction{
			ActionType: "GU_EFFECT_OPERATION", Name: spec.Name, SourceID: spec.ID,
			Path: append([]string(nil), spec.Path...), Rank: spec.Rank, IsEffectOperation: true,
		}
		if op.Type == gameplayeffects.OpDamageModifier {
			child.DamageModifiers = append(child.DamageModifiers, simulator.DamageModifier{
				SubType: op.SubType, BaseMultiplier: op.Value, PerMissingHP: op.PerMissingHP,
				PerTargetDebuff: op.PerTargetDebuff, PerMinion: op.PerMinion, CounterKey: op.Key, PerCounter: op.PerStack,
				StatusID: op.StatusID, Phase: op.Phase, LogAction: op.LogAction, Message: op.Message,
			})
		} else if op.Type == gameplayeffects.OpCleanse && op.SubType == "STATUS" {
			child.CleanseStatusIDs = []string{op.StatusID}
		} else if op.Type == gameplayeffects.OpHeal && op.SubType == "COUNTER_PERCENT" {
			child.CounterHeals = append(child.CounterHeals, simulator.CounterHeal{Key: op.Key, PerCounter: op.Value})
		} else if op.Type == gameplayeffects.OpNotify {
			logAction := op.LogAction
			if logAction == "" {
				logAction = "EFFECT_NOTIFY"
			}
			child.CombatNotifications = append(child.CombatNotifications, simulator.CombatNotification{Action: logAction, Message: op.Message, CounterKey: op.Key, Phase: op.Phase, EffectID: fmt.Sprintf("%s:%d:%d", op.BlockID, op.BlockIndex, op.OperationIndex)})
		}
		result = append(result, simulator.SimActionOperation{
			Compiled: op, SourceGuID: spec.ID, SourcePaths: append([]string(nil), spec.Path...), SourceRank: spec.Rank, Action: child,
		})
	}
	return result
}

func combatBeforeEnemyAttackOperationEffects(spec models.GuDictionaryItem) []simulator.SimActionOperation {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.SimActionOperation
	for _, op := range compiled.Operations(gameplayeffects.TriggerBeforeEnemyAttack) {
		if (op.Type != gameplayeffects.OpGuSeal || op.GuSeal == nil) &&
			(op.Type != gameplayeffects.OpTargetRedirection || op.TargetRedirection == nil) {
			continue
		}
		child := &simulator.SimAction{
			ActionType: "GU_EFFECT_OPERATION", Name: spec.Name, SourceID: spec.ID,
			Path: append([]string(nil), spec.Path...), Rank: spec.Rank, IsEffectOperation: true,
		}
		if op.Type == gameplayeffects.OpTargetRedirection && op.TargetRedirection != nil {
			child.Mechanics = append(child.Mechanics, simulator.CombatMechanic{
				Trigger:    "BEFORE_ENEMY_ATTACK",
				ActionCode: "FORGET_WILL_CONFUSE",
				Params:     append([]byte(nil), op.Params...),
				Values:     op.Values,
			})
		}
		result = append(result, simulator.SimActionOperation{
			Compiled: op, SourceGuID: spec.ID, SourcePaths: append([]string(nil), spec.Path...), SourceRank: spec.Rank, Action: child,
		})
	}
	return result
}

func combatTurnStartCounterAdds(spec models.GuDictionaryItem) []simulator.CounterAdd {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.CounterAdd
	for _, op := range compiled.Operations(gameplayeffects.TriggerOnTurnStart) {
		if op.Type != gameplayeffects.OpCounterAdd {
			continue
		}
		if op.Target == nil || op.Target.Selector != gameplayeffects.TargetSelf || effectcatalog.ValidateCounterAdd(op.Key, op.Delta, op.Min, op.Max) != nil {
			continue // invalid authored operations fail compilation before this projection
		}
		result = append(result, simulator.CounterAdd{Key: op.Key, Delta: op.Delta, Min: op.Min, Max: op.Max})
	}
	return result
}

func combatBeforeHitCounterAdds(spec models.GuDictionaryItem) []simulator.CounterAdd {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.CounterAdd
	for _, op := range compiled.Operations(gameplayeffects.TriggerBeforeHit) {
		if op.Type != gameplayeffects.OpCounterAdd || op.Target == nil || op.Target.Selector != gameplayeffects.TargetSelf {
			continue
		}
		if err := effectcatalog.ValidateCounterAdd(op.Key, op.Delta, op.Min, op.Max); err != nil {
			continue // invalid authored operations fail compilation before this projection
		}
		result = append(result, simulator.CounterAdd{Key: op.Key, Delta: op.Delta, Min: op.Min, Max: op.Max})
	}
	return result
}

func combatBeforeHitCounterStatModifiers(spec models.GuDictionaryItem) []simulator.CounterStatModifier {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.CounterStatModifier
	for _, op := range compiled.Operations(gameplayeffects.TriggerBeforeHit) {
		if op.Type != gameplayeffects.OpStatMod || op.SubType != "ADD_COUNTER_DELTA" || op.Target == nil || op.Target.Selector != gameplayeffects.TargetSelf {
			continue
		}
		if err := effectcatalog.ValidateCounterStatModifier(op.Stat, op.Key, op.PerStack); err != nil {
			continue // invalid authored operations fail compilation before this projection
		}
		result = append(result, simulator.CounterStatModifier{Key: op.Key, Stat: op.Stat, PerCounter: op.PerStack})
	}
	return result
}

func combatBeforeHitCounterNotifications(spec models.GuDictionaryItem) []simulator.CombatNotification {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.CombatNotification
	for _, op := range compiled.Operations(gameplayeffects.TriggerBeforeHit) {
		if op.Type != gameplayeffects.OpNotify || op.Target == nil || op.Target.Selector != gameplayeffects.TargetSelf {
			continue
		}
		if err := effectcatalog.ValidateNotifyOperation(string(op.Trigger), string(op.Target.Selector), op.Phase, op.Key, op.Message); err != nil {
			continue // invalid authored operations fail compilation before this projection
		}
		logAction := op.LogAction
		if logAction == "" {
			logAction = "EFFECT_NOTIFY"
		}
		result = append(result, simulator.CombatNotification{Action: logAction, Message: op.Message, CounterKey: op.Key, Phase: op.Phase})
	}
	return result
}

func combatBeforeHitRecoveries(spec models.GuDictionaryItem) []simulator.ThresholdRecovery {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.ThresholdRecovery
	for _, op := range compiled.Operations(gameplayeffects.TriggerBeforeHit) {
		if op.Type != gameplayeffects.OpThresholdRecovery || op.Target == nil || op.Target.Selector != gameplayeffects.TargetSelf {
			continue
		}
		if err := effectcatalog.ValidateThresholdRecovery(op.HPThreshold, op.HealPercent, op.MaxUses, op.CleanseDebuffs, op.LogAction, op.Message); err != nil {
			continue // invalid authored operations fail compilation before this projection
		}
		result = append(result, simulator.ThresholdRecovery{
			SourceID: spec.ID, OperationID: fmt.Sprintf("%s:%d:%d", op.BlockID, op.BlockIndex, op.OperationIndex),
			HPThreshold: op.HPThreshold, HealPercent: op.HealPercent, MaxUses: op.MaxUses,
			CleanseDebuffs: op.CleanseDebuffs, LogAction: op.LogAction, Message: op.Message,
		})
	}
	return result
}

func combatBattleStartEndCommands(spec models.GuDictionaryItem) []simulator.BattleEndCommand {
	compiled := models.CompiledGuEffects(spec)
	if compiled == nil {
		return nil
	}
	var result []simulator.BattleEndCommand
	for _, op := range compiled.Operations(gameplayeffects.TriggerOnBattleStart) {
		if op.Type != gameplayeffects.OpEndBattle || op.SubType != "FORCE_VICTORY" || op.Target == nil || op.Target.Selector != gameplayeffects.TargetAllEnemies {
			continue // invalid authoring is rejected by the effect compiler
		}
		result = append(result, simulator.BattleEndCommand{Outcome: op.SubType, LogAction: op.LogAction, Message: op.Message})
	}
	return result
}

// adaptBlockCombatOperations turns compiled combat action/cast operations into the
// simulator's operation-level combat carrier. Each block/target/condition
// group becomes one action, so operation provenance and conditional targeting
// survive without re-reading authored JSON in the simulator.
func adaptBlockCombatOperations(actor *models.Combatant, spec *models.GuDictionaryItem, scale float64, operations []gameplayeffects.CompiledOperation) ([]simulator.SimActionOperation, error) {
	groups := make([]combatOperationGroup, 0)
	groupIndexes := make(map[string]int)
	actions := make([]simulator.SimAction, 0)
	compiledByGroup := make([]gameplayeffects.CompiledOperation, 0)
	groupKey := func(op gameplayeffects.CompiledOperation, target gameplayeffects.Target) string {
		return fmt.Sprintf("%d|%s|%t|%v", op.BlockIndex, target.Selector, target.BypassFrontline, op.Conditions)
	}
	for _, op := range operations {
		if op.Trigger != gameplayeffects.TriggerOnAction && op.Trigger != gameplayeffects.TriggerOnCast {
			continue
		}
		if op.Trigger == gameplayeffects.TriggerOnCast && op.Type != gameplayeffects.OpDispel && op.Type != gameplayeffects.OpStatus && op.Type != gameplayeffects.OpResourceConversion && op.Type != gameplayeffects.OpOutcomeRoulette {
			return nil, fmt.Errorf("effect %q operation %s is not supported for ON_CAST combat execution", op.BlockID, op.Type)
		}
		if op.Trigger == gameplayeffects.TriggerOnCast && len(op.Conditions) > 0 {
			return nil, fmt.Errorf("effect %q ON_CAST operations do not support block conditions", op.BlockID)
		}
		if op.Trigger == gameplayeffects.TriggerOnCast && (op.Target == nil ||
			(op.Type == gameplayeffects.OpDispel && op.Target.Selector != gameplayeffects.TargetSingle) ||
			((op.Type == gameplayeffects.OpStatus || op.Type == gameplayeffects.OpResourceConversion || (op.Type == gameplayeffects.OpOutcomeRoulette && op.SubType == "CHAOS")) && op.Target.Selector != gameplayeffects.TargetSelf)) {
			return nil, fmt.Errorf("effect %q ON_CAST DISPEL requires TARGET and STATUS/RESOURCE_CONVERSION/CHAOS requires SELF", op.BlockID)
		}
		if reason := unsupportedConditionReason("TURN_BASED_COMBAT", op.Conditions); reason != "" {
			return nil, fmt.Errorf("effect %q has unsupported turn-based conditions: %s", op.BlockID, reason)
		}
		if op.Type == gameplayeffects.OpUnique && len(op.Conditions) > 0 {
			return nil, fmt.Errorf("effect %q native combat hook cannot evaluate Block conditions", op.BlockID)
		}
		if op.Type == gameplayeffects.OpHPCost && len(op.Conditions) > 0 {
			return nil, fmt.Errorf("effect %q HP_COST cannot evaluate Block conditions", op.BlockID)
		}
		if op.Target == nil || !effectcatalogCombatTarget(op.Target.Selector) {
			return nil, fmt.Errorf("effect %q has no supported combat target", op.BlockID)
		}
		key := groupKey(op, *op.Target)
		index, exists := groupIndexes[key]
		if !exists {
			index = len(groups)
			groupIndexes[key] = index
			groups = append(groups, combatOperationGroup{conditions: append([]gameplayeffects.Condition(nil), op.Conditions...)})
			actions = append(actions, simulator.SimAction{
				ActionType: "GU_EFFECT_OPERATION", Name: spec.Name, SourceID: spec.ID,
				Path: append([]string(nil), spec.Path...), Rank: spec.Rank,
				DamageMult: scale, ShieldMult: scale,
				TargetType: string(op.Target.Selector), Bypass: op.Target.BypassFrontline,
				TargetX: actor.GridX, TargetY: actor.GridY,
				IsEffectOperation: true,
			})
			compiledByGroup = append(compiledByGroup, op)
		}
		child := &actions[index]
		if op.Path != "" {
			child.Path = []string{op.Path}
		}
		switch op.Type {
		case gameplayeffects.OpBonusStrike:
			if op.Target.Selector != gameplayeffects.TargetSingle {
				return nil, fmt.Errorf("effect %q bonus strike requires TARGET", op.BlockID)
			}
			if err := effectcatalog.ValidateBonusStrike(op.Chance, op.Value, op.MinDamage, op.LogAction, op.Message); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			child.BonusStrikes = append(child.BonusStrikes, simulator.BonusStrike{
				Chance: op.Chance, DamageModifier: op.Value, MinDamage: op.MinDamage,
				LogAction: op.LogAction, Message: op.Message,
			})
		case gameplayeffects.OpChainDamage:
			if len(op.Conditions) > 0 {
				return nil, fmt.Errorf("effect %q chain damage does not support block conditions", op.BlockID)
			}
			if op.Target.Selector != gameplayeffects.TargetAllEnemies {
				return nil, fmt.Errorf("effect %q chain damage requires ALL_ENEMIES", op.BlockID)
			}
			if err := effectcatalog.ValidateChainDamage(op.Value, op.MinDamage, op.TargetCount, op.LogAction, op.Message); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			child.ChainDamages = append(child.ChainDamages, simulator.ChainDamage{
				Fraction: op.Value, MinDamage: op.MinDamage, TargetCount: op.TargetCount,
				LogAction: op.LogAction, Message: op.Message,
			})
		case gameplayeffects.OpDamage:
			switch strings.ToUpper(op.SubType) {
			case "", "FLAT":
				child.Damage += op.Value
			case "MULTIPLIER":
				if op.Phase == "PRE_CHAIN" {
					child.PreChainDamageModifiers = append(child.PreChainDamageModifiers, simulator.PhaseDamageModifier{
						Multiplier: op.Value,
						EffectID:   fmt.Sprintf("%s:%d:%d", op.BlockID, op.BlockIndex, op.OperationIndex),
					})
				} else {
					child.DamageMult *= op.Value
				}
			case "MAX_HP_PERCENT":
				child.DamageHPPercent += percentFraction(op.Value)
			default:
				return nil, fmt.Errorf("effect %q has unsupported combat damage subtype %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpDamageModifier:
			if err := effectcatalog.ValidateDamageModifier(op.SubType, op.Value, op.PerMissingHP, op.PerTargetDebuff, op.PerMinion, op.PerStack, op.Key, op.StatusID, op.LogAction, op.Message); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			child.DamageModifiers = append(child.DamageModifiers, simulator.DamageModifier{
				SubType: op.SubType, BaseMultiplier: op.Value, PerMissingHP: op.PerMissingHP, PerTargetDebuff: op.PerTargetDebuff, PerMinion: op.PerMinion, CounterKey: op.Key, PerCounter: op.PerStack,
				StatusID: op.StatusID, LogAction: op.LogAction, Message: op.Message,
			})
		case gameplayeffects.OpDefensePenetration:
			if err := effectcatalog.ValidateDefensePenetration(op.Value); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			if op.Value > child.DefensePenetration {
				child.DefensePenetration = op.Value
			}
		case gameplayeffects.OpTargetDamageModifier:
			if err := effectcatalog.ValidateTargetDamageModifier(op.SubType, op.Value, op.MinMultiplier, op.LogAction, op.Message); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			child.TargetDamageModifiers = append(child.TargetDamageModifiers, simulator.TargetDamageModifier{
				Value: op.Value, MinMultiplier: op.MinMultiplier, LogAction: op.LogAction, Message: op.Message,
			})
		case gameplayeffects.OpDetonateStatus:
			if op.Target.Selector != gameplayeffects.TargetSingle {
				return nil, fmt.Errorf("effect %q detonates status on an unsupported combat target", op.BlockID)
			}
			if err := effectcatalog.ValidateStatusDetonation(op.StatusID, op.PerStack, op.PerStrength, op.MaxHPPercent, op.MinDamage, op.Phase); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			if err := effectcatalog.ValidateCombatLogAction(op.LogAction); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			if err := effectcatalog.ValidateNotifyMessage(op.Message); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			child.StatusDetonations = append(child.StatusDetonations, simulator.StatusDetonation{
				StatusID: op.StatusID, PerStack: op.PerStack, PerStrength: op.PerStrength,
				MaxHPPercent: op.MaxHPPercent, MinDamage: op.MinDamage, Phase: op.Phase,
				LogAction: op.LogAction, Message: op.Message,
			})
		case gameplayeffects.OpHPCost:
			if op.Target.Selector != gameplayeffects.TargetSelf {
				return nil, fmt.Errorf("effect %q charges HP from an unsupported combat target", op.BlockID)
			}
			if err := effectcatalog.ValidateHPCost(op.SubType, op.Value, op.MinCost, op.MinRemaining, op.LogAction, op.Message); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			child.HPCosts = append(child.HPCosts, simulator.HPCost{
				CurrentHPPercent: op.Value, MinCost: op.MinCost, MinRemaining: op.MinRemaining, Phase: op.Phase,
				LogAction: op.LogAction, Message: op.Message,
			})
		case gameplayeffects.OpCounterAdd:
			if err := effectcatalog.ValidateCounterAdd(op.Key, op.Delta, op.Min, op.Max); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			child.CounterAdds = append(child.CounterAdds, simulator.CounterAdd{Key: op.Key, Delta: op.Delta, Min: op.Min, Max: op.Max})
		case gameplayeffects.OpNotify:
			if op.Target.Selector != gameplayeffects.TargetSelf {
				return nil, fmt.Errorf("effect %q combat notification requires SELF target", op.BlockID)
			}
			if err := effectcatalog.ValidateNotifyMessage(op.Message); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			logAction := op.LogAction
			if logAction == "" {
				logAction = "EFFECT_NOTIFY"
			}
			if err := effectcatalog.ValidateCombatLogAction(logAction); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			child.CombatNotifications = append(child.CombatNotifications, simulator.CombatNotification{Action: logAction, Message: op.Message, Phase: op.Phase, EffectID: fmt.Sprintf("%s:%d:%d", op.BlockID, op.BlockIndex, op.OperationIndex)})
		case gameplayeffects.OpShield:
			if op.Target.Selector != gameplayeffects.TargetSelf {
				return nil, fmt.Errorf("effect %q shields an unsupported combat target", op.BlockID)
			}
			switch strings.ToUpper(op.SubType) {
			case "", "FLAT":
				child.Defense += op.Value
			case "PERCENT":
				child.Defense += actor.MaxHP * percentFraction(op.Value)
			case "MULTIPLIER":
				child.ShieldMult += op.Value - 1
			default:
				return nil, fmt.Errorf("effect %q has unsupported combat shield subtype %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpHeal:
			switch strings.ToUpper(op.SubType) {
			case "", "FLAT":
				if op.Target.Selector != gameplayeffects.TargetSelf {
					return nil, fmt.Errorf("effect %q heals an unsupported combat target", op.BlockID)
				}
				child.Healing += op.Value
			case "PERCENT":
				if op.Target.Selector != gameplayeffects.TargetSelf {
					return nil, fmt.Errorf("effect %q heals an unsupported combat target", op.BlockID)
				}
				child.Healing += actor.MaxHP * percentFraction(op.Value)
			case "LIFESTEAL":
				if op.Target.Selector == gameplayeffects.TargetSelf {
					return nil, fmt.Errorf("effect %q has lifesteal without a hostile target", op.BlockID)
				}
				value := op.Value
				if value > 1 {
					value /= 100
				}
				child.LifeStealPct += value
			case "RESOURCE", "MANA":
				if op.Resource != "" && !strings.EqualFold(op.Resource, "MANA") {
					return nil, fmt.Errorf("effect %q has unsupported combat resource %q", op.BlockID, op.Resource)
				}
				if op.Target.Selector == gameplayeffects.TargetSelf {
					if op.SuppressLog {
						return nil, fmt.Errorf("effect %q only supports suppress_log for party mana restore", op.BlockID)
					}
					if strings.EqualFold(op.Unit, "PERCENT") {
						child.ManaRestore += actor.MaxMana * percentFraction(op.Value)
					} else if op.Unit == "" || strings.EqualFold(op.Unit, "FLAT") {
						child.ManaRestore += op.Value
					} else {
						return nil, fmt.Errorf("effect %q has unsupported combat mana unit %q", op.BlockID, op.Unit)
					}
				} else if op.Target.Selector == gameplayeffects.TargetAllAllies {
					if child.ManaRestoreSuppressLogSet && child.ManaRestoreSuppressLog != op.SuppressLog {
						return nil, fmt.Errorf("effect %q mixes suppress_log policies for grouped party mana restores", op.BlockID)
					}
					if strings.EqualFold(op.Unit, "PERCENT") {
						child.ManaRestoreTargetPercent += percentFraction(op.Value)
					} else if op.Unit == "" || strings.EqualFold(op.Unit, "FLAT") {
						child.ManaRestoreTargetFlat += op.Value
					} else {
						return nil, fmt.Errorf("effect %q has unsupported combat mana unit %q", op.BlockID, op.Unit)
					}
					child.ManaRestoreSuppressLog = op.SuppressLog
					child.ManaRestoreSuppressLogSet = true
				} else {
					return nil, fmt.Errorf("effect %q restores mana to unsupported combat target %q", op.BlockID, op.Target.Selector)
				}
			default:
				return nil, fmt.Errorf("effect %q has unsupported combat heal subtype %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpTransferResource:
			if strings.EqualFold(op.Resource, "HP") && len(op.Conditions) > 0 {
				return nil, fmt.Errorf("effect %q HP transfer cannot evaluate Block conditions", op.BlockID)
			}
			if op.Target.Selector != gameplayeffects.TargetSingle {
				return nil, fmt.Errorf("effect %q has unsupported resource transfer %q/%q/%q/%v", op.BlockID, op.Resource, op.SubType, op.Target.Selector, op.Value)
			}
			if err := effectcatalog.ValidateResourceTransfer(op.SubType, op.Resource, op.Value, op.MinAmount, op.Phase, op.LogAction, op.Message); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			child.ResourceTransfers = append(child.ResourceTransfers, simulator.ResourceTransfer{
				Resource: op.Resource, SubType: op.SubType, Value: op.Value,
				MinAmount: op.MinAmount, Phase: op.Phase, Message: op.Message, LogAction: op.LogAction,
			})
		case gameplayeffects.OpResourceConversion:
			if strings.EqualFold(op.SubType, "CURRENT_HP_TO_SHIELD") {
				if err := effectcatalog.ValidateResourceConversion(op.SubType, op.Value, op.MinRemaining, op.LogAction, op.Message); err != nil {
					return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
				}
				child.ResourceConversions = append(child.ResourceConversions, simulator.ResourceConversion{Fraction: op.Value, MinRemaining: op.MinRemaining, Message: op.Message, LogAction: op.LogAction})
			} else if strings.EqualFold(op.SubType, "DISPEL_TO_RESOURCES") {
				child.Mechanics = append(child.Mechanics, simulator.CombatMechanic{
					Trigger:    string(op.Trigger),
					ActionCode: "DREAM_UNRAVEL",
					Params:     append([]byte(nil), op.Params...),
					Values:     op.Values,
				})
			} else {
				return nil, fmt.Errorf("effect %q has unsupported resource conversion subtype %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpOutcomeRoulette:
			code := "LUCK_ROULETTE"
			if strings.EqualFold(op.SubType, "CHAOS") {
				code = "CHAOS_GAMBLE"
			}
			child.Mechanics = append(child.Mechanics, simulator.CombatMechanic{
				Trigger:    string(op.Trigger),
				ActionCode: code,
				Params:     append([]byte(nil), op.Params...),
				Values:     op.Values,
			})
		case gameplayeffects.OpDirectHP:
			switch strings.ToUpper(op.SubType) {
			case "MAX_HP_PERCENT":
				child.DamageHPPercent += percentFraction(op.Value)
			case "SHIELD_STRIP":
				child.ShieldStripPercent += percentFraction(op.Value)
				child.ShieldStripIgnoreRankSuppression = op.RankSuppression != nil && !*op.RankSuppression
			default:
				return nil, fmt.Errorf("effect %q has unsupported direct HP subtype %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpBarrier:
			if op.Target.Selector != gameplayeffects.TargetSelf {
				return nil, fmt.Errorf("effect %q creates a barrier on an unsupported combat target", op.BlockID)
			}
			if strings.EqualFold(op.SubType, "PERCENT") {
				child.Defense += actor.MaxHP * percentFraction(op.Value)
			} else if op.SubType == "" || strings.EqualFold(op.SubType, "FLAT") {
				child.Defense += op.Value
			} else {
				return nil, fmt.Errorf("effect %q has unsupported combat barrier subtype %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpCleanse:
			if op.Target.Selector != gameplayeffects.TargetSelf && op.Target.Selector != gameplayeffects.TargetAllAllies {
				return nil, fmt.Errorf("effect %q cleanses an unsupported combat target", op.BlockID)
			}
			switch strings.ToUpper(op.SubType) {
			case "", "ALL", "DEBUFF":
				child.CleanseDebuffs = true
			case "STATUS":
				if err := effectcatalog.ValidateCombatStatusID(op.StatusID); err != nil {
					return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
				}
				child.CleanseStatusIDs = append(child.CleanseStatusIDs, strings.ToUpper(op.StatusID))
			default:
				return nil, fmt.Errorf("effect %q has unsupported cleanse subtype %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpDispel:
			if err := effectcatalog.ValidateDispel(op.SubType, op.Value, op.LogAction, op.Message); err != nil {
				return nil, fmt.Errorf("effect %q: %w", op.BlockID, err)
			}
			if strings.EqualFold(op.SubType, "ALL_BUFFS") {
				child.DispelBuffs = -1
			} else if strings.EqualFold(op.SubType, "PROTECTION") {
				child.DispelProtections = true
			} else {
				child.DispelBuffs = int(op.Value)
			}
			child.DispelLogAction = op.LogAction
			child.DispelMessage = op.Message
		case gameplayeffects.OpStatus:
			if op.Status == nil {
				return nil, fmt.Errorf("effect %q has an unresolved combat status", op.BlockID)
			}
			status := *op.Status
			status.ApplicationMode = op.ApplicationMode
			status.Message = op.Message
			status.LogAction = op.LogAction
			status.SuppressLog = op.SuppressLog
			status.Context = "combat"
			status.EffectID = spec.ID + ":" + status.EffectID
			if status.Chance > 0 && !strings.EqualFold(status.ApplicationMode, "DIRECT") {
				status.Chance += 0.05 * float64(max(0, spec.Rank-1))
				if status.Chance > 0.95 {
					status.Chance = 0.95
				}
				status.Turns += max(0, spec.Rank-1) / 3
			}
			child.StatusOps = append(child.StatusOps, status)
		case gameplayeffects.OpDisplace:
			subType := strings.ToUpper(op.SubType)
			wantSelf := subType == "RETREAT"
			if wantSelf != (op.Target.Selector == gameplayeffects.TargetSelf) || (!wantSelf && op.Target.Selector != gameplayeffects.TargetSingle) {
				return nil, fmt.Errorf("effect %q has unsupported target %q for displacement %q", op.BlockID, op.Target.Selector, subType)
			}
			steps := int(op.Value)
			if op.Value < 0 || float64(steps) != op.Value {
				return nil, fmt.Errorf("effect %q has invalid displacement value %v; expected a non-negative integer", op.BlockID, op.Value)
			}
			params := []byte(fmt.Sprintf(`{"direction":%q,"steps":%d}`, subType, steps))
			child.Mechanics = append(child.Mechanics, simulator.NewCombatMechanic(string(op.Trigger), "DISPLACE", params))
		case gameplayeffects.OpHitRule:
			switch strings.ToUpper(op.SubType) {
			case "MULTI_HIT":
				var params struct {
					Count int `json:"count"`
				}
				if err := json.Unmarshal(op.Params, &params); err != nil || params.Count < 2 || params.Count > 10 {
					return nil, fmt.Errorf("effect %q has invalid multi-hit count", op.BlockID)
				}
				child.MultiHit = params.Count
			case "GUARANTEED":
				child.GuaranteedHit = true
			case "BYPASS_SHIELD":
				child.Mechanics = append(child.Mechanics, simulator.CombatMechanic{Trigger: "ON_HIT", ActionCode: "BYPASS_SHIELD"})
			default:
				return nil, fmt.Errorf("effect %q has unsupported hit rule %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpUnique:
			switch strings.ToUpper(op.Code) {
			case "GUARANTEED_HIT":
				child.Mechanics = append(child.Mechanics, simulator.CombatMechanic{Trigger: "ON_HIT", ActionCode: "GUARANTEED_HIT"})
			default:
				child.Mechanics = append(child.Mechanics, simulator.CombatMechanic{
					Trigger:    string(op.Trigger),
					ActionCode: op.Code,
					Params:     append([]byte(nil), op.Params...),
					Values:     op.Values,
				})
			}
		default:
			return nil, fmt.Errorf("effect %q has unsupported combat operation %q", op.BlockID, op.Type)
		}
	}
	result := make([]simulator.SimActionOperation, 0, len(actions))
	for i, action := range actions {
		if len(action.DamageModifiers) > 0 && action.Damage <= 0 && action.DamageHPPercent <= 0 && action.ShieldStripPercent <= 0 && action.PhantomDmg <= 0 {
			return nil, fmt.Errorf("effect %q has a damage modifier without a damage operation", compiledByGroup[i].BlockID)
		}
		if action.LifeStealPct > 0 && action.Damage <= 0 && action.DamageHPPercent <= 0 && action.ShieldStripPercent <= 0 {
			return nil, fmt.Errorf("effect %q has lifesteal without a damaging operation", compiledByGroup[i].BlockID)
		}
		conditions := append([]gameplayeffects.Condition(nil), groups[i].conditions...)
		action.OperationConditions = conditions
		result = append(result, simulator.SimActionOperation{Compiled: compiledByGroup[i], SourceGuID: spec.ID, SourceRank: spec.Rank, Action: &action})
	}
	return result, nil
}

func effectcatalogCombatTarget(target gameplayeffects.Selector) bool {
	switch target {
	case gameplayeffects.TargetSelf, gameplayeffects.TargetSingle, gameplayeffects.TargetRow, gameplayeffects.TargetCol, gameplayeffects.TargetAllEnemies, gameplayeffects.TargetAllAllies, gameplayeffects.TargetAll:
		return true
	default:
		return false
	}
}

func percentFraction(value float64) float64 {
	if value > 1 {
		return value / 100
	}
	return value
}
