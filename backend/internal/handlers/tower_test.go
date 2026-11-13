package handlers

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestTowerEnemyGrowthIncreasesAndStaysFinite(t *testing.T) {
	if towerEnemyGrowth(1) != 1 || towerEnemyGrowth(2) <= 1 {
		t.Fatal("tower floor scaling should start at baseline and grow")
	}
	if got := towerEnemyGrowth(100000); got != 1e12 || math.IsInf(got, 0) || math.IsNaN(got) {
		t.Fatalf("extreme floor growth = %g, want finite cap 1e12", got)
	}
}

func TestTowerDoesNotOfferConsumableActions(t *testing.T) {
	actions := []simulator.SimAction{
		{ActionType: "BASIC_ATTACK"},
		{ActionType: "ITEM"},
		{ActionType: "GU_SKILL"},
		{ActionType: "CONSUMABLE"},
	}
	got := filterTowerActions(actions)
	if len(got) != 2 || got[0].ActionType != "BASIC_ATTACK" || got[1].ActionType != "GU_SKILL" {
		t.Fatalf("tower actions = %#v", got)
	}
}
