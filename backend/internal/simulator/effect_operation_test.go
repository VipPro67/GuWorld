package simulator

import (
	"math/rand"
	"strings"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestEffectOperationCleansesAllAllies(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 1000, 0)
	ally := newTestCombatant("ally", "Player", 1000, 0)
	for _, member := range []*CombatantAuto{actor, ally} {
		member.ActiveEffects = []Effect{
			{ID: "WEAK", Type: "debuff", Value: 20, Duration: 3},
			{ID: "HASTE", Type: "buff", Value: 20, Duration: 3},
		}
		member.ActiveDoT = []DoTEffect{{Type: "bleed", Value: 10, Turns: 2}}
	}
	state := &CombatState{Allies: []*CombatantAuto{actor, ally}}
	logs := []CombatLog{}
	summonCount := 0
	runner := combatActionRunner{
		state: state, rng: rand.New(rand.NewSource(1)), turn: 1,
		turnLogs: &logs, summonCount: &summonCount,
	}
	action := &SimAction{
		ActionType: "GU_EFFECT_OPERATION", Name: "Tịnh hóa phe ta",
		TargetType: "ALL_ALLIES", IsEffectOperation: true, CleanseDebuffs: true,
	}
	runner.runOne(actor, action, true, nil)
	for _, member := range []*CombatantAuto{actor, ally} {
		if len(member.ActiveDoT) != 0 {
			t.Errorf("%s retained DoT effects: %+v", member.ID, member.ActiveDoT)
		}
		if len(member.ActiveEffects) != 1 || member.ActiveEffects[0].ID != "HASTE" {
			t.Errorf("%s effects after cleanse = %+v, want only HASTE", member.ID, member.ActiveEffects)
		}
	}
}

func TestOnCastCurrentHPToShieldConversion(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 2000, 0)
	actor.HP = 1000
	logs := []CombatLog{}
	runner := &combatActionRunner{rng: rand.New(rand.NewSource(1)), turn: 1, turnLogs: &logs}
	status := statusfx.StatusOp{ID: statusfx.StatusAtkUp, Value: 50, Turns: 3, Chance: 1, ApplicationMode: "DIRECT", StackPolicy: "STACK", SuppressLog: true}
	action := &SimAction{OperationEffects: []SimActionOperation{{
		Compiled:   gameplayeffects.CompiledOperation{Trigger: gameplayeffects.TriggerOnCast},
		SourceRank: 7,
		Action: &SimAction{
			ResourceConversions: []ResourceConversion{{Fraction: 1, MinRemaining: 1, LogAction: "TRANSFORM", Message: "{actor} converts {amount} HP."}},
			StatusOps:           []statusfx.StatusOp{status},
		},
	}}}
	runner.dispatchPhase(combatPhaseOnCast, combatPhaseContext{actor: actor, action: action})
	if actor.HP != 1 || actor.VirtualShield != 999 {
		t.Fatalf("conversion yielded HP=%v shield=%v, want HP=1 shield=999", actor.HP, actor.VirtualShield)
	}
	if len(actor.ActiveEffects) != 1 || actor.ActiveEffects[0].ID != statusfx.StatusAtkUp || actor.ActiveEffects[0].Value != 50 || actor.ActiveEffects[0].Duration != 3 {
		t.Fatalf("converted status = %+v, want ATK_UP 50 for 3 turns", actor.ActiveEffects)
	}
	if len(logs) != 1 || logs[0].Action != "TRANSFORM" || logs[0].Detail != "actor converts 999 HP." {
		t.Fatalf("conversion log = %+v", logs)
	}
}

