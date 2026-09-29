package gamefx

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestStatusRegistryDefaultsLoaded(t *testing.T) {
	valid := ValidStatuses()
	for _, key := range []string{"severely_wounded", "poisoned", "exhausted", "blessed", "death_penalty"} {
		if !valid[key] {
			t.Fatalf("status %s missing from fallback registry", key)
		}
	}
	if got := StatusDisplayName("severely_wounded"); got != "Trọng Thương" {
		t.Fatalf("display name = %q", got)
	}
	if StatusDisplayName("unknown_key") != "unknown_key" {
		t.Fatal("unknown keys should fall back to the raw key")
	}
}

func TestGetResolvesStatusEffects(t *testing.T) {
	p := &models.Player{}
	p.Race = "human"
	p.Statuses = models.StatusList{{ID: "severely_wounded"}}

	fx := Get(p)
	if fx.StatMods["attack"] != -0.20 || fx.StatMods["defense"] != -0.20 || fx.StatMods["speed"] != -0.20 {
		t.Fatalf("severely_wounded stat mods missing: %+v", fx.StatMods)
	}
	if !fx.BlockedActions[ActionExplore] {
		t.Fatal("severely_wounded should block explore")
	}
}

func TestGetSkipsExpiredStatuses(t *testing.T) {
	past := time.Now().Add(-time.Minute)
	future := time.Now().Add(time.Hour)
	p := &models.Player{}
	p.Race = "human"
	p.Statuses = models.StatusList{
		{ID: "exhausted", ExpiresAt: &past},
		{ID: "blessed", ExpiresAt: &future},
	}

	fx := Get(p)
	if fx.BlockedActions[ActionCultivate] {
		t.Fatal("expired exhausted must not block cultivation")
	}
	if fx.LuckBonus != 5 {
		t.Fatalf("active blessed luck bonus missing: %v", fx.LuckBonus)
	}
}

func TestHasBlockedActionAndBlockingStatuses(t *testing.T) {
	p := &models.Player{}
	p.Statuses = models.StatusList{{ID: "exhausted"}}

	if !HasBlockedAction(p, ActionCultivate) {
		t.Fatal("exhausted should block cultivate")
	}
	if HasBlockedAction(nil, ActionCultivate) {
		t.Fatal("nil player should never be blocked")
	}
	defs := BlockingStatuses(p, ActionCultivate)
	if len(defs) != 1 || defs[0].Key != "exhausted" || defs[0].Name != "Kiệt Sức" {
		t.Fatalf("BlockingStatuses mismatch: %+v", defs)
	}
	if defs := BlockingStatuses(p, ActionExplore); len(defs) != 0 {
		t.Fatalf("explore should not be blocked: %+v", defs)
	}
}

func TestEffectsMergeBlockedActions(t *testing.T) {
	dst := Effects{}
	addEffects(&dst, Effects{StatMods: map[string]float64{"attack": 0.1}, BlockedActions: map[string]bool{ActionExplore: true}})
	addEffects(&dst, Effects{BlockedActions: map[string]bool{ActionCultivate: true}})

	if !dst.BlockedActions[ActionExplore] || !dst.BlockedActions[ActionCultivate] {
		t.Fatalf("blocked actions not merged: %+v", dst.BlockedActions)
	}
}

func TestRefineStatuses(t *testing.T) {
	valid := ValidStatuses()
	if !valid["refine_fatigue"] || !valid["refine_backfire"] {
		t.Fatal("refine_fatigue and refine_backfire missing from fallback registry")
	}

	pFatigue := &models.Player{}
	pFatigue.Statuses = models.StatusList{{ID: "refine_fatigue"}}
	fxFatigue := Get(pFatigue)
	if fxFatigue.RefineBonus != -0.50 {
		t.Fatalf("refine_fatigue RefineBonus = %v, want -0.50", fxFatigue.RefineBonus)
	}

	pBackfire := &models.Player{}
	pBackfire.Statuses = models.StatusList{{ID: "refine_backfire"}}
	fxBackfire := Get(pBackfire)
	if fxBackfire.RefineBonus != -0.95 {
		t.Fatalf("refine_backfire RefineBonus = %v, want -0.95", fxBackfire.RefineBonus)
	}
}

func TestRefineInsightStatuses(t *testing.T) {
	for rank := 1; rank <= 9; rank++ {
		key := RefineInsightKey(rank)
		if !ValidStatuses()[key] {
			t.Fatalf("insight status %s missing from fallback registry", key)
		}
	}
	// Insight statuses carry no global RefineBonus; bonus is computed
	// per-rank in the refine handler from the status Value.
	p := &models.Player{}
	p.Statuses = models.StatusList{{ID: "refine_insight_3", Value: 7}}
	if fx := Get(p); fx.RefineBonus != 0 {
		t.Fatalf("insight status must not contribute global RefineBonus, got %v", fx.RefineBonus)
	}
}


func TestEffectsForValueNegativeSeverity(t *testing.T) {
	def, ok := effectiveStatuses()["regional_wall_suppression"]
	if !ok {
		t.Fatal("regional_wall_suppression missing from registry")
	}

	// Negative Value (rank-scaled suppression intensity) overrides every
	// stat_mod percentage.
	eff := def.EffectsForValue(-0.35)
	for _, key := range []string{"attack", "defense", "speed"} {
		if eff.StatMods[key] != -0.35 {
			t.Fatalf("stat_mods[%s] = %v, want -0.35 (severity override)", key, eff.StatMods[key])
		}
	}

	// Zero / positive Value keeps the dictionary percentages.
	eff = def.EffectsForValue(0)
	if eff.StatMods["attack"] != -0.20 {
		t.Fatalf("stat_mods[attack] = %v, want dictionary value -0.20", eff.StatMods["attack"])
	}
}
