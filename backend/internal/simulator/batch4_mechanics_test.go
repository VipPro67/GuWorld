package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestMetalPathAegisAndArmorBreak(t *testing.T) {
	// Kim Đạo: Cương Thiết Kim Thân & Phá Giáp Kim Mang
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Cương Thiết Vô Song",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "METAL_AEGIS"},
				{Trigger: "ON_ACTION", ActionCode: "ARMOR_BREAK_HEAVY"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundAegis := false
	foundBreak := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Kim Đạo - Cương Thiết Kim Thân]") {
			foundAegis = true
		}
		if strings.Contains(l.Detail, "[Kim Đạo - Phá Giáp Kim Mang]") {
			foundBreak = true
		}
	}

	if !foundAegis {
		t.Fatalf("expected METAL_AEGIS in logs, got: %+v", logs)
	}
	if !foundBreak {
		t.Fatalf("expected ARMOR_BREAK_HEAVY in logs, got: %+v", logs)
	}
}

func TestWoodPathRegenAndBind(t *testing.T) {
	// Mộc Đạo: Sinh Sinh Bất Tức & Mộc Lao Trói Buộc
	player := newTestCombatant("player", "Player", 1000, 100)
	player.Mana = 20
	player.MaxMana = 100
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Sinh Cơ Trói Buộc",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "WOOD_REGEN_BURST"},
				{Trigger: "ON_ACTION", ActionCode: "WOOD_BIND"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundRegen := false
	foundBind := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Mộc Đạo - Sinh Sinh Bất Tức]") {
			foundRegen = true
		}
		if strings.Contains(l.Detail, "[Mộc Đạo - Mộc Lao Trói Buộc]") {
			foundBind = true
		}
	}

	if !foundRegen {
		t.Fatalf("expected WOOD_REGEN_BURST in logs, got: %+v", logs)
	}
	if !foundBind {
		t.Fatalf("expected WOOD_BIND in logs, got: %+v", logs)
	}
}

func TestWaterPathCleanseAndMirrorShield(t *testing.T) {
	// Thủy Đạo: Thủy Lưu Triều Tịch & Thủy Kính Hóa Giải
	player := newTestCombatant("player", "Player", 1000, 100)
	player.HP = 800
	player.ActiveEffects = []Effect{
		{ID: statusfx.StatusBurn, Value: 10, Duration: 3},
		{ID: statusfx.StatusBleed, Value: 20, Duration: 3},
	}
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Thủy Lưu Hóa Cảnh",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "WATER_CLEANSE"},
				{Trigger: "ON_CAST", ActionCode: "WATER_MIRROR_SHIELD"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundCleanse := false
	foundMirror := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Thủy Đạo - Thủy Lưu Triều Tịch]") {
			foundCleanse = true
		}
		if strings.Contains(l.Detail, "[Thủy Đạo - Thủy Kính Hóa Giải]") {
			foundMirror = true
		}
	}

	if !foundCleanse {
		t.Fatalf("expected WATER_CLEANSE in logs, got: %+v", logs)
	}
	if !foundMirror {
		t.Fatalf("expected WATER_MIRROR_SHIELD in logs, got: %+v", logs)
	}
}

func TestFirePathInfernoAndIgniteBurst(t *testing.T) {
	// Hỏa Đạo: Liệt Hỏa Liêu Nguyên & Nộ Hỏa Bạo Phát
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusBurn, Value: 10, Duration: 3},
	}
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Hỏa Diễm Bạo Liệt",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "FIRE_INFERNO"},
				{Trigger: "ON_ACTION", ActionCode: "IGNITE_BURST"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundInferno := false
	foundIgnite := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Hỏa Đạo - Liệt Hỏa Liêu Nguyên]") {
			foundInferno = true
		}
		if strings.Contains(l.Detail, "[Hỏa Đạo - Nộ Hỏa Bạo Phát]") {
			foundIgnite = true
		}
	}

	if !foundInferno {
		t.Fatalf("expected FIRE_INFERNO in logs, got: %+v", logs)
	}
	if !foundIgnite {
		t.Fatalf("expected IGNITE_BURST in logs, got: %+v", logs)
	}
}

