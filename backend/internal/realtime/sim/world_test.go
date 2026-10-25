package sim

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"runtime"
	"sync"
	"testing"
)

func newPlayer(id string) Entity {
	return Entity{ID: EntityID(id), Kind: EntityPlayer, Vitals: Vitals{HP: 100, MaxHP: 100}, Speed: 4, Radius: 0.4}
}

func TestAddEntitiesMatchesIndividualSetup(t *testing.T) {
	individual, _ := NewWorldForKeyWithSeed(ZoneKey{MapID: "demo", InstanceID: "default", RulesetVersion: "v1"}, 8, 1)
	batched, _ := NewWorldForKeyWithSeed(ZoneKey{MapID: "demo", InstanceID: "default", RulesetVersion: "v1"}, 8, 1)
	entities := []Entity{newPlayer("b"), newPlayer("a")}
	for _, entity := range entities {
		if err := individual.AddEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	if err := batched.AddEntities(entities); err != nil {
		t.Fatal(err)
	}
	if got, want := batched.Snapshot(), individual.Snapshot(); !reflect.DeepEqual(got, want) {
		t.Fatalf("batch snapshot differs from individual setup: got %+v, want %+v", got, want)
	}
	if batched.activePlayers != individual.activePlayers {
		t.Fatalf("active players: got %d, want %d", batched.activePlayers, individual.activePlayers)
	}
}

func TestAddEntitiesRejectsWholeInvalidBatch(t *testing.T) {
	for _, test := range []struct {
		name     string
		entities []Entity
	}{
		{name: "duplicate within batch", entities: []Entity{newPlayer("a"), newPlayer("a")}},
		{name: "invalid entity", entities: []Entity{newPlayer("a"), Entity{ID: "invalid"}}},
		{name: "duplicate existing", entities: []Entity{newPlayer("a"), newPlayer("existing")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, _ := NewWorld("demo", 8)
			if err := w.AddEntity(newPlayer("existing")); err != nil {
				t.Fatal(err)
			}
			before := w.Snapshot()
			if err := w.AddEntities(test.entities); !errors.Is(err, ErrInvalidEntity) {
				t.Fatalf("expected invalid entity, got %v", err)
			}
			if got := w.Snapshot(); !reflect.DeepEqual(got, before) {
				t.Fatalf("failed batch changed snapshot: got %+v, want %+v", got, before)
			}
			if w.activePlayers != 1 {
				t.Fatalf("failed batch changed player count: %d", w.activePlayers)
			}
		})
	}
}

func TestAddEntitiesRejectsCollisionWithoutPartialInstall(t *testing.T) {
	grid, err := NewGridCollision(4, 4, []bool{
		false, false, false, false,
		false, false, false, false,
		false, false, true, false,
		false, false, false, false,
	})
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorldWithCollision("demo", 8, grid)
	if err != nil {
		t.Fatal(err)
	}
	first := newPlayer("first")
	first.Transform.Position = Vec2{X: 1.5, Y: 1.5}
	blocked := newPlayer("blocked")
	blocked.Transform.Position = Vec2{X: 2.5, Y: 2.5}
	if err := w.AddEntities([]Entity{first, blocked}); !errors.Is(err, ErrInvalidEntity) {
		t.Fatalf("expected collision failure, got %v", err)
	}
	if len(w.Snapshot().Entities) != 0 || w.activePlayers != 0 {
		t.Fatal("collision failure partially installed batch")
	}
}

func TestConcurrentSnapshotReads(t *testing.T) {
	w, _ := NewWorld("demo", 8)
	if err := w.AddEntity(newPlayer("a")); err != nil {
		t.Fatal(err)
	}
	var readers sync.WaitGroup
	for i := 0; i < 4; i++ {
		readers.Add(1)
		go func() {
			defer readers.Done()
			for n := 0; n < 200; n++ {
				_ = w.Snapshot()
			}
		}()
	}
	for n := 0; n < 200; n++ {
		w.Step()
	}
	readers.Wait()
}

func TestStructuredZoneKeySeparatesInstancesAndRulesets(t *testing.T) {
	base := ZoneKey{MapID: "forest", InstanceID: "alpha", RulesetVersion: "combat-v1"}
	a, err := NewWorldForKey(base, 8)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewWorldForKey(ZoneKey{MapID: "forest", InstanceID: "beta", RulesetVersion: "combat-v1"}, 8)
	if err != nil {
		t.Fatal(err)
	}
	c, err := NewWorldForKey(ZoneKey{MapID: "forest", InstanceID: "alpha", RulesetVersion: "combat-v2"}, 8)
	if err != nil {
		t.Fatal(err)
	}
	if a.Snapshot().ZoneKey == b.Snapshot().ZoneKey || a.Snapshot().ZoneKey == c.Snapshot().ZoneKey {
		t.Fatal("different zone instances or rulesets shared a zone key")
	}
	if got := a.Snapshot().ZoneKey.String(); got != `"forest"/"alpha"@"combat-v1"` {
		t.Fatalf("unexpected stable zone key: %q", got)
	}
	if (ZoneKey{MapID: "a/b", InstanceID: "c", RulesetVersion: "v"}).String() == (ZoneKey{MapID: "a", InstanceID: "b/c", RulesetVersion: "v"}).String() {
		t.Fatal("zone key string representation is ambiguous")
	}
	if _, err := NewWorldForKey(ZoneKey{MapID: "forest", InstanceID: " ", RulesetVersion: "combat-v1"}, 8); !errors.Is(err, ErrInvalidZoneKey) {
		t.Fatalf("invalid zone key: %v", err)
	}
}

func TestZoneRandomStreamIsSeededAndRepeatable(t *testing.T) {
	key := ZoneKey{MapID: "forest", InstanceID: "alpha", RulesetVersion: "combat-v1"}
	first, err := NewWorldForKeyWithSeed(key, 8, 42)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewWorldForKeyWithSeed(key, 8, 42)
	if err != nil {
		t.Fatal(err)
	}
	otherSeed, err := NewWorldForKeyWithSeed(key, 8, 99)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if left, right := first.NextRandomInt63(), second.NextRandomInt63(); left != right {
			t.Fatalf("same seed diverged at draw %d: %d != %d", i, left, right)
		}
	}
	if first.NextRandomInt63() == otherSeed.NextRandomInt63() {
		t.Fatal("different seeds produced identical first draw")
	}
}

