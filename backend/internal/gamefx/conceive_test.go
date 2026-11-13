package gamefx

import (
	"math"
	"testing"
)

func TestRequiredAttainmentRealm(t *testing.T) {
	tests := []struct {
		rank int
		want int
	}{
		{1, 0},
		{2, 0},
		{3, 1},
		{4, 1},
		{5, 2},
		{6, 2},
		{7, 3},
		{8, 3},
		{9, 4},
	}

	for _, tt := range tests {
		got := RequiredAttainmentRealm(tt.rank)
		if got != tt.want {
			t.Errorf("RequiredAttainmentRealm(%d) = %d, want %d", tt.rank, got, tt.want)
		}
	}
}

func TestCalculateConceiveAccuracy(t *testing.T) {
	// Rank 3 (requires realm 1). Target realm 1, wisdom 1, no catalyst:
	// 0.60 + 0 (diff) + 1*0.04 (wisdom) = 0.64
	acc1 := CalculateConceiveAccuracy(3, 1, 1, 0, 0.0)
	if math.Abs(acc1-0.64) > 0.001 {
		t.Errorf("CalculateConceiveAccuracy(3, 1, 1, 0, 0) = %v, want ~0.64", acc1)
	}

	// With catalyst (+0.20) -> 0.84
	acc2 := CalculateConceiveAccuracy(3, 1, 1, 0, 0.20)
	if math.Abs(acc2-0.84) > 0.001 {
		t.Errorf("CalculateConceiveAccuracy with catalyst = %v, want ~0.84", acc2)
	}

	// Rank 5 (requires realm 2). Target realm 0 (diff -2), wisdom 2:
	// 0.60 + (-2 * 0.20) + (2 * 0.04) = 0.28
	acc3 := CalculateConceiveAccuracy(5, 0, 2, 0, 0.0)
	if math.Abs(acc3-0.28) > 0.001 {
		t.Errorf("CalculateConceiveAccuracy(5, 0, 2, 0, 0) = %v, want ~0.28", acc3)
	}

	// Ceiling check: should never exceed ConceiveAccuracyMax (0.88)
	accMax := CalculateConceiveAccuracy(3, 4, 4, 100000, 0.50)
	if accMax > ConceiveAccuracyMax {
		t.Errorf("Accuracy exceeded max cap: %v > %v", accMax, ConceiveAccuracyMax)
	}

	// Floor check: should never fall below ConceiveAccuracyMin (0.15)
	accMin := CalculateConceiveAccuracy(9, 0, 0, 0, -1.0)
	if accMin < ConceiveAccuracyMin {
		t.Errorf("Accuracy fell below min floor: %v < %v", accMin, ConceiveAccuracyMin)
	}
}

func TestCalculateBacklashRisk(t *testing.T) {
	// Mortal trying immortal
	if risk := CalculateBacklashRisk(5, 6, 2, 2); risk < 1.0 {
		t.Errorf("Mortal trying immortal risk = %v, want 1.0", risk)
	}

	// Player rank 2 trying rank 3 (diff = 1) -> 0.35 + 1*0.25 = 0.60
	if risk := CalculateBacklashRisk(2, 3, 1, 1); math.Abs(risk-0.60) > 0.001 {
		t.Errorf("Exceeding rank diff 1 risk = %v, want 0.60", risk)
	}

	// Player rank 2 trying rank 4 (diff = 2) -> 0.35 + 2*0.25 = 0.85
	if risk := CalculateBacklashRisk(2, 4, 1, 1); math.Abs(risk-0.85) > 0.001 {
		t.Errorf("Exceeding rank diff 2 risk = %v, want 0.85", risk)
	}

	// Safe condition: player rank 4, target rank 3, realm 1
	if risk := CalculateBacklashRisk(4, 3, 1, 1); risk != 0.0 {
		t.Errorf("Safe condition risk = %v, want 0.0", risk)
	}

	// Equal rank (rank 4) but lacking both wisdom and refine realm (0) -> 0.15
	if risk := CalculateBacklashRisk(4, 4, 0, 0); math.Abs(risk-0.15) > 0.001 {
		t.Errorf("Equal rank lacking realms risk = %v, want 0.15", risk)
	}
}

func TestConceiveCosts(t *testing.T) {
	if cost := ConceiveThoughtCost(1); cost != 30 {
		t.Errorf("Thought cost rank 1 = %v, want 30", cost)
	}
	if cost := ConceiveThoughtCost(3); cost != 80 {
		t.Errorf("Thought cost rank 3 = %v, want 80", cost)
	}
	if cost := ConceiveThoughtCost(5); cost != 200 {
		t.Errorf("Thought cost rank 5 = %v, want 200", cost)
	}
	if cost := ConceiveThoughtCost(6); cost != 400 {
		t.Errorf("Thought cost rank 6 = %v, want 400", cost)
	}

	if cost := ConceiveStoneCost(1); cost != 50 {
		t.Errorf("Stone cost rank 1 = %v, want 50", cost)
	}
	if cost := ConceiveStoneCost(5); cost != 10000 {
		t.Errorf("Stone cost rank 5 = %v, want 10000", cost)
	}
	if cost := ConceiveStoneCost(6); cost != 0 {
		t.Errorf("Stone cost rank 6 = %v, want 0", cost)
	}
	if cost := ConceiveImmortalStoneCost(6); cost != 100 {
		t.Errorf("Immortal stone cost rank 6 = %v, want 100", cost)
	}
}

func TestCatalystBonusesAndQuantities(t *testing.T) {
	// Gu catalyst
	if b := GuCatalystBonus(3, 3); math.Abs(b-0.20) > 0.001 {
		t.Errorf("Gu catalyst same rank bonus = %v, want 0.20", b)
	}
	if b := GuCatalystBonus(2, 3); math.Abs(b-0.16) > 0.001 {
		t.Errorf("Gu catalyst rank-1 bonus = %v, want 0.16", b)
	}
	if b := GuCatalystBonus(1, 3); math.Abs(b-0.12) > 0.001 {
		t.Errorf("Gu catalyst rank-2 bonus = %v, want 0.12", b)
	}

	// Material catalyst
	if b := MaterialCatalystBonus(3, 3); math.Abs(b-0.12) > 0.001 {
		t.Errorf("Material catalyst same rank bonus = %v, want 0.12", b)
	}
	if b := MaterialCatalystBonus(2, 3); math.Abs(b-0.08) > 0.001 {
		t.Errorf("Material catalyst lower rank bonus = %v, want 0.08", b)
	}

	// Material required qty
	if qty := MaterialCatalystRequiredQty(3, 3); qty != 10 {
		t.Errorf("Mortal same rank qty = %d, want 10", qty)
	}
	if qty := MaterialCatalystRequiredQty(6, 6); qty != 5 {
		t.Errorf("Immortal same rank qty = %d, want 5", qty)
	}
	if qty := MaterialCatalystRequiredQty(2, 3); qty != 25 {
		t.Errorf("Rank-1 qty = %d, want 25", qty)
	}
	if qty := MaterialCatalystRequiredQty(1, 3); qty != 50 {
		t.Errorf("Rank-2 qty = %d, want 50", qty)
	}
}
