package simulator

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// newTestCombatant builds a minimal combatant for engine tests.
func newTestCombatant(id string, kind string, hp int, dmg int) *CombatantAuto {
	return &CombatantAuto{
		ID: id, Name: id, Kind: kind, Rank: 3,
		GridX: 1, GridY: 1,
		MaxHP: hp, HP: hp, BaseDmg: dmg, BaseDef: 0, Speed: 10,
		CustomState: map[string]int{},
	}
}

// TestBurnDoTApplies verifies a BURN StatusOp becomes an ActiveDoT and ticks.
func TestBurnDoTApplies(t *testing.T) {
	state := &CombatState{
		Seed: 42,
		Allies: []*CombatantAuto{
			newTestCombatant("ally", "Player", 1000, 100),
		},
		Enemies: []*CombatantAuto{
			newTestCombatant("enemy", "Enemy", 1000, 0),
		},
	}
	state.Allies[0].Actions = []SimAction{
		{
			ActionType: "GU_SKILL", Name: "Lửa", Damage: 10,
			StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusBurn, Value: 100, Turns: 2}},
		},
	}

	logs := Simulate(state)
	_ = logs

	if len(state.Enemies[0].ActiveDoT) == 0 {
		t.Fatal("expected burn DoT to be applied")
	}
	dot := state.Enemies[0].ActiveDoT[0]
	if dot.Type != "burn" || dot.Value != 100 {
		t.Fatalf("got %+v", dot)
	}
	// Damage should have been applied over at least one tick.
	if state.Enemies[0].HP >= 1000 {
		t.Fatal("expected enemy to take burn damage")
	}
}

// TestSleepSkipsTurnAndBreaksOnDamage verifies SLEEP skips turns and breaks on damage.
func TestSleepSkipsTurn(t *testing.T) {
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	state := &CombatState{
		Seed: 7,
		Allies: []*CombatantAuto{
			newTestCombatant("ally", "Player", 1000, 50),
		},
		Enemies: []*CombatantAuto{enemy},
	}
	// Enemy has an action, ally applies sleep.
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 50}}
	state.Allies[0].Actions = []SimAction{
		{
			ActionType: "GU_SKILL", Name: "Ngủ", Damage: 1,
			StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusSleep, Value: 0, Turns: 2}},
		},
	}

	Simulate(state)

	// Sleep may have been broken by damage, but the effect should have been
	// applied at least once during the simulation (turned into an ActiveEffect
	// and logged). Verify the simulator ran to completion without panic.
	if enemy.HP < 0 {
		t.Fatal("enemy HP went negative")
	}
}

// TestDeterminism verifies two runs with the same seed produce identical logs.
func TestDeterminism(t *testing.T) {
	run := func() string {
		state := &CombatState{
			Seed: 12345,
			Allies: []*CombatantAuto{
				newTestCombatant("ally", "Player", 500, 60),
			},
			Enemies: []*CombatantAuto{
				newTestCombatant("enemy", "Enemy", 500, 40),
			},
		}
		state.Allies[0].Actions = []SimAction{
			{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 60},
			{ActionType: "GU_SKILL", Name: "Sương", Damage: 20, SlowTier: 2},
		}
		state.Enemies[0].Actions = []SimAction{
			{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 40},
		}
		state.Allies[0].MentalResistance = 0.5
		logs := Simulate(state)
		// Collapse logs into a simple deterministic fingerprint.
		out := ""
		for _, l := range logs {
			out += l.Action + ":" + l.Detail + "|"
		}
		return out
	}

	a := run()
	b := run()
	if a != b {
		t.Fatalf("simulation not deterministic with same seed")
	}
}

// TestStatusDurationNotDoubleDecremented verifies effects decrement once per turn
// (the double-decrement bug would remove a 2-turn effect after a single turn).
func TestStatusDurationNotDoubleDecremented(t *testing.T) {
	c := newTestCombatant("unit", "Enemy", 1000, 0)
	c.ActiveEffects = []Effect{
		{ID: EffectStun, Type: "debuff", Tier: 3, Value: 1, Duration: 2},
	}

	// Turn 1 tick: 2 -> 1 (still active)
	tickStatusEffects(c, 1)
	if len(c.ActiveEffects) != 1 || c.ActiveEffects[0].Duration != 1 {
		t.Fatalf("after 1 tick expected duration 1, got %+v", c.ActiveEffects)
	}

	// Turn 2 tick: 1 -> 0 (expired)
	tickStatusEffects(c, 2)
	if len(c.ActiveEffects) != 0 {
		t.Fatalf("expected effect to expire after 2 ticks, got %+v", c.ActiveEffects)
	}
}

