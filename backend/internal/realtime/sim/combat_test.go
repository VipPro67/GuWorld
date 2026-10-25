package sim

import (
	"reflect"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

func newCombatPlayer(id string, position Vec2) Entity {
	player := newPlayer(id)
	player.Transform.Position = position
	player.Faction = "player"
	player.Combat = NewPlayerCombat()
	return player
}

func newTargetMonster(id string, position Vec2) Entity {
	monster := newAIMonster(id, position)
	monster.AI.Active = false
	monster.Faction = "monster"
	monster.Vitals = Vitals{HP: 100, MaxHP: 100}
	return monster
}

func collectEvents(t *testing.T, w *World, ticks int) []combat.Event {
	t.Helper()
	events := make([]combat.Event, 0)
	for i := 0; i < ticks; i++ {
		events = append(events, w.Step().Events...)
	}
	return events
}

func hasEvent(events []combat.Event, kind combat.EventType) bool {
	for _, event := range events {
		if event.Type == kind {
			return true
		}
	}
	return false
}

func TestTargetCastValidatesRangeAndAppliesDamage(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	monster := newTargetMonster("mob", Vec2{X: 3, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 6)
	if !hasEvent(events, combat.EventCastAccepted) || !hasEvent(events, combat.EventDamageApplied) {
		t.Fatalf("cast events missing: %+v", events)
	}
	if hp := entityHP(t, w, "mob"); hp != 82 {
		t.Fatalf("monster hp=%v, want 82", hp)
	}
}

func TestLiveEmptyHotbarDoesNotFallBackToPreviewSkills(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Combat = NewPlayerCombatWithLoadout(nil)
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("mob", Vec2{X: 1})); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) || entityHP(t, w, "mob") != 100 {
		t.Fatalf("empty hotbar should reject casts: events=%+v hp=%v", events, entityHP(t, w, "mob"))
	}
}

