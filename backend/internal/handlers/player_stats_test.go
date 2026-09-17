package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestGetPassiveStats_SpeedIsolation(t *testing.T) {
	setupTestGuRegistry(t)
	defer resetTestGuRegistry(t)

	player := &models.Player{
		CultivationState: models.CultivationState{
			Rank:    1,
			SubRank: 1,
		},
		GuInventory: models.GuInventory{
			{InstanceID: "inst1", GuID: "GU_PASSIVE_SPD", Rank: 1, Status: "ACTIVE", Equipped: true},
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

	// Test hasCombatEffects vs hasCombatActiveEffects
	activeSpec, _ := models.GetGu("GU_PASSIVE_SPD", 1)
	if !hasCombatEffects(&activeSpec) {
		t.Fatalf("GU_PASSIVE_SPD should be eligible for combat loadout (hasCombatEffects == true)")
	}
	if hasCombatActiveEffects(&activeSpec) {
		t.Fatalf("GU_PASSIVE_SPD should not generate an empty 0-damage active action (hasCombatActiveEffects == false)")
	}
}

// setupTestGuRegistry installs a test Gu registry with passive and active speed effects.
func setupTestGuRegistry(t *testing.T) {
	t.Helper()
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
}

// resetTestGuRegistry clears the test Gu registry.
func resetTestGuRegistry(t *testing.T) {
	t.Helper()
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{}
}

// TestCombatSpeedNoDoubleCount verifies that combat speed equals passive stats speed
// with no double-counting of speed_boost_flat effects.
func TestCombatSpeedNoDoubleCount(t *testing.T) {
	setupTestGuRegistry(t)
	defer resetTestGuRegistry(t)

	player := &models.Player{
		CultivationState: models.CultivationState{
			Rank:    1,
			SubRank: 1,
		},
		GuInventory: models.GuInventory{
			{InstanceID: "inst1", GuID: "GU_ACTIVE_SPD", Rank: 1, Status: "ACTIVE", Equipped: true},
		},
	}

	// Get passive stats speed (should NOT include speed_boost_flat)
	passiveStats := GetPassiveStats(player)
	passiveSpeed := passiveStats.Stats["speed"]

	// Build combatant with loadout including the speed-boost Gu
	loadout := &simulator.CombatLoadout{
		GuIds: []string{"GU_ACTIVE_SPD"},
	}
	simCombatant, err := buildSimCombatant(nil, player, false, true, loadout)
	if err != nil {
		t.Fatalf("failed to build sim combatant: %v", err)
	}

	// Combat speed should equal passive speed (no double-counting)
	// Before fix: simCombatant.Speed = passiveSpeed + 20 (double-counted)
	// After fix: simCombatant.Speed = passiveSpeed (correct)
	if simCombatant.Speed != passiveSpeed {
		t.Fatalf("expected combat speed %.2f to equal passive speed %.2f, got combat speed %.2f",
			passiveSpeed, passiveSpeed, simCombatant.Speed)
	}
}
