package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// 1. Âm Đạo: Ly Ca (SONG_OF_SEPARATION) - Phượng Cửu Ca
func TestSongOfSeparation(t *testing.T) {
	player := newTestCombatant("p1", "Phượng Cửu Ca", 1000, 100)
	enemy := newTestCombatant("e1", "Địch Thủ", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	// Enemy has 2 buffs and 200 shield
	enemy.VirtualShield = 200
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusAtkUp, Value: 20, Duration: 3, Type: "buff"},
		{ID: statusfx.StatusDefUp, Value: 20, Duration: 3, Type: "buff"},
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "SONG_OF_SEPARATION"},
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
		if strings.Contains(l.Detail, "[Âm Đạo - Ly Ca]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Ly Ca log, got: %+v", logs)
	}

	// Should dispel the 2 buffs
	buffCount := 0
	for _, eff := range enemy.ActiveEffects {
		if eff.Type == "buff" {
			buffCount++
		}
	}
	if buffCount > 0 {
		t.Fatalf("expected buffs to be dispelled, remaining: %d", buffCount)
	}
}

// 2. Âm / Vận Đạo: Vận Mệnh Ca (SONG_OF_DESTINY) - Phượng Cửu Ca
func TestSongOfDestiny(t *testing.T) {
	player := newTestCombatant("p1", "Phượng Cửu Ca", 1000, 100)
	ally := newTestCombatant("p2", "Đồng Đội", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	ally.Speed = 15
	enemy.Speed = 10
	player.Mana = 50
	player.MaxMana = 100
	ally.Mana = 20
	ally.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "SONG_OF_DESTINY"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player, ally},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	found := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Âm Vận Đạo - Vận Mệnh Ca]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vận Mệnh Ca log, got: %+v", logs)
	}
	if ally.DamageMultiplier < 1.30 {
		t.Fatalf("expected ally damage multiplier boosted, got: %f", ally.DamageMultiplier)
	}
	if enemy.DamageMultiplier > 0.80 {
		t.Fatalf("expected enemy damage multiplier reduced, got: %f", enemy.DamageMultiplier)
	}
}

// 3. Phong Đạo: Tống Hữu Phong (FAREWELL_FRIEND_WIND) - Vũ Dung
func TestFarewellFriendWind(t *testing.T) {
	player := newTestCombatant("p1", "Vũ Dung", 1000, 100)
	enemy := newTestCombatant("e1", "Bạn Cũ", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	// Enemy has huge shield
	enemy.VirtualShield = 500

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "FAREWELL_FRIEND_WIND"},
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
	foundCurseTick := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Phong Đạo - Tống Hữu Phong]") {
			found = true
		}
		if strings.Contains(l.Detail, "Gió Tiễn Bạn") && strings.Contains(l.Detail, "tự tổn") {
			foundCurseTick = true
		}
	}
	if !found {
		t.Fatalf("expected Tống Hữu Phong log, got: %+v", logs)
	}
	if !foundCurseTick {
		t.Fatalf("expected Tống Hữu Chi Phong curse DoT tick log, got: %+v", logs)
	}
	// Damage should bypass shield and hit HP directly
	if enemy.HP >= 1000 {
		t.Fatalf("expected enemy HP to be damaged directly through shield, got: %f", enemy.HP)
	}
}

// 4. Phong Đạo: Vô Hạn Phong (INFINITE_WIND) - Vũ Dung
func TestInfiniteWind(t *testing.T) {
	player := newTestCombatant("p1", "Vũ Dung", 1000, 100)
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
				{Trigger: "ON_ACTION", ActionCode: "INFINITE_WIND"},
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
		if strings.Contains(l.Detail, "[Phong Đạo - Vô Hạn Phong]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vô Hạn Phong log, got: %+v", logs)
	}
	if player.EvasionChance < 0.35 {
		t.Fatalf("expected player evasion >= 0.35, got: %f", player.EvasionChance)
	}
}

// 5. Huyết Đạo: Huyết Nhiễm Chinh Bào (BLOOD_STAINED_ROBE)
func TestBloodStainedRobe(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	attacker := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	attacker.Speed = 20
	player.Speed = 10

	player.Mechanics = []CombatMechanic{
		{Trigger: "BEFORE_HIT", ActionCode: "BLOOD_STAINED_ROBE"},
	}

	attacker.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Damage:     100,
		},
	}
	// Player finishes combat after receiving strike
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
		if strings.Contains(l.Detail, "[Huyết Đạo - Huyết Nhiễm Chinh Bào]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Huyết Nhiễm Chinh Bào log, got: %+v", logs)
	}
	if player.HP <= 900 {
		t.Fatalf("expected player HP to be >= 950 due to robe mitigation, got: %f", player.HP)
	}
}

