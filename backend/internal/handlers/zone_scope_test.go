package handlers

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// withTestZones swaps the in-memory zone registry for a unit test and restores
// it afterwards.
func withTestZones(t *testing.T, zones ...models.Region) {
	t.Helper()
	zoneReg.mu.Lock()
	previousZones, previousByID := zoneReg.zones, zoneReg.byID
	zoneReg.zones = zones
	zoneReg.byID = make(map[string]*models.Region, len(zones))
	for i := range zoneReg.zones {
		zoneReg.byID[zoneReg.zones[i].ID] = &zoneReg.zones[i]
	}
	zoneReg.mu.Unlock()
	t.Cleanup(func() {
		zoneReg.mu.Lock()
		zoneReg.zones, zoneReg.byID = previousZones, previousByID
		zoneReg.mu.Unlock()
	})
}

func testScopeZones() []models.Region {
	return []models.Region{
		{ID: "parent", ZoneType: ZoneTypeLarge, MinX: 0, MinY: 0, MaxX: 1000, MaxY: 1000, MinRank: 1, MaxRank: 9},
		{ID: "low_child", ParentID: "parent", ZoneType: ZoneTypeNamed, MinX: 0, MinY: 0, MaxX: 400, MaxY: 400, MinRank: 3, MaxRank: 5},
		{ID: "high_sibling", ParentID: "parent", ZoneType: ZoneTypeNamed, MinX: 401, MinY: 0, MaxX: 800, MaxY: 400, MinRank: 6, MaxRank: 8},
	}
}

// TestIsRegionWithinScopeIsDirectional guards the border-leak bug: a rank-8
// monster in a sibling zone must not share aggro/encounter scope with a low
// rank child zone, while parent-region actors stay visible to children.
func TestIsRegionWithinScopeIsDirectional(t *testing.T) {
	withTestZones(t, testScopeZones()...)

	if !isRegionWithinScope("parent", "low_child") {
		t.Fatal("parent region must cover its child zones")
	}
	if !isRegionWithinScope("low_child", "low_child") {
		t.Fatal("a region must be within its own scope")
	}
	if isRegionWithinScope("high_sibling", "low_child") {
		t.Fatal("sibling zones must not share scope (rank-8 monsters could aggro into a low zone)")
	}
	if isRegionWithinScope("low_child", "high_sibling") {
		t.Fatal("sibling zones must not share scope in either direction")
	}
	if isRegionWithinScope("low_child", "parent") {
		t.Fatal("a child zone must not claim its parent region's actors")
	}
}

func TestClampRankToZoneBand(t *testing.T) {
	withTestZones(t, testScopeZones()...)

	if got := clampRankToZoneBand("low_child", 8); got != 5 {
		t.Fatalf("clampRankToZoneBand(low_child, 8) = %d, want 5", got)
	}
	if got := clampRankToZoneBand("low_child", 3); got != 3 {
		t.Fatalf("rank inside the band must stay untouched, got %d", got)
	}
	if got := clampRankToZoneBand("missing_zone", 8); got != 8 {
		t.Fatalf("unknown zone must not clamp, got %d", got)
	}
}

func TestMonsterCannotRoamIntoLowerRankZone(t *testing.T) {
	withTestZones(t, testScopeZones()...)

	if monsterCanRoamInto(8, 200, 200) {
		t.Fatal("rank 8 monster must not enter a rank 3-5 zone")
	}
	if !monsterCanRoamInto(5, 200, 200) {
		t.Fatal("rank 5 monster must be allowed in a rank 3-5 zone")
	}
	if !monsterCanRoamInto(8, 600, 200) {
		t.Fatal("rank 8 monster must still roam its own rank 6-8 zone")
	}
}

func TestMoveMonsterTowardStopsAtLowerRankBorder(t *testing.T) {
	withTestZones(t, testScopeZones()...)

	monster := WorldMonster{
		RegionID: "high_sibling", MonsterTemplateID: "monster_roam_test", Rank: 8,
		X: 410, Y: 200, SpawnX: 410, SpawnY: 200, LeashRadius: 100, Speed: 10,
	}
	moveMonsterToward(&monster, 200, 200, 1)
	if monster.X != 410 || monster.Y != 200 {
		t.Fatalf("monster crossed into the low-rank zone: (%v,%v)", monster.X, monster.Y)
	}
}

func TestWorldMonsterCandidateClampsRankToZoneBand(t *testing.T) {
	withTestZones(t, testScopeZones()...)

	const templateID = "monster_rank_clamp_test"
	previous, existed := models.MonsterRegistry[templateID]
	models.MonsterRegistry[templateID] = models.MonsterTemplate{ID: templateID, MaxHP: 100, BaseAttack: 10, BaseDefense: 5}
	t.Cleanup(func() {
		if existed {
			models.MonsterRegistry[templateID] = previous
		} else {
			delete(models.MonsterRegistry, templateID)
		}
	})

	spawn := monsterSpawnRow{
		ID: "stale-spawn", RegionID: "low_child", MonsterID: templateID,
		X: 200, Y: 200, Rank: 8, SubRank: 1, RespawnSeconds: 60,
	}
	monster, err := worldMonsterCandidateForSpawn(spawn, time.Now().UTC())
	if err != nil {
		t.Fatalf("worldMonsterCandidateForSpawn: %v", err)
	}
	if monster.Rank != 5 {
		t.Fatalf("materialized rank = %d, want zone cap 5", monster.Rank)
	}
}
