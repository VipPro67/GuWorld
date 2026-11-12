package simulator

import (
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestHeavenPathSuppressionAndTiepThienDia(t *testing.T) {
	// Thiên Đạo: Thiên Ý Khó Dò & Tiếp Thiên Tiếp Địa
	player := newTestCombatant("player", "Player", 1000, 100)
	player.Mana = 20
	player.MaxMana = 100
	player.IsStunned = true

	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusAtkUp, Type: "buff", Value: 20, Duration: 3},
		{ID: statusfx.StatusDefUp, Type: "buff", Value: 20, Duration: 3},
	}
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Thiên Ý Giáng Phạt",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "TIEP_THIEN_DIA"},
				{Trigger: "ON_ACTION", ActionCode: "HEAVEN_SUPPRESSION"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundTiep := false
	foundSuppression := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Thiên Đạo - Tiếp Thiên Tiếp Địa]") {
			foundTiep = true
		}
		if strings.Contains(l.Detail, "[Thiên Đạo - Thiên Ý Khó Dò]") {
			foundSuppression = true
		}
	}

	if !foundTiep {
		t.Fatalf("expected TIEP_THIEN_DIA in logs, got: %+v", logs)
	}
	if !foundSuppression {
		t.Fatalf("expected HEAVEN_SUPPRESSION in logs, got: %+v", logs)
	}
	// Player should be freed from stun and have mana
	if player.Mana <= 20 {
		t.Fatalf("expected mana restored by TIEP_THIEN_DIA, got: %.0f", player.Mana)
	}
}

func TestHumanPathWillAndVanDanDongTam(t *testing.T) {
	// Nhân Đạo: Nhân Định Thắng Thiên & Vạn Dân Đồng Tâm
	player := newTestCombatant("player", "Player", 1000, 100)
	player.HP = 300 // <= 40% HP triggers HUMAN_WILL
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Nhân Đạo Hy Vọng",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "VAN_DAN_DONG_TAM"},
				{Trigger: "ON_ACTION", ActionCode: "HUMAN_WILL"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundVanDan := false
	foundWill := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Nhân Đạo - Vạn Dân Đồng Tâm]") {
			foundVanDan = true
		}
		if strings.Contains(l.Detail, "[Nhân Đạo - Nhân Định Thắng Thiên]") {
			foundWill = true
		}
	}

	if !foundVanDan {
		t.Fatalf("expected VAN_DAN_DONG_TAM in logs, got: %+v", logs)
	}
	if !foundWill {
		t.Fatalf("expected HUMAN_WILL in logs, got: %+v", logs)
	}
	if player.VirtualShield <= 0 {
		t.Fatalf("expected VirtualShield from VAN_DAN_DONG_TAM, got: %.0f", player.VirtualShield)
	}
}

func TestQiPathWallAndExplosion(t *testing.T) {
	// Khí Đạo: Khí Tường Hộ Thể & Khí Bạo Trảm Thiên
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Nguyên Khí Bạo Liệt",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "QI_WALL"},
				{Trigger: "ON_ACTION", ActionCode: "QI_EXPLOSION"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundWall := false
	foundExplosion := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Khí Đạo - Khí Tường Hộ Thể]") {
			foundWall = true
		}
		if strings.Contains(l.Detail, "[Khí Đạo - Khí Bạo Trảm Thiên]") {
			foundExplosion = true
		}
	}

	if !foundWall {
		t.Fatalf("expected QI_WALL in logs, got: %+v", logs)
	}
	if !foundExplosion {
		t.Fatalf("expected QI_EXPLOSION in logs, got: %+v", logs)
	}
}

func TestSpacePathSpatialShiftAndDimensionSlash(t *testing.T) {
	// Không Đạo: Hư Không Thuấn Di & Thứ Nguyên Trảm
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Thứ Nguyên Liệt Trảm",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "SPATIAL_SHIFT"},
				{Trigger: "ON_ACTION", ActionCode: "DIMENSION_SLASH"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundShift := false
	foundSlash := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Không Đạo - Hư Không Thuấn Di]") {
			foundShift = true
		}
		if strings.Contains(l.Detail, "[Không Đạo - Thứ Nguyên Trảm]") {
			foundSlash = true
		}
	}

	if !foundShift {
		t.Fatalf("expected SPATIAL_SHIFT in logs, got: %+v", logs)
	}
	if !foundSlash {
		t.Fatalf("expected DIMENSION_SLASH in logs, got: %+v", logs)
	}
}

