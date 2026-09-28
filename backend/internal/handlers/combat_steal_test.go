package handlers

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestStealChanceFormulas(t *testing.T) {
	// Permanent Steal (Vô Tướng Thủ, Rank 9)
	// Base 25% + 10% * (9 - targetRank)
	testsPerm := []struct {
		targetRank int
		isVital    bool
		expected   float64
	}{
		{9, false, 0.25},
		{8, false, 0.35},
		{7, false, 0.45},
		{6, false, 0.55},
		{5, false, 0.65},
		{3, false, 0.85},
		{1, false, 1.00}, // 0.25 + 0.80 = 1.05 -> clamped to 1.0
		{5, true, 0.325}, // 0.65 * 0.5 = 0.325
		{9, true, 0.125}, // 0.25 * 0.5 = 0.125
		{1, true, 0.500}, // 1.0 * 0.5 = 0.5
	}

	for _, tt := range testsPerm {
		got := permanentStealChance(9, tt.targetRank, tt.isVital)
		if math.Abs(got-tt.expected) > 0.001 {
			t.Errorf("permanentStealChance(9, %d, %v) = %f; want %f", tt.targetRank, tt.isVital, got, tt.expected)
		}
	}

	// Temporary Steal (Ám Thâu Cổ, Rank 8)
	// Base 40% + 5% * (8 - targetRank)
	testsTemp := []struct {
		targetRank int
		isVital    bool
		expected   float64
	}{
		{8, false, 0.40},
		{7, false, 0.45},
		{5, false, 0.55},
		{3, false, 0.65},
		{1, false, 0.75},
		{8, true, 0.20},  // 0.40 * 0.5 = 0.20
		{5, true, 0.275}, // 0.55 * 0.5 = 0.275
	}

	for _, tt := range testsTemp {
		got := tempStealChance(8, tt.targetRank, tt.isVital)
		if math.Abs(got-tt.expected) > 0.001 {
			t.Errorf("tempStealChance(8, %d, %v) = %f; want %f", tt.targetRank, tt.isVital, got, tt.expected)
		}
	}
}

func TestIsOpenWorldPvP(t *testing.T) {
	// Valid open world PvP
	valid := &models.CombatState{
		EnemyType:      "PVP",
		CombatMode:     "real",
		MineSlotID:     0,
		TargetPlayerID: "11111111-1111-1111-1111-111111111111",
	}
	if !isOpenWorldPvP(valid) {
		t.Errorf("expected open-world PvP to be true")
	}

	// Virtual Arena combat should be excluded
	arena := &models.CombatState{
		EnemyType:      "PVP",
		CombatMode:     "virtual",
		MineSlotID:     0,
		TargetPlayerID: "11111111-1111-1111-1111-111111111111",
	}
	if isOpenWorldPvP(arena) {
		t.Errorf("expected arena (virtual mode) to be excluded from steal")
	}

	// Mine PvP should be excluded
	mine := &models.CombatState{
		EnemyType:      "PVP",
		CombatMode:     "real",
		MineSlotID:     3,
		TargetPlayerID: "11111111-1111-1111-1111-111111111111",
	}
	if isOpenWorldPvP(mine) {
		t.Errorf("expected mine PvP to be excluded from steal")
	}

	// PvE monster should be excluded
	pve := &models.CombatState{
		EnemyType:      "MINE_PVE",
		CombatMode:     "real",
		MineSlotID:     0,
		TargetPlayerID: "",
	}
	if isOpenWorldPvP(pve) {
		t.Errorf("expected PvE monster to be excluded from steal")
	}

	// Empty combat state
	if isOpenWorldPvP(nil) {
		t.Errorf("expected nil combat state to return false")
	}
}

func TestFindStealGus(t *testing.T) {
	// Seed dummy registry specs if needed
	inv := models.GuInventory{
		{
			InstanceID: "inst-1",
			GuID:       "GU_FORMLESS_THIEF_9",
			Rank:       9,
			Equipped:   true,
			Status:     "ACTIVE",
		},
		{
			InstanceID: "inst-2",
			GuID:       "GU_SHADOW_STEAL_8",
			Rank:       8,
			Equipped:   true,
			Status:     "ACTIVE",
		},
		{
			InstanceID: "inst-3",
			GuID:       "GU_FORMLESS_THIEF_9",
			Rank:       9,
			Equipped:   false, // unequipped
			Status:     "ACTIVE",
		},
	}

	// Ensure registry has them for lookup
	models.GuRegistry["GU_FORMLESS_THIEF_9"] = map[int]models.GuDictionaryItem{
		9: {ID: "GU_FORMLESS_THIEF_9", Rank: 9, Name: "Vô Tướng Thủ"},
	}
	models.GuRegistry["GU_SHADOW_STEAL_8"] = map[int]models.GuDictionaryItem{
		8: {ID: "GU_SHADOW_STEAL_8", Rank: 8, Name: "Ám Thâu Cổ"},
	}

	permGu, permSpec, tempGu, tempSpec := findStealGus(inv)
	if permGu == nil || permGu.InstanceID != "inst-1" {
		t.Errorf("expected permGu inst-1, got %v", permGu)
	}
	if permSpec == nil || permSpec.Name != "Vô Tướng Thủ" {
		t.Errorf("expected permSpec Vô Tướng Thủ, got %v", permSpec)
	}

	if tempGu == nil || tempGu.InstanceID != "inst-2" {
		t.Errorf("expected tempGu inst-2, got %v", tempGu)
	}
	if tempSpec == nil || tempSpec.Name != "Ám Thâu Cổ" {
		t.Errorf("expected tempSpec Ám Thâu Cổ, got %v", tempSpec)
	}
}
