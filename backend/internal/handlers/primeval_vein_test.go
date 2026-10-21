package handlers

import (
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestVeinAccrualDailyAndLifetimeCaps(t *testing.T) {
	now := time.Now()
	base := veinRow{ID: "vein-1", RegionID: "r", X: 1, Y: 2, VeinType: "mortal", Rarity: "Thường", AccumulatedSince: now.Add(-72 * time.Hour)}

	minutes, stones, _, _ := veinAccrual(base, now)
	if minutes != veinDailyCapMinutes {
		t.Fatalf("daily cap = %d, want %d", minutes, veinDailyCapMinutes)
	}
	if stones != int64(veinDailyCapMinutes)*int64(veinRarityRates["Thường"].StonesPerMin) {
		t.Fatalf("stones = %d, not derived from capped minutes", stones)
	}

	// Nearly depleted vein: only the remaining lifetime output is claimable.
	capMinutes := veinCapMinutes("Thường")
	nearEnd := base
	nearEnd.ProducedMinutes = capMinutes - 10
	minutes, _, _, _ = veinAccrual(nearEnd, now)
	if minutes != 10 {
		t.Fatalf("remaining production = %d, want 10", minutes)
	}

	// Fully depleted.
	nearEnd.ProducedMinutes = capMinutes
	minutes, _, _, _ = veinAccrual(nearEnd, now)
	if minutes != 0 {
		t.Fatalf("depleted vein accrued %d minutes, want 0", minutes)
	}
}

func TestVeinAccrualDeterministicChests(t *testing.T) {
	now := time.Now()
	v := veinRow{ID: "vein-2", RegionID: "r", X: 3, Y: 4, VeinType: "mortal", Rarity: "Siêu Cấp", AccumulatedSince: now.Add(-24 * time.Hour)}
	_, _, _, chestsA := veinAccrual(v, now)
	_, _, _, chestsB := veinAccrual(v, now.Add(2*time.Second))
	if chestsA != chestsB {
		t.Fatalf("chest roll not stable within the same minute: %d vs %d", chestsA, chestsB)
	}
	if chestsA < 0 || chestsA > veinDailyCapMinutes {
		t.Fatalf("chest count out of range: %d", chestsA)
	}
}

func TestVeinRarityCapsIncreaseWithRarity(t *testing.T) {
	prev := 0
	for _, rarity := range []string{"Thường", "Hiếm", "Sử Thi", "Truyền Thuyết", "Siêu Cấp"} {
		capMinutes := veinCapMinutes(rarity)
		if capMinutes <= prev {
			t.Fatalf("cap for %s (%d) is not larger than previous (%d)", rarity, capMinutes, prev)
		}
		prev = capMinutes
	}
}

func TestSpringEggHatchRegistered(t *testing.T) {
	if ApplyItemEffects["spring_egg_hatch"] == nil {
		t.Fatal("spring_egg_hatch is not registered in ApplyItemEffects")
	}
	if _, err := models.CompileItemEffects(models.ItemEffects{"spring_egg_hatch": 0}); err != nil {
		t.Fatalf("CompileItemEffects rejects spring_egg_hatch: %v", err)
	}
}

func TestWithinVeinHarvestRange(t *testing.T) {
	cases := []struct {
		px, py, vx, vy int
		want           bool
	}{
		{100, 100, 100, 100, true},
		{103, 97, 100, 100, true},
		{104, 100, 100, 100, false},
		{100, 96, 100, 100, false},
		{97, 103, 100, 100, true},
	}
	for _, tc := range cases {
		if got := withinVeinHarvestRange(tc.px, tc.py, tc.vx, tc.vy); got != tc.want {
			t.Fatalf("withinVeinHarvestRange(%d,%d,%d,%d) = %v, want %v", tc.px, tc.py, tc.vx, tc.vy, got, tc.want)
		}
	}
}
