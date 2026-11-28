package handlers

import (
	"context"
	"testing"
	"time"
)

func TestBuildBracketPairs(t *testing.T) {
	cases := []struct {
		n           int
		wantSlots   int
		wantByes    int
		wantNonByes int
	}{
		{n: 0, wantSlots: 0, wantByes: 0, wantNonByes: 0},
		{n: 1, wantSlots: 0, wantByes: 0, wantNonByes: 0},
		{n: 2, wantSlots: 1, wantByes: 0, wantNonByes: 1},
		{n: 3, wantSlots: 2, wantByes: 1, wantNonByes: 1},
		{n: 4, wantSlots: 2, wantByes: 0, wantNonByes: 2},
		{n: 5, wantSlots: 4, wantByes: 3, wantNonByes: 1},
		{n: 8, wantSlots: 4, wantByes: 0, wantNonByes: 4},
		{n: 9, wantSlots: 8, wantByes: 7, wantNonByes: 1},
	}

	for _, tc := range cases {
		slots := buildBracketPairs(tc.n)
		if len(slots) != tc.wantSlots {
			t.Fatalf("n=%d: expected %d slots, got %d", tc.n, tc.wantSlots, len(slots))
		}
		var byes, nonByes int
		seen := map[int]bool{}
		for _, s := range slots {
			if s.IsBye {
				byes++
			} else {
				nonByes++
				seen[s.SideBIdx] = true
			}
			seen[s.SideAIdx] = true
		}
		if byes != tc.wantByes || nonByes != tc.wantNonByes {
			t.Fatalf("n=%d: expected %d byes / %d matches, got %d / %d", tc.n, tc.wantByes, tc.wantNonByes, byes, nonByes)
		}
		if tc.n >= 2 && len(seen) != tc.n {
			t.Fatalf("n=%d: expected all %d participants placed, got %d", tc.n, tc.n, len(seen))
		}
	}
}

func TestArenaAdaptersConfiguredIndependently(t *testing.T) {
	da := deathArenaAdapter()
	ma := martialArenaAdapter()

	if da.RoundDuration <= 0 || ma.RoundDuration <= 0 {
		t.Fatalf("round durations must be positive: da=%v ma=%v", da.RoundDuration, ma.RoundDuration)
	}
	if da.RoundDuration == ma.RoundDuration {
		t.Fatalf("expected independent round durations for Death Arena (%v) and Martial Arena (%v)", da.RoundDuration, ma.RoundDuration)
	}
	for _, a := range []*tournamentAdapter{da, ma} {
		if a.ResolveMatch == nil || a.SettleRewards == nil || a.InvalidateCache == nil || a.BroadcastEvent == nil {
			t.Fatalf("adapter %s missing required callbacks", a.Name)
		}
		if a.RoundReward(1) <= 0 || a.RoundReward(8) <= 0 {
			t.Fatalf("adapter %s returned non-positive round reward", a.Name)
		}
	}
}

func TestRealArenaClockSleepCancelsOnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if ok := (realArenaClock{}).Sleep(ctx, 10*time.Second); ok {
		t.Fatal("expected Sleep to return false immediately when context is cancelled")
	}
}
