package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

func getSoulTierLimit(tier string) int {
	switch tier {
	case "", "NHAN_HON":
		return 100
	case "BACH_NHAN":
		return 1000
	case "THIEN_NHAN":
		return 10000
	case "VAN_NHAN":
		return 1000000
	case "BACH_VAN":
		return 10000000
	case "THIEN_VAN":
		return 100000000
	default:
		return 100
	}
}

func getNextSoulTier(tier string) string {
	switch tier {
	case "", "NHAN_HON":
		return "BACH_NHAN"
	case "BACH_NHAN":
		return "THIEN_NHAN"
	case "THIEN_NHAN":
		return "VAN_NHAN"
	case "VAN_NHAN":
		return "BACH_VAN"
	case "BACH_VAN":
		return "THIEN_VAN"
	case "THIEN_VAN":
		return "UC_NHAN"
	default:
		return "BACH_NHAN"
	}
}

// RefineSoul Luyện Hồn tại Cốc Lạc Phách
func RefineSoul(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player: " + err.Error()})
		return
	}

	if !player.NeedsSoulRefine {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hồn phách của đạo hữu chưa đến giới hạn cần luyện!"})
		return
	}

	oldTier := player.SoulTier
	newTier := getNextSoulTier(oldTier)
	player.SoulTier = newTier
	player.NeedsSoulRefine = false

	// Trừ 80% soul_hp hiện tại (suy yếu hồn phách do luyện hồn khốc liệt)
	reduction := int(float64(player.SoulHP) * 0.8)
	player.SoulHP -= reduction
	if player.SoulHP < 1 {
		player.SoulHP = 1
	}

	qiGained := 0.0

	_, err = tx.Exec(`
		UPDATE players
		SET soul_tier = $1, needs_soul_refine = $2, soul_hp = $3, human_qi = $4
		WHERE id = $5
	`, player.SoulTier, player.NeedsSoulRefine, player.SoulHP, player.HumanQi, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	responseWithPlayer(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Luyện Hồn thành công! Hồn phách đột phá lên %s, nhưng bị suy yếu do gió Lạc Phách (Soul HP giảm mạnh). Hãy dưỡng hồn để khôi phục. Nhân Khí +%.0f.", newTier, qiGained),
	})
}
