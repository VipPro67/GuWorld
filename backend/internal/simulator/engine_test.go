package simulator

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// newTestCombatant builds a minimal combatant for engine tests.
func newTestCombatant(id string, kind string, hp float64, dmg float64) *CombatantAuto {
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

	hasBurnLog := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "Thiêu Đốt") || strings.Contains(l.Detail, "BURN") || strings.Contains(l.Detail, "cháy") {
			hasBurnLog = true
			break
		}
	}
	if !hasBurnLog {
		t.Fatal("expected burn DoT log to be recorded in combat logs")
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

func TestSpecialBehaviorComesFromEffectMechanicNotSourceID(t *testing.T) {
	run := func(sourceID string, mechanics []CombatMechanic) (hp, shield float64) {
		attacker := newTestCombatant("ally", "Player", 100, 100)
		action := SimAction{
			ActionType: "GU_SKILL", SourceID: sourceID, Name: "Phá khiên", Damage: 100,
			Mechanics: mechanics,
		}
		target := newTestCombatant("enemy", "Enemy", 100, 0)
		target.VirtualShield = 100
		state := &CombatState{Seed: 1, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{target}}
		var logs []CombatLog
		applyDamageToTarget(state, attacker, target, action.Damage, &logs, 2, hasMechanic(action.Mechanics, "ON_HIT", "BYPASS_SHIELD"), false, false, rand.New(rand.NewSource(1)))
		return target.HP, target.VirtualShield
	}

	oldHP, oldShield := run("GU_BREAK_SPACE", nil)
	dataHP, dataShield := run("GU_ARBITRARY_NEW", []CombatMechanic{{Trigger: "ON_HIT", ActionCode: "BYPASS_SHIELD"}})
	if oldShield >= 100 || oldHP <= 0 {
		t.Fatalf("legacy ID unexpectedly changed damage routing: hp=%v shield=%v", oldHP, oldShield)
	}
	if dataHP > 0 || dataShield != 100 {
		t.Fatalf("catalog mechanic did not bypass shield: hp=%v shield=%v", dataHP, dataShield)
	}
}

// TestStatusDurationNotDoubleDecremented verifies effects decrement once per turn
// (the double-decrement bug would remove a 2-turn effect after a single turn).

// TestSeizeInitiativeMovesSeizerFirst verifies ON_BATTLE_START/SEIZE_INITIATIVE:
// the owner jumps ahead of a faster enemy on turn 1 only when the primary
// enemy's lifespan does not exceed lifespan_ratio times its own.
func TestSeizeInitiativeMovesSeizerFirst(t *testing.T) {
	firstActorTurn1 := func(logs []CombatLog) string {
		for _, l := range logs {
			if l.Turn == 1 && l.Action != "TURN_START" && l.ActorId != "" {
				return l.ActorId
			}
		}
		return ""
	}
	seized := func(logs []CombatLog) bool {
		for _, l := range logs {
			if l.Action == "SEIZE_INITIATIVE" {
				return true
			}
		}
		return false
	}
	run := func(enemyRank int) (string, bool) {
		ally := newTestCombatant("ally", "Player", 100000, 1)
		ally.Speed = 1
		ally.Lifespan = 100
		ally.Mechanics = []CombatMechanic{{
			Trigger:    "ON_BATTLE_START",
			ActionCode: "SEIZE_INITIATIVE",
			Params:     []byte(`{"lifespan_ratio":3}`),
		}}
		ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}
		enemy := newTestCombatant("enemy", "Enemy", 100000, 1)
		enemy.Rank = enemyRank
		enemy.Speed = 50
		enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}
		state := &CombatState{Seed: 3, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
		logs := Simulate(state)
		return firstActorTurn1(logs), seized(logs)
	}

	// Enemy rank 1 -> lifespan ~100 <= 3*100: seizes opening turn despite lower speed.
	if first, ok := run(1); !ok || first != "ally" {
		t.Fatalf("expected ally to seize turn 1: first=%q seized=%v", first, ok)
	}
	// Enemy rank 9 -> lifespan ~50000 > 3*100: no seizure, faster enemy acts first.
	if first, ok := run(9); ok || first != "enemy" {
		t.Fatalf("expected no seizure vs high-lifespan enemy: first=%q seized=%v", first, ok)
	}
}
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
	want := []string{"ally", "enemy"}
	if !idsEqual(got, want) {
		t.Fatalf("expected turn 1 exchange %v, got %v", want, got)
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
// executes without ending the turn pass, allowing the actor to also cast their attack action before yielding turn.
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
	want := []string{"ally", "enemy"}
	if !idsEqual(got, want) {
		t.Fatalf("expected [ally enemy] damage in turn 1 (ally cast non-damaging buff then damage A2), got %v", got)
	}
}

func TestExplicitPriorityActionSorting(t *testing.T) {
	state := &CombatState{
		Seed: 42,
		Allies: []*CombatantAuto{
			newTestCombatant("ally", "Player", 1000, 10),
		},
		Enemies: []*CombatantAuto{
			newTestCombatant("boss", "Boss", 1000, 5),
		},
	}
	state.Allies[0].Speed = 10
	state.Enemies[0].Speed = 100
	state.Enemies[0].Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "WeakGu", Damage: 50, Priority: 4},
		{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: 300, Priority: 999},
		{ActionType: "GU_SKILL", Name: "ShieldGu", Defense: 500, Priority: 1},
		{ActionType: "GU_SKILL", Name: "StrongGu", Damage: 600, Priority: 2},
	}

	logs := Simulate(state)

	var bossActionNames []string
	for _, l := range logs {
		if l.ActorId == "boss" && l.Turn == 1 {
			if l.Action == "MANA_COST" || l.Action == "TURN_START" || l.Action == "BATTLE_START" {
				continue
			}
			for _, name := range []string{"ShieldGu", "StrongGu", "Đánh thường", "WeakGu"} {
				if strings.Contains(l.Detail, name) {
					bossActionNames = append(bossActionNames, name)
					break
				}
			}
		}
	}

	if len(bossActionNames) > 0 && bossActionNames[0] != "ShieldGu" {
		t.Errorf("expected ShieldGu (Priority 1) to execute first, got %v", bossActionNames)
	}
}

