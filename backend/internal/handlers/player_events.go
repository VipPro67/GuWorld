package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const (
	PlayerEventUpdated   = "player.updated"
	CombatEventStarted   = "combat.started"
	CombatEventEnded     = "combat.ended"
	TradeEventUpdated    = "trade.updated"
	ManaEventSnapshot    = "mana.snapshot"
	PlayerEventHeartbeat = "player.heartbeat"
)

const playerEventsHeartbeatInterval = 60 * time.Second

type PlayerRealtimeSnapshot struct {
	ID                  string              `json:"id"`
	CurrentMana         float64             `json:"current_mana"`
	MaxMana             float64             `json:"max_mana"`
	LastManaUpdate      time.Time           `json:"last_mana_update"`
	TimeSpeedMultiplier float64             `json:"time_speed_multiplier"`
	ActiveCombat        *models.CombatState `json:"active_combat"`
	ApertureStress      float64             `json:"aperture_stress"`
	HP                  float64             `json:"hp"`
	BaseHP              float64             `json:"base_hp"`
	Rank                int                 `json:"rank"`
	SubRank             int                 `json:"sub_rank"`
	PrimevalStones      float64             `json:"primeval_stones"`
	ImmortalStones      float64             `json:"immortal_stones"`
	WallProgress        float64             `json:"wall_progress"`
	HumanQi             float64             `json:"human_qi"`
	Lifespan            float64             `json:"lifespan"`
	X                   int                 `json:"x"`
	Y                   int                 `json:"y"`
	CurrentRegion       string              `json:"current_region"`
	ApertureStressLevel string              `json:"aperture_stress_level,omitempty"`
}

type PlayerRealtimeEvent struct {
	Type string                 `json:"type"`
	Data PlayerRealtimeSnapshot `json:"data"`
}

// StreamPlayerEvents exposes a server-to-client realtime stream for authenticated players.
// It sends an initial minimal player snapshot immediately and only performs sparse heartbeat
// syncs afterwards so each connection does not repeatedly call evaluateManaTx in a tight loop.
func StreamPlayerEvents(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Header().Set("Content-Encoding", "identity")

	if err := writePlayerSnapshotEvent(c, playerID, ManaEventSnapshot); err != nil {
		writeSSEError(c, err)
		flusher.Flush()
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(playerEventsHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			db.DB.Exec(`UPDATE players SET last_activity = NOW() WHERE id = $1`, playerID)
			if err := writePlayerSnapshotEvent(c, playerID, PlayerEventHeartbeat); err != nil {
				writeSSEError(c, err)
				flusher.Flush()
				return
			}
			flusher.Flush()
		}
	}
}

func writePlayerSnapshotEvent(c *gin.Context, playerID uuid.UUID, eventType string) error {
	snapshot, err := loadPlayerRealtimeSnapshot(playerID)
	if err != nil {
		return err
	}
	return writeSSE(c, eventType, PlayerRealtimeEvent{Type: eventType, Data: snapshot})
}

func loadPlayerRealtimeSnapshot(playerID uuid.UUID) (PlayerRealtimeSnapshot, error) {
	var player models.Player
	var activeCombatBytes []byte
	err := db.DB.QueryRow(`
		SELECT p.id, p.username, p.rank, p.sub_rank, p.current_mana, p.max_mana, p.last_mana_update,
			p.time_speed_multiplier, p.active_combat, p.aperture_stress, p.hp, p.base_hp,
			p.primeval_stones, p.immortal_stones, p.wall_progress, p.human_qi,
			p.lifespan, p.x, p.y, p.current_region, p.path_masteries,
			`+db.GuInventoryAggregateEquipped("p.id")+`, COALESCE(ia.dao_marks, '{}'::jsonb)
		FROM players p
		LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
		WHERE p.id = $1
	`, playerID).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.CurrentMana, &player.MaxMana, &player.LastManaUpdate,
		&player.TimeSpeedMultiplier, &activeCombatBytes, &player.ApertureStress,
		&player.HP, &player.BaseHP, &player.PrimevalStones, &player.ImmortalStones,
		&player.WallProgress, &player.HumanQi, &player.Lifespan,
		&player.X, &player.Y, &player.CurrentRegion, &player.PathMasteries,
		&player.GuInventory, &player.DaoMarks,
	)
	if err != nil {
		return PlayerRealtimeSnapshot{}, err
	}

	if len(activeCombatBytes) > 0 && string(activeCombatBytes) != "null" {
		var cs models.CombatState
		if err := json.Unmarshal(activeCombatBytes, &cs); err == nil {
			player.ActiveCombat = &cs
		}
	}
	player.BaseHP = gamefx.EffectiveMaxHP(&player)

	return playerRealtimeSnapshot(player), nil
}

