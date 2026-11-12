package simulator

import (
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

func coalesceKillerMoveDamageEffects(parent *SimAction) []SimActionOperation {
	if parent == nil {
		return nil
	}
	result := make([]SimActionOperation, 0, len(parent.OperationEffects))
	groupIndex := make(map[string]int)
	for _, effect := range parent.OperationEffects {
		action := effect.Action
		if action == nil || effect.Compiled.Type != gameplayeffects.OpDamage || !strings.EqualFold(effect.Compiled.SubType, "FLAT") && effect.Compiled.SubType != "" {
			result = append(result, effect)
			continue
		}
		target := "SINGLE"
		bypass := false
		if effect.Compiled.Target != nil {
			target = string(effect.Compiled.Target.Selector)
			bypass = effect.Compiled.Target.BypassFrontline
		} else if action.TargetType != "" {
			target = action.TargetType
		}
		path := effect.Compiled.Path
		conditionKey := fmt.Sprintf("%v", effect.Compiled.Conditions)
		key := fmt.Sprintf("%s|%t|%s|%s", target, bypass, path, conditionKey)
		if index, ok := groupIndex[key]; ok {
			group := &result[index]
			if group.Action != nil {
				group.Action.Damage += action.Damage
				if effect.SourceRank > group.SourceRank {
					group.SourceRank = effect.SourceRank
				}
			}
			continue
		}
		combined := *action
		combined.DamageMult = parent.DamageMult
		combined.Path = append([]string(nil), parent.Path...)
		groupIndex[key] = len(result)
		effect.Action = &combined
		result = append(result, effect)
	}
	return result
}

func filterOperationTargets(actor *CombatantAuto, targets []*CombatantAuto, conditions []gameplayeffects.Condition, sourceRank int, paths []string, turn int) []*CombatantAuto {
	if len(conditions) == 0 {
		return targets
	}
	filtered := make([]*CombatantAuto, 0, len(targets))
	for _, target := range targets {
		if target != nil && operationConditionsMatch(actor, target, conditions, sourceRank, paths, turn) {
			filtered = append(filtered, target)
		}
	}
	return filtered
}

func operationConditionsMatch(actor, target *CombatantAuto, conditions []gameplayeffects.Condition, sourceRank int, paths []string, turn int) bool {
	values := gameplayeffects.ConditionValues{
		"RANK": float64(sourceRank),
		"TURN": float64(turn),
	}
	if actor != nil {
		if actor.MaxHP > 0 {
			values["HP_PERCENT"] = actor.HP / actor.MaxHP
			values["ACTOR_HP_PERCENT"] = actor.HP / actor.MaxHP
		}
		if actor.MaxMana > 0 {
			values["MANA_PERCENT"] = actor.Mana / actor.MaxMana
		}
		if len(paths) > 0 {
			values["MASTERY"] = float64(gamefx.PathMasteryLevel(actor.PathMasteries[gamefx.CleanPath(paths[0])]))
		}
	}
	if target != nil && target.MaxHP > 0 {
		values["TARGET_HP_PERCENT"] = target.HP / target.MaxHP
	}
	matches, err := gameplayeffects.ConditionsMatch(conditions, values)
	return err == nil && matches
}
