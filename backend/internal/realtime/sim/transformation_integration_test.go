package sim

import (
	"path/filepath"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/realtime/combat"
)

func init() {
	// Ensure TransformationForm.csv is loaded
	csvPath := filepath.Join("..", "..", "..", "data", "TransformationForm.csv")
	_ = LoadTransformationForms(csvPath)
}

func TestTransformation_FullMetamorphosisFlow(t *testing.T) {
	w, err := NewWorld("zone_test", 64)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	humanLoadout := [13]combat.Skill{
		{},
		{ID: "HUMAN_SKILL_1", Slot: 1, Name: "Nhân Tộc Chiêu 1", Mode: combat.ModeDirection, Range: 5, Damage: 50, ManaCost: 10},
		{ID: "HUMAN_SKILL_2", Slot: 2, Name: "Nhân Tộc Chiêu 2", Mode: combat.ModeDirection, Range: 5, Damage: 60, ManaCost: 10},
		{ID: "HUMAN_SKILL_3", Slot: 3, Name: "Nhân Tộc Chiêu 3", Mode: combat.ModeGround, Radius: 3, Damage: 70, ManaCost: 15},
		{ID: "HUMAN_SKILL_4", Slot: 4, Name: "Nhân Tộc Chiêu 4", Mode: combat.ModeSelf, Damage: 0, ManaCost: 20},
		{ID: "HUMAN_SKILL_5", Slot: 5, Name: "Nhân Tộc Chiêu 5", Mode: combat.ModeDirection, Range: 8, Damage: 120, ManaCost: 30},
		{ID: "HUMAN_SKILL_6", Slot: 6, Name: "Nhân Tộc Chiêu 6", Mode: combat.ModeSelf, Damage: 0, ManaCost: 50},
		{ID: "HUMAN_SKILL_7", Slot: 7, Name: "Nhân Tộc Chiêu 7", Mode: combat.ModeDirection, Range: 6, Damage: 80, ManaCost: 25},
	}

	player := Entity{
		ID:        "player_1",
		Kind:      EntityPlayer,
		Radius:    0.5,
		Speed:     4.0,
		Transform: Transform{Position: Vec2{X: 10, Y: 10}},
		Vitals:    Vitals{HP: 1000, MaxHP: 1000},
		Combat: CombatState{
			Mana:    500,
			MaxMana: 500,
			Skills:  humanLoadout,
		},
	}

	if err := w.AddEntity(player); err != nil {
		t.Fatalf("AddEntity failed: %v", err)
	}

	p := w.entities["player_1"]

	// 1. Initiate Full Transformation into Thái Cổ Viêm Hùng
	req := FormActivationRequest{
		FormID:              "FORM_FLAME_BEAR",
		FormName:            "Thái Cổ Viêm Hùng",
		FormKind:            TransformationFull,
		ModelScale:          1.5,
		CollisionRadiusMult: 1.4,
		MaxFormHP:           150,
		FormDefenseBonus:    20,
		DamageReduction:     0.15,
		ManaUpkeepPerSecond: 10,
		InitialManaCost:     20,
	}

	ok := w.StartTransformation(p, req)
	if !ok {
		t.Fatalf("StartTransformation failed")
	}

	if !p.Transformation.Active {
		t.Fatalf("expected Transformation.Active to be true")
	}
	if p.Transformation.FormKind != TransformationFull {
		t.Fatalf("expected FormKind FULL, got %s", p.Transformation.FormKind)
	}
	expectedRadius := 0.5 * 1.4
	if mathAbs(p.Radius-expectedRadius) > 0.001 {
		t.Fatalf("expected radius %f, got %f", expectedRadius, p.Radius)
	}
	if p.Transformation.CurrentFormHP != 150 {
		t.Fatalf("expected FormHP 150, got %f", p.Transformation.CurrentFormHP)
	}

	// 2. Verify Skill Bar Swap 1..6
	if p.Combat.Skills[1].ID != "BEAR_CLAW_SWIPE" {
		t.Fatalf("expected slot 1 BEAR_CLAW_SWIPE, got %s", p.Combat.Skills[1].ID)
	}
	if p.Combat.Skills[2].ID != "BEAR_FERAL_CHARGE" {
		t.Fatalf("expected slot 2 BEAR_FERAL_CHARGE, got %s", p.Combat.Skills[2].ID)
	}
	if p.Combat.Skills[6].ID != "REVERT_METAMORPHOSIS" {
		t.Fatalf("expected slot 6 REVERT_METAMORPHOSIS, got %s", p.Combat.Skills[6].ID)
	}

	// 3. Test form skill lock on slot > 6
	w.startCastLocked(p, CastCommand{
		EntityID: "player_1",
		Slot:     7,
		Mode:     combat.ModeDirection,
		Aim:      Vec2{X: 1, Y: 0},
		Seq:      1,
	})
	events := w.events
	foundReject := false
	for _, ev := range events {
		if ev.Type == combat.EventCastRejected && ev.Reason == "form_skill_locked" {
			foundReject = true
			break
		}
	}
	if !foundReject {
		t.Fatalf("expected form_skill_locked event for slot 7 in beast form")
	}

	// 4. Test Revert Transformation (using Slot 6)
	w.events = nil
	w.startCastLocked(p, CastCommand{
		EntityID: "player_1",
		Slot:     6,
		Mode:     combat.ModeSelf,
		Seq:      2,
	})

	if p.Transformation.Active {
		t.Fatalf("expected Transformation.Active to be false after revert")
	}
	if mathAbs(p.Radius-0.5) > 0.001 {
		t.Fatalf("expected original radius 0.5 restored, got %f", p.Radius)
	}
	if p.Combat.Skills[1].ID != "HUMAN_SKILL_1" {
		t.Fatalf("expected human skill 1 restored, got %s", p.Combat.Skills[1].ID)
	}
	if p.Combat.Skills[6].ID != "HUMAN_SKILL_6" {
		t.Fatalf("expected human skill 6 restored, got %s", p.Combat.Skills[6].ID)
	}
}

