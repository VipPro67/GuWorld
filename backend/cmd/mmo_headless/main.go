// mmo_headless exercises the isolated zone simulation without DB or sockets.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
)

func main() {
	ticks := flag.Int("ticks", 10000, "number of fixed 20 Hz steps")
	flag.Parse()
	if *ticks < 1 || *ticks > 1_000_000 {
		log.Fatal("ticks must be between 1 and 1000000")
	}
	w, err := sim.NewWorldForKeyWithCollisionAndSeed(
		sim.ZoneKey{MapID: "headless_demo", InstanceID: "default", RulesetVersion: "v1"},
		sim.DefaultQueueSize,
		nil,
		1,
	)
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		kind := sim.EntityMonster
		if i < 2 {
			kind = sim.EntityPlayer
		}
		entity := sim.Entity{
			ID: sim.EntityID(fmt.Sprintf("actor-%02d", i)), Kind: kind,
			Transform: sim.Transform{Position: sim.Vec2{X: float64(i), Y: 0}},
			Vitals:    sim.Vitals{HP: 100, MaxHP: 100}, Speed: 4, Radius: 0.4,
		}
		if err := w.AddEntity(entity); err != nil {
			log.Fatal(err)
		}
	}
	var snapshot sim.Snapshot
	for tick := 1; tick <= *ticks; tick++ {
		if tick%4 == 1 {
			for i := 0; i < 12; i++ {
				if err := w.SubmitInput(sim.InputCommand{EntityID: sim.EntityID(fmt.Sprintf("actor-%02d", i)), Seq: uint64(tick), Move: sim.Vec2{X: 0.6, Y: 0.8}}); err != nil {
					log.Fatal(err)
				}
			}
		}
		snapshot = w.Step()
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("zone=%s tick=%d entities=%d sha256=%x\n", snapshot.ZoneKey.String(), snapshot.Tick, len(snapshot.Entities), sha256.Sum256(data))
}
