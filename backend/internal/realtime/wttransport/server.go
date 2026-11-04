// Package wttransport serves the authoritative realtime world over
// WebTransport. It depends only on the admission surface and the UI event bus,
// never on a network connection type.
package wttransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/quic-go/webtransport-go"
	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	v2 "github.com/vippro67/GuWorld/backend/internal/realtime/protocol/v2"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Admission is the subset of gateway.Admission the transport needs.
type Admission interface {
	JoinWithTab(ctx context.Context, playerID, controllerID uuid.UUID, tabID string, request protocol.JoinRequest) (any, error)
	Input(playerID, controllerID uuid.UUID, input protocol.Input) error
	MoveTarget(playerID, controllerID uuid.UUID, target protocol.MoveTarget) error
	Cast(playerID, controllerID uuid.UUID, cast protocol.Cast) error
	BasicAttack(playerID, controllerID uuid.UUID, attack protocol.Attack) error
	Capture(playerID, controllerID uuid.UUID, capture protocol.CaptureIntent) error
	CompanionStance(playerID, controllerID uuid.UUID, stance protocol.CompanionStanceIntent) error
	SubscribeSnapshots(playerID, controllerID uuid.UUID) (<-chan sim.Snapshot, func(), error)
	Leave(playerID, controllerID uuid.UUID) error
	Disconnect(playerID, controllerID uuid.UUID)
}

const (
	joinFrameTimeout        = 10 * time.Second
	baselineTicks           = 40
	CloseCodeServerDraining = 4010
)

type edgeTransferAdmission interface {
	TransferAtEdge(playerID, controllerID uuid.UUID, position sim.Vec2) (protocol.Transfer, bool, error)
}

// Server upgrades WebTransport requests and runs one world/UI session each.
type Server struct {
	admission Admission
	wt        *webtransport.Server
	ui        UIOptions
	mu        sync.Mutex
	sessions  map[*webtransport.Session]struct{}
	draining  atomic.Bool
}

func New(admission Admission, wt *webtransport.Server) *Server {
	return &Server{
		admission: admission,
		wt:        wt,
		sessions:  make(map[*webtransport.Session]struct{}),
	}
}

// Handle must run after authentication; playerID/tabID come from the
// authenticated request, never from the join payload.
func (s *Server) Handle(w http.ResponseWriter, r *http.Request, playerID uuid.UUID, tabID string) {
	if s.draining.Load() {
		http.Error(w, "server draining", http.StatusServiceUnavailable)
		return
	}
	session, err := s.wt.Upgrade(w, r)
	if err != nil {
		logger.S.Warnf("WebTransport upgrade failed for %s: %v", playerID, err)
		http.Error(w, "webtransport upgrade failed", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if s.sessions == nil {
		s.sessions = make(map[*webtransport.Session]struct{})
	}
	s.sessions[session] = struct{}{}
	s.mu.Unlock()
	controllerID := uuid.New()
	go s.serve(session, playerID, controllerID, tabID)
}

// Drain marks the server as draining (rejecting new sessions with 503) and
// sends application close code 4010 ("server_draining") to all live sessions.
func (s *Server) Drain() int {
	s.draining.Store(true)
	s.mu.Lock()
	active := make([]*webtransport.Session, 0, len(s.sessions))
	for sess := range s.sessions {
		active = append(active, sess)
	}
	s.mu.Unlock()
	for _, sess := range active {
		_ = sess.CloseWithError(CloseCodeServerDraining, "server_draining")
	}
	return len(active)
}

// ActiveSessions reports how many WebTransport sessions are still open.
func (s *Server) ActiveSessions() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.sessions)
}

type worldSession struct {
	server       *Server
	session      *webtransport.Session
	stream       *webtransport.Stream
	playerID     uuid.UUID
	controllerID uuid.UUID
	tabID        string

	writeMu sync.Mutex
	stateMu sync.Mutex

	registry       *netRegistry
	spawned        map[uint16]bool
	visible        map[uint16]bool
	baselineSent   map[uint16]v2.DeltaEntity
	lastSent       map[uint16]v2.DeltaEntity
	pendingRemoved map[uint16]uint8
	baselineID     uint16
	baseTick       uint32
	lastTick       atomic.Uint32

	lastSnapshot       sim.Snapshot
	lastEventID        uint64
	eventSeq           uint64
	lastTransferReject time.Time
	resyncRequested    atomic.Bool
	snapshotMu         sync.RWMutex
}

func (s *Server) serve(session *webtransport.Session, playerID, initialControllerID uuid.UUID, tabID string) {
	defer func() {
		s.mu.Lock()
		delete(s.sessions, session)
		s.mu.Unlock()
		_ = session.CloseWithError(0, "session ended")
	}()
	ctx := session.Context()

	var wg sync.WaitGroup
	var firstWorldClaimed atomic.Bool
	for {
		stream, err := session.AcceptStream(ctx)
		if err != nil {
			break
		}
		wg.Add(1)
		go func(st *webtransport.Stream) {
			defer wg.Done()
			_ = st.SetReadDeadline(time.Now().Add(joinFrameTimeout))
			var streamType [1]byte
			if _, err := io.ReadFull(st, streamType[:]); err != nil {
				_ = st.Close()
				return
			}
			switch streamType[0] {
			case v2.StreamWorld:
				controllerID := initialControllerID
				if !firstWorldClaimed.CompareAndSwap(false, true) {
					controllerID = uuid.New()
				}
				s.serveWorldStream(ctx, session, st, playerID, controllerID, tabID)
			case v2.StreamUI:
				_ = st.SetReadDeadline(time.Time{})
				s.serveUIStream(ctx, st, playerID)
			default:
				_ = st.Close()
			}
		}(stream)
	}
	wg.Wait()
}