func TestOnCastConversionDoesNotSuppressParentHit(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 2000, 100)
	actor.HP = 1000
	target := newTestCombatant("target", "Monster", 1000, 0)
	target.GridX, target.GridY = 1, 1
	state := &CombatState{Allies: []*CombatantAuto{actor}, Enemies: []*CombatantAuto{target}, VirtualMode: true}
	logs := []CombatLog{}
	summonCount := 0
	runner := &combatActionRunner{state: state, rng: rand.New(rand.NewSource(1)), turn: 1, turnLogs: &logs, summonCount: &summonCount}
	action := &SimAction{
		ActionType: "GU_SKILL", Name: "Kiếm Long Biến", Damage: 100,
		OperationEffects: []SimActionOperation{{
			Compiled:   gameplayeffects.CompiledOperation{Trigger: gameplayeffects.TriggerOnCast},
			SourceRank: 7,
			Action:     &SimAction{ResourceConversions: []ResourceConversion{{Fraction: 1, MinRemaining: 1, LogAction: "TRANSFORM", Message: "{actor} hóa thân."}}},
		}},
	}
	if got := resolveTargets(state.Enemies, action); len(got) != 1 {
		t.Fatalf("pre-cast target resolution = %v", got)
	}
	out := runner.runOne(actor, action, true, state.Enemies)
	if !out.executed || target.HP >= 1000 {
		t.Fatalf("cast conversion suppressed parent hit: outcome=%+v targetHP=%v logs=%+v", out, target.HP, logs)
	}
}

func TestEffectOperationRestoresManaPerAllyMaxMana(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 1000, 0)
	actor.MaxMana, actor.Mana = 100, 0
	ally := newTestCombatant("ally", "Player", 1000, 0)
	ally.MaxMana, ally.Mana = 200, 0
	dead := newTestCombatant("dead", "Player", 1000, 0)
	dead.MaxMana, dead.Mana, dead.HP = 300, 0, 0
	state := &CombatState{Allies: []*CombatantAuto{actor, ally, dead}}
	logs := []CombatLog{}
	summonCount := 0
	runner := combatActionRunner{state: state, rng: rand.New(rand.NewSource(1)), turn: 1, turnLogs: &logs, summonCount: &summonCount}
	action := &SimAction{
		ActionType: "GU_EFFECT_OPERATION", Name: "Vận Mệnh Ca", TargetType: "ALL_ALLIES", IsEffectOperation: true,
		ManaRestoreTargetPercent: 0.20, ManaRestoreSuppressLog: true,
	}
	runner.runOne(actor, action, true, nil)
	if actor.Mana != 20 || ally.Mana != 40 || dead.Mana != 0 {
		t.Fatalf("party mana after restore = actor %.0f, ally %.0f, dead %.0f; want 20, 40, 0", actor.Mana, ally.Mana, dead.Mana)
	}
	for _, log := range logs {
		if log.Action == "MANA_RESTORE" {
			t.Fatalf("suppressed party restore emitted a per-target log: %+v", log)
		}
	}
}

func TestSelfTargetStatusOperationHonorsPartyTargetSelector(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 1000, 0)
	ally := newTestCombatant("ally", "Player", 1000, 0)
	state := &CombatState{Allies: []*CombatantAuto{actor, ally}}
	logs := []CombatLog{}
	summonCount := 0
	runner := combatActionRunner{state: state, rng: rand.New(rand.NewSource(1)), turn: 1, turnLogs: &logs, summonCount: &summonCount}
	action := &SimAction{
		ActionType: "GU_EFFECT_OPERATION", Name: "Vận Mệnh Ca", TargetType: "ALL_ALLIES", IsEffectOperation: true,
		StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusSongOfDestiny, Value: 40, Turns: 1, ApplicationMode: "DIRECT", StackPolicy: "REFRESH", ApplyToTargets: true}},
	}
	runner.runOne(actor, action, true, nil)
	for _, member := range []*CombatantAuto{actor, ally} {
		if len(member.ActiveEffects) != 1 || member.ActiveEffects[0].ID != statusfx.StatusSongOfDestiny || member.ActiveEffects[0].Value != 40 {
			t.Fatalf("%s status effects = %+v, want one 40%% SONG_OF_DESTINY", member.ID, member.ActiveEffects)
		}
	}
}

