// Package sim contains the deterministic, in-memory core of a realtime zone.
// State-changing operations are serialized by World. Running zones process
// trusted lifecycle requests from a bounded queue on the same tick owner.
package sim

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

const (
	TickRate                    = 20
	TickDuration                = time.Second / TickRate
	DefaultQueueSize            = 1024
	MaxCommandsPerTick          = 1024
	MaxLifecycleCommandsPerTick = 64
	MaxEntityIDLength           = 80
	MaxWorldCoordinate          = 1_000_000
	MaxMovementSubstep          = 0.25
	MaxMovementSteps            = 64
	MaxEntitySpeed              = MaxMovementSubstep * MaxMovementSteps * TickRate
	MaxEntityRadius             = 16
	InputLeaseTicks             = 5 // Stop after 250ms without a fresh movement intent.
	MaxMoveTargetDistance       = 64.0
	MaxMoveTargetTicks          = TickRate * 30 // 30 seconds per click-to-move request.
	MoveTargetArriveEpsilon     = 0.15
)

var (
	ErrInvalidEntity       = errors.New("invalid realtime entity")
	ErrInvalidInput        = errors.New("invalid realtime input")
	ErrQueueFull           = errors.New("realtime input queue is full")
	ErrInvalidCollisionMap = errors.New("invalid realtime collision map")
	ErrInvalidZoneKey      = errors.New("invalid realtime zone key")
	ErrLifecycleQueueFull  = errors.New("realtime lifecycle queue is full")
)

type EntityKind string

type EntityID string
type CharacterID string

// ZoneKey identifies one simulation instance and its active ruleset. MapID
// selects content, InstanceID separates concurrent copies, and RulesetVersion
// prevents incompatible runtime configurations from sharing ownership.
type ZoneKey struct {
	MapID          string `json:"map_id"`
	InstanceID     string `json:"instance_id"`
	RulesetVersion string `json:"ruleset_version"`
}

func (key ZoneKey) Validate() error {
	for _, value := range []string{key.MapID, key.InstanceID, key.RulesetVersion} {
		if value == "" || len(value) > MaxEntityIDLength || strings.TrimSpace(value) != value {
			return fmt.Errorf("%w: fields must be non-empty, trimmed, and bounded", ErrInvalidZoneKey)
		}
	}
	return nil
}

func (key ZoneKey) String() string {
	return fmt.Sprintf("%q/%q@%q", key.MapID, key.InstanceID, key.RulesetVersion)
}

const (
	EntityPlayer  EntityKind = "player"
	EntityMonster EntityKind = "monster"
)

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Transform struct {
	Position Vec2 `json:"position"`
}

type Vitals struct {
	HP     float64 `json:"hp"`
	MaxHP  float64 `json:"max_hp"`
	Shield float64 `json:"shield,omitempty"`
}

// Entity is authoritative zone state. Fields are exported for construction and
// snapshots; callers must not mutate an entity after passing it to AddEntity.
type Entity struct {
	ID             EntityID                 `json:"id"`
	CharacterID    CharacterID              `json:"character_id,omitempty"`
	Name           string                   `json:"name,omitempty"`
	Kind           EntityKind               `json:"kind"`
	Transform      Transform                `json:"transform"`
	Vitals         Vitals                   `json:"vitals"`
	Defense        float64                  `json:"-"`
	DaoProfile     *gamefx.DaoCombatProfile `json:"-"`
	PathResistance map[string]float64       `json:"-"`
	Speed          float64                  `json:"speed"`
	Radius         float64                  `json:"radius"`
	Faction        string                   `json:"faction"`
	PartyID        string                   `json:"party_id,omitempty"`
	Revision       uint64                   `json:"revision"`
	AckSeq         uint64                   `json:"ack_seq,omitempty"`
	LeaseToken     string                   `json:"-"`
	AI             AIState                  `json:"-"`
	Combat         CombatState              `json:"-"`

	moveIntent    Vec2
	moveTarget    *Vec2
	targetTick    uint64
	lastInputTick uint64
}

type InputCommand struct {
	EntityID   EntityID
	LeaseToken string
	Seq        uint64
	Move       Vec2
	// Target is set for a click-to-move intent. It shares the movement sequence
	// stream with WASD input; a non-zero Move cancels an active target.
	Target *Vec2
	Cast   *CastCommand
	Attack *AttackCommand
}

type lifecycleCommand struct {
	apply  func() error
	result chan error
}

