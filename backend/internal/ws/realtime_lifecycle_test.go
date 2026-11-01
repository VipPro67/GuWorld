package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/realtime/config"
	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
	"github.com/vippro67/GuWorld/backend/internal/realtime/gateway"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
)

type fakeRealtimeAdmission struct {
	joinPlayer       uuid.UUID
	joinController   uuid.UUID
	joinRequest      protocol.JoinRequest
	leaveCalls       int
	disconnects      int
	joinErr          error
	leaveErr         error
	inputPlayer      uuid.UUID
	inputController  uuid.UUID
	inputValue       protocol.Input
	targetPlayer     uuid.UUID
	targetController uuid.UUID
	targetValue      protocol.MoveTarget
	castPlayer       uuid.UUID
	castController   uuid.UUID
	castValue        protocol.Cast
}

func TestMonsterSnapshotCarriesPresentationAndAggro(t *testing.T) {
	entity := sim.Entity{ID: "m:1", Kind: sim.EntityMonster, AI: sim.AIState{TemplateID: "monster_wolf", Name: "Lang Yêu", State: sim.AIChase, Threat: "player-1"}}
	state := protocolEntityState(entity)
	if state.TemplateID != "monster_wolf" || state.Name != "Lang Yêu" || state.AIState != "chase" || state.AggroTarget != "player-1" {
		t.Fatalf("monster presentation missing from snapshot: %+v", state)
	}
}

func (f *fakeRealtimeAdmission) Join(_ context.Context, playerID, controllerID uuid.UUID, request protocol.JoinRequest) (any, error) {
	f.joinPlayer, f.joinController, f.joinRequest = playerID, controllerID, request
	return map[string]string{"type": "world.welcome"}, f.joinErr
}

func (f *fakeRealtimeAdmission) Input(playerID, controllerID uuid.UUID, input protocol.Input) error {
	f.inputPlayer, f.inputController, f.inputValue = playerID, controllerID, input
	return nil
}

func (f *fakeRealtimeAdmission) MoveTarget(playerID, controllerID uuid.UUID, target protocol.MoveTarget) error {
	f.targetPlayer, f.targetController, f.targetValue = playerID, controllerID, target
	return nil
}

func (f *fakeRealtimeAdmission) Cast(playerID, controllerID uuid.UUID, cast protocol.Cast) error {
	f.castPlayer, f.castController, f.castValue = playerID, controllerID, cast
	return nil
}

func (f *fakeRealtimeAdmission) BasicAttack(_, _ uuid.UUID, _ protocol.Attack) error {
	return nil
}

func (f *fakeRealtimeAdmission) Capture(_, _ uuid.UUID, _ protocol.CaptureIntent) error {
	return nil
}

func (f *fakeRealtimeAdmission) CompanionStance(_, _ uuid.UUID, _ protocol.CompanionStanceIntent) error {
	return nil
}

func (f *fakeRealtimeAdmission) TransferAtEdge(_, _ uuid.UUID, _ sim.Vec2) (protocol.Transfer, bool, error) {
	return protocol.Transfer{}, false, nil
}

func (f *fakeRealtimeAdmission) BeginHandoff(_ uuid.UUID) (bool, error) { return false, nil }
func (f *fakeRealtimeAdmission) AbortHandoff(_ uuid.UUID)               {}
func (f *fakeRealtimeAdmission) FinishHandoff(_ uuid.UUID)              {}
func (f *fakeRealtimeAdmission) PlayerLocation(_ uuid.UUID) (string, int, int, bool) {
	return "", 0, 0, false
}

func (f *fakeRealtimeAdmission) SubscribeSnapshots(_, _ uuid.UUID) (<-chan sim.Snapshot, func(), error) {
	return make(chan sim.Snapshot), func() {}, nil
}

func (f *fakeRealtimeAdmission) Leave(_, _ uuid.UUID) error {
	f.leaveCalls++
	return f.leaveErr
}

func (f *fakeRealtimeAdmission) Disconnect(_, _ uuid.UUID) { f.disconnects++ }

