package handlers

import (
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
	PositionEventUpdated = "player.position"
	CombatStateEvent     = "player.combat_state"
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

// PlayerDeltaEvent sends only changed fields for efficient bandwidth usage.
type PlayerDeltaEvent struct {
	Type string         `json:"type"`
	Data PlayerDeltaData `json:"data"`
}

// PlayerDeltaData contains only the fields that changed.
type PlayerDeltaData struct {
	X         int               `json:"x,omitempty"`
	Y         int               `json:"y,omitempty"`
	HP        *float64          `json:"hp,omitempty"`
	MaxHP     *float64          `json:"max_hp,omitempty"`
	CurrentMana *float64        `json:"current_mana,omitempty"`
	MaxMana   *float64          `json:"max_mana,omitempty"`
	ActiveCombat *models.CombatState `json:"active_combat,omitempty"`
	ApertureStress *float64      `json:"aperture_stress,omitempty"`
}

// ClearPendingPlayerQueues removes a player's ID from async batch queues so uncommitted stale snapshots don't overwrite immediate DB commits.
func ClearPendingPlayerQueues(playerID string) {
	if playerID == "" {
		return
	}
	_ = cache.RemovePendingIDs("player_stats", playerID)
	_ = cache.RemovePendingIDs("player_movement", playerID)
	_ = cache.Delete("player_snapshot:" + playerID)
	_ = cache.Delete("player_movement:" + playerID)
}

// PublishPlayerSnapshot publishes a pre-loaded player snapshot to Redis Pub/Sub for instant WebSocket broadcast without hitting PostgreSQL.
func PublishPlayerSnapshot(player *models.Player, eventType string) {
	if player == nil {
		return
	}
	snapshot := playerRealtimeSnapshot(*player)
	_ = cache.SetJSON("player_snapshot:"+player.ID.String(), snapshot, 5*time.Minute)
	_ = cache.PublishEventQuick("player:"+player.ID.String(), PlayerRealtimeEvent{
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
	_ = cache.PublishEventQuick("player:"+playerID.String(), PlayerRealtimeEvent{
		Type: eventType,
		Data: snapshot,
	})
}

// PublishPlayerPositionDelta publishes only position changes (minimal payload).
func PublishPlayerPositionDelta(playerID uuid.UUID, x, y int) {
	delta := PlayerDeltaEvent{
		Type: PositionEventUpdated,
		Data: PlayerDeltaData{
			X: x,
			Y: y,
		},
	}
	_ = cache.PublishEventQuick("player:"+playerID.String(), delta)
}

// PublishPlayerCombatDelta publishes combat state changes efficiently.
func PublishPlayerCombatDelta(playerID uuid.UUID, eventType string, combat *models.CombatState) {
	// Also update the full snapshot cache
	snapshot, err := LoadPlayerRealtimeSnapshot(playerID)
	if err != nil {
		return
	}
	_ = cache.SetJSON("player_snapshot:"+playerID.String(), snapshot, 5*time.Minute)

	delta := PlayerDeltaEvent{
		Type: eventType,
		Data: PlayerDeltaData{
			ActiveCombat: combat,
		},
	}
	_ = cache.PublishEventQuick("player:"+playerID.String(), delta)
}

// PublishPlayerHPDelta publishes HP changes (common during combat ticks).
func PublishPlayerHPDelta(playerID uuid.UUID, hp, maxHP float64) {
	delta := PlayerDeltaEvent{
		Type: "player.hp",
		Data: PlayerDeltaData{
			HP:    &hp,
			MaxHP: &maxHP,
		},
	}
	_ = cache.PublishEventQuick("player:"+playerID.String(), delta)
}

// PublishPlayerManaDelta publishes mana changes (common during regen ticks).
func PublishPlayerManaDelta(playerID uuid.UUID, currentMana, maxMana float64) {
	delta := PlayerDeltaEvent{
		Type: "player.mana",
		Data: PlayerDeltaData{
			CurrentMana: &currentMana,
			MaxMana:     &maxMana,
		},
	}
	_ = cache.PublishEventQuick("player:"+playerID.String(), delta)
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
	_ = cache.PublishEventQuick("player:"+playerID.String(), PlayerRealtimeEvent{
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
