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
	ZoneKey    sim.ZoneKey
	EventID    uint64
	Tick       uint64
	SourceID   sim.EntityID
	TargetID   sim.EntityID
	TemplateID string
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

// ZoneMap carries the immutable map artifact used by one configured zone.
type ZoneMap struct {
	Collision     sim.Collision
	Spawns        []sim.Vec2
	Monsters      []MonsterSpawn
	PvP           bool
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

// MonsterSpawn is a server-constructed monster instance. Template values are
// resolved by the process bootstrap from the monster registry; clients never
// supply them.
type MonsterSpawn struct {
	TemplateID          string
	Name                string
	X                   float64
	Y                   float64
	MaxHP               float64
	AttackDamage        float64
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
}

const MaxMonstersPerZone = 500

// Manager is a process-local owner for configured zones and their player leases.
// Authentication and loading trusted player state remain gateway responsibilities.
type Manager struct {
	mu                sync.Mutex
	zones             map[sim.ZoneKey]*managedZone
	maps              map[sim.ZoneKey]ZoneMap
	keys              []sim.ZoneKey
	owners            map[sim.CharacterID]OwnershipLease
	maxPlayersPerZone int
	cancel            context.CancelFunc
	started           bool
	closed            bool
	deathHandler      func(context.Context, DeathEvent)
	deathQueue        chan DeathEvent
	deathDone         chan struct{}
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

	zones := make(map[sim.ZoneKey]*managedZone, len(ordered))
	for _, key := range ordered {
		if _, exists := zones[key]; exists {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateZone, key.String())
		}
		var collision sim.Collision
		var monsters []MonsterSpawn
		var pvp bool
		if zoneMap, ok := maps[key]; ok {
			collision = zoneMap.Collision
			monsters = zoneMap.Monsters
			pvp = zoneMap.PvP
		}
		if len(monsters) > MaxMonstersPerZone {
			return nil, fmt.Errorf("zone %s has %d monsters, maximum is %d", key.String(), len(monsters), MaxMonstersPerZone)
		}
		zone, err := sim.NewWorldForKeyWithCollision(key, queueSize, collision)
		if err != nil {
			return nil, fmt.Errorf("create zone %s: %w", key.String(), err)
		}
		zone.SetPvPEnabled(pvp)
		zone.SetDormantWhenEmpty(true)
		for index, spawn := range monsters {
			entity, spawnErr := monsterEntity(index, spawn)
			if spawnErr != nil {
				return nil, fmt.Errorf("spawn monster %d in zone %s: %w", index, key.String(), spawnErr)
			}
			if spawnErr = zone.AddEntity(entity); spawnErr != nil {
				return nil, fmt.Errorf("add monster %d in zone %s: %w", index, key.String(), spawnErr)
			}
		}
		zones[key] = &managedZone{
			world: zone, snapshots: make(chan sim.Snapshot, 1), subscribers: make(map[uint64]snapshotSubscriber),
		}
	}
	return &Manager{
		zones: zones, maps: maps, keys: ordered, owners: make(map[sim.CharacterID]OwnershipLease),
		maxPlayersPerZone: maxPlayersPerZone, deathQueue: make(chan DeathEvent, 256),
	}, nil
}

