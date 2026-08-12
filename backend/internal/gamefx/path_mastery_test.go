package gamefx

import "testing"

func TestPathMasteryLevel(t *testing.T) {
	cases := []struct {
		points, want int
	}{
		{0, 0},
		{50, 0},
		{99, 0},
		{100, 1},
		{499, 1},
		{500, 2},
		{2499, 2},
		{2500, 3},
		{9999, 3},
		{10000, 4},
		{100000, 4},
	}
	for _, c := range cases {
		if got := PathMasteryLevel(c.points); got != c.want {
			t.Errorf("PathMasteryLevel(%d) = %d, want %d", c.points, got, c.want)
		}
	}
}

func TestPathMasteryLevelName(t *testing.T) {
	if got := PathMasteryLevelName(0); got != "Ordinary" {
		t.Errorf("level 0 name = %q", got)
	}
	if got := PathMasteryLevelName(2); got != "Grandmaster" {
		t.Errorf("level 2 name = %q", got)
	}
	if got := PathMasteryLevelName(4); got != "Supreme Grandmaster" {
		t.Errorf("level 4 name = %q", got)
	}
	if got := PathMasteryLevelName(99); got != "Ordinary" {
		t.Errorf("out-of-range name = %q", got)
	}
}

func TestPathMasteryLevelMultiplier(t *testing.T) {
	want := [5]float64{1.0, 1.1, 1.25, 1.4, 1.5}
	for lvl := 0; lvl <= 4; lvl++ {
		if got := PathMasteryLevelMultiplier(lvl); got != want[lvl] {
			t.Errorf("multiplier level %d = %v, want %v", lvl, got, want[lvl])
		}
	}
	if got := PathMasteryLevelMultiplier(7); got != 1.0 {
		t.Errorf("out-of-range multiplier = %v", got)
	}
}

func TestPathMasteryMultiplier(t *testing.T) {
	pm := map[string]int{"fire": 600, "water": 200}
	// fire=600 -> Grandmaster (level 2) -> 1.25
	if got := PathMasteryMultiplier(pm, []string{"Fire Path", "Water Path"}); got != 1.25 {
		t.Errorf("multiplier = %v, want 1.25", got)
	}
	// water=200 -> Master (1.1)
	if got := PathMasteryMultiplier(pm, []string{"water"}); got != 1.1 {
		t.Errorf("water multiplier = %v, want 1.1", got)
	}
	// unknown path -> 1.0
	if got := PathMasteryMultiplier(pm, []string{"dream"}); got != 1.0 {
		t.Errorf("unknown path multiplier = %v, want 1.0", got)
	}
	if got := PathMasteryMultiplier(nil, []string{"fire"}); got != 1.0 {
		t.Errorf("nil map multiplier = %v, want 1.0", got)
	}
}

func TestPathMasteryLevelRequiredForRank(t *testing.T) {
	cases := []struct{ rank, want int }{
		{1, 0}, {6, 0}, {7, 2}, {8, 3}, {9, 4},
	}
	for _, c := range cases {
		if got := PathMasteryLevelRequiredForRank(c.rank); got != c.want {
			t.Errorf("PathMasteryLevelRequiredForRank(%d) = %d, want %d", c.rank, got, c.want)
		}
	}
}

func TestHasPathMasteryAtLeast(t *testing.T) {
	pm := map[string]int{"fire": 600, "water": 40}
	if !HasPathMasteryAtLeast(pm, 2) {
		t.Errorf("fire=600 should reach Grandmaster (level 2)")
	}
	if HasPathMasteryAtLeast(pm, 3) {
		t.Errorf("no path reaches level 3")
	}
	if !HasPathMasteryAtLeast(pm, 0) {
		t.Errorf("minLevel 0 should always pass")
	}
	if HasPathMasteryAtLeast(nil, 1) {
		t.Errorf("nil map should fail for minLevel 1")
	}
	if !HasPathMasteryAtLeast(map[string]int{"dream": 10000}, 4) {
		t.Errorf("dream=10000 should reach Supreme (level 4)")
	}
}

func TestPathMasteryPointsForLevel(t *testing.T) {
	cases := []struct{ level, want int }{
		{0, 0}, {1, 100}, {2, 500}, {3, 2500}, {4, 10000},
	}
	for _, c := range cases {
		if got := PathMasteryPointsForLevel(c.level); got != c.want {
			t.Errorf("PathMasteryPointsForLevel(%d) = %d, want %d", c.level, got, c.want)
		}
	}
}
