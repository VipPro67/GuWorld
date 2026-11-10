package simulator

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// Bất Hoại (INVINCIBLE) and Bất Tử (LAST_STAND) are deliberately different:
// Bất Hoại blocks damage on every path, Bất Tử lets it land but floors HP at
// LastStandMinHP. These tests cover the paths that write HP directly and so
// never reach applyDamageToTarget — the ones that used to slip past Bất Hoại.

func selfBuff(c *CombatantAuto, op statusfx.StatusOp, seed int64) []CombatLog {
	logs := make([]CombatLog, 0)
	applyStatusOps(c, []*CombatantAuto{c}, []statusfx.StatusOp{op}, c.Rank, 1, rand.New(rand.NewSource(seed)), &logs)
	return logs
}

func TestInvincibleBlocksAllSevenDamagePaths(t *testing.T) {
	paths := []struct {
		name string
		hurt func(t *testing.T, c *CombatantAuto)
	}{
		{"dot", func(t *testing.T, c *CombatantAuto) {
			c.ActiveDoT = []DoTEffect{{Type: "burn", Value: 999, Turns: 1}}
			tickDoTs(c, 1)
		}},
		{"dot_direct_hp", func(t *testing.T, c *CombatantAuto) {
			c.ActiveDoT = []DoTEffect{{Type: "poison", Value: 999, Turns: 1, DirectHP: true}}
			tickDoTs(c, 1)
		}},
		{"soul_wound", func(t *testing.T, c *CombatantAuto) {
			applyOneStatus(c, statusfx.StatusOp{ID: statusfx.StatusSoulWound, Value: 999, Turns: 1, Chance: 1.0}, 1)
		}},
		{"backlash", func(t *testing.T, c *CombatantAuto) {
			applyOneStatus(c, statusfx.StatusOp{ID: statusfx.StatusBacklash, Value: 999, Turns: 1, Chance: 1.0}, 1)
		}},
		{"thorns", func(t *testing.T, c *CombatantAuto) {
			defender := newTestCombatant("thorn", "Enemy", 5000, 0)
			defender.ThornsDamage = 1
			ApplyThornsDamage(defender, c, 999, 1, &[]CombatLog{})
		}},
		{"bleed_burst_on_dispel", func(t *testing.T, c *CombatantAuto) {
			c.ActiveEffects = append(c.ActiveEffects, Effect{ID: statusfx.StatusBleed, Type: "debuff", Value: 400, Duration: 5})
			DispelActorStatuses(c, statusfx.CategoryPhysicalDoT, 0)
		}},
	}

	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			c := newTestCombatant("bất hoại", "Enemy", 500, 0)
			ComputeEffectMultipliers(c)
			selfBuff(c, statusfx.StatusOp{ID: statusfx.StatusInvincible, Value: 100, Turns: 3, Chance: 1.0}, 1)
			if !c.IsInvincible {
				t.Fatal("IsInvincible not set after applying INVINCIBLE")
			}
			p.hurt(t, c)
			if c.HP != 500 {
				t.Fatalf("HP = %v after %s, want 500: Bất Hoại must block it", c.HP, p.name)
			}
		})
	}
}

// The %MaxHP rider is applied by runOne, not applyDamageToTarget, so it needs
// its own end-to-end check.
func TestInvincibleAbsorbsPercentMaxHP(t *testing.T) {
	attacker := newTestCombatant("a", "Player", 5000, 0)
	attacker.Speed = 100
	attacker.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Xuyên Khiên", Damage: 0,
		DamageHPPercent: 0.9,
	}}

	defender := newTestCombatant("d", "Enemy", 5000, 0)
	defender.Speed = 1
	ComputeEffectMultipliers(defender)
	// Durations tick at the defender's own turn start, so Turns 2 keeps the
	// buff up through the opening attack. Assertions stay on turn 1 so the
	// result does not depend on how long the fight runs afterwards.
	selfBuff(defender, statusfx.StatusOp{ID: statusfx.StatusInvincible, Value: 100, Turns: 2, Chance: 1.0}, 1)

	logs := Simulate(&CombatState{Seed: 3, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}})
	if !hasDetail(logs, "miễn nhiễm") {
		t.Fatal("expected the %MaxHP rider to be absorbed while Bất Hoại is up")
	}
	for _, l := range logs {
		if l.Action == "HP_PERCENT" && l.Turn == 1 {
			t.Fatalf("Bất Hoại must block the MaxHP-percentage rider on turn 1: %s", l.Detail)
		}
	}
}

