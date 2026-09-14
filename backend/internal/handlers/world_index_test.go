package handlers

import (
	"testing"
	"time"
)

func TestPositionIndexNearbyIsRegionScoped(t *testing.T) {
	idx := newPositionIndex(32)
	now := time.Now().UnixMilli()
	idx.Set(WorldPosEntry{ID: "same", Region: "north", X: 10, Y: 10, TS: now})
	idx.Set(WorldPosEntry{ID: "other-region", Region: "south", X: 11, Y: 11, TS: now})

	got := idx.Nearby("north", 10, 10, 10, "")
	if len(got) != 1 || got[0].ID != "same" {
		t.Fatalf("Nearby(north) = %#v, want only north-region player", got)
	}
}

func TestPositionIndexSetMovesPlayerBetweenRegions(t *testing.T) {
	idx := newPositionIndex(32)
	now := time.Now().UnixMilli()
	idx.Set(WorldPosEntry{ID: "player", Region: "north", X: 10, Y: 10, TS: now})
	idx.Set(WorldPosEntry{ID: "player", Region: "south", X: 10, Y: 10, TS: now})

	if got := idx.Nearby("north", 10, 10, 10, ""); len(got) != 0 {
		t.Fatalf("old region still contains moved player: %#v", got)
	}
	if got := idx.Nearby("south", 10, 10, 10, ""); len(got) != 1 || got[0].ID != "player" {
		t.Fatalf("new region = %#v, want moved player", got)
	}
}