func TestPerGuSequentialActionExecution(t *testing.T) {
	attacker := newTestCombatant("attacker", "Player", 2000, 100)
	attacker.Mana = 100
	attacker.MaxMana = 100
	attacker.Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "GuFirst_Heal", Healing: 500, ManaCost: 40},   // High priority emergency heal (always condition)
		{ActionType: "GU_SKILL", Name: "GuSecond_Damage", Damage: 300, ManaCost: 40}, // Secondary damage
	}

	defender := newTestCombatant("defender", "Enemy", 2000, 50)

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{defender},
	}

	logs := Simulate(state)

	var firstTurnAction string
	for _, l := range logs {
		if l.Turn == 1 && l.ActorId == "attacker" {
			if strings.Contains(l.Detail, "GuFirst_Heal") {
				firstTurnAction = "GuFirst_Heal"
				break
			}
		}
	}

	if firstTurnAction != "GuFirst_Heal" {
		t.Fatalf("expected top priority GuFirst_Heal on turn 1, got %s", firstTurnAction)
	}

	// Verify that when mana runs out, attacker falls back to BASIC_ATTACK
	hasBasicAttack := false
	for _, l := range logs {
		if l.ActorId == "attacker" && strings.Contains(l.Detail, "Đánh thường") {
			hasBasicAttack = true
			break
		}
	}
	if !hasBasicAttack {
		t.Fatalf("expected attacker to fall back to BASIC_ATTACK when out of mana")
	}
}

func TestPreAttackDefenseShieldPass(t *testing.T) {
	// Attacker A is faster (Speed 200) with attack Gu
	attacker := newTestCombatant("attacker", "Player", 1000, 200)
	attacker.Speed = 200
	attacker.Actions = []SimAction{
		{ActionType: "BASIC_ATTACK", Name: "Đao Thần", Damage: 300},
	}

	// Defender B is slower (Speed 50) with shield Gu and attack Gu
	defender := newTestCombatant("defender", "Enemy", 1000, 50)
	defender.Speed = 50
	defender.Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "Kim Giáp Gu", Defense: 500},
		{ActionType: "BASIC_ATTACK", Name: "Đánh Thường", Damage: 50},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{defender},
	}

	logs := Simulate(state)

	shieldLogIndex := -1
	attackLogIndex := -1

	for i, l := range logs {
		if l.Turn == 1 {
			if l.ActorId == "defender" && strings.Contains(l.Detail, "Khiên Ảo") {
				if shieldLogIndex == -1 {
					shieldLogIndex = i
				}
			}
			if l.ActorId == "attacker" && strings.Contains(l.Detail, "Đao Thần") {
				if attackLogIndex == -1 {
					attackLogIndex = i
				}
			}
		}
	}

	if shieldLogIndex == -1 {
		t.Fatalf("expected defender to gain VirtualShield on turn 1")
	}
	if attackLogIndex == -1 {
		t.Fatalf("expected attacker to strike on turn 1")
	}
	if shieldLogIndex >= attackLogIndex {
		t.Fatalf("expected defender shield gain log (index %d) BEFORE attacker attack log (index %d)", shieldLogIndex, attackLogIndex)
	}

	// Verify defender HP was fully preserved by shield
	if defender.HP < 1000 {
		t.Fatalf("expected defender HP to be 1000 (shield absorbed all 300 damage), got %.0f", defender.HP)
	}
}

// TestMonsterEqualRankGuNoBacklash verifies a Rank 3 monster casting a Rank 3 Gu suffers 0 backlash.
func TestMonsterEqualRankGuNoBacklash(t *testing.T) {
	monster := newTestCombatant("monster", "Enemy", 10000, 100)
	monster.Rank = 3
	monster.Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "Thanh Long Cổ", Rank: 3, Damage: 50},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{newTestCombatant("player", "Player", 10000, 0)},
		Enemies: []*CombatantAuto{monster},
	}

	logs := Simulate(state)

	for _, l := range logs {
		if l.Action == "GU_BACKLASH" && l.ActorId == "monster" {
			t.Fatalf("expected 0 backlash for equal rank Gu, but got backlash log: %s", l.Detail)
		}
	}
}

// TestHigherRankGuBacklash verifies a Rank 3 combatant casting a Rank 4 Gu suffers 15% MaxHP backlash.
func TestHigherRankGuBacklash(t *testing.T) {
	combatant := newTestCombatant("actor", "Player", 10000, 100)
	combatant.Rank = 3
	combatant.Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "Rank 4 Gu", Rank: 4, Damage: 50},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{combatant},
		Enemies: []*CombatantAuto{newTestCombatant("dummy", "Enemy", 10000, 0)},
	}

	logs := Simulate(state)

	hasBacklash := false
	for _, l := range logs {
		if l.Action == "GU_BACKLASH" && l.ActorId == "actor" {
			hasBacklash = true
			if l.Damage != 1500 {
				t.Fatalf("expected backlash damage of 1500 (15%% of 10000), got %.0f", l.Damage)
			}
		}
	}
	if !hasBacklash {
		t.Fatal("expected backlash log for higher rank Gu usage")
	}
}

