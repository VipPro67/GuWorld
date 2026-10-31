package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
	"github.com/vippro67/GuWorld/backend/internal/realtime/config"
	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
	"github.com/vippro67/GuWorld/backend/internal/realtime/persistence"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	joinReadTimeout       = 5 * time.Second
	joinLimitPerMinute    = 15
	maxTrackedJoinPlayers = 10_000
	playerSpeed           = 4.0
	playerRadius          = 0.4
)

func realtimePlayerSpeed(speed float64) float64 {
	if speed <= 0 {
		return playerSpeed
	}
	return speed
}

type JoinStateLoader func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error)

// WorldStateStore is the durable checkpoint adapter used on join/leave and by
// the periodic checkpoint worker.
type WorldStateStore interface {
	Load(context.Context, uuid.UUID) (persistence.State, bool, error)
	Save(context.Context, persistence.State) error
}

// TribulationZoneLoader resolves a per-player event encounter and its dynamic
// zone map. ok is false when the player has no active tribulation event.
type TribulationZoneLoader func(ctx context.Context, playerID uuid.UUID) (sim.ZoneKey, realtimeworld.ZoneMap, bool, error)

// CombatInstanceZone is a server-authored arena or training map instance.
// OverrideState is optional and lets isolated modes supply temporary vitals
// while keeping the player's durable character state untouched.
type CombatInstanceZone struct {
	Key                sim.ZoneKey
	Map                realtimeworld.ZoneMap
	OverrideState      *handlers.RealtimeJoinState
	SpawnPosition      *sim.Vec2
	PartyIDOverride    string
	Isolated           bool
	PersistCombatState bool
}

type CombatInstanceZoneLoader func(ctx context.Context, playerID uuid.UUID, state handlers.RealtimeJoinState) (CombatInstanceZone, bool, error)

// ApertureStateSink persists authoritative entity HP from an aperture
// tribulation snapshot (monsters and resource nodes).
type ApertureStateSink func(ctx context.Context, playerID uuid.UUID, snapshot sim.Snapshot) error

type CombatInstanceStateSink func(ctx context.Context, playerID uuid.UUID, key sim.ZoneKey, entity sim.Entity) error

type Admission struct {
	config   config.Config
	manager  *realtimeworld.Manager
	load     JoinStateLoader
	compiled *realtimeeffects.CompiledCatalog
	states   WorldStateStore
	paused   atomic.Bool

	mu                sync.Mutex
	sessions          map[uuid.UUID]session
	joinAttempts      map[uuid.UUID][]time.Time
	tribulationZones  TribulationZoneLoader
	combatZones       CombatInstanceZoneLoader
	apertureStateSink ApertureStateSink
	combatStateSink   CombatInstanceStateSink
}

type session struct {
	playerID            uuid.UUID
	regionID            string
	lease               realtimeworld.OwnershipLease
	mapVersion          string
	contentVersion      string
	skills              []combat.Skill
	loadoutRevision     string
	movementSpeed       float64
	mana                *manaSettlement
	locationRevision    *atomic.Int64
	hasLocationRevision bool
	closing             bool
	transferring        bool
	lastTransferAt      time.Time
	transferBlockedAt   time.Time
	safeX               float64
	safeY               float64
	mapWidth            float64
	mapHeight           float64
	persistState        bool
	persistCombatState  bool
}

// edgeTransferCooldown prevents border ping-pong when a player holds movement
// into the wall while the client reconnects into the neighbouring zone.
const edgeTransferCooldown = 2 * time.Second

type manaSettlement struct {
	mu                 sync.Mutex
	last               float64
	lastUses           [13]uint64
	lastMasteryUses    [13]uint64
	lastKillerMoveUses [13]uint64
	lastBasicUses      uint64
}

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string     { return e.Message }
func (e *Error) HTTPStatus() int   { return e.Status }
func (e *Error) ErrorCode() string { return e.Code }

func NewAdmission(cfg config.Config, manager *realtimeworld.Manager, load JoinStateLoader) *Admission {
	if load == nil {
		load = handlers.LoadRealtimeJoinState
	}
	admission := &Admission{
		config: cfg, manager: manager, load: load,
		sessions: make(map[uuid.UUID]session), joinAttempts: make(map[uuid.UUID][]time.Time),
	}
	admission.paused.Store(cfg.AdmissionPaused)
	return admission
}

// SetLoadoutCatalog installs the compiled Gu catalog used for join-time
// loadouts. It must be called during startup before admission.
func (a *Admission) SetLoadoutCatalog(catalog *realtimeeffects.CompiledCatalog) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.compiled = catalog
}

// SetAdmissionPaused blocks new joins without stopping zones. Existing sessions
// continue until Drain is called.
func (a *Admission) SetAdmissionPaused(paused bool) {
	a.paused.Store(paused)
}

// Drain pauses admission, checkpoints and releases every process-local session.
// WebSocket clients receive no further snapshots; they must reconnect to legacy
// mode. Returns the number of drained sessions.
func (a *Admission) Drain() int {
	a.paused.Store(true)
	a.mu.Lock()
	active := make(map[uuid.UUID]session, len(a.sessions))
	for controllerID, current := range a.sessions {
		active[controllerID] = current
	}
	a.mu.Unlock()
	drained := 0
	for controllerID, current := range active {
		if current.transferring {
			continue
		}
		if err := a.flushState(current); err != nil {
			logger.S.Errorf("Realtime drain checkpoint failed for %s: %v", current.playerID, err)
			a.mu.Lock()
			if held, ok := a.sessions[controllerID]; ok && held.lease.Token == current.lease.Token {
				held.closing = true
				a.sessions[controllerID] = held
			}
			a.mu.Unlock()
			continue
		}
		a.manager.LeaveCharacter(current.lease)
		drained++
		a.mu.Lock()
		delete(a.sessions, controllerID)
		a.mu.Unlock()
	}
	return drained
}

func (a *Admission) ActiveSessionCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return len(a.sessions)
}

func (a *Admission) SetStateStore(store WorldStateStore) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.states = store
}

// SetTribulationZoneLoader installs the per-event encounter builder used when a
// player joins aperture_tribulation. Passing nil keeps the startup catalog.
func (a *Admission) SetTribulationZoneLoader(loader TribulationZoneLoader) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.tribulationZones = loader
}

// SetCombatInstanceZoneLoader installs the loader for authenticated arena and
// training instances. It is consulted only when the durable player row has a
// server-created world_instance_id using the combat: namespace.
func (a *Admission) SetCombatInstanceZoneLoader(loader CombatInstanceZoneLoader) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.combatZones = loader
}

// SetApertureStateSink installs the durable HP sink for tribulation monsters and
// resource nodes. It runs after player checkpoints and handoffs.
func (a *Admission) SetApertureStateSink(sink ApertureStateSink) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.apertureStateSink = sink
}

// SetCombatInstanceStateSink persists temporary vitals and position for
// isolated combat sessions that must survive disconnect/reconnect.
func (a *Admission) SetCombatInstanceStateSink(sink CombatInstanceStateSink) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.combatStateSink = sink
}

