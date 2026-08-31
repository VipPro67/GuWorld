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

	materialQty := map[string]int{"herb": 2}
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