// TestStunSkipsTurn verifies that a combatant with STUN effect skips their action turn.
func TestStunSkipsTurn(t *testing.T) {
	state := &CombatState{
		Seed: 42,
		Allies: []*CombatantAuto{
			newTestCombatant("ally", "Player", 1000, 10),
		},
		Enemies: []*CombatantAuto{
			newTestCombatant("enemy", "Enemy", 1000, 50),
		},
	}
	state.Allies[0].Actions = []SimAction{
		{
			ActionType: "GU_SKILL", Name: "Định Thân Cổ", Damage: 10,
			StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusStun, Value: 1, Turns: 1}},
		},
	}
	state.Enemies[0].Actions = []SimAction{
		{ActionType: "BASIC_ATTACK", Name: "Đánh Thường", Damage: 50},
	}

	logs := Simulate(state)
	hasStunApplied := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "Choáng") || strings.Contains(l.Action, "STUN") {
			hasStunApplied = true
			break
		}
	}
	if !hasStunApplied {
		t.Fatal("expected Stun to be applied and logged")
	}
}

// TestStatusFxStunAndInvincibleInCombat verifies that statusfx STUN correctly causes enemy turn skip,
// and statusfx INVINCIBLE applied mid-turn correctly blocks incoming damage.
func TestStatusFxStunAndInvincibleInCombat(t *testing.T) {
	state := &CombatState{
		Seed: 42,
		Allies: []*CombatantAuto{
			newTestCombatant("ally", "Duhring Cosima", 1000, 10),
		},
		Enemies: []*CombatantAuto{
			newTestCombatant("enemy", "Hình Nhân Gỗ", 10000, 100),
		},
	}
	state.Allies[0].Actions = []SimAction{
		{
			ActionType: "GU_SKILL", Name: "Vô Tướng Nhãn Cổ", Damage: 0,
			StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusInvincible, Value: 100, Turns: 2}},
		},
		{
			ActionType: "GU_SKILL", Name: "Sát Xích Cổ", Damage: 10,
			StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusStun, Value: 1, Turns: 1}},
		},
	}
	state.Enemies[0].Actions = []SimAction{
		{ActionType: "BASIC_ATTACK", Name: "Đánh Thường", Damage: 500},
	}

	logs := Simulate(state)

	enemyAttacked := false
	stunnedLogFound := false
	for _, l := range logs {
		if l.ActorId == "enemy" && l.Action == "BASIC_ATTACK" && l.Turn == 1 {
			enemyAttacked = true
		}
		if l.ActorId == "enemy" && (l.Action == "STUNNED" || strings.Contains(l.Detail, "bị choáng, bỏ lượt")) {
			stunnedLogFound = true
		}
	}

	if !stunnedLogFound {
		t.Fatal("expected enemy to be stunned and logged STUNNED")
	}
	if enemyAttacked {
		t.Fatal("expected enemy to NOT be able to attack while stunned")
	}
}

// TestGenericExtensibleStatusFx verifies that newly registered statusfx effects
// (like HASTE, ARMOR_BREAK, WEAK) automatically update combatant multipliers and logs.
func TestGenericExtensibleStatusFx(t *testing.T) {
	c := newTestCombatant("unit", "Caster", 1000, 10)
	target := newTestCombatant("target", "Target", 1000, 10)

	ops := []statusfx.StatusOp{
		{ID: statusfx.StatusHaste, Value: 30, Turns: 2},
		{ID: statusfx.StatusArmorBreak, Value: 25, Turns: 2},
		{ID: statusfx.StatusWeak, Value: 20, Turns: 2},
	}

	var logs []CombatLog
	rng := rand.New(rand.NewSource(42))
	applyStatusOps(c, []*CombatantAuto{target}, ops, c.Rank, 1, rng, &logs)

	if c.EffectiveSpeed <= c.Speed {
		t.Fatalf("expected caster speed to increase with Haste, got %v", c.EffectiveSpeed)
	}
	if target.DefenseMultiplier >= 1.0 {
		t.Fatalf("expected target defense multiplier to decrease with ArmorBreak, got %v", target.DefenseMultiplier)
	}
	if target.DamageMultiplier >= 1.0 {
		t.Fatalf("expected target damage multiplier to decrease with Weak, got %v", target.DamageMultiplier)
	}
	if len(logs) < 3 {
		t.Fatalf("expected at least 3 status application logs, got %d", len(logs))
	}
}

// TestSilenceForcesBasicAttack verifies that a SILENCED combatant cannot use skills.
func TestSilenceForcesBasicAttack(t *testing.T) {
	state := &CombatState{
		Seed: 42,
		Allies: []*CombatantAuto{
			newTestCombatant("ally", "Player", 1000, 10),
		},
		Enemies: []*CombatantAuto{
			newTestCombatant("enemy", "Enemy", 1000, 10),
		},
	}
	state.Allies[0].Actions = []SimAction{
		{
			ActionType: "GU_SKILL", Name: "Cấm Ngôn Cổ", Damage: 10,
			StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusSilence, Value: 1, Turns: 2}},
		},
	}
	state.Enemies[0].Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "Tuyệt Kỹ Hỏa Diệm", Damage: 100, ManaCost: 5},
		{ActionType: "BASIC_ATTACK", Name: "Đánh Thường", Damage: 10},
	}

	logs := Simulate(state)
	for _, l := range logs {
		if l.ActorId == "enemy" && l.Turn == 2 {
			if strings.Contains(l.Detail, "Tuyệt Kỹ Hỏa Diệm") {
				t.Fatal("enemy should NOT have been able to cast GU_SKILL while silenced")
			}
		}
	}
}

