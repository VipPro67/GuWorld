package ws

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

func TestSnapshotDeltaTracker_EntityStateEqual(t *testing.T) {
	base := protocol.EntityState{
		EntityID:  "player-1",
		Revision:  10,
		Kind:      "player",
		PartyID:   "party-a",
		X:         12.5,
		Y:         20.0,
		HP:        100,
		MaxHP:     100,
		Shield:    25,
		Mana:      50,
		MaxMana:   100,
		Statuses:  []string{"HASTE", "SHIELD"},
		Cooldowns: []uint64{10, 20},
	}

	identical := base
	identical.Statuses = []string{"HASTE", "SHIELD"}
	identical.Cooldowns = []uint64{10, 20}
	if !entityStateEqual(base, identical) {
		t.Fatalf("expected identical entity states to be equal")
	}

	diffRevision := base
	diffRevision.Revision = 11
	if entityStateEqual(base, diffRevision) {
		t.Fatalf("expected different revision to not be equal")
	}

	diffPos := base
	diffPos.X = 13.0
	if entityStateEqual(base, diffPos) {
		t.Fatalf("expected different position to not be equal")
	}

	diffStatuses := base
	diffStatuses.Statuses = []string{"HASTE"}
	if entityStateEqual(base, diffStatuses) {
		t.Fatalf("expected different statuses length to not be equal")
	}

	diffCooldowns := base
	diffCooldowns.Cooldowns = []uint64{10, 25}
	if entityStateEqual(base, diffCooldowns) {
		t.Fatalf("expected different cooldowns to not be equal")
	}
}

func TestSnapshotDeltaTracker_StealthFiltering(t *testing.T) {
	tracker := NewSnapshotDeltaTracker()
	myPlayerID := uuid.New().String()
	partyMemberID := uuid.New().String()
	strangerID := uuid.New().String()

	snapshot := sim.Snapshot{
		Tick: 1,
		Entities: []sim.Entity{
			{
				ID:          sim.EntityID(myPlayerID),
				CharacterID: sim.CharacterID(myPlayerID),
				PartyID:     "team-alpha",
				Kind:        sim.EntityPlayer,
				Transform:   sim.Transform{Position: sim.Vec2{X: 1, Y: 1}},
			},
			{
				ID:          sim.EntityID(partyMemberID),
				CharacterID: sim.CharacterID(partyMemberID),
				PartyID:     "team-alpha",
				Kind:        sim.EntityPlayer,
				Combat: sim.CombatState{
					Statuses: []sim.StatusEffect{{ID: "STEALTH"}},
				},
				Transform: sim.Transform{Position: sim.Vec2{X: 2, Y: 2}},
			},
			{
				ID:          sim.EntityID(strangerID),
				CharacterID: sim.CharacterID(strangerID),
				PartyID:     "team-bravo",
				Kind:        sim.EntityPlayer,
				Combat: sim.CombatState{
					Statuses: []sim.StatusEffect{{ID: "STEALTH"}},
				},
				Transform: sim.Transform{Position: sim.Vec2{X: 3, Y: 3}},
			},
		},
	}

	baseline := tracker.BuildDelta(snapshot, myPlayerID)
	if !baseline.IsBaseline {
		t.Fatalf("expected first snapshot to be baseline")
	}

	foundStranger := false
	foundPartyMember := false
	foundSelf := false

	for _, e := range baseline.Entities {
		if e.EntityID == strangerID {
			foundStranger = true
		}
		if e.EntityID == partyMemberID {
			foundPartyMember = true
		}
		if e.EntityID == myPlayerID {
			foundSelf = true
		}
	}

	if foundStranger {
		t.Fatalf("stealthed stranger should not be visible in client snapshot")
	}
	if !foundPartyMember {
		t.Fatalf("stealthed party member should be visible in client snapshot")
	}
	if !foundSelf {
		t.Fatalf("self should always be visible in client snapshot")
	}
}

