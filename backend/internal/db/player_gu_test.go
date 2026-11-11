package db

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestPlayerGuColumnsIncludeRefinementAndDurability(t *testing.T) {
	if !strings.Contains(playerGuColumns, "refinement_progress") {
		t.Fatalf("expected playerGuColumns to include refinement_progress: %s", playerGuColumns)
	}
	if !strings.Contains(playerGuColumns, "durability") {
		t.Fatalf("expected playerGuColumns to include durability: %s", playerGuColumns)
	}
}

func TestGuInventoryAggregateIncludesRefinementAndDurability(t *testing.T) {
	agg := GuInventoryAggregate("p.id")
	if !strings.Contains(agg, "'refinement_progress', pg.refinement_progress") {
		t.Fatalf("expected GuInventoryAggregate to include refinement_progress: %s", agg)
	}
	if !strings.Contains(agg, "'durability', pg.durability") {
		t.Fatalf("expected GuInventoryAggregate to include durability: %s", agg)
	}

	aggEquipped := GuInventoryAggregateEquipped("p.id")
	if !strings.Contains(aggEquipped, "'refinement_progress', pg.refinement_progress") {
		t.Fatalf("expected GuInventoryAggregateEquipped to include refinement_progress: %s", aggEquipped)
	}
	if !strings.Contains(aggEquipped, "'durability', pg.durability") {
		t.Fatalf("expected GuInventoryAggregateEquipped to include durability: %s", aggEquipped)
	}
}

func TestSameGuInstanceWithRefinementAndDurability(t *testing.T) {
	a := models.GuInstance{
		InstanceID:         "gu-1",
		GuID:               "spring_autumn_cicada",
		Rank:               6,
		Hunger:             100,
		RefinementProgress: 100.0,
		Durability:         100.0,
		Status:             "ACTIVE",
		Equipped:           true,
		IsVital:            true,
	}
	b := a
	if !sameGuInstance(a, b) {
		t.Fatalf("identical instances must match")
	}

	b.RefinementProgress = 90.0
	if sameGuInstance(a, b) {
		t.Fatalf("instances with different refinement progress must not match")
	}

	b = a
	b.Durability = 85.0
	if sameGuInstance(a, b) {
		t.Fatalf("instances with different durability must not match")
	}
}

func TestMortalApertureCapacity(t *testing.T) {
	tests := []struct {
		rank    int
		subRank int
		want    int
	}{
		{1, 1, 15},
		{2, 1, 25},
		{3, 1, 40},
		{4, 1, 60},
		{5, 1, 80},
		{5, 4, 100},
		{6, 1, -1},
		{7, 1, -1},
	}
	for _, tc := range tests {
		got := MortalApertureCapacity(tc.rank, tc.subRank)
		if got != tc.want {
			t.Errorf("MortalApertureCapacity(%d, %d) = %d, want %d", tc.rank, tc.subRank, got, tc.want)
		}
	}
}

