package simulator

import (
	"math/rand"
	"strings"
	"testing"

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