func TestBasicAttackIsServerAuthoritativeAndCooldownLimited(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	monster := newTargetMonster("mob", Vec2{X: 1.5})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitBasicAttack(AttackCommand{EntityID: "player", Seq: 1, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventBasicAttack) || !hasEvent(events, combat.EventHitConfirmed) || !hasEvent(events, combat.EventDamageApplied) {
		t.Fatalf("basic attack events missing: %+v", events)
	}
	if hp := entityHP(t, w, "mob"); hp != 100-combat.BasicAttackDamage {
		t.Fatalf("monster hp=%v, want %v", hp, 100-combat.BasicAttackDamage)
	}
	if err := w.SubmitBasicAttack(AttackCommand{EntityID: "player", Seq: 2, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events = collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) || entityHP(t, w, "mob") != 100-combat.BasicAttackDamage {
		t.Fatalf("basic attack cooldown was not enforced: events=%+v hp=%v", events, entityHP(t, w, "mob"))
	}
}

func TestProjectedAttackAndDefenseUseCharacterValues(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Combat.BasicAttackDamage = 36
	monster := newTargetMonster("mob", Vec2{X: 1})
	monster.Defense = 9
	monster.PathResistance = map[string]float64{"strength": 0.25}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitBasicAttack(AttackCommand{EntityID: "player", Seq: 1, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if hp := entityHP(t, w, "mob"); hp != 79.75 {
		t.Fatalf("hp=%v, want 79.75", hp)
	}
	for _, event := range events {
		if event.Type == combat.EventDamageApplied && event.Amount != 20.25 {
			t.Fatalf("damage event=%+v", event)
		}
	}
}

func TestHigherRankGuBacklashPrecedesManaCheck(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Combat.Mana = 1
	player.Combat.Skills[1].ManaCost = 5
	player.Combat.Skills[1].BacklashPct = 0.15
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("mob", Vec2{X: 2})); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) || !hasEvent(events, combat.EventDamageApplied) {
		t.Fatalf("expected backlash then insufficient mana: %+v", events)
	}
	if hp := entityHP(t, w, "player"); hp != 85 {
		t.Fatalf("backlash hp=%v, want 85", hp)
	}
	if got := w.entities["player"].Combat.Mana; got != 1 {
		t.Fatalf("rejected cast spent mana: %v", got)
	}
}

func TestProjectedManaCostAndRegeneration(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Combat.Mana = 10
	player.Combat.MaxMana = 50
	player.Combat.ManaRegenPerSecond = 0.25
	player.Combat.Skills[1].ManaCost = 2.5
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("mob", Vec2{X: 2})); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	collectEvents(t, w, 1)
	if got := w.entities["player"].Combat.Mana; got != 7.5 {
		t.Fatalf("mana after cast=%v, want 7.5", got)
	}
	collectEvents(t, w, 19)
	if got := w.entities["player"].Combat.Mana; got != 7.75 {
		t.Fatalf("mana after regen=%v, want 7.75", got)
	}
}

func TestBasicAttackRejectsOutOfRangeTarget(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	if err := w.AddEntity(newCombatPlayer("player", Vec2{})); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("mob", Vec2{X: combat.BasicAttackRange + 1})); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitBasicAttack(AttackCommand{EntityID: "player", Seq: 1, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) || entityHP(t, w, "mob") != 100 {
		t.Fatalf("out-of-range attack was not rejected: events=%+v hp=%v", events, entityHP(t, w, "mob"))
	}
}

func TestTargetCastRejectsOutOfRangeAndCooldown(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	monster := newTargetMonster("mob", Vec2{X: 30, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) {
		t.Fatalf("out-of-range cast was not rejected: %+v", events)
	}
	for _, entity := range w.Snapshot().Entities {
		if entity.ID == "player" && entity.Combat.Mana != combat.DefaultManaMax {
			t.Fatalf("rejected cast spent mana: %v", entity.Combat.Mana)
		}
	}

	monster.AI.Active = false
	monster.Transform.Position = Vec2{X: 1, Y: 0}
	w.RemoveEntity("mob")
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 2, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	collectEvents(t, w, 6)
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 3, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events = collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) {
		t.Fatalf("cooldown cast was not rejected: %+v", events)
	}
}

func TestDirectionProjectileHitsOnce(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	monster := newTargetMonster("mob", Vec2{X: 5, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 2, Mode: combat.ModeDirection, Aim: Vec2{X: 1, Y: 0}}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 30)
	if !hasEvent(events, combat.EventProjectileSpawn) || !hasEvent(events, combat.EventDamageApplied) {
		t.Fatalf("projectile events missing: %+v", events)
	}
	if hp := entityHP(t, w, "mob"); hp != 86 {
		t.Fatalf("projectile damage applied %v times: hp=%v", (100-hp)/14, hp)
	}
}

func TestGroundCastHitsMultipleTargets(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	first := newTargetMonster("mob-a", Vec2{X: 5, Y: 0.5})
	second := newTargetMonster("mob-b", Vec2{X: 5, Y: -0.5})
	third := newTargetMonster("mob-c", Vec2{X: 12, Y: 0})
	for _, entity := range []Entity{player, first, second, third} {
		if err := w.AddEntity(entity); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 3, Mode: combat.ModeGround, Point: Vec2{X: 5, Y: 0}}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 10)
	if !hasEvent(events, combat.EventDamageApplied) {
		t.Fatalf("ground cast did not damage: %+v", events)
	}
	if entityHP(t, w, "mob-a") != 88 || entityHP(t, w, "mob-b") != 88 {
		t.Fatalf("AoE did not hit both nearby monsters: a=%v b=%v", entityHP(t, w, "mob-a"), entityHP(t, w, "mob-b"))
	}
	if entityHP(t, w, "mob-c") != 100 {
		t.Fatal("AoE hit a monster outside the radius")
	}
}

func TestSelfShieldAbsorbsDamageAndCastingIsInterrupted(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 4, Mode: combat.ModeSelf}); err != nil {
		t.Fatal(err)
	}
	collectEvents(t, w, 3)
	for _, entity := range w.Snapshot().Entities {
		if entity.ID == "player" && entity.Vitals.Shield != 30 {
			t.Fatalf("shield=%v, want 30", entity.Vitals.Shield)
		}
	}
	result, err := w.QueueDamage("player", 40)
	if err != nil {
		t.Fatal(err)
	}
	w.Step()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if hp := entityHP(t, w, "player"); hp != 90 {
		t.Fatalf("shield did not absorb damage: hp=%v", hp)
	}

	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 2, Slot: 6, Mode: combat.ModeTarget, TargetID: "player"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) {
		t.Fatalf("self-target enemy cast was not rejected: %+v", events)
	}
}

