package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
)

// SUMMON_MONSTER spawns a minion from the injected template stats, scaled by the
// rank of the summoning Gu, and the minion is clamped to the caster's scale.
func TestSummonMonsterSpawnsRankScaledMinion(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100000, 1000)
	ally.Rank = 3
	ally.Speed = 50
	ally.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Ngự Xích Cổ", Rank: 3, Damage: 100,
		Mechanics: []CombatMechanic{mech("ON_CAST", "SUMMON_MONSTER",
			`{"monster_id":"monster_wolf_leader","hp":220,"damage":25,"defense":10,"hp_scale":0.5,"damage_scale":0.33,"position":"frontline"}`)},
	}}
	enemy := newTestCombatant("enemy", "Enemy", 100000, 100)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}

	state := &CombatState{Seed: 5, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	// RankScale(3) = 5 → hp = 220*0.5*5 = 550, damage = 25*0.33*5 = 41.25.
	minions := 0
	for _, c := range state.Allies {
		if c.Kind == "Minion" {
			minions++
			if c.MaxHP != 550 {
				t.Fatalf("minion HP = %.0f, want 550", c.MaxHP)
			}
			if c.HP != c.MaxHP {
				t.Fatalf("minion spawned at %.0f HP, want full", c.HP)
			}
			if c.BaseDmg != 41.25 {
				t.Fatalf("minion damage = %.2f, want 41.25", c.BaseDmg)
			}
			if c.GridX != 1 || c.GridY != 2 {
				t.Fatalf("minion grid = (%d,%d), want frontline (1,2)", c.GridX, c.GridY)
			}
			if c.Speed != ally.Speed {
				t.Fatalf("minion speed = %.0f, want caster speed %.0f", c.Speed, ally.Speed)
			}
		}
	}
	if minions != 1 {
		t.Fatalf("summoned minions = %d, want 1", minions)
	}
	if !hasDetail(logs, "triệu hồi") {
		t.Fatal("missing SUMMONED log for the summoned unit")
	}
}

// A high-rank summon cannot out-scale its caster: HP is capped at 2x and damage
// at 3x the caster's own stats.
func TestSummonMonsterClampedToCaster(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 1000, 100)
	ally.Rank = 9
	ally.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Ngự Ấn Tiên Cổ", Rank: 9, Damage: 100,
		Mechanics: []CombatMechanic{mech("ON_CAST", "SUMMON_MONSTER",
			`{"monster_id":"boss_wolf_king","hp":650,"damage":45,"defense":20}`)},
	}}
	enemy := newTestCombatant("enemy", "Enemy", 100000, 100)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}

	state := &CombatState{Seed: 3, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	Simulate(state)

	for _, c := range state.Allies {
		if c.Kind != "Minion" {
			continue
		}
		if c.MaxHP != 2000 {
			t.Fatalf("minion HP = %.0f, want clamped 2000", c.MaxHP)
		}
		if c.BaseDmg != 300 {
			t.Fatalf("minion damage = %.0f, want clamped 300", c.BaseDmg)
		}
		return
	}
	t.Fatal("no minion summoned")
}

// Enemies summon onto their own side (allied minions are not created).
func TestSummonMonsterEnemySide(t *testing.T) {
	enemy := newTestCombatant("enemy", "Enemy", 10000, 100)
	enemy.Kind = "Boss"
	enemy.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Triệu hồi", Rank: 1, Damage: 10,
		Mechanics: []CombatMechanic{mech("ON_CAST", "SUMMON_MONSTER",
			`{"monster_id":"monster_iron_bear","hp":260,"damage":48,"defense":10}`)},
	}}
	ally := newTestCombatant("ally", "Player", 100000, 1000)
	ally.Speed = 100
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Chém", Damage: 1}}

	state := &CombatState{Seed: 9, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	Simulate(state)

	for _, c := range state.Allies {
		if c.Kind == "Minion" {
			t.Fatal("enemy summon must not create an allied minion")
		}
	}
	found := false
	for _, c := range state.Enemies {
		if c.Kind == "Minion" {
			found = true
			if !strings.Contains(c.Name, "Triệu hồi") {
				t.Fatalf("minion name %q missing the summon suffix", c.Name)
			}
		}
	}
	if !found {
		t.Fatal("expected an enemy-side minion")
	}
}

// Thú triệu hồi scale theo đạo ngân của Cổ triệu hồi, không chỉ theo rank: minion
// nhận đúng daoMult mà damage/shield/heal của action vẫn dùng.
func TestSummonScalesWithEnslavementDaoMarks(t *testing.T) {
	profile := gamefx.BuildDaoCombatProfile(map[string]int64{"enslavement": 10000}, nil, false, nil)
	amp, realmEff, conflict := profile.GetOffensiveFactors("enslavement")
	want := (1.0 + (amp-1.0)*realmEff) * conflict
	if want <= 1.5 {
		t.Fatalf("test setup: dao multiplier %.3f too small to prove scaling", want)
	}

	ally := newTestCombatant("ally", "Player", 1000000, 10000)
	ally.Rank = 3
	ally.Speed = 50
	ally.DaoMarks = map[string]int64{"enslavement": 10000}
	ally.Actions = []SimAction{{
		ActionType: "GU_SKILL", Name: "Ngự Xích Cổ", Rank: 3, Damage: 100, Path: []string{"Enslavement Path"},
		Mechanics: []CombatMechanic{mech("ON_CAST", "SUMMON_MONSTER",
			`{"monster_id":"monster_wolf_leader","hp":220,"damage":25,"defense":10,"hp_scale":0.5,"damage_scale":0.33,"position":"frontline"}`)},
	}}
	enemy := newTestCombatant("enemy", "Enemy", 1000000, 1)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}

	state := &CombatState{Seed: 5, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	// RankScale(3) = 5 → nền hp 550 / damage 41.25, nhân tiếp daoMult.
	minions := 0
	for _, c := range state.Allies {
		if c.Kind != "Minion" {
			continue
		}
		minions++
		if c.MaxHP != 550*want {
			t.Fatalf("minion HP = %.2f, want %.2f (550 × %.3f dao)", c.MaxHP, 550*want, want)
		}
		if c.BaseDmg != 41.25*want {
			t.Fatalf("minion damage = %.2f, want %.2f (41.25 × %.3f dao)", c.BaseDmg, 41.25*want, want)
		}
	}
	if minions != 1 {
		t.Fatalf("summoned minions = %d, want 1", minions)
	}
	if !hasDetail(logs, "Đạo Ngân") {
		t.Fatal("SUMMONED log should surface the dao mark multiplier")
	}
}

