package sim

import (
	"math"
	"testing"
)

func FuzzSubmitInputNeverPanics(f *testing.F) {
	f.Add("player", uint64(1), 0.5, 0.5)
	f.Add("", uint64(0), math.NaN(), 0.0)
	f.Add("player", uint64(2), 1e300, -1e300)
	f.Add("player", uint64(3), 1.01, 1.01)
	f.Fuzz(func(t *testing.T, entityID string, seq uint64, x, y float64) {
		w, err := NewWorld("fuzz", 8)
		if err != nil {
			t.Fatal(err)
		}
		entity := newPlayer("player")
		if err := w.AddEntity(entity); err != nil {
			t.Fatal(err)
		}
		_ = w.SubmitInput(InputCommand{EntityID: EntityID(entityID), Seq: seq, Move: Vec2{X: x, Y: y}})
		w.Step()
	})
}

func FuzzMoveTargetNeverPanics(f *testing.F) {
	f.Add("player", uint64(1), 5.0, 5.0)
	f.Add("", uint64(0), math.Inf(1), 0.0)
	f.Add("player", uint64(2), 1e6, -1e6)
	f.Fuzz(func(t *testing.T, entityID string, seq uint64, x, y float64) {
		w, err := NewWorld("fuzz", 8)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.AddEntity(newPlayer("player")); err != nil {
			t.Fatal(err)
		}
		_ = w.SubmitMoveTarget(InputCommand{EntityID: EntityID(entityID), Seq: seq, Target: &Vec2{X: x, Y: y}})
		w.Step()
	})
}