// CheckpointActive persists each live controller's position, HP and mana.
// Calls may overlap Leave; each session serializes its mana settlement.
func (a *Admission) CheckpointActive() {
	a.mu.Lock()
	active := make(map[uuid.UUID]session, len(a.sessions))
	for controllerID, current := range a.sessions {
		active[controllerID] = current
	}
	a.mu.Unlock()
	for controllerID, current := range active {
		if current.transferring {
			continue
		}
		if err := a.flushState(current); err != nil {
			logger.S.Errorf("Realtime periodic checkpoint failed for %s: %v", current.playerID, err)
			continue
		}
		if current.closing {
			a.manager.LeaveCharacter(current.lease)
			a.mu.Lock()
			// A reconnect can replace this controller while the checkpoint is
			// running. Never let the old checkpoint delete the replacement.
			if held, ok := a.sessions[controllerID]; ok && held.lease.Token == current.lease.Token {
				delete(a.sessions, controllerID)
			}
			a.mu.Unlock()
		}
	}
}

func (a *Admission) Join(ctx context.Context, playerID, controllerID uuid.UUID, request protocol.JoinRequest) (any, error) {
	if a.manager == nil {
		reason := handlers.RealtimeRuntimeError()
		if reason == "" {
			reason = "realtime manager was not installed; check map artifacts and startup logs"
		}
		return nil, admissionError(503, "realtime_manager_unavailable", "Realtime world manager is unavailable: "+reason)
	}
	if !a.config.Enabled {
		return nil, admissionError(503, "realtime_disabled", "Realtime world runtime is unavailable")
	}
	if !a.config.PhaserClientEnabled {
		return nil, admissionError(503, "realtime_client_disabled", "Realtime world client is unavailable")
	}
	if a.paused.Load() {
		return nil, admissionError(503, "realtime_admission_paused", "Realtime world admission is paused for cutover/rollback")
	}
	if !a.config.AllowsJoin(playerID.String(), request.ZoneID) {
		return nil, admissionError(403, "realtime_world_forbidden", "Player or map is not available")
	}
	if !a.config.AllowsWorldJoin(playerID.String(), request.ZoneID, request.MapVersion, request.ContentVersion) {
		return nil, admissionError(409, "realtime_version_mismatch", "Map or content version is stale")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	a.mu.Lock()
	if active, exists := a.sessions[controllerID]; exists {
		a.mu.Unlock()
		if active.playerID != playerID {
			return nil, admissionError(403, "realtime_controller_mismatch", "Controller is bound to another character")
		}
		if active.mapVersion != request.MapVersion || active.contentVersion != request.ContentVersion || active.lease.ZoneKey.MapID != request.ZoneID {
			return nil, admissionError(409, "realtime_session_already_joined", "Connection already joined a different world version or zone")
		}
		return a.joinResponse(active, controllerID)
	}
	a.mu.Unlock()
	if !a.allowJoinAttempt(playerID, time.Now()) {
		return nil, admissionError(429, "realtime_join_rate_limited", "Too many realtime join attempts")
	}
	a.reapClosingCharacter(playerID)
	if a.manager.OwnsCharacter(sim.CharacterID(playerID.String())) {
		return nil, admissionError(409, "realtime_character_already_owned", "Character already has an active realtime controller")
	}
	joinCtx, cancel := context.WithTimeout(ctx, joinReadTimeout)
	defer cancel()
	state, err := a.load(joinCtx, playerID)
	if err != nil {
		return nil, admissionError(503, "realtime_character_unavailable", "Could not load authoritative character state")
	}
	if state.RegionID == "" || state.MapID != request.ZoneID {
		return nil, admissionError(409, "realtime_map_mismatch", "Requested map does not match the character's current region")
	}
	isSpatialRealm := strings.HasPrefix(state.WorldInstanceID, "realm:")
	if state.MapInstanceID != "" && !isSpatialRealm {
		return nil, admissionError(409, "realtime_instance_conflict", "Character is already assigned to another map instance")
	}
	if state.NeedsOnboarding {
		return nil, admissionError(409, "realtime_character_not_ready", "Character cannot join the realtime world in its current state")
	}
	if math.IsNaN(state.HP) || math.IsInf(state.HP, 0) || math.IsNaN(state.BaseHP) || math.IsInf(state.BaseHP, 0) || state.HP <= 0 || state.BaseHP <= 0 {
		return nil, admissionError(409, "realtime_character_invalid", "Character vitals are invalid for admission")
	}
	var key sim.ZoneKey
	tribulationZone := false
	privateCombatZone := false
	spatialRealmZone := false
	var privateCombatSpawn *sim.Vec2
	persistCombatState := false
	if request.ZoneID == "aperture_tribulation" {
		a.mu.Lock()
		loader := a.tribulationZones
		a.mu.Unlock()
		if loader == nil {
			return nil, admissionError(503, "realtime_tribulation_unavailable", "The private aperture tribulation map loader is not configured; check the realtime map artifact and startup logs")
		}
		loadedKey, zoneMap, ok, loadErr := loader(ctx, playerID)
		if loadErr != nil {
			logger.S.Errorf("Realtime tribulation map load failed for player %s: %v", playerID, loadErr)
			return nil, admissionError(503, "realtime_tribulation_unavailable", "The aperture tribulation map or encounter could not be loaded; check backend startup logs and map/content versions")
		}
		if !ok {
			return nil, admissionError(409, "realtime_tribulation_inactive", "No active tribulation event for this character")
		}
		if err := a.manager.RegisterZoneMap(loadedKey, zoneMap); err != nil {
			if errors.Is(err, realtimeworld.ErrZoneFull) {
				return nil, admissionError(429, "realtime_zone_full", "Realtime zone instances are at capacity")
			}
			logger.S.Errorf("Realtime tribulation map registration failed for player %s: %v", playerID, err)
			return nil, admissionError(503, "realtime_tribulation_unavailable", "The aperture tribulation map could not be configured; check its artifact and startup logs")
		}
		key = loadedKey
		tribulationZone = true
	}
	if !tribulationZone && strings.HasPrefix(state.WorldInstanceID, "combat:") {
		a.mu.Lock()
		loader := a.combatZones
		a.mu.Unlock()
		if loader == nil {
			return nil, admissionError(503, "realtime_combat_instance_unavailable", "The private combat map loader is not configured")
		}
		combatZone, found, loadErr := loader(joinCtx, playerID, state)
		if loadErr != nil {
			logger.S.Errorf("Realtime combat map load failed for player %s: %v", playerID, loadErr)
			return nil, admissionError(503, "realtime_combat_instance_unavailable", "The private combat map could not be loaded")
		}
		if !found {
			return nil, admissionError(409, "realtime_combat_instance_inactive", "No active realtime combat session exists for this character")
		}
		if combatZone.Key.MapID != request.ZoneID || combatZone.Key.InstanceID == "" {
			return nil, admissionError(503, "realtime_combat_instance_unavailable", "The private combat map has an invalid instance key")
		}
		if combatZone.OverrideState != nil {
			state = *combatZone.OverrideState
			if state.RegionID == "" || state.MapID != request.ZoneID || state.MapInstanceID != "" {
				return nil, admissionError(503, "realtime_combat_instance_unavailable", "The private combat map returned invalid character state")
			}
			if math.IsNaN(state.HP) || math.IsInf(state.HP, 0) || math.IsNaN(state.BaseHP) || math.IsInf(state.BaseHP, 0) || state.HP <= 0 || state.BaseHP <= 0 {
				return nil, admissionError(503, "realtime_combat_instance_unavailable", "The private combat map returned invalid character vitals")
			}
		}
		if err := a.manager.RegisterZoneMap(combatZone.Key, combatZone.Map); err != nil {
			if errors.Is(err, realtimeworld.ErrZoneFull) {
				return nil, admissionError(429, "realtime_zone_full", "Realtime zone instances are at capacity")
			}
			logger.S.Errorf("Realtime combat map registration failed for player %s: %v", playerID, err)
			return nil, admissionError(503, "realtime_combat_instance_unavailable", "The private combat map could not be configured")
		}
		key = combatZone.Key
		privateCombatZone = combatZone.Isolated
		privateCombatSpawn = combatZone.SpawnPosition
		persistCombatState = combatZone.PersistCombatState
	} else if isSpatialRealm {
		a.mu.Lock()
		loader := a.combatZones
		a.mu.Unlock()
		if loader == nil {
			return nil, admissionError(503, "realtime_spatial_realm_unavailable", "The realtime spatial realm loader is not configured")
		}
		realmZone, found, loadErr := loader(joinCtx, playerID, state)
		if loadErr != nil {
			logger.S.Errorf("Realtime spatial realm load failed for player %s: %v", playerID, loadErr)
			return nil, admissionError(503, "realtime_spatial_realm_unavailable", "The realtime spatial realm could not be loaded")
		}
		if !found || realmZone.Key.MapID != request.ZoneID || realmZone.Key.InstanceID == "" {
			return nil, admissionError(409, "realtime_spatial_realm_inactive", "No active realtime spatial realm exists for this character")
		}
		if err := a.manager.RegisterZoneMap(realmZone.Key, realmZone.Map); err != nil {
			if errors.Is(err, realtimeworld.ErrZoneFull) {
				return nil, admissionError(429, "realtime_zone_full", "Realtime zone instances are at capacity")
			}
			logger.S.Errorf("Realtime spatial realm map registration failed for player %s: %v", playerID, err)
			return nil, admissionError(503, "realtime_spatial_realm_unavailable", "The realtime spatial realm map could not be configured")
		}
		key = realmZone.Key
		if realmZone.PartyIDOverride != "" {
			state.PartyID = realmZone.PartyIDOverride
		}
		spatialRealmZone = true
	}
	if state.MapInstanceID != "" && !spatialRealmZone {
		return nil, admissionError(409, "realtime_instance_conflict", "Character is already assigned to another map instance")
	}
	if !tribulationZone {
		if key.MapID == "" {
			var zoneErr error
			key, zoneErr = a.manager.ZoneKeyForMapWithCapacity(request.ZoneID)
			if zoneErr != nil {
				if errors.Is(zoneErr, realtimeworld.ErrZoneFull) {
					return nil, admissionError(429, "realtime_zone_full", "Realtime zone instances are at capacity")
				}
				return nil, admissionError(403, "realtime_map_unavailable", "Realtime zone is not configured for this map")
			}
		}
	}
	lease, err := a.manager.ClaimCharacter(sim.CharacterID(playerID.String()), controllerID.String(), key)
	if err != nil {
		return nil, mapManagerError(err)
	}
	attached := false
	defer func() {
		if !attached {
			a.manager.ReleaseCharacter(lease)
		}
	}()

	a.mu.Lock()
	compiled := a.compiled
	stateStore := a.states
	a.mu.Unlock()
	baseX := float64(state.X)
	baseY := float64(state.Y)
	position := a.manager.LegacyPosition(key, baseX, baseY)
	if privateCombatSpawn != nil {
		position = *privateCombatSpawn
	}
	locationRevision := &atomic.Int64{}
	locationRevision.Store(state.WorldLocationRevision)
	hasLocationRevision := state.HasWorldLocationRevision
	if stateStore == nil && state.HasWorldLocation && state.MapID == request.ZoneID {
		baseX, baseY = state.WorldX, state.WorldY
		position = sim.Vec2{X: baseX, Y: baseY}
	}
	baseHP := state.HP
	baseShield := 0.0
	var durableCombat *persistence.CombatState
	if stateStore != nil && !privateCombatZone {
		durable, ok, loadErr := stateStore.Load(ctx, playerID)
		if loadErr != nil {
			return nil, admissionError(503, "realtime_checkpoint_unavailable", "Could not load the realtime world checkpoint")
		}
		if ok && durable.ZoneID == request.ZoneID && durable.InstanceID == key.InstanceID &&
			a.manager.ContainsPosition(key, sim.Vec2{X: durable.X, Y: durable.Y}) {
			locationRevision.Store(durable.LocationRevision)
			hasLocationRevision = true
			if durable.MapVersion == request.MapVersion && durable.ContentVersion == request.ContentVersion {
				baseX, baseY = durable.X, durable.Y
				position = sim.Vec2{X: baseX, Y: baseY}
				if durable.HP > 0 {
					baseHP = durable.HP
				}
				baseShield = durable.Shield
			}
		}
		// Cooldowns, statuses and hidden-proc counters follow the character
		// across zones and map revisions. Only the content (rules) version must
		// match so a re-authored skill cannot inherit a stale timer.
		if ok && durable.ContentVersion == request.ContentVersion {
			durableCombat = durable.Combat
		}
	}
	maxHP := math.Max(math.Max(state.HP, state.BaseHP), baseHP)
	equipped := make([]realtimeeffects.EquippedGu, 0, len(state.EquippedGu))
	for _, instance := range state.EquippedGu {
		equipped = append(equipped, realtimeeffects.EquippedGu{
			InstanceID: instance.InstanceID, GU: instance.GuID, Rank: instance.Rank,
		})
	}
	loadout := realtimeeffects.BuildLoadout(compiled, equipped)
	killerMoves := make([]realtimeeffects.KillerMove, 0, len(state.KillerMoves))
	for _, move := range state.KillerMoves {
		killerMoves = append(killerMoves, move.ProjectRealtimeEffects())
	}
	loadout.Skills, loadout.Revision = realtimeeffects.AppendKillerMoveLoadout(loadout.Skills, loadout.Revision, killerMoves)
	skills := make([]combat.Skill, 0, len(loadout.Skills))
	for _, skill := range loadout.Skills {
		if skill.Slot < 1 || skill.Slot > len(state.EquippedGu) {
			skills = append(skills, skill)
			continue
		}
		entry := state.EquippedGu[skill.Slot-1]
		// Test/demo loaders without a character projection retain their authored
		// profile values; live DB admission always provides a ranked player.
		if state.Rank > 0 {
			if !entry.Usable {
				continue
			}
			skill.GuInstanceID = entry.InstanceID
			skill.GuHunger = entry.Hunger
			skill.ManaCost = entry.ManaCost
			guRatio := 1.0
			if skill.GuPowerRatio > 0 {
				guRatio = skill.GuPowerRatio
			}
			baseDmg := skill.Damage * guRatio
			if skill.PlayerAtkRatio > 0 && state.BaseAttack > 0 {
				baseDmg += state.BaseAttack * skill.PlayerAtkRatio
			}
			skill.Damage = baseDmg * entry.DamageScale
			if entry.ShieldScale > 0 {
				skill.Shield *= entry.ShieldScale
			}
			skill.BacklashPct = entry.BacklashPct
			skill.Path = entry.Path
			skill.Rank = entry.Rank
		}
		skills = append(skills, skill)
	}
	loadoutRevision := loadout.Revision
	safeSpawn, ok := a.manager.SafeSpawn(key, position, playerRadius)
	if !ok {
		return nil, admissionError(503, "realtime_spawn_unavailable", "No safe spawn is available in this zone")
	}
	position = safeSpawn
	playerCombat := sim.NewPlayerCombatWithLoadout(skills)
	if state.Rank > 0 {
		playerCombat.Mana = state.CurrentMana
		playerCombat.MaxMana = state.MaxMana
		playerCombat.ManaRegenPerSecond = state.ManaRegenPerSecond
		playerCombat.BasicAttackDamage = state.BaseAttack
	}
	if durableCombat != nil {
		if zoneTick, ok := a.manager.ZoneTick(key); ok {
			restoreCombat(&playerCombat, *durableCombat, zoneTick)
		}
	}
	moveSpeed := realtimePlayerSpeed(state.MovementSpeed)
	entity := sim.Entity{
		ID: sim.EntityID(playerID.String()), CharacterID: sim.CharacterID(playerID.String()), Kind: sim.EntityPlayer,
		Name:           state.Username,
		Transform:      sim.Transform{Position: position},
		Vitals:         sim.Vitals{HP: baseHP, MaxHP: maxHP, Shield: baseShield},
		Defense:        state.Defense,
		DaoProfile:     state.DaoProfile,
		PathResistance: state.PathResistance,
		Speed:          moveSpeed,
		Radius:         playerRadius,
		Faction:        "player",
		PartyID:        state.PartyID,
		PvPEnabled:     state.PvPEnabled,
		Combat:         playerCombat,
	}
	lease, err = a.manager.AttachCharacterEntity(lease, entity)
	if err != nil {
		return nil, mapManagerError(err)
	}
	attached = true
	zoneMap, _ := a.manager.ZoneMapFor(key)
	var mana *manaSettlement
	if !privateCombatZone {
		mana = &manaSettlement{last: playerCombat.Mana}
	}
	active := session{
		playerID: playerID, regionID: state.RegionID, lease: lease, mapVersion: request.MapVersion, contentVersion: request.ContentVersion,
		skills: skills, loadoutRevision: loadoutRevision, movementSpeed: moveSpeed,
		mana:             mana,
		locationRevision: locationRevision, hasLocationRevision: hasLocationRevision,
		safeX: safeSpawn.X, safeY: safeSpawn.Y,
		mapWidth: float64(zoneMap.Width), mapHeight: float64(zoneMap.Height),
		persistState:       !privateCombatZone,
		persistCombatState: persistCombatState,
	}
	a.mu.Lock()
	a.sessions[controllerID] = active
	a.mu.Unlock()
	return a.joinResponse(active, controllerID)
}

func (a *Admission) reapClosingCharacter(playerID uuid.UUID) {
	a.mu.Lock()
	var controllerID uuid.UUID
	var active session
	for id, current := range a.sessions {
		if current.playerID == playerID && current.closing {
			controllerID, active = id, current
			break
		}
	}
	a.mu.Unlock()
	if controllerID == uuid.Nil || a.flushState(active) != nil {
		return
	}
	a.manager.LeaveCharacter(active.lease)
	a.mu.Lock()
	if held, ok := a.sessions[controllerID]; ok && held.lease.Token == active.lease.Token {
		delete(a.sessions, controllerID)
	}
	a.mu.Unlock()
}

func (a *Admission) allowJoinAttempt(playerID uuid.UUID, now time.Time) bool {
	cutoff := now.Add(-time.Minute)
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.joinAttempts) >= maxTrackedJoinPlayers {
		for id, attempts := range a.joinAttempts {
			if !hasRecentJoinAttempt(attempts, cutoff) {
				delete(a.joinAttempts, id)
			}
		}
		if len(a.joinAttempts) >= maxTrackedJoinPlayers {
			if _, tracked := a.joinAttempts[playerID]; !tracked {
				return false
			}
		}
	}
	attempts := a.joinAttempts[playerID]
	kept := attempts[:0]
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			kept = append(kept, attempt)
		}
	}
	if len(kept) >= joinLimitPerMinute {
		a.joinAttempts[playerID] = kept
		return false
	}
	kept = append(kept, now)
	a.joinAttempts[playerID] = kept
	return true
}

