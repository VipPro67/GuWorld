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
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type chatEventPayload struct {
	Messages []ChatItem `json:"messages"`
}

// chatAppendPayload is for append-only updates (single new message).
type chatAppendPayload struct {
	Message ChatItem `json:"message"`
}

func notifyBaoHoangMessages(channel string, sectID sql.NullString) {
	key := chatChannelKey(channel, sectID)
	messages, err := loadBaoHoangMessages(channel, sectID)
	if err != nil {
		return
	}

	payload := chatEventPayload{Messages: messages}
	_ = cache.PublishEventQuick("chat:"+key, payload)
}

// notifyNewChatMessage publishes only the new message for efficient append-only updates.
// Use this for live chat updates instead of reloading the entire history.
func notifyNewChatMessage(channel string, sectID sql.NullString, message ChatItem) {
	key := chatChannelKey(channel, sectID)
	payload := chatAppendPayload{Message: message}
	_ = cache.PublishEventQuick("chat:"+key, payload)
}

func chatChannelKey(channel string, sectID sql.NullString) string {
	if channel == "region" && sectID.Valid {
		return sectID.String
	}
	if channel == "sect" && sectID.Valid {
		return fmt.Sprintf("sect:%s", sectID.String)
	}
	return channel
}

func resolveChatChannelKey(c *gin.Context, channel string) (string, bool) {
	if channel != "world" && channel != "general" && channel != "sect" && channel != "region" {
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
	if channel == "region" {
		key, err := currentRegionChatKey(c.Request.Context(), playerID.String())
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Không xác định được khu vực chat hiện tại"})
			return "", false
		}
		return chatChannelKey(channel, sql.NullString{String: key, Valid: true}), true
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

		hasConnectingHeaven = models.InventoryHasGuCapability(player.GuInventory, "CHAT_CONNECT", false)
		hasDivineSense = models.InventoryHasGuCapability(player.GuInventory, "CHAT_PERCEIVE", false)
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

// currentRegionChatKey is derived from the player's persisted location so clients
// cannot read or write another area's chat by selecting an arbitrary region.
func currentRegionChatKey(ctx context.Context, playerID string) (string, error) {
	if entry, ok := worldPosIndex.Get(playerID); ok && entry.Region != "" {
		if entry.InstanceID != "" {
			return entry.Region + ":" + entry.InstanceID, nil
		}
		return entry.Region, nil
	}

	var region string
	var instanceID sql.NullString
	err := db.DB.QueryRowContext(ctx, `SELECT current_region, map_instance_id::text FROM players WHERE id=$1`, playerID).Scan(&region, &instanceID)
	if err != nil {
		return "", err
	}
	if region == "" {
		return "", fmt.Errorf("player has no region")
	}
	if instanceID.Valid && instanceID.String != "" {
		return region + ":" + instanceID.String, nil
	}
	return region, nil
}

// CurrentRegionChatScope exposes the authoritative chat scope to the WebSocket
// layer for subscription authorization.
func CurrentRegionChatScope(playerID string) (string, error) {
	return currentRegionChatKey(context.Background(), playerID)
}

func loadBaoHoangMessages(channel string, sectID sql.NullString) ([]ChatItem, error) {
	return loadBaoHoangMessagesContext(context.Background(), channel, sectID)
}

func loadBaoHoangMessagesContext(ctx context.Context, channel string, sectID sql.NullString) ([]ChatItem, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	globalUUID := "00000000-0000-0000-0000-000000000000"
	var rows *sql.Rows
	var err error

	if channel == "region" {
		rows, err = db.DB.QueryContext(ctx, `
			SELECT * FROM (
				SELECT m.id, m.sender_id, p.username, p.avatar_url, COALESCE(p.selected_title, ''), COALESCE(p.rank, 0), m.content, m.created_at, m.channel
				FROM messages m JOIN players p ON m.sender_id = p.id
				WHERE m.channel = 'region' AND m.chat_scope = $1
				ORDER BY m.created_at DESC LIMIT 50
			) sub ORDER BY sub.created_at ASC`, sectID.String)
	} else if channel == "sect" {
		rows, err = db.DB.QueryContext(ctx, `
			SELECT * FROM (
				SELECT m.id, m.sender_id, p.username, p.avatar_url, COALESCE(p.selected_title, ''), COALESCE(p.rank, 0), m.content, m.created_at, m.channel
				FROM messages m
				JOIN players p ON m.sender_id = p.id
				WHERE m.receiver_id = $1 AND m.channel = $2 AND m.sect_id = $3
				ORDER BY m.created_at DESC
				LIMIT 50
			) sub
			ORDER BY sub.created_at ASC
		`, globalUUID, channel, sectID.String)
	} else {
		rows, err = db.DB.QueryContext(ctx, `
			SELECT * FROM (
				SELECT m.id, m.sender_id, p.username, p.avatar_url, COALESCE(p.selected_title, ''), COALESCE(p.rank, 0), m.content, m.created_at, m.channel
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
		if errScan := rows.Scan(&m.ID, &m.SenderID, &m.SenderUsername, &m.SenderAvatar, &m.SenderTitle, &m.SenderRank, &m.Content, &m.CreatedAt, &m.Channel); errScan == nil {
			messages = append(messages, m)
		} else {
			logger.S.Errorf("loadBaoHoangMessages scan error: %v", errScan)
		}
	}
	return messages, rows.Err()
}
