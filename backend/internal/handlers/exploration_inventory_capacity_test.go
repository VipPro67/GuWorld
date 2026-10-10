package handlers

import (
	"strconv"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestHasExplorationInventorySpaceUsesLoadedInventorySnapshot(t *testing.T) {
	player := &models.Player{
		CultivationState:   models.CultivationState{Rank: 6, SubRank: 1},
		GuInventory:        make(models.GuInventory, 2),
		MaterialsInventory: make(models.MaterialsInventory, 997),
	}
	for i := 0; i < 997; i++ {
		player.MaterialsInventory["item_"+strconv.Itoa(i)] = 1
	}

	if !HasExplorationInventorySpace(player, 0, SectTerritoryInfo{}) {
		t.Fatal("full loaded inventory should fit when no additional slot is requested")
	}
	if HasExplorationInventorySpace(player, 1, SectTerritoryInfo{}) {
		t.Fatal("full loaded inventory should reject one additional slot")
	}
}
