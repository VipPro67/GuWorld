package handlers

import "testing"

func TestTribulationBossTemplateSelection(t *testing.T) {
	cases := map[string]string{
		"EARTH_CALAMITY":       "monster_earth_calamity_boss",
		"HEAVENLY_TRIBULATION": "monster_heaven_tribulation_boss",
		"GRAND_TRIBULATION":    "monster_grand_calamity_boss",
		"MYRIAD_TRIBULATION":   "monster_myriad_tribulation_boss",
		"CHAOS_CALAMITY":       "monster_chaos_calamity_boss",
		"CHAOTIC_TRIBULATION":  "monster_chaos_calamity_boss",
		"UNKNOWN":              "monster_earth_calamity_boss",
	}
	for tribulationType, want := range cases {
		if got := tribulationBossTemplateID(tribulationType); got != want {
			t.Errorf("tribulationBossTemplateID(%q) = %q, want %q", tribulationType, got, want)
		}
	}
}

func TestTribulationNodeDamage(t *testing.T) {
	cases := map[string]float64{
		"EARTH_CALAMITY":       1,
		"HEAVENLY_TRIBULATION": 2,
		"GRAND_TRIBULATION":    4,
		"MYRIAD_TRIBULATION":   8,
		"CHAOS_CALAMITY":       16,
		"CHAOTIC_TRIBULATION":  24,
	}
	for tribulationType, want := range cases {
		if got := tribulationNodeDamage(tribulationType); got != want {
			t.Errorf("tribulationNodeDamage(%q) = %v, want %v", tribulationType, got, want)
		}
	}
}

func TestClampApertureCell(t *testing.T) {
	cases := map[int]float64{-1: 2, 0: 2, 1: 2, 2: 2, 10: 10, 17: 17, 18: 17, 19: 17, 99: 17}
	for in, want := range cases {
		if got := clampApertureCell(in); got != want {
			t.Errorf("clampApertureCell(%d) = %v, want %v", in, got, want)
		}
	}
}
