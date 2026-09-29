package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestRefineInsightStacksPerRank(t *testing.T) {
	p := &models.Player{
		CharacterState: models.CharacterState{
			Statuses: models.StatusList{
				{ID: "refine_insight_3", Value: 5},
				{ID: "refine_insight_4", Value: 2},
			},
		},
	}
	if got := refineInsightStacks(p, 3); got != 5 {
		t.Errorf("refineInsightStacks(R3) = %d, want 5", got)
	}
	if got := refineInsightStacks(p, 4); got != 2 {
		t.Errorf("refineInsightStacks(R4) = %d, want 2", got)
	}
	if got := refineInsightStacks(p, 5); got != 0 {
		t.Errorf("refineInsightStacks(R5) = %d, want 0 (rank khác không ké)", got)
	}
	// Bonus 1%/stack, không cap.
	if bonus := float64(refineInsightStacks(p, 3)) * RefineInsightBonusPerStack; bonus != 0.05 {
		t.Errorf("insight bonus R3 = %v, want 0.05", bonus)
	}
	if RefineInsightBonusPerStack != 0.01 {
		t.Errorf("RefineInsightBonusPerStack = %v, want 0.01", RefineInsightBonusPerStack)
	}
}

func TestCalculateRefinementPathGain(t *testing.T) {
	tests := []struct {
		targetRank      int
		refinementRealm int
		want            int
	}{
		// refinementRealm = 0 (Ordinary)
		{targetRank: 1, refinementRealm: 0, want: 2},
		{targetRank: 2, refinementRealm: 0, want: 5},
		{targetRank: 3, refinementRealm: 0, want: 15},
		{targetRank: 4, refinementRealm: 0, want: 40},
		{targetRank: 5, refinementRealm: 0, want: 100},
		{targetRank: 6, refinementRealm: 0, want: 500},
		{targetRank: 7, refinementRealm: 0, want: 2500},
		{targetRank: 8, refinementRealm: 0, want: 15000},
		{targetRank: 9, refinementRealm: 0, want: 50000},

		// refinementRealm = 1 (Master: points 100-999)
		{targetRank: 1, refinementRealm: 1, want: 2},   // Rank 1 gives gain for realm < 2 (points < 1000)
		{targetRank: 2, refinementRealm: 1, want: 4},   // base 5 * 0.8 = 4.0 -> 4
		{targetRank: 5, refinementRealm: 1, want: 80},  // base 100 * 0.8 = 80
		{targetRank: 6, refinementRealm: 1, want: 400}, // base 500 * 0.8 = 400

		// refinementRealm = 2 (Grandmaster: points 1000-9999)
		{targetRank: 1, refinementRealm: 2, want: 0},   // Rank 1 capped at realm 2 (>= 1000)
		{targetRank: 2, refinementRealm: 2, want: 3},   // base 5 * 0.6 = 3.0 -> 3
		{targetRank: 3, refinementRealm: 2, want: 9},   // base 15 * 0.6 = 9.0 -> 9
		{targetRank: 5, refinementRealm: 2, want: 60},  // base 100 * 0.6 = 60
		{targetRank: 6, refinementRealm: 2, want: 300}, // base 500 * 0.6 = 300

		// refinementRealm = 3 (Great Grandmaster: points 10000-99999)
		{targetRank: 1, refinementRealm: 3, want: 0},
		{targetRank: 2, refinementRealm: 3, want: 0},   // Rank 2 capped at realm 3 (>= 10000)
		{targetRank: 3, refinementRealm: 3, want: 0},   // Rank 3 capped at realm 3 (>= 10000)
		{targetRank: 4, refinementRealm: 3, want: 0},  // base 40 * 0.4 = 16
		{targetRank: 5, refinementRealm: 3, want: 40},  // base 100 * 0.4 = 40
		{targetRank: 6, refinementRealm: 3, want: 200}, // base 500 * 0.4 = 200

		// refinementRealm = 4 (Supreme Grandmaster: points >= 100000)
		{targetRank: 4, refinementRealm: 4, want: 0},
		{targetRank: 5, refinementRealm: 4, want: 0},
		{targetRank: 6, refinementRealm: 4, want: 100},
		{targetRank: 9, refinementRealm: 4, want: 10000},
	}

	for _, tt := range tests {
		got := calculateRefinementPathGain(tt.targetRank, tt.refinementRealm)
		if got != tt.want {
			t.Errorf("calculateRefinementPathGain(%d, %d) = %d, want %d", tt.targetRank, tt.refinementRealm, got, tt.want)
		}
	}
}

func TestCollectRefineSupportBonus(t *testing.T) {
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_LITTLE_CARE": {
			1: {
				ID:   "GU_LITTLE_CARE",
				Name: "Tiểu Tâm Cổ R1",
				Rank: 1,
				Effects: models.GuEffects{
					Passive: &models.PassiveEffect{
						SuccessRateBonus: 6.0,
					},
				},
			},
			2: {
				ID:   "GU_LITTLE_CARE",
				Name: "Tiểu Tâm Cổ R2",
				Rank: 2,
				Effects: models.GuEffects{
					Passive: &models.PassiveEffect{
						SuccessRateBonus: 10.0,
					},
				},
			},
		},
		"GU_ICE_MUSCLE": {
			1: {
				ID:   "GU_ICE_MUSCLE",
				Name: "Băng Cơ Cổ R1",
				Rank: 1,
				Effects: models.GuEffects{
					Passive: &models.PassiveEffect{
						SuccessRateBonus: 6.0,
					},
				},
			},
		},
	}

	p := &models.Player{
		GuInventory: models.GuInventory{
			// Two copies of GU_LITTLE_CARE (R1 and R2) equipped
			{InstanceID: "care-1", GuID: "GU_LITTLE_CARE", Rank: 1, Status: "ACTIVE", Equipped: true},
			{InstanceID: "care-2", GuID: "GU_LITTLE_CARE", Rank: 2, Status: "ACTIVE", Equipped: true},
			// One copy of GU_ICE_MUSCLE equipped
			{InstanceID: "ice-1", GuID: "GU_ICE_MUSCLE", Rank: 1, Status: "ACTIVE", Equipped: true},
			// Unequipped or dead copies
			{InstanceID: "ice-2", GuID: "GU_ICE_MUSCLE", Rank: 1, Status: "DEAD", Equipped: true},
			{InstanceID: "ice-3", GuID: "GU_ICE_MUSCLE", Rank: 1, Status: "ACTIVE", Equipped: false},
		},
	}

	// Should only take GU_LITTLE_CARE R2 (+10%) and GU_ICE_MUSCLE R1 (+6%) = 16% total -> 0.16
	got := collectRefineSupportBonus(p)
	want := 0.16
	if got != want {
		t.Errorf("collectRefineSupportBonus() = %v, want %v", got, want)
	}
}