func TestPartyStatusReplaceRefreshesMatchingStatusAcrossSources(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 1000, 0)
	ally := newTestCombatant("ally", "Player", 1000, 0)
	for _, member := range []*CombatantAuto{actor, ally} {
		member.ActiveEffects = []Effect{{
			ID: statusfx.StatusSongOfDestiny, Type: "buff", Value: 25, Duration: 3,
			SourceID: "old-gu", EffectID: "old-gu:block", Context: "combat",
		}}
	}
	logs := []CombatLog{}
	applyStatusOps(actor, []*CombatantAuto{actor, ally}, []statusfx.StatusOp{{
		ID: statusfx.StatusSongOfDestiny, Value: 40, Turns: 1, ApplicationMode: "DIRECT",
		ApplyToTargets: true, SourceID: "new-gu", EffectID: "new-gu:block", Context: "combat", StackPolicy: "REPLACE",
	}}, 1, 1, rand.New(rand.NewSource(1)), &logs)
	for _, member := range []*CombatantAuto{actor, ally} {
		if len(member.ActiveEffects) != 1 || member.ActiveEffects[0].SourceID != "new-gu" || member.ActiveEffects[0].Value != 40 {
			t.Fatalf("%s did not replace the same status from another source: %+v", member.ID, member.ActiveEffects)
		}
	}
}

func TestCombatNotificationReplacesGenericOperationLog(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 1000, 0)
	logs := []CombatLog{}
	summonCount := 0
	runner := combatActionRunner{
		state: &CombatState{Allies: []*CombatantAuto{actor}}, rng: rand.New(rand.NewSource(1)), turn: 1,
		turnLogs: &logs, summonCount: &summonCount,
	}
	action := &SimAction{
		ActionType: "GU_EFFECT_OPERATION", Name: "Vận Mệnh Ca", TargetType: "SELF", IsEffectOperation: true,
		CombatNotifications: []CombatNotification{{Action: "SONG_OF_DESTINY", Message: "configured battle message"}},
	}
	runner.runOne(actor, action, true, nil)
	if len(logs) != 1 || logs[0].Action != "SONG_OF_DESTINY" || logs[0].Detail != "configured battle message" {
		t.Fatalf("combat notification logs = %+v, want only authored SONG_OF_DESTINY log", logs)
	}
}

func TestBlockV2GuSkillRunsOperationCarrierOnce(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	enemy := newTestCombatant("enemy", "Enemy", 100, 0)
	ally.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Cổ thử", SourceID: "GU_TEST", Rank: 1,
		Damage: 100, DamageMult: 1,
		OperationEffects: []SimActionOperation{{
			Compiled:   gameplayeffects.CompiledOperation{BlockID: "ACTION", BlockIndex: 0, Trigger: gameplayeffects.TriggerOnAction, Type: gameplayeffects.OpDamage},
			SourceGuID: "GU_TEST", SourceRank: 1,
			Action: &SimAction{ActionType: "GU_EFFECT_OPERATION", Name: "Cổ thử", Damage: 100, DamageMult: 1, TargetType: "SINGLE", IsEffectOperation: true},
		}},
	}}

	logs := Simulate(&CombatState{Seed: 27, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
	if enemy.HP != 0 {
		t.Fatalf("expected one 100-damage operation to defeat the target, remaining HP %.0f", enemy.HP)
	}
	applications := 0
	for _, log := range logs {
		if log.SourceGuID == "GU_TEST" && log.TargetId == "enemy" && log.Action == "GU_EFFECT_OPERATION" && log.Damage > 0 {
			applications++
			if log.Damage != 100 {
				t.Fatalf("operation damage = %.0f, want 100", log.Damage)
			}
		}
	}
	if applications != 1 {
		t.Fatalf("operation applied %d times, want exactly once", applications)
	}
}

func TestBlockV2CounterAddRunsOnceAndHonorsBounds(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	ally.CustomState = map[string]int{"test_counter": 0}
	logs := []CombatLog{}
	summonCount := 0
	runner := combatActionRunner{state: &CombatState{Seed: 31, Allies: []*CombatantAuto{ally}}, rng: rand.New(rand.NewSource(31)), turn: 1, turnLogs: &logs, summonCount: &summonCount}
	action := &SimAction{ActionType: "GU_EFFECT_OPERATION", Name: "Tích tầng", TargetType: "SELF", IsEffectOperation: true, CounterAdds: []CounterAdd{{Key: "test_counter", Delta: 1, Min: 0, Max: 5}}}
	runner.runOne(ally, action, true, nil)
	if got := ally.CustomState["test_counter"]; got != 1 {
		t.Fatalf("counter after one operation = %d, want 1", got)
	}
}

func TestOnCastDispelOperationRunsAtCastPhase(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 1000, 100)
	target := newTestCombatant("target", "Enemy", 1000, 0)
	target.ActiveEffects = []Effect{
		{ID: "HASTE", Type: "buff", Duration: 3},
		{ID: "DEF_UP", Type: "buff", Duration: 3},
		{ID: "STUN", Type: "debuff", Duration: 1},
	}
	logs := []CombatLog{}
	summonCount := 0
	runner := combatActionRunner{state: &CombatState{Allies: []*CombatantAuto{actor}, Enemies: []*CombatantAuto{target}}, rng: rand.New(rand.NewSource(4)), turn: 1, turnLogs: &logs, summonCount: &summonCount}
	action := &SimAction{
		ActionType: "GU_SKILL", Name: "Giải Mê", TargetType: "SINGLE", Damage: 1,
		OperationEffects: []SimActionOperation{{
			Compiled: gameplayeffects.CompiledOperation{BlockID: "UNRAVEL_RIDDLE", Trigger: gameplayeffects.TriggerOnCast, Type: gameplayeffects.OpDispel},
			Action:   &SimAction{ActionType: "GU_EFFECT_OPERATION", TargetType: "TARGET", IsEffectOperation: true, DispelBuffs: 2, DispelLogAction: "DISPEL", DispelMessage: "{actor} removes {dispelled} buffs from {target}."},
		}},
	}
	runner.runOne(actor, action, true, []*CombatantAuto{target})
	if len(target.ActiveEffects) != 1 || target.ActiveEffects[0].ID != "STUN" {
		t.Fatalf("ON_CAST DISPEL left effects %+v, want only STUN", target.ActiveEffects)
	}
	if len(logs) == 0 || logs[0].Action != "GU_SKILL" || !strings.Contains(logs[0].Detail, "removes 2 buffs") {
		t.Fatalf("ON_CAST DISPEL log missing: %+v", logs)
	}
}

