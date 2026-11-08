package sim

import (
	"math/rand"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

func newTestMonster(id string, pos Vec2, hp float64) Entity {
	return Entity{
		ID:        EntityID(id),
		Kind:      EntityMonster,
		Transform: Transform{Position: pos},
		Vitals:    Vitals{HP: hp, MaxHP: 100},
		Speed:     3,
		Radius:    0.5,
		Faction:   "monster",
		AI: AIState{
			Active:       true,
			State:        AIIdle,
			Home:         pos,
			AggroRadius:  6,
			LeashRadius:  10,
			AttackRange:  1.2,
			PatrolRadius: 0,
			AttackDamage: 5,
			Rank:         1,
		},
	}
}

func TestCaptureFlow(t *testing.T) {
	world, err := NewWorld("arena", 16)
	if err != nil {
		t.Fatalf("failed to create world: %v", err)
	}

	player := Entity{
		ID:     "p1",
		Kind:   EntityPlayer,
		Vitals: Vitals{HP: 100, MaxHP: 100},
		Combat: CombatState{Mana: 50, MaxMana: 100},
		CaptureGuBonuses: map[string]float64{
			"gu_enslave_wolf": 0.2,
		},
		Transform: Transform{Position: Vec2{X: 10, Y: 10}},
		Speed:     5.0,
		Radius:    0.5,
	}

	monster := newTestMonster("m1", Vec2{X: 12, Y: 10}, 15)
	// Stun monster so it cannot attack and interrupt the player.
	monster.Combat.Statuses = []StatusEffect{{ID: "STUN", ExpireTick: 200}}

	if err := world.AddEntities([]Entity{player, monster}); err != nil {
		t.Fatalf("AddEntities failed: %v", err)
	}

	cmd := CaptureCommand{
		EntityID: "p1",
		Seq:      1,
		TargetID: "m1",
		GuID:     "gu_enslave_wolf",
	}
	if err := world.SubmitCapture(cmd); err != nil {
		t.Fatalf("SubmitCapture failed: %v", err)
	}

	world.Step()

	p := world.entities["p1"]
	if p.Combat.Pending == nil || !p.Combat.Pending.IsCapture {
		t.Fatalf("expected pending capture cast, got nil or not capture")
	}
	if p.Combat.Mana != 30 {
		t.Errorf("expected 20 mana consumed, got %f", p.Combat.Mana)
	}

	// Step through windup (60 ticks)
	var allEvents []combat.Event
	for i := 0; i < 60; i++ {
		stepSnap := world.Step()
		allEvents = append(allEvents, stepSnap.Events...)
	}

	var foundSuccess, foundFailed bool
	for _, ev := range allEvents {
		if ev.Type == combat.EventCaptureSuccess {
			foundSuccess = true
		}
		if ev.Type == combat.EventCaptureFailed {
			foundFailed = true
		}
	}
	if !foundSuccess && !foundFailed {
		t.Errorf("expected capture success or failed event, got events: %+v", allEvents)
	}
}

func TestCaptureCanStartOnHealthyMonsterWithCaptureGu(t *testing.T) {
	world, err := NewWorld("arena", 16)
	if err != nil {
		t.Fatalf("failed to create world: %v", err)
	}

	player := Entity{
		ID:     "p1",
		Kind:   EntityPlayer,
		Vitals: Vitals{HP: 100, MaxHP: 100},
		Combat: CombatState{Mana: 50, MaxMana: 100},
		CaptureGuBonuses: map[string]float64{
			"capture-gu-instance": 0.2,
		},
		Transform: Transform{Position: Vec2{X: 10, Y: 10}},
		Radius:    0.5,
	}
	monster := newTestMonster("m1", Vec2{X: 12, Y: 10}, 100)
	monster.AI.AggroRadius = 1

	if err := world.AddEntities([]Entity{player, monster}); err != nil {
		t.Fatalf("AddEntities failed: %v", err)
	}
	if err := world.SubmitCapture(CaptureCommand{
		EntityID: "p1",
		Seq:      1,
		TargetID: "m1",
		GuID:     "capture-gu-instance",
	}); err != nil {
		t.Fatalf("SubmitCapture failed: %v", err)
	}

	snapshot := world.Step()
	if !hasEvent(snapshot.Events, combat.EventCaptureStarted) {
		t.Fatalf("healthy monster capture was not started: %+v", snapshot.Events)
	}
	if pending := world.entities["p1"].Combat.Pending; pending == nil || !pending.IsCapture {
		t.Fatalf("expected pending capture, got %+v", pending)
	}
}

func TestCaptureRangeMatchesHudLimit(t *testing.T) {
	tests := []struct {
		name        string
		distance    float64
		wantStarted bool
	}{
		{name: "at six units", distance: 6, wantStarted: true},
		{name: "past six units", distance: 6.01, wantStarted: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			world, err := NewWorld("arena", 16)
			if err != nil {
				t.Fatal(err)
			}
			player := Entity{
				ID:               "p1",
				Kind:             EntityPlayer,
				Vitals:           Vitals{HP: 100, MaxHP: 100},
				Combat:           CombatState{Mana: 50, MaxMana: 100},
				CaptureGuBonuses: map[string]float64{"capture-gu-instance": 0},
				Transform:        Transform{Position: Vec2{X: 10, Y: 10}},
			}
			monster := newTestMonster("m1", Vec2{X: 10 + test.distance, Y: 10}, 100)
			monster.AI.AggroRadius = 1
			if err := world.AddEntities([]Entity{player, monster}); err != nil {
				t.Fatal(err)
			}
			if err := world.SubmitCapture(CaptureCommand{
				EntityID: "p1", Seq: 1, TargetID: "m1", GuID: "capture-gu-instance",
			}); err != nil {
				t.Fatal(err)
			}

			events := world.Step().Events
			if gotStarted := hasEvent(events, combat.EventCaptureStarted); gotStarted != test.wantStarted {
				t.Fatalf("capture started=%v, want %v; events=%+v", gotStarted, test.wantStarted, events)
			}
			if !test.wantStarted && (len(events) == 0 || events[0].Reason != "out_of_range") {
				t.Fatalf("expected out_of_range rejection, got %+v", events)
			}
		})
	}
}