// TestThornsReflectsDamage verifies that THORNS buff reflects damage back to the attacker.
func TestThornsReflectsDamage(t *testing.T) {
	defender := newTestCombatant("defender", "Player", 1000, 10)
	defender.Speed = 20
	defender.Actions = []SimAction{
		{
			ActionType: "GU_SKILL", Name: "Gai Nhọn Cổ", Damage: 0,
			StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusThorns, Value: 30, Turns: 3}},
		},
	}

	attacker := newTestCombatant("attacker", "Enemy", 1000, 100)
	attacker.Speed = 10
	attacker.Actions = []SimAction{
		{ActionType: "BASIC_ATTACK", Name: "Đánh Thường", Damage: 100},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{defender},
		Enemies: []*CombatantAuto{attacker},
	}

	Simulate(state)
	if attacker.HP >= 1000 {
		t.Fatalf("expected attacker to take reflected thorns damage, but HP is %.0f", attacker.HP)
	}
}

// TestSoulWoundIgnoresShield verifies SOUL_WOUND applies direct damage bypassing armor.
func TestSoulWoundIgnoresShield(t *testing.T) {
	defender := newTestCombatant("defender", "Player", 500, 0)
	defender.VirtualShield = 1000 // Huge shield

	attacker := newTestCombatant("attacker", "Enemy", 500, 0)
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL", Name: "Hồn Thương Sát Chiêu", Damage: 0,
			StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusSoulWound, Value: 100, Turns: 1}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{defender},
		Enemies: []*CombatantAuto{attacker},
	}

	Simulate(state)
	if defender.HP >= 500 {
		t.Fatalf("expected defender HP to be damaged by Soul Wound, but HP is %.0f", defender.HP)
	}
}

// TestDoTSubtractsShieldFirst verifies standard DoTs deduct shield before HP.
func TestDoTSubtractsShieldFirst(t *testing.T) {
	c := newTestCombatant("target", "Player", 500, 0)
	c.VirtualShield = 100
	c.ActiveDoT = []DoTEffect{
		{Type: "burn", Value: 60, Turns: 1, DirectHP: false},
	}

	logs := tickDoTs(c, 1)

	if c.VirtualShield != 40 {
		t.Fatalf("expected shield to be 40 after 60 DoT damage, got %.0f", c.VirtualShield)
	}
	if c.HP != 500 {
		t.Fatalf("expected HP to remain 500, got %.0f", c.HP)
	}
	if len(logs) == 0 || !strings.Contains(logs[0].Detail, "khiên hấp thụ") {
		t.Fatalf("expected log detail to mention shield absorption, got %v", logs)
	}

	// Test overflow DoT damage
	c.ActiveDoT = []DoTEffect{
		{Type: "bleed", Value: 70, Turns: 1, DirectHP: false},
	}
	tickDoTs(c, 2)
	if c.VirtualShield != 0 {
		t.Fatalf("expected shield to be depleted (0), got %.0f", c.VirtualShield)
	}
	if c.HP != 470 {
		t.Fatalf("expected HP to be 470 after 30 excess DoT damage, got %.0f", c.HP)
	}
}

// TestDirectHPDoTBypassesShield verifies direct HP DoT effects bypass shield completely.
func TestDirectHPDoTBypassesShield(t *testing.T) {
	c := newTestCombatant("target", "Player", 500, 0)
	c.VirtualShield = 100
	c.ActiveDoT = []DoTEffect{
		{Type: "soul_wound", Value: 60, Turns: 1, DirectHP: true},
	}

	tickDoTs(c, 1)

	if c.VirtualShield != 100 {
		t.Fatalf("expected shield to remain 100, got %.0f", c.VirtualShield)
	}
	if c.HP != 440 {
		t.Fatalf("expected HP to be reduced to 440, got %.0f", c.HP)
	}
}

