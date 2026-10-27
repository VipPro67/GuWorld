package sim

import (
	"testing"
)

func TestTribulationHazardTelegraphAndStrike(t *testing.T) {
	w, err := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "hazard_test", RulesetVersion: "v1"}, 8, 99)
	if err != nil {
		t.Fatal(err)
	}

	// Player at (5, 5)
	player := newPlayer("player")
	player.Transform.Position = Vec2{X: 5, Y: 5}

	// Resource node at (6, 5)
	node := Entity{
		ID:        "node_tree",
		Kind:      EntityMonster,
		Name:      "Cổ Thụ",
		Transform: Transform{Position: Vec2{X: 6, Y: 5}},
		Vitals:    Vitals{HP: 1000, MaxHP: 1000},
		Speed:     0,
		Radius:    0.8,
		Faction:   "player",
	}

	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(node); err != nil {
		t.Fatal(err)
	}

	// Add hazard at (5.5, 5) with radius 2.0 and damage 200, delay 3 ticks
	w.AddTribulationHazard(5.5, 5, 2.0, 200, 3, "LIGHTNING_STRIKE")

	// Step 1 tick: hazard is telegraphing, no damage yet
	w.Step()
	if w.entities["player"].Vitals.HP != 100 {
		t.Fatalf("player damaged prematurely: %v", w.entities["player"].Vitals.HP)
	}
	if w.entities["node_tree"].Vitals.HP != 1000 {
		t.Fatalf("node damaged prematurely: %v", w.entities["node_tree"].Vitals.HP)
	}

	// Step 2 more ticks (total 3 ticks): strike happens!
	w.Step()
	events := w.Step().Events

	// Check that strike event was emitted
	foundStrike := false
	for _, ev := range events {
		if ev.Reason == "tribulation_strike" && ev.SkillID == "LIGHTNING_STRIKE" {
			foundStrike = true
			break
		}
	}
	if !foundStrike {
		t.Fatalf("expected tribulation_strike event in tick 3 events, got: %+v", events)
	}

	// Both player and node should have taken damage
	if w.entities["player"].Vitals.HP >= 100 {
		t.Fatalf("player did not take hazard damage: %v", w.entities["player"].Vitals.HP)
	}
	if w.entities["node_tree"].Vitals.HP >= 1000 {
		t.Fatalf("resource node did not take hazard damage: %v", w.entities["node_tree"].Vitals.HP)
	}
}
