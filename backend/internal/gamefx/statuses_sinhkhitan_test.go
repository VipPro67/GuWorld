package gamefx

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestSinhKhiTanRegistry(t *testing.T) {
	if !ValidStatuses()[StatusSinhKhiTan] {
		t.Fatal("sinh_khi_tan missing from fallback registry")
	}
	if got := StatusDefaultDuration(StatusSinhKhiTan); got != SinhKhiTanDurationMinutes {
		t.Fatalf("default duration = %d, want %d", got, SinhKhiTanDurationMinutes)
	}
}

func TestSinhKhiTanTiers(t *testing.T) {
	want := map[int]float64{1: -0.10, 2: -0.25, 3: -0.35, 4: -0.50, 5: -0.99}
	for tier, value := range want {
		if got := SinhKhiTanValueForTier(tier); got != value {
			t.Fatalf("tier %d value = %v, want %v", tier, got, value)
		}
		if got := SinhKhiTanTierFromValue(value); got != tier {
			t.Fatalf("value %v tier = %d, want %d", value, got, tier)
		}
	}
	// Clamp + empty.
	if got := SinhKhiTanTierFromValue(0); got != 0 {
		t.Fatalf("zero value tier = %d, want 0", got)
	}
	if got := SinhKhiTanValueForTier(99); got != -0.99 {
		t.Fatalf("overflow tier value = %v, want -0.99", got)
	}
	if SinhKhiTanTierName(5) != "Sinh Khí Tàn V" {
		t.Fatalf("tier 5 name = %q", SinhKhiTanTierName(5))
	}
}

func TestSinhKhiTanEffectsPerTier(t *testing.T) {
	def, ok := effectiveStatuses()[StatusSinhKhiTan]
	if !ok {
		t.Fatal("sinh_khi_tan missing from registry")
	}
	for tier := 1; tier <= 5; tier++ {
		eff := def.EffectsForValue(SinhKhiTanValueForTier(tier))
		for _, key := range []string{"attack", "defense", "speed"} {
			if eff.StatMods[key] != SinhKhiTanTierValues[tier] {
				t.Fatalf("tier %d stat_mods[%s] = %v, want %v",
					tier, key, eff.StatMods[key], SinhKhiTanTierValues[tier])
			}
		}
	}
}

func TestSinhKhiTanResolvesThroughGet(t *testing.T) {
	p := &models.Player{}
	p.Race = "human"
	p.Statuses = models.StatusList{{ID: StatusSinhKhiTan, Value: -0.50}}
	fx := Get(p)
	if fx.StatMods["attack"] != -0.50 || fx.StatMods["defense"] != -0.50 || fx.StatMods["speed"] != -0.50 {
		t.Fatalf("tier IV mods missing: %+v", fx.StatMods)
	}
}
