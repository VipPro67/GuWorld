package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

const practiceRealtimeMapID = "realtime_preview"

type PracticeRealtimeRequest struct {
	DummyHP     float64 `json:"dummy_hp"`
	DummyShield float64 `json:"dummy_shield"`
	DummyAtk    float64 `json:"dummy_atk"`
	DummyDef    float64 `json:"dummy_def"`
	DummySpeed  float64 `json:"dummy_speed"`
}

type PracticeRealtimeConfig struct {
	DummyHP     float64 `json:"dummy_hp"`
	DummyShield float64 `json:"dummy_shield"`
	DummyAtk    float64 `json:"dummy_atk"`
	DummyDef    float64 `json:"dummy_def"`
	DummySpeed  float64 `json:"dummy_speed"`
}

// StartRealtimePractice creates an isolated training instance. The durable
// character keeps its real resources; only its realtime routing fields change.
func StartRealtimePractice(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if reason := RealtimeRuntimeError(); reason != "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "realtime_unavailable", "error": "Phòng tập realtime hiện chưa khả dụng."})
		return
	}
	var req PracticeRealtimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config := PracticeRealtimeConfig{
		DummyHP:     boundedPracticeValue(req.DummyHP, 100000, 1, 1e12),
		DummyShield: boundedPracticeValue(req.DummyShield, 0, 0, 1e12),
		DummyAtk:    boundedPracticeValue(req.DummyAtk, 10, 0, 1e9),
		DummyDef:    boundedPracticeValue(req.DummyDef, 5, 0, 1e9),
		DummySpeed:  boundedPracticeValue(req.DummySpeed, 1, 0, 100),
	}

	handedOff, err := middleware.BeginRealtimeHandoff(playerID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "realtime_checkpoint_unavailable", "error": "Không thể tạm dừng thế giới realtime."})
		return
	}
	if handedOff {
		defer middleware.AbortRealtimeHandoff(playerID)
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở phiên tập luyện."})
		return
	}
	defer tx.Rollback()
	var regionID, currentZone, currentInstance string
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT COALESCE(current_region,''), COALESCE(NULLIF(world_zone_id,''), current_region), COALESCE(world_instance_id,'')
		FROM players WHERE id=$1 AND (active_combat IS NULL OR active_combat='null'::jsonb) FOR UPDATE
	`, playerID).Scan(&regionID, &currentZone, &currentInstance)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "Yêu cầu đã bị hủy."})
		} else if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusConflict, gin.H{"code": "character_not_ready", "error": "Nhân vật đang chiến đấu hoặc không khả dụng."})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc trạng thái nhân vật."})
		}
		return
	}
	if regionID == "" || currentInstance != "" {
		c.JSON(http.StatusConflict, gin.H{"code": "combat_instance_conflict", "error": "Nhân vật đã ở trong một phiên riêng."})
		return
	}
	instanceID := uuid.New()
	payload, err := json.Marshal(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo cấu hình hình nhân."})
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `
		INSERT INTO realtime_combat_instances (id,player_id,mode,zone_id,origin_zone_id,payload)
		VALUES ($1,$2,'training',$3,$4,$5)
	`, instanceID, playerID, practiceRealtimeMapID, currentZone, payload); err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "combat_instance_conflict", "error": "Không thể tạo phiên tập luyện."})
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `
		UPDATE players SET world_zone_id=$2, world_instance_id=$3, world_location_revision=world_location_revision+1
		WHERE id=$1
	`, playerID, practiceRealtimeMapID, "combat:"+instanceID.String()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuyển vào bản đồ tập luyện."})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu phiên tập luyện."})
		return
	}
	if handedOff {
		middleware.FinishRealtimeHandoff(playerID)
	}
	c.JSON(http.StatusOK, gin.H{"instance_id": instanceID.String(), "zone_id": practiceRealtimeMapID, "origin_zone_id": currentZone, "is_practice": true})
}

// EndRealtimePractice restores the zone that was active before training.
func EndRealtimePractice(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	handedOff, err := middleware.BeginRealtimeHandoff(playerID)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "realtime_checkpoint_unavailable", "error": "Không thể tạm dừng thế giới realtime."})
		return
	}
	if handedOff {
		defer middleware.AbortRealtimeHandoff(playerID)
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kết thúc phiên tập luyện."})
		return
	}
	defer tx.Rollback()
	var originZone string
	err = tx.QueryRowContext(c.Request.Context(), `
		SELECT origin_zone_id FROM realtime_combat_instances
		WHERE player_id=$1 AND mode='training' AND status='active' FOR UPDATE
	`, playerID).Scan(&originZone)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"code": "training_session_inactive", "error": "Không có phiên tập luyện đang mở."})
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE players SET world_zone_id=$2, world_instance_id='', world_location_revision=world_location_revision+1 WHERE id=$1 AND world_instance_id LIKE 'combat:%'`, playerID, originZone); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể khôi phục bản đồ trước đó."})
		return
	}
	if _, err = tx.ExecContext(c.Request.Context(), `UPDATE realtime_combat_instances SET status='completed', ended_at=NOW() WHERE player_id=$1 AND mode='training' AND status='active'`, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đóng phiên tập luyện."})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu kết quả tập luyện."})
		return
	}
	if handedOff {
		middleware.FinishRealtimeHandoff(playerID)
	}
	c.JSON(http.StatusOK, gin.H{"zone_id": originZone, "is_practice": false})
}

func boundedPracticeValue(value, fallback, minimum, maximum float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	if value == 0 && fallback != 0 {
		return fallback
	}
	return math.Max(minimum, math.Min(maximum, value))
}