func TestOnCastStatusAppliesBeforeParentActionDamage(t *testing.T) {
	actor := newTestCombatant("actor", "Player", 1000, 100)
	target := newTestCombatant("target", "Enemy", 1000, 0)
	logs := []CombatLog{}
	summonCount := 0
	runner := combatActionRunner{state: &CombatState{Allies: []*CombatantAuto{actor}, Enemies: []*CombatantAuto{target}}, rng: rand.New(rand.NewSource(2)), turn: 1, turnLogs: &logs, summonCount: &summonCount}
	status := statusfx.StatusOp{ID: statusfx.StatusAtkUp, Value: 40, Turns: 3, Chance: 1, ApplicationMode: "DIRECT", LogAction: "BUFF", Message: "{actor} borrows past power."}
	action := &SimAction{ActionType: "GU_SKILL", Name: "Mượn Lực", TargetType: "SINGLE", OperationEffects: []SimActionOperation{{
		Compiled:   gameplayeffects.CompiledOperation{BlockID: "BORROW_PAST_POWER", Trigger: gameplayeffects.TriggerOnCast, Type: gameplayeffects.OpStatus},
		SourceRank: 1, Action: &SimAction{ActionType: "GU_EFFECT_OPERATION", IsEffectOperation: true, StatusOps: []statusfx.StatusOp{status}},
	}}}
	runner.dispatchPhase(combatPhaseOnCast, combatPhaseContext{actor: actor, action: action, firstTarget: target, targets: []*CombatantAuto{target}})
	if actor.DamageMultiplier != 1.4 {
		t.Fatalf("cast status damage multiplier = %v, want 1.4 before parent hit", actor.DamageMultiplier)
	}
	if len(actor.ActiveEffects) != 1 || actor.ActiveEffects[0].ID != statusfx.StatusAtkUp {
		t.Fatalf("cast status effects = %+v", actor.ActiveEffects)
	}
	if len(logs) != 1 || logs[0].Action != "BUFF" || logs[0].Detail != "actor borrows past power." {
		t.Fatalf("cast status log = %+v", logs)
	}
}