type Snapshot struct {
	ZoneKey     ZoneKey           `json:"zone_key"`
	Tick        uint64            `json:"tick"`
	Entities    []Entity          `json:"entities"`
	Projectiles []ProjectileState `json:"projectiles,omitempty"`
	Events      []combat.Event    `json:"events,omitempty"`
}

type World struct {
	stateMu   sync.Mutex
	zoneKey   ZoneKey
	tick      uint64
	step      float64
	collision Collision
	allowPvP  bool

	dormantWhenEmpty bool
	activePlayers    int

	// The tick owner mutates entities; setup-only direct methods also take
	// stateMu, while running lifecycle changes enter through lifecycleInbox.
	entities       map[EntityID]*Entity
	inbox          chan InputCommand
	lifecycleInbox chan lifecycleCommand
	rng            *rand.Rand
	published      atomic.Pointer[Snapshot]

	projectiles      []*projectile
	nextProjectileID uint64
	events           []combat.Event
	nextEventID      uint64
	lastStepNanos    atomic.Int64
}

func NewWorld(zoneID string, queueSize int) (*World, error) {
	return NewWorldForKey(ZoneKey{MapID: zoneID, InstanceID: "default", RulesetVersion: "v1"}, queueSize)
}

// NewWorldWithCollision creates a zone with an immutable static collision
// map. The map must be fully built before the world starts stepping.
func NewWorldWithCollision(zoneID string, queueSize int, collision Collision) (*World, error) {
	return NewWorldForKeyWithCollision(ZoneKey{MapID: zoneID, InstanceID: "default", RulesetVersion: "v1"}, queueSize, collision)
}

// NewWorldForKey creates a zone for a fully qualified map/instance/ruleset key.
func NewWorldForKey(key ZoneKey, queueSize int) (*World, error) {
	return NewWorldForKeyWithCollision(key, queueSize, nil)
}

// NewWorldForKeyWithCollision creates a keyed zone with an immutable static
// collision map and a fresh server-generated seed. The map must be fully built
// before the world starts stepping.
func NewWorldForKeyWithCollision(key ZoneKey, queueSize int, collision Collision) (*World, error) {
	if err := key.Validate(); err != nil {
		return nil, err
	}
	seed, err := newZoneSeed()
	if err != nil {
		return nil, err
	}
	return NewWorldForKeyWithCollisionAndSeed(key, queueSize, collision, seed)
}

// NewWorldForKeyWithSeed creates a keyed zone with an explicit deterministic
// random seed, useful for repeatable replays and simulation tests.
func NewWorldForKeyWithSeed(key ZoneKey, queueSize int, seed int64) (*World, error) {
	return NewWorldForKeyWithCollisionAndSeed(key, queueSize, nil, seed)
}

// NewWorldForKeyWithCollisionAndSeed is the fully configured deterministic
// zone constructor.
func NewWorldForKeyWithCollisionAndSeed(key ZoneKey, queueSize int, collision Collision, seed int64) (*World, error) {
	if err := key.Validate(); err != nil {
		return nil, err
	}
	if queueSize <= 0 {
		queueSize = DefaultQueueSize
	}
	w := &World{
		zoneKey:        key,
		step:           TickDuration.Seconds(),
		collision:      collision,
		entities:       make(map[EntityID]*Entity),
		inbox:          make(chan InputCommand, queueSize),
		lifecycleInbox: make(chan lifecycleCommand, queueSize),
		rng:            rand.New(rand.NewSource(seed)),
	}
	w.publish()
	return w, nil
}

// NextRandomInt63 returns the next zone-seeded simulation value. Do not use it
// for secrets or security-sensitive reward draws; use crypto/rand for those.
func (w *World) canApplySkill(source, target *Entity, skill combat.Skill) bool {
	if source == nil || target == nil || source.ID == target.ID || target.Vitals.HP <= 0 {
		return false
	}
	if skill.Damage > 0 || (skill.StatusID != "" && skill.StatusDebuff) {
		return w.canHarm(source, target)
	}
	if target.Faction != source.Faction {
		return false
	}
	if source.Kind == EntityPlayer && target.Kind == EntityPlayer {
		return source.ID == target.ID || (source.PartyID != "" && source.PartyID == target.PartyID)
	}
	return false
}

// SetPvPEnabled configures whether players in different parties may damage each
// other. It must be called before the zone runner starts.
func (w *World) SetPvPEnabled(enabled bool) {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	w.allowPvP = enabled
}