func TestTransformation_UpkeepDrainAndExhaustion(t *testing.T) {
	w, err := NewWorld("zone_test_upkeep", 64)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := Entity{
		ID:        "player_upkeep",
		Kind:      EntityPlayer,
		Radius:    0.5,
		Transform: Transform{Position: Vec2{X: 10, Y: 10}},
		Vitals:    Vitals{HP: 1000, MaxHP: 1000},
		Combat: CombatState{
			Mana:    2.0, // Only enough for 2 ticks of upkeep (1.0 mana/tick = 20 mana/sec at 20Hz)
			MaxMana: 100,
		},
	}
	_ = w.AddEntity(player)
	p := w.entities["player_upkeep"]

	req := FormActivationRequest{
		FormID:              "FORM_FLAME_BEAR",
		FormName:            "Thái Cổ Viêm Hùng",
		FormKind:            TransformationFull,
		ManaUpkeepPerSecond: 20.0, // 1.0 mana per tick at 20Hz
		InitialManaCost:     0,
	}
	w.StartTransformation(p, req)

	// Step 1: Mana drains from 2.0 to 1.0
	w.Step()
	if mathAbs(p.Combat.Mana-1.0) > 0.001 {
		t.Fatalf("expected mana 1.0 after tick 1, got %f", p.Combat.Mana)
	}
	if !p.Transformation.Active {
		t.Fatalf("expected form to remain active after tick 1")
	}

	// Step 2: Mana drains from 1.0 to 0.0
	w.Step()
	if mathAbs(p.Combat.Mana-0.0) > 0.001 {
		t.Fatalf("expected mana 0.0 after tick 2, got %f", p.Combat.Mana)
	}
	if !p.Transformation.Active {
		t.Fatalf("expected form to remain active after tick 2")
	}

	// Step 3: Out of mana on tick 3 -> Force revert with Exhaustion
	w.Step()
	if p.Transformation.Active {
		t.Fatalf("expected form to be deactivated due to mana depletion")
	}

	// Verify Exhaustion debuff
	hasExhaustion := w.hasStatus(p, "EXHAUSTION")
	if !hasExhaustion {
		t.Fatalf("expected EXHAUSTION debuff applied after form cancellation from mana depletion")
	}
}

