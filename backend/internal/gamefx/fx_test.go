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
