package simulator

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestStatusStacksReadsEffectsAndDots(t *testing.T) {
	c := &CombatantAuto{
		ActiveEffects: []Effect{
			{ID: statusfx.StatusBurn, Value: 8, Duration: 3},
			{ID: statusfx.StatusDefUp, Value: 25, Duration: 3},
			{ID: statusfx.StatusBleed, Value: 30, Duration: 0}, // expired
		},
		ActiveDoT: []DoTEffect{
			{Type: "bleed", Value: 20, Turns: 2},
			{Type: "poison", Value: 12, Turns: 3},
		},
	}

	count, strength := statusStacks(c, statusfx.StatusBurn)
	if count != 1 || strength != 8 {
		t.Fatalf("statusStacks(BURN) = (%d, %v); want (1, 8)", count, strength)
	}

	count, strength = statusStacks(c, statusfx.StatusBleed)
	if count != 1 || strength != 20 {
		t.Fatalf("statusStacks(BLEED) = (%d, %v); want (1, 20) — expired effects must not count", count, strength)
	}

	if !hasStatus(c, statusfx.StatusPoison) {
		t.Fatal("hasStatus(POISON) = false; want true from ActiveDoT")
	}
	if hasStatus(c, statusfx.StatusBleed) == false {
		t.Fatal("hasStatus(BLEED) = false; want true from ActiveDoT")
	}
}

func TestRemoveStatusStacksClearsBothLists(t *testing.T) {
	c := &CombatantAuto{
		ActiveEffects: []Effect{
			{ID: statusfx.StatusBurn, Value: 8, Duration: 3},
			{ID: statusfx.StatusDefUp, Value: 25, Duration: 3},
		},
		ActiveDoT: []DoTEffect{
			{Type: "burn", Value: 10, Turns: 2},
			{Type: "regen", Value: 15, Turns: 2},
		},
	}

	count, strength := removeStatusStacks(c, statusfx.StatusBurn)
	if count != 2 || strength != 18 {
		t.Fatalf("removeStatusStacks(BURN) = (%d, %v); want (2, 18)", count, strength)
	}
	if len(c.ActiveEffects) != 1 || c.ActiveEffects[0].ID != statusfx.StatusDefUp {
		t.Fatalf("ActiveEffects = %+v; want only DEF_UP", c.ActiveEffects)
	}
	if len(c.ActiveDoT) != 1 || c.ActiveDoT[0].Type != "regen" {
		t.Fatalf("ActiveDoT = %+v; want only regen", c.ActiveDoT)
	}
}

func TestSelfTargetStatusDoesNotLeakToEnemies(t *testing.T) {
	actor := &CombatantAuto{ID: "actor", Name: "Actor", HP: 100, MaxHP: 100}
	enemy1 := &CombatantAuto{ID: "e1", Name: "Enemy 1", HP: 100, MaxHP: 100}
	enemy2 := &CombatantAuto{ID: "e2", Name: "Enemy 2", HP: 100, MaxHP: 100}

	ops := []statusfx.StatusOp{
		{ID: statusfx.StatusHaste, Value: 30, Turns: 2, Chance: 1.0, ApplicationMode: "DIRECT"},
	}
	var logs []CombatLog
	applyStatusOps(actor, []*CombatantAuto{enemy1, enemy2}, ops, 1, 1, nil, &logs)

	if !hasStatus(actor, statusfx.StatusHaste) {
		t.Fatal("expected actor to receive SelfTarget status HASTE")
	}
	if hasStatus(enemy1, statusfx.StatusHaste) || hasStatus(enemy2, statusfx.StatusHaste) {
		t.Fatal("SelfTarget status must NEVER leak to enemy targets")
	}
}
