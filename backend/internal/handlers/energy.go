package handlers

import (
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/util"
)

func energyName(player *models.Player) string {
	if util.IsImmortalPlayer(player) {
		return "Tiên Nguyên"
	}
	return "Chân Nguyên"
}

func insufficientEnergyMessage(player *models.Player) string {
	if util.IsImmortalPlayer(player) {
		return "Không đủ Tiên Nguyên. Cổ Tiên cần chờ Tiên Khiếu ngưng tụ hoặc luyện hóa Tiên Nguyên Thạch để bổ sung."
	}
	return "Không đủ Chân Nguyên. Vui lòng nghỉ ngơi tĩnh tọa để hồi phục."
}

func normalizeEnergyPool(player *models.Player) bool {
	if !util.IsImmortalPlayer(player) {
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
	if util.IsImmortalPlayer(player) {
		baseByRank := map[int]float64{
			6: 0.5,
			7: 0.8,
			8: 1.2,
			9: 2.0,
		}
		base := baseByRank[player.Rank]
		if base == 0 {
			base = 0.5 + float64(player.Rank-util.ImmortalRankThreshold)*0.35
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

	refineTier := gamefx.RefineManaTier(player.Rank, player.GuInventory)
	if refineTier > 0 {
		rate *= 0.4 + refineTier*0.1
	}

	return rate * (1.0 + gamefx.Get(player).ManaRegenBonus)
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
