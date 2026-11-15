package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

type AdminFeedbackStatusRequest struct {
	Status            string `json:"status" binding:"required"`
	DeveloperResponse string `json:"developer_response"`
	RewardStones      int64  `json:"reward_stones"`
}

// AdminListFeedback returns paginated feedback items with filters and search.
func AdminListFeedback(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}

	search := strings.TrimSpace(c.Query("search"))
	category := strings.TrimSpace(c.Query("category"))
	status := strings.TrimSpace(c.Query("status"))

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	baseQuery := `
		FROM feedback f
		JOIN players p ON f.user_id = p.id
		WHERE 1=1
	`
	var args []interface{}
	argIdx := 1

	if search != "" {
		baseQuery += fmt.Sprintf(" AND (f.title ILIKE $%d OR f.description ILIKE $%d OR p.username ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}
	if category != "" && category != "ALL" {
		baseQuery += fmt.Sprintf(" AND f.category = $%d", argIdx)
		args = append(args, category)
		argIdx++
	}
	if status != "" && status != "ALL" {
		baseQuery += fmt.Sprintf(" AND f.status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	if err := db.DB.QueryRowContext(c.Request.Context(), countQuery, args...).Scan(&total); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đếm phản hồi", err))
		return
	}

	selectQuery := fmt.Sprintf(`
		SELECT f.id, f.user_id, p.username, f.title, f.description, f.category, f.status, f.total_votes, f.developer_response, f.rewarded, f.created_at,
		       (SELECT COUNT(1) FROM feedback_comments fc WHERE fc.feedback_id = f.id) AS comment_count
		%s
		ORDER BY f.created_at DESC
		LIMIT $%d OFFSET $%d
	`, baseQuery, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := db.DB.QueryContext(c.Request.Context(), selectQuery, args...)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đọc danh sách phản hồi", err))
		return
	}
	defer rows.Close()

	list := make([]gin.H, 0)
	for rows.Next() {
		var id, userID uuid.UUID
		var username, title, description, categoryVal, statusVal string
		var totalVotes, commentCount int
		var devResponse sql.NullString
		var rewarded bool
		var createdAt interface{}

		if err := rows.Scan(&id, &userID, &username, &title, &description, &categoryVal, &statusVal, &totalVotes, &devResponse, &rewarded, &createdAt, &commentCount); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu phản hồi", err))
			return
		}

		list = append(list, gin.H{
			"id":                 id.String(),
			"user_id":            userID.String(),
			"username":           username,
			"title":              title,
			"description":        description,
			"category":           categoryVal,
			"status":             statusVal,
			"total_votes":        totalVotes,
			"comment_count":      commentCount,
			"developer_response": devResponse.String,
			"rewarded":           rewarded,
			"created_at":         createdAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// AdminUpdateFeedbackStatus updates status, dev response, and optionally rewards player.
func AdminUpdateFeedbackStatus(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	feedbackID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phản hồi không hợp lệ"})
		return
	}

	var req AdminFeedbackStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}

	validStatuses := map[string]bool{
		"pending":     true,
		"active":      true,
		"in_progress": true,
		"completed":   true,
		"rejected":    true,
	}
	statusLower := strings.ToLower(req.Status)
	if !validStatuses[statusLower] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Trạng thái không hợp lệ"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể mở transaction", err))
		return
	}
	defer tx.Rollback()

	var authorID uuid.UUID
	var title string
	var alreadyRewarded bool
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT user_id, title, rewarded FROM feedback WHERE id = $1 FOR UPDATE`, feedbackID).Scan(&authorID, &title, &alreadyRewarded); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phản hồi"})
		return
	}

	rewardGranted := alreadyRewarded
	if req.RewardStones > 0 && !alreadyRewarded {
		_, err = tx.ExecContext(c.Request.Context(), `UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, req.RewardStones, authorID)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Không thể phát thưởng nguyên thạch", err))
			return
		}
		rewardGranted = true
	}

	_, err = tx.ExecContext(c.Request.Context(), `
		UPDATE feedback
		SET status = $1, developer_response = $2, rewarded = $3, updated_at = NOW()
		WHERE id = $4
	`, statusLower, req.DeveloperResponse, rewardGranted, feedbackID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể cập nhật phản hồi", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể commit transaction", err))
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		EntityType:     "feedback",
		EntityID:       feedbackID.String(),
		Action:         "admin.update_feedback_status",
		RequestSummary: fmt.Sprintf("status=%s title=%s reward_stones=%d", statusLower, title, req.RewardStones),
		Reason:         c.GetHeader("X-Admin-Reason"),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã cập nhật phản hồi thành công"})
}

// AdminDeleteFeedback deletes a feedback row along with its votes and comments.
func AdminDeleteFeedback(c *gin.Context) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}

	feedbackID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID phản hồi không hợp lệ"})
		return
	}

	var title string
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT title FROM feedback WHERE id = $1`, feedbackID).Scan(&title); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phản hồi"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể mở transaction", err))
		return
	}
	defer tx.Rollback()

	_, _ = tx.ExecContext(c.Request.Context(), `DELETE FROM feedback_votes WHERE feedback_id = $1`, feedbackID)
	_, _ = tx.ExecContext(c.Request.Context(), `DELETE FROM feedback_comments WHERE feedback_id = $1`, feedbackID)
	if _, err := tx.ExecContext(c.Request.Context(), `DELETE FROM feedback WHERE id = $1`, feedbackID); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể xóa phản hồi", err))
		return
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể commit transaction", err))
		return
	}

	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &adminID,
		EntityType:     "feedback",
		EntityID:       feedbackID.String(),
		Action:         "admin.delete_feedback",
		RequestSummary: fmt.Sprintf("title=%s", title),
		Reason:         c.GetHeader("X-Admin-Reason"),
		IPAddress:      c.ClientIP(),
		Success:        true,
	})

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa phản hồi thành công"})
}
