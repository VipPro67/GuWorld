package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type MailRequest struct {
	ReceiverUsername string `json:"receiver_username" binding:"required"`
	Content          string `json:"content" binding:"required,max=500"`
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
	if muted, muteErr := playerChatMuted(db.DB, authID); muteErr == nil && muted {
		c.JSON(http.StatusForbidden, gin.H{"error": chatMutedMessage})
		return
	}

	var req MailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Receiver and content are required"})
		return
	}
	if err := ValidateChatMessage(authID.String(), req.Content); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
	if gamefx.IsDeadState(&player) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể gửi tin nhắn!"})
		return
	}

	var receiverID uuid.UUID
	var receiverUsername string
	err = tx.QueryRow("SELECT id, username FROM players WHERE LOWER(username) = LOWER($1)", strings.TrimSpace(req.ReceiverUsername)).Scan(&receiverID, &receiverUsername)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người nhận!"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tìm người nhận"})
		return
	}
	if receiverID == player.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể gửi tin nhắn cho chính mình"})
		return
	}
	pairA, pairB := orderedPlayerPair(player.ID, receiverID)
	connectionResult, connectionErr := tx.ExecContext(c.Request.Context(), `
		INSERT INTO private_chat_connections (player_a, player_b) VALUES ($1,$2)
		ON CONFLICT DO NOTHING
	`, pairA, pairB)
	if connectionErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở kết nối truyền tin"})
		return
	}
	connectionRows, _ := connectionResult.RowsAffected()
	firstMessage := connectionRows > 0
	fee := int64(1) // each message costs one Primeval Stone
	if firstMessage {
		fee++ // first message between a pair also establishes a paid connection
	}
	res, err := tx.ExecContext(c.Request.Context(), `UPDATE players SET primeval_stones=primeval_stones-$1 WHERE id=$2 AND primeval_stones >= $1`, fee, player.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể trừ Nguyên Thạch"})
		return
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch! Tin nhắn này cần %d đá.", fee)})
		return
	}

	var messageID uuid.UUID
	err = tx.QueryRowContext(c.Request.Context(), `
		INSERT INTO messages (sender_id, receiver_id, content)
		VALUES ($1, $2, $3)
		RETURNING id
	`, player.ID, receiverID, req.Content).Scan(&messageID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	var sentAt time.Time
	if err := db.DB.QueryRowContext(c.Request.Context(), `SELECT created_at FROM messages WHERE id=$1`, messageID).Scan(&sentAt); err != nil {
		sentAt = time.Now()
	}
	item := ChatItem{ID: messageID.String(), SenderID: player.ID.String(), SenderUsername: player.Username, SenderAvatar: player.AvatarURL, SenderRank: player.Rank, Content: req.Content, Channel: "private", CreatedAt: sentAt}
	event := gin.H{"message": item, "conversation_username": receiverUsername}
	_ = cache.PublishEventQuick("chat:private:"+player.ID.String(), event)
	_ = cache.PublishEventQuick("chat:private:"+receiverID.String(), gin.H{"message": item, "conversation_username": player.Username})
	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated)
	if firstMessage {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Đã kết nối riêng với %s và gửi tin nhắn (phí kết nối 1 + phí tin 1 Nguyên Thạch).", receiverUsername), "connection_fee": 1, "message_fee": 1, "message_id": messageID})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":        "Đã gửi tin nhắn riêng (phí 1 Nguyên Thạch).",
		"connection_fee": 0,
		"message_fee":    1,
		"message_id":     messageID,
	})
}

func orderedPlayerPair(a, b uuid.UUID) (uuid.UUID, uuid.UUID) {
	if a.String() < b.String() {
		return a, b
	}
	return b, a
}

type PrivateConversation struct {
	PlayerID        string    `json:"player_id"`
	Username        string    `json:"username"`
	Avatar          string    `json:"avatar"`
	LastMessage     string    `json:"last_message"`
	LastMessageAt   time.Time `json:"last_message_at"`
	UnreadCount     int       `json:"unread_count"`
	LastMessageMine bool      `json:"last_message_mine"`
}

