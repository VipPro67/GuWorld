package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/realtime/config"
	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
	"github.com/vippro67/GuWorld/backend/internal/realtime/persistence"
	"github.com/vippro67/GuWorld/backend/internal/realtime/protocol"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
)

func TestAdmissionJoinUsesServerStateAndLeaveReleasesMembership(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	otherPlayerID := uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 2)
	if _, err := manager.JoinCharacter(sim.CharacterID(otherPlayerID.String()), uuid.NewString(), key, sim.Entity{
		ID: sim.EntityID(otherPlayerID.String()), CharacterID: sim.CharacterID(otherPlayerID.String()), Kind: sim.EntityPlayer,
		Transform: sim.Transform{Position: sim.Vec2{X: 500, Y: 500}}, Vitals: sim.Vitals{HP: 70, MaxHP: 100}, Speed: 4, Radius: 0.4,
	}); err != nil {
		t.Fatal(err)
	}
	loaderCalls := 0
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		loaderCalls++
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest-clearing", X: 17, Y: 29, HP: 70, BaseHP: 100, Rank: 3, CurrentMana: 42, MaxMana: 500, BaseAttack: 36, Defense: 7, MovementSpeed: 5.6}, nil
	})

	request := protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}
	result, err := admission.Join(context.Background(), playerID, controllerID, request)
	if err != nil {
		t.Fatal(err)
	}
	welcome, ok := result.(protocol.Welcome)
	if !ok || welcome.ZoneID != "forest#realtime-preview" || welcome.MapVersion != "map-1" {
		t.Fatalf("unexpected welcome response: %#v", result)
	}
	if welcome.MovementSpeed != 5.6 {
		t.Fatalf("welcome movement speed = %v, want 5.6", welcome.MovementSpeed)
	}
	var initialSnapshot protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &initialSnapshot); err != nil {
		t.Fatal(err)
	}
	if loaderCalls != 1 || len(initialSnapshot.Entities) != 1 {
		t.Fatalf("join snapshot or loader count mismatch: calls=%d entities=%d", loaderCalls, len(initialSnapshot.Entities))
	}
	entity := initialSnapshot.Entities[0]
	if entity.EntityID != playerID.String() || entity.X != 17 || entity.Y != 29 || entity.HP != 70 || !initialSnapshot.IsBaseline || initialSnapshot.BaselineID != welcome.BaselineID {
		t.Fatalf("entity was not built from trusted database state: %+v", entity)
	}
	if entity.Mana != 42 || entity.MaxMana != 500 || len(entity.Cooldowns) != 13 {
		t.Fatalf("welcome combat state is invalid: %+v", entity)
	}
	if _, err := admission.Join(context.Background(), playerID, controllerID, request); err != nil || loaderCalls != 1 {
		t.Fatalf("same connection join should be idempotent: calls=%d err=%v", loaderCalls, err)
	}

	if err := admission.Leave(playerID, controllerID); err != nil {
		t.Fatal(err)
	}
	if manager.OwnsCharacter(sim.CharacterID(playerID.String())) {
		t.Fatal("leave did not release ownership")
	}
	if snapshot, _ := manager.Snapshot(key); len(snapshot.Entities) != 1 || snapshot.Entities[0].ID != sim.EntityID(otherPlayerID.String()) {
		t.Fatalf("leave did not remove only the leaving player entity: %+v", snapshot.Entities)
	}
}

