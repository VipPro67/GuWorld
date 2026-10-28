// Package world owns configured realtime zones, lifecycle queues, and
// process-local player ownership leases. Authentication and snapshot loading
// remain gateway responsibilities.
package world

import (
	"context"
	cryptorand "crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	"github.com/vippro67/GuWorld/backend/internal/realtime/spatial"
	"github.com/vippro67/GuWorld/backend/internal/worker"
)

var (
	ErrNoZones                 = errors.New("realtime world manager requires at least one zone")
	ErrDuplicateZone           = errors.New("duplicate realtime zone key")
	ErrManagerStarted          = errors.New("realtime world manager already started or stopped")
	ErrUnknownZone             = errors.New("unknown realtime zone")
	ErrTooManyZones            = errors.New("too many realtime zones configured")
	ErrManagerNotRunning       = errors.New("realtime world manager is not running")
	ErrCharacterOwned          = errors.New("character already has an active realtime owner")
	ErrInvalidLease            = errors.New("invalid or stale realtime ownership lease")
	ErrLeaseAlreadyHasEntity   = errors.New("realtime lease already has a zone entity")
	ErrSnapshotSubscribersFull = errors.New("realtime zone snapshot subscriber capacity reached")
	ErrInvalidMonsterSpawn     = errors.New("invalid realtime monster spawn")
)

// DeathEvent is a durable-reward intent extracted from an authoritative death.
// It is delivered off the zone tick to the loot service.
type DeathEvent struct {
	ZoneKey      sim.ZoneKey
	Generation   string
	EventID      uint64
	Tick         uint64
	SourceID     sim.EntityID
	TargetID     sim.EntityID
	TemplateID   string
	PersistentID string
	IsNode       bool
}

// Region-local rollout loads one independent world per active KHU_VUC/SAFE_ZONE.
// Keep a finite process cap, but leave room for the current 125-map regional
// pack and incremental world growth.
const MaxZonesPerProcess = 256

const (
	AOICellSize   = 16.0
	AOIViewRadius = 32.0
)

type managedZone struct {
	world          *sim.World
	cancel         context.CancelFunc
	generation     string
	idleSince      time.Time
	evicting       bool
	done           <-chan struct{}
	snapshots      chan sim.Snapshot
	broadcastDone  <-chan struct{}
	subscribers    map[uint64]snapshotSubscriber
	nextSubscriber uint64
}

type snapshotSubscriber struct {
	feed        chan sim.Snapshot
	minTick     uint64
	characterID sim.CharacterID
	viewRadius  float64
	leaseToken  string
}

// OwnershipLease is a process-local controller lease. Its token is opaque and
// must be returned for replacement or release; callers should not accept a
// token from an unauthenticated client.
type OwnershipLease struct {
	CharacterID  sim.CharacterID
	ZoneKey      sim.ZoneKey
	ControllerID string
	HasEntity    bool   `json:"-"`
	Token        string `json:"-"`
}

// EdgeSide is one authored map border that can hand off to a neighbouring
// region. Child region maps are 2000x2000 and carry at most one portal per
// side, so every point on a side routes to the same target region.
type EdgeSide string

const (
	EdgeNorth EdgeSide = "north"
	EdgeEast  EdgeSide = "east"
	EdgeSouth EdgeSide = "south"
	EdgeWest  EdgeSide = "west"
)

// Portal is the map-authoritative edge handoff metadata. Coordinates are map
// units (0..width/height); legacy world coordinates are derived at transfer
// time with the destination ZoneMap transform.
type Portal struct {
	Side         EdgeSide
	TargetZoneID string
	TargetX      float64
	TargetY      float64
}

// EdgeCrossing resolves one authoritative border step into the destination
// zone key and map-local arrival point.
type EdgeCrossing struct {
	FromKey       sim.ZoneKey
	TargetKey     sim.ZoneKey
	ToZoneID      string
	LocalX        float64
	LocalY        float64
	LegacyX       float64
	LegacyY       float64
	LegacyOriginX float64
	LegacyOriginY float64
	LegacyScale   float64
	TriggerPos    sim.Vec2
}

const (
	// EdgeTriggerMargin is measured from the inner face of the border wall.
	// The player collision radius keeps their center >= 1.4 units from the wall,
	// so the band must be wider than that or a 20 Hz tick can step over it.
	EdgeTriggerMargin = 2.5
	// EdgeArrivalInset keeps the arrival point away from the opposite border so
	// a held movement key cannot immediately bounce the player back.
	EdgeArrivalInset = 4.0
)

// ZoneMap carries the immutable map artifact used by one configured zone.
type ZoneMap struct {
	Collision     sim.Collision
	Spawns        []sim.Vec2
	Monsters      []MonsterSpawn
	Portals       []Portal
	PvP           bool
	Safe          bool
	Hash          string
	Width         int
	Height        int
	LegacyOriginX float64
	LegacyOriginY float64
	LegacyScale   float64
	SafeSpawn     func(preferred sim.Vec2, radius float64) (sim.Vec2, bool)
}

// LegacyPosition converts persisted legacy world coordinates into map units.
func (m *Manager) LegacyPosition(key sim.ZoneKey, x, y float64) sim.Vec2 {
	m.mu.Lock()
	defer m.mu.Unlock()
	zoneMap := m.maps[key]
	scale := zoneMap.LegacyScale
	if scale <= 0 {
		scale = 1
	}
	return sim.Vec2{X: (x - zoneMap.LegacyOriginX) / scale, Y: (y - zoneMap.LegacyOriginY) / scale}
}

// ContainsPosition reports whether a checkpoint is already in the authored
// MMO coordinate space. Out-of-bounds checkpoints are treated as legacy data.
func (m *Manager) ContainsPosition(key sim.ZoneKey, point sim.Vec2) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	zoneMap, ok := m.maps[key]
	if !ok || zoneMap.Width <= 0 || zoneMap.Height <= 0 {
		return true
	}
	return finite(point.X) && finite(point.Y) && point.X >= 0 && point.Y >= 0 && point.X < float64(zoneMap.Width) && point.Y < float64(zoneMap.Height)
}

// LocalToLegacy converts a zone-local map position back into the legacy world
// coordinate space persisted in players.x/y. The transform is the inverse of
// LegacyPosition and must stay that way.
func (m *Manager) LocalToLegacy(key sim.ZoneKey, point sim.Vec2) (float64, float64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	zoneMap, ok := m.maps[key]
	if !ok {
		return 0, 0, false
	}
	scale := zoneMap.LegacyScale
	if scale <= 0 {
		scale = 1
	}
	return zoneMap.LegacyOriginX + point.X*scale, zoneMap.LegacyOriginY + point.Y*scale, true
}

