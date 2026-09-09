package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type chatEventPayload struct {
	Messages []ChatItem `json:"messages"`
}

func notifyBaoHoangMessages(channel string, sectID sql.NullString) {
	key := chatChannelKey(channel, sectID)
	messages, err := loadBaoHoangMessages(channel, sectID)
	if err != nil {
		return
	}

	payload := chatEventPayload{Messages: messages}
	_ = cache.PublishEvent(context.Background(), "chat:"+key, payload)
}

func chatChannelKey(channel string, sectID sql.NullString) string {
	if channel == "sect" && sectID.Valid {
		return fmt.Sprintf("sect:%s", sectID.String)
	}
	return channel
}

func resolveChatChannelKey(c *gin.Context, channel string) (string, bool) {
	if channel != "world" && channel != "general" && channel != "sect" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Kênh chat không hợp lệ"})
		return "", false
	}

	if channel == "world" {
		return chatChannelKey(channel, sql.NullString{}), true
	}

	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return "", false
	}

	if channel == "general" {
		tx, err := db.DB.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return "", false
		}
		defer tx.Rollback()
		var hasConnectingHeaven, hasDivineSense bool
		player, err := evaluateManaTx(tx, playerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load player"})
			return "", false
		}

		for _, gu := range player.GuInventory {
			if gu.GuID == "GU_CONNECTING_HEAVEN" {
				hasConnectingHeaven = true
			}
			if gu.GuID == "GU_DIVINE_SENSE" {
				hasDivineSense = true
			}
		}
		if !hasConnectingHeaven || !hasDivineSense {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu cần sở hữu Thông Thiên Cổ (GU_CONNECTING_HEAVEN) và Thần Niệm Cổ (GU_DIVINE_SENSE) để xem tin nhắn kênh Bảo Hoàng Thiên!"})
			return "", false
		}
		return chatChannelKey(channel, sql.NullString{}), true
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

func loadBaoHoangMessages(channel string, sectID sql.NullString) ([]ChatItem, error) {
	globalUUID := "00000000-0000-0000-0000-000000000000"
	var rows *sql.Rows
	var err error

	if channel == "sect" {
		rows, err = db.DB.Query(`
			SELECT * FROM (
				SELECT m.id, m.sender_id, p.username, p.avatar_url, COALESCE(p.selected_title, ''), m.content, m.created_at, m.channel
				FROM messages m
				JOIN players p ON m.sender_id = p.id
				WHERE m.receiver_id = $1 AND m.channel = $2 AND m.sect_id = $3
				ORDER BY m.created_at DESC
				LIMIT 50
			) sub
			ORDER BY sub.created_at ASC
		`, globalUUID, channel, sectID.String)
	} else {
		rows, err = db.DB.Query(`
			SELECT * FROM (
				SELECT m.id, m.sender_id, p.username, p.avatar_url, COALESCE(p.selected_title, ''), m.content, m.created_at, m.channel
				FROM messages m
				JOIN players p ON m.sender_id = p.id
				WHERE m.receiver_id = $1 AND m.channel = $2
				ORDER BY m.created_at DESC
				LIMIT 50
			) sub
			ORDER BY sub.created_at ASC
		`, globalUUID, channel)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []ChatItem{}
	for rows.Next() {
		var m ChatItem
		if errScan := rows.Scan(&m.ID, &m.SenderID, &m.SenderUsername, &m.SenderAvatar, &m.SenderTitle, &m.Content, &m.CreatedAt, &m.Channel); errScan == nil {
			messages = append(messages, m)
		} else {
			logger.S.Errorf("loadBaoHoangMessages scan error: %v", errScan)
		}
	}
	return messages, rows.Err()
}
