package handlers

import (
	"math"
	"testing"
)

func TestCoopTribulationScalingFormula(t *testing.T) {
	calcScaling := func(hostRank int, helperRanks []int) float64 {
		mult := 1.0
		for _, hr := range helperRanks {
			delta := hr - hostRank + 1
			if delta < 0 {
				delta = 0
			}
			mult += 0.4 + 0.3*float64(delta)
		}
		return mult
	}

	// 1. Solo Rank 6 -> multiplier should be 1.0x
	if got := calcScaling(6, []int{}); got != 1.0 {
		t.Fatalf("solo scaling = %v, want 1.0", got)
	}

	// 2. Host Rank 6 with 1 helper Rank 6 -> delta = 1 -> mult = 1.0 + 0.4 + 0.3 = 1.7x
	if got := calcScaling(6, []int{6}); math.Abs(got-1.7) > 0.001 {
		t.Fatalf("1 helper R6 scaling = %v, want 1.7", got)
	}

	// 3. Host Rank 6 with 1 helper Rank 7 -> delta = 2 -> mult = 1.0 + 0.4 + 0.6 = 2.0x
	if got := calcScaling(6, []int{7}); math.Abs(got-2.0) > 0.001 {
		t.Fatalf("1 helper R7 scaling = %v, want 2.0", got)
	}

	// 4. Host Rank 6 with 2 helpers (Rank 6 and Rank 7) -> mult = 1.0 + 0.7 + 1.0 = 2.7x
	if got := calcScaling(6, []int{6, 7}); math.Abs(got-2.7) > 0.001 {
		t.Fatalf("2 helpers (R6, R7) scaling = %v, want 2.7", got)
	}
}

func TestApertureDaoStressFormula(t *testing.T) {
	calcStress := func(apertureGradeRank int, guestRank int, isSuppressed bool) float64 {
		if isSuppressed || guestRank <= apertureGradeRank {
			return 0
		}
		delta := guestRank - apertureGradeRank
		return float64(delta * 250)
	}

	// 1. Equal or lower rank -> 0 stress
	if got := calcStress(6, 6, false); got != 0 {
		t.Fatalf("equal rank stress = %v, want 0", got)
	}
	if got := calcStress(6, 5, false); got != 0 {
		t.Fatalf("lower rank stress = %v, want 0", got)
	}

	// 2. Higher rank (Rank 7 in Rank 6 aperture) -> delta 1 * 250 = 250 HP/s
	if got := calcStress(6, 7, false); got != 250 {
		t.Fatalf("R7 in R6 aperture stress = %v, want 250", got)
	}

	// 3. Higher rank with suppression active -> 0 stress
	if got := calcStress(6, 7, true); got != 0 {
		t.Fatalf("suppressed R7 stress = %v, want 0", got)
	}

	// 4. Rank 8 in Rank 6 aperture -> delta 2 * 250 = 500 HP/s
	if got := calcStress(6, 8, false); got != 500 {
		t.Fatalf("R8 in R6 aperture stress = %v, want 500", got)
	}
}