// 6. Huyết Đạo: Huyết Ma Giải Thể (BLOOD_DEMON_DISINTEGRATION)
func TestBloodDemonDisintegration(t *testing.T) {
	player := newTestCombatant("p1", "Ma Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Chính Đạo", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     150,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "BLOOD_DEMON_DISINTEGRATION"},
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
		if strings.Contains(l.Detail, "[Huyết Đạo - Huyết Ma Giải Thể]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Huyết Ma Giải Thể log, got: %+v", logs)
	}
}

// 7. Hồn Đạo: Diêm Đế / Quỷ Quan Y (YAMA_EMPEROR)
func TestYamaEmperor(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	enemy := newTestCombatant("e1", "Bạch Ngưng Băng", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "YAMA_EMPEROR"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundCast := false
	foundLeech := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Hồn Đạo - Diêm Đế]") && strings.Contains(l.Detail, "khoác lên mình Quỷ Quan Y") {
			foundCast = true
		}
		if strings.Contains(l.Detail, "Diêm La Tử lao ra") {
			foundLeech = true
		}
	}
	if !foundCast {
		t.Fatalf("expected Diêm Đế cast log, got: %+v", logs)
	}
	if !foundLeech {
		t.Fatalf("expected Diêm La Tử soul leech log, got: %+v", logs)
	}
}

// 8. Khí Đạo: Vô Lượng Khí Hải (IMMEASURABLE_QI_SEA)
func TestImmeasurableQiSea(t *testing.T) {
	player := newTestCombatant("p1", "Khí Hải Lão Tổ", 1000, 100)
	e1 := newTestCombatant("e1", "Kẻ Địch 1", 1000, 100)
	e2 := newTestCombatant("e2", "Kẻ Địch 2", 1000, 100)
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
				{Trigger: "ON_ACTION", ActionCode: "IMMEASURABLE_QI_SEA"},
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
		if strings.Contains(l.Detail, "[Khí Đạo - Vô Lượng Khí Hải]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vô Lượng Khí Hải log, got: %+v", logs)
	}
	if e1.DamageMultiplier > 0.80 || e2.DamageMultiplier > 0.80 {
		t.Fatalf("expected all enemies damage multiplier reduced, got: e1=%f, e2=%f", e1.DamageMultiplier, e2.DamageMultiplier)
	}
}

// 9. Khí Đạo: Tam Khí Quy Lai (THREE_QI_RETURN)
func TestThreeQiReturn(t *testing.T) {
	player := newTestCombatant("p1", "Nguyên Thủy Tiên Tôn", 1000, 100)
	enemy := newTestCombatant("e1", "Túc Mệnh Cố Thủ", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.HP = 400
	player.MaxHP = 1000
	player.Mana = 10
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "THREE_QI_RETURN"},
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
		if strings.Contains(l.Detail, "[Khí Đạo - Tam Khí Quy Lai]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Tam Khí Quy Lai log, got: %+v", logs)
	}
	if player.HP <= 400 {
		t.Fatalf("expected HP heal from Three Qi Return, got: %f", player.HP)
	}
	if player.Mana < 100 {
		t.Fatalf("expected full mana restore, got: %f", player.Mana)
	}
}

// 10. Tín Đạo: Tam Nhân Thành Hổ (THREE_MEN_MAKE_TIGER)
func TestThreeMenMakeTiger(t *testing.T) {
	player := newTestCombatant("p1", "Tín Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Nghi Tâm Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "THREE_MEN_MAKE_TIGER"},
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
		if strings.Contains(l.Detail, "[Tín Đạo - Tam Nhân Thành Hổ]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Tam Nhân Thành Hổ log, got: %+v", logs)
	}
}

// 11. Tín Đạo: Nhân Ngôn Khả Úy (WORDS_ARE_FRIGHTENING)
func TestWordsAreFrightening(t *testing.T) {
	player := newTestCombatant("p1", "Tín Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Bị Khẩu Nghiệp", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	// Pre-apply 2 debuffs to enemy
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusBleed, Value: 10, Duration: 3, Type: "debuff"},
		{ID: statusfx.StatusWeak, Value: 10, Duration: 3, Type: "debuff"},
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "WORDS_ARE_FRIGHTENING"},
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
		if strings.Contains(l.Detail, "[Tín Đạo - Nhân Ngôn Khả Úy]") && strings.Contains(l.Detail, "Kích hoạt 2 debuff") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Nhân Ngôn Khả Úy debuff scaling log, got: %+v", logs)
	}
	if !enemy.IsSilenced {
		t.Fatalf("expected enemy to be silenced")
	}
}

