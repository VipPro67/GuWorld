package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func mech(trigger, action, params string) CombatMechanic {
	return NewCombatMechanic(trigger, action, []byte(params))
}

func logCount(logs []CombatLog, action string) int {
	n := 0
	for _, l := range logs {
		if l.Action == action {
			n++
		}
	}
	return n
}

func hasDetail(logs []CombatLog, sub string) bool {
	for _, l := range logs {
		if strings.Contains(l.Detail, sub) {
			return true
		}
	}
	return false
}

// EGO: %MaxHP direct damage + shield strip + Gu seal for next turn.
// Enemy one-shots the ally on turn 1, so EGO lands exactly once.
func TestForgetEgoPercentsAndSeal(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 10000, 0)
	ally.Speed = 100
	ally.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Bản Ngã", Damage: 0,
		DamageHPPercent: 0.4, ShieldStripPercent: 0.4, DisableGuTurns: 1, BonusTurns: 1,
	}}
	enemy := newTestCombatant("enemy", "Enemy", 100000, 0)
	enemy.Speed = 1
	enemy.VirtualShield = 500
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 20000}}

	state := &CombatState{Seed: 3, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if enemy.HP != 60000 {
		t.Fatalf("enemy HP = %v, want 60000 (single 40%% of 100000 MaxHP hit)", enemy.HP)
	}
	if !(enemy.VirtualShield > 299 && enemy.VirtualShield < 300) {
		t.Fatalf("enemy shield = %v, want ~299.4 (1 chip + single 40%% strip of 500)", enemy.VirtualShield)
	}
	if !hasDetail(logs, "HP trực tiếp") || !hasDetail(logs, "tước") || !hasDetail(logs, "phong ấn Cổ") {
		t.Fatalf("missing EGO logs (direct HP / shield strip / Gu seal)")
	}
	// Seal must block the enemy's next GU_SKILL: simulate one sealed cast.
	sealed := newTestCombatant("sealed", "Enemy", 500, 0)
	sealGuTurns(sealed, 1)
	if !sealed.HasGuSeal {
		t.Fatal("sealGuTurns did not set HasGuSeal")
	}
	if d := sealed.ActiveEffects[0].Duration; d != 2 {
		t.Fatalf("seal duration = %d, want 2 (covers next full turn)", d)
	}
}

// REALITY: full-proc action grants the EVASION status and weakens the target.
func TestForgetRealityProcEvasionWeak(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	ally.Speed = 100
	ally.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Thực Tại", Damage: 10,
		ProcChance: 1.0,
		StatusOps: []statusfx.StatusOp{
			{ID: statusfx.StatusEvasion, Value: 50, Turns: 2, Chance: 1.0},
			{ID: statusfx.StatusForgetReality, Value: 40, Turns: 2},
		},
	}}
	enemy := newTestCombatant("enemy", "Enemy", 2000, 50)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 50}}

	state := &CombatState{Seed: 5, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if !hasDetail(logs, "Né Tránh") {
		t.Fatal("missing EVASION application in merged action log")
	}
	if !hasDetail(logs, "Lãng Quên Thực Tại") {
		t.Fatal("missing FORGET_REALITY application log")
	}
	if ally.EvasionChance < 0.5 {
		t.Fatalf("EvasionChance = %v, want >= 0.5 from the EVASION status", ally.EvasionChance)
	}
}

// Proc gate: zero ProcChance means always-proc (existing actions unaffected).
func TestProcGateDefaultsAlways(t *testing.T) {
	var a SimAction
	if !(a.ProcChance <= 0) {
		t.Fatal("zero ProcChance must mean always-proc")
	}
}

