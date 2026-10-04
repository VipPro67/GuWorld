package statusfx

import (
	"math/rand"
	"testing"
)

func TestResolveStatusKnown(t *testing.T) {
	op, err := ResolveStatus("BURN", 100, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.ID != StatusBurn || op.Value != 100 || op.Turns != 5 {
		t.Fatalf("got %+v", op)
	}
}

func TestResolveStatusDefaultTurns(t *testing.T) {
	op, err := ResolveStatus("SLEEP", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.Turns != 2 {
		t.Fatalf("expected default 2 turns, got %d", op.Turns)
	}
}

func TestResolveStatusCaseInsensitive(t *testing.T) {
	op, err := ResolveStatus("burn", 10, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.ID != StatusBurn {
		t.Fatalf("expected canonical ID %s, got %s", StatusBurn, op.ID)
	}
}

func TestResolveStatusUnknown(t *testing.T) {
	if _, err := ResolveStatus("NOT_A_STATUS", 0, 0); err == nil {
		t.Fatal("expected unknown status error")
	}
}

func TestSpecKnownBehaviours(t *testing.T) {
	for _, id := range []string{
		StatusBurn, StatusFear, StatusConfuse, StatusStealth, StatusSleep,
		StatusBlind, StatusCharm, StatusBind, StatusHaste, StatusArmorBreak,
		StatusDamageReduction, StatusInvincible, StatusLastStand, StatusEvasion, StatusReviveLock,
		StatusBleed, StatusPoison, StatusStun, StatusSlow, StatusSlow1,
		StatusSlowDamage, StatusWeak, StatusFreeze, StatusSilence,
		StatusThorns, StatusRegen, StatusAtkUp, StatusDefUp, StatusTaunt,
		StatusEssenceLeak, StatusGuSeal, StatusSoulWound, StatusThoughtChaos,
		StatusDaoConflict, StatusBacklash,
	} {
		if _, ok := Spec(id); !ok {
			t.Fatalf("status %s missing from registry", id)
		}
	}
}

func TestResolveStatusWithTier(t *testing.T) {
	op, err := ResolveStatusWithTier(StatusStun, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.ID != StatusStun || op.Turns != 1 {
		t.Fatalf("expected stun tier 2 to have 1 turn, got %+v", op)
	}

	opWeak, err := ResolveStatusWithTier(StatusWeak, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opWeak.Value != 30 || opWeak.Turns != 4 {
		t.Fatalf("expected weak tier 3 to have 30%% and 4 turns, got %+v", opWeak)
	}
}

func TestCanApplyStatusUsesInjectedRNG(t *testing.T) {
	spec := StatusSpec{BaseChance: 0.5}
	first := rand.New(rand.NewSource(42))
	second := rand.New(rand.NewSource(42))
	for i := 0; i < 20; i++ {
		got := CanApplyStatus(first, 1, 1, nil, nil, spec, 0)
		want := CanApplyStatus(second, 1, 1, nil, nil, spec, 0)
		if got != want {
			t.Fatalf("roll %d differs for identical seeds: got %v want %v", i, got, want)
		}
	}
}

func TestCasterAttenuationFactor(t *testing.T) {
	// Equal or lower caster rank -> no attenuation
	if got := CasterAttenuationFactor(4, 4); got != 1.0 {
		t.Fatalf("expected 1.0 for equal rank, got %f", got)
	}
	if got := CasterAttenuationFactor(5, 3); got != 1.0 {
		t.Fatalf("expected 1.0 when caster < gu, got %f", got)
	}

	// Rank 5 caster using Rank 4 Gu: 20 / 100 = 0.2
	if got := CasterAttenuationFactor(4, 5); got != 0.2 {
		t.Fatalf("expected 0.2 for Rank 4 Gu on Rank 5 caster, got %f", got)
	}

	// Rank 9 caster using Rank 4 Gu: 20 / 1,000,000 = 0.00002 -> capped at 0.0001
	if got := CasterAttenuationFactor(4, 9); got != 0.0001 {
		t.Fatalf("expected 0.0001 floor for Rank 4 Gu on Rank 9 caster, got %f", got)
	}
}

func TestTargetSuppressionFactor(t *testing.T) {
	// Equal or lower target rank -> no suppression
	if factor, penalty := TargetSuppressionFactor(4, 4); factor != 1.0 || penalty != 0 {
		t.Fatalf("expected (1.0, 0) for equal rank, got (%f, %d)", factor, penalty)
	}

	// Gap 1 (Target 5 vs Gu 4) -> 1 - 0.2*1 = 0.8, penalty 0
	if factor, penalty := TargetSuppressionFactor(4, 5); factor != 0.8 || penalty != 0 {
		t.Fatalf("expected (0.8, 0) for gap 1, got (%f, %d)", factor, penalty)
	}

	// Gap 2 (Target 6 vs Gu 4) -> 1 - 0.2*2 = 0.6, penalty 1
	if factor, penalty := TargetSuppressionFactor(4, 6); factor != 0.6 || penalty != 1 {
		t.Fatalf("expected (0.6, 1) for gap 2, got (%f, %d)", factor, penalty)
	}

	// Gap 5+ (Target 9 vs Gu 4) -> factor 0.0 (immune), penalty 2
	if factor, penalty := TargetSuppressionFactor(4, 9); factor != 0.0 || penalty != 2 {
		t.Fatalf("expected (0.0, 2) for gap 5, got (%f, %d)", factor, penalty)
	}
}
