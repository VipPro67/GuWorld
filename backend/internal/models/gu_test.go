package models

import "testing"

func TestGuEffectsScanNormalizesBooleanFlags(t *testing.T) {
	var effects GuEffects

	err := effects.Scan([]byte(`{"aoe":true,"disabled":false,"damage":25.5}`))
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if effects["aoe"] != 1 {
		t.Errorf(`effects["aoe"] = %v, want 1`, effects["aoe"])
	}
	if effects["disabled"] != 0 {
		t.Errorf(`effects["disabled"] = %v, want 0`, effects["disabled"])
	}
	if effects["damage"] != 25.5 {
		t.Errorf(`effects["damage"] = %v, want 25.5`, effects["damage"])
	}
}

func TestGuEffectsScanRejectsUnsupportedValues(t *testing.T) {
	var effects GuEffects

	err := effects.Scan([]byte(`{"damage":"high"}`))
	if err == nil {
		t.Fatal("Scan() error = nil, want unsupported value error")
	}
}
