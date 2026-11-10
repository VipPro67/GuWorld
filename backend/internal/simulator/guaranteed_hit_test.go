package simulator

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// TestGuaranteedHitBypassesEvasion verifies that an action with GUARANTEED_HIT
// completely ignores the target's evasion chance (even at max cap 75%).
func TestGuaranteedHitBypassesEvasion(t *testing.T) {
	attacker := newTestCombatant("a", "Player", 1000, 0)
	attacker.Speed = 100
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vạn Thú Lực Cổ",
			Damage:     500,
			DamageMult: 1.0,
			TargetType: "SINGLE",
			Mechanics:  []CombatMechanic{{Trigger: "ON_HIT", ActionCode: "GUARANTEED_HIT"}},
		},
	}

	defender := newTestCombatant("d", "Enemy", 1000, 0)
	defender.Speed = 1
	defender.MaxHP = 100000
	defender.HP = 100000
	applyEvasionOps(defender, []statusfx.StatusOp{{
		ID: statusfx.StatusEvasion, Value: 75, Turns: 5, Chance: 1.0,
	}}, 1)

	evaded := 0
	for seed := int64(1); seed <= 50; seed++ {
		logs := Simulate(&CombatState{Seed: seed, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}})
		evaded += logCount(logs, "EVADE_MISS")
	}
	if evaded != 0 {
		t.Fatalf("EVADE_MISS logged %d times against GUARANTEED_HIT action", evaded)
	}
}

// TestGuaranteedHitBypassesBlind verifies that an attacker with BLIND status
// never misses when using a GUARANTEED_HIT action.
func TestGuaranteedHitBypassesBlind(t *testing.T) {
	attacker := newTestCombatant("a", "Player", 1000, 0)
	attacker.Speed = 100
	attacker.IsBlinded = true
	attacker.AccuracyPenalty = 0.90 // 90% miss chance
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vạn Thú Lực Cổ",
			Damage:     500,
			DamageMult: 1.0,
			TargetType: "SINGLE",
			Mechanics:  []CombatMechanic{{Trigger: "ON_HIT", ActionCode: "GUARANTEED_HIT"}},
		},
	}

	defender := newTestCombatant("d", "Enemy", 10000, 0)
	defender.Speed = 1

	missed := 0
	for seed := int64(1); seed <= 50; seed++ {
		logs := Simulate(&CombatState{Seed: seed, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}})
		missed += logCount(logs, "BLIND_MISS")
	}
	if missed != 0 {
		t.Fatalf("BLIND_MISS logged %d times against GUARANTEED_HIT action", missed)
	}
}

// TestGuaranteedHitPiercesFirstTurnEvade verifies that turn 1 FIRST_TURN_EVADE
// (Định Tiên Du) is pierced by GUARANTEED_HIT.
func TestGuaranteedHitPiercesFirstTurnEvade(t *testing.T) {
	attacker := newTestCombatant("a", "Player", 1000, 0)
	attacker.Speed = 100
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vạn Thú Lực Cổ",
			Damage:     500,
			DamageMult: 1.0,
			TargetType: "SINGLE",
			Mechanics:  []CombatMechanic{{Trigger: "ON_HIT", ActionCode: "GUARANTEED_HIT"}},
		},
	}

	defender := newTestCombatant("d", "Enemy", 1000, 0)
	defender.Speed = 1
	defender.Mechanics = []CombatMechanic{
		{Trigger: "BEFORE_HIT", ActionCode: "FIRST_TURN_EVADE"},
	}

	state := &CombatState{Seed: 1, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}}
	logs := Simulate(state)

	// FIRST_TURN_EVADE should not trigger EVADE log; damage must be dealt on turn 1
	if logCount(logs, "EVADE") != 0 {
		t.Fatal("FIRST_TURN_EVADE was unexpectedly triggered against GUARANTEED_HIT")
	}
	foundPierceMsg := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "Đòn đánh Tất Trúng phá vỡ [Định Tiên Du]") {
			foundPierceMsg = true
			break
		}
	}
	if !foundPierceMsg {
		t.Fatal("expected pierce message in log detail when GUARANTEED_HIT pierces FIRST_TURN_EVADE")
	}
	if defender.HP >= 1000 {
		t.Fatalf("defender took no damage: hp=%v", defender.HP)
	}
}