// ResolveEdgeForLease turns one authoritative border position into a destination
// zone and arrival point. Only the current lease holder can use it.
func (m *Manager) ResolveEdgeForLease(lease OwnershipLease, position sim.Vec2) (EdgeCrossing, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.owners[lease.CharacterID]
	if !ok || current.Token != lease.Token || current.ZoneKey != lease.ZoneKey {
		return EdgeCrossing{}, false
	}
	source, ok := m.maps[lease.ZoneKey]
	if !ok || source.Width <= 0 || source.Height <= 0 {
		return EdgeCrossing{}, false
	}
	if !finite(position.X) || !finite(position.Y) {
		return EdgeCrossing{}, false
	}

	type candidate struct {
		side     EdgeSide
		distance float64
	}
	candidates := []candidate{
		{side: EdgeNorth, distance: position.Y},
		{side: EdgeEast, distance: float64(source.Width) - position.X},
		{side: EdgeSouth, distance: float64(source.Height) - position.Y},
		{side: EdgeWest, distance: position.X},
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].distance < candidates[j].distance })

	for _, option := range candidates {
		if option.distance > EdgeTriggerMargin {
			continue
		}
		portal, found := portalOnSide(source.Portals, option.side)
		if !found {
			continue
		}
		targetKey, target, found := m.mapForIDLocked(portal.TargetZoneID, lease.ZoneKey.InstanceID)
		if !found || target.Width <= 0 || target.Height <= 0 {
			continue
		}
		inset := EdgeArrivalInset
		local := sim.Vec2{X: position.X, Y: position.Y}
		switch option.side {
		case EdgeWest:
			local.X = float64(target.Width) - inset
			local.Y = clampFloat(position.Y, inset, float64(target.Height)-inset)
		case EdgeEast:
			local.X = inset
			local.Y = clampFloat(position.Y, inset, float64(target.Height)-inset)
		case EdgeNorth:
			local.Y = float64(target.Height) - inset
			local.X = clampFloat(position.X, inset, float64(target.Width)-inset)
		case EdgeSouth:
			local.Y = inset
			local.X = clampFloat(position.X, inset, float64(target.Width)-inset)
		}
		scale := target.LegacyScale
		if scale <= 0 {
			scale = 1
		}
		return EdgeCrossing{
			FromKey: lease.ZoneKey, TargetKey: targetKey, ToZoneID: portal.TargetZoneID,
			LocalX: local.X, LocalY: local.Y,
			LegacyX: target.LegacyOriginX + local.X*scale, LegacyY: target.LegacyOriginY + local.Y*scale,
			LegacyOriginX: target.LegacyOriginX, LegacyOriginY: target.LegacyOriginY, LegacyScale: scale,
			TriggerPos: position,
		}, true
	}
	return EdgeCrossing{}, false
}

// ZoneMapFor returns the configured artifact for a zone.
func (m *Manager) ZoneMapFor(key sim.ZoneKey) (ZoneMap, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	zoneMap, ok := m.maps[key]
	return zoneMap, ok
}

func (m *Manager) mapForIDLocked(mapID, preferredInstance string) (sim.ZoneKey, ZoneMap, bool) {
	preferred := sim.ZoneKey{MapID: mapID, InstanceID: preferredInstance, RulesetVersion: "v1"}
	if zoneMap, ok := m.maps[preferred]; ok {
		return preferred, zoneMap, true
	}
	for _, key := range m.keys {
		if key.MapID != mapID {
			continue
		}
		return key, m.maps[key], true
	}
	return sim.ZoneKey{}, ZoneMap{}, false
}

func portalOnSide(portals []Portal, side EdgeSide) (Portal, bool) {
	for _, portal := range portals {
		if portal.Side == side {
			return portal, true
		}
	}
	return Portal{}, false
}

func clampFloat(value, minimum, maximum float64) float64 {
	if maximum < minimum {
		return minimum
	}
	return math.Max(minimum, math.Min(maximum, value))
}

// MonsterSpawn is a server-constructed monster instance. Template values are
// resolved by the process bootstrap from the monster registry; clients never
// supply them.
type MonsterSpawn struct {
	TemplateID          string
	Name                string
	X                   float64
	Y                   float64
	MaxHP               float64
	CurrentHP           float64
	AttackDamage        float64
	NodeDamage          float64
	Defense             float64
	Speed               float64
	Radius              float64
	AggroRadius         float64
	LeashRadius         float64
	AttackRange         float64
	PatrolRadius        float64
	AttackCooldownTicks uint32
	RespawnTicks        uint32
	SplashRadius        float64
	SplashDamage        float64
	// Passive monsters (SAFE_ZONE content) never initiate combat. They keep
	// their retaliation table so a player can opt into a training fight.
	Passive bool
	// PersistentID links a replayed realtime entity to its durable row
	// (tribulation_enemies.id or aperture_resource_nodes.id).
	PersistentID string
	// NoRespawn keeps one-shot event entities dead for the lifetime of the
	// zone instead of scheduling the normal spawn respawn.
	NoRespawn bool
	// Node marks a stationary player-faction aperture resource node. Node
	// entities never move or attack; tribulation monsters target them.
	Node bool
}

// MaxMonstersPerZone bounds one zone's monster entities. Region maps seed
// density from Region.csv cell_type_weights; the largest authored map is close
// to 1100.
const MaxMonstersPerZone = sim.MaxMonsterSpawnCatalog

// Manager is a process-local owner for configured zones and their player leases.
// Authentication and loading trusted player state remain gateway responsibilities.
type Manager struct {
	mu                sync.Mutex
	cond              *sync.Cond
	zones             map[sim.ZoneKey]*managedZone
	maps              map[sim.ZoneKey]ZoneMap
	keys              []sim.ZoneKey
	runtimeKeys       map[sim.ZoneKey]struct{}
	owners            map[sim.CharacterID]OwnershipLease
	maxPlayersPerZone int
	cancel            context.CancelFunc
	started           bool
	closed            bool
	deathHandler      func(context.Context, DeathEvent)
	deathQueue        chan DeathEvent
	deathDone         chan struct{}
	runCtx            context.Context
	lazy              bool
	queueSize         int
	idleTimeout       time.Duration
	idleDone          <-chan struct{}
	evictWG           sync.WaitGroup
}

func NewManager(keys []sim.ZoneKey, queueSize int) (*Manager, error) {
	return NewManagerWithCapacity(keys, queueSize, 100)
}

func NewManagerWithCapacity(keys []sim.ZoneKey, queueSize, maxPlayersPerZone int) (*Manager, error) {
	return NewManagerWithMaps(keys, queueSize, maxPlayersPerZone, nil)
}

// NewManagerWithMaps creates zones with their loaded static collision/nav
// artifacts. A nil or missing map keeps the previous no-collision behavior for
// isolated tests and staged zones that have not authored a map yet.
func NewManagerWithMaps(keys []sim.ZoneKey, queueSize, maxPlayersPerZone int, maps map[sim.ZoneKey]ZoneMap) (*Manager, error) {
	return newManagerWithMaps(keys, queueSize, maxPlayersPerZone, maps, false)
}

// NewLazyManagerWithMaps validates all configured zones at startup, but only
// keeps a World and its runners while an instance is in use.
func NewLazyManagerWithMaps(keys []sim.ZoneKey, queueSize, maxPlayersPerZone int, maps map[sim.ZoneKey]ZoneMap) (*Manager, error) {
	return newManagerWithMaps(keys, queueSize, maxPlayersPerZone, maps, true)
}

