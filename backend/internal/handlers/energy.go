package handlers

import (
	"math"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const immortalRankThreshold = 6

func isImmortalCultivator(player *models.Player) bool {
	return player.Rank >= immortalRankThreshold
}

func energyName(player *models.Player) string {
	if isImmortalCultivator(player) {
		return "Tiên Nguyên"
	}
	return "Chân Nguyên"
}

func insufficientEnergyMessage(player *models.Player) string {
	if isImmortalCultivator(player) {
		return "Không đủ Tiên Nguyên. Cổ Tiên cần chờ Tiên Khiếu ngưng tụ hoặc luyện hóa Tiên Nguyên Thạch để bổ sung."
	}
	return "Không đủ Chân Nguyên. Vui lòng nghỉ ngơi tĩnh tọa để hồi phục."
}

func normalizeEnergyPool(player *models.Player) bool {
	if !isImmortalCultivator(player) {
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
			return true
		}
		return false
	}

	changed := false
	unlimitedMaxMana := 99999999999.0
	if player.MaxMana != unlimitedMaxMana {
		player.MaxMana = unlimitedMaxMana
		changed = true
	}
	return changed
}

func energyRegenRatePerHour(player *models.Player) float64 {
	var rate float64
	if isImmortalCultivator(player) {
		baseByRank := map[int]float64{
			6: 0.5,
			7: 0.8,
			8: 1.2,
			9: 2.0,
		}
		base := baseByRank[player.Rank]
		if base == 0 {
			base = 0.5 + float64(player.Rank-immortalRankThreshold)*0.35
		}
		subRank := player.SubRank
		if subRank < 1 {
			subRank = 1
		}
		if subRank > 4 {
			subRank = 4
		}
		rate = base * (1.0 + float64(subRank-1)*0.15)
	} else {
		switch player.AptitudeType {
		case "Giáp":
			rate = 8.0
		case "Ất":
			rate = 6.0
		case "Bính":
			rate = 4.0
		case "Đinh":
			rate = 2.0
		default:
			rate = 2.0
		}
	}

	refineTier := getRefineManaTier(player.Rank, player.GuInventory)
	if refineTier > 0 {
		rate *= 0.4 + refineTier*0.1
	}

	return rate * (1.0 + gamefx.Get(player).ManaRegenBonus)
}

func calculateEnergyCost(player *models.Player, guSpec *models.GuDictionaryItem) float64 {
	if isImmortalCultivator(player) {
		if guSpec.Rank < immortalRankThreshold {
			return 0
		}
		if player.Rank < guSpec.Rank {
			return 999999999
		}
		baseCost := 1.0
		if guSpec.BaseManaCost > 0 {
			baseCost = guSpec.BaseManaCost
		}
		if player.Rank > guSpec.Rank {
			return baseCost / math.Pow(100, float64(player.Rank-guSpec.Rank))
		}
		return baseCost
	}

	baseManaPct := 0.10
	if guSpec.BaseManaCost > 0.0 {
		baseManaPct = guSpec.BaseManaCost / 100.0
	}
	cost := baseManaPct * player.MaxMana

	if guSpec.Rank < immortalRankThreshold {
		rankDiff := player.Rank - guSpec.Rank
		if rankDiff > 0 {
			cost /= math.Pow(10, float64(rankDiff))
		} else if rankDiff < 0 {
			cost *= math.Pow(10, float64(-rankDiff))
		}
	}

	if !isImmortalCultivator(player) {
		refineTier := getRefineManaTier(player.Rank, player.GuInventory)
		if refineTier > 0 {
			cost *= 0.5
		}
	}

	return cost
}

func calculateEnergyCostForCombatant(actor *models.Combatant, guSpec *models.GuDictionaryItem) float64 {
	isImmortal := actor.Rank >= 6

	if isImmortal {
		if guSpec.Rank < immortalRankThreshold {
			return 0
		}
		if actor.Rank < guSpec.Rank {
			return 999999999
		}
		baseCost := 1.0
		if guSpec.BaseManaCost > 0 {
			baseCost = guSpec.BaseManaCost
		}
		if actor.Rank > guSpec.Rank {
			return baseCost / math.Pow(100, float64(actor.Rank-guSpec.Rank))
		}
		return baseCost
	}

	baseManaPct := 0.10
	if guSpec.BaseManaCost > 0.0 {
		baseManaPct = guSpec.BaseManaCost / 100.0
	}
	cost := baseManaPct * actor.MaxMana

	if guSpec.Rank < immortalRankThreshold {
		rankDiff := actor.Rank - guSpec.Rank
		if rankDiff > 0 {
			cost /= math.Pow(10, float64(rankDiff))
		} else if rankDiff < 0 {
			cost *= math.Pow(10, float64(-rankDiff))
		}
	}

	if !isImmortal {
		refineTier := getRefineManaTier(actor.Rank, actor.GuInventory)
		if refineTier > 0 {
			cost *= 0.5
		}
	}

	return cost
}


func getRefineManaTier(rank int, inventory []models.GuInstance) float64 {
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

func isPathInGu(paths []string, path string) bool {
	cleanPath := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(path), " path"))
	for _, p := range paths {
		cleanP := strings.TrimSpace(strings.TrimSuffix(strings.ToLower(p), " path"))
		if cleanP == cleanPath {
			return true
		}
	}
	return false
}