func TestRealtimeLifecycleUsesAuthenticatedConnectionIdentity(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	admission := &fakeRealtimeAdmission{}
	client := &Client{playerID: playerID, controllerID: controllerID, realtimeAdmission: admission, done: make(chan struct{})}
	status, data := client.handleRealtimeLifecycle(WSMessage{Channel: "world.join", Payload: json.RawMessage(`{"zone_id":"forest","map_version":"map-1","content_version":"content-1"}`)})
	if status != http.StatusOK || admission.joinPlayer != playerID || admission.joinController != controllerID || admission.joinRequest.ZoneID != "forest" {
		t.Fatalf("join identity/request mismatch: status=%d call=%+v body=%s", status, admission, data)
	}

	status, data = client.handleRealtimeLifecycle(WSMessage{Channel: "world.leave"})
	if status != http.StatusOK || admission.leaveCalls != 1 || string(data) != `{"type":"world.left"}` {
		t.Fatalf("leave response/call mismatch: status=%d calls=%d body=%s", status, admission.leaveCalls, data)
	}
	client.Close()
	client.Close()
	if admission.disconnects != 1 {
		t.Fatalf("disconnect cleanup calls=%d, want once", admission.disconnects)
	}
}

func TestRealtimeLifecycleRejectsMalformedJoinAndMissingGateway(t *testing.T) {
	client := &Client{playerID: uuid.New(), controllerID: uuid.New(), done: make(chan struct{}), realtimeAdmission: &fakeRealtimeAdmission{}}
	status, _ := client.handleRealtimeLifecycle(WSMessage{Channel: "world.join", Payload: json.RawMessage(`{"zone_id":""}`)})
	if status != http.StatusBadRequest {
		t.Fatalf("empty map id status=%d", status)
	}
	client.realtimeAdmission = nil
	status, _ = client.handleRealtimeLifecycle(WSMessage{Channel: "world.join", Payload: json.RawMessage(`{"zone_id":"forest"}`)})
	if status != http.StatusServiceUnavailable {
		t.Fatalf("missing admission gateway status=%d", status)
	}
}

func TestWorldJoinRoutesStrictV1EnvelopeAndIgnoresSpoofedIdentity(t *testing.T) {
	playerID := uuid.New()
	client := &Client{playerID: playerID, controllerID: uuid.New(), done: make(chan struct{}), commands: make(chan WSMessage, 1), send: make(chan []byte, 1)}
	raw := []byte(`{"v":1,"type":"world.join","request_id":"join-1","payload":{"zone_id":"forest","map_version":"map-1","content_version":"content-1"}}`)
	client.handleMessage(WSMessage{Type: protocol.TypeJoin, RequestID: "join-1", Raw: raw})
	queued := <-client.commands
	if queued.Type != protocol.TypeJoin || queued.Channel != "world.join" || queued.RequestID != "join-1" {
		t.Fatalf("unexpected queued join message: %+v", queued)
	}

	spoof := []byte(`{"v":1,"type":"world.join","request_id":"join-2","payload":{"zone_id":"forest","map_version":"map-1","content_version":"content-1","player_id":"` + uuid.NewString() + `"}}`)
	client.handleMessage(WSMessage{Type: protocol.TypeJoin, RequestID: "join-2", Raw: spoof})
	var reject protocol.Envelope
	if err := json.Unmarshal(<-client.send, &reject); err != nil {
		t.Fatal(err)
	}
	if reject.Type != protocol.TypeReject || reject.RequestID != "join-2" || len(client.commands) != 0 {
		t.Fatalf("spoofed or malformed join was not rejected: envelope=%+v queued=%d", reject, len(client.commands))
	}
}

