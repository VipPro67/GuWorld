// Package wttransport serves the authoritative realtime world over
// WebTransport. It is the v2 counterpart of internal/ws and intentionally
// depends only on the admission surface, not on the WebSocket hub.
package wttransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/quic-go/webtransport-go"
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
	joinFrameTimeout = 10 * time.Second
	baselineTicks    = 40
)

// Server upgrades WebTransport requests and runs one world session each.
type Server struct {
	admission Admission
	wt        *webtransport.Server
}

func New(admission Admission, wt *webtransport.Server) *Server {
	return &Server{admission: admission, wt: wt}
}

// Handle must run after authentication; playerID/tabID come from the
// authenticated request, never from the join payload.
func (s *Server) Handle(w http.ResponseWriter, r *http.Request, playerID uuid.UUID, tabID string) {
	session, err := s.wt.Upgrade(w, r)
	if err != nil {
		logger.S.Warnf("WebTransport upgrade failed for %s: %v", playerID, err)
		http.Error(w, "webtransport upgrade failed", http.StatusBadRequest)
		return
	}
	controllerID := uuid.New()
	go s.serve(session, playerID, controllerID, tabID)
}

type worldSession struct {
	server       *Server
	session      *webtransport.Session
	stream       *webtransport.Stream
	playerID     uuid.UUID
	controllerID uuid.UUID
	tabID        string

	writeMu sync.Mutex

	registry   *netRegistry
	spawned    map[uint16]bool
	visible    map[uint16]bool
	baselineID uint16
	lastTick   uint32

	lastSnapshot sim.Snapshot
	snapshotMu   sync.RWMutex
}

func (s *Server) serve(session *webtransport.Session, playerID, controllerID uuid.UUID, tabID string) {
	defer session.CloseWithError(0, "session ended")
	ctx := session.Context()

	stream, err := session.AcceptStream(ctx)
	if err != nil {
		logger.S.Warnf("WebTransport accept stream failed: %v", err)
		return
	}
	logger.S.Debugf("WebTransport stream accepted for %s", playerID)
	world := &worldSession{
		server: s, session: session, stream: stream,
		playerID: playerID, controllerID: controllerID, tabID: tabID,
		registry: newNetRegistry(), spawned: map[uint16]bool{}, visible: map[uint16]bool{},
	}
	defer func() {
		s.admission.Disconnect(playerID, controllerID)
	}()

	request, err := world.readJoin()
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

	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		world.datagramLoop(ctx)
	}()
	go func() {
		defer workers.Done()
		world.snapshotLoop(ctx, feed)
	}()

	// Frame loop runs on the accept goroutine.
	world.frameLoop(ctx)
	session.CloseWithError(0, "world stream closed")
	workers.Wait()
}

func (w *worldSession) readJoin() (v2.JoinRequest, error) {
	handshake := make([]byte, 5)
	if _, err := io.ReadFull(w.stream, handshake); err != nil {
		return v2.JoinRequest{}, err
	}
	if handshake[0] != v2.StreamWorld || string(handshake[1:4]) != v2.Magic || handshake[4] != v2.Version {
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
	var snapshot protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &snapshot); err != nil {
		return fmt.Errorf("decode welcome snapshot: %w", err)
	}
	// Assign net ids for every visible entity first so owner references resolve.
	for _, entity := range snapshot.Entities {
		w.registry.assign(sim.EntityID(entity.EntityID))
	}
	spawns := make([]v2.Spawn, 0, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		netID, _ := w.registry.byEntity[sim.EntityID(entity.EntityID)]
		spawn := spawnFromProtocol(entity, netID, w.registry)
		w.spawned[netID] = true
		w.visible[netID] = true
		spawns = append(spawns, spawn)
	}
	full := v2.SnapshotFull{
		Tick:        uint32(snapshot.Tick),
		BaselineID:  uint16(welcome.BaselineID),
		BaseTick:    uint32(snapshot.Tick),
		AckSeq:      snapshot.AckSeq,
		IsBaseline:  true,
		Spawns:      spawns,
		Projectiles: projectilesFromProtocol(snapshot.Projectiles, w.registry),
	}
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
		BaselineID:        uint16(welcome.BaselineID),
		MapHash:           welcome.MapHash,
		SkillsJSON:        skills,
		LoadoutRevision:   welcome.LoadoutRevision,
		Snapshot:          fullPayload,
	})
	if err != nil {
		return err
	}
	w.baselineID = uint16(welcome.BaselineID)
	w.lastTick = full.Tick
	return w.writeFrame(v2.OpWelcome, payload)
}