func TestAdmissionSubmitsInputOnlyForActiveController(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 2)
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest-clearing", HP: 70, BaseHP: 100}, nil
	})
	request := protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}
	if err := admission.Input(playerID, controllerID, protocol.Input{Seq: 1, Move: &protocol.Vector{X: 1}}); admissionErrorCode(err) != "realtime_session_not_joined" {
		t.Fatalf("input before join error=%v", err)
	}
	if _, err := admission.Join(context.Background(), playerID, controllerID, request); err != nil {
		t.Fatal(err)
	}
	if err := admission.Input(playerID, uuid.New(), protocol.Input{Seq: 1, Move: &protocol.Vector{X: 1}}); admissionErrorCode(err) != "realtime_session_not_joined" {
		t.Fatalf("input from another controller error=%v", err)
	}
	if err := admission.Input(playerID, controllerID, protocol.Input{Seq: 1, Move: &protocol.Vector{X: 1}}); err != nil {
		t.Fatalf("active controller input failed: %v", err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := manager.Snapshot(key)
		if err == nil && len(snapshot.Entities) == 1 && snapshot.Entities[0].Transform.Position.X > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("authoritative zone did not apply movement input")
}

func TestAdmissionRejectsUntrustedOrIneligibleJoinState(t *testing.T) {
	for _, test := range []struct {
		name  string
		state handlers.RealtimeJoinState
		mapID string
		code  string
	}{
		{name: "map mismatch", state: handlers.RealtimeJoinState{MapID: "cave", RegionID: "cave", HP: 50, BaseHP: 50}, mapID: "forest", code: "realtime_map_mismatch"},
		{name: "legacy instance", state: handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", MapInstanceID: "legacy-instance", HP: 50, BaseHP: 50}, mapID: "forest", code: "realtime_instance_conflict"},
		{name: "onboarding", state: handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", HP: 50, BaseHP: 50, NeedsOnboarding: true}, mapID: "forest", code: "realtime_character_not_ready"},
	} {
		t.Run(test.name, func(t *testing.T) {
			playerID, controllerID := uuid.New(), uuid.New()
			key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
			manager := startTestManager(t, key, 2)
			admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
				return test.state, nil
			})
			request := protocol.JoinRequest{ZoneID: test.mapID, MapVersion: "map-1", ContentVersion: "content-1"}
			if _, err := admission.Join(context.Background(), playerID, controllerID, request); admissionErrorCode(err) != test.code {
				t.Fatalf("join error=%v code=%q want %q", err, admissionErrorCode(err), test.code)
			}
			if manager.OwnsCharacter(sim.CharacterID(playerID.String())) {
				t.Fatal("rejected join reserved ownership")
			}
		})
	}
}

func TestAdmissionAllowsEveryPlayerAndEnforcesManagerCapacity(t *testing.T) {
	playerID, otherPlayerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 1)
	loader := func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", HP: 50, BaseHP: 50}, nil
	}
	admission := NewAdmission(testConfig(playerID), manager, loader)
	request := protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}
	otherController := uuid.New()
	if _, err := admission.Join(context.Background(), otherPlayerID, otherController, request); err != nil {
		t.Fatalf("player outside the legacy allowlist should join: %v", err)
	}
	if _, err := admission.Join(context.Background(), playerID, uuid.New(), request); admissionErrorCode(err) != "realtime_zone_full" {
		t.Fatalf("second player should be limited by zone capacity: %v", err)
	}
	if _, err := admission.Join(context.Background(), otherPlayerID, uuid.New(), request); admissionErrorCode(err) != "realtime_character_already_owned" {
		t.Fatalf("duplicate controller join error: %v", err)
	}
}

func TestAdmissionRejectsStaleVersionsBeforeLoadingCharacter(t *testing.T) {
	playerID := uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 1)
	loaded := false
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		loaded = true
		return handlers.RealtimeJoinState{}, nil
	})
	request := protocol.JoinRequest{ZoneID: "forest", MapVersion: "old-map", ContentVersion: "content-1"}
	if _, err := admission.Join(context.Background(), playerID, uuid.New(), request); admissionErrorCode(err) != "realtime_version_mismatch" {
		t.Fatalf("stale version error: %v", err)
	}
	if loaded {
		t.Fatal("stale version should be rejected before loading character state")
	}
}

