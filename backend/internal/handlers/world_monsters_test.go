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
	previousPlayers, previousAction, previousBroadcaster := loadWorldCombatPlayers, applyWorldMonsterAction, WorldCombatBroadcaster
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
		WorldCombatBroadcaster = previousBroadcaster
		SetWSConnectedProvider(nil)
	})
	return &events, &calls
}

func TestWorldMonsterAttackCooldownAndSlowTickResolveOneAction(t *testing.T) {
	now := time.Now().UTC()
	player := worldCombatPlayer{ID: uuid.New(), HP: 100, Region: "field", X: 1, Y: 0, Defense: 2}
	monster := testMonster(now.Add(-time.Second))
	monster.State = MonsterStateAttack
	monster.TargetID = player.ID.String()
	monster.AttackRange = 2
	monster.BaseAttack = 12
	monster.AttackIntervalMS = 1500
	monster.NextAttackAt = now.Add(time.Second)
	events, calls := installWorldAttackTest(t, now, []WorldMonster{monster}, player)
	processWorldMonsterTick(context.Background(), now)
	if calls.Load() != 0 {
		t.Fatalf("attack before cooldown: %d", calls.Load())
	}
	*events = (*events)[:0]
	monster.NextAttackAt = now.Add(-10 * time.Second)
	worldMonsterBucketCursor.Store(0)
	// Reinstalling is unnecessary: expose the delayed monster through the loader again.
	loaded := false
	worldMonstersInBucket = func(context.Context, string, int, int) ([]WorldMonster, error) {
		if loaded {
			return nil, nil
		}
		loaded = true
		return []WorldMonster{monster}, nil
	}
	processWorldMonsterTick(context.Background(), now.Add(20*time.Second))
	if calls.Load() != 1 {
		t.Fatalf("slow tick resolved %d actions, want one", calls.Load())
	}
	var kinds []string
	for _, e := range *events {
		kinds = append(kinds, e.EventType)
	}
	if len(kinds) != 3 || kinds[0] != "action_started" || kinds[1] != "action_resolved" || kinds[2] != "entity_damaged" {
		t.Fatalf("events=%v", kinds)
	}
}

func TestWorldMonsterAttackRejectsInvalidTargets(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name   string
		mutate func(*worldCombatPlayer)
	}{
		{"dead", func(p *worldCombatPlayer) { p.HP = 0; p.Dead = true }},
		{"teleported", func(p *worldCombatPlayer) { p.Region = "elsewhere" }},
		{"out_of_range", func(p *worldCombatPlayer) { p.X = 20 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := worldCombatPlayer{ID: uuid.New(), HP: 100, Region: "field", X: 1}
			tc.mutate(&p)
			m := testMonster(now.Add(-time.Second))
			m.State = MonsterStateAttack
			m.TargetID = p.ID.String()
			m.AttackRange = 2
			m.NextAttackAt = now.Add(-time.Second)
			_, calls := installWorldAttackTest(t, now, []WorldMonster{m}, p)
			processWorldMonsterTick(context.Background(), now)
			if calls.Load() != 0 {
				t.Fatalf("invalid target was damaged")
			}
		})
	}
}

func TestWorldMonsterAttackRejectsDisconnectedTarget(t *testing.T) {
	now := time.Now().UTC()
	p := worldCombatPlayer{ID: uuid.New(), HP: 100, Region: "field", X: 1}
	m := testMonster(now.Add(-time.Second))
	m.State, m.TargetID, m.AttackRange, m.NextAttackAt = MonsterStateAttack, p.ID.String(), 2, now.Add(-time.Second)
	_, calls := installWorldAttackTest(t, now, []WorldMonster{m}, p)
	SetWSConnectedProvider(func(uuid.UUID) bool { return false })
	processWorldMonsterTick(context.Background(), now)
	if calls.Load() != 0 {
		t.Fatal("disconnected target was damaged")
	}
}

func TestWorldMonsterAttackRejectsSafeZoneTarget(t *testing.T) {
	now := time.Now().UTC()
	p := worldCombatPlayer{ID: uuid.New(), HP: 100, Region: HubRegionID, X: 15000, Y: 10000}
	if !IsSafeZone(p.Region, p.X, p.Y) {
		t.Skip("test zone registry has no hub fixture")
	}
	m := testMonster(now.Add(-time.Second))
	m.RegionID, m.X, m.Y = p.Region, float64(p.X), float64(p.Y)
	m.SpawnX, m.SpawnY = m.X, m.Y
	m.State, m.TargetID, m.AttackRange, m.NextAttackAt = MonsterStateAttack, p.ID.String(), 2, now.Add(-time.Second)
	_, calls := installWorldAttackTest(t, now, []WorldMonster{m}, p)
	processWorldMonsterTick(context.Background(), now)
	if calls.Load() != 0 {
		t.Fatal("safe-zone target was damaged")
	}
}

func TestWorldMonsterSimultaneousAttacksDeathExactlyOnce(t *testing.T) {
	now := time.Now().UTC()
	p := worldCombatPlayer{ID: uuid.New(), HP: 10, Region: "field", X: 1}
	m1 := testMonster(now.Add(-time.Second))
	m1.State = MonsterStateAttack
	m1.TargetID = p.ID.String()
	m1.AttackRange = 2
	m1.NextAttackAt = now.Add(-time.Second)
	m2 := m1
	m2.InstanceID = "m2"
	_, _ = installWorldAttackTest(t, now, []WorldMonster{m1, m2}, p)
	var hp atomic.Int64
	hp.Store(10)
	var deaths atomic.Int64
	applyWorldMonsterAction = func(context.Context, WorldMonster, worldCombatPlayer, string) (worldMonsterActionResult, error) {
		for {
			old := hp.Load()
			if old <= 0 {
				return worldMonsterActionResult{}, nil
			}
			next := old - 10
			if hp.CompareAndSwap(old, next) {
				d := next == 0
				if d {
					deaths.Add(1)
				}
				return worldMonsterActionResult{Applied: true, Died: d, Damage: 10, HP: float64(next)}, nil
			}
		}
	}
	processWorldMonsterTick(context.Background(), now)
	if deaths.Load() != 1 || hp.Load() != 0 {
		t.Fatalf("hp=%d deaths=%d, want exactly one death", hp.Load(), deaths.Load())
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