// 12. Thiên Đạo: Mệnh Bại (FATE_VANQUISHED) - Hồng Liên Ma Tôn
func TestFateVanquished(t *testing.T) {
	player := newTestCombatant("p1", "Hồng Liên Ma Tôn", 1000, 100)
	enemy := newTestCombatant("e1", "Thiên Đình Cự Đầu", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	// Enemy has Invincible (Bất Hoại), LastStand (Bất Tử), and 500 shield
	enemy.IsInvincible = true
	enemy.IsLastStand = true
	enemy.VirtualShield = 500
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusInvincible, Duration: 5, Type: "buff"},
		{ID: statusfx.StatusLastStand, Duration: 5, Type: "buff"},
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     200,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "FATE_VANQUISHED"},
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
		if strings.Contains(l.Detail, "[Thiên Đạo - Mệnh Bại]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Mệnh Bại log, got: %+v", logs)
	}
	if enemy.IsInvincible || enemy.IsLastStand {
		t.Fatalf("expected Mệnh Bại to strip Invincible and LastStand")
	}
	if enemy.VirtualShield > 0 {
		t.Fatalf("expected Mệnh Bại to zero VirtualShield, got: %f", enemy.VirtualShield)
	}
}

// 13. Thiên Đạo: Lưới Trời Tuy Thưa (HEAVEN_NET_RESTRICTION)
func TestHeavenNetRestriction(t *testing.T) {
	player := newTestCombatant("p1", "Thiên Ý", 1000, 100)
	enemy := newTestCombatant("e1", "Nghịch Thiên Giả", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100
	enemy.EvasionChance = 0.60

	player.Mechanics = []CombatMechanic{
		{Trigger: "ON_BATTLE_START", ActionCode: "HEAVEN_NET_RESTRICTION"},
	}
	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "HEAVEN_NET_RESTRICTION"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundBattleStart := false
	foundAction := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Thiên Đạo - Lưới Trời Tuy Thưa]") && strings.Contains(l.Detail, "Khôi Khôi Thiên Võng phong tỏa") {
			foundBattleStart = true
		}
		if strings.Contains(l.Detail, "[Thiên Đạo - Lưới Trời Tuy Thưa]") && strings.Contains(l.Detail, "Thiên võng giăng kín tứ phương") {
			foundAction = true
		}
	}
	if !foundBattleStart {
		t.Fatalf("expected battle start Heaven Net log, got: %+v", logs)
	}
	if !foundAction {
		t.Fatalf("expected action Heaven Net log, got: %+v", logs)
	}
}

// 14. Trụ Đạo: Vị Lai Thân (FUTURE_SELF)
func TestFutureSelf(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	enemy := newTestCombatant("e1", "Bát Chuyển Cổ Tiên", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "FUTURE_SELF"},
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
		if strings.Contains(l.Detail, "[Trụ Đạo - Vị Lai Thân]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vị Lai Thân log, got: %+v", logs)
	}
	if player.DamageMultiplier < 1.70 {
		t.Fatalf("expected player damage multiplier boosted, got: %f", player.DamageMultiplier)
	}
}

// 15. Trụ Đạo: Giang Sơn Như Cố (LANDSCAPE_AS_BEFORE)
func TestLandscapeAsBefore(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	enemy := newTestCombatant("e1", "Kẻ Địch", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	// Player is debuffed
	player.ActiveEffects = []Effect{
		{ID: statusfx.StatusBleed, Value: 20, Duration: 3, Type: "debuff"},
		{ID: statusfx.StatusWeak, Value: 20, Duration: 3, Type: "debuff"},
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "LANDSCAPE_AS_BEFORE"},
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
		if strings.Contains(l.Detail, "[Trụ Đạo - Giang Sơn Như Cố]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Giang Sơn Như Cố log, got: %+v", logs)
	}
	if player.VirtualShield <= 0 {
		t.Fatalf("expected virtual shield restore from Landscape as Before")
	}
}