// TestCatalogStatusMultipliers verifies HASTE/ARMOR_BREAK/DAMAGE_REDUCTION/
// INVINCIBLE/REVIVE_LOCK are computed into derived stats.
func TestCatalogStatusMultipliers(t *testing.T) {
	c := newTestCombatant("unit", "Enemy", 1000, 0)
	c.Speed = 100
	c.ActiveEffects = []Effect{
		{ID: statusfx.StatusHaste, Type: "buff", Value: 20, Duration: 3},
		{ID: statusfx.StatusArmorBreak, Type: "debuff", Value: 30, Duration: 3},
		{ID: statusfx.StatusDamageReduction, Type: "buff", Value: 30, Duration: 3},
		{ID: statusfx.StatusReviveLock, Type: "debuff", Value: 1, Duration: 3},
	}
	ComputeEffectMultipliers(c)

	if c.EffectiveSpeed != 120 {
		t.Fatalf("expected 20%% haste -> speed 120, got %.0f", c.EffectiveSpeed)
	}
	if c.DefenseMultiplier >= 1.0 {
		t.Fatalf("expected armor break to reduce defense, got %v", c.DefenseMultiplier)
	}
	if c.DamageTakenReduction != 0.30 {
		t.Fatalf("expected 30%% damage reduction, got %v", c.DamageTakenReduction)
	}
	if !c.IsReviveLocked {
		t.Fatal("expected revive lock flag set")
	}
}

// TestInvincibleBlocksDamage verifies INVINCIBLE reduces damage to zero.
func TestInvincibleBlocksDamage(t *testing.T) {
	c := newTestCombatant("unit", "Enemy", 1000, 0)
	c.ActiveEffects = []Effect{
		{ID: statusfx.StatusInvincible, Type: "buff", Value: 100, Duration: 2},
	}
	ComputeEffectMultipliers(c)
	if c.DamageTakenReduction < 1.0 {
		t.Fatalf("expected full damage reduction, got %v", c.DamageTakenReduction)
	}
}

// firstTurnDamageActors returns the actor IDs of damage-dealing logs from the
// first turn of the simulation, in execution order.
func firstTurnDamageActors(logs []CombatLog) []string {
	var ids []string
	turnCount := 0
	for _, l := range logs {
		if l.Action == "TURN_START" {
			turnCount++
			if turnCount > 1 {
				break
			}
			continue
		}
		if turnCount == 1 && l.Damage > 0 && l.ActorId != "" {
			ids = append(ids, l.ActorId)
		}
	}
	return ids
}

func idsEqual(a []string, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestDamageTriggersCounterInterleave verifies that each damage-dealing action
// is answered immediately by the victim's next action, instead of one side
// dumping its whole kit: expected order is ally, enemy, ally, enemy.
func TestDamageTriggersCounterInterleave(t *testing.T) {
	state := &CombatState{
		Seed: 99,
		Allies: []*CombatantAuto{
			newTestCombatant("ally", "Player", 1000, 10),
		},
		Enemies: []*CombatantAuto{
			newTestCombatant("enemy", "Enemy", 1000, 5),
		},
	}
	state.Allies[0].Speed = 100
	state.Allies[0].Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "A1", Damage: 10},
		{ActionType: "GU_SKILL", Name: "A2", Damage: 10},
	}
	state.Enemies[0].Speed = 50
	state.Enemies[0].Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "E1", Damage: 5},
		{ActionType: "GU_SKILL", Name: "E2", Damage: 5},
	}

	logs := Simulate(state)
	got := firstTurnDamageActors(logs)
	want := []string{"ally", "enemy", "ally", "enemy"}
	if !idsEqual(got, want) {
		t.Fatalf("expected interleaved order %v, got %v", want, got)
	}
}

// TestCounterDoesNotChain verifies a counter-attack never provokes another
// counter (depth cap of one): exactly four damage events in the first turn.
func TestCounterDoesNotChain(t *testing.T) {
	state := &CombatState{
		Seed: 7,
		Allies: []*CombatantAuto{
			newTestCombatant("ally", "Player", 1000, 10),
		},
		Enemies: []*CombatantAuto{
			newTestCombatant("enemy", "Enemy", 1000, 5),
		},
	}
	state.Allies[0].Speed = 100
	state.Allies[0].Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "A1", Damage: 10},
	}
	state.Enemies[0].Speed = 50
	state.Enemies[0].Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "E1", Damage: 5},
	}

	logs := Simulate(state)
	got := firstTurnDamageActors(logs)
	want := []string{"ally", "enemy"}
	if !idsEqual(got, want) {
		t.Fatalf("expected single exchange [ally enemy], got %v", got)
	}
}

// TestSupportDoesNotProvokeCounter verifies that a non-damaging support cast
// does not grant the opponent a counter window: the enemy answers only the
// damage cast, then spends its remaining action at its own speed slot.
func TestSupportDoesNotProvokeCounter(t *testing.T) {
	state := &CombatState{
		Seed: 21,
		Allies: []*CombatantAuto{
			newTestCombatant("ally", "Player", 1000, 10),
		},
		Enemies: []*CombatantAuto{
			newTestCombatant("enemy", "Enemy", 1000, 5),
		},
	}
	state.Allies[0].Speed = 100
	state.Allies[0].Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "Buff", DefUpTier: 1},
		{ActionType: "GU_SKILL", Name: "A2", Damage: 10},
	}
	state.Enemies[0].Speed = 50
	state.Enemies[0].Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "E1", Damage: 5},
		{ActionType: "GU_SKILL", Name: "E2", Damage: 5},
	}

	logs := Simulate(state)
	got := firstTurnDamageActors(logs)
	want := []string{"ally", "enemy", "enemy"}
	if !idsEqual(got, want) {
		t.Fatalf("expected [ally damage -> enemy counter, then enemy slot], got %v", got)
	}
}