func TestPreRiderHPCostMatchesBloodSacrificeNativeCost(t *testing.T) {
	makeRunner := func(actor *CombatantAuto) *combatActionRunner {
		logs := []CombatLog{}
		summonCount := 0
		return &combatActionRunner{state: &CombatState{Seed: 41}, rng: rand.New(rand.NewSource(41)), turn: 1, turnLogs: &logs, summonCount: &summonCount}
	}
	legacyActor := newTestCombatant("legacy", "Player", 1000, 0)
	typedActor := newTestCombatant("typed", "Player", 1000, 0)
	legacyActor.HP, typedActor.HP = 101, 101
	enemy := newTestCombatant("enemy", "Enemy", 1000, 0)
	legacy := &SimAction{ActionType: "GU_SKILL", Name: "Huyết Tế", TargetType: "SINGLE", Mechanics: []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "BLOOD_SACRIFICE"}}}
	typed := &SimAction{ActionType: "GU_SKILL", Name: "Huyết Tế", TargetType: "SINGLE", HPCosts: []HPCost{{CurrentHPPercent: 0.15, MinCost: 1, MinRemaining: 1, Phase: "PRE_RIDER", LogAction: "BLOOD_SACRIFICE", Message: "{actor} sacrifices {actual_cost} HP"}}}
	makeRunner(legacyActor).runOne(legacyActor, legacy, true, []*CombatantAuto{enemy})
	makeRunner(typedActor).runOne(typedActor, typed, true, []*CombatantAuto{enemy})
	if legacyActor.HP != typedActor.HP || legacyActor.HP != 85.85 {
		t.Fatalf("legacy and typed HP cost differ: legacy %.2f, typed %.2f; want 85.85", legacyActor.HP, typedActor.HP)
	}
}

