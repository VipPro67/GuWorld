package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// 1. Kiếm Đạo: Kiếm Lãng Tam Điệp (SWORD_WAVE_TRIPLE)
func TestSwordWaveTriple(t *testing.T) {
	player := newTestCombatant("p1", "Bạc Thanh", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "SWORD_WAVE_TRIPLE"},
			},
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
		if strings.Contains(l.Detail, "[Kiếm Đạo - Kiếm Lãng Tam Điệp]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Kiếm Lãng Tam Điệp log, got: %+v", logs)
	}
}

// 2. Kiếm Đạo: Kiếm Ngân Tác Mệnh (SWORD_MARK_DEATH)
func TestSwordMarkDeath(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	enemy := newTestCombatant("e1", "Mục Tiêu", 500, 100) // survive until the first simulated-time mark tick
	enemy.MaxHP = 1000
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "SWORD_MARK_DEATH"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundMark := false
	foundExec := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Kiếm Đạo - Kiếm Ngân Tác Mệnh]") && strings.Contains(l.Detail, "Khắc sâu 2 đạo kiếm ngân") {
			foundMark = true
		}
		if strings.Contains(l.Detail, "trảm sát tức khắc") {
			foundExec = true
		}
	}
	if !foundMark {
		t.Fatalf("expected Sword Mark inscription log, got: %+v", logs)
	}
	if !foundExec {
		t.Fatalf("expected Sword Mark execute under 25%% HP log, got: %+v", logs)
	}
}

// 3. Hồn Đạo: Hồn Bạo Đãng Hồn Sơn (SOUL_EXPLOSION_DANG_HUN)
func TestSoulExplosionDangHun(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	e1 := newTestCombatant("e1", "Địch 1", 1000, 100)
	e2 := newTestCombatant("e2", "Địch 2", 1000, 100)
	player.Speed = 20
	e1.Speed = 10
	e2.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "SOUL_EXPLOSION_DANG_HUN"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{e1, e2},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Hồn Đạo - Hồn Bạo Đãng Hồn Sơn]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Hồn Bạo Đãng Hồn Sơn log, got: %+v", logs)
	}
}

// 4. Hồn Đạo: Hồn Thú Triệu Lai (SOUL_BEAST_SUMMON)
func TestSoulBeastSummon(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "SOUL_BEAST_SUMMON"},
			},
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
		if strings.Contains(l.Detail, "[Hồn Đạo - Hồn Thú Triệu Lai]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Hồn Thú Triệu Lai log, got: %+v", logs)
	}
	if player.VirtualShield <= 0 {
		t.Fatalf("expected soul beast shield gain")
	}
}

// 5. Thâu Đạo: Thâu Sinh (STEAL_LIFE)
func TestStealLife(t *testing.T) {
	player := newTestCombatant("p1", "Đạo Tặc Tiên Tôn", 800, 100)
	enemy := newTestCombatant("e1", "Kẻ Bị Hút", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "STEAL_LIFE"},
			},
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
		if strings.Contains(l.Detail, "[Thâu Đạo - Thâu Sinh]") && strings.Contains(l.Detail, "Hút thẳng") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Thâu Sinh log, got: %+v", logs)
	}
}

// 6. Vũ Đạo: Hỗn Động Toản (CHAOS_CAVE_DRILL)
func TestChaosCaveDrill(t *testing.T) {
	player := newTestCombatant("p1", "Vũ Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Hộ Thuẫn Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100
	enemy.VirtualShield = 500

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     150,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "CHAOS_CAVE_DRILL"},
			},
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
		if strings.Contains(l.Detail, "[Vũ Đạo - Hỗn Động Toản]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Hỗn Động Toản log, got: %+v", logs)
	}
	if enemy.VirtualShield > 0 {
		t.Fatalf("expected virtual shield to be wiped by Chaos Cave Drill, got: %f", enemy.VirtualShield)
	}
}

