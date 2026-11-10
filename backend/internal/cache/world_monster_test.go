package cache

import (
	"testing"
	"time"
)

func TestWorldMonstersInBucketsWithoutRedis(t *testing.T) {
	previous := Client
	Client = nil
	t.Cleanup(func() { Client = previous })

	if got, err := WorldMonstersInBuckets(t.Context(), "region", nil); err != nil || len(got) != 0 {
		t.Fatalf("empty coords = (%v, %v), want empty map and nil error", got, err)
	}
	if _, err := WorldMonstersInBuckets(t.Context(), "region", []WorldMonsterBucketCoord{{X: 1, Y: 2}}); err == nil {
		t.Fatal("expected redis unavailable error")
	}
	if _, err := WorldMonstersInBucket(t.Context(), "region", 1, 2); err == nil {
		t.Fatal("expected redis unavailable error from single-bucket wrapper")
	}
}

func TestFreshestWorldBossesKeepsOneActorPerSpawn(t *testing.T) {
	older := time.Date(2026, 10, 18, 10, 0, 0, 0, time.UTC)
	newer := older.Add(time.Minute)
	members := []worldBossSetMember{
		{id: "old", monster: WorldMonster{InstanceID: "old", SpawnID: "mapcell:r1:5:5", LastUpdateAt: older, Revision: 40}},
		{id: "new", monster: WorldMonster{InstanceID: "new", SpawnID: "mapcell:r1:5:5", LastUpdateAt: newer, Revision: 2}},
		{id: "other", monster: WorldMonster{InstanceID: "other", SpawnID: "mapcell:r1:9:9", LastUpdateAt: older, Revision: 1}},
		{id: "empty", monster: WorldMonster{InstanceID: "empty", SpawnID: "", LastUpdateAt: older}},
	}

	kept, duplicates := freshestWorldBosses(members)

	if len(kept) != 3 || len(duplicates) != 1 {
		t.Fatalf("kept=%d duplicates=%d, want 3/1", len(kept), len(duplicates))
	}
	if duplicates[0].id != "old" {
		t.Fatalf("duplicate = %q, want the stale instance", duplicates[0].id)
	}
	for _, member := range kept {
		if member.id == "old" {
			t.Fatal("stale duplicate must not be kept")
		}
	}
}
