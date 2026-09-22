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

func TestStatusApplicationsUnmarshalAllForms(t *testing.T) {
	// Legacy string form.
	var legacy GuEffects
	if err := legacy.Scan([]byte(`{"active":{"apply_status":"BIND","status_duration_ticks":6}}`)); err != nil {
		t.Fatalf("string form rejected: %v", err)
	}
	if len(legacy.Active.ApplyStatus) != 1 || legacy.Active.ApplyStatus[0].ID != "BIND" || legacy.Active.ApplyStatus[0].Turns != 0 {
		t.Fatalf("string form parsed wrong: %+v", legacy.Active.ApplyStatus)
	}

	// Array form with multiple assembled statuses.
	var assembled GuEffects
	if err := assembled.Scan([]byte(`{"active":{"apply_status":[{"id":"BLEED","value":10,"turns":3},{"id":"BIND","turns":2}]}}`)); err != nil {
		t.Fatalf("array form rejected: %v", err)
	}
	if len(assembled.Active.ApplyStatus) != 2 || assembled.Active.ApplyStatus[1].ID != "BIND" || assembled.Active.ApplyStatus[1].Turns != 2 {
		t.Fatalf("array form parsed wrong: %+v", assembled.Active.ApplyStatus)
	}

	// Legacy object form.
	var single GuEffects
	if err := single.Scan([]byte(`{"active":{"apply_status":{"id":"STUN","turns":1}}}`)); err != nil {
		t.Fatalf("object form rejected: %v", err)
	}
	if len(single.Active.ApplyStatus) != 1 || single.Active.ApplyStatus[0].ID != "STUN" {
		t.Fatalf("object form parsed wrong: %+v", single.Active.ApplyStatus)
	}
}

func TestResolveStatusApplicationsLegacyEquivalence(t *testing.T) {
	// Legacy single-status row: tick damage set → direct value, top-level duration.
	legacy := &ActiveEffect{
		ApplyStatus:         StatusApplications{{ID: "BLEED"}},
		StatusTickDamage:    12,
		StatusDurationTicks: 4,
	}
	ops := legacy.ResolveStatusApplications()
	if len(ops) != 1 || ops[0].ID != "BLEED" || ops[0].Value != 12 || ops[0].Turns != 4 {
		t.Fatalf("legacy resolution changed: %+v", ops)
	}

	// Bare duration <= 5 keeps the tier pipeline.
	tier := &ActiveEffect{
		ApplyStatus:         StatusApplications{{ID: "WEAK"}},
		StatusDurationTicks: 2,
	}
	ops = tier.ResolveStatusApplications()
	if len(ops) != 1 || ops[0].Value == 0 || ops[0].Turns == 0 {
		t.Fatalf("tier resolution changed: %+v", ops)
	}

	// Assembled entries: each entry resolves with its own value/turns.
	assembled := &ActiveEffect{
		ApplyStatus: StatusApplications{{ID: "BLEED", Value: 10, Turns: 3}, {ID: "BIND", Turns: 2}},
	}
	ops = assembled.ResolveStatusApplications()
	if len(ops) != 2 {
		t.Fatalf("assembled entries dropped: %+v", ops)
	}
	if ops[0].Value != 10 || ops[0].Turns != 3 || ops[1].ID != "BIND" || ops[1].Turns != 2 {
		t.Fatalf("assembled resolution wrong: %+v", ops)
	}

	// FORGET_REALITY intensity from target_damage_reduce_percent.
	reality := &ActiveEffect{
		ApplyStatus:           StatusApplications{{ID: "FORGET_REALITY", Turns: 2}},
		TargetDamageReducePct: 0.4,
	}
	ops = reality.ResolveStatusApplications()
	if len(ops) != 1 || ops[0].Value != 40 {
		t.Fatalf("FORGET_REALITY intensity wrong: %+v", ops)
	}
}

func TestNormalizeLegacyShorthandBecomesEntries(t *testing.T) {
	var effects GuEffects
	if err := effects.Scan([]byte(`{"active":{"bleed":3}}`)); err != nil {
		t.Fatalf("shorthand rejected: %v", err)
	}
	if len(effects.Active.ApplyStatus) != 1 || effects.Active.ApplyStatus[0].ID != "BLEED" {
		t.Fatalf("shorthand not normalized to entries: %+v", effects.Active.ApplyStatus)
	}
	ops := effects.Active.ResolveStatusApplications()
	if len(ops) != 1 || ops[0].Value != 3 || ops[0].Turns != 3 {
		t.Fatalf("normalized shorthand resolves wrong: %+v", ops)
	}
}
