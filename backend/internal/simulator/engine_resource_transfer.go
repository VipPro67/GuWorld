package simulator

import (
	"fmt"
	"math"
	"strings"
)

func (r *combatActionRunner) applyResourceTransfer(actor, target *CombatantAuto, transfer ResourceTransfer) {
	if actor == nil || target == nil {
		return
	}
	var amount float64
	switch {
	case transfer.Resource == "MANA" && transfer.SubType == "PERCENT_OF_TARGET" && transfer.Value > 0 && transfer.Value <= 1:
		amount = target.Mana * transfer.Value
		if amount <= 0 {
			return
		}
		target.Mana -= amount
		actor.Mana = math.Min(actor.MaxMana, actor.Mana+amount)
	case transfer.Resource == "HP" && transfer.SubType == "PERCENT_OF_MAX_HP" && transfer.Value > 0 && transfer.Value <= 1 && transfer.Phase == "LATE_RIDER":
		amount = math.Max(target.MaxHP*transfer.Value, transfer.MinAmount)
		if amount <= 0 {
			return
		}
		target.HP = math.Max(0, target.HP-amount)
		actor.HP = math.Min(actor.MaxHP, actor.HP+amount)
	case transfer.Resource == "HP" && transfer.SubType == "PERCENT_OF_TARGET_SHIELD" && transfer.Value > 0 && transfer.Value <= 1 && transfer.Phase == "EARLY_RIDER":
		amount = target.VirtualShield * transfer.Value
		if amount <= 0 {
			return
		}
		target.VirtualShield = math.Max(0, target.VirtualShield-amount)
		actor.HP = math.Min(actor.MaxHP, actor.HP+amount)
	default:
		return
	}

	message := transfer.Message
	if strings.TrimSpace(message) == "" {
		message = "{actor} transfers {amount} MANA from {target} to self."
	}
	detail := strings.NewReplacer(
		"{actor}", actor.Name,
		"{target}", target.Name,
		"{amount}", fmt.Sprintf("%.0f", amount),
		"{resource}", transfer.Resource,
	).Replace(message)
	logAction := transfer.LogAction
	if logAction == "" {
		logAction = "RESOURCE_TRANSFER"
	}
	loggedDamage := 0.0
	if transfer.Resource == "HP" && transfer.SubType == "PERCENT_OF_MAX_HP" {
		loggedDamage = amount
	}
	*r.turnLogs = append(*r.turnLogs, CombatLog{
		Turn:      r.turn,
		ActorId:   actor.ID,
		ActorName: actor.Name,
		Action:    logAction,
		TargetId:  target.ID,
		Damage:    loggedDamage,
		Detail:    detail,
	})
}

func (r *combatActionRunner) applyOperationResourceTransfers(ctx combatPhaseContext, phase string) {
	if ctx.actor == nil || ctx.action == nil || ctx.firstTarget == nil {
		return
	}
	apply := func(action *SimAction) {
		if action == nil {
			return
		}
		remaining := action.ResourceTransfers[:0]
		for _, transfer := range action.ResourceTransfers {
			if transfer.Resource == "HP" && transfer.Phase == phase {
				r.applyResourceTransfer(ctx.actor, ctx.firstTarget, transfer)
				continue
			}
			remaining = append(remaining, transfer)
		}
		action.ResourceTransfers = remaining
	}
	apply(ctx.action)
	if !ctx.action.IsEffectOperation {
		for i := range ctx.action.OperationEffects {
			apply(ctx.action.OperationEffects[i].Action)
		}
	}
}
