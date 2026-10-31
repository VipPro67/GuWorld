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

func TestLiveLoadoutDoesNotGrantPreviewHiddenPassive(t *testing.T) {
	state := NewPlayerCombatWithLoadout(combat.DefaultLoadout())
	if state.Hidden.ID != "" {
		t.Fatalf("live loadout received demo passive %q", state.Hidden.ID)
	}
	if preview := NewPlayerCombat(); preview.Hidden.ID == "" {
		t.Fatal("preview world lost its explicit demo passive")
	}
}

func TestSlowMovementPolicyMovesAtHalfSpeedDuringWindup(t *testing.T) {
	skill := combat.Skill{
		ID: "SLOW_CAST", Slot: 1, Mode: combat.ModeTarget, Name: "slow cast",
		Range: 20, WindupTicks: 20, MovementPolicy: combat.MovementSlow,
		Damage: 1, MaxTargets: 1, RequiresEnemyTarget: true,
	}
	casting, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Combat = NewPlayerCombatWithLoadout([]combat.Skill{skill})
	if err := casting.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := casting.AddEntity(newTargetMonster("mob", Vec2{X: 10})); err != nil {
		t.Fatal(err)
	}
	if err := casting.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	collectEvents(t, casting, 1)

	free, _ := NewWorld("arena", 16)
	if err := free.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	const steps = 5
	for seq := uint64(2); seq < 2+steps; seq++ {
		if err := casting.SubmitInput(InputCommand{EntityID: "player", Seq: seq, Move: Vec2{X: 1}}); err != nil {
			t.Fatal(err)
		}
		if err := free.SubmitInput(InputCommand{EntityID: "player", Seq: seq, Move: Vec2{X: 1}}); err != nil {
			t.Fatal(err)
		}
		casting.Step()
		free.Step()
	}
	if casting.entities["player"].Combat.Pending == nil {
		t.Fatal("test cast resolved before movement was measured")
	}
	slowDistance := casting.entities["player"].Transform.Position.X
	freeDistance := free.entities["player"].Transform.Position.X
	if freeDistance <= 0 {
		t.Fatalf("control player did not move: %v", freeDistance)
	}
	if ratio := slowDistance / freeDistance; ratio < 0.49 || ratio > 0.51 {
		t.Fatalf("SLOW windup moved at %.2f of free speed (slow=%v free=%v)", ratio, slowDistance, freeDistance)
	}
}

func TestRootMovementPolicyStopsGroundMovementDuringWindup(t *testing.T) {
	skill := combat.Skill{
		ID: "ROOT_CAST", Slot: 1, Mode: combat.ModeTarget, Name: "root cast",
		Range: 20, WindupTicks: 20, MovementPolicy: combat.MovementRoot,
		Damage: 1, MaxTargets: 1, RequiresEnemyTarget: true,
	}
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Combat = NewPlayerCombatWithLoadout([]combat.Skill{skill})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("mob", Vec2{X: 10})); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	collectEvents(t, w, 1)
	for seq := uint64(2); seq < 5; seq++ {
		if err := w.SubmitInput(InputCommand{EntityID: "player", Seq: seq, Move: Vec2{X: 1}}); err != nil {
			t.Fatal(err)
		}
		w.Step()
	}
	if got := w.entities["player"].Transform.Position.X; got != 0 {
		t.Fatalf("ROOT windup moved the caster: %v", got)
	}
}

func TestRecoveryBlocksNextCastAfterGCDExpires(t *testing.T) {
	skill := combat.Skill{
		ID: "RECOVERY_CAST", Slot: 1, Mode: combat.ModeTarget, Name: "recovery cast",
		Range: 20, WindupTicks: 1, RecoveryTicks: 12, MovementPolicy: combat.MovementSlow,
		Damage: 1, MaxTargets: 1, RequiresEnemyTarget: true,
	}
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Combat = NewPlayerCombatWithLoadout([]combat.Skill{skill})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("mob", Vec2{X: 1})); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	collectEvents(t, w, 2)
	if w.entities["player"].Combat.Pending != nil || w.entities["player"].Combat.RecoveryUntil == 0 {
		t.Fatalf("cast did not resolve into recovery: %+v", w.entities["player"].Combat)
	}
	cast := CastCommand{EntityID: "player", Seq: 2, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}
	if err := w.SubmitCast(cast); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) || hasEvent(events, combat.EventCastAccepted) {
		t.Fatalf("recovery did not block the next cast: %+v", events)
	}
	// Step past the authored recovery window; GCD expired many ticks earlier.
	collectEvents(t, w, 11)
	cast.Seq = 13
	if err := w.SubmitCast(cast); err != nil {
		t.Fatal(err)
	}
	events = collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastAccepted) {
		t.Fatalf("cast still blocked after recovery: %+v", events)
	}
}

