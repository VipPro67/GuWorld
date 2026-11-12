package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestReverseFlowSealReflection(t *testing.T) {
	// Nghịch Lưu Hộ Thân Quỹ: 100% reflect
	attacker := newTestCombatant("attacker", "Attacker", 1000, 100)
	defender := newTestCombatant("defender", "Defender", 1000, 100)
	attacker.Speed = 20
	defender.Speed = 10
	defender.Mechanics = []CombatMechanic{
		{Trigger: "BEFORE_HIT", ActionCode: "REVERSE_FLOW_SEAL"},
	}

	attacker.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Damage:     200,
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{defender},
	}

	logs := Simulate(state)

	foundReflect := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Sát Chiêu - Nghịch Lưu Hộ Thân Quỹ]") {
			foundReflect = true
			break
		}
	}

	if !foundReflect {
		t.Fatalf("expected REVERSE_FLOW_SEAL reflection log, got: %+v", logs)
	}
	if defender.HP < 1000 {
		t.Fatalf("expected defender to take 0 damage, got HP: %f", defender.HP)
	}
	if attacker.HP >= 1000 {
		t.Fatalf("expected attacker to take reflected damage, got HP: %f", attacker.HP)
	}
}

func TestGhostHandStealManaAndShield(t *testing.T) {
	// Đâu Thiên Thâu Thủ: Steal 30% mana and 50% shield
	thief := newTestCombatant("thief", "Thief", 1000, 100)
	victim := newTestCombatant("victim", "Victim", 1000, 100)
	thief.Speed = 20
	victim.Speed = 10
	thief.Mana = 50
	thief.MaxMana = 200
	victim.Mana = 100
	victim.VirtualShield = 200

	thief.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Đâu Thiên Thâu Thủ",
			Damage:     100,
			ManaCost:   10,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "GHOST_HAND_STEAL"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{thief},
		Enemies: []*CombatantAuto{victim},
	}

	logs := Simulate(state)

	foundSteal := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Trộm Đạo - Đâu Thiên Thâu Thủ]") {
			foundSteal = true
			break
		}
	}

	if !foundSteal {
		t.Fatalf("expected GHOST_HAND_STEAL log, got: %+v", logs)
	}
	if victim.Mana > 70 {
		t.Fatalf("expected victim mana reduced by at least 30, got: %f", victim.Mana)
	}
}

func TestLeadSoulToDream(t *testing.T) {
	// Dẫn Hồn Nhập Mộng: StatusDreamPrison and soul HP damage
	caster := newTestCombatant("caster", "Caster", 1000, 100)
	target := newTestCombatant("target", "Target", 1000, 100)
	caster.Speed = 20
	target.Speed = 10
	target.VirtualShield = 500

	caster.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Dẫn Hồn Nhập Mộng",
			Damage:     50,
			ManaCost:   10,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "LEAD_SOUL_TO_DREAM"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{caster},
		Enemies: []*CombatantAuto{target},
	}

	logs := Simulate(state)

	foundDream := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Mộng Đạo - Dẫn Hồn Nhập Mộng]") {
			foundDream = true
			break
		}
	}

	if !foundDream {
		t.Fatalf("expected LEAD_SOUL_TO_DREAM log, got: %+v", logs)
	}
}

func TestMyriadSelfGiantHandShieldBreak(t *testing.T) {
	// Vạn Ngã Đại Thủ Ấn: Completely shatter shield
	player := newTestCombatant("fy", "Fang Yuan", 1000, 100)
	boss := newTestCombatant("boss", "Boss", 1000, 100)
	player.Speed = 20
	boss.Speed = 10
	boss.VirtualShield = 1000

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vạn Ngã Đại Thủ Ấn",
			Damage:     200,
			ManaCost:   20,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "MYRIAD_SELF_GIANT_HAND"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{boss},
	}

	logs := Simulate(state)

	foundShatter := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Lực Đạo - Vạn Ngã Đại Thủ Ấn]") {
			foundShatter = true
			break
		}
	}

	if !foundShatter {
		t.Fatalf("expected MYRIAD_SELF_GIANT_HAND log, got: %+v", logs)
	}
}

