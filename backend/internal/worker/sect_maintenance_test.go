package worker

import (
	"encoding/json"
	"testing"
)

func TestMaterialFeeIsFlatPerTwoHundredItems(t *testing.T) {
	cases := []struct {
		name string
		qty  int64
		want int64
	}{
		{"empty vault", 0, 0},
		{"single item", 1, 0},
		{"just under one unit", 199, 0},
		{"exactly one unit", 200, 1},
		{"just under two units", 399, 1},
		{"two units", 400, 2},
		{"bulk stockpile", 50000, 250},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			materials, _, _, _, discount := computeVaultFees(vaultFeeInput{
				MaterialQty:   tc.qty,
				TreasuryLevel: 1,
			})
			if materials != tc.want {
				t.Fatalf("MaterialsFee(%d items) = %d, want %d", tc.qty, materials, tc.want)
			}
			if discount != 0 {
				t.Fatalf("Linh tài must bypass the treasury discount, got discount %d", discount)
			}
		})
	}
}

func TestMaterialFeeIsExemptFromTreasuryDiscount(t *testing.T) {
	for _, treasuryLevel := range []int{1, 5, 9} {
		materials, _, _, _, _ := computeVaultFees(vaultFeeInput{
			MaterialQty:   4000,
			TreasuryLevel: treasuryLevel,
		})
		if materials != 20 {
			t.Fatalf("treasury %d: 4000 items cost %d Quỹ, want 20 (flat, never discounted)",
				treasuryLevel, materials)
		}
	}
}

func TestVaultFeeSplitsAndDiscounts(t *testing.T) {
	// Treasury level 1 keeps 50% of every discounted category.
	materials, mortal, immortal, recipes, discount := computeVaultFees(vaultFeeInput{
		MaterialQty:   5000,                            // 5000/200 = 25, flat
		MortalGu:      []rankedQty{{Rank: 3, Qty: 10}}, // 3*10*10 = 300 -> 150
		ImmortalGu:    []rankedQty{{Rank: 6, Qty: 1}},  // 6*200 = 1200 -> 600
		RecipeRanks:   []int{4, 4},                     // 2*4*50 = 400 -> 200
		TreasuryLevel: 1,
	})
	if materials != 25 {
		t.Errorf("materials = %d, want 25", materials)
	}
	if mortal != 150 {
		t.Errorf("mortal gu = %d, want 150", mortal)
	}
	if immortal != 600 {
		t.Errorf("immortal gu = %d, want 600", immortal)
	}
	if recipes != 200 {
		t.Errorf("recipes = %d, want 200", recipes)
	}
	if discount != 950 {
		t.Errorf("discount = %d, want 950 (150+600+200, materials excluded)", discount)
	}
}

func TestVaultFeeTreasuryLevelNineKeepsTenPercent(t *testing.T) {
	_, mortal, immortal, recipes, discount := computeVaultFees(vaultFeeInput{
		MortalGu:      []rankedQty{{Rank: 3, Qty: 10}}, // 300 -> 30
		ImmortalGu:    []rankedQty{{Rank: 6, Qty: 1}},  // 1200 -> 120
		RecipeRanks:   []int{4, 4},                     // 400 -> 40
		TreasuryLevel: 9,
	})
	if mortal != 30 || immortal != 120 || recipes != 40 {
		t.Fatalf("treasury 9 fees = %d/%d/%d, want 30/120/40", mortal, immortal, recipes)
	}
	if discount != 1710 {
		t.Fatalf("discount = %d, want 1710 (270+1080+360)", discount)
	}
}

func TestVaultFeeNormalisesBadRanks(t *testing.T) {
	_, mortal, _, recipes, _ := computeVaultFees(vaultFeeInput{
		MortalGu:      []rankedQty{{Rank: 0, Qty: 50}},
		RecipeRanks:   []int{0},
		TreasuryLevel: 9,
	})
	// rank 0 is treated as rank 1: 1*10*50 = 500, then 90% off -> 50.
	if mortal != 50 {
		t.Errorf("mortal gu = %d, want 50", mortal)
	}
	// rank 0 recipe -> rank 1: 1*50 = 50, then 90% off -> 5.
	if recipes != 5 {
		t.Errorf("recipes = %d, want 5", recipes)
	}
}

