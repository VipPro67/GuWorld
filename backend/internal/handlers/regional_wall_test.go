package handlers

import (
	"math"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGetRegionalWallDebuffScaling(t *testing.T) {
	epsilon := 1e-6
	expectedDur := 36 * time.Hour

	// Mortals (Rank 1-5)
	r1Pct, r1Dur := GetRegionalWallDebuff(1, 1)
	if math.Abs(r1Pct - (-0.10)) > epsilon || r1Dur != expectedDur {
		t.Errorf("Rank 1.1 debuff expected -0.10, 36h, got %f, %v", r1Pct, r1Dur)
	}

	r3Pct, r3Dur := GetRegionalWallDebuff(3, 2)
	if math.Abs(r3Pct - (-0.21)) > epsilon || r3Dur != expectedDur {
		t.Errorf("Rank 3.2 debuff expected -0.21, 36h, got %f, %v", r3Pct, r3Dur)
	}

	r5Pct, r5Dur := GetRegionalWallDebuff(5, 5)
	if math.Abs(r5Pct - (-0.34)) > epsilon || r5Dur != expectedDur {
		t.Errorf("Rank 5.5 debuff expected -0.34, 36h, got %f, %v", r5Pct, r5Dur)
	}

	// Immortals (Rank 6-9) — higher rank = significantly higher debuff
	r6Pct, r6Dur := GetRegionalWallDebuff(6, 1)
	if math.Abs(r6Pct - (-0.45)) > epsilon || r6Dur != expectedDur {
		t.Errorf("Rank 6.1 debuff expected -0.45, 36h, got %f, %v", r6Pct, r6Dur)
	}

	r8Pct, r8Dur := GetRegionalWallDebuff(8, 3)
	if math.Abs(r8Pct - (-0.69)) > epsilon || r8Dur != expectedDur {
		t.Errorf("Rank 8.3 debuff expected -0.69, 36h, got %f, %v", r8Pct, r8Dur)
	}

	r9Pct, r9Dur := GetRegionalWallDebuff(9, 5)
	if math.Abs(r9Pct - (-0.83)) > epsilon || r9Dur != expectedDur {
		t.Errorf("Rank 9.5 debuff expected -0.83, 36h, got %f, %v", r9Pct, r9Dur)
	}
}

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
	if math.Abs(wallStatus.Value - (-0.20)) > 1e-6 {
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

