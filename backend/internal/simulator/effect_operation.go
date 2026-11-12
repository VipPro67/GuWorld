package simulator

import (
	"strings"
)

// hasEligibleHostileDamage reports whether an action has a hostile damage
// operation whose target and conditions currently resolve. A Block v2 parent
// must not trigger hostile hit-intent hooks from its aggregate base damage when
// all authored damage operations are conditional and currently ineligible.
func hasEligibleHostileDamage(actor *CombatantAuto, action *SimAction, hostileTargets []*CombatantAuto, turn int) bool {
	if action == nil {
		return false
	}
	if action.ActionType == "BASIC_ATTACK" {
		return true
	}
	if len(action.OperationEffects) > 0 {
		for _, effect := range action.OperationEffects {
			if effect.Action == nil || !isHostileDamageOperation(effect.Action) {
				continue
			}
			targets := resolveTargets(hostileTargets, effect.Action)
			conditions := effect.Compiled.Conditions
			if len(conditions) == 0 {
				conditions = effect.Action.OperationConditions
			}
			if len(conditions) > 0 {
				rank := effect.SourceRank
				if rank <= 0 {
					rank = effect.Action.Rank
				}
				targets = filterOperationTargets(actor, targets, conditions, rank, effect.Action.Path, turn)
			}
			if len(targets) > 0 {
				return true
			}
		}
		return false
	}
	if !isHostileDamageOperation(action) {
		return false
	}
	targets := resolveTargets(hostileTargets, action)
	if len(action.OperationConditions) > 0 {
		rank := action.Rank
		if rank <= 0 {
			rank = actor.Rank
		}
		targets = filterOperationTargets(actor, targets, action.OperationConditions, rank, action.Path, turn)
	}
	return len(targets) > 0
}

func isHostileDamageOperation(action *SimAction) bool {
	if action == nil || strings.EqualFold(strings.TrimSpace(action.TargetType), "SELF") {
		return false
	}
	return action.Damage > 0 || action.DamageHPPercent > 0 || action.ShieldStripPercent > 0 || action.PhantomDmg > 0
}

// guActionHasAvailableHunger prevents pre-action hit hooks from firing for a
// Gu action that the actor cannot pay for because its Gu is exhausted.
func guActionHasAvailableHunger(actor *CombatantAuto, action *SimAction) bool {
	if actor == nil || action == nil || action.ActionType != "GU_SKILL" || actor.GuHunger == nil {
		return true
	}
	key := action.InstanceID
	if key == "" {
		key = action.SourceID
	}
	hunger, exists := actor.GuHunger[key]
	if !exists && action.SourceID != "" {
		hunger, exists = actor.GuHunger[action.SourceID]
	}
	return !exists || hunger > 0
}
