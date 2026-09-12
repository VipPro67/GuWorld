package handlers

import (
	"testing"
)

func TestSealingStoneRequirementByRank(t *testing.T) {
	tests := []struct {
		rank     int
		expected int64
	}{
		{rank: 0, expected: 1},
		{rank: -1, expected: 1},
		{rank: 1, expected: 1},
		{rank: 2, expected: 2},
		{rank: 3, expected: 5},
		{rank: 4, expected: 20},
		{rank: 5, expected: 100},
		{rank: 6, expected: 10000},
		{rank: 7, expected: 1000000},
		{rank: 8, expected: 100000000},
		{rank: 9, expected: 10000000000},
	}

	for _, tt := range tests {
		actual := getSealingStoneCostByRank(tt.rank)
		if actual != tt.expected {
			t.Errorf("For rank %d, expected %d sealing stones, but got %d", tt.rank, tt.expected, actual)
		}
	}
}
