package worker

import "testing"

func TestLaborIntensityModifier(t *testing.T) {
	tests := map[string]float64{"LOW": 0.7, "NORMAL": 1, "HIGH": 1.5, "EXPLOIT": 2, "": 1}
	for input, want := range tests {
		if got := laborIntensityModifier(input); got != want {
			t.Errorf("laborIntensityModifier(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestSimulateRaceGrowsWithinSettlementCapacity(t *testing.T) {
	v := societyVariant{population: 500, happiness: 70, prosperity: 50, foodStored: 5000, foodPerCapita: 1, populationGrowth: .001, mortalityRate: .0005, laborIntensity: "NORMAL"}
	got := simulateRace(v, 0, 500)
	if got.population != 500 {
		t.Fatalf("population = %d, want capacity-limited 500", got.population)
	}
	if got.foodStored != 4500 {
		t.Fatalf("foodStored = %v, want 4500", got.foodStored)
	}
}

func TestRequiredHairyMenIncreasesWithGuRank(t *testing.T) {
	want := []int{100, 200, 300, 400, 500, 600, 700, 800, 900}
	for rank, expected := range want {
		got := requiredHairyMen(rank + 1)
		if got != expected {
			t.Errorf("requiredHairyMen(%d) = %d, want %d", rank+1, got, expected)
		}
		if rank > 0 && got <= want[rank-1] {
			t.Errorf("rank %d workforce %d is not greater than rank %d workforce %d", rank+1, got, rank, want[rank-1])
		}
	}
}

func TestRequiredHairyMenClampsUnknownRanks(t *testing.T) {
	if got := requiredHairyMen(0); got != 100 {
		t.Errorf("rank 0 = %d, want 100", got)
	}
	if got := requiredHairyMen(10); got != 900 {
		t.Errorf("rank 10 = %d, want 900", got)
	}
}

func TestSelectRefinementCandidatesReservesRankedWorkforce(t *testing.T) {
	candidates := []refinementCandidate{{id: "rank-6", rank: 6}, {id: "rank-3", rank: 3}, {id: "rank-2", rank: 2}}
	selected := selectRefinementCandidates(candidates, 500)
	if len(selected) != 2 || selected[0].id != "rank-3" || selected[1].id != "rank-2" {
		t.Fatalf("selected = %#v, want rank-3 and rank-2", selected)
	}
}
