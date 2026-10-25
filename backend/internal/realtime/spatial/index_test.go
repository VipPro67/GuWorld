package spatial

import (
	"reflect"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

func TestQueryReturnsOnlyNearbyEntitiesInStableOrder(t *testing.T) {
	index := NewIndex(16)
	index.Rebuild([]sim.Entity{
		{ID: "z", Transform: sim.Transform{Position: sim.Vec2{X: 33, Y: 0}}},
		{ID: "b", Transform: sim.Transform{Position: sim.Vec2{X: -4, Y: 3}}},
		{ID: "a", Transform: sim.Transform{Position: sim.Vec2{X: 32, Y: 0}}},
		{ID: "far", Transform: sim.Transform{Position: sim.Vec2{X: 200, Y: 0}}},
	})
	visible := index.Query(sim.Vec2{}, 32, "owner")
	got := make([]sim.EntityID, 0, len(visible))
	for _, entity := range visible {
		got = append(got, entity.ID)
	}
	want := []sim.EntityID{"a", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("visible ids=%v, want %v", got, want)
	}
}

func TestQueryAlwaysIncludesOwnerAndRebuildRemovesOldCells(t *testing.T) {
	index := NewIndex(8)
	index.Rebuild([]sim.Entity{{ID: "owner", Transform: sim.Transform{Position: sim.Vec2{X: 100}}}})
	if got := index.Query(sim.Vec2{}, 0, "owner"); len(got) != 1 || got[0].ID != "owner" {
		t.Fatalf("owner missing from query: %+v", got)
	}
	index.Rebuild(nil)
	if got := index.Query(sim.Vec2{}, 10, "owner"); len(got) != 0 {
		t.Fatalf("rebuild retained stale entities: %+v", got)
	}
}
