package handlers

import (
	"testing"
	"time"
)

func TestPositionIndexNearbyIsCoordinateScoped(t *testing.T) {
	idx := newPositionIndex(32)
	now := time.Now().UnixMilli()
	idx.Set(WorldPosEntry{ID: "same", Region: "north", X: 10, Y: 10, TS: now})
	idx.Set(WorldPosEntry{ID: "border-neighbor", Region: "south", X: 11, Y: 11, TS: now})
	idx.Set(WorldPosEntry{ID: "far-away", Region: "south", X: 500, Y: 500, TS: now})

	got := idx.Nearby(10, 10, 10, "")
	if len(got) != 2 {
		t.Fatalf("Nearby(10, 10, 10) = %#v, want 2 players within range across regions", got)
	}
}

func TestPositionIndexSetMovesPlayerBetweenCoordinates(t *testing.T) {
	idx := newPositionIndex(32)
	now := time.Now().UnixMilli()
	idx.Set(WorldPosEntry{ID: "player", Region: "north", X: 10, Y: 10, TS: now})
	idx.Set(WorldPosEntry{ID: "player", Region: "south", X: 200, Y: 200, TS: now})

	if got := idx.Nearby(10, 10, 10, ""); len(got) != 0 {
		t.Fatalf("old coordinates still contain moved player: %#v", got)
	}
	if got := idx.Nearby(200, 200, 10, ""); len(got) != 1 || got[0].ID != "player" {
		t.Fatalf("new coordinates = %#v, want moved player", got)
	}
}
