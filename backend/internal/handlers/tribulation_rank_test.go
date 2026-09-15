package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGetTribulationRankAndSubRank(t *testing.T) {
	tests := []struct {
		name        string
		playerRank  int
		tribType    string
		wantRank    int
		wantSubRank int
	}{
		// Monster rank is determined by the tribulation type.
		{"Rank 6 Earth Calamity", 6, "EARTH_CALAMITY", 6, 1},
		{"Rank 6 Heavenly Tribulation", 6, "HEAVENLY_TRIBULATION", 7, 2},

		{"Rank 7 Earth Calamity", 7, "EARTH_CALAMITY", 6, 1},
		{"Rank 7 Heavenly Tribulation", 7, "HEAVENLY_TRIBULATION", 7, 2},
		{"Rank 7 Grand Tribulation", 7, "GRAND_TRIBULATION", 8, 3},

		{"Rank 8 Heavenly Tribulation", 8, "HEAVENLY_TRIBULATION", 7, 2},
		{"Rank 8 Grand Tribulation", 8, "GRAND_TRIBULATION", 8, 3},
		{"Rank 8 Myriad Tribulation", 8, "MYRIAD_TRIBULATION", 9, 4},

		{"Rank 9 Grand Tribulation", 9, "GRAND_TRIBULATION", 8, 3},
		{"Rank 9 Myriad Tribulation", 9, "MYRIAD_TRIBULATION", 9, 4},
		{"Rank 9 Chaos Calamity", 9, "CHAOS_CALAMITY", 10, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRank, gotSubRank := models.GetTribulationRankAndSubRank(tt.playerRank, tt.tribType)
			if gotRank != tt.wantRank || gotSubRank != tt.wantSubRank {
				t.Errorf("GetTribulationRankAndSubRank(%d, %s) = (%d, %d); want (%d, %d)",
					tt.playerRank, tt.tribType, gotRank, gotSubRank, tt.wantRank, tt.wantSubRank)
			}
		})
	}
}

func TestTribulationIntervalsMatchWiki(t *testing.T) {
	// Lục Chuyển: Địa Tai (10 năm), Thiên Kiếp (100 năm)
	types6 := models.GetTribulationTypesForRank(6)
	if len(types6) != 2 || types6[0].Type != "EARTH_CALAMITY" || types6[0].Interval != 10 ||
		types6[1].Type != "HEAVENLY_TRIBULATION" || types6[1].Interval != 100 {
		t.Errorf("Rank 6 tribulation types mismatch: %+v", types6)
	}

	// Thất Chuyển: Địa Tai (10 năm), Thiên Kiếp (50 năm), Hạo Kiếp (100 năm)
	types7 := models.GetTribulationTypesForRank(7)
	if len(types7) != 3 ||
		types7[0].Type != "EARTH_CALAMITY" || types7[0].Interval != 10 ||
		types7[1].Type != "HEAVENLY_TRIBULATION" || types7[1].Interval != 50 ||
		types7[2].Type != "GRAND_TRIBULATION" || types7[2].Interval != 100 {
		t.Errorf("Rank 7 tribulation types mismatch: %+v", types7)
	}

	// Bát Chuyển: Thiên Kiếp (10 năm), Hạo Kiếp (50 năm), Vạn Kiếp (100 năm)
	types8 := models.GetTribulationTypesForRank(8)
	if len(types8) != 3 ||
		types8[0].Type != "HEAVENLY_TRIBULATION" || types8[0].Interval != 10 ||
		types8[1].Type != "GRAND_TRIBULATION" || types8[1].Interval != 50 ||
		types8[2].Type != "MYRIAD_TRIBULATION" || types8[2].Interval != 100 {
		t.Errorf("Rank 8 tribulation types mismatch: %+v", types8)
	}

	// Cửu Chuyển: Hạo Kiếp (10 năm), Vạn Kiếp (50 năm), Hỗn Độn Đại Nạn (100 năm)
	types9 := models.GetTribulationTypesForRank(9)
	if len(types9) != 3 ||
		types9[0].Type != "GRAND_TRIBULATION" || types9[0].Interval != 10 ||
		types9[1].Type != "MYRIAD_TRIBULATION" || types9[1].Interval != 50 ||
		types9[2].Type != "CHAOS_CALAMITY" || types9[2].Interval != 100 {
		t.Errorf("Rank 9 tribulation types mismatch: %+v", types9)
	}
}

func TestNextTribulationInfoWithCountsKeepsOverlappingMilestones(t *testing.T) {
	year, tribType := models.GetNextTribulationInfoWithCounts(6, 0, map[string]int{
		"EARTH_CALAMITY":       9,
		"HEAVENLY_TRIBULATION": 0,
	})
	if year != 100 || tribType != "HEAVENLY_TRIBULATION" {
		t.Fatalf("first rank 6 year-100 event = (%v, %s); want (100, HEAVENLY_TRIBULATION)", year, tribType)
	}

	year, tribType = models.GetNextTribulationInfoWithCounts(6, 0, map[string]int{
		"EARTH_CALAMITY":       9,
		"HEAVENLY_TRIBULATION": 1,
	})
	if year != 100 || tribType != "EARTH_CALAMITY" {
		t.Fatalf("second rank 6 year-100 event = (%v, %s); want (100, EARTH_CALAMITY)", year, tribType)
	}
}

