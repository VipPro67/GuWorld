package simulator

import (
	"math"
	"testing"
)

func TestTargetDamageModifierReducesAndClamps(t *testing.T) {
	actor := &CombatantAuto{ID: "actor", Name: "Người Chơi"}
	targets := []*CombatantAuto{
		{ID: "one", Name: "Địch 1", HP: 100, DamageMultiplier: 1},
		{ID: "two", Name: "Địch 2", HP: 100, ActiveEffects: []Effect{{ID: "WEAK", Type: "debuff", Value: 50, Duration: 3}}},
	}
	logs := []CombatLog{}
	runner := &combatActionRunner{turn: 1, turnLogs: &logs}
	ctx := combatPhaseContext{
		actor: actor,
		action: &SimAction{TargetDamageModifiers: []TargetDamageModifier{{
			Value: 0.3, MinMultiplier: 0.2, LogAction: "QI_SEA", Message: "{actor} reduces enemy damage.",
		}}},
		targets: targets,
	}

	runner.applyTargetDamageModifiers(ctx)
	if math.Abs(targets[0].DamageMultiplier-0.7) > 1e-9 || targets[1].DamageMultiplier != 0.2 {
		t.Fatalf("first reduction got multipliers %v and %v", targets[0].DamageMultiplier, targets[1].DamageMultiplier)
	}
	if len(logs) != 1 || logs[0].Action != "QI_SEA" || logs[0].Detail != "Người Chơi reduces enemy damage." {
		t.Fatalf("expected one aggregate authored log, got %+v", logs)
	}
	ComputeEffectMultipliers(targets[0])
	if math.Abs(targets[0].DamageMultiplier-0.7) > 1e-9 {
		t.Fatalf("recomputing derived effects lost persistent output reduction: %v", targets[0].DamageMultiplier)
	}

	runner.applyTargetDamageModifiers(ctx)
	runner.applyTargetDamageModifiers(ctx)
	if targets[0].DamageMultiplier != 0.2 || targets[1].DamageMultiplier != 0.2 {
		t.Fatalf("repeated reductions did not respect floor: %v and %v", targets[0].DamageMultiplier, targets[1].DamageMultiplier)
	}
	if len(logs) != 3 {
		t.Fatalf("expected one log per successful application, got %d", len(logs))
	}
}