func newManagerWithMaps(keys []sim.ZoneKey, queueSize, maxPlayersPerZone int, maps map[sim.ZoneKey]ZoneMap, lazy bool) (*Manager, error) {
	if len(keys) == 0 {
		return nil, ErrNoZones
	}
	if maxPlayersPerZone < 1 || maxPlayersPerZone > 100 {
		return nil, fmt.Errorf("realtime world manager capacity must be between 1 and 100")
	}
	if len(keys) > MaxZonesPerProcess {
		return nil, ErrTooManyZones
	}
	ordered := append([]sim.ZoneKey(nil), keys...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].String() < ordered[j].String() })

	manager := &Manager{
		zones: make(map[sim.ZoneKey]*managedZone), maps: maps, keys: ordered,
		runtimeKeys:       make(map[sim.ZoneKey]struct{}),
		owners:            make(map[sim.CharacterID]OwnershipLease),
		maxPlayersPerZone: maxPlayersPerZone, deathQueue: make(chan DeathEvent, 256),
		lazy: lazy, queueSize: queueSize, idleTimeout: 5 * time.Minute,
	}
	manager.cond = sync.NewCond(&manager.mu)

	seen := make(map[sim.ZoneKey]struct{}, len(ordered))
	validatedMaps := make(map[zoneValidationKey]struct{}, len(ordered))
	for _, key := range ordered {
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateZone, key.String())
		}
		seen[key] = struct{}{}
		if err := key.Validate(); err != nil {
			return nil, fmt.Errorf("validate zone %s: %w", key.String(), err)
		}
		if lazy {
			// Validate map-backed spawns now so a bad artifact fails rollout
			// startup, but do not allocate a World, monster entities, or runners.
			// Instances of one map share the artifact, so validate it once.
			zoneMap := maps[key]
			validationKey := zoneValidationKey{MapID: key.MapID, Hash: zoneMap.Hash, Monsters: len(zoneMap.Monsters)}
			if validationKey.Hash != "" {
				if _, validated := validatedMaps[validationKey]; validated {
					continue
				}
			}
			if err := validateZoneMap(key, zoneMap); err != nil {
				return nil, err
			}
			if validationKey.Hash != "" {
				validatedMaps[validationKey] = struct{}{}
			}
			continue
		}
		zone, err := buildManagedZone(key, queueSize, maps[key])
		if err != nil {
			return nil, err
		}
		manager.zones[key] = zone
	}
	return manager, nil
}

type zoneValidationKey struct {
	MapID    string
	Hash     string
	Monsters int
}

// validateZoneMap applies the setup validation that a lazy world would perform
// on its first activation without allocating simulation state.
func validateZoneMap(key sim.ZoneKey, zoneMap ZoneMap) error {
	entities, err := monsterEntities(key, zoneMap)
	if err != nil {
		return err
	}
	if err := sim.ValidateSetupEntities(zoneMap.Collision, entities); err != nil {
		return fmt.Errorf("validate zone %s monsters: %w", key.String(), err)
	}
	return nil
}

func monsterEntities(key sim.ZoneKey, zoneMap ZoneMap) ([]sim.Entity, error) {
	if len(zoneMap.Monsters) > MaxMonstersPerZone {
		return nil, fmt.Errorf("zone %s has %d monsters, maximum is %d", key.String(), len(zoneMap.Monsters), MaxMonstersPerZone)
	}
	entities := make([]sim.Entity, 0, len(zoneMap.Monsters))
	for index, spawn := range zoneMap.Monsters {
		entity, err := monsterEntity(index, spawn)
		if err != nil {
			return nil, fmt.Errorf("spawn monster %d in zone %s: %w", index, key.String(), err)
		}
		entities = append(entities, entity)
	}
	return entities, nil
}

func buildManagedZone(key sim.ZoneKey, queueSize int, zoneMap ZoneMap) (*managedZone, error) {
	entities, err := monsterEntities(key, zoneMap)
	if err != nil {
		return nil, err
	}
	world, err := sim.NewWorldForKeyWithCollision(key, queueSize, zoneMap.Collision)
	if err != nil {
		return nil, fmt.Errorf("create zone %s: %w", key.String(), err)
	}
	world.SetPvPEnabled(zoneMap.PvP)
	world.SetDormantWhenEmpty(true)
	if err := world.ConfigureMonsterSpawns(entities); err != nil {
		return nil, fmt.Errorf("configure monster spawns in zone %s: %w", key.String(), err)
	}
	generation, err := newZoneGeneration()
	if err != nil {
		return nil, err
	}
	return &managedZone{
		world: world, generation: generation, idleSince: time.Now(),
		snapshots: make(chan sim.Snapshot, 1), subscribers: make(map[uint64]snapshotSubscriber),
	}, nil
}

func monsterEntity(index int, spawn MonsterSpawn) (sim.Entity, error) {
	if spawn.TemplateID == "" || spawn.TemplateID != strings.TrimSpace(spawn.TemplateID) {
		return sim.Entity{}, ErrInvalidMonsterSpawn
	}
	if !finite(spawn.X) || !finite(spawn.Y) || spawn.MaxHP <= 0 {
		return sim.Entity{}, ErrInvalidMonsterSpawn
	}
	entityID := sim.EntityID(fmt.Sprintf("m:%d", index))
	faction := "monster"
	active := true
	if spawn.Node {
		entityID = sim.EntityID(fmt.Sprintf("n:%d", index))
		faction = "player"
		active = false
	}
	attackDamage := spawn.AttackDamage
	if spawn.Node {
		attackDamage = 0
	}
	currentHP := spawn.CurrentHP
	if currentHP <= 0 || currentHP > spawn.MaxHP {
		currentHP = spawn.MaxHP
	}
	return sim.Entity{
		ID:        entityID,
		Kind:      sim.EntityMonster,
		Transform: sim.Transform{Position: sim.Vec2{X: spawn.X, Y: spawn.Y}},
		Vitals:    sim.Vitals{HP: currentHP, MaxHP: spawn.MaxHP},
		Defense:   spawn.Defense,
		Speed:     spawn.Speed,
		Radius:    spawn.Radius,
		Faction:   faction,
		AI: sim.AIState{
			Active: active, Passive: spawn.Passive, TemplateID: spawn.TemplateID, PersistentID: spawn.PersistentID,
			Name: spawn.Name, State: sim.AIIdle,
			Home:        sim.Vec2{X: spawn.X, Y: spawn.Y},
			AggroRadius: spawn.AggroRadius, LeashRadius: spawn.LeashRadius,
			AttackRange: spawn.AttackRange, PatrolRadius: spawn.PatrolRadius,
			AttackDamage: attackDamage, NodeDamage: spawn.NodeDamage, AttackCooldownTicks: spawn.AttackCooldownTicks,
			RespawnTicks: spawn.RespawnTicks, NoRespawn: spawn.NoRespawn, ThinkTicks: 5,
			SplashRadius: spawn.SplashRadius, SplashDamage: spawn.SplashDamage,
		},
	}, nil
}

// MapHash returns the loaded collision/map hash for the join handshake.
func (m *Manager) MapHash(key sim.ZoneKey) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.maps[key].Hash
}

// SafeSpawn resolves a walkable position for an admitted character. Legacy
// coordinates outside the authored map are replaced by an authored spawn.
func (m *Manager) SafeSpawn(key sim.ZoneKey, preferred sim.Vec2, radius float64) (sim.Vec2, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	zoneMap, ok := m.maps[key]
	if !ok || zoneMap.Collision == nil {
		return preferred, true
	}
	if zoneMap.SafeSpawn != nil {
		return zoneMap.SafeSpawn(preferred, radius)
	}
	if !zoneMap.Collision.Blocked(preferred, radius) {
		return preferred, true
	}
	for _, spawn := range zoneMap.Spawns {
		if !zoneMap.Collision.Blocked(spawn, radius) {
			return spawn, true
		}
	}
	return preferred, false
}

// SetDeathHandler installs the durable reward sink. It must be called before
// Start; nil disables death processing.
func (m *Manager) SetDeathHandler(handler func(context.Context, DeathEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		m.deathHandler = handler
	}
}

