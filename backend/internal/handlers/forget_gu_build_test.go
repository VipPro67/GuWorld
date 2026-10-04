package handlers

import (
	"math"
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
			ApplyStatus: models.StatusApplications{{ID: "FORGET_EGO", Turns: 1}},
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
			ForgetRealityChance: 0.4, TargetDamageReducePct: 0.4,
			ApplyStatus: models.StatusApplications{
				{ID: "EVASION", Value: 50, Chance: 0.15, Turns: 1},
				{ID: "FORGET_REALITY", Turns: 2},
			},
		}},
	}
	act, err := buildSimActionFromGu(actor, spec, "GU_SKILL")
	if err != nil {
		t.Fatalf("buildSimActionFromGu: %v", err)
	}
	if act.ProcChance != 0.4 || act.TargetWeakPct != 0.4 || act.BonusTurns != 2 {
		t.Fatalf("reality fields not mapped: %+v", act)
	}
	if len(act.StatusOps) != 2 {
		t.Fatalf("expected EVASION + FORGET_REALITY ops, got %+v", act.StatusOps)
	}
	if act.StatusOps[0].ID != "EVASION" || act.StatusOps[0].Value != 50 {
		t.Fatalf("EVASION op must carry strength 50: %+v", act.StatusOps[0])
	}
	// Rank 7 scaling: chance 0.15 + 0.05*(7-1) = 0.45, turns 1 + (7-1)/3 = 3.
	if math.Abs(act.StatusOps[0].Chance-0.45) > 1e-9 || act.StatusOps[0].Turns != 3 {
		t.Fatalf("EVASION rank scaling wrong: %+v", act.StatusOps[0])
	}
	if act.StatusOps[1].Value != 40 {
		t.Fatalf("FORGET_REALITY op must carry intensity 40: %+v", act.StatusOps[1])
	}
}

func TestBuildSimActionImmersionDamagePath(t *testing.T) {
	actor := &models.Combatant{ID: "p1", Kind: "Player", Rank: 9, BaseAttack: 100, BaseDefense: 50, MaxMana: 100000, CurrentMana: 100000}
	spec := &models.GuDictionaryItem{
		ID: "GU_IMMERSION", Name: "Trầm Luân", Rank: 9, Path: []string{"Dream Path"},
		Effects: models.GuEffects{Active: &models.ActiveEffect{
			DamageFlat: 250000000, ApplyStatus: models.StatusApplications{{ID: "SLEEP", Turns: 5}},
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
	raws := map[string]string{
		"heaven": `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"PATH_RESISTANCE","sub_type":"ADD_FLAT","path":"heaven","value":0.65}]}]`,
		"dream":  `[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"PATH_RESISTANCE","sub_type":"ADD_FLAT","path":"dream","value":0.5},{"op":"UNIQUE","code":"IMMUNE_DREAM_STATUS"}]}]`,
	}
	for name, raw := range raws {
		var eff models.GuEffects
		if err := eff.Scan(raw); err != nil {
			t.Fatalf("%s scan: %v", name, err)
		}
		spec := &models.GuDictionaryItem{ID: name, Effects: eff}
		if !hasCombatEffects(spec) {
			t.Fatalf("%s passive incorrectly filtered from combat", name)
		}
	}
}
