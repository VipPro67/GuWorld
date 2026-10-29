package handlers

import (
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
)

func TestOrderCombatSkillsRespectsStoredOrderWithoutBackfill(t *testing.T) {
	resolutions := []combatSkillResolution{
		{entry: combatLoadoutEntry{Kind: "move", ID: "a"}, move: RealtimeKillerMove{ID: "a"}},
		{entry: combatLoadoutEntry{Kind: "move", ID: "b"}, move: RealtimeKillerMove{ID: "b"}},
		{entry: combatLoadoutEntry{Kind: "house", ID: "c"}, move: RealtimeKillerMove{ID: "c"}},
	}
	loadout := []combatLoadoutEntry{
		{Kind: "house", ID: "c"},
		{Kind: "move", ID: "missing"},
		{Kind: "move", ID: "b"},
	}
	ordered := orderCombatSkills(loadout, true, resolutions)
	if len(ordered) != 2 {
		t.Fatalf("expected 2 explicitly bound slots, got %d", len(ordered))
	}
	if ordered[0].entry.ID != "c" || ordered[1].entry.ID != "b" {
		t.Fatalf("unexpected order: %v, %v", ordered[0].entry.ID, ordered[1].entry.ID)
	}
}

func TestOrderCombatSkillsBackfillsWhenNoStoredOrder(t *testing.T) {
	resolutions := []combatSkillResolution{
		{entry: combatLoadoutEntry{Kind: "move", ID: "a"}, move: RealtimeKillerMove{ID: "a"}},
		{entry: combatLoadoutEntry{Kind: "house", ID: "c"}, move: RealtimeKillerMove{ID: "c"}},
	}
	ordered := orderCombatSkills(nil, false, resolutions)
	if len(ordered) != 2 || ordered[0].entry.ID != "a" || ordered[1].entry.ID != "c" {
		t.Fatalf("unexpected backfill order: %v", ordered)
	}
}

func TestOrderCombatSkillsCapsAtFourSlots(t *testing.T) {
	resolutions := make([]combatSkillResolution, 0, 6)
	for _, id := range []string{"a", "b", "c", "d", "e", "f"} {
		resolutions = append(resolutions, combatSkillResolution{
			entry: combatLoadoutEntry{Kind: "move", ID: id},
			move:  RealtimeKillerMove{ID: id},
		})
	}
	ordered := orderCombatSkills(nil, false, resolutions)
	if len(ordered) != combatLoadoutSlots {
		t.Fatalf("expected %d slots, got %d", combatLoadoutSlots, len(ordered))
	}
}

func TestSupportedKillerMoveAcceptsHouseDamageBlock(t *testing.T) {
	move := realtimeMoveForTest(120)
	ok, reason := supportedKillerMove(move)
	if !ok {
		t.Fatalf("expected synthesized house strike to compile, got reason %q", reason)
	}
}

func TestSupportedKillerMoveAllowsStarvingComponents(t *testing.T) {
	move := realtimeMoveForTest(120)
	move.Components[0].Hunger = 0
	ok, reason := supportedKillerMove(move)
	if !ok {
		t.Fatalf("starving components must not block loadout authoring, got reason %q", reason)
	}
	if len(realtimeeffects.BuildKillerMoveSkills([]realtimeeffects.KillerMove{move})) != 0 {
		t.Fatal("runtime compiler must still reject starving components")
	}
}

func TestHouseStrikeCompiledSkillHasNoMasteryTarget(t *testing.T) {
	house := realtimeMoveForTest(120)
	house.HouseStrike = true
	compiled := realtimeeffects.BuildKillerMoveSkills([]realtimeeffects.KillerMove{house})
	if len(compiled) != 1 {
		t.Fatalf("expected house strike to compile, got %d skills", len(compiled))
	}
	if compiled[0].KillerMoveID != "" {
		t.Fatalf("house strike must not target killer-move mastery, got %q", compiled[0].KillerMoveID)
	}
	playerMove := realtimeMoveForTest(120)
	compiled = realtimeeffects.BuildKillerMoveSkills([]realtimeeffects.KillerMove{playerMove})
	if len(compiled) != 1 || compiled[0].KillerMoveID != playerMove.ID {
		t.Fatalf("player killer move must keep its mastery target, got %+v", compiled)
	}
}

func realtimeMoveForTest(damage float64) realtimeeffects.KillerMove {
	return realtimeeffects.KillerMove{
		ID: "house-strike", Name: "Trận Kích: Test", Rank: 6,
		Components:         []realtimeeffects.KillerMoveComponent{{InstanceID: "component", Hunger: 10}},
		ExpectedComponents: 1, ManaCost: 5, SuccessChance: 0.95, Complexity: 1,
		Effects: []gameplayeffects.Block{{
			Trigger: gameplayeffects.TriggerOnAction,
			Target:  gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
			Operations: []gameplayeffects.Operation{{
				Op: gameplayeffects.OpDamage, Value: damage, Path: "strength",
			}},
		}},
	}
}
