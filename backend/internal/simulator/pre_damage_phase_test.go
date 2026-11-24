package simulator

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestSplitStatusOpsByPhase(t *testing.T) {
	pre, post := splitStatusOpsByPhase([]statusfx.StatusOp{
		{ID: statusfx.StatusArmorBreak, Value: 40, Turns: 3, Phase: "PRE_DAMAGE"},
		{ID: statusfx.StatusBurn, Value: 8, Turns: 3},
		{ID: statusfx.StatusWeak, Value: 20, Turns: 2, Phase: "pre_damage"},
		{ID: statusfx.StatusStun, Turns: 1, Phase: "PRE_CHAIN"},
	})
	if len(pre) != 3 {
		t.Fatalf("pre = %+v; want ARMOR_BREAK + WEAK + STUN", pre)
	}
	if len(post) != 1 || post[0].ID != statusfx.StatusBurn {
		t.Fatalf("post = %+v; want BURN only", post)
	}
}

func TestPreChainStatusOperationsApplyAtTheChainRiderSeam(t *testing.T) {
	actor := newTestCombatant("actor", "Actor", 1000, 100)
	target := newTestCombatant("target", "Target", 1000, 50)
	actor.Rank, target.Rank = 1, 1
	actor.Actions = []SimAction{{StatusOps: []statusfx.StatusOp{{
		ID: statusfx.StatusStun, Turns: 1, Chance: 1, ApplicationMode: "DIRECT", Phase: "PRE_CHAIN",
	}}}}
	var logs []CombatLog
	runner := &combatActionRunner{rng: rand.New(rand.NewSource(7)), turn: 1, turnLogs: &logs}
	runner.dispatchPhase(combatPhasePreChain, combatPhaseContext{
		actor: actor, action: &actor.Actions[0], firstTarget: target, targets: []*CombatantAuto{target},
	})
	if !hasStatus(target, statusfx.StatusStun) {
		t.Fatalf("PRE_CHAIN status was not applied before chain riders: %+v", target.ActiveEffects)
	}
}

func TestDirectStatusCanUseAuthoredProcChance(t *testing.T) {
	actor := newTestCombatant("actor", "Actor", 1000, 100)
	target := newTestCombatant("target", "Target", 1000, 50)
	actor.Rank, target.Rank = 1, 1
	var logs []CombatLog
	applyStatusOpsRaw(actor, []*CombatantAuto{target}, []statusfx.StatusOp{{
		ID: statusfx.StatusStun, Turns: 1, Chance: 0.01, ApplicationMode: "DIRECT",
	}}, 1, 1, rand.New(rand.NewSource(1)), &logs)
	if target.IsStunned {
		t.Fatal("DIRECT status ignored an authored chance that should fail for seed 1")
	}
	if len(logs) != 1 || logs[0].Action != "STATUS_MISS" {
		t.Fatalf("expected one deterministic status miss log, got %+v", logs)
	}
}

// TestPreDamageStatusOpsKeepSameActionDamage locks the parity rule behind the
// PRE_DAMAGE phase: migrated hooks must not change the damage of the strike
// they precede, because the legacy native hooks never recomputed derived
// multipliers for the same action.
func TestPreDamageStatusOpsKeepSameActionDamage(t *testing.T) {
	totalDamage := func(phase string) (float64, bool) {
		ally := newTestCombatant("ally", "Player", 1200, 100)
		enemy := newTestCombatant("enemy", "Enemy", 1200, 60)
		enemy.MentalResistance = 0
		enemy.CCResist = 0
		ally.Speed, enemy.Speed = 30, 1
		ally.Actions = []SimAction{{
			ActionType: "GU_SKILL", Name: "Cổ thử", SourceID: "GU_PHASE", Rank: 3, Damage: 45,
			StatusOps: []statusfx.StatusOp{{
				ID: statusfx.StatusArmorBreak, Value: 40, Turns: 3,
				ApplicationMode: "DIRECT", Phase: phase,
			}},
		}}
		state := &CombatState{Seed: 3, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
		logs := Simulate(state)
		dealt := 0.0
		applied := false
		for _, l := range logs {
			if l.Action == "GU_SKILL" {
				dealt += l.Damage
				if strings.Contains(l.Detail, "Phá Giáp") {
					applied = true
				}
			}
		}
		return dealt, applied
	}

	preDamage, preApplied := totalDamage("PRE_DAMAGE")
	postDamage, _ := totalDamage("")
	if preDamage != postDamage {
		t.Fatalf("PRE_DAMAGE dealt %v, post-damage dealt %v; want identical", preDamage, postDamage)
	}
	if !preApplied {
		t.Fatal("PRE_DAMAGE ARMOR_BREAK never applied")
	}
}
