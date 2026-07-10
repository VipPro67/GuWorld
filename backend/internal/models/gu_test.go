package models

import "testing"

func TestGuEffectsScanNormalizesBooleanFlagsIntoActiveSection(t *testing.T) {
	var effects GuEffects
	err := effects.Scan([]byte(`{"aoe":true,"disabled":false,"damage":25.5}`))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	gu := Gu{Effects: effects}
	active := GetGuEffectSection(gu, "active")
	if active["aoe"] != 1.0 {
		t.Errorf(`active["aoe"] = %v, want 1.0`, active["aoe"])
	}
	if active["disabled"] != 0.0 {
		t.Errorf(`active["disabled"] = %v, want 0.0`, active["disabled"])
	}
	if active["damage"] != 25.5 {
		t.Errorf(`active["damage"] = %v, want 25.5`, active["damage"])
	}
}

func TestGuEffectsScanIgnoresUnsupportedValues(t *testing.T) {
	var effects GuEffects
	err := effects.Scan([]byte(`{"active":{"damage":"high"}}`))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	gu := Gu{Effects: effects}
	active := GetGuEffectSection(gu, "active")
	if len(active) != 0 {
		t.Errorf("active effects = %v, want empty", active)
	}
}