func TestBlockV2GuSkillOperationConditionsAreApplied(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	enemy := newTestCombatant("enemy", "Enemy", 100, 0)
	enemy.Mechanics = []CombatMechanic{{Trigger: "BEFORE_ENEMY_ATTACK", ActionCode: "FORGET_INTENT_INTERRUPT", Values: map[string]any{"chance": 1.0}}}
	ally.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Cổ điều kiện", SourceID: "GU_CONDITIONAL", Rank: 1,
		Damage: 20, DamageMult: 1,
		OperationEffects: []SimActionOperation{{
			Compiled:   gameplayeffects.CompiledOperation{BlockID: "LOW_HP", BlockIndex: 0, Trigger: gameplayeffects.TriggerOnAction, Type: gameplayeffects.OpDamage, Conditions: []gameplayeffects.Condition{{Field: "TARGET_HP_PERCENT", Operator: "LTE", Value: 0.5}}},
			SourceGuID: "GU_CONDITIONAL", SourceRank: 1,
			Action: &SimAction{ActionType: "GU_EFFECT_OPERATION", Name: "Cổ điều kiện", Damage: 20, DamageMult: 1, TargetType: "SINGLE", IsEffectOperation: true},
		}},
	}}

	logs := Simulate(&CombatState{Seed: 28, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
	if enemy.HP != 100 {
		t.Fatalf("false target HP condition applied damage; remaining HP %.0f", enemy.HP)
	}
	for _, log := range logs {
		if log.Action == "FORGET_INTENT" || log.Action == "FORGET_INTENT_MISS" {
			t.Fatalf("ineligible conditional damage triggered hostile hit intent: %+v", log)
		}
	}
}

func TestBlockV2GuSkillCarriesOnHitMechanicsToPhantomOperations(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	enemy := newTestCombatant("enemy", "Enemy", 100, 0)
	enemy.VirtualShield = 50
	ally.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Cổ phá khiên", SourceID: "GU_PIERCE", Rank: 1,
		Damage: 100, DamageMult: 1,
		Mechanics: []CombatMechanic{{Trigger: "ON_HIT", ActionCode: "BYPASS_SHIELD"}},
		OperationEffects: []SimActionOperation{{
			Compiled:   gameplayeffects.CompiledOperation{BlockID: "ACTION", BlockIndex: 0, Trigger: gameplayeffects.TriggerOnAction, Type: gameplayeffects.OpDamage},
			SourceGuID: "GU_PIERCE", SourceRank: 1,
			Action: &SimAction{ActionType: "GU_EFFECT_OPERATION", Name: "Cổ phá khiên", PhantomDmg: 100, TargetType: "SINGLE", IsEffectOperation: true},
		}},
	}}

	Simulate(&CombatState{Seed: 29, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
	if enemy.HP != 0 || enemy.VirtualShield != 50 {
		t.Fatalf("ON_HIT mechanic did not reach the damage operation; hp=%.0f shield=%.0f", enemy.HP, enemy.VirtualShield)
	}
}

func TestChaosGambleCritMultiplierScalesBlockV2DamageOperation(t *testing.T) {
	for seed := int64(1); seed <= 80; seed++ {
		ally := newTestCombatant("ally", "Player", 1000, 0)
		enemy := newTestCombatant("enemy", "Enemy", 2000, 0)
		ally.Actions = []SimAction{{
			ActionType: "GU_SKILL", Name: "Cổ đánh cược", SourceID: "GU_CHAOS", Rank: 1,
			Damage: 100, DamageMult: 1,
			Mechanics: []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "CHAOS_GAMBLE"}},
			OperationEffects: []SimActionOperation{{
				Compiled:   gameplayeffects.CompiledOperation{BlockID: "ACTION", BlockIndex: 0, Trigger: gameplayeffects.TriggerOnAction, Type: gameplayeffects.OpDamage},
				SourceGuID: "GU_CHAOS", SourceRank: 1,
				Action: &SimAction{ActionType: "GU_EFFECT_OPERATION", Name: "Cổ đánh cược", Damage: 100, DamageMult: 1, TargetType: "SINGLE", IsEffectOperation: true},
			}},
		}}
		logs := Simulate(&CombatState{Seed: seed, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
		for _, log := range logs {
			if log.SourceGuID == "GU_CHAOS" && log.Action == "GU_EFFECT_OPERATION" && log.Damage == 500 {
				return
			}
		}
	}
	t.Fatal("no seeded Chaos Gamble crit_500 result scaled the Block v2 damage operation")
}

func TestExhaustedGuCannotTriggerHostileIntentEligibility(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	ally.GuHunger = map[string]int{"GU_EXHAUSTED": 0}
	action := &SimAction{ActionType: "GU_SKILL", SourceID: "GU_EXHAUSTED"}
	if guActionHasAvailableHunger(ally, action) {
		t.Fatal("exhausted Gu should not be eligible for pre-action hostile intent hooks")
	}
}

func TestChaosGambleCritMultiplierScalesBlockV2HPPercentDamage(t *testing.T) {
	for seed := int64(1); seed <= 80; seed++ {
		ally := newTestCombatant("ally", "Player", 1000, 0)
		enemy := newTestCombatant("enemy", "Enemy", 2000, 0)
		ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Cổ đánh cược", SourceID: "GU_CHAOS_HP", Rank: 1,
			Mechanics: []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "CHAOS_GAMBLE"}},
			OperationEffects: []SimActionOperation{{Compiled: gameplayeffects.CompiledOperation{BlockID: "ACTION", BlockIndex: 0, Trigger: gameplayeffects.TriggerOnAction, Type: gameplayeffects.OpDamage}, SourceGuID: "GU_CHAOS_HP", SourceRank: 1,
				Action: &SimAction{ActionType: "GU_EFFECT_OPERATION", Name: "Cổ đánh cược", DamageHPPercent: 0.1, TargetType: "SINGLE", IsEffectOperation: true}}},
		}}
		logs := Simulate(&CombatState{Seed: seed, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
		for _, log := range logs {
			if log.Action == "CHAOS_GAMBLE" && enemy.HP <= 1000 {
				return
			}
		}
	}
	t.Fatal("no seeded Chaos Gamble crit_500 result scaled Block v2 HP-percent damage")
}

