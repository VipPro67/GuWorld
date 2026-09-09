package handlers

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

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
		t.Errorf("expected 3000 stones, got %d", p.PrimevalStones)
	}

	// Primeval stones insufficient
	if err := deductGachaCostAmount(p, "primeval_stones", 4000); err == nil {
		t.Errorf("expected error for insufficient stones, got nil")
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
		},
		CultivationState: models.CultivationState{
			Lifespan: 50,
			HumanQi:  10,
		},
	}

	applyGachaRewardInMemory(p, GachaResultItem{Type: "primeval_stones", Amount: 500})
	if p.PrimevalStones != 1500 {
		t.Errorf("expected 1500 stones, got %d", p.PrimevalStones)
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

func TestAggregateGachaRewards(t *testing.T) {
	rewards := []GachaResultItem{
		{Type: "primeval_stones", Amount: 2000},
		{Type: "primeval_stones", Amount: 3000},
		{Type: "item", ID: "mat_iron", Amount: 2},
	}

	result := aggregateGachaRewards(rewards)
	if !strings.Contains(result, "5000 Nguyên Thạch") {
		t.Errorf("expected aggregated primeval stones 5000 Nguyên Thạch, got: %s", result)
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
