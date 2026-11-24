package simulator

import (
	"testing"

	effects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestLateConditionalPassiveModifierAndStatusCleanse(t *testing.T) {
	condition := []effects.Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.35}}
	actor := &CombatantAuto{
		ID: "actor", Name: "actor", HP: 35, MaxHP: 100,
		IsStunned: true, IsFrozen: true,
		ActiveEffects: []Effect{{ID: statusfx.StatusStun, Type: "debuff"}, {ID: statusfx.StatusFreeze, Type: "debuff"}, {ID: statusfx.StatusBurn, Type: "debuff"}},
		PassiveOperationEffects: []SimActionOperation{
			{Compiled: effects.CompiledOperation{Conditions: condition, Phase: "LATE_RIDER"}, Action: &SimAction{DamageModifiers: []DamageModifier{{SubType: "MISSING_HP", BaseMultiplier: 1.5, LogAction: "PASSION_OVERLOAD", Message: "{actor}"}}}},
			{Compiled: effects.CompiledOperation{Conditions: condition, Phase: "LATE_RIDER"}, Action: &SimAction{CleanseStatusIDs: []string{statusfx.StatusStun, statusfx.StatusFreeze}}},
		},
	}
	action := &SimAction{Damage: 100}
	damageScale := 1.0
	var logs []CombatLog
	runner := &combatActionRunner{turn: 1, turnLogs: &logs}
	runner.applyPassiveOperationsPhase(combatPhaseContext{actor: actor, action: action, damageScale: &damageScale}, "LATE_RIDER")
	if action.Damage != 150 || damageScale != 1.5 {
		t.Fatalf("conditional modifier damage=%v scale=%v; want 150 and 1.5", action.Damage, damageScale)
	}
	if actor.IsStunned || actor.IsFrozen || len(actor.ActiveEffects) != 1 || actor.ActiveEffects[0].ID != statusfx.StatusBurn {
		t.Fatalf("selective cleanse did not preserve unrelated debuff: flags stun=%v freeze=%v effects=%+v", actor.IsStunned, actor.IsFrozen, actor.ActiveEffects)
	}
	if len(logs) != 1 || logs[0].Action != "PASSION_OVERLOAD" {
		t.Fatalf("expected one passive log, got %+v", logs)
	}
}

func TestLateConditionalPassiveDoesNothingAboveThreshold(t *testing.T) {
	actor := &CombatantAuto{
		Name: "actor", HP: 36, MaxHP: 100, IsStunned: true,
		PassiveOperationEffects: []SimActionOperation{{
			Compiled: effects.CompiledOperation{Conditions: []effects.Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.35}}, Phase: "LATE_RIDER"},
			Action:   &SimAction{DamageModifiers: []DamageModifier{{BaseMultiplier: 1.5}}, CleanseStatusIDs: []string{statusfx.StatusStun}},
		}},
	}
	action := &SimAction{Damage: 100}
	damageScale := 1.0
	var logs []CombatLog
	runner := &combatActionRunner{turnLogs: &logs}
	runner.applyPassiveOperationsPhase(combatPhaseContext{actor: actor, action: action, damageScale: &damageScale}, "LATE_RIDER")
	if action.Damage != 100 || damageScale != 1 || !actor.IsStunned {
		t.Fatalf("passive unexpectedly applied above threshold: damage=%v scale=%v stunned=%v", action.Damage, damageScale, actor.IsStunned)
	}
	if len(logs) != 0 {
		t.Fatalf("expected no passive log above threshold, got %+v", logs)
	}
}

