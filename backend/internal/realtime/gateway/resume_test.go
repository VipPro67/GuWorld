package gateway

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

func newResumeTest(t *testing.T) (*Admission, uuid.UUID, protocol.JoinRequest, *int) {
	t.Helper()
	playerID := uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 2)
	loaderCalls := 0
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		loaderCalls++
		return handlers.RealtimeJoinState{
			MapID: "forest", RegionID: "forest-clearing", X: 17, Y: 29,
			HP: 70, BaseHP: 100, Rank: 3, CurrentMana: 42, MaxMana: 500,
			BaseAttack: 36, Defense: 7, MovementSpeed: 5.6,
		}, nil
	})
	request := protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}
	return admission, playerID, request, &loaderCalls
}

func welcomePosition(t *testing.T, value any) (entityID string, x, y float64) {
	t.Helper()
	welcome, ok := value.(protocol.Welcome)
	if !ok {
		t.Fatalf("expected protocol.Welcome, got %T", value)
	}
	var snapshot protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &snapshot); err != nil {
		t.Fatalf("decode welcome snapshot: %v", err)
	}
	if len(snapshot.Entities) == 0 {
		t.Fatal("welcome snapshot has no entities")
	}
	return snapshot.Entities[0].EntityID, snapshot.Entities[0].X, snapshot.Entities[0].Y
}

func TestSuspendedSessionResumesSameTabWithoutReclaim(t *testing.T) {
	admission, playerID, request, loaderCalls := newResumeTest(t)
	controllerA, controllerB := uuid.New(), uuid.New()

	first, err := admission.JoinWithTab(context.Background(), playerID, controllerA, "tab-a", request)
	if err != nil {
		t.Fatalf("join A: %v", err)
	}
	firstID, firstX, firstY := welcomePosition(t, first)
	if *loaderCalls != 1 {
		t.Fatalf("loader calls = %d, want 1", *loaderCalls)
	}

	admission.Disconnect(playerID, controllerA)
	if admission.SuspendedCount() != 1 {
		t.Fatalf("suspended = %d, want 1", admission.SuspendedCount())
	}
	if admission.ActiveSessionCount() != 1 {
		t.Fatalf("active sessions should include suspended: %d", admission.ActiveSessionCount())
	}

	second, err := admission.JoinWithTab(context.Background(), playerID, controllerB, "tab-a", request)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	secondID, secondX, secondY := welcomePosition(t, second)
	if secondID != firstID || secondX != firstX || secondY != firstY {
		t.Fatalf("resume moved the character: first=%s@%v,%v second=%s@%v,%v", firstID, firstX, firstY, secondID, secondX, secondY)
	}
	if *loaderCalls != 1 {
		t.Fatalf("resume must not reload durable state, calls = %d", *loaderCalls)
	}
	if admission.SuspendedCount() != 0 {
		t.Fatalf("suspended after resume = %d", admission.SuspendedCount())
	}
}

func TestSuspendedSessionExpiresAndReleasesLease(t *testing.T) {
	admission, playerID, request, _ := newResumeTest(t)
	controllerID := uuid.New()
	if _, err := admission.JoinWithTab(context.Background(), playerID, controllerID, "tab-a", request); err != nil {
		t.Fatalf("join: %v", err)
	}
	admission.Disconnect(playerID, controllerID)

	key := suspendKey{playerID: playerID, tabID: "tab-a"}
	admission.mu.Lock()
	held := admission.suspended[key]
	held.expiresAt = time.Now().Add(-time.Second)
	admission.suspended[key] = held
	admission.mu.Unlock()

	if drained := admission.SweepSuspended(); drained != 1 {
		t.Fatalf("swept = %d, want 1", drained)
	}
	if admission.SuspendedCount() != 0 {
		t.Fatalf("suspended after sweep = %d", admission.SuspendedCount())
	}
	if admission.manager.OwnsCharacter(sim.CharacterID(playerID.String())) {
		t.Fatal("expired session still owns the character")
	}
}

func TestDifferentTabTakesOverSuspendedSession(t *testing.T) {
	admission, playerID, request, loaderCalls := newResumeTest(t)
	controllerA, controllerB := uuid.New(), uuid.New()
	first, err := admission.JoinWithTab(context.Background(), playerID, controllerA, "tab-a", request)
	if err != nil {
		t.Fatalf("join A: %v", err)
	}
	firstID, _, _ := welcomePosition(t, first)
	admission.Disconnect(playerID, controllerA)

	second, err := admission.JoinWithTab(context.Background(), playerID, controllerB, "tab-b", request)
	if err != nil {
		t.Fatalf("takeover join: %v", err)
	}
	secondID, _, _ := welcomePosition(t, second)
	if secondID != firstID {
		t.Fatalf("takeover changed entity: %s -> %s", firstID, secondID)
	}
	if *loaderCalls != 1 {
		t.Fatalf("takeover must reuse the live entity, calls = %d", *loaderCalls)
	}
	if admission.SuspendedCount() != 0 {
		t.Fatalf("suspended after takeover = %d", admission.SuspendedCount())
	}
}
