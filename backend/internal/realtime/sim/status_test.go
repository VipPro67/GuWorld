package sim

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

func testStatusSkill(id string, damagePerTick, healPerTick float64, duration, period uint64) combat.Skill {
	return combat.Skill{
		ID: "TEST_STATUS", Slot: 1, Mode: combat.ModeTarget, Name: "test",
		StatusID: id, StatusDurationTicks: duration, StatusPeriodTicks: period,
		StatusDamagePerTick: damagePerTick, StatusHealPerTick: healPerTick,
		StatusMoveMultiplier: 1, StatusMaxStacks: 3, StatusStackPolicy: combat.StackRefresh,
		StatusDebuff: damagePerTick > 0, PvPDrCategory: "SOFT_CC",
	}
}

func TestFreezeBlocksMovementAndCasting(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	monster := newTargetMonster("mob", Vec2{X: 3, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	freeze := testStatusSkill("FREEZE", 0, 0, 10, 0)
	freeze.StatusMoveMultiplier = 0
	freeze.StatusDebuff = true
	w.applyStatusLocked(worldEntity(t, w, "player"), worldEntity(t, w, "mob"), freeze)
	worldEntity(t, w, "mob").moveIntent = Vec2{X: 1, Y: 0}
	before := worldEntity(t, w, "mob").Transform.Position
	w.Step()
	if after := entityPosition(t, w, "mob"); after != before {
		t.Fatalf("frozen monster moved: before=%+v after=%+v", before, after)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	// Freeze the caster to verify cast rejection.
	w.applyStatusLocked(worldEntity(t, w, "mob"), worldEntity(t, w, "player"), freeze)
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 2, Slot: 1, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 1)
	if !hasEvent(events, combat.EventCastRejected) {
		t.Fatalf("stunned caster was not rejected: %+v", events)
	}
}

func TestDamageOverTimeAndHealOverTimeTickOnSchedule(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	monster := newTargetMonster("mob", Vec2{X: 3, Y: 0})
	monster.Vitals.HP = 100
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	dot := testStatusSkill("POISON", 4, 0, 6, 2)
	w.applyStatusLocked(worldEntity(t, w, "player"), worldEntity(t, w, "mob"), dot)
	collectEvents(t, w, 6)
	if hp := entityHP(t, w, "mob"); hp != 88 {
		t.Fatalf("DoT hp=%v, want 88", hp)
	}
	// Status should have expired after 6 ticks.
	w.Step()
	for _, entity := range w.Snapshot().Entities {
		if entity.ID == "mob" && len(entity.Combat.Statuses) != 0 {
			t.Fatalf("status did not expire: %+v", entity.Combat.Statuses)
		}
	}

	worldEntity(t, w, "mob").Vitals.HP = 50
	hot := testStatusSkill("REGEN", 0, 5, 6, 2)
	hot.StatusDebuff = false
	w.applyStatusLocked(worldEntity(t, w, "player"), worldEntity(t, w, "mob"), hot)
	collectEvents(t, w, 6)
	if hp := entityHP(t, w, "mob"); hp != 65 {
		t.Fatalf("HoT hp=%v, want 65", hp)
	}
}

func TestStatusStackPoliciesAndCleanse(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	monster := newTargetMonster("mob", Vec2{X: 3, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	stack := testStatusSkill("BLEED", 3, 0, 10, 2)
	stack.StatusStackPolicy = combat.StackStack
	stack.StatusMaxStacks = 3
	w.applyStatusLocked(worldEntity(t, w, "player"), worldEntity(t, w, "mob"), stack)
	w.applyStatusLocked(worldEntity(t, w, "player"), worldEntity(t, w, "mob"), stack)
	statuses := worldEntity(t, w, "mob").Combat.Statuses
	if len(statuses) != 1 || statuses[0].Stacks != 2 {
		t.Fatalf("stack policy produced %+v, want one status with 2 stacks", statuses)
	}
	buff := testStatusSkill("REGEN", 0, 2, 10, 2)
	buff.StatusDebuff = false
	w.applyStatusLocked(worldEntity(t, w, "player"), worldEntity(t, w, "mob"), buff)
	before := len(worldEntity(t, w, "mob").Combat.Statuses)
	if removed := w.cleanseDebuffsLocked(worldEntity(t, w, "mob")); removed != 1 {
		t.Fatalf("cleansed %d, want 1", removed)
	}
	if len(worldEntity(t, w, "mob").Combat.Statuses) != before-1 || !w.hasStatus(worldEntity(t, w, "mob"), "REGEN") {
		t.Fatalf("cleanse removed the wrong statuses: %+v", worldEntity(t, w, "mob").Combat.Statuses)
	}
}

func TestHiddenPassiveTriggersBelowThresholdWithICD(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	for _, amount := range []float64{30, 30, 5, 5} {
		result, err := w.QueueDamage("player", amount)
		if err != nil {
			t.Fatal(err)
		}
		events := w.Step().Events
		if err := <-result; err != nil {
			t.Fatal(err)
		}
		if amount == 5 && hasEvent(events, combat.EventHiddenPassive) {
			// Triggered after the third low-HP hit.
			break
		}
	}
	snapshot := w.Snapshot()
	var self Entity
	for _, entity := range snapshot.Entities {
		if entity.ID == "player" {
			self = entity
		}
	}
	if self.Combat.Hidden.Shield != 25 && self.Vitals.Shield != 25 {
		t.Fatalf("hidden passive did not grant shield: hidden=%+v vitals=%+v", self.Combat.Hidden, self.Vitals)
	}
	if !self.Combat.Hidden.Discovered {
		t.Fatal("hidden passive discovery not recorded")
	}
	nextReady := self.Combat.Hidden.NextReadyTick
	result, _ := w.QueueDamage("player", 5)
	w.Step()
	<-result
	for _, entity := range w.Snapshot().Entities {
		if entity.ID == "player" && entity.Combat.Hidden.NextReadyTick != nextReady {
			t.Fatalf("hidden passive ignored ICD: before=%d after=%d", nextReady, entity.Combat.Hidden.NextReadyTick)
		}
	}
}

func worldEntity(t *testing.T, w *World, id EntityID) *Entity {
	t.Helper()
	entity := w.entities[id]
	if entity == nil {
		t.Fatalf("entity %q missing from world", id)
	}
	return entity
}

func TestPreviewFrostCastAppliesFreezeStatus(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	monster := newTargetMonster("mob", Vec2{X: 3, Y: 0})
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	if err := w.SubmitCast(CastCommand{EntityID: "player", Seq: 1, Slot: 6, Mode: combat.ModeTarget, TargetID: "mob"}); err != nil {
		t.Fatal(err)
	}
	events := collectEvents(t, w, 6)
	if !hasEvent(events, combat.EventStatusApplied) {
		t.Fatalf("frost cast did not apply a status: %+v", events)
	}
	target := worldEntity(t, w, "mob")
	if !w.hasStatus(target, "FREEZE") || w.moveMultiplier(target) != 0 {
		t.Fatalf("frost did not freeze the target: %+v", target.Combat.Statuses)
	}
}

func TestPvPDurationDRCategories(t *testing.T) {
	if got := applyPvPDurationDR(30, "HARD_CC"); got != 18 {
		t.Fatalf("hard CC DR=%d, want 18", got)
	}
	if got := applyPvPDurationDR(30, "SOFT_CC"); got != 23 {
		t.Fatalf("soft CC DR=%d, want 23", got)
	}
	if got := applyPvPDurationDR(30, ""); got != 30 {
		t.Fatalf("unclassified DR=%d, want 30", got)
	}
}

func TestStatusUpkeepDrainAndAutoExpire(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	player.Combat.Mana = 10
	player.Combat.MaxMana = 100
	player.Combat.ManaRegenPerSecond = 0
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	drainSkill := testStatusSkill("UPKEEP_BUFF", 0, 0, 100, 5)
	drainSkill.ManaDrainPerTick = 4
	drainSkill.StatusDebuff = false
	pEntity := worldEntity(t, w, "player")
	w.applyStatusLocked(pEntity, pEntity, drainSkill)

	if !w.hasStatus(pEntity, "UPKEEP_BUFF") {
		t.Fatal("expected UPKEEP_BUFF to be active")
	}

	// Step 5 ticks: first period fires, mana drains 4 -> 6
	for i := 0; i < 5; i++ {
		w.Step()
	}
	if pEntity.Combat.Mana != 6 {
		t.Fatalf("mana after 5 ticks = %v, want 6", pEntity.Combat.Mana)
	}

	// Step 5 more ticks (tick 10): second period fires, mana drains 4 -> 2
	for i := 0; i < 5; i++ {
		w.Step()
	}
	if pEntity.Combat.Mana != 2 {
		t.Fatalf("mana after 10 ticks = %v, want 2", pEntity.Combat.Mana)
	}

	// Step 5 more ticks (tick 15): third period fires, mana is 2 < 4 -> expired!
	for i := 0; i < 5; i++ {
		w.Step()
	}
	if w.hasStatus(pEntity, "UPKEEP_BUFF") {
		t.Fatalf("expected UPKEEP_BUFF to expire due to insufficient mana, statuses: %+v", pEntity.Combat.Statuses)
	}
}

func TestStatusHasteAndFlightAndStealth(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	monster := newTargetMonster("mob", Vec2{X: 2, Y: 0})
	monster.AI.AggroRadius = 10
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(monster); err != nil {
		t.Fatal(err)
	}
	pEntity := worldEntity(t, w, "player")
	mEntity := worldEntity(t, w, "mob")

	// 1. HASTE boosts moveMultiplier
	hasteSkill := testStatusSkill("HASTE", 0, 0, 50, 0)
	hasteSkill.StatusMoveMultiplier = 1.35
	hasteSkill.StatusDebuff = false
	w.applyStatusLocked(pEntity, pEntity, hasteSkill)
	if mult := w.moveMultiplier(pEntity); mult != 1.35 {
		t.Fatalf("moveMultiplier with HASTE = %v, want 1.35", mult)
	}

	// 2. STEALTH: Monster will not target player
	stealthSkill := testStatusSkill("STEALTH", 0, 0, 50, 0)
	stealthSkill.StatusDebuff = false
	w.applyStatusLocked(pEntity, pEntity, stealthSkill)
	if !w.hasStatus(pEntity, "STEALTH") {
		t.Fatal("expected STEALTH status")
	}

	w.thinkMonsterLocked(mEntity)
	if mEntity.AI.Threat == pEntity.ID {
		t.Fatalf("monster targeted stealthed player")
	}

	// 3. Breaking stealth on basic attack
	w.startBasicAttackLocked(pEntity, AttackCommand{EntityID: "player", Seq: 1, TargetID: "mob"})
	if w.hasStatus(pEntity, "STEALTH") {
		t.Fatalf("basic attack did not break stealth")
	}

	// 4. FLYING: immune to Ground AoE
	flySkill := testStatusSkill("FLYING", 0, 0, 50, 0)
	flySkill.StatusDebuff = false
	w.applyStatusLocked(pEntity, pEntity, flySkill)

	groundSkill := combat.Skill{
		ID: "EARTHQUAKE", Slot: 3, Mode: combat.ModeGround, Name: "Quake",
		Range: 10, Radius: 5, Damage: 50,
	}
	pEntity.Vitals.HP = 100
	mEntity.Combat.Pending = &PendingCast{
		Skill: groundSkill, Point: Vec2{X: 0, Y: 0},
	}
	w.resolveCastLocked(mEntity)
	if pEntity.Vitals.HP != 100 {
		t.Fatalf("flying player took ground damage: HP=%v, want 100", pEntity.Vitals.HP)
	}
}

func TestToggleSkillCancelsActiveStatus(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	player := newCombatPlayer("player", Vec2{X: 0, Y: 0})
	toggleSkill := combat.Skill{
		ID: "TOGGLE_SHIELD", Slot: 4, Mode: combat.ModeSelf, Name: "Toggle",
		StatusID: "AURA_SHIELD", StatusDurationTicks: 100, StatusDebuff: false,
		IsToggle: true, Shield: 10,
	}
	player.Combat.Skills[4] = toggleSkill
	if err := w.AddEntity(player); err != nil {
		t.Fatal(err)
	}
	pEntity := worldEntity(t, w, "player")

	// Apply status first
	w.applyStatusLocked(pEntity, pEntity, toggleSkill)
	if !w.hasStatus(pEntity, "AURA_SHIELD") {
		t.Fatal("expected AURA_SHIELD to be active")
	}

	// Cast the toggle skill again -> should remove the status!
	w.startCastLocked(pEntity, CastCommand{EntityID: "player", Seq: 1, Slot: 4, Mode: combat.ModeSelf})
	if w.hasStatus(pEntity, "AURA_SHIELD") {
		t.Fatal("re-casting toggle skill did not cancel active status")
	}
}

func TestInvincibleAndLastStandAndEvasion(t *testing.T) {
	w, _ := NewWorld("arena", 8)
	attacker := newCombatPlayer("attacker", Vec2{X: 0, Y: 0})
	defender := newCombatPlayer("defender", Vec2{X: 2, Y: 0})
	defender.Vitals.HP = 50
	defender.Vitals.MaxHP = 100
	defender.Vitals.Shield = 20
	if err := w.AddEntity(attacker); err != nil {
		t.Fatal(err)
	}
	if err := w.AddEntity(defender); err != nil {
		t.Fatal(err)
	}
	atk := worldEntity(t, w, "attacker")
	def := worldEntity(t, w, "defender")

	// 1. Test INVINCIBLE (Bất Hoại): 100% damage reduction
	invincibleSkill := testStatusSkill("INVINCIBLE", 0, 0, 50, 0)
	invincibleSkill.StatusDebuff = false
	w.applyStatusLocked(def, def, invincibleSkill)

	dmg := w.applyDamageLocked(atk, def, 100, "NUKE")
	if dmg != 0 || def.Vitals.HP != 50 || def.Vitals.Shield != 20 {
		t.Fatalf("INVINCIBLE failed: dealt %v dmg, HP=%v Shield=%v", dmg, def.Vitals.HP, def.Vitals.Shield)
	}
	events := w.events
	w.events = nil
	var foundImmune bool
	for _, ev := range events {
		if ev.Type == combat.EventDamageApplied && ev.Reason == "immune" && ev.Amount == 0 {
			foundImmune = true
			break
		}
	}
	if !foundImmune {
		t.Fatalf("expected EventDamageApplied with reason=immune: %+v", events)
	}

	// Remove INVINCIBLE
	w.removeStatusLocked(def, "INVINCIBLE")

	// 2. Test EVASION: Dodges completely
	evasionSkill := testStatusSkill("EVASION", 0, 0, 50, 0)
	evasionSkill.StatusDebuff = false
	w.applyStatusLocked(def, def, evasionSkill)

	dmg = w.applyDamageLocked(atk, def, 100, "NUKE")
	if dmg != 0 || def.Vitals.HP != 50 {
		t.Fatalf("EVASION failed: dealt %v dmg, HP=%v", dmg, def.Vitals.HP)
	}
	events = w.events
	w.events = nil
	var foundEvaded bool
	for _, ev := range events {
		if ev.Type == combat.EventDamageApplied && ev.Reason == "evaded" && ev.Amount == 0 {
			foundEvaded = true
			break
		}
	}
	if !foundEvaded {
		t.Fatalf("expected EventDamageApplied with reason=evaded: %+v", events)
	}

	// Remove EVASION
	w.removeStatusLocked(def, "EVASION")

	// 3. Test LAST_STAND (Bất Tử): Floored at 1 HP, prevents death
	lastStandSkill := testStatusSkill("LAST_STAND", 0, 0, 50, 0)
	lastStandSkill.StatusDebuff = false
	w.applyStatusLocked(def, def, lastStandSkill)

	// Hit with 999 lethal damage
	dmg = w.applyDamageLocked(atk, def, 999, "FATAL_STRIKE")
	if def.Vitals.HP != 1.0 {
		t.Fatalf("LAST_STAND failed to floor at 1 HP: HP=%v", def.Vitals.HP)
	}
	events = w.events
	w.events = nil
	var foundLastStand bool
	for _, ev := range events {
		if ev.Type == combat.EventEntityDied {
			t.Fatal("LAST_STAND allowed player to die!")
		}
		if ev.Type == combat.EventDamageApplied && ev.Reason == "last_stand" && ev.HP == 1.0 {
			foundLastStand = true
		}
	}
	if !foundLastStand {
		t.Fatalf("expected EventDamageApplied with reason=last_stand: %+v", events)
	}

	// Once LAST_STAND is removed, next hit should kill
	w.removeStatusLocked(def, "LAST_STAND")
	w.applyDamageLocked(atk, def, 10, "FINISHER")
	if def.Vitals.HP > 0 {
		t.Fatalf("expected player to die after LAST_STAND expired, HP=%v", def.Vitals.HP)
	}
	events = w.events
	w.events = nil
	var foundDied bool
	for _, ev := range events {
		if ev.Type == combat.EventEntityDied {
			foundDied = true
			break
		}
	}
	if !foundDied {
		t.Fatalf("expected EventEntityDied after LAST_STAND removed: %+v", events)
	}
}
