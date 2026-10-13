package worker

import (
	"math"
	"testing"
)

func TestSectVeinCoverage(t *testing.T) {
	cases := []struct {
		level int
		want  float64
	}{
		{1, 0.65},
		{4, 0.80},
		{8, 1.00},
		{9, 1.05},
		{20, 1.05}, // capped
	}
	for _, tc := range cases {
		if got := sectVeinCoverage(tc.level); math.Abs(got-tc.want) > 1e-9 {
			t.Fatalf("coverage(%d) = %v, want %v", tc.level, got, tc.want)
		}
	}
}

// The point of the whole mechanic: with enough vein levels the passive income
// cancels the level + member maintenance fee, so a stable sect neither bleeds
// nor hoards. Vault fees stay uncovered.
func TestSectVeinBreaksEvenOnLevelAndMemberFee(t *testing.T) {
	const sectLevel, members = 3, 12
	upkeep := float64(sectLevel*dailyLevelFee + members*dailyMemberFee)

	if got := sectVeinDailyCredit(sectLevel, members, 8, 0); math.Abs(got-upkeep) > 1e-9 {
		t.Fatalf("vein level 8 credit = %v, want break-even %v", got, upkeep)
	}
	if got := sectVeinDailyCredit(sectLevel, members, 1, 0); got >= upkeep {
		t.Fatalf("vein level 1 credit = %v, want below upkeep %v", got, upkeep)
	}
	// Defenders push income above break-even.
	if got := sectVeinDailyCredit(sectLevel, members, 8, 2); got <= upkeep {
		t.Fatalf("2 defenders credit = %v, want above upkeep %v", got, upkeep)
	}
}
