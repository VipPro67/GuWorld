package simulator

import "testing"

func TestResolveBaseDamageMatchesSimulatorDefenseFloor(t *testing.T) {
	if got := ResolveBaseDamage(80, 30); got != 50 {
		t.Fatalf("got %v want 50", got)
	}
	if got := ResolveBaseDamage(5, 30); got != 1 {
		t.Fatalf("got %v want floor 1", got)
	}
}
