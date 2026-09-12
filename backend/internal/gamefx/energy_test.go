package gamefx

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestEnergyCostImmortal(t *testing.T) {
	player := &models.Player{}
	player.Rank = 7
	gu := &models.GuDictionaryItem{Rank: 7, BaseManaCost: 50}

	// Immortal, equal rank -> base cost
	if got := EnergyCost(player, gu); got != 50 {
		t.Errorf("immortal equal rank cost = %v, want 50", got)
	}

	// Immortal, lower-rank Gu -> free
	lowGu := &models.GuDictionaryItem{Rank: 5}
	if got := EnergyCost(player, lowGu); got != 0 {
		t.Errorf("immortal lower-rank cost = %v, want 0", got)
	}

	// Immortal, higher-rank Gu -> base cost x multiplier (immortal gap 1 -> x100)
	highGu := &models.GuDictionaryItem{Rank: 8, BaseManaCost: 50}
	if got := EnergyCost(player, highGu); got != 5000 {
		t.Errorf("immortal higher-rank cost = %v, want 5000 (50 x100)", got)
	}

	// Immortal, much higher player rank -> scaled down
	player9 := &models.Player{}
	player9.Rank = 9
	if got := EnergyCost(player9, gu); got >= 50 {
		t.Errorf("higher rank cost = %v, want < 50", got)
	}
}

func TestEnergyCostMortal(t *testing.T) {
	player := &models.Player{}
	player.Rank = 3
	player.MaxMana = 1000
	gu := &models.GuDictionaryItem{Rank: 3, BaseManaCost: 10}

	// mortal baseManaCost=10 -> 10% of 1000 = 100
	if got := EnergyCost(player, gu); got != 100 {
		t.Errorf("mortal cost = %v, want 100", got)
	}

	// higher player rank lowers cost
	player4 := &models.Player{}
	player4.Rank = 4
	player4.MaxMana = 1000
	if got := EnergyCost(player4, gu); got >= 100 {
		t.Errorf("mortal higher-rank cost = %v, want < 100", got)
	}
}

func TestHigherRankManaMultiplier(t *testing.T) {
	cases := []struct {
		guRank, userRank int
		want             float64
	}{
		{5, 5, 1},
		{2, 1, 2},
		{3, 1, 5},
		{4, 1, 20},
		{5, 1, 100},
		{5, 4, 100 / 20},
		{3, 5, 1},

		{6, 6, 1},
		{7, 6, 100},
		{8, 6, 10000},
		{9, 7, 10000},
		{9, 8, 100},

		{6, 5, math.Inf(1)},
		{7, 5, math.Inf(1)},
		{9, 1, math.Inf(1)},

		// Safety bounds test for zero / uninitialized ranks
		{1, 0, 1},
		{5, 0, 100},
		{7, 0, math.Inf(1)},
	}

	for _, c := range cases {
		got := HigherRankManaMultiplier(c.guRank, c.userRank)

		if math.IsInf(c.want, 1) {
			if !math.IsInf(got, 1) {
				t.Errorf(
					"HigherRankManaMultiplier(%d, %d) = %v, want +Inf",
					c.guRank,
					c.userRank,
					got,
				)
			}
			continue
		}

		if got != c.want {
			t.Errorf(
				"HigherRankManaMultiplier(%d, %d) = %v, want %v",
				c.guRank,
				c.userRank,
				got,
				c.want,
			)
		}
	}
}

func TestEnergyCostForCombatant(t *testing.T) {
	actor := &models.Combatant{Rank: 2, MaxMana: 500}
	gu := &models.GuDictionaryItem{Rank: 2, BaseManaCost: 10}
	if got := EnergyCostForCombatant(actor, gu); got != 50 {
		t.Errorf("combatant mortal cost = %v, want 50", got)
	}

	immortal := &models.Combatant{Rank: 7}
	igu := &models.GuDictionaryItem{Rank: 7, BaseManaCost: 50}
	if got := EnergyCostForCombatant(immortal, igu); got != 50 {
		t.Errorf("combatant immortal cost = %v, want 50", got)
	}

	// higher-rank immortal combatant -> multiplier (immortal gap 1 -> x100)
	highIgu := &models.GuDictionaryItem{Rank: 8, BaseManaCost: 50}
	if got := EnergyCostForCombatant(immortal, highIgu); got != 5000 {
		t.Errorf("combatant immortal higher-rank cost = %v, want 5000", got)
	}
}

