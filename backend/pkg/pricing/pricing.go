// Package pricing provides unified price calculation for Gu, Items, and Recipes.
// All sell/donate/vault prices are derived from a single base_price stored in DB.
//
// Price flow:
//
//	base_price (DB or rank-based fallback)
//	  × SellBackRatio  → sell price at Caravan shop
//	  × DonateRatio    → contribution points when donating to sect
//	  × VaultMarkup    → cost in contribution points to buy from sect vault
//	                     (always > donate so vault-farming is impossible)
package pricing

import "math"

// Quantize rounds currency to its smallest supported unit (0.001).
func Quantize(value float64) float64 { return math.Round(value*1000) / 1000 }

// IsValidAmount checks that a value is finite and aligned to 0.001.
func IsValidAmount(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Abs(value-Quantize(value)) < 1e-9
}

// ---- Ratios ----------------------------------------------------------------

const (
	// SellBackRatio: fraction of base_price returned when selling to Caravan Shop.
	SellBackRatio = 0.10

	// DonateRatio: fraction of base_price awarded as contribution points on donation.
	DonateRatio = 0.50

	// VaultMarkup: multiplier applied on top of DonateRatio for vault buy cost.
	// Must be > 1 to prevent donation->buy farming loop.
	// Net vault cost = base_price × DonateRatio × VaultMarkup
	VaultMarkup = 1.50

	// RecipeRankDivisor: recipe vault cost = GuBasePrice(targetRank) / RecipeRankDivisor
	RecipeRankDivisor = 10

	// RecipeCompletionFloor: minimum fraction of base price a recipe keeps when
	// sold to the system (Caravan / roaming merchant), regardless of completion
	// rate. A 0%-complete recipe still fetches this much; 100% fetches full.
	RecipeCompletionFloor = 0.10
)

// ---- Currency tiers --------------------------------------------------------

// StoneTierByRank returns the currency tier used to price an entity (item, Gu,
// recipe) of the given rank:
//
//	rank < 6 → "primeval" (Nguyên Thạch)
//	rank >= 6 → "immortal" (Tiên Nguyên Thạch)
func StoneTierByRank(rank int) string {
	if rank >= 6 {
		return "immortal"
	}
	return "primeval"
}

// ---- Base price tables -----------------------------------------------------

// guRankBasePrice maps Gu rank (1-9) to base price.
// Ranks 1-5 are in Primeval Stones (Nguyên Thạch); ranks 6-9 are Tiên Cổ and
// are in Immortal Stones (Tiên Nguyên Thạch) - see StoneTierByRank.
// Index 0 unused; index i = rank i.
// Index 0 unused; index i = rank i.
var guRankBasePrice = [10]float64{
	0,    // unused
	200,  // Rank 1 - Nhat Chuyen
	500,  // Rank 2 - Nhi Chuyen
	1200, // Rank 3 - Tam Chuyen
	2500, // Rank 4 - Tu Chuyen
	5000, // Rank 5 - Ngu Chuyen
	200,  // Rank 6 - Luc Chuyen (Tien Co, Immortal Stones)
	500,  // Rank 7 - That Chuyen
	1200, // Rank 8 - Bat Chuyen
	2500, // Rank 9 - Cuu Chuyen
}

// itemRankBasePrice maps Item rank (1-9) to base price.
// Ranks 1-5 are in Primeval Stones (Nguyên Thạch); ranks 6-9 are in Immortal
// Stones (Tiên Nguyên Thạch) - see StoneTierByRank.
// Rank 1 = 1, scales proportionally to the Gu curve (Gu / 200).
var itemRankBasePrice = [10]float64{
	0,   // unused
	1,   // Rank 1
	3,   // Rank 2
	6,   // Rank 3
	13,  // Rank 4
	25,  // Rank 5
	10,  // Rank 6
	25,  // Rank 7
	60,  // Rank 8
	150, // Rank 9
}

// ---- Base price helpers ----------------------------------------------------

