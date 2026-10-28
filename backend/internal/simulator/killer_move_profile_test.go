package simulator

import (
	"strings"
	"testing"
)

func TestSimulator_ActionCooldown(t *testing.T) {
	actor := newTestCombatant("p1", "Player", 500, 50)
	actor.MaxMana = 200
	actor.Mana = 200
	actor.Speed = 20
	actor.Actions = []SimAction{
		{
			ActionType:    "KILLER_MOVE",
			Name:          "U Nguyệt Độc Nhận",
			SourceID:      "km_1",
			KillerMoveID:  "km_1",
			Damage:        150,
			DamageMult:    1.0,
			SuccessChance: 1.0, // 100% success
			CooldownTurns: 2,   // 2 turns cooldown
		},
		{
			ActionType: "BASIC_ATTACK",
			Name:       "Đánh thường",
			Damage:     50,
			DamageMult: 1.0,
		},
	}

	enemy := newTestCombatant("m1", "Enemy", 1000, 20)
	enemy.Speed = 10

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{actor},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)
	kmCount := 0
	basicCount := 0
	for _, l := range logs {
		if strings.Contains(l.Detail, "U Nguyệt Độc Nhận") {
			kmCount++
		}
		if strings.Contains(l.Detail, "Đánh thường") && l.ActorId == "p1" {
			basicCount++
		}
	}
	if kmCount < 2 {
		t.Fatalf("expected killer move to be used at least twice across long battle, got %d", kmCount)
	}
	if basicCount < 2 {
		t.Fatalf("expected basic attack to fill turns while on cooldown, got %d", basicCount)
	}
}

func TestSimulator_BackfireOnFailure(t *testing.T) {
	actor := newTestCombatant("p1", "Player", 500, 50)
	actor.MaxMana = 200
	actor.Mana = 200
	actor.Speed = 20
	actor.Actions = []SimAction{
		{
			ActionType:     "KILLER_MOVE",
			Name:           "Bạo Liệt Trảm",
			SourceID:       "km_fail",
			KillerMoveID:   "km_fail",
			Damage:         300,
			DamageMult:     1.0,
			SuccessChance:  0.01, // 1% chance, guaranteed to fail with seed 1
			BackfireDamage: 80,
			BackfireStatus: "THOUGHT_CHAOS",
			CooldownTurns:  3,
		},
	}

	enemy := newTestCombatant("m1", "Enemy", 1000, 10)
	enemy.Speed = 10

	state := &CombatState{
		Seed:    1,
		Allies:  []*CombatantAuto{actor},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)
	hasBackfireLog := false
	for _, l := range logs {
		if l.Action == "KILLER_MOVE_FAILED" && strings.Contains(l.Detail, "phản phệ") {
			hasBackfireLog = true
			break
		}
	}
	if !hasBackfireLog {
		for _, l := range logs {
			t.Logf("BackfireLog: %s | %s", l.Action, l.Detail)
		}
		t.Fatalf("expected backfire log message")
	}
}

func TestSimulator_GuHouseDamageAbsorption(t *testing.T) {
	player := newTestCombatant("p1", "Player", 1000, 50)
	player.Speed = 5 // let enemy attack first
	player.GuHouse = &GuHouseState{
		ID:              "house_1",
		Name:            "Vấn Đạo Lâu",
		Rank:            6,
		MaxHP:           600,
		HP:              600,
		DamageReduction: 0.20, // 20% damage reduction
	}

	enemy := newTestCombatant("boss_1", "Enemy", 300, 100)
	enemy.Speed = 20 // attacks first
	enemy.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Name:       "Hống Thiên Kích",
			Damage:     100,
			DamageMult: 1.0,
		},
	}

	state := &CombatState{
		Seed:    99,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	hasHouseAbsorbLog := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Tiên Cổ Ốc]") {
			hasHouseAbsorbLog = true
			break
		}
	}
	if !hasHouseAbsorbLog {
		for _, l := range logs {
			t.Logf("HouseLog: %s | %s", l.Action, l.Detail)
		}
		t.Fatalf("expected GU_HOUSE_ABSORB log")
	}
	if player.HP != 1000 {
		t.Fatalf("player HP should be protected by GuHouse, got %.0f", player.HP)
	}
}