func (w *World) PvPEnabled() bool {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.allowPvP
}

func (w *World) SetDormantWhenEmpty(dormant bool) {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	w.dormantWhenEmpty = dormant
}

func (w *World) DormantWhenEmpty() bool {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.dormantWhenEmpty
}

func (w *World) canHarm(source, target *Entity) bool {
	if source == nil || target == nil || source.ID == target.ID || target.Vitals.HP <= 0 {
		return false
	}
	if target.Faction != source.Faction {
		return true
	}
	if source.Kind == EntityPlayer && target.Kind == EntityPlayer {
		if !w.allowPvP {
			return false
		}
		if source.PartyID != "" && source.PartyID == target.PartyID {
			return false
		}
		return true
	}
	return false
}

// LastStepDuration is the most recent Step wall-clock cost. It is telemetry
// only; the simulation still advances by a fixed 50ms regardless of this value.
func (w *World) LastStepDuration() time.Duration {
	return time.Duration(w.lastStepNanos.Load())
}

func (w *World) NextRandomInt63() int64 {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.rng.Int63()
}

func newZoneSeed() (int64, error) {
	var bytes [8]byte
	if _, err := cryptorand.Read(bytes[:]); err != nil {
		return 0, fmt.Errorf("generate realtime zone seed: %w", err)
	}
	return int64(binary.LittleEndian.Uint64(bytes[:])), nil
}

// AddEntity is intended for trusted setup before a zone runner starts. Running
// zones should use QueueAddEntity so membership changes stay on the tick owner.
func (w *World) AddEntity(entity Entity) error {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	return w.addEntityLocked(entity)
}

func (w *World) addEntityLocked(entity Entity) error {
	if err := validateEntity(entity); err != nil {
		return err
	}
	if w.blocked(entity.Transform.Position, entity.Radius) {
		return fmt.Errorf("%w: spawn intersects static collision", ErrInvalidEntity)
	}
	if _, exists := w.entities[entity.ID]; exists {
		return fmt.Errorf("%w: duplicate id %q", ErrInvalidEntity, entity.ID)
	}
	copy := entity
	copy.moveIntent = Vec2{}
	if copy.Kind == EntityPlayer {
		w.activePlayers++
	}
	w.entities[entity.ID] = &copy
	w.publish()
	return nil
}

// QueueAddEntity schedules a trusted lifecycle mutation for the zone tick
// owner. The result channel is buffered so a caller can stop waiting safely.
func (w *World) QueueAddEntity(entity Entity) (<-chan error, error) {
	request := lifecycleCommand{apply: func() error { return w.addEntityLocked(entity) }, result: make(chan error, 1)}
	select {
	case w.lifecycleInbox <- request:
		return request.result, nil
	default:
		return nil, ErrLifecycleQueueFull
	}
}

// RemoveEntity is intended for trusted teardown before a zone runner starts.
// Running zones should use QueueRemoveEntity.
func (w *World) RemoveEntity(entityID EntityID) {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	w.removeEntityLocked(entityID)
}

func (w *World) removeEntityLocked(entityID EntityID) {
	if existing, exists := w.entities[entityID]; exists {
		if existing.Kind == EntityPlayer {
			w.activePlayers--
		}
		delete(w.entities, entityID)
	}
	w.publish()
}

func (w *World) QueueRemoveEntity(entityID EntityID) (<-chan error, error) {
	request := lifecycleCommand{apply: func() error {
		w.removeEntityLocked(entityID)
		return nil
	}, result: make(chan error, 1)}
	select {
	case w.lifecycleInbox <- request:
		return request.result, nil
	default:
		return nil, ErrLifecycleQueueFull
	}
}

func (w *World) QueueSetLeaseToken(entityID EntityID, leaseToken string) (<-chan error, error) {
	request := lifecycleCommand{apply: func() error {
		entity := w.entities[entityID]
		if entity == nil {
			return ErrInvalidEntity
		}
		entity.LeaseToken = leaseToken
		return nil
	}, result: make(chan error, 1)}
	select {
	case w.lifecycleInbox <- request:
		return request.result, nil
	default:
		return nil, ErrLifecycleQueueFull
	}
}

