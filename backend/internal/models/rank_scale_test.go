package models

import "testing"

// After Phase 1 normalization Monster.csv base stats are rank-independent: the
// median base at every rank equals the rank-1 median (90 HP / 12 ATK / 5 DEF).
// Monster effective stats must therefore be exactly base * RankScale(rank).
func TestScaleMonsterStatsUsesSharedRankScale(t *testing.T) {
	for rank := 1; rank <= 9; rank++ {
		hp, atk, def := ScaleMonsterStats(1000, 100, 60, rank, 1, "Normal")
		s := RankScale(rank)
		if hp != 1000*s || atk != 100*s || def != 60*s {
			t.Fatalf("rank %d: got (%.3g, %.3g, %.3g), want RankScale-derived values", rank, hp, atk, def)
		}
	}
}

// A single standard Gu cast deals guRatio * RankScale(rank) flat damage. With
// rank-independent monster base its time-to-kill must stay in one band across
// all ranks — this fails loudly if base stats ever double-count the rank again.
func TestMonsterTTKBandAcrossRanks(t *testing.T) {
	const (
		baseHP  = 90.0 // rank-1 median after normalization
		baseDEF = 5.0
		guRatio = 20.0 // standard Gu flat damage / RankScale
	)
	for rank := 1; rank <= 9; rank++ {
		hp, _, def := ScaleMonsterStats(baseHP, 0, baseDEF, rank, 1, "Normal")
		net := guRatio*RankScale(rank) - def
		if net <= 0 {
			t.Fatalf("rank %d: standard Gu cannot pierce defense (net %.3g)", rank, net)
		}
		if ttk := hp / net; ttk < 4 || ttk > 10 {
			t.Errorf("rank %d: ttk %.2f out of band [4,10]", rank, ttk)
		}
	}
}

// Chaos/Hybrid tribulation spawns at rank 10. That rank is outside the table,
// but it must keep its historical 10^13 bump instead of clamping to rank 9.
func TestScaleMonsterStatsRank10NotClamped(t *testing.T) {
	hp9, _, _ := ScaleMonsterStats(100, 0, 0, 9, 1, "Normal")
	hp10, _, _ := ScaleMonsterStats(100, 0, 0, 10, 1, "Normal")
	if hp10 <= hp9 {
		t.Fatalf("rank 10 HP (%.3g) must exceed rank 9 (%.3g)", hp10, hp9)
	}
}
