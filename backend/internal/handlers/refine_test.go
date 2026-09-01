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
		{targetRank: 2, refinementRealm: 0, want: 5},
		{targetRank: 3, refinementRealm: 0, want: 15},
		{targetRank: 4, refinementRealm: 0, want: 40},
		{targetRank: 5, refinementRealm: 0, want: 100},
		{targetRank: 6, refinementRealm: 0, want: 500},
		{targetRank: 7, refinementRealm: 0, want: 2500},
		{targetRank: 8, refinementRealm: 0, want: 15000},
		{targetRank: 9, refinementRealm: 0, want: 50000},

		// refinementRealm = 1 (Master: points 100-999)
		{targetRank: 1, refinementRealm: 1, want: 2}, // Rank 1 gives gain for realm < 2 (points < 1000)
		{targetRank: 2, refinementRealm: 1, want: 4}, // base 5 * 0.8 = 4.0 -> 4
		{targetRank: 5, refinementRealm: 1, want: 80}, // base 100 * 0.8 = 80
		{targetRank: 6, refinementRealm: 1, want: 400}, // base 500 * 0.8 = 400

		// refinementRealm = 2 (Grandmaster: points 1000-9999)
		{targetRank: 1, refinementRealm: 2, want: 0}, // Rank 1 capped at realm 2 (>= 1000)
		{targetRank: 2, refinementRealm: 2, want: 3}, // base 5 * 0.6 = 3.0 -> 3
		{targetRank: 3, refinementRealm: 2, want: 9}, // base 15 * 0.6 = 9.0 -> 9
		{targetRank: 5, refinementRealm: 2, want: 60}, // base 100 * 0.6 = 60
		{targetRank: 6, refinementRealm: 2, want: 300}, // base 500 * 0.6 = 300

		// refinementRealm = 3 (Great Grandmaster: points 10000-99999)
		{targetRank: 1, refinementRealm: 3, want: 0},
		{targetRank: 2, refinementRealm: 3, want: 0}, // Rank 2 capped at realm 3 (>= 10000)
		{targetRank: 3, refinementRealm: 3, want: 0}, // Rank 3 capped at realm 3 (>= 10000)
		{targetRank: 4, refinementRealm: 3, want: 16}, // base 40 * 0.4 = 16
		{targetRank: 5, refinementRealm: 3, want: 40}, // base 100 * 0.4 = 40
		{targetRank: 6, refinementRealm: 3, want: 200}, // base 500 * 0.4 = 200

		// refinementRealm = 4 (Supreme Grandmaster: points >= 100000)
		{targetRank: 4, refinementRealm: 4, want: 0},
		{targetRank: 5, refinementRealm: 4, want: 0},
		{targetRank: 6, refinementRealm: 4, want: 0},
		{targetRank: 9, refinementRealm: 4, want: 0},
	}

	for _, tt := range tests {
		got := calculateRefinementPathGain(tt.targetRank, tt.refinementRealm)
		if got != tt.want {
			t.Errorf("calculateRefinementPathGain(%d, %d) = %d, want %d", tt.targetRank, tt.refinementRealm, got, tt.want)
		}
	}
}