// TestGuaranteedHitRespectsShieldAndInvincible verifies that GUARANTEED_HIT does
// not bypass shield or damage immunity (Bất Hoại).
func TestGuaranteedHitRespectsShieldAndInvincible(t *testing.T) {
	attacker := newTestCombatant("a", "Player", 1000, 0)
	action := SimAction{
		ActionType: "GU_SKILL",
		Name:       "Vạn Thú Lực Cổ",
		Damage:     300,
		Mechanics:  []CombatMechanic{{Trigger: "ON_HIT", ActionCode: "GUARANTEED_HIT"}},
	}

	// 1. Shield absorbs damage normally
	defenderWithShield := newTestCombatant("d1", "Enemy", 1000, 0)
	defenderWithShield.VirtualShield = 500
	state1 := &CombatState{Seed: 1, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defenderWithShield}}
	var logs1 []CombatLog
	applyDamageToTarget(state1, attacker, defenderWithShield, action.Damage, &logs1, 2, false, true, false, rand.New(rand.NewSource(1)))
	if defenderWithShield.VirtualShield != 200 {
		t.Fatalf("expected shield to absorb damage: remaining shield=%v, want 200", defenderWithShield.VirtualShield)
	}
	if defenderWithShield.HP != 1000 {
		t.Fatalf("expected HP untouched while shield holds: hp=%v", defenderWithShield.HP)
	}

	// 2. Invincible blocks damage completely
	defenderInvincible := newTestCombatant("d2", "Enemy", 1000, 0)
	defenderInvincible.IsInvincible = true
	state2 := &CombatState{Seed: 1, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defenderInvincible}}
	var logs2 []CombatLog
	dmgDealt := applyDamageToTarget(state2, attacker, defenderInvincible, action.Damage, &logs2, 2, false, true, false, rand.New(rand.NewSource(1)))
	if dmgDealt != 0 || defenderInvincible.HP != 1000 {
		t.Fatalf("expected Invincible to block damage: dealt=%v hp=%v", dmgDealt, defenderInvincible.HP)
	}
}

// TestGuaranteedHitAllEnemies verifies that an AoE GUARANTEED_HIT hits all enemies
// simultaneously, even if some have evasion.
func TestGuaranteedHitAllEnemies(t *testing.T) {
	attacker := newTestCombatant("a", "Player", 1000, 0)
	attacker.Speed = 100
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vạn Thú Lực Cổ",
			Damage:     1000,
			DamageMult: 1.0,
			TargetType: "ALL_ENEMIES",
			Mechanics:  []CombatMechanic{{Trigger: "ON_HIT", ActionCode: "GUARANTEED_HIT"}},
		},
	}

	d1 := newTestCombatant("d1", "Enemy", 500, 0)
	d1.Speed = 1
	d2 := newTestCombatant("d2", "Enemy", 500, 0)
	d2.Speed = 1
	d2.EvasionChance = 0.70 // 70% evasion
	d3 := newTestCombatant("d3", "Enemy", 500, 0)
	d3.Speed = 1

	state := &CombatState{Seed: 42, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{d1, d2, d3}}
	logs := Simulate(state)

	if logCount(logs, "EVADE_MISS") != 0 {
		t.Fatal("EVADE_MISS occurred during AoE GUARANTEED_HIT")
	}
	for i, d := range []*CombatantAuto{d1, d2, d3} {
		if d.HP > 0 {
			t.Fatalf("enemy d%d should be defeated by 1000 dmg, hp=%v", i+1, d.HP)
		}
	}
	fmt.Println("All enemies hit and defeated successfully!")
}
