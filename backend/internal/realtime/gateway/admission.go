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
	joinLimitPerMinute    = 15
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
	playerID            uuid.UUID
	regionID            string
	lease               realtimeworld.OwnershipLease
	mapVersion          string
	contentVersion      string
	skills              []combat.Skill
	loadoutRevision     string
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
}

// edgeTransferCooldown prevents border ping-pong when a player holds movement
// into the wall while the client reconnects into the neighbouring zone.
const edgeTransferCooldown = 2 * time.Second

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
			delete(a.sessions, controllerID)
			a.mu.Unlock()
		}
	}
}

func (a *Admission) Join(ctx context.Context, playerID, controllerID uuid.UUID, request protocol.JoinRequest) (any, error) {
	if a.manager == nil {
		reason := handlers.RealtimeRuntimeError()
		if reason == "" {
			reason = "realtime manager was not installed (check REALTIME_WORLD_ENABLED, MMO_ZONE_IDS and startup logs)"
		}
		return nil, admissionError(503, "realtime_manager_unavailable", "Realtime world manager is unavailable: "+reason)
	}
	if !a.config.Enabled {
		return nil, admissionError(503, "realtime_disabled", "Realtime world is disabled: REALTIME_WORLD_ENABLED=false")
	}
	if !a.config.PhaserClientEnabled {
		return nil, admissionError(503, "realtime_client_disabled", "Realtime world is disabled: PHASER_WORLD_CLIENT_ENABLED=false")
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
	a.mu.Lock()
	compiled := a.compiled
	stateStore := a.states
	a.mu.Unlock()
	baseX := float64(state.X)
	baseY := float64(state.Y)
	position := a.manager.LegacyPosition(key, baseX, baseY)
	locationRevision := &atomic.Int64{}
	locationRevision.Store(state.WorldLocationRevision)
	hasLocationRevision := state.HasWorldLocationRevision
	if stateStore == nil && state.HasWorldLocation && state.MapID == request.ZoneID {
		baseX, baseY = state.WorldX, state.WorldY
		position = sim.Vec2{X: baseX, Y: baseY}
	}
	baseHP := state.HP
	baseShield := 0.0
	if stateStore != nil {
		durable, ok, loadErr := stateStore.Load(ctx, playerID)
		if loadErr != nil {
			return nil, admissionError(503, "realtime_checkpoint_unavailable", "Could not load the realtime world checkpoint")
		}
		if ok && durable.ZoneID == request.ZoneID &&
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
			guRatio := 1.0
			if skill.GuPowerRatio > 0 {
				guRatio = skill.GuPowerRatio
			}
			baseDmg := skill.Damage * guRatio
			if skill.PlayerAtkRatio > 0 && state.BaseAttack > 0 {
				baseDmg += state.BaseAttack * skill.PlayerAtkRatio
			}
			skill.Damage = baseDmg * entry.DamageScale
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
	zoneMap, _ := a.manager.ZoneMapFor(key)
	active := session{
		playerID: playerID, regionID: state.RegionID, lease: lease, mapVersion: request.MapVersion, contentVersion: request.ContentVersion,
		skills: skills, loadoutRevision: loadoutRevision,
		mana:             &manaSettlement{last: playerCombat.Mana},
		locationRevision: locationRevision, hasLocationRevision: hasLocationRevision,
		safeX: safeSpawn.X, safeY: safeSpawn.Y,
		mapWidth: float64(zoneMap.Width), mapHeight: float64(zoneMap.Height),
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
		logger.S.Errorf("Realtime transfer cleanup checkpoint failed for %s: %v", active.playerID, err)
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
		}
		if active.locationRevision != nil {
			checkpoint.LocationRevision = active.locationRevision.Load()
		}
		delta := 0.0
		if active.mana != nil {
			delta = entity.Combat.Mana - active.mana.last
		}
		var revision int64
		manaApplied := false
		if atomicStore, ok := a.states.(interface {
			SaveAndApplyManaDeltaRevision(context.Context, persistence.State, float64) (int64, error)
		}); ok {
			var saveErr error
			revision, saveErr = atomicStore.SaveAndApplyManaDeltaRevision(ctx, checkpoint, delta)
			if saveErr != nil {
				return saveErr
			}
			active.mana.last = entity.Combat.Mana
			manaApplied = true
		} else if atomicStore, ok := a.states.(interface {
			SaveAndApplyManaDelta(context.Context, persistence.State, float64) error
		}); ok {
			if err := atomicStore.SaveAndApplyManaDelta(ctx, checkpoint, delta); err != nil {
				return err
			}
			active.mana.last = entity.Combat.Mana
			manaApplied = true
		} else if revisionStore, ok := a.states.(interface {
			SaveRevision(context.Context, persistence.State) (int64, error)
		}); ok {
			var saveErr error
			revision, saveErr = revisionStore.SaveRevision(ctx, checkpoint)
			if saveErr != nil {
				return saveErr
			}
		} else if err := a.states.Save(ctx, checkpoint); err != nil {
			return err
		}
		if active.locationRevision != nil && revision > 0 {
			active.locationRevision.Store(revision)
			active.hasLocationRevision = true
		}
		if manaStore, ok := a.states.(interface {
			ApplyManaDelta(context.Context, uuid.UUID, float64) error
		}); ok && active.mana != nil && !manaApplied {
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
