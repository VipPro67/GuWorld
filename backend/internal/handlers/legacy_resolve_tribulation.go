package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func ResolveTribulation(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	eventIDStr := c.Param("eventId")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã đợt tai kiếp không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// 1. Fetch active event
	var ev models.TribulationEvent
	err = tx.QueryRow(`
		SELECT id, aperture_id, player_id, tribulation_type, path, reward_dao_marks, started_at
		FROM tribulation_events
		WHERE id = $1 AND player_id = $2 AND status = 'ACTIVE'
	`, eventID, playerID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.RewardDaoMarks, &ev.StartedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Đợt tai kiếp này đã được giải quyết hoặc không tồn tại"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn tai kiếp"})
		return
	}

	// 2. Check if all enemies are dead
	var remainingEnemies int
	err = tx.QueryRow(`
		SELECT COUNT(1) FROM tribulation_enemies WHERE event_id = $1 AND is_dead = false
	`, ev.ID).Scan(&remainingEnemies)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra số quái còn sống"})
		return
	}

	var ap models.ImmortalAperture
	err = tx.QueryRow(`
		SELECT id, aperture_hp, max_aperture_hp, next_tribulation_year,
			rank_start_year, earth_calamity_count, heavenly_tribulation_count, grand_tribulation_count, myriad_tribulation_count, chaos_calamity_count, internal_years_passed
		FROM immortal_apertures WHERE id = $1
	`, ev.ApertureID).Scan(&ap.ID, &ap.ApertureHP, &ap.MaxApertureHP, &ap.NextTribulationYear,
		&ap.RankStartYear, &ap.EarthCalamityCount, &ap.HeavenlyTribulationCount, &ap.GrandTribulationCount, &ap.MyriadTribulationCount, &ap.ChaosCalamityCount, &ap.InternalYearsPassed)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy thông tin Tiên Khiếu"})
		return
	}

	var msg string
	now := time.Now()
	didBreakthrough := false
	breakthroughMsg := ""
	updatedDaoMarks := make(models.DaoMarksMap)

	// Helper: recalc next_year, check breakthrough
	handleSuccess := func(tribType string) {
		switch tribType {
		case "EARTH_CALAMITY":
			ap.EarthCalamityCount++
		case "HEAVENLY_TRIBULATION":
			ap.HeavenlyTribulationCount++
		case "GRAND_TRIBULATION":
			ap.GrandTribulationCount++
		case "MYRIAD_TRIBULATION":
			ap.MyriadTribulationCount++
		case "CHAOS_CALAMITY":
			ap.ChaosCalamityCount++
		case "CHAOTIC_TRIBULATION":
			ap.ChaosCalamityCount++
		}

		// Recalculate next tribulation year based on rank & counters
		var playerRank int
		var username string
		_ = tx.QueryRow(`SELECT rank, username FROM players WHERE id = $1`, playerID).Scan(&playerRank, &username)

		nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.RankStartYear, ap.InternalYearsPassed)
		ap.NextTribulationYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)

		// Check breakthrough conditions
		canBreak, newRank := models.CheckImmortalBreakthrough(playerRank, ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount)
		if canBreak {
			// Rank-up requirement: player must own at least one path (lưu phái)
			// at the required Cảnh Giới — Rank 7 = Tông Sư, Rank 8 = Đại Tông Sư,
			// Rank 9 = Vô Thượng Đại Tông Sư.
			var pathMasteries models.PathMasteriesMap
			_ = tx.QueryRow(`SELECT COALESCE(path_masteries, '{}'::jsonb) FROM players WHERE id = $1`, playerID).Scan(&pathMasteries)
			reqLevel := gamefx.PathMasteryLevelRequiredForRank(newRank)
			if !gamefx.HasPathMasteryAtLeast(pathMasteries, reqLevel) {
				blockedMsg := fmt.Sprintf(" Chưa đủ điều kiện đột phá lên %s: cần ít nhất 1 lưu phái đạt %s (%d điểm).", models.GetRankName(newRank), gamefx.PathMasteryLevelName(reqLevel), gamefx.PathMasteryPointsForLevel(reqLevel))
				_, _ = tx.Exec(`INSERT INTO messages (sender_id, receiver_id, content, channel) VALUES ($1, $1, $2, 'system')`, playerID, "Đột phá bị chặn: "+blockedMsg)
				breakthroughMsg = blockedMsg
			} else {
				_, _ = tx.Exec(`UPDATE players SET rank = $1 WHERE id = $2`, newRank, playerID)
				ap.RankStartYear = math.Floor(ap.InternalYearsPassed)
				// Reset per-type counters: each rank has its own tribulation set
				// and cadence (lower types drop off), so the schedule restarts.
				ap.EarthCalamityCount = 0
				ap.HeavenlyTribulationCount = 0
				ap.GrandTribulationCount = 0
				ap.MyriadTribulationCount = 0
				ap.ChaosCalamityCount = 0
				didBreakthrough = true
				breakthroughMsg = fmt.Sprintf(" Đột phá! Đạo hữu đã tiến vào %s!", models.GetRankName(newRank))

				if newRank == 7 || newRank == 8 || newRank == 9 {
					_ = SendSystemMessage("world", sql.NullString{}, fmt.Sprintf("Chúc mừng đạo hữu %s đã đột phá thành công lên %s!", username, models.GetRankName(newRank)))
					auditlog.Log(tx, auditlog.Entry{
						PlayerID:       &playerID,
						Action:         "player.breakthrough_success",
						EntityType:     "player",
						EntityID:       playerID.String(),
						RequestSummary: fmt.Sprintf("Thăng rank %d (%s) thành công", newRank, models.GetRankName(newRank)),
						Success:        true,
					})
				}

				// Recalculate next tribulation year based on the new rank!
				nextYear, _ = models.GetNextTribulationInfo(newRank, ap.RankStartYear, ap.InternalYearsPassed)
				ap.NextTribulationYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)
			}
		}

		// Update current event status and dead enemies to evaluate stability correctly
		_, _ = tx.Exec(`UPDATE tribulation_events SET status = 'SUCCESS', aperture_hp_after = $1, ended_at = $2 WHERE id = $3`, ap.ApertureHP, now, ev.ID)
		_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

		hasActive, errActive := db.HasActiveTribulationOrEnemies(tx, ap.ID)
		if errActive == nil && !hasActive {
			ap.State = "STABLE"
		} else {
			ap.State = "TRIBULATION_ACTIVE"
		}
	}

	if remainingEnemies == 0 {
		// Clear Victory! All enemies defeated = guaranteed success
		rewards := distributeDaoMarks(ev.Path, ev.RewardDaoMarks)
		daoMap, errDao := addMultipleDaoMarksTx(tx, playerID.String(), rewards)
		if errDao != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cộng Đạo Ngân: " + errDao.Error()})
			return
		}
		updatedDaoMarks = daoMap

		ap.ApertureHP = ap.MaxApertureHP
		handleSuccess(ev.TribulationType)

		_, err = tx.Exec(`
			UPDATE immortal_apertures 
			SET state = $1, next_tribulation_year = $2, aperture_hp = $3,
				rank_start_year = $4, earth_calamity_count = $5, heavenly_tribulation_count = $6,
				grand_tribulation_count = $7, myriad_tribulation_count = $8, chaos_calamity_count = $9
			WHERE id = $10
		`, ap.State, ap.NextTribulationYear, ap.ApertureHP,
			ap.RankStartYear, ap.EarthCalamityCount, ap.HeavenlyTribulationCount,
			ap.GrandTribulationCount, ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Tiên Khiếu"})
			return
		}

		rewardStrings := []string{}
		for path, amount := range rewards {
			rewardStrings = append(rewardStrings, fmt.Sprintf("%d Đạo Ngân hệ %s", amount, strings.ToUpper(path)))
		}
		rewardText := strings.Join(rewardStrings, ", ")
		msg = fmt.Sprintf("Chúc mừng đạo hữu vượt kiếp nạn thành công! Khai quang nhận %s. Tiên Khiếu của bạn đã ổn định trở lại.%s", rewardText, breakthroughMsg)

		if didBreakthrough {
			_, _ = tx.Exec(`
				INSERT INTO messages (sender_id, receiver_id, content, channel)
				VALUES ('11111111-1111-1111-1111-111111111111', $1, $2, 'private')
			`, playerID, breakthroughMsg)
		}
	} else {
		// Player resolves manually with remaining enemies (rút lui) => ALWAYS FAIL
		var playerRank int
		_ = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&playerRank)

		// Counts are only incremented on success — a failed tribulation does
		// not count toward the total / per-type counters, but the next year
		// is still pushed forward to the next milestone for recovery time.
		nextYear, _ := models.GetNextTribulationInfo(playerRank, ap.RankStartYear, ap.InternalYearsPassed)
		ap.NextTribulationYear = guardNextTribulationYear(nextYear, ap.InternalYearsPassed)

		ap.State = "DAMAGED"
		ap.ApertureHP = ap.MaxApertureHP / 10

		// Phá hủy 30% tài nguyên + phạt Nguyên Thạch/Tiên Nguyên.
		applyTribulationFailurePenalty(tx, ap.ID, playerID.String())

		// Giảm diện tích và tốc độ thời gian
		_, _ = tx.Exec(`
			UPDATE immortal_apertures 
			SET state = $1, next_tribulation_year = $2, aperture_hp = $3,
				earth_calamity_count = $4, heavenly_tribulation_count = $5, grand_tribulation_count = $6,
				myriad_tribulation_count = $7, chaos_calamity_count = $8,
				area_size = GREATEST(0, CAST(area_size AS numeric) * 0.9),
				time_rate = GREATEST(1.0, time_rate * 0.9)
			WHERE id = $9
		`, ap.State, ap.NextTribulationYear, ap.ApertureHP,
			ap.EarthCalamityCount, ap.HeavenlyTribulationCount, ap.GrandTribulationCount,
			ap.MyriadTribulationCount, ap.ChaosCalamityCount, ap.ID)

		_, _ = tx.Exec(`
			UPDATE tribulation_events
			SET status = 'FAILED', aperture_hp_after = $1, ended_at = $2, reward_dao_marks = 0
			WHERE id = $3
		`, ap.ApertureHP, now, ev.ID)

		msg = "Đạo hữu rút lui khi kiếp nạn chưa dứt! Thiên Ý cuồng bạo tàn phá Tiên Khiếu, kiếp nạn bị phán định THẤT BẠI: Nguyên khí hư hao, bạn mất 2,000 Nguyên Thạch, 5 Tiên Nguyên, diện tích Tiên Khiếu và tốc độ thời gian bị giảm 10%, 30% điểm tài nguyên bị san phẳng."
	}

	// Clean up dead/remaining enemies
	_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 10, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật Nhân Khí"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Check if the player is immediately due for another tribulation
	_ = checkAndTriggerSingleAperture(ev.ApertureID)

	var finalState string
	_ = db.DB.QueryRow(`SELECT state FROM immortal_apertures WHERE id = $1`, ev.ApertureID).Scan(&finalState)

	var finalDaoMarks models.DaoMarksMap
	if len(updatedDaoMarks) > 0 {
		finalDaoMarks = updatedDaoMarks
	} else {
		var finalDaoJSON []byte
		if err2 := db.DB.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&finalDaoJSON); err2 == nil {
			json.Unmarshal(finalDaoJSON, &finalDaoMarks)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   msg + " Nhân Khí +10.",
		"state":     finalState,
		"dao_marks": finalDaoMarks,
	})
}