func TestFamiliarFaceEvasionAndCrit(t *testing.T) {
	// Gặp Mặt Từng Quen Biết: Battle start buff
	player := newTestCombatant("fy", "Fang Yuan", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 100)
	player.Mechanics = []CombatMechanic{
		{Trigger: "ON_BATTLE_START", ActionCode: "FAMILIAR_FACE"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundFace := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Sát Chiêu - Gặp Mặt Từng Quen Biết]") {
			foundFace = true
			break
		}
	}

	if !foundFace {
		t.Fatalf("expected FAMILIAR_FACE battle start log, got: %+v", logs)
	}
}

func TestFiveFingerSword(t *testing.T) {
	// Ngũ Chỉ Quyền Tâm Kiếm: Ignore 50% defense and 65% dmg boost
	swordsman := newTestCombatant("bo_qing", "Bo Qing", 1000, 100)
	target := newTestCombatant("target", "Target", 1000, 100)
	target.BaseDef = 100
	swordsman.Speed = 20
	target.Speed = 10

	swordsman.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Ngũ Chỉ Quyền Tâm Kiếm",
			Damage:     200,
			ManaCost:   25,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "FIVE_FINGER_SWORD"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{swordsman},
		Enemies: []*CombatantAuto{target},
	}

	logs := Simulate(state)

	foundSword := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Kiếm Đạo - Ngũ Chỉ Quyền Tâm Kiếm]") {
			foundSword = true
			break
		}
	}

	if !foundSword {
		t.Fatalf("expected FIVE_FINGER_SWORD log, got: %+v", logs)
	}
}

func TestTimeCutterAging(t *testing.T) {
	// Tuế Nguyệt Như Đao: Mana drain, slow1, damage
	player := newTestCombatant("fy", "Fang Yuan", 1000, 100)
	enemy := newTestCombatant("immortal", "Immortal", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	enemy.Mana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Tuế Nguyệt Như Đao",
			Damage:     150,
			ManaCost:   30,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "TIME_CUTTER"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundCutter := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Thời Đạo - Tuế Nguyệt Như Đao]") {
			foundCutter = true
			break
		}
	}

	if !foundCutter {
		t.Fatalf("expected TIME_CUTTER log, got: %+v", logs)
	}
	if enemy.Mana > 75 {
		t.Fatalf("expected enemy mana <= 75 after time cutter, got: %f", enemy.Mana)
	}
}

func TestSpringScissorsSummerFan(t *testing.T) {
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Xuân Tiễn Hạ Phiến",
			Damage:     120,
			ManaCost:   20,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "SPRING_SCISSORS_SUMMER_FAN"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Thời Đạo - Xuân Tiễn Hạ Phiến]") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected SPRING_SCISSORS_SUMMER_FAN log, got: %+v", logs)
	}
}

func TestDragonBreathSwordFreeze(t *testing.T) {
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Long Tức Kiếm",
			Damage:     100,
			ManaCost:   20,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "DRAGON_BREATH_SWORD"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Băng Kiếm - Long Tức Kiếm]") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected DRAGON_BREATH_SWORD log, got: %+v", logs)
	}
}

func TestMyriadYearWarship(t *testing.T) {
	// Vạn Niên Đấu Phi Thuyền: Battle start haste surge + turn-start sword strike
	warshipUser := newTestCombatant("fy", "Fang Yuan", 1000, 100)
	ally := newTestCombatant("ally", "Ally", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 100)
	warshipUser.Mechanics = []CombatMechanic{
		{Trigger: "ON_BATTLE_START", ActionCode: "MYRIAD_YEAR_WARSHIP"},
		{Trigger: "ON_TURN_START", ActionCode: "MYRIAD_YEAR_WARSHIP"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{warshipUser, ally},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundStart := false
	foundTurn := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Tiên Cổ Ốc - Vạn Niên Đấu Phi Thuyền]") {
			if l.Turn == 0 {
				foundStart = true
			} else {
				foundTurn = true
			}
		}
	}

	if !foundStart {
		t.Fatalf("expected MYRIAD_YEAR_WARSHIP battle start log, got: %+v", logs)
	}
	if !foundTurn {
		t.Fatalf("expected MYRIAD_YEAR_WARSHIP turn-start sword log, got: %+v", logs)
	}
}

