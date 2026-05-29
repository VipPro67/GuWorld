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
)

const maxCultivationRank = 9
const peakSubRank = 4

type TemperApertureRequest struct {
	EnergyAmount float64 `json:"energy_amount" binding:"required,gt=0"`
}

type TemperApertureResponse struct {
	Player       models.Player `json:"player"`
	ProgressGain float64       `json:"progress_gain"`
	Threshold    float64       `json:"threshold"`
	LeveledUp    bool          `json:"leveled_up"`
	Ascended     bool          `json:"ascended"`
	Message      string        `json:"message"`
}

func cultivationThreshold(player *models.Player) float64 {
	if isImmortalCultivator(player) {
		baseByRank := map[int]float64{6: 800, 7: 1200, 8: 1800, 9: 2600}
		base := baseByRank[player.Rank]
		if base == 0 {
			base = 800 + float64(player.Rank-6)*500
		}
		return base * (1 + float64(player.SubRank-1)*0.35)
	}

	baseByRank := map[int]float64{1: 100, 2: 160, 3: 240, 4: 360, 5: 500}
	base := baseByRank[player.Rank]
	if base == 0 {
		base = 100
	}
	return base * (1 + float64(player.SubRank-1)*0.25)
}

func cultivationProgressGain(player *models.Player, energyAmount float64) float64 {
	if isImmortalCultivator(player) {
		return energyAmount * (1.0 + float64(player.SubRank-1)*0.08)
	}

	aptitudeMultiplier := 0.75 + float64(player.AptitudePercent)/100.0
	return energyAmount * aptitudeMultiplier
}

func advanceCultivation(player *models.Player) (leveledUp bool, ascended bool, message string) {
	oldRank := player.Rank
	oldSubRank := player.SubRank

	if player.Rank >= maxCultivationRank && player.SubRank >= peakSubRank {
		player.CultivationProgress = cultivationThreshold(player)
		return false, false, "Cửu chuyển đỉnh phong đã chạm giới hạn hiện tại của thiên địa."
	}

	if player.SubRank < peakSubRank {
		player.SubRank++
	} else {
		player.Rank++
		player.SubRank = 1
	}

	leveledUp = true
	player.CultivationProgress = 0

	if oldRank == 5 && oldSubRank == peakSubRank && player.Rank == 6 {
		ascended = true
		player.MaxMana = immortalApertureCapacity(player.Rank, player.SubRank)
		player.CurrentMana = math.Max(player.CurrentMana, player.MaxMana*0.25)
		if player.DaoMarks == nil {
			player.DaoMarks = models.DaoMarksMap{}
		}
		player.DaoMarks["heaven"] += 10
		player.DaoMarks["human"] += 10
		message = "Độ kiếp thành công! Không Khiếu hóa thành Tiên Khiếu, Chân Nguyên chuyển thành Tiên Nguyên. Đạo hữu đã bước vào Lục Chuyển Cổ Tiên."
		return leveledUp, ascended, message
	}

	if isImmortalCultivator(player) {
		capacity := immortalApertureCapacity(player.Rank, player.SubRank)
		if player.MaxMana < capacity {
			player.MaxMana = capacity
		}
		message = fmt.Sprintf("Tiên Khiếu mở rộng! Tu vi tăng từ Rank %d.%d lên Rank %d.%d.", oldRank, oldSubRank, player.Rank, player.SubRank)
	} else {
		message = fmt.Sprintf("Ôn dưỡng Không Khiếu viên mãn! Tu vi tăng từ Rank %d.%d lên Rank %d.%d.", oldRank, oldSubRank, player.Rank, player.SubRank)
	}
	return leveledUp, ascended, message
}

func TemperAperture(c *gin.Context) {
	var req TemperApertureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
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
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Player not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player Aperture"})
		}
		return
	}

	if player.ActiveCombat != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đang chiến đấu, không thể tĩnh tọa ôn dưỡng."})
		return
	}
	if player.Rank >= maxCultivationRank && player.SubRank >= peakSubRank {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã đạt Cửu chuyển đỉnh phong, chưa thể tiếp tục đột phá."})
		return
	}
	if req.EnergyAmount > player.CurrentMana {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}

	threshold := cultivationThreshold(&player)
	progressGain := cultivationProgressGain(&player, req.EnergyAmount)
	player.CurrentMana -= req.EnergyAmount
	player.CultivationProgress += progressGain
	player.LastManaUpdate = time.Now()

	leveledUp := false
	ascended := false
	message := fmt.Sprintf("Đạo hữu tiêu hao %s ôn dưỡng %s, tiến độ tăng %.1f/%.1f.", describeEnergyCost(&player, req.EnergyAmount), func() string {
		if isImmortalCultivator(&player) {
			return "Tiên Khiếu"
		}
		return "Không Khiếu"
	}(), progressGain, threshold)

	if player.CultivationProgress >= threshold {
		leveledUp, ascended, message = advanceCultivation(&player)
		threshold = cultivationThreshold(&player)
	}

	_, err = tx.Exec(`
		UPDATE players
		SET rank = $1, sub_rank = $2, current_mana = $3, max_mana = $4, cultivation_progress = $5, dao_marks = $6, last_mana_update = $7
		WHERE id = $8
	`, player.Rank, player.SubRank, player.CurrentMana, player.MaxMana, player.CultivationProgress, player.DaoMarks, player.LastManaUpdate, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save cultivation progress"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, TemperApertureResponse{
		Player:       player,
		ProgressGain: progressGain,
		Threshold:    threshold,
		LeveledUp:    leveledUp,
		Ascended:     ascended,
		Message:      message,
	})
}
