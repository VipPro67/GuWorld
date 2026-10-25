package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func AdminListPlayerSectMissions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT pm.id,pm.mission_id,sm.title,sm.sect_id,s.name,sm.mission_type,sm.target_id,sm.target_amount,sm.reward_contribution,pm.current_progress,pm.status,pm.assigned_at,pm.expires_at FROM player_sect_missions pm JOIN sect_missions sm ON sm.id=pm.mission_id JOIN sects s ON s.id=sm.sect_id WHERE pm.player_id=$1 ORDER BY pm.assigned_at DESC,pm.id`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhiệm vụ tông môn"})
		return
	}
	defer rows.Close()
	data := make([]gin.H, 0)
	for rows.Next() {
		var id, missionID, title, sectID, sectName, missionType, targetID, status string
		var targetAmount, reward, progress int64
		var assigned time.Time
		var expires sql.NullTime
		if err := rows.Scan(&id, &missionID, &title, &sectID, &sectName, &missionType, &targetID, &targetAmount, &reward, &progress, &status, &assigned, &expires); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc nhiệm vụ tông môn"})
			return
		}
		var expiresValue any
		if expires.Valid {
			expiresValue = expires.Time
		}
		data = append(data, gin.H{"id": id, "mission_id": missionID, "title": title, "sect_id": sectID, "sect_name": sectName, "mission_type": missionType, "target_id": targetID, "target_amount": targetAmount, "reward_contribution": reward, "current_progress": progress, "status": status, "assigned_at": assigned, "expires_at": expiresValue})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc nhiệm vụ tông môn"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "total": len(data)})
}

func AdminUpdatePlayerSectMission(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	missionRowID, err := uuid.Parse(c.Param("mission_row_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tiến độ nhiệm vụ không hợp lệ"})
		return
	}
	var req struct {
		CurrentProgress *int64  `json:"current_progress"`
		Status          *string `json:"status"`
		ExpiresAt       *string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.CurrentProgress == nil || req.Status == nil || req.ExpiresAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần đủ tiến độ, trạng thái và hạn nhiệm vụ"})
		return
	}
	status := strings.ToUpper(strings.TrimSpace(*req.Status))
	if status != "IN_PROGRESS" && status != "EXPIRED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái nhiệm vụ không hợp lệ"})
		return
	}
	if *req.CurrentProgress < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tiến độ không thể âm"})
		return
	}
	var expires any
	if value := strings.TrimSpace(*req.ExpiresAt); value != "" {
		parsed, parseErr := time.Parse(time.RFC3339, value)
		if parseErr != nil {
			parsed, parseErr = time.ParseInLocation("2006-01-02T15:04", value, time.Local)
		}
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thời hạn nhiệm vụ không hợp lệ"})
			return
		}
		expires = parsed
	}
	var missionID string
	if err := db.DB.QueryRowContext(c.Request.Context(), `UPDATE player_sect_missions SET current_progress=$1,status=$2,expires_at=$3 WHERE id=$4 AND player_id=$5 RETURNING mission_id::text`, *req.CurrentProgress, status, expires, missionRowID, playerID).Scan(&missionID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tiến độ nhiệm vụ của người chơi"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật tiến độ nhiệm vụ"})
		}
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.update_player_sect_mission", EntityType: "player_sect_missions", EntityID: missionRowID.String(), RequestSummary: "mission=" + missionID + " progress=" + strconv.FormatInt(*req.CurrentProgress, 10) + " status=" + status, IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật nhiệm vụ tông môn"})
}