// 7. Vũ Đạo: Vô Gian Đạo (NO_EXTREME_SPACE)
func TestNoExtremeSpace(t *testing.T) {
	player := newTestCombatant("p1", "Vũ Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "NO_EXTREME_SPACE"},
			},
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
		if strings.Contains(l.Detail, "[Vũ Đạo - Vô Gian Đạo]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vô Gian Đạo log, got: %+v", logs)
	}
	if player.EvasionChance < 0.50 {
		t.Fatalf("expected evasion >= 0.50, got: %f", player.EvasionChance)
	}
}

// 8. Trụ Đạo: Quang Âm Phi Nhận (LIGHT_TIME_FLYING_BLADE)
func TestLightTimeFlyingBlade(t *testing.T) {
	player := newTestCombatant("p1", "Trụ Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Mục Tiêu", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100
	enemy.Mana = 100
	enemy.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "LIGHT_TIME_FLYING_BLADE"},
			},
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
		if strings.Contains(l.Detail, "[Trụ Đạo - Quang Âm Phi Nhận]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Quang Âm Phi Nhận log, got: %+v", logs)
	}
	if enemy.Mana >= 100 {
		t.Fatalf("expected enemy mana drain, got: %f", enemy.Mana)
	}
}

// 9. Nhân Đạo: Vạn Chúng Nhất Tâm (ALL_PEOPLE_UNITED)
func TestAllPeopleUnited(t *testing.T) {
	p1 := newTestCombatant("p1", "Thiên Đình Cổ Tiên", 1000, 100)
	p2 := newTestCombatant("p2", "Trung Châu Cổ Tiên", 1000, 100)
	enemy := newTestCombatant("e1", "Ma Đầu", 1000, 100)

	p1.Mechanics = []CombatMechanic{
		{Trigger: "ON_BATTLE_START", ActionCode: "ALL_PEOPLE_UNITED"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{p1, p2},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Nhân Đạo - Vạn Chúng Nhất Tâm]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vạn Chúng Nhất Tâm log, got: %+v", logs)
	}
	if p2.VirtualShield <= 0 {
		t.Fatalf("expected ally p2 to gain shield, got: %f", p2.VirtualShield)
	}
}

// 10. Nhân Đạo: Nhân Trung Hào Kiệt (HERO_AMONG_MEN)
func TestHeroAmongMen(t *testing.T) {
	player := newTestCombatant("p1", "Thẩm Thương", 200, 100) // 200/1000 HP (80% missing)
	player.MaxHP = 1000
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "HERO_AMONG_MEN"},
			},
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
		if strings.Contains(l.Detail, "[Nhân Đạo - Nhân Trung Hào Kiệt]") && strings.Contains(l.Detail, "Máu hao hụt") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Nhân Trung Hào Kiệt missing HP scaling log, got: %+v", logs)
	}
}

// 11. Khí Đạo: Khí Tuyệt Phùng Sinh (QI_JUE_RETURN_LIFE)
func TestQiJueReturnLife(t *testing.T) {
	player := newTestCombatant("p1", "Khí Tuyệt Ma Tiên", 250, 100) // 250/1000 HP (<= 30%)
	player.MaxHP = 1000
	attacker := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	attacker.Speed = 20
	player.Speed = 10

	player.Mechanics = []CombatMechanic{
		{Trigger: "BEFORE_HIT", ActionCode: "QI_JUE_RETURN_LIFE"},
	}

	attacker.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Damage:     50,
		},
	}
	player.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Damage:     2000,
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{player},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Khí Đạo - Khí Tuyệt Phùng Sinh]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Khí Tuyệt Phùng Sinh log, got: %+v", logs)
	}
}

// 12. Viêm Đạo: Phẫn Nộ Đích Tiểu Điểu (RAGING_ANGRY_BIRD)
func TestRagingAngryBird(t *testing.T) {
	player := newTestCombatant("p1", "Hỏa Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "RAGING_ANGRY_BIRD"},
			},
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
		if strings.Contains(l.Detail, "[Viêm Đạo - Phẫn Nộ Đích Tiểu Điểu]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Phẫn Nộ Đích Tiểu Điểu log, got: %+v", logs)
	}
}

