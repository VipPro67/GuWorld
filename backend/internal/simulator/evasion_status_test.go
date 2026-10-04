package simulator

import (
	"math/rand"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// The EVASION status is the only dodge source in combat. These tests pin the
// spec's contract: strength comes from op.Value (percent), stacks merge by max
// and are capped, the status is a self buff that ignores the CC resist gate,
// and the dodge roll no longer depends on any attacker accuracy stat.

// applyEvasionOps applies self-targeted ops to c. guRank matches c.Rank so the
// Caster Attenuation branch stays out of the way.
func applyEvasionOps(c *CombatantAuto, ops []statusfx.StatusOp, seed int64) []CombatLog {
	logs := make([]CombatLog, 0)
	applyStatusOps(c, []*CombatantAuto{c}, ops, c.Rank, 1, rand.New(rand.NewSource(seed)), &logs)
	return logs
}

func TestEvasionStatusSetsEvasionChance(t *testing.T) {
	c := newTestCombatant("c", "Player", 1000, 0)
	applyEvasionOps(c, []statusfx.StatusOp{{
		ID: statusfx.StatusEvasion, Value: 30, Turns: 2, Chance: 1.0,
	}}, 1)

	if got := c.EvasionChance; got != 0.30 {
		t.Fatalf("EvasionChance = %v, want 0.30 from EVASION value 30", got)
	}
}

func TestEvasionStatusMergesByMaxNotSum(t *testing.T) {
	c := newTestCombatant("c", "Player", 1000, 0)
	applyEvasionOps(c, []statusfx.StatusOp{
		{ID: statusfx.StatusEvasion, Value: 30, Turns: 2, Chance: 1.0},
		{ID: statusfx.StatusEvasion, Value: 50, Turns: 2, Chance: 1.0},
	}, 1)

	if got := c.EvasionChance; got != 0.50 {
		t.Fatalf("EvasionChance = %v, want 0.50 (max merge, not 0.80)", got)
	}
}

func TestEvasionStatusCappedAt75Percent(t *testing.T) {
	c := newTestCombatant("c", "Player", 1000, 0)
	applyEvasionOps(c, []statusfx.StatusOp{{
		ID: statusfx.StatusEvasion, Value: 90, Turns: 2, Chance: 1.0,
	}}, 1)

	if got := c.EvasionChance; got != EvasionChanceCap {
		t.Fatalf("EvasionChance = %v, want cap %v", got, EvasionChanceCap)
	}
}

func TestEvasionStatusDefaultStrengthWhenValueMissing(t *testing.T) {
	c := newTestCombatant("c", "Player", 1000, 0)
	applyEvasionOps(c, []statusfx.StatusOp{{
		ID: statusfx.StatusEvasion, Turns: 1, Chance: 1.0,
	}}, 1)

	if got := c.EvasionChance; got != 0.30 {
		t.Fatalf("EvasionChance = %v, want the 30%% default", got)
	}
}

func TestEvasionStatusProcChanceCanMiss(t *testing.T) {
	spec := mustEvasionSpec(t)

	applied := newTestCombatant("hit", "Player", 1000, 0)
	logs := applyEvasionOps(applied, []statusfx.StatusOp{{
		ID: statusfx.StatusEvasion, Value: 25, Turns: 1, Chance: 1.0,
	}}, 7)
	if logCount(logs, "EVASION_APPLIED") != 1 {
		t.Fatalf("expected one EVASION_APPLIED log, got %+v", logs)
	}
	if applied.EvasionChance != 0.25 {
		t.Fatalf("EvasionChance = %v, want 0.25", applied.EvasionChance)
	}

	// A 1% proc chance must both land and miss across seeds, leaving no trace
	// on a miss (the roll happens before the effect is appended).
	landed, missed := 0, 0
	for seed := int64(1); seed <= 200; seed++ {
		c := newTestCombatant("c", "Player", 1000, 0)
		// CanApplySelfStatus adds +0.05 per Gu rank above 1, so rank 3 keeps the
		// authored 1% scaled to 11%.
		if statusfx.CanApplySelfStatus(rand.New(rand.NewSource(seed)), c.Rank, c.Rank, spec, 0.01) {
			landed++
		} else {
			missed++
		}
	}
	if landed == 0 {
		t.Fatal("low proc chance never landed across 200 seeds")
	}
	if missed == 0 {
		t.Fatal("low proc chance never missed across 200 seeds")
	}
}

// The self-target branch must not consult CC resist: EVASION is a buff on the
// caster, so a fully CC-resistant build still dodges.
func TestEvasionSelfTargetIgnoresResistGate(t *testing.T) {
	c := newTestCombatant("c", "Player", 1000, 0)
	c.MentalResistance = 0.75
	c.CCResist = 0.75
	c.Tenacity = 0.75
	applyEvasionOps(c, []statusfx.StatusOp{{
		ID: statusfx.StatusEvasion, Value: 40, Turns: 1, Chance: 1.0,
	}}, 1)

	if got := c.EvasionChance; got != 0.40 {
		t.Fatalf("EvasionChance = %v, want 0.40 despite full resist", got)
	}
}

// CategoryBuff keeps EVASION out of every dispel path.
func TestEvasionIsNotDispellable(t *testing.T) {
	c := newTestCombatant("c", "Player", 1000, 0)
	applyEvasionOps(c, []statusfx.StatusOp{{
		ID: statusfx.StatusEvasion, Value: 40, Turns: 3, Chance: 1.0,
	}}, 1)

	if isDispellableEffect(statusfx.StatusEvasion) {
		t.Fatal("EVASION must not be dispellable")
	}
	DispelActorStatuses(c, statusfx.CategoryBuff, 0)
	if c.EvasionChance != 0.40 {
		t.Fatalf("EvasionChance = %v after category dispel, want 0.40", c.EvasionChance)
	}
}

// The dodge roll must fire purely off EvasionChance. It previously subtracted a
// flat Accuracy stat (values up to 1000) from a 0..1 chance, so any Accuracy
// holder could never be evaded.
func TestEvadeMissRollsOffEvasionChanceAlone(t *testing.T) {
	evaded := 0
	for seed := int64(1); seed <= 40; seed++ {
		attacker := newTestCombatant("a", "Player", 1000, 0)
		attacker.Speed = 100
		attacker.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đấm", Damage: 50}}

		defender := newTestCombatant("d", "Enemy", 1000, 0)
		defender.Speed = 1
		defender.MaxHP = 100000
		defender.HP = 100000
		applyEvasionOps(defender, []statusfx.StatusOp{{
			ID: statusfx.StatusEvasion, Value: 50, Turns: 5, Chance: 1.0,
		}}, seed)

		logs := Simulate(&CombatState{Seed: seed, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}})
		if logCount(logs, "EVADE_MISS") > 0 {
			evaded++
		}
	}
	if evaded == 0 {
		t.Fatal("expected EVADE_MISS on at least one seed with 50% evasion")
	}
	if evaded == 40 {
		t.Fatal("50% evasion dodged every strike; the roll is not probabilistic")
	}
}

