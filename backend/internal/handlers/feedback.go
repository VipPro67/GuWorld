package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/auth"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type SubmitFeedbackRequest struct {
	Title       string `json:"title" binding:"required,min=5,max=100"`
	Description string `json:"description" binding:"required,min=10,max=1000"`
	Category    string `json:"category" binding:"required"` // 'BUG', 'GU', 'ITEM', 'RECIPE', 'BALANCE', 'FEATURE', 'OTHER'
}

type VoteRequest struct {
	VoteType int `json:"vote_type" binding:"required"` // 1 for Upvote, -1 for Downvote
}

type UpdateFeedbackStatusRequest struct {
	Status            string `json:"status" binding:"required"` // 'pending', 'active', 'in_progress', 'completed', 'rejected'
	DeveloperResponse string `json:"developer_response"`
}

// GetFeedbacks returns all feedbacks matching optional filters
func GetFeedbacks(c *gin.Context) {
	// Optional auth
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		// Prefer the Authorization header (current token from localStorage);
		// fall back to the cookie for clients that only send cookies.
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				claims, err := auth.ValidateToken(parts[1])
				if err == nil {
					if parsedID, err := uuid.Parse(claims.PlayerID); err == nil {
						authID = parsedID
						exists = true
					}
				}
			}
		} else if tokenStr, err := c.Cookie("gu_auth_token"); err == nil && tokenStr != "" {
			claims, err := auth.ValidateToken(tokenStr)
			if err == nil {
				if parsedID, err := uuid.Parse(claims.PlayerID); err == nil {
					authID = parsedID
					exists = true
				}
			}
		}
	}

	category := c.Query("category")
	status := c.Query("status")

	// Get player's admin status
	var isAdmin bool
	if exists {
		err := db.DB.QueryRowContext(c.Request.Context(), "SELECT COALESCE(is_admin, false) FROM players WHERE id = $1", authID).Scan(&isAdmin)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}
	}

	queryAuthID := authID
	if !exists {
		queryAuthID = uuid.Nil
	}

	cacheKey := fmt.Sprintf("feedbacks_user_%s_cat_%s_status_%s", queryAuthID.String(), category, status)
	var cachedFeedbacks []models.Feedback
	if err := cache.GetJSON(cacheKey, &cachedFeedbacks); err == nil {
		c.JSON(http.StatusOK, cachedFeedbacks)
		return
	}

	// Base query
	query := `
		SELECT f.id, f.user_id, p.username, f.title, f.description, f.category, f.status, f.total_votes, f.developer_response, f.rewarded, f.created_at,
		       v.vote_type,
		       (SELECT COUNT(1) FROM feedback_comments fc WHERE fc.feedback_id = f.id) AS comment_count
		FROM feedback f
		JOIN players p ON f.user_id = p.id
		LEFT JOIN feedback_votes v ON f.id = v.feedback_id AND v.user_id = $1
		WHERE 1=1
	`
	args := []interface{}{queryAuthID}
	argCount := 1

	// Filter by category
	if category != "" {
		argCount++
		query += fmt.Sprintf(" AND f.category = $%d", argCount)
		args = append(args, category)
	}

	// Filter by status
	if status != "" {
		argCount++
		query += fmt.Sprintf(" AND f.status = $%d", argCount)
		args = append(args, status)
	}

	// If not admin, only show active/in_progress/completed/rejected OR user's own pending feedback
	if !isAdmin {
		if exists {
			query += " AND (f.status != 'pending' OR f.user_id = $1)"
		} else {
			query += " AND f.status != 'pending'"
		}
	}

	// Order by votes desc, then created_at desc
	query += " ORDER BY f.total_votes DESC, f.created_at DESC"

	rows, err := db.DB.QueryContext(c.Request.Context(), query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch feedbacks: " + err.Error()})
		return
	}
	defer rows.Close()

	feedbacks := []models.Feedback{}
	for rows.Next() {
		var f models.Feedback
		var devResp sql.NullString
		var userVote sql.NullInt64

		err := rows.Scan(
			&f.ID, &f.UserID, &f.Username, &f.Title, &f.Description, &f.Category, &f.Status, &f.TotalVotes, &devResp, &f.Rewarded, &f.CreatedAt,
			&userVote, &f.CommentCount,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error scanning feedback: " + err.Error()})
			return
		}

		if devResp.Valid {
			f.DeveloperResponse = &devResp.String
		}

		if userVote.Valid {
			voteVal := int(userVote.Int64)
			f.UserVote = &voteVal
		}

		feedbacks = append(feedbacks, f)
	}

	_ = cache.SetJSON(cacheKey, feedbacks, 10*time.Second)
	c.JSON(http.StatusOK, feedbacks)
}

