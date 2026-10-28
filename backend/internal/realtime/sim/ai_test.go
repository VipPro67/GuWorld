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

func TestMonsterAttacksApertureResourceNodeAndSwitchesToPlayerOnThreat(t *testing.T) {
	w, err := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "aperture_test", RulesetVersion: "v1"}, 8, 42)
	if err != nil {
		t.Fatal(err)
	}
	// Player is far away (at 15, 0)
	player := newPlayer("player")
	player.Transform.Position = Vec2{X: 15, Y: 0}

	// Resource node (stationary plant/mine) at (2, 0) with player faction
	node := Entity{
		ID:        "node_moon_orchid",
		Kind:      EntityMonster,
		Name:      "Ruộng Lan Nguyệt",
		Transform: Transform{Position: Vec2{X: 2, Y: 0}},
		Vitals:    Vitals{HP: 500, MaxHP: 500},
		Speed:     0,
		Radius:    0.8,
		Faction:   "player",
		AI:        AIState{Active: false},
	}

	// Tribulation monster at (4, 0)
	monster := newAIMonster("trib_beast", Vec2{X: 4, Y: 0})
	monster.AI.Home = Vec2{X: 4, Y: 0}
	monster.AI.LeashRadius = 20

	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(node); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}

	// Step simulation: monster should approach and attack the resource node
	for i := 0; i < 30; i++ {
		w.Step()
	}

	nodeEntity := w.entities["node_moon_orchid"]
	if nodeEntity == nil || nodeEntity.Vitals.HP >= 500 {
		t.Fatalf("tribulation monster did not damage the resource node: %+v", nodeEntity.Vitals)
	}

	// Now player deals heavy threat to the monster
	w.recordThreatLocked(w.entities["trib_beast"], w.entities["player"], 1000)
	for i := 0; i < 5; i++ {
		w.Step()
	}

	mobEntity := w.entities["trib_beast"]
	if mobEntity.AI.Threat != "player" {
		t.Fatalf("tribulation monster failed to switch threat to player, threat=%s", mobEntity.AI.Threat)
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

func TestNoRespawnMonsterStaysDead(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	monster := newAIMonster("one-shot", Vec2{X: 5, Y: 5})
	monster.AI.NoRespawn = true
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	result, err := w.QueueDamage("one-shot", 999)
	if err != nil {
		t.Fatal(err)
	}
	w.Step()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 600; i++ {
		w.Step()
	}
	if hp := entityHP(t, w, "one-shot"); hp != 0 {
		t.Fatalf("no-respawn monster revived: hp=%v", hp)
	}
}

func TestMonsterSpawnsActivateOnlyNearPlayersAndShareAcrossPlayers(t *testing.T) {
	w, err := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "spawn-radius", RulesetVersion: "v1"}, 8, 13)
	if err != nil {
		t.Fatal(err)
	}
	near := newAIMonster("m:0", Vec2{X: 64, Y: 0})
	far := newAIMonster("m:1", Vec2{X: 100, Y: 0})
	if err := w.ConfigureMonsterSpawns([]Entity{near, far}); err != nil {
		t.Fatal(err)
	}
	first := newPlayer("first")
	first.Transform.Position = Vec2{X: 0, Y: 0}
	second := newPlayer("second")
	second.Transform.Position = Vec2{X: 20, Y: 0}
	if err := w.AddEntities([]Entity{first, second}); err != nil {
		t.Fatal(err)
	}
	w.Step()
	if !containsEntity(w.Snapshot(), "m:0") || containsEntity(w.Snapshot(), "m:1") {
		t.Fatalf("spawn radius activation mismatch: %+v", w.Snapshot().Entities)
	}

	w.stateMu.Lock()
	w.entities["first"].Transform.Position = Vec2{X: 500, Y: 500}
	w.stateMu.Unlock()
	w.Step()
	if !containsEntity(w.Snapshot(), "m:0") {
		t.Fatal("monster unloaded while a second player remained nearby")
	}
}

