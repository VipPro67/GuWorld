package gamefx

import (
	"math"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/util"
)

// EnergyCost computes the mana (Chân Nguyên / Tiên Nguyên) required to use a
// Gu skill for a player. Immortal cultivators pay Tiên Nguyên (scaled by
// rank/refinement), mortals pay a percentage of MaxMana.
func EnergyCost(player *models.Player, guSpec *models.GuDictionaryItem) float64 {
	if util.IsImmortalPlayer(player) {
		if guSpec.Rank < util.ImmortalRankThreshold {
			return 0
		}
		baseCost := 0.0
		if guSpec.BaseManaCost > 0 {
			baseCost = guSpec.BaseManaCost
		}
		cost := baseCost
		if player.Rank > guSpec.Rank {
			cost = baseCost / math.Pow(100, float64(player.Rank-guSpec.Rank))
		} else if player.Rank < guSpec.Rank {
			cost = baseCost * HigherRankManaMultiplier(guSpec.Rank, player.Rank)
		}
		refineTier := RefineManaTier(player.Rank, player.GuInventory)
		if refineTier > 0 {
			cost *= 0.5
		}
		return cost
	}

	baseManaPct := 0.0
	if guSpec.BaseManaCost > 0.0 {
		baseManaPct = guSpec.BaseManaCost / 100.0
	}
	cost := baseManaPct * player.MaxMana

	if player.Rank > guSpec.Rank && guSpec.Rank < util.ImmortalRankThreshold {
		cost /= math.Pow(10, float64(player.Rank-guSpec.Rank))
	} else if player.Rank < guSpec.Rank {
		cost *= HigherRankManaMultiplier(guSpec.Rank, player.Rank)
	} else {
		cost *= SubRankManaFactor(player.SubRank)
	}

	refineTier := RefineManaTier(player.Rank, player.GuInventory)
	if refineTier > 0 {
		cost *= 0.5
	}

	return cost
}

// EnergyCostForCombatant computes the mana cost for a combatant (player or
// enemy) using the same rules as EnergyCost but driven by a Combatant struct.
func EnergyCostForCombatant(actor *models.Combatant, guSpec *models.GuDictionaryItem) float64 {
	isImmortal := actor.Rank >= 6

	if isImmortal {
		if guSpec.Rank < util.ImmortalRankThreshold {
			return 0
		}
		baseCost := 0.0
		if guSpec.BaseManaCost > 0 {
			baseCost = guSpec.BaseManaCost
		}
		cost := baseCost
		if actor.Rank > guSpec.Rank {
			cost = baseCost / math.Pow(100, float64(actor.Rank-guSpec.Rank))
		} else if actor.Rank < guSpec.Rank {
			cost = baseCost * HigherRankManaMultiplier(guSpec.Rank, actor.Rank)
		}
		refineTier := RefineManaTier(actor.Rank, actor.GuInventory)
		if refineTier > 0 {
			cost *= 0.5
		}
		return cost
	}

	baseManaPct := 0.0
	if guSpec.BaseManaCost > 0.0 {
		baseManaPct = guSpec.BaseManaCost / 100.0
	}
	cost := baseManaPct * actor.MaxMana

	if actor.Rank > guSpec.Rank && guSpec.Rank < util.ImmortalRankThreshold {
		cost /= math.Pow(10, float64(actor.Rank-guSpec.Rank))
	} else if actor.Rank < guSpec.Rank {
		cost *= HigherRankManaMultiplier(guSpec.Rank, actor.Rank)
	} else {
		cost *= SubRankManaFactor(actor.SubRank)
	}

	refineTier := RefineManaTier(actor.Rank, actor.GuInventory)
	if refineTier > 0 {
		cost *= 0.5
	}

	return cost
}

func HigherRankManaMultiplier(guRank, userRank int) float64 {
	if guRank <= userRank {
		return 1.0
	}

	if userRank < util.ImmortalRankThreshold && guRank >= util.ImmortalRankThreshold {
		return math.Inf(1)
	}

	if userRank < util.ImmortalRankThreshold {
		values := []float64{1, 2, 5, 20, 100}
		uIdx := userRank - 1
		if uIdx < 0 {
			uIdx = 0
		} else if uIdx >= len(values) {
			uIdx = len(values) - 1
		}
		gIdx := guRank - 1
		if gIdx < 0 {
			gIdx = 0
		} else if gIdx >= len(values) {
			gIdx = len(values) - 1
		}
		return values[gIdx] / values[uIdx]
	}

	values := []float64{1, 100, 10000, 1000000}
	uIdx := userRank - 6
	if uIdx < 0 {
		uIdx = 0
	} else if uIdx >= len(values) {
		uIdx = len(values) - 1
	}
	gIdx := guRank - 6
	if gIdx < 0 {
		gIdx = 0
	} else if gIdx >= len(values) {
		gIdx = len(values) - 1
	}
	return values[gIdx] / values[uIdx]
}

// SubRankManaFactor halves mortal Gu mana cost for every sub-rank the user sits
// above sub 1 (Gu dictionary has no sub-rank of its own, so it counts as sub 1).
// Only applied when user and Gu share the same rank — once ranks differ, the
// rank ladder alone decides the cost.
func SubRankManaFactor(userSubRank int) float64 {
	gap := userSubRank - 1
	if gap <= 0 {
		return 1.0
	}
	return 1.0 / math.Pow(2.0, float64(gap))
}

// CanUseGuWithMana reports whether a user of the given rank may spend mana to
// activate the given Gu. Mortal users (rank below the immortal threshold) are
// blocked from using immortal Gu (rank >= threshold).
func CanUseGuWithMana(userRank int, guSpec *models.GuDictionaryItem) bool {
	if userRank < util.ImmortalRankThreshold && guSpec.Rank >= util.ImmortalRankThreshold {
		return false
	}
	return true
}

// CanEquipGu reports whether a user of playerRank can equip a Gu of guRank.
// Mortals (rank < 6) can equip Mortal Gu (rank < 6).
// Immortals (rank >= 6) can equip any Gu.
// Mortals cannot equip Immortal Gu (rank >= 6).
func CanEquipGu(playerRank, guRank int) bool {
	if playerRank >= util.ImmortalRankThreshold {
		return true
	}
	return guRank < util.ImmortalRankThreshold
}

// HigherRankHPCost returns the % MaxHP backlash damage incurred when using a Gu
// whose rank is higher than the user's rank (15% MaxHP per rank difference).
func HigherRankHPCost(guRank, userRank int, maxHP float64) float64 {
	if userRank < 1 {
		userRank = 1
	}
	if guRank <= userRank {
		return 0
	}
	diff := guRank - userRank
	backlashPct := float64(diff) * 0.15
	return maxHP * backlashPct
}

// RefineManaTier returns the highest refinement mana tier among equipped ACTIVE
// Gus whose rank matches the given rank. 0 means no refinement tier.
func RefineManaTier(rank int, inventory []models.GuInstance) float64 {
	var maxTier float64
	for _, gu := range inventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			spec, ok := models.GetGu(gu.GuID, gu.Rank)
			if tier := models.GuStatValue(spec, "REFINE_MANA_TIER"); ok && tier > 0 {
				if spec.Rank == rank {
					if tier > maxTier {
						maxTier = tier
					}
				}
			}
		}
	}
	return maxTier
}
