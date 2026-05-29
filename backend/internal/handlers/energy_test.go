package handlers

import (
	"testing"

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
	if immortal.MaxMana != 120 {
		t.Fatalf("immortal max mana = %v, want 120", immortal.MaxMana)
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

func TestAdvanceCultivationMortalSubRank(t *testing.T) {
	player := &models.Player{Rank: 2, SubRank: 2, CultivationProgress: 999, MaxMana: 70, CurrentMana: 10}
	leveled, ascended, _ := advanceCultivation(player)
	if !leveled || ascended {
		t.Fatalf("leveled=%v ascended=%v, want leveled mortal without ascension", leveled, ascended)
	}
	if player.Rank != 2 || player.SubRank != 3 || player.CultivationProgress != 0 {
		t.Fatalf("player after advancement = rank %d sub %d progress %v", player.Rank, player.SubRank, player.CultivationProgress)
	}
}

func TestAdvanceCultivationRankFivePeakAscendsToImmortal(t *testing.T) {
	player := &models.Player{Rank: 5, SubRank: 4, CultivationProgress: 999, CurrentMana: 5, MaxMana: 80}
	leveled, ascended, _ := advanceCultivation(player)
	if !leveled || !ascended {
		t.Fatalf("leveled=%v ascended=%v, want immortal ascension", leveled, ascended)
	}
	if player.Rank != 6 || player.SubRank != 1 {
		t.Fatalf("rank/sub = %d/%d, want 6/1", player.Rank, player.SubRank)
	}
	if player.MaxMana != immortalApertureCapacity(6, 1) {
		t.Fatalf("max mana = %v, want immortal capacity", player.MaxMana)
	}
	if player.DaoMarks["heaven"] == 0 || player.DaoMarks["human"] == 0 {
		t.Fatal("ascension should grant heaven and human dao marks")
	}
}
