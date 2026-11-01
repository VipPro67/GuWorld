package sim

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

func dashSkill(behavior string) combat.Skill {
	damage := 0.0
	if behavior == "IMPACT" {
		damage = 20
	}
	return combat.Skill{ID: "DASH", Slot: 1, Name: "Dash", Mode: combat.ModeTarget, Range: 10,
		MoveDistance: 4, MoveDurationTicks: 6, MoveBehavior: behavior, Damage: damage}
}

func TestDashImpactHitsFirstContactOnlyAfterTravel(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 2, Y: 2})
	player.Combat.Skills[1] = dashSkill("IMPACT")
	for _, entity := range []Entity{player, newTargetMonster("near", Vec2{X: 4, Y: 2}), newTargetMonster("far", Vec2{X: 6, Y: 2})} {
		if err := w.AddEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, Aim: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	first := w.Step()
	if entityHP(t, w, "near") != 100 || hasEvent(first.Events, combat.EventHitConfirmed) {
		t.Fatal("impact resolved before contact")
	}
	var accepted *combat.Event
	for i := range first.Events {
		if first.Events[i].Type == combat.EventCastAccepted {
			accepted = &first.Events[i]
		}
	}
	if accepted == nil || accepted.DashDirX != 1 || accepted.DashDirY != 0 || accepted.DashDurationTicks != 6 {
		t.Fatalf("dash acceptance: %+v", accepted)
	}
	events := collectEvents(t, w, 6)
	if !hasEvent(events, combat.EventHitConfirmed) || !hasEvent(events, combat.EventDashEnded) || entityHP(t, w, "near") >= 100 || entityHP(t, w, "far") != 100 {
		t.Fatalf("first contact not applied exactly once: %+v", events)
	}
	if x := entityPosition(t, w, "player").X; x >= 4 {
		t.Fatalf("dash passed first target: %v", x)
	}
}

func TestDashMoveOnlyPassesEntitiesButStopsAtWall(t *testing.T) {
	solids := make([]bool, 10*5)
	for y := 0; y < 5; y++ {
		solids[y*10+5] = true
	}
	grid, err := NewGridCollision(10, 5, solids)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorldWithCollision("arena", 16, grid)
	if err != nil {
		t.Fatal(err)
	}
	player := newCombatPlayer("player", Vec2{X: 2, Y: 2})
	player.Combat.Skills[1] = dashSkill("MOVE_ONLY")
	for _, entity := range []Entity{player, newTargetMonster("mob", Vec2{X: 3, Y: 2})} {
		if err := w.AddEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, Aim: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 8)
	if x := entityPosition(t, w, "player").X; x < 3.5 || x >= 5 {
		t.Fatalf("movement-only path/wall: %v", x)
	}
	if entityHP(t, w, "mob") != 100 || hasEvent(events, combat.EventHitConfirmed) {
		t.Fatal("movement-only dash dealt contact damage")
	}
	if !hasEvent(events, combat.EventDashEnded) {
		t.Fatal("wall stop did not notify client")
	}
}

func TestDashWallBlocksImpactBehindIt(t *testing.T) {
	solids := make([]bool, 10*5)
	for y := 0; y < 5; y++ {
		solids[y*10+5] = true
	}
	grid, err := NewGridCollision(10, 5, solids)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorldWithCollision("arena", 16, grid)
	if err != nil {
		t.Fatal(err)
	}
	player := newCombatPlayer("player", Vec2{X: 2, Y: 2})
	player.Combat.Skills[1] = dashSkill("IMPACT")
	for _, entity := range []Entity{player, newTargetMonster("mob", Vec2{X: 6.5, Y: 2})} {
		if err := w.AddEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, Aim: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 8)
	if entityHP(t, w, "mob") != 100 || hasEvent(events, combat.EventHitConfirmed) {
		t.Fatal("impact passed through wall")
	}
	if x := entityPosition(t, w, "player").X; x >= 5 {
		t.Fatalf("dash passed wall: %v", x)
	}
}

func TestDashRootDoesNotStopActiveDashButHardCCDoes(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 2, Y: 2})
	player.Combat.Skills[1] = dashSkill("MOVE_ONLY")
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, Aim: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	w.Step()
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 2, Slot: 1, Mode: combat.ModeTarget, Aim: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	if !hasEvent(w.Step().Events, combat.EventCastRejected) {
		t.Fatal("second dash accepted while active")
	}
	root := testStatusSkill("ROOT", 0, 0, 20, 0)
	root.StatusDebuff, root.StatusMoveMultiplier = true, 0
	w.applyStatusLocked(nil, worldEntity(t, w, "player"), root)
	before := entityPosition(t, w, "player").X
	w.Step()
	if after := entityPosition(t, w, "player").X; after <= before {
		t.Fatal("root stopped active dash")
	}
	stun := testStatusSkill("STUN", 0, 0, 20, 0)
	stun.StatusDebuff, stun.StatusMoveMultiplier = true, 0
	w.applyStatusLocked(nil, worldEntity(t, w, "player"), stun)
	before = entityPosition(t, w, "player").X
	w.Step()
	if after := entityPosition(t, w, "player").X; math.Abs(after-before) > 0.001 {
		t.Fatal("hard CC failed to stop dash")
	}
}

func TestDashUsesHeldMovementAndLocksDirectionAtCast(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 2, Y: 2})
	player.Combat.Skills[1] = dashSkill("MOVE_ONLY")
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitInput(InputCommand{EntityID: "player", Seq: 1, Move: Vec2{X: 0, Y: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 2, Slot: 1, Mode: combat.ModeTarget}); err != nil {
		t.Fatal(err)
	}
	events := w.Step().Events
	var accepted *combat.Event
	for i := range events {
		if events[i].Type == combat.EventCastAccepted {
			accepted = &events[i]
		}
	}
	if accepted == nil || accepted.DashDirX != 0 || accepted.DashDirY != 1 {
		t.Fatalf("held movement direction lost: %+v", accepted)
	}
	if err := w.SubmitInput(InputCommand{EntityID: "player", Seq: 3, Move: Vec2{X: -1}}); err != nil {
		t.Fatal(err)
	}
	collectEvents(t, w, 6)
	position := entityPosition(t, w, "player")
	if position.Y < 5.8 || math.Abs(position.X-2) > 0.01 {
		t.Fatalf("dash direction changed after acceptance: %+v", position)
	}
}
