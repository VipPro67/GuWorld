package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestPackFrenzyEnslavementDamageScaling(t *testing.T) {
	// Nô Đạo: Quân Đoàn Hợp Kích (PACK_FRENZY)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 2000, 50)
	player.Speed = 20
	enemy.Speed = 10

	minion1 := &CombatantAuto{
		ID:      "minion1",
		Name:    "Dị Thú Lang Nhất",
		Kind:    "Minion",
		MaxHP:   500,
		HP:      500,
		BaseDmg: 20,
		BaseDef: 20,
	}
	minion2 := &CombatantAuto{
		ID:      "minion2",
		Name:    "Dị Thú Lang Nhị",
		Kind:    "Minion",
		MaxHP:   500,
		HP:      500,
		BaseDmg: 20,
		BaseDef: 20,
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vạn Thú Hợp Kích",
			Damage:     100,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "PACK_FRENZY"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player, minion1, minion2},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundPackFrenzy := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Nô Đạo - Quân Đoàn Hợp Kích]") {
			foundPackFrenzy = true
			break
		}
	}

	if !foundPackFrenzy {
		t.Fatalf("expected PACK_FRENZY log in battle, got: %+v", logs)
	}
}

func TestFormBearAndEagleTransformation(t *testing.T) {
	// Biến Hóa Đạo: Hóa Thân Cự Hùng (FORM_BEAR) & Hóa Thân Kim Điêu (FORM_EAGLE)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 2000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Viễn Cổ Hùng Biến",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "FORM_BEAR"},
				{Trigger: "ON_CAST", ActionCode: "FORM_EAGLE"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundBear := false
	foundEagle := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Biến Hóa Đạo - Hóa Thân Cự Hùng]") {
			foundBear = true
		}
		if strings.Contains(l.Detail, "[Biến Hóa Đạo - Hóa Thân Kim Điêu]") {
			foundEagle = true
		}
	}

	if !foundBear {
		t.Fatalf("expected FORM_BEAR in logs, got: %+v", logs)
	}
	if !foundEagle {
		t.Fatalf("expected FORM_EAGLE in logs, got: %+v", logs)
	}
	// Player must have gained VirtualShield from Bear form
	if player.VirtualShield <= 0 {
		t.Fatalf("expected player to have VirtualShield from bear form, got: %.0f", player.VirtualShield)
	}
}

func TestDaoAdaptiveResist(t *testing.T) {
	// Biến Hóa Đạo: Vạn Vật Thích Ứng (DAO_ADAPTIVE_RESIST)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Thích Ứng Quyết",
			Damage:     50,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "DAO_ADAPTIVE_RESIST"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundAdaptive := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Biến Hóa Đạo - Vạn Vật Thích Ứng]") {
			foundAdaptive = true
			break
		}
	}

	if !foundAdaptive {
		t.Fatalf("expected DAO_ADAPTIVE_RESIST in logs, got: %+v", logs)
	}
}

func TestPoisonDetonate(t *testing.T) {
	// Độc Đạo: Vạn Độc Bộc Phát (POISON_DETONATE)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 2000, 50)
	player.Speed = 20
	enemy.Speed = 10

	// Target has poison stacks
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusPoison, Value: 40, Duration: 3},
		{ID: statusfx.StatusPoison, Value: 50, Duration: 3},
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vạn Độc Bộc Phát",
			Damage:     50,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "POISON_DETONATE"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundPoisonDetonate := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Độc Đạo - Vạn Độc Bộc Phát]") {
			foundPoisonDetonate = true
			break
		}
	}

	if !foundPoisonDetonate {
		t.Fatalf("expected POISON_DETONATE in logs, got: %+v", logs)
	}
	// Target should have lost poison and taken damage
	if enemy.HP >= 1900 {
		t.Fatalf("expected substantial poison detonation damage, got HP: %.0f", enemy.HP)
	}
}

func TestSwordArmorPierceAndExecuteStrike(t *testing.T) {
	// Kiếm Đạo: Vô Kiên Bất Tồi (SWORD_ARMOR_PIERCE)
	// Đao Đạo: Trảm Diệt Đao Ý (EXECUTE_STRIKE)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	enemy.HP = 300 // <= 35% HP triggers EXECUTE_STRIKE
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Đao Kiếm Song Tuyệt",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "SWORD_ARMOR_PIERCE"},
				{Trigger: "ON_ACTION", ActionCode: "EXECUTE_STRIKE"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundSwordPierce := false
	foundExecute := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Kiếm Đạo - Vô Kiên Bất Tồi]") {
			foundSwordPierce = true
		}
		if strings.Contains(l.Detail, "[Đao Đạo - Trảm Diệt Đao Ý]") {
			foundExecute = true
		}
	}

	if !foundSwordPierce {
		t.Fatalf("expected SWORD_ARMOR_PIERCE in logs, got: %+v", logs)
	}
	if !foundExecute {
		t.Fatalf("expected EXECUTE_STRIKE in logs, got: %+v", logs)
	}
}

func TestBoneSpikeAndThorns(t *testing.T) {
	// Cốt Đạo: Cốt Thứ Xuyên Tâm (BONE_SPIKE) & Bạch Cốt Gai Khôi (BONE_THORNS)
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Bạch Cốt Sát Chiêu",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "BONE_THORNS"},
				{Trigger: "ON_ACTION", ActionCode: "BONE_SPIKE"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundBoneThorns := false
	foundBoneSpike := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Cốt Đạo - Bạch Cốt Gai Khôi]") {
			foundBoneThorns = true
		}
		if strings.Contains(l.Detail, "[Cốt Đạo - Cốt Thứ Xuyên Tâm]") {
			foundBoneSpike = true
		}
	}

	if !foundBoneThorns {
		t.Fatalf("expected BONE_THORNS in logs, got: %+v", logs)
	}
	if !foundBoneSpike {
		t.Fatalf("expected BONE_SPIKE in logs, got: %+v", logs)
	}
}

func TestManaEat(t *testing.T) {
	// Thực Đạo: Thôn Phệ Bổ Khiếu (MANA_EAT)
	player := newTestCombatant("player", "Player", 1000, 100)
	player.Mana = 20
	player.MaxMana = 100
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Đại Đỗ Ba Ba",
			Damage:     50,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "MANA_EAT"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundManaEat := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Thực Đạo - Thôn Phệ Bổ Khiếu]") {
			foundManaEat = true
			break
		}
	}

	if !foundManaEat {
		t.Fatalf("expected MANA_EAT in logs, got: %+v", logs)
	}
	if player.Mana <= 20 {
		t.Fatalf("expected player mana to be restored by MANA_EAT, got: %.0f", player.Mana)
	}
}
