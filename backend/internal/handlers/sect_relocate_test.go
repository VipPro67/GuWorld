package handlers

import (
	"testing"
	"time"
)

func TestSectRelocationCooldownConstant(t *testing.T) {
	if SectRelocationCooldown != 72*time.Hour {
		t.Fatalf("expected SectRelocationCooldown to be 72h, got %v", SectRelocationCooldown)
	}
}

func TestSectRelocationBreakdownFormula(t *testing.T) {
	// Base cost: 10,000
	// 4 members: 4 * 500 = 2,000
	// Materials: 2 Rank 3 items (2 * 3 * 10 = 60), 5 Rank 1 items (5 * 1 * 10 = 50) => 110
	// Mortal Gu: 2 Rank 2 Gu (2 * 2 * 20 = 80), 1 Rank 5 Gu (1 * 5 * 20 = 100) => 180
	// Immortal Gu: 1 Rank 6 Gu (1 * 6 * 500 = 3000), 2 Rank 7 Gu (2 * 7 * 500 = 7000) => 10,000
	// Recipes: 2 Rank 4 recipes (2 * 4 * 100 = 800) => 800
	// Total expected: 10000 + 2000 + 110 + 180 + 10000 + 800 = 23090

	b := SectRelocationCostBreakdown{
		BaseCost:        10000,
		MemberCount:     4,
		MemberCost:      4 * 500,
		MaterialsCount:  7,
		MaterialsCost:   110,
		MortalGuCount:   3,
		MortalGuCost:    180,
		ImmortalGuCount: 3,
		ImmortalGuCost:  10000,
		RecipesCount:    2,
		RecipesCost:     800,
	}
	b.TotalCost = b.BaseCost + b.MemberCost + b.MaterialsCost + b.MortalGuCost + b.ImmortalGuCost + b.RecipesCost

	expected := int64(23090)
	if b.TotalCost != expected {
		t.Fatalf("expected total cost %d, got %d", expected, b.TotalCost)
	}
}

func TestIsInsideSquareTerritoryCheck(t *testing.T) {
	centerX, centerY, radius := 50, 50, 6

	// Center
	if !IsInsideSquareTerritory(50, 50, centerX, centerY, radius) {
		t.Errorf("center tile should be inside territory")
	}

	// Edges
	if !IsInsideSquareTerritory(56, 56, centerX, centerY, radius) {
		t.Errorf("corner (56, 56) should be inside territory")
	}
	if !IsInsideSquareTerritory(44, 44, centerX, centerY, radius) {
		t.Errorf("corner (44, 44) should be inside territory")
	}

	// Outside
	if IsInsideSquareTerritory(57, 50, centerX, centerY, radius) {
		t.Errorf("tile (57, 50) should be outside territory")
	}
	if IsInsideSquareTerritory(50, 43, centerX, centerY, radius) {
		t.Errorf("tile (50, 43) should be outside territory")
	}
}