func TestAdmissionRequiresRunningManager(t *testing.T) {
	playerID := uuid.New()
	managerUnavailable := NewAdmission(testConfig(playerID), nil, nil)
	if _, err := managerUnavailable.Join(context.Background(), playerID, uuid.New(), protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}); admissionErrorCode(err) != "realtime_manager_unavailable" {
		t.Fatalf("nil manager join error: %v", err)
	}
}

func TestAdmissionRateLimitsRepeatedFailedJoinAttempts(t *testing.T) {
	playerID := uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 2)
	loadCalls := 0
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		loadCalls++
		return handlers.RealtimeJoinState{}, errors.New("database unavailable")
	})
	request := protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}
	for attempt := 0; attempt < joinLimitPerMinute; attempt++ {
		if _, err := admission.Join(context.Background(), playerID, uuid.New(), request); admissionErrorCode(err) != "realtime_character_unavailable" {
			t.Fatalf("attempt %d error=%v, want character unavailable", attempt+1, err)
		}
		if manager.OwnsCharacter(sim.CharacterID(playerID.String())) {
			t.Fatalf("failed attempt %d retained character ownership", attempt+1)
		}
	}
	if _, err := admission.Join(context.Background(), playerID, uuid.New(), request); admissionErrorCode(err) != "realtime_join_rate_limited" {
		t.Fatalf("attempt after limit error=%v, want rate limited", err)
	}
	if loadCalls != joinLimitPerMinute {
		t.Fatalf("authoritative state loaded %d times, want %d", loadCalls, joinLimitPerMinute)
	}
}

func TestJoinRateLimitExpiresAfterOneMinute(t *testing.T) {
	admission := NewAdmission(config.Config{}, nil, nil)
	playerID := uuid.New()
	now := time.Unix(1_800_000_000, 0)
	for attempt := 0; attempt < joinLimitPerMinute; attempt++ {
		if !admission.allowJoinAttempt(playerID, now) {
			t.Fatalf("attempt %d was unexpectedly limited", attempt+1)
		}
	}
	if admission.allowJoinAttempt(playerID, now) {
		t.Fatal("attempt over the limit was allowed")
	}
	if !admission.allowJoinAttempt(playerID, now.Add(time.Minute+time.Nanosecond)) {
		t.Fatal("attempt was still limited after the rolling minute expired")
	}
}

func startTestManager(t *testing.T, key sim.ZoneKey, capacity int) *realtimeworld.Manager {
	t.Helper()
	// Exercise the production lazy lifecycle: the world is created by the
	// claim/attach path rather than eagerly at manager construction.
	manager, err := realtimeworld.NewLazyManagerWithMaps([]sim.ZoneKey{key}, 8, capacity, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := manager.Stop(ctx); err != nil {
			t.Errorf("stop manager: %v", err)
		}
	})
	return manager
}

func testConfig(playerID uuid.UUID) config.Config {
	_ = playerID
	return config.Config{
		Enabled: true, PhaserClientEnabled: true, MaxPlayersPerZone: 2, MapVersion: "map-1", ContentVersion: "content-1",
		ZoneIDs: map[string]struct{}{"forest": {}},
	}
}

func admissionErrorCode(err error) string {
	var admissionErr *Error
	if errors.As(err, &admissionErr) {
		return admissionErr.Code
	}
	return ""
}

func TestAdmissionSubmitsBoundedClickToMoveForActiveController(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 2)
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest-clearing", X: 10, Y: 10, HP: 70, BaseHP: 100}, nil
	})
	if _, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}); err != nil {
		t.Fatal(err)
	}
	if err := admission.MoveTarget(playerID, uuid.New(), protocol.MoveTarget{Seq: 1, X: 11, Y: 10}); admissionErrorCode(err) != "realtime_session_not_joined" {
		t.Fatalf("move target from another controller error=%v", err)
	}
	if err := admission.MoveTarget(playerID, controllerID, protocol.MoveTarget{Seq: 1, X: 11, Y: 10}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := manager.Snapshot(key)
		if err == nil && len(snapshot.Entities) == 1 && snapshot.Entities[0].Transform.Position.X > 10 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("authoritative zone did not apply click-to-move target")
}

