package handlers

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func installMonsterTransitionTest(t *testing.T, monster WorldMonster, now time.Time, casOK bool) *[]WorldCombatEvent {
	t.Helper()
	playerID := uuid.New()
	previousIndex, previousLoader, previousCAS := worldPosIndex, worldMonstersInBucket, casWorldMonster
	previousBroadcaster := WorldCombatBroadcaster
	worldPosIndex = newPositionIndex(WorldPosIndexBucket)
	worldPosIndex.Reconcile([]WorldPosEntry{{ID: playerID.String(), Region: monster.RegionID, X: 4, Y: 0, TS: now.UnixMilli()}})
	loaded := false
	worldMonstersInBucket = func(_ context.Context, region string, bx, by int) ([]WorldMonster, error) {
		if loaded || region != monster.RegionID || bx != 0 || by != 0 {
			return nil, nil
		}
		loaded = true
		return []WorldMonster{monster}, nil
	}
	casWorldMonster = func(_ context.Context, _, _ cache.WorldMonster, _ int64, _, _, _, _ int, _ time.Duration) (bool, error) {
		return casOK, nil
	}
	SetWSConnectedProvider(func(id uuid.UUID) bool { return id == playerID })
	events := make([]WorldCombatEvent, 0, 1)
	WorldCombatBroadcaster = func(_ string, payload []byte, _ map[uuid.UUID]struct{}) {
		var event WorldCombatEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			t.Fatalf("decode combat event: %v", err)
		}
		events = append(events, event)
	}
	t.Cleanup(func() {
		worldPosIndex, worldMonstersInBucket, casWorldMonster = previousIndex, previousLoader, previousCAS
		WorldCombatBroadcaster = previousBroadcaster
		SetWSConnectedProvider(nil)
	})
	return &events
}

func TestMonsterMovedBroadcastsOnlyAfterSuccessfulCAS(t *testing.T) {
	now := time.Now().UTC()
	monster := testMonster(now.Add(-time.Second))
	monster.State = MonsterStateChase
	monster.TargetID = "existing-target"
	monster.RegionID = "field"
	monster.RespawnSeconds = 30

	failedEvents := installMonsterTransitionTest(t, monster, now, false)
	processWorldMonsterTick(context.Background(), now)
	if len(*failedEvents) != 0 {
		t.Fatalf("broadcast %d events after failed CAS", len(*failedEvents))
	}
}

func TestMonsterMovedBroadcastContainsPostCASAuthoritativeState(t *testing.T) {
	now := time.Now().UTC()
	playerID := uuid.NewString()
	monster := testMonster(now.Add(-time.Second))
	monster.State, monster.TargetID, monster.RegionID, monster.RespawnSeconds = MonsterStateChase, playerID, "field", 30
	events := installMonsterTransitionTest(t, monster, now, true)
	processWorldMonsterTick(context.Background(), now)
	if len(*events) != 1 || (*events)[0].EventType != "monster_moved" {
		t.Fatalf("events=%+v, want one monster_moved", *events)
	}
	event := (*events)[0]
	if event.Payload["state"] != MonsterStateChase || event.Payload["target_id"] == "" || event.Position.X <= monster.X || event.EntityRevision <= monster.Revision {
		t.Fatalf("event did not carry authoritative post-CAS entity: %+v", event)
	}
}

func TestMovementCoalescingDoesNotSuppressStateTransition(t *testing.T) {
	now := time.Now().UTC()
	monster := testMonster(now.Add(-time.Second))
	monster.State, monster.X, monster.LastSeenAt, monster.RegionID, monster.RespawnSeconds = MonsterStateChase, 3.5, now.Add(-100*time.Millisecond), "field", 30
	events := installMonsterTransitionTest(t, monster, now, true)
	processWorldMonsterTick(context.Background(), now)
	if len(*events) != 1 || (*events)[0].EventType != "action_started" || (*events)[0].Payload["state"] != MonsterStateAttack {
		t.Fatalf("coalescing lost transition: %+v", *events)
	}
}

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