func monsterEntity(index int, spawn MonsterSpawn) (sim.Entity, error) {
	if spawn.TemplateID == "" || spawn.TemplateID != strings.TrimSpace(spawn.TemplateID) {
		return sim.Entity{}, ErrInvalidMonsterSpawn
	}
	if !finite(spawn.X) || !finite(spawn.Y) || spawn.MaxHP <= 0 {
		return sim.Entity{}, ErrInvalidMonsterSpawn
	}
	return sim.Entity{
		ID:        sim.EntityID(fmt.Sprintf("m:%d", index)),
		Kind:      sim.EntityMonster,
		Transform: sim.Transform{Position: sim.Vec2{X: spawn.X, Y: spawn.Y}},
		Vitals:    sim.Vitals{HP: spawn.MaxHP, MaxHP: spawn.MaxHP},
		Defense:   spawn.Defense,
		Speed:     spawn.Speed,
		Radius:    spawn.Radius,
		Faction:   "monster",
		AI: sim.AIState{
			Active: true, TemplateID: spawn.TemplateID, Name: spawn.Name, State: sim.AIIdle,
			Home:        sim.Vec2{X: spawn.X, Y: spawn.Y},
			AggroRadius: spawn.AggroRadius, LeashRadius: spawn.LeashRadius,
			AttackRange: spawn.AttackRange, PatrolRadius: spawn.PatrolRadius,
			AttackDamage: spawn.AttackDamage, AttackCooldownTicks: spawn.AttackCooldownTicks,
			RespawnTicks: spawn.RespawnTicks, ThinkTicks: 5,
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

// Start begins one fixed-step runner for each configured zone. It does not
// import legacy world state; JoinCharacter accepts only server-constructed entities.
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
	m.started = true
	m.deathDone = make(chan struct{})
	go m.deathWorker(ctx)
	for _, key := range m.keys {
		zone := m.zones[key]
		zone.done = sim.Run(ctx, zone.world, zone.snapshots)
		broadcastDone := make(chan struct{})
		zone.broadcastDone = broadcastDone
		go m.broadcastSnapshots(ctx, zone, broadcastDone)
	}
	return nil
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
	done := make([]<-chan struct{}, 0, len(m.keys)+1)
	for _, key := range m.keys {
		done = append(done, m.zones[key].done, m.zones[key].broadcastDone)
	}
	if m.deathDone != nil {
		done = append(done, m.deathDone)
	}
	// Close admission before releasing the lock so a concurrent join cannot
	// succeed after shutdown has begun but before the runner exits.
	m.closed = true
	m.mu.Unlock()

	cancel()
	for _, zoneDone := range done {
		select {
		case <-zoneDone:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	m.mu.Lock()
	m.started = false
	clear(m.owners)
	m.closeAllSubscribersLocked()
	m.mu.Unlock()
	return nil
}

func (m *Manager) ZoneKeys() []sim.ZoneKey {
	return append([]sim.ZoneKey(nil), m.keys...)
}

// ZoneStats is the minimal per-zone telemetry needed for the debug endpoints
// and load reports. It does not expose player identities.
type ZoneStats struct {
	Key           sim.ZoneKey
	Tick          uint64
	Entities      int
	Players       int
	Subscribers   int
	LastStep      time.Duration
	MapLoaded     bool    `json:"map_loaded"`
	MapHash       string  `json:"map_hash"`
	MapWidth      int     `json:"map_width"`
	MapHeight     int     `json:"map_height"`
	LegacyOriginX float64 `json:"legacy_origin_x"`
	LegacyOriginY float64 `json:"legacy_origin_y"`
	LegacyScale   float64 `json:"legacy_scale"`
	MonsterSpawns int     `json:"monster_spawns"`
}

func (m *Manager) Stats() []ZoneStats {
	m.mu.Lock()
	defer m.mu.Unlock()
	stats := make([]ZoneStats, 0, len(m.keys))
	for _, key := range m.keys {
		zone := m.zones[key]
		if zone == nil {
			continue
		}
		snapshot := zone.world.Snapshot()
		players := 0
		for _, entity := range snapshot.Entities {
			if entity.Kind == sim.EntityPlayer {
				players++
			}
		}
		zoneMap, mapLoaded := m.maps[key]
		stats = append(stats, ZoneStats{
			Key: key, Tick: snapshot.Tick, Entities: len(snapshot.Entities),
			Players: players, Subscribers: len(zone.subscribers),
			LastStep: zone.world.LastStepDuration(), MapLoaded: mapLoaded,
			MapHash: zoneMap.Hash, MapWidth: zoneMap.Width, MapHeight: zoneMap.Height,
			LegacyOriginX: zoneMap.LegacyOriginX, LegacyOriginY: zoneMap.LegacyOriginY,
			LegacyScale: zoneMap.LegacyScale, MonsterSpawns: len(zoneMap.Monsters),
		})
	}
	return stats
}

// ZoneKeyForMapWithCapacity selects the least-loaded instance of a map that
// still has player capacity. It enables in-process instance scaling without a
// client-visible zone list.
func (m *Manager) ZoneKeyForMapWithCapacity(mapID string) (sim.ZoneKey, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var best sim.ZoneKey
	bestLoad := m.maxPlayersPerZone + 1
	found := false
	for _, key := range m.keys {
		if key.MapID != mapID {
			continue
		}
		found = true
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
		return sim.ZoneKey{}, ErrZoneFull
	}
	return best, nil
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
	zone := m.zones[key]
	if zone == nil {
		return sim.Snapshot{}, ErrUnknownZone
	}
	return zone.world.Snapshot(), nil
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
	feed <- initial
	m.mu.Unlock()
	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			m.mu.Lock()
			delete(zone.subscribers, subscriberID)
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
	for {
		select {
		case <-ctx.Done():
			return
		case snapshot := <-zone.snapshots:
			m.enqueueDeaths(snapshot)
			m.mu.Lock()
			if len(zone.subscribers) == 0 {
				m.mu.Unlock()
				continue
			}
			index := spatial.NewIndex(AOICellSize)
			index.Rebuild(snapshot.Entities)
			for _, subscriber := range zone.subscribers {
				if snapshot.Tick < subscriber.minTick {
					continue
				}
				owner, exists := index.Entity(sim.EntityID(subscriber.characterID))
				if !exists {
					continue
				}
				visible := snapshot
				visible.Entities = index.Query(owner.Transform.Position, subscriber.viewRadius, owner.ID)
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

func (m *Manager) enqueueDeaths(snapshot sim.Snapshot) {
	for _, event := range snapshot.Events {
		if event.Type != "ENTITY_DIED" || event.TemplateID == "" || event.SourceID == "" {
			continue
		}
		death := DeathEvent{
			ZoneKey: snapshot.ZoneKey, EventID: event.ID, Tick: event.Tick,
			SourceID: sim.EntityID(event.SourceID), TargetID: sim.EntityID(event.TargetID), TemplateID: event.TemplateID,
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
	source := m.zones[active.ZoneKey]
	target := m.zones[targetKey]
	if source == nil || target == nil {
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
	active.ZoneKey = targetKey
	m.owners[lease.CharacterID] = active
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
	zone := m.zones[key]
	if zone == nil {
		return OwnershipLease{}, ErrUnknownZone
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
	result, queueErr := zone.world.QueueAddEntity(entity)
	if queueErr != nil {
		delete(m.owners, characterID)
		return OwnershipLease{}, queueErr
	}
	if err = waitForZoneMutation(zone, result); err != nil {
		delete(m.owners, characterID)
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
	if m.zones[key] == nil {
		return OwnershipLease{}, ErrUnknownZone
	}
	if _, exists := m.owners[characterID]; exists {
		return OwnershipLease{}, ErrCharacterOwned
	}
	if m.ownedPlayerCountLocked(key) >= m.maxPlayersPerZone {
		return OwnershipLease{}, ErrZoneFull
	}
	lease := OwnershipLease{CharacterID: characterID, ZoneKey: key, ControllerID: controllerID, Token: token}
	m.owners[characterID] = lease
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
	if active.HasEntity && m.started && !m.closed {
		zone := m.zones[active.ZoneKey]
		if zone == nil {
			return false
		}
		if err := removeEntityFromRunningZone(zone, sim.EntityID(lease.CharacterID)); err != nil {
			return false
		}
	} else if active.HasEntity {
		if zone := m.zones[active.ZoneKey]; zone != nil {
			zone.world.RemoveEntity(sim.EntityID(lease.CharacterID))
		}
	}
	m.removeLeaseSubscribersLocked(active)
	delete(m.owners, lease.CharacterID)
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

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