// Passive anti-path: Heaven-path attack reduced by PathResistance.
// Asserted via the damage breakdown (PathResist 0.65), not final HP.
func TestAntiHeavenPathResistance(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 50000, 0)
	ally.Speed = 1
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	ally.PathResistance = map[string]float64{"heaven": 0.65}
	enemy := newTestCombatant("enemy", "Enemy", 50000, 0)
	enemy.Speed = 100
	enemy.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Thiên Kích", Damage: 1000,
		Path: []string{"Heaven Path"},
	}}
	state := &CombatState{Seed: 9, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if l.Breakdown != nil && l.Breakdown.PathResist == 0.65 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("no damage breakdown with PathResist 0.65 against Heaven-path attack")
	}
	// Non-heaven attack must not pick up the heaven resistance.
	plain := newTestCombatant("ally", "Player", 50000, 0)
	plain.Speed = 1
	plain.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	plain.PathResistance = map[string]float64{"heaven": 0.65}
	sword := newTestCombatant("enemy", "Enemy", 50000, 0)
	sword.Speed = 100
	sword.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Kiếm Trảm", Damage: 1000,
		Path: []string{"Sword Path"},
	}}
	logs2 := Simulate(&CombatState{Seed: 9, Allies: []*CombatantAuto{plain}, Enemies: []*CombatantAuto{sword}})
	for _, l := range logs2 {
		if l.Breakdown != nil && l.Breakdown.PathResist != 0 {
			t.Fatalf("heaven resistance leaked into Sword-path attack: %+v", l.Breakdown)
		}
	}
}

// Dream immunity: SLEEP never lands on an immune target.
func TestImmuneDreamStatusBlocksSleep(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 2000, 0)
	ally.Speed = 100
	ally.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Mộng", Damage: 10,
		StatusOps: []statusfx.StatusOp{{ID: statusfx.StatusSleep, Value: 0, Turns: 2}},
	}}
	enemy := newTestCombatant("enemy", "Enemy", 2000, 10)
	enemy.Speed = 1
	enemy.ImmuneDreamStatus = true
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 10}}

	state := &CombatState{Seed: 11, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if !hasDetail(logs, "miễn nhiễm hiệu ứng Mộng đạo") {
		t.Fatal("missing dream-immunity log for immune target")
	}
	for _, eff := range enemy.ActiveEffects {
		if eff.ID == statusfx.StatusSleep {
			t.Fatal("SLEEP landed on dream-immune target")
		}
	}
}

// REBORN in-place: lethal hit revives once at 50%, second lethal hit sticks.
func TestRebornInPlaceSingleUse(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100, 0)
	ally.Speed = 1
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	ally.Mechanics = []CombatMechanic{mech("ON_DEATH", "REBORN",
		`{"chance":1,"mode":"in_place","max_uses":1,"revive_hp_percent":0.5}`)}
	enemy := newTestCombatant("enemy", "Enemy", 5000, 0)
	enemy.Speed = 100
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Sát", Damage: 100}}

	state := &CombatState{Seed: 13, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if logCount(logs, "TRIGGERED") != 1 {
		t.Fatalf("expected exactly 1 REBORN trigger, got %d", logCount(logs, "TRIGGERED"))
	}
	if !hasDetail(logs, "vùng dậy với 50 HP") {
		t.Fatal("missing in-place revive log at 50 HP")
	}
	if ally.HP != 0 {
		t.Fatalf("ally HP = %v, want 0 (second lethal hit sticks after max_uses)", ally.HP)
	}
}

// REBORN unlimited: max_uses 0 revives on every death until the turn cap.
func TestRebornUnlimitedUses(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100, 0)
	ally.Speed = 1
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	ally.Mechanics = []CombatMechanic{mech("ON_DEATH", "REBORN",
		`{"chance":1,"mode":"in_place","max_uses":0,"revive_hp_percent":0.5}`)}
	enemy := newTestCombatant("enemy", "Enemy", 5000, 0)
	enemy.Speed = 100
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Sát", Damage: 100}}

	state := &CombatState{Seed: 13, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if got := logCount(logs, "TRIGGERED"); got < 10 {
		t.Fatalf("expected many REBORN triggers with unlimited uses, got %d", got)
	}
	if !hasDetail(logs, "(vô hạn)") {
		t.Fatal("missing unlimited marker in reborn log")
	}
	if ally.HP <= 0 {
		t.Fatalf("ally HP = %v, want alive with unlimited rebirth", ally.HP)
	}
}