func TestAdmissionUsesSafeSpawnFromLoadedMap(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	solid := make([]bool, 10*10)
	for i := 0; i < 10; i++ {
		solid[i], solid[9*10+i], solid[i*10], solid[i*10+9] = true, true, true, true
	}
	collision, err := sim.NewGridCollision(10, 10, solid)
	if err != nil {
		t.Fatal(err)
	}
	manager, err := realtimeworld.NewManagerWithMaps([]sim.ZoneKey{key}, 8, 2, map[sim.ZoneKey]realtimeworld.ZoneMap{
		key: {Collision: collision, Spawns: []sim.Vec2{{X: 5.5, Y: 5.5}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = manager.Stop(ctx)
	})
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", X: 0, Y: 0, HP: 70, BaseHP: 100}, nil
	})
	result, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"})
	if err != nil {
		t.Fatal(err)
	}
	welcome, ok := result.(protocol.Welcome)
	if !ok {
		t.Fatalf("unexpected welcome: %#v", result)
	}
	var snapshot protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entities) != 1 || snapshot.Entities[0].X != 5.5 || snapshot.Entities[0].Y != 5.5 {
		t.Fatalf("blocked legacy position was not replaced by a safe spawn: %+v", snapshot.Entities)
	}
}

func TestAdmissionDoesNotInjectDefaultSkillsWithoutAnOwnedRealtimeGu(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager, err := realtimeworld.NewManagerWithMaps([]sim.ZoneKey{key}, 8, 2, map[sim.ZoneKey]realtimeworld.ZoneMap{
		key: {Monsters: []realtimeworld.MonsterSpawn{{
			TemplateID: "test_mob", Name: "Sói thử nghiệm", X: 2, Y: 0, MaxHP: 100, AttackDamage: 1,
			Speed: 1, Radius: 0.5, AggroRadius: 1, LeashRadius: 2, AttackRange: 0.5,
			PatrolRadius: 0, AttackCooldownTicks: 20, RespawnTicks: 100,
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = manager.Stop(ctx)
	})
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", X: 0, Y: 0, HP: 100, BaseHP: 100}, nil
	})
	result, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"})
	if err != nil {
		t.Fatal(err)
	}
	welcome := result.(protocol.Welcome)
	var welcomeSnapshot protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &welcomeSnapshot); err != nil {
		t.Fatal(err)
	}
	foundMonster := false
	for _, entity := range welcomeSnapshot.Entities {
		if entity.Kind == string(sim.EntityMonster) {
			foundMonster = true
			if entity.TemplateID != "test_mob" || entity.Name != "Sói thử nghiệm" || entity.AIState == "" {
				t.Fatalf("monster presentation missing from welcome: %+v", entity)
			}
		}
	}
	if !foundMonster {
		t.Fatal("welcome did not include nearby monster")
	}
	if len(welcome.Skills) != 0 {
		t.Fatalf("unassigned loadout got preview skills: %+v", welcome.Skills)
	}
	if err := admission.Cast(playerID, controllerID, protocol.Cast{Seq: 1, Slot: 1, Mode: "TARGET", TargetID: "m:0"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(150 * time.Millisecond)
	for time.Now().Before(deadline) {
		snapshot, err := manager.Snapshot(key)
		if err != nil {
			t.Fatal(err)
		}
		for _, entity := range snapshot.Entities {
			if entity.Kind == sim.EntityMonster && entity.Vitals.HP != 100 {
				t.Fatalf("cast used a default skill without an assigned Gu: hp=%v", entity.Vitals.HP)
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestAdmissionBuildsOwnedGuLoadoutAndCastsRealtimeSkill(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager, err := realtimeworld.NewManagerWithMaps([]sim.ZoneKey{key}, 8, 2, map[sim.ZoneKey]realtimeworld.ZoneMap{
		key: {Monsters: []realtimeworld.MonsterSpawn{{
			TemplateID: "test_mob", X: 3, Y: 0, MaxHP: 200, AttackDamage: 1,
			Speed: 1, Radius: 0.5, AggroRadius: 1, LeashRadius: 2, AttackRange: 0.5,
			PatrolRadius: 0, AttackCooldownTicks: 20, RespawnTicks: 100,
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = manager.Stop(ctx)
	})
	profiles, err := realtimeeffects.LoadProfiles("../../../data/mmo/gu/transpiled_profiles.json")
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := realtimeeffects.CompileCatalog(profiles, nil, realtimeeffects.DefaultMVP())
	if err != nil {
		t.Fatal(err)
	}
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{
			MapID: "forest", RegionID: "forest", X: 0, Y: 0, HP: 100, BaseHP: 100, Rank: 3, CurrentMana: 20, MaxMana: 80, BaseAttack: 30,
			EquippedGu: []handlers.RealtimeEquippedGu{
				{InstanceID: "inst-frost", GuID: "GU_WHITE_FROST", Rank: 3, Hunger: 10, Usable: true, ManaCost: 2.5, DamageScale: 1.5},
				{InstanceID: "inst-arrow", GuID: "GU_WATER_ARROW", Rank: 1, Hunger: 10, Usable: true, ManaCost: 0.25, DamageScale: 2},
			},
		}, nil
	})
	admission.SetLoadoutCatalog(compiled)
	store := &fakeStateStore{}
	admission.SetStateStore(store)
	result, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"})
	if err != nil {
		t.Fatal(err)
	}
	welcome, ok := result.(protocol.Welcome)
	if !ok || len(welcome.Skills) != 2 {
		t.Fatalf("unexpected welcome skills: %#v", result)
	}
	if welcome.Skills[0].ID != "GU_WHITE_FROST" || welcome.Skills[0].Slot != 1 ||
		welcome.Skills[1].ID != "GU_WATER_ARROW" || welcome.Skills[1].Slot != 2 {
		t.Fatalf("owned Gu loadout was not re-slotted: %+v", welcome.Skills)
	}
	if welcome.Skills[0].ManaCost != 2.5 || welcome.Skills[1].ManaCost != 0.25 {
		t.Fatalf("welcome costs differ from server loadout: %+v", welcome.Skills)
	}
	if welcome.LoadoutRevision == "" {
		t.Fatal("welcome is missing the loadout revision")
	}
	if err := admission.Cast(playerID, controllerID, protocol.Cast{Seq: 1, Slot: 1, Mode: "TARGET", TargetID: "m:0"}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snapshot, err := manager.Snapshot(key)
		if err != nil {
			t.Fatal(err)
		}
		for _, entity := range snapshot.Entities {
			if entity.Kind == sim.EntityMonster {
				for _, status := range entity.Combat.Statuses {
					if status.ID == "FREEZE" {
						admission.CheckpointActive()
						admission.CheckpointActive()
						if len(store.guUseDeltas) != 2 || store.guUseDeltas[0]["inst-frost"] != 1 || len(store.guUseDeltas[1]) != 0 {
							t.Fatalf("Gu usage was not settled once: %+v", store.guUseDeltas)
						}
						if len(store.saved) == 0 || store.saved[len(store.saved)-1].Combat == nil {
							t.Fatalf("checkpoint did not persist combat state: %+v", store.saved)
						}
						combat := store.saved[len(store.saved)-1].Combat
						if combat.CooldownRemainingTicks[1] == 0 || combat.CooldownSkillIDs[1] != "GU_WHITE_FROST" {
							t.Fatalf("checkpoint did not persist the hotbar cooldown: %+v", combat)
						}
						return
					}
				}
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("owned Gu slot 1 did not apply FREEZE")
}

type fakeStateStore struct {
	state       persistence.State
	ok          bool
	saved       []persistence.State
	manaDeltas  []float64
	guUseDeltas []map[string]uint64
	loadErr     error
	settleErr   error
}

type revisionStateStore struct {
	state    persistence.State
	revision int64
	saved    []persistence.State
}

func (f *revisionStateStore) Load(context.Context, uuid.UUID) (persistence.State, bool, error) {
	return f.state, true, nil
}

func (f *revisionStateStore) Save(ctx context.Context, state persistence.State) error {
	_, err := f.SaveRevision(ctx, state)
	return err
}

func (f *revisionStateStore) SaveRevision(_ context.Context, state persistence.State) (int64, error) {
	if state.LocationRevision != f.revision {
		return 0, persistence.ErrInvalidState
	}
	f.revision++
	f.saved = append(f.saved, state)
	return f.revision, nil
}

func (f *revisionStateStore) SaveAndApplyManaDeltaRevision(ctx context.Context, state persistence.State, _ float64) (int64, error) {
	return f.SaveRevision(ctx, state)
}

func TestLateCheckpointCannotOverwriteNewerPlayerLocation(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 1)
	store := &revisionStateStore{
		revision: 8,
		state: persistence.State{
			PlayerID: playerID, RegionID: "forest", ZoneID: "forest", InstanceID: "old-instance",
			X: 4, Y: 5, HP: 70, SafeX: 4, SafeY: 5,
			MapVersion: "map-1", ContentVersion: "content-1", LocationRevision: 8, HasLocationRevision: true,
		},
	}
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", HP: 70, BaseHP: 100, HasWorldLocationRevision: true, WorldLocationRevision: 8}, nil
	})
	admission.SetStateStore(store)
	if _, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}); err != nil {
		t.Fatal(err)
	}
	// A region transfer or a legacy movement commits after the session snapshot.
	store.revision++
	if err := admission.Leave(playerID, controllerID); admissionErrorCode(err) != "realtime_checkpoint_unavailable" {
		t.Fatalf("late checkpoint error = %v, want stale checkpoint rejection", err)
	}
	if len(store.saved) != 0 {
		t.Fatalf("stale checkpoint wrote location: %+v", store.saved)
	}
}

func (f *fakeStateStore) Load(context.Context, uuid.UUID) (persistence.State, bool, error) {
	return f.state, f.ok, f.loadErr
}

func (f *fakeStateStore) Save(_ context.Context, state persistence.State) error {
	f.saved = append(f.saved, state)
	return nil
}

func (f *fakeStateStore) SaveAndApplyManaDelta(ctx context.Context, state persistence.State, delta float64) error {
	if f.settleErr != nil {
		return f.settleErr
	}
	if err := f.Save(ctx, state); err != nil {
		return err
	}
	f.manaDeltas = append(f.manaDeltas, delta)
	return nil
}

func (f *fakeStateStore) SaveAndApplyCombatDeltaRevision(ctx context.Context, state persistence.State, delta float64, uses, killerMoveUses map[string]uint64) (int64, error) {
	if err := f.SaveAndApplyManaDelta(ctx, state, delta); err != nil {
		return 0, err
	}
	copyUses := make(map[string]uint64, len(uses))
	for id, count := range uses {
		copyUses[id] = count
	}
	f.guUseDeltas = append(f.guUseDeltas, copyUses)
	return 0, nil
}

func TestAdmissionCheckpointSettlesManaOnceAndRetriesFailure(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 1)
	store := &fakeStateStore{}
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", HP: 100, BaseHP: 100, Rank: 2, CurrentMana: 42, MaxMana: 50, BaseAttack: 20}, nil
	})
	admission.SetStateStore(store)
	if _, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}); err != nil {
		t.Fatal(err)
	}
	active := admission.sessions[controllerID]
	active.mana.last = 50 // Simulate eight mana spent since the last committed checkpoint.
	store.settleErr = errors.New("database unavailable")
	if err := admission.Leave(playerID, controllerID); admissionErrorCode(err) != "realtime_checkpoint_unavailable" {
		t.Fatalf("leave err=%v", err)
	}
	if !manager.OwnsCharacter(sim.CharacterID(playerID.String())) {
		t.Fatal("failed settlement released character")
	}
	store.settleErr = nil
	admission.CheckpointActive()
	admission.CheckpointActive()
	if err := admission.Leave(playerID, controllerID); err != nil {
		t.Fatal(err)
	}
	total := 0.0
	for _, delta := range store.manaDeltas {
		total += delta
	}
	if total != -8 {
		t.Fatalf("settled delta=%v, want -8: %+v", total, store.manaDeltas)
	}
}