func TestDragonPalaceDominion(t *testing.T) {
	// Long Cung Ngự Hồn: 35% confuse before hit
	palaceUser := newTestCombatant("palace", "Dragon Palace", 1000, 100)
	attacker := newTestCombatant("attacker", "Attacker", 1000, 100)
	attacker.Speed = 20
	palaceUser.Speed = 10
	palaceUser.Mechanics = []CombatMechanic{
		{Trigger: "BEFORE_HIT", ActionCode: "DRAGON_PALACE_DOMINION"},
	}
	attacker.Actions = []SimAction{
		{ActionType: "BASIC_ATTACK", Damage: 100},
	}

	state := &CombatState{
		Seed:    7, // Choose seed where rng < 0.35
		Allies:  []*CombatantAuto{palaceUser},
		Enemies: []*CombatantAuto{attacker},
	}

	logs := Simulate(state)

	foundDominion := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Long Cung Ngự Hồn]") {
			foundDominion = true
			break
		}
	}

	if !foundDominion {
		t.Fatalf("expected DRAGON_PALACE_DOMINION confuse log, got: %+v", logs)
	}
}

func TestDivineBeanAbundance(t *testing.T) {
	// Thần Đậu Cung: Turn start cleanse + heal
	user := newTestCombatant("user", "User", 1000, 100)
	ally := newTestCombatant("ally", "Ally", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 100)
	ally.HP = 500
	ally.ActiveEffects = []Effect{{ID: statusfx.StatusBleed, Type: "debuff", Duration: 3}}
	user.Mechanics = []CombatMechanic{
		{Trigger: "ON_TURN_START", ActionCode: "DIVINE_BEAN_ABUNDANCE"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{user, ally},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundHeal := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Tiên Cổ Ốc - Thần Đậu Cung]") {
			foundHeal = true
			break
		}
	}

	if !foundHeal {
		t.Fatalf("expected DIVINE_BEAN_ABUNDANCE log, got: %+v", logs)
	}
	if ally.HP <= 500 {
		t.Fatalf("expected ally to be healed, got: %f", ally.HP)
	}
}

func TestEightExtremitiesLuck(t *testing.T) {
	// Bát Cực Quán: Enemy crit/evasion reduction
	user := newTestCombatant("user", "User", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 100)
	enemy.EvasionChance = 0.50
	user.Mechanics = []CombatMechanic{
		{Trigger: "ON_BATTLE_START", ActionCode: "EIGHT_EXTREMITIES_LUCK"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{user},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Tiên Cổ Ốc - Bát Cực Quán]") {
			found = true
			break
		}
	}

	if !found {
		t.Fatalf("expected EIGHT_EXTREMITIES_LUCK log, got: %+v", logs)
	}
	if enemy.EvasionChance > 0.25 {
		t.Fatalf("expected enemy evasion reduced from 0.50 to 0.25, got: %f", enemy.EvasionChance)
	}
}