func TestRank10UsesChaosCalamity(t *testing.T) {
	types := models.GetTribulationTypesForRank(10)
	if len(types) != 1 || types[0].Type != "CHAOS_CALAMITY" {
		t.Fatalf("rank 10 tribulation types = %+v; want only CHAOS_CALAMITY", types)
	}

	rank, subRank := models.GetTribulationRankAndSubRank(10, "CHAOS_CALAMITY")
	if rank != 10 || subRank != 4 {
		t.Fatalf("rank 10 chaos level = (%d, %d); want (10, 4)", rank, subRank)
	}
}

func TestCalculateTribulationReward(t *testing.T) {
	// Ensure higher rank and higher tier tribulations yield progressively greater rewards
	r6Earth := models.CalculateTribulationReward(6, "EARTH_CALAMITY", 1)
	r6Heaven := models.CalculateTribulationReward(6, "HEAVENLY_TRIBULATION", 1)
	if r6Earth >= r6Heaven {
		t.Errorf("Expected r6Heaven > r6Earth, got %d vs %d", r6Heaven, r6Earth)
	}

	r7Earth := models.CalculateTribulationReward(7, "EARTH_CALAMITY", 1)
	r7Heaven := models.CalculateTribulationReward(7, "HEAVENLY_TRIBULATION", 1)
	r7Grand := models.CalculateTribulationReward(7, "GRAND_TRIBULATION", 1)
	if !(r7Earth < r7Heaven && r7Heaven < r7Grand) {
		t.Errorf("Expected r7Earth (%d) < r7Heaven (%d) < r7Grand (%d)", r7Earth, r7Heaven, r7Grand)
	}
	if r7Earth <= r6Earth {
		t.Errorf("Expected r7Earth > r6Earth, got %d vs %d", r7Earth, r6Earth)
	}

	r8Heaven := models.CalculateTribulationReward(8, "HEAVENLY_TRIBULATION", 1)
	r8Grand := models.CalculateTribulationReward(8, "GRAND_TRIBULATION", 1)
	r8Myriad := models.CalculateTribulationReward(8, "MYRIAD_TRIBULATION", 1)
	if !(r8Heaven < r8Grand && r8Grand < r8Myriad) {
		t.Errorf("Expected r8Heaven (%d) < r8Grand (%d) < r8Myriad (%d)", r8Heaven, r8Grand, r8Myriad)
	}
	if r8Heaven <= r7Heaven {
		t.Errorf("Expected r8Heaven > r7Heaven, got %d vs %d", r8Heaven, r7Heaven)
	}

	r9Grand := models.CalculateTribulationReward(9, "GRAND_TRIBULATION", 1)
	r9Myriad := models.CalculateTribulationReward(9, "MYRIAD_TRIBULATION", 1)
	r9Chaos := models.CalculateTribulationReward(9, "CHAOS_CALAMITY", 1)
	if !(r9Grand < r9Myriad && r9Myriad < r9Chaos) {
		t.Errorf("Expected r9Grand (%d) < r9Myriad (%d) < r9Chaos (%d)", r9Grand, r9Myriad, r9Chaos)
	}
	if r9Grand <= r8Grand {
		t.Errorf("Expected r9Grand > r8Grand, got %d vs %d", r9Grand, r8Grand)
	}
}

func TestSanitizeTribulationTypeForRank(t *testing.T) {
	tests := []struct {
		rank     int
		input    string
		expected string
	}{
		{6, "EARTH_CALAMITY", "EARTH_CALAMITY"},
		{6, "HEAVENLY_TRIBULATION", "HEAVENLY_TRIBULATION"},
		{6, "MYRIAD_TRIBULATION", "EARTH_CALAMITY"}, // Invalid for Rank 6 -> defaults to EARTH_CALAMITY

		{8, "EARTH_CALAMITY", "HEAVENLY_TRIBULATION"}, // Earth Calamity invalid for Rank 8 -> defaults to HEAVENLY_TRIBULATION
		{8, "", "HEAVENLY_TRIBULATION"},

		{9, "EARTH_CALAMITY", "GRAND_TRIBULATION"},       // Earth Calamity invalid for Rank 9 -> defaults to GRAND_TRIBULATION
		{9, "HEAVENLY_TRIBULATION", "GRAND_TRIBULATION"}, // Heavenly Tribulation invalid for Rank 9 -> defaults to GRAND_TRIBULATION
		{9, "MYRIAD_TRIBULATION", "MYRIAD_TRIBULATION"},
		{9, "", "GRAND_TRIBULATION"},

		{10, "EARTH_CALAMITY", "CHAOS_CALAMITY"},
		{10, "GRAND_TRIBULATION", "CHAOS_CALAMITY"},
	}

	for _, tt := range tests {
		got := models.SanitizeTribulationTypeForRank(tt.rank, tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeTribulationTypeForRank(%d, %q) = %q; want %q", tt.rank, tt.input, got, tt.expected)
		}
	}
}
