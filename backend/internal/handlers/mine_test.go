package handlers

import (
	"math"
	"testing"
)

func TestMineChestItemID(t *testing.T) {
	cases := map[string]string{
		"immortal": "mine_chest_immortal",
		"mortal":   "mine_chest",
		"":         "mine_chest",
	}
	for mineType, want := range cases {
		if got := mineChestItemID(mineType); got != want {
			t.Errorf("mineChestItemID(%q) = %q, want %q", mineType, got, want)
		}
	}
}

func TestImmortalChestDecay(t *testing.T) {
	if got := immortalChestDecay(0); math.Abs(got-1.0) > 1e-9 {
		t.Errorf("decay(0) = %v, want 1.0", got)
	}
	if got := immortalChestDecay(120); math.Abs(got-0.5) > 1e-9 {
		t.Errorf("decay(120) = %v, want 0.5", got)
	}
	if got := immortalChestDecay(240); math.Abs(got-0.25) > 1e-9 {
		t.Errorf("decay(240) = %v, want 0.25", got)
	}
	// Sàn 20% base.
	if got := immortalChestDecay(10080); got != 0.2 {
		t.Errorf("decay(1 tuần) = %v, want floor 0.2", got)
	}
	// Đơn điệu không tăng.
	prev := math.Inf(1)
	for m := 0; m <= 2000; m += 7 {
		got := immortalChestDecay(m)
		if got > prev+1e-12 {
			t.Fatalf("decay không đơn điệu tại phút %d: %v > %v", m, got, prev)
		}
		prev = got
	}
}