func TestChaosGambleCritShieldStripCannotMakeShieldNegative(t *testing.T) {
	for seed := int64(1); seed <= 80; seed++ {
		ally := newTestCombatant("ally", "Player", 1000, 0)
		enemy := newTestCombatant("enemy", "Enemy", 2000, 0)
		enemy.VirtualShield = 1000
		ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Cổ đánh cược", SourceID: "GU_CHAOS_SHIELD", Rank: 1,
			Mechanics: []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "CHAOS_GAMBLE"}},
			OperationEffects: []SimActionOperation{{Compiled: gameplayeffects.CompiledOperation{BlockID: "ACTION", BlockIndex: 0, Trigger: gameplayeffects.TriggerOnAction, Type: gameplayeffects.OpDirectHP}, SourceGuID: "GU_CHAOS_SHIELD", SourceRank: 1,
				Action: &SimAction{ActionType: "GU_EFFECT_OPERATION", Name: "Cổ đánh cược", ShieldStripPercent: 0.25, TargetType: "SINGLE", IsEffectOperation: true}}},
		}}
		Simulate(&CombatState{Seed: seed, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
		if enemy.VirtualShield < 0 {
			t.Fatalf("seed %d left negative shield after crit-scaled strip: %.0f", seed, enemy.VirtualShield)
		}
	}
}

func TestChaosGambleCritScalesMultiplierOnlyOperationBuff(t *testing.T) {
	for seed := int64(1); seed <= 80; seed++ {
		ally := newTestCombatant("ally", "Player", 1000, 0)
		enemy := newTestCombatant("enemy", "Enemy", 2000, 0)
		ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Cổ đánh cược", SourceID: "GU_CHAOS_MULT", Rank: 1,
			Mechanics: []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "CHAOS_GAMBLE"}},
			OperationEffects: []SimActionOperation{{Compiled: gameplayeffects.CompiledOperation{BlockID: "ACTION", BlockIndex: 0, Trigger: gameplayeffects.TriggerOnAction, Type: gameplayeffects.OpDamage}, SourceGuID: "GU_CHAOS_MULT", SourceRank: 1,
				Action: &SimAction{ActionType: "GU_EFFECT_OPERATION", Name: "Cổ đánh cược", DamageMult: 2, TargetType: "SINGLE", IsEffectOperation: true}}},
		}}
		logs := Simulate(&CombatState{Seed: seed, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
		crit := false
		for _, log := range logs {
			if log.Action == "CHAOS_GAMBLE" && strings.Contains(log.Detail, "Tấn công chí mạng") {
				crit = true
			}
			if crit && log.Action == "GU_EFFECT_OPERATION" && strings.Contains(log.Detail, "tăng 10.0x sát thương") {
				return
			}
		}
	}
	t.Fatal("no seeded Chaos Gamble crit_500 result for multiplier-only operation")
}

func TestBlockV2ParentDoesNotDuplicateForgetWillForOperation(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	enemy := newTestCombatant("enemy", "Enemy", 10, 0)
	enemy.Mechanics = []CombatMechanic{{Trigger: "BEFORE_ENEMY_ATTACK", ActionCode: "FORGET_WILL_CONFUSE", Values: map[string]any{
		"activation_chance": 1.0, "backfire_chance": 0.0, "hit_user_chance": 1.0, "hit_self_chance": 0.0,
	}}}
	ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Cổ lạc mục tiêu", SourceID: "GU_FORGET_WILL", Rank: 1,
		OperationEffects: []SimActionOperation{{Compiled: gameplayeffects.CompiledOperation{BlockID: "ACTION", BlockIndex: 0, Trigger: gameplayeffects.TriggerOnAction, Type: gameplayeffects.OpDamage}, SourceGuID: "GU_FORGET_WILL", SourceRank: 1,
			Action: &SimAction{ActionType: "GU_EFFECT_OPERATION", Name: "Cổ lạc mục tiêu", Damage: 10, TargetType: "SINGLE", IsEffectOperation: true}}},
	}}
	logs := Simulate(&CombatState{Seed: 42, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
	redirects := 0
	for _, log := range logs {
		redirects += strings.Count(log.Detail, "đòn [Cổ lạc mục tiêu] lạc sang enemy")
	}
	if redirects != 1 {
		t.Fatalf("FORGET_WILL redirect ran %d times for one damage operation; want once", redirects)
	}
}
