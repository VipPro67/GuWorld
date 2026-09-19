package handlers

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGetRenameCost(t *testing.T) {
	tests := []struct {
		rank         int
		wantCostType string
		wantCost     float64
	}{
		{rank: 1, wantCostType: "free", wantCost: 0},
		{rank: 2, wantCostType: "free", wantCost: 0},
		{rank: 3, wantCostType: "primeval", wantCost: 5000},
		{rank: 4, wantCostType: "primeval", wantCost: 20000},
		{rank: 5, wantCostType: "primeval", wantCost: 100000},
		{rank: 6, wantCostType: "immortal", wantCost: 100},
		{rank: 7, wantCostType: "immortal", wantCost: 500},
		{rank: 8, wantCostType: "immortal", wantCost: 2500},
		{rank: 9, wantCostType: "immortal", wantCost: 10000},
	}

	for _, tt := range tests {
		gotType, gotCost := GetRenameCost(tt.rank)
		if gotType != tt.wantCostType || gotCost != tt.wantCost {
			t.Errorf("GetRenameCost(%d) = (%s, %v), want (%s, %v)", tt.rank, gotType, gotCost, tt.wantCostType, tt.wantCost)
		}
	}
}

func TestGetImmortalPenaltyRates(t *testing.T) {
	rates := []struct {
		rank       int
		wantDao    float64
		wantMaster float64
		wantHP     float64
		wantStress float64
	}{
		{rank: 6, wantDao: 0.30, wantMaster: 0.30, wantHP: 0.20, wantStress: 30.0},
		{rank: 7, wantDao: 0.40, wantMaster: 0.40, wantHP: 0.30, wantStress: 50.0},
		{rank: 8, wantDao: 0.50, wantMaster: 0.50, wantHP: 0.40, wantStress: 80.0},
		{rank: 9, wantDao: 0.70, wantMaster: 0.70, wantHP: 0.50, wantStress: 100.0},
	}

	for _, tt := range rates {
		dao, master, hp, stress := GetImmortalPenaltyRates(tt.rank)
		if math.Abs(dao-tt.wantDao) > 0.001 || math.Abs(master-tt.wantMaster) > 0.001 || math.Abs(hp-tt.wantHP) > 0.001 || math.Abs(stress-tt.wantStress) > 0.001 {
			t.Errorf("GetImmortalPenaltyRates(%d) = (%v, %v, %v, %v), want (%v, %v, %v, %v)", tt.rank, dao, master, hp, stress, tt.wantDao, tt.wantMaster, tt.wantHP, tt.wantStress)
		}
	}
}

func TestApplyImmortalRenamePenalty(t *testing.T) {
	dao := models.DaoMarksMap{
		"sword": 10000,
		"fire":  5000,
	}
	masteries := models.PathMasteriesMap{
		"sword": 1000,
		"fire":  500,
	}
	apertureHP := 10000.0

	// Rank 6: 30% dao & masteries loss, 20% hp loss, 30 stress
	newDao, newMasteries, newApertureHP, stressGain := ApplyImmortalRenamePenalty(6, dao, masteries, apertureHP)
	if newDao["sword"] != 7000 || newDao["fire"] != 3500 {
		t.Errorf("Rank 6: Expected 30%% dao reduction, got %v", newDao)
	}
	if newMasteries["sword"] != 700 || newMasteries["fire"] != 350 {
		t.Errorf("Rank 6: Expected 30%% masteries reduction, got %v", newMasteries)
	}
	if math.Abs(newApertureHP-8000.0) > 0.001 {
		t.Errorf("Rank 6: Expected aperture HP 8000, got %v", newApertureHP)
	}
	if math.Abs(stressGain-30.0) > 0.001 {
		t.Errorf("Rank 6: Expected stressGain 30, got %v", stressGain)
	}

	// Rank 7: 40% dao & masteries loss, 30% hp loss, 50 stress
	newDao, newMasteries, newApertureHP, stressGain = ApplyImmortalRenamePenalty(7, dao, masteries, apertureHP)
	if newDao["sword"] != 6000 || newDao["fire"] != 3000 {
		t.Errorf("Rank 7: Expected 40%% dao reduction, got %v", newDao)
	}
	if newMasteries["sword"] != 600 || newMasteries["fire"] != 300 {
		t.Errorf("Rank 7: Expected 40%% masteries reduction, got %v", newMasteries)
	}
	if math.Abs(newApertureHP-7000.0) > 0.001 {
		t.Errorf("Rank 7: Expected aperture HP 7000, got %v", newApertureHP)
	}
	if math.Abs(stressGain-50.0) > 0.001 {
		t.Errorf("Rank 7: Expected stressGain 50, got %v", stressGain)
	}

	// Rank 8: 50% dao & masteries loss, 40% hp loss, 80 stress
	newDao, newMasteries, newApertureHP, stressGain = ApplyImmortalRenamePenalty(8, dao, masteries, apertureHP)
	if newDao["sword"] != 5000 || newDao["fire"] != 2500 {
		t.Errorf("Rank 8: Expected 50%% dao reduction, got %v", newDao)
	}
	if newMasteries["sword"] != 500 || newMasteries["fire"] != 250 {
		t.Errorf("Rank 8: Expected 50%% masteries reduction, got %v", newMasteries)
	}
	if math.Abs(newApertureHP-6000.0) > 0.001 {
		t.Errorf("Rank 8: Expected aperture HP 6000, got %v", newApertureHP)
	}
	if math.Abs(stressGain-80.0) > 0.001 {
		t.Errorf("Rank 8: Expected stressGain 80, got %v", stressGain)
	}

	// Rank 9: 70% dao & masteries loss, 50% hp loss, 100 stress
	newDao, newMasteries, newApertureHP, stressGain = ApplyImmortalRenamePenalty(9, dao, masteries, apertureHP)
	if newDao["sword"] != 3000 || newDao["fire"] != 1500 {
		t.Errorf("Rank 9: Expected 70%% dao reduction, got %v", newDao)
	}
	if newMasteries["sword"] != 300 || newMasteries["fire"] != 150 {
		t.Errorf("Rank 9: Expected 70%% masteries reduction, got %v", newMasteries)
	}
	if math.Abs(newApertureHP-5000.0) > 0.001 {
		t.Errorf("Rank 9: Expected aperture HP 5000, got %v", newApertureHP)
	}
	if math.Abs(stressGain-100.0) > 0.001 {
		t.Errorf("Rank 9: Expected stressGain 100, got %v", stressGain)
	}
}
