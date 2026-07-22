package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
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

// UseGutsGu tráng hồn: tiêu hao 1 guts_gu, tăng 10 soul_hp & max_soul
func UseGutsGu(c *gin.Context) {
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

	if player.NeedsSoulRefine {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hồn phách đang bị phù nề, cần luyện hồn tại Cốc Lạc Phách để tiếp tục tráng hồn!"})
		return
	}

	gutsQty, err := InventorySvc.GetQuantity(tx, playerID.String(), "guts_gu", "materials")
	if err != nil || gutsQty <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không có Cổ Can Đảm (guts_gu)!"})
		return
	}

	limit := getSoulTierLimit(player.SoulTier)
	if player.MaxSoul >= limit {
		player.NeedsSoulRefine = true
		_, err = tx.Exec(`
			UPDATE players
			SET needs_soul_refine = $1
			WHERE id = $2
		`, player.NeedsSoulRefine, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update player status: " + err.Error()})
			return
		}
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Hồn phách đã đạt giới hạn của " + player.SoulTier + ". Hãy luyện hồn tại Cốc Lạc Phách để bứt phá giới hạn!",
			"player": player,
		})
		return
	}

	// Consume guts_gu
	if err := InventorySvc.RemoveItem(tx, playerID.String(), "guts_gu", "materials", 1); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tiêu hao Cổ Can Đảm!"})
		return
	}

	player.MaxSoul += 10
	player.SoulHP += 10
	if player.SoulHP > player.MaxSoul {
		player.SoulHP = player.MaxSoul
	}

	if player.MaxSoul >= limit {
		player.NeedsSoulRefine = true
	}

	_, err = tx.Exec(`
		UPDATE players
		SET max_soul = $1, soul_hp = $2, needs_soul_refine = $3
		WHERE id = $4
	`, player.MaxSoul, player.SoulHP, player.NeedsSoulRefine, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "soul.fortify",
		RequestSummary: fmt.Sprintf("maxSoul=%d soulHP=%d tier=%s", player.MaxSoul, player.SoulHP, player.SoulTier),
		Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message": "Tráng Hồn thành công! Hồn phách tăng thêm 10 điểm.",
		"player":  player,
	})
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

	_, err = tx.Exec(`
		UPDATE players
		SET soul_tier = $1, needs_soul_refine = $2, soul_hp = $3
		WHERE id = $4
	`, player.SoulTier, player.NeedsSoulRefine, player.SoulHP, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "soul.refine",
		RequestSummary: fmt.Sprintf("tier %s → %s", oldTier, newTier),
		Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message": "Luyện Hồn thành công! Hồn phách đột phá lên " + newTier + ", nhưng bị suy yếu do gió Lạc Phách (Soul HP giảm mạnh). Hãy dưỡng hồn để khôi phục.",
		"player":  player,
	})
}

// CalmSoul An Hồn (tĩnh dưỡng hồn phách)
func CalmSoul(c *gin.Context) {
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

	if player.SoulHP >= player.MaxSoul {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hồn phách của đạo hữu đã hoàn toàn ổn định và tràn đầy!"})
		return
	}

	if player.PrimevalStones < 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ 50 Nguyên Thạch để mua nước hồ An Hồn tĩnh dưỡng!"})
		return
	}

	player.PrimevalStones -= 50
	player.SoulHP = player.MaxSoul

	_, err = tx.Exec(`
		UPDATE players
		SET primeval_stones = $1, soul_hp = $2
		WHERE id = $3
	`, player.PrimevalStones, player.SoulHP, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save player: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &playerID, Action: "soul.calm",
		RequestSummary: fmt.Sprintf("maxSoul=%d soulHP=%d", player.MaxSoul, player.SoulHP),
		Success: true,
	})
	c.JSON(http.StatusOK, gin.H{
		"message": "An Hồn thành công! Hồn phách tĩnh dưỡng hồi phục đầy sinh lực.",
		"player":  player,
	})
}
