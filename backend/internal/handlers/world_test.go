package handlers

import (
	"math/rand"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestPickWeightedType(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	weights := map[string]int{"Monster": 50, "Empty": 50}
	counts := map[string]int{}
	for i := 0; i < 10000; i++ {
		counts[pickWeightedType(weights, rng)]++
	}
	if counts["Monster"] == 0 || counts["Empty"] == 0 {
		t.Fatalf("expected both types rolled, got %v", counts)
	}
	if pickWeightedType(map[string]int{}, rng) != "Empty" {
		t.Fatalf("empty weights should default to Empty")
	}
	if pickWeightedType(nil, rng) != "Empty" {
		t.Fatalf("nil weights should default to Empty")
	}
}

func TestGetCellSeedDeterministic(t *testing.T) {
	s1 := GetCellSeed(10, 20, "dai_vuc_lam_son")
	s2 := GetCellSeed(10, 20, "dai_vuc_lam_son")
	if s1 != s2 {
		t.Fatalf("seed not deterministic: %d != %d", s1, s2)
	}
	s3 := GetCellSeed(11, 20, "dai_vuc_lam_son")
	if s1 == s3 {
		t.Fatalf("different coords produced same seed")
	}
	s4 := GetCellSeed(10, 20, "dai_vuc_am_anh")
	if s1 == s4 {
		t.Fatalf("different zones produced same seed")
	}
}

func TestZoneContainment(t *testing.T) {
	z := &models.Region{ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999}
	if !zoneContains(z, 0, 0) {
		t.Fatalf("boundary min should be contained")
	}
	if !zoneContains(z, 9999, 9999) {
		t.Fatalf("boundary max should be contained")
	}
	if zoneContains(z, 10000, 10000) {
		t.Fatalf("outside should not be contained")
	}
}

func TestInWorld(t *testing.T) {
	if !inWorld(0, 0) {
		t.Fatalf("origin should be in world")
	}
	if inWorld(WorldWidth, WorldHeight) {
		t.Fatalf("out-of-range should be rejected")
	}
	if inWorld(-1, 0) || inWorld(0, -1) {
		t.Fatalf("negative coords should be rejected")
	}
}

func TestFindZoneAtPrecedence(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "dai_vuc_test", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999},
		{ID: "ngu_vuc_nested", ZoneType: ZoneTypeNamed, ParentID: "dai_vuc_test", MinX: 1000, MinY: 1000, MaxX: 2000, MaxY: 2000},
		{ID: "hub_test", ZoneType: ZoneTypeHub, MinX: 15000, MinY: 9000, MaxX: 16000, MaxY: 11000},
		{ID: "safe_test", ZoneType: ZoneTypeSafe, MinX: 3000, MinY: 3000, MaxX: 4000, MaxY: 4000},
		{ID: "boss_test", ZoneType: ZoneTypeBoss, MinX: 5000, MinY: 5000, MaxX: 6000, MaxY: 6000},
		{ID: "secret_test", ZoneType: ZoneTypeSecret, MinX: 7000, MinY: 7000, MaxX: 8000, MaxY: 8000},
		{ID: "dungeon_test", ZoneType: ZoneTypeDungeon, MinX: 3000, MinY: 7000, MaxX: 4000, MaxY: 8000},
		{ID: "pvp_test", ZoneType: ZoneTypePvp, MinX: 7000, MinY: 3000, MaxX: 8000, MaxY: 4000},
	}
	zoneReg.mu.Unlock()

	z := FindZoneAt(1500, 1500)
	if z == nil || z.ID != "ngu_vuc_nested" {
		t.Fatalf("expected innermost named zone, got %+v", z)
	}
	z = FindZoneAt(2500, 2500)
	if z == nil || z.ID != "dai_vuc_test" {
		t.Fatalf("expected large zone fallback, got %+v", z)
	}
	z = FindZoneAt(15500, 10000)
	if z == nil || z.ID != "hub_test" {
		t.Fatalf("expected hub precedence, got %+v", z)
	}
	// New zone types beat the underlying NGU_VUC fallback.
	z = FindZoneAt(3500, 3500)
	if z == nil || z.ID != "safe_test" {
		t.Fatalf("expected SAFE_ZONE precedence, got %+v", z)
	}
	z = FindZoneAt(5500, 5500)
	if z == nil || z.ID != "boss_test" {
		t.Fatalf("expected BOSS_ZONE precedence, got %+v", z)
	}
	z = FindZoneAt(7500, 7500)
	if z == nil || z.ID != "secret_test" {
		t.Fatalf("expected SECRET_REALM precedence, got %+v", z)
	}
	z = FindZoneAt(3500, 7500)
	if z == nil || z.ID != "dungeon_test" {
		t.Fatalf("expected DUNGEON precedence, got %+v", z)
	}
	z = FindZoneAt(7500, 3500)
	if z == nil || z.ID != "pvp_test" {
		t.Fatalf("expected PVP_ZONE precedence, got %+v", z)
	}
	if DeriveRegionAt(999999, 999999) != HubRegionID {
		t.Fatalf("out-of-zone coords should fall back to hub")
	}
}