// Start begins the death worker and, for eager managers, one fixed-step runner
// per configured zone. Lazy managers start a zone only when a lease needs it.
// It does not import legacy world state; JoinCharacter accepts only
// server-constructed entities.
func (m *Manager) Start(parent context.Context) error {
	if parent == nil {
		return errors.New("realtime world manager requires a context")
	}
	if err := parent.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started || m.closed {
		return ErrManagerStarted
	}
	ctx, cancel := context.WithCancel(parent)
	m.cancel = cancel
	m.runCtx = ctx
	m.started = true
	m.deathDone = make(chan struct{})
	go m.deathWorker(ctx)
	if m.lazy {
		m.idleDone = worker.RunEvery(ctx, time.Minute, "Realtime Zone Idle Cleanup", func(context.Context) {
			m.evictIdle(time.Now())
		})
	} else {
		for _, key := range m.keys {
			m.startZoneLocked(m.zones[key])
		}
	}
	return nil
}

// startZoneLocked starts the tick runner and broadcaster for a world that is
// already installed in the zone catalog. It is called only while m.mu is held.
func (m *Manager) startZoneLocked(zone *managedZone) {
	zoneCtx, cancel := context.WithCancel(m.runCtx)
	zone.cancel = cancel
	zone.done = sim.Run(zoneCtx, zone.world, zone.snapshots)
	broadcastDone := make(chan struct{})
	zone.broadcastDone = broadcastDone
	go m.broadcastSnapshots(zoneCtx, zone, broadcastDone)
}

// configuredKeyLocked reports whether key belongs to the immutable startup
// catalog. It is called with m.mu held.
func (m *Manager) configuredKeyLocked(key sim.ZoneKey) bool {
	for _, candidate := range m.keys {
		if candidate == key {
			return true
		}
	}
	return false
}

// ensureZoneLocked returns the live world for key, creating and starting it on
// first use. If the previous generation is still shutting down, it waits for
// that same transition before creating the next world. It may release m.mu
// while waiting but never holds it while a goroutine stops.
func (m *Manager) ensureZoneLocked(key sim.ZoneKey) (*managedZone, error) {
	for {
		if m.runCtx == nil || m.runCtx.Err() != nil || !m.started || m.closed {
			return nil, ErrManagerNotRunning
		}
		if zone := m.zones[key]; zone != nil {
			if zone.evicting {
				m.cond.Wait()
				continue
			}
			return zone, nil
		}
		if !m.configuredKeyLocked(key) {
			return nil, ErrUnknownZone
		}
		zone, err := buildManagedZone(key, m.queueSize, m.maps[key])
		if err != nil {
			return nil, err
		}
		m.zones[key] = zone
		m.startZoneLocked(zone)
		return zone, nil
	}
}

// markZoneActiveLocked cancels a pending idle countdown because the instance
// has a lease or subscriber again. It is called with m.mu held.
func (m *Manager) markZoneActiveLocked(zone *managedZone) {
	zone.idleSince = time.Time{}
}

// markZoneIdleLocked starts the idle countdown once the last lease and
// subscriber are gone. Projectiles and pending events are checked again when
// the countdown expires. It is called with m.mu held.
func (m *Manager) markZoneIdleLocked(key sim.ZoneKey, zone *managedZone) {
	if m.ownedPlayerCountLocked(key) > 0 || len(zone.subscribers) > 0 {
		return
	}
	if zone.idleSince.IsZero() {
		zone.idleSince = time.Now()
	}
}

// evictIdle tears down instances that have had no lease, subscriber,
// projectile, or pending snapshot event for at least idleTimeout.
func (m *Manager) evictIdle(now time.Time) {
	type victim struct {
		key  sim.ZoneKey
		zone *managedZone
	}
	m.mu.Lock()
	if !m.started || m.closed {
		m.mu.Unlock()
		return
	}
	var victims []victim
	for key, zone := range m.zones {
		if zone == nil || zone.evicting {
			continue
		}
		if m.ownedPlayerCountLocked(key) > 0 || len(zone.subscribers) > 0 {
			zone.idleSince = time.Time{}
			continue
		}
		if zone.idleSince.IsZero() {
			zone.idleSince = now
			continue
		}
		if now.Sub(zone.idleSince) < m.idleTimeout {
			continue
		}
		if zone.world.HasPendingActivity() || len(zone.snapshots) > 0 {
			continue
		}
		zone.evicting = true
		victims = append(victims, victim{key: key, zone: zone})
	}
	for range victims {
		m.evictWG.Add(1)
	}
	m.mu.Unlock()

	for _, current := range victims {
		go func(current victim) {
			defer m.evictWG.Done()
			m.stopZone(current.key, current.zone)
		}(current)
	}
}

// stopZone stops one idle world without holding the manager lock, then removes
// it from the catalog and wakes joins that were waiting for the transition.
func (m *Manager) stopZone(key sim.ZoneKey, zone *managedZone) {
	if zone.cancel != nil {
		zone.cancel()
	}
	if zone.done != nil {
		<-zone.done
	}
	if zone.broadcastDone != nil {
		<-zone.broadcastDone
	}
	// The runner and broadcaster are stopped, so any final published snapshot
	// is stable and its death events must be queued before the world is dropped.
	m.drainSnapshots(zone)
	m.mu.Lock()
	if m.zones[key] == zone {
		delete(m.zones, key)
	}
	zone.evicting = false
	m.cond.Broadcast()
	m.mu.Unlock()
}

// drainSnapshots moves deaths from any buffered snapshot into the death queue
// without blocking. It is called only after the zone runner has stopped.
func (m *Manager) drainSnapshots(zone *managedZone) {
	for {
		select {
		case snapshot := <-zone.snapshots:
			m.enqueueDeaths(zone, snapshot)
		default:
			return
		}
	}
}

