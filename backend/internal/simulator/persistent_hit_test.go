package simulator

import (
	"testing"
)

// TestStrikeLogsCarrySourceGuID verifies damage logs attribute hits to the
// data-driven Gu action (SourceGuID) and flag contact via HitLanded, so
// post-combat persistent hooks don't parse Detail strings.
func TestStrikeLogsCarrySourceGuID(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 0)
	ally.Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "Sát Sinh Khí Cổ", SourceID: "GU_KILL_ENERGY", Damage: 100, DamageMult: 1.0, TargetType: "SINGLE"},
	}
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}
	state := &CombatState{Seed: 1, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if l.SourceGuID != "GU_KILL_ENERGY" {
			continue
		}
		found = true
		if !l.HitLanded {
			t.Fatalf("contact strike must set HitLanded: %+v", l)
		}
		if l.Damage <= 0 {
			t.Fatalf("expected damage on contact log: %+v", l)
		}
	}
	if !found {
		t.Fatal("expected at least one log with SourceGuID=GU_KILL_ENERGY")
	}
}

// TestShieldAbsorbCountsAsHit verifies a strike fully absorbed by
// VirtualShield still lands (HitLanded=true with damage attributed).
func TestShieldAbsorbCountsAsHit(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 0)
	enemy.VirtualShield = 100000
	ally.Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "Sát Sinh Khí Cổ", SourceID: "GU_KILL_ENERGY", Damage: 100, DamageMult: 1.0, TargetType: "SINGLE"},
	}
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}
	state := &CombatState{Seed: 1, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}

	logs := Simulate(state)

	for _, l := range logs {
		if l.SourceGuID == "GU_KILL_ENERGY" && l.HitLanded {
			return // shield-absorbed hit still counts
		}
	}
	t.Fatal("expected a HitLanded log for shield-absorbed strike")
}