func TestTransformation_FormHPLayerAndDamageSpillover(t *testing.T) {
	w, err := NewWorld("zone_test_damage", 64)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := Entity{
		ID:        "player_hp",
		Kind:      EntityPlayer,
		Radius:    0.5,
		Transform: Transform{Position: Vec2{X: 10, Y: 10}},
		Vitals:    Vitals{HP: 1000, MaxHP: 1000},
		Combat: CombatState{
			Mana:    100,
			MaxMana: 100,
		},
	}
	_ = w.AddEntity(player)
	p := w.entities["player_hp"]

	req := FormActivationRequest{
		FormID:          "FORM_FLAME_BEAR",
		FormName:        "Thái Cổ Viêm Hùng",
		FormKind:        TransformationFull,
		MaxFormHP:       100,
		DamageReduction: 0.20, // 20% damage reduction
	}
	w.StartTransformation(p, req)

	// Hit 1: 50 incoming damage. 20% reduction -> 40 effective damage.
	// Absorbed completely by Form HP (100 -> 60). Real HP untouched.
	w.applyDamageLocked(nil, p, 50, "ENEMY_HIT_1")
	if mathAbs(p.Transformation.CurrentFormHP-60) > 0.001 {
		t.Fatalf("expected FormHP 60, got %f", p.Transformation.CurrentFormHP)
	}
	if p.Vitals.HP != 1000 {
		t.Fatalf("expected real HP 1000 unchanged, got %f", p.Vitals.HP)
	}

	// Hit 2: 100 incoming damage. 20% reduction -> 80 effective damage.
	// Form HP has 60 -> absorbed 60, Form Broken! Spillover is 80 - 60 = 20 damage.
	// Real HP should be 1000 - 20 = 980.
	w.applyDamageLocked(nil, p, 100, "ENEMY_HIT_2")
	if p.Transformation.Active {
		t.Fatalf("expected form to be broken and deactivated")
	}
	if mathAbs(p.Vitals.HP-980) > 0.001 {
		t.Fatalf("expected real HP 980 after spillover, got %f", p.Vitals.HP)
	}
	if !w.hasStatus(p, "EXHAUSTION") {
		t.Fatalf("expected EXHAUSTION debuff applied after Form HP broken")
	}
}

func TestTransformation_WindupInterruptedByHardCC(t *testing.T) {
	w, err := NewWorld("zone_test_cc", 64)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := Entity{
		ID:        "player_windup",
		Kind:      EntityPlayer,
		Radius:    0.5,
		Transform: Transform{Position: Vec2{X: 10, Y: 10}},
		Vitals:    Vitals{HP: 1000, MaxHP: 1000},
		Combat: CombatState{
			Mana:    100,
			MaxMana: 100,
		},
	}
	_ = w.AddEntity(player)
	p := w.entities["player_windup"]

	req := FormActivationRequest{
		FormID:      "FORM_FLAME_BEAR",
		FormName:    "Thái Cổ Viêm Hùng",
		FormKind:    TransformationFull,
		WindupTicks: 10,
	}
	w.StartTransformation(p, req)

	if !p.Transformation.IsWindup {
		t.Fatalf("expected IsWindup to be true")
	}
	if p.Transformation.Active {
		t.Fatalf("expected Active to be false during windup")
	}

	// Interrupt with Stun hard CC
	w.applyStatusLocked(nil, p, combat.Skill{
		StatusID:            "STUN",
		StatusDurationTicks: 20,
		StatusDebuff:        true,
	})

	if p.Transformation.IsWindup {
		t.Fatalf("expected IsWindup to be false after hard CC interrupt")
	}
	if p.Transformation.Active {
		t.Fatalf("expected form not to activate after interrupt")
	}
}

func TestTransformation_SnapshotStateMapping(t *testing.T) {
	w, err := NewWorld("zone_test_snap", 64)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	player := Entity{
		ID:        "player_snap",
		Kind:      EntityPlayer,
		Radius:    0.5,
		Transform: Transform{Position: Vec2{X: 10, Y: 10}},
		Vitals:    Vitals{HP: 1000, MaxHP: 1000},
		Combat: CombatState{
			Mana:    100,
			MaxMana: 100,
		},
	}
	_ = w.AddEntity(player)
	p := w.entities["player_snap"]

	req := FormActivationRequest{
		FormID:              "FORM_FLAME_BEAR",
		FormName:            "Thái Cổ Viêm Hùng",
		FormKind:            TransformationFull,
		ModelScale:          1.6,
		CollisionRadiusMult: 1.4,
		MaxFormHP:           180,
		FormDefenseBonus:    25,
		DamageReduction:     0.20,
		ManaUpkeepPerSecond: 10,
	}
	w.StartTransformation(p, req)

	snap := w.Step()
	found := false
	for _, ent := range snap.Entities {
		if ent.ID == "player_snap" {
			found = true
			if mathAbs(ent.Radius-0.7) > 0.001 {
				t.Fatalf("expected snapshot radius 0.7, got %f", ent.Radius)
			}
			if !ent.Transformation.Active {
				t.Fatalf("expected snapshot transformation active")
			}
			if ent.Transformation.ModelScale != 1.6 {
				t.Fatalf("expected model scale 1.6, got %f", ent.Transformation.ModelScale)
			}
			if ent.Transformation.MaxFormHP != 180 {
				t.Fatalf("expected max form hp 180, got %f", ent.Transformation.MaxFormHP)
			}
			break
		}
	}
	if !found {
		t.Fatalf("entity player_snap not found in snapshot")
	}
}

func mathAbs(val float64) float64 {
	if val < 0 {
		return -val
	}
	return val
}
