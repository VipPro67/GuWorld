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
	if immortal.MaxMana != 999999999 {
		t.Fatalf("immortal max mana = %v, want 999999999", immortal.MaxMana)
	}
}

func TestCalculateEnergyCostSeparatesMortalAndImmortalGu(t *testing.T) {
	mortal := &models.Player{Rank: 3, SubRank: 1, MaxMana: 80}
	immortal := &models.Player{Rank: 6, SubRank: 1, MaxMana: 120}
	mortalGu := &models.GuDictionaryItem{Rank: 3, BaseManaCost: 10}
	immortalGu := &models.GuDictionaryItem{Rank: 6, BaseManaCost: 0}

	if got, want := calculateEnergyCost(mortal, mortalGu), 8.0; got != want {
		t.Fatalf("mortal gu cost = %v, want %v", got, want)
	}
	if got, want := calculateEnergyCost(immortal, mortalGu), 0.1; got != want {
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
