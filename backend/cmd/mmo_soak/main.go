// mmo_soak runs a wall-clock 20Hz soak with reconnects. It is intentionally
// short-run by default for local verification; use a long -duration in staging.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
)

type soakSession struct {
	character  sim.CharacterID
	controller string
	lease      realtimeworld.OwnershipLease
	seq        uint64
}

func main() {
	duration := flag.Duration("duration", 60*time.Second, "soak duration")
	players := flag.Int("players", 10, "player entities (1..30)")
	mobs := flag.Int("mobs", 20, "monsters (1..50)")
	reconnectEvery := flag.Duration("reconnect-every", 5*time.Second, "reconnect one player at this interval (0 disables)")
	flag.Parse()
	if *players < 1 || *players > 30 {
		log.Fatal("players must be between 1 and 30")
	}
	if *mobs < 1 || *mobs > 50 {
		log.Fatal("mobs must be between 1 and 50")
	}
	if *duration < time.Second || *duration > 24*time.Hour {
		log.Fatal("duration must be between 1s and 24h")
	}
	key := sim.ZoneKey{MapID: "soak", InstanceID: "gate-c", RulesetVersion: "v1"}
	spawns := make([]realtimeworld.MonsterSpawn, 0, *mobs)
	for index := 0; index < *mobs; index++ {
		spawns = append(spawns, realtimeworld.MonsterSpawn{
			TemplateID: fmt.Sprintf("soak_mob_%02d", index),
			X:          10 + float64(index%10)*2, Y: 10 + float64(index/10)*2,
			MaxHP: 200, AttackDamage: 2, Speed: 2.5, Radius: 0.5,
			AggroRadius: 8, LeashRadius: 20, AttackRange: 1.2, PatrolRadius: 2,
			AttackCooldownTicks: 30, RespawnTicks: 600,
		})
	}
	manager, err := realtimeworld.NewManagerWithMaps(
		[]sim.ZoneKey{key}, sim.DefaultQueueSize, *players,
		map[sim.ZoneKey]realtimeworld.ZoneMap{key: {Monsters: spawns}},
	)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()
	if err := manager.Start(ctx); err != nil {
		log.Fatal(err)
	}
	defer func() {
		shutdown, done := context.WithTimeout(context.Background(), 2*time.Second)
		defer done()
		_ = manager.Stop(shutdown)
	}()

	sessions := make([]*soakSession, 0, *players)
	for index := 0; index < *players; index++ {
		session, joinErr := joinPlayer(manager, key, nil, nil)
		if joinErr != nil {
			log.Fatal(joinErr)
		}
		sessions = append(sessions, session)
		_ = index
	}

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	started := time.Now()
	ticker := time.NewTicker(sim.TickDuration)
	defer ticker.Stop()
	var reconnectTimer *time.Ticker
	var reconnectChannel <-chan time.Time
	if *reconnectEvery > 0 {
		reconnectTimer = time.NewTicker(*reconnectEvery)
		reconnectChannel = reconnectTimer.C
		defer reconnectTimer.Stop()
	}
	eventCounts := map[string]int{}
	reconnects, inputErrors, castErrors := 0, 0, 0
	maxEntities, maxPlayers := 0, 0
	var final sim.Snapshot
	nextReconnect := 0
	for {
		select {
		case <-ctx.Done():
			final, _ = manager.Snapshot(key)
			goto report
		case <-reconnectChannel:
			victim := sessions[nextReconnect%len(sessions)]
			nextReconnect++
			position := sim.Vec2{X: 2 + float64(nextReconnect%10), Y: 2}
			if current, ok := findEntity(manager, key, sim.EntityID(victim.character)); ok {
				position = current.Transform.Position
			}
			manager.LeaveCharacter(victim.lease)
			rejoined, joinErr := joinPlayer(manager, key, &victim.character, &position)
			if joinErr != nil {
				log.Printf("reconnect failed: %v", joinErr)
				continue
			}
			victim.controller = rejoined.controller
			victim.lease = rejoined.lease
			victim.seq++
			reconnects++
		case <-ticker.C:
			snapshot, _ := manager.Snapshot(key)
			playersAlive := 0
			for _, entity := range snapshot.Entities {
				if entity.Kind == sim.EntityPlayer && entity.Vitals.HP > 0 {
					playersAlive++
				}
			}
			if len(snapshot.Entities) > maxEntities {
				maxEntities = len(snapshot.Entities)
			}
			if playersAlive > maxPlayers {
				maxPlayers = playersAlive
			}
			for index := range sessions {
				current, ok := findEntityInSnapshot(snapshot, sim.EntityID(sessions[index].character))
				if !ok {
					continue
				}
				target := nearestMonsterInSnapshot(snapshot, current)
				if target == nil {
					continue
				}
				dx := target.Transform.Position.X - current.Transform.Position.X
				dy := target.Transform.Position.Y - current.Transform.Position.Y
				distance := math.Hypot(dx, dy)
				if distance > 0.1 {
					sessions[index].seq++
					if err := manager.SubmitInput(sessions[index].lease, sim.InputCommand{
						EntityID: sim.EntityID(sessions[index].character), Seq: sessions[index].seq,
						Move: sim.Vec2{X: dx / distance, Y: dy / distance},
					}); err != nil {
						inputErrors++
					}
				}
				if distance < 16 {
					sessions[index].seq++
					if err := manager.SubmitCast(sessions[index].lease, sim.InputCommand{
						EntityID: sim.EntityID(sessions[index].character), Seq: sessions[index].seq,
						Cast: &sim.CastCommand{
							EntityID: sim.EntityID(sessions[index].character), Seq: sessions[index].seq,
							Slot: 1, Mode: "TARGET", TargetID: target.ID,
						},
					}); err != nil {
						castErrors++
					}
				}
			}
			for _, event := range snapshot.Events {
				eventCounts[string(event.Type)]++
			}
			final = snapshot
		}
	}
report:
	runtime.ReadMemStats(&after)
	data, _ := json.Marshal(final)
	uniqueOwners := 0
	for _, session := range sessions {
		if manager.OwnsCharacter(session.character) {
			uniqueOwners++
		}
	}
	fmt.Printf("mmo-soak: duration=%s players=%d mobs=%d reconnects=%d\n", time.Since(started).Round(time.Millisecond), *players, *mobs, reconnects)
	fmt.Printf("ownership_unique=%d/%d max_entities=%d max_players=%d tick=%d checksum=%x\n",
		uniqueOwners, len(sessions), maxEntities, maxPlayers, final.Tick, sha256.Sum256(data))
	fmt.Printf("input_errors=%d cast_errors=%d heap_alloc_delta=%dKB\n",
		inputErrors, castErrors, (int64(after.HeapAlloc)-int64(before.HeapAlloc))/1024)
	for eventType, count := range eventCounts {
		fmt.Printf("  event %-28s %d\n", eventType, count)
	}
	if uniqueOwners != len(sessions) {
		log.Fatalf("ownership duplication/loss after reconnect soak: %d/%d", uniqueOwners, len(sessions))
	}
}

