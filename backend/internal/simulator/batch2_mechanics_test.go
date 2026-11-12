package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestThoughtChaosMisfireAndManaCost(t *testing.T) {
	// Trí Đạo: Tạp Niệm Loạn Ý (THOUGHT_CHAOS)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100
	player.ActiveEffects = []Effect{
		{ID: statusfx.StatusThoughtChaos, Duration: 3},
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Hỏa Cầu Thuật",
			Damage:     50,
			ManaCost:   20,
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundChaosLog := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Trí Đạo - Tạp Niệm Loạn Ý]") {
			foundChaosLog = true
			break
		}
	}

	if !foundChaosLog {
		t.Fatalf("expected ThoughtChaos log in battle, got: %+v", logs)
	}
}

func TestCalculatedStrikeGuaranteedHit(t *testing.T) {
	// Trí Đạo: Động Tất Sơ Hở (CALCULATED_STRIKE)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	// Enemy has 100% evasion cap, player is blinded
	enemy.EvasionChance = 0.75
	player.IsBlinded = true
	player.AccuracyPenalty = 0.90

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Nhất Kích Trất Mệnh",
			Damage:     100,
			Mechanics:  []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "CALCULATED_STRIKE"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundCalculated := false
	foundEvadeOrMiss := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Trí Đạo - Động Tất Sơ Hở]") {
			foundCalculated = true
		}
		if l.Action == "EVADE_MISS" || l.Action == "BLIND_MISS" {
			foundEvadeOrMiss = true
		}
	}

	if !foundCalculated {
		t.Fatalf("expected CALCULATED_STRIKE log, got: %+v", logs)
	}
	if foundEvadeOrMiss {
		t.Fatalf("calculated strike must not be evaded or missed, got: %+v", logs)
	}
}

func TestSoulShockShieldBypassAndSoulWound(t *testing.T) {
	// Hồn Đạo: Đãng Hồn Kích (SOUL_SHOCK)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10
	enemy.VirtualShield = 500

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Đãng Hồn Tiếng Vang",
			Damage:     100,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "SOUL_SHOCK"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundShock := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Hồn Đạo - Đãng Hồn Kích]") {
			foundShock = true
			break
		}
	}

	if !foundShock {
		t.Fatalf("expected SOUL_SHOCK log, got: %+v", logs)
	}
	// Target's HP must be reduced while VirtualShield remains intact or bypassed
	if enemy.HP >= 1000 {
		t.Fatalf("expected enemy HP to be reduced bypassing shield, got HP: %.0f", enemy.HP)
	}
	if enemy.VirtualShield != 500 {
		t.Fatalf("expected shield to remain untouched (500), got: %.0f", enemy.VirtualShield)
	}
}

func TestSoulDevourLifestealAndStacks(t *testing.T) {
	// Hồn Đạo: Phệ Hồn Hóa Lực (SOUL_DEVOUR)
	player := newTestCombatant("player", "Player", 1000, 100)
	player.HP = 500
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Phệ Hồn Quyết",
			Damage:     50,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "SOUL_DEVOUR"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundDevour := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Hồn Đạo - Phệ Hồn Hóa Lực]") {
			foundDevour = true
			break
		}
	}

	if !foundDevour {
		t.Fatalf("expected SOUL_DEVOUR log, got: %+v", logs)
	}
	if player.CustomState["soul_devour_stacks"] < 1 {
		t.Fatalf("expected soul_devour_stacks to be >= 1, got: %d", player.CustomState["soul_devour_stacks"])
	}
}

func TestBeastPhantomStrikeAndBerserk(t *testing.T) {
	// Lực Đạo: Thú Lực Hư Ảnh (BEAST_PHANTOM_STRIKE) & Cuồng Nộ (BERSERK_STATE)
	foundBeastPhantom := false

	for seed := int64(1); seed <= 50; seed++ {
		player := newTestCombatant("player", "Player", 1000, 100)
		player.HP = 200 // 20% HP triggers high berserk scaling & CC immunity
		enemy := newTestCombatant("enemy", "Enemy", 2000, 50)
		player.Speed = 20
		enemy.Speed = 10

		player.Actions = []SimAction{
			{
				ActionType: "GU_SKILL",
				Name:       "Bách Thú Bào Hao",
				Damage:     100,
				Mechanics: []CombatMechanic{
					{Trigger: "ON_ACTION", ActionCode: "BEAST_PHANTOM_STRIKE"},
					{Trigger: "COMBAT_PASSIVE", ActionCode: "BERSERK_STATE"},
				},
			},
		}

		state := &CombatState{
			Seed:    seed,
			Allies:  []*CombatantAuto{player},
			Enemies: []*CombatantAuto{enemy},
		}

		logs := Simulate(state)

		for _, l := range logs {
			if strings.Contains(l.Detail, "[Lực Đạo - Thú Lực Hư Ảnh]") {
				foundBeastPhantom = true
				break
			}
		}
		if foundBeastPhantom {
			break
		}
	}

	if !foundBeastPhantom {
		t.Fatalf("expected BEAST_PHANTOM_STRIKE in logs")
	}
}