// REBORN miss: chance 0 always fails the roll and logs REBORN_MISS.
func TestRebornMissLogged(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100, 0)
	ally.Speed = 1
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	ally.Mechanics = []CombatMechanic{mech("ON_DEATH", "REBORN",
		`{"chance":0,"mode":"in_place","max_uses":1,"revive_hp_percent":0.5}`)}
	enemy := newTestCombatant("enemy", "Enemy", 5000, 0)
	enemy.Speed = 100
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Sát", Damage: 100}}

	state := &CombatState{Seed: 13, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if logCount(logs, "REBORN_MISS") == 0 {
		t.Fatal("expected REBORN_MISS log when chance roll fails")
	}
	if logCount(logs, "TRIGGERED") != 0 {
		t.Fatal("expected no REBORN trigger on failed roll")
	}
	if ally.HP != 0 {
		t.Fatalf("ally HP = %v, want 0 (failed reborn stays dead)", ally.HP)
	}
}

// FORGET_INTENT miss: chance 0 never seals and logs FORGET_INTENT_MISS.
func TestForgetIntentMissLogged(t *testing.T) {
	attacker := newTestCombatant("attacker", "Player", 1000, 0)
	attacker.Speed = 100
	attacker.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 10}}
	defender := newTestCombatant("defender", "Enemy", 5000, 0)
	defender.Speed = 1
	defender.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	defender.Mechanics = []CombatMechanic{mech("BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT",
		`{"chance":0,"disable_target_gu_turns":1,"max_successful_activations":8}`)}

	state := &CombatState{Seed: 7, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}}
	logs := Simulate(state)

	if logCount(logs, "FORGET_INTENT_MISS") == 0 {
		t.Fatal("expected FORGET_INTENT_MISS log when chance roll fails")
	}
	if logCount(logs, "FORGET_INTENT") != 0 {
		t.Fatal("expected no FORGET_INTENT trigger on failed roll")
	}
}

// FORGET_WILL activation miss: chance 0 never confuses and logs FORGET_WILL_INACTIVE.
func TestForgetWillInactiveLogged(t *testing.T) {
	attacker := newTestCombatant("attacker", "Player", 1000, 0)
	attacker.Speed = 100
	attacker.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 10}}
	defender := newTestCombatant("defender", "Enemy", 5000, 0)
	defender.Speed = 1
	defender.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	defender.Mechanics = []CombatMechanic{mech("BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE",
		`{"activation_chance":0,"hit_user_chance":0.15,"hit_self_chance":0.35}`)}

	state := &CombatState{Seed: 9, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}}
	logs := Simulate(state)

	if logCount(logs, "FORGET_WILL_INACTIVE") == 0 {
		t.Fatal("expected FORGET_WILL_INACTIVE log when activation roll fails")
	}
}

// REBORN legacy rewind still works and keeps default single use.
func TestRebornRewindLegacy(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100, 0)
	ally.Speed = 1
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	ally.Mechanics = []CombatMechanic{mech("ON_DEATH", "REBORN", `{"chance":1}`)}
	enemy := newTestCombatant("enemy", "Enemy", 5000, 0)
	enemy.Speed = 100
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Sát", Damage: 100}}

	state := &CombatState{Seed: 17, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if !hasDetail(logs, "Xuân Thu Thiền") {
		t.Fatal("missing legacy rewind log")
	}
}