// SubmitInput is safe for concurrent network callers. Commands are bounded and
// never block a zone tick when a client floods the input queue.
func (w *World) SubmitInput(command InputCommand) error {
	if strings.TrimSpace(string(command.EntityID)) != string(command.EntityID) || command.EntityID == "" || command.Seq == 0 || !finite(command.Move.X) || !finite(command.Move.Y) {
		return ErrInvalidInput
	}
	length := math.Hypot(command.Move.X, command.Move.Y)
	if length > 1.01 {
		return ErrInvalidInput
	}
	if length > 1 {
		command.Move.X /= length
		command.Move.Y /= length
	}
	select {
	case w.inbox <- command:
		return nil
	default:
		return ErrQueueFull
	}
}

// SubmitMoveTarget accepts a bounded click-to-move destination. The destination
// is clamped to MaxMoveTargetDistance from the actor on the next tick; the
// server never trusts a client path or teleports to the raw click position.
func (w *World) SubmitMoveTarget(command InputCommand) error {
	if command.Target == nil || strings.TrimSpace(string(command.EntityID)) != string(command.EntityID) || command.EntityID == "" || command.Seq == 0 || !validCoordinate(command.Target.X) || !validCoordinate(command.Target.Y) {
		return ErrInvalidInput
	}
	target := *command.Target
	command.Target = &target
	command.Move = Vec2{}
	select {
	case w.inbox <- command:
		return nil
	default:
		return ErrQueueFull
	}
}

// Step advances exactly one 50ms simulation step, regardless of wall-clock
// timing. It performs no I/O and returns a detached, stable-ordered snapshot.
func (w *World) Step() Snapshot {
	started := time.Now()
	defer func() { w.lastStepNanos.Store(time.Since(started).Nanoseconds()) }()
	w.stateMu.Lock()
	defer w.stateMu.Unlock()
	w.processLifecycleCommands()
	w.tick++

	if w.dormantWhenEmpty && w.activePlayers == 0 && len(w.projectiles) == 0 && len(w.events) == 0 {
		return w.publishDormant()
	}

	commands := w.collectCommands()
	sort.Slice(commands, func(i, j int) bool {
		if commands[i].EntityID == commands[j].EntityID {
			if commands[i].Seq != commands[j].Seq {
				return commands[i].Seq < commands[j].Seq
			}
			if commands[i].Move.X != commands[j].Move.X {
				return commands[i].Move.X < commands[j].Move.X
			}
			return commands[i].Move.Y < commands[j].Move.Y
		}
		return commands[i].EntityID < commands[j].EntityID
	})
	for _, command := range commands {
		entity := w.entities[command.EntityID]
		if entity == nil || command.LeaseToken != entity.LeaseToken || command.Seq <= entity.AckSeq || entity.Vitals.HP <= 0 {
			continue
		}
		entity.AckSeq = command.Seq
		if command.Cast != nil {
			w.startCastLocked(entity, *command.Cast)
			continue
		}
		if command.Attack != nil {
			w.startBasicAttackLocked(entity, *command.Attack)
			continue
		}
		if command.Target != nil {
			target := clampMoveTarget(entity.Transform.Position, *command.Target)
			if w.blocked(target, entity.Radius) {
				entity.moveTarget = nil
				entity.moveIntent = Vec2{}
				w.emitEvent(combat.Event{
					Type: combat.EventMoveRejected, SourceID: string(entity.ID),
					X: target.X, Y: target.Y, Reason: "blocked_destination",
				})
				continue
			}
			entity.moveTarget = &target
			entity.targetTick = w.tick
			entity.moveIntent = Vec2{}
			continue
		}
		entity.lastInputTick = w.tick
		entity.moveIntent = command.Move
		if command.Move.X != 0 || command.Move.Y != 0 {
			entity.moveTarget = nil
		}
	}

	w.stepMonsterAI()
	w.tickCombat()

	for _, entity := range w.entities {
		if entity.Kind == EntityPlayer && w.tick-entity.lastInputTick >= InputLeaseTicks {
			entity.moveIntent = Vec2{}
		}
		if entity.Combat.Pending != nil {
			entity.moveIntent = Vec2{}
			entity.moveTarget = nil
			continue
		}
		if entity.Vitals.HP <= 0 {
			entity.moveIntent = Vec2{}
			entity.moveTarget = nil
			continue
		}
		if entity.moveTarget != nil {
			if w.tick-entity.targetTick > MaxMoveTargetTicks {
				entity.moveTarget = nil
				entity.moveIntent = Vec2{}
			} else {
				dx := entity.moveTarget.X - entity.Transform.Position.X
				dy := entity.moveTarget.Y - entity.Transform.Position.Y
				distance := math.Hypot(dx, dy)
				if distance <= MoveTargetArriveEpsilon {
					entity.moveTarget = nil
					entity.moveIntent = Vec2{}
				} else {
					entity.moveIntent = Vec2{X: dx / distance, Y: dy / distance}
				}
			}
		}
		if entity.moveIntent.X == 0 && entity.moveIntent.Y == 0 {
			continue
		}
		if w.isStunned(entity) {
			entity.moveIntent = Vec2{}
			continue
		}
		movingToTarget := entity.moveTarget != nil
		before := entity.Transform.Position
		moveDistance := entity.Speed * w.step * w.moveMultiplier(entity)
		if movingToTarget {
			remaining := math.Hypot(entity.moveTarget.X-before.X, entity.moveTarget.Y-before.Y)
			if remaining < moveDistance {
				moveDistance = remaining
			}
		}
		dx := entity.moveIntent.X * moveDistance
		dy := entity.moveIntent.Y * moveDistance
		// Substep each axis so an actor cannot tunnel through a solid tile at
		// high speed. Resolve axes independently to preserve wall sliding.
		substeps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy)) / MaxMovementSubstep))
		if substeps < 1 {
			substeps = 1
		}
		stepX, stepY := dx/float64(substeps), dy/float64(substeps)
		for step := 0; step < substeps; step++ {
			candidateX := Vec2{X: clampWorldCoordinate(entity.Transform.Position.X + stepX), Y: entity.Transform.Position.Y}
			if !w.blocked(candidateX, entity.Radius) {
				entity.Transform.Position.X = candidateX.X
			}
			candidateY := Vec2{X: entity.Transform.Position.X, Y: clampWorldCoordinate(entity.Transform.Position.Y + stepY)}
			if !w.blocked(candidateY, entity.Radius) {
				entity.Transform.Position.Y = candidateY.Y
			}
		}
		if entity.Transform.Position != before {
			entity.Revision++
		}
		if movingToTarget {
			if entity.Transform.Position == before {
				entity.moveTarget = nil
				entity.moveIntent = Vec2{}
				w.emitEvent(combat.Event{
					Type: combat.EventMoveRejected, SourceID: string(entity.ID),
					X: before.X, Y: before.Y, Reason: "path_blocked",
				})
			} else if math.Hypot(entity.moveTarget.X-entity.Transform.Position.X, entity.moveTarget.Y-entity.Transform.Position.Y) <= MoveTargetArriveEpsilon {
				entity.moveTarget = nil
				entity.moveIntent = Vec2{}
			}
		}
	}

	return w.publish()
}

