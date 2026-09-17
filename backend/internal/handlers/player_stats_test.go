package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGetPassiveStats_SpeedIsolation(t *testing.T) {
	// Register two mock Gus:
	// 1) GU_PASSIVE_SPD: only passive stat_base_speed = 5
	// 2) GU_ACTIVE_SPD: only active speed_boost_flat = 20
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_PASSIVE_SPD": {
			1: {
				ID:   "GU_PASSIVE_SPD",
				Name: "Bị Động Tốc Cổ",
				Rank: 1,
				Path: []string{"Cloud Path"},
				Effects: models.GuEffects{
					Passive: &models.PassiveEffect{
						StatBaseSpeed: 5,
					},
				},
			},
		},
		"GU_ACTIVE_SPD": {
			1: {
				ID:   "GU_ACTIVE_SPD",
				Name: "Chủ Động Tốc Cổ",
				Rank: 1,
				Path: []string{"Lightning Path"},
				Effects: models.GuEffects{
					Active: &models.ActiveEffect{
						SpeedBoostFlat: 20,
					},
				},
			},
		},
	}

	player := &models.Player{
		CultivationState: models.CultivationState{
			Rank:    1,
			SubRank: 1,
		},
		GuInventory: models.GuInventory{
			{InstanceID: "inst1", GuID: "GU_PASSIVE_SPD", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "inst2", GuID: "GU_ACTIVE_SPD", Rank: 1, Status: "ACTIVE", Equipped: true},
		},
	}

	result := GetPassiveStats(player)

	// Base speed for Rank 1, SubRank 1 is 10.0 + 0 + 0 = 10.0
	// GU_PASSIVE_SPD adds 5.0 -> total passive speed = 15.0
	// GU_ACTIVE_SPD should NOT be included in passive stats (speed_boost_flat is combat-only).
	expectedPassiveSpeed := 15.0
	if actual := result.Stats["speed"]; actual != expectedPassiveSpeed {
		t.Fatalf("expected passive speed %.1f, got %.1f", expectedPassiveSpeed, actual)
	}

	// Verify details contains EXACTLY ONE entry for Key == "speed"
	speedDetailCount := 0
	for _, d := range result.Details {
		if d.Key == "speed" {
			speedDetailCount++
		}
	}
	if speedDetailCount != 1 {
		t.Fatalf("expected exactly 1 speed detail in Details, got %d", speedDetailCount)
	}

	// Test computeGlobalSpeedBoost in combat loadout
	loadoutSpec := func(guID string) (models.GuDictionaryItem, bool) {
		return models.GetGu(guID, 1)
	}
	combatSpeedBoost := computeGlobalSpeedBoost([]string{"GU_ACTIVE_SPD", "GU_PASSIVE_SPD"}, loadoutSpec)
	if combatSpeedBoost != 20.0 {
		t.Fatalf("expected combat speed boost of 20.0 from active loadout, got %.1f", combatSpeedBoost)
	}

	// Test hasCombatEffects vs hasCombatActiveEffects
	activeSpec, _ := models.GetGu("GU_ACTIVE_SPD", 1)
	if !hasCombatEffects(&activeSpec) {
		t.Fatalf("GU_ACTIVE_SPD should be eligible for combat loadout (hasCombatEffects == true)")
	}
	if hasCombatActiveEffects(&activeSpec) {
		t.Fatalf("GU_ACTIVE_SPD should not generate an empty 0-damage active action (hasCombatActiveEffects == false)")
	}
}
