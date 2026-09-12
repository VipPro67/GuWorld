package handlers

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
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
		{
			name:         "Corrupted / stale past year (e.g. 100 when current is 13770)",
			nextYear:     100.0,
			currentYears: 13770.0,
			expected:     13780.0,
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

func TestTribulationNextYearHighYearsPassed(t *testing.T) {
	// Rank 6 with 13770 years passed: next milestone must be 13780 (Earth Calamity)
	nextYear, tribType := models.GetNextTribulationInfo(6, 0, 13770.0)
	if nextYear != 13780.0 || tribType != "EARTH_CALAMITY" {
		t.Errorf("GetNextTribulationInfo(6, 0, 13770) = (%v, %v); want (13780, EARTH_CALAMITY)", nextYear, tribType)
	}

	// At 13790 years passed: next milestone is 13800 (Heavenly Tribulation)
	nextYear, tribType = models.GetNextTribulationInfo(6, 0, 13790.0)
	if nextYear != 13800.0 || tribType != "HEAVENLY_TRIBULATION" {
		t.Errorf("GetNextTribulationInfo(6, 0, 13790) = (%v, %v); want (13800, HEAVENLY_TRIBULATION)", nextYear, tribType)
	}
}
