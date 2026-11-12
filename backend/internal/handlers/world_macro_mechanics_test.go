package handlers

import (
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func parseEffects(t *testing.T, raw string) models.GuEffects {
	t.Helper()
	var fx models.GuEffects
	if err := fx.Scan(raw); err != nil {
		t.Fatalf("scan effects failed: %v", err)
	}
	return fx
}

func TestFoodNutritionBonusForPlayer(t *testing.T) {
	previous := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = previous })

	fishFX := parseEffects(t, `{"schema_version":2,"blocks":[{"id":"PASSIVE_STORAGE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"STORAGE","value":20,"sub_type":"ADD_FLAT"}]}]}`)
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_ANCIENT_FISH_1": {
			1: {
				ID:      "GU_ANCIENT_FISH_1",
				Rank:    1,
				Path:    []string{"Food Path"},
				Effects: fishFX,
			},
		},
	}

	base := computeFoodNutrition("food_essence_rank1", 1)
	if base <= 0 {
		t.Fatalf("expected base nutrition > 0, got %d", base)
	}

	p := &models.Player{}
	p.ID = uuid.New()
	p.DaoMarks = models.DaoMarksMap{"food": 5000} // +50% bonus from 5000 dao marks (min(0.5, 5000/10000))

	withDaoMarks := computeFoodNutritionForPlayer(p, "food_essence_rank1", 1)
	expectedBoosted := int(math.Round(float64(base) * 1.5))
	if withDaoMarks != expectedBoosted {
		t.Errorf("expected boosted nutrition %d, got %d", expectedBoosted, withDaoMarks)
	}

	// Player with equipped food Gu
	pEquipped := &models.Player{}
	pEquipped.ID = uuid.New()
	pEquipped.GuInventory = []models.GuInstance{
		{
			InstanceID: "inst_ancient_fish",
			GuID:       "GU_ANCIENT_FISH_1",
			Rank:       1,
			Equipped:   true,
			Status:     "ACTIVE",
		},
	}
	withEquipped := computeFoodNutritionForPlayer(pEquipped, "food_essence_rank1", 1)
	expectedEquipped := int(math.Round(float64(base) * 1.15))
	if withEquipped != expectedEquipped {
		t.Errorf("expected equipped food Gu to boost nutrition %d, got %d", expectedEquipped, withEquipped)
	}
}

func TestBeastWhipCaptureBonus(t *testing.T) {
	previous := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = previous })

	whipFX := parseEffects(t, `{"schema_version":2,"blocks":[{"id":"CAPTURE_BONUS","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"CAPTURE_BONUS","params":{"value":0.15}}]}]}`)
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_BEAST_WHIP_1": {
			1: {
				ID:      "GU_BEAST_WHIP_1",
				Rank:    1,
				Path:    []string{"Enslavement Path"},
				Effects: whipFX,
			},
		},
	}

	spec, ok := models.GetGu("GU_BEAST_WHIP_1", 1)
	if !ok {
		t.Fatalf("GU_BEAST_WHIP_1 rank 1 not found in registry")
	}

	bonus := models.GuSpecCapabilityFloat(spec, "CAPTURE_BONUS", "value")
	if bonus != 0.15 {
		t.Fatalf("GU_BEAST_WHIP_1 expected CAPTURE_BONUS 0.15, got %f", bonus)
	}
}

func TestWisdomGuCapabilities(t *testing.T) {
	previous := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = previous })

	wisdomFX := parseEffects(t, `{"schema_version":2,"blocks":[{"id":"DEDUCTION_INSTANT_FINISH","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"DEDUCTION_INSTANT_FINISH","params":{"base_chance":0.25,"per_rank_step":0.05,"max_chance":0.95}}]},{"id":"LOOT_PREVIEW","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"LOOT_PREVIEW","params":{}}]}]}`)
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_WISDOM": {
			9: {
				ID:      "GU_WISDOM",
				Rank:    9,
				Path:    []string{"Wisdom Path"},
				Effects: wisdomFX,
			},
		},
	}

	spec, ok := models.GetGu("GU_WISDOM", 9)
	if !ok {
		t.Fatalf("GU_WISDOM rank 9 not found in registry")
	}

	// Instant deduction mechanic
	mech, ok := models.GuNativeMechanic(spec, "DEDUCTION_INSTANT_FINISH")
	if !ok {
		t.Fatalf("GU_WISDOM expected DEDUCTION_INSTANT_FINISH mechanic, but not found")
	}
	baseChance, _ := mech.Values["base_chance"].(float64)
	if baseChance != 0.25 {
		t.Errorf("expected base_chance 0.25, got %f", baseChance)
	}

	// Loot preview capability
	_, hasPreview := models.GuSpecCapability(spec, "LOOT_PREVIEW")
	if !hasPreview {
		t.Fatalf("GU_WISDOM expected LOOT_PREVIEW capability, but not found")
	}
}

func TestAncientFishStorageBonus(t *testing.T) {
	previous := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = previous })

	fishFX := parseEffects(t, `{"schema_version":2,"blocks":[{"id":"PASSIVE_STORAGE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"STORAGE","value":20,"sub_type":"ADD_FLAT"}]}]}`)
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_ANCIENT_FISH_1": {
			1: {
				ID:      "GU_ANCIENT_FISH_1",
				Rank:    1,
				Path:    []string{"Food Path"},
				Effects: fishFX,
			},
		},
	}

	spec, ok := models.GetGu("GU_ANCIENT_FISH_1", 1)
	if !ok {
		t.Fatalf("GU_ANCIENT_FISH_1 rank 1 not found in registry")
	}

	storage := models.GuStatValue(spec, "STORAGE")
	if storage != 20 {
		t.Fatalf("GU_ANCIENT_FISH_1 expected STORAGE 20, got %f", storage)
	}
}

func TestDogShitBookLuckBonus(t *testing.T) {
	previous := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = previous })

	dogFX := parseEffects(t, `{"schema_version":2,"blocks":[{"id":"PASSIVE_STATS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","value":1,"sub_type":"ADD_FLAT"},{"op":"STAT_MOD","stat":"LUCK","value":10,"sub_type":"ADD_FLAT"}]}]}`)
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_DOG_SHIT_BOOK_1": {
			1: {
				ID:      "GU_DOG_SHIT_BOOK_1",
				Rank:    1,
				Path:    []string{"Luck Path"},
				Effects: dogFX,
			},
		},
	}

	spec, ok := models.GetGu("GU_DOG_SHIT_BOOK_1", 1)
	if !ok {
		t.Fatalf("GU_DOG_SHIT_BOOK_1 rank 1 not found in registry")
	}

	luck := models.GuStatValue(spec, "LUCK")
	if luck != 10 {
		t.Fatalf("GU_DOG_SHIT_BOOK_1 expected LUCK 10, got %f", luck)
	}

	var p models.Player
	p.ID = uuid.New()
	p.GuInventory = []models.GuInstance{
		{
			InstanceID: "inst_dog_shit",
			GuID:       "GU_DOG_SHIT_BOOK_1",
			Rank:       1,
			Equipped:   true,
			Status:     "ACTIVE",
		},
	}

	fx := gamefx.Get(&p)
	if fx.LuckBonus != 10 {
		t.Fatalf("expected fx.LuckBonus 10 from equipped GU_DOG_SHIT_BOOK_1, got %f", fx.LuckBonus)
	}
}