func installMonsterWorkerTestState(t *testing.T, entries []WorldPosEntry, connected map[uuid.UUID]bool) *[]ActiveWorldBucket {
	t.Helper()
	previousIndex := worldPosIndex
	previousLoader := worldMonstersInBucket
	worldPosIndex = newPositionIndex(WorldPosIndexBucket)
	worldPosIndex.Reconcile(entries)
	queried := make([]ActiveWorldBucket, 0)
	var mu sync.Mutex
	worldMonstersInBucket = func(_ context.Context, region string, bx, by int) ([]WorldMonster, error) {
		mu.Lock()
		queried = append(queried, ActiveWorldBucket{Region: region, BX: bx, BY: by})
		mu.Unlock()
		return nil, nil
	}
	SetWSConnectedProvider(func(id uuid.UUID) bool { return connected[id] })
	t.Cleanup(func() {
		SetWSConnectedProvider(nil)
		worldPosIndex = previousIndex
		worldMonstersInBucket = previousLoader
	})
	return &queried
}

func TestWorldMonsterTickSkipsRegionWithoutOnlinePlayers(t *testing.T) {
	now := time.Now()
	playerID := uuid.New()
	queried := installMonsterWorkerTestState(t, []WorldPosEntry{{ID: playerID.String(), Region: "empty", X: 16, Y: 16, TS: now.UnixMilli()}}, map[uuid.UUID]bool{})

	processWorldMonsterTick(context.Background(), now)
	if len(*queried) != 0 {
		t.Fatalf("queried %d buckets for an offline region: %#v", len(*queried), *queried)
	}
}

func TestWorldMonsterTickIgnoresRecentlyActiveDisconnectedPlayer(t *testing.T) {
	now := time.Now()
	playerID := uuid.New()
	queried := installMonsterWorkerTestState(t, []WorldPosEntry{{ID: playerID.String(), Region: "field", X: 16, Y: 16, TS: now.Add(-WorldPosActiveWindow / 2).UnixMilli()}}, map[uuid.UUID]bool{})

	processWorldMonsterTick(context.Background(), now)
	if len(*queried) != 0 {
		t.Fatalf("disconnected player activated %d buckets", len(*queried))
	}
}

func TestWorldMonsterTickDeduplicatesSharedAOIBuckets(t *testing.T) {
	now := time.Now()
	p1, p2 := uuid.New(), uuid.New()
	queried := installMonsterWorkerTestState(t, []WorldPosEntry{
		{ID: p1.String(), Region: "field", X: 16, Y: 16, TS: now.UnixMilli()},
		{ID: p2.String(), Region: "field", X: 17, Y: 17, TS: now.UnixMilli()},
	}, map[uuid.UUID]bool{p1: true, p2: true})

	processWorldMonsterTick(context.Background(), now)
	seen := make(map[ActiveWorldBucket]struct{}, len(*queried))
	for _, bucket := range *queried {
		if _, duplicate := seen[bucket]; duplicate {
			t.Fatalf("bucket queried more than once: %#v", bucket)
		}
		seen[bucket] = struct{}{}
	}
	if len(*queried) == 0 {
		t.Fatal("online shared AOI did not activate any bucket")
	}
}

func TestWorldMonsterTickStopsOldRegionAfterTeleport(t *testing.T) {
	now := time.Now()
	playerID := uuid.New()
	queried := installMonsterWorkerTestState(t, []WorldPosEntry{{ID: playerID.String(), Region: "old", X: 16, Y: 16, TS: now.UnixMilli()}}, map[uuid.UUID]bool{playerID: true})
	processWorldMonsterTick(context.Background(), now)
	if len(*queried) == 0 {
		t.Fatal("old region was not initially active")
	}

	*queried = (*queried)[:0]
	worldPosIndex.Set(WorldPosEntry{ID: playerID.String(), Region: "new", X: 16, Y: 16, TS: now.UnixMilli()})
	processWorldMonsterTick(context.Background(), now.Add(time.Second))
	if len(*queried) == 0 {
		t.Fatal("new region did not become active after teleport")
	}
	for _, bucket := range *queried {
		if bucket.Region != "new" {
			t.Fatalf("teleported player kept old region active: %#v", *queried)
		}
	}
}

