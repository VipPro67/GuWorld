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
	}
	zoneReg.mu.Unlock()

	z := FindZoneAt(1500, 1500)
	if z == nil || z.ID != "ngu_vuc_nested" {
		t.Fatalf("expected innermost named zone, got %+v", z)
	}
	z = FindZoneAt(5000, 5000)
	if z == nil || z.ID != "dai_vuc_test" {
		t.Fatalf("expected large zone fallback, got %+v", z)
	}
	z = FindZoneAt(15500, 10000)
	if z == nil || z.ID != "hub_test" {
		t.Fatalf("expected hub precedence, got %+v", z)
	}
	if DeriveRegionAt(999999, 999999) != HubRegionID {
		t.Fatalf("out-of-zone coords should fall back to hub")
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