func TestAdmissionResumesDurableCheckpointAndFlushesOnLeave(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 2)
	store := &fakeStateStore{ok: true, state: persistence.State{
		PlayerID: playerID, ZoneID: "forest", InstanceID: "realtime-preview",
		X: 42.5, Y: 31.5, HP: 88, Shield: 7, SafeX: 42.5, SafeY: 31.5,
		MapVersion: "map-1", ContentVersion: "content-1",
	}}
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", X: 1, Y: 2, HP: 70, BaseHP: 100}, nil
	})
	admission.SetStateStore(store)
	result, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"})
	if err != nil {
		t.Fatal(err)
	}
	welcome := result.(protocol.Welcome)
	var snapshot protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	entity := snapshot.Entities[0]
	if entity.X != 42.5 || entity.Y != 31.5 || entity.HP != 88 || entity.Shield != 7 {
		t.Fatalf("durable checkpoint was not resumed: %+v", entity)
	}
	if err := admission.Leave(playerID, controllerID); err != nil {
		t.Fatal(err)
	}
	if len(store.saved) != 1 || store.saved[0].HP != 88 || store.saved[0].ZoneID != "forest" || store.saved[0].MapVersion != "map-1" {
		t.Fatalf("leave did not flush durable state: %+v", store.saved)
	}
}