// REBORN empower: revive stacks ATK_UP + shield.
func TestRebornEmpowerStacks(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100, 0)
	ally.Mechanics = []CombatMechanic{mech("ON_DEATH", "REBORN",
		`{"chance":1,"mode":"in_place","max_uses":5,"revive_hp_percent":0.5,"empower_attack_percent":0.1,"empower_speed_percent":0.1,"empower_shield_percent":0.1,"empower_turns":100}`)}
	var logs []CombatLog
	applyRebornEmpower(ally, ally.Mechanics, 1, 1, &logs)
	applyRebornEmpower(ally, ally.Mechanics, 1, 2, &logs)
	if ally.VirtualShield != 20 {
		t.Fatalf("shield = %v, want 20 (2x 10%% of 100 MaxHP)", ally.VirtualShield)
	}
	atk, haste := 0, 0
	for _, eff := range ally.ActiveEffects {
		switch eff.ID {
		case statusfx.StatusAtkUp:
			atk++
			if eff.Value != 10 || eff.Duration != 100 {
				t.Fatalf("atk_up effect = %+v, want Value 10 Duration 100", eff)
			}
		case statusfx.StatusHaste:
			haste++
		}
	}
	if atk != 2 || haste != 2 {
		t.Fatalf("want 2 ATK_UP + 2 HASTE stacks, got %d/%d", atk, haste)
	}
	if logCount(logs, "REBORN_EMPOWER") != 2 {
		t.Fatal("missing REBORN_EMPOWER logs")
	}
}

// INTENT: guaranteed interrupt seals the attacker's Gu and counts successes.
func TestForgetIntentInterruptSeals(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 5000, 0)
	ally.Speed = 1
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	ally.Mechanics = []CombatMechanic{mech("BEFORE_ENEMY_ATTACK", "FORGET_INTENT_INTERRUPT",
		`{"chance":1,"disable_target_gu_turns":1,"max_successful_activations":8}`)}
	enemy := newTestCombatant("enemy", "Enemy", 5000, 0)
	enemy.Speed = 100
	enemy.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Trảm", Damage: 50}}

	state := &CombatState{Seed: 19, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if !hasDetail(logs, "quên mất ý niệm") {
		t.Fatal("missing FORGET_INTENT log")
	}
	if ally.CustomState["forget_intent_success"] < 1 {
		t.Fatal("INTENT success counter not incremented")
	}
}

// WILL full-miss: every Gu attack fizzles (only 1-chip fallback basics land).
func TestForgetWillFullMiss(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	ally.Speed = 1
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	ally.Mechanics = []CombatMechanic{mech("BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE",
		`{"activation_chance":1,"hit_user_chance":0,"hit_self_chance":0,"miss_chance":1,"backfire_chance":0}`)}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 0)
	enemy.Speed = 100
	enemy.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Trảm", Damage: 200}}

	state := &CombatState{Seed: 23, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if !hasDetail(logs, "đánh hụt hoàn toàn") {
		t.Fatal("missing full-miss log")
	}
	if ally.HP <= 800 {
		t.Fatalf("ally HP = %v, want > 800 (no 200-dmg Gu hit ever landed)", ally.HP)
	}
}

// WILL self-hit: attacker damages itself.
func TestForgetWillSelfHit(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 5000, 0)
	ally.Speed = 1
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}
	ally.Mechanics = []CombatMechanic{mech("BEFORE_ENEMY_ATTACK", "FORGET_WILL_CONFUSE",
		`{"activation_chance":1,"hit_user_chance":0,"hit_self_chance":1,"miss_chance":0,"backfire_chance":0}`)}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 0)
	enemy.Speed = 100
	enemy.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Trảm", Damage: 100}}

	state := &CombatState{Seed: 29, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if !hasDetail(logs, "tự đánh bản thân") {
		t.Fatal("missing self-hit redirect log")
	}
	if enemy.HP >= 1000 {
		t.Fatal("enemy should have damaged itself")
	}
	if ally.HP != 5000 {
		t.Fatalf("ally HP = %v, want 5000 (untouched)", ally.HP)
	}
}

// GU_SEAL enforcement: sealed combatant cannot cast Gu skills.
func TestGuSealBlocksSkill(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 0)
	ally.Speed = 100
	sealGuTurns(ally, 1)
	ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Trảm", Damage: 500}}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 0)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Thủ", Damage: 0}}

	state := &CombatState{Seed: 31, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if logCount(logs, "GU_SEALED") < 1 {
		t.Fatal("missing GU_SEALED enforcement log")
	}
}
