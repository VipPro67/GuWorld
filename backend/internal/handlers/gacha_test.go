package handlers

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestRollGuNeverReturnsExcludedGu(t *testing.T) {
	pc := &precomputedPools{
		guByRank: map[string][]weightedGuEntry{
			"6": {
				{id: "excluded", rank: 6, weight: 99},
				{id: "eligible", rank: 6, weight: 1},
			},
		},
	}

	for i := 0; i < 100; i++ {
		id, _ := pc.rollGu("6", map[string]bool{"excluded": true})
		if id != "eligible" {
			t.Fatalf("excluded Gu was returned: %q", id)
		}
	}
}

func TestDeductGachaCostAmount(t *testing.T) {
	p := &models.Player{
		ResourceState: models.ResourceState{
			PrimevalStones: 5000,
			ImmortalStones: 10,
		},
		CultivationState: models.CultivationState{
			Lifespan: 50,
		},
	}

	// Primeval stones success
	if err := deductGachaCostAmount(p, "primeval_stones", 2000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.PrimevalStones != 3000 {
		t.Errorf("expected 3000 stones, got %.3f", p.PrimevalStones)
	}

	// Primeval stones insufficient
	if err := deductGachaCostAmount(p, "primeval_stones", 4000); err == nil {
		t.Errorf("expected error for insufficient stones, got nil")
	}

	// Immortal stones success
	if err := deductGachaCostAmount(p, "immortal_stones", 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ImmortalStones != 7 {
		t.Errorf("expected 7 immortal stones, got %.3f", p.ImmortalStones)
	}

	// Immortal stones insufficient
	if err := deductGachaCostAmount(p, "immortal_stones", 100); err == nil {
		t.Errorf("expected error for insufficient immortal stones, got nil")
	}

	// Lifespan success
	if err := deductGachaCostAmount(p, "lifespan", 10); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Lifespan != 40 {
		t.Errorf("expected 40 lifespan, got %f", p.Lifespan)
	}

	// Lifespan insufficient
	if err := deductGachaCostAmount(p, "lifespan", 100); err == nil {
		t.Errorf("expected error for insufficient lifespan, got nil")
	}

	// Invalid cost type
	if err := deductGachaCostAmount(p, "invalid_cost", 10); err == nil {
		t.Errorf("expected error for invalid cost type, got nil")
	}
}

func TestApplyGachaRewardInMemory(t *testing.T) {
	p := &models.Player{
		ResourceState: models.ResourceState{
			PrimevalStones: 1000,
			ImmortalStones: 5,
		},
		CultivationState: models.CultivationState{
			Lifespan: 50,
			HumanQi:  10,
		},
	}

	applyGachaRewardInMemory(p, GachaResultItem{Type: "primeval_stones", Amount: 500})
	if p.PrimevalStones != 1500 {
		t.Errorf("expected 1500 stones, got %.3f", p.PrimevalStones)
	}

	applyGachaRewardInMemory(p, GachaResultItem{Type: "immortal_stones", Amount: 20})
	if p.ImmortalStones != 25 {
		t.Errorf("expected 25 immortal stones, got %.3f", p.ImmortalStones)
	}

	applyGachaRewardInMemory(p, GachaResultItem{Type: "lifespan", Amount: 5})
	if p.Lifespan != 55 {
		t.Errorf("expected 55 lifespan, got %f", p.Lifespan)
	}

	applyGachaRewardInMemory(p, GachaResultItem{Type: "human_qi", Amount: 2})
	if p.HumanQi != 12 {
		t.Errorf("expected 12 human qi, got %f", p.HumanQi)
	}
}

func TestBuildGachaUpdateQuery(t *testing.T) {
	p := &models.Player{
		ResourceState: models.ResourceState{
			PrimevalStones: 2000,
			ImmortalStones: 50,
		},
		CultivationState: models.CultivationState{
			Lifespan:     60,
			HumanQi:      100,
			WallProgress: 0.5,
		},
	}

	query, args := buildGachaUpdateQuery(p)
	if !strings.Contains(query, "immortal_stones") {
		t.Fatalf("expected query to contain immortal_stones, got: %s", query)
	}
	if !strings.Contains(query, "primeval_stones") {
		t.Fatalf("expected query to contain primeval_stones, got: %s", query)
	}

	// Verify args contain immortal_stones value
	foundImmortal := false
	for _, arg := range args {
		if v, ok := arg.(float64); ok && v == 50 {
			foundImmortal = true
			break
		}
	}
	if !foundImmortal {
		t.Fatalf("expected args to contain immortal_stones value (50), got: %v", args)
	}
}

func TestAggregateGachaRewards(t *testing.T) {
	rewards := []GachaResultItem{
		{Type: "primeval_stones", Amount: 2000},
		{Type: "primeval_stones", Amount: 3000},
		{Type: "immortal_stones", Amount: 100},
		{Type: "immortal_stones", Amount: 50},
		{Type: "item", ID: "mat_iron", Amount: 2},
	}

	result := aggregateGachaRewards(rewards)
	if !strings.Contains(result, "5000 Nguyên Thạch") {
		t.Errorf("expected aggregated primeval stones 5000 Nguyên Thạch, got: %s", result)
	}
	if !strings.Contains(result, "150 Tiên Nguyên Thạch") {
		t.Errorf("expected aggregated immortal stones 150 Tiên Nguyên Thạch, got: %s", result)
	}
}

func TestBuildGachaRewardLabel(t *testing.T) {
	rLabel := GachaResultItem{Type: "primeval_stones", Amount: 2000, Label: "2.000 Nguyên Thạch"}
	label := buildGachaRewardLabel(rLabel)
	if label != "2.000 Nguyên Thạch" {
		t.Errorf("unexpected label: %s", label)
	}

	rFallback := GachaResultItem{Type: "lifespan", Amount: 5}
	labelFallback := buildGachaRewardLabel(rFallback)
	if labelFallback != "5 lifespan" {
		t.Errorf("unexpected fallback label: %s", labelFallback)
	}
}

// TestGachaPoolsAreValid guards the hard-coded pool tables: no duplicate ids,
// no invalid costs, and no reward type the roll/apply switch cannot handle
// (e.g. the removed "recipe_rank").
func TestGachaPoolsAreValid(t *testing.T) {
	known := map[string]bool{
		"item": true, "food": true, "gu": true, "gu_rank": true,
		"primeval_stones": true, "immortal_stones": true,
	}
	seen := map[string]bool{}
	for _, pool := range gachaPools {
		if seen[pool.ID] {
			t.Fatalf("duplicate pool id %q", pool.ID)
		}
		seen[pool.ID] = true
		if pool.CostAmount <= 0 {
			t.Fatalf("pool %q has invalid cost %v", pool.ID, pool.CostAmount)
		}
		var total float64
		for _, r := range pool.Rewards {
			if !known[r.Type] {
				t.Fatalf("pool %q has unhandled reward type %q", pool.ID, r.Type)
			}
			if r.Weight <= 0 {
				t.Fatalf("pool %q reward %q has non-positive weight", pool.ID, r.Type)
			}
			total += r.Weight
		}
		if total <= 0 {
			t.Fatalf("pool %q has no rewards", pool.ID)
		}
	}
}
