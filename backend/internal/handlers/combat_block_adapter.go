package handlers

import (
	"fmt"
	"strings"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

type combatOperationGroup struct {
	conditions []gameplayeffects.Condition
}

// adaptBlockCombatOperations turns compiled ON_ACTION operations into the
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
		if op.Trigger != gameplayeffects.TriggerOnAction {
			continue
		}
		if reason := unsupportedConditionReason("TURN_BASED_COMBAT", op.Conditions); reason != "" {
			return nil, fmt.Errorf("effect %q has unsupported turn-based conditions: %s", op.BlockID, reason)
		}
		if op.Type == gameplayeffects.OpUnique && len(op.Conditions) > 0 {
			return nil, fmt.Errorf("effect %q native combat hook cannot evaluate Block conditions", op.BlockID)
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
		case gameplayeffects.OpDamage:
			switch strings.ToUpper(op.SubType) {
			case "", "FLAT":
				child.Damage += op.Value
			case "MULTIPLIER":
				child.DamageMult += op.Value - 1
			case "MAX_HP_PERCENT":
				child.DamageHPPercent += percentFraction(op.Value)
			default:
				return nil, fmt.Errorf("effect %q has unsupported combat damage subtype %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpShield:
			if op.Target.Selector != gameplayeffects.TargetSelf {
				return nil, fmt.Errorf("effect %q shields an unsupported combat target", op.BlockID)
			}
			switch strings.ToUpper(op.SubType) {
			case "", "FLAT":
				child.Defense += op.Value
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
				if op.Target.Selector != gameplayeffects.TargetSelf {
					return nil, fmt.Errorf("effect %q restores mana to an unsupported combat target", op.BlockID)
				}
				if op.Resource != "" && !strings.EqualFold(op.Resource, "MANA") {
					return nil, fmt.Errorf("effect %q has unsupported combat resource %q", op.BlockID, op.Resource)
				}
				child.ManaRestore += op.Value
			default:
				return nil, fmt.Errorf("effect %q has unsupported combat heal subtype %q", op.BlockID, op.SubType)
			}
		case gameplayeffects.OpDirectHP:
			switch strings.ToUpper(op.SubType) {
			case "MAX_HP_PERCENT":
				child.DamageHPPercent += percentFraction(op.Value)
			case "SHIELD_STRIP":
				child.ShieldStripPercent += percentFraction(op.Value)
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
			if op.Target.Selector != gameplayeffects.TargetSelf {
				return nil, fmt.Errorf("effect %q cleanses an unsupported combat target", op.BlockID)
			}
			child.CleanseDebuffs = true
		case gameplayeffects.OpStatus:
			if op.Status == nil {
				return nil, fmt.Errorf("effect %q has an unresolved combat status", op.BlockID)
			}
			status := *op.Status
			if status.Chance > 0 {
				status.Chance += 0.05 * float64(max(0, spec.Rank-1))
				if status.Chance > 0.95 {
					status.Chance = 0.95
				}
				status.Turns += max(0, spec.Rank-1) / 3
			}
			child.StatusOps = append(child.StatusOps, status)
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
	case gameplayeffects.TargetSelf, gameplayeffects.TargetSingle, gameplayeffects.TargetRow, gameplayeffects.TargetCol, gameplayeffects.TargetAllEnemies, gameplayeffects.TargetAll:
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
