package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	joinLimitPerMinute    = 6
	maxTrackedJoinPlayers = 10_000
	playerSpeed           = 4.0
	playerRadius          = 0.4
)

type JoinStateLoader func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error)

// WorldStateStore is the durable checkpoint adapter used on join/leave and by
// the periodic checkpoint worker.
type WorldStateStore interface {
	Load(context.Context, uuid.UUID) (persistence.State, bool, error)
	Save(context.Context, persistence.State) error
}

type Admission struct {
	config   config.Config
	manager  *realtimeworld.Manager
	load     JoinStateLoader
	compiled *realtimeeffects.CompiledCatalog
	states   WorldStateStore
	paused   atomic.Bool

	mu           sync.Mutex
	sessions     map[uuid.UUID]session
	joinAttempts map[uuid.UUID][]time.Time
}

type session struct {
	playerID        uuid.UUID
	lease           realtimeworld.OwnershipLease
	mapVersion      string
	contentVersion  string
	skills          []combat.Skill
	loadoutRevision string
	mana            *manaSettlement
	closing         bool
	safeX           float64
	safeY           float64
}

type manaSettlement struct {
	mu   sync.Mutex
	last float64
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
		if err := a.flushState(current); err != nil {
			logger.S.Errorf("Realtime periodic checkpoint failed for %s: %v", current.playerID, err)
			continue
		}
		if current.closing {
			a.manager.LeaveCharacter(current.lease)
			a.mu.Lock()
			delete(a.sessions, controllerID)
			a.mu.Unlock()
		}
	}
}

