package models

import (
	"testing"
)

func TestGetFlatGuEffectsPassiveOnlyDoesNotPanic(t *testing.T) {
	gu := Gu{Effects: GuEffects{Passive: &PassiveEffect{StatBaseAttack: 10}}}
	effects := GetFlatGuEffects(gu)
	if effects["flat_damage"] != 10 {
		t.Fatalf("flat_damage = %v, want 10", effects["flat_damage"])
	}
}

func TestGetFlatGuEffectsIncludesActiveCaptureBonus(t *testing.T) {
	gu := Gu{Effects: GuEffects{Active: &ActiveEffect{CaptureBonus: 0.25}}}
	effects := GetFlatGuEffects(gu)
	if effects["capture_bonus"] != 0.25 {
		t.Fatalf("capture_bonus = %v, want 0.25", effects["capture_bonus"])
	}
}

func TestGuEffectsScanRejectsStructurallyInvalidCatalogData(t *testing.T) {
	var effects GuEffects
	if err := effects.Scan([]byte(`{"active":[]}`)); err == nil {
		t.Fatal("invalid active namespace was accepted")
	}
}

func TestGuEffectsScanAcceptsLegacyUnwiredSections(t *testing.T) {
	var effects GuEffects
	if err := effects.Scan([]byte(`{"support":{"heal_flat":10}}`)); err != nil {
		t.Fatalf("legacy support section rejected: %v", err)
	}
}
