package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// EvaluateGuDurabilityHealing passively restores durability for Gus with durability >= 20 and < 100,
// burning hunger at a rate of 2.5 hunger per 1.0 durability (2.0 durability / hr -> 5 hunger / hr).
// Lazy-evaluated when inventory is accessed.
func EvaluateGuDurabilityHealing(q db.QueryExecer, playerID string) error {
	_, err := q.Exec(`
		WITH candidates AS (
			SELECT id,
			       durability,
			       hunger,
			       LEAST(
			           LEAST(100.0 - durability, (EXTRACT(EPOCH FROM (NOW() - updated_at)) / 3600.0) * 2.0),
			           hunger / 2.5
			       ) AS actual_gain
			FROM player_gu
			WHERE player_id = $1::uuid
			  AND durability >= 20
			  AND durability < 100
			  AND hunger > 0
			  AND state NOT IN ('DEAD', 'DAMAGED')
			  AND updated_at < NOW() - INTERVAL '1 hour'
		)
		UPDATE player_gu g
		SET durability = LEAST(100.0, g.durability + c.actual_gain),
		    hunger = GREATEST(0, g.hunger - (c.actual_gain * 2.5)),
		    updated_at = NOW()
		FROM candidates c
		WHERE g.id = c.id AND c.actual_gain > 0
	`, playerID)
	return err
}

type NourishGuRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
}

var durabilityManaRankWeights = [...]float64{1, 2, 5, 20, 100, 1_000, 10_000, 100_000, 1_000_000}

func durabilityManaCost(playerRank, guRank int, baseCost float64) float64 {
	if playerRank < 1 {
		playerRank = 1
	} else if playerRank > len(durabilityManaRankWeights) {
		playerRank = len(durabilityManaRankWeights)
	}
	if guRank < 1 {
		guRank = 1
	} else if guRank > len(durabilityManaRankWeights) {
		guRank = len(durabilityManaRankWeights)
	}
	return baseCost * durabilityManaRankWeights[guRank-1] / durabilityManaRankWeights[playerRank-1]
}

// NourishGu actively channels player mana to nourish an injured Gu (durability >= 20).
// Consumes 10 mana at equal rank 5, scaled by Gu rank power / player rank power.
// Restores +15.0 durability (max 100.0).
func NourishGu(c *gin.Context) {
	var req NourishGuRequest
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

	_ = EvaluateGuDurabilityHealing(tx, authID.String())

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể nạp thông tin Không Khiếu"})
		return
	}

	targetGu, err := db.LoadPlayerGuForUpdate(tx, authID.String(), req.InstanceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này trong Không Khiếu"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải Cổ trùng cần ôn dưỡng"})
		return
	}

	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đã chết, không thể ôn dưỡng"})
		return
	}
	if targetGu.Status == "DAMAGED" || targetGu.Durability < 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng bị trọng thương hư hại (độ bền < 20), không thể ôn dưỡng bằng Chân Nguyên mà cần Tu Bổ"})
		return
	}
	if targetGu.Durability >= 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đã ở trạng thái hoàn hảo (độ bền 100%), không cần ôn dưỡng"})
		return
	}

	manaCost := durabilityManaCost(player.Rank, targetGu.Rank, 10)
	if player.CurrentMana < manaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Chân Nguyên để Ôn Dưỡng (cần %.0f, hiện có %.1f)", manaCost, player.CurrentMana)})
		return
	}

	player.CurrentMana -= manaCost
	newDurability := math.Min(100.0, targetGu.Durability+15.0)
	targetGu.Durability = newDurability

	if err := db.UpdatePlayerGu(tx, *targetGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật độ bền Cổ trùng"})
		return
	}

	if _, err := tx.Exec(`UPDATE players SET current_mana = $1, last_mana_update = NOW() WHERE id = $2`, player.CurrentMana, player.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật Chân Nguyên"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu dữ liệu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        fmt.Sprintf("Ôn dưỡng thành công! Độ bền tăng lên %.1f/100", newDurability),
		"instance_id":    targetGu.InstanceID,
		"durability":     newDurability,
		"mana_spent":     manaCost,
		"remaining_mana": player.CurrentMana,
	})
}

type RepairGuRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
}

// RepairGu restores heavily damaged Gus (durability < 20 or state == 'DAMAGED') using mana.
// Sets durability back to 50.0 and state to 'ACTIVE'.
func RepairGu(c *gin.Context) {
	var req RepairGuRequest
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

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể nạp thông tin Chân Nguyên"})
		return
	}

	targetGu, err := db.LoadPlayerGuForUpdate(tx, authID.String(), req.InstanceID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này trong Không Khiếu"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải Cổ trùng cần tu bổ"})
		return
	}

	if targetGu.Status == "DEAD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng đã chết, không thể tu bổ"})
		return
	}
	if targetGu.Durability >= 20 && targetGu.Status != "DAMAGED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cổ trùng chưa bị tổn hại nghiêm trọng (độ bền >= 20), hãy Ôn Dưỡng bằng Chân Nguyên"})
		return
	}

	manaCost := durabilityManaCost(player.Rank, targetGu.Rank, 5)
	if player.CurrentMana < manaCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Chân Nguyên để tu bổ (cần %.2f, hiện có %.2f)", manaCost, player.CurrentMana)})
		return
	}
	player.CurrentMana -= manaCost
	if _, err := tx.Exec(`UPDATE players SET current_mana = $1, last_mana_update = NOW() WHERE id = $2`, player.CurrentMana, player.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trừ Chân Nguyên để tu bổ"})
		return
	}

	targetGu.Durability = 50.0
	targetGu.Status = "ACTIVE"

	if err := db.UpdatePlayerGu(tx, *targetGu); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trạng thái Cổ trùng"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu dữ liệu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Tu bổ Cổ trùng thành công, đã khôi phục nguyên khí căn bản!",
		"instance_id":    targetGu.InstanceID,
		"durability":     50.0,
		"status":         "ACTIVE",
		"mana_spent":     manaCost,
		"remaining_mana": player.CurrentMana,
	})
}