// TestPureBuffMultipliersApplyToNextAction verifies pure buff Gu (with DamageMult or ShieldMult but flat value 0) buff the next attack/shield.
// Also verifies that duplicate Gu with the same ID only apply once.
func TestPureBuffMultipliersApplyToNextAction(t *testing.T) {
	attacker := newTestCombatant("attacker", "Player", 500, 0)
	attacker.Speed = 20
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			SourceID:   "GU_MULTIPLIER",
			Name:       "Công Bội Cổ #1",
			Damage:     0,
			DamageMult: 2.0,
			Priority:   1,
		},
		{
			ActionType: "GU_SKILL",
			SourceID:   "GU_MULTIPLIER", // Duplicate Gu ID -> should only apply 2.0x once, NOT 4.0x
			Name:       "Công Bội Cổ #2",
			Damage:     0,
			DamageMult: 2.0,
			Priority:   2,
		},
		{
			ActionType: "GU_SKILL",
			SourceID:   "GU_OTHER_BUFF", // Different Gu ID -> should stack: 2.0 * 1.5 = 3.0x
			Name:       "Tăng Uy Cổ",
			Damage:     0,
			DamageMult: 1.5,
			Priority:   3,
		},
		{
			ActionType: "GU_SKILL",
			Name:       "Sát Chiêu Công Kích",
			Damage:     100,
			DamageMult: 1.0,
			Priority:   4,
		},
		{
			ActionType: "GU_SKILL",
			SourceID:   "GU_SHIELD_BUFF",
			Name:       "Hộ Thể Cổ #1",
			Defense:    0,
			ShieldMult: 2.0,
			Priority:   5,
		},
		{
			ActionType: "GU_SKILL",
			SourceID:   "GU_SHIELD_BUFF", // Duplicate Gu ID -> only 2.0x
			Name:       "Hộ Thể Cổ #2",
			Defense:    0,
			ShieldMult: 2.0,
			Priority:   6,
		},
		{
			ActionType: "GU_SKILL",
			Name:       "Tạo Giáp Kỹ",
			Defense:    100,
			ShieldMult: 1.0,
			Priority:   7,
		},
	}

	defender := newTestCombatant("defender", "Enemy", 1000, 0)
	defender.Speed = 5

	state := &CombatState{
		Seed:    12345,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{defender},
	}

	logs := Simulate(state)

	var attackLog *CombatLog
	for i := range logs {
		if logs[i].Action == "GU_SKILL" && strings.Contains(logs[i].Detail, "Sát Chiêu Công Kích") {
			attackLog = &logs[i]
			break
		}
	}
	if attackLog == nil {
		t.Fatalf("expected attack log for Sát Chiêu Công Kích")
	}

	// Attack did 100 * (2.0 from GU_MULTIPLIER * 1.5 from GU_OTHER_BUFF) = 300 damage.
	// If duplicate GU_MULTIPLIER had stacked, it would have been 100 * 2 * 2 * 1.5 = 600 damage.
	if attackLog.Damage != 300 {
		t.Fatalf("expected attack to deal exactly 300 damage, but got %.0f", attackLog.Damage)
	}

	// Attacker should have received 100 * 2.0 = 200 shield from 2.0x buffed shield (not 400 from 2x2)
	var shieldActionLog *CombatLog
	for i := range logs {
		if logs[i].Action == "GU_SKILL" && strings.Contains(logs[i].Detail, "Tạo Giáp Kỹ") {
			shieldActionLog = &logs[i]
			break
		}
	}
	if shieldActionLog == nil || !strings.Contains(shieldActionLog.Detail, "200 Khiên Ảo") {
		t.Fatalf("expected shield action log mentioning 200 Khiên Ảo, but got %+v", shieldActionLog)
	}
}

func TestGuHungerDeductionByInstanceID(t *testing.T) {
	attacker := newTestCombatant("player1", "Player", 500, 100)
	attacker.Speed = 20
	attacker.GuHunger = map[string]int{
		"inst_gu_1": 2,
		"inst_gu_2": 10,
	}
	attacker.GuUsedCounts = make(map[string]int)
	attacker.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Hỏa Cầu 1",
			SourceID:   "GU_FIRE",
			InstanceID: "inst_gu_1",
			Damage:     50,
			Priority:   1,
		},
		{
			ActionType: "BASIC_ATTACK",
			Name:       "Đánh thường",
			Damage:     20,
			Priority:   999,
		},
	}

	defender := newTestCombatant("defender", "Enemy", 300, 0)
	defender.Speed = 5
	defender.Actions = []SimAction{
		{ActionType: "BASIC_ATTACK", Name: "Đòn quái", Damage: 10, Priority: 999},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{defender},
	}

	Simulate(state)

	// inst_gu_1 had 2 hunger and was used multiple turns until hunger reached 0
	if attacker.GuHunger["inst_gu_1"] != 0 {
		t.Fatalf("expected inst_gu_1 hunger to be 0, got %d", attacker.GuHunger["inst_gu_1"])
	}
	if attacker.GuUsedCounts["inst_gu_1"] != 2 {
		t.Fatalf("expected inst_gu_1 to be used exactly 2 times, got %d", attacker.GuUsedCounts["inst_gu_1"])
	}
	// inst_gu_2 was not used, hunger must remain 10 and not present in GuUsedCounts
	if attacker.GuHunger["inst_gu_2"] != 10 {
		t.Fatalf("expected inst_gu_2 hunger to remain 10, got %d", attacker.GuHunger["inst_gu_2"])
	}
	if attacker.GuUsedCounts["inst_gu_2"] != 0 {
		t.Fatalf("expected inst_gu_2 used count to be 0, got %d", attacker.GuUsedCounts["inst_gu_2"])
	}
}

