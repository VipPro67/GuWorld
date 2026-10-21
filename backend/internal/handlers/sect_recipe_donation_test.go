package handlers

import "testing"

func TestUniqueTrimmedIDs(t *testing.T) {
	got := uniqueTrimmedIDs([]string{
		" recipe_GU_A_1_1 ",
		"",
		"recipe_GU_B_2_1",
		"recipe_GU_A_1_1",
		"   ",
		"recipe_GU_B_2_1",
	})
	want := []string{"recipe_GU_A_1_1", "recipe_GU_B_2_1"}
	if len(got) != len(want) {
		t.Fatalf("uniqueTrimmedIDs length = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("uniqueTrimmedIDs[%d] = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

// A nil queryer means the recipe is not in the catalog, so the rank must come
// from the id suffix (legacy/orphaned rows) instead of defaulting to 1.
func TestRecipeTargetGuRankFallsBackToID(t *testing.T) {
	cases := []struct {
		recipeID string
		want     int
	}{
		{"recipe_GU_ALERT_BELL_3_2", 3},
		{"recipe_GU_BLOOD_BONE_SHIELD_6_1", 6},
		{"recipe_GU_MYRIAD_WATERS_RETURN_ORIGIN_IMMORTAL_9_1", 9},
		{"recipe_GU_WITHOUT_SUFFIX", 1},
		{"", 1},
	}
	for _, tc := range cases {
		if got := recipeTargetGuRank(nil, tc.recipeID); got != tc.want {
			t.Errorf("recipeTargetGuRank(nil, %q) = %d, want %d", tc.recipeID, got, tc.want)
		}
	}
}