func (s *Server) serveWorldStream(ctx context.Context, session *webtransport.Session, stream *webtransport.Stream, playerID, controllerID uuid.UUID, tabID string) {
	defer func() { _ = stream.Close() }()
	if s.admission == nil {
		return
	}
	logger.S.Debugf("WebTransport world stream accepted for %s", playerID)
	world := &worldSession{
		server: s, session: session, stream: stream,
		playerID: playerID, controllerID: controllerID, tabID: tabID,
		registry:       newNetRegistry(),
		spawned:        make(map[uint16]bool),
		visible:        make(map[uint16]bool),
		baselineSent:   make(map[uint16]v2.DeltaEntity),
		lastSent:       make(map[uint16]v2.DeltaEntity),
		pendingRemoved: make(map[uint16]uint8),
	}
	defer func() {
		s.admission.Disconnect(playerID, controllerID)
	}()

	_ = stream.SetReadDeadline(time.Now().Add(joinFrameTimeout))
	request, err := world.readJoin()
	_ = stream.SetReadDeadline(time.Time{})
	if err != nil {
		logger.S.Warnf("WebTransport join handshake failed for %s: %v", playerID, err)
		world.writeReject("invalid_join", err.Error(), false)
		return
	}
	ctxJoin, cancel := context.WithTimeout(ctx, joinFrameTimeout)
	result, err := s.admission.JoinWithTab(ctxJoin, playerID, controllerID, tabID, protocol.JoinRequest{
		ZoneID: request.ZoneID, MapVersion: request.MapVersion, ContentVersion: request.ContentVersion,
	})
	cancel()
	if err != nil {
		world.writeReject(admissionCode(err), err.Error(), admissionRetryable(err))
		return
	}
	welcome, ok := result.(protocol.Welcome)
	if !ok {
		world.writeReject("realtime_response_failed", "unexpected join response", false)
		return
	}
	if err := world.sendWelcome(welcome); err != nil {
		logger.S.Warnf("WebTransport welcome failed for %s: %v", playerID, err)
		return
	}

	feed, unsubscribe, err := s.admission.SubscribeSnapshots(playerID, controllerID)
	if err != nil {
		world.writeReject("realtime_snapshot_unavailable", err.Error(), true)
		return
	}
	defer unsubscribe()

	worldCtx, cancelWorld := context.WithCancel(ctx)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		world.datagramLoop(worldCtx)
	}()
	go func() {
		defer workers.Done()
		world.snapshotLoop(worldCtx, feed)
	}()

	// Frame loop runs on the stream goroutine.
	world.frameLoop(worldCtx)
	cancelWorld()
	workers.Wait()
}

func (w *worldSession) readJoin() (v2.JoinRequest, error) {
	handshake := make([]byte, 4)
	if _, err := io.ReadFull(w.stream, handshake); err != nil {
		return v2.JoinRequest{}, err
	}
	if string(handshake[0:3]) != v2.Magic || handshake[3] != v2.Version {
		return v2.JoinRequest{}, fmt.Errorf("unsupported world stream handshake")
	}
	op, payload, err := readFrame(w.stream)
	if err != nil {
		return v2.JoinRequest{}, err
	}
	if op != v2.OpJoin {
		return v2.JoinRequest{}, fmt.Errorf("expected join, got opcode %d", op)
	}
	return v2.DecodeJoin(payload)
}

func (w *worldSession) sendWelcome(welcome protocol.Welcome) error {
	snapshot, err := extractWelcomeSnapshot(welcome)
	if err != nil {
		return fmt.Errorf("decode welcome snapshot: %w", err)
	}
	snapshot = snapshotForObserver(snapshot, sim.CharacterID(w.playerID.String()))
	w.snapshotMu.Lock()
	w.lastSnapshot = snapshot
	w.snapshotMu.Unlock()

	w.stateMu.Lock()
	w.baselineID = uint16(welcome.BaselineID)
	if w.baselineID == 0 {
		w.baselineID = 1
	}
	full := w.buildBaselineLocked(snapshot, w.baselineID, nil)
	w.baseTick = full.Tick
	w.lastTick.Store(full.Tick)
	w.stateMu.Unlock()

	fullPayload, err := v2.EncodeSnapshotFull(full)
	if err != nil {
		return err
	}
	skills, _ := json.Marshal(welcome.Skills)
	payload, err := v2.EncodeWelcome(v2.Welcome{
		SessionID:         welcome.SessionID,
		ZoneID:            welcome.ZoneID,
		MapVersion:        welcome.MapVersion,
		ContentVersion:    welcome.ContentVersion,
		TickRate:          welcome.TickRate,
		MovementSpeedX100: speedX100(welcome.MovementSpeed),
		ServerTick:        uint32(welcome.ServerTick),
		EntityID:          welcome.EntityID,
		EntityRevision:    welcome.EntityRevision,
		BaselineID:        w.baselineID,
		MapHash:           welcome.MapHash,
		SkillsJSON:        skills,
		LoadoutRevision:   welcome.LoadoutRevision,
		Snapshot:          fullPayload,
	})
	if err != nil {
		return err
	}
	return w.writeFrame(v2.OpWelcome, payload)
}

