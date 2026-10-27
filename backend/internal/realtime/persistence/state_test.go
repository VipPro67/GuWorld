package persistence

import (
	"math"
	"testing"

	"github.com/google/uuid"
)

func TestStateValidation(t *testing.T) {
	state := State{PlayerID: uuid.New(), ZoneID: "forest", InstanceID: "i", X: 1, Y: 2, HP: 10, SafeX: 1, SafeY: 2}
	if err := state.Validate(); err != nil {
		t.Fatal(err)
	}
	state.HP = -1
	if err := state.Validate(); err == nil {
		t.Fatal("negative HP accepted")
	}
	state.HP = 10
	state.X = math.NaN()
	if err := state.Validate(); err == nil {
		t.Fatal("NaN coordinate accepted")
	}
	state.X = 1
	state.PlayerID = uuid.Nil
	if err := state.Validate(); err == nil {
		t.Fatal("nil player accepted")
	}
	state.PlayerID = uuid.New()
	state.HasLocationRevision = true
	state.LocationRevision = -1
	if err := state.Validate(); err == nil {
		t.Fatal("negative location revision accepted")
	}
}
