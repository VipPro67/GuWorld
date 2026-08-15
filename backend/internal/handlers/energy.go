package handlers

import (
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
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