// SubmitFeedback creates a new feedback entry (default: pending status for moderation)
func SubmitFeedback(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req SubmitFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate category
	validCategories := map[string]bool{
		"BUG":     true,
		"GU":      true,
		"ITEM":    true,
		"RECIPE":  true,
		"BALANCE": true,
		"FEATURE": true,
		"OTHER":   true,
	}
	categoryUpper := strings.ToUpper(req.Category)
	if !validCategories[categoryUpper] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Danh mục thỉnh cầu không hợp lệ!"})
		return
	}

	// Default status is pending
	status := "pending"

	// If player is admin, default to active immediately
	var isAdmin bool
	_ = db.DB.QueryRow("SELECT COALESCE(is_admin, false) FROM players WHERE id = $1", authID).Scan(&isAdmin)
	if isAdmin {
		status = "active"
	}

	var newID uuid.UUID
	err := db.DB.QueryRow(`
		INSERT INTO feedback (user_id, title, description, category, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, authID, req.Title, req.Description, categoryUpper, status).Scan(&newID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Thỉnh nguyện lên Thiên Đạo thất bại: " + err.Error()})
		return
	}

	msg := "Đạo hữu đã truyền tin Thỉnh Thiên Đạo thành công! Ý kiến sẽ hiển thị sau khi Thiên Đạo xét duyệt."
	if isAdmin {
		msg = "Admin đã đăng góp ý thành công!"
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     msg,
		"feedback_id": newID,
	})
}

// VoteFeedback handles upvoting or downvoting using transaction logic to avoid race conditions
func VoteFeedback(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	feedbackIDStr := c.Param("id")
	feedbackID, err := uuid.Parse(feedbackIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID không hợp lệ"})
		return
	}

	var req VoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.VoteType != 1 && req.VoteType != -1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại bình chọn không hợp lệ (chỉ chấp nhận 1 hoặc -1)"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}
	defer tx.Rollback()

	// 1. Fetch voter info to enforce rank requirement
	var rank int
	err = tx.QueryRow("SELECT rank FROM players WHERE id = $1", authID).Scan(&rank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải thông tin cổ sư: " + err.Error()})
		return
	}

	if rank < 2 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu phải đạt Nhị Chuyển trở lên mới có thể Thỉnh Thiên Đạo (Bình chọn)!"})
		return
	}

	// 2. Check if feedback exists and is active
	var feedbackStatus string
	err = tx.QueryRow("SELECT status FROM feedback WHERE id = $1 FOR UPDATE", feedbackID).Scan(&feedbackStatus)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Góp ý không tồn tại!"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	if feedbackStatus != "active" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Góp ý này không trong trạng thái bình chọn (chỉ sớ trình đang mở mới được bình chọn)!"})
		return
	}

	// 3. Check existing vote
	var existingVoteType int
	err = tx.QueryRow("SELECT vote_type FROM feedback_votes WHERE feedback_id = $1 AND user_id = $2", feedbackID, authID).Scan(&existingVoteType)

	var voteDiff int
	var hasVoted bool

	if err == sql.ErrNoRows {
		// No existing vote
		hasVoted = false
	} else if err == nil {
		// Existing vote found
		hasVoted = true
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error checking vote: " + err.Error()})
		return
	}

	if !hasVoted {
		// Insert new vote
		_, err = tx.Exec("INSERT INTO feedback_votes (feedback_id, user_id, vote_type) VALUES ($1, $2, $3)", feedbackID, authID, req.VoteType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu lượt vote: " + err.Error()})
			return
		}
		voteDiff = req.VoteType
	} else {
		if existingVoteType == req.VoteType {
			// Clicked same vote type again -> Cancel vote
			_, err = tx.Exec("DELETE FROM feedback_votes WHERE feedback_id = $1 AND user_id = $2", feedbackID, authID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xóa lượt vote: " + err.Error()})
				return
			}
			voteDiff = -existingVoteType
		} else {
			// Changed vote type
			_, err = tx.Exec("UPDATE feedback_votes SET vote_type = $1 WHERE feedback_id = $2 AND user_id = $3", req.VoteType, feedbackID, authID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật lượt vote: " + err.Error()})
				return
			}
			voteDiff = 2 * req.VoteType
		}
	}

	// 4. Update total_votes in feedback table atomically
	var newTotalVotes int
	err = tx.QueryRow(`
		UPDATE feedback 
		SET total_votes = total_votes + $1 
		WHERE id = $2 
		RETURNING total_votes
	`, voteDiff, feedbackID).Scan(&newTotalVotes)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật tổng số lượt vote: " + err.Error()})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failure: " + err.Error()})
		return
	}

	var action string
	if !hasVoted {
		if req.VoteType == 1 {
			action = "Upvote thành công"
		} else {
			action = "Downvote thành công"
		}
	} else if existingVoteType == req.VoteType {
		action = "Đã hủy bình chọn"
	} else {
		if req.VoteType == 1 {
			action = "Chuyển sang Upvote thành công"
		} else {
			action = "Chuyển sang Downvote thành công"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     action,
		"total_votes": newTotalVotes,
		"user_vote":   req.VoteType, // or nil if deleted
	})
}

// UpdateFeedbackStatus allows admins to moderate feedbacks, update status, respond, and distribute rewards
func UpdateFeedbackStatus(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	feedbackIDStr := c.Param("id")
	feedbackID, err := uuid.Parse(feedbackIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID không hợp lệ"})
		return
	}

	var req UpdateFeedbackStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}
	defer tx.Rollback()

	// 1. Check if updater is admin
	var isAdmin bool
	err = tx.QueryRow("SELECT COALESCE(is_admin, false) FROM players WHERE id = $1", authID).Scan(&isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error validating admin status"})
		return
	}

	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không có quyền của Thiên Đạo (Admin) để thực hiện thao tác này!"})
		return
	}

	// 2. Fetch feedback details
	var authorID uuid.UUID
	var title string
	var alreadyRewarded bool
	var oldStatus string

	err = tx.QueryRow("SELECT user_id, title, rewarded, status FROM feedback WHERE id = $1 FOR UPDATE", feedbackID).Scan(
		&authorID, &title, &alreadyRewarded, &oldStatus,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy góp ý này!"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	// 3. Update feedback status and response
	var devResp sql.NullString
	if req.DeveloperResponse != "" {
		devResp.String = req.DeveloperResponse
		devResp.Valid = true
	}

	_, err = tx.Exec(`
		UPDATE feedback 
		SET status = $1, developer_response = $2 
		WHERE id = $3
	`, statusLower, devResp, feedbackID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật trạng thái góp ý: " + err.Error()})
		return
	}

	// 4. Check if we should distribute rewards
	// Reward criteria: status becomes 'in_progress' or 'completed' and not already rewarded
	shouldReward := (statusLower == "in_progress" || statusLower == "completed") && !alreadyRewarded
	rewardMsg := ""

	if shouldReward {
		// Fetch author tu vi and current balances
		var authorUsername string
		var authorRank int
		var primevalStones int
		var immortalStones float64

		err = tx.QueryRow(`
			SELECT username, rank, FLOOR(primeval_stones)::BIGINT, immortal_stones 
			FROM players 
			WHERE id = $1 FOR UPDATE
		`, authorID).Scan(&authorUsername, &authorRank, &primevalStones, &immortalStones)

		if err == nil {
			globalUUID := "00000000-0000-0000-0000-000000000000" // Bảo Hoàng Thiên
			var content string

			if authorRank >= 6 {
				// Cổ Tiên: Thưởng 5 Tiên Nguyên Thạch
				immortalStones += 5.0
				_, err = tx.Exec("UPDATE players SET immortal_stones = $1 WHERE id = $2", immortalStones, authorID)
				if err == nil {
					content = fmt.Sprintf("Chúc mừng đạo hữu! Góp ý '%s' của đạo hữu đã được Thiên Đạo tiếp nhận phát triển (Trạng thái: %s). Thiên Đạo ban thưởng đạo hữu 5 Tiên Nguyên Thạch làm khích lệ!", title, statusLower)
					_, _ = tx.Exec(`
						INSERT INTO messages (sender_id, receiver_id, content) 
						VALUES ($1, $2, $3)
					`, globalUUID, authorID, content)
					rewardMsg = " [Ban thưởng 5 Tiên Nguyên Thạch cho Cổ Tiên thành công]"
				}
			} else {
				// Cổ Sư Phàm nhân: Thưởng 500 Nguyên Thạch
				primevalStones += 500
				_, err = tx.Exec("UPDATE players SET primeval_stones = $1 WHERE id = $2", primevalStones, authorID)
				if err == nil {
					content = fmt.Sprintf("Chúc mừng đạo hữu! Góp ý '%s' của đạo hữu đã được Thiên Đạo tiếp nhận phát triển (Trạng thái: %s). Thiên Đạo ban thưởng đạo hữu 500 Nguyên Thạch làm khích lệ!", title, statusLower)
					_, _ = tx.Exec(`
						INSERT INTO messages (sender_id, receiver_id, content) 
						VALUES ($1, $2, $3)
					`, globalUUID, authorID, content)
					rewardMsg = " [Ban thưởng 500 Nguyên Thạch cho Cổ sư thành công]"
				}
			}

			// Update feedback table to mark rewarded = true
			if err == nil {
				_, _ = tx.Exec("UPDATE feedback SET rewarded = true WHERE id = $1", feedbackID)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi commit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Cập nhật trạng thái góp ý thành công!%s", rewardMsg),
	})
}

// GetFeedbackComments returns all góp ý (comments) on a feedback topic, newest last.
func GetFeedbackComments(c *gin.Context) {
	feedbackIDStr := c.Param("id")
	feedbackID, err := uuid.Parse(feedbackIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID không hợp lệ"})
		return
	}

	var exists bool
	err = db.DB.QueryRowContext(c.Request.Context(), "SELECT EXISTS(SELECT 1 FROM feedback WHERE id = $1)", feedbackID).Scan(&exists)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Góp ý không tồn tại!"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT c.id, c.feedback_id, c.user_id, p.username, c.content, c.created_at
		FROM feedback_comments c
		JOIN players p ON c.user_id = p.id
		WHERE c.feedback_id = $1
		ORDER BY c.created_at ASC
	`, feedbackID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải góp ý: " + err.Error()})
		return
	}
	defer rows.Close()

	comments := []models.FeedbackComment{}
	for rows.Next() {
		var cm models.FeedbackComment
		if err := rows.Scan(&cm.ID, &cm.FeedbackID, &cm.UserID, &cm.Username, &cm.Content, &cm.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc góp ý: " + err.Error()})
			return
		}
		comments = append(comments, cm)
	}

	c.JSON(http.StatusOK, comments)
}

type AddFeedbackCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=500"`
}

// AddFeedbackComment lets a rank >= 2 cultivator post a góp ý on an approved topic.
// Góp ý chỉ được phép khi status là active / in_progress / completed (đã qua duyệt).
func AddFeedbackComment(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req AddFeedbackCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nội dung góp ý không hợp lệ (1-500 ký tự)"})
		return
	}

	feedbackIDStr := c.Param("id")
	feedbackID, err := uuid.Parse(feedbackIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Feedback ID không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}
	defer tx.Rollback()

	// 1. Rank requirement (giống vote: rank >= 2)
	var rank int
	err = tx.QueryRow("SELECT rank FROM players WHERE id = $1", authID).Scan(&rank)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải thông tin cổ sư: " + err.Error()})
		return
	}
	if rank < 2 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu phải đạt Nhị Chuyển trở lên mới có thể góp ý trên sớ trình!"})
		return
	}

	// 2. Status gating: chỉ góp ý được khi sớ trình đã được duyệt (active/in_progress/completed)
	var status string
	err = tx.QueryRow("SELECT status FROM feedback WHERE id = $1 FOR UPDATE", feedbackID).Scan(&status)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Góp ý không tồn tại!"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	switch status {
	case "active", "in_progress", "completed":
		// allowed
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sớ trình này chưa được duyệt hoặc đã khép lại, không thể góp ý!"})
		return
	}

	var commentID uuid.UUID
	var createdAt time.Time
	err = tx.QueryRow(`
		INSERT INTO feedback_comments (feedback_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`, feedbackID, authID, strings.TrimSpace(req.Content)).Scan(&commentID, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu góp ý: " + err.Error()})
		return
	}

	var commentCount int
	err = tx.QueryRow("SELECT COUNT(1) FROM feedback_comments WHERE feedback_id = $1", feedbackID).Scan(&commentCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đếm góp ý: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failure: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Đã góp ý lên sớ trình thành công!",
		"comment_id":    commentID,
		"created_at":    createdAt,
		"comment_count": commentCount,
	})
}

// DeleteFeedbackComment lets the comment author or an admin remove a góp ý.
func DeleteFeedbackComment(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	commentIDStr := c.Param("commentId")
	commentID, err := uuid.Parse(commentIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Comment ID không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}
	defer tx.Rollback()

	var authorID uuid.UUID
	err = tx.QueryRow("SELECT user_id FROM feedback_comments WHERE id = $1 FOR UPDATE", commentID).Scan(&authorID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Góp ý không tồn tại!"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	var isAdmin bool
	_ = tx.QueryRow("SELECT COALESCE(is_admin, false) FROM players WHERE id = $1", authID).Scan(&isAdmin)

	if authorID != authID && !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không có quyền xóa góp ý này!"})
		return
	}

	_, err = tx.Exec("DELETE FROM feedback_comments WHERE id = $1", commentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xóa góp ý: " + err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failure: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa góp ý!"})
}