func joinPlayer(manager *realtimeworld.Manager, key sim.ZoneKey, character *sim.CharacterID, position *sim.Vec2) (*soakSession, error) {
	characterID := sim.CharacterID(uuid.NewString())
	if character != nil {
		characterID = *character
	}
	spawn := sim.Vec2{X: 2, Y: 2}
	if position != nil {
		spawn = *position
	}
	controller := uuid.NewString()
	lease, err := manager.JoinCharacter(characterID, controller, key, sim.Entity{
		ID: sim.EntityID(characterID), CharacterID: characterID, Kind: sim.EntityPlayer,
		Transform: sim.Transform{Position: spawn},
		Vitals:    sim.Vitals{HP: 10000, MaxHP: 10000},
		Speed:     4, Radius: 0.4, Faction: "player", PartyID: "soak-party",
		Combat: sim.NewPlayerCombat(),
	})
	if err != nil {
		return nil, err
	}
	return &soakSession{character: characterID, controller: controller, lease: lease}, nil
}

func findEntity(manager *realtimeworld.Manager, key sim.ZoneKey, id sim.EntityID) (sim.Entity, bool) {
	snapshot, err := manager.Snapshot(key)
	if err != nil {
		return sim.Entity{}, false
	}
	return findEntityInSnapshot(snapshot, id)
}

func findEntityInSnapshot(snapshot sim.Snapshot, id sim.EntityID) (sim.Entity, bool) {
	for _, entity := range snapshot.Entities {
		if entity.ID == id {
			return entity, true
		}
	}
	return sim.Entity{}, false
}

func nearestMonsterInSnapshot(snapshot sim.Snapshot, from sim.Entity) *sim.Entity {
	var nearest *sim.Entity
	best := math.Inf(1)
	for index := range snapshot.Entities {
		entity := &snapshot.Entities[index]
		if entity.Kind != sim.EntityMonster || entity.Vitals.HP <= 0 {
			continue
		}
		distance := math.Hypot(entity.Transform.Position.X-from.Transform.Position.X, entity.Transform.Position.Y-from.Transform.Position.Y)
		if distance < best {
			nearest = entity
			best = distance
		}
	}
	return nearest
}
