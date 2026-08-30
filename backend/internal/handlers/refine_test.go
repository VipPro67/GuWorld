package handlers

import (
	"testing"
)

func TestCalculateRefinementPathGain(t *testing.T) {
	tests := []struct {
		targetRank      int
		refinementRealm int
		want            int
	}{
		// refinementRealm = 0 (Ordinary)
		{targetRank: 1, refinementRealm: 0, want: 2},
		{targetRank: 2, refinementRealm: 0, want: 4},
		{targetRank: 3, refinementRealm: 0, want: 6},
		{targetRank: 4, refinementRealm: 0, want: 8},
		{targetRank: 5, refinementRealm: 0, want: 10},
		{targetRank: 6, refinementRealm: 0, want: 20},
		{targetRank: 7, refinementRealm: 0, want: 35},
		{targetRank: 8, refinementRealm: 0, want: 50},
		{targetRank: 9, refinementRealm: 0, want: 75},

		// refinementRealm = 1 (Master)
		{targetRank: 1, refinementRealm: 1, want: 0}, // targetRank <= refinementRealm -> 0
		{targetRank: 2, refinementRealm: 1, want: 3}, // base 4 * 0.8 = 3.2 -> 3
		{targetRank: 5, refinementRealm: 1, want: 8}, // base 10 * 0.8 = 8
		{targetRank: 6, refinementRealm: 1, want: 16}, // base 20 * 0.8 = 16

		// refinementRealm = 2 (Grandmaster)
		{targetRank: 1, refinementRealm: 2, want: 0},
		{targetRank: 2, refinementRealm: 2, want: 0},
		{targetRank: 3, refinementRealm: 2, want: 4}, // base 6 * 0.6 = 3.6 -> 4
		{targetRank: 5, refinementRealm: 2, want: 6}, // base 10 * 0.6 = 6
		{targetRank: 6, refinementRealm: 2, want: 12}, // base 20 * 0.6 = 12

		// refinementRealm = 3 (Great Grandmaster)
		{targetRank: 3, refinementRealm: 3, want: 0},
		{targetRank: 4, refinementRealm: 3, want: 3}, // base 8 * 0.4 = 3.2 -> 3
		{targetRank: 5, refinementRealm: 3, want: 4}, // base 10 * 0.4 = 4
		{targetRank: 6, refinementRealm: 3, want: 8}, // base 20 * 0.4 = 8

		// refinementRealm = 4 (Supreme Grandmaster)
		{targetRank: 4, refinementRealm: 4, want: 0},
		{targetRank: 5, refinementRealm: 4, want: 2}, // base 10 * 0.2 = 2
		{targetRank: 6, refinementRealm: 4, want: 4}, // base 20 * 0.2 = 4
		{targetRank: 9, refinementRealm: 4, want: 15}, // base 75 * 0.2 = 15
	}

	for _, tt := range tests {
		got := calculateRefinementPathGain(tt.targetRank, tt.refinementRealm)
		if got != tt.want {
			t.Errorf("calculateRefinementPathGain(%d, %d) = %d, want %d", tt.targetRank, tt.refinementRealm, got, tt.want)
		}
	}
}