func (a *Admission) Join(ctx context.Context, playerID, controllerID uuid.UUID, request protocol.JoinRequest) (any, error) {
	if a.manager == nil || !a.config.Enabled || !a.config.PhaserClientEnabled {
		return nil, admissionError(503, "realtime_world_unavailable", "Realtime world admission is unavailable")
	}
	if a.paused.Load() {
		return nil, admissionError(503, "realtime_admission_paused", "Realtime world admission is paused for cutover/rollback")
	}
	if !a.config.AllowsJoin(playerID.String(), request.ZoneID) {
		return nil, admissionError(403, "realtime_world_forbidden", "Player or map is not allowlisted")
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
	key, err := a.manager.ZoneKeyForMapWithCapacity(request.ZoneID)
	if err != nil {
		if errors.Is(err, realtimeworld.ErrZoneFull) {
			return nil, admissionError(429, "realtime_zone_full", "Realtime zone instances are at capacity")
		}
		return nil, admissionError(403, "realtime_map_unavailable", "Realtime zone is not configured for this map")
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

	joinCtx, cancel := context.WithTimeout(ctx, joinReadTimeout)
	defer cancel()
	state, err := a.load(joinCtx, playerID)
	if err != nil {
		return nil, admissionError(503, "realtime_character_unavailable", "Could not load authoritative character state")
	}
	if state.RegionID == "" || state.MapID != request.ZoneID {
		return nil, admissionError(409, "realtime_map_mismatch", "Requested map does not match the character's current region")
	}
	if state.MapInstanceID != "" {
		return nil, admissionError(409, "realtime_instance_conflict", "Character is already assigned to another map instance")
	}
	if state.HasActiveCombat || state.NeedsOnboarding {
		return nil, admissionError(409, "realtime_character_not_ready", "Character cannot join the realtime world in its current state")
	}
	if math.IsNaN(state.HP) || math.IsInf(state.HP, 0) || math.IsNaN(state.BaseHP) || math.IsInf(state.BaseHP, 0) || state.HP <= 0 || state.BaseHP <= 0 {
		return nil, admissionError(409, "realtime_character_invalid", "Character vitals are invalid for admission")
	}
	baseX := float64(state.X)
	baseY := float64(state.Y)
	position := a.manager.LegacyPosition(key, baseX, baseY)
	baseHP := state.HP
	baseShield := 0.0
	a.mu.Lock()
	compiled := a.compiled
	stateStore := a.states
	a.mu.Unlock()
	if stateStore != nil {
		durable, ok, loadErr := stateStore.Load(ctx, playerID)
		if loadErr != nil {
			return nil, admissionError(503, "realtime_checkpoint_unavailable", "Could not load the realtime world checkpoint")
		}
		if ok && durable.ZoneID == request.ZoneID && durable.MapVersion == request.MapVersion && durable.ContentVersion == request.ContentVersion &&
			a.manager.ContainsPosition(key, sim.Vec2{X: durable.X, Y: durable.Y}) {
			baseX, baseY = durable.X, durable.Y
			position = sim.Vec2{X: baseX, Y: baseY}
			if durable.HP > 0 {
				baseHP = durable.HP
			}
			baseShield = durable.Shield
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
			skill.ManaCost = entry.ManaCost
			skill.Damage *= entry.DamageScale
			skill.BacklashPct = entry.BacklashPct
			skill.Path = entry.Path
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
	entity := sim.Entity{
		ID: sim.EntityID(playerID.String()), CharacterID: sim.CharacterID(playerID.String()), Kind: sim.EntityPlayer,
		Name:           state.Username,
		Transform:      sim.Transform{Position: position},
		Vitals:         sim.Vitals{HP: baseHP, MaxHP: maxHP, Shield: baseShield},
		Defense:        state.Defense,
		DaoProfile:     state.DaoProfile,
		PathResistance: state.PathResistance,
		Speed:          playerSpeed,
		Radius:         playerRadius,
		Faction:        "player",
		PartyID:        state.PartyID,
		Combat:         playerCombat,
	}
	lease, err = a.manager.AttachCharacterEntity(lease, entity)
	if err != nil {
		return nil, mapManagerError(err)
	}
	attached = true
	active := session{
		playerID: playerID, lease: lease, mapVersion: request.MapVersion, contentVersion: request.ContentVersion,
		skills: skills, loadoutRevision: loadoutRevision,
		mana:  &manaSettlement{last: playerCombat.Mana},
		safeX: safeSpawn.X, safeY: safeSpawn.Y,
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
	delete(a.sessions, controllerID)
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
		return admissionError(409, "realtime_session_not_joined", "This connection has no active realtime session")
	}
	if err := a.flushState(active); err != nil {
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

// BasicAttack forwards a target-only melee intent; the zone owns its range,
// cooldown and damage resolution.
func (a *Admission) BasicAttack(playerID, controllerID uuid.UUID, attack protocol.Attack) error {
	return a.submitCommand(playerID, controllerID, sim.InputCommand{
		Seq:    attack.Seq,
		Attack: &sim.AttackCommand{Seq: attack.Seq, TargetID: sim.EntityID(attack.TargetID)},
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
	if a.states == nil {
		return nil
	}
	if active.mana != nil {
		active.mana.mu.Lock()
		defer active.mana.mu.Unlock()
	}
	snapshot, err := a.manager.SnapshotForLease(active.lease)
	if err != nil {
		return err
	}
	for _, entity := range snapshot.Entities {
		if entity.CharacterID != active.lease.CharacterID {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		checkpoint := persistence.State{
			PlayerID: active.playerID, ZoneID: active.lease.ZoneKey.MapID,
			InstanceID: active.lease.ZoneKey.InstanceID,
			X:          entity.Transform.Position.X, Y: entity.Transform.Position.Y,
			HP: entity.Vitals.HP, Shield: entity.Vitals.Shield,
			SafeX: active.safeX, SafeY: active.safeY,
			MapVersion: active.mapVersion, ContentVersion: active.contentVersion,
		}
		delta := 0.0
		if active.mana != nil {
			delta = entity.Combat.Mana - active.mana.last
		}
		if atomicStore, ok := a.states.(interface {
			SaveAndApplyManaDelta(context.Context, persistence.State, float64) error
		}); ok {
			if err := atomicStore.SaveAndApplyManaDelta(ctx, checkpoint, delta); err != nil {
				return err
			}
			active.mana.last = entity.Combat.Mana
			return nil
		}
		if err := a.states.Save(ctx, checkpoint); err != nil {
			return err
		}
		if manaStore, ok := a.states.(interface {
			ApplyManaDelta(context.Context, uuid.UUID, float64) error
		}); ok && active.mana != nil {
			if err := manaStore.ApplyManaDelta(ctx, active.playerID, delta); err != nil {
				return err
			}
			active.mana.last = entity.Combat.Mana
		}
		return nil
	}
	return fmt.Errorf("realtime character %s missing from checkpoint snapshot", active.playerID)
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
			X: entity.Transform.Position.X, Y: entity.Transform.Position.Y, HP: entity.Vitals.HP, MaxHP: entity.Vitals.MaxHP, Shield: entity.Vitals.Shield,
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
		TickRate: sim.TickRate, ServerTick: snapshot.Tick, EntityID: string(active.lease.CharacterID),
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