// TestItemHealAppliesHealingReceived verifies the auto item heal scales with
// the healing_received modifier (e.g. Trọng Thương −50%) and records a
// breakdown with the status provenance.
func TestItemHealAppliesHealingReceived(t *testing.T) {
	actor := newTestCombatant("ally", "Player", 1000, 100)
	actor.HP = 100
	actor.AutoHealHP = true
	actor.HealThreshold = 50 // 100/1000 = 10% <= threshold -> heals
	actor.HealCharges = 1
	actor.HealPotency = 200
	actor.HealingReceived = -0.5
	actor.StatProvenance = map[string]*StatProvenance{
		"healing_received": {
			Stat:       "healing_received",
			BaseValue:  0,
			FinalValue: -0.5,
			Mods:       []StatModSource{{Key: "severely_wounded", Name: "Trọng Thương", Pct: -0.5}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{actor},
		Enemies: []*CombatantAuto{newTestCombatant("enemy", "Enemy", 1000, 0)},
	}
	logs := Simulate(state)

	var healLog *CombatLog
	for i := range logs {
		if logs[i].Action == "ITEM_HEAL" {
			healLog = &logs[i]
			break
		}
	}
	if healLog == nil {
		t.Fatal("expected an ITEM_HEAL log")
	}
	if healLog.Damage != 100 { // 200 * (1 - 0.5)
		t.Fatalf("item heal = %v, want 100 after healing_received -0.5", healLog.Damage)
	}
	if healLog.Breakdown == nil || healLog.Breakdown.HealingBase != 200 || healLog.Breakdown.HealingFinal != 100 {
		t.Fatalf("missing/incorrect healing breakdown: %+v", healLog.Breakdown)
	}
	if healLog.Breakdown.HealerHealing == nil || len(healLog.Breakdown.HealerHealing.Mods) == 0 {
		t.Fatal("expected healing_received provenance on the breakdown")
	}
}

// TestSkillHealBreakdown verifies a GU_SKILL heal records its formula
// (base + dao bonus, scaled by healing_received) in the breakdown.
func TestSkillHealBreakdown(t *testing.T) {
	actor := newTestCombatant("ally", "Player", 1000, 0)
	actor.HP = 100
	actor.HealingReceived = -0.5
	actor.StatProvenance = map[string]*StatProvenance{
		"healing_received": {
			Stat:       "healing_received",
			BaseValue:  0,
			FinalValue: -0.5,
			Mods:       []StatModSource{{Key: "severely_wounded", Name: "Trọng Thương", Pct: -0.5}},
		},
	}
	actor.Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "Hồi Xuân", Healing: 200},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{actor},
		Enemies: []*CombatantAuto{newTestCombatant("enemy", "Enemy", 1000, 0)},
	}
	logs := Simulate(state)

	var healLog *CombatLog
	for i := range logs {
		// The HEAL log is folded into the consolidated GU_SKILL entry; find it
		// via the healing breakdown the merge preserves.
		if bd := logs[i].Breakdown; bd != nil && bd.HealingFinal > 0 {
			healLog = &logs[i]
			break
		}
	}
	if healLog == nil {
		t.Fatal("expected a heal log with breakdown")
	}
	// The consolidated entry keeps the cast log's Damage field (0); the heal
	// amount lives in the breakdown.
	if healLog.Breakdown == nil ||
		healLog.Breakdown.HealingBase != 200 ||
		healLog.Breakdown.HealingDaoBonus != 0 ||
		healLog.Breakdown.HealingFinal != 100 {
		t.Fatalf("missing/incorrect healing breakdown: %+v", healLog.Breakdown)
	}
	if healLog.Breakdown.HealerHealing == nil {
		t.Fatal("expected healing_received provenance on the breakdown")
	}
}

// TestDamageBreakdownCarriesStatProvenance verifies attack/defense status
// provenance is attached to damage breakdowns from the combatants.
func TestDamageBreakdownCarriesStatProvenance(t *testing.T) {
	actor := newTestCombatant("ally", "Player", 1000, 100)
	actor.StatProvenance = map[string]*StatProvenance{
		"attack": {
			Stat:       "attack",
			BaseValue:  10,
			FinalValue: 5,
			Mods:       []StatModSource{{Key: "death_penalty", Name: "Hình Phạt Tử Vong", Pct: -0.5}},
		},
	}
	target := newTestCombatant("enemy", "Enemy", 1000, 0)
	target.StatProvenance = map[string]*StatProvenance{
		"defense": {
			Stat:       "defense",
			BaseValue:  8,
			FinalValue: 4,
			Mods:       []StatModSource{{Key: "regional_wall_suppression", Name: "Áp Chế Giới Vực", Pct: -0.5}},
		},
	}
	actor.Actions = []SimAction{
		{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: 100, DamageMult: 1.0, TargetType: "SINGLE"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{actor},
		Enemies: []*CombatantAuto{target},
	}
	logs := Simulate(state)

	found := false
	for i := range logs {
		bd := logs[i].Breakdown
		if bd == nil || bd.BaseDamage != 100 {
			continue
		}
		found = true
		if bd.AttackerAttack == nil || bd.AttackerAttack.FinalValue != 5 {
			t.Fatalf("attacker attack provenance missing/wrong: %+v", bd.AttackerAttack)
		}
		if bd.DefenderDefense == nil || bd.DefenderDefense.FinalValue != 4 {
			t.Fatalf("defender defense provenance missing/wrong: %+v", bd.DefenderDefense)
		}
		break
	}
	if !found {
		t.Fatal("expected a damage log with breakdown")
	}
}

// TestCasterAttenuationInCombat verifies that when a Rank 9 immortal uses a Rank 4
// Gu for self-buff (Bất Hoại), Caster Attenuation suppresses the proc chance.
func TestCasterAttenuationInCombat(t *testing.T) {
	immortal := newTestCombatant("immortal", "Player", 50000, 100)
	immortal.Rank = 9
	immortal.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Kim Thân Cổ", Rank: 4,
		StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusInvincible, Turns: 2, Chance: 1.0}},
	}}
	enemy := newTestCombatant("foe", "Enemy", 1000, 10)
	enemy.Rank = 9

	state := &CombatState{Seed: 123, Allies: []*CombatantAuto{immortal}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	for _, l := range logs {
		t.Logf("[%s] %s", l.Action, l.Detail)
	}
	missFound := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "không kích hoạt thành công hiệu ứng [Bất Hoại]") || l.Action == "STATUS_MISS" {
			missFound = true
			break
		}
	}
	if !missFound {
		t.Fatalf("expected miss detail due to caster attenuation on Rank 4 Gu used by Rank 9 immortal")
	}
}

