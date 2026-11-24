package simulator

import (
	"math/rand"
	"strings"
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// Enemy REBORN in_place: previously only Allies were scanned, so an enemy
// carrying REBORN could never revive. Enemies stand back up (no rewind).
func TestEnemyRebornInPlace(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100000, 0)
	ally.Speed = 100
	ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Sát", Damage: 10000}}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 1)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}
	enemy.Mechanics = []CombatMechanic{
		mech("ON_DEATH", "REBORN", `{"chance":1,"mode":"in_place","max_uses":1,"revive_hp_percent":0.5}`),
	}

	state := &CombatState{Seed: 7, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if !hasDetail(logs, "vùng dậy") {
		t.Fatal("expected enemy to revive in place (missing 'vùng dậy' log)")
	}
	if logCount(logs, "TRIGGERED") < 1 {
		t.Fatal("expected at least one REBORN TRIGGERED log")
	}
}

// FIRST_TURN_EVADE dodges the first strike on turn 1 only: with MultiHit 2,
// exactly one EVADE fires and the follow-up hit lands.
func TestFirstTurnEvadeConsumesOnce(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100000, 0)
	ally.Speed = 100
	ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Song", Damage: 100, MultiHit: 2}}
	enemy := newTestCombatant("enemy", "Enemy", 100000, 0)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}
	enemy.Mechanics = []CombatMechanic{mech("BEFORE_HIT", "FIRST_TURN_EVADE", `{}`)}

	state := &CombatState{Seed: 11, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	// Per-hit EVADE logs are merged into the consolidated action entry, so
	// count by detail text: exactly one strike was negated on turn 1.
	evaded := 0
	for _, l := range logs {
		if l.Turn == 1 && strings.Contains(l.Detail, "Định Tiên Du") {
			evaded++
		}
	}
	if evaded != 1 {
		t.Fatalf("evaded strikes on turn 1 = %d, want exactly 1 (first strike only)", evaded)
	}
	if enemy.HP >= 100000 {
		t.Fatal("expected follow-up hits to land after the evaded first strike")
	}
}

func TestTypedFirstTurnEvadeConsumesOnce(t *testing.T) {
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "FIRST_TURN_EVADE", Trigger: gameplayeffects.TriggerBeforeHit,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpHitRule, SubType: "FIRST_TURN_EVADE"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	ally := newTestCombatant("ally", "Player", 100000, 0)
	ally.Speed = 100
	ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Song", Damage: 100, MultiHit: 2}}
	enemy := newTestCombatant("enemy", "Enemy", 100000, 0)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}
	enemy.Mechanics = CompiledCombatMechanics(compiled, false)
	if len(enemy.Mechanics) != 1 || enemy.Mechanics[0].Trigger != "BEFORE_HIT" || enemy.Mechanics[0].ActionCode != "FIRST_TURN_EVADE" {
		t.Fatalf("typed rule was not projected to the defensive hit phase: %+v", enemy.Mechanics)
	}
	state := &CombatState{Seed: 11, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)
	evaded := 0
	for _, log := range logs {
		if log.Turn == 1 && strings.Contains(log.Detail, "Định Tiên Du") {
			evaded++
		}
	}
	if evaded != 1 || enemy.HP >= 100000 {
		t.Fatalf("typed first-turn evade: evaded=%d targetHP=%v, want one evasion then a landed follow-up hit", evaded, enemy.HP)
	}
}

func TestTypedLethalLuckWardUsesExistingDeathConsumer(t *testing.T) {
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "LETHAL_LUCK_WARD", Trigger: gameplayeffects.TriggerBeforeHit,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpHitRule, SubType: "LETHAL_LUCK_WARD", Params: []byte(`{"chance":1,"shield_percent":0.5,"shield_min":250}`)}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mechanics := CompiledCombatMechanics(compiled, false)
	if len(mechanics) != 1 || mechanics[0].Trigger != "BEFORE_HIT" || mechanics[0].ActionCode != "LETHAL_LUCK_WARD" {
		t.Fatalf("typed lethal rule was not projected to its existing death consumer: %+v", mechanics)
	}
	if mechanics[0].Values["chance"] != float64(1) || mechanics[0].Values["shield_percent"] != 0.5 || mechanics[0].Values["shield_min"] != float64(250) {
		t.Fatalf("authored lethal rule parameters were not projected: %+v", mechanics[0].Values)
	}
	target := newTestCombatant("ward", "Ward", 1000, 0)
	target.HP = 0
	target.Mechanics = mechanics
	var logs []CombatLog
	if !applyLethalLuckWard(target, &logs, 1, rand.New(rand.NewSource(7))) {
		t.Fatal("chance=1 should activate")
	}
	if target.HP != 1 || target.VirtualShield != 500 {
		t.Fatalf("configured lethal ward result hp=%v shield=%v, want hp=1 shield=500", target.HP, target.VirtualShield)
	}
}

