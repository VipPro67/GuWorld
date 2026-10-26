package sim

import (
	"math"
	"testing"
)

func newAIMonster(id string, home Vec2) Entity {
	return Entity{
		ID: EntityID(id), Kind: EntityMonster,
		Transform: Transform{Position: home},
		Vitals:    Vitals{HP: 50, MaxHP: 50},
		Speed:     3, Radius: 0.5, Faction: "monster",
		AI: AIState{
			Active: true, State: AIIdle, Home: home,
			AggroRadius: 6, LeashRadius: 10, AttackRange: 1.2, PatrolRadius: 0,
			AttackDamage: 5, AttackCooldownTicks: 2, ThinkTicks: 1, RespawnTicks: 10,
		},
	}
}

func TestMonsterAIAggroChaseAndAttack(t *testing.T) {
	w, err := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "ai", RulesetVersion: "v1"}, 8, 7)
	if err != nil {
		t.Fatal(err)
	}
	player := newPlayer("player")
	player.Transform.Position = Vec2{X: 0, Y: 0}
	monster := newAIMonster("mob", Vec2{X: 4, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		w.Step()
	}
	snapshot := w.Snapshot()
	var playerAfter, monsterAfter Entity
	for _, entity := range snapshot.Entities {
		switch entity.ID {
		case "player":
			playerAfter = entity
		case "mob":
			monsterAfter = entity
		}
	}
	if playerAfter.Vitals.HP >= 100 {
		t.Fatalf("monster never damaged the player: %+v", playerAfter.Vitals)
	}
	if monsterAfter.AI.State != AIAttack {
		t.Fatalf("monster state=%q, want attack", monsterAfter.AI.State)
	}
	distance := math.Hypot(monsterAfter.Transform.Position.X-playerAfter.Transform.Position.X, monsterAfter.Transform.Position.Y-playerAfter.Transform.Position.Y)
	if distance > monster.AI.AttackRange+0.01 {
		t.Fatalf("monster did not stop in attack range: distance=%.2f", distance)
	}
}

func TestPassiveMonsterIgnoresProximityButRetaliates(t *testing.T) {
	w, err := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "passive", RulesetVersion: "v1"}, 8, 11)
	if err != nil {
		t.Fatal(err)
	}
	player := newPlayer("player")
	player.Transform.Position = Vec2{X: 0, Y: 0}
	monster := newAIMonster("mob", Vec2{X: 2, Y: 0})
	monster.AI.Passive = true
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		w.Step()
	}
	for _, entity := range w.Snapshot().Entities {
		if entity.ID == "player" && entity.Vitals.HP < 100 {
			t.Fatalf("passive monster initiated combat: %+v", entity.Vitals)
		}
		if entity.ID == "mob" && entity.AI.Threat != "" {
			t.Fatalf("passive monster acquired a proximity target: %q", entity.AI.Threat)
		}
	}

	// A threat entry (player attacked first) still opts into retaliation.
	retaliating, _ := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "passive-revenge", RulesetVersion: "v1"}, 8, 11)
	player = newPlayer("player")
	player.Transform.Position = Vec2{X: 0, Y: 0}
	monster = newAIMonster("mob", Vec2{X: 2, Y: 0})
	monster.AI.Passive = true
	monster.AI.ThreatTable = map[EntityID]float64{player.ID: 1}
	if err := retaliating.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := retaliating.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 40; i++ {
		retaliating.Step()
	}
	for _, entity := range retaliating.Snapshot().Entities {
		if entity.ID == "player" && entity.Vitals.HP >= 100 {
			t.Fatal("passive monster did not retaliate against a threat target")
		}
	}
}

func TestMonsterAILeashesBackToHome(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	player := newPlayer("player")
	player.Transform.Position = Vec2{X: 30, Y: 0}
	monster := newAIMonster("mob", Vec2{X: 0, Y: 0})
	monster.Transform.Position = Vec2{X: 20, Y: 0}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 20; i++ {
		w.Step()
	}
	position := entityPosition(t, w, "mob")
	if position.X >= 20 {
		t.Fatalf("monster did not leash toward home: %+v", position)
	}
	for _, entity := range w.Snapshot().Entities {
		if entity.ID == "mob" {
			if entity.AI.Threat != "" {
				t.Fatalf("leashed monster kept a threat: %q", entity.AI.Threat)
			}
			if entity.AI.State != AILeash && entity.AI.State != AIIdle {
				t.Fatalf("unexpected leash state %q", entity.AI.State)
			}
		}
	}
}

func TestMonsterPatrolIsDeterministicWithZoneSeed(t *testing.T) {
	key := ZoneKey{MapID: "arena", InstanceID: "patrol", RulesetVersion: "v1"}
	first, _ := NewWorldForKeyWithSeed(key, 8, 99)
	second, _ := NewWorldForKeyWithSeed(key, 8, 99)
	for _, world := range []*World{first, second} {
		monster := newAIMonster("mob", Vec2{X: 10, Y: 10})
		monster.AI.PatrolRadius = 4
		if err := world.AddEntity(monster); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 100; i++ {
		first.Step()
		second.Step()
	}
	if entityPosition(t, first, "mob") != entityPosition(t, second, "mob") {
		t.Fatal("monster patrol diverged with the same zone seed")
	}
}

func TestMonsterDamageDeathAndRespawn(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	monster := newAIMonster("mob", Vec2{X: 5, Y: 5})
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	result, err := w.QueueDamage("mob", 999)
	if err != nil {
		t.Fatal(err)
	}
	w.Step()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if hp := entityHP(t, w, "mob"); hp != 0 {
		t.Fatalf("monster hp=%v, want 0", hp)
	}
	for i := 0; i < 10; i++ {
		w.Step()
	}
	if hp := entityHP(t, w, "mob"); hp != 50 {
		t.Fatalf("monster did not respawn: hp=%v", hp)
	}
	if position := entityPosition(t, w, "mob"); position != (Vec2{X: 5, Y: 5}) {
		t.Fatalf("monster did not respawn at home: %+v", position)
	}
}

func entityHP(t *testing.T, w *World, id EntityID) float64 {
	t.Helper()
	for _, entity := range w.Snapshot().Entities {
		if entity.ID == id {
			return entity.Vitals.HP
		}
	}
	t.Fatalf("entity %q missing from snapshot", id)
	return 0
}
