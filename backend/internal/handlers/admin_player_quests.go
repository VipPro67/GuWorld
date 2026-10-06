package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminQuestProgressRequest struct {
	Status       string          `json:"status"`
	ProgressData json.RawMessage `json:"progress_data"`
	CompletedAt  *string         `json:"completed_at"`
}

func AdminListPlayerQuests(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	search := strings.ToLower(strings.TrimSpace(c.Query("search")))
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT quest_id, status, progress_data, completed_at, created_at, updated_at FROM player_quest_progress WHERE player_id=$1 ORDER BY quest_id`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải tiến độ nhiệm vụ"})
		return
	}
	defer rows.Close()
	data := make([]gin.H, 0)
	for rows.Next() {
		var questID, status string
		var progressData json.RawMessage
		var completedAt, createdAt, updatedAt *time.Time
		if err := rows.Scan(&questID, &status, &progressData, &completedAt, &createdAt, &updatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc tiến độ nhiệm vụ"})
			return
		}
		if search != "" && !strings.Contains(strings.ToLower(questID), search) {
			continue
		}
		data = append(data, gin.H{"quest_id": questID, "status": status, "progress_data": progressData, "completed_at": completedAt, "created_at": createdAt, "updated_at": updatedAt})
	}
	c.JSON(http.StatusOK, gin.H{"data": data, "total": len(data)})
}

func AdminUpdatePlayerQuest(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	questID := strings.TrimSpace(c.Param("quest_id"))
	if questID == "" || len(questID) > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quest ID không hợp lệ"})
		return
	}
	var req AdminQuestProgressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu nhiệm vụ không hợp lệ"})
		return
	}
	req.Status = strings.ToUpper(strings.TrimSpace(req.Status))
	if req.Status != "ACTIVE" && req.Status != "COMPLETED" && req.Status != "FAILED" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái nhiệm vụ không hợp lệ"})
		return
	}
	if len(req.ProgressData) == 0 {
		req.ProgressData = json.RawMessage(`{}`)
	}
	var decoded interface{}
	if err := json.Unmarshal(req.ProgressData, &decoded); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "progress_data phải là JSON hợp lệ"})
		return
	}
	var completedAt *time.Time
	if req.CompletedAt != nil && strings.TrimSpace(*req.CompletedAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.CompletedAt))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "completed_at phải theo RFC3339"})
			return
		}
		completedAt = &parsed
	}
	if req.Status == "COMPLETED" && completedAt == nil {
		now := time.Now()
		completedAt = &now
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()
	if _, err := loadPlayerForStatus(tx, playerID, true); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}
	if err := QuestSvc.Upsert(tx, &services.QuestProgress{PlayerID: playerID.String(), QuestID: questID, Status: req.Status, ProgressData: req.ProgressData, CompletedAt: completedAt}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật nhiệm vụ"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu nhiệm vụ"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.update_quest_progress", EntityType: "player_quest_progress", EntityID: playerID.String() + ":" + questID, RequestSummary: fmt.Sprintf("quest=%s status=%s", questID, req.Status), IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật tiến độ nhiệm vụ", "quest_id": questID, "status": req.Status, "progress_data": req.ProgressData, "completed_at": completedAt})
}

func AdminDeletePlayerQuest(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	playerID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID người chơi không hợp lệ"})
		return
	}
	questID := strings.TrimSpace(c.Param("quest_id"))
	if questID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quest ID không hợp lệ"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi database"})
		return
	}
	defer tx.Rollback()
	res, err := tx.Exec(`DELETE FROM player_quest_progress WHERE player_id=$1 AND quest_id=$2`, playerID, questID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa nhiệm vụ"})
		return
	}
	count, _ := res.RowsAffected()
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy tiến độ nhiệm vụ"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu thay đổi"})
		return
	}
	auditlog.Log(db.DB, auditlog.Entry{PlayerID: &adminID, Action: "admin.delete_quest_progress", EntityType: "player_quest_progress", EntityID: playerID.String() + ":" + questID, IPAddress: c.ClientIP(), Success: true})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa tiến độ nhiệm vụ", "quest_id": questID})
}