func TestSpawnMaterializationUsesAuthoritativePlayerRadius(t *testing.T) {
	player := WorldPosEntry{Region: "field", X: 100, Y: 200}
	if !spawnWithinPlayerRadius(monsterSpawnRow{RegionID: "field", X: 115, Y: 220}, player, WorldMonsterActivationRadius) {
		t.Fatal("spawn exactly on authoritative radius was excluded")
	}
	if spawnWithinPlayerRadius(monsterSpawnRow{RegionID: "field", X: 125, Y: 225}, player, WorldMonsterActivationRadius) {
		t.Fatal("spawn in query square but outside authoritative radius was included")
	}
}

func TestWorldMonsterWorkerStopsWithContext(t *testing.T) {
	now := time.Now()
	playerID := uuid.New()
	installMonsterWorkerTestState(t, []WorldPosEntry{{ID: playerID.String(), Region: "field", X: 16, Y: 16, TS: now.UnixMilli()}}, map[uuid.UUID]bool{playerID: true})
	var calls atomic.Int64
	worldMonstersInBucket = func(context.Context, string, int, int) ([]WorldMonster, error) {
		calls.Add(1)
		return nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := startWorldMonsterWorker(ctx, 5*time.Millisecond)
	deadline := time.Now().Add(time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if calls.Load() == 0 {
		t.Fatal("worker never ticked")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}
	stoppedAt := calls.Load()
	time.Sleep(15 * time.Millisecond)
	if calls.Load() != stoppedAt {
		t.Fatalf("worker ticked after stopping: calls %d -> %d", stoppedAt, calls.Load())
	}
}

func TestMaterializeSpawnReusesRuntimeWhileTTLRemains(t *testing.T) {
	previousMaterialize := materializeWorldMonster
	previousTemplate, hadTemplate := models.MonsterRegistry["ttl-test"]
	t.Cleanup(func() {
		materializeWorldMonster = previousMaterialize
		if hadTemplate {
			models.MonsterRegistry["ttl-test"] = previousTemplate
		} else {
			delete(models.MonsterRegistry, "ttl-test")
		}
	})
	models.MonsterRegistry["ttl-test"] = models.MonsterTemplate{ID: "ttl-test", MaxHP: 100, BaseAttack: 10, BaseDefense: 5}
	var runtime *WorldMonster
	var expiresAt time.Time
	materializeWorldMonster = func(_ context.Context, candidate WorldMonster, _, _ int, ttl time.Duration) (WorldMonster, bool, error) {
		now := time.Now()
		if runtime != nil && now.Before(expiresAt) {
			return *runtime, false, nil
		}
		runtime = &candidate
		expiresAt = now.Add(ttl)
		return candidate, true, nil
	}
	spawn := monsterSpawnRow{ID: "spawn-1", RegionID: "field", MonsterID: "ttl-test", X: 4, Y: 4, RespawnSeconds: 30, Rank: 1, SubRank: 1}
	first, created, err := materializeSpawn(context.Background(), spawn, time.Now().UTC())
	if err != nil || !created {
		t.Fatalf("first materialization = (%+v, %v, %v), want created", first, created, err)
	}
	second, created, err := materializeSpawn(context.Background(), spawn, time.Now().UTC().Add(time.Minute))
	if err != nil || created || second.InstanceID != first.InstanceID {
		t.Fatalf("returning player materialization = (%+v, %v, %v), want existing %s", second, created, err, first.InstanceID)
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
