package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func reverseCalamityMech() CombatMechanic {
	return CombatMechanic{Trigger: "COMBAT_PASSIVE", ActionCode: "REVERSE_CALAMITY"}
}

func suppressPriorityMech() CombatMechanic {
	return CombatMechanic{Trigger: "ON_BATTLE_START", ActionCode: "SUPPRESS_PRIORITY"}
}

func logActionCount(logs []CombatLog, action string) int {
	n := 0
	for _, l := range logs {
		if l.Action == action {
			n++
		}
	}
	return n
}

// logDetailCount counts logs whose Detail contains substr. Per-action logs are
// consolidated into single entries, so mechanic traces live inside Detail.
func logDetailCount(logs []CombatLog, substr string) int {
	n := 0
	for _, l := range logs {
		if strings.Contains(l.Detail, substr) {
			n++
		}
	}
	return n
}

// TestReverseCalamityBlocksHostileStatusButNotDamage verifies Nghịch Chuyển Cổ:
// hostile statuses aimed at the holder are reversed while raw damage passes.
func TestReverseCalamityBlocksHostileStatusButNotDamage(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 10)
	ally.Speed = 1
	ally.Mechanics = []CombatMechanic{reverseCalamityMech()}
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 10}}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 0)
	enemy.Speed = 50
	enemy.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Trấn Hồn Cổ", Damage: 100,
		StatusOps: []statusfx.StatusOp{
			{ID: statusfx.StatusStun, Value: 1, Turns: 1},
			{ID: statusfx.StatusPoison, Value: 10, Turns: 2},
		},
	}}
	state := &CombatState{Seed: 7, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if got := logDetailCount(logs, "đảo nghịch"); got < 2 {
		t.Fatalf("expected >=2 reversal traces (stun+poison), got %d", got)
	}
	if state.Allies[0].HP >= 1000 {
		t.Fatalf("expected raw damage to pass through reversal, ally HP=%v", state.Allies[0].HP)
	}
	if got := logActionCount(logs, "BASIC_ATTACK"); got == 0 {
		t.Fatal("expected holder to keep acting (no stun skip)")
	}
	for _, l := range logs {
		if l.ActorId == "ally" && l.Action == "STUNNED" {
			t.Fatal("holder must never be stunned through reversal")
		}
	}
}

// TestReverseCalamityBlocksGuSeal verifies disable_gu_next_turn is reversed.
func TestReverseCalamityBlocksGuSeal(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 10)
	ally.Speed = 1
	ally.Mechanics = []CombatMechanic{reverseCalamityMech()}
	ally.Actions = []SimAction{{ActionType: "GU_SKILL", Name: "Đánh", Damage: 10}}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 0)
	enemy.Speed = 50
	enemy.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Phong Cổ", Damage: 10, DisableGuTurns: 2,
	}}
	state := &CombatState{Seed: 7, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if got := logDetailCount(logs, "đảo nghịch phong ấn"); got == 0 {
		t.Fatal("expected reversal trace for sealed strike")
	}
	if got := logActionCount(logs, "GU_SEALED"); got != 0 {
		t.Fatalf("expected no GU_SEALED enforcement on holder, got %d", got)
	}
}

