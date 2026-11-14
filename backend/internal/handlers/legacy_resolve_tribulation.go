package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
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
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	// 1. Lock the event before settlement. Recovery after reconnect and the
	// normal combat completion callback can race; serializing on this row
	// ensures only one transaction can award rewards or apply penalties.
	var ev models.TribulationEvent
	err = tx.QueryRow(`
		SELECT id, aperture_id, player_id, tribulation_type, path, reward_dao_marks, started_at, status, COALESCE(stage, 'ORDEAL_ACTIVE')
		FROM tribulation_events
		WHERE id = $1 AND player_id = $2
		FOR UPDATE
	`, eventID, playerID).Scan(&ev.ID, &ev.ApertureID, &ev.PlayerID, &ev.TribulationType, &ev.Path, &ev.RewardDaoMarks, &ev.StartedAt, &ev.Status, &ev.Stage)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Đợt tai kiếp này đã được giải quyết hoặc không tồn tại"})
		return
	} else if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi truy vấn tai kiếp", err))
		return
	}
	if ev.Status != "ACTIVE" {
		if err := tx.Commit(); err != nil {
			httperr.Respond(c, httperr.Internal("Transaction commit failed", err))
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Tai kiếp này đã được kết toán trước đó.", "state": ev.Status})
		return
	}

	// If already in AFTERMATH stage, direct resolution finalizes the aftermath phase
	if ev.Stage == "AFTERMATH" {
		res, err := finalizeTribulationSettlementTx(tx, eventID, playerID)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi kết toán hậu kiếp nạn: "+err.Error(), err))
			return
		}

		if err := tx.Commit(); err != nil {
			httperr.Respond(c, httperr.Internal("Transaction commit failed", err))
			return
		}

		_ = checkAndTriggerSingleAperture(ev.ApertureID)

		c.JSON(http.StatusOK, gin.H{
			"message":   res.Message,
			"state":     res.State,
			"dao_marks": res.UpdatedDaoMarks,
			"stage":     "RESOLVED",
			"status":    "SUCCESS",
		})
		return
	}

	// 2. Check if all enemies are dead
	var remainingEnemies int
	err = tx.QueryRow(`
		SELECT COUNT(1) FROM tribulation_enemies WHERE event_id = $1 AND is_dead = false
	`, ev.ID).Scan(&remainingEnemies)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi kiểm tra số quái còn sống", err))
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
		httperr.Respond(c, httperr.Internal("Lỗi lấy thông tin Tiên Khiếu", err))
		return
	}

	var playerRank int
	_ = tx.QueryRow(`SELECT rank FROM players WHERE id = $1`, playerID).Scan(&playerRank)

	if remainingEnemies == 0 {
		// All enemies defeated: generate spoils and soil contaminations, and transition to AFTERMATH
		if err := GenerateTribulationSpoilsAndFalloutTx(tx, ev.ID, ev.ApertureID, playerID.String(), ev.Path, playerRank, ev.TribulationType); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi sinh kiếp tài hậu kiếp: "+err.Error(), err))
			return
		}

		if err := tx.Commit(); err != nil {
			httperr.Respond(c, httperr.Internal("Transaction commit failed", err))
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Đã đẩy lùi toàn bộ kiếp nạn! Tiên Khiếu bước vào giai đoạn [DƯ BA & THU HOẠCH KIẾP TÀI]. Hãy thu hoạch kiếp tài và thanh tẩy thổ nhưỡng trước khi hoàn tất độ kiếp.",
			"state":   "TRIBULATION_ACTIVE",
			"stage":   "AFTERMATH",
		})
		return
	}

	// Player resolves manually with remaining enemies (rút lui) => ALWAYS FAIL
	now := time.Now()

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

	msg := "Đạo hữu rút lui khi kiếp nạn chưa dứt! Thiên Ý cuồng bạo tàn phá Tiên Khiếu, kiếp nạn bị phán định THẤT BẠI: Nguyên khí hư hao, bạn mất 2,000 Nguyên Thạch, 5 Tiên Nguyên, diện tích Tiên Khiếu và tốc độ thời gian bị giảm 10%, 30% điểm tài nguyên bị san phẳng."

	// Clean up dead/remaining enemies
	_, _ = tx.Exec(`UPDATE tribulation_enemies SET is_dead = true WHERE event_id = $1`, ev.ID)

	_, err = tx.Exec("UPDATE players SET human_qi = LEAST(human_qi + 10, $1) WHERE id = $2", MaxHumanQi, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi cập nhật Nhân Khí", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Transaction commit failed", err))
		return
	}

	// Check if the player is immediately due for another tribulation
	_ = checkAndTriggerSingleAperture(ev.ApertureID)

	var finalState string
	_ = db.DB.QueryRow(`SELECT state FROM immortal_apertures WHERE id = $1`, ev.ApertureID).Scan(&finalState)

	var finalDaoMarks models.DaoMarksMap
	var finalDaoJSON []byte
	if err2 := db.DB.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&finalDaoJSON); err2 == nil {
		_ = json.Unmarshal(finalDaoJSON, &finalDaoMarks)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   msg + " Nhân Khí +10.",
		"state":     finalState,
		"dao_marks": finalDaoMarks,
	})
}
