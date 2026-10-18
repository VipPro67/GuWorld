package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type adminInstanceSessionUpdate struct {
	CurrentStage  int             `json:"current_stage"`
	Status        string          `json:"status"`
	TempInventory json.RawMessage `json:"temp_inventory"`
}

func AdminListPlayerInstanceSessions(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id, instance_id, current_stage, status, temp_inventory, created_at FROM player_instance_sessions WHERE player_id=$1 ORDER BY created_at DESC`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải phiên mộng cảnh"})
		return
	}
	defer rows.Close()
	data := make([]gin.H, 0)
	for rows.Next() {
		var id uuid.UUID
		var instanceID, status string
		var stage int
		var tempInventory json.RawMessage
		var createdAt time.Time
		if err := rows.Scan(&id, &instanceID, &stage, &status, &tempInventory, &createdAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc phiên mộng cảnh"})
			return
		}
		data = append(data, gin.H{"id": id, "player_id": playerID, "instance_id": instanceID, "current_stage": stage, "status": status, "temp_inventory": tempInventory, "created_at": createdAt})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc phiên mộng cảnh"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "total": len(data)})
}

func AdminUpdatePlayerInstanceSession(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phiên không hợp lệ"})
		return
	}
	var req adminInstanceSessionUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu phiên mộng cảnh không hợp lệ"})
		return
	}
	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	if req.CurrentStage < 0 || (req.Status != "ACTIVE" && req.Status != "COMPLETED" && req.Status != "FAILED") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu phiên mộng cảnh không hợp lệ"})
		return
	}
	if len(req.TempInventory) == 0 {
		req.TempInventory = json.RawMessage(`[]`)
	}
	var inventory []json.RawMessage
	if err := json.Unmarshal(req.TempInventory, &inventory); err != nil || inventory == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "temp_inventory phải là một JSON array hợp lệ"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()
	var instanceID string
	err = tx.QueryRow(`UPDATE player_instance_sessions SET current_stage=$1, status=$2, temp_inventory=$3 WHERE id=$4 AND player_id=$5 RETURNING instance_id`, req.CurrentStage, req.Status, req.TempInventory, sessionID, playerID).Scan(&instanceID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phiên mộng cảnh"})
		} else {
			var pqErr *pq.Error
			if errors.As(err, &pqErr) && pqErr.Code == "23505" {
				c.JSON(http.StatusConflict, gin.H{"error": "Người chơi đã có một phiên ACTIVE khác"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật phiên mộng cảnh"})
			}
		}
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu phiên mộng cảnh"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.update_instance_session", EntityType: "player_instance_sessions", EntityID: sessionID.String(), RequestSummary: fmt.Sprintf("player=%s instance=%s stage=%d status=%s", playerID, instanceID, req.CurrentStage, req.Status), IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật phiên mộng cảnh", "id": sessionID, "instance_id": instanceID, "current_stage": req.CurrentStage, "status": req.Status, "temp_inventory": req.TempInventory})
}

func AdminDeletePlayerInstanceSession(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phiên không hợp lệ"})
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM player_instance_sessions WHERE id=$1 AND player_id=$2`, sessionID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa phiên mộng cảnh"})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phiên mộng cảnh"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.delete_instance_session", EntityType: "player_instance_sessions", EntityID: sessionID.String(), RequestSummary: strings.TrimSpace(playerID.String()), IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa phiên mộng cảnh"})
}
