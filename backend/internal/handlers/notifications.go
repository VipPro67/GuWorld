package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

// GetNotifications returns a paginated list of the authenticated player's
// notifications, newest first.
func GetNotifications(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	page := 1
	pageSize := 20
	if p := c.Query("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := c.Query("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}
	offset := (page - 1) * pageSize

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id, player_id, type, title, content, is_read, created_at
		FROM notifications
		WHERE player_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, playerID, pageSize, offset)
	if err != nil {
		logger.S.Errorf("notify: query failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load notifications"})
		return
	}
	defer rows.Close()

	notifs := []notify.Notification{}
	for rows.Next() {
		var n notify.Notification
		if err := rows.Scan(&n.ID, &n.PlayerID, &n.Type, &n.Title, &n.Content, &n.IsRead, &n.CreatedAt); err != nil {
			logger.S.Errorf("notify: scan failed: %v", err)
			continue
		}
		notifs = append(notifs, n)
	}
	c.JSON(http.StatusOK, gin.H{"notifications": notifs, "page": page, "pageSize": pageSize})
}

// GetUnreadNotificationsCount returns the number of unread notifications for
// the authenticated player.
func GetUnreadNotificationsCount(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var count int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM notifications WHERE player_id = $1 AND is_read = FALSE`, playerID).Scan(&count); err != nil {
		logger.S.Errorf("notify: unread count failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load unread count"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread": count})
}

// MarkNotificationsRead marks the given notification IDs as read for the
// authenticated player.
func MarkNotificationsRead(c *gin.Context) {
	var payload struct {
		IDs []string `json:"ids"`
	}
	if err := c.BindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if len(payload.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no IDs provided"})
		return
	}
	if _, err := db.DB.Exec(`UPDATE notifications SET is_read = TRUE WHERE player_id = $1 AND id = ANY($2)`, playerID, pq.Array(payload.IDs)); err != nil {
		logger.S.Errorf("notify: mark read failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// MarkAllNotificationsRead marks every notification as read for the
// authenticated player.
func MarkAllNotificationsRead(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	if _, err := db.DB.Exec(`UPDATE notifications SET is_read = TRUE WHERE player_id = $1 AND is_read = FALSE`, playerID); err != nil {
		logger.S.Errorf("notify: mark all read failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