func TestWorldInputUsesStrictV1EnvelopeAndBoundedQueue(t *testing.T) {
	client := &Client{commands: make(chan WSMessage, 1), worldInputs: make(chan WSMessage, 1), send: make(chan []byte, 2)}
	raw := []byte(`{"v":1,"type":"world.input","request_id":"input-1","payload":{"seq":1,"move":{"x":1,"y":0}}}`)
	client.handleMessage(WSMessage{Type: protocol.TypeInput, RequestID: "input-1", Raw: raw})
	queued := <-client.worldInputs
	if queued.Channel != protocol.TypeInput || queued.RequestID != "input-1" {
		t.Fatalf("unexpected input queue item: %+v", queued)
	}

	client.worldInputs <- WSMessage{Type: protocol.TypeInput}
	client.handleMessage(WSMessage{Type: protocol.TypeInput, RequestID: "input-2", Raw: raw})
	var reject protocol.Envelope
	if err := json.Unmarshal(<-client.send, &reject); err != nil {
		t.Fatal(err)
	}
	if reject.Type != protocol.TypeReject || reject.RequestID != "input-2" || len(client.worldInputs) != 1 {
		t.Fatalf("full input queue was not rejected: envelope=%+v queued=%d", reject, len(client.worldInputs))
	}
}

func TestRealtimeInputUsesAuthenticatedConnectionIdentity(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	admission := &fakeRealtimeAdmission{}
	client := &Client{playerID: playerID, controllerID: controllerID, realtimeAdmission: admission, done: make(chan struct{})}
	message := WSMessage{Raw: []byte(`{"v":1,"type":"world.input","payload":{"seq":7,"move":{"x":0,"y":1}}}`)}
	status, data := client.handleRealtimeInput(message)
	if status != http.StatusAccepted || admission.inputPlayer != playerID || admission.inputController != controllerID || admission.inputValue.Seq != 7 || admission.inputValue.Move == nil || admission.inputValue.Move.Y != 1 {
		t.Fatalf("input identity or payload mismatch: status=%d admission=%+v body=%s", status, admission, data)
	}
}

func TestRealtimeInputRateLimitIsBoundedPerConnection(t *testing.T) {
	client := &Client{}
	now := time.Now()
	for i := 0; i < realtimeInputsPerSecond; i++ {
		if !client.allowRealtimeInput(now) {
			t.Fatalf("input %d was unexpectedly limited", i+1)
		}
	}
	if client.allowRealtimeInput(now) {
		t.Fatal("input over the per-second limit was allowed")
	}
	if !client.allowRealtimeInput(now.Add(time.Second)) {
		t.Fatal("input remained limited after the one-second window")
	}
}

