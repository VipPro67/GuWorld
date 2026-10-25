package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

func AdminListPlayerTrackingMarks(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT m.id,m.tracker_id,COALESCE(t.username,''),m.target_id,COALESCE(p.username,''),m.mark_type,m.power,m.expires_at,m.created_at FROM player_tracking_marks m JOIN players t ON t.id=m.tracker_id JOIN players p ON p.id=m.target_id WHERE m.tracker_id=$1 OR m.target_id=$1 ORDER BY m.created_at DESC,m.id`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải ấn ký truy tung"})
		return
	}
	defer rows.Close()
	data := make([]gin.H, 0)
	for rows.Next() {
		var id, trackerID, trackerName, targetID, targetName, markType string
		var power float64
		var expires, created time.Time
		if err := rows.Scan(&id, &trackerID, &trackerName, &targetID, &targetName, &markType, &power, &expires, &created); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc ấn ký truy tung"})
			return
		}
		data = append(data, gin.H{"id": id, "tracker_id": trackerID, "tracker_username": trackerName, "target_id": targetID, "target_username": targetName, "mark_type": markType, "power": power, "expires_at": expires, "created_at": created, "is_tracker": trackerID == playerID.String()})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc ấn ký truy tung"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "total": len(data)})
}

func AdminUpdatePlayerTrackingMark(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	markID, err := uuid.Parse(c.Param("mark_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID ấn ký không hợp lệ"})
		return
	}
	var req struct {
		MarkType  *string  `json:"mark_type"`
		Power     *float64 `json:"power"`
		ExpiresAt *string  `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.MarkType == nil || req.Power == nil || req.ExpiresAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần đủ loại ấn, uy lực và hạn dùng"})
		return
	}
	markType := strings.ToLower(strings.TrimSpace(*req.MarkType))
	if markType != "blood" && markType != "wisdom" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại ấn chỉ nhận blood hoặc wisdom"})
		return
	}
	if *req.Power < 0 || *req.Power > 1e9 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Uy lực phải nằm trong khoảng 0 đến 1.000.000.000"})
		return
	}
	var expiry time.Time
	if expiry, err = time.Parse(time.RFC3339, *req.ExpiresAt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hạn dùng không hợp lệ"})
		return
	}
	var trackerID, targetID string
	if err := db.DB.QueryRowContext(c.Request.Context(), `UPDATE player_tracking_marks SET mark_type=$1,power=$2,expires_at=$3 WHERE id=$4 AND (tracker_id=$5 OR target_id=$5) RETURNING tracker_id::text,target_id::text`, markType, *req.Power, expiry, markID, playerID).Scan(&trackerID, &targetID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy ấn ký của người chơi"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật ấn ký"})
		}
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.update_player_tracking_mark", EntityType: "player_tracking_marks", EntityID: markID.String(), RequestSummary: fmt.Sprintf("tracker=%s target=%s type=%s power=%g", trackerID, targetID, markType, *req.Power), IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật ấn ký truy tung"})
}

func AdminDeletePlayerTrackingMark(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	markID, err := uuid.Parse(c.Param("mark_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID ấn ký không hợp lệ"})
		return
	}
	var req struct {
		Reason             string `json:"reason"`
		ConfirmationTarget string `json:"confirmation_target"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Reason) == "" || strings.TrimSpace(req.ConfirmationTarget) != c.Request.URL.Path {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cần lý do và xác nhận đúng ấn ký cần thu hồi"})
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM player_tracking_marks WHERE id=$1 AND (tracker_id=$2 OR target_id=$2)`, markID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thu hồi ấn ký"})
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy ấn ký của người chơi"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.delete_player_tracking_mark", EntityType: "player_tracking_marks", EntityID: markID.String(), RequestSummary: "reason=" + req.Reason, IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã thu hồi ấn ký"})
}