func TestSnapshotDeltaTracker_InitializeValidation(t *testing.T) {
	tracker := NewSnapshotDeltaTracker()

	// Invalid JSON
	err := tracker.Initialize(protocol.Welcome{Snapshot: []byte(`{invalid`)})
	if err == nil {
		t.Fatalf("expected error on invalid JSON snapshot")
	}

	// Mismatched baseline ID
	raw, _ := json.Marshal(protocol.Snapshot{
		IsBaseline: true,
		BaselineID: 5,
		Tick:       100,
	})
	err = tracker.Initialize(protocol.Welcome{
		BaselineID: 6,
		ServerTick: 100,
		Snapshot:   raw,
	})
	if err == nil {
		t.Fatalf("expected error on mismatched baseline ID")
	}

	// Valid welcome
	validRaw, _ := json.Marshal(protocol.Snapshot{
		IsBaseline: true,
		BaselineID: 5,
		Tick:       100,
		Entities: []protocol.EntityState{
			{EntityID: "p1", Revision: 1, X: 10, Y: 10},
		},
	})
	err = tracker.Initialize(protocol.Welcome{
		BaselineID: 5,
		ServerTick: 100,
		Snapshot:   validRaw,
	})
	if err != nil {
		t.Fatalf("expected success on valid handshake: %v", err)
	}
	if tracker.BaselineID() != 5 || tracker.BaseTick() != 100 {
		t.Fatalf("tracker state not initialized correctly: %d, %d", tracker.BaselineID(), tracker.BaseTick())
	}
}

type recordingRealtimeAdmission struct {
	mu           sync.Mutex
	inputsCalled int
}

func (r *recordingRealtimeAdmission) Join(_ context.Context, _, _ uuid.UUID, _ protocol.JoinRequest) (any, error) {
	return map[string]string{"type": "world.welcome"}, nil
}
func (r *recordingRealtimeAdmission) Leave(_, _ uuid.UUID) error { return nil }
func (r *recordingRealtimeAdmission) Disconnect(_, _ uuid.UUID)  {}
func (r *recordingRealtimeAdmission) Input(_, _ uuid.UUID, _ protocol.Input) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inputsCalled++
	return nil
}
func (r *recordingRealtimeAdmission) MoveTarget(_, _ uuid.UUID, _ protocol.MoveTarget) error {
	return nil
}
func (r *recordingRealtimeAdmission) Cast(_, _ uuid.UUID, _ protocol.Cast) error { return nil }
func (r *recordingRealtimeAdmission) BasicAttack(_, _ uuid.UUID, _ protocol.Attack) error {
	return nil
}
func (r *recordingRealtimeAdmission) TransferAtEdge(_, _ uuid.UUID, _ sim.Vec2) (protocol.Transfer, bool, error) {
	return protocol.Transfer{}, false, nil
}
func (r *recordingRealtimeAdmission) BeginTeleport(_ uuid.UUID) (bool, error) { return false, nil }
func (r *recordingRealtimeAdmission) AbortTeleport(_ uuid.UUID)               {}
func (r *recordingRealtimeAdmission) FinishTeleport(_ uuid.UUID)              {}
func (r *recordingRealtimeAdmission) PlayerLocation(_ uuid.UUID) (string, int, int, bool) {
	return "", 0, 0, false
}
func (r *recordingRealtimeAdmission) SubscribeSnapshots(_, _ uuid.UUID) (<-chan sim.Snapshot, func(), error) {
	return make(chan sim.Snapshot), func() {}, nil
}

func TestRealtimeInputPumpDecoupledFromCommands(t *testing.T) {
	playerID := uuid.New()
	controllerID := uuid.New()
	admission := &recordingRealtimeAdmission{}

	client := &Client{
		playerID:          playerID,
		controllerID:      controllerID,
		realtimeAdmission: admission,
		commands:          make(chan WSMessage, 10),
		worldInputs:       make(chan WSMessage, 10),
		send:              make(chan []byte, 10),
		done:              make(chan struct{}),
	}

	go client.commandPump()
	go client.realtimeInputPump()
	defer client.Close()

	// Feed input into worldInputs
	client.worldInputs <- WSMessage{
		Type: protocol.TypeInput,
		Raw:  []byte(`{"v":1,"type":"world.input","payload":{"seq":1,"move":{"x":1,"y":0}}}`),
	}

	// Wait for input to be consumed by realtimeInputPump
	timeout := time.After(500 * time.Millisecond)
	for {
		admission.mu.Lock()
		count := admission.inputsCalled
		admission.mu.Unlock()
		if count > 0 {
			break
		}
		select {
		case <-timeout:
			t.Fatalf("realtime input was not processed by isolated realtimeInputPump")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}