// GuBasePriceByRank returns the default base price for a Gu of the given rank.
func GuBasePriceByRank(rank int) float64 {
	if rank < 1 || rank > 9 {
		return guRankBasePrice[1]
	}
	return guRankBasePrice[rank]
}

// GuBasePrice resolves the effective base price for a Gu.
// If dbBasePrice is positive it is used as-is (manual override from CSV/DB).
// Otherwise falls back to the rank table.
func GuBasePrice(dbBasePrice float64, rank int) float64 {
	if dbBasePrice > 0 {
		return dbBasePrice
	}
	return GuBasePriceByRank(rank)
}

// ItemBasePriceByRank returns the default base price for an Item of the given rank.
func ItemBasePriceByRank(rank int) float64 {
	if rank < 1 || rank > 9 {
		return itemRankBasePrice[1]
	}
	return itemRankBasePrice[rank]
}

// ItemBasePrice resolves the effective base price for an Item.
func ItemBasePrice(dbBasePrice float64, rank int) float64 {
	if dbBasePrice > 0 {
		return dbBasePrice
	}
	return ItemBasePriceByRank(rank)
}

// ---- Derived prices --------------------------------------------------------

// SellPrice returns Primeval Stones refunded when selling to the Caravan Shop.
// = SellBackRatio * basePrice, minimum 1.
func SellPrice(basePrice float64) float64 {
	p := Quantize(basePrice * SellBackRatio)
	if p < 0.001 {
		return 0.001
	}
	return p
}

// DonatePoints returns contribution points earned when donating to a sect.
// = DonateRatio * basePrice, minimum 1.
func DonatePoints(basePrice float64) int64 {
	p := int64(float64(basePrice) * DonateRatio)
	if p < 1 {
		return 1
	}
	return p
}

// VaultCost returns contribution points needed to buy from the sect vault.
// = DonatePoints * VaultMarkup, always > DonatePoints to prevent farming.
func VaultCost(basePrice float64) int64 {
	p := int64(float64(DonatePoints(basePrice)) * VaultMarkup)
	if p < 1 {
		return 1
	}
	return p
}

// RecipeCostByGuRank returns contribution points cost for a recipe in the sect
// vault, scaled by the rank of the target Gu.
func RecipeCostByGuRank(targetGuRank int) int64 {
	p := int64(GuBasePriceByRank(targetGuRank)) / RecipeRankDivisor
	if p < 20 {
		return 20 // minimum floor
	}
	return p
}

// RecipeDonatePoints returns contribution points awarded when donating a recipe
// to a sect. It mirrors the material formula: donate value = vault cost /
// VaultMarkup, ensuring buying from the vault is always more expensive than
// donating (no contribution-point farming loop).
func RecipeDonatePoints(targetGuRank int) int64 {
	p := int64(float64(RecipeCostByGuRank(targetGuRank)) / VaultMarkup)
	if p < 1 {
		return 1
	}
	return p
}

// RecipeBasePrice returns the effective base price of a recipe.
// We set it equal to GuBasePrice(targetRank) so it scales properly when selling.
func RecipeBasePrice(targetGuRank int) float64 {
	return GuBasePriceByRank(targetGuRank)
}

// RecipeCompletionFactor maps a recipe completion rate (0-100) to the fraction
// of its base price it is worth. Linear from RecipeCompletionFloor at 0% up to
// 1.0 at 100%. Values outside 0-100 are clamped.
func RecipeCompletionFactor(completionRate float64) float64 {
	if completionRate < 0 {
		completionRate = 0
	}
	if completionRate > 100 {
		completionRate = 100
	}
	frac := completionRate / 100
	if frac < RecipeCompletionFloor {
		return RecipeCompletionFloor
	}
	return frac
}

// RecipeSellPrice returns Primeval Stones refunded when selling a Recipe to the
// Caravan Shop, scaled by the recipe's completion rate.
func RecipeSellPrice(targetGuRank int, completionRate float64) float64 {
	return SellPrice(RecipeBasePrice(targetGuRank) * RecipeCompletionFactor(completionRate))
}
