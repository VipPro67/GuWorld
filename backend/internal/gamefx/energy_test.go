package gamefx

import (
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
	// Mortal Gu: gap 0..4 -> 1,2,5,25,125
	cases := []struct {
		guRank, userRank int
		want             float64
	}{
		{5, 5, 1},  // same rank
		{2, 1, 2},  // gap 1 -> x2
		{3, 1, 5},  // gap 2 -> x5
		{4, 1, 25}, // gap 3 -> x25
		{5, 1, 125}, // rank1 sài rank5 => 125
		{5, 4, 2},  // gap 1 -> x2
		{3, 5, 1},  // gu lower than user -> x1
		{7, 6, 100},    // immortal gap 1 -> x100
		{8, 6, 10000},  // immortal gap 2 -> x100^2
		{9, 7, 10000},  // immortal gap 2 -> x100^2
		{6, 5, 100},    // mortal user using rank-6 immortal Gu -> x100
	}
	for _, c := range cases {
		if got := HigherRankManaMultiplier(c.guRank, c.userRank); got != c.want {
			t.Errorf("HigherRankManaMultiplier(%d, %d) = %v, want %v", c.guRank, c.userRank, got, c.want)
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
}
