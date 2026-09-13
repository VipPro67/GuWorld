package handlers

import (
	"math"
	"testing"
)

func TestMergePackLimit(t *testing.T) {
	cases := []struct {
		maxSoul int
		want    int
	}{
		{0, 1},
		{5, 1},
		{9, 1},
		{10, 1},
		{19, 1},
		{20, 2},
		{99, 9},
		{100, 10},
		{1000, 100},
	}
	for _, c := range cases {
		if got := mergePackLimit(c.maxSoul); got != c.want {
			t.Errorf("mergePackLimit(%d) = %d, want %d", c.maxSoul, got, c.want)
		}
	}
}

func TestRebaseStat(t *testing.T) {
	// A single individual keeps its stat as base.
	if got := rebaseStat(100, 1); got != 100 {
		t.Errorf("rebaseStat(100, 1) = %v, want 100", got)
	}

	// A pack of 4 with current stat = base * 4^0.65 rebases back to base.
	scale := math.Pow(4, packScaleExponent)
	base := 100.0
	current := base * scale
	got := rebaseStat(current, 4)
	if math.Abs(got-base) > 1.0 {
		t.Errorf("rebaseStat(%v, 4) = %v, want ~%v", current, got, base)
	}
}

func TestPackScaledStatSubLinear(t *testing.T) {
	base := 100.0
	// 2^0.65 < 2, so a merged pair is weaker than double the base.
	two := packScaledStat(base, 2)
	if two >= base*2 {
		t.Errorf("packScaledStat(100, 2) = %v, expected sub-linear (< 200)", two)
	}

	// Growth must flatten: scaling 2 -> 4 should not double the pack of 2.
	four := packScaledStat(base, 4)
	if four >= two*2 {
		t.Errorf("packScaledStat(100, 4) = %v, expected less than 2x pack of 2 (%v)", four, two)
	}
}

func TestNoCompoundOnRemerge(t *testing.T) {
	// Simulate: merge 2 into 2 (count 2 -> 4) using stored stat rebasing.
	// This must equal scaling a fresh base by 4^0.65 (no compound blow-up).
	base := 100.0
	pack2 := packScaledStat(base, 2)                 // what a pack of 2 stores
	pack4 := packScaledStat(rebaseStat(pack2, 2), 4) // remerge result
	want := packScaledStat(base, 4)
	if math.Abs(pack4-want) > 0.2 {
		t.Errorf("remerge gave %v, want %v (compound detected)", pack4, want)
	}
}