func extractWelcomeSnapshot(welcome protocol.Welcome) (sim.Snapshot, error) {
	if raw, ok := welcome.RawSnapshot.(sim.Snapshot); ok {
		return raw, nil
	}
	if rawPtr, ok := welcome.RawSnapshot.(*sim.Snapshot); ok && rawPtr != nil {
		return *rawPtr, nil
	}
	var protoSnap protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &protoSnap); err != nil {
		return sim.Snapshot{}, err
	}
	entities := make([]sim.Entity, 0, len(protoSnap.Entities))
	for _, e := range protoSnap.Entities {
		kind := sim.EntityPlayer
		switch e.Kind {
		case "monster":
			kind = sim.EntityMonster
		case "companion":
			kind = sim.EntityCompanion
		}
		var statuses []sim.StatusEffect
		if e.Stealthed {
			statuses = append(statuses, sim.StatusEffect{ID: "STEALTH"})
		}
		if e.Flying {
			statuses = append(statuses, sim.StatusEffect{ID: "FLYING"})
		}
		ent := sim.Entity{
			ID:          sim.EntityID(e.EntityID),
			CharacterID: sim.CharacterID(e.EntityID),
			Kind:        kind,
			Name:        e.Name,
			PartyID:     e.PartyID,
			PvPEnabled:  e.PvPEnabled,
			Radius:      e.Radius,
			Revision:    e.Revision,
			AckSeq:      protoSnap.AckSeq,
			Transform:   sim.Transform{Position: sim.Vec2{X: e.X, Y: e.Y}},
			Vitals:      sim.Vitals{HP: e.HP, MaxHP: e.MaxHP, Shield: e.Shield},
			Combat: sim.CombatState{
				Mana: e.Mana, MaxMana: e.MaxMana,
				Thought: e.Thought, MaxThought: e.MaxThought,
				FormActive: e.FormActive,
				Statuses:   statuses,
			},
			AI: sim.AIState{
				TemplateID: e.TemplateID,
				Name:       e.Name,
				State:      sim.AIStateKind(e.AIState),
				Threat:     sim.EntityID(e.AggroTarget),
			},
			Companion: sim.CompanionState{
				OwnerID:   sim.EntityID(e.OwnerID),
				ActorKind: e.CompanionKind,
				Stance:    sim.CompanionStance(e.Stance),
			},
		}
		entities = append(entities, ent)
	}
	projectiles := make([]sim.ProjectileState, 0, len(protoSnap.Projectiles))
	for _, p := range protoSnap.Projectiles {
		projectiles = append(projectiles, sim.ProjectileState{
			ID: p.ID, SourceID: p.SourceID, SkillID: p.SkillID,
			ProjectileVFX: p.ProjectileVFX, X: p.X, Y: p.Y, Radius: p.Radius,
		})
	}
	return sim.Snapshot{
		Tick:        protoSnap.Tick,
		Entities:    entities,
		Projectiles: projectiles,
	}, nil
}

func (w *worldSession) buildBaselineLocked(snapshot sim.Snapshot, baselineID uint16, removed []uint16) v2.SnapshotFull {
	for _, entity := range snapshot.Entities {
		w.registry.assign(sim.EntityID(entity.ID))
	}
	current := make(map[uint16]bool, len(snapshot.Entities))
	spawns := make([]v2.Spawn, 0, len(snapshot.Entities))
	states := make([]v2.DeltaEntity, 0, len(snapshot.Entities))
	w.baselineSent = make(map[uint16]v2.DeltaEntity, len(snapshot.Entities))
	w.lastSent = make(map[uint16]v2.DeltaEntity, len(snapshot.Entities))
	ack := uint64(0)
	selfID := w.playerID.String()
	for _, entity := range snapshot.Entities {
		if string(entity.CharacterID) == selfID || string(entity.ID) == selfID {
			ack = entity.AckSeq
		}
		netID := w.registry.assign(sim.EntityID(entity.ID))
		current[netID] = true
		w.spawned[netID] = true
		delete(w.pendingRemoved, netID)
		spawn := spawnFromSim(entity, netID, w.registry)
		state := deltaFromSim(entity, netID, w.registry)
		w.baselineSent[netID] = state
		w.lastSent[netID] = state
		spawns = append(spawns, spawn)
		states = append(states, state)
	}
	w.visible = current
	tick := uint32(snapshot.Tick)
	return v2.SnapshotFull{
		Tick:        tick,
		BaselineID:  baselineID,
		BaseTick:    tick,
		AckSeq:      ack,
		IsBaseline:  true,
		Spawns:      spawns,
		States:      states,
		Removed:     removed,
		Projectiles: projectilesFromSim(snapshot.Projectiles, w.registry),
	}
}

func (w *worldSession) snapshotLoop(ctx context.Context, feed <-chan sim.Snapshot) {
	edgeAdm, hasEdgeTransfer := w.server.admission.(edgeTransferAdmission)
	selfCharID := sim.CharacterID(w.playerID.String())
	for {
		select {
		case <-ctx.Done():
			return
		case snapshot, ok := <-feed:
			if !ok {
				return
			}
			w.snapshotMu.Lock()
			w.lastSnapshot = snapshot
			w.snapshotMu.Unlock()
			observerView := snapshotForObserver(snapshot, selfCharID)
			if hasEdgeTransfer {
				for _, entity := range snapshot.Entities {
					if entity.CharacterID == selfCharID {
						transfer, crossing, transferErr := edgeAdm.TransferAtEdge(w.playerID, w.controllerID, entity.Transform.Position)
						if transferErr != nil {
							if coded, ok := transferErr.(interface{ ErrorCode() string }); ok && coded.ErrorCode() == "realtime_transfer_denied" {
								if time.Since(w.lastTransferReject) > 10*time.Second {
									w.lastTransferReject = time.Now()
									w.writeReject(coded.ErrorCode(), transferErr.Error(), false)
								}
							}
						} else if crossing {
							payload, err := v2.EncodeTransfer(v2.Transfer{
								FromZoneID: transfer.FromZoneID,
								ToZoneID:   transfer.ToZoneID,
								TransferID: transfer.TransferID,
								TargetX:    quantizeSafe(transfer.TargetX),
								TargetY:    quantizeSafe(transfer.TargetY),
								LegacyX:    int32(transfer.LegacyX),
								LegacyY:    int32(transfer.LegacyY),
								HP:         clampCount(transfer.HP),
							})
							if err == nil {
								_ = w.writeFrame(v2.OpTransfer, payload)
							}
							return
						}
						break
					}
				}
			}
			if err := w.sendSnapshot(observerView); err != nil {
				logger.S.Debugf("WebTransport snapshot to %s stopped: %v", w.playerID, err)
				return
			}
			if err := w.sendEvents(observerView.Events); err != nil {
				logger.S.Debugf("WebTransport events to %s stopped: %v", w.playerID, err)
				return
			}
		}
	}
}