// Stop cancels all zone loops and waits for each owner goroutine to exit.
func (m *Manager) Stop(ctx context.Context) error {
	if ctx == nil {
		return errors.New("realtime world manager requires a shutdown context")
	}
	m.mu.Lock()
	if !m.started {
		m.closed = true
		clear(m.owners)
		m.mu.Unlock()
		return nil
	}
	cancel := m.cancel
	done := make([]<-chan struct{}, 0, 2*len(m.zones)+2)
	for _, zone := range m.zones {
		if zone.done != nil {
			done = append(done, zone.done)
		}
		if zone.broadcastDone != nil {
			done = append(done, zone.broadcastDone)
		}
	}
	if m.deathDone != nil {
		done = append(done, m.deathDone)
	}
	if m.idleDone != nil {
		done = append(done, m.idleDone)
	}
	// Close admission before releasing the lock so a concurrent join cannot
	// succeed after shutdown has begun but before the runner exits.
	m.closed = true
	m.cond.Broadcast()
	m.mu.Unlock()

	cancel()
	for _, zoneDone := range done {
		select {
		case <-zoneDone:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	// Eviction goroutines only wait for the runners above; let them finish
	// before the manager forgets the zone catalog and wakes waiters.
	m.evictWG.Wait()

	m.mu.Lock()
	m.started = false
	clear(m.owners)
	m.closeAllSubscribersLocked()
	m.cond.Broadcast()
	m.mu.Unlock()
	return nil
}

func (m *Manager) ZoneKeys() []sim.ZoneKey {
	return append([]sim.ZoneKey(nil), m.keys...)
}

// ZoneStats is the minimal per-zone telemetry needed for the debug endpoints
// and load reports. It does not expose player identities.
type ZoneStats struct {
	Key            sim.ZoneKey
	Tick           uint64
	Entities       int
	ActiveMonsters int `json:"active_monsters"`
	Players        int
	Subscribers    int
	LastStep       time.Duration
	MapLoaded      bool    `json:"map_loaded"`
	Running        bool    `json:"running"`
	Evicting       bool    `json:"evicting,omitempty"`
	MapHash        string  `json:"map_hash"`
	MapWidth       int     `json:"map_width"`
	MapHeight      int     `json:"map_height"`
	LegacyOriginX  float64 `json:"legacy_origin_x"`
	LegacyOriginY  float64 `json:"legacy_origin_y"`
	LegacyScale    float64 `json:"legacy_scale"`
	MonsterSpawns  int     `json:"monster_spawns"`
}

func (m *Manager) Stats() []ZoneStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Every configured map stays visible even when its instance is idle. This
	// distinguishes "map artifact checked at startup" from "world running now".
	stats := make([]ZoneStats, 0, len(m.keys))
	for _, key := range m.keys {
		zoneMap, mapLoaded := m.maps[key]
		entry := ZoneStats{
			Key: key, MapLoaded: mapLoaded,
			MapHash: zoneMap.Hash, MapWidth: zoneMap.Width, MapHeight: zoneMap.Height,
			LegacyOriginX: zoneMap.LegacyOriginX, LegacyOriginY: zoneMap.LegacyOriginY,
			LegacyScale: zoneMap.LegacyScale, MonsterSpawns: len(zoneMap.Monsters),
		}
		if zone := m.zones[key]; zone != nil {
			snapshot := zone.world.Snapshot()
			players := 0
			monsters := 0
			for _, entity := range snapshot.Entities {
				if entity.Kind == sim.EntityPlayer {
					players++
				} else if entity.Kind == sim.EntityMonster {
					monsters++
				}
			}
			entry.Tick = snapshot.Tick
			entry.Entities = len(snapshot.Entities)
			entry.ActiveMonsters = monsters
			entry.Players = players
			entry.Subscribers = len(zone.subscribers)
			entry.LastStep = zone.world.LastStepDuration()
			entry.Running = m.started && !m.closed && !zone.evicting
			entry.Evicting = zone.evicting
		}
		stats = append(stats, entry)
	}
	return stats
}

// ZoneKeyForMapWithCapacity selects the least-loaded instance of a map that
// still has player capacity. It enables in-process instance scaling without a
// client-visible zone list.
func (m *Manager) ZoneKeyForMapWithCapacity(mapID string) (sim.ZoneKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var best, transitioning sim.ZoneKey
	bestLoad := m.maxPlayersPerZone + 1
	found := false
	for _, key := range m.keys {
		if key.MapID != mapID {
			continue
		}
		found = true
		// Prefer a live instance. A key that is already tearing down can only
		// be selected when no other instance of the map is available; the
		// claim then waits for that same transition instead of racing it.
		if zone := m.zones[key]; zone != nil && zone.evicting {
			if transitioning.MapID == "" {
				transitioning = key
			}
			continue
		}
		load := m.ownedPlayerCountLocked(key)
		if load < m.maxPlayersPerZone && load < bestLoad {
			best = key
			bestLoad = load
		}
	}
	if !found {
		return sim.ZoneKey{}, ErrUnknownZone
	}
	if best.MapID == "" {
		if transitioning.MapID != "" {
			return transitioning, nil
		}
		return sim.ZoneKey{}, ErrZoneFull
	}
	return best, nil
}

// RegisterZoneMap installs a runtime-created catalog entry before the first
// claim. It is used for per-event instances (tribulation) whose monster list is
// derived from durable state instead of the startup artifact. If the key is
// already live, the existing zone and its authoritative entity state win so a
// reconnect does not reset an in-progress encounter.
func (m *Manager) RegisterZoneMap(key sim.ZoneKey, zoneMap ZoneMap) error {
	if err := key.Validate(); err != nil {
		return err
	}
	if _, err := monsterEntities(key, zoneMap); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return ErrManagerNotRunning
	}
	if m.zones[key] != nil {
		// A live zone keeps its authoritative entity state. Still refresh the
		// catalog copy so the next rebuild after idle eviction starts from the
		// latest durable HP/dead set.
		m.maps[key] = zoneMap
		return nil
	}
	if _, exists := m.maps[key]; !exists {
		if len(m.keys) >= MaxZonesPerProcess {
			return ErrZoneFull
		}
		m.keys = append(m.keys, key)
		m.runtimeKeys[key] = struct{}{}
	}
	m.maps[key] = zoneMap
	return nil
}

// BaseZoneMap returns the immutable map artifact shared by every instance of
// mapID. It is used to build runtime event zones without reading the file again.
func (m *Manager) BaseZoneMap(mapID string) (ZoneMap, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for key, zoneMap := range m.maps {
		if key.MapID == mapID {
			return zoneMap, true
		}
	}
	return ZoneMap{}, false
}

func (m *Manager) ZoneKeyForMap(mapID string) (sim.ZoneKey, error) {
	var match sim.ZoneKey
	for _, key := range m.keys {
		if key.MapID != mapID {
			continue
		}
		if match.MapID != "" {
			return sim.ZoneKey{}, fmt.Errorf("multiple realtime instances configured for map %q", mapID)
		}
		match = key
	}
	if match.MapID == "" {
		return sim.ZoneKey{}, ErrUnknownZone
	}
	return match, nil
}

func (m *Manager) Snapshot(key sim.ZoneKey) (sim.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	zone := m.zones[key]
	if zone == nil {
		return sim.Snapshot{}, ErrUnknownZone
	}
	return zone.world.Snapshot(), nil
}

// ZoneTick returns the current tick of one configured zone. Admission uses it
// to translate durable combat deadlines against the zone that will own the
// character, before the entity is queued into that zone.
func (m *Manager) ZoneTick(key sim.ZoneKey) (uint64, bool) {
	m.mu.Lock()
	zone := m.zones[key]
	m.mu.Unlock()
	if zone == nil {
		return 0, false
	}
	return zone.world.Tick(), true
}

func (m *Manager) SnapshotForLease(lease OwnershipLease) (sim.Snapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return sim.Snapshot{}, ErrManagerNotRunning
	}
	active, exists := m.owners[lease.CharacterID]
	if !exists || !active.HasEntity || active.ZoneKey != lease.ZoneKey || active.ControllerID != lease.ControllerID || subtle.ConstantTimeCompare([]byte(active.Token), []byte(lease.Token)) != 1 {
		return sim.Snapshot{}, ErrInvalidLease
	}
	zone := m.zones[active.ZoneKey]
	if zone == nil {
		return sim.Snapshot{}, ErrUnknownZone
	}
	snapshot, ok := filterSnapshotForOwner(zone.world.Snapshot(), active.CharacterID)
	if !ok {
		return sim.Snapshot{}, ErrInvalidLease
	}
	return snapshot, nil
}

