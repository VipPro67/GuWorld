package simulator

import (
	"math/rand"
	"reflect"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestTypedDamageMitigationMatchesLegacyBloodStainedRobe(t *testing.T) {
	attacker := newTestCombatant("attacker", "Attacker", 1000, 100)
	legacyTarget := newTestCombatant("target", "Target", 1000, 100)
	legacyTarget.Mechanics = []CombatMechanic{{Trigger: "BEFORE_HIT", ActionCode: "BLOOD_STAINED_ROBE"}}
	typedTarget := newTestCombatant("target", "Target", 1000, 100)
	typedTarget.DamageMitigations = []DamageMitigation{{
		BlockID: "BLOOD_STAINED_ROBE", Ratio: 0.4, ConvertToShield: true,
		LogAction: "BLOOD_STAINED_ROBE",
		Message:   "[Huyết Đạo - Huyết Nhiễm Chinh Bào] Chiến bào đỏ thẫm hấp thụ 40% sát thương ({mitigated}), chuyển hóa thành Huyết Thuẫn!",
	}}
	var legacyLogs, typedLogs []CombatLog
	applyDamageToTarget(nil, attacker, legacyTarget, 100, &legacyLogs, 1, false, true, false, rand.New(rand.NewSource(1)))
	applyDamageToTarget(nil, attacker, typedTarget, 100, &typedLogs, 1, false, true, false, rand.New(rand.NewSource(1)))
	if legacyTarget.HP != typedTarget.HP || legacyTarget.VirtualShield != typedTarget.VirtualShield {
		t.Fatalf("typed mitigation state differs: legacy HP/shield=%v/%v typed=%v/%v", legacyTarget.HP, legacyTarget.VirtualShield, typedTarget.HP, typedTarget.VirtualShield)
	}
	if !reflect.DeepEqual(legacyLogs, typedLogs) {
		t.Fatalf("typed mitigation logs differ:\nlegacy: %+v\ntyped: %+v", legacyLogs, typedLogs)
	}

	legacyStatusTarget := newTestCombatant("status-target", "Status Target", 1000, 100)
	legacyStatusTarget.ActiveEffects = []Effect{{ID: statusfx.StatusBloodRobe, Type: "buff", Duration: 2}}
	typedStatusTarget := newTestCombatant("status-target", "Status Target", 1000, 100)
	typedStatusTarget.ActiveEffects = []Effect{{ID: statusfx.StatusBloodRobe, Type: "buff", Duration: 2}}
	typedStatusTarget.DamageMitigations = typedTarget.DamageMitigations
	var legacyStatusLogs, typedStatusLogs []CombatLog
	applyDamageToTarget(nil, attacker, legacyStatusTarget, 100, &legacyStatusLogs, 1, false, true, false, rand.New(rand.NewSource(1)))
	applyDamageToTarget(nil, attacker, typedStatusTarget, 100, &typedStatusLogs, 1, false, true, false, rand.New(rand.NewSource(1)))
	if legacyStatusTarget.HP != typedStatusTarget.HP || legacyStatusTarget.VirtualShield != typedStatusTarget.VirtualShield || !reflect.DeepEqual(legacyStatusLogs, typedStatusLogs) {
		t.Fatalf("typed mitigation duplicated or changed the matching status rule: legacy=%+v typed=%+v", legacyStatusLogs, typedStatusLogs)
	}
}

func TestAddBoundedCounter(t *testing.T) {
	tests := []struct {
		name    string
		initial int
		delta   int
		minimum int
		maximum int
		want    int
	}{
		{name: "initializes and increments", delta: 1, maximum: 5, want: 1},
		{name: "clamps maximum", initial: 5, delta: 1, maximum: 5, want: 5},
		{name: "clamps minimum", initial: 1, delta: -3, minimum: 0, maximum: 5, want: 0},
		{name: "unbounded maximum", initial: 5, delta: 2, want: 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			combatant := &CombatantAuto{CustomState: map[string]int{"counter": tt.initial}}
			got := addBoundedCounter(combatant, "counter", tt.delta, tt.minimum, tt.maximum)
			if got != tt.want || combatant.CustomState["counter"] != tt.want {
				t.Fatalf("addBoundedCounter() = %d, stored %d; want %d", got, combatant.CustomState["counter"], tt.want)
			}
		})
	}
}

func TestCheckThreshold(t *testing.T) {
	combatant := &CombatantAuto{CustomState: map[string]int{"sat_khi": 5}}

	// Not met
	if CheckThreshold(combatant, "sat_khi", 10, false) {
		t.Fatal("expected threshold 10 not met with value 5")
	}
	if combatant.CustomState["sat_khi"] != 5 {
		t.Fatalf("counter modified on failure: %d", combatant.CustomState["sat_khi"])
	}

	// Met without reset
	ApplyCounter(combatant, "sat_khi", 5, 0, 20)
	if !CheckThreshold(combatant, "sat_khi", 10, false) {
		t.Fatal("expected threshold 10 met with value 10")
	}
	if combatant.CustomState["sat_khi"] != 10 {
		t.Fatalf("counter modified when autoReset=false: %d", combatant.CustomState["sat_khi"])
	}

	// Met with autoReset
	if !CheckThreshold(combatant, "sat_khi", 10, true) {
		t.Fatal("expected threshold 10 met")
	}
	if combatant.CustomState["sat_khi"] != 0 {
		t.Fatalf("counter not reset to 0: %d", combatant.CustomState["sat_khi"])
	}
}