func TestThunderGodCity(t *testing.T) {
	// Lôi Thần Thành: Strike lowest HP enemy
	user := newTestCombatant("user", "User", 1000, 100)
	enemy1 := newTestCombatant("e1", "Enemy 1", 1000, 100)
	enemy2 := newTestCombatant("e2", "Enemy 2", 300, 100) // Lower HP
	user.Speed = 20

	user.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Lôi Thần Phạt Tội",
			Damage:     50,
			ManaCost:   10,
			Mechanics:  []CombatMechanic{{Trigger: "ON_ACTION", ActionCode: "THUNDER_GOD_CITY"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{user},
		Enemies: []*CombatantAuto{enemy1, enemy2},
	}

	logs := Simulate(state)

	foundThunder := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Tiên Cổ Ốc - Lôi Thần Thành]") && strings.Contains(l.Detail, "lên e2!") {
			foundThunder = true
			break
		}
	}

	if !foundThunder {
		t.Fatalf("expected THUNDER_GOD_CITY log targeting e2, got: %+v", logs)
	}
}

func TestFiveRegionalBoundarySuppression(t *testing.T) {
	// Ngũ Giới Phân Tranh Trận: Battle start suppression + self-harm on action
	user := newTestCombatant("user", "User", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 100)
	user.Mechanics = []CombatMechanic{
		{Trigger: "ON_BATTLE_START", ActionCode: "FIVE_REGIONAL_BOUNDARY"},
	}
	enemy.Actions = []SimAction{
		{ActionType: "GU_SKILL", Name: "Đột Kích", Damage: 50, ManaCost: 10},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{user},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundStart := false
	foundSuppression := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Chiến Trường Sát Chiêu - Ngũ Giới Phân Tranh]") {
			foundStart = true
		}
		if strings.Contains(l.Detail, "[Ngũ Giới Áp Chế]") {
			foundSuppression = true
		}
	}

	if !foundStart {
		t.Fatalf("expected FIVE_REGIONAL_BOUNDARY start log, got: %+v", logs)
	}
	if !foundSuppression {
		t.Fatalf("expected FIVE_REGIONAL_SUPPRESSION backlash log, got: %+v", logs)
	}
}

func TestNinetyNineArray(t *testing.T) {
	// Cửu Cửu Liên Hoàn Trận: 50% damage reduction
	defender := newTestCombatant("defender", "Defender", 1000, 100)
	attacker := newTestCombatant("attacker", "Attacker", 1000, 100)
	attacker.Speed = 20
	defender.Speed = 10
	defender.Mechanics = []CombatMechanic{
		{Trigger: "BEFORE_HIT", ActionCode: "NINETY_NINE_ARRAY"},
	}
	attacker.Actions = []SimAction{
		{ActionType: "BASIC_ATTACK", Damage: 200},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{defender},
		Enemies: []*CombatantAuto{attacker},
	}

	logs := Simulate(state)

	foundArray := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Cửu Cửu Liên Hoàn]") {
			foundArray = true
			break
		}
	}

	if !foundArray {
		t.Fatalf("expected NINETY_NINE_ARRAY log, got: %+v", logs)
	}
}

func TestSoulConcealmentArray(t *testing.T) {
	// Yểm Hồn Ma Trận: Stealth and evasion for team
	user := newTestCombatant("user", "User", 1000, 100)
	ally := newTestCombatant("ally", "Ally", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 100)
	user.Mechanics = []CombatMechanic{
		{Trigger: "ON_BATTLE_START", ActionCode: "SOUL_CONCEALMENT_ARRAY"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{user, ally},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundConceal := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Đại Trận - Yểm Hồn Ma Trận]") {
			foundConceal = true
			break
		}
	}

	if !foundConceal {
		t.Fatalf("expected SOUL_CONCEALMENT_ARRAY log, got: %+v", logs)
	}
}

func TestSixExtremitiesArrayRamping(t *testing.T) {
	// Lục Cực Tuyệt Sát Trận: Ramping AoE damage each turn
	user := newTestCombatant("user", "User", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 100)
	user.Mechanics = []CombatMechanic{
		{Trigger: "ON_TURN_START", ActionCode: "SIX_EXTREMITIES_ARRAY"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{user},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundSix := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Sát Trận - Lục Cực Tuyệt Sát]") {
			foundSix = true
			break
		}
	}

	if !foundSix {
		t.Fatalf("expected SIX_EXTREMITIES_ARRAY log, got: %+v", logs)
	}
}