// SubscribeSnapshots registers one bounded latest-state feed for a current
// owner. Slow consumers receive the newest snapshot and skip superseded ones.
func (m *Manager) SubscribeSnapshots(lease OwnershipLease) (<-chan sim.Snapshot, func(), error) {
	m.mu.Lock()
	active, exists := m.owners[lease.CharacterID]
	if !m.started || m.closed {
		m.mu.Unlock()
		return nil, nil, ErrManagerNotRunning
	}
	if !exists || !active.HasEntity || active.ZoneKey != lease.ZoneKey || active.ControllerID != lease.ControllerID || subtle.ConstantTimeCompare([]byte(active.Token), []byte(lease.Token)) != 1 {
		m.mu.Unlock()
		return nil, nil, ErrInvalidLease
	}
	zone := m.zones[active.ZoneKey]
	if zone == nil {
		m.mu.Unlock()
		return nil, nil, ErrUnknownZone
	}
	zoneKey := active.ZoneKey
	if len(zone.subscribers) >= m.maxPlayersPerZone {
		m.mu.Unlock()
		return nil, nil, ErrSnapshotSubscribersFull
	}
	zone.nextSubscriber++
	subscriberID := zone.nextSubscriber
	feed := make(chan sim.Snapshot, 1)
	initial, ownerExists := filterSnapshotForOwner(zone.world.Snapshot(), lease.CharacterID)
	if !ownerExists {
		m.mu.Unlock()
		return nil, nil, ErrInvalidLease
	}
	zone.subscribers[subscriberID] = snapshotSubscriber{
		feed: feed, minTick: initial.Tick, characterID: lease.CharacterID, viewRadius: AOIViewRadius,
		leaseToken: lease.Token,
	}
	m.markZoneActiveLocked(zone)
	feed <- initial
	m.mu.Unlock()
	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			m.mu.Lock()
			delete(zone.subscribers, subscriberID)
			m.markZoneIdleLocked(zoneKey, zone)
			m.mu.Unlock()
		})
	}
	return feed, unsubscribe, nil
}

func filterSnapshotForOwner(snapshot sim.Snapshot, characterID sim.CharacterID) (sim.Snapshot, bool) {
	index := spatial.NewIndex(AOICellSize)
	index.Rebuild(snapshot.Entities)
	owner, exists := index.Entity(sim.EntityID(characterID))
	if !exists {
		return sim.Snapshot{}, false
	}
	snapshot.Entities = index.Query(owner.Transform.Position, AOIViewRadius, owner.ID)
	return snapshot, true
}

func (m *Manager) broadcastSnapshots(ctx context.Context, zone *managedZone, done chan<- struct{}) {
	defer close(done)
	spatialIndex := spatial.NewIndex(AOICellSize)
	for {
		select {
		case <-ctx.Done():
			// Drain snapshots published before cancellation so pending deaths
			// still reach the death worker. stopZone performs one final drain
			// after the runner exits.
			m.drainSnapshots(zone)
			return
		case snapshot := <-zone.snapshots:
			m.enqueueDeaths(zone, snapshot)
			m.mu.Lock()
			if len(zone.subscribers) == 0 {
				m.mu.Unlock()
				continue
			}
			spatialIndex.Rebuild(snapshot.Entities)
			for _, subscriber := range zone.subscribers {
				if snapshot.Tick < subscriber.minTick {
					continue
				}
				owner, exists := spatialIndex.Entity(sim.EntityID(subscriber.characterID))
				if !exists {
					continue
				}
				visible := snapshot
				visible.Entities = spatialIndex.Query(owner.Transform.Position, subscriber.viewRadius, owner.ID)
				select {
				case subscriber.feed <- visible:
				default:
					select {
					case <-subscriber.feed:
					default:
					}
					select {
					case subscriber.feed <- visible:
					default:
					}
				}
			}
			m.mu.Unlock()
		}
	}
}

func (m *Manager) enqueueDeaths(zone *managedZone, snapshot sim.Snapshot) {
	for _, event := range snapshot.Events {
		if event.Type != "ENTITY_DIED" || event.TemplateID == "" || event.SourceID == "" {
			continue
		}
		death := DeathEvent{
			ZoneKey: snapshot.ZoneKey, Generation: zone.generation,
			EventID: event.ID, Tick: event.Tick,
			SourceID: sim.EntityID(event.SourceID), TargetID: sim.EntityID(event.TargetID), TemplateID: event.TemplateID,
			PersistentID: event.PersistentID,
			IsNode:       len(event.TargetID) > 2 && event.TargetID[:2] == "n:",
		}
		select {
		case m.deathQueue <- death:
		default:
			// Keep the zone feed non-blocking; telemetry for dropped rewards is future work.
		}
	}
}

func (m *Manager) deathWorker(ctx context.Context) {
	defer close(m.deathDone)
	m.mu.Lock()
	handler := m.deathHandler
	m.mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			return
		case death := <-m.deathQueue:
			if handler == nil {
				continue
			}
			callCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			handler(callCtx, death)
			cancel()
		}
	}
}

// SubmitMoveTarget accepts a bounded click-to-move destination only while the
// caller still owns the active character lease. The world tick converts it into
// the same movement path as WASD input.
func (m *Manager) SubmitMoveTarget(lease OwnershipLease, command sim.InputCommand) error {
	if command.Target == nil {
		return sim.ErrInvalidInput
	}
	return m.submitForLease(lease, command)
}

// SubmitCast accepts a cast intent only while the caller owns the character.
// All damage/effect resolution stays on the zone tick.
func (m *Manager) SubmitCast(lease OwnershipLease, command sim.InputCommand) error {
	if command.Cast == nil {
		return sim.ErrInvalidInput
	}
	return m.submitForLease(lease, command)
}

// SubmitBasicAttack accepts a melee intent only while the caller owns the
// character. Range, cooldown and damage are resolved by the zone tick.
func (m *Manager) SubmitBasicAttack(lease OwnershipLease, command sim.InputCommand) error {
	if command.Attack == nil {
		return sim.ErrInvalidInput
	}
	return m.submitForLease(lease, command)
}

// TransferCharacter atomically moves an owned player entity between zones in
// this process. The controller/token stays valid but the old zone lease stops
// working immediately. Callers must restart the snapshot subscription and
// persist the new zone afterward (gateway/transfer layer).
func (m *Manager) TransferCharacter(lease OwnershipLease, targetKey sim.ZoneKey, spawn sim.Vec2) (OwnershipLease, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return OwnershipLease{}, ErrManagerNotRunning
	}
	active, exists := m.owners[lease.CharacterID]
	if !exists || !active.HasEntity || active.ZoneKey != lease.ZoneKey || active.ControllerID != lease.ControllerID || subtle.ConstantTimeCompare([]byte(active.Token), []byte(lease.Token)) != 1 {
		return OwnershipLease{}, ErrInvalidLease
	}
	if active.ZoneKey == targetKey {
		return active, nil
	}
	target, err := m.ensureZoneLocked(targetKey)
	if err != nil {
		return OwnershipLease{}, err
	}
	// ensureZoneLocked may have waited for a previous generation to stop; the
	// source lease could have been released or replaced in the meantime.
	active, exists = m.owners[lease.CharacterID]
	if !exists || !active.HasEntity || active.ZoneKey != lease.ZoneKey || active.ControllerID != lease.ControllerID || subtle.ConstantTimeCompare([]byte(active.Token), []byte(lease.Token)) != 1 {
		return OwnershipLease{}, ErrInvalidLease
	}
	if active.ZoneKey == targetKey {
		return active, nil
	}
	source := m.zones[active.ZoneKey]
	if source == nil {
		return OwnershipLease{}, ErrUnknownZone
	}
	if m.ownedPlayerCountLocked(targetKey) >= m.maxPlayersPerZone {
		return OwnershipLease{}, ErrZoneFull
	}
	original, ok := findSnapshotEntity(source.world.Snapshot(), sim.EntityID(active.CharacterID))
	if !ok {
		return OwnershipLease{}, ErrInvalidLease
	}
	moved := original
	moved.Transform.Position = spawn
	removeResult, err := source.world.QueueRemoveEntity(sim.EntityID(active.CharacterID))
	if err != nil {
		return OwnershipLease{}, err
	}
	if err := waitForZoneMutation(source, removeResult); err != nil {
		return OwnershipLease{}, err
	}
	addResult, err := target.world.QueueAddEntity(moved)
	if err != nil {
		restoreResult, _ := source.world.QueueAddEntity(original)
		_ = waitForZoneMutation(source, restoreResult)
		return OwnershipLease{}, err
	}
	if err := waitForZoneMutation(target, addResult); err != nil {
		restoreResult, _ := source.world.QueueAddEntity(original)
		_ = waitForZoneMutation(source, restoreResult)
		return OwnershipLease{}, err
	}
	sourceKey := active.ZoneKey
	active.ZoneKey = targetKey
	m.owners[lease.CharacterID] = active
	m.markZoneActiveLocked(target)
	m.markZoneIdleLocked(sourceKey, source)
	return active, nil
}

