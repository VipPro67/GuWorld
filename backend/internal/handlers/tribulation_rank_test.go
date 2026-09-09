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
		// Rank 6: Địa Tai (10y - subRank 1), Thiên Kiếp (100y - subRank 4)
		{"Rank 6 Earth Calamity", 6, "EARTH_CALAMITY", 6, 1},
		{"Rank 6 Heavenly Tribulation", 6, "HEAVENLY_TRIBULATION", 6, 4},

		// Rank 7: Địa Tai (10y - subRank 1), Thiên Kiếp (50y - subRank 2), Hạo Kiếp (100y - subRank 4)
		{"Rank 7 Earth Calamity", 7, "EARTH_CALAMITY", 7, 1},
		{"Rank 7 Heavenly Tribulation", 7, "HEAVENLY_TRIBULATION", 7, 2},
		{"Rank 7 Grand Tribulation", 7, "GRAND_TRIBULATION", 7, 4},

		// Rank 8: Thiên Kiếp (10y - subRank 1), Hạo Kiếp (50y - subRank 2), Vạn Kiếp (100y - subRank 4)
		{"Rank 8 Heavenly Tribulation", 8, "HEAVENLY_TRIBULATION", 8, 1},
		{"Rank 8 Grand Tribulation", 8, "GRAND_TRIBULATION", 8, 2},
		{"Rank 8 Myriad Tribulation", 8, "MYRIAD_TRIBULATION", 8, 4},

		// Rank 9: Hạo Kiếp (10y - subRank 1), Vạn Kiếp (50y - subRank 2), Hỗn Độn (100y - subRank 4)
		{"Rank 9 Grand Tribulation", 9, "GRAND_TRIBULATION", 9, 1},
		{"Rank 9 Myriad Tribulation", 9, "MYRIAD_TRIBULATION", 9, 2},
		{"Rank 9 Chaos Calamity", 9, "CHAOS_CALAMITY", 9, 4},
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
