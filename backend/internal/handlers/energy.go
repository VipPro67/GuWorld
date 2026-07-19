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
		switch player.Rank {
		case 1:
			cost *= 1
		case 2:
			cost *= 2
		case 3:
			cost *= 5
		case 4:
			cost *= 20
		case 5:
			cost *= 100
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
		switch actor.Rank {
		case 1:
			cost *= 1
		case 2:
			cost *= 2
		case 3:
			cost *= 5
		case 4:
			cost *= 20
		case 5:
			cost *= 100
		}
	}

	return cost
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