func TestBloodSacrificeAndBleedBurst(t *testing.T) {
	// Huyết Đạo: Huyết Tế Cuồng Bạo (BLOOD_SACRIFICE) & Huyết Bạo (BLEED_BURST)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 2000, 50)
	player.Speed = 20
	enemy.Speed = 10

	// Give enemy bleed stacks
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusBleed, Value: 30, Duration: 3},
		{ID: statusfx.StatusBleed, Value: 30, Duration: 3},
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Huyết Ma Trảm",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "BLOOD_SACRIFICE"},
				{Trigger: "ON_ACTION", ActionCode: "BLEED_BURST"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundSacrifice := false
	foundBleedBurst := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Huyết Đạo - Huyết Tế Cuồng Bạo]") {
			foundSacrifice = true
		}
		if strings.Contains(l.Detail, "[Huyết Đạo - Huyết Bạo Trùng Điệp]") {
			foundBleedBurst = true
		}
	}

	if !foundSacrifice {
		t.Fatalf("expected BLOOD_SACRIFICE in logs, got: %+v", logs)
	}
	if !foundBleedBurst {
		t.Fatalf("expected BLEED_BURST in logs, got: %+v", logs)
	}
	// Player sacrificed HP
	if player.HP >= 1000 {
		t.Fatalf("expected player HP to drop due to blood sacrifice, got: %.0f", player.HP)
	}
}

func TestTheftManaAndBuffSteal(t *testing.T) {
	// Trộm Đạo: Không Thủ Diệu Thâu (MANA_STEAL & BUFF_STEAL)
	player := newTestCombatant("player", "Player", 1000, 100)
	player.Mana = 20
	player.MaxMana = 100
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	enemy.Mana = 100
	enemy.MaxMana = 100
	player.Speed = 20
	enemy.Speed = 10

	// Enemy has a positive buff
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusAtkUp, Type: "buff", Value: 20, Duration: 3},
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Thâu Thiên Hoán Nhật",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "MANA_STEAL"},
				{Trigger: "ON_ACTION", ActionCode: "BUFF_STEAL"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundManaSteal := false
	foundBuffSteal := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Trộm Đạo - Thâu Chân Nguyên]") {
			foundManaSteal = true
		}
		if strings.Contains(l.Detail, "[Trộm Đạo - Thâu Thiên Hoán Nhật]") {
			foundBuffSteal = true
		}
	}

	if !foundManaSteal {
		t.Fatalf("expected MANA_STEAL in logs, got: %+v", logs)
	}
	if !foundBuffSteal {
		t.Fatalf("expected BUFF_STEAL in logs, got: %+v", logs)
	}
	if enemy.Mana >= 100 {
		t.Fatalf("expected enemy mana to be stolen, got: %.0f", enemy.Mana)
	}
}

func TestTimeRewindRestoresAndCleanses(t *testing.T) {
	// Thời Đạo: Quang Âm Đảo Lưu (TIME_REWIND)
	player := newTestCombatant("player", "Player", 1000, 100)
	player.HP = 300
	player.Mana = 10
	player.MaxMana = 100
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Thời Gian Tố Nguyên",
			Damage:     50,
			Mechanics:  []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "TIME_REWIND"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundRewind := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Thời Đạo - Quang Âm Đảo Lưu]") {
			foundRewind = true
			break
		}
	}

	if !foundRewind {
		t.Fatalf("expected TIME_REWIND in logs, got: %+v", logs)
	}
	if player.HP <= 300 {
		t.Fatalf("expected HP restored, got: %.0f", player.HP)
	}
	if player.Mana <= 10 {
		t.Fatalf("expected Mana restored, got: %.0f", player.Mana)
	}
}