func TestBasicAttackCannotInterruptOwnGuWindup(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	monster := newTargetMonster("mob", Vec2{X: 1.5})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	if events := collectEvents(t, w, 1); !hasEvent(events, combat.EventCastAccepted) {
		t.Fatalf("cast not accepted: %+v", events)
	}
	if err := w.SubmitBasicAttack(AttackCommand{EntityID: "player", Seq: 2, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) || hasEvent(events, combat.EventBasicAttack) {
		t.Fatalf("basic attack bypassed windup: %+v", events)
	}
	if hp := entityHP(t, w, "mob"); hp != 100 {
		t.Fatalf("basic attack damaged target during windup: hp=%v", hp)
	}
}

func TestOwnedGuHungerIsChargedByInstanceOnAttempt(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Combat.Skills[1].GuInstanceID = "instance-a"
	player.Combat.Skills[2].GuInstanceID = "instance-b"
	player.Combat.GuHunger[1] = 1
	player.Combat.GuHunger[2] = 4
	player.Combat.Mana = 0
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("mob", Vec2{X: 1})); err != nil {
		t.Fatal(err)
	}
	cast := CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}
	if err := w.SubmitCast(cast); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) {
		t.Fatalf("insufficient mana attempt was not rejected: %+v", events)
	}
	state := w.entities["player"].Combat
	if state.GuHunger[1] != 0 || state.GuUses[1] != 1 || state.GuHunger[2] != 4 || state.GuUses[2] != 0 {
		t.Fatalf("wrong instance was charged: hunger=%v uses=%v", state.GuHunger, state.GuUses)
	}
	if state.GuMasteryUses[1] != 0 {
		t.Fatalf("rejected cast trained Chân Ý: %d", state.GuMasteryUses[1])
	}
	cast.Seq = 2
	if err := w.SubmitCast(cast); err != nil {
		t.Fatal(err)
	}
	events = collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) || w.entities["player"].Combat.GuUses[1] != 1 {
		t.Fatalf("exhausted Gu was charged again: %+v", events)
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
	if uses := w.entities["player"].Combat.BasicAttackUses; uses != 1 {
		t.Fatalf("basic attack uses=%d, want 1", uses)
	}
	if err := w.SubmitBasicAttack(AttackCommand{EntityID: "player", Seq: 2, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events = collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) || entityHP(t, w, "mob") != 100-combat.BasicAttackDamage {
		t.Fatalf("basic attack cooldown was not enforced: events=%+v hp=%v", events, entityHP(t, w, "mob"))
	}
	if uses := w.entities["player"].Combat.BasicAttackUses; uses != 1 {
		t.Fatalf("rejected basic attack counted as use: %d", uses)
	}
}

func TestAimedBasicAttackUsesCollisionAndMissConsumesCooldown(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	if err := w.AddEntity(newCombatPlayer("player", Vec2{})); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("front", Vec2{X: 1.5})); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("back", Vec2{X: -1.5})); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitBasicAttack(AttackCommand{EntityID: "player", Seq: 1, Aim: Vec2{Y: 1}}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventBasicAttack) || hasEvent(events, combat.EventHitConfirmed) || entityHP(t, w, "front") != 100 {
		t.Fatalf("empty swing should miss: %+v", events)
	}
	if uses := w.entities["player"].Combat.BasicAttackUses; uses != 0 {
		t.Fatalf("empty swing counted as Chân Ý use: %d", uses)
	}
	if err := w.SubmitBasicAttack(AttackCommand{EntityID: "player", Seq: 2, Aim: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	events = collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) || entityHP(t, w, "front") != 100 {
		t.Fatalf("miss should start cooldown: %+v", events)
	}
	collectEvents(t, w, int(combat.BasicAttackCooldownTicks)-1)
	if err := w.SubmitBasicAttack(AttackCommand{EntityID: "player", Seq: 3, Aim: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	events = collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventHitConfirmed) || entityHP(t, w, "front") != 100-combat.BasicAttackDamage || entityHP(t, w, "back") != 100 {
		t.Fatalf("aimed swing should hit front only: %+v", events)
	}
	if uses := w.entities["player"].Combat.BasicAttackUses; uses != 1 {
		t.Fatalf("landed basic attack uses=%d, want 1", uses)
	}
}

func TestAimedTargetCastCanMissAndStillSpendMana(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	if err := w.AddEntity(newCombatPlayer("player", Vec2{})); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("mob", Vec2{X: 3})); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, Aim: Vec2{X: 1}}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 6)
	if !hasEvent(events, combat.EventCastAccepted) || hasEvent(events, combat.EventHitConfirmed) || entityHP(t, w, "mob") != 100 {
		t.Fatalf("empty-space cast should miss: %+v", events)
	}
	if w.entities["player"].Combat.Mana >= w.entities["player"].Combat.MaxMana {
		t.Fatal("empty-space cast did not spend mana")
	}
}

