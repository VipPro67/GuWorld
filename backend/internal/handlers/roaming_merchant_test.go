package handlers

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestRoamingZoneWeightDampsOccupiedZones(t *testing.T) {
	// Damping applies in every tier.
	if roamingZoneWeight(roamingRegionCandidate{Traders: 3}, false) >= roamingZoneWeight(roamingRegionCandidate{}, false) {
		t.Fatal("zone already holding traders must weigh less in the wildcard tier")
	}
	// Player bonus only applies inside the player tier.
	empty := roamingRegionCandidate{}
	withPlayer := roamingRegionCandidate{Players: 1}
	busy := roamingRegionCandidate{Players: 3}
	if roamingZoneWeight(withPlayer, true) <= roamingZoneWeight(empty, true) {
		t.Fatalf("zone with a player must outweigh an empty zone: %v <= %v",
			roamingZoneWeight(withPlayer, true), roamingZoneWeight(empty, true))
	}
	if roamingZoneWeight(busy, true) <= roamingZoneWeight(withPlayer, true) {
		t.Fatalf("more players must weigh more: %v <= %v",
			roamingZoneWeight(busy, true), roamingZoneWeight(withPlayer, true))
	}
	occupied := roamingRegionCandidate{Players: 1, Traders: 3}
	if roamingZoneWeight(occupied, true) >= roamingZoneWeight(withPlayer, true) {
		t.Fatalf("zone already holding traders must be damped: %v >= %v",
			roamingZoneWeight(occupied, true), roamingZoneWeight(withPlayer, true))
	}
}

// A draw below roamingPlayerZoneChance stays inside player zones; a draw above it
// may land anywhere. Both use the cumulative weighted roll.
func TestPickRoamingRegionTiers(t *testing.T) {
	zones := []roamingRegionCandidate{
		{ID: "hot", Players: 2},
		{ID: "empty-a"},
		{ID: "empty-b"},
	}

	rolls := []float64{0.10, 0.0} // player tier, first eligible zone
	if got := pickRoamingRegion(zones, func() float64 {
		r := rolls[0]
		rolls = rolls[1:]
		return r
	}); got != 0 {
		t.Fatalf("player-tier roll should pick the player zone, got %d", got)
	}

	rolls = []float64{0.90, 0.99} // wildcard tier, last zone
	if got := pickRoamingRegion(zones, func() float64 {
		r := rolls[0]
		rolls = rolls[1:]
		return r
	}); got != 2 {
		t.Fatalf("wildcard-tier roll should be able to pick an empty zone, got %d", got)
	}
}

// The player tier must not dead-end when nobody is playing: the picker falls
// through to the wildcard tier instead of returning -1.
func TestPickRoamingRegionFallsBackWithNoPlayers(t *testing.T) {
	zones := []roamingRegionCandidate{{ID: "a"}, {ID: "b"}}

	rolls := []float64{0.10, 0.50}
	got := pickRoamingRegion(zones, func() float64 {
		r := rolls[0]
		rolls = rolls[1:]
		return r
	})
	if got < 0 || got >= len(zones) {
		t.Fatalf("expected a wildcard-tier fallback pick, got %d", got)
	}

	if got := pickRoamingRegion(nil, rand.Float64); got != -1 {
		t.Fatalf("no candidates must return -1, got %d", got)
	}
}

// Most spawns (roamingPlayerZoneChance, plus whatever the wildcard tier happens
// to hand the player zones) land where players are, and every zone keeps a
// non-zero chance so the wilderness still gets traders.
func TestPickRoamingRegionDistribution(t *testing.T) {
	zones := []roamingRegionCandidate{
		{ID: "hot", Players: 1},
		{ID: "warm", Players: 2},
	}
	for i := 0; i < 8; i++ {
		zones = append(zones, roamingRegionCandidate{ID: fmt.Sprintf("cold-%d", i)})
	}
	rng := rand.New(rand.NewSource(7))

	const draws = 20000
	hits := make([]int, len(zones))
	for i := 0; i < draws; i++ {
		hits[pickRoamingRegion(zones, rng.Float64)]++
	}

	playerShare := float64(hits[0]+hits[1]) / float64(draws)
	// 75% forced into the player tier + 25% × (2 of 10 zones) from the wildcard
	// tier ≈ 80%; allow slack for the RNG.
	if playerShare < 0.70 || playerShare > 0.88 {
		t.Fatalf("player zones should receive ~80%% of spawns, got %.1f%%", playerShare*100)
	}
	for i, z := range zones {
		if hits[i] == 0 {
			t.Fatalf("zone %s never received a trader in %d draws", z.ID, draws)
		}
	}
}
