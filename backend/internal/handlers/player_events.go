package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
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
	PrimevalStones      int                 `json:"primeval_stones"`
	ImmortalStones      float64             `json:"immortal_stones"`
	WallProgress        float64             `json:"wall_progress"`
	HumanQi             float64             `json:"human_qi"`
	Lifespan            float64             `json:"lifespan"`
	MaxLifespan         float64             `json:"max_lifespan"`
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
	err := db.DB.QueryRow(`
		SELECT id, username, rank, sub_rank, current_mana, max_mana, last_mana_update,
			time_speed_multiplier, active_combat, aperture_stress, hp, base_hp,
			primeval_stones, immortal_stones, wall_progress, human_qi,
			lifespan, max_lifespan, x, y, current_region
		FROM players WHERE id = $1
	`, playerID).Scan(
		&player.ID, &player.Username, &player.Rank, &player.SubRank,
		&player.CurrentMana, &player.MaxMana, &player.LastManaUpdate,
		&player.TimeSpeedMultiplier, &player.ActiveCombat, &player.ApertureStress,
		&player.HP, &player.BaseHP, &player.PrimevalStones, &player.ImmortalStones,
		&player.WallProgress, &player.HumanQi, &player.Lifespan, &player.MaxLifespan,
		&player.X, &player.Y, &player.CurrentRegion,
	)
	if err != nil {
		return PlayerRealtimeSnapshot{}, err
	}

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
		MaxLifespan:         player.MaxLifespan,
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
