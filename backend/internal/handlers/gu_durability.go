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

// NourishGu actively channels player mana to nourish an injured Gu (durability >= 20).
// Consumes mana = targetRank * 20, restores +15.0 durability (max 100.0).
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

	guInv, err := db.LoadPlayerGus(tx, authID.String(), true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách Cổ trùng"})
		return
	}

	var targetGu *models.GuInstance
	for i := range guInv {
		if guInv[i].InstanceID == req.InstanceID {
			targetGu = &guInv[i]
			break
		}
	}
	if targetGu == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này trong Không Khiếu"})
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

	manaCost := float64(targetGu.Rank * 20)
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

	if _, err := tx.Exec(`UPDATE players SET current_mana = $1, last_mana_tick = NOW() WHERE id = $2`, player.CurrentMana, player.ID); err != nil {
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

// RepairGu restores heavily damaged Gus (durability < 20 or state == 'DAMAGED') using Primeval Stones / Immortal Stones.
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

	var primevalStones, immortalStones float64
	if err := tx.QueryRow(`SELECT primeval_stones, immortal_stones FROM players WHERE id = $1 FOR UPDATE`, authID).Scan(&primevalStones, &immortalStones); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể nạp thông tin người chơi"})
		return
	}

	guInv, err := db.LoadPlayerGus(tx, authID.String(), true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách Cổ trùng"})
		return
	}

	var targetGu *models.GuInstance
	for i := range guInv {
		if guInv[i].InstanceID == req.InstanceID {
			targetGu = &guInv[i]
			break
		}
	}
	if targetGu == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Cổ trùng này trong Không Khiếu"})
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

	if targetGu.Rank >= 6 {
		cost := 1.0
		if immortalStones < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên Thạch để tu bổ Tiên Cổ (cần %.0f)", cost)})
			return
		}
		immortalStones -= cost
		if _, err := tx.Exec(`UPDATE players SET immortal_stones = $1 WHERE id = $2`, immortalStones, authID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trừ Tiên Nguyên Thạch"})
			return
		}
	} else {
		cost := float64(targetGu.Rank * 50)
		if primevalStones < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch để tu bổ (cần %.0f)", cost)})
			return
		}
		primevalStones -= cost
		if _, err := tx.Exec(`UPDATE players SET primeval_stones = $1 WHERE id = $2`, primevalStones, authID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trừ Nguyên Thạch"})
			return
		}
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
		"message":     "Tu bổ Cổ trùng thành công, đã khôi phục nguyên khí căn bản!",
		"instance_id": targetGu.InstanceID,
		"durability":  50.0,
		"status":      "ACTIVE",
	})
}