// 16. Thực Đạo: Cật Khuy Thị Phúc (EATING_LOSS_BLESSING)
func TestEatingLossBlessing(t *testing.T) {
	player := newTestCombatant("p1", "Thực Tu", 1000, 100)
	attacker := newTestCombatant("e1", "Kẻ Tấn Công", 1000, 100)
	attacker.Speed = 20
	player.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Mechanics = []CombatMechanic{
		{Trigger: "BEFORE_HIT", ActionCode: "EATING_LOSS_BLESSING"},
	}
	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "EATING_LOSS_BLESSING"},
			},
		},
	}

	attacker.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Damage:     50,
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{player},
	}

	logs := Simulate(state)

	foundBeforeHit := false
	foundAction := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Thực Đạo - Cật Khuy Thị Phúc]") && strings.Contains(l.Detail, "tích lũy tầng") {
			foundBeforeHit = true
		}
		if strings.Contains(l.Detail, "[Thực Đạo - Cật Khuy Thị Phúc]") && strings.Contains(l.Detail, "Chuyển hóa") {
			foundAction = true
		}
	}
	if !foundBeforeHit {
		t.Fatalf("expected eating loss stack accumulation log, got: %+v", logs)
	}
	if !foundAction {
		t.Fatalf("expected eating loss action consumption log, got: %+v", logs)
	}
}

// 17. Luật Đạo: Vạn Giải (MYRIAD_DISMANTLE)
func TestMyriadDismantle(t *testing.T) {
	player := newTestCombatant("p1", "Luật Tu", 1000, 100)
	enemy := newTestCombatant("e1", "Vô Số Buff", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	enemy.VirtualShield = 400
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusAtkUp, Value: 30, Duration: 3, Type: "buff"},
		{ID: statusfx.StatusDefUp, Value: 30, Duration: 3, Type: "buff"},
	}

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     150,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "MYRIAD_DISMANTLE"},
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
		if strings.Contains(l.Detail, "[Luật Đạo - Vạn Giải]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Vạn Giải log, got: %+v", logs)
	}
	if enemy.VirtualShield > 0 {
		t.Fatalf("expected virtual shield to be completely dismantled, got: %f", enemy.VirtualShield)
	}
}

// 18. Sát Chiêu Đồ Đằng: U Đô Lực Bưu (YOU_DU_STRENGTH_TIGER)
func TestYouDuStrengthTiger(t *testing.T) {
	player := newTestCombatant("p1", "Phương Nguyên", 1000, 100)
	enemy := newTestCombatant("e1", "Địch Thủ", 1000, 100)
	player.Speed = 20
	enemy.Speed = 10
	player.Mana = 100
	player.MaxMana = 100

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Damage:     100,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "YOU_DU_STRENGTH_TIGER"},
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
		if strings.Contains(l.Detail, "[Đồ Đằng - U Đô Lực Bưu]") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected U Đô Lực Bưu log, got: %+v", logs)
	}
	if !enemy.IsStunned {
		t.Fatalf("expected enemy to be stunned")
	}
}

// 19. Viêm Đạo: Niết Bàn Hỏa (NIRVANA_FIRE)
func TestNirvanaFire(t *testing.T) {
	defender := newTestCombatant("p1", "Phượng Hoàng", 100, 100)
	attacker := newTestCombatant("e1", "Kẻ Giết", 50, 100) // 50 HP so Nirvana fire retaliatory blast kills attacker
	attacker.Speed = 20
	defender.Speed = 10
	defender.BaseDmg = 100

	defender.Mechanics = []CombatMechanic{
		{Trigger: "ON_DEATH", ActionCode: "NIRVANA_FIRE"},
	}

	// Lethal blow: 500 damage to 100 HP defender
	attacker.Actions = []SimAction{
		{
			ActionType: "BASIC_ATTACK",
			Damage:     500,
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{attacker},
		Enemies: []*CombatantAuto{defender},
	}

	logs := Simulate(state)

	foundRebirth := false
	foundBlast := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Viêm Đạo - Niết Bàn Hỏa]") && strings.Contains(l.Detail, "dục hỏa trùng sinh") {
			foundRebirth = true
		}
		if strings.Contains(l.Detail, "[Niết Bàn Hỏa] Dư chấn hỏa diễm") {
			foundBlast = true
		}
	}
	if !foundRebirth {
		t.Fatalf("expected Niết Bàn Hỏa rebirth log, got: %+v", logs)
	}
	if !foundBlast {
		t.Fatalf("expected Niết Bàn Hỏa retaliatory fire blast log, got: %+v", logs)
	}
	if defender.HP <= 0 {
		t.Fatalf("expected defender to be revived by Nirvana Fire, got HP: %f", defender.HP)
	}
}