// 13. Băng Tuyết Đạo: Cực Băng Trấn Tiên Quan (EXTREME_ICE_CRYSTAL_COFFIN)
func TestExtremeIceCrystalCoffin(t *testing.T) {
	player := newTestCombatant("p1", "Băng Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Bị Phong Ấn", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "EXTREME_ICE_CRYSTAL_COFFIN"},
			},
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
		if strings.Contains(l.Detail, "[Băng Tuyết - Trấn Tiên Quan]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Trấn Tiên Quan log, got: %+v", logs)
	}
}

// 14. Quang Đạo: Tam Thập Tam Thiên Quang (THIRTY_THREE_HEAVEN_LIGHT)
func TestThirtyThreeHeavenLight(t *testing.T) {
	player := newTestCombatant("p1", "Quang Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Quân Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "THIRTY_THREE_HEAVEN_LIGHT"},
			},
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
		if strings.Contains(l.Detail, "[Quang Đạo - Tam Thập Tam Thiên Quang]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Tam Thập Tam Thiên Quang log, got: %+v", logs)
	}
}

// 15. Ám Đạo: Đại Dạ Di Thiên (GREAT_NIGHT)
func TestGreatNight(t *testing.T) {
	p1 := newTestCombatant("p1", "Ám Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Quân Địch", 1000, 100)
	enemy.EvasionChance = 0.50

	p1.Mechanics = []CombatMechanic{
		{Trigger: "ON_BATTLE_START", ActionCode: "GREAT_NIGHT"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{p1},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Ám Đạo - Đại Dạ Di Thiên]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Đại Dạ Di Thiên log, got: %+v", logs)
	}
	if p1.CritRateBonus < 0.50 {
		t.Fatalf("expected crit rate bonus >= 0.50, got: %f", p1.CritRateBonus)
	}
}

// 16. Mộc Đạo: Khô Mộc Phùng Xuân (DRY_WOOD_SPRING)
func TestDryWoodSpring(t *testing.T) {
	player := newTestCombatant("p1", "Mộc Tu", 500, 100)
	player.MaxHP = 1000
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)

	player.ActiveEffects = []Effect{
		{ID: statusfx.StatusDryWoodSpring, Duration: 3, Type: "buff"},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Mộc Đạo - Khô Mộc Phùng Xuân]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Khô Mộc Phùng Xuân log, got: %+v", logs)
	}
}

// 17. Mộc Đạo: Khổn Tiên Đằng (IMMORTAL_BINDING_VINE)
func TestImmortalBindingVine(t *testing.T) {
	player := newTestCombatant("p1", "Mộc Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Bị Trói", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100
	enemy.Mana = 100
	enemy.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "IMMORTAL_BINDING_VINE"},
			},
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
		if strings.Contains(l.Detail, "[Mộc Đạo - Khổn Tiên Đằng]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Khổn Tiên Đằng log, got: %+v", logs)
	}
	if enemy.Mana >= 100 {
		t.Fatalf("expected enemy mana drain, got: %f", enemy.Mana)
	}
}

// 18. Huyết Đạo: Huyết Thần Tử (BLOOD_GOD_CHILD)
func TestBloodGodChild(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "BLOOD_GOD_CHILD"},
			},
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
		if strings.Contains(l.Detail, "[Huyết Đạo - Huyết Thần Tử]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Huyết Thần Tử log, got: %+v", logs)
	}
}

// 19. Sát Chiêu Đồ Đằng: Côn Bằng (TOTEM_KUN_PENG)
func TestTotemKunPeng(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     150,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "TOTEM_KUN_PENG"},
			},
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
		if strings.Contains(l.Detail, "[Đồ Đằng - Côn Bằng]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Côn Bằng log, got: %+v", logs)
	}
}

