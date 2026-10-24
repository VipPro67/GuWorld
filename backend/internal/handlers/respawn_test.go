package handlers

import (
	"strings"
	"testing"

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
		{ID: "hub_test", Name: "Thành Thử", ZoneType: ZoneTypeHub, ParentID: "dai_vuc_test", EntryX: 1500, EntryY: 1500},
	}
}

func TestResolveDeathRespawn_UsesNearestHub(t *testing.T) {
	withRespawnTestZones(t, respawnTestZones())

	result := ResolveDeathRespawn("khu_test", 2000, 2000)
	if result.Kind != RespawnHub {
		t.Fatalf("kind = %v, want RespawnHub (%s,%d,%d)", result.Kind, result.Region, result.X, result.Y)
	}
	if result.Region != "hub_test" || result.X != 1500 || result.Y != 1500 {
		t.Fatalf("respawn = %s (%d,%d), want hub_test (1500,1500)", result.Region, result.X, result.Y)
	}
}

func TestResolveDeathRespawn_NoSafeZoneStaysAtDeathSpot(t *testing.T) {
	withRespawnTestZones(t, nil)

	result := ResolveDeathRespawn("khu_test", 123, 456)
	if result.Kind != RespawnStay {
		t.Fatalf("kind = %v, want RespawnStay", result.Kind)
	}
	if result.Region != "khu_test" || result.X != 123 || result.Y != 456 {
		t.Fatalf("respawn = %s (%d,%d), want death spot", result.Region, result.X, result.Y)
	}
}

func TestRespawnResultMessage(t *testing.T) {
	withRespawnTestZones(t, respawnTestZones())

	hub := RespawnResult{Region: "hub_test", X: 1500, Y: 1500, Kind: RespawnHub}
	if msg := hub.Message(); !strings.Contains(msg, "Thành Thử") || !strings.Contains(msg, "1500") {
		t.Fatalf("hub message = %q", msg)
	}

	stay := RespawnResult{Region: "khu_test", X: 123, Y: 456, Kind: RespawnStay}
	if msg := stay.Message(); !strings.Contains(msg, "vị trí tử vong") {
		t.Fatalf("stay message = %q", msg)
	}
}