func TestHeadlessZoneRunsTenThousandTicksWithTwoPlayersAndTenMonsters(t *testing.T) {
	build := func() *World {
		w, err := NewWorld("headless_test", DefaultQueueSize)
		if err != nil {
			t.Fatal(err)
		}
		for index := 0; index < 12; index++ {
			entity := newPlayer("actor")
			entity.ID = EntityID(fmt.Sprintf("actor-%02d", index))
			if index >= 2 {
				entity.Kind = EntityMonster
			}
			entity.Transform.Position = Vec2{X: float64(index), Y: 1}
			if err := w.AddEntity(entity); err != nil {
				t.Fatal(err)
			}
		}
		return w
	}
	a, b := build(), build()
	for tick := 1; tick <= 10_000; tick++ {
		if tick%4 == 1 {
			for _, world := range []*World{a, b} {
				for index := 0; index < 2; index++ {
					if err := world.SubmitInput(InputCommand{EntityID: EntityID(fmt.Sprintf("actor-%02d", index)), Seq: uint64(tick), Move: Vec2{X: 0.6, Y: 0.8}}); err != nil {
						t.Fatal(err)
					}
				}
			}
		}
		if left, right := a.Step(), b.Step(); !reflect.DeepEqual(left, right) {
			t.Fatalf("headless worlds diverged at tick %d", tick)
		}
	}
	if snapshot := a.Snapshot(); snapshot.Tick != 10_000 || len(snapshot.Entities) != 12 {
		t.Fatalf("unexpected final headless snapshot: tick=%d entities=%d", snapshot.Tick, len(snapshot.Entities))
	}
}

func TestConcurrentInputSubmissionAndZoneStepping(t *testing.T) {
	w, err := NewWorld("demo", 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newPlayer("a")); err != nil {
		t.Fatal(err)
	}
	producerDone := make(chan struct{})
	go func() {
		defer close(producerDone)
		for seq := uint64(1); seq <= 2_000; {
			err := w.SubmitInput(InputCommand{EntityID: "a", Seq: seq, Move: Vec2{X: 1}})
			if errors.Is(err, ErrQueueFull) {
				runtime.Gosched()
				continue
			}
			if err != nil {
				t.Errorf("SubmitInput(%d): %v", seq, err)
				return
			}
			seq++
		}
	}()
	for {
		select {
		case <-producerDone:
			if len(w.inbox) == 0 {
				return
			}
		default:
		}
		w.Step()
		runtime.Gosched()
	}
}

