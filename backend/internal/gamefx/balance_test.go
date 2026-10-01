package gamefx

import (
	"math"
	"testing"
	"time"
)

func TestRankScale(t *testing.T) {
	tests := []struct {
		rank int
		want float64
	}{
		{-1, 1.0},
		{0, 1.0},
		{1, 1.0},
		{2, 2.0},
		{3, 5.0},
		{4, 20.0},
		{5, 100.0},
		{6, 10000.0},
		{7, 1000000.0},
		{8, 100000000.0},
		{9, 10000000000.0},
		{10, 10000000000.0},
	}

	for _, tt := range tests {
		got := RankScale(tt.rank)
		if got != tt.want {
			t.Errorf("RankScale(%d) = %v, want %v", tt.rank, got, tt.want)
		}
	}
}

func TestSealingStoneCostByRank(t *testing.T) {
	tests := []struct {
		rank int
		want int64
	}{
		{-1, 1},
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 5},
		{4, 20},
		{5, 100},
		{6, 10000},
		{7, 1000000},
		{8, 100000000},
		{9, 10000000000},
		{10, 10000000000},
	}

	for _, tt := range tests {
		got := SealingStoneCostByRank(tt.rank)
		if got != tt.want {
			t.Errorf("SealingStoneCostByRank(%d) = %d, want %d", tt.rank, got, tt.want)
		}
	}
}

func TestRefinementMasteryBonus(t *testing.T) {
	tests := []struct {
		realm int
		want  float64
	}{
		{0, 0.0},
		{1, 0.05},
		{2, 0.10},
		{3, 0.18},
		{4, 0.25},
		{5, 0.25},
	}

	for _, tt := range tests {
		got := RefinementMasteryBonus(tt.realm)
		if got != tt.want {
			t.Errorf("RefinementMasteryBonus(%d) = %v, want %v", tt.realm, got, tt.want)
		}
	}
}

func TestTargetPathMasteryBonus(t *testing.T) {
	tests := []struct {
		realm int
		want  float64
	}{
		{0, 0.0},
		{1, 0.02},
		{2, 0.05},
		{3, 0.08},
		{4, 0.10},
		{99, 0.10},
	}

	for _, tt := range tests {
		got := TargetPathMasteryBonus(tt.realm)
		if got != tt.want {
			t.Errorf("TargetPathMasteryBonus(%d) = %v, want %v", tt.realm, got, tt.want)
		}
	}
}

func TestRefinementImmortalPenalty(t *testing.T) {
	tests := []struct {
		rank int
		want float64
	}{
		{1, 1.0},
		{5, 1.0},
		{6, 2.0},
		{7, 5.0},
		{8, 20.0},
		{9, 100.0},
		{10, 100.0},
	}

	for _, tt := range tests {
		got := RefinementImmortalPenalty(tt.rank)
		if got != tt.want {
			t.Errorf("RefinementImmortalPenalty(%d) = %v, want %v", tt.rank, got, tt.want)
		}
	}
}

func TestRefinementDaoBonus(t *testing.T) {
	// Zero marks
	if got := RefinementDaoBonus(0, 100, 1); got != 0.0 {
		t.Errorf("RefinementDaoBonus(0, ...) = %v, want 0", got)
	}

	// 100 marks, 0 target path points, rank 1
	// sqrt(100) = 10; masteryMultiplier = 1.0 + 0 = 1.0; rankScale = 1.0
	// 0.00632 * 10 * 1.0 / 1.0 = 0.0632
	got := RefinementDaoBonus(100, 0, 1)
	want := 0.0632
	if got != want {
		t.Errorf("RefinementDaoBonus(100, 0, 1) = %v, want %v", got, want)
	}
}

func TestCalculateRefinementPathGain(t *testing.T) {
	tests := []struct {
		targetRank      int
		refinementRealm int
		want            int
	}{
		{targetRank: 1, refinementRealm: 0, want: 2},
		{targetRank: 1, refinementRealm: 1, want: 2},
		{targetRank: 2, refinementRealm: 1, want: 4},
		{targetRank: 6, refinementRealm: 1, want: 400},
		{targetRank: 1, refinementRealm: 2, want: 0},
		{targetRank: 2, refinementRealm: 2, want: 3},
		{targetRank: 3, refinementRealm: 2, want: 9},
		{targetRank: 5, refinementRealm: 2, want: 60},
		{targetRank: 6, refinementRealm: 2, want: 300},
		{targetRank: 1, refinementRealm: 3, want: 0},
		{targetRank: 2, refinementRealm: 3, want: 0},
		{targetRank: 3, refinementRealm: 3, want: 0},
		{targetRank: 4, refinementRealm: 3, want: 0},
		{targetRank: 5, refinementRealm: 3, want: 40},
		{targetRank: 6, refinementRealm: 3, want: 200},
		{targetRank: 4, refinementRealm: 4, want: 0},
		{targetRank: 5, refinementRealm: 4, want: 0},
		{targetRank: 6, refinementRealm: 4, want: 100},
		{targetRank: 9, refinementRealm: 4, want: 10000},
	}

	for _, tt := range tests {
		got := CalculateRefinementPathGain(tt.targetRank, tt.refinementRealm)
		if got != tt.want {
			t.Errorf("CalculateRefinementPathGain(%d, %d) = %d, want %d", tt.targetRank, tt.refinementRealm, got, tt.want)
		}
	}
}