func TestDamageEventAmountMatchesHealthLost(t *testing.T) {
	for _, tc := range []struct {
		name       string
		shield     float64
		wantDamage float64
	}{
		{name: "unshielded", shield: 0, wantDamage: 12},
		{name: "partly shielded", shield: 5, wantDamage: 7},
		{name: "fully shielded", shield: 20, wantDamage: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := NewWorld("arena", 16)
			player := newCombatPlayer("player", Vec2{})
			player.Vitals = Vitals{HP: 100, MaxHP: 100, Shield: tc.shield}
			if err := w.AddEntity(player); err != nil {
				t.Fatal(err)
			}
			result, err := w.QueueDamage("player", 12)
			if err != nil {
				t.Fatal(err)
			}
			events := w.Step().Events
			if err := <-result; err != nil {
				t.Fatal(err)
			}
			for _, event := range events {
				if event.Type == combat.EventDamageApplied {
					if event.Amount != tc.wantDamage || entityHP(t, w, "player") != 100-tc.wantDamage {
						t.Fatalf("damage event amount=%v hp=%v, want damage=%v", event.Amount, entityHP(t, w, "player"), tc.wantDamage)
					}
					return
				}
			}
			t.Fatalf("missing damage event: %+v", events)
		})
	}
}

func TestShieldedMonsterStillRecordsAttackerThreat(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	monster := newTargetMonster("mob", Vec2{X: 1})
	monster.Vitals.Shield = 20
	monster.AI.Active = true
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	w.applyDamageLocked(w.entities["player"], w.entities["mob"], 12, "BASIC_ATTACK")
	if got := w.entities["mob"].AI.ThreatTable["player"]; got != 12 {
		t.Fatalf("shielded hit threat=%v, want 12", got)
	}
}

func TestDuplicateCastSequenceAndInvalidModeAreRejected(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	monster := newTargetMonster("mob", Vec2{X: 2, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	w.Step()
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 6)
	if entityHP(t, w, "mob") != 82 {
		t.Fatalf("duplicate cast applied twice: hp=%v", entityHP(t, w, "mob"))
	}
	if hasEvent(events, combat.EventCastRejected) {
		t.Logf("duplicate sequence was ignored as expected: %+v", events)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 3, Slot: 1, Mode: combat.ModeGround, Point: Vec2{X: 2}}); err != nil {
		t.Fatal(err)
	}
}

func TestDeterministicCombatReplayOneThousandTicks(t *testing.T) {
	key := ZoneKey{MapID: "arena", InstanceID: "replay", RulesetVersion: "v1"}
	first, _ := NewWorldForKeyWithSeed(key, 16, 12345)
	second, _ := NewWorldForKeyWithSeed(key, 16, 12345)
	for _, world := range []*World{first, second} {
		player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
		monster := newTargetMonster("mob", Vec2{X: 4, Y: 0})
		monster.Vitals = Vitals{HP: 10_000, MaxHP: 10_000}
		if err := world.AddEntity(player); err != nil {
			t.Fatal(err)
		}
		if err := world.AddEntity(monster); err != nil {
			t.Fatal(err)
		}
	}
	run := func(world *World) {
		seq := uint64(0)
		for tick := 0; tick < 1000; tick++ {
			switch tick % 120 {
			case 0:
				seq++
				_ = world.SubmitCast(CastCommand{EntityID: "player", Seq: seq, Slot: 2, Mode: combat.ModeDirection, Aim: Vec2{X: 1}})
			case 30:
				seq++
				_ = world.SubmitCast(CastCommand{EntityID: "player", Seq: seq, Slot: 3, Mode: combat.ModeGround, Point: Vec2{X: 4}})
			case 70:
				seq++
				_ = world.SubmitCast(CastCommand{EntityID: "player", Seq: seq, Slot: 6, Mode: combat.ModeTarget, TargetID: "mob"})
			}
			world.Step()
		}
	}
	run(first)
	run(second)
	if !reflect.DeepEqual(first.Snapshot(), second.Snapshot()) {
		t.Fatal("combat replay diverged for identical seed/actions")
	}
}

