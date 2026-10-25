package world

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

func TestManagerValidatesConfiguredZones(t *testing.T) {
	if _, err := NewManager(nil, 8); !errors.Is(err, ErrNoZones) {
		t.Fatalf("empty zone list: %v", err)
	}
	key := sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}
	if _, err := NewManager([]sim.ZoneKey{key, key}, 8); !errors.Is(err, ErrDuplicateZone) {
		t.Fatalf("duplicate zone key: %v", err)
	}
	tooMany := make([]sim.ZoneKey, MaxZonesPerProcess+1)
	for index := range tooMany {
		tooMany[index] = sim.ZoneKey{MapID: "map", InstanceID: "instance-" + strconv.Itoa(index), RulesetVersion: "v1"}
	}
	if _, err := NewManager(tooMany, 8); !errors.Is(err, ErrTooManyZones) {
		t.Fatalf("zone limit: %v", err)
	}

	regionalPack := make([]sim.ZoneKey, 125)
	for index := range regionalPack {
		regionalPack[index] = sim.ZoneKey{MapID: "region-" + strconv.Itoa(index), InstanceID: "realtime-preview", RulesetVersion: "v1"}
	}
	if _, err := NewManager(regionalPack, 8); err != nil {
		t.Fatalf("125-map KHU_VUC/SAFE_ZONE pack should fit in one process: %v", err)
	}
}

func TestManagerStartsEmptyZonesAndWaitsForShutdown(t *testing.T) {
	key := sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}
	manager, err := NewManager([]sim.ZoneKey{key}, 8)
	if err != nil {
		t.Fatal(err)
	}
	keys := manager.ZoneKeys()
	keys[0].MapID = "mutated"
	if manager.ZoneKeys()[0] != key {
		t.Fatal("ZoneKeys exposed mutable manager state")
	}
	if snapshot, err := manager.Snapshot(key); err != nil || snapshot.Tick != 0 || len(snapshot.Entities) != 0 {
		t.Fatalf("initial zone should be empty: snapshot=%+v err=%v", snapshot, err)
	}
	if _, err := manager.Snapshot(sim.ZoneKey{MapID: "missing", InstanceID: "preview", RulesetVersion: "v1"}); !errors.Is(err, ErrUnknownZone) {
		t.Fatalf("unknown zone: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for {
		snapshot, err := manager.Snapshot(key)
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Tick > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("zone runner did not advance")
		}
		time.Sleep(5 * time.Millisecond)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	if err := manager.Stop(shutdownCtx); err != nil {
		t.Fatal(err)
	}
	stoppedAt, err := manager.Snapshot(key)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * sim.TickDuration)
	afterWait, err := manager.Snapshot(key)
	if err != nil {
		t.Fatal(err)
	}
	if afterWait.Tick != stoppedAt.Tick {
		t.Fatalf("zone continued after Stop returned: stopped=%d current=%d", stoppedAt.Tick, afterWait.Tick)
	}
	if err := manager.Start(context.Background()); !errors.Is(err, ErrManagerStarted) {
		t.Fatalf("stopped manager restarted: %v", err)
	}
}

func TestManagerRefusesCanceledStartContext(t *testing.T) {
	manager, err := NewManager([]sim.ZoneKey{{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}}, 8)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := manager.Start(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Start with canceled context: %v", err)
	}
	if snapshot, err := manager.Snapshot(sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}); err != nil || snapshot.Tick != 0 {
		t.Fatalf("zone started despite canceled context: snapshot=%+v err=%v", snapshot, err)
	}
}