func TestWorldSnapshotsCoalesceToNewestBaselineWithOwnInputAck(t *testing.T) {
	playerID := uuid.New()
	client := &Client{playerID: playerID, done: make(chan struct{}), snapshots: make(chan []byte, 1)}
	client.sendWorldSnapshot(sim.Snapshot{Tick: 10, Entities: []sim.Entity{{ID: "other"}}})
	client.sendWorldSnapshot(sim.Snapshot{Tick: 11, Entities: []sim.Entity{{ID: sim.EntityID(playerID.String()), CharacterID: sim.CharacterID(playerID.String()), AckSeq: 9}}})
	var envelope protocol.Envelope
	if err := json.Unmarshal(<-client.snapshots, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Type != protocol.TypeSnapshot {
		t.Fatalf("snapshot envelope type=%q", envelope.Type)
	}
	snapshot, err := protocol.DecodePayload[protocol.Snapshot](envelope)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Tick != 11 || snapshot.IsBaseline || snapshot.BaselineID != 1 || snapshot.BaseTick != 10 || snapshot.AckSeq != 9 || len(snapshot.Entities) != 1 || snapshot.Entities[0].EntityID != playerID.String() || len(snapshot.Removed) != 1 || snapshot.Removed[0] != "other" {
		t.Fatalf("snapshot delta was not coalesced or ack was lost: %+v", snapshot)
	}
}

func TestSnapshotDeltaRemovalsAndPeriodicBaseline(t *testing.T) {
	client := &Client{playerID: uuid.New()}
	playerID := client.playerID.String()
	base := client.buildSnapshotDelta(sim.Snapshot{Tick: 10, Entities: []sim.Entity{
		{ID: sim.EntityID(playerID), CharacterID: sim.CharacterID(playerID), Transform: sim.Transform{Position: sim.Vec2{X: 1}}},
		{ID: "mob-1", Transform: sim.Transform{Position: sim.Vec2{X: 2}}},
	}})
	if !base.IsBaseline || base.BaselineID != 1 || len(base.Entities) != 2 {
		t.Fatalf("unexpected initial baseline: %+v", base)
	}
	delta := client.buildSnapshotDelta(sim.Snapshot{Tick: 11, Entities: []sim.Entity{
		{ID: sim.EntityID(playerID), CharacterID: sim.CharacterID(playerID), Revision: 1, Transform: sim.Transform{Position: sim.Vec2{X: 3}}},
	}})
	if delta.IsBaseline || delta.BaseTick != 10 || len(delta.Entities) != 1 || delta.Entities[0].EntityID != playerID || len(delta.Removed) != 1 || delta.Removed[0] != "mob-1" {
		t.Fatalf("unexpected AOI delta/removal: %+v", delta)
	}
	baseline := client.buildSnapshotDelta(sim.Snapshot{Tick: 50, Entities: []sim.Entity{
		{ID: sim.EntityID(playerID), CharacterID: sim.CharacterID(playerID), Transform: sim.Transform{Position: sim.Vec2{X: 1}}},
	}})
	if !baseline.IsBaseline || baseline.BaselineID != 2 || baseline.BaseTick != 50 || len(baseline.Removed) != 0 {
		t.Fatalf("periodic full baseline was not refreshed: %+v", baseline)
	}
}

func TestSnapshotStreamContinuesFromWelcomeBaselineID(t *testing.T) {
	client := &Client{playerID: uuid.New()}
	baseline := protocol.Snapshot{
		Tick: 25, BaselineID: 4, BaseTick: 25, IsBaseline: true,
		Entities: []protocol.EntityState{{EntityID: client.playerID.String(), Revision: 1, X: 1, Y: 2, HP: 100}},
	}
	encoded, err := json.Marshal(baseline)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.initializeSnapshotBaseline(protocol.Welcome{BaselineID: 4, ServerTick: 25, Snapshot: encoded}); err != nil {
		t.Fatal(err)
	}
	next := client.buildSnapshotDelta(sim.Snapshot{Tick: 26, Entities: []sim.Entity{{
		ID: sim.EntityID(client.playerID.String()), CharacterID: sim.CharacterID(client.playerID.String()),
		Revision: 2, Transform: sim.Transform{Position: sim.Vec2{X: 3, Y: 2}}, Vitals: sim.Vitals{HP: 100, MaxHP: 100},
	}}})
	if next.IsBaseline || next.BaselineID != 4 || next.BaseTick != 25 || len(next.Entities) != 1 {
		t.Fatalf("snapshot stream did not continue from welcome baseline: %+v", next)
	}
}

func TestRealtimeInputDispatchesMoveTarget(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	admission := &fakeRealtimeAdmission{}
	client := &Client{playerID: playerID, controllerID: controllerID, realtimeAdmission: admission, done: make(chan struct{})}
	raw := []byte(`{"v":1,"type":"world.move_target","request_id":"path-1","payload":{"seq":3,"x":12.5,"y":8}}`)

	status, _ := client.handleRealtimeInput(WSMessage{Raw: raw})
	if status != http.StatusAccepted {
		t.Fatalf("status=%d, want accepted", status)
	}
	if admission.targetPlayer != playerID || admission.targetController != controllerID || admission.targetValue.Seq != 3 || admission.targetValue.X != 12.5 || admission.targetValue.Y != 8 {
		t.Fatalf("move target was not dispatched with authenticated identity: %+v", admission)
	}
}

func TestRealtimeInputDispatchesCast(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	admission := &fakeRealtimeAdmission{}
	client := &Client{playerID: playerID, controllerID: controllerID, realtimeAdmission: admission, done: make(chan struct{})}
	raw := []byte(`{"v":1,"type":"world.cast","request_id":"cast-1","payload":{"seq":4,"slot":2,"mode":"DIRECTION","aim_x":1,"aim_y":0}}`)

	status, _ := client.handleRealtimeInput(WSMessage{Raw: raw})
	if status != http.StatusAccepted {
		t.Fatalf("status=%d, want accepted", status)
	}
	if admission.castPlayer != playerID || admission.castController != controllerID || admission.castValue.Slot != 2 || admission.castValue.Mode != "DIRECTION" || admission.castValue.AimX != 1 {
		t.Fatalf("cast was not dispatched with authenticated identity: %+v", admission)
	}
}

func TestWebSocketCastReachesAuthoritativeZoneResolution(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "ws-cast-test", RulesetVersion: "v1"}
	manager, err := realtimeworld.NewLazyManagerWithMaps([]sim.ZoneKey{key}, 8, 2, map[sim.ZoneKey]realtimeworld.ZoneMap{
		key: {Monsters: []realtimeworld.MonsterSpawn{{
			TemplateID: "test_mob", X: 3, Y: 0, MaxHP: 200, AttackDamage: 1,
			Speed: 1, Radius: 0.5, AggroRadius: 1, LeashRadius: 2, AttackRange: 0.5,
			PatrolRadius: 0, AttackCooldownTicks: 20, RespawnTicks: 100,
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := manager.Stop(ctx); err != nil {
			t.Errorf("stop manager: %v", err)
		}
	})

	profiles, err := realtimeeffects.LoadProfiles("../../data/mmo/gu/transpiled_profiles.json")
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := realtimeeffects.CompileCatalog(profiles, nil, realtimeeffects.DefaultMVP())
	if err != nil {
		t.Fatal(err)
	}
	admission := gateway.NewAdmission(config.Config{
		Enabled: true, PhaserClientEnabled: true, MaxPlayersPerZone: 2,
		MapVersion: "map-1", ContentVersion: "content-1", ZoneIDs: map[string]struct{}{"forest": {}},
	}, manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{
			MapID: "forest", RegionID: "forest", X: 0, Y: 0, HP: 100, BaseHP: 100,
			Rank: 3, CurrentMana: 20, MaxMana: 80, BaseAttack: 30,
			EquippedGu: []handlers.RealtimeEquippedGu{{
				InstanceID: "inst-frost", GuID: "GU_WHITE_FROST", Rank: 3, Hunger: 10, Usable: true, ManaCost: 2.5, DamageScale: 1.5,
			}},
		}, nil
	})
	admission.SetLoadoutCatalog(catalog)
	if _, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{
		ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1",
	}); err != nil {
		t.Fatal(err)
	}

	client := &Client{
		playerID: playerID, controllerID: controllerID, realtimeAdmission: admission,
		done: make(chan struct{}), worldInputs: make(chan WSMessage, 1), send: make(chan []byte, 1),
	}
	raw := []byte(`{"v":1,"type":"world.cast","request_id":"cast-e2e","payload":{"seq":1,"slot":1,"mode":"TARGET","target_id":"m:0"}}`)
	client.handleMessage(WSMessage{Type: protocol.TypeCast, RequestID: "cast-e2e", Raw: raw})
	queued := <-client.worldInputs
	status, body := client.handleRealtimeInput(queued)
	if status != http.StatusAccepted || string(body) != `{"accepted":true}` {
		t.Fatalf("cast ingress status=%d body=%s", status, body)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := manager.Snapshot(key)
		if err != nil {
			t.Fatal(err)
		}
		for _, entity := range snapshot.Entities {
			if entity.Kind != sim.EntityMonster {
				continue
			}
			for _, status := range entity.Combat.Statuses {
				if status.ID == "FREEZE" {
					return
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("authoritative zone did not apply FREEZE from the WebSocket cast intent")
}
