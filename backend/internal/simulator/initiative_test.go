package simulator

import (
	"encoding/json"
	"math"
	"testing"
)

func TestInitiativeQueueFastActorGetsExtraActivation(t *testing.T) {
	q := NewInitiativeQueue([]*CombatantAuto{
		{ID: "slow", EffectiveSpeed: 1},
		{ID: "fast", EffectiveSpeed: 9},
	})
	got := make([]string, 0, 4)
	for range 4 {
		actor, _, ok := q.Next()
		if !ok {
			t.Fatal("queue unexpectedly empty")
		}
		got = append(got, actor)
		q.Complete(actor)
	}
	for _, actor := range got[:2] {
		if actor != "fast" {
			t.Fatalf("first two activations = %v; fast actor should act twice", got)
		}
	}
}

func TestInitiativeQueueSpeedChangePreservesProgress(t *testing.T) {
	q := NewInitiativeQueue([]*CombatantAuto{{ID: "a", EffectiveSpeed: 4}})
	q.Time = q.Entries[0].ReadyAt / 2
	q.SetSpeed("a", 16)
	want := q.Time + InitiativeInterval(16)/2
	if math.Abs(q.Entries[0].ReadyAt-want) > 1e-9 {
		t.Fatalf("ready at %g, want %g", q.Entries[0].ReadyAt, want)
	}
}

func TestInitiativeQueueStableTieAfterSerialization(t *testing.T) {
	q := NewInitiativeQueue([]*CombatantAuto{
		{ID: "first", EffectiveSpeed: 4},
		{ID: "second", EffectiveSpeed: 4},
	})
	data, err := json.Marshal(q)
	if err != nil {
		t.Fatal(err)
	}
	var restored InitiativeQueue
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	actor, _, ok := restored.Next()
	if !ok || actor != "first" {
		t.Fatalf("first activation = %q, ok=%t", actor, ok)
	}
	restored.Complete(actor)
	actor, _, ok = restored.Next()
	if !ok || actor != "second" {
		t.Fatalf("second activation = %q, ok=%t", actor, ok)
	}
}

func TestInitiativeQueuePrioritizesOpeningActorOnlyOnce(t *testing.T) {
	q := NewInitiativeQueue([]*CombatantAuto{
		{ID: "fast", EffectiveSpeed: 20},
		{ID: "priority", EffectiveSpeed: 5},
	})
	actorID, _, ok := q.Next()
	if !ok || actorID != "fast" {
		t.Fatalf("initial speed order = %q, want fast", actorID)
	}
	if !q.Prioritize("priority") {
		t.Fatal("expected opening-priority actor to be present")
	}
	actorID, _, ok = q.Next()
	if !ok || actorID != "priority" {
		t.Fatalf("opening priority actor = %q, want priority", actorID)
	}
	q.Complete(actorID)
	actorID, _, ok = q.Next()
	if !ok || actorID != "fast" {
		t.Fatalf("subsequent speed order = %q, want fast", actorID)
	}
}
