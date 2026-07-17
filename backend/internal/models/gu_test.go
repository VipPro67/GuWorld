package models

import "testing"

func TestGuEffectsScan(t *testing.T) {
	var effects GuEffects
	err := effects.Scan([]byte(`{"active":{"damage_flat":25.5,"curve_attack":true}}`))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if effects.Active == nil {
		t.Fatalf("Active is nil")
	}
	if effects.Active.DamageFlat != 25.5 {
		t.Errorf("DamageFlat = %v, want 25.5", effects.Active.DamageFlat)
	}
	if !effects.Active.CurveAttack {
		t.Errorf("CurveAttack = false, want true")
	}
}