func playerRealtimeSnapshot(player models.Player) PlayerRealtimeSnapshot {
	return PlayerRealtimeSnapshot{
		ID:                  player.ID.String(),
		CurrentMana:         player.CurrentMana,
		MaxMana:             player.MaxMana,
		LastManaUpdate:      player.LastManaUpdate,
		TimeSpeedMultiplier: player.TimeSpeedMultiplier,
		ActiveCombat:        player.ActiveCombat,
		ApertureStress:      player.ApertureStress,
		HP:                  player.HP,
		BaseHP:              player.BaseHP,
		Rank:                player.Rank,
		SubRank:             player.SubRank,
		PrimevalStones:      player.PrimevalStones,
		ImmortalStones:      player.ImmortalStones,
		WallProgress:        player.WallProgress,
		HumanQi:             player.HumanQi,
		Lifespan:            player.Lifespan,
		X:                   player.X,
		Y:                   player.Y,
		CurrentRegion:       player.CurrentRegion,
		ApertureStressLevel: apertureStressLevel(player.ApertureStress),
	}
}

func writeSSE(c *gin.Context, eventName string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", eventName, encoded)
	return err
}

func writeSSEError(c *gin.Context, err error) {
	_ = writeSSE(c, "error", gin.H{"error": err.Error()})
}

func apertureStressLevel(stress float64) string {
	switch {
	case stress >= 80:
		return "critical"
	case stress >= 50:
		return "high"
	case stress >= 20:
		return "medium"
	default:
		return "low"
	}
}

// StreamUnifiedEvents combines player events, trade events, and chat channel messages into a single connection.
func StreamUnifiedEvents(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming unsupported"})
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache, no-transform")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Header().Set("Content-Encoding", "identity")

	// 1. Subscribe to Trade
	tradeCh := subscribeTrade(playerID)
	defer unsubscribeTrade(playerID, tradeCh)

	// 2. Resolve Sect for Sect Chat
	var sectID string
	err := db.DB.QueryRow(`SELECT sect_id FROM sect_members WHERE player_id = $1`, playerID).Scan(&sectID)
	hasSect := err == nil && sectID != ""

	// 3. Subscribe to Chat Channels
	worldCh := subscribeChat(chatChannelKey("world", sql.NullString{}))
	defer unsubscribeChat(chatChannelKey("world", sql.NullString{}), worldCh)

	var sectCh chan chatEventPayload
	if hasSect {
		sectCh = subscribeChat(chatChannelKey("sect", sql.NullString{String: sectID, Valid: true}))
		defer unsubscribeChat(chatChannelKey("sect", sql.NullString{String: sectID, Valid: true}), sectCh)
	}

	// Check eligibility for General Chat: must have GU_CONNECTING_HEAVEN and GU_DIVINE_SENSE
	tx, err := db.DB.Begin()
	var hasConnectingHeaven, hasDivineSense bool
	if err == nil {
		player, errEval := evaluateManaTx(tx, playerID)
		if errEval == nil {
			for _, gu := range player.GuInventory {
				if gu.GuID == "GU_CONNECTING_HEAVEN" && gu.Status != "DEAD" {
					hasConnectingHeaven = true
				}
				if gu.GuID == "GU_DIVINE_SENSE" && gu.Status != "DEAD" {
					hasDivineSense = true
				}
			}
		}
		_ = tx.Rollback()
	}

	var generalCh chan chatEventPayload
	if hasConnectingHeaven && hasDivineSense {
		generalCh = subscribeChat(chatChannelKey("general", sql.NullString{}))
		defer unsubscribeChat(chatChannelKey("general", sql.NullString{}), generalCh)
	}

	// Send initial data to initialize client state
	// Player snapshot
	if err := writePlayerSnapshotEvent(c, playerID, ManaEventSnapshot); err != nil {
		writeSSEError(c, err)
		flusher.Flush()
		return
	}

	// Trade state
	if err := writeTradeEvent(c, playerID); err != nil {
		// Log warning but don't fail connection
	}

	// Chat history for subscribed channels
	if err := writeChatEvent(c, "world", chatChannelKey("world", sql.NullString{})); err != nil {
		// Log warning but don't fail connection
	}
	if hasSect {
		if err := writeChatEvent(c, "sect", chatChannelKey("sect", sql.NullString{String: sectID, Valid: true})); err != nil {
			// Log warning
		}
	}
	if hasConnectingHeaven && hasDivineSense {
		if err := writeChatEvent(c, "general", chatChannelKey("general", sql.NullString{})); err != nil {
			// Log warning
		}
	}

	flusher.Flush()

	ticker := time.NewTicker(playerEventsHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return

		case <-ticker.C:
			db.DB.Exec(`UPDATE players SET last_activity = NOW() WHERE id = $1`, playerID)
			if err := writePlayerSnapshotEvent(c, playerID, PlayerEventHeartbeat); err != nil {
				writeSSEError(c, err)
				flusher.Flush()
				return
			}
			flusher.Flush()

		case payload := <-tradeCh:
			if err := writeSSE(c, "trade.updated", payload); err != nil {
				return
			}
			flusher.Flush()

		case payload := <-worldCh:
			if err := writeSSE(c, "chat.messages.world", payload); err != nil {
				return
			}
			flusher.Flush()

		case payload := <-generalCh:
			if err := writeSSE(c, "chat.messages.general", payload); err != nil {
				return
			}
			flusher.Flush()

		case payload := <-sectCh:
			if err := writeSSE(c, "chat.messages.sect", payload); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
