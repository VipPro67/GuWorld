package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGetSpoilDropsForTribulation(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		rank     int
		tribType string
		wantItem string
	}{
		{
			name:     "Fire path drops fire crystal",
			path:     "fire",
			rank:     6,
			tribType: "EARTH_CALAMITY",
			wantItem: "fire_crystal_rank6",
		},
		{
			name:     "Ice path drops ice crystal",
			path:     "ice",
			rank:     6,
			tribType: "EARTH_CALAMITY",
			wantItem: "ice_crystal_rank6",
		},
		{
			name:     "Lightning path drops lightning crystal",
			path:     "lightning",
			rank:     7,
			tribType: "HEAVENLY_TRIBULATION",
			wantItem: "lightning_crystal_rank7",
		},
		{
			name:     "Metal path drops metal ore",
			path:     "metal",
			rank:     6,
			tribType: "EARTH_CALAMITY",
			wantItem: "metal_ore_rank6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			drops := getSpoilDropsForTribulation(tt.path, tt.rank, tt.tribType)
			if len(drops) == 0 {
				t.Fatalf("expected spoils drops, got none")
			}
			found := false
			for _, d := range drops {
				if d.ItemID == tt.wantItem {
					found = true
					if d.Quantity <= 0 {
						t.Errorf("expected positive quantity, got %d", d.Quantity)
					}
					break
				}
			}
			if !found {
				t.Errorf("expected to find item %s in drops, got %+v", tt.wantItem, drops)
			}
		})
	}
}

func TestGetTribulationTypeName(t *testing.T) {
	if got := models.GetTribulationTypeName("EARTH_CALAMITY"); got != "Địa Tai" {
		t.Errorf("expected 'Địa Tai', got '%s'", got)
	}
	if got := models.GetTribulationTypeName("HEAVENLY_TRIBULATION"); got != "Thiên Kiếp" {
		t.Errorf("expected 'Thiên Kiếp', got '%s'", got)
	}
	if got := models.GetTribulationTypeName("GRAND_TRIBULATION"); got != "Hạo Kiếp" {
		t.Errorf("expected 'Hạo Kiếp', got '%s'", got)
	}
	if got := models.GetTribulationTypeName("MYRIAD_TRIBULATION"); got != "Vạn Kiếp" {
		t.Errorf("expected 'Vạn Kiếp', got '%s'", got)
	}
	if got := models.GetTribulationTypeName("CHAOS_CALAMITY"); got != "Hỗn Độn Kiếp" {
		t.Errorf("expected 'Hỗn Độn Kiếp', got '%s'", got)
	}
}

func TestGetBeastCaptureDrop(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		rank     int
		wantItem string
	}{
		{"Fire beast drops fire rat fur", "fire", 6, "fire_rat_fur_rank6"},
		{"Ice beast drops ice cicada wing", "ice", 6, "ice_cicada_wing_rank6"},
		{"Water beast drops ice cicada wing", "water", 6, "ice_cicada_wing_rank6"},
		{"Poison beast drops poison sac", "poison", 6, "poison_sac_rank6"},
		{"Lightning beast drops lightning crystal", "lightning", 7, "lightning_crystal_rank7"},
		{"Blood beast drops blood bat wing", "blood", 7, "blood_bat_wing_rank7"},
		{"Default beast drops wild boar tusk", "earth", 6, "wild_boar_tusk_rank6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBeastCaptureDrop(tt.path, tt.rank)
			if got.ItemID != tt.wantItem {
				t.Errorf("getBeastCaptureDrop(%s, %d) item = %s, want %s", tt.path, tt.rank, got.ItemID, tt.wantItem)
			}
			if got.Quantity <= 0 {
				t.Errorf("getBeastCaptureDrop(%s, %d) quantity = %d, want > 0", tt.path, tt.rank, got.Quantity)
			}
		})
	}
}
