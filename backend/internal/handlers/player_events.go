package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

const (
	PlayerEventUpdated   = "player.updated"
	CombatEventStarted   = "combat.started"
	CombatEventEnded     = "combat.ended"
	TradeEventUpdated    = "trade.updated"
	ManaEventSnapshot    = "mana.snapshot"
	PlayerEventHeartbeat = "player.heartbeat"
	SessionEventKicked   = "session.kicked"
)

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
	PrimevalStones      int64               `json:"primeval_stones"`
	ImmortalStones      int64               `json:"immortal_stones"`
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

// ClearPendingPlayerQueues removes a player's ID from async batch queues so uncommitted stale snapshots don't overwrite immediate DB commits.
func ClearPendingPlayerQueues(playerID string) {
	if playerID == "" {
		return
	}
	_ = cache.RemovePendingIDs("player_stats", playerID)
	_ = cache.RemovePendingIDs("player_movement", playerID)
	_ = cache.Delete("player_snapshot:" + playerID)
}

// PublishPlayerSnapshot publishes a pre-loaded player snapshot to Redis Pub/Sub for instant WebSocket broadcast without hitting PostgreSQL.
func PublishPlayerSnapshot(player *models.Player, eventType string) {
	if player == nil {
		return
	}
	snapshot := playerRealtimeSnapshot(*player)
	_ = cache.SetJSON("player_snapshot:"+player.ID.String(), snapshot, 5*time.Minute)
	_ = cache.PublishEvent(context.Background(), "player:"+player.ID.String(), PlayerRealtimeEvent{
		Type: eventType,
		Data: snapshot,
	})
}

// PublishPlayerRealtimeEvent publishes a player realtime snapshot to Redis for WebSocket broadcast.
func PublishPlayerRealtimeEvent(playerID uuid.UUID, eventType string) {
	snapshot, err := LoadPlayerRealtimeSnapshot(playerID)
	if err != nil {
		return
	}
	_ = cache.SetJSON("player_snapshot:"+playerID.String(), snapshot, 5*time.Minute)
	_ = cache.PublishEvent(context.Background(), "player:"+playerID.String(), PlayerRealtimeEvent{
		Type: eventType,
		Data: snapshot,
	})
}

func LoadPlayerRealtimeSnapshot(playerID uuid.UUID) (PlayerRealtimeSnapshot, error) {
	var cached PlayerRealtimeSnapshot
	if err := cache.GetJSON("player_snapshot:"+playerID.String(), &cached); err == nil && cached.ID != "" {
		return cached, nil
	}

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

	snapshot := playerRealtimeSnapshot(player)
	_ = cache.SetJSON("player_snapshot:"+playerID.String(), snapshot, 5*time.Minute)
	return snapshot, nil
}

func playerRealtimeSnapshot(player models.Player) PlayerRealtimeSnapshot {
	return PlayerRealtimeSnapshot{
		ID:                  player.ID.String(),
		CurrentMana:         player.CurrentMana,
		MaxMana:             player.MaxMana,
		LastManaUpdate:      player.LastManaUpdate,
		TimeSpeedMultiplier: player.TimeSpeedMultiplier + models.EquippedTimeSpeedBonus(player.GuInventory),
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

// SendSessionKickedEvent sends a session.kicked event to the player's WebSocket connection when invalidated.
func SendSessionKickedEvent(playerID uuid.UUID) {
	notify.Send(db.DB, playerID, "session.kicked", "Phiên đăng nhập đã hết hạn", "Tài khoản đã đăng nhập từ thiết bị khác")
	_ = cache.PublishEvent(context.Background(), "player:"+playerID.String(), PlayerRealtimeEvent{
		Type: SessionEventKicked,
	})
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