func TestEarthPathShieldAndGravity(t *testing.T) {
	// Thổ Đạo: Đại Địa Hộ Thuẫn & Địa Chấn Trọng Lực
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Đại Địa Nham Thuẫn",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "EARTH_SHIELD"},
				{Trigger: "ON_ACTION", ActionCode: "EARTH_GRAVITY"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundShield := false
	foundGravity := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Thổ Đạo - Đại Địa Hộ Thuẫn]") {
			foundShield = true
		}
		if strings.Contains(l.Detail, "[Thổ Đạo - Địa Chấn Trọng Lực]") {
			foundGravity = true
		}
	}

	if !foundShield {
		t.Fatalf("expected EARTH_SHIELD in logs, got: %+v", logs)
	}
	if !foundGravity {
		t.Fatalf("expected EARTH_GRAVITY in logs, got: %+v", logs)
	}
}

func TestWindPathHaste(t *testing.T) {
	// Phong Đạo: Tật Phong Tốc Độ
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Ngự Phong Thuật",
			Damage:     50,
			Mechanics:  []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "WIND_HASTE_BURST"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundHaste := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Phong Đạo - Tật Phong Tốc Độ]") {
			foundHaste = true
			break
		}
	}

	if !foundHaste {
		t.Fatalf("expected WIND_HASTE_BURST in logs, got: %+v", logs)
	}
}

func TestLightningPathStunAndChain(t *testing.T) {
	// Lôi Đạo: Lôi Đình Vạn Quân & Lôi Điện Truyền Dẫn
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy1 := newTestCombatant("enemy1", "Enemy 1", 1000, 50)
	enemy2 := newTestCombatant("enemy2", "Enemy 2", 1000, 50)
	player.Speed = 20
	enemy1.Speed = 10
	enemy2.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vạn Lôi Oanh Đỉnh",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "LIGHTNING_STUN_BURST"},
				{Trigger: "ON_ACTION", ActionCode: "CHAIN_LIGHTNING"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy1, enemy2},
	}

	logs := Simulate(state)

	foundStun := false
	foundChain := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Lôi Đạo - Lôi Đình Vạn Quân]") {
			foundStun = true
		}
		if strings.Contains(l.Detail, "[Lôi Đạo - Lôi Điện Truyền Dẫn]") {
			foundChain = true
		}
	}

	if !foundStun {
		t.Fatalf("expected LIGHTNING_STUN_BURST in logs, got: %+v", logs)
	}
	if !foundChain {
		t.Fatalf("expected CHAIN_LIGHTNING in logs, got: %+v", logs)
	}
}

func TestIcePathFreezeAndFrostbite(t *testing.T) {
	// Băng Đạo: Băng Phong Vạn Lý & Hàn Băng Tê Cóng
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Băng Hà Vạn Cổ",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "ICE_DEEP_FREEZE"},
				{Trigger: "ON_ACTION", ActionCode: "ICE_FROSTBITE"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundFreeze := false
	foundFrostbite := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Băng Đạo - Băng Phong Vạn Lý]") {
			foundFreeze = true
		}
		if strings.Contains(l.Detail, "[Băng Đạo - Hàn Băng Tê Cóng]") {
			foundFrostbite = true
		}
	}

	if !foundFreeze {
		t.Fatalf("expected ICE_DEEP_FREEZE in logs, got: %+v", logs)
	}
	if !foundFrostbite {
		t.Fatalf("expected ICE_FROSTBITE in logs, got: %+v", logs)
	}
}