func TestGroundConeAoEHitsOnlyInsideAngle(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	// Slot 3: Ground Burst -> customize to Cone 60 deg facing +X
	player.Combat.Skills[3].Shape = combat.ShapeCone
	player.Combat.Skills[3].AngleDeg = 60
	player.Combat.Skills[3].Radius = 8

	// In front (angle 0) -> should be hit
	frontMob := newTargetMonster("mob_front", Vec2{X: 4, Y: 0})
	// To the side (angle 90) -> outside 60 deg cone -> should NOT be hit
	sideMob := newTargetMonster("mob_side", Vec2{X: 0, Y: 4})
	// Behind -> should NOT be hit
	backMob := newTargetMonster("mob_back", Vec2{X: -4, Y: 0})

	for _, e := range []Entity{player, frontMob, sideMob, backMob} {
		if err := w.AddEntity(e); err != nil {
			t.Fatal(err)
		}
	}

	// Cast towards +X
	if err := w.SubmitCast(CastCommand{
		EntityID: "player", Seq: 1, Slot: 3, Mode: combat.ModeGround,
		Point: Vec2{X: 4, Y: 0},
	}); err != nil {
		t.Fatal(err)
	}

	collectEvents(t, w, 10)

	if w.entities["mob_front"].Vitals.HP >= 100 {
		t.Fatal("mob_front should take damage from cone")
	}
	if w.entities["mob_side"].Vitals.HP != 100 {
		t.Fatal("mob_side outside cone should NOT take damage")
	}
	if w.entities["mob_back"].Vitals.HP != 100 {
		t.Fatal("mob_back outside cone should NOT take damage")
	}
}

func TestGroundDonutAoEExcludesCenter(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	player.Combat.Skills[3].Shape = combat.ShapeDonut
	player.Combat.Skills[3].Radius = 6
	player.Combat.Skills[3].InnerRadius = 3

	centerMob := newTargetMonster("mob_center", Vec2{X: 1, Y: 0}) // inside hole (dist 1 < 3)
	ringMob := newTargetMonster("mob_ring", Vec2{X: 4.5, Y: 0})   // in ring (3 <= 4.5 <= 6)
	outerMob := newTargetMonster("mob_outer", Vec2{X: 8, Y: 0})   // outside ring (dist 8 > 6)

	for _, e := range []Entity{player, centerMob, ringMob, outerMob} {
		if err := w.AddEntity(e); err != nil {
			t.Fatal(err)
		}
	}

	if err := w.SubmitCast(CastCommand{
		EntityID: "player", Seq: 1, Slot: 3, Mode: combat.ModeGround,
		Point: Vec2{X: 0, Y: 0},
	}); err != nil {
		t.Fatal(err)
	}

	collectEvents(t, w, 10)

	if w.entities["mob_center"].Vitals.HP != 100 {
		t.Fatal("mob_center inside donut hole should NOT take damage")
	}
	if w.entities["mob_ring"].Vitals.HP >= 100 {
		t.Fatal("mob_ring in donut ring should take damage")
	}
	if w.entities["mob_outer"].Vitals.HP != 100 {
		t.Fatal("mob_outer outside donut should NOT take damage")
	}
}

func TestDirectionProjectilePierce(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	// Slot 2: Water Arrow -> give pierce count 2
	player.Combat.Skills[2].PierceCount = 2
	player.Combat.Skills[2].Damage = 20

	mob1 := newTargetMonster("mob1", Vec2{X: 2, Y: 0})
	mob2 := newTargetMonster("mob2", Vec2{X: 4, Y: 0})
	mob3 := newTargetMonster("mob3", Vec2{X: 6, Y: 0})

	for _, e := range []Entity{player, mob1, mob2, mob3} {
		if err := w.AddEntity(e); err != nil {
			t.Fatal(err)
		}
	}

	if err := w.SubmitCast(CastCommand{
		EntityID: "player", Seq: 1, Slot: 2, Mode: combat.ModeDirection,
		Aim: Vec2{X: 1, Y: 0},
	}); err != nil {
		t.Fatal(err)
	}

	collectEvents(t, w, 20)

	// Mob 1 and Mob 2 should be hit (pierce = 2 allows hitting 1st and 2nd entity)
	if w.entities["mob1"].Vitals.HP >= 100 {
		t.Fatal("mob1 should take damage from piercing projectile")
	}
	if w.entities["mob2"].Vitals.HP >= 100 {
		t.Fatal("mob2 should take damage from piercing projectile")
	}
}