func (w *worldSession) snapshotLoop(ctx context.Context, feed <-chan sim.Snapshot) {
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
			if err := w.sendSnapshot(snapshot); err != nil {
				logger.S.Debugf("WebTransport snapshot to %s stopped: %v", w.playerID, err)
				return
			}
		}
	}
}

func (w *worldSession) sendSnapshot(snapshot sim.Snapshot) error {
	current := make(map[uint16]bool, len(snapshot.Entities))
	entities := make([]v2.DeltaEntity, 0, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		netID := w.registry.assign(sim.EntityID(entity.ID))
		current[netID] = true
		if !w.spawned[netID] {
			// New AOI entity between baselines: send its metadata once.
			spawn := spawnFromSim(entity, netID, w.registry)
			payload, err := encodeSpawnPayload(spawn)
			if err != nil {
				return err
			}
			if err := w.writeFrame(v2.OpSpawn, payload); err != nil {
				return err
			}
			w.spawned[netID] = true
		}
		entities = append(entities, deltaFromSim(entity, netID))
	}
	removed := make([]uint16, 0)
	for netID := range w.visible {
		if !current[netID] {
			removed = append(removed, netID)
			delete(w.spawned, netID)
		}
	}
	w.visible = current
	projectiles := projectilesFromSim(snapshot.Projectiles, w.registry)
	tick := uint32(snapshot.Tick)
	ack := uint64(0)
	for _, entity := range snapshot.Entities {
		if string(entity.CharacterID) == w.playerID.String() {
			ack = entity.AckSeq
			break
		}
	}

	if tick%baselineTicks == 0 {
		w.baselineID++
		spawns := make([]v2.Spawn, 0, len(current))
		for _, entity := range snapshot.Entities {
			netID := w.registry.byEntity[sim.EntityID(entity.ID)]
			spawns = append(spawns, spawnFromSim(entity, netID, w.registry))
		}
		payload, err := v2.EncodeSnapshotFull(v2.SnapshotFull{
			Tick: tick, BaselineID: w.baselineID, BaseTick: tick, AckSeq: ack,
			IsBaseline: true, Spawns: spawns, Removed: removed, Projectiles: projectiles,
		})
		if err != nil {
			return err
		}
		w.lastTick = tick
		return w.writeFrame(v2.OpSnapshot, payload)
	}

	datagram, err := v2.EncodeSnapshotDelta(v2.SnapshotDelta{
		Tick: tick, BaselineID: w.baselineID, AckSeq: ack,
		Entities: entities, Removed: removed, Projectiles: projectiles,
	})
	if err != nil || len(datagram) > v2.MaxDatagramSize {
		// Overflow: send the same tick as a reliable full snapshot frame.
		spawns := make([]v2.Spawn, 0, len(current))
		for _, entity := range snapshot.Entities {
			netID := w.registry.byEntity[sim.EntityID(entity.ID)]
			spawns = append(spawns, spawnFromSim(entity, netID, w.registry))
		}
		fullPayload, encodeErr := v2.EncodeSnapshotFull(v2.SnapshotFull{
			Tick: tick, BaselineID: w.baselineID, BaseTick: tick, AckSeq: ack,
			IsBaseline: false, Spawns: spawns, Removed: removed, Projectiles: projectiles,
		})
		if encodeErr != nil {
			return encodeErr
		}
		return w.writeFrame(v2.OpSnapshot, fullPayload)
	}
	w.lastTick = tick
	return w.session.SendDatagram(datagram)
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
			putUint32LE(pong[8:12], w.lastTick)
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
		return nil
	}
	spawns := make([]v2.Spawn, 0, len(snapshot.Entities))
	for _, entity := range snapshot.Entities {
		netID := w.registry.assign(sim.EntityID(entity.ID))
		spawns = append(spawns, spawnFromSim(entity, netID, w.registry))
	}
	w.baselineID++
	payload, err := v2.EncodeSnapshotFull(v2.SnapshotFull{
		Tick: uint32(snapshot.Tick), BaselineID: w.baselineID, BaseTick: uint32(snapshot.Tick),
		IsBaseline: true, Spawns: spawns,
		Projectiles: projectilesFromSim(snapshot.Projectiles, w.registry),
	})
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
	next       uint16
	byEntity   map[sim.EntityID]uint16
	byNet      map[uint16]sim.EntityID
	projectile map[uint64]uint16
	nextProj   uint16
}

