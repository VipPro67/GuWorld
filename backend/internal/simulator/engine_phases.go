package simulator

import (
	"fmt"
	"math"
	"strings"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

type combatPhase uint8

const (
	combatPhasePreAction combatPhase = iota
	combatPhaseBeforeEnemyAttack
	combatPhaseOnCast
	combatPhaseBeforeHit
	combatPhaseBonusStrike
	combatPhasePreChain
	combatPhaseChainDamage
	combatPhasePassiveDamageModifier
	combatPhaseOnActionRider
	combatPhasePreDamage
	combatPhaseDamage
	combatPhaseOnHitRider
	combatPhaseLethal
	combatPhaseDeath
)

type combatPhaseContext struct {
	actor       *CombatantAuto
	action      *SimAction
	damageScale *float64
	firstTarget *CombatantAuto
	targets     []*CombatantAuto
	allies      []*CombatantAuto
	didDamage   *bool
}

type combatPhaseHook func(*combatActionRunner, combatPhaseContext)

// combatPhaseHooks fixes the execution order for handlers registered at each
// phase. The table is immutable after package initialization.
var combatPhaseHooks = map[combatPhase][]combatPhaseHook{
	combatPhaseBeforeEnemyAttack: {
		(*combatActionRunner).applyForgetIntentInterrupt,
	},
	combatPhaseOnCast: {
		(*combatActionRunner).applyOnCastOperations,
	},
	combatPhaseBonusStrike: {
		(*combatActionRunner).applyBonusStrikes,
	},
	combatPhaseChainDamage: {
		(*combatActionRunner).applyChainDamage,
	},
	combatPhasePreChain: {
		(*combatActionRunner).applyPreChainDamageModifiers,
		(*combatActionRunner).applyPreChainStatusOps,
		(*combatActionRunner).applyPreChainNotifications,
	},
	combatPhasePassiveDamageModifier: {
		(*combatActionRunner).applyPassiveDamageModifiers,
	},
	combatPhaseOnActionRider: {
		(*combatActionRunner).applyKnockback,
		(*combatActionRunner).applyPull,
		(*combatActionRunner).applySwap,
		(*combatActionRunner).applyRetreat,
		(*combatActionRunner).applyDispel,
		(*combatActionRunner).applyDamageModifiers,
		(*combatActionRunner).applyDefensePenetration,
	},
	combatPhasePreDamage: {
		(*combatActionRunner).applyShieldStrip,
		(*combatActionRunner).applyPreDamageStatusOps,
		(*combatActionRunner).applyTargetDamageModifiers,
	},
}

func (r *combatActionRunner) applyOnCastOperations(ctx combatPhaseContext) {
	if ctx.actor == nil || ctx.action == nil {
		return
	}
	foundDispel := false
	for _, effect := range ctx.action.OperationEffects {
		if effect.Compiled.Trigger != gameplayeffects.TriggerOnCast || effect.Action == nil {
			continue
		}
		if (effect.Action.DispelBuffs != 0 || effect.Action.DispelProtections) && ctx.firstTarget != nil {
			foundDispel = true
			r.applyDispel(combatPhaseContext{actor: ctx.actor, action: effect.Action, firstTarget: ctx.firstTarget, targets: []*CombatantAuto{ctx.firstTarget}})
		}
		conversionSucceeded := true
		for _, conversion := range effect.Action.ResourceConversions {
			available := math.Max(0, ctx.actor.HP-conversion.MinRemaining)
			amount := math.Min(ctx.actor.HP*conversion.Fraction, available)
			if amount <= 0 {
				conversionSucceeded = false
				break
			}
			ctx.actor.HP -= amount
			ctx.actor.VirtualShield += amount
			detail := strings.NewReplacer("{actor}", ctx.actor.Name, "{amount}", fmt.Sprintf("%.0f", amount)).Replace(conversion.Message)
			*r.turnLogs = append(*r.turnLogs, CombatLog{Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name, Action: conversion.LogAction, Detail: detail})
		}
		if !conversionSucceeded {
			continue
		}
		if len(effect.Action.StatusOps) > 0 {
			applyStatusOps(ctx.actor, []*CombatantAuto{ctx.actor}, effect.Action.StatusOps, effect.SourceRank, r.turn, r.rng, r.turnLogs)
		}
	}
	if foundDispel || ctx.firstTarget == nil || !hasMechanic(ctx.action.Mechanics, "ON_CAST", "UNRAVEL_RIDDLE") {
		return
	}
	dispelledCount := 0
	keptEffects := make([]Effect, 0, len(ctx.firstTarget.ActiveEffects))
	for _, effect := range ctx.firstTarget.ActiveEffects {
		if effect.Type == "buff" && dispelledCount < 2 {
			dispelledCount++
			*r.turnLogs = append(*r.turnLogs, CombatLog{
				Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
				Action: "DISPEL", TargetId: ctx.firstTarget.ID,
				Detail: fmt.Sprintf("%s hóa giải hiệu ứng lợi ích [%s] của %s!", ctx.actor.Name, effect.ID, ctx.firstTarget.Name),
			})
			continue
		}
		keptEffects = append(keptEffects, effect)
	}
	ctx.firstTarget.ActiveEffects = keptEffects
	*r.turnLogs = append(*r.turnLogs, CombatLog{
		Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
		Action: "INTERRUPT", TargetId: ctx.firstTarget.ID,
		Detail: fmt.Sprintf("%s sử dụng Giải Mê Cổ ngắt đòn tụ lực sát chiêu của %s!", ctx.actor.Name, ctx.firstTarget.Name),
	})
}

func (r *combatActionRunner) applyChainDamage(ctx combatPhaseContext) {
	if ctx.actor == nil || ctx.action == nil || ctx.firstTarget == nil {
		return
	}
	var configured []ChainDamage
	for _, effect := range ctx.action.OperationEffects {
		if effect.Action != nil {
			configured = append(configured, effect.Action.ChainDamages...)
		}
	}
	if len(configured) > 0 {
		for _, chain := range configured {
			hit := 0
			for _, target := range ctx.targets {
				if target == nil || target.HP <= 0 || target == ctx.firstTarget {
					continue
				}
				damage := ctx.action.Damage * chain.Fraction
				if damage < chain.MinDamage {
					damage = chain.MinDamage
				}
				target.HP -= damage
				if target.HP < 0 {
					target.HP = 0
				}
				detail := strings.NewReplacer(
					"{actor}", ctx.actor.Name,
					"{target}", target.Name,
					"{damage}", fmt.Sprintf("%.0f", damage),
				).Replace(chain.Message)
				*r.turnLogs = append(*r.turnLogs, CombatLog{
					Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
					Action: chain.LogAction, TargetId: target.ID, Damage: damage, Detail: detail,
				})
				hit++
				if hit >= chain.TargetCount {
					break
				}
			}
		}
		return
	}
	if !hasMechanic(ctx.action.Mechanics, "ON_ACTION", "CHAIN_LIGHTNING") && !hasMechanic(ctx.action.Mechanics, "ON_HIT", "CHAIN_LIGHTNING") {
		return
	}
	if len(ctx.targets) <= 1 {
		return
	}
	for _, target := range ctx.targets {
		if target == nil || target.HP <= 0 || target == ctx.firstTarget {
			continue
		}
		damage := ctx.action.Damage * 0.50
		if damage < 30 {
			damage = 30
		}
		target.HP -= damage
		if target.HP < 0 {
			target.HP = 0
		}
		*r.turnLogs = append(*r.turnLogs, CombatLog{
			Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
			Action: "CHAIN_LIGHTNING", TargetId: target.ID, Damage: damage,
			Detail: fmt.Sprintf("[Lôi Đạo - Lôi Điện Truyền Dẫn] Tia lôi điện phóng lan sang %s, gây thêm %.0f sát thương điện giật!", target.Name, damage),
		})
		break
	}
}

// applyBonusStrikes evaluates typed follow-up hits at the legacy phantom-strike
// position so their shared RNG draw remains in the same deterministic order.
func (r *combatActionRunner) applyBonusStrikes(ctx combatPhaseContext) {
	if ctx.actor == nil || ctx.action == nil || ctx.firstTarget == nil {
		return
	}
	for _, strike := range ctx.action.BonusStrikes {
		if applyBonusStrikeHit(r, ctx.actor, ctx.action, ctx.firstTarget, strike) && ctx.didDamage != nil {
			*ctx.didDamage = true
		}
	}
	if len(ctx.action.BonusStrikes) > 0 {
		return
	}
	if hasMechanic(ctx.action.Mechanics, "ON_ACTION", "BEAST_PHANTOM_STRIKE") || hasMechanic(ctx.action.Mechanics, "ON_HIT", "BEAST_PHANTOM_STRIKE") {
		if applyBonusStrikeHit(r, ctx.actor, ctx.action, ctx.firstTarget, BonusStrike{
			Chance: 0.45, DamageModifier: 0.75, MinDamage: 50,
			LogAction: "BEAST_PHANTOM_STRIKE",
			Message:   "[Lực Đạo - Thú Lực Hư Ảnh] Tiếng thú gầm thét kinh thiên! Hư Ảnh bồi thêm một đòn bạt sơn bạt hải lên {target}, gây thêm {damage} sát thương!",
		}) && ctx.didDamage != nil {
			*ctx.didDamage = true
		}
	}
}

func applyBonusStrikeHit(r *combatActionRunner, actor *CombatantAuto, action *SimAction, target *CombatantAuto, strike BonusStrike) bool {
	roll := r.rng.Float64()
	if roll >= strike.Chance {
		return false
	}
	damage := math.Max(action.Damage*strike.DamageModifier, strike.MinDamage)
	target.HP -= damage
	if target.HP < 0 {
		target.HP = 0
	}
	detail := strings.NewReplacer(
		"{actor}", actor.Name,
		"{target}", target.Name,
		"{damage}", fmt.Sprintf("%.0f", damage),
	).Replace(strike.Message)
	*r.turnLogs = append(*r.turnLogs, CombatLog{
		Turn: r.turn, ActorId: actor.ID, ActorName: actor.Name,
		Action: strike.LogAction, TargetId: target.ID, Damage: damage, Detail: detail,
	})
	return true
}

// applyShieldStrip runs before hit damage so the strike interacts with the
// shield left after the strip, matching the native effect timing.
func (r *combatActionRunner) applyShieldStrip(ctx combatPhaseContext) {
	if ctx.actor == nil || ctx.action == nil || ctx.action.ShieldStripPercent <= 0 {
		return
	}
	actionRank := ctx.action.Rank
	if actionRank <= 0 {
		actionRank = ctx.actor.Rank
	}
	for _, target := range ctx.targets {
		if target == nil || target.HP <= 0 || target.VirtualShield <= 0 {
			continue
		}
		stripPct := ctx.action.ShieldStripPercent
		if !ctx.action.ShieldStripIgnoreRankSuppression && target.Rank > actionRank {
			suppression, _ := statusfx.TargetSuppressionFactor(actionRank, target.Rank)
			stripPct *= suppression
		}
		strip := target.VirtualShield * stripPct
		target.VirtualShield -= strip
		if ctx.action.DispelMessage == "" {
			*r.turnLogs = append(*r.turnLogs, CombatLog{
				Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
				Action: "SHIELD_STRIP", TargetId: target.ID, Damage: strip,
				Detail: fmt.Sprintf("[%s] tước %.0f khiên của %s (còn %.0f khiên)!", ctx.action.Name, strip, target.Name, target.VirtualShield),
			})
		}
	}
}

func (r *combatActionRunner) applyDispel(ctx combatPhaseContext) {
	if ctx.actor == nil || ctx.action == nil || ctx.firstTarget == nil || (ctx.action.DispelBuffs == 0 && !ctx.action.DispelProtections) {
		return
	}
	dispelled := 0
	shieldStripped := 0.0
	if ctx.action.DispelProtections {
		ctx.firstTarget.IsInvincible = false
		ctx.firstTarget.IsLastStand = false
		ctx.firstTarget.DamageTakenReduction = 0
		shieldStripped = ctx.firstTarget.VirtualShield
		ctx.firstTarget.VirtualShield = 0
		remaining := make([]Effect, 0, len(ctx.firstTarget.ActiveEffects))
		for _, effect := range ctx.firstTarget.ActiveEffects {
			switch strings.ToLower(effect.ID) {
			case strings.ToLower(statusfx.StatusInvincible), strings.ToLower(statusfx.StatusLastStand), "invulnerable", "bat_tu", "bat_hoai":
				continue
			}
			remaining = append(remaining, effect)
		}
		ctx.firstTarget.ActiveEffects = remaining
	}
	remaining := make([]Effect, 0, len(ctx.firstTarget.ActiveEffects))
	for _, effect := range ctx.firstTarget.ActiveEffects {
		if effect.Type == "buff" && (ctx.action.DispelBuffs < 0 || dispelled < ctx.action.DispelBuffs) {
			dispelled++
			continue
		}
		remaining = append(remaining, effect)
	}
	ctx.firstTarget.ActiveEffects = remaining
	if ctx.action.DispelMessage == "" {
		return
	}
	if !ctx.action.DispelProtections {
		shieldStripped = ctx.firstTarget.VirtualShield * ctx.action.ShieldStripPercent
	}
	detail := strings.NewReplacer(
		"{actor}", ctx.actor.Name,
		"{target}", ctx.firstTarget.Name,
		"{dispelled}", fmt.Sprintf("%d", dispelled),
		"{shield_stripped}", fmt.Sprintf("%.0f", shieldStripped),
	).Replace(ctx.action.DispelMessage)
	*r.turnLogs = append(*r.turnLogs, CombatLog{
		Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
		Action: ctx.action.DispelLogAction, TargetId: ctx.firstTarget.ID, Detail: detail,
	})
}

func (r *combatActionRunner) applyDamageModifiers(ctx combatPhaseContext) {
	if ctx.actor == nil || ctx.action == nil || len(ctx.action.DamageModifiers) == 0 {
		return
	}
	r.applyDamageModifiersFrom(ctx, ctx.action.DamageModifiers)
}

func (r *combatActionRunner) applyDefensePenetration(ctx combatPhaseContext) {
	if ctx.actor == nil || ctx.action == nil || ctx.action.DefensePenetration <= 0 {
		return
	}
	ctx.actor.IgnoreDefensePct = math.Max(ctx.actor.IgnoreDefensePct, ctx.action.DefensePenetration)
}

func (r *combatActionRunner) applyTargetDamageModifiers(ctx combatPhaseContext) {
	if ctx.actor == nil || ctx.action == nil || len(ctx.action.TargetDamageModifiers) == 0 {
		return
	}
	for _, modifier := range ctx.action.TargetDamageModifiers {
		applied := false
		for _, target := range ctx.targets {
			if target == nil || target.HP <= 0 || target.IsDead {
				continue
			}
			applyDamageOutputModifier(target, modifier)
			applied = true
		}
		if !applied || modifier.Message == "" {
			continue
		}
		detail := strings.NewReplacer("{actor}", ctx.actor.Name).Replace(modifier.Message)
		*r.turnLogs = append(*r.turnLogs, CombatLog{
			Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
			Action: modifier.LogAction, Detail: detail,
		})
	}
}

func (r *combatActionRunner) applyPassiveDamageModifiers(ctx combatPhaseContext) {
	if ctx.actor == nil || ctx.action == nil || len(ctx.actor.PassiveDamageModifiers) == 0 {
		return
	}
	r.applyDamageModifiersFrom(ctx, ctx.actor.PassiveDamageModifiers)
}

// applyPassiveOperationsPhase dispatches condition-aware passives at the
// authored point in the action, preserving each legacy hook's timing.
func (r *combatActionRunner) applyPassiveOperationsPhase(ctx combatPhaseContext, phase string) {
	if ctx.actor == nil || ctx.action == nil {
		return
	}
	for _, effect := range ctx.actor.PassiveOperationEffects {
		if effect.Compiled.Phase != phase {
			continue
		}
		if !operationConditionsMatch(ctx.actor, ctx.firstTarget, effect.Compiled.Conditions, effect.SourceRank, effect.SourcePaths, r.turn) {
			continue
		}
		if effect.Action == nil {
			continue
		}
		if len(effect.Action.DamageModifiers) > 0 {
			r.applyDamageModifiersFrom(ctx, effect.Action.DamageModifiers)
		}
		for _, heal := range effect.Action.CounterHeals {
			stacks := 0
			if ctx.actor.CustomState != nil {
				stacks = ctx.actor.CustomState[heal.Key]
			}
			if stacks <= 0 || ctx.actor.MaxHP <= 0 {
				continue
			}
			ctx.actor.HP = math.Min(ctx.actor.MaxHP, ctx.actor.HP+ctx.actor.MaxHP*float64(stacks)*heal.PerCounter)
		}
		for _, statusID := range effect.Action.CleanseStatusIDs {
			cleanseCombatStatus(ctx.actor, statusID)
		}
		for _, notification := range effect.Action.CombatNotifications {
			detail := renderCounterNotification(ctx.actor, notification)
			*r.turnLogs = append(*r.turnLogs, CombatLog{
				Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
				Action: notification.Action, Detail: detail,
			})
		}
	}
}

func renderCounterNotification(actor *CombatantAuto, notification CombatNotification) string {
	stacks := 0
	if actor.CustomState != nil && notification.CounterKey != "" {
		stacks = actor.CustomState[notification.CounterKey]
	}
	damagePercent := 0.0
	healAmount := 0.0
	for _, effect := range actor.PassiveOperationEffects {
		if effect.Action == nil {
			continue
		}
		for _, modifier := range effect.Action.DamageModifiers {
			if modifier.SubType == "COUNTER_COUNT" && modifier.CounterKey == notification.CounterKey {
				multiplier := modifier.BaseMultiplier + float64(stacks)*modifier.PerCounter
				damagePercent += (multiplier - 1) * 100
			}
		}
		for _, heal := range effect.Action.CounterHeals {
			if heal.Key == notification.CounterKey {
				healAmount += float64(stacks) * heal.PerCounter * actor.MaxHP
			}
		}
	}
	return strings.NewReplacer(
		"{actor}", actor.Name,
		"{counter_count}", fmt.Sprintf("%d", stacks),
		"{damage_percent}", fmt.Sprintf("%.0f", damagePercent),
		"{heal_amount}", fmt.Sprintf("%.0f", healAmount),
	).Replace(notification.Message)
}

func cleanseCombatStatus(target *CombatantAuto, statusID string) {
	if target == nil {
		return
	}
	statusID = strings.ToUpper(strings.TrimSpace(statusID))
	removed := false
	kept := target.ActiveEffects[:0]
	for _, effect := range target.ActiveEffects {
		if strings.EqualFold(effect.ID, statusID) {
			removed = true
			continue
		}
		kept = append(kept, effect)
	}
	target.ActiveEffects = kept
	if removed {
		ComputeEffectMultipliers(target)
	}
	switch statusID {
	case statusfx.StatusStun:
		target.IsStunned = false
	case statusfx.StatusFreeze:
		target.IsFrozen = false
	}
}

func (r *combatActionRunner) applyStatusDetonations(ctx combatPhaseContext, phase string) {
	if ctx.actor == nil || ctx.action == nil || ctx.firstTarget == nil || ctx.firstTarget.HP <= 0 {
		return
	}
	var detonations []StatusDetonation
	if ctx.action.IsEffectOperation {
		detonations = append(detonations, ctx.action.StatusDetonations...)
	} else {
		detonations = append(detonations, ctx.action.StatusDetonations...)
		for _, effect := range ctx.action.OperationEffects {
			if effect.Action != nil {
				detonations = append(detonations, effect.Action.StatusDetonations...)
			}
		}
	}
	for _, detonation := range detonations {
		if detonation.Phase != phase {
			continue
		}
		stacks, strength := removeStatusStacks(ctx.firstTarget, detonation.StatusID)
		if stacks == 0 {
			continue
		}
		damage := float64(stacks)*detonation.PerStack + strength*detonation.PerStrength + ctx.firstTarget.MaxHP*detonation.MaxHPPercent
		if damage < detonation.MinDamage {
			damage = detonation.MinDamage
		}
		ctx.firstTarget.HP -= damage
		if ctx.firstTarget.HP < 0 {
			ctx.firstTarget.HP = 0
		}
		detail := strings.NewReplacer(
			"{actor}", ctx.actor.Name,
			"{target}", ctx.firstTarget.Name,
			"{stack_count}", fmt.Sprintf("%d", stacks),
			"{strength}", fmt.Sprintf("%.0f", strength),
			"{damage}", fmt.Sprintf("%.0f", damage),
		).Replace(detonation.Message)
		*r.turnLogs = append(*r.turnLogs, CombatLog{
			Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
			Action: detonation.LogAction, TargetId: ctx.firstTarget.ID,
			Damage: damage, Detail: detail,
		})
	}
}

func (r *combatActionRunner) applyHPCosts(ctx combatPhaseContext, phase string) {
	if ctx.actor == nil || ctx.action == nil || ctx.actor.HP <= 0 {
		return
	}
	var costs []HPCost
	if ctx.action.IsEffectOperation {
		costs = append(costs, ctx.action.HPCosts...)
	} else {
		costs = append(costs, ctx.action.HPCosts...)
		for _, effect := range ctx.action.OperationEffects {
			if effect.Action != nil {
				costs = append(costs, effect.Action.HPCosts...)
			}
		}
	}
	for _, cost := range costs {
		costPhase := cost.Phase
		if costPhase == "" {
			costPhase = "LATE_RIDER"
		}
		if costPhase != phase {
			continue
		}
		nominalCost := math.Max(ctx.actor.HP*cost.CurrentHPPercent, cost.MinCost)
		maxCharge := math.Max(0, ctx.actor.HP-cost.MinRemaining)
		actualCost := math.Min(nominalCost, maxCharge)
		ctx.actor.HP -= actualCost
		if ctx.actor.HP < cost.MinRemaining {
			ctx.actor.HP = cost.MinRemaining
		}
		detail := strings.NewReplacer(
			"{actor}", ctx.actor.Name,
			"{cost}", fmt.Sprintf("%.0f", nominalCost),
			"{actual_cost}", fmt.Sprintf("%.0f", actualCost),
			"{remaining_hp}", fmt.Sprintf("%.0f", ctx.actor.HP),
		).Replace(cost.Message)
		*r.turnLogs = append(*r.turnLogs, CombatLog{
			Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
			Action: cost.LogAction, Damage: nominalCost, Detail: detail,
		})
	}
}

func (r *combatActionRunner) applyDamageModifiersFrom(ctx combatPhaseContext, modifiers []DamageModifier) {
	missingHPRatio := 0.0
	if ctx.actor.MaxHP > 0 {
		missingHPRatio = 1 - ctx.actor.HP/ctx.actor.MaxHP
	}
	targetDebuffCount := 0
	if ctx.firstTarget != nil {
		for _, effect := range ctx.firstTarget.ActiveEffects {
			if effect.Type == "debuff" {
				targetDebuffCount++
			}
		}
	}
	minionCount := 0
	for _, member := range ctx.allies {
		if member != nil && member.Kind == "Minion" && member.HP > 0 {
			minionCount++
		}
	}
	for _, modifier := range modifiers {
		counterCount := 0
		if ctx.actor.CustomState != nil {
			counterCount = ctx.actor.CustomState[modifier.CounterKey]
		}
		multiplier := modifier.BaseMultiplier + missingHPRatio*modifier.PerMissingHP + float64(targetDebuffCount)*modifier.PerTargetDebuff + float64(counterCount)*modifier.PerCounter
		if modifier.SubType == "TARGET_HAS_STATUS" {
			if ctx.firstTarget == nil || !hasStatus(ctx.firstTarget, modifier.StatusID) {
				continue
			}
			multiplier = modifier.BaseMultiplier
		}
		if modifier.SubType == "MINION_COUNT" {
			if minionCount == 0 {
				continue
			}
			multiplier = modifier.BaseMultiplier + float64(minionCount)*modifier.PerMinion
		}
		if modifier.SubType == "COUNTER_COUNT" {
			multiplier = modifier.BaseMultiplier + float64(counterCount)*modifier.PerCounter
		}
		if ctx.action.Damage > 0 {
			ctx.action.Damage *= multiplier
		}
		if ctx.damageScale != nil {
			*ctx.damageScale *= multiplier
		}
		if modifier.Message == "" {
			continue
		}
		detail := strings.NewReplacer(
			"{actor}", ctx.actor.Name,
			"{target}", targetName(ctx.firstTarget),
			"{debuff_count}", fmt.Sprintf("%d", targetDebuffCount),
			"{missing_hp_percent}", fmt.Sprintf("%.0f", missingHPRatio*100),
			"{damage_percent}", fmt.Sprintf("%.0f", (multiplier-1)*100),
			"{status_id}", modifier.StatusID,
			"{minion_count}", fmt.Sprintf("%d", minionCount),
		).Replace(modifier.Message)
		*r.turnLogs = append(*r.turnLogs, CombatLog{
			Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
			Action: modifier.LogAction, Detail: detail,
		})
	}
}

func targetName(target *CombatantAuto) string {
	if target == nil {
		return "kẻ địch"
	}
	return target.Name
}

func (r *combatActionRunner) dispatchPhase(phase combatPhase, ctx combatPhaseContext) {
	for _, hook := range combatPhaseHooks[phase] {
		hook(r, ctx)
	}
}

// applyPreDamageStatusOps lands status operations flagged PRE_DAMAGE before the
// damage step, matching the timing of the legacy native hooks they replace.
// Derived multipliers are intentionally not recomputed: legacy hooks appended
// effects without recomputing, so the same action keeps its existing damage
// numbers. The ops are also not gated by SimAction.ProcChance, because native
// hooks never were.
func (r *combatActionRunner) applyPreDamageStatusOps(ctx combatPhaseContext) {
	if ctx.action == nil || ctx.actor == nil {
		return
	}
	pre := statusOpsForPhase(collectAllStatusOps(ctx.action), "PRE_DAMAGE")
	if len(pre) == 0 {
		return
	}
	r.applyAuthoredStatusOps(ctx, pre)
}

// applyPreChainStatusOps lands explicitly authored statuses immediately
// before chain riders, matching effects whose legacy proc ran at that seam.
func (r *combatActionRunner) applyPreChainStatusOps(ctx combatPhaseContext) {
	if ctx.action == nil || ctx.actor == nil {
		return
	}
	for _, effect := range ctx.action.OperationEffects {
		if effect.Compiled.Trigger != gameplayeffects.TriggerOnAction || effect.Action == nil {
			continue
		}
		var selected []statusfx.StatusOp
		for _, status := range effect.Action.StatusOps {
			if strings.EqualFold(status.Phase, "PRE_CHAIN") {
				selected = append(selected, status)
			}
		}
		if len(selected) == 0 {
			continue
		}
		allPreApplied := true
		for _, status := range selected {
			if ctx.action.PreAppliedEffectIDs == nil || !ctx.action.PreAppliedEffectIDs[status.EffectID] {
				allPreApplied = false
				break
			}
		}
		if allPreApplied {
			continue
		}
		targets := []*CombatantAuto{ctx.actor}
		if !strings.EqualFold(effect.Action.TargetType, string(gameplayeffects.TargetSelf)) {
			targets = resolveTargets(ctx.targets, effect.Action)
		}
		conditions := effect.Compiled.Conditions
		if len(conditions) == 0 {
			conditions = effect.Action.OperationConditions
		}
		rank := effect.SourceRank
		if rank <= 0 {
			rank = ctx.actor.Rank
		}
		if len(conditions) > 0 {
			targets = filterOperationTargets(ctx.actor, targets, conditions, rank, effect.Action.Path, r.turn)
		}
		if len(targets) == 0 {
			continue
		}
		pending := selected[:0]
		for _, status := range selected {
			if ctx.action.PreAppliedEffectIDs == nil || !ctx.action.PreAppliedEffectIDs[status.EffectID] {
				pending = append(pending, status)
			}
		}
		selected = pending
		selectedAction := *effect.Action
		selectedAction.StatusOps = selected
		r.applyAuthoredStatusOps(combatPhaseContext{actor: ctx.actor, action: &selectedAction, firstTarget: ctx.firstTarget, targets: targets}, selected)
		if ctx.action.PreAppliedEffectIDs == nil {
			ctx.action.PreAppliedEffectIDs = make(map[string]bool)
		}
		for _, status := range selected {
			ctx.action.PreAppliedEffectIDs[status.EffectID] = true
		}
	}
	ops := statusOpsForPhase(collectAllStatusOps(ctx.action), "PRE_CHAIN")
	if len(ops) == 0 {
		return
	}
	r.applyAuthoredStatusOps(ctx, ops)
}

func (r *combatActionRunner) applyPreChainDamageModifiers(ctx combatPhaseContext) {
	if ctx.action == nil {
		return
	}
	multiplier := 1.0
	for _, effect := range ctx.action.OperationEffects {
		if effect.Action == nil {
			continue
		}
		if !operationConditionsPass(ctx, effect, r.turn) {
			continue
		}
		for _, modifier := range effect.Action.PreChainDamageModifiers {
			if ctx.action.PreAppliedEffectIDs != nil && ctx.action.PreAppliedEffectIDs[modifier.EffectID] {
				continue
			}
			multiplier *= modifier.Multiplier
			if ctx.action.PreAppliedEffectIDs == nil {
				ctx.action.PreAppliedEffectIDs = make(map[string]bool)
			}
			ctx.action.PreAppliedEffectIDs[modifier.EffectID] = true
		}
	}
	for _, modifier := range ctx.action.PreChainDamageModifiers {
		if ctx.action.PreAppliedEffectIDs != nil && ctx.action.PreAppliedEffectIDs[modifier.EffectID] {
			continue
		}
		multiplier *= modifier.Multiplier
		if ctx.action.PreAppliedEffectIDs == nil {
			ctx.action.PreAppliedEffectIDs = make(map[string]bool)
		}
		ctx.action.PreAppliedEffectIDs[modifier.EffectID] = true
	}
	if multiplier == 1 {
		return
	}
	ctx.action.Damage *= multiplier
	if ctx.damageScale != nil {
		*ctx.damageScale *= multiplier
	}
}

func (r *combatActionRunner) applyPreChainNotifications(ctx combatPhaseContext) {
	r.applyCombatNotifications(ctx, "PRE_CHAIN")
}

func (r *combatActionRunner) applyCombatNotifications(ctx combatPhaseContext, phase string) {
	if ctx.action == nil || ctx.actor == nil {
		return
	}
	targetName := "kẻ địch"
	if ctx.firstTarget != nil && ctx.firstTarget.Name != "" {
		targetName = ctx.firstTarget.Name
	}
	for _, notification := range ctx.action.CombatNotifications {
		if notification.Phase != phase {
			continue
		}
		detail := strings.NewReplacer("{actor}", ctx.actor.Name, "{target}", targetName).Replace(notification.Message)
		*r.turnLogs = append(*r.turnLogs, CombatLog{
			Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
			Action: notification.Action, TargetId: targetID(ctx.firstTarget), Detail: detail,
		})
	}
	if phase != "PRE_CHAIN" {
		return
	}
	for _, effect := range ctx.action.OperationEffects {
		if effect.Compiled.Trigger != gameplayeffects.TriggerOnAction || effect.Action == nil {
			continue
		}
		if !operationConditionsPass(ctx, effect, r.turn) {
			continue
		}
		for _, notification := range effect.Action.CombatNotifications {
			if notification.Phase != phase || (ctx.action.PreAppliedEffectIDs != nil && ctx.action.PreAppliedEffectIDs[notification.EffectID]) {
				continue
			}
			detail := strings.NewReplacer("{actor}", ctx.actor.Name, "{target}", targetName).Replace(notification.Message)
			*r.turnLogs = append(*r.turnLogs, CombatLog{
				Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
				Action: notification.Action, TargetId: targetID(ctx.firstTarget), Detail: detail,
			})
			if ctx.action.PreAppliedEffectIDs == nil {
				ctx.action.PreAppliedEffectIDs = make(map[string]bool)
			}
			ctx.action.PreAppliedEffectIDs[notification.EffectID] = true
		}
	}
}

func operationConditionsPass(ctx combatPhaseContext, effect SimActionOperation, turn int) bool {
	conditions := effect.Compiled.Conditions
	if len(conditions) == 0 && effect.Action != nil {
		conditions = effect.Action.OperationConditions
	}
	if len(conditions) == 0 {
		return true
	}
	targets := ctx.targets
	if targets == nil && ctx.firstTarget != nil {
		targets = []*CombatantAuto{ctx.firstTarget}
	}
	rank := effect.SourceRank
	if rank <= 0 && ctx.actor != nil {
		rank = ctx.actor.Rank
	}
	path := []string(nil)
	if effect.Action != nil {
		path = effect.Action.Path
	}
	return len(filterOperationTargets(ctx.actor, targets, conditions, rank, path, turn)) > 0
}

func targetID(target *CombatantAuto) string {
	if target == nil {
		return ""
	}
	return target.ID
}

func (r *combatActionRunner) applyAuthoredStatusOps(ctx combatPhaseContext, ops []statusfx.StatusOp) {
	actionRank := ctx.action.Rank
	if actionRank <= 0 {
		actionRank = ctx.actor.Rank
	}
	for i := range ops {
		ops[i].SourcePath = daoPathForAction(ctx.action.Path)
	}
	targets := ctx.targets
	if targets == nil && ctx.firstTarget != nil {
		targets = []*CombatantAuto{ctx.firstTarget}
	}
	applyStatusOpsRaw(ctx.actor, targets, ops, actionRank, r.turn, r.rng, r.turnLogs)
}

func actionMechanicTrigger(mechanics []CombatMechanic, preferred, fallback, code string) (string, bool) {
	if hasMechanic(mechanics, preferred, code) {
		return preferred, true
	}
	if fallback != "" && hasMechanic(mechanics, fallback, code) {
		return fallback, true
	}
	return "", false
}

func (r *combatActionRunner) applyKnockback(ctx combatPhaseContext) {
	if ctx.firstTarget == nil {
		return
	}
	trigger, ok := actionMechanicTrigger(ctx.action.Mechanics, "ON_ACTION", "ON_HIT", "KNOCKBACK")
	if !ok {
		return
	}
	steps := mechanicInt(ctx.action.Mechanics, trigger, "KNOCKBACK", "steps", 1)
	if steps <= 0 {
		steps = 1
	}
	oldX := ctx.firstTarget.GridX
	if oldX < 1 {
		oldX = 1
	}
	newX := oldX + steps
	if newX > 5 {
		newX = 5
	}
	ctx.firstTarget.GridX = newX
	detail := fmt.Sprintf("[%s] Đánh lui %s từ hàng %d về hàng %d!", ctx.action.Name, ctx.firstTarget.Name, oldX, newX)
	if oldX == 5 {
		detail = fmt.Sprintf("[%s] Đẩy mạnh %s nhưng đối phương đã chạm vách sau (hàng 5)!", ctx.action.Name, ctx.firstTarget.Name)
	}
	*r.turnLogs = append(*r.turnLogs, CombatLog{
		Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
		Action: "KNOCKBACK", TargetId: ctx.firstTarget.ID, Detail: detail,
	})
}

func (r *combatActionRunner) applyPull(ctx combatPhaseContext) {
	if ctx.firstTarget == nil {
		return
	}
	trigger, ok := actionMechanicTrigger(ctx.action.Mechanics, "ON_ACTION", "ON_HIT", "PULL_TARGET")
	if !ok {
		return
	}
	steps := mechanicInt(ctx.action.Mechanics, trigger, "PULL_TARGET", "steps", 2)
	if steps <= 0 {
		steps = 2
	}
	oldX := ctx.firstTarget.GridX
	if oldX < 1 {
		oldX = 1
	}
	newX := oldX - steps
	if newX < 1 {
		newX = 1
	}
	ctx.firstTarget.GridX = newX
	detail := fmt.Sprintf("[%s] Kéo mạnh %s từ hàng %d lên hàng %d tiền tuyến!", ctx.action.Name, ctx.firstTarget.Name, oldX, newX)
	if oldX == 1 {
		detail = fmt.Sprintf("[%s] Khóa chặt %s ở hàng tiền tuyến (hàng 1)!", ctx.action.Name, ctx.firstTarget.Name)
	}
	*r.turnLogs = append(*r.turnLogs, CombatLog{
		Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
		Action: "PULL_TARGET", TargetId: ctx.firstTarget.ID, Detail: detail,
	})
}

func (r *combatActionRunner) applySwap(ctx combatPhaseContext) {
	if ctx.firstTarget == nil {
		return
	}
	if _, ok := actionMechanicTrigger(ctx.action.Mechanics, "ON_ACTION", "ON_HIT", "SWAP_TARGET"); !ok {
		return
	}
	actorOldX, actorOldY := ctx.actor.GridX, ctx.actor.GridY
	if actorOldX < 1 {
		actorOldX = 1
	}
	if actorOldY < 1 {
		actorOldY = 1
	}
	targetOldX, targetOldY := ctx.firstTarget.GridX, ctx.firstTarget.GridY
	if targetOldX < 1 {
		targetOldX = 1
	}
	if targetOldY < 1 {
		targetOldY = 1
	}
	ctx.actor.GridX, ctx.actor.GridY = targetOldX, targetOldY
	ctx.firstTarget.GridX, ctx.firstTarget.GridY = actorOldX, actorOldY
	*r.turnLogs = append(*r.turnLogs, CombatLog{
		Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
		Action: "SWAP_TARGET", TargetId: ctx.firstTarget.ID,
		Detail: fmt.Sprintf("[%s] Hoán vị không gian! %s tráo đổi vị trí với %s (ô [%d,%d] ↔ [%d,%d])!", ctx.action.Name, ctx.actor.Name, ctx.firstTarget.Name, actorOldX, actorOldY, targetOldX, targetOldY),
	})
}

func (r *combatActionRunner) applyRetreat(ctx combatPhaseContext) {
	trigger, ok := actionMechanicTrigger(ctx.action.Mechanics, "ON_CAST", "ON_ACTION", "RETREAT_SELF")
	if !ok {
		return
	}
	steps := mechanicInt(ctx.action.Mechanics, trigger, "RETREAT_SELF", "steps", 1)
	if steps <= 0 {
		steps = 1
	}
	oldX := ctx.actor.GridX
	if oldX < 1 {
		oldX = 1
	}
	newX := oldX + steps
	if newX > 5 {
		newX = 5
	}
	ctx.actor.GridX = newX
	detail := fmt.Sprintf("[%s] %s chủ động rút lui từ hàng %d về hàng %d!", ctx.action.Name, ctx.actor.Name, oldX, newX)
	if oldX == 5 {
		detail = fmt.Sprintf("[%s] %s đã ở hàng sau cùng (hàng 5)!", ctx.action.Name, ctx.actor.Name)
	}
	*r.turnLogs = append(*r.turnLogs, CombatLog{
		Turn: r.turn, ActorId: ctx.actor.ID, ActorName: ctx.actor.Name,
		Action: "RETREAT_SELF", Detail: detail,
	})
}
