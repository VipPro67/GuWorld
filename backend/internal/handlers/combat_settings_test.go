package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func setupGuRegistry(t *testing.T) {
	t.Helper()
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_A": {1: {ID: "GU_A", Name: "A", Rank: 1, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 10}}},
			2: {ID: "GU_A", Name: "A", Rank: 2, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 12}}}},
		"GU_B": {1: {ID: "GU_B", Name: "B", Rank: 1, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 20}}}},
		"GU_C": {1: {ID: "GU_C", Name: "C", Rank: 1, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 30}}}},
	}
	t.Cleanup(func() { models.GuRegistry = map[string]map[int]models.GuDictionaryItem{} })
}

func TestFilterGuLoadoutEmptyLoadout(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "b1", GuID: "GU_B", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "c1", GuID: "GU_C", Rank: 1, Status: "SLEEPING", Equipped: true},
		{InstanceID: "a2", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: false},
	}
	got := filterGuLoadout(inv, models.AutoCombatConfig{})
	if len(got) != 2 {
		t.Fatalf("expected 2 Gus (equipped+ACTIVE combat), got %d: %+v", len(got), got)
	}
	if got[0].InstanceID != "a1" || got[1].InstanceID != "b1" {
		t.Fatalf("unexpected fallback set: %+v", got)
	}
}

func TestFilterGuLoadoutSubsetOrder(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "b1", GuID: "GU_B", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "c1", GuID: "GU_C", Rank: 1, Status: "ACTIVE", Equipped: true},
	}
	cs := models.AutoCombatConfig{GuPriorities: []models.GuTactic{
		{GuID: "GU_C", Priority: 1},
		{GuID: "GU_A", Priority: 2},
	}}
	got := filterGuLoadout(inv, cs)
	if len(got) != 2 {
		t.Fatalf("expected 2 Gus, got %d: %+v", len(got), got)
	}
	if got[0].GuID != "GU_C" || got[1].GuID != "GU_A" {
		t.Fatalf("expected loadout order [GU_C, GU_A], got %+v", got)
	}
}

func TestFilterGuLoadoutUnknownFallback(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "b1", GuID: "GU_B", Rank: 1, Status: "ACTIVE", Equipped: true},
	}
	cs := models.AutoCombatConfig{GuPriorities: []models.GuTactic{
		{GuID: "GU_ZZZ", Priority: 1},
	}}
	got := filterGuLoadout(inv, cs)
	if len(got) != 2 {
		t.Fatalf("expected fallback to all combat Gus (2), got %d: %+v", len(got), got)
	}
}

func TestFilterGuLoadoutPicksHighestRank(t *testing.T) {
	setupGuRegistry(t)
	inv := models.GuInventory{
		{InstanceID: "a_low", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true},
		{InstanceID: "a_high", GuID: "GU_A", Rank: 2, Status: "ACTIVE", Equipped: true},
	}
	cs := models.AutoCombatConfig{GuPriorities: []models.GuTactic{
		{GuID: "GU_A", Priority: 1},
	}}
	got := filterGuLoadout(inv, cs)
	if len(got) != 1 {
		t.Fatalf("expected 1 Gu, got %d: %+v", len(got), got)
	}
	if got[0].InstanceID != "a_high" {
		t.Fatalf("expected highest-rank instance a_high, got %+v", got)
	}
}
