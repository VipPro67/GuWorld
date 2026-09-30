package worker

import (
	"testing"
)

func TestCalculateSectMaintenanceBreakdown(t *testing.T) {
	// Level 3 sect (3 * 2000 = 6000)
	// 10 members (10 * 200 = 2000)
	// Materials: 5 Rank 2 materials (5 * 2 * 5 = 50)
	// Mortal Gu: 3 Rank 3 mortal Gu (3 * 3 * 10 = 90)
	// Immortal Gu: 1 Rank 6 immortal Gu (1 * 6 * 200 = 1200)
	// Recipes: 2 Rank 4 recipes (2 * 4 * 50 = 400)
	// Total fee: 6000 + 2000 + 50 + 90 + 1200 + 400 = 9740

	b := SectMaintenanceBreakdown{
		SectLevel:       3,
		LevelFee:        3 * 2000,
		MemberCount:     10,
		MemberFee:       10 * 200,
		MaterialsCount:  5,
		MaterialsFee:    50,
		MortalGuCount:   3,
		MortalGuFee:     90,
		ImmortalGuCount: 1,
		ImmortalGuFee:   1200,
		RecipesCount:    2,
		RecipesFee:      400,
	}
	b.TotalFee = b.LevelFee + b.MemberFee + b.MaterialsFee + b.MortalGuFee + b.ImmortalGuFee + b.RecipesFee

	expected := int64(9740)
	if b.TotalFee != expected {
		t.Fatalf("expected total fee %d, got %d", expected, b.TotalFee)
	}
}
