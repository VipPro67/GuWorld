package worker

import "testing"

func TestNamedNPCBonusMatchesRoleAndCaps(t *testing.T) {
	npcs := []namedNPCWorker{
		{role: "MINER", rank: 5, talent: 100, loyalty: 100},
		{role: "FARMER", rank: 5, talent: 100, loyalty: 100},
		{role: "MINER", rank: 9, talent: 1000, loyalty: 1000},
	}
	if got := namedNPCBonus(npcs, "MINER", .05); got != .05 {
		t.Fatalf("expected capped bonus .05, got %v", got)
	}
	if got := namedNPCBonus(npcs, "REFINER", .25); got != 0 {
		t.Fatalf("unmatched roles must not contribute, got %v", got)
	}
}

func TestNamedNPCNodeRole(t *testing.T) {
	for _, test := range []struct{ node, key, want string }{
		{"MINING_ZONE", "ore", "MINER"},
		{"FARMING_ZONE", "rice", "FARMER"},
		{"BREEDING_ZONE", "food_meat", "FARMER"},
	} {
		if got := namedNPCNodeRole(test.node, test.key); got != test.want {
			t.Fatalf("%s/%s: want %s, got %s", test.node, test.key, test.want, got)
		}
	}
}