func snapshotForObserver(snapshot sim.Snapshot, selfID sim.CharacterID) sim.Snapshot {
	partyID := ""
	for _, entity := range snapshot.Entities {
		if entity.CharacterID == selfID || string(entity.ID) == string(selfID) {
			partyID = entity.PartyID
			break
		}
	}
	hidden := make(map[string]bool)
	for _, entity := range snapshot.Entities {
		if entity.Kind != sim.EntityPlayer || (!hasStatus(entity, "STEALTH") && !hasStatus(entity, "INVISIBILITY")) {
			continue
		}
		isSelf := entity.CharacterID == selfID || string(entity.ID) == string(selfID)
		isAlly := partyID != "" && entity.PartyID == partyID
		if !isSelf && !isAlly {
			hidden[string(entity.ID)] = true
		}
	}
	if len(hidden) == 0 {
		return snapshot
	}
	view := snapshot
	view.Entities = make([]sim.Entity, 0, len(snapshot.Entities)-len(hidden))
	for _, entity := range snapshot.Entities {
		if !hidden[string(entity.ID)] {
			view.Entities = append(view.Entities, entity)
		}
	}
	view.Events = make([]combat.Event, 0, len(snapshot.Events))
	for _, event := range snapshot.Events {
		if hidden[event.SourceID] || hidden[event.TargetID] {
			continue
		}
		view.Events = append(view.Events, event)
	}
	view.Projectiles = make([]sim.ProjectileState, 0, len(snapshot.Projectiles))
	for _, projectile := range snapshot.Projectiles {
		if !hidden[projectile.SourceID] {
			view.Projectiles = append(view.Projectiles, projectile)
		}
	}
	return view
}

func hasStatus(entity sim.Entity, id string) bool {
	for _, status := range entity.Combat.Statuses {
		if status.ID == id {
			return true
		}
	}
	return false
}

func (w *worldSession) sendSnapshot(snapshot sim.Snapshot) error {
	w.stateMu.Lock()
	defer w.stateMu.Unlock()

	for _, entity := range snapshot.Entities {
		w.registry.assign(sim.EntityID(entity.ID))
	}

	current := make(map[uint16]bool, len(snapshot.Entities))
	var newSpawns []v2.Spawn
	for _, entity := range snapshot.Entities {
		netID, _ := w.registry.lookup(sim.EntityID(entity.ID))
		current[netID] = true
		delete(w.pendingRemoved, netID)
		if !w.spawned[netID] {
			newSpawns = append(newSpawns, spawnFromSim(entity, netID, w.registry))
			w.spawned[netID] = true
		}
	}
	if len(newSpawns) > 0 {
		payload, err := encodeSpawnsPayload(newSpawns)
		if err != nil {
			return err
		}
		if err := w.writeFrame(v2.OpSpawn, payload); err != nil {
			return err
		}
	}

	for netID := range w.visible {
		if !current[netID] {
			delete(w.spawned, netID)
			delete(w.baselineSent, netID)
			delete(w.lastSent, netID)
			w.pendingRemoved[netID] = 3
		}
	}
	w.visible = current

	tick := uint32(snapshot.Tick)
	if tick%baselineTicks == 0 || w.resyncRequested.Swap(false) {
		removed := make([]uint16, 0, len(w.pendingRemoved))
		for netID := range w.pendingRemoved {
			removed = append(removed, netID)
			w.registry.releaseEntity(netID)
		}
		clear(w.pendingRemoved)
		w.baselineID++
		full := w.buildBaselineLocked(snapshot, w.baselineID, removed)
		w.baseTick = full.Tick
		w.lastTick.Store(full.Tick)
		payload, err := v2.EncodeSnapshotFull(full)
		if err != nil {
			return err
		}
		return w.writeFrame(v2.OpSnapshot, payload)
	}

	removed := make([]uint16, 0, len(w.pendingRemoved))
	for netID, remaining := range w.pendingRemoved {
		removed = append(removed, netID)
		if remaining <= 1 {
			delete(w.pendingRemoved, netID)
			w.registry.releaseEntity(netID)
		} else {
			w.pendingRemoved[netID] = remaining - 1
		}
	}

	entities := make([]v2.DeltaEntity, 0, len(snapshot.Entities))
	ack := uint64(0)
	selfID := w.playerID.String()
	for _, entity := range snapshot.Entities {
		if string(entity.CharacterID) == selfID || string(entity.ID) == selfID {
			ack = entity.AckSeq
		}
		netID, _ := w.registry.lookup(sim.EntityID(entity.ID))
		fullState := deltaFromSim(entity, netID, w.registry)
		baseState, hasBase := w.baselineSent[netID]
		prevState, hasPrev := w.lastSent[netID]
		if !hasBase {
			w.baselineSent[netID] = fullState
			w.lastSent[netID] = fullState
			entities = append(entities, fullState)
			continue
		}
		diff, changed := diffDeltaEntity(baseState, prevState, hasPrev, fullState)
		w.lastSent[netID] = fullState
		if changed {
			entities = append(entities, diff)
		}
	}

	projectiles := projectilesFromSim(snapshot.Projectiles, w.registry)

	datagram, err := v2.EncodeSnapshotDelta(v2.SnapshotDelta{
		Tick: tick, BaselineID: w.baselineID, AckSeq: ack,
		Entities: entities, Removed: removed, Projectiles: projectiles,
	})
	if err != nil || len(datagram) > v2.MaxDatagramSize {
		// Overflow: promote to a reliable snapshot on the world stream with the
		// active baseTick so SnapshotBuffer applies it cleanly without resync.
		fullPayload, encodeErr := v2.EncodeSnapshotFull(v2.SnapshotFull{
			Tick: tick, BaselineID: w.baselineID, BaseTick: w.baseTick, AckSeq: ack,
			IsBaseline: false, Spawns: nil, States: entities, Removed: removed, Projectiles: projectiles,
		})
		if encodeErr != nil {
			return encodeErr
		}
		w.lastTick.Store(tick)
		return w.writeFrame(v2.OpSnapshot, fullPayload)
	}
	w.lastTick.Store(tick)
	return w.session.SendDatagram(datagram)
}