func hasRecentJoinAttempt(attempts []time.Time, cutoff time.Time) bool {
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			return true
		}
	}
	return false
}

func (a *Admission) Leave(playerID, controllerID uuid.UUID) error {
	a.mu.Lock()
	active, exists := a.sessions[controllerID]
	a.mu.Unlock()
	if !exists || active.playerID != playerID {
		// A retried checkpoint may have already released the lease. Leave is
		// idempotent in that case so the client can finish disconnecting.
		if !a.manager.OwnsCharacter(sim.CharacterID(playerID.String())) {
			return nil
		}
		return admissionError(409, "realtime_session_not_joined", "This connection has no active realtime session")
	}
	if err := a.flushState(active); err != nil {
		// Keep the lease owned and let the periodic checkpoint retry instead of
		// dropping unsettled mana/hunger or releasing a dirty character.
		a.markClosing(controllerID, active.lease.Token)
		return admissionError(503, "realtime_checkpoint_unavailable", "Could not save character combat resources")
	}
	if !a.manager.LeaveCharacter(active.lease) {
		return admissionError(409, "realtime_lease_stale", "Realtime session lease is no longer current")
	}
	a.mu.Lock()
	delete(a.sessions, controllerID)
	a.mu.Unlock()
	return nil
}

func (a *Admission) Input(playerID, controllerID uuid.UUID, input protocol.Input) error {
	return a.submitCommand(playerID, controllerID, sim.InputCommand{
		Seq: input.Seq, Move: moveVector(input.Move),
	})
}

