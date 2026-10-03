package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func testMonster(now time.Time) WorldMonster {
	return WorldMonster{InstanceID: "m1", SpawnID: "s1", RegionID: "field", X: 0, Y: 0, SpawnX: 0, SpawnY: 0, HP: 100, MaxHP: 100, State: MonsterStateIdle, Revision: 1, LastUpdateAt: now, AggroRadius: 10, AttackRange: 1, LeashRadius: 15, Speed: 2}
}

func TestSelectMonsterTargetDeterministicAndRegionSafe(t *testing.T) {
	now := time.Now()
	m := testMonster(now)
	candidates := []WorldPosEntry{{ID: "b", Region: "field", X: 3, Y: 0, TS: now.UnixMilli()}, {ID: "a", Region: "field", X: -3, Y: 0, TS: now.UnixMilli()}, {ID: "other", Region: "elsewhere", X: 1, Y: 0, TS: now.UnixMilli()}}
	got, ok := selectMonsterTarget(m, candidates, func(string) bool { return true })
	if !ok || got.ID != "a" {
		t.Fatalf("target=%+v ok=%v, want deterministic a", got, ok)
	}
}

func TestWorldMonsterChaseAttackChaseAndElapsedMovement(t *testing.T) {
	now := time.Now()
	m := testMonster(now)
	id := uuid.NewString()
	active := func(string) bool { return true }
	c := []WorldPosEntry{{ID: id, Region: "field", X: 5, Y: 0, TS: now.UnixMilli()}}
	m, event := tickWorldMonster(m, c, now.Add(2*time.Second), active)
	if m.State != MonsterStateChase || m.X != 4 || event != "monster_targeted" {
		t.Fatalf("chase=%+v event=%s", m, event)
	}
	c[0].X = 5
	m, event = tickWorldMonster(m, c, now.Add(2500*time.Millisecond), active)
	if m.State != MonsterStateAttack || event != "action_started" {
		t.Fatalf("attack=%+v event=%s", m, event)
	}
	c[0].X = 8
	m, event = tickWorldMonster(m, c, now.Add(3*time.Second), active)
	if m.State != MonsterStateChase {
		t.Fatalf("want chase after target leaves range: %+v", m)
	}
}

func TestWorldMonsterLeashReturnAndStatusExpiry(t *testing.T) {
	now := time.Now()
	m := testMonster(now)
	m.State = MonsterStateChase
	m.X = 10
	m.TargetID = "p"
	m.Statuses = []StatusEffect{{ID: "old", ExpiresAt: now.Add(-time.Second)}, {ID: "live", ExpiresAt: now.Add(time.Minute)}}
	c := []WorldPosEntry{{ID: "p", Region: "field", X: 20, Y: 0, TS: now.UnixMilli()}}
	m, _ = tickWorldMonster(m, c, now.Add(time.Second), func(string) bool { return true })
	if m.State != MonsterStateReturn || len(m.Statuses) != 1 {
		t.Fatalf("return/expiry failed: %+v", m)
	}
	for i := 0; i < 10 && m.State != MonsterStateIdle; i++ {
		m, _ = tickWorldMonster(m, nil, m.LastUpdateAt.Add(time.Second), func(string) bool { return false })
	}
	if m.State != MonsterStateIdle || m.X != m.SpawnX || m.HP != m.MaxHP {
		t.Fatalf("did not reset at spawn: %+v", m)
	}
}

func TestWorldMonsterWorkerStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	StartWorldMonsterWorker(ctx)
	cancel()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("context did not cancel")
	}
}

func TestMonsterBucketMatchesPlayerVisionScale(t *testing.T) {
	tests := []struct{ x, y, bx, by int }{
		{0, 0, 0, 0}, {31, 31, 0, 0}, {32, 32, 1, 1},
		{63, 95, 1, 2}, {-1, -1, -1, -1}, {-32, -32, -1, -1}, {-33, -33, -2, -2},
	}
	for _, tt := range tests {
		bx, by := monsterBucket(tt.x, tt.y)
		if bx != tt.bx || by != tt.by {
			t.Errorf("monsterBucket(%d,%d)=(%d,%d), want (%d,%d)", tt.x, tt.y, bx, by, tt.bx, tt.by)
		}
	}
	if WorldMonsterAOIBucket <= WorldMonsterActivationRadius {
		t.Fatalf("bucket side %d should exceed activation radius %d", WorldMonsterAOIBucket, WorldMonsterActivationRadius)
	}
}

func TestActiveMonsterBucketsCoverOnlyIntersectingVisionBuckets(t *testing.T) {
	players := []WorldPosEntry{{ID: "p1", Region: "field", X: 16, Y: 16}}
	buckets := activeMonsterBuckets(players, 10, 20)
	if len(buckets) != 1 || buckets[0].BX != 0 || buckets[0].BY != 0 {
		t.Fatalf("centered 10-cell view buckets = %#v, want only bucket 0,0", buckets)
	}

	players[0].X, players[0].Y = 31, 31
	buckets = activeMonsterBuckets(players, 10, 20)
	if len(buckets) != 4 {
		t.Fatalf("boundary view loaded %d buckets, want 4", len(buckets))
	}

	duplicated := []WorldPosEntry{
		{ID: "p1", Region: "field", X: 16, Y: 16},
		{ID: "p2", Region: "field", X: 17, Y: 17},
	}
	if got := activeMonsterBuckets(duplicated, 10, 20); len(got) != 1 {
		t.Fatalf("overlapping players loaded %d buckets, want one deduplicated bucket", len(got))
	}
}

func TestRotatingMonsterBucketWindowEventuallyVisitsEveryBucket(t *testing.T) {
	buckets := make([]ActiveWorldBucket, 10)
	for i := range buckets {
		buckets[i].BX = i
	}
	seen := map[int]bool{}
	for cursor := uint64(0); cursor < 12; cursor += 3 {
		for _, bucket := range rotatingMonsterBucketWindow(buckets, 3, cursor) {
			seen[bucket.BX] = true
		}
	}
	if len(seen) != len(buckets) {
		t.Fatalf("rotating windows visited %d/%d buckets: %#v", len(seen), len(buckets), seen)
	}
}
