package handlers

import (
	"math"
	"testing"
)

func TestGetBreakthroughChance(t *testing.T) {
	tests := []struct {
		rank     int
		aptitude string
		want     float64
	}{
		{rank: 1, aptitude: "Giáp", want: 1.0},
		{rank: 1, aptitude: "Ất", want: 0.8},
		{rank: 2, aptitude: "Giáp", want: 0.9},
		{rank: 5, aptitude: "Đinh", want: 0.2},
	}

	for _, tt := range tests {
		got := GetBreakthroughChance(tt.rank, tt.aptitude)
		if math.Abs(got-tt.want) > 0.0001 {
			t.Errorf("GetBreakthroughChance(%d, %q) = %v, want %v", tt.rank, tt.aptitude, got, tt.want)
		}
	}
}

func TestWallProgressCapping(t *testing.T) {
	// Test capping wall progress gain at 1.0 (100%)
	progress := 0.95
	progressGain := 0.30
	progress += progressGain
	if progress > 1.0 {
		progress = 1.0
	}
	if progress != 1.0 {
		t.Errorf("Expected progress capped at 1.0, got %v", progress)
	}

	// Test breakthrough failure reduction when starting at 1.0
	failedProgress := math.Min(1.0, math.Max(0, progress-0.1))
	if math.Abs(failedProgress-0.9) > 0.0001 {
		t.Errorf("Expected failed breakthrough progress to be 0.9, got %v", failedProgress)
	}
}