func TestCaptureRejectsUnownedCaptureGu(t *testing.T) {
	world, _ := NewWorld("arena", 16)
	player := Entity{
		ID:               "p1",
		Kind:             EntityPlayer,
		Vitals:           Vitals{HP: 100, MaxHP: 100},
		Combat:           CombatState{Mana: 50, MaxMana: 100},
		CaptureGuBonuses: map[string]float64{"owned-capture-gu": 0.2},
		Transform:        Transform{Position: Vec2{X: 10, Y: 10}},
	}
	monster := newTestMonster("m1", Vec2{X: 12, Y: 10}, 100)
	monster.AI.AggroRadius = 1
	if err := world.AddEntities([]Entity{player, monster}); err != nil {
		t.Fatal(err)
	}
	if err := world.SubmitCapture(CaptureCommand{
		EntityID: "p1", Seq: 1, TargetID: "m1", GuID: "unowned-capture-gu",
	}); err != nil {
		t.Fatal(err)
	}

	snapshot := world.Step()
	for _, event := range snapshot.Events {
		if event.Type == combat.EventCastRejected && event.Reason == "capture_gu_required" {
			return
		}
	}
	t.Fatalf("capture with unowned Gu was not rejected: %+v", snapshot.Events)
}

func TestCaptureUsesSelectedGuBonus(t *testing.T) {
	world, err := NewWorld("arena", 16)
	if err != nil {
		t.Fatal(err)
	}

	player := Entity{
		ID:               "p1",
		Kind:             EntityPlayer,
		Vitals:           Vitals{HP: 100, MaxHP: 100},
		Transform:        Transform{Position: Vec2{X: 10, Y: 10}},
		CaptureGuBonuses: map[string]float64{"capture-gu-instance": 0.2},
	}
	monster := newTestMonster("m1", Vec2{X: 12, Y: 10}, 100)
	monster.AI.Rank = 4
	if err := world.AddEntities([]Entity{player, monster}); err != nil {
		t.Fatal(err)
	}

	const lowerChance = 0.15*0.55 + 0.15
	const selectedGuChance = 0.15*0.55 + 0.2
	for seed := int64(0); ; seed++ {
		roll := rand.New(rand.NewSource(seed)).Float64()
		if roll > lowerChance && roll < selectedGuChance {
			world.rng = rand.New(rand.NewSource(seed))
			break
		}
	}
	world.resolveCaptureLocked(world.entities["p1"], &PendingCast{
		TargetID:    "m1",
		IsCapture:   true,
		CaptureGuID: "capture-gu-instance",
	})

	if !hasEvent(world.events, combat.EventCaptureSuccess) {
		t.Fatalf("selected Gu bonus did not affect capture chance: %+v", world.events)
	}
}

