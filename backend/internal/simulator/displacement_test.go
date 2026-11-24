package simulator

import (
	"strings"
	"testing"
)

func TestKnockbackDisplacement(t *testing.T) {
	attacker := newTestCombatant("attacker", "Attacker", 1000, 100)
	attacker.GridX, attacker.GridY = 1, 1
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Thỉ Trảo",
			Damage:     50,
			TargetType: "SINGLE",
			Mechanics:  []CombatMechanic{NewCombatMechanic("ON_ACTION", "KNOCKBACK", []byte(`{"steps": 2}`))},
		},
	}

	enemy := newTestCombatant("enemy", "Enemy", 50, 10)
	enemy.GridX, enemy.GridY = 2, 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}

	state := &CombatState{
		Seed:    100,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	if enemy.GridX != 4 {
		t.Fatalf("expected enemy GridX = 4 after knockback(steps=2) from 2, got %d", enemy.GridX)
	}
	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "Đánh lui enemy từ hàng 2 về hàng 4") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected knockback message in logs, got: %+v", logs)
	}
}

func TestTypedDisplaceKnockbackMatchesLegacyResolution(t *testing.T) {
	attacker := newTestCombatant("attacker", "Attacker", 1000, 100)
	attacker.GridX, attacker.GridY = 1, 1
	attacker.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Thỉ Trảo", Damage: 50, TargetType: "SINGLE",
		Mechanics: []CombatMechanic{NewCombatMechanic("ON_ACTION", "DISPLACE", []byte(`{"direction":"KNOCKBACK","steps":2}`))},
	}}
	enemy := newTestCombatant("enemy", "Enemy", 50, 10)
	enemy.GridX, enemy.GridY = 2, 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}
	state := &CombatState{Seed: 100, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)
	if enemy.GridX != 4 {
		t.Fatalf("typed DISPLACE moved target to GridX=%d, want legacy result 4", enemy.GridX)
	}
	for _, log := range logs {
		if strings.Contains(log.Detail, "Đánh lui enemy từ hàng 2 về hàng 4") {
			return
		}
	}
	t.Fatalf("typed DISPLACE did not preserve legacy log: %+v", logs)
}

func TestPullTargetDisplacement(t *testing.T) {
	attacker := newTestCombatant("attacker", "Attacker", 1000, 100)
	attacker.GridX, attacker.GridY = 1, 1
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Giả Xích",
			Damage:     50,
			TargetType: "SINGLE",
			Bypass:     true, // Allow targeting backline
			Mechanics:  []CombatMechanic{NewCombatMechanic("ON_ACTION", "PULL_TARGET", []byte(`{"steps": 3}`))},
		},
	}

	enemy := newTestCombatant("enemy", "Enemy", 50, 10)
	enemy.GridX, enemy.GridY = 4, 2
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}

	state := &CombatState{
		Seed:    100,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	if enemy.GridX != 1 {
		t.Fatalf("expected enemy GridX = 1 after pull(steps=3) from 4, got %d", enemy.GridX)
	}
	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "Kéo mạnh enemy từ hàng 4 lên hàng 1 tiền tuyến") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected pull message in logs, got: %+v", logs)
	}
}

func TestSwapTargetDisplacement(t *testing.T) {
	attacker := newTestCombatant("attacker", "Attacker", 1000, 100)
	attacker.GridX, attacker.GridY = 1, 2
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Hư Không Giới",
			Damage:     50,
			TargetType: "SINGLE",
			Bypass:     true,
			Mechanics:  []CombatMechanic{NewCombatMechanic("ON_ACTION", "SWAP_TARGET", []byte(`{}`))},
		},
	}

	enemy := newTestCombatant("enemy", "Enemy", 50, 10)
	enemy.GridX, enemy.GridY = 5, 4
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}

	state := &CombatState{
		Seed:    100,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	if attacker.GridX != 5 || attacker.GridY != 4 {
		t.Fatalf("expected attacker at (5,4), got (%d,%d)", attacker.GridX, attacker.GridY)
	}
	if enemy.GridX != 1 || enemy.GridY != 2 {
		t.Fatalf("expected enemy at (1,2), got (%d,%d)", enemy.GridX, enemy.GridY)
	}

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "Hoán vị không gian!") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected swap message in logs, got: %+v", logs)
	}
}

func TestRetreatSelfDisplacement(t *testing.T) {
	attacker := newTestCombatant("attacker", "Attacker", 1000, 100)
	attacker.GridX, attacker.GridY = 1, 3
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Kim Thiền Thoát Xác",
			Damage:     50,
			TargetType: "SELF",
			Mechanics:  []CombatMechanic{NewCombatMechanic("ON_ACTION", "RETREAT_SELF", []byte(`{"steps": 2}`))},
		},
	}

	enemy := newTestCombatant("enemy", "Enemy", 50, 10)
	enemy.GridX, enemy.GridY = 2, 3
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}

	state := &CombatState{
		Seed:    100,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	if attacker.GridX != 3 {
		t.Fatalf("expected attacker GridX = 3 after retreat(steps=2) from 1, got %d", attacker.GridX)
	}
	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "chủ động rút lui từ hàng 1 về hàng 3") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected retreat message in logs, got: %+v", logs)
	}
}

func TestDisplacementChangesTargetPriority(t *testing.T) {
	attacker := newTestCombatant("attacker", "Attacker", 10000, 10)
	attacker.GridX, attacker.GridY = 1, 1
	// Action has KNOCKBACK with steps: 3.
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Trọng Kích",
			Damage:     10,
			TargetType: "SINGLE",
			Mechanics:  []CombatMechanic{NewCombatMechanic("ON_ACTION", "KNOCKBACK", []byte(`{"steps": 3}`))},
		},
	}

	// Enemy A starts at front (GridX=2). Enemy B starts behind (GridX=3).
	enemyA := newTestCombatant("enemyA", "Enemy A", 1000, 1)
	enemyA.GridX, enemyA.GridY = 2, 1
	enemyA.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}

	enemyB := newTestCombatant("enemyB", "Enemy B", 1000, 1)
	enemyB.GridX, enemyB.GridY = 3, 1
	enemyB.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}

	state := &CombatState{
		Seed:    100,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{enemyA, enemyB},
	}

	logs := Simulate(state)

	// In Turn 1, attacker attacks enemyA (because enemyA was at 2 < 3).
	// In Turn 4 (attacker's next turn), enemyA was knocked back to 5. Now enemyB is at 3 < 5, so attacker targets enemyB!
	foundFirstHitA := false
	foundSecondHitB := false
	hitCount := 0
	for _, l := range logs {
		if l.ActorId == "attacker" && l.Action == "GU_SKILL" {
			hitCount++
			if hitCount == 1 && l.TargetId == "enemyA" {
				foundFirstHitA = true
			}
			if hitCount == 2 && l.TargetId == "enemyB" {
				foundSecondHitB = true
				break
			}
		}
	}

	if !foundFirstHitA {
		t.Fatalf("expected attacker to target enemyA on first attack")
	}
	if !foundSecondHitB {
		t.Fatalf("expected attacker to target enemyB on second attack after enemyA was knocked back behind enemyB")
	}
}