// 20. Sát Chiêu Đồ Đằng: Kim Chung (TOTEM_GOLDEN_BELL)
func TestTotemGoldenBell(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	attacker := newTestCombatant("e1", "Kẻ Tấn Công", 1000, 100)
	attacker.Speed = 20
	player.Speed = 10

	player.Mechanics = []CombatMechanic{
		{Trigger: "BEFORE_HIT", ActionCode: "TOTEM_GOLDEN_BELL"},
	}

	attacker.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Damage:     100,
		},
	}
	player.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Damage:     2000,
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{player},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Đồ Đằng - Kim Chung]") && strings.Contains(l.Detail, "phản phệ") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Đồ Đằng Kim Chung reflect log, got: %+v", logs)
	}
}

// 21. Luyện Đạo: Luyện Kỷ (REFINE_SELF)
func TestRefineSelf(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100
	player.BaseDmg = 100
	player.BaseDef = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "REFINE_SELF"},
			},
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
		if strings.Contains(l.Detail, "[Luyện Đạo - Luyện Kỷ]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Luyện Kỷ log, got: %+v", logs)
	}
	if player.BaseDmg < 120 || player.BaseDef < 120 {
		t.Fatalf("expected BaseDmg & BaseDef to increase by 20%%, got Dmg=%f, Def=%f", player.BaseDmg, player.BaseDef)
	}
}

// 22. Trí Đạo: Vạn Tinh Phi Huỳnh (MYRIAD_STAR_FIREFLY)
func TestMyriadStarFirefly(t *testing.T) {
	player := newTestCombatant("p1", "Đổng Phương Trường Phàm", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Bị Thiêu Đốt", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100
	enemy.Mana = 100
	enemy.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "MYRIAD_STAR_FIREFLY"},
			},
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
		if strings.Contains(l.Detail, "[Trí Đạo - Vạn Tinh Phi Huỳnh]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vạn Tinh Phi Huỳnh log, got: %+v", logs)
	}
	if enemy.Mana >= 100 {
		t.Fatalf("expected enemy mana burned, got: %f", enemy.Mana)
	}
}

// 23. Cấm Đạo: Ngũ Cấm Huyền Quang Khí (FIVE_RESTRICTION_DEEP_LIGHT)
func TestFiveRestrictionDeepLight(t *testing.T) {
	player := newTestCombatant("p1", "Cấm Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Bị Phong Ấn", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "FIVE_RESTRICTION_DEEP_LIGHT"},
			},
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
		if strings.Contains(l.Detail, "[Cấm Đạo - Ngũ Cấm Huyền Quang Khí]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Ngũ Cấm Huyền Quang Khí log, got: %+v", logs)
	}
}

// 24. Mị Đạo: Tử Quỷ Khoái Lai (CHARM_GHOST_COME)
func TestCharmGhostCome(t *testing.T) {
	player := newTestCombatant("p1", "Mị Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Bị Mê Hoặc", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "CHARM_GHOST_COME"},
			},
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
		if strings.Contains(l.Detail, "[Mị Đạo - Tử Quỷ Khoái Lai]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Tử Quỷ Khoái Lai log, got: %+v", logs)
	}
}

// 25. Thực Đạo: Thôn Thiên Địa (DEVOUR_HEAVEN_EARTH)
func TestDevourHeavenEarth(t *testing.T) {
	player := newTestCombatant("p1", "Thực Tu", 800, 100)
	enemy := newTestCombatant("e1", "Kẻ Có Khiên Khủng", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100
	enemy.VirtualShield = 400

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "DEVOUR_HEAVEN_EARTH"},
			},
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
		if strings.Contains(l.Detail, "[Thực Đạo - Thôn Thiên Địa]") && strings.Contains(l.Detail, "Nuốt chửng") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Thôn Thiên Địa log, got: %+v", logs)
	}
}