func TestPassiveOperationRunsOnlyAtAuthoredRiderPhase(t *testing.T) {
	actor := &CombatantAuto{
		Name: "actor", HP: 40, MaxHP: 100,
		PassiveOperationEffects: []SimActionOperation{{
			Compiled: effects.CompiledOperation{Conditions: []effects.Condition{{Field: "HP_PERCENT", Operator: "LTE", Value: 0.4}}, Phase: "EARLY_RIDER"},
			Action:   &SimAction{DamageModifiers: []DamageModifier{{BaseMultiplier: 1.5}}},
		}},
	}
	action := &SimAction{Damage: 100}
	damageScale := 1.0
	runner := &combatActionRunner{}
	ctx := combatPhaseContext{actor: actor, action: action, damageScale: &damageScale}
	runner.applyPassiveOperationsPhase(ctx, "LATE_RIDER")
	if action.Damage != 100 {
		t.Fatalf("EARLY_RIDER modifier ran during LATE_RIDER: %v", action.Damage)
	}
	runner.applyPassiveOperationsPhase(ctx, "EARLY_RIDER")
	if action.Damage != 150 {
		t.Fatalf("EARLY_RIDER modifier did not run: %v", action.Damage)
	}
}

func TestFinalRiderCounterPayoutAppliesDamageAndHealing(t *testing.T) {
	actor := &CombatantAuto{
		Name: "actor", HP: 200, MaxHP: 1000,
		CustomState: map[string]int{"eating_loss_stacks": 4},
		PassiveOperationEffects: []SimActionOperation{
			{Compiled: effects.CompiledOperation{Phase: "FINAL_RIDER"}, Action: &SimAction{DamageModifiers: []DamageModifier{{SubType: "COUNTER_COUNT", BaseMultiplier: 1.3, CounterKey: "eating_loss_stacks", PerCounter: 0.15}}}},
			{Compiled: effects.CompiledOperation{Phase: "FINAL_RIDER"}, Action: &SimAction{CounterHeals: []CounterHeal{{Key: "eating_loss_stacks", PerCounter: 0.05}}}},
		},
	}
	action := &SimAction{Damage: 100}
	damageScale := 1.0
	runner := &combatActionRunner{}
	runner.applyPassiveOperationsPhase(combatPhaseContext{actor: actor, action: action, damageScale: &damageScale}, "FINAL_RIDER")
	if action.Damage != 190 || damageScale != 1.9 {
		t.Fatalf("counter damage payout = damage %v, scale %v; want 190 and 1.9", action.Damage, damageScale)
	}
	if actor.HP != 400 {
		t.Fatalf("counter healing payout HP = %v, want 400", actor.HP)
	}
}

func TestFinalRiderCounterNotificationRendersPrimitivePayout(t *testing.T) {
	actor := &CombatantAuto{
		Name: "actor", HP: 200, MaxHP: 1000,
		CustomState: map[string]int{"eating_loss_stacks": 4},
		PassiveOperationEffects: []SimActionOperation{
			{Compiled: effects.CompiledOperation{Phase: "FINAL_RIDER"}, Action: &SimAction{DamageModifiers: []DamageModifier{{SubType: "COUNTER_COUNT", BaseMultiplier: 1.3, CounterKey: "eating_loss_stacks", PerCounter: 0.15}}}},
			{Compiled: effects.CompiledOperation{Phase: "FINAL_RIDER"}, Action: &SimAction{CounterHeals: []CounterHeal{{Key: "eating_loss_stacks", PerCounter: 0.05}}}},
			{Compiled: effects.CompiledOperation{Phase: "FINAL_RIDER"}, Action: &SimAction{CombatNotifications: []CombatNotification{{Action: "EATING_LOSS_BLESSING", CounterKey: "eating_loss_stacks", Message: "{actor}: {counter_count} stacks, +{damage_percent}%, heal {heal_amount}."}}}},
		},
	}
	var logs []CombatLog
	runner := &combatActionRunner{turn: 3, turnLogs: &logs}
	runner.applyPassiveOperationsPhase(combatPhaseContext{actor: actor, action: &SimAction{Damage: 100}}, "FINAL_RIDER")
	if len(logs) != 1 || logs[0].Detail != "actor: 4 stacks, +90%, heal 200." {
		t.Fatalf("counter payout notification = %+v", logs)
	}
}
