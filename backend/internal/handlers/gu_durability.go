package handlers

import (
	"database/sql"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
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

type GuRestorationRequest struct {
	InstanceID string `json:"instance_id" binding:"required"`
}

type NourishGuRequest = GuRestorationRequest
type RepairGuRequest = GuRestorationRequest

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

func handleGuRestoration(c *gin.Context, isRepair bool) {
	var req GuRestorationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Chưa đăng nhập"})
		return
	}

	var targetGu *models.GuInstance
	var manaCost float64
	var errStatus int
	var errMsg string

	updatedPlayer, err := MutatePlayer(authID, PlayerMutationOptions{
		Scope:        playerstate.GuEquipped,
		EventDomains: []PlayerEventDomain{PlayerDomain, GuInventoryDomain},
	}, func(tx *sql.Tx, player *models.Player) error {
		var loadErr error
		targetGu, loadErr = db.LoadPlayerGuForUpdate(tx, authID.String(), req.InstanceID)
		if loadErr == sql.ErrNoRows {
			errStatus = http.StatusNotFound
			errMsg = "Không tìm thấy Cổ trùng này trong Không Khiếu"
			return loadErr
		}
		if loadErr != nil {
			errStatus = http.StatusInternalServerError
			errMsg = "Không thể tải Cổ trùng cần xử lý"
			return loadErr
		}

		if targetGu.Status == "DEAD" {
			errStatus = http.StatusBadRequest
			if isRepair {
				errMsg = "Cổ trùng đã chết, không thể tu bổ"
			} else {
				errMsg = "Cổ trùng đã chết, không thể ôn dưỡng"
			}
			return fmt.Errorf("%s", errMsg)
		}

		if isRepair {
			if targetGu.Durability >= 20 && targetGu.Status != "DAMAGED" {
				errStatus = http.StatusBadRequest
				errMsg = "Cổ trùng chưa bị tổn hại nghiêm trọng (độ bền >= 20), hãy Ôn Dưỡng bằng Chân Nguyên"
				return fmt.Errorf("%s", errMsg)
			}
			manaCost = durabilityManaCost(player.Rank, targetGu.Rank, 5)
		} else {
			if targetGu.Status == "DAMAGED" || targetGu.Durability < 20 {
				errStatus = http.StatusBadRequest
				errMsg = "Cổ trùng bị trọng thương hư hại (độ bền < 20), không thể ôn dưỡng bằng Chân Nguyên mà cần Tu Bổ"
				return fmt.Errorf("%s", errMsg)
			}
			if targetGu.Durability >= 100 {
				errStatus = http.StatusBadRequest
				errMsg = "Cổ trùng đã ở trạng thái hoàn hảo (độ bền 100%), không cần ôn dưỡng"
				return fmt.Errorf("%s", errMsg)
			}
			manaCost = durabilityManaCost(player.Rank, targetGu.Rank, 10)
		}

		if player.CurrentMana < manaCost {
			errStatus = http.StatusBadRequest
			actionName := "ôn dưỡng"
			if isRepair {
				actionName = "tu bổ"
			}
			errMsg = fmt.Sprintf("Không đủ Chân Nguyên để %s (cần %.1f, hiện có %.1f)", actionName, manaCost, player.CurrentMana)
			return fmt.Errorf("%s", errMsg)
		}

		player.CurrentMana -= manaCost
		player.LastManaUpdate = time.Now()

		if isRepair {
			targetGu.Durability = 50.0
			targetGu.Status = "ACTIVE"
		} else {
			targetGu.Durability = math.Min(100.0, targetGu.Durability+15.0)
		}

		if updateErr := db.UpdatePlayerGu(tx, *targetGu); updateErr != nil {
			errStatus = http.StatusInternalServerError
			errMsg = "Không thể cập nhật trạng thái Cổ trùng"
			return updateErr
		}

		return nil
	})

	if err != nil {
		if errStatus == 0 {
			errStatus = http.StatusInternalServerError
		}
		if errMsg == "" {
			errMsg = err.Error()
		}
		c.JSON(errStatus, gin.H{"error": errMsg})
		return
	}

	if isRepair {
		c.JSON(http.StatusOK, gin.H{
			"message":        "Tu bổ Cổ trùng thành công, đã khôi phục nguyên khí căn bản!",
			"instance_id":    targetGu.InstanceID,
			"durability":     targetGu.Durability,
			"status":         targetGu.Status,
			"mana_spent":     manaCost,
			"remaining_mana": updatedPlayer.CurrentMana,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"message":        fmt.Sprintf("Ôn dưỡng thành công! Độ bền tăng lên %.1f/100", targetGu.Durability),
			"instance_id":    targetGu.InstanceID,
			"durability":     targetGu.Durability,
			"mana_spent":     manaCost,
			"remaining_mana": updatedPlayer.CurrentMana,
		})
	}
}

// NourishGu actively channels player mana to nourish an injured Gu (durability >= 20).
// Consumes 10 mana at equal rank 5, scaled by Gu rank power / player rank power.
// Restores +15.0 durability (max 100.0).
func NourishGu(c *gin.Context) {
	handleGuRestoration(c, false)
}

// RepairGu restores heavily damaged Gus (durability < 20 or state == 'DAMAGED') using mana.
// Sets durability back to 50.0 and state to 'ACTIVE'.
func RepairGu(c *gin.Context) {
	handleGuRestoration(c, true)
}
