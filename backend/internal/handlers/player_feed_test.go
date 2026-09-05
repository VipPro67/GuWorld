package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestChooseFeedCandidate(t *testing.T) {
	player := models.Player{}
	player.PrimevalStones = 0
	player.ImmortalStones = 0
	player.GuInventory = models.GuInventory{
		{InstanceID: "gu_food_1", GuID: "GU_WOLF", Rank: 2, Status: "ACTIVE"},
	}

	materialQty := map[string]int64{"herb": 2}
	foodID, consumedIndex := chooseFeedCandidate(player, models.GuInstance{InstanceID: "target", GuID: "GU_RAVEN", Rank: 3, Status: "ACTIVE"}, []string{"herb", "GU_WOLF"}, "target", materialQty)
	if foodID != "herb" || consumedIndex != -1 {
		t.Fatalf("expected herb material choice, got foodID=%q index=%d", foodID, consumedIndex)
	}

	foodID, consumedIndex = chooseFeedCandidate(player, models.GuInstance{InstanceID: "target", GuID: "GU_RAVEN", Rank: 3, Status: "ACTIVE"}, []string{"GU_WOLF"}, "target", materialQty)
	if foodID != "GU_WOLF" || consumedIndex != 0 {
		t.Fatalf("expected gu food choice, got foodID=%q index=%d", foodID, consumedIndex)
	}

	player.PrimevalStones = 8
	foodID, consumedIndex = chooseFeedCandidate(player, models.GuInstance{InstanceID: "target", GuID: "GU_RAVEN", Rank: 3, Status: "ACTIVE"}, []string{"primeval_stone"}, "target", nil)
	if foodID != "primeval_stone" || consumedIndex != -1 {
		t.Fatalf("expected primeval stone choice, got foodID=%q index=%d", foodID, consumedIndex)
	}
}

func TestComputeFoodNutrition(t *testing.T) {
	// Seed dummy registry items for testing if needed
	models.ItemRegistry["food_r1"] = models.ItemSpec{ID: "food_r1", Rank: 1}
	models.ItemRegistry["food_r2"] = models.ItemSpec{ID: "food_r2", Rank: 2}
	models.ItemRegistry["food_r3"] = models.ItemSpec{ID: "food_r3", Rank: 3}
	models.ItemRegistry["food_r4"] = models.ItemSpec{ID: "food_r4", Rank: 4}
	models.ItemRegistry["food_r5"] = models.ItemSpec{ID: "food_r5", Rank: 5}

	tests := []struct {
		foodID   string
		guRank   int
		expected int
	}{
		// Same rank -> 100
		{"food_r1", 1, 100},
		{"food_r2", 2, 100},
		{"food_r3", 3, 100},
		{"food_r4", 4, 100},
		{"food_r5", 5, 100},

		// Higher rank -> 100
		{"food_r5", 1, 100},
		{"food_r3", 2, 100},

		// Lower rank scaling by new P table: [1, 2, 5, 20, 100, 10000, 1000000, 100000000, 10000000000]
		{"food_r1", 2, 50}, // 1 / 2 * 100 = 50
		{"food_r1", 3, 20}, // 1 / 5 * 100 = 20
		{"food_r2", 3, 40}, // 2 / 5 * 100 = 40
		{"food_r1", 4, 5},  // 1 / 20 * 100 = 5
		{"food_r2", 4, 10}, // 2 / 20 * 100 = 10
		{"food_r3", 4, 25}, // 5 / 20 * 100 = 25
		{"food_r1", 5, 1},  // 1 / 100 * 100 = 1
		{"food_r2", 5, 2},  // 2 / 100 * 100 = 2
		{"food_r3", 5, 5},  // 5 / 100 * 100 = 5
		{"food_r4", 5, 20}, // 20 / 100 * 100 = 20
		{"food_r5", 6, 1},  // 100 / 10000 * 100 = 1
		{"food_r1", 6, 0},  // 1 / 10000 * 100 = 0.01 -> 0 (no min floor limit)

		// Edge cases
		{"unknown_food", 3, 100}, // Unknown food / fallback
		{"food_r1", 9, 0},        // R1 food fed to R9 Gu: 1 / 10^10 * 100 = 0 (no min floor limit)
		{"food_r1", 0, 100},      // Out-of-bounds guRank < 1 -> clamped to 1 -> foodRank >= guRank -> 100
	}

	for _, tt := range tests {
		actual := computeFoodNutrition(tt.foodID, tt.guRank)
		if actual != tt.expected {
			t.Errorf("computeFoodNutrition(%q, guRank=%d) = %d; want %d", tt.foodID, tt.guRank, actual, tt.expected)
		}
	}
}