// MoveTarget submits a server-bounded click-to-move destination for the active
// controller. The simulation clamps it near the actor before it moves.
func (a *Admission) MoveTarget(playerID, controllerID uuid.UUID, target protocol.MoveTarget) error {
	return a.submitCommand(playerID, controllerID, sim.InputCommand{
		Seq: target.Seq, Target: &sim.Vec2{X: target.X, Y: target.Y},
	})
}

// BeginHandoff flushes the live character and pauses zone checkpoints so a
// legacy HTTP world mutation (teleport or tribulation transition) can read the
// real position and persist its destination in one transaction. The lease is
// released by FinishHandoff after the commit, or resumed by AbortHandoff.
func (a *Admission) BeginHandoff(playerID uuid.UUID) (bool, error) {
	a.mu.Lock()
	var controllerID uuid.UUID
	var active session
	found := false
	for id, candidate := range a.sessions {
		if candidate.playerID != playerID || candidate.closing || candidate.transferring {
			continue
		}
		candidate.transferring = true
		candidate.lastTransferAt = time.Now()
		a.sessions[id] = candidate
		controllerID, active, found = id, candidate, true
		break
	}
	a.mu.Unlock()
	if !found {
		return false, nil
	}
	if err := a.flushState(active); err != nil {
		a.finishFailedTransfer(active, controllerID)
		return false, err
	}
	return true, nil
}

// TribulationEnemyStates returns the authoritative alive/dead persistent IDs
// of the player's live aperture tribulation encounter. ok is false when the
// player has no active tribulation realtime session. The tribulation zone is
// private, so its snapshot covers the whole encounter.
func (a *Admission) TribulationEnemyStates(playerID uuid.UUID) (alive, dead map[string]bool, ok bool) {
	a.mu.Lock()
	var active session
	found := false
	for _, candidate := range a.sessions {
		if candidate.playerID != playerID || candidate.closing {
			continue
		}
		active, found = candidate, true
		break
	}
	a.mu.Unlock()
	if !found || active.lease.ZoneKey.MapID != "aperture_tribulation" {
		return nil, nil, false
	}
	snapshot, err := a.manager.SnapshotForLease(active.lease)
	if err != nil {
		return nil, nil, false
	}
	alive = make(map[string]bool)
	dead = make(map[string]bool)
	for _, entity := range snapshot.Entities {
		if entity.Kind != sim.EntityMonster || entity.Faction != "monster" {
			continue
		}
		persistentID := strings.TrimSpace(entity.AI.PersistentID)
		if persistentID == "" {
			continue
		}
		if entity.Vitals.HP > 0 {
			alive[persistentID] = true
		} else {
			dead[persistentID] = true
		}
	}
	return alive, dead, true
}

