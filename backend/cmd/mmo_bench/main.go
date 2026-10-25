// mmo_bench is a reproducible synthetic load harness for the zone tick. It
// measures CPU step percentiles with scripted movement/casts; it is not a
// browser FPS or network benchmark. Results are reported with that caveat.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"runtime"
	"sort"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

func main() {
	players := flag.Int("players", 30, "player entities (1..100)")
	mobs := flag.Int("mobs", 50, "monster entities (1..50)")
	ticks := flag.Int("ticks", 1000, "fixed 20 Hz steps")
	seed := flag.Int64("seed", 42, "zone RNG seed")
	flag.Parse()
	if *players < 1 || *players > 100 {
		log.Fatal("players must be between 1 and 100")
	}
	if *mobs < 1 || *mobs > 50 {
		log.Fatal("mobs must be between 1 and 50")
	}
	if *ticks < 100 || *ticks > 100000 {
		log.Fatal("ticks must be between 100 and 100000")
	}
	key := sim.ZoneKey{MapID: "bench", InstanceID: "synthetic", RulesetVersion: "bench-v1"}
	world, err := sim.NewWorldForKeyWithSeed(key, sim.DefaultQueueSize*4, *seed)
	if err != nil {
		log.Fatal(err)
	}
	for index := 0; index < *players; index++ {
		entity := sim.Entity{
			ID: sim.EntityID(fmt.Sprintf("p:%03d", index)), CharacterID: sim.CharacterID(fmt.Sprintf("char-%03d", index)),
			Kind: sim.EntityPlayer, Transform: sim.Transform{Position: sim.Vec2{X: 2 + float64(index%10)*2, Y: 2 + float64(index/10)*2}},
			Vitals: sim.Vitals{HP: 10000, MaxHP: 10000}, Speed: 4, Radius: 0.4, Faction: "player",
			PartyID: "bench-party", Combat: sim.NewPlayerCombat(),
		}
		if err := world.AddEntity(entity); err != nil {
			log.Fatal(err)
		}
	}
	for index := 0; index < *mobs; index++ {
		entity := sim.Entity{
			ID: sim.EntityID(fmt.Sprintf("m:%02d", index)), Kind: sim.EntityMonster,
			Transform: sim.Transform{Position: sim.Vec2{X: 30 + float64(index%10)*2, Y: 2 + float64(index/10)*2}},
			Vitals:    sim.Vitals{HP: 80, MaxHP: 80}, Speed: 2.5, Radius: 0.5, Faction: "monster",
			AI: sim.AIState{
				Active: true, State: sim.AIIdle, Home: sim.Vec2{X: 30 + float64(index%10)*2, Y: 2 + float64(index/10)*2},
				AggroRadius: 10, LeashRadius: 24, AttackRange: 1.2, PatrolRadius: 2, AttackDamage: 4,
				AttackCooldownTicks: 30, RespawnTicks: 600, ThinkTicks: 5,
			},
		}
		if err := world.AddEntity(entity); err != nil {
			log.Fatal(err)
		}
	}

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	durations := make([]time.Duration, 0, *ticks)
	seq := uint64(0)
	deaths := 0
	started := time.Now()
	var final sim.Snapshot
	for tick := 1; tick <= *ticks; tick++ {
		snapshot := world.Snapshot()
		for index := 0; index < *players; index++ {
			player := find(snapshot, sim.EntityID(fmt.Sprintf("p:%03d", index)))
			target := nearest(snapshot, player)
			if player == nil || target == nil {
				continue
			}
			dx := target.Transform.Position.X - player.Transform.Position.X
			dy := target.Transform.Position.Y - player.Transform.Position.Y
			distance := math.Hypot(dx, dy)
			if distance > 0.1 {
				seq++
				_ = world.SubmitInput(sim.InputCommand{EntityID: player.ID, Seq: seq, Move: sim.Vec2{X: dx / distance, Y: dy / distance}})
			}
			if tick%10 == 0 && distance < 16 {
				seq++
				cast := &sim.CastCommand{EntityID: player.ID, Seq: seq, Slot: 1, Mode: combat.ModeTarget, TargetID: target.ID}
				if distance > 9 {
					cast.Slot, cast.Mode, cast.TargetID = 2, combat.ModeDirection, ""
					cast.Aim = sim.Vec2{X: dx / distance, Y: dy / distance}
				}
				_ = world.SubmitCast(sim.CastCommand{
					EntityID: player.ID, Seq: seq, Slot: cast.Slot, Mode: cast.Mode,
					TargetID: cast.TargetID, Aim: cast.Aim, Point: cast.Point,
				})
			}
		}
		stepStart := time.Now()
		final = world.Step()
		durations = append(durations, time.Since(stepStart))
		for _, event := range final.Events {
			if event.Type == combat.EventEntityDied {
				deaths++
			}
		}
	}
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	data, _ := json.Marshal(final)
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
	p50 := percentile(durations, 0.50)
	p95 := percentile(durations, 0.95)
	p99 := percentile(durations, 0.99)
	snapshotBytes := len(data)

	fmt.Printf("mmo-bench synthetic zone: players=%d mobs=%d ticks=%d seed=%d\n", *players, *mobs, *ticks, *seed)
	fmt.Printf("elapsed=%s avg_step=%s p50=%s p95=%s p99=%s deaths=%d\n",
		elapsed.Round(time.Millisecond), (elapsed / time.Duration(*ticks)).Round(time.Microsecond),
		p50.Round(time.Microsecond), p95.Round(time.Microsecond), p99.Round(time.Microsecond), deaths)
	fmt.Printf("final_snapshot_json_bytes=%d heap_alloc_delta=%dKB total_alloc_delta=%dKB\n",
		snapshotBytes, (int64(after.HeapAlloc)-int64(before.HeapAlloc))/1024,
		(int64(after.TotalAlloc)-int64(before.TotalAlloc))/1024)
	fmt.Println("NOTE: synthetic CPU step only; browser FPS, RTT, jitter, packet loss, AOI bytes/client and DB/Redis latency are NOT MEASURED.")
}

func find(snapshot sim.Snapshot, id sim.EntityID) *sim.Entity {
	for index := range snapshot.Entities {
		if snapshot.Entities[index].ID == id {
			return &snapshot.Entities[index]
		}
	}
	return nil
}

func nearest(snapshot sim.Snapshot, from *sim.Entity) *sim.Entity {
	if from == nil {
		return nil
	}
	var nearest *sim.Entity
	best := math.Inf(1)
	for index := range snapshot.Entities {
		entity := &snapshot.Entities[index]
		if entity.Kind != sim.EntityMonster || entity.Vitals.HP <= 0 {
			continue
		}
		distance := math.Hypot(entity.Transform.Position.X-from.Transform.Position.X, entity.Transform.Position.Y-from.Transform.Position.Y)
		if distance < best || (distance == best && nearest != nil && entity.ID < nearest.ID) {
			nearest = entity
			best = distance
		}
	}
	return nearest
}

func percentile(sorted []time.Duration, ratio float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(ratio*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