type wireEventPayload struct {
	SkillID           string  `json:"skill_id,omitempty"`
	SkillName         string  `json:"skill_name,omitempty"`
	CastVFX           string  `json:"cast_vfx,omitempty"`
	ProjectileVFX     string  `json:"projectile_vfx,omitempty"`
	ImpactVFX         string  `json:"impact_vfx,omitempty"`
	X                 float64 `json:"x,omitempty"`
	Y                 float64 `json:"y,omitempty"`
	FromX             float64 `json:"from_x,omitempty"`
	FromY             float64 `json:"from_y,omitempty"`
	Amount            float64 `json:"amount,omitempty"`
	ShieldAbsorbed    float64 `json:"shield_absorbed,omitempty"`
	HPDamage          float64 `json:"hp_damage,omitempty"`
	HP                float64 `json:"hp,omitempty"`
	Shield            float64 `json:"shield,omitempty"`
	DurationTicks     uint64  `json:"duration_ticks,omitempty"`
	DashDirX          float64 `json:"dash_dir_x,omitempty"`
	DashDirY          float64 `json:"dash_dir_y,omitempty"`
	DashDurationTicks uint64  `json:"dash_duration_ticks,omitempty"`
	Reason            string  `json:"reason,omitempty"`
	TemplateID        string  `json:"template_id,omitempty"`
	MonsterRank       int     `json:"monster_rank,omitempty"`
	PersistentID      string  `json:"persistent_id,omitempty"`
	TargetKind        string  `json:"target_kind,omitempty"`
}

func (w *worldSession) sendEvents(events []combat.Event) error {
	if len(events) == 0 {
		return nil
	}
	out := make([]v2.Event, 0, len(events))
	for _, event := range events {
		w.eventSeq++
		eventID := event.ID
		if eventID == 0 {
			eventID = w.eventSeq
		}
		payload, err := json.Marshal(wireEventPayload{
			SkillID:           event.SkillID,
			SkillName:         event.SkillName,
			CastVFX:           event.CastVFX,
			ProjectileVFX:     event.ProjectileVFX,
			ImpactVFX:         event.ImpactVFX,
			X:                 event.X,
			Y:                 event.Y,
			FromX:             event.FromX,
			FromY:             event.FromY,
			Amount:            event.Amount,
			ShieldAbsorbed:    event.ShieldAbsorbed,
			HPDamage:          event.HPDamage,
			HP:                event.HP,
			Shield:            event.Shield,
			DurationTicks:     event.DurationTicks,
			DashDirX:          event.DashDirX,
			DashDirY:          event.DashDirY,
			DashDurationTicks: event.DashDurationTicks,
			Reason:            event.Reason,
			TemplateID:        event.TemplateID,
			MonsterRank:       event.MonsterRank,
			PersistentID:      event.PersistentID,
			TargetKind:        event.TargetKind,
		})
		if err != nil {
			continue
		}
		sourceNetID, _ := w.registry.lookup(sim.EntityID(event.SourceID))
		targetNetID, _ := w.registry.lookup(sim.EntityID(event.TargetID))
		out = append(out, v2.Event{
			EventID: eventID, Tick: event.Tick, TypeID: v2.EventTypeID(string(event.Type)),
			Source: sourceNetID, Target: targetNetID, Payload: payload,
		})
	}
	if len(out) == 0 {
		return nil
	}
	payload, err := v2.EncodeEvents(w.lastEventID, out)
	if err != nil {
		return err
	}
	w.lastEventID = out[len(out)-1].EventID
	return w.writeFrame(v2.OpEvents, payload)
}

func (w *worldSession) datagramLoop(ctx context.Context) {
	for {
		raw, err := w.session.ReceiveDatagram(ctx)
		if err != nil {
			return
		}
		op, payload, err := v2.DecodeDatagram(raw)
		if err != nil {
			continue
		}
		switch op {
		case v2.OpInput:
			input, err := v2.DecodeInput(payload)
			if err != nil {
				continue
			}
			var move *protocol.Vector
			if input.MoveX != 0 || input.MoveY != 0 {
				move = &protocol.Vector{X: float64(input.MoveX) / 127, Y: float64(input.MoveY) / 127}
			}
			_ = w.server.admission.Input(w.playerID, w.controllerID, protocol.Input{Seq: input.Seq, Move: move})
		case v2.OpPing:
			if len(payload) < 8 {
				continue
			}
			pong := make([]byte, 12)
			copy(pong[0:8], payload[0:8])
			putUint32LE(pong[8:12], w.lastTick.Load())
			if datagram, err := v2.EncodeDatagram(v2.OpPong, pong); err == nil {
				_ = w.session.SendDatagram(datagram)
			}
		}
	}
}

