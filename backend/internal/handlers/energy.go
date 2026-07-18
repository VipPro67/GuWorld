package handlers

import (
	"math"
	"strings"

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

func immortalApertureCapacity(rank, subRank int) float64 {
	if rank < immortalRankThreshold {
		return 0
	}

	baseByRank := map[int]float64{
		6: 120,
		7: 180,
		8: 260,
		9: 360,
	}
	base := baseByRank[rank]
	if base == 0 {
		base = 120 + float64(rank-immortalRankThreshold)*70
	}

	if subRank < 1 {
		subRank = 1
	}
	if subRank > 4 {
		subRank = 4
	}

	return base + float64(subRank-1)*20
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
	unlimitedMaxMana := 999999999.0
	if player.MaxMana != unlimitedMaxMana {
		player.MaxMana = unlimitedMaxMana
		changed = true
	}
	return changed
}

func energyRegenRatePerHour(player *models.Player) float64 {
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
		return base * (1.0 + float64(subRank-1)*0.15)
	}

	switch player.AptitudeType {
	case "Giáp":
		return 8.0
	case "Ất":
		return 6.0
	case "Bính":
		return 4.0
	case "Đinh":
		return 2.0
	default:
		return 2.0
	}
}

func immortalGuBaseCost(rank int, configuredCost float64) float64 {
	if configuredCost > 0 {
		return configuredCost
	}

	switch rank {
	case 6:
		return 1.0
	case 7:
		return 2.5
	case 8:
		return 6.0
	case 9:
		return 15.0
	default:
		return math.Max(0.1, float64(rank-5))
	}
}

func calculateEnergyCost(player *models.Player, guSpec *models.GuDictionaryItem) float64 {
	if isImmortalCultivator(player) {
		if guSpec.Rank < immortalRankThreshold {
			// To an Immortal Aperture, mortal Gu draw only a trace of immortal essence.
			return 0.1
		}

		cost := immortalGuBaseCost(guSpec.Rank, guSpec.BaseManaCost)
		if player.Rank > guSpec.Rank {
			for i := 0; i < player.Rank-guSpec.Rank; i++ {
				cost *= 0.85
			}
		}
		if player.SubRank > 1 {
			cost *= math.Pow(0.92, float64(player.SubRank-1))
		}
		if cost < 0.1 {
			return 0.1
		}
		return cost
	}

	baseManaPct := 0.10
	if guSpec.BaseManaCost > 0.0 {
		baseManaPct = guSpec.BaseManaCost / 100.0
	}
	cost := baseManaPct * player.MaxMana

	if player.Rank > guSpec.Rank {
		for i := 0; i < player.Rank-guSpec.Rank; i++ {
			cost /= 10.0
		}
	}

	divisor := 1
	for i := 1; i < player.SubRank; i++ {
		divisor *= 2
	}
	cost /= float64(divisor)
	return cost
}

func calculateEnergyCostForCombatant(actor *models.Combatant, guSpec *models.GuDictionaryItem) float64 {
	isImmortal := actor.Rank >= 6

	if isImmortal {
		if guSpec.Rank < immortalRankThreshold {
			// To an Immortal Aperture, mortal Gu draw only a trace of immortal essence.
			return 0.1
		}

		cost := immortalGuBaseCost(guSpec.Rank, guSpec.BaseManaCost)
		if actor.Rank > guSpec.Rank {
			for i := 0; i < actor.Rank-guSpec.Rank; i++ {
				cost *= 0.85
			}
		}
		if actor.SubRank > 1 {
			cost *= math.Pow(0.92, float64(actor.SubRank-1))
		}
		if cost < 0.1 {
			cost = 0.1
		}

		if cost < 0.1 {
			return 0.1
		}
		return cost
	}

	baseManaPct := 0.10
	if guSpec.BaseManaCost > 0.0 {
		baseManaPct = guSpec.BaseManaCost / 100.0
	}
	cost := baseManaPct * actor.MaxMana

	if actor.Rank > guSpec.Rank {
		for i := 0; i < actor.Rank-guSpec.Rank; i++ {
			cost /= 10.0
		}
	} else if actor.Rank < guSpec.Rank {
		rankDiff := guSpec.Rank - actor.Rank
		for i := 0; i < rankDiff; i++ {
			cost *= 1.5
		}
	}

	divisor := 1
	for i := 1; i < actor.SubRank; i++ {
		divisor *= 2
	}
	cost /= float64(divisor)

	return cost
}

func hasSovereignFetus(actor *models.Combatant) bool {
	for _, gu := range actor.GuInventory {
		if gu.GuID == "GU_SOVEREIGN_FETUS" && gu.Equipped && gu.Status == "ACTIVE" {
			return true
		}
	}
	return false
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