func TestTypedReviveUsesExistingDeathConsumer(t *testing.T) {
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "REVIVE", Trigger: gameplayeffects.TriggerOnDeath,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpRevive, Params: []byte(`{"chance":1,"mode":"in_place","max_uses":1,"revive_hp_percent":0.5}`)}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mechanics := CompiledCombatMechanics(compiled, false)
	if len(mechanics) != 1 || mechanics[0].Trigger != "ON_DEATH" || mechanics[0].ActionCode != "REBORN" {
		t.Fatalf("typed REVIVE was not projected to the existing death consumer: %+v", mechanics)
	}

	ally := newTestCombatant("ally", "Player", 100000, 0)
	ally.Speed = 100
	ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Sát", Damage: 10000}}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 1)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}
	enemy.Mechanics = mechanics
	logs := Simulate(&CombatState{Seed: 7, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
	if !hasDetail(logs, "vùng dậy") {
		t.Fatal("typed REVIVE did not restore the combatant")
	}
}

func TestTypedNirvanaReviveProjectsToNirvanaConsumer(t *testing.T) {
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "NIRVANA", Trigger: gameplayeffects.TriggerOnDeath,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpRevive, SubType: "NIRVANA_FIRE", Params: []byte(`{"revive_hp_percent":0.4,"minimum_revive_hp":20,"clear_debuffs":true,"retaliation_damage_percent":0.8,"minimum_retaliation_damage":30}`)}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mechanics := CompiledCombatMechanics(compiled, false)
	if len(mechanics) != 1 || mechanics[0].Trigger != "ON_DEATH" || mechanics[0].ActionCode != "NIRVANA_FIRE" {
		t.Fatalf("typed NIRVANA_FIRE REVIVE was not projected to its executor: %+v", mechanics)
	}

	target := newTestCombatant("target", "Player", 100, 0)
	target.HP = 0
	target.BaseDmg = 100
	target.Mechanics = mechanics
	opponent := newTestCombatant("opponent", "Enemy", 200, 1)
	state := &CombatState{Allies: []*CombatantAuto{target}, Enemies: []*CombatantAuto{opponent}}
	if !applyNirvanaRebirth(state, target, nil, 1) || target.HP != 40 || opponent.HP != 120 {
		t.Fatalf("typed NIRVANA_FIRE result hp=%v opponent hp=%v", target.HP, opponent.HP)
	}
}

func TestTypedCounterThresholdHitRuleControlsHitShieldAndImmunity(t *testing.T) {
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "SAT_KHI", Trigger: gameplayeffects.TriggerOnBattleStart,
		Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Operations: []gameplayeffects.Operation{{
			Op: gameplayeffects.OpHitRule, SubType: "COUNTER_THRESHOLD",
			Params: []byte(`{"key":"sat_khi_level","threshold":10,"guaranteed":true,"bypass_shield":true,"bypass_immunity":true}`),
		}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	mechanics := CompiledCombatMechanics(compiled, false)
	if len(mechanics) != 1 || mechanics[0].ActionCode != "HIT_RULE_COUNTER_THRESHOLD" {
		t.Fatalf("typed counter hit rule projection = %+v", mechanics)
	}

	actor := newTestCombatant("a", "Player", 1000, 0)
	actor.CustomState = map[string]int{"sat_khi_level": 11}
	actor.Mechanics = mechanics
	if !counterThresholdHitRule(actor, "guaranteed") || !counterThresholdHitRule(actor, "bypass_shield") || !satKhiPiercesImmunity(actor) {
		t.Fatal("counter level above configured threshold must enable all three authored hit effects")
	}
	actor.CustomState["sat_khi_level"] = 10
	if counterThresholdHitRule(actor, "guaranteed") || counterThresholdHitRule(actor, "bypass_shield") || satKhiPiercesImmunity(actor) {
		t.Fatal("counter equal to threshold must not enable strict-greater-than effects")
	}

	actor.CustomState["sat_khi_level"] = 11
	actor.Speed = 100
	actor.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đấm", Damage: 50}}
	defender := newTestCombatant("d", "Enemy", 100000, 0)
	defender.Speed = 1
	defender.HP = 100000
	defender.MaxHP = 100000
	defender.EvasionChance = 0.75
	defender.VirtualShield = 500
	defender.IsInvincible = true
	logs := Simulate(&CombatState{Seed: 6, Allies: []*CombatantAuto{actor}, Enemies: []*CombatantAuto{defender}})
	if logCount(logs, "EVADE_MISS") != 0 {
		t.Fatal("typed counter hit rule should guarantee the attack through evasion")
	}
	if defender.HP >= 100000 || defender.VirtualShield != 500 || defender.IsInvincible {
		t.Fatalf("typed counter hit rule should pierce shield and immunity: hp=%v shield=%v invincible=%v", defender.HP, defender.VirtualShield, defender.IsInvincible)
	}
}

func TestTypedTurnOrderRulesProjectParameters(t *testing.T) {
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{
		{ID: "SEIZE", Trigger: gameplayeffects.TriggerOnBattleStart, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}, Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpTurnOrderRule, SubType: "SEIZE_INITIATIVE", Params: []byte(`{"lifespan_ratio":4}`)}}},
		{ID: "SUPPRESS", Trigger: gameplayeffects.TriggerOnBattleStart, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}, Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpTurnOrderRule, SubType: "SUPPRESS_ENEMY_PRIORITY"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mechanics := CompiledCombatMechanics(compiled, false)
	if len(mechanics) != 2 || mechanics[0].ActionCode != "SEIZE_INITIATIVE" || mechanics[0].Values["lifespan_ratio"] != float64(4) || mechanics[1].ActionCode != "SUPPRESS_PRIORITY" {
		t.Fatalf("typed turn-order operations did not project to current consumers: %+v", mechanics)
	}
	ally := newTestCombatant("ally", "Ally", 1000, 1)
	ally.Lifespan = 100
	ally.Mechanics = []CombatMechanic{mechanics[0]}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 1)
	enemy.Lifespan = 300
	enemy.Mechanics = []CombatMechanic{mechanics[1]}
	state := &CombatState{Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	if winner, suppressed := seizeInitiative(state, []*CombatantAuto{ally, enemy}); winner != nil || len(suppressed) != 1 || suppressed[0] != ally.Name {
		t.Fatalf("typed suppressor did not block the qualifying seizer: winner=%v suppressed=%v", winner, suppressed)
	}
	enemy.Mechanics = nil
	if winner, suppressed := seizeInitiative(state, []*CombatantAuto{ally, enemy}); winner != ally || len(suppressed) != 0 {
		t.Fatalf("typed seizer did not honor lifespan_ratio: winner=%v suppressed=%v", winner, suppressed)
	}
}

