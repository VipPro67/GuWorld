package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sort"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

type PlayerEventDomain string

const (
	PlayerDomain        PlayerEventDomain = "player"
	GuInventoryDomain   PlayerEventDomain = "gu_inventory"
	ItemInventoryDomain PlayerEventDomain = "item_inventory"
	RecipesDomain       PlayerEventDomain = "recipes"
	CombatStatsDomain   PlayerEventDomain = "combat_stats"
)

var playerEventRevision atomic.Int64

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
	StateRevision       int64               `json:"state_revision"`
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
	Type           string                 `json:"type"`
	Revision       int64                  `json:"revision"`
	ChangedDomains []PlayerEventDomain    `json:"changed_domains"`
	Data           PlayerRealtimeSnapshot `json:"data"`
}

// PlayerDeltaEvent sends only changed fields for efficient bandwidth usage.
type PlayerDeltaEvent struct {
	Type           string              `json:"type"`
	StateRevision  int64               `json:"state_revision"`
	Revision       int64               `json:"revision"`
	ChangedDomains []PlayerEventDomain `json:"changed_domains"`
	Data           PlayerDeltaData     `json:"data"`
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
func PublishPlayerSnapshot(player *models.Player, eventType string, domains ...PlayerEventDomain) {
	if player == nil || db.DB == nil {
		return
	}
	// The caller commonly holds the pre-commit model. Reload after invalidating
	// the cache so a fresh revision is never paired with stale fields.
	_ = cache.Delete("player_snapshot:" + player.ID.String())
	snapshot, err := LoadPlayerRealtimeSnapshot(player.ID)
	if err != nil {
		logger.S.Warnf("PublishPlayerSnapshot: snapshot reload failed for player=%s event=%s: %v", player.ID, eventType, err)
		return
	}
	_ = cache.SetJSON("player_snapshot:"+player.ID.String(), snapshot, 5*time.Minute)
	_ = cache.PublishEventQuick("player:"+player.ID.String(), PlayerRealtimeEvent{
		Type: eventType, Revision: nextPlayerEventRevision(),
		ChangedDomains: normalizePlayerEventDomains(domains, PlayerDomain), Data: snapshot,
	})
}

// PublishPlayerRealtimeEvent publishes a player realtime snapshot to Redis for WebSocket broadcast.
func PublishPlayerRealtimeEvent(playerID uuid.UUID, eventType string, domains ...PlayerEventDomain) {
	ClearPendingPlayerQueues(playerID.String())
	snapshot, err := LoadPlayerRealtimeSnapshot(playerID)
	if err != nil {
		logger.S.Warnf("PublishPlayerRealtimeEvent: snapshot load failed for player=%s event=%s: %v", playerID, eventType, err)
		return
	}
	_ = cache.SetJSON("player_snapshot:"+playerID.String(), snapshot, 5*time.Minute)
	_ = cache.PublishEventQuick("player:"+playerID.String(), PlayerRealtimeEvent{
		Type: eventType, Revision: nextPlayerEventRevision(),
		ChangedDomains: normalizePlayerEventDomains(domains, PlayerDomain), Data: snapshot,
	})
}

// PublishPlayerPositionDelta publishes only position changes (minimal payload).
func PublishPlayerPositionDelta(playerID uuid.UUID, _, _ int) {
	snapshot, err := LoadPlayerRealtimeSnapshot(playerID)
	if err != nil {
		logger.S.Warnf("PublishPlayerPositionDelta: snapshot load failed for player=%s: %v", playerID, err)
		return
	}
	delta := PlayerDeltaEvent{
		StateRevision: snapshot.StateRevision,
		Type:          PositionEventUpdated, Revision: nextPlayerEventRevision(),
		ChangedDomains: []PlayerEventDomain{PlayerDomain},
		Data: PlayerDeltaData{
			X: snapshot.X,
			Y: snapshot.Y,
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
		StateRevision: snapshot.StateRevision,
		Type:          eventType, Revision: nextPlayerEventRevision(),
		ChangedDomains: []PlayerEventDomain{PlayerDomain},
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
	revision, err := loadPlayerStateRevision(db.DB, playerID)
	if err != nil {
		return
	}
	delta := PlayerDeltaEvent{
		StateRevision: revision,
		Type:          "player.hp", Revision: nextPlayerEventRevision(),
		ChangedDomains: []PlayerEventDomain{PlayerDomain},
		Data: PlayerDeltaData{
			HP:    &hp,
			MaxHP: &maxHP,
		},
	}
	_ = cache.PublishEventQuick("player:"+playerID.String(), delta)
}

// PublishPlayerManaDelta publishes mana changes (common during regen ticks).
func PublishPlayerManaDelta(playerID uuid.UUID, currentMana, maxMana float64) {
	revision, err := loadPlayerStateRevision(db.DB, playerID)
	if err != nil {
		return
	}
	delta := PlayerDeltaEvent{
		StateRevision: revision,
		Type:          "player.mana", Revision: nextPlayerEventRevision(),
		ChangedDomains: []PlayerEventDomain{PlayerDomain},
		Data: PlayerDeltaData{
			CurrentMana: &currentMana,
			MaxMana:     &maxMana,
		},
	}
	_ = cache.PublishEventQuick("player:"+playerID.String(), delta)
}

func nextPlayerEventRevision() int64 {
	floor := time.Now().UnixMilli() * 1000
	for {
		current := playerEventRevision.Load()
		next := current + 1
		if next < floor {
			next = floor
		}
		if playerEventRevision.CompareAndSwap(current, next) {
			return next
		}
	}
}

func normalizePlayerEventDomains(domains []PlayerEventDomain, fallback PlayerEventDomain) []PlayerEventDomain {
	if len(domains) == 0 {
		domains = []PlayerEventDomain{fallback}
	}
	seen := make(map[PlayerEventDomain]struct{}, len(domains))
	result := make([]PlayerEventDomain, 0, len(domains))
	for _, domain := range domains {
		if domain == "" {
			continue
		}
		if _, exists := seen[domain]; exists {
			continue
		}
		seen[domain] = struct{}{}
		result = append(result, domain)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func LoadPlayerRealtimeSnapshot(playerID uuid.UUID) (PlayerRealtimeSnapshot, error) {
	if db.DB == nil {
		return PlayerRealtimeSnapshot{}, errors.New("database not initialized")
	}
	var cached PlayerRealtimeSnapshot
	if err := cache.GetJSON("player_snapshot:"+playerID.String(), &cached); err == nil && cached.ID != "" {
		// Redis accelerates the broad realtime snapshot, but active_combat drives
		// navigation and must never outlive the database value. A narrow query
		// prevents a stale cached combat from forcing the UI back into combat.
		activeCombat, revision, err := loadAuthoritativePlayerMarker(db.DB, playerID)
		if err != nil {
			return PlayerRealtimeSnapshot{}, err
		}
		if cached.StateRevision != revision {
			_ = cache.Delete("player_snapshot:" + playerID.String())
		} else {
			cached.ActiveCombat = activeCombat
			syncCombatCache(playerID, activeCombat)
			_ = cache.SetJSON("player_snapshot:"+playerID.String(), cached, 5*time.Minute)
			return cached, nil
		}
	}

	var player models.Player
	var activeCombatBytes []byte
	err := db.DB.QueryRow(`
		SELECT p.id, p.username, p.rank, p.sub_rank, p.current_mana, p.max_mana, p.last_mana_update,
			p.time_speed_multiplier, p.active_combat, p.aperture_stress, p.hp, p.base_hp,
			p.primeval_stones, p.immortal_stones, p.wall_progress, p.human_qi,
			p.lifespan, p.x, p.y, p.current_region, p.path_masteries,
			p.state_revision + COALESCE((SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id), 0),
			COALESCE(p.version, 0),
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
		&player.X, &player.Y, &player.CurrentRegion, &player.PathMasteries, &player.StateRevision, &player.Version,
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

func loadAuthoritativePlayerMarker(q interface {
	QueryRow(query string, args ...any) *sql.Row
}, playerID uuid.UUID) (*models.CombatState, int64, error) {
	var raw []byte
	var revision int64
	if err := q.QueryRow(`SELECT p.active_combat, p.state_revision + COALESCE((SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id), 0) FROM players p WHERE p.id = $1`, playerID).Scan(&raw, &revision); err != nil {
		return nil, 0, err
	}
	combat, err := decodeActiveCombat(raw)
	return combat, revision, err
}

func loadPlayerStateRevision(q interface {
	QueryRow(query string, args ...any) *sql.Row
}, playerID uuid.UUID) (int64, error) {
	var revision int64
	err := q.QueryRow(`SELECT p.state_revision + COALESCE((SELECT SUM(pcr.state_revision) FROM player_collection_revisions pcr WHERE pcr.player_id = p.id), 0) FROM players p WHERE p.id = $1`, playerID).Scan(&revision)
	return revision, err
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
		StateRevision:       player.StateRevision,
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
		Type:           SessionEventKicked,
		Revision:       nextPlayerEventRevision(),
		ChangedDomains: []PlayerEventDomain{PlayerDomain},
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
