package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func setupGuRegistry(t *testing.T) {
	t.Helper()
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_A": {1: {ID: "GU_A", Name: "A", Rank: 1, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 10, DamageMult: 1.0}}},
			2: {ID: "GU_A", Name: "A", Rank: 2, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 12, DamageMult: 1.5}}}},
		"GU_B": {1: {ID: "GU_B", Name: "B", Rank: 1, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 20, DamageMult: 1.2}}}},
		"GU_C": {1: {ID: "GU_C", Name: "C", Rank: 1, Effects: models.GuEffects{Active: &models.ActiveEffect{DamageFlat: 30, DamageMult: 1.3}}}},
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

func TestComputeGlobalGuDamageMultiplierDedupesByGuID(t *testing.T) {
	setupGuRegistry(t)
	loadoutSpec := func(guID string) (models.GuDictionaryItem, bool) {
		switch guID {
		case "GU_A":
			return models.GetGu("GU_A", 2)
		case "GU_B":
			return models.GetGu("GU_B", 1)
		case "GU_C":
			return models.GetGu("GU_C", 1)
		default:
			return models.GuDictionaryItem{}, false
		}
	}

	mult := computeGlobalGuDamageMultiplier([]string{"GU_A", "GU_A", "GU_B", "GU_C", "GU_B"}, loadoutSpec)
	if mult != 1.5*1.2*1.3 {
		t.Fatalf("expected 1.5*1.2*1.3=2.34, got %.4f", mult)
	}
}

func TestStartCombat_AllyFiltering(t *testing.T) {
	playerID := "11111111-1111-1111-1111-111111111111"
	partyMemberID := "22222222-2222-2222-2222-222222222222"
	petID := "33333333-3333-3333-3333-333333333333"

	allies := []models.Combatant{
		{ID: playerID, Kind: "Player", Name: "Player1"},
		{ID: partyMemberID, Kind: "Ally", Name: "PartyMember"},
		{ID: petID, Kind: "Pet", Name: "Pet1"},
	}

	var notified []string
	for _, ally := range allies {
		if ally.Kind != "Ally" && ally.Kind != "Player" {
			continue
		}
		if ally.ID != playerID {
			notified = append(notified, ally.ID)
		}
	}

	if len(notified) != 1 || notified[0] != partyMemberID {
		t.Fatalf("expected only party member %s to be notified, got %+v", partyMemberID, notified)
	}
}
