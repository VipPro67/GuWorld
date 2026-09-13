package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
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
	if player.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể gửi tin nhắn!"})
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

	// Deduct stone and add human qi atomically
	player.PrimevalStones -= 1
	qiGained := 0.0
	_, err = tx.Exec("UPDATE players SET primeval_stones = GREATEST(0, primeval_stones - 1) WHERE id = $1", player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct primeval stones"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated)
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Truyền tin thành công! Cổ trùng đã cất cánh bay đi. Nhân Khí +%.0f.", qiGained),
	})
}

func GetInbox(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
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
	Content string `json:"content" binding:"required,max=500"`
	Channel string `json:"channel"`
}

type ChatItem struct {
	ID             string    `json:"id"`
	SenderID       string    `json:"sender_id"`
	SenderUsername string    `json:"sender_username"`
	SenderAvatar   string    `json:"sender_avatar"`
	SenderTitle    string    `json:"sender_title"`
	Content        string    `json:"content"`
	Channel        string    `json:"channel"`
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

	if req.Channel == "" {
		req.Channel = "world"
	}

	if err := ValidateChatMessage(authID.String(), req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	if player.IsDead {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể gửi tin nhắn!"})
		return
	}

	var fee int64
	switch req.Channel {
	case "world":
		fee = 10
	case "general":
		fee = 0
		hasConnectingHeaven := false
		hasDivineSense := false
		for _, gu := range player.GuInventory {
			if gu.GuID == "GU_CONNECTING_HEAVEN" {
				hasConnectingHeaven = true
			}
			if gu.GuID == "GU_DIVINE_SENSE" {
				hasDivineSense = true
			}
		}
		if !hasConnectingHeaven || !hasDivineSense {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chat kênh Bảo Hoàng Thiên yêu cầu cần có Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) và Thần Niệm Cổ (GU_DIVINE_SENSE) trong túi đồ!"})
			return
		}
	case "sect":
		fee = 0
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kênh chat không hợp lệ"})
		return
	}

	if player.PrimevalStones < fee {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch! Cần %d Nguyên Thạch để gửi tin nhắn kênh này.", fee)})
		return
	}

	globalUUID := "00000000-0000-0000-0000-000000000000"
	var sectID sql.NullString
	if req.Channel == "sect" {
		var sID string
		err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, player.ID).Scan(&sID)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa gia nhập tông môn nào, không thể gửi tin nhắn kênh tông môn!"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			}
			return
		}
		sectID.String = sID
		sectID.Valid = true
	}

	_, err = tx.Exec(`
		INSERT INTO messages (sender_id, receiver_id, content, channel, sect_id)
		VALUES ($1, $2, $3, $4, $5)
	`, player.ID, globalUUID, req.Content, req.Channel, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send chat"})
		fmt.Println("Error inserting chat message:", err)
		return
	}

	if fee > 0 {
		res, err := tx.Exec("UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2 AND primeval_stones >= $1", fee, player.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct stones"})
			return
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch để chat kênh này"})
			return
		}
		player.PrimevalStones -= fee
	}

	qiGained := 0.0
	_, err = tx.Exec("UPDATE players SET primeval_stones = primeval_stones WHERE id = $1", player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update human qi"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit chat"})
		return
	}

	// Broadcast the new message efficiently (append-only)
	var newMsg ChatItem
	err = db.DB.QueryRow(`
		SELECT id, sender_id, username, COALESCE(avatar_url, ''), COALESCE(selected_title, ''), content, channel, created_at
		FROM messages m
		JOIN players p ON m.sender_id = p.id
		WHERE m.id = (SELECT id FROM messages ORDER BY created_at DESC LIMIT 1)
		  AND m.channel = $1
	`, req.Channel).Scan(
		&newMsg.ID, &newMsg.SenderID, &newMsg.SenderUsername,
		&newMsg.SenderAvatar, &newMsg.SenderTitle,
		&newMsg.Content, &newMsg.Channel, &newMsg.CreatedAt,
	)
	if err == nil {
		notifyNewChatMessage(req.Channel, sectID, newMsg)
	} else {
		// Fallback to full history on error
		notifyBaoHoangMessages(req.Channel, sectID)
	}
	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated)

	msg := "Đã truyền tin nhắn thành công! Nhân Khí +1."
	if fee > 0 {
		msg = fmt.Sprintf("Đã truyền tin nhắn thành công! Trừ %d Nguyên Thạch phí truyền phát. Nhân Khí +%.0f.", fee, qiGained)
	}
	c.JSON(http.StatusOK, gin.H{
		"message": msg,
	})
}

func GetBaoHoangMessages(c *gin.Context) {
	channel := c.DefaultQuery("channel", "world")
	key, ok := resolveChatChannelKey(c, channel)
	if !ok {
		return
	}

	sectID := sql.NullString{}
	if channel == "sect" {
		sectID = sql.NullString{String: key[len("sect:"):], Valid: true}
	}

	messages, err := loadBaoHoangMessages(channel, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load global chat"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// SendSystemMessage broadcasts a system message from NPC "Hệ thống"
func SendSystemMessage(channel string, sectID sql.NullString, content string) error {
	globalUUID := "00000000-0000-0000-0000-000000000000"
	systemUUID := "11111111-1111-1111-1111-111111111111"

	_, err := db.DB.Exec(`
		INSERT INTO messages (sender_id, receiver_id, content, channel, sect_id)
		VALUES ($1, $2, $3, $4, $5)
	`, systemUUID, globalUUID, content, channel, sectID)
	if err != nil {
		return err
	}

	notifyBaoHoangMessages(channel, sectID)
	return nil
}

// BroadcastImmortalGuLoot sends a world chat announcement and system notification when a player loots an Immortal Gu.
func BroadcastImmortalGuLoot(playerID uuid.UUID, playerName, bossName, guID string, rank int) {
	guSpec, _ := models.GetGuAny(guID)
	guName := guID
	if guSpec.Name != "" {
		guName = guSpec.Name
	}
	if rank <= 0 {
		rank = guSpec.Rank
	}

	content := fmt.Sprintf("【THIÊN HẠ CHẤN ĐỘNG】 Chúc mừng Cổ sư [%s] trong khi đánh bại [%s] đã thu phục thành công Tiên Cổ [%s] (Chuyển %d)! Tiên Cổ độc nhất vô nhị đã có chủ!", playerName, bossName, guName, rank)
	_ = SendSystemMessage("world", sql.NullString{}, content)

	if playerID != uuid.Nil {
		notify.Send(db.DB, playerID, "IMMORTAL_GU_LOOT", "Thu Phục Tiên Cổ!", fmt.Sprintf("Chúc mừng đạo hữu đã đánh bại %s và thu phục thành công Tiên Cổ [%s] (Chuyển %d)!", bossName, guName, rank))
	}
}