// ON_BATTLE_START/SUMMON_MONSTER reinforces the field once, at turn 0, before
// any combatant acts — instead of recasting the summon on every turn.
func TestBattleStartSummonMonsterSpawnsOnce(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100000, 1000)
	ally.Rank = 3
	ally.Speed = 50
	m := mech("ON_BATTLE_START", "SUMMON_MONSTER",
		`{"monster_id":"monster_wolf_leader","hp":220,"damage":25,"defense":10,"hp_scale":0.5,"damage_scale":0.33,"position":"frontline"}`)
	m.Rank = 3
	ally.Mechanics = []CombatMechanic{m}
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 10}}

	enemy := newTestCombatant("enemy", "Enemy", 100000, 100)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}

	state := &CombatState{Seed: 5, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	minions := 0
	for _, c := range state.Allies {
		if c.Kind != "Minion" {
			continue
		}
		minions++
		if c.MaxHP != 550 {
			t.Fatalf("minion HP = %.0f, want 550 (220 × 0.5 × RankScale(3))", c.MaxHP)
		}
	}
	if minions != 1 {
		t.Fatalf("battle-start minions = %d, want 1", minions)
	}
	summonedAt := -1
	for _, l := range logs {
		if l.Action == "SUMMONED" {
			summonedAt = l.Turn
		}
	}
	if summonedAt != 0 {
		t.Fatalf("SUMMONED log turn = %d, want 0 (battle start)", summonedAt)
	}
}

// ON_BATTLE_START/SUMMON_WOLF_FRONTLINE summons Địa Lang Vương immediately and
// pulls the caster back to the mid line.
func TestBattleStartSummonWolfFrontline(t *testing.T) {
	ally := newTestCombatant("ally", "Player", 100000, 1000)
	ally.GridX = 1
	ally.Mechanics = []CombatMechanic{mech("ON_BATTLE_START", "SUMMON_WOLF_FRONTLINE", `{}`)}
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 10}}

	enemy := newTestCombatant("enemy", "Enemy", 100000, 100)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}

	state := &CombatState{Seed: 5, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	found := false
	for _, c := range state.Allies {
		if c.ID == "ally_dialang" && c.Kind == "Minion" {
			found = true
			if c.MaxHP != 50000 {
				t.Fatalf("wolf HP = %.0f, want 50000 (50%% caster)", c.MaxHP)
			}
		}
	}
	if !found {
		t.Fatal("expected Địa Lang Vương minion at battle start")
	}
	if ally.GridX != 2 {
		t.Fatalf("caster GridX = %d, want 2 (pulled to mid)", ally.GridX)
	}
	if !hasDetail(logs, "Địa Lang Vương") {
		t.Fatal("missing SUMMONED log for Địa Lang Vương")
	}
}

// Battle-start summons keep the đạo ngân scaling of the source Gu, using the
// Path stamped on the mechanic instead of a SimAction.
func TestBattleStartSummonScalesWithDaoMarks(t *testing.T) {
	profile := gamefx.BuildDaoCombatProfile(map[string]int64{"enslavement": 10000}, nil, false, nil)
	amp, realmEff, conflict := profile.GetOffensiveFactors("enslavement")
	want := (1.0 + (amp-1.0)*realmEff) * conflict
	if want <= 1.5 {
		t.Fatalf("test setup: dao multiplier %.3f too small to prove scaling", want)
	}

	ally := newTestCombatant("ally", "Player", 1000000, 10000)
	ally.Rank = 3
	ally.Speed = 50
	ally.DaoMarks = map[string]int64{"enslavement": 10000}
	m := mech("ON_BATTLE_START", "SUMMON_MONSTER",
		`{"monster_id":"monster_wolf_leader","hp":220,"damage":25,"defense":10,"hp_scale":0.5,"damage_scale":0.33,"position":"frontline"}`)
	m.Rank = 3
	m.Path = []string{"Enslavement Path"}
	ally.Mechanics = []CombatMechanic{m}
	ally.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Đánh", Damage: 10}}

	enemy := newTestCombatant("enemy", "Enemy", 1000000, 1)
	enemy.Speed = 1
	enemy.Actions = []SimAction{{ActionType: "BASIC_ATTACK", Name: "Cắn", Damage: 1}}

	state := &CombatState{Seed: 5, Allies: []*CombatantAuto{ally}, Enemies: []*CombatantAuto{enemy}}
	logs := Simulate(state)

	for _, c := range state.Allies {
		if c.Kind != "Minion" {
			continue
		}
		if c.MaxHP != 550*want {
			t.Fatalf("minion HP = %.2f, want %.2f (550 × %.3f dao)", c.MaxHP, 550*want, want)
		}
	}
	if !hasDetail(logs, "Đạo Ngân") {
		t.Fatal("battle-start SUMMONED log should surface the dao mark multiplier")
	}
}
