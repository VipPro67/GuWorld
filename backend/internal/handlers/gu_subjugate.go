package handlers

import (
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// EvaluateWildWillResurgence decrements refinement_progress of incomplete Gus
// that have not been touched for at least 1 hour, at a rate of 5.0% per hour.
func EvaluateWildWillResurgence(q db.QueryExecer, playerID string) error {
	_, err := q.Exec(`
		UPDATE player_gu
		SET refinement_progress = GREATEST(0, refinement_progress - (EXTRACT(EPOCH FROM (NOW() - updated_at)) / 3600.0 * 5.0)),
		    updated_at = NOW()
		WHERE player_id = $1::uuid
		  AND refinement_progress > 0
		  AND refinement_progress < 100
		  AND updated_at < NOW() - INTERVAL '1 hour'`, playerID)
	return err
}

type SubjugateGuRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
}

// SubjugateGu refines / subjugates the wild will in a Gu bug.
// If the player's Vital Gu rank is >= target Gu rank + 2, it achieves instant 100% refinement for 0 mana.
// Otherwise, it expends mana (targetRank * 25) to grant +25% refinement progress per step.
func SubjugateGu(c *gin.Context) {
	var req SubjugateGuRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa đăng nhập"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kết nối cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	// Lazy evaluate wild will decay for incomplete Gus before processing
	_ = EvaluateWildWillResurgence(tx, authID.String())

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể nạp thông tin Không Khiếu"})
		return
	}

	targetIdx := -1
	for i, gu := range player.GuInventory {
		if gu.InstanceID == req.InstanceID {
			targetIdx = i
			break
		}
	}
	if targetIdx == -1 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng trong Không Khiếu"})
		return
	}

	targetGu := &player.GuInventory[targetIdx]
	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể luyện hóa Cổ trùng đã chết"})
		return
	}

	if targetGu.RefinementProgress >= 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng này đã được luyện hóa hoàn tất (100%)."})
		return
	}

	// Check Vital Gu suppression rule:
	// If Vital Gu rank >= target Gu rank + 2, the wild will submits instantly.
	var vitalGu *models.GuInstance
	for i := range player.GuInventory {
		if player.GuInventory[i].IsVital {
			vitalGu = &player.GuInventory[i]
			break
		}
	}

	isSuppressed := vitalGu != nil && vitalGu.Rank >= targetGu.Rank+2
	var msg string

	if isSuppressed {
		targetGu.RefinementProgress = 100.0
		vitalName := vitalGu.GuID
		if spec, ok := models.GetGu(vitalGu.GuID, vitalGu.Rank); ok && spec.Name != "" {
			vitalName = spec.Name
		}
		targetName := targetGu.GuID
		if spec, ok := models.GetGu(targetGu.GuID, targetGu.Rank); ok && spec.Name != "" {
			targetName = spec.Name
		}
		msg = fmt.Sprintf("Khí tức uy nghiêm của Bản Mệnh Cổ [%s - Rank %d] đã hoàn toàn áp chế ý chí của [%s]! Luyện hóa tức khắc thành công (100%%).",
			vitalName, vitalGu.Rank, targetName)
	} else {
		// Normal refinement: 25 mana per target rank per 25% step
		manaCost := float64(targetGu.Rank * 25)
		if player.CurrentMana < manaCost {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Chân Nguyên không đủ để luyện hóa. Yêu cầu %.1f Chân Nguyên, hiện có %.1f.", manaCost, player.CurrentMana),
			})
			return
		}

		player.CurrentMana -= manaCost
		targetGu.RefinementProgress = math.Min(100.0, targetGu.RefinementProgress+25.0)

		targetName := targetGu.GuID
		if spec, ok := models.GetGu(targetGu.GuID, targetGu.Rank); ok && spec.Name != "" {
			targetName = spec.Name
		}

		if targetGu.RefinementProgress >= 100.0 {
			msg = fmt.Sprintf("Luyện hóa [%s] thành công! Ý chí Cổ trùng đã hoàn toàn thần phục (100%%).", targetName)
		} else {
			msg = fmt.Sprintf("Đã truyền %.0f Chân Nguyên thanh tẩy ý chí [%s]. Tiến độ luyện hóa hiện tại: %.1f%%.",
				manaCost, targetName, targetGu.RefinementProgress)
		}

		if _, err := tx.Exec(`UPDATE players SET current_mana = $1, last_mana_update = NOW() WHERE id = $2`, player.CurrentMana, player.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật Chân Nguyên"})
			return
		}
	}

	if _, err := tx.Exec(`UPDATE player_gu SET refinement_progress = $1, updated_at = NOW() WHERE id = $2`, targetGu.RefinementProgress, targetGu.InstanceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tiến độ luyện hóa"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất giao dịch"})
		return
	}

	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated)

	c.JSON(http.StatusOK, gin.H{
		"message":             msg,
		"refinement_progress": targetGu.RefinementProgress,
		"current_mana":        player.CurrentMana,
		"is_suppressed":       isSuppressed,
		"is_completed":        targetGu.RefinementProgress >= 100,
	})
}