func TestCaptureInterruptedByMove(t *testing.T) {
	world, _ := NewWorld("arena", 16)
	player := Entity{
		ID:     "p1",
		Kind:   EntityPlayer,
		Vitals: Vitals{HP: 100, MaxHP: 100},
		Combat: CombatState{Mana: 50, MaxMana: 100},
		CaptureGuBonuses: map[string]float64{
			"capture-gu-instance": 0.2,
		},
		Transform: Transform{Position: Vec2{X: 10, Y: 10}},
		Radius:    0.5,
	}
	monster := newTestMonster("m1", Vec2{X: 11, Y: 10}, 20)
	monster.Combat.Statuses = []StatusEffect{{ID: "FREEZE", ExpireTick: 200, Debuff: true}}

	if err := world.AddEntities([]Entity{player, monster}); err != nil {
		t.Fatalf("AddEntities failed: %v", err)
	}

	world.SubmitCapture(CaptureCommand{EntityID: "p1", Seq: 1, TargetID: "m1", GuID: "capture-gu-instance"})
	world.Step()

	p := world.entities["p1"]
	if p.Combat.Pending == nil || !p.Combat.Pending.IsCapture {
		t.Fatalf("expected pending capture")
	}

	// Submit move command to interrupt
	world.SubmitInput(InputCommand{
		EntityID: "p1",
		Seq:      2,
		Move:     Vec2{X: 1, Y: 0},
	})
	snap := world.Step()

	if p.Combat.Pending != nil {
		t.Errorf("expected pending capture to be cancelled by movement")
	}

	var foundInterrupted bool
	for _, ev := range snap.Events {
		if ev.Type == combat.EventCaptureInterrupted && ev.Reason == "moved" {
			foundInterrupted = true
		}
	}
	if !foundInterrupted {
		t.Errorf("expected EventCaptureInterrupted with reason 'moved'")
	}
}

func TestCompanionAIAndDeath(t *testing.T) {
	world, _ := NewWorld("arena", 16)
	player := Entity{
		ID:        "p1",
		Kind:      EntityPlayer,
		Vitals:    Vitals{HP: 100, MaxHP: 100},
		Transform: Transform{Position: Vec2{X: 10, Y: 10}},
		Radius:    0.5,
	}
	comp := Entity{
		ID:        "comp1",
		Kind:      EntityCompanion,
		Vitals:    Vitals{HP: 50, MaxHP: 50},
		Transform: Transform{Position: Vec2{X: 12, Y: 10}},
		Speed:     4.0,
		Radius:    0.5,
		Companion: CompanionState{
			Active:       true,
			OwnerID:      "p1",
			ActorID:      "actor-uuid-1",
			ActorKind:    "COMPANION_BEAST",
			Stance:       StanceDefend,
			FollowRadius: 4.0,
			LeashRadius:  15.0,
			AttackRange:  2.0,
			Rank:         2,
			Loyalty:      80,
		},
	}

	if err := world.AddEntities([]Entity{player, comp}); err != nil {
		t.Fatalf("AddEntities failed: %v", err)
	}

	// Companion follows owner
	world.Step()
	c := world.entities["comp1"]
	if c == nil {
		t.Fatalf("companion entity not found")
	}

	// Test kill companion to check death resolution (Injured vs Permadeath)
	world.applyDamageLocked(nil, c, 100, "FATAL_STRIKE")
	snap := world.Step()
	events := snap.Events

	// Companion should be removed from world
	if _, exists := world.entities["comp1"]; exists {
		t.Errorf("companion should be removed from world after lethal damage")
	}

	var foundCasualty bool
	for _, ev := range events {
		if ev.Type == combat.EventCompanionInjured || ev.Type == combat.EventCompanionPermadeath {
			foundCasualty = true
			if ev.PersistentID != "actor-uuid-1" {
				t.Errorf("expected PersistentID 'actor-uuid-1', got %s", ev.PersistentID)
			}
		}
	}
	if !foundCasualty {
		t.Errorf("expected EventCompanionInjured or EventCompanionPermadeath event in %+v", events)
	}
}
