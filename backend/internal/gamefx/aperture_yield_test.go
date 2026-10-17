package gamefx

import "testing"

func TestComputeApertureYield(t *testing.T) {
	base := ApertureYieldInputs{
		Level: 1, AreaSize: 0,
		DaoMult: 1, ConflictMult: 1, EcologyMult: 1, EcoMult: 1, LaborMult: 1, PopBonus: 1,
	}

	tests := []struct {
		name       string
		mutate     func(in *ApertureYieldInputs)
		want       float64
		capped     bool
		daoClamped bool
	}{
		{"base level 1", nil, 10, false, false},
		{"level 5 = x3", func(in *ApertureYieldInputs) { in.Level = 5 }, 30, false, false},
		{"level 0 clamps to 1", func(in *ApertureYieldInputs) { in.Level = 0 }, 10, false, false},
		{"dao marks clamped at x50", func(in *ApertureYieldInputs) { in.DaoMult = 101 }, 500, false, true},
		{"combined cap binds", func(in *ApertureYieldInputs) {
			in.Level = 5
			in.DaoMult = 3
			in.LaborMult = 2
			in.PopBonus = 5
			in.AreaSize = 10000000
		}, 500, true, false},
		{"uncapped mid game", func(in *ApertureYieldInputs) {
			in.Level = 5
			in.DaoMult = 2
			in.AreaSize = 700000
		}, 64.2, false, false},
		{"debuffed node keeps the 10 floor", func(in *ApertureYieldInputs) {
			in.ConflictMult = 0.55
			in.EcologyMult = 0.25
			in.EcoMult = 0.2
		}, 10, false, false},
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
			if got.Bonus > ApertureYieldTotalCap {
				t.Errorf("Bonus %v exceeds total cap %v", got.Bonus, ApertureYieldTotalCap)
			}
			if len(got.Parts) != 8 {
				t.Errorf("expected 8 breakdown parts, got %d", len(got.Parts))
			}
		})
	}
}

// Every rank is planted with the same 100 seeds and starts from the same 10
// per-harvest base, so a plain node always repays its seed within 10 harvests.
func TestComputeApertureYieldRepaysSeedWithinTenHarvests(t *testing.T) {
	got := ComputeApertureYield(ApertureYieldInputs{
		Level: 1, AreaSize: 0,
		DaoMult: 1, ConflictMult: 1, EcologyMult: 1, EcoMult: 1, LaborMult: 1, PopBonus: 1,
	})
	if got.Count*ApertureSeedRepayHarvests < 100 {
		t.Errorf("plain node yields %v/harvest, %v over %d harvests < 100 seed",
			got.Count, got.Count*ApertureSeedRepayHarvests, ApertureSeedRepayHarvests)
	}
}