func TestFixedStepSequenceAndLease(t *testing.T) {
	w, err := NewWorld("demo", 8)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newPlayer("a")); err != nil {
		t.Fatal(err)
	}
	for _, input := range []InputCommand{{EntityID: "a", Seq: 2, Move: Vec2{X: 1}}, {EntityID: "a", Seq: 1, Move: Vec2{Y: 1}}} {
		if err := w.SubmitInput(input); err != nil {
			t.Fatal(err)
		}
	}
	snap := w.Step()
	if snap.Tick != 1 || snap.Entities[0].Transform.Position != (Vec2{X: 0.2}) || snap.Entities[0].AckSeq != 2 {
		t.Fatalf("unexpected first tick: %+v", snap)
	}
	if err := w.SubmitInput(InputCommand{EntityID: "a", Seq: 1, Move: Vec2{Y: 1}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < InputLeaseTicks+2; i++ {
		w.Step()
	}
	stationary := w.Snapshot().Entities[0].Transform.Position
	w.Step()
	if w.Snapshot().Entities[0].Transform.Position != stationary {
		t.Fatal("movement continued after input lease expired")
	}
	if w.Snapshot().Entities[0].AckSeq != 2 {
		t.Fatal("replayed input was accepted")
	}
}

func TestQueueLimitAndInvalidVectors(t *testing.T) {
	w, _ := NewWorld("demo", 1)
	if err := w.SubmitInput(InputCommand{EntityID: "a", Seq: 1, Move: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitInput(InputCommand{EntityID: "a", Seq: 2}); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("queue: %v", err)
	}
	if err := w.SubmitInput(InputCommand{EntityID: "a", Seq: 3, Move: Vec2{X: math.NaN()}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("nan: %v", err)
	}
	if err := w.SubmitInput(InputCommand{EntityID: "a", Seq: 4, Move: Vec2{X: 10}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("speed hack: %v", err)
	}
}

func TestStaleLeaseInputCannotMoveReplacementEntity(t *testing.T) {
	w, _ := NewWorld("demo", 8)
	entity := newPlayer("a")
	entity.LeaseToken = "new-controller-token"
	if err := w.AddEntity(entity); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitInput(InputCommand{EntityID: "a", LeaseToken: "old-controller-token", Seq: 1, Move: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	snapshot := w.Step()
	if snapshot.Entities[0].AckSeq != 0 || snapshot.Entities[0].Transform.Position.X != 0 {
		t.Fatalf("stale controller input affected replacement entity: %+v", snapshot.Entities[0])
	}
	if err := w.SubmitInput(InputCommand{EntityID: "a", LeaseToken: "new-controller-token", Seq: 1, Move: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	snapshot = w.Step()
	if snapshot.Entities[0].AckSeq != 1 || snapshot.Entities[0].Transform.Position.X <= 0 {
		t.Fatalf("current controller input was not applied: %+v", snapshot.Entities[0])
	}
}

func TestWorldDeterminismAndSnapshotIsolation(t *testing.T) {
	build := func() *World {
		w, _ := NewWorld("demo", 32)
		for _, id := range []string{"b", "a"} {
			if err := w.AddEntity(newPlayer(id)); err != nil {
				t.Fatal(err)
			}
		}
		return w
	}
	a, b := build(), build()
	for tick := 1; tick <= 100; tick++ {
		for _, w := range []*World{a, b} {
			for _, id := range []string{"b", "a"} {
				if err := w.SubmitInput(InputCommand{EntityID: EntityID(id), Seq: uint64(tick), Move: Vec2{X: 0.6, Y: 0.8}}); err != nil {
					t.Fatal(err)
				}
			}
		}
		if left, right := a.Step(), b.Step(); !reflect.DeepEqual(left, right) {
			t.Fatalf("diverged on tick %d", tick)
		}
	}
	snapshot := a.Snapshot()
	snapshot.Entities[0].Vitals.HP = 0
	if a.Snapshot().Entities[0].Vitals.HP != 100 {
		t.Fatal("snapshot mutated authoritative state")
	}
}

func TestStaticCollisionBlocksMovementAndAllowsWallSliding(t *testing.T) {
	solids := make([]bool, 16)
	solids[1] = true // tile (1, 0)
	grid, err := NewGridCollision(4, 4, solids)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorldWithCollision("demo", 8, grid)
	if err != nil {
		t.Fatal(err)
	}
	player := newPlayer("a")
	player.Transform.Position = Vec2{X: 0.5, Y: 0.5}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitInput(InputCommand{EntityID: "a", Seq: 1, Move: Vec2{X: 0.70710678, Y: 0.70710678}}); err != nil {
		t.Fatal(err)
	}
	position := w.Step().Entities[0].Transform.Position
	if position.X != 0.5 {
		t.Fatalf("actor crossed solid tile: %+v", position)
	}
	if position.Y <= 0.5 {
		t.Fatalf("actor failed to slide along wall: %+v", position)
	}
}

func TestGridCollisionTreatsMapBoundsAsSolidAndCopiesTiles(t *testing.T) {
	solids := make([]bool, 4)
	grid, err := NewGridCollision(2, 2, solids)
	if err != nil {
		t.Fatal(err)
	}
	solids[0] = true
	if grid.Blocked(Vec2{X: 0.5, Y: 0.5}, 0) {
		t.Fatal("caller mutation changed collision map")
	}
	if !grid.Blocked(Vec2{X: -0.1, Y: 0.5}, 0) {
		t.Fatal("position outside map should be blocked")
	}
	if !grid.Blocked(Vec2{X: 0.5, Y: 0.5}, 1e100) {
		t.Fatal("unreasonably large radius should be rejected quickly")
	}
	if _, err := NewGridCollision(2, 2, []bool{false}); !errors.Is(err, ErrInvalidCollisionMap) {
		t.Fatalf("invalid dimensions: %v", err)
	}
	if _, err := NewGridCollision(MaxCollisionTiles+1, 1, nil); !errors.Is(err, ErrInvalidCollisionMap) {
		t.Fatalf("oversized map: %v", err)
	}
}

func TestCompactCollisionMatchesGridAtEdgesAndRadii(t *testing.T) {
	for _, size := range [][2]int{{1, 1}, {2, 5}, {7, 6}} {
		width, height := size[0], size[1]
		borderTiles := make([]bool, width*height)
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				borderTiles[y*width+x] = x == 0 || y == 0 || x == width-1 || y == height-1
			}
		}
		grid, _ := NewGridCollision(width, height, borderTiles)
		border, err := NewBorderCollision(width, height)
		if err != nil {
			t.Fatal(err)
		}
		dense := make([]int, len(borderTiles))
		for i, solid := range borderTiles {
			if solid || i%7 == 0 {
				dense[i] = 1
			}
		}
		packed, err := NewPackedCollision(width, height, dense)
		if err != nil {
			t.Fatal(err)
		}
		denseGridTiles := make([]bool, len(dense))
		for i, value := range dense {
			denseGridTiles[i] = value == 1
		}
		denseGrid, _ := NewGridCollision(width, height, denseGridTiles)
		// Include exact tile edges, corners, outside coordinates and radii
		// that reach neighboring tiles only by touching their boundary.
		for _, radius := range []float64{0, 0.1, 0.5, 1, 2, float64(width + height), math.NaN()} {
			for y := -1.0; y <= float64(height)+1; y += 0.5 {
				for x := -1.0; x <= float64(width)+1; x += 0.5 {
					position := Vec2{X: x, Y: y}
					if got, want := border.Blocked(position, radius), grid.Blocked(position, radius); got != want {
						t.Fatalf("border %dx%d at %+v radius %v: got %v want %v", width, height, position, radius, got, want)
					}
					if got, want := packed.Blocked(position, radius), denseGrid.Blocked(position, radius); got != want {
						t.Fatalf("packed %dx%d at %+v radius %v: got %v want %v", width, height, position, radius, got, want)
					}
				}
			}
		}
	}
}

func TestEntityRadiusIsBounded(t *testing.T) {
	player := newPlayer("a")
	player.Radius = MaxEntityRadius + 1
	if err := validateEntity(player); !errors.Is(err, ErrInvalidEntity) {
		t.Fatalf("oversized radius: %v", err)
	}
}

func TestEntityCannotSpawnInsideStaticCollision(t *testing.T) {
	grid, err := NewGridCollision(2, 2, []bool{true, false, false, false})
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorldWithCollision("demo", 8, grid)
	if err != nil {
		t.Fatal(err)
	}
	player := newPlayer("a")
	player.Transform.Position = Vec2{X: 0.5, Y: 0.5}
	if err := w.AddEntity(player); !errors.Is(err, ErrInvalidEntity) {
		t.Fatalf("spawn in a solid tile: %v", err)
	}
}

func TestHighSpeedMovementCannotTunnelThroughSolidTiles(t *testing.T) {
	solids := make([]bool, 8*2)
	solids[3] = true
	solids[8+3] = true
	grid, err := NewGridCollision(8, 2, solids)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorldWithCollision("demo", 8, grid)
	if err != nil {
		t.Fatal(err)
	}
	player := newPlayer("a")
	player.Transform.Position = Vec2{X: 1, Y: 0.5}
	player.Speed = MaxEntitySpeed
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitInput(InputCommand{EntityID: "a", Seq: 1, Move: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	position := w.Step().Entities[0].Transform.Position
	if position.X >= 3 {
		t.Fatalf("high-speed actor tunneled through wall: %+v", position)
	}
}

func TestMoveTargetMovesActorAndStopsAtDestination(t *testing.T) {
	w, _ := NewWorld("demo", 8)
	if err := w.AddEntity(newPlayer("a")); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitMoveTarget(InputCommand{EntityID: "a", Seq: 1, Target: &Vec2{X: 1, Y: 0}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		w.Step()
	}
	position := entityPosition(t, w, "a")
	if math.Abs(position.X-1) > MoveTargetArriveEpsilon || math.Abs(position.Y) > 0.001 {
		t.Fatalf("actor did not stop at click destination: %+v", position)
	}
	settled := position
	w.Step()
	if after := entityPosition(t, w, "a"); after != settled {
		t.Fatalf("actor kept moving after reaching destination: before=%+v after=%+v", settled, after)
	}
}

func TestMoveTargetIsClampedNearTheActor(t *testing.T) {
	w, _ := NewWorld("demo", 8)
	if err := w.AddEntity(newPlayer("a")); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitMoveTarget(InputCommand{EntityID: "a", Seq: 1, Target: &Vec2{X: 1000, Y: 0}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2000; i++ {
		w.Step()
	}
	position := entityPosition(t, w, "a")
	if position.X > MaxMoveTargetDistance+MoveTargetArriveEpsilon || math.Abs(position.Y) > 0.001 {
		t.Fatalf("click destination was not bounded: %+v", position)
	}
}

func TestWASDCancelsActiveMoveTarget(t *testing.T) {
	w, _ := NewWorld("demo", 8)
	if err := w.AddEntity(newPlayer("a")); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitMoveTarget(InputCommand{EntityID: "a", Seq: 1, Target: &Vec2{X: 10, Y: 0}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		w.Step()
	}
	if err := w.SubmitInput(InputCommand{EntityID: "a", Seq: 2, Move: Vec2{X: 0, Y: 1}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		w.Step()
	}
	position := entityPosition(t, w, "a")
	if position.Y < 0.5 {
		t.Fatalf("WASD input did not take over from click path: %+v", position)
	}
	if position.X > 1.5 {
		t.Fatalf("click path kept moving after WASD takeover: %+v", position)
	}
}

func TestMoveTargetValidationAndStaleLease(t *testing.T) {
	w, _ := NewWorld("demo", 8)
	player := newPlayer("a")
	player.LeaseToken = "lease"
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitMoveTarget(InputCommand{EntityID: "a", Seq: 1, Target: nil}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("nil target error=%v", err)
	}
	if err := w.SubmitMoveTarget(InputCommand{EntityID: "a", Seq: 1, Target: &Vec2{X: math.NaN()}}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("NaN target error=%v", err)
	}
	if err := w.SubmitMoveTarget(InputCommand{EntityID: "a", Seq: 1, LeaseToken: "stale", Target: &Vec2{X: 5}}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		w.Step()
	}
	if position := entityPosition(t, w, "a"); position.X != 0 {
		t.Fatalf("stale lease target moved the actor: %+v", position)
	}
}

func entityPosition(t *testing.T, w *World, id EntityID) Vec2 {
	t.Helper()
	for _, entity := range w.Snapshot().Entities {
		if entity.ID == id {
			return entity.Transform.Position
		}
	}
	t.Fatalf("entity %q missing from snapshot", id)
	return Vec2{}
}

func TestBlockedMoveTargetEmitsRejectionEvent(t *testing.T) {
	solid := make([]bool, 10*10)
	for i := 0; i < 10; i++ {
		solid[i], solid[9*10+i], solid[i*10], solid[i*10+9] = true, true, true, true
	}
	collision, err := NewGridCollision(10, 10, solid)
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorldWithCollision("arena", 8, collision)
	if err != nil {
		t.Fatal(err)
	}
	player := newCombatPlayer("player", Vec2{X: 2, Y: 2})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitMoveTarget(InputCommand{EntityID: "player", Seq: 1, Target: &Vec2{X: 0.5, Y: 2}}); err != nil {
		t.Fatal(err)
	}
	events := w.Step().Events
	found := false
	for _, event := range events {
		if event.Type == "MOVE_TARGET_REJECTED" && event.Reason == "blocked_destination" {
			found = true
		}
	}
	if !found {
		t.Fatalf("blocked destination did not emit a rejection event: %+v", events)
	}
}
