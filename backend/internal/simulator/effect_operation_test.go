package simulator

import (
	"strings"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

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