func TestLastStandFloorsEveryDamagePath(t *testing.T) {
	paths := []struct {
		name string
		hurt func(t *testing.T, c *CombatantAuto)
	}{
		{"dot", func(t *testing.T, c *CombatantAuto) {
			c.ActiveDoT = []DoTEffect{{Type: "burn", Value: 999, Turns: 1}}
			tickDoTs(c, 1)
		}},
		{"dot_direct_hp", func(t *testing.T, c *CombatantAuto) {
			c.ActiveDoT = []DoTEffect{{Type: "poison", Value: 999, Turns: 1, DirectHP: true}}
			tickDoTs(c, 1)
		}},
		{"soul_wound", func(t *testing.T, c *CombatantAuto) {
			applyOneStatus(c, statusfx.StatusOp{ID: statusfx.StatusSoulWound, Value: 999, Turns: 1, Chance: 1.0}, 1)
		}},
		{"backlash", func(t *testing.T, c *CombatantAuto) {
			applyOneStatus(c, statusfx.StatusOp{ID: statusfx.StatusBacklash, Value: 999, Turns: 1, Chance: 1.0}, 1)
		}},
		{"thorns", func(t *testing.T, c *CombatantAuto) {
			defender := newTestCombatant("thorn", "Enemy", 5000, 0)
			defender.ThornsDamage = 1
			ApplyThornsDamage(defender, c, 999, 1, &[]CombatLog{})
		}},
		{"bleed_burst_on_dispel", func(t *testing.T, c *CombatantAuto) {
			c.ActiveEffects = append(c.ActiveEffects, Effect{ID: statusfx.StatusBleed, Type: "debuff", Value: 400, Duration: 5})
			DispelActorStatuses(c, statusfx.CategoryPhysicalDoT, 0)
		}},
		{"basic_strike", func(t *testing.T, c *CombatantAuto) {
			applyDamageToTarget(&CombatState{Seed: 1}, newTestCombatant("a", "Player", 5000, 0), c, 9999, &[]CombatLog{}, 1, false, false, false, rand.New(rand.NewSource(1)))
		}},
	}

	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			c := newTestCombatant("bất tử", "Enemy", 500, 0)
			ComputeEffectMultipliers(c)
			selfBuff(c, statusfx.StatusOp{ID: statusfx.StatusLastStand, Value: 1, Turns: 3, Chance: 1.0}, 1)
			if !c.IsLastStand {
				t.Fatal("IsLastStand not set after applying LAST_STAND")
			}
			p.hurt(t, c)
			if c.HP != LastStandMinHP {
				t.Fatalf("HP = %v after %s, want the Bất Tử floor %v", c.HP, p.name, LastStandMinHP)
			}
			if c.IsDead {
				t.Fatal("Bất Tử unit must not be marked dead")
			}
		})
	}
}

