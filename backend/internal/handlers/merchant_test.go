package handlers

import "testing"

func TestMerchantBuyRatio(t *testing.T) {
	cases := []struct {
		name               string
		bought, quota      int
		start, floor, want float64
	}{
		{"fresh demand uses start", 0, 100, 0.40, 0.10, 0.40},
		{"half quota is midpoint", 50, 100, 0.40, 0.10, 0.25},
		{"full quota hits floor", 100, 100, 0.40, 0.10, 0.10},
		{"over quota clamps to floor", 250, 100, 0.40, 0.10, 0.10},
		{"no quota (unlimited) stays start", 999, 0, 0.40, 0.10, 0.40},
		{"flat profile stays flat", 50, 100, 0.10, 0.10, 0.10},
	}
	for _, tc := range cases {
		got := merchantBuyRatio(tc.bought, tc.quota, tc.start, tc.floor)
		if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestMerchantBuyTagsAccepts(t *testing.T) {
	mat := merchantBuyTags{Types: []string{"Material", "Consumable"}, MinRank: 1, MaxRank: 5}
	if !mat.accepts("Material", 3, "x") {
		t.Fatal("rank 3 material should be accepted")
	}
	if mat.accepts("Material", 6, "x") {
		t.Fatal("rank 6 material should be rejected by max_rank")
	}
	if mat.accepts("Gu", 3, "x") {
		t.Fatal("Gu should be rejected when only materials are wanted")
	}
	allow := merchantBuyTags{ItemIDs: []string{"needle"}}
	if !allow.accepts("Gu", 9, "needle") {
		t.Fatal("allow-list id should be accepted at any rank")
	}
	if allow.accepts("Gu", 1, "other") {
		t.Fatal("non-listed id should be rejected")
	}
}