func (w *World) processLifecycleCommands() {
	for count := 0; count < MaxLifecycleCommandsPerTick; count++ {
		select {
		case request := <-w.lifecycleInbox:
			request.result <- request.apply()
		default:
			return
		}
	}
}

func (w *World) blocked(position Vec2, radius float64) bool {
	return w.collision != nil && w.collision.Blocked(position, radius)
}

// Snapshot returns a detached copy that is safe to read from gateway goroutines.
func (w *World) Snapshot() Snapshot {
	snapshot := w.published.Load()
	if snapshot == nil {
		return Snapshot{}
	}
	copy := *snapshot
	copy.Entities = append([]Entity(nil), snapshot.Entities...)
	copy.Projectiles = append([]ProjectileState(nil), snapshot.Projectiles...)
	copy.Events = append([]combat.Event(nil), snapshot.Events...)
	return copy
}

// publish runs only on the zone actor.
func (w *World) publish() Snapshot {
	entities := make([]Entity, 0, len(w.entities))
	for _, entity := range w.entities {
		copy := *entity
		copy.moveIntent = Vec2{}
		copy.lastInputTick = 0
		entities = append(entities, copy)
	}
	sort.Slice(entities, func(i, j int) bool { return entities[i].ID < entities[j].ID })
	projectiles := make([]ProjectileState, 0, len(w.projectiles))
	for _, current := range w.projectiles {
		projectiles = append(projectiles, ProjectileState{
			ID: current.ID, SourceID: string(current.SourceID), SkillID: current.SkillID,
			X: current.Position.X, Y: current.Position.Y, Radius: current.Radius,
		})
	}
	events := append([]combat.Event(nil), w.events...)
	w.events = nil
	snapshot := Snapshot{ZoneKey: w.zoneKey, Tick: w.tick, Entities: entities, Projectiles: projectiles, Events: events}
	w.published.Store(&snapshot)
	return w.Snapshot()
}