func TestRefineManaTier(t *testing.T) {
	// No inventory -> 0
	if got := RefineManaTier(5, nil); got != 0 {
		t.Errorf("empty refine tier = %v, want 0", got)
	}

	// Test 50% discount for Immortal player with active refine tier Gu
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_REFINE": {
			7: {ID: "GU_REFINE", Name: "Tinh Luyện Cổ", Rank: 7, Type: "Passive", Effects: models.GuEffects{Passive: &models.PassiveEffect{RefineManaTier: 1}}},
		},
	}
	immortal := &models.Player{
		CultivationState: models.CultivationState{Rank: 7},
		GuInventory: models.GuInventory{
			{InstanceID: "refine_inst", GuID: "GU_REFINE", Rank: 7, Status: "ACTIVE", Equipped: true},
		},
	}
	igu := &models.GuDictionaryItem{Rank: 7, BaseManaCost: 100}
	if got := EnergyCost(immortal, igu); got != 50 {
		t.Errorf("immortal cost with refine tier = %v, want 50", got)
	}
}

func TestCanUseGuWithMana(t *testing.T) {
	specBaseManaZero := &models.GuDictionaryItem{Rank: 7, BaseManaCost: 0}
	specBaseManaTen := &models.GuDictionaryItem{Rank: 7, BaseManaCost: 10}
	specMortalGu := &models.GuDictionaryItem{Rank: 3, BaseManaCost: 10}

	// Mortal player (Rank 3) cannot use Immortal Gu (Rank 7) even if BaseManaCost == 0
	if CanUseGuWithMana(3, specBaseManaZero) {
		t.Errorf("expected CanUseGuWithMana(3, rank 7 baseManaCost=0) to be false")
	}
	if CanUseGuWithMana(3, specBaseManaTen) {
		t.Errorf("expected CanUseGuWithMana(3, rank 7 baseManaCost=10) to be false")
	}
	if !CanUseGuWithMana(3, specMortalGu) {
		t.Errorf("expected CanUseGuWithMana(3, rank 3) to be true")
	}
}

func TestCanEquipGu(t *testing.T) {
	tests := []struct {
		playerRank int
		guRank     int
		want       bool
	}{
		{1, 1, true},
		{1, 3, true},  // Mortal equipping higher rank Mortal Gu
		{1, 5, true},  // Mortal equipping Rank 5 Mortal Gu
		{1, 6, false}, // Mortal equipping Rank 6 Immortal Gu -> false
		{5, 6, false}, // Mortal Rank 5 equipping Rank 6 Immortal Gu -> false
		{6, 3, true},  // Immortal equipping Mortal Gu -> true
		{6, 6, true},  // Immortal equipping Rank 6 Immortal Gu -> true
		{6, 8, true},  // Immortal equipping Rank 8 Immortal Gu -> true
	}

	for _, tt := range tests {
		if got := CanEquipGu(tt.playerRank, tt.guRank); got != tt.want {
			t.Errorf("CanEquipGu(%d, %d) = %v, want %v", tt.playerRank, tt.guRank, got, tt.want)
		}
	}
}

func TestHigherRankHPCost(t *testing.T) {
	maxHP := 1000.0

	// Same or lower rank -> 0
	if got := HigherRankHPCost(2, 2, maxHP); got != 0 {
		t.Errorf("same rank HP cost = %v, want 0", got)
	}
	if got := HigherRankHPCost(1, 3, maxHP); got != 0 {
		t.Errorf("lower rank HP cost = %v, want 0", got)
	}

	// Higher rank (1 rank difference -> 15% of MaxHP)
	if got := HigherRankHPCost(3, 2, maxHP); got != 150.0 {
		t.Errorf("1 rank diff HP cost = %v, want 150.0", got)
	}

	// 2 ranks difference -> 30% of MaxHP
	if got := HigherRankHPCost(4, 2, maxHP); got != 300.0 {
		t.Errorf("2 ranks diff HP cost = %v, want 300.0", got)
	}
}
