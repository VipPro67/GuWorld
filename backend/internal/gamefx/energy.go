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
		if player.Rank > guSpec.Rank {
			return baseCost / math.Pow(100, float64(player.Rank-guSpec.Rank))
		}
		if player.Rank < guSpec.Rank {
			return baseCost * HigherRankManaMultiplier(guSpec.Rank, player.Rank)
		}
		return baseCost
	}

	baseManaPct := 0.10
	if guSpec.BaseManaCost > 0.0 {
		baseManaPct = guSpec.BaseManaCost / 100.0
	}
	cost := baseManaPct * player.MaxMana

	if player.Rank > guSpec.Rank && guSpec.Rank < util.ImmortalRankThreshold {
		cost /= math.Pow(10, float64(player.Rank-guSpec.Rank))
	} else if player.Rank < guSpec.Rank {
		cost *= HigherRankManaMultiplier(guSpec.Rank, player.Rank)
	}

	if util.IsImmortalPlayer(player) {
		refineTier := RefineManaTier(player.Rank, player.GuInventory)
		if refineTier > 0 {
			cost *= 0.5
		}
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
		if actor.Rank > guSpec.Rank {
			return baseCost / math.Pow(100, float64(actor.Rank-guSpec.Rank))
		}
		if actor.Rank < guSpec.Rank {
			return baseCost * HigherRankManaMultiplier(guSpec.Rank, actor.Rank)
		}
		return baseCost
	}

	baseManaPct := 0.10
	if guSpec.BaseManaCost > 0.0 {
		baseManaPct = guSpec.BaseManaCost / 100.0
	}
	cost := baseManaPct * actor.MaxMana

	if actor.Rank > guSpec.Rank && guSpec.Rank < util.ImmortalRankThreshold {
		cost /= math.Pow(10, float64(actor.Rank-guSpec.Rank))
	} else if actor.Rank < guSpec.Rank {
		cost *= HigherRankManaMultiplier(guSpec.Rank, actor.Rank)
	}

	if !isImmortal {
		refineTier := RefineManaTier(actor.Rank, actor.GuInventory)
		if refineTier > 0 {
			cost *= 0.5
		}
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
		values := []float64{1, 2, 5, 25, 125}
		return values[guRank-1] / values[userRank-1]
	}

	values := []float64{1, 100, 10000, 1000000, 100000000}
	return values[guRank-6] / values[userRank-6]
}

// CanUseGuWithMana reports whether a user of the given rank may spend mana to
// activate the given Gu. Mortal users (rank below the immortal threshold) are
// blocked from using immortal Gu (rank >= threshold) that cost mana. The cost
// multiplier returns +Inf for such combinations; callers should reject the use
// entirely instead of leaking +Inf into mana pools.
func CanUseGuWithMana(userRank int, guSpec *models.GuDictionaryItem) bool {
	if userRank < util.ImmortalRankThreshold && guSpec.Rank >= util.ImmortalRankThreshold && guSpec.BaseManaCost > 0 {
		return false
	}
	return true
}

// RefineManaTier returns the highest refinement mana tier among equipped ACTIVE
// Gus whose rank matches the given rank. 0 means no refinement tier.
func RefineManaTier(rank int, inventory []models.GuInstance) float64 {
	var maxTier float64
	for _, gu := range inventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			spec, ok := models.GetGu(gu.GuID, gu.Rank)
			if ok && spec.Effects.Passive != nil && spec.Effects.Passive.RefineManaTier > 0 {
				if spec.Rank == rank {
					if spec.Effects.Passive.RefineManaTier > maxTier {
						maxTier = spec.Effects.Passive.RefineManaTier
					}
				}
			}
		}
	}
	return maxTier
}
