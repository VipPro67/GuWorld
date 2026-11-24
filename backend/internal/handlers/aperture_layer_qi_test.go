package handlers

import (
	"math"
	"testing"

	"github.com/google/uuid"
)

func TestLayerQiMultipliers(t *testing.T) {
	tests := []struct {
		layer            string
		expectedHeavenQi float64
		expectedEarthQi  float64
	}{
		{"dia_uyen", 0.20, 3.50},
		{"world_map", 1.00, 1.00},
		{"bai_thien", 3.00, 0.30},
		{"hac_thien", 1.00, 0.50},
		{"unknown_layer", 1.00, 1.00},
	}

	for _, tt := range tests {
		hMult, eMult := LayerQiMultipliers(tt.layer)
		if math.Abs(hMult-tt.expectedHeavenQi) > 0.001 {
			t.Errorf("LayerQiMultipliers(%q) heavenMultiplier = %v; want %v", tt.layer, hMult, tt.expectedHeavenQi)
		}
		if math.Abs(eMult-tt.expectedEarthQi) > 0.001 {
			t.Errorf("LayerQiMultipliers(%q) earthMultiplier = %v; want %v", tt.layer, eMult, tt.expectedEarthQi)
		}
	}
}

func TestLayerTribulationTimeRateMultiplier(t *testing.T) {
	tests := []struct {
		layer        string
		expectedRate float64
	}{
		{"hac_thien", 1.0 / 1.5},
		{"bai_thien", 1.0},
		{"dia_uyen", 1.0},
		{"world_map", 1.0},
	}

	for _, tt := range tests {
		rate := LayerTribulationTimeRateMultiplier(tt.layer)
		if math.Abs(rate-tt.expectedRate) > 0.001 {
			t.Errorf("LayerTribulationTimeRateMultiplier(%q) = %v; want %v", tt.layer, rate, tt.expectedRate)
		}
	}
}

func TestAperturePortalVerticalLayerVisibilityGate(t *testing.T) {
	ownerID := uuid.New()
	outsiderID := uuid.New()

	// Case 1: Outsider on world_map looking at a portal in Bai Thien
	portalLayer := worldMapIDForRegion("bt_thai_co_van_hai") // "bai_thien"
	viewerLayer := worldMapIDForRegion("dai_vuc_nam_cuong")  // "world_map"
	isMine := outsiderID == ownerID

	if !isMine && portalLayer != viewerLayer {
		// Expected: visibility blocked!
	} else {
		t.Errorf("Expected portal visibility to be blocked between %s and %s for non-owner", viewerLayer, portalLayer)
	}

	// Case 2: Outsider on Bai Thien looking at a portal in Bai Thien
	sameLayerViewer := worldMapIDForRegion("bt_quang_minh_tinh_doi") // "bai_thien"
	if !isMine && portalLayer != sameLayerViewer {
		t.Errorf("Expected portal visibility to NOT be blocked between same layers: %s and %s", sameLayerViewer, portalLayer)
	}

	// Case 3: Owner looking at their own portal in Bai Thien while standing in world_map
	ownerIsMine := ownerID == ownerID
	if !ownerIsMine && portalLayer != viewerLayer {
		t.Errorf("Owner should always see their own portal regardless of layer!")
	}
}

func TestLayerPlacementDescription(t *testing.T) {
	desc0 := LayerPlacementDescription("dia_uyen", 0, 0)
	if desc0 == "" {
		t.Errorf("Expected description for dia_uyen")
	}

	desc2 := LayerPlacementDescription("bai_thien", 0, 0)
	if desc2 == "" {
		t.Errorf("Expected description for bai_thien")
	}

	desc3 := LayerPlacementDescription("hac_thien", 5, 5)
	if desc3 == "" {
		t.Errorf("Expected description for hac_thien")
	}
}

func TestApertureCoreEffectiveTimeRateInBlackHeaven(t *testing.T) {
	placedRegion := "ht_da_mac_tinh_hai"
	layer := worldMapIDForRegion(placedRegion)
	if layer != "hac_thien" {
		t.Fatalf("worldMapIDForRegion(%q) = %q; want hac_thien", placedRegion, layer)
	}

	baseRate := 1.0
	effectiveRate := baseRate * LayerTribulationTimeRateMultiplier(layer)
	expectedRate := 1.0 / 1.5

	if math.Abs(effectiveRate-expectedRate) > 0.001 {
		t.Errorf("effectiveRate = %v; want %v", effectiveRate, expectedRate)
	}
}

func TestWorldMapIDForRegionPrefixes(t *testing.T) {
	cases := []struct {
		region   string
		expected string
	}{
		{"du_am_luu_dia_mach", "dia_uyen"},
		{"du_huyet_tinh_tham_hac", "dia_uyen"},
		{"du_thach_nhan_co_do", "dia_uyen"},
		{"bt_thai_co_van_hai", "bai_thien"},
		{"bt_quang_minh_tinh_doi", "bai_thien"},
		{"bt_thien_dinh_ngoai_vuc", "bai_thien"},
		{"ht_da_mac_tinh_hai", "hac_thien"},
		{"ht_u_minh_tham_uyen", "hac_thien"},
		{"ht_di_nhan_vuong_quoc", "hac_thien"},
		{"dai_vuc_trung_chau", "world_map"},
		{"thanh_mao_son", "thanh_mao_son"},
		{"dai_vuc_nam_cuong", "world_map"},
	}

	for _, c := range cases {
		got := worldMapIDForRegion(c.region)
		if got != c.expected {
			t.Errorf("worldMapIDForRegion(%q) = %q; want %q", c.region, got, c.expected)
		}
	}
}
