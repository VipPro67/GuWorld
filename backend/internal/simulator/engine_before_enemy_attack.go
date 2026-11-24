package simulator

import (
	"fmt"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

// applyForgetIntentInterrupt resolves the legacy BEFORE_ENEMY_ATTACK reaction
// at the same point in action resolution. Keeping it behind a phase hook gives
// typed reactions a stable dispatch seam without changing RNG order.
func (r *combatActionRunner) applyForgetIntentInterrupt(ctx combatPhaseContext) {
	actor, action := ctx.actor, ctx.action
	if actor == nil || action == nil || action.IsEffectOperation {
		return
	}
	if !hasEligibleHostileDamage(actor, action, ctx.targets, r.turn) || !guActionHasAvailableHunger(actor, action) {
		return
	}
	for _, defender := range ctx.targets {
		if defender == nil || defender.HP <= 0 || defender.IsDead {
			continue
		}
		typedSealFound := false
		for _, effect := range defender.PassiveOperationEffects {
			if effect.Compiled.Type != gameplayeffects.OpGuSeal || effect.Compiled.Trigger != gameplayeffects.TriggerBeforeEnemyAttack || effect.Compiled.GuSeal == nil {
				continue
			}
			typedSealFound = true
			params := effect.Compiled.GuSeal
			r.applyGuSeal(actor, action, defender, GuSealOperation{CounterKey: params.CounterKey, Chance: params.Chance, Turns: params.Turns, MaxActivations: params.MaxActivations, BackfireAtLimit: params.BackfireAtLimit, BackfireTurns: params.BackfireTurns})
		}
		if typedSealFound || !hasMechanic(defender.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT") {
			continue
		}
		r.applyGuSeal(actor, action, defender, GuSealOperation{
			CounterKey:      "forget_intent_success",
			Chance:          mechanicFloat(defender.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "chance", 0),
			Turns:           mechanicInt(defender.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "disable_target_gu_turns", 1),
			MaxActivations:  mechanicInt(defender.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "max_successful_activations", 8),
			BackfireAtLimit: mechanicBool(defender.Mechanics, "BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT", "backfire_after_max", true),
			BackfireTurns:   1,
		})
	}
}

func (r *combatActionRunner) applyGuSeal(actor *CombatantAuto, action *SimAction, defender *CombatantAuto, seal GuSealOperation) {
	if immuneToHostileEffects(actor) {
		*r.turnLogs = append(*r.turnLogs, CombatLog{
			Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name,
			Action: "REVERSE_NEGATE",
			Detail: fmt.Sprintf("Nghịch Chuyển Cổ đảo nghịch [Lãng Quên Ý Niệm] của %s về hư vô!", defender.Name),
		})
		return
	}
	roll := r.rng.Float64()
	if roll >= seal.Chance {
		*r.turnLogs = append(*r.turnLogs, CombatLog{
			Turn: r.turn, ActorId: defender.ID, ActorName: defender.Name,
			Action: "FORGET_INTENT_MISS",
			Detail: fmt.Sprintf("[Lãng Quên Ý Niệm] của %s không kích hoạt: roll %.0f%% >= tỉ lệ %.0f%%!", defender.Name, roll*100, seal.Chance*100),
		})
		return
	}
	if defender.CustomState == nil {
		defender.CustomState = make(map[string]int)
	}
	counterKey := seal.CounterKey
	if counterKey == "" {
		counterKey = "forget_intent_success"
	}
	maxSuccesses := seal.MaxActivations
	if defender.CustomState[counterKey] >= maxSuccesses {
		if seal.BackfireAtLimit {
			sealGuTurns(defender, seal.BackfireTurns)
			*r.turnLogs = append(*r.turnLogs, CombatLog{
				Turn: r.turn, ActorId: defender.ID, ActorName: defender.Name,
				Action: "FORGET_INTENT_BACKFIRE",
				Detail: fmt.Sprintf("%s đã lãng quên quá nhiều (%d lần), bị phản phệ phong ấn Cổ!", defender.Name, maxSuccesses),
			})
		}
		return
	}
	defender.CustomState[counterKey]++
	sealGuTurns(actor, seal.Turns)
	*r.turnLogs = append(*r.turnLogs, CombatLog{
		Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name,
		Action: "FORGET_INTENT",
		Detail: fmt.Sprintf("%s quên mất ý niệm sau đòn này, [%s] bị phong ấn %d lượt! (%d/%d)", actor.Name, action.Name, seal.Turns, defender.CustomState[counterKey], maxSuccesses),
	})
}
