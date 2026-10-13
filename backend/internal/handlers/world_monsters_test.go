package handlers

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestMonsterRespawnDelayDefaultsForUnsetSpawnValue(t *testing.T) {
	if got, want := monsterRespawnDelay(0), 300*time.Second; got != want {
		t.Fatalf("monsterRespawnDelay(0) = %s, want %s", got, want)
	}
	if got, want := monsterRespawnDelay(-1), 300*time.Second; got != want {
		t.Fatalf("monsterRespawnDelay(-1) = %s, want %s", got, want)
	}
	if got, want := monsterRespawnDelay(45), 45*time.Second; got != want {
		t.Fatalf("monsterRespawnDelay(45) = %s, want %s", got, want)
	}
}

// stubSafeZoneLookup removes the zone registry from the safe-zone check so these
// tests never reach db.DB. Safe-zone behaviour has its own dedicated tests
// (TestSelectMonsterTargetRejectsSafeZone) that stub the seam directly.
func stubSafeZoneLookup(t *testing.T, safe func(string, int, int) bool) {
	t.Helper()
	previous := isWorldMonsterSafeZone
	isWorldMonsterSafeZone = safe
	t.Cleanup(func() { isWorldMonsterSafeZone = previous })
}

// stubCounterAttack removes the Redis-backed counterattack from a tick so the
// harness never touches cache.Client. It reports REVISION_CONFLICT, which is a
// no-land outcome that broadcasts no counterattack event — a real hit would
// always add an extra "entity_damaged". Tests that assert counterattack
// behaviour install their own stub.
func stubCounterAttack(t *testing.T) {
	t.Helper()
	applyWorldPlayerCounter = func(_ context.Context, monster WorldMonster, _ worldCombatPlayer, _ string, _ time.Time) (cache.WorldMonsterHitResult, WorldMonster, float64, error) {
		return cache.WorldMonsterRevisionConflict, monster, 0, nil
	}
	t.Cleanup(func() { applyWorldPlayerCounter = applyWorldCounterattack })
}

