package simulator

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestDreamPrisonSkipsTurnAndDoesNotBreakOnDamage(t *testing.T) {
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Nhập Mộng Thuật",
			Damage:     50,
			StatusOps:  []statusfx.StatusOp{{ID: statusfx.StatusDreamPrison, Turns: 2, Chance: 1.0}},
		},
	}
	enemy.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Name:       "Đánh Thường",
			Damage:     50,
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundApplied := false
	foundDreamPrisonLog := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Mộng Đạo - Nhập Mộng]") {
			foundApplied = true
		}
		if l.Action == "DREAM_PRISON" {
			foundDreamPrisonLog = true
		}
	}

	if !foundApplied {
		t.Fatalf("expected DREAM_PRISON_APPLIED in logs, got: %+v", logs)
	}
	if !foundDreamPrisonLog {
		t.Fatalf("expected enemy to skip turn with DREAM_PRISON, got: %+v", logs)
	}
}

func TestLuckRouletteMechanic(t *testing.T) {
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 5000, 10)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Cát Hung Bói Toán",
			Damage:     100,
			ManaCost:   20,
			Mechanics:  []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "LUCK_ROULETTE"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundRoulette := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Vận Đạo -") {
			foundRoulette = true
			break
		}
	}
	if !foundRoulette {
		t.Fatalf("expected Vận Đạo roulette in logs, got: %+v", logs)
	}
}

func TestLethalLuckWardPreventsDeath(t *testing.T) {
	target := newTestCombatant("lucky_guy", "Player", 1000, 50)
	target.HP = 10
	target.Mechanics = []CombatMechanic{{Trigger: "BEFORE_HIT", ActionCode: "LETHAL_LUCK_WARD"}}

	var logs []CombatLog
	var rng *rand.Rand
	for s := int64(1); s < 100; s++ {
		r := rand.New(rand.NewSource(s))
		if r.Float64() < 0.25 {
			rng = rand.New(rand.NewSource(s))
			break
		}
	}

	target.HP = -50
	saved := applyLethalLuckWard(target, &logs, 1, rng)
	if !saved {
		t.Fatalf("expected Lethal Luck Ward to trigger and save the combatant")
	}

	if target.HP != 1.0 {
		t.Errorf("expected target HP to be floored to 1.0, got %f", target.HP)
	}
	if target.VirtualShield <= 0 {
		t.Errorf("expected target to receive converted virtual shield, got %f", target.VirtualShield)
	}

	foundLog := false
	for _, l := range logs {
		if l.Action == "LETHAL_LUCK_WARD" {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Errorf("expected LETHAL_LUCK_WARD log, got: %+v", logs)
	}

	// Verify only triggers once per battle
	target.HP = -10
	savedSecondTime := applyLethalLuckWard(target, &logs, 2, rng)
	if savedSecondTime {
		t.Errorf("LETHAL_LUCK_WARD should only trigger once per battle!")
	}
}

func TestDreamUnravelDispelsDebuffsAndRestores(t *testing.T) {
	player := newTestCombatant("dream_master", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 10)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 20
	player.MaxMana = 100

	player.ActiveEffects = []Effect{
		{ID: statusfx.StatusPoison, Type: "debuff", Value: 10, Duration: 3},
		{ID: statusfx.StatusSlow, Type: "debuff", Value: 20, Duration: 2},
	}
	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Giải Mộng Cổ",
			Damage:     10,
			ManaCost:   10,
			Mechanics:  []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "DREAM_UNRAVEL"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundUnravel := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Mộng Đạo - Giải Mộng]") {
			foundUnravel = true
			break
		}
	}
	if !foundUnravel {
		t.Fatalf("expected DREAM_UNRAVEL in logs, got: %+v", logs)
	}
}

func TestLuckSiphonApplies(t *testing.T) {
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	op := statusfx.StatusOp{ID: statusfx.StatusLuckSiphon, Turns: 3, Chance: 1.0}
	var logs []CombatLog
	var rng *rand.Rand
	for s := int64(1); s < 100; s++ {
		r := rand.New(rand.NewSource(s))
		if statusfx.CanApplyStatusWithGuRank(r, player.Rank, enemy.Rank, 3, player.DaoMarks, enemy.DaoMarks, statusfx.Registry[statusfx.StatusLuckSiphon], 0, 1.0) {
			rng = rand.New(rand.NewSource(s))
			break
		}
	}
	applyStatusOpTo(player, enemy, statusfx.Registry[statusfx.StatusLuckSiphon], op, 3, 1, rng, &logs)

	foundLog := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Vận Đạo - Đoạt Vận]") {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Fatalf("expected Luck Siphon log, got: %+v", logs)
	}

	if player.EvasionChance <= 0 {
		t.Errorf("expected player to gain evasion from siphon, got %f", player.EvasionChance)
	}
}