func TestPathMasteryGuUseBaseGain(t *testing.T) {
	wants := []int{1, 1, 2, 5, 10, 20, 50, 100, 200, 300, 300}
	for rank, want := range wants {
		if got := PathMasteryGuUseBaseGain(rank); got != want {
			t.Errorf("PathMasteryGuUseBaseGain(%d) = %d, want %d", rank, got, want)
		}
	}
}

func TestDeductionRealmMultiplier(t *testing.T) {
	// wisdom=2, refine=1, target=3
	// 1 + 2*3.0 + 1*2.0 + 3*1.0 = 1 + 6 + 2 + 3 = 12.0
	got := DeductionRealmMultiplier(2, 1, 3)
	want := 12.0
	if got != want {
		t.Errorf("DeductionRealmMultiplier(2, 1, 3) = %v, want %v", got, want)
	}
}

func TestLootMultipliers(t *testing.T) {
	if got := LootChanceMultiplier(1); got != 1.0 {
		t.Errorf("LootChanceMultiplier(1) = %v, want 1.0", got)
	}
	if got := LootChanceMultiplier(9); got != 4.0 {
		t.Errorf("LootChanceMultiplier(9) = %v, want 4.0", got)
	}
	if got := LootQuantityMultiplier(1); got != 1.0 {
		t.Errorf("LootQuantityMultiplier(1) = %v, want 1.0", got)
	}
	if got := LootQuantityMultiplier(9); got != 8.0 {
		t.Errorf("LootQuantityMultiplier(9) = %v, want 8.0", got)
	}
}

func TestRenameCost(t *testing.T) {
	costType, amount := RenameCost(2)
	if costType != "free" || amount != 0 {
		t.Errorf("RenameCost(2) = (%s, %v), want (free, 0)", costType, amount)
	}

	costType, amount = RenameCost(5)
	if costType != "primeval" || amount != 100000 {
		t.Errorf("RenameCost(5) = (%s, %v), want (primeval, 100000)", costType, amount)
	}

	costType, amount = RenameCost(6)
	if costType != "immortal" || amount != 100 {
		t.Errorf("RenameCost(6) = (%s, %v), want (immortal, 100)", costType, amount)
	}
}

func TestRegionalWallDebuff(t *testing.T) {
	pct, dur := RegionalWallDebuff(1, 1)
	if math.Abs(pct-(-0.10)) > 1e-6 || dur != 5*time.Hour {
		t.Errorf("RegionalWallDebuff(1, 1) = (%v, %v), want (-0.10, 5h)", pct, dur)
	}

	pct, dur = RegionalWallDebuff(6, 2)
	// base 0.45 + (2-1)*0.02 = 0.47 -> -0.47, 20h
	if math.Abs(pct-(-0.47)) > 1e-6 || dur != 20*time.Hour {
		t.Errorf("RegionalWallDebuff(6, 2) = (%v, %v), want (-0.47, 20h)", pct, dur)
	}
}

func TestInstantDeductionChance(t *testing.T) {
	tests := []struct {
		name                   string
		guRank, targetRank     int
		base, step, minC, maxC float64
		want                   float64
	}{
		{"equal rank", 3, 3, 0.05, 0.02, 0.01, 0.2, 0.05},
		{"higher gu rank", 5, 3, 0.05, 0.02, 0.01, 0.2, 0.09},
		{"lower gu rank", 3, 5, 0.05, 0.02, 0.01, 0.2, 0.01},
		{"clamp max", 9, 1, 0.05, 0.02, 0.01, 0.2, 0.2},
		{"clamp min", 1, 9, 0.05, 0.02, 0.01, 0.2, 0.01},
	}

	for _, tt := range tests {
		got := InstantDeductionChance(tt.guRank, tt.targetRank, tt.base, tt.step, tt.minC, tt.maxC)
		if math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("%s: InstantDeductionChance(%d, %d, %v, %v, %v, %v) = %v, want %v",
				tt.name, tt.guRank, tt.targetRank, tt.base, tt.step, tt.minC, tt.maxC, got, tt.want)
		}
	}
}
