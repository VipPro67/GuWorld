package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type MailRequest struct {
	ReceiverUsername string `json:"receiver_username" binding:"required"`
	Content          string `json:"content" binding:"required"`
}

type MailItem struct {
	ID             string    `json:"id"`
	SenderID       string    `json:"sender_id"`
	SenderUsername string    `json:"sender_username"`
	Content        string    `json:"content"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
}

func SendPrivateMail(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req MailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Receiver and content are required"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction failure"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch player status"})
		return
	}

	// Rule: Sending messenger gu consumes 1 Primeval Stone
	if player.PrimevalStones < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch để sài Cổ trùng truyền tin (Cần 1 Nguyên Thạch)!"})
		return
	}

	var receiverID uuid.UUID
	err = tx.QueryRow("SELECT id FROM players WHERE username = $1", req.ReceiverUsername).Scan(&receiverID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người nhận!"})
		return
	}

	// Insert message
	_, err = tx.Exec(`
		INSERT INTO messages (sender_id, receiver_id, content) 
		VALUES ($1, $2, $3)
	`, player.ID, receiverID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Deduct stone
	player.PrimevalStones -= 1
	_, err = tx.Exec("UPDATE players SET primeval_stones = $1 WHERE id = $2", player.PrimevalStones, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct primeval stones"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Truyền tin thành công! Cổ trùng đã cất cánh bay đi."})
}

func GetInbox(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT m.id, m.sender_id, p.username, m.content, m.is_read, m.created_at
		FROM messages m
		JOIN players p ON m.sender_id = p.id
		WHERE m.receiver_id = $1
		ORDER BY m.created_at DESC
		LIMIT 50
	`, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load messages"})
		return
	}
	defer rows.Close()

	mails := []MailItem{}
	for rows.Next() {
		var m MailItem
		if errScan := rows.Scan(&m.ID, &m.SenderID, &m.SenderUsername, &m.Content, &m.IsRead, &m.CreatedAt); errScan == nil {
			mails = append(mails, m)
		}
	}

	// Mark all as read
	_, _ = db.DB.Exec("UPDATE messages SET is_read = TRUE WHERE receiver_id = $1", authID)

	c.JSON(http.StatusOK, gin.H{"inbox": mails})
}

type BaoHoangRequest struct {
	Content string `json:"content" binding:"required"`
}

type ChatItem struct {
	ID             string    `json:"id"`
	SenderID       string    `json:"sender_id"`
	SenderUsername string    `json:"sender_username"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
}

func SendBaoHoangChat(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req BaoHoangRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Content is required"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
		return
	}

	// Restricted to Rank >= 6 (immortals)
	if player.Rank < 6 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Cổ Tiên (Rank >= 6) mới được phép kết nối thần niệm vào Hoàng Bảo Thiên!"})
		return
	}

	globalUUID := "00000000-0000-0000-0000-000000000000"

	_, err = tx.Exec(`
		INSERT INTO messages (sender_id, receiver_id, content)
		VALUES ($1, $2, $3)
	`, player.ID, globalUUID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send chat"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Đã truyền phát thần niệm lên Hoàng Bảo Thiên!"})
}

func GetBaoHoangMessages(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Verify player is Rank >= 6
	var rank int
	err := db.DB.QueryRow("SELECT rank FROM players WHERE id = $1", authID).Scan(&rank)
	if err != nil || rank < 6 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Chỉ có Cổ Tiên (Rank >= 6) mới được xem thần niệm Hoàng Bảo Thiên"})
		return
	}

	globalUUID := "00000000-0000-0000-0000-000000000000"

	rows, err := db.DB.Query(`
		SELECT m.id, m.sender_id, p.username, m.content, m.created_at
		FROM messages m
		JOIN players p ON m.sender_id = p.id
		WHERE m.receiver_id = $1
		ORDER BY m.created_at ASC
		LIMIT 50
	`, globalUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load global chat"})
		return
	}
	defer rows.Close()

	messages := []ChatItem{}
	for rows.Next() {
		var m ChatItem
		if errScan := rows.Scan(&m.ID, &m.SenderID, &m.SenderUsername, &m.Content, &m.CreatedAt); errScan == nil {
			messages = append(messages, m)
		}
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}
