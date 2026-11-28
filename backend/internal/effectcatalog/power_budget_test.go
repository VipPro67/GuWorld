package effectcatalog

import (
	"testing"
)

func TestPowerCostCatalogCoversOperationCatalog(t *testing.T) {
	for opName := range OperationCatalog {
		if _, ok := PowerCostCatalog[opName]; !ok {
			t.Fatalf("OperationCatalog op %q missing from PowerCostCatalog", opName)
		}
	}
}

func TestPowerBudgetCurveAndOpCaps(t *testing.T) {
	if got := PowerBudget(1); got != 100 {
		t.Fatalf("rank 1 budget = %v, want 100", got)
	}
	if got := PowerBudget(5); got != 10000 {
		t.Fatalf("rank 5 budget = %v, want 10000", got)
	}
	if got := PowerBudget(6); got != 1000000 {
		t.Fatalf("rank 6 budget = %v, want 1000000", got)
	}
	if got := MaxOpsForRank(2); got != 2 {
		t.Fatalf("rank 2 op cap = %d, want 2", got)
	}
	if got := MaxOpsForRank(5); got != 4 {
		t.Fatalf("rank 5 op cap = %d, want 4", got)
	}
	if got := MaxOpsForRank(8); got != 8 {
		t.Fatalf("rank 8 op cap = %d, want 8", got)
	}
}

func TestEvaluatePowerBudgetFlatDamageWithinBudget(t *testing.T) {
	blocks := []PowerBlockInput{
		{
			ID:      "EFFECT_01",
			Trigger: "ON_ACTION",
			Target:  PowerTargetInput{Selector: "TARGET"},
			Operations: []PowerOpInput{
				{Op: "DAMAGE", SubType: "FLAT", Value: 3000},
			},
		},
	}
	rep := EvaluatePowerBudget(5, blocks)
	if rep.PowerBudget != 10000 || rep.PowerCost != 3000 {
		t.Fatalf("got budget=%v cost=%v, want 10000 / 3000", rep.PowerBudget, rep.PowerCost)
	}
	if len(rep.Violations) != 0 {
		t.Fatalf("unexpected violations: %+v", rep.Violations)
	}
}

func TestEvaluatePowerBudgetReviveMinRankGateAndRankScale(t *testing.T) {
	blocks := []PowerBlockInput{
		{
			ID:      "EFFECT_01",
			Trigger: "ON_DEATH",
			Target:  PowerTargetInput{Selector: "SELF"},
			Operations: []PowerOpInput{
				{Op: "REVIVE"},
			},
		},
	}
	r5 := EvaluatePowerBudget(5, blocks)
	if r5.MinRankRequired != 6 || len(r5.Violations) == 0 {
		t.Fatalf("expected rank 5 REVIVE violation, got %+v", r5)
	}

	r6 := EvaluatePowerBudget(6, blocks)
	r7 := EvaluatePowerBudget(7, blocks)
	if len(r6.Violations) != 0 {
		t.Fatalf("unexpected rank 6 REVIVE violation: %+v", r6.Violations)
	}
	if r7.PowerCost != r6.PowerCost*100 {
		t.Fatalf("expected rank 7 REVIVE cost (%v) to be 100x rank 6 (%v)", r7.PowerCost, r6.PowerCost)
	}
}

func TestEvaluatePowerBudgetModifiers(t *testing.T) {
	chance := 0.10
	blocks := []PowerBlockInput{
		{
			ID:         "EFFECT_01",
			Trigger:    "ALWAYS_ON",
			Target:     PowerTargetInput{Selector: "ALL_ENEMIES", BypassFrontline: true},
			Conditions: []any{map[string]any{"field": "HP_PERCENT", "operator": "<", "value": 50}},
			Operations: []PowerOpInput{
				{Op: "STATUS", Status: "STUN", Turns: 3, Chance: &chance},
			},
		},
	}
	rep := EvaluatePowerBudget(6, blocks)
	// Base 35 * RankScale(10000) * Target(2.2+0.25=2.45) * Trigger(1.5) * Duration(1.0 for ALWAYS_ON) * ChanceFloor(0.25) * Cond(0.85) = 273328.125 -> 273328.13
	want := 273328.13
	if rep.PowerCost != want {
		t.Fatalf("got cost=%v, want %v", rep.PowerCost, want)
	}
}
