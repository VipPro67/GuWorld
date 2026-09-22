package handlers

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
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
	PrimevalStones      float64             `json:"primeval_stones"`
	ImmortalStones      float64             `json:"immortal_stones"`
	WallProgress        float64             `json:"wall_progress"`
	HumanQi             float64             `json:"human_qi"`
	Lifespan            float64             `json:"lifespan"`
	X                   int                 `json:"x"`
	Y                   int                 `json:"y"`
	CurrentRegion       string              `json:"current_region"`
	ApertureStressLevel string              `json:"aperture_stress_level,omitempty"`
	Version             int                 `json:"version"`
}

type PlayerRealtimeEvent struct {
	Type string                 `json:"type"`
	Data PlayerRealtimeSnapshot `json:"data"`
}

// PlayerDeltaEvent sends only changed fields for efficient bandwidth usage.
type PlayerDeltaEvent struct {
	Type string          `json:"type"`
	Data PlayerDeltaData `json:"data"`
}

// PlayerDeltaData contains only the fields that changed.
type PlayerDeltaData struct {
	X              int                 `json:"x,omitempty"`
	Y              int                 `json:"y,omitempty"`
	HP             *float64            `json:"hp,omitempty"`
	MaxHP          *float64            `json:"max_hp,omitempty"`
	CurrentMana    *float64            `json:"current_mana,omitempty"`
	MaxMana        *float64            `json:"max_mana,omitempty"`
	ActiveCombat   *models.CombatState `json:"active_combat"`
	ApertureStress *float64            `json:"aperture_stress,omitempty"`
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

// PublishPlayerSnapshot publishes a pre-loaded player snapshot to Redis Pub/Sub.
// The broad snapshot stays cheap, while active_combat is reconciled against its
// PostgreSQL source of truth before it can affect client navigation.
func PublishPlayerSnapshot(player *models.Player, eventType string) {
	if player == nil {
		return
	}
	snapshot := playerRealtimeSnapshot(*player)
	activeCombat, err := loadAuthoritativeActiveCombat(db.DB, player.ID)
	if err != nil {
		logger.S.Warnf("PublishPlayerSnapshot: active_combat reconciliation failed for player=%s event=%s: %v", player.ID, eventType, err)
		return
	}
	snapshot.ActiveCombat = activeCombat
	syncCombatCache(player.ID, activeCombat)
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
		logger.S.Warnf("PublishPlayerRealtimeEvent: snapshot load failed for player=%s event=%s: %v", playerID, eventType, err)
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
func PublishPlayerCombatDelta(playerID uuid.UUID, eventType string, _ *models.CombatState) {
	// Invalidate stale snapshot so LoadPlayerRealtimeSnapshot reloads from DB
	_ = cache.Delete("player_snapshot:" + playerID.String())
	snapshot, err := LoadPlayerRealtimeSnapshot(playerID)
	if err != nil {
		logger.S.Warnf("PublishPlayerCombatDelta: authoritative snapshot load failed for player=%s event=%s: %v", playerID, eventType, err)
		return
	}
	_ = cache.SetJSON("player_snapshot:"+playerID.String(), snapshot, 5*time.Minute)

	delta := PlayerDeltaEvent{
		Type: eventType,
		Data: PlayerDeltaData{
			// PostgreSQL is authoritative. The caller's copy may have been built
			// before the transaction committed (or may simply be stale), while the
			// snapshot above has just been reconciled with players.active_combat.
			ActiveCombat: snapshot.ActiveCombat,
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
		// Redis accelerates the broad realtime snapshot, but active_combat drives
		// navigation and must never outlive the database value. A narrow query
		// prevents a stale cached combat from forcing the UI back into combat.
		activeCombat, err := loadAuthoritativeActiveCombat(db.DB, playerID)
		if err != nil {
			return PlayerRealtimeSnapshot{}, err
		}
		cached.ActiveCombat = activeCombat
		syncCombatCache(playerID, activeCombat)
		_ = cache.SetJSON("player_snapshot:"+playerID.String(), cached, 5*time.Minute)
		return cached, nil
	}

	var player models.Player
	var activeCombatBytes []byte
	err := db.DB.QueryRow(`
		SELECT p.id, p.username, p.rank, p.sub_rank, p.current_mana, p.max_mana, p.last_mana_update,
			p.time_speed_multiplier, p.active_combat, p.aperture_stress, p.hp, p.base_hp,
			p.primeval_stones, p.immortal_stones, p.wall_progress, p.human_qi,
			p.lifespan, p.x, p.y, p.current_region, p.path_masteries, COALESCE(p.version, 0),
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
		&player.X, &player.Y, &player.CurrentRegion, &player.PathMasteries, &player.Version,
		&player.GuInventory, &player.DaoMarks,
	)
	if err != nil {
		return PlayerRealtimeSnapshot{}, err
	}

	player.ActiveCombat, err = decodeActiveCombat(activeCombatBytes)
	if err != nil {
		return PlayerRealtimeSnapshot{}, err
	}
	player.BaseHP = gamefx.EffectiveMaxHP(&player)

	snapshot := playerRealtimeSnapshot(player)
	syncCombatCache(playerID, player.ActiveCombat)
	_ = cache.SetJSON("player_snapshot:"+playerID.String(), snapshot, 5*time.Minute)
	return snapshot, nil
}

func syncCombatCache(playerID uuid.UUID, combat *models.CombatState) {
	if combat == nil {
		_ = cache.ClearCombatState(playerID.String())
		return
	}
	_ = cache.SetCombatState(playerID.String(), combat)
}

func loadAuthoritativeActiveCombat(q interface {
	QueryRow(query string, args ...any) *sql.Row
}, playerID uuid.UUID) (*models.CombatState, error) {
	var raw []byte
	if err := q.QueryRow(`SELECT active_combat FROM players WHERE id = $1`, playerID).Scan(&raw); err != nil {
		return nil, err
	}
	return decodeActiveCombat(raw)
}

func decodeActiveCombat(raw []byte) (*models.CombatState, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var combat models.CombatState
	if err := json.Unmarshal(raw, &combat); err != nil {
		return nil, err
	}
	if combat.IsEmpty() {
		return nil, nil
	}
	return &combat, nil
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
		Version:             player.Version,
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