func findSnapshotEntity(snapshot sim.Snapshot, entityID sim.EntityID) (sim.Entity, bool) {
	for _, entity := range snapshot.Entities {
		if entity.ID == entityID {
			return entity, true
		}
	}
	return sim.Entity{}, false
}

func (m *Manager) SubmitInput(lease OwnershipLease, input sim.InputCommand) error {
	return m.submitForLease(lease, input)
}

func (m *Manager) submitForLease(lease OwnershipLease, command sim.InputCommand) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return ErrManagerNotRunning
	}
	active, exists := m.owners[lease.CharacterID]
	if !exists || !active.HasEntity || active.ZoneKey != lease.ZoneKey || active.ControllerID != lease.ControllerID || subtle.ConstantTimeCompare([]byte(active.Token), []byte(lease.Token)) != 1 {
		return ErrInvalidLease
	}
	zone := m.zones[active.ZoneKey]
	if zone == nil {
		return ErrUnknownZone
	}
	if command.EntityID != sim.EntityID(lease.CharacterID) {
		return ErrInvalidLease
	}
	command.LeaseToken = active.Token
	switch {
	case command.Cast != nil:
		command.Cast.EntityID = sim.EntityID(lease.CharacterID)
		command.Cast.LeaseToken = active.Token
		return zone.world.SubmitCast(*command.Cast)
	case command.Attack != nil:
		command.Attack.EntityID = sim.EntityID(lease.CharacterID)
		command.Attack.LeaseToken = active.Token
		return zone.world.SubmitBasicAttack(*command.Attack)
	case command.Target != nil:
		return zone.world.SubmitMoveTarget(command)
	default:
		return zone.world.SubmitInput(command)
	}
}

// JoinCharacter atomically adds a player entity and reserves the character's
// controller lease. The caller must construct entity state from an authenticated
// server-side character snapshot; client-provided stats or coordinates are not trusted.
func (m *Manager) JoinCharacter(characterID sim.CharacterID, controllerID string, key sim.ZoneKey, entity sim.Entity) (OwnershipLease, error) {
	if !validOwnerID(string(characterID)) || !validOwnerID(controllerID) || entity.Kind != sim.EntityPlayer || entity.CharacterID != characterID || entity.ID != sim.EntityID(characterID) {
		return OwnershipLease{}, ErrInvalidLease
	}
	token, err := newLeaseToken()
	if err != nil {
		return OwnershipLease{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return OwnershipLease{}, ErrManagerNotRunning
	}
	if _, exists := m.owners[characterID]; exists {
		return OwnershipLease{}, ErrCharacterOwned
	}
	zone, err := m.ensureZoneLocked(key)
	if err != nil {
		return OwnershipLease{}, err
	}
	if _, exists := m.owners[characterID]; exists {
		return OwnershipLease{}, ErrCharacterOwned
	}
	if m.ownedPlayerCountLocked(key) >= m.maxPlayersPerZone {
		return OwnershipLease{}, ErrZoneFull
	}
	lease := OwnershipLease{CharacterID: characterID, ZoneKey: key, ControllerID: controllerID, HasEntity: true, Token: token}
	entity.LeaseToken = token
	m.owners[characterID] = lease
	m.markZoneActiveLocked(zone)
	result, queueErr := zone.world.QueueAddEntity(entity)
	if queueErr != nil {
		delete(m.owners, characterID)
		m.markZoneIdleLocked(key, zone)
		return OwnershipLease{}, queueErr
	}
	if err = waitForZoneMutation(zone, result); err != nil {
		delete(m.owners, characterID)
		m.markZoneIdleLocked(key, zone)
		return OwnershipLease{}, err
	}
	return lease, nil
}

// AttachCharacterEntity completes a reserved ClaimCharacter lease after the
// gateway has loaded and validated authoritative character state.
func (m *Manager) AttachCharacterEntity(lease OwnershipLease, entity sim.Entity) (OwnershipLease, error) {
	if entity.Kind != sim.EntityPlayer || entity.CharacterID != lease.CharacterID || entity.ID != sim.EntityID(lease.CharacterID) {
		return OwnershipLease{}, ErrInvalidLease
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return OwnershipLease{}, ErrManagerNotRunning
	}
	active, exists := m.owners[lease.CharacterID]
	if !exists || active.ZoneKey != lease.ZoneKey || subtle.ConstantTimeCompare([]byte(active.Token), []byte(lease.Token)) != 1 {
		return OwnershipLease{}, ErrInvalidLease
	}
	if active.HasEntity {
		return OwnershipLease{}, ErrLeaseAlreadyHasEntity
	}
	zone := m.zones[active.ZoneKey]
	if zone == nil {
		return OwnershipLease{}, ErrUnknownZone
	}
	entity.LeaseToken = active.Token
	result, err := zone.world.QueueAddEntity(entity)
	if err != nil {
		return OwnershipLease{}, err
	}
	if err := waitForZoneMutation(zone, result); err != nil {
		return OwnershipLease{}, err
	}
	active.HasEntity = true
	m.owners[lease.CharacterID] = active
	m.markZoneActiveLocked(zone)
	return active, nil
}

// LeaveCharacter removes zone membership only for the current lease. Stale
// disconnects cannot remove an entity controlled by a replacement session.
func (m *Manager) LeaveCharacter(lease OwnershipLease) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.releaseCharacterLocked(lease)
}

var ErrZoneFull = errors.New("realtime zone is at player capacity")

