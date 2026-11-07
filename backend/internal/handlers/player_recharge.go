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

	var stonesToUse int64
	var manaGain float64
	var isImmortal bool
	var addedMana float64

	player, err := MutatePlayer(authID, PlayerMutationOptions{
		Scope:        playerstate.GuNone,
		EventDomains: []PlayerEventDomain{PlayerDomain},
	}, func(tx *sql.Tx, player *models.Player) error {
		// Cổ Tiên (Rank 6+) sử dụng Tiên Nguyên Thạch, không nạp bằng Nguyên Thạch phàm tục
		if player.Rank >= 6 {
			isImmortal = true
			if player.ImmortalStones < float64(req.Stones) {
				return NewMutationError(http.StatusBadRequest, "Đạo hữu không đủ Tiên Nguyên Thạch.")
			}

			stonesToUse = int64(req.Stones)
			manaGain = float64(stonesToUse) / math.Pow(100, float64(player.Rank-6))
			player.CurrentMana += manaGain
			player.ImmortalStones -= float64(stonesToUse)
			if player.CurrentMana > player.MaxMana {
				player.CurrentMana = player.MaxMana
			}
			player.LastManaUpdate = time.Now()

			_, _ = tx.Exec(`UPDATE immortal_apertures SET immortal_essence = $1 WHERE player_id = $2`, int(player.CurrentMana), player.ID)
			return nil
		}

		if player.CurrentMana >= player.MaxMana {
			return NewMutationError(http.StatusBadRequest, "Chân nguyên đã đầy, không cần nạp thêm.")
		}

		baseManaPerStone := 20.0
		subRankMultiplier := math.Pow(2.0, float64(player.SubRank-1))
		rankMultiplier := math.Pow(10.0, float64(player.Rank-1))
		manaPerStone := baseManaPerStone / (rankMultiplier * subRankMultiplier)

		neededMana := player.MaxMana - player.CurrentMana
		maxUsableStones := math.Ceil(neededMana / manaPerStone)

		stonesToUse = int64(req.Stones)
		if float64(stonesToUse) > maxUsableStones {
			stonesToUse = int64(maxUsableStones)
		}

		if float64(stonesToUse) > player.PrimevalStones {
			return NewMutationError(http.StatusBadRequest, "Không đủ Nguyên Thạch trong túi.")
		}

		addedMana = float64(stonesToUse) * manaPerStone
		if player.CurrentMana+addedMana > player.MaxMana {
			addedMana = player.MaxMana - player.CurrentMana
			player.CurrentMana = player.MaxMana
		} else {
			player.CurrentMana += addedMana
		}

		player.PrimevalStones -= float64(stonesToUse)
		player.LastManaUpdate = time.Now()
		return nil
	})
	if err != nil {
		HandleMutationError(c, err)
		return
	}

	if isImmortal {
		auditlog.Log(db.DB, auditlog.Entry{
			PlayerID: &authID, Action: "player.recharge",
			RequestSummary: fmt.Sprintf("%d Tiên Nguyên Thạch → %d Tiên Nguyên", stonesToUse, int(manaGain)),
			Metadata: map[string]interface{}{
				"stones_change": -stonesToUse,
				"mana_change":   int(manaGain),
			},
			IPAddress: c.ClientIP(), Success: true,
		})

		respondActionWithInventory(c, http.StatusOK, *player, gin.H{
			"message": fmt.Sprintf("Luyện hóa thành công %d Tiên Nguyên Thạch thành %d hạt Tiên Nguyên!", stonesToUse, int(manaGain)),
		})
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

	respondActionWithInventory(c, http.StatusOK, *player, gin.H{
		"message": fmt.Sprintf("Nạp %d nguyên thạch, khôi phục %.4f%% chân nguyên.", stonesToUse, addedMana),
	})
}