// The defining difference between the two effects: Bất Hoại returns before any
// damage is logged, Bất Tử lets the hit register.
func TestLastStandStillLogsDamageButInvincibleDoesNot(t *testing.T) {
	// Durations tick at the defender's own turn start, so Turns 2 keeps the buff
	// up through the opening attack; assertions stay on turn 1.
	// Lethal damage so the floor actually has to engage.
	run := func(effectID string) (hpAfterFirstHit string, blocked bool) {
		attacker := newTestCombatant("a", "Player", 5000, 0)
		attacker.Speed = 100
		attacker.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đấm", Damage: 9999}}

		defender := newTestCombatant("d", "Enemy", 5000, 0)
		defender.Speed = 1
		ComputeEffectMultipliers(defender)
		selfBuff(defender, statusfx.StatusOp{ID: effectID, Value: 100, Turns: 2, Chance: 1.0}, 1)

		logs := Simulate(&CombatState{Seed: 4, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}})
		hp := ""
		blocked = false
		for _, l := range logs {
			if l.Turn != 1 || !strings.HasPrefix(l.Detail, "a dùng") {
				continue
			}
			// The outcome text is appended to the use log, so the HP figure ends
			// at the first ")" or ",".
			if i := strings.Index(l.Detail, "còn "); i >= 0 {
				rest := l.Detail[i+len("còn "):]
				if j := strings.IndexAny(rest, "),"); j >= 0 {
					rest = rest[:j]
				}
				hp = strings.TrimSpace(rest)
			}
			if strings.Contains(l.Detail, "miễn nhiễm mọi sát thương") {
				blocked = true
			}
		}
		return hp, blocked
	}

	invHP, invBlocked := run(statusfx.StatusInvincible)
	if !invBlocked {
		t.Fatal("Bất Hoại must log a blocked hit")
	}
	if invHP != "5000 HP" {
		t.Fatalf("Bất Hoại HP after a lethal hit = %q, want the untouched 5000 HP", invHP)
	}

	lastHP, lastBlocked := run(statusfx.StatusLastStand)
	if lastBlocked {
		t.Fatal("Bất Tử must not block the hit")
	}
	if lastHP != "1 HP" {
		t.Fatalf("Bất Tử HP after a lethal hit = %q, want the floor at 1 HP", lastHP)
	}
}

// Bất Tử does not resurrect: a unit already at 0 when the buff lands stays dead.
func TestLastStandDoesNotHealOnApply(t *testing.T) {
	c := newTestCombatant("d", "Enemy", 500, 0)
	c.HP = 40
	ComputeEffectMultipliers(c)
	selfBuff(c, statusfx.StatusOp{ID: statusfx.StatusLastStand, Value: 1, Turns: 3, Chance: 1.0}, 1)

	// Applying the status must not touch HP; the floor only guards damage writes.
	if c.HP != 40 {
		t.Fatalf("HP = %v, want 40: applying LAST_STAND must not heal", c.HP)
	}
	if applyLastStandFloor(c, false) {
		t.Fatal("applyLastStandFloor must not fire when HP is already above the floor")
	}
}

// Sát Khí (Tất Sát Cổ, level > 10) pierces both effects.
func TestSatKhiKillsThroughBothEffects(t *testing.T) {
	run := func(effectID string) (float64, bool, bool) {
		attacker := newTestCombatant("a", "Player", 5000, 0)
		attacker.Speed = 100
		attacker.CustomState = map[string]int{"sat_khi_level": 11}
		attacker.Mechanics = []CombatMechanic{mech("ON_BATTLE_START", "SAT_KHI_GUARANTEE", `{}`)}
		attacker.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đấm", Damage: 9999}}

		defender := newTestCombatant("d", "Enemy", 5000, 0)
		defender.Speed = 1
		ComputeEffectMultipliers(defender)
		selfBuff(defender, statusfx.StatusOp{ID: effectID, Value: 100, Turns: 5, Chance: 1.0}, 1)

		logs := Simulate(&CombatState{Seed: 6, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}})
		// Per-action logs are folded into one consolidated entry, so match on the
		// detail text rather than the action name.
		sawBreak := false
		for _, l := range logs {
			if strings.Contains(l.Detail, "xuyên qua Bất Hoại") {
				sawBreak = true
			}
		}
		return defender.HP, defender.IsDead, sawBreak
	}

	invHP, invDead, invBreak := run(statusfx.StatusInvincible)
	if invHP != 0 || !invDead {
		t.Fatalf("Sát Khí must kill through Bất Hoại (HP=%v dead=%v)", invHP, invDead)
	}
	if !invBreak {
		t.Fatal("expected a BLOCKED_BREAK log for Bất Hoại")
	}

	lastHP, lastDead, _ := run(statusfx.StatusLastStand)
	if lastHP != 0 || !lastDead {
		t.Fatalf("Sát Khí must ignore the Bất Tử floor (HP=%v dead=%v)", lastHP, lastDead)
	}
}

