package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// New active keys must compile into SimAction fields (Gu-new.csv coverage).
func TestBuildSimActionForgetEgo(t *testing.T) {
	actor := &models.Combatant{ID: "p1", Kind: "Player", Rank: 7, BaseAttack: 100, BaseDefense: 50, MaxMana: 1000, CurrentMana: 1000}
	spec := &models.GuDictionaryItem{
		ID: "GU_FORGET_EGO", Name: "Bản Ngã", Rank: 7, Path: []string{"Wisdom Path"},
		Effects: models.GuEffects{Active: &models.ActiveEffect{
			DamageHPPercent: 0.4, ShieldFlatReducePercent: 0.4, DisableGuNextTurn: true,
			ApplyStatus: "FORGET_EGO", StatusDurationTicks: 1,
		}},
	}
	act, err := buildSimActionFromGu(actor, spec, "GU_SKILL")
	if err != nil {
		t.Fatalf("buildSimActionFromGu: %v", err)
	}
	if act.DamageHPPercent != 0.4 || act.ShieldStripPercent != 0.4 || act.DisableGuTurns != 1 {
		t.Fatalf("percent/seal fields not mapped: %+v", act)
	}
	if len(act.StatusOps) != 1 || act.StatusOps[0].ID != "FORGET_EGO" {
		t.Fatalf("FORGET_EGO status op missing: %+v", act.StatusOps)
	}
}

func TestBuildSimActionForgetReality(t *testing.T) {
	actor := &models.Combatant{ID: "p1", Kind: "Player", Rank: 7, BaseAttack: 100, BaseDefense: 50, MaxMana: 1000, CurrentMana: 1000}
	spec := &models.GuDictionaryItem{
		ID: "GU_FORGET_REALITY", Name: "Thực Tại", Rank: 7, Path: []string{"Wisdom Path"},
		Effects: models.GuEffects{Active: &models.ActiveEffect{
			ForgetRealityChance: 0.4, SelfEvasionBoostPercent: 0.5, TargetDamageReducePct: 0.4,
			ApplyStatus: "FORGET_REALITY", StatusDurationTicks: 2,
		}},
	}
	act, err := buildSimActionFromGu(actor, spec, "GU_SKILL")
	if err != nil {
		t.Fatalf("buildSimActionFromGu: %v", err)
	}
	if act.ProcChance != 0.4 || act.SelfEvasionPct != 0.5 || act.TargetWeakPct != 0.4 || act.BonusTurns != 2 {
		t.Fatalf("reality fields not mapped: %+v", act)
	}
	if len(act.StatusOps) != 1 || act.StatusOps[0].Value != 40 {
		t.Fatalf("FORGET_REALITY op must carry intensity 40: %+v", act.StatusOps)
	}
}

func TestBuildSimActionImmersionDamagePath(t *testing.T) {
	actor := &models.Combatant{ID: "p1", Kind: "Player", Rank: 9, BaseAttack: 100, BaseDefense: 50, MaxMana: 100000, CurrentMana: 100000}
	spec := &models.GuDictionaryItem{
		ID: "GU_IMMERSION", Name: "Trầm Luân", Rank: 9, Path: []string{"Dream Path"},
		Effects: models.GuEffects{Active: &models.ActiveEffect{
			DamageFlat: 250000000, ApplyStatus: "SLEEP", StatusDurationTicks: 5,
			StatBaseAttackFlat: 10000000, DamagePath: "Dream Path",
		}},
	}
	act, err := buildSimActionFromGu(actor, spec, "GU_SKILL")
	if err != nil {
		t.Fatalf("buildSimActionFromGu: %v", err)
	}
	if len(act.Path) != 1 || act.Path[0] != "Dream Path" {
		t.Fatalf("damage_path override missing: %+v", act.Path)
	}
	if act.Damage < 180000000 {
		t.Fatalf("damage missing stat_base_attack bonus: %v", act.Damage)
	}
	plain := *spec
	plain.Effects.Active = &models.ActiveEffect{DamageFlat: 250000000}
	base, err := buildSimActionFromGu(actor, &plain, "GU_SKILL")
	if err != nil {
		t.Fatalf("buildSimActionFromGu: %v", err)
	}
	if act.Damage-base.Damage != 10000000 {
		t.Fatalf("stat_base_attack bonus = %v, want 10000000", act.Damage-base.Damage)
	}
	found := false
	for _, op := range act.StatusOps {
		// durations <= 5 go through the legacy tier pipeline (tier 5 → 3 turns).
		if op.ID == "SLEEP" {
			found = true
		}
	}
	if !found {
		t.Fatalf("SLEEP op missing: %+v", act.StatusOps)
	}
}

// Passive-only counter Gu must survive the combat loadout filter.
func TestHasCombatEffectsForgetPassives(t *testing.T) {
	cases := map[string]models.GuEffects{
		"heaven": {Passive: &models.PassiveEffect{AntiHeavenPct: 0.65, ForgetHeavenProtection: true}},
		"dream":  {Passive: &models.PassiveEffect{AntiDreamPct: 0.5, ImmuneDreamStatus: true}},
	}
	for name, eff := range cases {
		spec := &models.GuDictionaryItem{ID: name, Effects: eff}
		if !hasCombatEffects(spec) {
			t.Fatalf("%s passive incorrectly filtered from combat", name)
		}
	}
}
