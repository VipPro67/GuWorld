package handlers

import "testing"

// TestMartialBracketShape verifies the bye/slot invariants for every bracket size
// the event can produce: byes non-negative, remaining combatants pair cleanly into
// an even count, and slot count is the smallest power of two >= n.
func TestMartialBracketShape(t *testing.T) {
	for n := 2; n <= MaxMartialPlayersPerRank; n++ {
		numSlots, byes := martialBracketShape(n)
		if numSlots != nextPowerOf2(n) {
			t.Fatalf("n=%d: numSlots=%d want %d", n, numSlots, nextPowerOf2(n))
		}
		if numSlots < n {
			t.Fatalf("n=%d: numSlots=%d < n", n, numSlots)
		}
		if byes < 0 {
			t.Fatalf("n=%d: byes=%d < 0", n, byes)
		}
		if byes >= n {
			t.Fatalf("n=%d: byes=%d >= n", n, byes)
		}
		// Remaining (non-bye) combatants must pair into whole matches.
		if (n-byes)%2 != 0 {
			t.Fatalf("n=%d: remaining %d combatants is odd", n, n-byes)
		}
		// Round-1 match count is exactly half the slots.
		if byes+(n-byes)/2 != numSlots/2 {
			t.Fatalf("n=%d: round-1 match count mismatch", n)
		}
	}
}