// PlayerLocation returns the live region and legacy-projected position for a
// zone-owned character. Legacy HTTP reads (teleport directory) use it instead
// of the periodic checkpoint coordinates.
func (a *Admission) PlayerLocation(playerID uuid.UUID) (string, int, int, bool) {
	a.mu.Lock()
	var active session
	found := false
	for _, candidate := range a.sessions {
		if candidate.playerID == playerID && !candidate.closing {
			active, found = candidate, true
			break
		}
	}
	a.mu.Unlock()
	if !found {
		return "", 0, 0, false
	}
	snapshot, err := a.manager.SnapshotForLease(active.lease)
	if err != nil {
		return "", 0, 0, false
	}
	for _, entity := range snapshot.Entities {
		if entity.CharacterID != active.lease.CharacterID {
			continue
		}
		x, y, ok := a.manager.LocalToLegacy(active.lease.ZoneKey, entity.Transform.Position)
		if !ok {
			return "", 0, 0, false
		}
		return active.regionID, int(math.Round(x)), int(math.Round(y)), true
	}
	return "", 0, 0, false
}

// AbortHandoff resumes periodic checkpoints after a refused world mutation.
func (a *Admission) AbortHandoff(playerID uuid.UUID) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for controllerID, active := range a.sessions {
		if active.playerID == playerID && active.transferring {
			active.transferring = false
			a.sessions[controllerID] = active
		}
	}
}

// FinishHandoff releases the suspended lease and session after the legacy
// handler persisted the new region and position. The client rejoins the
// destination zone with a fresh lease.
func (a *Admission) FinishHandoff(playerID uuid.UUID) {
	a.mu.Lock()
	var lease realtimeworld.OwnershipLease
	found := false
	for controllerID, active := range a.sessions {
		if active.playerID != playerID {
			continue
		}
		lease = active.lease
		delete(a.sessions, controllerID)
		found = true
		break
	}
	a.mu.Unlock()
	if !found {
		return
	}
	if !a.manager.LeaveCharacter(lease) {
		logger.S.Warnf("Realtime teleport left a stale lease for %s; disconnect will retry release", playerID)
	}
}