func TestCalculateSectMaintenanceBreakdown(t *testing.T) {
	// Level 3 sect: 3 * 2000 = 6000
	// 10 members: 10 * 200 = 2000
	// 1000 Linh tài: 1000 / 200 = 5 (flat, undiscounted)
	// 10 Rank 3 mortal Gu: 3*10*10 = 300 -> 150 at treasury 1
	// 1 Rank 6 immortal Gu: 6*200 = 1200 -> 600 at treasury 1
	// 2 Rank 4 recipes: 2*4*50 = 400 -> 200 at treasury 1
	var b SectMaintenanceBreakdown
	b.SectLevel = 3
	b.LevelFee = 3 * 2000
	b.MemberCount = 10
	b.MemberFee = 10 * 200
	b.MaterialsCount = 1000
	b.MortalGuCount = 10
	b.ImmortalGuCount = 1
	b.RecipesCount = 2

	materials, mortal, immortal, recipes, saved := computeVaultFees(vaultFeeInput{
		MaterialQty:   b.MaterialsCount,
		MortalGu:      []rankedQty{{Rank: 3, Qty: 10}},
		ImmortalGu:    []rankedQty{{Rank: 6, Qty: 1}},
		RecipeRanks:   []int{4, 4},
		TreasuryLevel: 1,
	})
	b.MaterialsFee = materials
	b.MortalGuFee = mortal
	b.ImmortalGuFee = immortal
	b.RecipesFee = recipes
	b.TreasuryDiscount = saved
	b.TotalFee = b.LevelFee + b.MemberFee + b.MaterialsFee + b.MortalGuFee + b.ImmortalGuFee + b.RecipesFee

	// 6000 + 2000 + 5 + 150 + 600 + 200 = 8955
	if b.TotalFee != 8955 {
		t.Fatalf("TotalFee = %d, want 8955", b.TotalFee)
	}
	if b.MaterialsFee != 5 {
		t.Fatalf("MaterialsFee = %d, want 5", b.MaterialsFee)
	}
	if b.TreasuryDiscount != 950 {
		t.Fatalf("TreasuryDiscount = %d, want 950", b.TreasuryDiscount)
	}
}

func TestClampBuildings(t *testing.T) {
	t.Run("caps every building at the new level", func(t *testing.T) {
		raw, err := json.Marshal(map[string]int{
			"main_hall":       5,
			"spirit_vein":     5,
			"treasury":        3,
			"training_ground": 1,
			"defensive_array": 4,
		})
		if err != nil {
			t.Fatal(err)
		}
		got, err := clampBuildings(raw, 4)
		if err != nil {
			t.Fatalf("clampBuildings: %v", err)
		}
		var out map[string]int
		if err := json.Unmarshal(got, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		want := map[string]int{
			"main_hall":       4,
			"spirit_vein":     4,
			"treasury":        3,
			"training_ground": 1,
			"defensive_array": 4,
		}
		for name, wantLevel := range want {
			if out[name] != wantLevel {
				t.Errorf("%s = %d, want %d", name, out[name], wantLevel)
			}
		}
	})

	t.Run("materialises a missing main_hall", func(t *testing.T) {
		got, err := clampBuildings([]byte(`{"defensive_array": 3}`), 2)
		if err != nil {
			t.Fatalf("clampBuildings: %v", err)
		}
		var out map[string]int
		if err := json.Unmarshal(got, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out["main_hall"] != 2 {
			t.Errorf("main_hall = %d, want 2", out["main_hall"])
		}
		if out["defensive_array"] != 2 {
			t.Errorf("defensive_array = %d, want 2", out["defensive_array"])
		}
	})

	t.Run("handles empty and malformed input", func(t *testing.T) {
		got, err := clampBuildings(nil, 3)
		if err != nil {
			t.Fatalf("clampBuildings(nil): %v", err)
		}
		var out map[string]int
		if err := json.Unmarshal(got, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(out) != 1 || out["main_hall"] != 3 {
			t.Errorf("empty input -> %v, want {main_hall:3}", out)
		}

		// A JSON null must not panic on the re-assignment below.
		got, err = clampBuildings([]byte(`null`), 2)
		if err != nil {
			t.Fatalf("clampBuildings(null): %v", err)
		}
		if err := json.Unmarshal(got, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out["main_hall"] != 2 {
			t.Errorf("null input -> main_hall %d, want 2", out["main_hall"])
		}

		if _, err := clampBuildings([]byte(`{not json`), 2); err == nil {
			t.Error("malformed buildings JSON must return an error, not silently reset the base")
		}
	})
}

// A level-1 sect is disbanded exactly when the unpaid-day count reaches the
// grace window — this is the data-loss boundary, so pin it.
func TestSectShouldDisband(t *testing.T) {
	cases := []struct {
		unpaidDays int
		want       bool
	}{
		{0, false},
		{sectDisbandGraceDays - 1, false},
		{sectDisbandGraceDays, true},
		{sectDisbandGraceDays + 1, true},
	}
	for _, tc := range cases {
		if got := sectShouldDisband(tc.unpaidDays); got != tc.want {
			t.Fatalf("sectShouldDisband(%d) = %v, want %v", tc.unpaidDays, got, tc.want)
		}
	}
}
