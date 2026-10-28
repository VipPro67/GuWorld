package handlers

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type RechargeWithStoneRequest struct {
	Stones int64 `json:"stones" binding:"required,min=1"`
}

// RechargeWithStones converts primeval stones into current mana for mortals
func RechargeWithStones(c *gin.Context) {
	var req RechargeWithStoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player stats"})
		return
	}

	// Cổ Tiên (Rank 6+) sử dụng Tiên Nguyên Thạch, không nạp bằng Nguyên Thạch phàm tục
	if player.Rank >= 6 {
		if player.ImmortalStones < float64(req.Stones) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không đủ Tiên Nguyên Thạch."})
			return
		}

		stonesToUse := int64(req.Stones)
		manaGain := float64(stonesToUse) / math.Pow(100, float64(player.Rank-6))
		player.CurrentMana += manaGain
		player.ImmortalStones -= float64(stonesToUse)
		if player.CurrentMana > player.MaxMana {
			player.CurrentMana = player.MaxMana
		}

		_, err = tx.Exec(`
			UPDATE players 
			SET current_mana = $1, immortal_stones = $2, last_mana_update = NOW()
			WHERE id = $3
		`, player.CurrentMana, player.ImmortalStones, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Tiên Nguyên"})
			return
		}

		_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2`, int(player.CurrentMana), player.ID)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
			return
		}
		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID: &authID, Action: "player.recharge",
			RequestSummary: fmt.Sprintf("%d Tiên Nguyên Thạch → %d Tiên Nguyên", stonesToUse, int(manaGain)),
			Metadata: map[string]interface{}{
				"stones_change": -stonesToUse,
				"mana_change":   int(manaGain),
			},
			IPAddress: c.ClientIP(), Success: true,
		})

		respondActionWithInventory(c, http.StatusOK, player, gin.H{
			"message": fmt.Sprintf("Luyện hóa thành công %d Tiên Nguyên Thạch thành %d hạt Tiên Nguyên!", stonesToUse, int(manaGain)),
		})
		return
	}

	if player.CurrentMana >= player.MaxMana {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chân nguyên đã đầy, không cần nạp thêm."})
		return
	}

	// --- BẮT ĐẦU CHỈNH SỬA LOGIC MATCH VỚI TRUYỆN ---
	// Mốc cơ bản: 1 viên nguyên thạch hồi 10% Chân nguyên cho Cổ sư Nhất chuyển Sơ giai (Rank 1, SubRank 1)
	baseManaPerStone := 20.0

	// 1. Hệ số nén của Tiểu cảnh giới (Sơ, Trung, Cao, Đỉnh)
	// Nguyên tác: 2 thành Sơ giai luyện thành 1 thành Trung giai -> Hệ số x2 mỗi tiểu cảnh giới.
	subRankMultiplier := math.Pow(2.0, float64(player.SubRank-1))

	// 2. Hệ số nén của Đại cảnh giới (Nhất chuyển lên Nhị chuyển...)
	// Nguyên tác: Chênh lệch chất lượng chân nguyên giữa các Đại cảnh giới cực lớn, lấy mốc x10 lần.
	rankMultiplier := math.Pow(10.0, float64(player.Rank-1))

	// Lượng % chân nguyên thực tế mà 1 viên nguyên thạch hồi được ở Level hiện tại của Player
	manaPerStone := baseManaPerStone / (rankMultiplier * subRankMultiplier)
	// ------------------------------------------------

	neededMana := player.MaxMana - player.CurrentMana

	// Tính số Nguyên Thạch TỐI ĐA cần dùng để sạc đầy Không khiếu
	maxUsableStones := math.Ceil(neededMana / manaPerStone)

	stonesToUse := int64(req.Stones)
	if float64(stonesToUse) > maxUsableStones {
		stonesToUse = int64(maxUsableStones)
	}

	if float64(stonesToUse) > player.PrimevalStones {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch trong túi."})
		return
	}

	// Tính toán số chân nguyên cộng vào
	addedMana := float64(stonesToUse) * manaPerStone

	// Tránh việc nạp bị lố MaxMana do làm tròn math.Ceil
	if player.CurrentMana+addedMana > player.MaxMana {
		addedMana = player.MaxMana - player.CurrentMana
		player.CurrentMana = player.MaxMana
	} else {
		player.CurrentMana += addedMana
	}

	player.PrimevalStones -= float64(stonesToUse)
	player.LastManaUpdate = time.Now()

	updateQuery := `
		UPDATE players SET current_mana = $1, primeval_stones = $2, last_mana_update = $3
		WHERE id = $4
	`
	_, err = tx.Exec(updateQuery, player.CurrentMana, player.PrimevalStones, player.LastManaUpdate, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save recharge"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID: &authID, Action: "player.recharge",
		RequestSummary: fmt.Sprintf("%d Nguyên Thạch → %.4f%% chân nguyên", stonesToUse, addedMana),
		Metadata: map[string]interface{}{
			"stones_change": -stonesToUse,
			"mana_change":   addedMana,
		},
		IPAddress: c.ClientIP(), Success: true,
	})

	respondActionWithInventory(c, http.StatusOK, player, gin.H{
		"message": fmt.Sprintf("Nạp %d nguyên thạch, khôi phục %.4f%% chân nguyên.", stonesToUse, addedMana),
	})
}