func TestAdmissionIgnoresCheckpointForDifferentVersion(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 2)
	store := &fakeStateStore{ok: true, state: persistence.State{
		PlayerID: playerID, ZoneID: "forest", InstanceID: "realtime-preview",
		X: 42.5, Y: 31.5, HP: 88, SafeX: 42.5, SafeY: 31.5,
		MapVersion: "old-map", ContentVersion: "content-1",
	}}
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", X: 1, Y: 2, HP: 70, BaseHP: 100}, nil
	})
	admission.SetStateStore(store)
	result, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"})
	if err != nil {
		t.Fatal(err)
	}
	welcome := result.(protocol.Welcome)
	var snapshot protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Entities[0].X != 1 || snapshot.Entities[0].Y != 2 || snapshot.Entities[0].HP != 70 {
		t.Fatalf("stale checkpoint was applied: %+v", snapshot.Entities[0])
	}
}

func TestAdmissionPauseAndDrainForRollback(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 2)
	store := &fakeStateStore{}
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", X: 1, Y: 1, HP: 70, BaseHP: 100}, nil
	})
	admission.SetStateStore(store)
	if _, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}); err != nil {
		t.Fatal(err)
	}
	admission.SetAdmissionPaused(true)
	if _, err := admission.Join(context.Background(), uuid.New(), uuid.New(), protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}); admissionErrorCode(err) == "" {
		t.Fatal("join while paused should fail")
	}
	if drained := admission.Drain(); drained != 1 {
		t.Fatalf("drained=%d, want 1", drained)
	}
	if manager.OwnsCharacter(sim.CharacterID(playerID.String())) {
		t.Fatal("drain did not release ownership")
	}
	if len(store.saved) != 1 {
		t.Fatalf("drain did not checkpoint session: %+v", store.saved)
	}
}

