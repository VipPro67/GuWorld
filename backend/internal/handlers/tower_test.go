package handlers

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestTowerEnemyGrowthIncreasesAndStaysFinite(t *testing.T) {
	if towerEnemyGrowth(1) != 1 || towerEnemyGrowth(2) <= towerEnemyGrowth(1) {
		t.Fatal("tower floor scaling should start at Rank 1 baseline and grow")
	}
	for _, milestone := range []struct{ floor, rank int }{{10, 2}, {20, 3}, {30, 4}, {40, 5}, {90, 6}, {140, 7}, {190, 8}, {240, 9}} {
		if got, want := towerEnemyGrowth(milestone.floor), models.RankScale(milestone.rank); math.Abs(got-want) > want*1e-9 {
			t.Errorf("floor %d growth = %g, want rank %d scale %g", milestone.floor, got, milestone.rank, want)
		}
	}
	if got := towerEnemyGrowth(100000); got != 1e22 || math.IsInf(got, 0) || math.IsNaN(got) {
		t.Fatalf("extreme floor growth = %g, want finite cap 1e22", got)
	}
}

func TestTowerUsesSharedTierScheduleAndDeterministicMonsterList(t *testing.T) {
	monsters := []models.MonsterTemplate{
		{ID: "boss_b", Name: "Boss B", Type: "BOSS"},
		{ID: "elite_b", Name: "Elite B", Type: "Elite"},
		{ID: "normal_b", Name: "Normal B", Type: "Normal"},
		{ID: "boss_a", Name: "Boss A", Type: "BOSS"},
		{ID: "elite_a", Name: "Elite A", Type: "Elite"},
		{ID: "normal_a", Name: "Normal A", Type: "NORMAL"},
	}
	for _, test := range []struct {
		floor int
		want  string
	}{{1, "normal_a"}, {5, "elite_a"}, {10, "boss_a"}, {15, "elite_b"}, {20, "boss_b"}, {21, "normal_a"}} {
		got, ok := towerMonsterForFloor(test.floor, monsters)
		if !ok || got.ID != test.want {
			t.Errorf("floor %d monster = %q, %v; want %q", test.floor, got.ID, ok, test.want)
		}
	}
	reversed := []models.MonsterTemplate{monsters[5], monsters[4], monsters[3], monsters[2], monsters[1], monsters[0]}
	got, ok := towerMonsterForFloor(1, reversed)
	if !ok || got.ID != "normal_a" {
		t.Fatalf("floor 1 changed with registry iteration order: %q, %v", got.ID, ok)
	}
}

func TestTowerEnemyRankComesFromFloor(t *testing.T) {
	for _, test := range []struct{ floor, rank int }{{1, 1}, {9, 1}, {10, 2}, {40, 5}, {90, 6}, {240, 9}, {500, 9}} {
		if got := towerEnemyRank(test.floor); got != test.rank {
			t.Errorf("towerEnemyRank(%d) = %d, want %d", test.floor, got, test.rank)
		}
	}
	if got := towerEnemySubRank(100000); got != 10 {
		t.Fatalf("tower subrank = %d, want cap 10", got)
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