// ClaimCharacter grants one authenticated controller the process-local right
// to own a character in a configured running zone. The gateway must bind the
// character ID from its verified session, never from a client payload.
func (m *Manager) ClaimCharacter(characterID sim.CharacterID, controllerID string, key sim.ZoneKey) (OwnershipLease, error) {
	if !validOwnerID(string(characterID)) || !validOwnerID(controllerID) {
		return OwnershipLease{}, ErrInvalidLease
	}
	token, err := newLeaseToken()
	if err != nil {
		return OwnershipLease{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return OwnershipLease{}, ErrManagerNotRunning
	}
	if _, exists := m.owners[characterID]; exists {
		return OwnershipLease{}, ErrCharacterOwned
	}
	zone, err := m.ensureZoneLocked(key)
	if err != nil {
		return OwnershipLease{}, err
	}
	if _, exists := m.owners[characterID]; exists {
		return OwnershipLease{}, ErrCharacterOwned
	}
	if m.ownedPlayerCountLocked(key) >= m.maxPlayersPerZone {
		return OwnershipLease{}, ErrZoneFull
	}
	lease := OwnershipLease{CharacterID: characterID, ZoneKey: key, ControllerID: controllerID, Token: token}
	m.owners[characterID] = lease
	m.markZoneActiveLocked(zone)
	return lease, nil
}

func (m *Manager) ownedPlayerCountLocked(key sim.ZoneKey) int {
	count := 0
	for _, owner := range m.owners {
		if owner.ZoneKey == key {
			count++
		}
	}
	return count
}

// ReplaceCharacterLease is the reconnect path. Only the holder of the current
// lease token can replace the controller; a stale tab cannot take ownership.
func (m *Manager) ReplaceCharacterLease(current OwnershipLease, newControllerID string) (OwnershipLease, error) {
	if !validOwnerID(newControllerID) {
		return OwnershipLease{}, ErrInvalidLease
	}
	token, err := newLeaseToken()
	if err != nil {
		return OwnershipLease{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started || m.closed {
		return OwnershipLease{}, ErrManagerNotRunning
	}
	active, exists := m.owners[current.CharacterID]
	if !exists || active.ZoneKey != current.ZoneKey || subtle.ConstantTimeCompare([]byte(active.Token), []byte(current.Token)) != 1 {
		return OwnershipLease{}, ErrInvalidLease
	}
	if active.HasEntity {
		zone := m.zones[active.ZoneKey]
		if zone == nil {
			return OwnershipLease{}, ErrUnknownZone
		}
		var result <-chan error
		for {
			var queueErr error
			result, queueErr = zone.world.QueueSetLeaseToken(sim.EntityID(active.CharacterID), token)
			if queueErr == nil {
				break
			}
			if !errors.Is(queueErr, sim.ErrLifecycleQueueFull) {
				return OwnershipLease{}, queueErr
			}
			select {
			case <-zone.done:
				return OwnershipLease{}, ErrManagerNotRunning
			case <-time.After(5 * time.Millisecond):
			}
		}
		if err := waitForZoneMutation(zone, result); err != nil {
			return OwnershipLease{}, err
		}
	}
	m.removeLeaseSubscribersLocked(active)
	replacement := OwnershipLease{CharacterID: active.CharacterID, ZoneKey: active.ZoneKey, ControllerID: newControllerID, HasEntity: active.HasEntity, Token: token}
	m.owners[current.CharacterID] = replacement
	return replacement, nil
}

// ReleaseCharacter relinquishes ownership only when the supplied lease is
// still current. A stale disconnect cannot release a replacement controller.
func (m *Manager) ReleaseCharacter(lease OwnershipLease) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.releaseCharacterLocked(lease)
}

func (m *Manager) releaseCharacterLocked(lease OwnershipLease) bool {
	active, exists := m.owners[lease.CharacterID]
	if !exists || active.ZoneKey != lease.ZoneKey || subtle.ConstantTimeCompare([]byte(active.Token), []byte(lease.Token)) != 1 {
		return false
	}
	zone := m.zones[active.ZoneKey]
	if active.HasEntity && m.started && !m.closed {
		if zone == nil {
			return false
		}
		if err := removeEntityFromRunningZone(zone, sim.EntityID(lease.CharacterID)); err != nil {
			return false
		}
	} else if active.HasEntity {
		if zone != nil {
			zone.world.RemoveEntity(sim.EntityID(lease.CharacterID))
		}
	}
	m.removeLeaseSubscribersLocked(active)
	delete(m.owners, lease.CharacterID)
	if zone != nil {
		m.markZoneIdleLocked(active.ZoneKey, zone)
	}
	return true
}

func (m *Manager) removeLeaseSubscribersLocked(lease OwnershipLease) {
	zone := m.zones[lease.ZoneKey]
	if zone == nil {
		return
	}
	for id, subscriber := range zone.subscribers {
		if subscriber.characterID == lease.CharacterID && subscriber.leaseToken == lease.Token {
			close(subscriber.feed)
			delete(zone.subscribers, id)
		}
	}
}

func (m *Manager) closeAllSubscribersLocked() {
	for _, zone := range m.zones {
		for id, subscriber := range zone.subscribers {
			close(subscriber.feed)
			delete(zone.subscribers, id)
		}
	}
}

func removeEntityFromRunningZone(zone *managedZone, entityID sim.EntityID) error {
	for {
		result, err := zone.world.QueueRemoveEntity(entityID)
		if err == nil {
			if mutationErr := waitForZoneMutation(zone, result); mutationErr == nil {
				return nil
			} else if !errors.Is(mutationErr, ErrManagerNotRunning) {
				return mutationErr
			}
		}
		if !errors.Is(err, sim.ErrLifecycleQueueFull) && err != nil {
			return err
		}
		select {
		case <-zone.done:
			zone.world.RemoveEntity(entityID)
			return nil
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func waitForZoneMutation(zone *managedZone, result <-chan error) error {
	select {
	case err := <-result:
		return err
	case <-zone.done:
		select {
		case err := <-result:
			return err
		default:
			return ErrManagerNotRunning
		}
	}
}

func (m *Manager) OwnsCharacter(characterID sim.CharacterID) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.owners[characterID]
	return exists
}

// PlayerVitals reads the live zone HP for a character currently owned by this
// manager. ok is false when the character has no attached entity.
func (m *Manager) PlayerVitals(characterID sim.CharacterID) (hp, maxHP float64, ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	active, exists := m.owners[characterID]
	if !exists || !active.HasEntity {
		return 0, 0, false
	}
	zone := m.zones[active.ZoneKey]
	if zone == nil {
		return 0, 0, false
	}
	return zone.world.PlayerVitals(sim.EntityID(characterID))
}

// ApplyHPDelta applies a trusted HP change from a legacy HTTP action to the
// owned zone entity. It waits for the next zone tick to apply the delta so the
// caller's response and the HUD agree.
func (m *Manager) ApplyHPDelta(characterID sim.CharacterID, delta float64) error {
	if !finite(delta) || delta == 0 {
		return nil
	}
	m.mu.Lock()
	active, exists := m.owners[characterID]
	if !exists || !active.HasEntity {
		m.mu.Unlock()
		return ErrInvalidLease
	}
	zone := m.zones[active.ZoneKey]
	if zone == nil {
		m.mu.Unlock()
		return ErrUnknownZone
	}
	m.mu.Unlock()
	result, err := zone.world.QueueHPDelta(sim.EntityID(characterID), delta)
	if err != nil {
		return err
	}
	return waitForZoneMutation(zone, result)
}

func validOwnerID(value string) bool {
	return value != "" && len(value) <= sim.MaxEntityIDLength && strings.TrimSpace(value) == value
}

func newLeaseToken() (string, error) {
	var token [32]byte
	if _, err := cryptorand.Read(token[:]); err != nil {
		return "", fmt.Errorf("generate realtime ownership token: %w", err)
	}
	return hex.EncodeToString(token[:]), nil
}

// newZoneGeneration identifies one World lifetime. Death/loot identifiers
// include it so event IDs that restart at 1 after reactivation cannot collide
// with rewards from a previous generation or another process.
func newZoneGeneration() (string, error) {
	var generation [16]byte
	if _, err := cryptorand.Read(generation[:]); err != nil {
		return "", fmt.Errorf("generate realtime zone generation: %w", err)
	}
	return hex.EncodeToString(generation[:]), nil
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
