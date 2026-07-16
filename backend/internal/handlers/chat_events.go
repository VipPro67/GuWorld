package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type chatEventPayload struct {
	Messages []ChatItem `json:"messages"`
}

var chatEventHub = struct {
	sync.Mutex
	subs map[string]map[chan chatEventPayload]struct{}
}{subs: map[string]map[chan chatEventPayload]struct{}{}}

func SubscribeBaoHoangMessages(c *gin.Context) {
	channel := c.DefaultQuery("channel", "world")
	key, ok := resolveChatChannelKey(c, channel)
	if !ok {
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	ch := subscribeChat(key)
	defer unsubscribeChat(key, ch)

	if err := writeChatEvent(c, channel, key); err != nil {
		_ = writeSSE(c, "error", gin.H{"error": err.Error()})
		flusher.Flush()
		return
	}
	flusher.Flush()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case payload := <-ch:
			if err := writeSSE(c, "chat.messages", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func subscribeChat(key string) chan chatEventPayload {
	ch := make(chan chatEventPayload, 1)
	chatEventHub.Lock()
	defer chatEventHub.Unlock()
	if chatEventHub.subs[key] == nil {
		chatEventHub.subs[key] = map[chan chatEventPayload]struct{}{}
	}
	chatEventHub.subs[key][ch] = struct{}{}
	return ch
}

func unsubscribeChat(key string, ch chan chatEventPayload) {
	chatEventHub.Lock()
	defer chatEventHub.Unlock()
	delete(chatEventHub.subs[key], ch)
	if len(chatEventHub.subs[key]) == 0 {
		delete(chatEventHub.subs, key)
	}
	close(ch)
}

func notifyBaoHoangMessages(channel string, sectID sql.NullString) {
	key := chatChannelKey(channel, sectID)
	messages, err := loadBaoHoangMessages(channel, sectID)
	if err != nil {
		return
	}

	payload := chatEventPayload{Messages: messages}
	chatEventHub.Lock()
	defer chatEventHub.Unlock()
	for ch := range chatEventHub.subs[key] {
		select {
		case ch <- payload:
		default:
			select {
			case <-ch:
			default:
			}
			ch <- payload
		}
	}
}

func writeChatEvent(c *gin.Context, channel, key string) error {
	sectID := sql.NullString{}
	if channel == "sect" {
		sectID = sql.NullString{String: key[len("sect:"):], Valid: true}
	}
	messages, err := loadBaoHoangMessages(channel, sectID)
	if err != nil {
		return err
	}
	return writeSSE(c, "chat.messages", chatEventPayload{Messages: messages})
}

func resolveChatChannelKey(c *gin.Context, channel string) (string, bool) {
	if channel != "world" && channel != "general" && channel != "sect" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kênh chat không hợp lệ"})
		return "", false
	}

	if channel != "sect" {
		return chatChannelKey(channel, sql.NullString{}), true
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return "", false
	}

	var sectID string
	err := db.DB.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa gia nhập tông môn nào, không thể xem tin nhắn kênh tông môn!"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		}
		return "", false
	}

	return chatChannelKey(channel, sql.NullString{String: sectID, Valid: true}), true
}

func chatChannelKey(channel string, sectID sql.NullString) string {
	if channel == "sect" && sectID.Valid {
		return fmt.Sprintf("sect:%s", sectID.String)
	}
	return channel
}

func loadBaoHoangMessages(channel string, sectID sql.NullString) ([]ChatItem, error) {
	globalUUID := "00000000-0000-0000-0000-000000000000"
	var rows *sql.Rows
	var err error

	if channel == "sect" {
		rows, err = db.DB.Query(`
			SELECT m.id, m.sender_id, p.username, m.content, m.created_at, m.channel
			FROM messages m
			JOIN players p ON m.sender_id = p.id
			WHERE m.receiver_id = $1 AND m.channel = $2 AND m.sect_id = $3
			ORDER BY m.created_at ASC
			LIMIT 50
		`, globalUUID, channel, sectID.String)
	} else {
		rows, err = db.DB.Query(`
			SELECT m.id, m.sender_id, p.username, m.content, m.created_at, m.channel
			FROM messages m
			JOIN players p ON m.sender_id = p.id
			WHERE m.receiver_id = $1 AND m.channel = $2
			ORDER BY m.created_at ASC
			LIMIT 50
		`, globalUUID, channel)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []ChatItem{}
	for rows.Next() {
		var m ChatItem
		if errScan := rows.Scan(&m.ID, &m.SenderID, &m.SenderUsername, &m.Content, &m.CreatedAt, &m.Channel); errScan == nil {
			messages = append(messages, m)
		} else {
			log.Printf("loadBaoHoangMessages scan error: %v", errScan)
		}
	}
	return messages, rows.Err()
}
