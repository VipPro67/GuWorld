package sectmechanics

import "math"

const (
	minPlunderChance       = 0.05
	basePlunderChance      = 0.20
	plunderReductionPerLvl = 0.02
	minVaultFeeMultiplier  = 0.10
)

// PlunderChance is the chance that each stored resource is lost after a sect defeat.
func PlunderChance(treasuryLevel int) float64 {
	if treasuryLevel < 1 {
		treasuryLevel = 1
	}
	chance := basePlunderChance - float64(treasuryLevel)*plunderReductionPerLvl
	if chance < minPlunderChance {
		return minPlunderChance
	}
	return chance
}

// RollPlunderQuantity rolls the fractional remainder of qty*chance. Unlike ceil,
// this lets a treasury protect singleton valuables instead of always losing one.
func RollPlunderQuantity(qty int64, chance, roll float64) int64 {
	if qty <= 0 || chance <= 0 {
		return 0
	}
	if chance >= 1 {
		return qty
	}
	expected := float64(qty) * chance
	result := int64(math.Floor(expected))
	if roll < expected-float64(result) {
		result++
	}
	return result
}

// DiscountVaultFee applies the treasury's large storage-cost reduction: 50%
// at level 1, improving by 5 percentage points per level, capped at 90%.
func DiscountVaultFee(raw int64, treasuryLevel int) (discounted, saved int64) {
	if raw <= 0 {
		return 0, 0
	}
	if treasuryLevel < 1 {
		treasuryLevel = 1
	}
	multiplier := 0.55 - 0.05*float64(treasuryLevel)
	if multiplier < minVaultFeeMultiplier {
		multiplier = minVaultFeeMultiplier
	}
	discounted = int64(math.Round(float64(raw) * multiplier))
	return discounted, raw - discounted
}
