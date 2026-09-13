package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// GetJoinRequests fetches pending join requests for the sect
func GetJoinRequests(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sectID, callerRank string
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn!"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT r.id, r.sect_id, r.player_id, r.status, r.created_at, p.username, p.avatar_url, p.rank
		FROM sect_join_requests r
		JOIN players p ON r.player_id = p.id
		WHERE r.sect_id = $1 AND r.status = 'PENDING'
		ORDER BY r.created_at ASC
	`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch join requests: " + err.Error()})
		return
	}
	defer rows.Close()

	var reqs []SectJoinRequest
	for rows.Next() {
		var r SectJoinRequest
		if err := rows.Scan(&r.ID, &r.SectID, &r.PlayerID, &r.Status, &r.CreatedAt, &r.Username, &r.AvatarURL, &r.PlayerRank); err == nil {
			reqs = append(reqs, r)
		}
	}

	c.JSON(http.StatusOK, reqs)
}

type ActionJoinRequestRequest struct {
	Action string `json:"action" binding:"required"` // approve, reject
}

// ActionJoinRequest handles join request approval/rejection
func ActionJoinRequest(c *gin.Context) {
	requestIDStr := c.Param("requestId")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID format"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ActionJoinRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn để duyệt yêu cầu!"})
		return
	}

	// Verify request belongs to this sect
	var targetPlayerID uuid.UUID
	var reqSectID string
	err = tx.QueryRow(`SELECT player_id, sect_id FROM sect_join_requests WHERE id = $1`, requestID).Scan(&targetPlayerID, &reqSectID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Yêu cầu gia nhập không tồn tại!"})
		return
	}

	if reqSectID != sectID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Yêu cầu gia nhập này thuộc tông môn khác!"})
		return
	}

	if strings.ToLower(req.Action) == "approve" {
		// Check if player is already in a sect
		var existingSect string
		err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, targetPlayerID).Scan(&existingSect)
		if err == nil {
			// Delete their requests since they already joined somewhere else
			_, _ = tx.Exec(`DELETE FROM sect_join_requests WHERE player_id = $1`, targetPlayerID)
			_ = tx.Commit()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu này đã gia nhập tông môn khác!"})
			return
		}

		// Add as Ngoại Môn
		_, err = tx.Exec(`
			INSERT INTO sect_members (sect_id, player_id, member_rank, contribution_points)
			VALUES ($1, $2, 'NGOAI_MON', 0)
		`, sectID, targetPlayerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member to sect: " + err.Error()})
			return
		}

		// Delete all pending requests for this player (since they joined this sect now)
		_, err = tx.Exec(`DELETE FROM sect_join_requests WHERE player_id = $1`, targetPlayerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clean join requests: " + err.Error()})
			return
		}
	} else if strings.ToLower(req.Action) == "reject" {
		// Just delete the request to allow them to apply again later
		_, err = tx.Exec(`DELETE FROM sect_join_requests WHERE id = $1`, requestID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reject join request: " + err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hành động không hợp lệ (chỉ chấp nhận approve hoặc reject)"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	msg := "Đã từ chối yêu cầu gia nhập!"
	if strings.ToLower(req.Action) == "approve" {
		msg = "Đã đồng ý cho đệ tử gia nhập tông môn!"
		var username string
		_ = db.DB.QueryRow(`SELECT username FROM players WHERE id = $1`, targetPlayerID).Scan(&username)
		if username != "" {
			_ = SendSystemMessage("sect", sql.NullString{String: sectID, Valid: true}, fmt.Sprintf("Chào mừng đạo hữu %s đã gia nhập tông môn!", username))
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": msg})
}

// GetPromotionRequests fetches pending promotion requests for the sect
func GetPromotionRequests(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var sectID, callerRank string
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn!"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT r.id, r.sect_id, r.player_id, r.current_rank, r.target_rank, r.status, r.created_at, p.username, p.avatar_url, p.rank
		FROM sect_promotion_requests r
		JOIN players p ON r.player_id = p.id
		WHERE r.sect_id = $1 AND r.status = 'PENDING'
		ORDER BY r.created_at ASC
	`, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch promotion requests: " + err.Error()})
		return
	}
	defer rows.Close()

	var reqs []SectPromotionRequest
	for rows.Next() {
		var r SectPromotionRequest
		if err := rows.Scan(&r.ID, &r.SectID, &r.PlayerID, &r.CurrentRank, &r.TargetRank, &r.Status, &r.CreatedAt, &r.Username, &r.AvatarURL, &r.PlayerRank); err == nil {
			reqs = append(reqs, r)
		}
	}

	c.JSON(http.StatusOK, reqs)
}

type ActionPromotionRequestRequest struct {
	Action string `json:"action" binding:"required"` // approve, reject
}

// ActionPromotionRequest handles promotion request approval/rejection
func ActionPromotionRequest(c *gin.Context) {
	requestIDStr := c.Param("requestId")
	requestID, err := uuid.Parse(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID format"})
		return
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ActionPromotionRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var sectID, callerRank string
	err = tx.QueryRow(`SELECT sect_id, member_rank FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID, &callerRank)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu không thuộc tông môn nào!"})
		return
	}

	if !canManageSect(callerRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Không đủ quyền hạn để duyệt yêu cầu thăng chức!"})
		return
	}

	// Fetch promotion request details
	var targetPlayerID uuid.UUID
	var reqSectID, currentRank, targetRank string
	err = tx.QueryRow(`SELECT player_id, sect_id, current_rank, target_rank FROM sect_promotion_requests WHERE id = $1`, requestID).Scan(&targetPlayerID, &reqSectID, &currentRank, &targetRank)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Yêu cầu thăng chức không tồn tại!"})
		return
	}

	if reqSectID != sectID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Yêu cầu thăng chức này thuộc tông môn khác!"})
		return
	}

	if sectRankValue(callerRank) <= sectRankValue(targetRank) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu không thể duyệt thăng chức lên cấp bậc ngang hoặc cao hơn mình!"})
		return
	}

	if strings.ToLower(req.Action) == "approve" {
		// Update member rank in sect_members
		_, err = tx.Exec(`UPDATE sect_members SET member_rank = $1 WHERE player_id = $2`, targetRank, targetPlayerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update member rank: " + err.Error()})
			return
		}

		// Delete request
		_, err = tx.Exec(`DELETE FROM sect_promotion_requests WHERE id = $1`, requestID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete request: " + err.Error()})
			return
		}
	} else if strings.ToLower(req.Action) == "reject" {
		// Refund contribution points
		var costPoints int
		switch targetRank {
		case "NOI_MON":
			costPoints = 100
		case "TINH_ANH":
			costPoints = 500
		case "CHAN_TRUYEN":
			costPoints = 2000
		case "TRUONG_LAO":
			costPoints = 10000
		case "HO_PHAP":
			costPoints = 30000
		}

		if costPoints > 0 {
			_, err = tx.Exec(`UPDATE sect_members SET contribution_points = contribution_points + $1 WHERE player_id = $2`, costPoints, targetPlayerID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to refund contribution points: " + err.Error()})
				return
			}
		}

		// Delete request
		_, err = tx.Exec(`DELETE FROM sect_promotion_requests WHERE id = $1`, requestID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete request: " + err.Error()})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Hành động không hợp lệ"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
		return
	}

	msg := "Đã bác bỏ yêu cầu thăng chức!"
	if strings.ToLower(req.Action) == "approve" {
		msg = "Đã đồng ý thăng chức cho đệ tử!"
	}

	c.JSON(http.StatusOK, gin.H{"message": msg})
}
