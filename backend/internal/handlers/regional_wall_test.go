package handlers

import (
	"math"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestCheckAndApplyRegionalWallDebuff(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "dai_vuc_nam_cuong", Name: "Đại Vực Nam Cương", ZoneType: "NGU_VUC", MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999},
		{ID: "dai_vuc_trung_chau", Name: "Đại Vực Trung Châu", ZoneType: "NGU_VUC", MinX: 10000, MinY: 0, MaxX: 19999, MaxY: 9999},
	}
	zoneReg.byID = make(map[string]*models.Region)
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}
	zoneReg.mu.Unlock()

	p := &models.Player{}
	p.Rank = 3
	p.SubRank = 1
	p.Statuses = models.StatusList{}

	// Same NGU_VUC (Nam Cương: x in 0..9999, y in 0..9999)
	msgSame := CheckAndApplyRegionalWallDebuff(p, 1000, 1000, 2000, 2000)
	if msgSame != "" {
		t.Errorf("expected empty message for same region move, got %q", msgSame)
	}
	if p.Statuses.Has("regional_wall_suppression") {
		t.Errorf("status should not be applied for same region move")
	}

	// Move from Nam Cương (5000, 5000) to Trung Châu (15000, 5000)
	msgCross := CheckAndApplyRegionalWallDebuff(p, 5000, 5000, 15000, 5000)
	if msgCross == "" {
		t.Errorf("expected debuff message when crossing NGU_VUC")
	}
	if !p.Statuses.Has("regional_wall_suppression") {
		t.Fatalf("regional_wall_suppression status should be active on player")
	}

	// Verify status instance value
	active := p.Statuses.Active()
	var wallStatus *models.StatusInstance
	for i := range active {
		if active[i].ID == "regional_wall_suppression" {
			wallStatus = &active[i]
			break
		}
	}
	if wallStatus == nil {
		t.Fatalf("regional_wall_suppression status instance not found")
	}
	if math.Abs(wallStatus.Value-(-0.20)) > 1e-6 {
		t.Errorf("expected debuff value -0.20 for Rank 3.1, got %f", wallStatus.Value)
	}

	// Calculate stats with debuff applied
	stats := CalculatePlayerStats(p)
	// Base attack for Rank 3 SubRank 1 is 10 + (3-1)*10 = 30
	// Reduced by 20%: 30 * 0.8 = 24
	if math.Abs(stats["attack"]-24.0) > 1e-6 {
		t.Errorf("expected debuffed attack 24.0, got %f", stats["attack"])
	}
}
