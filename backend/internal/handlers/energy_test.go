package handlers

import (
	"math"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestEnergyRegenMortalUsesAptitude(t *testing.T) {
	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 3, SubRank: 1, AptitudeType: "Giáp"},
		ResourceState:    models.ResourceState{MaxMana: 80},
	}
	if got := energyRegenRatePerHour(player); got != 8.0 {
		t.Fatalf("mortal regen = %v, want 8.0", got)
	}
}

func TestEnergyRegenImmortalIgnoresAptitudeAndUsesRank(t *testing.T) {
	lowAptitude := &models.Player{
		CultivationState: models.CultivationState{Rank: 6, SubRank: 1, AptitudeType: "Đinh"},
		ResourceState:    models.ResourceState{MaxMana: 80},
	}
	highAptitude := &models.Player{
		CultivationState: models.CultivationState{Rank: 6, SubRank: 1, AptitudeType: "Giáp"},
		ResourceState:    models.ResourceState{MaxMana: 80},
	}
	if got, want := energyRegenRatePerHour(lowAptitude), 0.5; got != want {
		t.Fatalf("immortal regen = %v, want %v", got, want)
	}
	if energyRegenRatePerHour(lowAptitude) != energyRegenRatePerHour(highAptitude) {
		t.Fatal("immortal regen must not depend on mortal aptitude")
	}
}

func TestNormalizeEnergyPoolExpandsImmortalApertureOnly(t *testing.T) {
	mortal := &models.Player{
		CultivationState: models.CultivationState{Rank: 5, SubRank: 4},
		ResourceState:    models.ResourceState{CurrentMana: 40, MaxMana: 40},
	}
	if normalizeEnergyPool(mortal) {
		t.Fatal("mortal max mana should not be expanded by immortal aperture rules")
	}

	immortal := &models.Player{
		CultivationState: models.CultivationState{Rank: 6, SubRank: 1},
		ResourceState:    models.ResourceState{CurrentMana: 40, MaxMana: 40},
	}
	if !normalizeEnergyPool(immortal) {
		t.Fatal("immortal aperture should normalize to immortal capacity")
	}
	if immortal.MaxMana != 99999999999.0 {
		t.Fatalf("immortal max mana = %v, want 99999999999.0", immortal.MaxMana)
	}
}

func TestCalculateEnergyCostSeparatesMortalAndImmortalGu(t *testing.T) {
	mortal := &models.Player{
		CultivationState: models.CultivationState{Rank: 3, SubRank: 1},
		ResourceState:    models.ResourceState{MaxMana: 80},
	}
	immortal := &models.Player{
		CultivationState: models.CultivationState{Rank: 6, SubRank: 1},
		ResourceState:    models.ResourceState{MaxMana: 120},
	}
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
		CultivationState: models.CultivationState{
			Rank:                3,
			SubRank:             1,
			TimeSpeedMultiplier: 1.0,
		},
		BodyState: models.BodyState{
			HP:     50.0,
			BaseHP: 100.0,
		},
		ResourceState: models.ResourceState{
			LastManaUpdate: time.Now().Add(-2 * time.Hour), // 2 hours ago
		},
	}

	applyProjectedStatus(player, 10, time.Now())

	if math.Abs(player.HP-60.0) > 0.01 {
		t.Fatalf("HP after 2 hours = %v, want 60.0", player.HP)
	}
}

func TestManaRegenTickBonusIntegrated(t *testing.T) {
	// Setup mock Gu registry
	guID := "GU_HEAVENLY_ESSENCE_LOTUS_MOCK"
	if models.GuRegistry == nil {
		models.GuRegistry = make(map[string]map[int]models.GuDictionaryItem)
	}
	models.GuRegistry[guID] = map[int]models.GuDictionaryItem{
		3: {
			ID:   guID,
			Name: "Thiên Nguyên Bảo Liên Mock",
			Rank: 3,
			Effects: models.GuEffects{
				Other: []models.SpecialMechanic{
					{
						EventTrigger: "ON_TICK",
						ActionCode:   "MANA_REGEN_TICK",
						Params:       []byte(`{"value":10.0}`),
					},
				},
			},
		},
	}

	player := &models.Player{
		CultivationState: models.CultivationState{
			Rank:                3,
			SubRank:             1,
			AptitudeType:        "Giáp", // base rate 8.0
			TimeSpeedMultiplier: 1.0,
			Lifespan:            100.0,
		},
		ResourceState: models.ResourceState{
			LastManaUpdate: time.Now(),
			MaxMana:        100,
			CurrentMana:    0,
		},
		GuInventory: []models.GuInstance{
			{
				GuID:     guID,
				Rank:     3,
				Equipped: true,
				Status:   "ACTIVE",
			},
		},
	}

	// Calculate base + bonus (base Giáp: 8.0, Lotus multiplier: same rank rank=3 vs rank=3 -> tnMultiplier=2.0 -> bonus 10*2 = 20.0)
	// Expected regen rate: 8.0 + 20.0 = 28.0
	applyProjectedStatus(player, 10, time.Now())

	if got, want := player.ManaRegenRate, 28.0; got != want {
		t.Fatalf("expected ManaRegenRate to be %v, got %v", want, got)
	}
}