// TestSuppressPrioritySeizesFirstOnSuccess verifies Cấm Cổ: when the
// suppression negates an enemy priority effect that would actually trigger,
// the fastest living holder acts first on turn 1 (even when slower).
func TestSuppressPrioritySeizesFirstOnSuccess(t *testing.T) {
	build := func(allySpeed, enemySpeed float64, allySuppress, enemySuppress bool, allyLife, enemyLife float64, allySeize bool) (string, []CombatLog) {
		ally := newTestCombatant("ally", "Player", 100000, 1)
		ally.Speed = allySpeed
		ally.Lifespan = allyLife
		if allySuppress {
			ally.Mechanics = []CombatMechanic{suppressPriorityMech()}
		}
		if allySeize {
			ally.Mechanics = append(ally.Mechanics, CombatMechanic{
				Trigger: "ON_BATTLE_START", ActionCode: "SEIZE_INITIATIVE",
				Params: []byte(`{"lifespan_ratio":3}`),
			})
		}
		ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}
		enemy := newTestCombatant("enemy", "Enemy", 100000, 1)
		enemy.Speed = enemySpeed
		enemy.Lifespan = enemyLife
		enemy.Mechanics = []CombatMechanic{{
			Trigger: "ON_BATTLE_START", ActionCode: "SEIZE_INITIATIVE",
			Params: []byte(`{"lifespan_ratio":3}`),
		}}
		if enemySuppress {
			enemy.Mechanics = append(enemy.Mechanics, suppressPriorityMech())
		}
		enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}
		state := &CombatState{Seed: 3, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
		logs := Simulate(state)
		first := ""
		for _, l := range logs {
			if l.Turn == 1 && l.Action != "TURN_START" && l.ActorId != "" {
				first = l.ActorId
				break
			}
		}
		return first, logs
	}

	// Sanity: without suppression the slower enemy seizes.
	if first, _ := build(50, 1, false, false, 100, 100, false); first != "enemy" {
		t.Fatalf("sanity: expected enemy to seize without suppression, first=%q", first)
	}
	// Slower holder suppresses an eligible seize: holder goes first anyway.
	if first, logs := build(1, 50, true, false, 100, 100, false); first != "ally" {
		t.Fatalf("expected suppressor first despite lower speed, first=%q", first)
	} else if logActionCount(logs, "SEIZE_INITIATIVE") != 0 {
		t.Fatal("expected no SEIZE_INITIATIVE log when suppressed")
	} else if logActionCount(logs, "PRIORITY_SUPPRESSED") == 0 {
		t.Fatal("expected PRIORITY_SUPPRESSED log")
	} else if logActionCount(logs, "SUPPRESS_PRIORITY") == 0 {
		t.Fatal("expected SUPPRESS_PRIORITY first-strike log")
	}
	// Mirror: both sides suppress and both have an eligible seize — the
	// faster suppressor goes first.
	if first, logs := build(50, 1, true, true, 100, 100, true); first != "ally" {
		t.Fatalf("expected faster suppressor first in mirror, first=%q", first)
	} else if logActionCount(logs, "SEIZE_INITIATIVE") != 0 {
		t.Fatal("expected no SEIZE_INITIATIVE log in mirror suppression")
	} else if logActionCount(logs, "SUPPRESS_PRIORITY") == 0 {
		t.Fatal("expected SUPPRESS_PRIORITY log in mirror suppression")
	}
	if first, _ := build(1, 50, true, true, 100, 100, true); first != "enemy" {
		t.Fatalf("expected faster enemy suppressor first in mirror, first=%q", first)
	}
}

// TestSuppressPriorityNoJumpWithoutTrigger verifies a suppression with
// nothing to negate grants no first strike: enemy seize ineligible on
// lifespan, so turn 1 stays in speed order with no SUPPRESS_PRIORITY log.
func TestSuppressPriorityNoJumpWithoutTrigger(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100000, 1)
	ally.Speed = 1
	ally.Lifespan = 10000
	ally.Mechanics = []CombatMechanic{suppressPriorityMech()}
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 1}}
	enemy := newTestCombatant("enemy", "Enemy", 100000, 1)
	enemy.Speed = 50
	enemy.Lifespan = 100
	enemy.Mechanics = []CombatMechanic{{
		Trigger: "ON_BATTLE_START", ActionCode: "SEIZE_INITIATIVE",
		Params: []byte(`{"lifespan_ratio":3}`),
	}}
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}
	state := &CombatState{Seed: 3, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	first := ""
	for _, l := range logs {
		if l.Turn == 1 && l.Action != "TURN_START" && l.ActorId != "" {
			first = l.ActorId
			break
		}
	}
	if first != "enemy" {
		t.Fatalf("expected speed order when nothing to suppress, first=%q", first)
	}
	if logActionCount(logs, "SUPPRESS_PRIORITY") != 0 {
		t.Fatal("expected no SUPPRESS_PRIORITY log without a successful suppression")
	}
}

// TestSuppressPriorityBlocksFirstTurnEvade verifies Cấm Cổ lets the holder's
// opening strike land through Định Tiên Du.
func TestSuppressPriorityBlocksFirstTurnEvade(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 100)
	ally.Speed = 50
	ally.Mechanics = []CombatMechanic{suppressPriorityMech()}
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 100}}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 0)
	enemy.Speed = 1
	enemy.Mechanics = []CombatMechanic{{Trigger: "BEFORE_HIT", ActionCode: "FIRST_TURN_EVADE"}}
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}
	state := &CombatState{Seed: 9, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	if logDetailCount(logs, "trấn áp") == 0 {
		t.Fatal("expected suppression trace against first-turn evade")
	}
	if logActionCount(logs, "SUPPRESS_PRIORITY") == 0 {
		t.Fatal("expected SUPPRESS_PRIORITY first-strike log against evade carrier")
	}
	if logActionCount(logs, "EVADE") != 0 {
		t.Fatal("expected no EVADE log when suppressed")
	}
	if state.Enemies[0].HP >= 1000 {
		t.Fatalf("expected opening strike to land, enemy HP=%v", state.Enemies[0].HP)
	}
}
