package models

import "testing"

// After Phase 1 normalization Monster.csv base stats are rank-independent: the
// median base at every rank equals the rank-1 median (90 HP / 12 ATK / 5 DEF).
// Monster effective stats must therefore be exactly base * RankScale(rank).
func TestScaleMonsterStatsUsesSharedRankScale(t *testing.T) {
	t.Setenv("MONSTER_RANK_SCALING_ENABLED", "true")
	t.Setenv("MONSTER_STAT_MULTIPLIER", "1")
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
	t.Setenv("MONSTER_RANK_SCALING_ENABLED", "true")
	t.Setenv("MONSTER_STAT_MULTIPLIER", "1")
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
	t.Setenv("MONSTER_RANK_SCALING_ENABLED", "true")
	t.Setenv("MONSTER_STAT_MULTIPLIER", "1")
	hp9, _, _ := ScaleMonsterStats(100, 0, 0, 9, 1, "Normal")
	hp10, _, _ := ScaleMonsterStats(100, 0, 0, 10, 1, "Normal")
	if hp10 <= hp9 {
		t.Fatalf("rank 10 HP (%.3g) must exceed rank 9 (%.3g)", hp10, hp9)
	}
}

// MONSTER_RANK_SCALING_ENABLED=false must drop the RankScale factor (including
// the rank-10 bump) while keeping the type and sub-rank multipliers.
func TestScaleMonsterStatsRankScalingDisabled(t *testing.T) {
	t.Setenv("MONSTER_RANK_SCALING_ENABLED", "false")
	t.Setenv("MONSTER_STAT_MULTIPLIER", "1")
	const (
		typeMult    = 3.0 // Boss
		subRankMult = 1.2 // subRank 2 -> 1 + 0.20
		wantBaseHP  = 1000.0
		wantBaseAtk = 100.0
		wantBaseDef = 60.0
	)
	for _, rank := range []int{1, 3, 9, 10} {
		hp, atk, def := ScaleMonsterStats(wantBaseHP, wantBaseAtk, wantBaseDef, rank, 2, "Boss")
		wantHP := wantBaseHP * typeMult * subRankMult
		wantAtk := wantBaseAtk * typeMult * subRankMult
		wantDef := wantBaseDef * typeMult * subRankMult
		if hp != wantHP || atk != wantAtk || def != wantDef {
			t.Fatalf("rank %d: got (%.3g, %.3g, %.3g), want (%.3g, %.3g, %.3g)",
				rank, hp, atk, def, wantHP, wantAtk, wantDef)
		}
	}
}

// MONSTER_STAT_MULTIPLIER is applied last, on top of rank/type/sub-rank scaling.
func TestScaleMonsterStatsGlobalMultiplier(t *testing.T) {
	t.Setenv("MONSTER_RANK_SCALING_ENABLED", "true")
	t.Setenv("MONSTER_STAT_MULTIPLIER", "0.5")
	rank := 4
	s := RankScale(rank) * 0.5
	hp, atk, def := ScaleMonsterStats(1000, 100, 60, rank, 1, "Normal")
	if hp != 1000*s || atk != 100*s || def != 60*s {
		t.Fatalf("got (%.3g, %.3g, %.3g), want multiplier-scaled values", hp, atk, def)
	}
}

func TestMonsterStatMultiplierFallback(t *testing.T) {
	for _, raw := range []string{"", "abc", "0", "-0.5"} {
		t.Setenv("MONSTER_STAT_MULTIPLIER", raw)
		if got := MonsterStatMultiplier(); got != 1.0 {
			t.Fatalf("MONSTER_STAT_MULTIPLIER=%q: got %v, want 1.0", raw, got)
		}
	}
	t.Setenv("MONSTER_STAT_MULTIPLIER", "0.35")
	if got := MonsterStatMultiplier(); got != 0.35 {
		t.Fatalf("got %v, want 0.35", got)
	}
}