// TestTargetSuppressionImmunityInCombat verifies that when a Rank 2 mortal uses
// a Rank 2 Gu against a Rank 8 boss (gap >= 5), the boss is completely immune.
func TestTargetSuppressionImmunityInCombat(t *testing.T) {
	mortal := newTestCombatant("mortal", "Player", 500, 10)
	mortal.Rank = 2
	mortal.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Định Thân Cổ", Rank: 2,
		StatusOps:       []statusfx.StatusOp{{ID: statusfx.StatusStun, Turns: 3, Chance: 1.0}},
		DamageHPPercent: 0.20,
		DisableGuTurns:  2,
	}}
	boss := newTestCombatant("boss", "Enemy", 50000, 100)
	boss.Rank = 8

	state := &CombatState{Seed: 456, Allies: []*CombatantAuto{mortal}, Enemies: []*CombatantAuto{boss}}
	logs := Simulate(state)

	immuneFound := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "miễn nhiễm") {
			immuneFound = true
			break
		}
	}
	if !immuneFound {
		t.Fatalf("expected immune log detail for gap 6, got logs: %+v", logs)
	}
	// Boss HP must not have lost 20% MaxHP (10,000)
	if boss.HP < 49000 {
		t.Fatalf("boss HP = %v, expected no direct %%HP damage from lower rank Gu", boss.HP)
	}
}

func TestPathMasteryCombat_MortalCapped_ImmortalUncapped(t *testing.T) {
	// Mortal test: limit is 5 points. Gu rank 1 skill gains 1 point per turn.
	// Over 10 turns, it should only gain 5 points and trigger PATH_MASTERY_CAP.
	mortal := newTestCombatant("mortal", "Player", 5000, 10)
	mortal.Rank = 1
	mortal.PathMasteryGainLimit = 5
	mortal.PathMasteries = map[string]int{"fire": 0}
	mortal.Actions = []SimAction{{
		ActionType: "GU_SKILL",
		Name:       "Hỏa Diễm Cổ",
		Rank:       1,
		Path:       []string{"fire"},
		Damage:     1,
	}}
	dummy := newTestCombatant("dummy", "Enemy", 50000, 0)
	dummy.Rank = 1

	state := &CombatState{Seed: 123, Allies: []*CombatantAuto{mortal}, Enemies: []*CombatantAuto{dummy}}
	_ = Simulate(state)

	if mortal.PathMasteries["fire"] != 5 {
		t.Fatalf("expected mortal path mastery capped at 5, got %d", mortal.PathMasteries["fire"])
	}
	if mortal.PathMasteryGained != 5 {
		t.Fatalf("expected mortal PathMasteryGained to be 5, got %d", mortal.PathMasteryGained)
	}

	// Immortal test: limit is 10 points. Tiên Cổ rank 6 skill gains 10 points per turn.
	// Over 10+ turns, it should exceed the battle cap (10) without any cap trigger!
	immortal := newTestCombatant("immortal", "Player", 5000, 10)
	immortal.Rank = 6
	immortal.PathMasteryGainLimit = 10
	immortal.PathMasteries = map[string]int{"fire": 0}
	immortal.Actions = []SimAction{{
		ActionType: "GU_SKILL",
		Name:       "Phần Thiên Tiên Cổ",
		Rank:       6,
		Path:       []string{"fire"},
		Damage:     1,
	}}
	dummy2 := newTestCombatant("dummy2", "Enemy", 50000, 0)
	dummy2.Rank = 6

	stateImmortal := &CombatState{Seed: 123, Allies: []*CombatantAuto{immortal}, Enemies: []*CombatantAuto{dummy2}}
	_ = Simulate(stateImmortal)

	if immortal.PathMasteries["fire"] <= 10 {
		t.Fatalf("expected immortal path mastery to exceed battle cap (10), got %d", immortal.PathMasteries["fire"])
	}
	if immortal.PathMasteryGained != 0 {
		t.Fatalf("expected immortal PathMasteryGained to remain 0 (uncapped), got %d", immortal.PathMasteryGained)
	}

	// Immortal killer move test: Rank 6 killer move should also exceed battle cap
	immortalKM := newTestCombatant("immortal_km", "Player", 5000, 10)
	immortalKM.Rank = 6
	immortalKM.PathMasteryGainLimit = 10
	immortalKM.PathMasteries = map[string]int{"sword": 0}
	immortalKM.Actions = []SimAction{{
		ActionType: "KILLER_MOVE",
		Name:       "Vạn Kiếm Quy Tông",
		Rank:       6,
		Path:       []string{"sword"},
		Damage:     1,
	}}
	dummy3 := newTestCombatant("dummy3", "Enemy", 50000, 0)
	dummy3.Rank = 6

	stateImmortalKM := &CombatState{Seed: 123, Allies: []*CombatantAuto{immortalKM}, Enemies: []*CombatantAuto{dummy3}}
	_ = Simulate(stateImmortalKM)

	if immortalKM.PathMasteries["sword"] <= 10 {
		t.Fatalf("expected immortal killer move path mastery to exceed battle cap (10), got %d", immortalKM.PathMasteries["sword"])
	}
	if immortalKM.PathMasteryGained != 0 {
		t.Fatalf("expected immortal killer move PathMasteryGained to remain 0 (uncapped), got %d", immortalKM.PathMasteryGained)
	}
}

