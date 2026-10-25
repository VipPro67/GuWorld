// mmo_slice is a headless vertical-slice smoke test for the realtime world. It
// joins multiple players into one manager-owned zone, moves them, casts the
// server preview loadout at living monsters, and prints deterministic metrics.
// It does not require PostgreSQL/Redis; durable loot/checkpoint paths are
// exercised by their own unit/integration tests and are reported NOT MEASURED.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
)

func main() {
	players := flag.Int("players", 5, "number of player entities (1..5)")
	mobs := flag.Int("mobs", 10, "number of monsters (1..50)")
	ticks := flag.Int("ticks", 400, "maximum fixed 20 Hz ticks")
	flag.Parse()
	if *players < 1 || *players > 5 {
		log.Fatal("players must be between 1 and 5")
	}
	if *mobs < 1 || *mobs > 50 {
		log.Fatal("mobs must be between 1 and 50")
	}
	if *ticks < 20 || *ticks > 100000 {
		log.Fatal("ticks must be between 20 and 100000")
	}

	key := sim.ZoneKey{MapID: "slice", InstanceID: "gate-b", RulesetVersion: "v1"}
	spawns := make([]realtimeworld.MonsterSpawn, 0, *mobs)
	for index := 0; index < *mobs; index++ {
		spawns = append(spawns, realtimeworld.MonsterSpawn{
			TemplateID: fmt.Sprintf("slice_mob_%02d", index),
			X:          8 + float64(index%5)*3, Y: 8 + float64(index/5)*3,
			MaxHP: 80, AttackDamage: 4,
			Speed: 2.5, Radius: 0.5, AggroRadius: 10, LeashRadius: 22,
			AttackRange: 1.2, PatrolRadius: 2, AttackCooldownTicks: 30, RespawnTicks: 600,
		})
	}
	manager, err := realtimeworld.NewManagerWithMaps(
		[]sim.ZoneKey{key}, sim.DefaultQueueSize, *players,
		map[sim.ZoneKey]realtimeworld.ZoneMap{key: {Monsters: spawns}},
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := manager.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = manager.Stop(ctx)
	}()

	type session struct {
		character sim.CharacterID
		lease     realtimeworld.OwnershipLease
		seq       uint64
	}
	sessions := make([]session, 0, *players)
	for index := 0; index < *players; index++ {
		characterID := sim.CharacterID(uuid.NewString())
		lease, err := manager.JoinCharacter(characterID, uuid.NewString(), key, sim.Entity{
			ID: sim.EntityID(characterID), CharacterID: characterID, Kind: sim.EntityPlayer,
			Transform: sim.Transform{Position: sim.Vec2{X: 2 + float64(index), Y: 2}},
			Vitals:    sim.Vitals{HP: 5000, MaxHP: 5000},
			Speed:     4, Radius: 0.4, Faction: "player",
			PartyID: "slice-party",
			Combat:  sim.NewPlayerCombat(),
		})
		if err != nil {
			log.Fatal(err)
		}
		sessions = append(sessions, session{character: characterID, lease: lease})
	}

	start := time.Now()
	eventCounts := map[string]int{}
	var final sim.Snapshot
	monstersDead := 0
	for tick := 1; tick <= *ticks; tick++ {
		final, _ = manager.Snapshot(key)
		if tick%4 == 1 {
			for index := range sessions {
				current := findEntity(final, sim.EntityID(sessions[index].character))
				target := nearestMonster(final, current)
				if current == nil || target == nil {
					continue
				}
				dx := target.Transform.Position.X - current.Transform.Position.X
				dy := target.Transform.Position.Y - current.Transform.Position.Y
				length := math.Hypot(dx, dy)
				if length > 0.1 {
					sessions[index].seq++
					_ = manager.SubmitInput(sessions[index].lease, sim.InputCommand{
						EntityID: sim.EntityID(sessions[index].character), Seq: sessions[index].seq,
						Move: sim.Vec2{X: dx / length, Y: dy / length},
					})
				}
			}
		}
		if tick%10 == 1 {
			for index := range sessions {
				current := findEntity(final, sim.EntityID(sessions[index].character))
				target := nearestMonster(final, current)
				if current == nil || target == nil {
					continue
				}
				distance := math.Hypot(target.Transform.Position.X-current.Transform.Position.X, target.Transform.Position.Y-current.Transform.Position.Y)
				sessions[index].seq++
				cast := &sim.CastCommand{
					EntityID: sim.EntityID(sessions[index].character), Seq: sessions[index].seq,
					Slot: 1, Mode: "TARGET", TargetID: target.ID,
				}
				if distance > 9 {
					cast.Slot = 2
					cast.Mode = "DIRECTION"
					cast.TargetID = ""
					cast.Aim = sim.Vec2{
						X: (target.Transform.Position.X - current.Transform.Position.X) / math.Max(0.001, distance),
						Y: (target.Transform.Position.Y - current.Transform.Position.Y) / math.Max(0.001, distance),
					}
				}
				_ = manager.SubmitCast(sessions[index].lease, sim.InputCommand{
					EntityID: sim.EntityID(sessions[index].character), Seq: sessions[index].seq, Cast: cast,
				})
			}
		}
		final, _ = manager.Snapshot(key)
		for _, event := range final.Events {
			eventCounts[string(event.Type)]++
			if string(event.Type) == "ENTITY_DIED" && event.Reason == "" {
				monstersDead++
			}
		}
		if monstersDead >= *mobs {
			break
		}
		time.Sleep(sim.TickDuration)
	}
	elapsed := time.Since(start)
	playersAlive := 0
	for _, entity := range final.Entities {
		if entity.Kind == sim.EntityPlayer && entity.Vitals.HP > 0 {
			playersAlive++
		}
	}
	data, _ := json.Marshal(final)
	keys := make([]string, 0, len(eventCounts))
	for eventType := range eventCounts {
		keys = append(keys, eventType)
	}
	sort.Strings(keys)
	fmt.Printf("mmo-slice: players=%d/%d mobs_dead=%d/%d zone=%s tick=%d\n",
		*players, *players, monstersDead, *mobs, final.ZoneKey.String(), final.Tick)
	fmt.Printf("mmo-slice: elapsed=%s tick_p50=%s checksum=%x\n",
		elapsed.Round(time.Millisecond), (elapsed / time.Duration(max(1, int(final.Tick)))).Round(time.Microsecond), sha256.Sum256(data))
	for _, eventType := range keys {
		fmt.Printf("  event %-28s %d\n", eventType, eventCounts[eventType])
	}
	fmt.Println("mmo-slice: durable loot/checkpoint = NOT MEASURED (requires PostgreSQL)")
	if monstersDead == 0 || playersAlive == 0 {
		log.Fatalf("slice failed: dead_monsters=%d alive_players=%d", monstersDead, playersAlive)
	}
}

func findEntity(snapshot sim.Snapshot, id sim.EntityID) *sim.Entity {
	for index := range snapshot.Entities {
		if snapshot.Entities[index].ID == id {
			return &snapshot.Entities[index]
		}
	}
	return nil
}

func nearestMonster(snapshot sim.Snapshot, from *sim.Entity) *sim.Entity {
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

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