func TestManagerStatsReportsConfiguredMapAndSpawnMetadata(t *testing.T) {
	key := sim.ZoneKey{MapID: "world_map", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager, err := NewManagerWithMaps([]sim.ZoneKey{key}, 8, 5, map[sim.ZoneKey]ZoneMap{
		key: {
			Width: 1200, Height: 800, Hash: "test-map-hash", LegacyScale: 25,
			Monsters: []MonsterSpawn{{
				TemplateID: "monster_test", X: 10, Y: 10, MaxHP: 100, AttackDamage: 5,
				Speed: 3, Radius: 0.5, AggroRadius: 6, LeashRadius: 10, AttackRange: 1.2,
				AttackCooldownTicks: 2, RespawnTicks: 100,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	stats := manager.Stats()
	if len(stats) != 1 {
		t.Fatalf("zone stats count=%d, want 1", len(stats))
	}
	got := stats[0]
	if !got.MapLoaded || got.MapHash != "test-map-hash" || got.MapWidth != 1200 || got.MapHeight != 800 || got.LegacyScale != 25 || got.MonsterSpawns != 1 {
		t.Fatalf("map diagnostics missing: %+v", got)
	}
}

func TestCharacterLeaseAllowsOneOwnerAndRejectsStaleReconnects(t *testing.T) {
	key := sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}
	manager, err := NewManager([]sim.ZoneKey{key}, 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer manager.Stop(shutdownCtx)

	first, err := manager.ClaimCharacter("character-1", "session-a", key)
	if err != nil {
		t.Fatal(err)
	}
	if !manager.OwnsCharacter(first.CharacterID) {
		t.Fatal("claimed character should be marked realtime-owned")
	}
	if _, err := manager.ClaimCharacter("character-1", "session-b", key); !errors.Is(err, ErrCharacterOwned) {
		t.Fatalf("second controller claim: %v", err)
	}

	replacement, err := manager.ReplaceCharacterLease(first, "session-b")
	if err != nil {
		t.Fatal(err)
	}
	if manager.ReleaseCharacter(first) {
		t.Fatal("stale controller released replacement lease")
	}
	if !manager.OwnsCharacter(replacement.CharacterID) {
		t.Fatal("replacement lease lost ownership")
	}
	if !manager.ReleaseCharacter(replacement) || manager.OwnsCharacter(replacement.CharacterID) {
		t.Fatal("current controller could not release its lease")
	}
}

func TestCharacterLeaseRequiresRunningConfiguredZone(t *testing.T) {
	key := sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}
	manager, err := NewManager([]sim.ZoneKey{key}, 8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ClaimCharacter("character-1", "session-a", key); !errors.Is(err, ErrManagerNotRunning) {
		t.Fatalf("claim before start: %v", err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer manager.Stop(shutdownCtx)
	missing := sim.ZoneKey{MapID: "missing", InstanceID: "preview", RulesetVersion: "v1"}
	if _, err := manager.ClaimCharacter("character-1", "session-a", missing); !errors.Is(err, ErrUnknownZone) {
		t.Fatalf("claim unknown zone: %v", err)
	}
	if _, err := manager.ClaimCharacter(" ", "session-a", key); !errors.Is(err, ErrInvalidLease) {
		t.Fatalf("claim invalid character ID: %v", err)
	}
}

func TestJoinCharacterAddsEntityAndEnforcesZoneCapacity(t *testing.T) {
	key := sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}
	manager, err := NewManagerWithCapacity([]sim.ZoneKey{key}, 8, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer manager.Stop(shutdownCtx)

	first, err := manager.JoinCharacter("character-1", "session-a", key, playerEntity("character-1", 1))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot, _ := manager.Snapshot(key); len(snapshot.Entities) != 1 || snapshot.Entities[0].CharacterID != "character-1" {
		t.Fatalf("joined player missing from zone snapshot: %+v", snapshot.Entities)
	}
	if _, err := manager.JoinCharacter("character-2", "session-b", key, playerEntity("character-2", 2)); !errors.Is(err, ErrZoneFull) {
		t.Fatalf("join over capacity: %v", err)
	}
	replacement, err := manager.ReplaceCharacterLease(first, "session-c")
	if err != nil {
		t.Fatal(err)
	}
	if manager.LeaveCharacter(first) {
		t.Fatal("stale lease removed zone member after reconnect")
	}
	if manager.LeaveCharacter(OwnershipLease{CharacterID: first.CharacterID, ZoneKey: first.ZoneKey, Token: "stale"}) {
		t.Fatal("stale lease removed zone member")
	}
	if !manager.LeaveCharacter(replacement) {
		t.Fatal("current lease could not remove zone member")
	}
	if snapshot, _ := manager.Snapshot(key); len(snapshot.Entities) != 0 {
		t.Fatalf("left player remains in zone: %+v", snapshot.Entities)
	}
	if _, err := manager.JoinCharacter("character-2", "session-b", key, playerEntity("character-2", 2)); err != nil {
		t.Fatalf("zone capacity not released after leave: %v", err)
	}
}

func TestLeaseSnapshotFeedPublishesLatestZoneState(t *testing.T) {
	key := sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}
	manager, err := NewManager([]sim.ZoneKey{key}, 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer manager.Stop(shutdownCtx)
	lease, err := manager.JoinCharacter("character-1", "session-a", key, playerEntity("character-1", 1))
	if err != nil {
		t.Fatal(err)
	}
	feed, unsubscribe, err := manager.SubscribeSnapshots(lease)
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribe()
	initial := <-feed
	if len(initial.Entities) != 1 || initial.Entities[0].CharacterID != lease.CharacterID {
		t.Fatalf("initial subscription baseline missing owner: %+v", initial)
	}
	manager.mu.Lock()
	beforeUnsubscribe := len(manager.zones[key].subscribers)
	manager.mu.Unlock()
	if beforeUnsubscribe != 1 {
		t.Fatalf("snapshot subscriber count=%d, want 1", beforeUnsubscribe)
	}
	deadline := time.Now().Add(time.Second)
	receivedNewer := false
	for time.Now().Before(deadline) {
		select {
		case snapshot := <-feed:
			if snapshot.Tick > initial.Tick {
				receivedNewer = true
			}
		default:
		}
		if receivedNewer {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !receivedNewer {
		current, _ := manager.Snapshot(key)
		t.Fatalf("snapshot feed did not publish a newer zone tick: initial=%d current=%d pending=%d zonePending=%d", initial.Tick, current.Tick, len(feed), len(manager.zones[key].snapshots))
	}
	unsubscribe()
	manager.mu.Lock()
	afterUnsubscribe := len(manager.zones[key].subscribers)
	manager.mu.Unlock()
	if afterUnsubscribe != 0 {
		t.Fatalf("snapshot subscriber count after unsubscribe=%d, want 0", afterUnsubscribe)
	}
}

func TestSnapshotFeedAppliesPerCharacterAOI(t *testing.T) {
	key := sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}
	manager, err := NewManagerWithCapacity([]sim.ZoneKey{key}, 16, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer manager.Stop(shutdownCtx)
	owner, err := manager.JoinCharacter("owner", "session-owner", key, playerEntity("owner", 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.JoinCharacter("nearby", "session-nearby", key, playerEntity("nearby", 20)); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.JoinCharacter("distant", "session-distant", key, playerEntity("distant", 100)); err != nil {
		t.Fatal(err)
	}
	feed, unsubscribe, err := manager.SubscribeSnapshots(owner)
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribe()
	initial := <-feed
	visible := make(map[sim.CharacterID]bool, len(initial.Entities))
	for _, entity := range initial.Entities {
		visible[entity.CharacterID] = true
	}
	if len(visible) != 2 || !visible["owner"] || !visible["nearby"] || visible["distant"] {
		t.Fatalf("unexpected AOI entities: %+v", initial.Entities)
	}
}

func TestReplaceCharacterLeaseInvalidatesOldSnapshotFeed(t *testing.T) {
	key := sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}
	manager, err := NewManager([]sim.ZoneKey{key}, 16)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer manager.Stop(shutdownCtx)
	current, err := manager.JoinCharacter("character-1", "session-a", key, playerEntity("character-1", 1))
	if err != nil {
		t.Fatal(err)
	}
	oldFeed, unsubscribeOld, err := manager.SubscribeSnapshots(current)
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribeOld()
	<-oldFeed
	replacement, err := manager.ReplaceCharacterLease(current, "session-b")
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.After(time.Second)
	closed := false
	for {
		select {
		case _, ok := <-oldFeed:
			if !ok {
				closed = true
			}
		case <-deadline:
			t.Fatal("replacement left the previous snapshot feed active")
		}
		if closed {
			break
		}
	}
	if _, _, err := manager.SubscribeSnapshots(current); !errors.Is(err, ErrInvalidLease) {
		t.Fatalf("stale controller subscribed to snapshots: %v", err)
	}
	newFeed, unsubscribeNew, err := manager.SubscribeSnapshots(replacement)
	if err != nil {
		t.Fatal(err)
	}
	defer unsubscribeNew()
	if initial := <-newFeed; len(initial.Entities) != 1 || initial.Entities[0].CharacterID != replacement.CharacterID {
		t.Fatalf("replacement feed missing owner: %+v", initial)
	}
}

func playerEntity(characterID sim.CharacterID, x float64) sim.Entity {
	return sim.Entity{
		ID: sim.EntityID(characterID), CharacterID: characterID, Kind: sim.EntityPlayer,
		Transform: sim.Transform{Position: sim.Vec2{X: x, Y: 1}},
		Vitals:    sim.Vitals{HP: 100, MaxHP: 100}, Speed: 4, Radius: 0.4,
	}
}

func TestConcurrentCharacterClaimsHaveSingleWinner(t *testing.T) {
	key := sim.ZoneKey{MapID: "forest", InstanceID: "preview", RulesetVersion: "v1"}
	manager, err := NewManager([]sim.ZoneKey{key}, 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	defer manager.Stop(shutdownCtx)

	const contenders = 16
	results := make(chan error, contenders)
	start := make(chan struct{})
	for index := 0; index < contenders; index++ {
		go func(index int) {
			<-start
			_, claimErr := manager.ClaimCharacter("shared-character", "session-"+strconv.Itoa(index), key)
			results <- claimErr
		}(index)
	}
	close(start)

	winners := 0
	for index := 0; index < contenders; index++ {
		claimErr := <-results
		switch {
		case claimErr == nil:
			winners++
		case errors.Is(claimErr, ErrCharacterOwned):
		default:
			t.Fatalf("unexpected claim error: %v", claimErr)
		}
	}
	if winners != 1 {
		t.Fatalf("successful claims=%d, want exactly one", winners)
	}
}

func TestManagerLoadsStaticCollisionAndSafeSpawn(t *testing.T) {
	key := sim.ZoneKey{MapID: "arena", InstanceID: "preview", RulesetVersion: "v1"}
	solid := make([]bool, 10*10)
	for i := 0; i < 10; i++ {
		solid[i] = true
		solid[9*10+i] = true
		solid[i*10] = true
		solid[i*10+9] = true
	}
	collision, err := sim.NewGridCollision(10, 10, solid)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManagerWithMaps([]sim.ZoneKey{key}, 8, 2, map[sim.ZoneKey]ZoneMap{
		key: {Collision: collision, Spawns: []sim.Vec2{{X: 5.5, Y: 5.5}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	spawn, ok := manager.SafeSpawn(key, sim.Vec2{X: 0.5, Y: 0.5}, 0.4)
	if !ok || spawn != (sim.Vec2{X: 5.5, Y: 5.5}) {
		t.Fatalf("blocked preferred spawn did not fall back: %+v ok=%v", spawn, ok)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdown, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		_ = manager.Stop(shutdown)
	})
	characterID := sim.CharacterID("11111111-1111-1111-1111-111111111111")
	lease, err := manager.JoinCharacter(characterID, "controller", key, sim.Entity{
		ID: sim.EntityID(characterID), CharacterID: characterID, Kind: sim.EntityPlayer,
		Transform: sim.Transform{Position: spawn}, Vitals: sim.Vitals{HP: 100, MaxHP: 100}, Speed: 4, Radius: 0.4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.SubmitMoveTarget(lease, sim.InputCommand{EntityID: sim.EntityID(characterID), Seq: 1, Target: &sim.Vec2{X: 1.6, Y: 5.5}}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := manager.Snapshot(key)
		if err != nil {
			t.Fatal(err)
		}
		if len(snapshot.Entities) == 1 && snapshot.Entities[0].Transform.Position.X <= 1.75 {
			if snapshot.Entities[0].Transform.Position.X < 1.39 {
				t.Fatalf("actor crossed the solid wall: %+v", snapshot.Entities[0].Transform.Position)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	snapshot, _ := manager.Snapshot(key)
	t.Fatalf("actor never reached the wall boundary: %+v", snapshot.Entities)
}

func TestLegacyCoordinatesConvertAndRejectStaleOutOfBoundsCheckpoint(t *testing.T) {
	key := sim.ZoneKey{MapID: "world_map", InstanceID: "preview", RulesetVersion: "v1"}
	manager, err := NewManagerWithMaps([]sim.ZoneKey{key}, 8, 2, map[sim.ZoneKey]ZoneMap{
		key: {Width: 1200, Height: 800, LegacyOriginX: 0, LegacyOriginY: 0, LegacyScale: 25},
	})
	if err != nil {
		t.Fatal(err)
	}
	position := manager.LegacyPosition(key, 4012, 5012)
	if position != (sim.Vec2{X: 160.48, Y: 200.48}) {
		t.Fatalf("legacy coordinate conversion=%+v", position)
	}
	if !manager.ContainsPosition(key, position) {
		t.Fatal("converted legacy position should be inside the MMO map")
	}
	if manager.ContainsPosition(key, sim.Vec2{X: 4012, Y: 5012}) {
		t.Fatal("stale legacy-scale checkpoint must not be treated as an MMO coordinate")
	}
}

func TestManagerSpawnsServerOwnedMonstersWithAI(t *testing.T) {
	key := sim.ZoneKey{MapID: "arena", InstanceID: "monsters", RulesetVersion: "v1"}
	solid := make([]bool, 10*10)
	for i := 0; i < 10; i++ {
		solid[i], solid[9*10+i], solid[i*10], solid[i*10+9] = true, true, true, true
	}
	collision, err := sim.NewGridCollision(10, 10, solid)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManagerWithMaps([]sim.ZoneKey{key}, 8, 2, map[sim.ZoneKey]ZoneMap{
		key: {
			Collision: collision,
			Monsters: []MonsterSpawn{{
				TemplateID: "monster_stone_ape", X: 5.5, Y: 5.5, MaxHP: 50, AttackDamage: 5,
				Speed: 3, Radius: 0.5, AggroRadius: 6, LeashRadius: 10, AttackRange: 1.2,
				PatrolRadius: 0, AttackCooldownTicks: 2, RespawnTicks: 100,
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdown, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		_ = manager.Stop(shutdown)
	})
	characterID := sim.CharacterID("22222222-2222-2222-2222-222222222222")
	if _, err := manager.JoinCharacter(characterID, "controller", key, sim.Entity{
		ID: sim.EntityID(characterID), CharacterID: characterID, Kind: sim.EntityPlayer,
		Transform: sim.Transform{Position: sim.Vec2{X: 6.5, Y: 5.5}},
		Vitals:    sim.Vitals{HP: 100, MaxHP: 100}, Speed: 4, Radius: 0.4,
	}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := manager.Snapshot(key)
		if err != nil {
			t.Fatal(err)
		}
		var monster, player *sim.Entity
		for index := range snapshot.Entities {
			switch snapshot.Entities[index].Kind {
			case sim.EntityMonster:
				monster = &snapshot.Entities[index]
			case sim.EntityPlayer:
				player = &snapshot.Entities[index]
			}
		}
		if monster == nil || monster.AI.TemplateID != "monster_stone_ape" {
			t.Fatalf("monster was not spawned with AI: %+v", monster)
		}
		if player != nil && player.Vitals.HP < 100 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("monster AI never damaged the joined player")
}

func TestManagerQueuesMonsterDeathForLootHandler(t *testing.T) {
	key := sim.ZoneKey{MapID: "arena", InstanceID: "loot", RulesetVersion: "v1"}
	manager, err := NewManagerWithMaps([]sim.ZoneKey{key}, 8, 2, map[sim.ZoneKey]ZoneMap{
		key: {Monsters: []MonsterSpawn{{
			TemplateID: "test_mob", X: 3, Y: 0, MaxHP: 1, AttackDamage: 1,
			Speed: 1, Radius: 0.5, AggroRadius: 1, LeashRadius: 2, AttackRange: 0.5,
			PatrolRadius: 0, AttackCooldownTicks: 20, RespawnTicks: 100,
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	deaths := make(chan DeathEvent, 1)
	manager.SetDeathHandler(func(_ context.Context, death DeathEvent) { deaths <- death })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdown, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		_ = manager.Stop(shutdown)
	})
	characterID := sim.CharacterID("33333333-3333-3333-3333-333333333333")
	lease, err := manager.JoinCharacter(characterID, "controller", key, sim.Entity{
		ID: sim.EntityID(characterID), CharacterID: characterID, Kind: sim.EntityPlayer,
		Transform: sim.Transform{Position: sim.Vec2{X: 0, Y: 0}},
		Vitals:    sim.Vitals{HP: 100, MaxHP: 100}, Speed: 4, Radius: 0.4,
		Combat: sim.NewPlayerCombat(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.SubmitCast(lease, sim.InputCommand{EntityID: sim.EntityID(characterID), Seq: 1, Cast: &sim.CastCommand{
		EntityID: sim.EntityID(characterID), Seq: 1, Slot: 1, Mode: "TARGET", TargetID: "m:0",
	}}); err != nil {
		t.Fatal(err)
	}
	select {
	case death := <-deaths:
		if death.TemplateID != "test_mob" || death.SourceID != sim.EntityID(characterID) || death.TargetID != "m:0" {
			t.Fatalf("unexpected death event: %+v", death)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("loot death handler was not called")
	}
}

func TestManagerTransfersCharacterWithSingleOwner(t *testing.T) {
	firstKey := sim.ZoneKey{MapID: "map-a", InstanceID: "transfer", RulesetVersion: "v1"}
	secondKey := sim.ZoneKey{MapID: "map-b", InstanceID: "transfer", RulesetVersion: "v1"}
	manager, err := NewManagerWithMaps([]sim.ZoneKey{firstKey, secondKey}, 8, 2, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdown, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		_ = manager.Stop(shutdown)
	})
	characterID := sim.CharacterID("44444444-4444-4444-4444-444444444444")
	lease, err := manager.JoinCharacter(characterID, "controller", firstKey, sim.Entity{
		ID: sim.EntityID(characterID), CharacterID: characterID, Kind: sim.EntityPlayer,
		Transform: sim.Transform{Position: sim.Vec2{X: 1, Y: 1}},
		Vitals:    sim.Vitals{HP: 100, MaxHP: 100}, Speed: 4, Radius: 0.4,
		Combat: sim.NewPlayerCombat(),
	})
	if err != nil {
		t.Fatal(err)
	}
	transferred, err := manager.TransferCharacter(lease, secondKey, sim.Vec2{X: 5, Y: 5})
	if err != nil {
		t.Fatal(err)
	}
	if transferred.ZoneKey != secondKey || transferred.Token != lease.Token {
		t.Fatalf("unexpected transferred lease: %+v", transferred)
	}
	firstSnapshot, _ := manager.Snapshot(firstKey)
	secondSnapshot, _ := manager.Snapshot(secondKey)
	if len(firstSnapshot.Entities) != 0 || len(secondSnapshot.Entities) != 1 {
		t.Fatalf("entity duplicated across zones: first=%d second=%d", len(firstSnapshot.Entities), len(secondSnapshot.Entities))
	}
	entity := secondSnapshot.Entities[0]
	if entity.Transform.Position != (sim.Vec2{X: 5, Y: 5}) || entity.CharacterID != characterID {
		t.Fatalf("transfer did not move entity: %+v", entity)
	}
	if err := manager.SubmitInput(lease, sim.InputCommand{EntityID: sim.EntityID(characterID), Seq: 1, Move: sim.Vec2{X: 1}}); !errors.Is(err, ErrInvalidLease) {
		t.Fatalf("source lease still accepted input: %v", err)
	}
	if err := manager.SubmitInput(transferred, sim.InputCommand{EntityID: sim.EntityID(characterID), Seq: 2, Move: sim.Vec2{X: 1}}); err != nil {
		t.Fatalf("target lease rejected input: %v", err)
	}
	if _, err := manager.TransferCharacter(transferred, sim.ZoneKey{MapID: "missing", InstanceID: "transfer", RulesetVersion: "v1"}, sim.Vec2{}); !errors.Is(err, ErrUnknownZone) {
		t.Fatalf("unknown target error: %v", err)
	}
}

func TestManagerAllocatesMapInstancesByCapacity(t *testing.T) {
	first := sim.ZoneKey{MapID: "map", InstanceID: "a", RulesetVersion: "v1"}
	second := sim.ZoneKey{MapID: "map", InstanceID: "b", RulesetVersion: "v1"}
	manager, err := NewManagerWithCapacity([]sim.ZoneKey{first, second}, 8, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdown, done := context.WithTimeout(context.Background(), time.Second)
		defer done()
		_ = manager.Stop(shutdown)
	})
	firstKey, err := manager.ZoneKeyForMapWithCapacity("map")
	if err != nil {
		t.Fatal(err)
	}
	playerOne := sim.CharacterID("55555555-5555-5555-5555-555555555555")
	if _, err := manager.JoinCharacter(playerOne, "controller-1", firstKey, sim.Entity{
		ID: sim.EntityID(playerOne), CharacterID: playerOne, Kind: sim.EntityPlayer,
		Transform: sim.Transform{Position: sim.Vec2{X: 1, Y: 1}},
		Vitals:    sim.Vitals{HP: 100, MaxHP: 100}, Speed: 4, Radius: 0.4,
		Combat: sim.NewPlayerCombat(),
	}); err != nil {
		t.Fatal(err)
	}
	secondKey, err := manager.ZoneKeyForMapWithCapacity("map")
	if err != nil {
		t.Fatal(err)
	}
	if secondKey == firstKey {
		t.Fatalf("allocator reused a full instance: %s", secondKey.String())
	}
	playerTwo := sim.CharacterID("66666666-6666-6666-6666-666666666666")
	if _, err := manager.JoinCharacter(playerTwo, "controller-2", secondKey, sim.Entity{
		ID: sim.EntityID(playerTwo), CharacterID: playerTwo, Kind: sim.EntityPlayer,
		Transform: sim.Transform{Position: sim.Vec2{X: 2, Y: 2}},
		Vitals:    sim.Vitals{HP: 100, MaxHP: 100}, Speed: 4, Radius: 0.4,
		Combat: sim.NewPlayerCombat(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ZoneKeyForMapWithCapacity("map"); !errors.Is(err, ErrZoneFull) {
		t.Fatalf("full instances error=%v, want ErrZoneFull", err)
	}
	if _, err := manager.ZoneKeyForMapWithCapacity("missing"); !errors.Is(err, ErrUnknownZone) {
		t.Fatalf("unknown map error=%v, want ErrUnknownZone", err)
	}
}
