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

	tanThuTags := merchantBuyTags{Types: []string{"Gu", "Material", "Consumable", "Recipe"}, MaxRank: 3}
	if !tanThuTags.accepts("Material", 1, "boar_meat") {
		t.Fatal("rank 1 material should be accepted by tan thu merchant")
	}
	if !tanThuTags.accepts("Gu", 2, "GU_MOONLIGHT") {
		t.Fatal("rank 2 Gu should be accepted by tan thu merchant")
	}
	if tanThuTags.accepts("Gu", 4, "GU_FIRE_DRAGON") {
		t.Fatal("rank 4 Gu should be rejected by tan thu merchant max rank")
	}
}

func TestMerchantToDTO_Radius(t *testing.T) {
	roaming := merchantRowData{
		ID:        "roam_test",
		RegionID:  "dai_vuc_nam_cuong",
		X:         500,
		Y:         500,
		ProfileID: "roaming_trader",
		Kind:      "ROAMING",
	}
	tanThu := merchantRowData{
		ID:        "merchant_tan_thu_nam_cuong",
		RegionID:  "tan_thu_coc_nam_cuong",
		X:         500,
		Y:         500,
		ProfileID: "tan_thu_merchant",
		Kind:      "FIXED",
	}

	// At distance 0 (500, 500): both should be in range
	if _, ok := roaming.toDTO(500, 500); !ok {
		t.Fatal("roaming merchant should be in range at distance 0")
	}
	if _, ok := tanThu.toDTO(500, 500); !ok {
		t.Fatal("tan thu merchant should be in range at distance 0")
	}

	// At distance 8 (508, 500): roaming out of range (radius 2), tan thu in range (radius 15)
	if _, ok := roaming.toDTO(508, 500); ok {
		t.Fatal("roaming merchant should be out of range at distance 8")
	}
	if _, ok := tanThu.toDTO(508, 500); !ok {
		t.Fatal("tan thu merchant should be in range at distance 8")
	}

	// At distance 20 (520, 500): both out of range
	if _, ok := tanThu.toDTO(520, 500); ok {
		t.Fatal("tan thu merchant should be out of range at distance 20")
	}
}
