package handlers

import (
	"math"
	"testing"
)

func TestGuardNextTribulationYear(t *testing.T) {
	tests := []struct {
		name         string
		nextYear     float64
		currentYears float64
		expected     float64
	}{
		{
			name:         "Future scheduled tribulation",
			nextYear:     100.0,
			currentYears: 90.0,
			expected:     100.0,
		},
		{
			name:         "Tribulation exactly at current year",
			nextYear:     100.0,
			currentYears: 100.0,
			expected:     100.0,
		},
		{
			name:         "Overdue tribulation (current year slightly past nextYear)",
			nextYear:     100.0,
			currentYears: 100.05,
			expected:     100.0, // Must NOT bump to 110/111!
		},
		{
			name:         "MaxFloat64 sentinel for unhandled rank",
			nextYear:     math.MaxFloat64,
			currentYears: 100.05,
			expected:     111.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := guardNextTribulationYear(tt.nextYear, tt.currentYears)
			if got != tt.expected {
				t.Errorf("guardNextTribulationYear(%v, %v) = %v; want %v", tt.nextYear, tt.currentYears, got, tt.expected)
			}
		})
	}
}