func TestAdmissionRestoresDurableCombatStateOnJoin(t *testing.T) {
	playerID, controllerID := uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 1)
	profiles, err := realtimeeffects.LoadProfiles("../../../data/mmo/gu/transpiled_profiles.json")
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := realtimeeffects.CompileCatalog(profiles, nil, realtimeeffects.DefaultMVP())
	if err != nil {
		t.Fatal(err)
	}
	store := &fakeStateStore{ok: true, state: persistence.State{
		PlayerID: playerID, RegionID: "forest", ZoneID: "forest", InstanceID: key.InstanceID,
		X: 17, Y: 29, HP: 90, SafeX: 17, SafeY: 29,
		MapVersion: "map-1", ContentVersion: "content-1",
		Combat: &persistence.CombatState{
			Version: persistence.CombatStateVersion, GCDRemainingTicks: 5,
			CooldownRemainingTicks: [13]uint64{0, 100}, CooldownSkillIDs: [13]string{"", "GU_WHITE_FROST"},
			RecoveryRemainingTicks: 4, RecoveryPolicy: "SLOW",
			Statuses: []persistence.StatusState{{
				ID: "FREEZE", RemainingTicks: 30, DurationTicks: 30, MoveMultiplier: 0,
				Debuff: true, PvPDrCategory: "HARD_CC",
			}},
		},
	}}
	admission := NewAdmission(testConfig(playerID), manager, func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{
			MapID: "forest", RegionID: "forest", X: 10, Y: 10, HP: 100, BaseHP: 100, Rank: 3, CurrentMana: 20, MaxMana: 80, BaseAttack: 30,
			EquippedGu: []handlers.RealtimeEquippedGu{
				{InstanceID: "inst-frost", GuID: "GU_WHITE_FROST", Rank: 3, Hunger: 10, Usable: true, ManaCost: 2.5, DamageScale: 1.5},
			},
		}, nil
	})
	admission.SetLoadoutCatalog(compiled)
	admission.SetStateStore(store)
	result, err := admission.Join(context.Background(), playerID, controllerID, protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"})
	if err != nil {
		t.Fatal(err)
	}
	welcome, ok := result.(protocol.Welcome)
	if !ok {
		t.Fatalf("unexpected join result: %#v", result)
	}
	var snapshot protocol.Snapshot
	if err := json.Unmarshal(welcome.Snapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entities) != 1 {
		t.Fatalf("join snapshot entities=%d, want 1", len(snapshot.Entities))
	}
	entity := snapshot.Entities[0]
	if len(entity.Cooldowns) != 13 || entity.Cooldowns[1] == 0 {
		t.Fatalf("durable cooldown was not restored: %+v", entity.Cooldowns)
	}
	foundFreeze := false
	for _, status := range entity.Statuses {
		if status == "FREEZE" {
			foundFreeze = true
		}
	}
	if !foundFreeze {
		t.Fatalf("durable status was not restored: %+v", entity.Statuses)
	}
}

func TestAdmissionRejectsControllerReuseAcrossCharacters(t *testing.T) {
	playerA, playerB, controllerID := uuid.New(), uuid.New(), uuid.New()
	key := sim.ZoneKey{MapID: "forest", InstanceID: "realtime-preview", RulesetVersion: "v1"}
	manager := startTestManager(t, key, 2)
	cfg := testConfig(playerA)
	loader := func(context.Context, uuid.UUID) (handlers.RealtimeJoinState, error) {
		return handlers.RealtimeJoinState{MapID: "forest", RegionID: "forest", HP: 70, BaseHP: 100}, nil
	}
	admission := NewAdmission(cfg, manager, loader)
	request := protocol.JoinRequest{ZoneID: "forest", MapVersion: "map-1", ContentVersion: "content-1"}
	if _, err := admission.Join(context.Background(), playerA, controllerID, request); err != nil {
		t.Fatal(err)
	}
	if _, err := admission.Join(context.Background(), playerB, controllerID, request); admissionErrorCode(err) != "realtime_controller_mismatch" {
		t.Fatalf("controller reuse error=%v code=%q", err, admissionErrorCode(err))
	}
}
