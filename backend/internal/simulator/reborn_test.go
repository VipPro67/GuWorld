package simulator

import (
	"strings"
	"testing"
)

func countRewinds(logs []CombatLog) int {
	n := 0
	for _, l := range logs {
		if strings.Contains(l.Detail, "Nghịch chuyển thời không") {
			n++
		}
	}
	return n
}

func countRevives(logs []CombatLog) int {
	n := 0
	for _, l := range logs {
		if strings.Contains(l.Detail, "vùng dậy với") {
			n++
		}
	}
	return n
}

// rebornMatchup pits a fragile reborn ally (100 HP) against an enemy that
// one-shots it (100 damage) every turn.
func rebornMatchup(params string) []CombatLog {
	ally := newTestCombatant("ally", "Player", 100, 0)
	ally.Speed = 1
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	ally.Mechanics = []CombatMechanic{mech("ON_DEATH", "REBORN", params)}

	enemy := newTestCombatant("enemy", "Enemy", 5000, 0)
	enemy.Speed = 100
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Sát", Damage: 100}}

	return Simulate(&CombatState{Seed: 13, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}})
}

// TestUnlimitedRebornRewindIsCapped verifies a max_uses=0 rewind (which resets
// the turn loop and would otherwise run forever) is bounded by
// maxInfiniteRebornRewinds.
func TestUnlimitedRebornRewindIsCapped(t *testing.T) {
	logs := rebornMatchup(`{"chance":1,"mode":"rewind","max_uses":0}`)
	if got := countRewinds(logs); got != maxInfiniteRebornRewinds {
		t.Fatalf("unlimited rewind used %d times, want %d", got, maxInfiniteRebornRewinds)
	}

	// Same seed must produce the same bounded result.
	again := rebornMatchup(`{"chance":1,"mode":"rewind","max_uses":0}`)
	if a, b := countRewinds(logs), countRewinds(again); a != b {
		t.Fatalf("rewind count not deterministic: %d vs %d", a, b)
	}
}

// TestUnlimitedRebornInPlaceIsNotCapped verifies the cap only guards rewinds:
// an unlimited in-place revive (already bounded by maxTurns) keeps reviving.
func TestUnlimitedRebornInPlaceIsNotCapped(t *testing.T) {
	logs := rebornMatchup(`{"chance":1,"mode":"in_place","max_uses":0,"revive_hp_percent":0.5}`)
	if got := countRevives(logs); got <= maxInfiniteRebornRewinds {
		t.Fatalf("in-place revives = %d, want more than the rewind cap %d", got, maxInfiniteRebornRewinds)
	}
	if got := countRewinds(logs); got != 0 {
		t.Fatalf("in-place mode should not rewind, got %d rewinds", got)
	}
}
