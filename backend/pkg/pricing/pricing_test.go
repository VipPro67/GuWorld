package pricing

import "testing"

func TestCurrencyPrecision(t *testing.T) {
	if got := SellPrice(1); got != 0.1 {
		t.Fatalf("SellPrice(1) = %v, want 0.1", got)
	}
	if got := SellPrice(0.001); got != 0.001 {
		t.Fatalf("minimum sell price = %v, want 0.001", got)
	}
	if got := Quantize(1.2346); got != 1.235 {
		t.Fatalf("Quantize = %v, want 1.235", got)
	}
	if !IsValidAmount(0.001) || IsValidAmount(0.0001) {
		t.Fatal("0.001 precision validation failed")
	}
}

func TestRecipeCompletionFactor(t *testing.T) {
	cases := []struct {
		rate float64
		want float64
	}{
		{0, RecipeCompletionFloor},
		{-5, RecipeCompletionFloor},
		{5, RecipeCompletionFloor},
		{50, 0.5},
		{100, 1.0},
		{120, 1.0},
	}
	for _, tc := range cases {
		if got := RecipeCompletionFactor(tc.rate); got != tc.want {
			t.Fatalf("RecipeCompletionFactor(%v) = %v, want %v", tc.rate, got, tc.want)
		}
	}
	// Rank 1 base = 200 → 50% → base 100 → SellPrice 10.
	if got := RecipeSellPrice(1, 50); got != 10 {
		t.Fatalf("RecipeSellPrice(1, 50) = %v, want 10", got)
	}
}