func TestFindZoneAtPriorityOrder(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "dai_vuc_test", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999},
		{ID: "pvp_test", ZoneType: ZoneTypePvp, MinX: 1000, MinY: 1000, MaxX: 3000, MaxY: 3000},
		{ID: "dungeon_test", ZoneType: ZoneTypeDungeon, MinX: 1500, MinY: 1500, MaxX: 2500, MaxY: 2500},
		{ID: "secret_test", ZoneType: ZoneTypeSecret, MinX: 1700, MinY: 1700, MaxX: 2300, MaxY: 2300},
	}
	zoneReg.mu.Unlock()

	// SECRET_REALM > DUNGEON > PVP_ZONE > NGU_VUC at the innermost point.
	z := FindZoneAt(2000, 2000)
	if z == nil || z.ID != "secret_test" {
		t.Fatalf("expected SECRET_REALM to win over DUNGEON/PVP/DAI, got %+v", z)
	}
	// DUNGEON > PVP_ZONE > NGU_VUC where SECRET_REALM does not cover.
	z = FindZoneAt(1600, 1600)
	if z == nil || z.ID != "dungeon_test" {
		t.Fatalf("expected DUNGEON to win over PVP/DAI, got %+v", z)
	}
	// PVP_ZONE > NGU_VUC outside DUNGEON/SECRET.
	z = FindZoneAt(1100, 1100)
	if z == nil || z.ID != "pvp_test" {
		t.Fatalf("expected PVP_ZONE to win over NGU_VUC, got %+v", z)
	}
}

func TestFindZoneAtSamePriorityAreaTieBreak(t *testing.T) {
	zoneReg.mu.Lock()
	zoneReg.zones = []models.Region{
		{ID: "large_ngu", ZoneType: ZoneTypeNamed, MinX: 0, MinY: 0, MaxX: 5000, MaxY: 5000},
		{ID: "small_ngu", ZoneType: ZoneTypeNamed, MinX: 1000, MinY: 1000, MaxX: 2000, MaxY: 2000},
	}
	zoneReg.mu.Unlock()

	// Same priority → smallest area (innermost) wins.
	z := FindZoneAt(1500, 1500)
	if z == nil || z.ID != "small_ngu" {
		t.Fatalf("expected smaller-area zone to win, got %+v", z)
	}
	// Outside the inner zone, the larger one still matches.
	z = FindZoneAt(3000, 3000)
	if z == nil || z.ID != "large_ngu" {
		t.Fatalf("expected large zone outside inner, got %+v", z)
	}
}

func TestRecordVisitedSubZone(t *testing.T) {
	p := &models.Player{}
	recordVisitedSubZone(p, "a:1:1")
	recordVisitedSubZone(p, "a:1:1")
	recordVisitedSubZone(p, "a:1:2")
	if len(p.VisitedSubZones) != 2 {
		t.Fatalf("expected 2 unique sub-zones, got %d", len(p.VisitedSubZones))
	}
}
