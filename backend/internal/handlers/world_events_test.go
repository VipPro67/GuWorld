package handlers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

type worldPositionBroadcast struct {
	event   WorldPosEvent
	targets map[uuid.UUID]struct{}
}

func captureWorldPositionBroadcasts(t *testing.T) *[]worldPositionBroadcast {
	t.Helper()
	previousIndex, previousBroadcaster := worldPosIndex, WorldPosBroadcaster
	worldPosIndex = newPositionIndex(WorldPosIndexBucket)
	broadcasts := make([]worldPositionBroadcast, 0, 2)
	WorldPosBroadcaster = func(_ string, payload []byte, targets map[uuid.UUID]struct{}) {
		var event WorldPosEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Errorf("decode world position event: %v", err)
			return
		}
		copyTargets := make(map[uuid.UUID]struct{}, len(targets))
		for target := range targets {
			copyTargets[target] = struct{}{}
		}
		broadcasts = append(broadcasts, worldPositionBroadcast{event: event, targets: copyTargets})
	}
	t.Cleanup(func() {
		worldPosIndex = previousIndex
		WorldPosBroadcaster = previousBroadcaster
	})
	return &broadcasts
}

func TestBroadcastWorldPosIncludesOldAndNewViewportTargets(t *testing.T) {
	broadcasts := captureWorldPositionBroadcasts(t)
	now := time.Now().UnixMilli()
	moverID, oldViewerID, newViewerID := uuid.New(), uuid.New(), uuid.New()
	worldPosIndex.Set(WorldPosEntry{ID: moverID.String(), X: 0, Y: 0, Region: "north", MapID: "world_map", TS: now})
	worldPosIndex.Set(WorldPosEntry{ID: oldViewerID.String(), X: 10, Y: 10, Region: "north", MapID: "world_map", TS: now})
	worldPosIndex.Set(WorldPosEntry{ID: newViewerID.String(), X: 1000, Y: 1000, Region: "north", MapID: "world_map", TS: now})

	broadcastWorldPos(WorldPosEvent{
		ID: moverID.String(), X: 1000, Y: 1000, Region: "north", MapID: "world_map",
		TS: now + 1, ScopeResolved: true,
	})

	if len(*broadcasts) != 1 {
		t.Fatalf("broadcast count = %d, want 1", len(*broadcasts))
	}
	targets := (*broadcasts)[0].targets
	if _, ok := targets[oldViewerID]; !ok {
		t.Fatal("old viewport player did not receive the mover's new position")
	}
	if _, ok := targets[newViewerID]; !ok {
		t.Fatal("new viewport player did not receive the mover's new position")
	}
}

func TestBroadcastWorldPosNotifiesPreviousScopeOnTransition(t *testing.T) {
	broadcasts := captureWorldPositionBroadcasts(t)
	now := time.Now().UnixMilli()
	moverID, oldViewerID, newViewerID := uuid.New(), uuid.New(), uuid.New()
	worldPosIndex.Set(WorldPosEntry{ID: moverID.String(), X: 0, Y: 0, Region: "north", MapID: "world_map", TS: now})
	worldPosIndex.Set(WorldPosEntry{ID: oldViewerID.String(), X: 10, Y: 10, Region: "north", MapID: "world_map", TS: now})
	worldPosIndex.Set(WorldPosEntry{ID: newViewerID.String(), X: 1000, Y: 1000, Region: "instance", MapID: "private_map", InstanceID: "instance-1", TS: now})

	broadcastWorldPos(WorldPosEvent{
		ID: moverID.String(), X: 1000, Y: 1000, Region: "instance", MapID: "private_map", InstanceID: "instance-1",
		TS: now + 1, ScopeResolved: true,
	})

	if len(*broadcasts) != 2 {
		t.Fatalf("broadcast count = %d, want leave and enter events", len(*broadcasts))
	}
	left, moved := (*broadcasts)[0], (*broadcasts)[1]
	if left.event.Action != "left" || left.event.MapID != "world_map" {
		t.Fatalf("old-scope event = %+v, want leave event for world_map", left.event)
	}
	if _, ok := left.targets[oldViewerID]; !ok {
		t.Fatal("old-scope viewer did not receive leave event")
	}
	if moved.event.Action != "" || moved.event.MapID != "private_map" || moved.event.InstanceID != "instance-1" {
		t.Fatalf("new-scope event = %+v, want movement in private_map/instance-1", moved.event)
	}
	if _, ok := moved.targets[newViewerID]; !ok {
		t.Fatal("new-scope viewer did not receive movement event")
	}
}
