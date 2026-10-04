package gamefx

import (
	"encoding/json"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestValidConstitutions(t *testing.T) {
	keys := ValidConstitutions()
	for _, key := range []string{
		"solar_sun_moon_body",
		"ancient_moon_yin_desolate_body",
		"northern_dark_ice_soul_body",
		"forest_sea_reincarnation_body",
		"myriad_gold_brilliant_flower_body",
		"blazing_glory_lightning_brilliance_body",
		"great_strength_true_martial_body",
		"carefree_wisdom_heart_body",
		"great_earth_origin_body",
		"universe_great_evolution_body",
		"pure_dream_reality_seeker_body",
	} {
		if !keys[key] {
			t.Errorf("expected valid constitution key %q", key)
		}
	}
}

func TestGetConstitutionEffects(t *testing.T) {
	p := &models.Player{}
	p.Constitution = "great_strength_true_martial_body"
	fx := Get(p)
	if got := fx.DaoNganBonus["strength"]; got != 0.50 {
		t.Errorf("strength dao bonus = %v, want 0.50", got)
	}
	if got := fx.StatMods["attack"]; got != 0.30 {
		t.Errorf("attack stat mod = %v, want 0.30", got)
	}
	if got := fx.StatMods["defense"]; got != 0.30 {
		t.Errorf("defense stat mod = %v, want 0.30", got)
	}
	if got := fx.MaxManaBonus; got != 0.50 {
		t.Errorf("max mana bonus = %v, want 0.50", got)
	}
	if got := fx.ManaRegenBonus; got != 1 {
		t.Errorf("mana regen bonus = %v, want 1", got)
	}
}

func TestGetConstitutionEffectsNoConstitution(t *testing.T) {
	fx := Get(&models.Player{})
	if fx.ManaRegenBonus != 0 {
		t.Errorf("expected no mana regen bonus without constitution, got %v", fx.ManaRegenBonus)
	}
	if len(fx.StatMods) != 0 {
		t.Errorf("expected no stat mods without constitution, got %v", fx.StatMods)
	}
}

func TestConstitutionDisplayName(t *testing.T) {
	if got := ConstitutionDisplayName("solar_sun_moon_body"); got != "Thái Nhật Dương Mãng Thể" {
		t.Errorf("display name = %q, want Thái Nhật Dương Mãng Thể", got)
	}
	if got := ConstitutionDisplayName("unknown_key"); got != "unknown_key" {
		t.Errorf("display name = %q, want raw key", got)
	}
}

func TestValidRaces(t *testing.T) {
	keys := ValidRaces()
	for _, key := range []string{
		"human",
		"fur_people",
		"merman",
		"feathered",
		"inkman",
		"rockman",
		"snowman",
		"variant_beastman",
		"dragonman",
		"giant_sun_descendant",
	} {
		if !keys[key] {
			t.Errorf("expected valid race key %q", key)
		}
	}
}

func TestGetRaceEffects(t *testing.T) {
	p := &models.Player{}
	p.Race = "merman"
	fx := Get(p)
	if got := fx.DaoNganBonus["water"]; got != 0.10 {
		t.Errorf("water dao bonus = %v, want 0.10", got)
	}
	if got := fx.PathDamageBonus["water"]; got != 0.05 {
		t.Errorf("water path damage = %v, want 0.05", got)
	}
}

func TestGetRaceEffectsNegative(t *testing.T) {
	p := &models.Player{}
	p.Race = "rockman"
	fx := Get(p)
	if got := fx.StatMods["defense"]; got != 0.15 {
		t.Errorf("defense stat mod = %v, want 0.15", got)
	}
	if got := fx.StatMods["speed"]; got != -0.10 {
		t.Errorf("speed stat mod = %v, want -0.10", got)
	}
}

func TestGetRaceEffectsNoRace(t *testing.T) {
	fx := Get(&models.Player{})
	if len(fx.StatMods) != 0 {
		t.Errorf("expected no stat mods without race, got %v", fx.StatMods)
	}
	if fx.RefineBonus != 0 {
		t.Errorf("expected no refine bonus without race, got %v", fx.RefineBonus)
	}
}

func TestRaceDisplayName(t *testing.T) {
	if got := RaceDisplayName("giant_sun_descendant"); got != "Cự Dương Huyết Mạch" {
		t.Errorf("display name = %q, want Cự Dương Huyết Mạch", got)
	}
	if got := RaceDisplayName("unknown_key"); got != "unknown_key" {
		t.Errorf("display name = %q, want raw key", got)
	}
}

func TestEffectsJSONRoundTrip(t *testing.T) {
	want := Effects{
		StatMods:                   map[string]float64{"attack": 0.10},
		DaoNganBonus:               map[string]float64{"fire": 0.20},
		ManaRegenBonus:             1,
		DaoMarkGainBonus:           0.15,
		MentalResistanceBonus:      0.20,
		PathResistanceBonus:        map[string]float64{"fire": 0.10},
		CultivationEfficiencyBonus: 0.20,
		SoulDefenseBonus:           0.15,
		SoulRegenBonus:             0.50,
		ApertureStressBonus:        1.0,
	}
	raw, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got Effects
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.DaoNganBonus["fire"] != 0.20 || got.ManaRegenBonus != 1 || got.StatMods["attack"] != 0.10 {
		t.Errorf("round trip mismatch: %+v", got)
	}
	if got.ApertureStressBonus != 1.0 {
		t.Errorf("aperture stress round trip = %v, want 1.0", got.ApertureStressBonus)
	}
	if got.HPRegenBonus != 0.0 {
		t.Errorf("hp regen bonus = %v, want 0.0", got.HPRegenBonus)
	}
	if got.SoulRegenBonus != 0.50 {
		t.Errorf("soul regen bonus = %v, want 0.50", got.SoulRegenBonus)
	}
}

func TestEffectsJSONHPRegen(t *testing.T) {
	var got Effects
	if err := json.Unmarshal([]byte(`{"hp_regen": 50, "healing_received": 0.5}`), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.HPRegenBonus != 50 {
		t.Errorf("hp_regen = %v, want 50", got.HPRegenBonus)
	}
	if got.HealingReceivedBonus != 0.5 {
		t.Errorf("healing_received = %v, want 0.5", got.HealingReceivedBonus)
	}
}

func TestEffectsJSONDaoMark(t *testing.T) {
	var got Effects
	if err := json.Unmarshal([]byte(`{"dao_mark": {"ice": 0.50}, "max_mana": 0.5, "mana_regen": 1}`), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.DaoNganBonus["ice"] != 0.50 {
		t.Errorf("dao_mark not decoded: %+v", got.DaoNganBonus)
	}
	if got.MaxManaBonus != 0.5 {
		t.Errorf("max mana = %v, want 0.5", got.MaxManaBonus)
	}
	if got.ManaRegenBonus != 1 {
		t.Errorf("mana regen = %v, want 1", got.ManaRegenBonus)
	}
}

func TestAddEffectsMerges(t *testing.T) {
	dst := Effects{StatMods: map[string]float64{"attack": 0.05}}
	addEffects(&dst, Effects{StatMods: map[string]float64{"attack": 0.10}, RefineBonus: 0.2, ApertureStressBonus: 0.5})
	if diff := abs(dst.StatMods["attack"] - 0.15); diff > 1e-9 {
		t.Errorf("stat merge = %v, want 0.15", dst.StatMods["attack"])
	}
	if diff := abs(dst.RefineBonus - 0.2); diff > 1e-9 {
		t.Errorf("refine merge = %v, want 0.2", dst.RefineBonus)
	}
	if diff := abs(dst.ApertureStressBonus - 0.5); diff > 1e-9 {
		t.Errorf("aperture stress merge = %v, want 0.5", dst.ApertureStressBonus)
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func TestCuongThiRaceEffects(t *testing.T) {
	p := &models.Player{}
	p.Race = "cuong_thi"
	fx := Get(p)
	if !fx.Immortal {
		t.Errorf("expected cuong_thi to be immortal")
	}
	if !fx.DisableHPRegen || !fx.DisableManaRegen || !fx.DisableResourceRegen || !fx.DisableSoulRegen {
		t.Errorf("expected cuong_thi to have all regen disabled: %+v", fx)
	}
}

func TestDeadPlayerBlockedAction(t *testing.T) {
	p := &models.Player{}
	p.CultivationState.Lifespan = 0
	if !IsDeadState(p) {
		t.Fatalf("expected lifespan 0 player to be dead")
	}
	if !HasBlockedAction(p, "explore") {
		t.Errorf("expected dead player to have explore action blocked")
	}
	if !HasBlockedAction(p, "cultivate") {
		t.Errorf("expected dead player to have cultivate action blocked")
	}

	alive := &models.Player{}
	alive.Lifespan = 12
	if IsDeadState(alive) {
		t.Errorf("expected player with lifespan to be alive")
	}
	if HasBlockedAction(alive, "explore") {
		t.Errorf("expected alive player to have explore action allowed")
	}
}

func TestImmortalRaceNeverDead(t *testing.T) {
	p := &models.Player{}
	p.Lifespan = 0
	p.Race = "cuong_thi"
	if IsDeadState(p) {
		t.Errorf("expected immortal race with zero lifespan to not be dead")
	}
	if HasBlockedAction(p, "cultivate") {
		t.Errorf("expected immortal race with zero lifespan to not be blocked")
	}
}

func TestIsDeadStateNilPlayer(t *testing.T) {
	if IsDeadState(nil) {
		t.Errorf("expected nil player to not be dead")
	}
}

func TestEffectsUnmarshalJSON_Blocks(t *testing.T) {
	raw := `[
		{"id":"STAT_MODS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"STAT_MODS","params":{"value":{"attack":0.1,"defense":0.1}}}]},
		{"id":"REFINE_SUCCESS","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"REFINE_SUCCESS","params":{"value":-0.1}}]},
		{"id":"IMMORTAL","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"IMMORTAL","params":{"value":true}}]}
	]`
	var eff Effects
	if err := json.Unmarshal([]byte(raw), &eff); err != nil {
		t.Fatalf("failed to unmarshal blocks: %v", err)
	}
	if eff.StatMods["attack"] != 0.1 || eff.StatMods["defense"] != 0.1 {
		t.Errorf("expected stat_mods attack/defense 0.1, got: %v", eff.StatMods)
	}
	if eff.RefineBonus != -0.1 {
		t.Errorf("expected refine -0.1, got: %v", eff.RefineBonus)
	}
	if !eff.Immortal {
		t.Errorf("expected immortal true, got false")
	}
}

func TestEffectsMarshalJSON_Blocks(t *testing.T) {
	eff := Effects{
		StatMods:       map[string]float64{"speed": 0.2},
		DaoNganBonus:   map[string]float64{"lightning": 0.5},
		ManaRegenBonus: 2,
		Immortal:       true,
	}
	data, err := json.Marshal(eff)
	if err != nil {
		t.Fatalf("failed to marshal effects: %v", err)
	}
	// Verify it's a JSON array
	if len(data) == 0 || data[0] != '[' {
		t.Fatalf("expected marshaled effects to be JSON array, got: %s", string(data))
	}

	var roundTripped Effects
	if err := json.Unmarshal(data, &roundTripped); err != nil {
		t.Fatalf("failed to unmarshal generated blocks: %v", err)
	}
	if roundTripped.StatMods["speed"] != 0.2 {
		t.Errorf("expected speed 0.2, got: %v", roundTripped.StatMods)
	}
	if roundTripped.DaoNganBonus["lightning"] != 0.5 {
		t.Errorf("expected lightning 0.5, got: %v", roundTripped.DaoNganBonus)
	}
	if roundTripped.ManaRegenBonus != 2 {
		t.Errorf("expected mana regen 2, got: %v", roundTripped.ManaRegenBonus)
	}
	if !roundTripped.Immortal {
		t.Errorf("expected immortal true, got false")
	}
}

func TestEffectsUnmarshalJSON_CanonicalStatMod(t *testing.T) {
	raw := `[
		{"id":"SPEED_BUFF","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"SPEED","value":0.15}]},
		{"id":"SPACE_DAO","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"DAO_MARK","path":"space","value":0.25}]},
		{"id":"MANA_HEAL","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"HEAL","sub_type":"MANA","value":3}]}
	]`
	var eff Effects
	if err := json.Unmarshal([]byte(raw), &eff); err != nil {
		t.Fatalf("failed to unmarshal canonical blocks: %v", err)
	}
	if eff.StatMods["speed"] != 0.15 {
		t.Errorf("expected speed 0.15, got: %v", eff.StatMods["speed"])
	}
	if eff.DaoNganBonus["space"] != 0.25 {
		t.Errorf("expected space dao 0.25, got: %v", eff.DaoNganBonus["space"])
	}
	if eff.ManaRegenBonus != 3 {
		t.Errorf("expected mana regen 3, got: %v", eff.ManaRegenBonus)
	}
}

func TestDefJSONMarshaling(t *testing.T) {
	c := ConstitutionDef{
		Tier: 3,
		Key:  "blazing_glory_lightning_brilliance_body",
		Name: "Viêm Hoàng Lôi Trạch Thể",
		Effects: Effects{
			StatMods:     map[string]float64{"speed": 0.2},
			DaoNganBonus: map[string]float64{"lightning": 0.5},
		},
	}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("failed to marshal ConstitutionDef: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	effectsList, ok := obj["effects"].([]any)
	if !ok {
		t.Fatalf("expected effects to be JSON array, got: %T (%+v)", obj["effects"], obj["effects"])
	}
	if len(effectsList) != 2 {
		t.Errorf("expected 2 effect blocks, got %d", len(effectsList))
	}
}