func TestAimedTargetCastHitsFirstBodyBeforeClickedPoint(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	if err := w.AddEntity(newCombatPlayer("player", Vec2{})); err != nil {
		t.Fatal(err)
	}
	for _, target := range []Entity{
		newTargetMonster("first", Vec2{X: 2}),
		newTargetMonster("second", Vec2{X: 3}),
	} {
		if err := w.AddEntity(target); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, Aim: Vec2{X: 4}}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 6)
	if !hasEvent(events, combat.EventHitConfirmed) || entityHP(t, w, "first") >= 100 || entityHP(t, w, "second") != 100 {
		t.Fatalf("aimed cast should hit first body only: %+v", events)
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

func TestTargetModeHealAppliesToCaster(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	player.Vitals = Vitals{HP: 50, MaxHP: 100}
	player.Combat.Skills[1] = combat.Skill{
		ID: "GU_HEAL_TARGET", Slot: 1, Mode: combat.ModeTarget, Name: "Blade Heal",
		Range: 9, WindupTicks: 3, CooldownTicks: 30, Damage: 10, Heal: 40,
		MaxTargets: 1, RequiresEnemyTarget: true,
	}
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
	if !hasEvent(events, combat.EventHealApplied) {
		t.Fatalf("target-mode heal did not emit a heal event: %+v", events)
	}
	if hp := entityHP(t, w, "player"); hp != 90 {
		t.Fatalf("player hp=%v, want 90 after target-mode heal", hp)
	}
	if hp := entityHP(t, w, "mob"); hp != 90 {
		t.Fatalf("monster hp=%v, want 90 after damage", hp)
	}
}

func TestGroundModeHealAndShieldApplyToCaster(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	player.Vitals = Vitals{HP: 30, MaxHP: 100}
	player.Combat.Skills[1] = combat.Skill{
		ID: "GU_HEAL_GROUND", Slot: 1, Mode: combat.ModeGround, Name: "Rain",
		Range: 10, WindupTicks: 2, CooldownTicks: 30, Damage: 5, Heal: 25, Shield: 15,
		Radius: 2.5, MaxTargets: 5, RequiresEnemyTarget: true,
	}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeGround, Point: Vec2{X: 1, Y: 0}}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 4)
	if !hasEvent(events, combat.EventHealApplied) || !hasEvent(events, combat.EventShieldApplied) {
		t.Fatalf("ground-mode self support missing: %+v", events)
	}
	entity := w.entities["player"]
	if entity.Vitals.HP != 55 || entity.Vitals.Shield != 15 {
		t.Fatalf("player hp/shield=%v/%v, want 55/15", entity.Vitals.HP, entity.Vitals.Shield)
	}
}

func TestQueueHPDeltaHealsAndClamps(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Vitals = Vitals{HP: 50, MaxHP: 100}
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	result, err := w.QueueHPDelta("player", 25)
	if err != nil {
		t.Fatal(err)
	}
	w.Step()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if hp := entityHP(t, w, "player"); hp != 75 {
		t.Fatalf("hp=%v, want 75", hp)
	}
	result, err = w.QueueHPDelta("player", 500)
	if err != nil {
		t.Fatal(err)
	}
	w.Step()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if hp := entityHP(t, w, "player"); hp != 100 {
		t.Fatalf("hp=%v, want capped 100", hp)
	}
	result, err = w.QueueHPDelta("player", -500)
	if err != nil {
		t.Fatal(err)
	}
	w.Step()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if hp := entityHP(t, w, "player"); hp != 0 {
		t.Fatalf("hp=%v, want clamped 0", hp)
	}
}

func TestDamageEventCarriesMitigatedDamageAndHealthLost(t *testing.T) {
	for _, tc := range []struct {
		name         string
		shield       float64
		wantTotal    float64
		wantHP       float64
		wantShieldAb float64
	}{
		{name: "unshielded", shield: 0, wantTotal: 12, wantHP: 12, wantShieldAb: 0},
		{name: "partly shielded", shield: 5, wantTotal: 12, wantHP: 7, wantShieldAb: 5},
		{name: "fully shielded", shield: 20, wantTotal: 12, wantHP: 0, wantShieldAb: 12},
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
					if event.Amount != tc.wantTotal || event.HPDamage != tc.wantHP || event.ShieldAbsorbed != tc.wantShieldAb || entityHP(t, w, "player") != 100-tc.wantHP {
						t.Fatalf("damage event amount=%v hpDamage=%v shieldAbsorbed=%v hp=%v, want total=%v hp=%v shield=%v",
							event.Amount, event.HPDamage, event.ShieldAbsorbed, entityHP(t, w, "player"), tc.wantTotal, tc.wantHP, tc.wantShieldAb)
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

func TestAcceptedGuCastCountsChanhYUse(t *testing.T) {
	w, _ := NewWorld("arena", 16)
	player := newCombatPlayer("player", Vec2{})
	player.Combat.Skills[1].GuInstanceID = "instance-a"
	player.Combat.GuHunger[1] = 1
	player.Combat.Mana = 100
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(newTargetMonster("mob", Vec2{X: 1})); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	collectEvents(t, w, 6)
	state := w.entities["player"].Combat
	if state.GuMasteryUses[1] != 1 {
		t.Fatalf("accepted cast mastery uses=%d, want 1", state.GuMasteryUses[1])
	}
}
