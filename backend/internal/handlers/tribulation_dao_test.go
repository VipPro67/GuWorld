package handlers

import (
	"reflect"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestEligibleOrdinaryDaoMarkPathsExcludesHeavenWithoutMutatingInput(t *testing.T) {
	input := []string{"fire", " Heaven Path ", "water", "HEAVEN"}
	original := append([]string(nil), input...)

	got := eligibleOrdinaryDaoMarkPaths(input)
	want := []string{"fire", "water"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("eligibleOrdinaryDaoMarkPaths() = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("eligibleOrdinaryDaoMarkPaths mutated input: got %v, want %v", input, original)
	}
}

func TestRandomOrdinaryDaoMarkPathsNeverSelectsHeaven(t *testing.T) {
	paths := []string{"fire", "heaven", "water", "earth"}
	for i := 0; i < 100; i++ {
		got := randomOrdinaryDaoMarkPaths(paths, 2)
		if len(got) != 2 || got[0] == heavenDaoPath || got[1] == heavenDaoPath || got[0] == got[1] {
			t.Fatalf("unexpected ascension paths: %v", got)
		}
	}
}

func TestRandomOrdinaryDaoMarkPathsCapsCountToEligiblePaths(t *testing.T) {
	got := randomOrdinaryDaoMarkPaths([]string{"heaven", "fire"}, 2)
	if !reflect.DeepEqual(got, []string{"fire"}) {
		t.Fatalf("randomOrdinaryDaoMarkPaths() = %v, want [fire]", got)
	}
}

func TestDistributeDaoMarksExcludesHeavenAndPreservesTotal(t *testing.T) {
	paths := []string{"fire", "heaven", "water", "earth"}
	for _, mainPath := range []string{"fire", "heaven"} {
		for i := 0; i < 100; i++ {
			rewards := distributeDaoMarksFromPaths(mainPath, 1000, paths)
			if _, exists := rewards[heavenDaoPath]; exists {
				t.Fatalf("distributeDaoMarksFromPaths(%q) rewarded Heaven Path: %v", mainPath, rewards)
			}
			var total int64
			for _, amount := range rewards {
				total += amount
			}
			if total != 1000 {
				t.Fatalf("total = %d, want 1000: %v", total, rewards)
			}
			if mainPath == "fire" && (rewards["fire"] != 1000 || len(rewards) != 1) {
				t.Fatalf("main path reward = %d (len %d), want 1000 (len 1): %v", rewards["fire"], len(rewards), rewards)
			}
		}
	}
}

func TestDistributeDaoMarksHandlesNoEligiblePaths(t *testing.T) {
	if got := distributeDaoMarksFromPaths("heaven", 1000, []string{"heaven"}); len(got) != 0 {
		t.Fatalf("got %v, want no rewards", got)
	}
	if got := distributeDaoMarksFromPaths("fire", 0, []string{"fire"}); len(got) != 0 {
		t.Fatalf("zero-total rewards = %v, want none", got)
	}
}

func TestVitalGuPathChanceIncreasesWithTribulationTier(t *testing.T) {
	tests := []struct {
		tribulationType string
		want            float64
	}{
		{"EARTH_CALAMITY", 0.80},
		{"HEAVENLY_TRIBULATION", 0.85},
		{"GRAND_TRIBULATION", 0.90},
		{"MYRIAD_TRIBULATION", 0.95},
		{"CHAOS_CALAMITY", 1.00},
		{"UNKNOWN", 0},
	}
	for _, tt := range tests {
		if got := vitalGuPathChance(tt.tribulationType); got != tt.want {
			t.Errorf("vitalGuPathChance(%q) = %.2f, want %.2f", tt.tribulationType, got, tt.want)
		}
	}
}

func TestChooseTribulationPathUsesTierScaledVitalAffinity(t *testing.T) {
	supported := []string{"fire", "water", "heaven"}
	vital := []string{"Heaven Path", "Fire Path"}

	path, ok := chooseTribulationPath(supported, vital, "EARTH_CALAMITY", 0.79)
	if !ok || path != "fire" {
		t.Fatalf("affinity roll chose (%q, %v), want fire", path, ok)
	}

	path, ok = chooseTribulationPath(supported, vital, "EARTH_CALAMITY", 0.80)
	if !ok || path != "water" {
		t.Fatalf("fallback roll chose (%q, %v), want water", path, ok)
	}

	path, ok = chooseTribulationPath(supported, vital, "CHAOS_CALAMITY", 0.99)
	if !ok || path != "fire" {
		t.Fatalf("higher-tier affinity roll chose (%q, %v), want fire", path, ok)
	}
}

func TestChooseTribulationPathIgnoresHeavenVitalPath(t *testing.T) {
	path, ok := chooseTribulationPath(
		[]string{"fire", "water", "heaven"},
		[]string{"Heaven Path"},
		"CHAOS_CALAMITY",
		0,
	)
	if !ok || path == heavenDaoPath {
		t.Fatalf("Heaven-only Vital Gu chose (%q, %v)", path, ok)
	}

	if path, ok := chooseTribulationPath([]string{"heaven"}, []string{"heaven"}, "CHAOS_CALAMITY", 0); ok || path != "" {
		t.Fatalf("Heaven-only pool chose (%q, %v), want no path", path, ok)
	}
}

func TestVitalGuPathForAscension(t *testing.T) {
	prev := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = prev })

	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"fire_gu": {
			1: {ID: "fire_gu", Name: "Hỏa Cổ", Path: []string{"Fire Path"}, Rank: 1},
		},
		"heaven_gu": {
			1: {ID: "heaven_gu", Name: "Thiên Đạo Cổ", Path: []string{"Heaven Path"}, Rank: 1},
		},
	}

	// 1. Player with vital Fire Gu -> returns "fire"
	pFire := &models.Player{
		GuInventory: models.GuInventory{
			{GuID: "fire_gu", Rank: 1, IsVital: true},
		},
	}
	if got := vitalGuPathForAscension(nil, pFire); got != "fire" {
		t.Fatalf("vitalGuPathForAscension with fire vital Gu = %q, want fire", got)
	}

	// 2. Player with vital Heaven Gu -> ignores heaven and returns an ordinary path (never heaven)
	pHeaven := &models.Player{
		GuInventory: models.GuInventory{
			{GuID: "heaven_gu", Rank: 1, IsVital: true},
		},
	}
	if got := vitalGuPathForAscension(nil, pHeaven); got == heavenDaoPath || got == "" {
		t.Fatalf("vitalGuPathForAscension with heaven vital Gu = %q, should not be heaven or empty", got)
	}

	// 3. Player with no vital Gu -> returns an ordinary path
	pNone := &models.Player{
		GuInventory: models.GuInventory{},
	}
	if got := vitalGuPathForAscension(nil, pNone); got == heavenDaoPath || got == "" {
		t.Fatalf("vitalGuPathForAscension with no vital Gu = %q, should not be heaven or empty", got)
	}
}