// TransferAtEdge performs the authoritative border handoff for one owned
// character. The destination zone and map-local arrival are resolved from the
// source map's authored edge portal; the caller only supplies the authoritative
// entity position from its snapshot.
func (a *Admission) TransferAtEdge(playerID, controllerID uuid.UUID, position sim.Vec2) (protocol.Transfer, bool, error) {
	a.mu.Lock()
	active, exists := a.sessions[controllerID]
	if !exists || active.playerID != playerID {
		a.mu.Unlock()
		return protocol.Transfer{}, false, admissionError(409, "realtime_session_not_joined", "This connection has no active realtime session")
	}
	if active.closing || time.Since(active.lastTransferAt) < edgeTransferCooldown || time.Now().Before(active.transferBlockedAt) {
		a.mu.Unlock()
		return protocol.Transfer{}, false, nil
	}
	margin := realtimeworld.EdgeTriggerMargin
	if active.mapWidth > margin && active.mapHeight > margin &&
		position.X > margin && position.X < active.mapWidth-margin &&
		position.Y > margin && position.Y < active.mapHeight-margin {
		a.mu.Unlock()
		return protocol.Transfer{}, false, nil
	}
	a.mu.Unlock()

	crossing, ok := a.manager.ResolveEdgeForLease(active.lease, position)
	if !ok {
		return protocol.Transfer{}, false, nil
	}
	a.markTransferAttempt(controllerID, active.lease.Token)

	validateCtx, cancelValidate := context.WithTimeout(context.Background(), 3*time.Second)
	validateErr := handlers.ValidateRealtimeRegionTransfer(validateCtx, playerID, crossing.ToZoneID)
	cancelValidate()
	if validateErr != nil {
		if errors.Is(validateErr, handlers.ErrRealtimeRegionTransferDenied) {
			a.blockTransfer(controllerID, active.lease.Token, 10*time.Second)
			return protocol.Transfer{}, false, admissionError(403, "realtime_transfer_denied", validateErr.Error())
		}
		return protocol.Transfer{}, false, admissionError(503, "realtime_transfer_unavailable", "Could not validate the region handoff")
	}

	hp := 0.0
	if snapshot, err := a.manager.SnapshotForLease(active.lease); err == nil {
		for _, entity := range snapshot.Entities {
			if entity.CharacterID == active.lease.CharacterID {
				hp = entity.Vitals.HP
				break
			}
		}
	}

	if !a.beginTransfer(controllerID, active.lease.Token) {
		return protocol.Transfer{}, false, nil
	}
	if err := a.flushState(active); err != nil {
		a.finishFailedTransfer(active, controllerID)
		return protocol.Transfer{}, false, admissionError(503, "realtime_checkpoint_unavailable", "Could not save character before crossing the border")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	result, err := handlers.ApplyRealtimeRegionTransfer(ctx, handlers.RealtimeRegionTransfer{
		PlayerID:     active.playerID,
		TargetRegion: crossing.ToZoneID,
		TargetZoneID: crossing.TargetKey.MapID, TargetInstanceID: crossing.TargetKey.InstanceID,
		LocationRevision: active.locationRevision.Load(),
		LocalX:           crossing.LocalX,
		LocalY:           crossing.LocalY,
		LegacyOriginX:    crossing.LegacyOriginX,
		LegacyOriginY:    crossing.LegacyOriginY,
		LegacyScale:      crossing.LegacyScale,
		HP:               hp,
	})
	if err != nil {
		a.finishFailedTransfer(active, controllerID)
		if errors.Is(err, handlers.ErrRealtimeRegionTransferDenied) {
			a.blockTransfer(controllerID, active.lease.Token, 10*time.Second)
			return protocol.Transfer{}, false, admissionError(403, "realtime_transfer_denied", err.Error())
		}
		logger.S.Errorf("Realtime edge transfer failed for %s -> %s: %v", playerID, crossing.ToZoneID, err)
		return protocol.Transfer{}, false, admissionError(503, "realtime_transfer_unavailable", "Could not persist the region handoff")
	}

	if !a.manager.LeaveCharacter(active.lease) {
		logger.S.Warnf("Realtime edge transfer left a stale lease for %s; disconnect will retry release", playerID)
	}
	// Drop the session before publishing: the SSE/publish can make the client
	// reconnect instantly, and the old socket's Disconnect must become a no-op
	// instead of trying to flush a lease that no longer exists.
	a.mu.Lock()
	delete(a.sessions, controllerID)
	a.mu.Unlock()
	handlers.PublishRegionTransfer(result)

	return protocol.Transfer{
		FromZoneID: active.lease.ZoneKey.MapID,
		ToZoneID:   crossing.ToZoneID,
		TransferID: uuid.NewString(),
		TargetX:    crossing.LocalX,
		TargetY:    crossing.LocalY,
		LegacyX:    result.Player.X,
		LegacyY:    result.Player.Y,
		HP:         result.Player.HP,
	}, true, nil
}

// markClosing keeps the session and its lease owned after a failed final
// checkpoint. The periodic worker retries flushState and releases the lease
// once the database recovers.
func (a *Admission) markClosing(controllerID uuid.UUID, leaseToken string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if held, ok := a.sessions[controllerID]; ok && held.lease.Token == leaseToken {
		held.closing = true
		a.sessions[controllerID] = held
	}
}

// beginTransfer marks the session as being handed off. Periodic checkpoints and
// drain skip it so a concurrent flush cannot rewrite players.x/y back to the
// source-zone edge after the destination projection was committed.
func (a *Admission) beginTransfer(controllerID uuid.UUID, leaseToken string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	active, ok := a.sessions[controllerID]
	if !ok || active.lease.Token != leaseToken {
		return false
	}
	active.transferring = true
	active.lastTransferAt = time.Now()
	a.sessions[controllerID] = active
	return true
}

// blockTransfer rate-limits a denied border (rank/Gu gate) so a player resting
// against the edge does not trigger a repeated DB validation every tick.
func (a *Admission) blockTransfer(controllerID uuid.UUID, leaseToken string, delay time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if active, ok := a.sessions[controllerID]; ok && active.lease.Token == leaseToken {
		active.transferBlockedAt = time.Now().Add(delay)
		a.sessions[controllerID] = active
	}
}

// finishFailedTransfer clears the transferring flag and, if the socket closed
// while the handoff was in flight, releases the lease and removes the session.
func (a *Admission) finishFailedTransfer(active session, controllerID uuid.UUID) {
	a.mu.Lock()
	held, ok := a.sessions[controllerID]
	if !ok || held.lease.Token != active.lease.Token {
		a.mu.Unlock()
		return
	}
	held.transferring = false
	held.lastTransferAt = time.Now()
	closing := held.closing
	a.sessions[controllerID] = held
	a.mu.Unlock()
	if !closing {
		return
	}
	if err := a.flushState(held); err != nil {
		// Keep the session owned so the periodic checkpoint can retry; a
		// database outage must not silently drop position/mana/hunger.
		logger.S.Errorf("Realtime transfer cleanup checkpoint failed for %s: %v", active.playerID, err)
		return
	}
	a.manager.LeaveCharacter(held.lease)
	a.mu.Lock()
	delete(a.sessions, controllerID)
	a.mu.Unlock()
}

func (a *Admission) markTransferAttempt(controllerID uuid.UUID, leaseToken string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if held, ok := a.sessions[controllerID]; ok && held.lease.Token == leaseToken {
		held.lastTransferAt = time.Now()
		a.sessions[controllerID] = held
	}
}

// Cast forwards a validated cast intent. Skill mode, range, resources and
// cooldowns are revalidated by the zone tick; the client never sends damage.
func (a *Admission) Cast(playerID, controllerID uuid.UUID, cast protocol.Cast) error {
	mode := combat.Mode(cast.Mode)
	if mode == "" {
		mode = combat.ModeTarget
	}
	return a.submitCommand(playerID, controllerID, sim.InputCommand{
		Seq: cast.Seq,
		Cast: &sim.CastCommand{
			Seq: cast.Seq, Slot: cast.Slot, Mode: mode, TargetID: sim.EntityID(cast.TargetID),
			Aim: sim.Vec2{X: cast.AimX, Y: cast.AimY}, Point: sim.Vec2{X: cast.X, Y: cast.Y},
		},
	})
}

// BasicAttack forwards a melee intent; the zone owns its range,
// cooldown and damage resolution.
func (a *Admission) BasicAttack(playerID, controllerID uuid.UUID, attack protocol.Attack) error {
	return a.submitCommand(playerID, controllerID, sim.InputCommand{
		Seq:    attack.Seq,
		Attack: &sim.AttackCommand{Seq: attack.Seq, TargetID: sim.EntityID(attack.TargetID), Aim: sim.Vec2{X: attack.AimX, Y: attack.AimY}},
	})
}

func (a *Admission) submitCommand(playerID, controllerID uuid.UUID, command sim.InputCommand) error {
	a.mu.Lock()
	active, exists := a.sessions[controllerID]
	a.mu.Unlock()
	if !exists || active.playerID != playerID {
		return admissionError(409, "realtime_session_not_joined", "This connection has no active realtime session")
	}
	command.EntityID = sim.EntityID(active.lease.CharacterID)
	var err error
	switch {
	case command.Cast != nil:
		err = a.manager.SubmitCast(active.lease, command)
	case command.Attack != nil:
		err = a.manager.SubmitBasicAttack(active.lease, command)
	case command.Target != nil:
		err = a.manager.SubmitMoveTarget(active.lease, command)
	default:
		err = a.manager.SubmitInput(active.lease, command)
	}
	if err != nil {
		switch {
		case errors.Is(err, sim.ErrQueueFull):
			return admissionError(429, "realtime_input_queue_full", "Realtime input queue is full")
		case errors.Is(err, sim.ErrInvalidInput):
			return admissionError(400, "invalid_realtime_input", "Realtime input is invalid")
		default:
			return mapManagerError(err)
		}
	}
	return nil
}

func moveVector(move *protocol.Vector) sim.Vec2 {
	if move == nil {
		return sim.Vec2{}
	}
	return sim.Vec2{X: move.X, Y: move.Y}
}

func (a *Admission) SubscribeSnapshots(playerID, controllerID uuid.UUID) (<-chan sim.Snapshot, func(), error) {
	a.mu.Lock()
	active, exists := a.sessions[controllerID]
	a.mu.Unlock()
	if !exists || active.playerID != playerID {
		return nil, nil, admissionError(409, "realtime_session_not_joined", "This connection has no active realtime session")
	}
	feed, unsubscribe, err := a.manager.SubscribeSnapshots(active.lease)
	if err != nil {
		return nil, nil, mapManagerError(err)
	}
	return feed, unsubscribe, nil
}

func (a *Admission) Disconnect(playerID, controllerID uuid.UUID) {
	a.mu.Lock()
	active, exists := a.sessions[controllerID]
	a.mu.Unlock()
	if !exists || active.playerID != playerID {
		return
	}
	if active.transferring {
		// The edge-transfer goroutine owns the lease now. Mark closing so a
		// failed handoff cleans the session up instead of leaking it.
		a.mu.Lock()
		if held, ok := a.sessions[controllerID]; ok && held.lease.Token == active.lease.Token {
			held.closing = true
			a.sessions[controllerID] = held
		}
		a.mu.Unlock()
		return
	}
	if err := a.flushState(active); err != nil {
		logger.S.Errorf("Realtime disconnect checkpoint failed for %s: %v", active.playerID, err)
		a.mu.Lock()
		if held, ok := a.sessions[controllerID]; ok && held.lease.Token == active.lease.Token {
			held.closing = true
			a.sessions[controllerID] = held
		}
		a.mu.Unlock()
		return
	}
	a.manager.LeaveCharacter(active.lease)
	a.mu.Lock()
	delete(a.sessions, controllerID)
	a.mu.Unlock()
}

func (a *Admission) flushState(active session) error {
	publishInventory, publishPlayer, err := a.flushStateLocked(active)
	if err != nil {
		return err
	}
	if active.regionID == "aperture_tribulation" {
		a.mu.Lock()
		sink := a.apertureStateSink
		a.mu.Unlock()
		if sink != nil {
			if snapshot, snapshotErr := a.manager.SnapshotForLease(active.lease); snapshotErr == nil {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				if sinkErr := sink(ctx, active.playerID, snapshot); sinkErr != nil {
					logger.S.Errorf("Realtime tribulation state sync failed for %s: %v", active.playerID, sinkErr)
				}
				cancel()
			}
		}
	}
	if publishInventory || publishPlayer {
		// Do not hold the per-session mana settlement lock while loading and
		// broadcasting the authoritative player snapshot.
		domains := make([]handlers.PlayerEventDomain, 0, 2)
		if publishInventory {
			domains = append(domains, handlers.GuInventoryDomain)
		}
		if publishPlayer {
			domains = append(domains, handlers.PlayerDomain)
		}
		handlers.PublishPlayerRealtimeEvent(active.playerID, handlers.PlayerEventUpdated, domains...)
	}
	return nil
}

// flushStateLocked serializes the mana/hunger settlement for one session and
// reports whether the committed Gu usage changed the inventory the client
// should refetch and whether Chân Ý changed the player snapshot.
func (a *Admission) flushStateLocked(active session) (bool, bool, error) {
	if active.persistCombatState {
		if active.mana != nil {
			active.mana.mu.Lock()
			defer active.mana.mu.Unlock()
		}
		snapshot, err := a.manager.SnapshotForLease(active.lease)
		if err != nil {
			return false, false, err
		}
		for _, entity := range snapshot.Entities {
			if entity.CharacterID != active.lease.CharacterID {
				continue
			}
			a.mu.Lock()
			sink := a.combatStateSink
			a.mu.Unlock()
			if sink == nil {
				return false, false, errors.New("realtime combat instance state sink is not configured")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			err := sink(ctx, active.playerID, active.lease.ZoneKey, entity)
			cancel()
			return false, false, err
		}
		return false, false, fmt.Errorf("realtime combat instance participant %s missing from checkpoint snapshot", active.playerID)
	}
	if !active.persistState {
		return false, false, nil
	}
	if a.states == nil {
		return false, false, nil
	}
	if active.mana != nil {
		active.mana.mu.Lock()
		defer active.mana.mu.Unlock()
	}
	snapshot, err := a.manager.SnapshotForLease(active.lease)
	if err != nil {
		return false, false, err
	}
	for _, entity := range snapshot.Entities {
		if entity.CharacterID != active.lease.CharacterID {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		legacyX, legacyY, legacyOK := a.manager.LocalToLegacy(active.lease.ZoneKey, entity.Transform.Position)
		checkpoint := persistence.State{
			PlayerID: active.playerID, RegionID: active.regionID, ZoneID: active.lease.ZoneKey.MapID,
			InstanceID: active.lease.ZoneKey.InstanceID,
			X:          entity.Transform.Position.X, Y: entity.Transform.Position.Y,
			HP: entity.Vitals.HP, Shield: entity.Vitals.Shield,
			SafeX: active.safeX, SafeY: active.safeY,
			MapVersion: active.mapVersion, ContentVersion: active.contentVersion,
			LegacyX: int(math.Round(legacyX)), LegacyY: int(math.Round(legacyY)),
			SyncLegacy:          legacyOK,
			HasLocationRevision: active.hasLocationRevision,
			Combat:              combatCheckpoint(entity.Combat, snapshot.Tick, entity.Vitals.HP),
		}
		if active.locationRevision != nil {
			checkpoint.LocationRevision = active.locationRevision.Load()
		}
		delta := 0.0
		guUses := make(map[string]uint64)
		killerMoveUses := make(map[string]uint64)
		skillUses := make(map[persistence.SkillUseKey]uint64)
		basicAttackUses := uint64(0)
		if active.mana != nil {
			delta = entity.Combat.Mana - active.mana.last
			for slot := 1; slot <= 12; slot++ {
				uses := entity.Combat.GuUses[slot]
				if uses < active.mana.lastUses[slot] {
					return false, false, fmt.Errorf("realtime Gu usage counter went backwards for slot %d", slot)
				}
				if spent := uses - active.mana.lastUses[slot]; spent > 0 {
					skill := entity.Combat.Skills[slot]
					if skill.GuInstanceID != "" {
						guUses[skill.GuInstanceID] += spent
					} else if len(skill.GuComponentIDs) > 0 {
						for _, instanceID := range skill.GuComponentIDs {
							guUses[instanceID] += spent
						}
					} else {
						return false, false, fmt.Errorf("realtime Gu usage has no instance for slot %d", slot)
					}
				}
				masteryUses := entity.Combat.GuMasteryUses[slot]
				if masteryUses < active.mana.lastMasteryUses[slot] {
					return false, false, fmt.Errorf("realtime Gu mastery counter went backwards for slot %d", slot)
				}
				if spentMastery := masteryUses - active.mana.lastMasteryUses[slot]; spentMastery > 0 {
					skill := entity.Combat.Skills[slot]
					// Only regular, accepted Gu casts train Chân Ý. Killer moves
					// settle their core path below; Trận Kích is excluded there.
					if skill.GuInstanceID != "" && skill.KillerMoveID == "" {
						skillUses[persistence.SkillUseKey{Path: skill.Path, Rank: skill.Rank}] += spentMastery
					}
				}
				moveUses := entity.Combat.KillerMoveUses[slot]
				if moveUses < active.mana.lastKillerMoveUses[slot] {
					return false, false, fmt.Errorf("realtime Sát Chiêu usage counter went backwards for slot %d", slot)
				}
				if spentMoveUses := moveUses - active.mana.lastKillerMoveUses[slot]; spentMoveUses > 0 {
					skill := entity.Combat.Skills[slot]
					if skill.KillerMoveID == "" {
						return false, false, fmt.Errorf("realtime Sát Chiêu usage has no move ID for slot %d", slot)
					}
					killerMoveUses[skill.KillerMoveID] += spentMoveUses
				}
			}
			if entity.Combat.BasicAttackUses < active.mana.lastBasicUses {
				return false, false, fmt.Errorf("realtime basic attack usage counter went backwards")
			}
			basicAttackUses = entity.Combat.BasicAttackUses - active.mana.lastBasicUses
		}
		var revision int64
		manaApplied := false
		if atomicStore, ok := a.states.(interface {
			SaveAndApplyCombatDeltaRevision(context.Context, persistence.State, float64, persistence.CombatUsage) (int64, error)
		}); ok {
			var saveErr error
			revision, saveErr = atomicStore.SaveAndApplyCombatDeltaRevision(ctx, checkpoint, delta, persistence.CombatUsage{
				Gu:           guUses,
				KillerMoves:  killerMoveUses,
				Skills:       skillUses,
				BasicAttacks: basicAttackUses,
			})
			if saveErr != nil {
				return false, false, saveErr
			}
			active.mana.last = entity.Combat.Mana
			active.mana.lastUses = entity.Combat.GuUses
			active.mana.lastMasteryUses = entity.Combat.GuMasteryUses
			active.mana.lastKillerMoveUses = entity.Combat.KillerMoveUses
			active.mana.lastBasicUses = entity.Combat.BasicAttackUses
			manaApplied = true
		} else if len(guUses) > 0 || len(killerMoveUses) > 0 || len(skillUses) > 0 || basicAttackUses > 0 {
			return false, false, fmt.Errorf("realtime state store cannot atomically settle combat usage")
		} else if atomicStore, ok := a.states.(interface {
			SaveAndApplyManaDeltaRevision(context.Context, persistence.State, float64) (int64, error)
		}); ok {
			var saveErr error
			revision, saveErr = atomicStore.SaveAndApplyManaDeltaRevision(ctx, checkpoint, delta)
			if saveErr != nil {
				return false, false, saveErr
			}
			active.mana.last = entity.Combat.Mana
			manaApplied = true
		} else if atomicStore, ok := a.states.(interface {
			SaveAndApplyManaDelta(context.Context, persistence.State, float64) error
		}); ok {
			if err := atomicStore.SaveAndApplyManaDelta(ctx, checkpoint, delta); err != nil {
				return false, false, err
			}
			active.mana.last = entity.Combat.Mana
			manaApplied = true
		} else if revisionStore, ok := a.states.(interface {
			SaveRevision(context.Context, persistence.State) (int64, error)
		}); ok {
			var saveErr error
			revision, saveErr = revisionStore.SaveRevision(ctx, checkpoint)
			if saveErr != nil {
				return false, false, saveErr
			}
		} else if err := a.states.Save(ctx, checkpoint); err != nil {
			return false, false, err
		}
		if active.locationRevision != nil && revision > 0 {
			active.locationRevision.Store(revision)
			active.hasLocationRevision = true
		}
		if manaStore, ok := a.states.(interface {
			ApplyManaDelta(context.Context, uuid.UUID, float64) error
		}); ok && active.mana != nil && !manaApplied {
			if err := manaStore.ApplyManaDelta(ctx, active.playerID, delta); err != nil {
				return false, false, err
			}
			active.mana.last = entity.Combat.Mana
		}
		// The zone decrements the local hunger counters while casting; the caller
		// asks the client to refetch the committed Gu inventory so the UI cannot
		// show a stale hunger value after a checkpoint, disconnect or transfer.
		masteryChanged := len(skillUses) > 0 || len(killerMoveUses) > 0 || basicAttackUses > 0
		return len(guUses) > 0, masteryChanged, nil
	}
	return false, false, fmt.Errorf("realtime character %s missing from checkpoint snapshot", active.playerID)
}

func (a *Admission) joinResponse(active session, controllerID uuid.UUID) (protocol.Welcome, error) {
	snapshot, err := a.manager.SnapshotForLease(active.lease)
	if err != nil {
		return protocol.Welcome{}, admissionError(503, "realtime_zone_unavailable", "Realtime zone is unavailable")
	}
	protocolSnapshot := protocol.Snapshot{
		Tick: snapshot.Tick, BaselineID: 1, BaseTick: snapshot.Tick, IsBaseline: true,
		Entities:    make([]protocol.EntityState, 0, len(snapshot.Entities)),
		Projectiles: make([]protocol.ProjectileState, 0, len(snapshot.Projectiles)),
	}
	for _, projectile := range snapshot.Projectiles {
		protocolSnapshot.Projectiles = append(protocolSnapshot.Projectiles, protocol.ProjectileState{
			ID: projectile.ID, SourceID: projectile.SourceID, SkillID: projectile.SkillID,
			X: projectile.X, Y: projectile.Y, Radius: projectile.Radius,
		})
	}
	var entityRevision uint64
	for _, entity := range snapshot.Entities {
		state := protocol.EntityState{
			EntityID: string(entity.ID), Revision: entity.Revision, Kind: string(entity.Kind), PartyID: entity.PartyID,
			PvPEnabled: entity.PvPEnabled,
			X:          entity.Transform.Position.X, Y: entity.Transform.Position.Y, HP: entity.Vitals.HP, MaxHP: entity.Vitals.MaxHP, Shield: entity.Vitals.Shield,
		}
		if entity.Kind == sim.EntityMonster {
			state.TemplateID = entity.AI.TemplateID
			state.Name = entity.AI.Name
			state.AIState = string(entity.AI.State)
			state.AggroTarget = string(entity.AI.Threat)
		} else if entity.Name != "" {
			state.Name = entity.Name
		}
		if entity.Combat.MaxMana > 0 {
			state.Mana = entity.Combat.Mana
			state.MaxMana = entity.Combat.MaxMana
			state.FormActive = entity.Combat.FormActive
			state.Cooldowns = make([]uint64, len(entity.Combat.Cooldowns))
			copy(state.Cooldowns, entity.Combat.Cooldowns[:])
			if entity.Combat.Pending != nil {
				state.CastSkillID = entity.Combat.Pending.Skill.ID
				state.CastEndTick = entity.Combat.Pending.ResolveTick
				state.CastWindup = entity.Combat.Pending.Skill.WindupTicks
			}
		}
		if len(entity.Combat.Statuses) > 0 {
			state.Statuses = make([]string, 0, len(entity.Combat.Statuses))
			for _, st := range entity.Combat.Statuses {
				state.Statuses = append(state.Statuses, st.ID)
				if st.ID == "STEALTH" || st.ID == "INVISIBILITY" {
					state.Stealthed = true
				}
				if st.ID == "FLYING" || st.ID == "FLIGHT" {
					state.Flying = true
				}
			}
		}
		protocolSnapshot.Entities = append(protocolSnapshot.Entities, state)
		if entity.CharacterID == active.lease.CharacterID {
			entityRevision = entity.Revision
			protocolSnapshot.AckSeq = entity.AckSeq
		}
	}
	snapshotJSON, err := json.Marshal(protocolSnapshot)
	if err != nil {
		return protocol.Welcome{}, admissionError(500, "realtime_snapshot_failed", "Could not encode realtime zone snapshot")
	}
	return protocol.Welcome{
		SessionID: controllerID.String(), ZoneID: active.lease.ZoneKey.MapID + "#" + active.lease.ZoneKey.InstanceID,
		MapVersion: active.mapVersion, ContentVersion: active.contentVersion,
		TickRate: sim.TickRate, MovementSpeed: active.movementSpeed, ServerTick: snapshot.Tick, EntityID: string(active.lease.CharacterID),
		EntityRevision: entityRevision, BaselineID: 1, MapHash: a.manager.MapHash(active.lease.ZoneKey),
		Skills: skillInfos(active.skills), LoadoutRevision: active.loadoutRevision, Snapshot: snapshotJSON,
	}, nil
}

func skillInfos(loadout []combat.Skill) []protocol.SkillInfo {
	infos := make([]protocol.SkillInfo, 0, len(loadout))
	for _, skill := range loadout {
		infos = append(infos, protocol.SkillInfo{
			Slot: uint8(skill.Slot), ID: skill.ID, Name: skill.Name, Mode: string(skill.Mode),
			Shape: string(skill.Shape), AngleDeg: skill.AngleDeg, LengthUnits: skill.LengthUnits,
			WidthUnits: skill.WidthUnits, InnerRadius: skill.InnerRadius,
			ManaCost: skill.ManaCost, Range: skill.Range, RadiusUnits: skill.Radius, CooldownTicks: skill.CooldownTicks,
			WindupTicks: skill.WindupTicks, IsToggle: skill.IsToggle,
			FormActivation: skill.FormActivation, FormAbility: skill.FormAbility,
		})
	}
	return infos
}

func admissionError(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func mapManagerError(err error) error {
	switch {
	case errors.Is(err, realtimeworld.ErrCharacterOwned):
		return admissionError(409, "realtime_character_already_owned", "Character already has an active realtime controller")
	case errors.Is(err, realtimeworld.ErrInvalidLease):
		return admissionError(409, "realtime_lease_stale", "Realtime session lease is no longer current")
	case errors.Is(err, realtimeworld.ErrZoneFull):
		return admissionError(429, "realtime_zone_full", "Realtime zone is at capacity")
	case errors.Is(err, realtimeworld.ErrSnapshotSubscribersFull):
		return admissionError(429, "realtime_snapshot_capacity", "Realtime snapshot subscriber capacity is full")
	case errors.Is(err, realtimeworld.ErrManagerNotRunning), errors.Is(err, realtimeworld.ErrUnknownZone):
		return admissionError(503, "realtime_zone_unavailable", "Realtime zone is unavailable")
	default:
		return fmt.Errorf("realtime admission: %w", err)
	}
}