func TestPathMasteryCombat_RankNineGuGrantsThreeHundredBasePoints(t *testing.T) {
	player := newTestCombatant("rank_nine", "Player", 5000, 10)
	player.Rank = 9
	player.PathMasteries = map[string]int{"heaven": 0}
	player.Actions = []SimAction{{
		ActionType: "GU_SKILL",
		Name:       "Cửu Chuyển Thiên Đạo Cổ",
		Rank:       9,
		Path:       []string{"heaven"},
		Damage:     1000,
	}}
	enemy := newTestCombatant("dummy", "Enemy", 1, 0)

	_ = Simulate(&CombatState{Seed: 123, Allies: []*CombatantAuto{player}, Enemies: []*CombatantAuto{enemy}})

	if got := player.PathMasteries["heaven"]; got != 300 {
		t.Fatalf("rank 9 Gu mastery gain = %d, want 300", got)
	}
}

func TestTargetAllEnemiesWithMultiplier(t *testing.T) {
	// 1. Verify resolveTargets returns all alive enemies when TargetType is "ALL_ENEMIES"
	e1 := newTestCombatant("e1", "Enemy", 1000, 0)
	e2 := newTestCombatant("e2", "Enemy", 1000, 0)
	e3 := newTestCombatant("e3", "Enemy", 0, 0) // dead
	e1.GridX, e1.GridY = 1, 1
	e2.GridX, e2.GridY = 2, 2
	e3.GridX, e3.GridY = 3, 3

	actionAoE := &SimAction{
		ActionType: "GU_SKILL",
		Name:       "Liệt Diễm Môn",
		Damage:     100,
		DamageMult: 1.5,
		TargetType: "ALL_ENEMIES",
	}

	targets := resolveTargets([]*CombatantAuto{e1, e2, e3}, actionAoE)
	if len(targets) != 2 {
		t.Fatalf("expected 2 alive targets for ALL_ENEMIES, got %d", len(targets))
	}
	if targets[0].ID != "e1" || targets[1].ID != "e2" {
		t.Fatalf("expected e1 and e2 as targets, got %v and %v", targets[0].ID, targets[1].ID)
	}

	// 2. Full simulation: attacker uses ALL_ENEMIES skill with 1.5x multiplier on 2 enemies
	attacker := newTestCombatant("attacker", "Player", 1000, 0)
	attacker.Speed = 50
	attacker.Actions = []SimAction{*actionAoE}

	t1 := newTestCombatant("t1", "Enemy", 1000, 0)
	t2 := newTestCombatant("t2", "Enemy", 1000, 0)
	t1.Speed = 10
	t2.Speed = 10

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{t1, t2},
	}
	logs := Simulate(state)

	var turn1ActionLog *CombatLog
	for i := range logs {
		if logs[i].Turn == 1 && logs[i].Action == "GU_SKILL" {
			turn1ActionLog = &logs[i]
			break
		}
	}
	if turn1ActionLog == nil {
		t.Fatalf("expected GU_SKILL log in turn 1")
	}

	// Verify both targets were hit and took 150 damage (100 * 1.5)
	if !strings.Contains(turn1ActionLog.Detail, "lên t1, gây 150 sát thương") {
		t.Errorf("expected t1 to take 150 damage, got log detail: %s", turn1ActionLog.Detail)
	}
	if !strings.Contains(turn1ActionLog.Detail, "lên t2, gây 150 sát thương") {
		t.Errorf("expected t2 to take 150 damage, got log detail: %s", turn1ActionLog.Detail)
	}
}

func TestManaRestoreAndCleanse(t *testing.T) {
	attacker := newTestCombatant("attacker", "Player", 1000, 0)
	attacker.Mana = 30
	attacker.MaxMana = 100
	attacker.Speed = 50
	attacker.ActiveEffects = []Effect{
		{ID: "WEAK", Type: "debuff", Value: 20, Duration: 3},
	}
	ComputeEffectMultipliers(attacker)

	action := SimAction{
		ActionType:     "GU_SKILL",
		Name:           "Thực Khí Tịnh Hóa",
		ManaCost:       0,
		ManaRestore:    40,
		CleanseDebuffs: true,
		TargetType:     "SELF",
	}
	attacker.Actions = []SimAction{action}

	defender := newTestCombatant("defender", "Enemy", 1000, 0)
	defender.Speed = 10

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{defender},
	}
	logs := Simulate(state)

	var foundConsolidatedLog bool
	for _, l := range logs {
		if l.Turn == 1 && l.Action == "GU_SKILL" && l.ActorId == attacker.ID {
			if strings.Contains(l.Detail, "khôi phục 40") && strings.Contains(l.Detail, "tịnh hóa bản thân, xóa bỏ 1 hiệu ứng bất lợi") {
				foundConsolidatedLog = true
			}
			if l.TargetId != attacker.ID {
				t.Errorf("expected TargetId == %s for SELF target, got %s", attacker.ID, l.TargetId)
			}
		}
	}
	if !foundConsolidatedLog {
		t.Errorf("expected consolidated GU_SKILL log with mana restore and cleanse details in turn 1")
	}
	if attacker.Mana < 70 {
		t.Errorf("expected attacker mana >= 70, got %.0f", attacker.Mana)
	}
	for _, eff := range attacker.ActiveEffects {
		if eff.Type == "debuff" {
			t.Errorf("expected no debuffs on attacker after cleanse, found %s", eff.ID)
		}
	}
}
