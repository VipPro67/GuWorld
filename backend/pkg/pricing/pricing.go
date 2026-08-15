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
)

// ---- Base price tables -----------------------------------------------------

// guRankBasePrice maps Gu rank (1-9) to base price in Primeval Stones.
// Index 0 unused; index i = rank i.
var guRankBasePrice = [10]int{
	0,       // unused
	200,     // Rank 1 - Nhat Chuyen
	500,     // Rank 2 - Nhi Chuyen
	1200,    // Rank 3 - Tam Chuyen
	2500,    // Rank 4 - Tu Chuyen
	5000,    // Rank 5 - Ngu Chuyen
	100000,  // Rank 6 - Luc Chuyen (Tien Co)
	250000,  // Rank 7 - That Chuyen
	600000,  // Rank 8 - Bat Chuyen
	1500000, // Rank 9 - Cuu Chuyen
}

// itemRankBasePrice maps Item rank (1-9) to base price in Primeval Stones.
// Rank 1 = 1, scales proportionally to the Gu curve (Gu / 200).
var itemRankBasePrice = [10]int{
	0,    // unused
	1,    // Rank 1
	3,    // Rank 2
	6,    // Rank 3
	13,   // Rank 4
	25,   // Rank 5
	500,  // Rank 6
	1250, // Rank 7
	3000, // Rank 8
	7500, // Rank 9
}

// ---- Base price helpers ----------------------------------------------------

// GuBasePriceByRank returns the default base price for a Gu of the given rank.
func GuBasePriceByRank(rank int) int {
	if rank < 1 || rank > 9 {
		return guRankBasePrice[1]
	}
	return guRankBasePrice[rank]
}

// GuBasePrice resolves the effective base price for a Gu.
// If dbBasePrice is positive it is used as-is (manual override from CSV/DB).
// Otherwise falls back to the rank table.
func GuBasePrice(dbBasePrice, rank int) int {
	if dbBasePrice > 0 {
		return dbBasePrice
	}
	return GuBasePriceByRank(rank)
}

// ItemBasePriceByRank returns the default base price for an Item of the given rank.
func ItemBasePriceByRank(rank int) int {
	if rank < 1 || rank > 9 {
		return itemRankBasePrice[1]
	}
	return itemRankBasePrice[rank]
}

// ItemBasePrice resolves the effective base price for an Item.
func ItemBasePrice(dbBasePrice, rank int) int {
	if dbBasePrice > 0 {
		return dbBasePrice
	}
	return ItemBasePriceByRank(rank)
}

// ---- Derived prices --------------------------------------------------------

// SellPrice returns Primeval Stones refunded when selling to the Caravan Shop.
// = SellBackRatio * basePrice, minimum 1.
func SellPrice(basePrice int) int {
	p := int(float64(basePrice) * SellBackRatio)
	if p < 1 {
		return 1
	}
	return p
}

// DonatePoints returns contribution points earned when donating to a sect.
// = DonateRatio * basePrice, minimum 1.
func DonatePoints(basePrice int) int {
	p := int(float64(basePrice) * DonateRatio)
	if p < 1 {
		return 1
	}
	return p
}

// VaultCost returns contribution points needed to buy from the sect vault.
// = DonatePoints * VaultMarkup, always > DonatePoints to prevent farming.
func VaultCost(basePrice int) int {
	p := int(float64(DonatePoints(basePrice)) * VaultMarkup)
	if p < 1 {
		return 1
	}
	return p
}

// RecipeCostByGuRank returns contribution points cost for a recipe in the sect
// vault, scaled by the rank of the target Gu.
func RecipeCostByGuRank(targetGuRank int) int {
	p := GuBasePriceByRank(targetGuRank) / RecipeRankDivisor
	if p < 20 {
		return 20 // minimum floor
	}
	return p
}

// RecipeDonatePoints returns contribution points awarded when donating a recipe
// to a sect. It mirrors the material formula: donate value = vault cost /
// VaultMarkup, ensuring buying from the vault is always more expensive than
// donating (no contribution-point farming loop).
func RecipeDonatePoints(targetGuRank int) int {
	p := int(float64(RecipeCostByGuRank(targetGuRank)) / VaultMarkup)
	if p < 1 {
		return 1
	}
	return p
}

// RecipeBasePrice returns the effective base price of a recipe.
// We set it equal to GuBasePrice(targetRank) so it scales properly when selling.
func RecipeBasePrice(targetGuRank int) int {
	return GuBasePriceByRank(targetGuRank)
}

// RecipeSellPrice returns Primeval Stones refunded when selling a Recipe to the Caravan Shop.
func RecipeSellPrice(targetGuRank int) int {
	return SellPrice(RecipeBasePrice(targetGuRank))
}
