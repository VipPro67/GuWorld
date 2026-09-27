package models

import (
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
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

func TestGuEffectsScanRejectsDeletedSupportSection(t *testing.T) {
	var effects GuEffects
	if err := effects.Scan([]byte(`{"support":{"heal_flat":10}}`)); err == nil {
		t.Fatal("deleted support section was accepted; healing lives in active.heal_flat")
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

func TestResolveStatusRankScaling(t *testing.T) {
	mk := func() *ActiveEffect {
		return &ActiveEffect{ApplyStatus: StatusApplications{
			{ID: "INVINCIBLE", Turns: 1, Chance: 0.35},
			{ID: "BLEED", Value: 10, Turns: 2},
		}}
	}
	// Rank 1: base values untouched.
	ops := mk().ResolveStatusApplicationsForRank(1)
	if ops[0].Chance != 0.35 || ops[0].Turns != 1 {
		t.Fatalf("rank1 = %+v, want chance 0.35 turns 1", ops[0])
	}
	// Rank 4: +0.15 chance, +1 turn. No-chance entry untouched.
	ops = mk().ResolveStatusApplicationsForRank(4)
	if ops[0].Chance != 0.50 || ops[0].Turns != 2 {
		t.Fatalf("rank4 = %+v, want chance 0.50 turns 2", ops[0])
	}
	if ops[1].Chance != 0 || ops[1].Turns != 2 {
		t.Fatalf("no-chance entry changed at rank4: %+v", ops[1])
	}
	// Rank 9: +0.40 chance, +2 turns.
	ops = mk().ResolveStatusApplicationsForRank(9)
	if ops[0].Chance != 0.75 || ops[0].Turns != 3 {
		t.Fatalf("rank9 = %+v, want chance 0.75 turns 3", ops[0])
	}
	// Legacy path carries chance raw without scaling.
	ops = mk().ResolveStatusApplications()
	if ops[0].Chance != 0.35 || ops[0].Turns != 1 {
		t.Fatalf("unranked = %+v, want chance 0.35 turns 1", ops[0])
	}
}

func TestGrowthProfileCompilesLegacyGrowth(t *testing.T) {
	spec := Gu{Effects: GuEffects{Growth: &GrowthEffect{MaxAttack: 10, MaxHP: 20, HoursToMax: 24}}}
	profile, ok := GrowthProfile(spec)
	if !ok {
		t.Fatal("expected growth profile")
	}
	if profile.MaxAttack != 10 || profile.MaxHP != 20 || profile.HoursToMax != 24 {
		t.Fatalf("unexpected profile: %+v", profile)
	}
}

func TestItemEffectCommandsAreSorted(t *testing.T) {
	item := ItemSpec{Effects: ItemEffects{"restore_mana": 2, "restore_hp": 1}}
	commands := item.EffectCommands()
	if len(commands) != 2 || commands[0].Code != "restore_hp" || commands[1].Code != "restore_mana" {
		t.Fatalf("unexpected commands: %+v", commands)
	}
}

func TestItemEffectsScan_Blocks(t *testing.T) {
	raw := `[{"id":"RESTORE_HUNGER","trigger":"ON_CONSUME","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"RESTORE_HUNGER","params":{"value":50}}]}]`
	var eff ItemEffects
	if err := eff.Scan(raw); err != nil {
		t.Fatalf("failed to scan item blocks: %v", err)
	}
	if eff["hunger"] != 50 {
		t.Fatalf("expected hunger 50, got: %v", eff)
	}
}

func TestCompiledProjectionSupplementsPartialAuthoredBlocks(t *testing.T) {
	spec := Gu{Effects: GuEffects{
		Blocks:  []gameplayeffects.Block{{Trigger: gameplayeffects.TriggerAlwaysOn, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}, Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpStatMod, Stat: "ACCURACY", SubType: "ADD_FLAT", Value: 4}}}},
		Passive: &PassiveEffect{Investigation: 7},
		Growth:  &GrowthEffect{MaxAttack: 10, HoursToMax: 24},
	}}
	if got := GuStatValue(spec, "INVESTIGATION"); got != 7 {
		t.Fatalf("investigation = %v", got)
	}
	profile, ok := GrowthProfile(spec)
	if !ok || profile.MaxAttack != 10 || profile.HoursToMax != 24 {
		t.Fatalf("growth = %+v, %v", profile, ok)
	}
}