func TestMonsterSpawnDespawnsAfterGraceAndResetsWhenReactivated(t *testing.T) {
	w, err := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "spawn-grace", RulesetVersion: "v1"}, 8, 17)
	if err != nil {
		t.Fatal(err)
	}
	monster := newAIMonster("m:0", Vec2{X: 0, Y: 0})
	if err := w.ConfigureMonsterSpawns([]Entity{monster}); err != nil {
		t.Fatal(err)
	}
	player := newPlayer("player")
	player.Transform.Position = Vec2{X: 0, Y: 0}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	w.Step()
	if !containsEntity(w.Snapshot(), "m:0") {
		t.Fatal("nearby monster did not activate")
	}
	w.stateMu.Lock()
	w.entities["m:0"].Vitals.HP = 1
	w.entities["player"].Transform.Position = Vec2{X: 200, Y: 0}
	w.stateMu.Unlock()

	for i := uint64(0); i < MonsterDespawnGraceTicks+1; i++ {
		w.Step()
	}
	if containsEntity(w.Snapshot(), "m:0") {
		t.Fatal("monster remained active after grace period with no nearby player")
	}

	w.stateMu.Lock()
	w.entities["player"].Transform.Position = Vec2{X: 0, Y: 0}
	w.stateMu.Unlock()
	w.Step()
	if !containsEntity(w.Snapshot(), "m:0") {
		t.Fatal("monster did not reactivate when player returned")
	}
	if hp := entityHP(t, w, "m:0"); hp != monster.Vitals.MaxHP {
		t.Fatalf("reactivated monster HP=%v, want reset HP=%v", hp, monster.Vitals.MaxHP)
	}
}

func TestMonsterSpawnKeepsRespawnCooldownWhileDormant(t *testing.T) {
	w, err := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "spawn-cooldown", RulesetVersion: "v1"}, 8, 19)
	if err != nil {
		t.Fatal(err)
	}
	monster := newAIMonster("m:0", Vec2{X: 0, Y: 0})
	monster.AI.RespawnTicks = MonsterDespawnGraceTicks + 100
	if err := w.ConfigureMonsterSpawns([]Entity{monster}); err != nil {
		t.Fatal(err)
	}
	player := newPlayer("player")
	player.Transform.Position = Vec2{X: 0, Y: 0}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	w.Step()
	result, err := w.QueueDamage("m:0", 999)
	if err != nil {
		t.Fatal(err)
	}
	w.Step()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	w.stateMu.Lock()
	w.entities["player"].Transform.Position = Vec2{X: 200, Y: 0}
	w.stateMu.Unlock()
	for i := uint64(0); i < MonsterDespawnGraceTicks+1; i++ {
		w.Step()
	}
	if containsEntity(w.Snapshot(), "m:0") {
		t.Fatal("dormant dead monster should not have materialized")
	}
	w.stateMu.Lock()
	w.entities["player"].Transform.Position = Vec2{X: 0, Y: 0}
	w.stateMu.Unlock()
	w.Step()
	if containsEntity(w.Snapshot(), "m:0") {
		t.Fatal("monster reappeared before its original respawn cooldown")
	}
	for i := 0; i < 100; i++ {
		w.Step()
	}
	if !containsEntity(w.Snapshot(), "m:0") {
		t.Fatal("monster did not materialize after its respawn cooldown")
	}
}

func TestMonsterSpawnCannotUnloadWhileEngaged(t *testing.T) {
	w, err := NewWorldForKeyWithSeed(ZoneKey{MapID: "arena", InstanceID: "spawn-engaged", RulesetVersion: "v1"}, 8, 23)
	if err != nil {
		t.Fatal(err)
	}
	monster := newAIMonster("m:0", Vec2{X: 0, Y: 0})
	monster.AI.LeashRadius = 120
	if err := w.ConfigureMonsterSpawns([]Entity{monster}); err != nil {
		t.Fatal(err)
	}
	player := newPlayer("player")
	player.Transform.Position = Vec2{X: 0, Y: 0}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	w.Step()
	w.stateMu.Lock()
	w.entities["m:0"].AI.Threat = "player"
	w.entities["m:0"].AI.ThreatTable = map[EntityID]float64{"player": 1}
	w.entities["player"].Transform.Position = Vec2{X: 100, Y: 0}
	w.stateMu.Unlock()
	for i := uint64(0); i < MonsterDespawnGraceTicks+5; i++ {
		w.Step()
	}
	if !containsEntity(w.Snapshot(), "m:0") {
		t.Fatal("engaged monster unloaded before disengaging")
	}
}

func containsEntity(snapshot Snapshot, id EntityID) bool {
	for _, entity := range snapshot.Entities {
		if entity.ID == id {
			return true
		}
	}
	return false
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