// Sát Khí (SAT_KHI_GUARANTEE, level > 10) is the hard counter to evasion and
// must keep working now that Accuracy no longer suppresses it.
func TestSatKhiBypassesEvasion(t *testing.T) {
	attacker := newTestCombatant("a", "Player", 1000, 0)
	attacker.Speed = 100
	attacker.CustomState = map[string]int{"sat_khi_level": 11}
	attacker.Mechanics = []CombatMechanic{mech("ON_BATTLE_START", "SAT_KHI_GUARANTEE", `{}`)}
	attacker.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đấm", Damage: 50}}

	defender := newTestCombatant("d", "Enemy", 1000, 0)
	defender.Speed = 1
	defender.MaxHP = 100000
	defender.HP = 100000
	applyEvasionOps(defender, []statusfx.StatusOp{{
		ID: statusfx.StatusEvasion, Value: 75, Turns: 5, Chance: 1.0,
	}}, 1)

	evaded := 0
	for seed := int64(1); seed <= 30; seed++ {
		logs := Simulate(&CombatState{Seed: seed, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}})
		evaded += logCount(logs, "EVADE_MISS")
	}
	if evaded != 0 {
		t.Fatalf("EVADE_MISS logged %d times against a Sát Khí guaranteed attacker", evaded)
	}
}

func mustEvasionSpec(t *testing.T) statusfx.StatusSpec {
	t.Helper()
	spec, ok := statusfx.Spec(statusfx.StatusEvasion)
	if !ok {
		t.Fatal("EVASION missing from the statusfx registry")
	}
	if !spec.SelfTarget {
		t.Fatal("EVASION spec must be SelfTarget so it routes to the caster")
	}
	if spec.Category != statusfx.CategoryBuff {
		t.Fatalf("EVASION category = %v, want BUFF", spec.Category)
	}
	return spec
}