// COMBAT_PASSIVE is the canonical alias of the PASSIVE trigger: both must
// double damage identically (and both beat the no-mechanic baseline).
func TestCombatPassiveAliasMatchesPassive(t *testing.T) {
	build := func(trigger string) *CombatState {
		ally := newTestCombatant("ally", "Player", 100000, 0)
		ally.Speed = 100
		ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Chém", Damage: 100}}
		if trigger != "" {
			ally.Mechanics = []CombatMechanic{mech(trigger, "DOUBLE_DAMAGE_DEALT", `{}`)}
		}
		enemy := newTestCombatant("enemy", "Enemy", 100000, 0)
		enemy.Speed = 1
		enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}
		return &CombatState{Seed: 21, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	}

	legacy, alias, plain := build("PASSIVE"), build("COMBAT_PASSIVE"), build("")
	Simulate(legacy)
	Simulate(alias)
	Simulate(plain)

	if legacy.Enemies[0].HP != alias.Enemies[0].HP {
		t.Fatalf("PASSIVE enemy HP = %v, COMBAT_PASSIVE = %v, want identical",
			legacy.Enemies[0].HP, alias.Enemies[0].HP)
	}
	if !(alias.Enemies[0].HP < plain.Enemies[0].HP) {
		t.Fatalf("aliased double-damage HP = %v, plain = %v, want aliased lower",
			alias.Enemies[0].HP, plain.Enemies[0].HP)
	}
}

// Opt-in status proc: Chance 1.0 always applies; a seed whose first RNG draw
// misses a 0.5 proc proves the miss path logs STATUS_MISS and applies nothing.
func TestStatusProcChanceAppliesAndMisses(t *testing.T) {
	mkDefender := func() *CombatantAuto {
		d := newTestCombatant("def", "Enemy", 1000, 0)
		d.Speed = 1
		d.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Can", Damage: 1}}
		return d
	}
	mkAttacker := func(chance float64) *CombatantAuto {
		a := newTestCombatant("atk", "Player", 100000, 0)
		a.Speed = 100
		a.Actions = []SimAction{{
			ActionType: "GU_SKILL", Name: "Danh", Damage: 10,
			StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusBleed, Value: 50, Turns: 2, Chance: chance}},
		}}
		return a
	}

	// Chance 1.0: always applies (Float64() < 1.0 guaranteed).
	state := &CombatState{Seed: 31, Allies: []*CombatantAuto{mkAttacker(1.0)}, Enemies: []*CombatantAuto{mkDefender()}}
	Simulate(state)
	if len(state.Enemies[0].ActiveDoT) == 0 {
		t.Fatal("chance 1.0 status never applied")
	}

	// Miss path: the proc roll is the first RNG draw in applyStatusOpTo, so a
	// seed whose first draw >= chance skips with exactly one STATUS_MISS.
	attacker := mkAttacker(0.5)
	applied := false
	for s := int64(0); s < 10000; s++ {
		var logs []CombatLog
		defender := mkDefender()
		applyStatusOps(attacker, []*CombatantAuto{defender}, attacker.Actions[0].StatusOps, 1, 1, rand.New(rand.NewSource(s)), &logs)
		if logCount(logs, "STATUS_MISS") != 1 {
			continue // this seed procced; keep searching for a miss
		}
		if len(defender.ActiveDoT) != 0 {
			t.Fatal("missed proc still applied DoT")
		}
		applied = true
		break
	}
	if !applied {
		t.Fatal("no miss seed found")
	}
}