func GetPrivateConversations(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		WITH relevant AS (
			SELECT m.*, CASE WHEN m.sender_id=$1 THEN m.receiver_id ELSE m.sender_id END AS other_id
			FROM messages m
			WHERE m.channel='private' AND (m.sender_id=$1 OR m.receiver_id=$1)
			  AND m.sender_id::text NOT IN ('00000000-0000-0000-0000-000000000000','11111111-1111-1111-1111-111111111111')
			  AND m.receiver_id::text NOT IN ('00000000-0000-0000-0000-000000000000','11111111-1111-1111-1111-111111111111')
		), ranked AS (
			SELECT r.*, ROW_NUMBER() OVER (PARTITION BY other_id ORDER BY created_at DESC, id DESC) AS rn,
				COUNT(*) FILTER (WHERE receiver_id=$1 AND is_read=FALSE) OVER (PARTITION BY other_id) AS unread
			FROM relevant r
		)
		SELECT p.id, p.username, COALESCE(p.avatar_url,''), ranked.content, ranked.created_at,
			ranked.unread, (ranked.sender_id=$1)
		FROM ranked JOIN players p ON p.id=ranked.other_id
		WHERE ranked.rn=1 ORDER BY ranked.created_at DESC LIMIT 100
	`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách hội thoại"})
		return
	}
	defer rows.Close()
	conversations := make([]PrivateConversation, 0)
	for rows.Next() {
		var item PrivateConversation
		if err := rows.Scan(&item.PlayerID, &item.Username, &item.Avatar, &item.LastMessage, &item.LastMessageAt, &item.UnreadCount, &item.LastMessageMine); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc hội thoại"})
			return
		}
		conversations = append(conversations, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc hội thoại"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

func GetPrivateThread(c *gin.Context) {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	username := c.Query("username")
	var otherID uuid.UUID
	var otherUsername, otherAvatar string
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT id,username,COALESCE(avatar_url,'') FROM players WHERE LOWER(username)=LOWER($1)`, strings.TrimSpace(username)).Scan(&otherID, &otherUsername, &otherAvatar)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người chơi"})
		return
	}
	if err != nil || otherID == playerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Người nhận không hợp lệ"})
		return
	}
	globalUUID := "00000000-0000-0000-0000-000000000000"
	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT * FROM (
			SELECT m.id,m.sender_id,p.username,COALESCE(p.avatar_url,''),COALESCE(p.selected_title,''),COALESCE(p.rank,0),m.content,m.channel,m.created_at
			FROM messages m JOIN players p ON p.id=m.sender_id
			WHERE m.channel='private' AND m.receiver_id <> $3
			  AND ((m.sender_id=$1 AND m.receiver_id=$2) OR (m.sender_id=$2 AND m.receiver_id=$1))
			ORDER BY m.created_at DESC LIMIT 50
		) thread ORDER BY created_at ASC
	`, playerID, otherID, globalUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải hội thoại"})
		return
	}
	messages := make([]ChatItem, 0)
	for rows.Next() {
		var item ChatItem
		if err := rows.Scan(&item.ID, &item.SenderID, &item.SenderUsername, &item.SenderAvatar, &item.SenderTitle, &item.SenderRank, &item.Content, &item.Channel, &item.CreatedAt); err != nil {
			rows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc hội thoại"})
			return
		}
		messages = append(messages, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi đọc hội thoại"})
		return
	}
	rows.Close()
	_, _ = db.DB.ExecContext(c.Request.Context(), `UPDATE messages SET is_read=TRUE WHERE channel='private' AND sender_id=$1 AND receiver_id=$2 AND is_read=FALSE`, otherID, playerID)
	c.JSON(http.StatusOK, gin.H{"username": otherUsername, "avatar": otherAvatar, "messages": messages})
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
	SenderRank     int       `json:"sender_rank"`
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

	code, res := ExecuteSendBaoHoangChat(c.Request.Context(), authID, req)
	c.JSON(code, res)
}

// ExecuteSendBaoHoangChat performs the core domain logic of sending a chat message,
// decoupled from HTTP context so it can be called directly by WebSockets or tests.
func ExecuteSendBaoHoangChat(ctx context.Context, authID uuid.UUID, req BaoHoangRequest) (int, gin.H) {
	if muted, muteErr := playerChatMuted(db.DB, authID); muteErr == nil && muted {
		return http.StatusForbidden, gin.H{"error": chatMutedMessage}
	}

	if req.Channel == "" {
		req.Channel = "world"
	}

	if err := ValidateChatMessage(authID.String(), req.Content); err != nil {
		return http.StatusBadRequest, gin.H{"error": err.Error()}
	}

	tx, err := db.DB.Begin()
	if err != nil {
		return http.StatusInternalServerError, gin.H{"error": "Database error"}
	}
	defer tx.Rollback()

	player, err := evaluateManaTx(tx, authID)
	if err != nil {
		return http.StatusInternalServerError, gin.H{"error": "Failed to load player"}
	}
	if gamefx.IsDeadState(&player) {
		return http.StatusForbidden, gin.H{"error": "Đạo hữu đã rơi vào cõi chết (hết thọ nguyên). Không thể gửi tin nhắn!"}
	}

	var fee int64
	var chatScope string
	switch req.Channel {
	case "world":
		fee = 10
	case "general":
		fee = 0
		hasConnectingHeaven := models.InventoryHasGuCapability(player.GuInventory, "CHAT_CONNECT", false)
		hasDivineSense := models.InventoryHasGuCapability(player.GuInventory, "CHAT_PERCEIVE", false)
		if !hasConnectingHeaven || !hasDivineSense {
			return http.StatusBadRequest, gin.H{"error": "Chat kênh Bảo Hoàng Thiên yêu cầu cần có Cổ trùng Thông Thiên Cổ (GU_CONNECTING_HEAVEN) và Thần Niệm Cổ (GU_DIVINE_SENSE) trong túi đồ!"}
		}
	case "sect":
		fee = 0
	case "region":
		fee = 0
		var region string
		var instanceID sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT current_region, map_instance_id::text FROM players WHERE id=$1`, player.ID).Scan(&region, &instanceID); err != nil || region == "" {
			return http.StatusConflict, gin.H{"error": "Không xác định được khu vực hiện tại để gửi chat"}
		}
		chatScope = region
		if instanceID.Valid && instanceID.String != "" {
			chatScope += ":" + instanceID.String
		}
	default:
		return http.StatusBadRequest, gin.H{"error": "Kênh chat không hợp lệ"}
	}

	if player.PrimevalStones < float64(fee) {
		return http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch! Cần %d Nguyên Thạch để gửi tin nhắn kênh này.", fee)}
	}

	globalUUID := "00000000-0000-0000-0000-000000000000"
	var sectID sql.NullString
	if req.Channel == "sect" {
		var sID string
		err = tx.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, player.ID).Scan(&sID)
		if err != nil {
			if err == sql.ErrNoRows {
				return http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa gia nhập tông môn nào, không thể gửi tin nhắn kênh tông môn!"}
			} else {
				return http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()}
			}
		}
		sectID.String = sID
		sectID.Valid = true
	}

	var insertedMessageID uuid.UUID
	err = tx.QueryRow(`
		INSERT INTO messages (sender_id, receiver_id, content, channel, sect_id, chat_scope)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, player.ID, globalUUID, req.Content, req.Channel, sectID, chatScope).Scan(&insertedMessageID)
	if err != nil {
		fmt.Println("Error inserting chat message:", err)
		return http.StatusInternalServerError, gin.H{"error": "Failed to send chat"}
	}

	if fee > 0 {
		res, err := tx.Exec("UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id = $2 AND primeval_stones >= $1", fee, player.ID)
		if err != nil {
			return http.StatusInternalServerError, gin.H{"error": "Failed to deduct stones"}
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			return http.StatusBadRequest, gin.H{"error": "Không đủ Nguyên Thạch để chat kênh này"}
		}
		player.PrimevalStones -= float64(fee)
	}

	qiGained := 0.0
	_, err = tx.Exec("UPDATE players SET primeval_stones = primeval_stones WHERE id = $1", player.ID)
	if err != nil {
		return http.StatusInternalServerError, gin.H{"error": "Failed to update human qi"}
	}

	if err := tx.Commit(); err != nil {
		return http.StatusInternalServerError, gin.H{"error": "Failed to commit chat"}
	}

	// Broadcast the new message efficiently (append-only)
	var newMsg ChatItem
	err = db.DB.QueryRow(`
		SELECT id, sender_id, username, COALESCE(avatar_url, ''), COALESCE(selected_title, ''), COALESCE(p.rank, 0), content, channel, created_at
		FROM messages m
		JOIN players p ON m.sender_id = p.id
		WHERE m.id = $1
	`, insertedMessageID).Scan(
		&newMsg.ID, &newMsg.SenderID, &newMsg.SenderUsername,
		&newMsg.SenderAvatar, &newMsg.SenderTitle, &newMsg.SenderRank,
		&newMsg.Content, &newMsg.Channel, &newMsg.CreatedAt,
	)
	if err == nil {
		if req.Channel == "region" {
			_ = cache.PublishEventQuick("chat:region:"+chatScope, chatAppendPayload{Message: newMsg})
		} else {
			notifyNewChatMessage(req.Channel, sectID, newMsg)
		}
	} else {
		// Fallback to full history on error
		if req.Channel == "region" {
			if messages, loadErr := loadBaoHoangMessagesContext(ctx, "region", sql.NullString{String: chatScope, Valid: true}); loadErr == nil {
				_ = cache.PublishEventQuick("chat:region:"+chatScope, chatEventPayload{Messages: messages})
			}
		} else {
			notifyBaoHoangMessages(req.Channel, sectID)
		}
	}
	PublishPlayerRealtimeEvent(player.ID, PlayerEventUpdated)

	msg := "Đã truyền tin nhắn thành công! Nhân Khí +1."
	if fee > 0 {
		msg = fmt.Sprintf("Đã truyền tin nhắn thành công! Trừ %d Nguyên Thạch phí truyền phát. Nhân Khí +%.0f.", fee, qiGained)
	}
	return http.StatusOK, gin.H{
		"message": msg,
	}
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
	} else if channel == "region" {
		sectID = sql.NullString{String: key, Valid: true}
		if c.Query("scope_only") == "1" {
			c.JSON(http.StatusOK, gin.H{"channel_key": "chat:region:" + key})
			return
		}
	}

	messages, err := loadBaoHoangMessagesContext(c.Request.Context(), channel, sectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load global chat"})
		return
	}

	response := gin.H{"messages": messages}
	if channel == "region" {
		response["channel_key"] = "chat:region:" + key
	}
	c.JSON(http.StatusOK, response)
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