func (w *worldSession) frameLoop(ctx context.Context) {
	for {
		op, payload, err := readFrame(w.stream)
		if err != nil {
			return
		}
		switch op {
		case v2.OpMoveTarget:
			action, err := v2.DecodeMoveTargetAction(payload)
			if err == nil {
				_ = w.server.admission.MoveTarget(w.playerID, w.controllerID, protocol.MoveTarget{
					Seq: action.Seq, X: v2.Dequantize(action.X), Y: v2.Dequantize(action.Y),
				})
			}
		case v2.OpCast:
			action, err := v2.DecodeCastAction(payload)
			if err == nil {
				targetID, _ := w.registry.target(action.TargetNetID)
				_ = w.server.admission.Cast(w.playerID, w.controllerID, protocol.Cast{
					Seq: action.Seq, Slot: action.Slot, Mode: castModeName(action.Mode), TargetID: targetID,
					AimX: float64(action.AimX) / 16, AimY: float64(action.AimY) / 16,
					X: v2.Dequantize(action.X), Y: v2.Dequantize(action.Y),
				})
			}
		case v2.OpAttack:
			action, err := v2.DecodeAttackAction(payload)
			if err == nil {
				targetID, _ := w.registry.target(action.TargetNetID)
				_ = w.server.admission.BasicAttack(w.playerID, w.controllerID, protocol.Attack{
					Seq: action.Seq, TargetID: targetID,
					AimX: float64(action.AimX) / 16, AimY: float64(action.AimY) / 16,
				})
			}
		case v2.OpCapture:
			action, err := v2.DecodeCaptureAction(payload)
			if err == nil {
				targetID, _ := w.registry.target(action.TargetNetID)
				_ = w.server.admission.Capture(w.playerID, w.controllerID, protocol.CaptureIntent{
					Seq: action.Seq, TargetID: targetID, GuID: action.GuID,
				})
			}
		case v2.OpStance:
			action, err := v2.DecodeStanceAction(payload)
			if err == nil {
				focusID, _ := w.registry.target(action.FocusNetID)
				_ = w.server.admission.CompanionStance(w.playerID, w.controllerID, protocol.CompanionStanceIntent{
					Seq: action.Seq, Stance: stanceName(action.Stance), FocusTargetID: focusID,
				})
			}
		case v2.OpLeave:
			_ = w.server.admission.Leave(w.playerID, w.controllerID)
			return
		case v2.OpResync:
			if err := w.sendBaseline(); err != nil {
				return
			}
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (w *worldSession) sendBaseline() error {
	w.snapshotMu.RLock()
	snapshot := w.lastSnapshot
	w.snapshotMu.RUnlock()
	if len(snapshot.Entities) == 0 {
		w.resyncRequested.Store(true)
		return nil
	}
	w.stateMu.Lock()
	tick := uint32(snapshot.Tick)
	if tick <= w.lastTick.Load() {
		// Ensure the very next tick on snapshotLoop is emitted as a baseline,
		// and also emit an immediate baseline advancing tick by 1 if needed so
		// SnapshotBuffer does not ignore a duplicate tick.
		w.resyncRequested.Store(true)
		snapshot.Tick = uint64(w.lastTick.Load() + 1)
	}
	w.baselineID++
	full := w.buildBaselineLocked(snapshot, w.baselineID, nil)
	w.baseTick = full.Tick
	w.lastTick.Store(full.Tick)
	w.stateMu.Unlock()

	payload, err := v2.EncodeSnapshotFull(full)
	if err != nil {
		return err
	}
	return w.writeFrame(v2.OpSnapshot, payload)
}

func (w *worldSession) writeFrame(op byte, payload []byte) error {
	frame, err := v2.EncodeFrame(op, payload)
	if err != nil {
		return err
	}
	w.writeMu.Lock()
	defer w.writeMu.Unlock()
	_ = w.stream.SetWriteDeadline(time.Now().Add(5 * time.Second))
	_, err = w.stream.Write(frame)
	return err
}

func (w *worldSession) writeReject(code, message string, retryable bool) {
	payload, err := v2.EncodeReject(code, message, retryable)
	if err != nil {
		return
	}
	_ = w.writeFrame(v2.OpReject, payload)
}

func readFrame(reader io.Reader) (byte, []byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return 0, nil, err
	}
	length := int(uint32(header[0]) | uint32(header[1])<<8 | uint32(header[2])<<16 | uint32(header[3])<<24)
	if length < 1 || length > v2.MaxFrameSize {
		return 0, nil, fmt.Errorf("invalid frame length %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return 0, nil, err
	}
	return buf[0], buf[1:], nil
}

func putUint32LE(dst []byte, value uint32) {
	dst[0] = byte(value)
	dst[1] = byte(value >> 8)
	dst[2] = byte(value >> 16)
	dst[3] = byte(value >> 24)
}

func admissionCode(err error) string {
	var coded interface{ ErrorCode() string }
	if errors.As(err, &coded) {
		return coded.ErrorCode()
	}
	return "realtime_world_rejected"
}

func admissionRetryable(err error) bool {
	var coded interface{ HTTPStatus() int }
	if errors.As(err, &coded) {
		return coded.HTTPStatus() == http.StatusTooManyRequests
	}
	return false
}

type netRegistry struct {
	mu         sync.RWMutex
	next       uint16
	byEntity   map[sim.EntityID]uint16
	byNet      map[uint16]sim.EntityID
	projectile map[uint64]uint16
	byProjNet  map[uint16]uint64
	nextProj   uint16
}

func newNetRegistry() *netRegistry {
	return &netRegistry{
		byEntity:   make(map[sim.EntityID]uint16),
		byNet:      make(map[uint16]sim.EntityID),
		projectile: make(map[uint64]uint16),
		byProjNet:  make(map[uint16]uint64),
	}
}

func (r *netRegistry) assign(id sim.EntityID) uint16 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if netID, ok := r.byEntity[id]; ok {
		return netID
	}
	if len(r.byNet) >= 0xFFFF {
		return 0
	}
	for {
		r.next++
		if r.next == 0 {
			r.next = 1
		}
		if _, used := r.byNet[r.next]; !used {
			break
		}
	}
	r.byEntity[id] = r.next
	r.byNet[r.next] = id
	return r.next
}

func (r *netRegistry) lookup(id sim.EntityID) (uint16, bool) {
	if id == "" {
		return 0, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	netID, ok := r.byEntity[id]
	return netID, ok
}

func (r *netRegistry) releaseEntity(netID uint16) {
	if netID == 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.byNet[netID]; ok {
		delete(r.byNet, netID)
		delete(r.byEntity, id)
	}
}

func (r *netRegistry) target(netID uint16) (string, bool) {
	if netID == 0 {
		return "", false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byNet[netID]
	return string(id), ok
}

func (r *netRegistry) projectileID(id uint64) uint16 {
	r.mu.Lock()
	defer r.mu.Unlock()
	if netID, ok := r.projectile[id]; ok {
		return netID
	}
	if len(r.byProjNet) >= 0xFFFF {
		return 0
	}
	for {
		r.nextProj++
		if r.nextProj == 0 {
			r.nextProj = 1
		}
		if _, used := r.byProjNet[r.nextProj]; !used {
			break
		}
	}
	r.projectile[id] = r.nextProj
	r.byProjNet[r.nextProj] = id
	return r.nextProj
}

func (r *netRegistry) pruneProjectiles(active []sim.ProjectileState) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.projectile) == 0 {
		return
	}
	if len(active) == 0 {
		clear(r.projectile)
		clear(r.byProjNet)
		return
	}
	keep := make(map[uint64]struct{}, len(active))
	for _, p := range active {
		keep[p.ID] = struct{}{}
	}
	for id, netID := range r.projectile {
		if _, ok := keep[id]; !ok {
			delete(r.projectile, id)
			delete(r.byProjNet, netID)
		}
	}
}

func speedX100(speed float64) uint16 {
	if speed < 0 || math.IsNaN(speed) || math.IsInf(speed, 0) {
		return 0
	}
	value := math.Round(speed * 100)
	if value > 65535 {
		value = 65535
	}
	return uint16(value)
}

func quantizeSafe(value float64) uint16 {
	quantized, err := v2.Quantize(value)
	if err != nil {
		return 0
	}
	return quantized
}

func clampCount(value float64) uint64 {
	if value <= 0 || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return uint64(math.Round(value))
}

func kindCode(kind sim.EntityKind) uint8 {
	switch kind {
	case sim.EntityMonster:
		return 1
	case sim.EntityCompanion:
		return 2
	default:
		return 0
	}
}

func aiStateCode(state sim.AIStateKind) uint8 {
	switch state {
	case sim.AIIdle:
		return 1
	case sim.AIPatrol:
		return 2
	case sim.AIAggro:
		return 3
	case sim.AIChase:
		return 4
	case sim.AIAttack:
		return 5
	case sim.AILeash:
		return 6
	case sim.AIDead:
		return 7
	case sim.AIRespawn:
		return 8
	default:
		return 0
	}
}

func stanceCode(stance string) uint8 {
	switch stance {
	case "FOCUS":
		return 1
	case "FOLLOW":
		return 2
	default:
		return 0
	}
}

func stanceName(code uint8) string {
	switch code {
	case 1:
		return "FOCUS"
	case 2:
		return "FOLLOW"
	default:
		return "DEFEND"
	}
}

func castModeName(mode uint8) string {
	switch mode {
	case 1:
		return "DIRECTION"
	case 2:
		return "GROUND"
	case 3:
		return "SELF"
	default:
		return "TARGET"
	}
}

func flagsFromSim(entity sim.Entity) uint8 {
	flags := uint8(0)
	if entity.PvPEnabled {
		flags |= 1 << 0
	}
	if len(entity.Combat.Statuses) > 0 {
		// Status ids stay out of v1 delta; STEALTH still needs the flag.
		for _, status := range entity.Combat.Statuses {
			if status.ID == "STEALTH" || status.ID == "INVISIBILITY" {
				flags |= 1 << 1
			}
			if status.ID == "FLYING" || status.ID == "FLIGHT" {
				flags |= 1 << 2
			}
			if status.ID == "HASTE" && status.MoveMultiplier > 1 {
				flags |= 1 << 4
			}
		}
	}
	if entity.Combat.FormActive {
		flags |= 1 << 3
	}
	return flags
}

func spawnFromSim(entity sim.Entity, netID uint16, registry *netRegistry) v2.Spawn {
	entityID := string(entity.CharacterID)
	if entityID == "" {
		entityID = string(entity.ID)
	}
	name := entity.Name
	templateID := ""
	if entity.Kind == sim.EntityMonster {
		templateID = entity.AI.TemplateID
		if name == "" {
			name = entity.AI.Name
		}
	}
	if entity.Kind == sim.EntityPlayer && entity.Transformation.Active {
		templateID = entity.Transformation.FormID
	}
	ownerNetID, _ := registry.lookup(sim.EntityID(entity.Companion.OwnerID))
	return v2.Spawn{
		NetID: netID, EntityID: entityID, Name: name, TemplateID: templateID,
		Kind: kindCode(entity.Kind), PartyID: entity.PartyID, Flags: flagsFromSim(entity),
		RadiusX10:     uint8(math.Round(math.Min(25.5, entity.Radius) * 10)),
		MaxHP:         clampCount(entity.Vitals.MaxHP),
		MaxMana:       clampCount(entity.Combat.MaxMana),
		MaxThought:    clampCount(entity.Combat.MaxThought),
		OwnerNetID:    ownerNetID,
		CompanionKind: entity.Companion.ActorKind,
		Stance:        stanceCode(string(entity.Companion.Stance)),
	}
}

func deltaFromSim(entity sim.Entity, netID uint16, registry *netRegistry) v2.DeltaEntity {
	out := v2.DeltaEntity{
		NetID:     netID,
		Mask:      v2.MaskPosition | v2.MaskVitals | v2.MaskRevision | v2.MaskFlags | v2.MaskStatuses,
		X:         quantizeSafe(entity.Transform.Position.X),
		Y:         quantizeSafe(entity.Transform.Position.Y),
		VitalsSub: v2.VitalHP | v2.VitalShield | v2.VitalMana | v2.VitalThought,
		HP:        clampCount(entity.Vitals.HP),
		Shield:    clampCount(entity.Vitals.Shield),
		Mana:      clampCount(entity.Combat.Mana),
		Thought:   clampCount(entity.Combat.Thought),
		Revision:  entity.Revision,
		Flags:     flagsFromSim(entity),
	}
	for _, status := range entity.Combat.Statuses {
		if code := v2.StatusCode(status.ID); code != 0 {
			out.Statuses = append(out.Statuses, code)
		}
	}
	if entity.Combat.Pending != nil {
		out.Mask |= v2.MaskCast
		out.SkillSlot = uint8(entity.Combat.Pending.Skill.Slot)
		out.CastEndTick = uint32(entity.Combat.Pending.ResolveTick)
		out.CastWindup = uint16(entity.Combat.Pending.Skill.WindupTicks)
	}
	if entity.Kind == sim.EntityMonster {
		aiState := aiStateCode(entity.AI.State)
		aggroNetID, _ := registry.lookup(entity.AI.Threat)
		if aiState != 0 || aggroNetID != 0 {
			out.HasExt = true
			out.Ext |= v2.ExtAIState | v2.ExtAggro
			out.AIState = aiState
			out.AggroNetID = aggroNetID
		}
	} else if entity.Kind == sim.EntityCompanion {
		stance := stanceCode(string(entity.Companion.Stance))
		out.HasExt = true
		out.Ext |= v2.ExtStance
		out.Stance = stance
	}
	return out
}

// diffDeltaEntity produces a minimal v2.DeltaEntity carrying only fields that
// differ from either the active baseline (for UDP packet-loss resilience) or
// the last sent tick (so return-to-baseline transitions are still emitted).
func diffDeltaEntity(base, prev v2.DeltaEntity, hasPrev bool, curr v2.DeltaEntity) (v2.DeltaEntity, bool) {
	if !hasPrev {
		prev = base
	}
	diff := v2.DeltaEntity{NetID: curr.NetID}

	if curr.X != base.X || curr.Y != base.Y || curr.X != prev.X || curr.Y != prev.Y {
		diff.Mask |= v2.MaskPosition
		diff.X = curr.X
		diff.Y = curr.Y
	}

	var vitalsSub uint8
	if curr.HP != base.HP || curr.HP != prev.HP {
		vitalsSub |= v2.VitalHP
		diff.HP = curr.HP
	}
	if curr.Shield != base.Shield || curr.Shield != prev.Shield {
		vitalsSub |= v2.VitalShield
		diff.Shield = curr.Shield
	}
	if curr.Mana != base.Mana || curr.Mana != prev.Mana {
		vitalsSub |= v2.VitalMana
		diff.Mana = curr.Mana
	}
	if curr.Thought != base.Thought || curr.Thought != prev.Thought {
		vitalsSub |= v2.VitalThought
		diff.Thought = curr.Thought
	}
	if vitalsSub != 0 {
		diff.Mask |= v2.MaskVitals
		diff.VitalsSub = vitalsSub
	}

	if curr.Flags != base.Flags || curr.Flags != prev.Flags {
		diff.Mask |= v2.MaskFlags
		diff.Flags = curr.Flags
	}
	if !slices.Equal(curr.Statuses, base.Statuses) || !slices.Equal(curr.Statuses, prev.Statuses) {
		diff.Mask |= v2.MaskStatuses
		diff.Statuses = append([]uint8(nil), curr.Statuses...)
	}

	currCasting := curr.Mask&v2.MaskCast != 0
	baseCasting := base.Mask&v2.MaskCast != 0
	prevCasting := prev.Mask&v2.MaskCast != 0
	if currCasting != baseCasting || currCasting != prevCasting ||
		(currCasting && (curr.SkillSlot != prev.SkillSlot || curr.CastEndTick != prev.CastEndTick || curr.CastWindup != prev.CastWindup)) {
		diff.Mask |= v2.MaskCast
		diff.SkillSlot = curr.SkillSlot
		diff.CastEndTick = curr.CastEndTick
		diff.CastWindup = curr.CastWindup
	}

	var ext uint8
	if (curr.Ext&v2.ExtAIState != 0 || base.Ext&v2.ExtAIState != 0 || prev.Ext&v2.ExtAIState != 0) &&
		(curr.AIState != base.AIState || curr.AIState != prev.AIState) {
		ext |= v2.ExtAIState
		diff.AIState = curr.AIState
	}
	if (curr.Ext&v2.ExtAggro != 0 || base.Ext&v2.ExtAggro != 0 || prev.Ext&v2.ExtAggro != 0) &&
		(curr.AggroNetID != base.AggroNetID || curr.AggroNetID != prev.AggroNetID) {
		ext |= v2.ExtAggro
		diff.AggroNetID = curr.AggroNetID
	}
	if (curr.Ext&v2.ExtStance != 0 || base.Ext&v2.ExtStance != 0 || prev.Ext&v2.ExtStance != 0) &&
		(curr.Stance != base.Stance || curr.Stance != prev.Stance) {
		ext |= v2.ExtStance
		diff.Stance = curr.Stance
	}
	if ext != 0 {
		diff.HasExt = true
		diff.Ext = ext
	}

	if curr.Revision != base.Revision || curr.Revision != prev.Revision || diff.Mask != 0 || diff.HasExt {
		if curr.Revision != base.Revision || curr.Revision != prev.Revision {
			diff.Mask |= v2.MaskRevision
			diff.Revision = curr.Revision
		}
	}

	changed := diff.Mask != 0 || diff.HasExt
	return diff, changed
}

func projectilesFromSim(projectiles []sim.ProjectileState, registry *netRegistry) []v2.Projectile {
	registry.pruneProjectiles(projectiles)
	if len(projectiles) == 0 {
		return nil
	}
	out := make([]v2.Projectile, 0, len(projectiles))
	for _, projectile := range projectiles {
		sourceNetID, _ := registry.lookup(sim.EntityID(projectile.SourceID))
		out = append(out, v2.Projectile{
			ID: registry.projectileID(projectile.ID), SourceNetID: sourceNetID,
			SkillID: projectile.SkillID, VFX: projectile.ProjectileVFX,
			X: quantizeSafe(projectile.X), Y: quantizeSafe(projectile.Y),
			RadiusX10: uint8(math.Round(math.Min(25.5, projectile.Radius) * 10)),
		})
	}
	return out
}

func encodeSpawnsPayload(spawns []v2.Spawn) ([]byte, error) {
	if len(spawns) > 255 {
		spawns = spawns[:255]
	}
	encoder := v2.NewEncoder(128 * len(spawns))
	encoder.U8(uint8(len(spawns)))
	for _, spawn := range spawns {
		v2.EncodeSpawn(encoder, spawn)
	}
	return encoder.Bytes(), encoder.Err()
}
