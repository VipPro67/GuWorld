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
