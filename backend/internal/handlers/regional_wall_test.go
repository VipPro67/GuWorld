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
	expectedSource := "Giới Vực Bích (Đại Vực Nam Cương ➔ Đại Vực Trung Châu)"
	if wallStatus.Source != expectedSource {
		t.Errorf("expected debuff source %q, got %q", expectedSource, wallStatus.Source)
	}

	// Calculate stats with debuff applied
	stats := CalculatePlayerStats(p)
	// Base attack for Rank 3 SubRank 1 is 10 + (3-1)*10 = 30
	// Reduced by 20%: 30 * 0.8 = 24
	if math.Abs(stats["attack"]-24.0) > 1e-6 {
		t.Errorf("expected debuffed attack 24.0, got %f", stats["attack"])
	}
}

func TestCheckAndApplyRegionalWallDebuffBetween_KhuVucVsNguVuc(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "dai_vuc_nam_cuong", Name: "Đại Vực Nam Cương", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 10000, MaxY: 10000},
		{ID: "dai_vuc_trung_chau", Name: "Đại Vực Trung Châu", ZoneType: ZoneTypeLarge, MinX: 10000, MinY: 0, MaxX: 20000, MaxY: 10000},
		{ID: "thanh_mao_son", Name: "Thanh Mao Sơn", ZoneType: ZoneTypeNamed, ParentID: "dai_vuc_nam_cuong", MinX: 2000, MinY: 4000, MaxX: 4000, MaxY: 6000},
		{ID: "bach_cot_son", Name: "Bạch Cốt Sơn", ZoneType: ZoneTypeNamed, ParentID: "dai_vuc_nam_cuong", MinX: 6000, MinY: 2000, MaxX: 8000, MaxY: 4000},
		{ID: "phi_hac_son", Name: "Phi Hạc Sơn", ZoneType: ZoneTypeNamed, ParentID: "dai_vuc_trung_chau", MinX: 10000, MinY: 0, MaxX: 12000, MaxY: 2000},
		{ID: "hub_unparented", Name: "Tân Thủ Cốc", ZoneType: ZoneTypeHub, MinX: 4900, MinY: 4900, MaxX: 5100, MaxY: 5100, EntryX: 5000, EntryY: 5000},
		{ID: "secret_realm_test", Name: "Bí Cảnh Test", ZoneType: ZoneTypeSecret, MinX: 1000, MinY: 1000, MaxX: 1050, MaxY: 1050},
	}
	zoneReg.byID = make(map[string]*models.Region)
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}
	zoneReg.mu.Unlock()

	p := &models.Player{}
	p.Rank = 2
	p.SubRank = 1
	p.Statuses = models.StatusList{}

	// 1. Moving between two different KHU_VUC in the SAME NGU_VUC (Thanh Mao Sơn -> Bạch Cốt Sơn)
	msgSame := CheckAndApplyRegionalWallDebuffBetween(p, "thanh_mao_son", "bach_cot_son")
	if msgSame != "" {
		t.Errorf("expected no suppression when switching KHU_VUC in same NGU_VUC, got %q", msgSame)
	}
	if p.Statuses.Has("regional_wall_suppression") {
		t.Fatalf("regional_wall_suppression should NOT be applied when switching KHU_VUC in same NGU_VUC")
	}

	// 2. Unparented hub inside Nam Cương (coordinates 5000, 5000) -> Thanh Mao Sơn (Nam Cương)
	msgHub := CheckAndApplyRegionalWallDebuffBetween(p, "hub_unparented", "thanh_mao_son")
	if msgHub != "" {
		t.Errorf("expected no suppression from unparented hub in same NGU_VUC, got %q", msgHub)
	}
	if p.Statuses.Has("regional_wall_suppression") {
		t.Fatalf("regional_wall_suppression should NOT be applied between unparented hub and KHU_VUC in same NGU_VUC")
	}

	// 3. Secret realm to world KHU_VUC -> should not apply regional wall debuff
	msgSecret := CheckAndApplyRegionalWallDebuffBetween(p, "secret_realm_test", "thanh_mao_son")
	if msgSecret != "" {
		t.Errorf("expected no suppression when transitioning from secret realm, got %q", msgSecret)
	}
	if p.Statuses.Has("regional_wall_suppression") {
		t.Fatalf("regional_wall_suppression should NOT be applied for secret realm")
	}

	// 4. Moving between KHU_VUC across DIFFERENT NGU_VUC (Thanh Mao Sơn [Nam Cương] -> Phi Hạc Sơn [Trung Châu])
	msgCross := CheckAndApplyRegionalWallDebuffBetween(p, "thanh_mao_son", "phi_hac_son")
	if msgCross == "" {
		t.Errorf("expected suppression message when switching NGU_VUC")
	}
	if !p.Statuses.Has("regional_wall_suppression") {
		t.Fatalf("regional_wall_suppression MUST be applied when crossing NGU_VUC")
	}
	expectedSource := "Giới Vực Bích (Đại Vực Nam Cương ➔ Đại Vực Trung Châu)"
	for _, st := range p.Statuses.Active() {
		if st.ID == "regional_wall_suppression" {
			if st.Source != expectedSource {
				t.Errorf("expected debuff source %q, got %q", expectedSource, st.Source)
			}
			break
		}
	}
}

func TestCheckAndApplyRegionalWallDebuff_DisabledBySetting(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "dai_vuc_nam_cuong", Name: "Đại Vực Nam Cương", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999},
		{ID: "dai_vuc_trung_chau", Name: "Đại Vực Trung Châu", ZoneType: ZoneTypeLarge, MinX: 10000, MinY: 0, MaxX: 19999, MaxY: 9999},
	}
	zoneReg.byID = make(map[string]*models.Region)
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}
	zoneReg.mu.Unlock()

	t.Setenv("REGIONAL_WALL_SUPPRESSION_ENABLED", "false")

	p := &models.Player{}
	p.Rank = 3
	p.SubRank = 1
	p.Statuses = models.StatusList{}

	msg := CheckAndApplyRegionalWallDebuff(p, 5000, 5000, 15000, 5000)
	if msg != "" {
		t.Errorf("expected no debuff message when setting disabled, got %q", msg)
	}
	if p.Statuses.Has("regional_wall_suppression") {
		t.Fatal("regional_wall_suppression must not be applied when setting disabled")
	}
}

func TestFindLargeZoneAt_BorderResolution(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "dai_vuc_nam_cuong", Name: "Đại Vực Nam Cương", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 10000, MaxY: 10000},
		{ID: "dai_vuc_trung_chau", Name: "Đại Vực Trung Châu", ZoneType: ZoneTypeLarge, MinX: 10000, MinY: 0, MaxX: 20000, MaxY: 10000},
		{ID: "phi_hac_son", Name: "Phi Hạc Sơn", ZoneType: ZoneTypeNamed, ParentID: "dai_vuc_trung_chau", MinX: 10000, MinY: 0, MaxX: 12000, MaxY: 2000},
	}
	zoneReg.byID = make(map[string]*models.Region)
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}
	zoneReg.mu.Unlock()

	// Coordinate (10000, 1000) falls on the border of dai_vuc_nam_cuong and dai_vuc_trung_chau,
	// but is specifically inside phi_hac_son (Trung Châu). It should resolve to dai_vuc_trung_chau.
	lz := FindLargeZoneAt(10000, 1000)
	if lz == nil || lz.ID != "dai_vuc_trung_chau" {
		t.Fatalf("expected border point inside phi_hac_son to resolve to dai_vuc_trung_chau, got %v", lz)
	}
}
