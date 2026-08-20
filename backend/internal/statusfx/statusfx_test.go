package statusfx

import "testing"

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
		StatusDamageReduction, StatusInvincible, StatusReviveLock,
		StatusBleed, StatusPoison,
	} {
		if _, ok := Spec(id); !ok {
			t.Fatalf("status %s missing from registry", id)
		}
	}
}
