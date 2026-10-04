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

func TestResolveBasicDamageRoutesThroughReductionAndShield(t *testing.T) {
	got := ResolveBasicDamage(50, 10, .25, 100, 20)
	if got.Damage != 30 || got.ShieldAbsorbed != 20 || got.HPDamage != 10 || got.HP != 90 || got.Shield != 0 {
		t.Fatalf("unexpected basic damage result: %+v", got)
	}
	blocked := ResolveBasicDamage(50, 0, 1, 100, 20)
	if blocked.Damage != 0 || blocked.HP != 100 || blocked.Shield != 20 {
		t.Fatalf("invulnerability did not block damage: %+v", blocked)
	}
}