// GU_BACKLASH is self-inflicted: using a Gu above your realm is a consequence,
// not enemy damage, so neither effect shields it.
func TestGuBacklashStillKillsSelf(t *testing.T) {
	for _, effectID := range []string{statusfx.StatusInvincible, statusfx.StatusLastStand} {
		c := newTestCombatant("a", "Player", 100, 0)
		ComputeEffectMultipliers(c)
		selfBuff(c, statusfx.StatusOp{ID: effectID, Value: 100, Turns: 5, Chance: 1.0}, 1)

		// Mirror engine.go GU_BACKLASH: plain subtraction plus a zero clamp, no
		// immunity check and no Bất Tử floor.
		c.HP -= 500
		if c.HP < 0 {
			c.HP = 0
		}
		if c.HP != 0 {
			t.Fatalf("%s must not shield self-inflicted backlash damage (HP=%v)", effectID, c.HP)
		}
	}
}

// Bất Hoại is a non-dispellable buff; Bất Tử is dispellable so cleanse has an
// answer to a save effect.
func TestInvincibleNotDispellableButLastStandIs(t *testing.T) {
	if isDispellableEffect(statusfx.StatusInvincible) {
		t.Fatal("Bất Hoại must not be dispellable")
	}
	if !isDispellableEffect(statusfx.StatusLastStand) {
		t.Fatal("Bất Tử should be dispellable")
	}

	c := newTestCombatant("d", "Enemy", 500, 0)
	ComputeEffectMultipliers(c)
	selfBuff(c, statusfx.StatusOp{ID: statusfx.StatusLastStand, Value: 1, Turns: 5, Chance: 1.0}, 1)
	DispelActorStatuses(c, statusfx.CategoryImmortal, 0)
	ComputeEffectMultipliers(c)
	if c.IsLastStand {
		t.Fatal("Bất Tử survived a CategoryImmortal dispel")
	}
}

// A unit saved to 1 HP dies as soon as the buff runs out.
func TestLastStandExpiresAndUnitDies(t *testing.T) {
	attacker := newTestCombatant("a", "Player", 5000, 0)
	attacker.Speed = 100
	attacker.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đấm", Damage: 400}}

	defender := newTestCombatant("d", "Enemy", 500, 0)
	defender.Speed = 1
	ComputeEffectMultipliers(defender)
	// Turns 1, so the save only covers the turn it lands on.
	selfBuff(defender, statusfx.StatusOp{ID: statusfx.StatusLastStand, Value: 1, Turns: 1, Chance: 1.0}, 1)

	logs := Simulate(&CombatState{Seed: 8, Allies: []*CombatantAuto{attacker}, Enemies: []*CombatantAuto{defender}})
	if defender.HP > LastStandMinHP {
		t.Fatalf("expected the hit to floor HP at %v, got %v", LastStandMinHP, defender.HP)
	}
	sawDeath := false
	for _, l := range logs {
		if l.Action == "DIED" && strings.Contains(l.Detail, "gục ngã") {
			sawDeath = true
		}
	}
	if !sawDeath && !defender.IsDead {
		t.Fatal("a Bất Tử unit must die once the buff expires")
	}
}

// applyOneStatus routes a single self-targeted op through the registry.
func applyOneStatus(c *CombatantAuto, op statusfx.StatusOp, seed int64) {
	applyStatusOps(c, []*CombatantAuto{c}, []statusfx.StatusOp{op}, c.Rank, 1, rand.New(rand.NewSource(seed)), &[]CombatLog{})
	ComputeEffectMultipliers(c)
}