func TestRulePathImpositionAndJusticeVerdict(t *testing.T) {
	// Quy Tắc Đạo: Định Quy Đặt Củ & Phán Quyết Công Lý
	player := newTestCombatant("player", "Player", 1000, 100)
	enemy := newTestCombatant("enemy", "Enemy", 1000, 50)
	enemy.ActiveEffects = []Effect{
		{ID: statusfx.StatusAtkUp, Type: "buff", Value: 20, Duration: 3},
	}
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Quy Củ Phán Quyết",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_ACTION", ActionCode: "RULE_IMPOSITION"},
				{Trigger: "ON_ACTION", ActionCode: "JUSTICE_VERDICT"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundRule := false
	foundJustice := false
	for _, l := range logs {
		if strings.Contains(l.Detail, "[Quy Tắc Đạo - Định Quy Đặt Củ]") {
			foundRule = true
		}
		if strings.Contains(l.Detail, "[Quy Tắc Đạo - Phán Quyết Công Lý]") {
			foundJustice = true
		}
	}

	if !foundRule {
		t.Fatalf("expected RULE_IMPOSITION in logs, got: %+v", logs)
	}
	if !foundJustice {
		t.Fatalf("expected JUSTICE_VERDICT in logs, got: %+v", logs)
	}
}

func TestFormationPhantomPaintingEmotionRefinement(t *testing.T) {
	// Trận Đạo, Huyễn Đạo, Họa Đạo, Tình Đạo, Luyện Đạo
	player := newTestCombatant("player", "Player", 1000, 100)
	player.HP = 300 // <= 35% HP for PASSION_OVERLOAD
	enemy := newTestCombatant("enemy", "Enemy", 2000, 50)
	player.Speed = 20
	enemy.Speed = 10

	player.Actions = []SimAction{
		{
			ActionType: "GU_SKILL",
			Name:       "Hỗn Hợp Đạo Kỹ",
			Damage:     50,
			Mechanics: []CombatMechanic{
				{Trigger: "ON_CAST", ActionCode: "FORMATION_DOMAIN"},
				{Trigger: "ON_ACTION", ActionCode: "BLINDING_MIRAGE"},
				{Trigger: "ON_ACTION", ActionCode: "PAINTING_IMPRISONMENT"},
				{Trigger: "ON_ACTION", ActionCode: "PASSION_OVERLOAD"},
				{Trigger: "ON_ACTION", ActionCode: "GU_DETONATION"},
			},
		},
	}

	state := &CombatState{
		Seed:    42,
		Allies:  []*CombatantAuto{player},
		Enemies: []*CombatantAuto{enemy},
	}

	logs := Simulate(state)

	foundFormation := false
	foundMirage := false
	foundPainting := false
	foundPassion := false
	foundDetonation := false

	for _, l := range logs {
		if strings.Contains(l.Detail, "[Trận Đạo - Trận Địa Khống Chế]") {
			foundFormation = true
		}
		if strings.Contains(l.Detail, "[Huyễn Đạo - Huyễn Ảnh Mê Trận]") {
			foundMirage = true
		}
		if strings.Contains(l.Detail, "[Họa Đạo - Họa Địa Vi Lao]") {
			foundPainting = true
		}
		if strings.Contains(l.Detail, "[Tình Đạo - Tình Cổ Vô Hối]") {
			foundPassion = true
		}
		if strings.Contains(l.Detail, "[Luyện Đạo - Tự Bạo Cổ Trùng]") {
			foundDetonation = true
		}
	}

	if !foundFormation {
		t.Fatalf("expected FORMATION_DOMAIN in logs, got: %+v", logs)
	}
	if !foundMirage {
		t.Fatalf("expected BLINDING_MIRAGE in logs, got: %+v", logs)
	}
	if !foundPainting {
		t.Fatalf("expected PAINTING_IMPRISONMENT in logs, got: %+v", logs)
	}
	if !foundPassion {
		t.Fatalf("expected PASSION_OVERLOAD in logs, got: %+v", logs)
	}
	if !foundDetonation {
		t.Fatalf("expected GU_DETONATION in logs, got: %+v", logs)
	}
}