func (w *World) publishDormant() Snapshot {
	last := w.published.Load()
	if last == nil {
		return w.publish()
	}
	snapshot := Snapshot{
		ZoneKey:     w.zoneKey,
		Tick:        w.tick,
		Entities:    last.Entities,
		Projectiles: nil,
		Events:      nil,
	}
	w.published.Store(&snapshot)
	return snapshot
}

func (w *World) collectCommands() []InputCommand {
	commands := make([]InputCommand, 0, min(len(w.inbox), MaxCommandsPerTick))
	for len(commands) < MaxCommandsPerTick {
		select {
		case command := <-w.inbox:
			commands = append(commands, command)
		default:
			return commands
		}
	}
	return commands
}

func validateEntity(entity Entity) error {
	if entity.ID == "" || len(entity.ID) > MaxEntityIDLength || strings.TrimSpace(string(entity.ID)) != string(entity.ID) {
		return fmt.Errorf("%w: invalid id", ErrInvalidEntity)
	}
	if entity.Kind != EntityPlayer && entity.Kind != EntityMonster {
		return fmt.Errorf("%w: invalid kind", ErrInvalidEntity)
	}
	if !validCoordinate(entity.Transform.Position.X) || !validCoordinate(entity.Transform.Position.Y) || !finite(entity.Speed) || entity.Speed < 0 || entity.Speed > MaxEntitySpeed || !finite(entity.Radius) || entity.Radius < 0 || entity.Radius > MaxEntityRadius {
		return fmt.Errorf("%w: invalid transform or movement stats", ErrInvalidEntity)
	}
	if !finite(entity.Vitals.HP) || !finite(entity.Vitals.MaxHP) || entity.Vitals.HP < 0 || entity.Vitals.MaxHP <= 0 || entity.Vitals.HP > entity.Vitals.MaxHP {
		return fmt.Errorf("%w: invalid vitals", ErrInvalidEntity)
	}
	if !finite(entity.Vitals.Shield) || entity.Vitals.Shield < 0 {
		return fmt.Errorf("%w: invalid shield", ErrInvalidEntity)
	}
	if entity.AI.Active {
		if entity.Kind != EntityMonster {
			return fmt.Errorf("%w: only monsters may enable AI", ErrInvalidEntity)
		}
		ai := entity.AI
		if !validCoordinate(ai.Home.X) || !validCoordinate(ai.Home.Y) ||
			!finite(ai.AggroRadius) || ai.AggroRadius <= 0 ||
			!finite(ai.LeashRadius) || ai.LeashRadius < ai.AggroRadius ||
			!finite(ai.AttackRange) || ai.AttackRange <= 0 ||
			!finite(ai.PatrolRadius) || ai.PatrolRadius < 0 ||
			!finite(ai.AttackDamage) || ai.AttackDamage < 0 {
			return fmt.Errorf("%w: invalid AI stats", ErrInvalidEntity)
		}
		if !finite(ai.SplashRadius) || ai.SplashRadius < 0 || !finite(ai.SplashDamage) || ai.SplashDamage < 0 {
			return fmt.Errorf("%w: invalid AI splash stats", ErrInvalidEntity)
		}
		for id, threat := range ai.ThreatTable {
			if id == "" || !finite(threat) || threat < 0 {
				return fmt.Errorf("%w: invalid AI threat table", ErrInvalidEntity)
			}
		}
	}
	return nil
}

func validCoordinate(value float64) bool {
	return finite(value) && math.Abs(value) <= MaxWorldCoordinate
}

func clampWorldCoordinate(value float64) float64 {
	return math.Max(-MaxWorldCoordinate, math.Min(MaxWorldCoordinate, value))
}

// clampMoveTarget limits a click destination to a bounded radius around the
// actor. It prevents a stale or malicious click from sending an actor across
// the whole map in one request.
func clampMoveTarget(from, target Vec2) Vec2 {
	dx, dy := target.X-from.X, target.Y-from.Y
	distance := math.Hypot(dx, dy)
	if distance <= MaxMoveTargetDistance {
		return target
	}
	scale := MaxMoveTargetDistance / distance
	return Vec2{X: from.X + dx*scale, Y: from.Y + dy*scale}
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}
