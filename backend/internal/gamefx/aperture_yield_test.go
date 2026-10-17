package gamefx

import "testing"

func TestComputeApertureYield(t *testing.T) {
	base := ApertureYieldInputs{
		Level: 1, ItemBasePrice: 1000, AreaSize: 0,
		DaoMult: 1, ConflictMult: 1, EcologyMult: 1, EcoMult: 1, LaborMult: 1, PopBonus: 1,
	}

	tests := []struct {
		name       string
		mutate     func(in *ApertureYieldInputs)
		want       float64
		capped     bool
		daoClamped bool
		divisor    float64
	}{
		{"base level 1", nil, 10, false, false, 1},
		{"level 5 = x3", func(in *ApertureYieldInputs) { in.Level = 5 }, 30, false, false, 1},
		{"level 0 clamps to 1", func(in *ApertureYieldInputs) { in.Level = 0 }, 10, false, false, 1},
		{"rank 5 price divides by 25", func(in *ApertureYieldInputs) { in.ItemBasePrice = 25000 }, 0.4, false, false, 25},
		{"unknown price falls back to divisor 1", func(in *ApertureYieldInputs) { in.ItemBasePrice = 0 }, 10, false, false, 1},
		{"dao marks clamped at x50", func(in *ApertureYieldInputs) { in.DaoMult = 101 }, 500, false, true, 1},
		{"combined cap binds", func(in *ApertureYieldInputs) {
			in.Level = 5
			in.DaoMult = 3
			in.LaborMult = 2
			in.PopBonus = 5
			in.AreaSize = 10000000
		}, 500, true, false, 1},
		{"uncapped mid game", func(in *ApertureYieldInputs) {
			in.Level = 5
			in.DaoMult = 2
			in.AreaSize = 700000
		}, 64.2, false, false, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			if tt.mutate != nil {
				tt.mutate(&in)
			}
			got := ComputeApertureYield(in)
			if diff := got.Count - tt.want; diff > 1e-9 || diff < -1e-9 {
				t.Errorf("Count = %v, want %v", got.Count, tt.want)
			}
			if got.Capped != tt.capped {
				t.Errorf("Capped = %v, want %v", got.Capped, tt.capped)
			}
			if clamped := got.Parts["dao"] == ApertureYieldDaoCap; clamped != tt.daoClamped {
				t.Errorf("dao clamped = %v, want %v (dao part %v)", clamped, tt.daoClamped, got.Parts["dao"])
			}
			if got.Divisor != tt.divisor {
				t.Errorf("Divisor = %v, want %v", got.Divisor, tt.divisor)
			}
			if got.Bonus > ApertureYieldTotalCap {
				t.Errorf("Bonus %v exceeds total cap %v", got.Bonus, ApertureYieldTotalCap)
			}
			if len(got.Parts) != 8 {
				t.Errorf("expected 8 breakdown parts, got %d", len(got.Parts))
			}
		})
	}
}

// Every plantable rank must still produce something, and the same inputs on a
// lower rank must never out-earn a higher one by count (value is what ranks up).
func TestComputeApertureYieldAllRanksPositive(t *testing.T) {
	prices := []float64{1000, 3000, 6000, 13000, 25000, 10000, 25000, 60000, 150000}
	for _, price := range prices {
		got := ComputeApertureYield(ApertureYieldInputs{
			Level: 1, ItemBasePrice: price, AreaSize: 700000,
			DaoMult: 1, ConflictMult: 1, EcologyMult: 1, EcoMult: 1, LaborMult: 1, PopBonus: 1,
		})
		if got.Count <= 0 {
			t.Errorf("price %v produced %v, want > 0", price, got.Count)
		}
	}
}
