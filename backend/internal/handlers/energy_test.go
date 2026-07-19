package handlers

import (
	"math"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestEnergyRegenMortalUsesAptitude(t *testing.T) {
	player := &models.Player{Rank: 3, SubRank: 1, AptitudeType: "Giáp", MaxMana: 80}
	if got := energyRegenRatePerHour(player); got != 8.0 {
		t.Fatalf("mortal regen = %v, want 8.0", got)
	}
}

func TestEnergyRegenImmortalIgnoresAptitudeAndUsesRank(t *testing.T) {
	lowAptitude := &models.Player{Rank: 6, SubRank: 1, AptitudeType: "Đinh", MaxMana: 80}
	highAptitude := &models.Player{Rank: 6, SubRank: 1, AptitudeType: "Giáp", MaxMana: 80}
	if got, want := energyRegenRatePerHour(lowAptitude), 0.5; got != want {
		t.Fatalf("immortal regen = %v, want %v", got, want)
	}
	if energyRegenRatePerHour(lowAptitude) != energyRegenRatePerHour(highAptitude) {
		t.Fatal("immortal regen must not depend on mortal aptitude")
	}
}

func TestNormalizeEnergyPoolExpandsImmortalApertureOnly(t *testing.T) {
	mortal := &models.Player{Rank: 5, SubRank: 4, CurrentMana: 40, MaxMana: 40}
	if normalizeEnergyPool(mortal) {
		t.Fatal("mortal max mana should not be expanded by immortal aperture rules")
	}

	immortal := &models.Player{Rank: 6, SubRank: 1, CurrentMana: 40, MaxMana: 40}
	if !normalizeEnergyPool(immortal) {
		t.Fatal("immortal aperture should normalize to immortal capacity")
	}
	if immortal.MaxMana != 99999999999.0 {
		t.Fatalf("immortal max mana = %v, want 99999999999.0", immortal.MaxMana)
	}
}

func TestCalculateEnergyCostSeparatesMortalAndImmortalGu(t *testing.T) {
	mortal := &models.Player{Rank: 3, SubRank: 1, MaxMana: 80}
	immortal := &models.Player{Rank: 6, SubRank: 1, MaxMana: 120}
	mortalGu := &models.GuDictionaryItem{Rank: 3, BaseManaCost: 10}
	immortalGu := &models.GuDictionaryItem{Rank: 6, BaseManaCost: 0}

	// mortal (rank 3) using mortal gu (rank 3): base=8, rank mult 3→5x, cost=40
	if got, want := calculateEnergyCost(mortal, mortalGu), 40.0; got != want {
		t.Fatalf("mortal gu cost = %v, want %v", got, want)
	}
	// immortal using mortal gu: free
	if got, want := calculateEnergyCost(immortal, mortalGu), 0.0; got != want {
		t.Fatalf("mortal gu in immortal aperture cost = %v, want %v", got, want)
	}
	if got, want := calculateEnergyCost(immortal, immortalGu), 1.0; got != want {
		t.Fatalf("immortal gu cost = %v, want %v", got, want)
	}
}

func TestHPRegeneration(t *testing.T) {
	player := &models.Player{
		Rank:                3,
		SubRank:             1,
		HP:                  50.0,
		BaseHP:              100.0,
		LastManaUpdate:      time.Now().Add(-2 * time.Hour), // 2 hours ago
		TimeSpeedMultiplier: 1.0,
	}

	applyProjectedStatus(player, 10, time.Now())

	if math.Abs(player.HP-60.0) > 0.01 {
		t.Fatalf("HP after 2 hours = %v, want 60.0", player.HP)
	}
}