func TestLightPathBlindAndPurify(t *testing.T) {
	// Quang Đạo: Manh Mục Cực Quang & Thánh Quang Phục Thể
	player := newTestCombatant("player", "Player", 1000, 100)
	player.HP = 700
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Thánh Quang Phổ Chiếu",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "LIGHT_BLINDING_FLASH"},
				{Trigger: "ON_CAST", ActionCode: "LIGHT_PURIFY"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundBlind := false
	foundPurify := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Quang Đạo - Manh Mục Cực Quang]") {
			foundBlind = true
		}
		if strings.Contains(l.Detail, "[Quang Đạo - Thánh Quang Phục Thể]") {
			foundPurify = true
		}
	}

	if !foundBlind {
		t.Fatalf("expected LIGHT_BLINDING_FLASH in logs, got: %+v", logs)
	}
	if !foundPurify {
		t.Fatalf("expected LIGHT_PURIFY in logs, got: %+v", logs)
	}
}

func TestDarkPathStealthAndCurse(t *testing.T) {
	// Ám Đạo: Ám Dạ Ẩn Nấp & Ám Ảnh Hủ Thực
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	enemy.Mana = 100
	enemy.MaxMana = 100
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Hắc Ám Thôn Phệ",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "DARK_STEALTH"},
				{Trigger: "ON_ACTION", ActionCode: "DARK_CURSE_WEAK"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundStealth := false
	foundCurse := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Ám Đạo - Ám Dạ Ẩn Nấp]") {
			foundStealth = true
		}
		if strings.Contains(l.Detail, "[Ám Đạo - Ám Ảnh Hủ Thực]") {
			foundCurse = true
		}
	}

	if !foundStealth {
		t.Fatalf("expected DARK_STEALTH in logs, got: %+v", logs)
	}
	if !foundCurse {
		t.Fatalf("expected DARK_CURSE_WEAK in logs, got: %+v", logs)
	}
}

func TestStarPathRainAndThought(t *testing.T) {
	// Tinh Đạo: Tinh Quang Lạc Trần & Tinh Niệm Tụ Hồn
	player := newTestCombatant("player", "Player", 1000, 100)
	player.Mana = 20
	player.MaxMana = 100
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vạn Tinh Trụy Lạc",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "STAR_FIRE_RAIN"},
				{Trigger: "ON_CAST", ActionCode: "STAR_THOUGHT_RESTORE"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundRain := false
	foundThought := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Tinh Đạo - Tinh Quang Lạc Trần]") {
			foundRain = true
		}
		if strings.Contains(l.Detail, "[Tinh Đạo - Tinh Niệm Tụ Hồn]") {
			foundThought = true
		}
	}

	if !foundRain {
		t.Fatalf("expected STAR_FIRE_RAIN in logs, got: %+v", logs)
	}
	if !foundThought {
		t.Fatalf("expected STAR_THOUGHT_RESTORE in logs, got: %+v", logs)
	}
}

func TestSoundPathSilenceAndResonance(t *testing.T) {
	// Âm Đạo: Ma Âm Quán Nhĩ & Ba Động Chấn Thần
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	enemy.VirtualShield = 400
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Cộng Hưởng Ma Âm",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "SOUND_DISRUPT_SILENCE"},
				{Trigger: "ON_ACTION", ActionCode: "SOUND_RESONANCE"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundSilence := false
	foundResonance := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Âm Đạo - Ma Âm Quán Nhĩ]") {
			foundSilence = true
		}
		if strings.Contains(l.Detail, "[Âm Đạo - Ba Động Chấn Thần]") {
			foundResonance = true
		}
	}

	if !foundSilence {
		t.Fatalf("expected SOUND_DISRUPT_SILENCE in logs, got: %+v", logs)
	}
	if !foundResonance {
		t.Fatalf("expected SOUND_RESONANCE in logs, got: %+v", logs)
	}
}

func TestCloudPathMist(t *testing.T) {
	// Vân Đạo: Vân Vụ Ảo Mộng
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Vân Ẩn Hộ Thể",
			Damage:     50,
			Mechanics:  []CombatMechanic{{Trigger: "ON_CAST", ActionCode: "CLOUD_MIST"}},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundMist := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Vân Đạo - Vân Vụ Ảo Mộng]") {
			foundMist = true
			break
		}
	}

	if !foundMist {
		t.Fatalf("expected CLOUD_MIST in logs, got: %+v", logs)
	}
}