func TestTurnStartCounterAddsApplyOnceAndRespectMaximum(t *testing.T) {
	combatant := &CombatantAuto{TurnStartCounterAdds: []CounterAdd{{Key: "thi_y", Delta: 1, Min: 0, Max: 10}}}
	for turn := 0; turn < 12; turn++ {
		if !applyTurnStartCounterAdds(combatant) {
			t.Fatal("typed thi_y counter should activate the poem turn-start rider")
		}
	}
	if got := combatant.CustomState["thi_y"]; got != 10 {
		t.Fatalf("thi_y = %d, want capped at 10", got)
	}
}

func TestStackingStatusLifecycleAndTiers(t *testing.T) {
	combatant := &CombatantAuto{CustomState: make(map[string]int)}
	tiers := []StackingStatusTier{
		{Tier: 1, StatPercent: -10},
		{Tier: 2, StatPercent: -25},
		{Tier: 3, StatPercent: -35},
		{Tier: 4, StatPercent: -50},
		{Tier: 5, StatPercent: -99},
	}

	// Apply 1 stack
	stacks := ApplyStackingStatus(combatant, "sinh_khi_tan", 1, 5, 3, DispelDecrementOne, tiers)
	if stacks != 1 {
		t.Fatalf("expected 1 stack, got %d", stacks)
	}
	if pct := GetStackingStatusPercent(combatant, "sinh_khi_tan", tiers); pct != -10 {
		t.Fatalf("expected tier 1 pct -10, got %f", pct)
	}

	// Apply up to 5 stacks
	ApplyStackingStatus(combatant, "sinh_khi_tan", 4, 5, 3, DispelDecrementOne, tiers)
	if pct := GetStackingStatusPercent(combatant, "sinh_khi_tan", tiers); pct != -99 {
		t.Fatalf("expected tier 5 pct -99, got %f", pct)
	}

	// Cannot exceed maxStacks=5
	ApplyStackingStatus(combatant, "sinh_khi_tan", 2, 5, 3, DispelDecrementOne, tiers)
	if combatant.CustomState["stacking_status_sinh_khi_tan"] != 5 {
		t.Fatalf("expected clamped at 5, got %d", combatant.CustomState["stacking_status_sinh_khi_tan"])
	}

	// Dispel decrement one stack
	removed, remaining := DispelStackingStatus(combatant, "sinh_khi_tan", DispelDecrementOne)
	if removed != 1 || remaining != 4 {
		t.Fatalf("expected removed=1, remaining=4; got removed=%d, remaining=%d", removed, remaining)
	}
	if pct := GetStackingStatusPercent(combatant, "sinh_khi_tan", tiers); pct != -50 {
		t.Fatalf("expected tier 4 pct -50, got %f", pct)
	}

	// Dispel clear all
	removed, remaining = DispelStackingStatus(combatant, "sinh_khi_tan", DispelClearAll)
	if removed != 4 || remaining != 0 {
		t.Fatalf("expected removed=4, remaining=0; got removed=%d, remaining=%d", removed, remaining)
	}
	if pct := GetStackingStatusPercent(combatant, "sinh_khi_tan", tiers); pct != 0 {
		t.Fatalf("expected 0 pct after clear all, got %f", pct)
	}
}

func TestApplyHPCostKeepsMinimumRemainingHP(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 100, 5)
	actor.HP = 5
	action := &SimAction{HPCosts: []HPCost{{CurrentHPPercent: 0.2, MinCost: 10, MinRemaining: 1, LogAction: "BLOOD_DEMON_DISINTEGRATION", Message: "{cost}/{actual_cost}/{remaining_hp}"}}}
	logs := []CombatLog{}
	runner := &combatActionRunner{turn: 1, turnLogs: &logs}
	runner.applyHPCosts(combatPhaseContext{actor: actor, action: action}, "LATE_RIDER")
	if actor.HP != 1 {
		t.Fatalf("actor HP after bounded cost = %.0f, want 1", actor.HP)
	}
	if len(logs) != 1 || logs[0].Damage != 10 || logs[0].Detail != "10/4/1" {
		t.Fatalf("bounded HP cost log = %+v, want nominal 10 / actual 4 / remaining 1", logs)
	}
}

func TestHPCostRunsOnlyAtAuthoredPhase(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 100, 100)
	action := &SimAction{HPCosts: []HPCost{{CurrentHPPercent: 0.05, MinCost: 1, MinRemaining: 1, Phase: "EARLY_RIDER", LogAction: "GU_DETONATION", Message: "{actor}"}}}
	var logs []CombatLog
	runner := &combatActionRunner{turn: 1, turnLogs: &logs}
	ctx := combatPhaseContext{actor: actor, action: action}
	runner.applyHPCosts(ctx, "LATE_RIDER")
	if actor.HP != 100 || len(logs) != 0 {
		t.Fatalf("early HP cost ran in late phase: hp=%v logs=%+v", actor.HP, logs)
	}
	runner.applyHPCosts(ctx, "EARLY_RIDER")
	if actor.HP != 95 || len(logs) != 1 || logs[0].Action != "GU_DETONATION" {
		t.Fatalf("early HP cost result hp=%v logs=%+v, want 95 HP and one authored log", actor.HP, logs)
	}
}
