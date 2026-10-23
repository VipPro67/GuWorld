package handlers

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func withRespawnTestZones(t *testing.T, zones []models.Region) {
	t.Helper()

	zoneReg.mu.Lock()
	oldZones, oldByID := zoneReg.zones, zoneReg.byID
	zoneReg.zones = zones
	zoneReg.byID = make(map[string]*models.Region, len(zones))
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}
	zoneReg.mu.Unlock()

	t.Cleanup(func() {
		zoneReg.mu.Lock()
		zoneReg.zones = oldZones
		zoneReg.byID = oldByID
		zoneReg.mu.Unlock()
	})
}

func respawnTestZones() []models.Region {
	return []models.Region{
		{ID: "dai_vuc_test", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 9999, MaxY: 9999},
		{ID: "khu_test", ZoneType: ZoneTypeNamed, ParentID: "dai_vuc_test", MinX: 1000, MinY: 1000, MaxX: 3000, MaxY: 3000},
		{ID: "hub_test", Name: "Thành Thử", ZoneType: ZoneTypeHub, ParentID: "dai_vuc_test", EntryX: 500, EntryY: 500},
		{ID: "sect_region", Name: "Sơn Môn", ZoneType: ZoneTypeNamed, ParentID: "dai_vuc_test", MinX: 4000, MinY: 4000, MaxX: 5000, MaxY: 5000},
	}
}

func TestResolveDeathRespawn_LowRankUsesNearestHub(t *testing.T) {
	withRespawnTestZones(t, respawnTestZones())

	result := resolveDeathRespawn(respawnProfile{Rank: 4}, 0.9, "khu_test", 2000, 2000)
	if result.Kind != RespawnHub {
		t.Fatalf("kind = %v, want RespawnHub (%s,%d,%d)", result.Kind, result.Region, result.X, result.Y)
	}
	if result.Region != "hub_test" || result.X != 500 || result.Y != 500 {
		t.Fatalf("respawn = %s (%d,%d), want hub_test (500,500)", result.Region, result.X, result.Y)
	}
}

func TestResolveDeathRespawn_HighRankWithoutSectRandomizes(t *testing.T) {
	withRespawnTestZones(t, respawnTestZones())

	result := resolveDeathRespawn(respawnProfile{Rank: 5}, 0.1, "khu_test", 2000, 2000)
	if result.Kind != RespawnRandom {
		t.Fatalf("kind = %v, want RespawnRandom", result.Kind)
	}
	if result.Region != "khu_test" {
		t.Fatalf("region = %s, want khu_test", result.Region)
	}
	if result.X < 1000 || result.X > 3000 || result.Y < 1000 || result.Y > 3000 {
		t.Fatalf("random point (%d,%d) is outside khu_test", result.X, result.Y)
	}
}

func TestResolveDeathRespawn_HighRankSectRoll(t *testing.T) {
	withRespawnTestZones(t, respawnTestZones())
	profile := respawnProfile{
		Rank:       5,
		HasSect:    true,
		SectRegion: "sect_region",
		SectName:   "Thanh Vân Tông",
		SectX:      4500,
		SectY:      4600,
	}

	sect := resolveDeathRespawn(profile, 0.49, "khu_test", 2000, 2000)
	if sect.Kind != RespawnSect {
		t.Fatalf("kind = %v, want RespawnSect", sect.Kind)
	}
	if sect.Region != "sect_region" || sect.X != 4500 || sect.Y != 4600 || sect.SectName != "Thanh Vân Tông" {
		t.Fatalf("sect respawn = %+v", sect)
	}

	random := resolveDeathRespawn(profile, 0.5, "khu_test", 2000, 2000)
	if random.Kind != RespawnRandom {
		t.Fatalf("roll 0.5 kind = %v, want RespawnRandom", random.Kind)
	}
	if random.Region != "khu_test" {
		t.Fatalf("random region = %s, want khu_test", random.Region)
	}
}

func TestResolveDeathRespawn_StaleSectRegionFallsBack(t *testing.T) {
	withRespawnTestZones(t, respawnTestZones())
	profile := respawnProfile{
		Rank:       6,
		HasSect:    true,
		SectRegion: "deleted_region",
		SectName:   "Tông Đã Tan",
		SectX:      4500,
		SectY:      4600,
	}

	result := resolveDeathRespawn(profile, 0.1, "khu_test", 2000, 2000)
	if result.Kind != RespawnRandom {
		t.Fatalf("kind = %v, want RespawnRandom fallback", result.Kind)
	}
	if result.Region != "khu_test" {
		t.Fatalf("region = %s, want khu_test", result.Region)
	}
}

func TestResolveDeathRespawn_NoZonesStaysAtDeathSpot(t *testing.T) {
	withRespawnTestZones(t, nil)

	result := resolveDeathRespawn(respawnProfile{Rank: 6}, 0.1, "khu_test", 123, 456)
	if result.Kind != RespawnStay {
		t.Fatalf("kind = %v, want RespawnStay", result.Kind)
	}
	if result.Region != "khu_test" || result.X != 123 || result.Y != 456 {
		t.Fatalf("respawn = %s (%d,%d), want death spot", result.Region, result.X, result.Y)
	}
}

func TestFetchRespawnProfile_NilTx(t *testing.T) {
	profile := fetchRespawnProfile(nil, uuid.New())
	if profile.HasSect || profile.Rank != 0 {
		t.Fatalf("nil tx profile = %+v, want zero value", profile)
	}
}

func TestRespawnResultMessage(t *testing.T) {
	withRespawnTestZones(t, respawnTestZones())

	sect := RespawnResult{Region: "sect_region", X: 4500, Y: 4600, Kind: RespawnSect, SectName: "Thanh Vân Tông"}
	if msg := sect.Message(); !strings.Contains(msg, "Thanh Vân Tông") || !strings.Contains(msg, "4500") {
		t.Fatalf("sect message = %q", msg)
	}

	random := RespawnResult{Region: "khu_test", X: 2000, Y: 2000, Kind: RespawnRandom}
	if msg := random.Message(); !strings.Contains(msg, "ngẫu nhiên") || !strings.Contains(msg, "khu_test") {
		t.Fatalf("random message = %q", msg)
	}

	stay := RespawnResult{Region: "khu_test", X: 2000, Y: 2000, Kind: RespawnStay}
	if msg := stay.Message(); !strings.Contains(msg, "vị trí tử vong") {
		t.Fatalf("stay message = %q", msg)
	}
}