func installMonsterTransitionTest(t *testing.T, monster WorldMonster, now time.Time, casOK bool) *[]WorldCombatEvent {
	t.Helper()
	playerID := uuid.New()
	previousIndex, previousLoader, previousCAS := worldPosIndex, worldMonstersInBucket, casWorldMonster
	previousBroadcaster := WorldCombatBroadcaster
	stubSafeZoneLookup(t, func(string, int, int) bool { return false })
	stubCounterAttack(t)
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
	monster.Rank = 9 // leash >= 4 cells so the target at X=4 stays reachable
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

func TestPausedAttackTransitionsBackToChase(t *testing.T) {
	now := time.Now().UTC()
	monster := testMonster(now.Add(-time.Second))
	monster.Rank = 9 // leash >= 4 cells so the target at X=4 stays reachable
	monster.State, monster.X, monster.LastSeenAt, monster.RegionID, monster.RespawnSeconds = MonsterStateAttack, 2.0, now.Add(-100*time.Millisecond), "field", 30
	events := installMonsterTransitionTest(t, monster, now, true)
	processWorldMonsterTick(context.Background(), now)
	if len(*events) != 1 || (*events)[0].EventType != "monster_targeted" || (*events)[0].Payload["state"] != MonsterStateChase {
		t.Fatalf("paused attack did not return to chase: %+v", *events)
	}
}

func TestIdleMonsterPersistsOnlyOnChangeOrHeartbeat(t *testing.T) {
	stubSafeZoneLookup(t, func(string, int, int) bool { return false })
	now := time.Now()
	m := testMonster(now)
	active := func(string) bool { return true }

	// Nothing changed and the heartbeat has not elapsed: no CAS write.
	idle, event := tickWorldMonster(m, nil, now.Add(time.Second), active)
	if idle.Revision != m.Revision || event != "" {
		t.Fatalf("idle tick wrote: revision %d->%d event %q", m.Revision, idle.Revision, event)
	}

	// Heartbeat elapsed: persist so Redis TTL/tuning stay fresh.
	beat, _ := tickWorldMonster(m, nil, now.Add(worldMonsterHeartbeatInterval+time.Second), active)
	if beat.Revision != m.Revision+1 {
		t.Fatalf("heartbeat did not persist: revision %d", beat.Revision)
	}
}

func testMonster(now time.Time) WorldMonster {
	return WorldMonster{InstanceID: "m1", SpawnID: "s1", RegionID: "field", Rank: 5, X: 0, Y: 0, SpawnX: 0, SpawnY: 0, HP: 100, MaxHP: 100, State: MonsterStateIdle, Revision: 1, LastUpdateAt: now, AggroRadius: 10, AttackRange: 1, LeashRadius: 15, Speed: 2}
}

func TestWorldMonsterAggroProfileScalesWithRankAndStaysLocal(t *testing.T) {
	rank1Aggro, rank1Leash := worldMonsterAggroProfile(1)
	rank5Aggro, rank5Leash := worldMonsterAggroProfile(5)
	rank9Aggro, rank9Leash := worldMonsterAggroProfile(9)
	if rank1Aggro != 6 || rank5Aggro != 10 || rank9Aggro != 14 {
		t.Fatalf("unexpected aggro radii: rank1=%v rank5=%v rank9=%v", rank1Aggro, rank5Aggro, rank9Aggro)
	}
	if rank1Leash != 10 || rank5Leash != 15 || rank9Leash != 20 {
		t.Fatalf("unexpected leash radii: rank1=%v rank5=%v rank9=%v", rank1Leash, rank5Leash, rank9Leash)
	}
	if aggro, leash := worldMonsterAggroProfile(99); aggro != rank9Aggro || leash != rank9Leash {
		t.Fatalf("rank cap not applied: aggro=%v leash=%v", aggro, leash)
	}
}

func TestEliteWorldMonsterUsesCampHunterProfile(t *testing.T) {
	const templateID = "monster_test_elite_hunter"
	previous, existed := models.MonsterRegistry[templateID]
	models.MonsterRegistry[templateID] = models.MonsterTemplate{ID: templateID, Type: "Elite"}
	t.Cleanup(func() {
		if existed {
			models.MonsterRegistry[templateID] = previous
		} else {
			delete(models.MonsterRegistry, templateID)
		}
	})

	monster := WorldMonster{MonsterTemplateID: templateID, RegionID: "elite_profile_test", Rank: 3}
	applyWorldMonsterTuning(context.Background(), &monster)
	if monster.AggroRadius != worldMonsterEliteAggro || monster.LeashRadius != worldMonsterEliteLeash || monster.Speed != worldMonsterEliteSpeed {
		t.Fatalf("elite hunter profile = aggro %v, leash %v, speed %v", monster.AggroRadius, monster.LeashRadius, monster.Speed)
	}
	if monster.PatrolMinX != 0 || monster.PatrolMinY != 0 || monster.PatrolMaxX != WorldWidth-1 || monster.PatrolMaxY != WorldHeight-1 {
		t.Fatalf("elite should be bounded to the owning region: %+v", monster)
	}
}

func TestApplyWorldMonsterTuningUpgradesLegacyCachedMonster(t *testing.T) {
	previous := resolveWorldMonsterCell
	resolveWorldMonsterCell = func(int, int) models.Cell { return models.Cell{Content: "wolf", Rank: 7, SubRank: 2} }
	t.Cleanup(func() { resolveWorldMonsterCell = previous })
	monster := WorldMonster{MonsterTemplateID: "wolf", Rank: 0, AggroRadius: 12, LeashRadius: 24, Speed: 0.5}
	applyWorldMonsterTuning(context.Background(), &monster)
	if monster.Rank != 7 || monster.SubRank != 2 || monster.AggroRadius != 12 || monster.LeashRadius != 17.5 || monster.Speed != 0.15 {
		t.Fatalf("legacy monster was not retuned: %+v", monster)
	}
}

func TestApplyWorldMonsterTuningRecoversLegacyFixedSpawnRank(t *testing.T) {
	previousCell, previousRank := resolveWorldMonsterCell, loadWorldMonsterSpawnRank
	resolveWorldMonsterCell = func(int, int) models.Cell { return models.Cell{} }
	loadWorldMonsterSpawnRank = func(_ context.Context, spawnID string) (int, error) {
		if spawnID != "fixed-spawn" {
			t.Fatalf("unexpected spawn id %q", spawnID)
		}
		return 8, nil
	}
	t.Cleanup(func() { resolveWorldMonsterCell, loadWorldMonsterSpawnRank = previousCell, previousRank })
	monster := WorldMonster{SpawnID: "fixed-spawn", MonsterTemplateID: "wolf", Rank: 0}
	applyWorldMonsterTuning(context.Background(), &monster)
	if monster.Rank != 8 || monster.AggroRadius != 13 || monster.LeashRadius != 18.75 || monster.Speed != 0.15 {
		t.Fatalf("fixed legacy monster was not retuned: %+v", monster)
	}
}

func TestRankScaledAggroControlsActualTickBehavior(t *testing.T) {
	stubSafeZoneLookup(t, func(string, int, int) bool { return false })
	now := time.Now()
	active := func(string) bool { return true }

	rank1 := testMonster(now.Add(-time.Second))
	rank1.Rank = 1
	applyWorldMonsterTuning(context.Background(), &rank1)
	outside := []WorldPosEntry{{ID: "p", Region: "field", X: 7, Y: 0}}
	if got, _ := tickWorldMonster(rank1, outside, now, active); got.State != MonsterStateIdle {
		t.Fatalf("rank-1 monster aggroed outside its aggro range: %+v", got)
	}

	rank4 := testMonster(now.Add(-time.Second))
	rank4.Rank = 4
	applyWorldMonsterTuning(context.Background(), &rank4)
	inside := []WorldPosEntry{{ID: "p", Region: "field", X: 9, Y: 0}}
	if got, _ := tickWorldMonster(rank4, inside, now, active); got.State != MonsterStateChase || got.X != 0.15 {
		t.Fatalf("rank-4 monster did not chase slowly from the increased aggro range: %+v", got)
	}

	rank9 := testMonster(now.Add(-time.Second))
	rank9.Rank = 9
	applyWorldMonsterTuning(context.Background(), &rank9)
	farther := []WorldPosEntry{{ID: "p", Region: "field", X: 4, Y: 0}}
	if got, _ := tickWorldMonster(rank9, farther, now, active); got.State != MonsterStateChase {
		t.Fatalf("rank-9 monster did not use its larger detection range: %+v", got)
	}
	rank9.State, rank9.X, rank9.TargetID = MonsterStateChase, 4, "p"
	beyondLeash := []WorldPosEntry{{ID: "p", Region: "field", X: 21, Y: 0}}
	if got, _ := tickWorldMonster(rank9, beyondLeash, now, active); got.State != MonsterStateReturn {
		t.Fatalf("monster did not return when target exceeded leash: %+v", got)
	}
}

func TestSelectMonsterTargetDeterministicAndRegionSafe(t *testing.T) {
	stubSafeZoneLookup(t, func(string, int, int) bool { return false })
	now := time.Now()
	m := testMonster(now)
	candidates := []WorldPosEntry{{ID: "b", Region: "field", X: 3, Y: 0, TS: now.UnixMilli()}, {ID: "a", Region: "field", X: -3, Y: 0, TS: now.UnixMilli()}, {ID: "other", Region: "elsewhere", X: 1, Y: 0, TS: now.UnixMilli()}}
	got, ok := selectMonsterTarget(m, candidates, func(string) bool { return true })
	if !ok || got.ID != "a" {
		t.Fatalf("target=%+v ok=%v, want deterministic a", got, ok)
	}
}

func TestSelectMonsterTargetRejectsSafeZone(t *testing.T) {
	previous := isWorldMonsterSafeZone
	isWorldMonsterSafeZone = func(_ string, x, _ int) bool { return x >= 2 }
	t.Cleanup(func() { isWorldMonsterSafeZone = previous })
	m := testMonster(time.Now())
	if _, ok := selectMonsterTarget(m, []WorldPosEntry{{ID: "safe", Region: "field", X: 2}}, func(string) bool { return true }); ok {
		t.Fatal("monster acquired a target inside a safe zone")
	}
}

func TestMoveMonsterTowardDoesNotTunnelThroughSafeZone(t *testing.T) {
	previous := isWorldMonsterSafeZone
	isWorldMonsterSafeZone = func(_ string, x, _ int) bool { return x == 2 }
	t.Cleanup(func() { isWorldMonsterSafeZone = previous })
	m := testMonster(time.Now())
	m.Speed, m.LeashRadius = 10, 20
	moveMonsterToward(&m, 5, 0, 1)
	if m.X != 0 || m.Y != 0 {
		t.Fatalf("monster crossed a safe cell: (%v,%v)", m.X, m.Y)
	}
}

func TestWorldMonsterChasesWithoutContactAttacking(t *testing.T) {
	stubSafeZoneLookup(t, func(string, int, int) bool { return false })
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
	if m.State != MonsterStateAttack || m.X != 4 || event != "monster_encounter_started" {
		t.Fatalf("monster should start encounter in attack range: monster=%+v event=%s", m, event)
	}
	c[0].X = 8
	m, event = tickWorldMonster(m, c, now.Add(3*time.Second), active)
	if m.State != MonsterStateChase || event != "monster_targeted" {
		t.Fatalf("want chase after target leaves range: %+v", m)
	}
}

func TestWorldMonsterLeashReturnAndStatusExpiry(t *testing.T) {
	stubSafeZoneLookup(t, func(string, int, int) bool { return false })
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

func TestProceduralMonsterCellBecomesRuntimeSpawn(t *testing.T) {
	entry := WorldPosEntry{Region: "dang_hon_son", X: 24798, Y: 13261}
	previousCell, previousRegion, previousCleared := resolveWorldMonsterCell, deriveWorldMonsterRegion, worldMonsterCellCleared
	resolveWorldMonsterCell = func(x, y int) models.Cell {
		if x == entry.X && y == entry.Y {
			return models.Cell{X: x, Y: y, Type: "Monster", Content: "monster_soul_devourer", Rank: 5, SubRank: 2}
		}
		return models.Cell{X: x, Y: y, Type: "Empty"}
	}
	deriveWorldMonsterRegion = func(int, int) string { return entry.Region }
	worldMonsterCellCleared = func(string, int, int) bool { return false }
	t.Cleanup(func() {
		resolveWorldMonsterCell, deriveWorldMonsterRegion, worldMonsterCellCleared = previousCell, previousRegion, previousCleared
	})
	cell := resolveWorldMonsterCell(entry.X, entry.Y)
	spawns := proceduralMonsterSpawns(entry, 1)
	found := false
	for _, spawn := range spawns {
		if spawn.X == entry.X && spawn.Y == entry.Y {
			found = spawn.MonsterID == cell.Content && spawn.ID == "procedural:dang_hon_son:24798:13261"
		}
	}
	if !found {
		t.Fatalf("procedural monster cell was not converted to a stable runtime spawn: %#v", spawns)
	}
	worldMonsterCellCleared = func(string, int, int) bool { return true }
	if cleared := proceduralMonsterSpawns(entry, 1); len(cleared) != 0 {
		t.Fatalf("cleared procedural cell rematerialized: %#v", cleared)
	}
}

func TestProceduralMonsterSpawnIdentityRoundTripsToCell(t *testing.T) {
	region, x, y, ok := proceduralMonsterCell("procedural:dang_hon_son:24798:13261")
	if !ok || region != "dang_hon_son" || x != 24798 || y != 13261 {
		t.Fatalf("round trip=(%q,%d,%d,%v)", region, x, y, ok)
	}
	if _, _, _, ok := proceduralMonsterCell("fixed-spawn-id"); ok {
		t.Fatal("fixed spawn was classified as procedural")
	}
}

func TestMonsterBucketCandidateRadiusCoversCornerAggro(t *testing.T) {
	centerX, centerY := WorldMonsterAOIBucket/2, WorldMonsterAOIBucket/2
	monsterX, monsterY := float64(WorldMonsterAOIBucket-1), float64(WorldMonsterAOIBucket-1)
	playerX, playerY := 39, 31
	if math.Hypot(float64(playerX)-monsterX, float64(playerY)-monsterY) > 12 {
		t.Fatal("fixture player must be in aggro range")
	}
	if math.Hypot(float64(playerX-centerX), float64(playerY-centerY)) > worldMonsterBucketQueryRadius {
		t.Fatal("bucket candidate query radius omits a valid corner target")
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

func installWorldAttackTest(t *testing.T, now time.Time, monsters []WorldMonster, player worldCombatPlayer) (*[]WorldCombatEvent, *atomic.Int64) {
	t.Helper()
	previousIndex, previousLoader, previousCAS := worldPosIndex, worldMonstersInBucket, casWorldMonster
	previousPlayers, previousAction, previousCounter, previousBroadcaster := loadWorldCombatPlayers, applyWorldMonsterAction, applyWorldPlayerCounter, WorldCombatBroadcaster
	stubSafeZoneLookup(t, func(string, int, int) bool { return false })
	stubCounterAttack(t)
	worldPosIndex = newPositionIndex(WorldPosIndexBucket)
	worldPosIndex.Set(WorldPosEntry{ID: player.ID.String(), Region: player.Region, X: player.X, Y: player.Y, TS: now.UnixMilli()})
	loaded := false
	worldMonstersInBucket = func(context.Context, string, int, int) ([]WorldMonster, error) {
		if loaded {
			return nil, nil
		}
		loaded = true
		return monsters, nil
	}
	loadWorldCombatPlayers = func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]worldCombatPlayer, error) {
		return map[uuid.UUID]worldCombatPlayer{player.ID: player}, nil
	}
	var calls atomic.Int64
	applyWorldMonsterAction = func(_ context.Context, _ WorldMonster, _ worldCombatPlayer, _ string) (worldMonsterActionResult, error) {
		calls.Add(1)
		return worldMonsterActionResult{Applied: true, Damage: 10, HP: player.HP - 10}, nil
	}
	casWorldMonster = func(_ context.Context, _, next cache.WorldMonster, _ int64, _, _, _, _ int, _ time.Duration) (bool, error) {
		monsters[0] = next
		return true, nil
	}
	SetWSConnectedProvider(func(id uuid.UUID) bool { return id == player.ID })
	events := []WorldCombatEvent{}
	WorldCombatBroadcaster = func(_ string, payload []byte, _ map[uuid.UUID]struct{}) {
		var e WorldCombatEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			t.Fatal(err)
		}
		events = append(events, e)
	}
	t.Cleanup(func() {
		worldPosIndex = previousIndex
		worldMonstersInBucket = previousLoader
		casWorldMonster = previousCAS
		loadWorldCombatPlayers = previousPlayers
		applyWorldMonsterAction = previousAction
		applyWorldPlayerCounter = previousCounter
		WorldCombatBroadcaster = previousBroadcaster
		SetWSConnectedProvider(nil)
	})
	return &events, &calls
}

func TestWorldMonsterContactDoesNotDamageOrCounterattack(t *testing.T) {
	now := time.Now().UTC()
	p := worldCombatPlayer{ID: uuid.New(), HP: 100, Region: "field", X: 1, Attack: 25}
	m := testMonster(now.Add(-time.Second))
	m.State, m.TargetID, m.AttackRange, m.NextAttackAt = MonsterStateAttack, p.ID.String(), 2, now.Add(-time.Second)
	_, attacks := installWorldAttackTest(t, now, []WorldMonster{m}, p)
	var counters atomic.Int64
	applyWorldPlayerCounter = func(_ context.Context, monster WorldMonster, target worldCombatPlayer, commandID string, _ time.Time) (cache.WorldMonsterHitResult, WorldMonster, float64, error) {
		counters.Add(1)
		monster.HP -= 20
		monster.Revision++
		return cache.WorldMonsterHit, monster, 20, nil
	}
	processWorldMonsterTick(context.Background(), now)
	if attacks.Load() != 0 || counters.Load() != 0 {
		t.Fatalf("contact combat calls: attacks=%d counterattacks=%d, want zero", attacks.Load(), counters.Load())
	}
}

func TestMaterializeSpawnReusesRuntimeWhileTTLRemains(t *testing.T) {
	stubSafeZoneLookup(t, func(string, int, int) bool { return false })
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