func newNetRegistry() *netRegistry {
	return &netRegistry{
		byEntity:   make(map[sim.EntityID]uint16),
		byNet:      make(map[uint16]sim.EntityID),
		projectile: make(map[uint64]uint16),
	}
}

func (r *netRegistry) assign(id sim.EntityID) uint16 {
	if netID, ok := r.byEntity[id]; ok {
		return netID
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

func (r *netRegistry) target(netID uint16) (string, bool) {
	if netID == 0 {
		return "", false
	}
	id, ok := r.byNet[netID]
	return string(id), ok
}

func (r *netRegistry) projectileID(id uint64) uint16 {
	if netID, ok := r.projectile[id]; ok {
		return netID
	}
	for {
		r.nextProj++
		if r.nextProj == 0 {
			r.nextProj = 1
		}
		if _, used := r.projectile[id]; !used {
			break
		}
	}
	r.projectile[id] = r.nextProj
	return r.nextProj
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

func protocolKindCode(kind string) uint8 {
	switch kind {
	case "monster":
		return 1
	case "companion":
		return 2
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
			if status.ID == "STEALTH" {
				flags |= 1 << 1
			}
			if status.ID == "FLYING" || status.ID == "FLIGHT" {
				flags |= 1 << 2
			}
		}
	}
	if entity.Combat.FormActive {
		flags |= 1 << 3
	}
	return flags
}

func flagsFromProtocol(entity protocol.EntityState) uint8 {
	flags := uint8(0)
	if entity.PvPEnabled {
		flags |= 1 << 0
	}
	if entity.Stealthed {
		flags |= 1 << 1
	}
	if entity.Flying {
		flags |= 1 << 2
	}
	if entity.FormActive {
		flags |= 1 << 3
	}
	return flags
}

func spawnFromSim(entity sim.Entity, netID uint16, registry *netRegistry) v2.Spawn {
	entityID := string(entity.CharacterID)
	if entityID == "" {
		entityID = string(entity.ID)
	}
	ownerNetID := uint16(0)
	if entity.Companion.OwnerID != "" {
		if owner, ok := registry.byEntity[sim.EntityID(entity.Companion.OwnerID)]; ok {
			ownerNetID = owner
		}
	}
	return v2.Spawn{
		NetID: netID, EntityID: entityID, Name: entity.Name, TemplateID: "",
		Kind: kindCode(entity.Kind), PartyID: entity.PartyID, Flags: flagsFromSim(entity),
		RadiusX10: uint8(math.Round(math.Min(25.5, entity.Radius) * 10)),
		MaxHP:     clampCount(entity.Vitals.MaxHP), MaxMana: clampCount(entity.Combat.MaxMana),
		MaxThought: clampCount(entity.Combat.MaxThought), OwnerNetID: ownerNetID,
		CompanionKind: entity.Companion.ActorKind, Stance: stanceCode(string(entity.Companion.Stance)),
	}
}

func spawnFromProtocol(entity protocol.EntityState, netID uint16, registry *netRegistry) v2.Spawn {
	ownerNetID := uint16(0)
	if entity.OwnerID != "" {
		if owner, ok := registry.byEntity[sim.EntityID(entity.OwnerID)]; ok {
			ownerNetID = owner
		}
	}
	return v2.Spawn{
		NetID: netID, EntityID: entity.EntityID, Name: entity.Name, TemplateID: entity.TemplateID,
		Kind: protocolKindCode(entity.Kind), PartyID: entity.PartyID, Flags: flagsFromProtocol(entity),
		RadiusX10: uint8(math.Round(math.Min(25.5, entity.Radius) * 10)),
		MaxHP:     clampCount(entity.MaxHP), MaxMana: clampCount(entity.MaxMana),
		MaxThought: clampCount(entity.MaxThought), OwnerNetID: ownerNetID,
		CompanionKind: entity.CompanionKind, Stance: stanceCode(entity.Stance),
	}
}

func deltaFromSim(entity sim.Entity, netID uint16) v2.DeltaEntity {
	return v2.DeltaEntity{
		NetID: netID,
		Mask:  v2.MaskPosition | v2.MaskVitals | v2.MaskRevision | v2.MaskFlags,
		X:     quantizeSafe(entity.Transform.Position.X), Y: quantizeSafe(entity.Transform.Position.Y),
		VitalsSub: v2.VitalHP | v2.VitalShield | v2.VitalMana | v2.VitalThought,
		HP:        clampCount(entity.Vitals.HP), Shield: clampCount(entity.Vitals.Shield),
		Mana: clampCount(entity.Combat.Mana), Thought: clampCount(entity.Combat.Thought),
		Revision: entity.Revision, Flags: flagsFromSim(entity),
	}
}

func deltaFromProtocol(entity protocol.EntityState, netID uint16) v2.DeltaEntity {
	return v2.DeltaEntity{
		NetID: netID,
		Mask:  v2.MaskPosition | v2.MaskVitals | v2.MaskRevision | v2.MaskFlags,
		X:     quantizeSafe(entity.X), Y: quantizeSafe(entity.Y),
		VitalsSub: v2.VitalHP | v2.VitalShield | v2.VitalMana | v2.VitalThought,
		HP:        clampCount(entity.HP), Shield: clampCount(entity.Shield),
		Mana: clampCount(entity.Mana), Thought: clampCount(entity.Thought),
		Revision: entity.Revision, Flags: flagsFromProtocol(entity),
	}
}

func projectilesFromSim(projectiles []sim.ProjectileState, registry *netRegistry) []v2.Projectile {
	if len(projectiles) == 0 {
		return nil
	}
	out := make([]v2.Projectile, 0, len(projectiles))
	for _, projectile := range projectiles {
		sourceNetID := uint16(0)
		if source, ok := registry.byEntity[sim.EntityID(projectile.SourceID)]; ok {
			sourceNetID = source
		}
		out = append(out, v2.Projectile{
			ID: registry.projectileID(projectile.ID), SourceNetID: sourceNetID,
			SkillID: projectile.SkillID, VFX: projectile.ProjectileVFX,
			X: quantizeSafe(projectile.X), Y: quantizeSafe(projectile.Y),
			RadiusX10: uint8(math.Round(math.Min(25.5, projectile.Radius) * 10)),
		})
	}
	return out
}

func projectilesFromProtocol(projectiles []protocol.ProjectileState, registry *netRegistry) []v2.Projectile {
	if len(projectiles) == 0 {
		return nil
	}
	out := make([]v2.Projectile, 0, len(projectiles))
	for _, projectile := range projectiles {
		sourceNetID := uint16(0)
		if source, ok := registry.byEntity[sim.EntityID(projectile.SourceID)]; ok {
			sourceNetID = source
		}
		out = append(out, v2.Projectile{
			ID: registry.projectileID(projectile.ID), SourceNetID: sourceNetID,
			SkillID: projectile.SkillID, VFX: projectile.ProjectileVFX,
			X: quantizeSafe(projectile.X), Y: quantizeSafe(projectile.Y),
			RadiusX10: uint8(math.Round(math.Min(25.5, projectile.Radius) * 10)),
		})
	}
	return out
}

func encodeSpawnPayload(spawn v2.Spawn) ([]byte, error) {
	encoder := v2.NewEncoder(128)
	encoder.U8(1)
	v2.EncodeSpawn(encoder, spawn)
	return encoder.Bytes(), encoder.Err()
}
