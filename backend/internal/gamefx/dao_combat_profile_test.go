package gamefx

import (
	"testing"
)

func TestGetCompatibility(t *testing.T) {
	tests := []struct {
		a, b     string
		expected float64
	}{
		{"fire", "fire", 1.0},
		{"fire", "water", 0.55},
		{"water", "fire", 0.55},
		{"Fire Path", "Water Path", 0.55},
		{"space", "time", 0.97},
		{"wisdom", "information", 0.98},
		{"wood", "fire", 0.95},
		{"fire", "lightning", 0.88}, // Same Elemental group
		{"fire", "space", 0.90},     // Elemental vs SpaceTime
		{"dark", "light", 0.55},     // Pair override
	}

	for _, tt := range tests {
		got := GetCompatibility(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("GetCompatibility(%s, %s) = %f; want %f", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestDaoAmplification(t *testing.T) {
	tests := []struct {
		marks int64
		want  float64
	}{
		{0, 1.0},
		{-10, 1.0},
		{100, 2.0},
		{1000, 11.0},
		{10000, 101.0},
		{50000, 501.0},
		{100000, 1001.0},
		{500000, 5001.0},
	}

	for _, tt := range tests {
		got := DaoAmplification(tt.marks)
		if got != tt.want {
			t.Errorf("DaoAmplification(%d) = %f; want %f", tt.marks, got, tt.want)
		}
	}
}

func TestBuildDaoCombatProfile_PureSpecialist(t *testing.T) {
	daoMarks := map[string]int64{
		"fire": 50000,
	}
	pathMasteries := map[string]int{
		"fire": 1000000, // Great Grandmaster (Level 3)
	}

	profile := BuildDaoCombatProfile(daoMarks, pathMasteries, false, nil)
	if profile.DominantPath != "fire" {
		t.Errorf("expected dominant path fire, got %s", profile.DominantPath)
	}
	if profile.TotalDaoMarks != 50000 {
		t.Errorf("expected total marks 50000, got %d", profile.TotalDaoMarks)
	}

	daoAmp, realmEff, conflictMult := profile.GetOffensiveFactors("fire")
	if daoAmp != 501.0 {
		t.Errorf("expected daoAmp 501.0, got %f", daoAmp)
	}
	if realmEff != 1.25 {
		t.Errorf("expected realmEff 1.25, got %f", realmEff)
	}
	if conflictMult != 1.0 {
		t.Errorf("expected pure specialist conflictMult 1.0, got %f", conflictMult)
	}

	// When attacked by water, natural resistance should trigger
	natRes := profile.GetDefensiveResistance("water")
	// defConflict = 50000 * (1 - 0.55) = 22500
	// natRes = 0.30 * (22500 / 72500) ≈ 0.0931 (9.31%)
	if natRes < 0.09 || natRes > 0.10 {
		t.Errorf("expected natural resistance around 0.093, got %f", natRes)
	}
}

func TestBuildDaoCombatProfile_DualPathWithMastery(t *testing.T) {
	daoMarks := map[string]int64{
		"fire":  50000,
		"water": 50000,
	}

	// 1. Level 0 (Ordinary)
	profileL0 := BuildDaoCombatProfile(daoMarks, nil, false, nil)
	_, _, conflictL0 := profileL0.GetOffensiveFactors("fire")
	if conflictL0 >= 0.50 || conflictL0 <= 0.44 {
		t.Errorf("expected conflictL0 around 0.471, got %f", conflictL0)
	}

	// 2. Level 3 (Great Grandmaster)
	masteriesL3 := map[string]int{
		"fire":  1000000,
		"water": 1000000,
	}
	profileL3 := BuildDaoCombatProfile(daoMarks, masteriesL3, false, nil)
	_, _, conflictL3 := profileL3.GetOffensiveFactors("fire")

	// Mastery should reduce effective conflict -> higher multiplier
	if conflictL3 <= conflictL0 {
		t.Errorf("expected higher conflict multiplier with mastery, got L3=%f <= L0=%f", conflictL3, conflictL0)
	}
	if conflictL3 < 0.59 || conflictL3 > 0.65 {
		t.Errorf("expected conflictL3 around 0.618, got %f", conflictL3)
	}
}

func TestBuildDaoCombatProfile_GraceThreshold(t *testing.T) {
	// Main fire 100k, random water 500 marks (below grace threshold 2000)
	daoMarks := map[string]int64{
		"fire":  100000,
		"water": 500,
	}
	profile := BuildDaoCombatProfile(daoMarks, nil, false, nil)
	_, _, conflict := profile.GetOffensiveFactors("fire")
	if conflict != 1.0 {
		t.Errorf("expected 0 conflict for marks within grace threshold, got %f", conflict)
	}
}

func TestBuildDaoCombatProfile_NonConflictingTrait(t *testing.T) {
	daoMarks := map[string]int64{
		"fire":  50000,
		"water": 50000,
	}
	profile := BuildDaoCombatProfile(daoMarks, nil, true, nil) // hasNonConflictingDao = true

	// Offense: No conflict penalty
	_, _, conflict := profile.GetOffensiveFactors("fire")
	if conflict != 1.0 {
		t.Errorf("expected non-conflicting trait to have 1.0 offense conflict mult, got %f", conflict)
	}

	// Defense: 0 natural resistance
	natRes := profile.GetDefensiveResistance("water")
	if natRes != 0.0 {
		t.Errorf("expected non-conflicting trait to have 0 natural resistance, got %f", natRes)
	}
}

func TestBuildDaoCombatProfile_KillPath(t *testing.T) {
	// Test backward compat: "kill" key in daoMarks/pathMasteries is normalized
	daoMarks := map[string]int64{
		"kill": 50000,
	}
	masteries := map[string]int{
		"killing": 1000000,
	}

	profile := BuildDaoCombatProfile(daoMarks, masteries, false, nil)
	if profile.DominantPath != "killing" {
		t.Errorf("expected dominant path killing, got %s", profile.DominantPath)
	}
	if profile.TotalDaoMarks != 50000 {
		t.Errorf("expected total marks 50000, got %d", profile.TotalDaoMarks)
	}

	for _, query := range []string{"Kill Path", "kill", "killing", "KILL PATH", "killing path"} {
		daoAmp, realmEff, conflictMult := profile.GetOffensiveFactors(query)
		if daoAmp != 501.0 {
			t.Errorf("query %s: expected daoAmp 501.0, got %f", query, daoAmp)
		}
		if realmEff != 1.25 {
			t.Errorf("query %s: expected realmEff 1.25, got %f", query, realmEff)
		}
		if conflictMult != 1.0 {
			t.Errorf("query %s: expected conflictMult 1.0, got %f", query, conflictMult)
		}
	}

	// Test with canonical "killing" key
	daoMarksKilling := map[string]int64{
		"killing": 50000,
	}
	masteriesKilling := map[string]int{
		"killing": 1000000,
	}
	profile2 := BuildDaoCombatProfile(daoMarksKilling, masteriesKilling, false, nil)
	daoAmp, realmEff, conflictMult := profile2.GetOffensiveFactors("Kill Path")
	if daoAmp != 501.0 || realmEff != 1.25 || conflictMult != 1.0 {
		t.Errorf("query Kill Path with canonical daoMarks: got amp=%f eff=%f conf=%f", daoAmp, realmEff, conflictMult)
	}
}

func TestMultiPathConflictMath(t *testing.T) {
	// RealmControlForLevel(4) must fully neutralise conflict.
	if got := RealmControlForLevel(4); got != 1.00 {
		t.Fatalf("RealmControlForLevel(4) = %v, want 1.00", got)
	}

	// F_Multi grows with irreconcilable paths and low purity.
	if f := MultiPathFactor(1.0, 0.5); f != 1.0 {
		t.Errorf("MultiPathFactor(1.0, .5) = %v, want 1.0", f)
	}
	if f := MultiPathFactor(1.9, 0.5); f < 1.85 || f > 1.89 {
		t.Errorf("MultiPathFactor(1.9, .5) = %v, want ~1.872", f)
	}

	// Grace: a foreign path within the 2%% threshold contributes nothing.
	pure := BuildDaoCombatProfile(map[string]int64{"fire": 100000, "water": 500}, nil, false, nil)
	if _, _, c := pure.GetOffensiveFactors("fire"); c != 1.0 {
		t.Errorf("grace threshold conflict = %v, want 1.0", c)
	}

	// A path whose realm control is 100%% casts without penalty.
	master := BuildDaoCombatProfile(
		map[string]int64{"fire": 50000, "water": 50000},
		map[string]int{"fire": 100000000}, // level 4
		false, nil,
	)
	if _, _, c := master.GetOffensiveFactors("fire"); c != 1.0 {
		t.Errorf("Vô Thượng Đại Tông Sư conflict = %v, want 1.0", c)
	}
}

func TestDaoConflictCastHelpers(t *testing.T) {
	if got := ManaTaxMultiplier(1.0); got != 1.0 {
		t.Errorf("ManaTaxMultiplier(1.0) = %v, want 1.0", got)
	}
	if got := ManaTaxMultiplier(0.5); got < 1.74 || got > 1.76 {
		t.Errorf("ManaTaxMultiplier(0.5) = %v, want 1.75", got)
	}
	if got := CastFailureChance(0.70); got != 0.0 {
		t.Errorf("CastFailureChance(0.70) = %v, want 0", got)
	}
	if got := CastFailureChance(0.08); got != 0.25 {
		t.Errorf("CastFailureChance(0.08) = %v, want 0.25", got)
	}
	if got := BackfireChance(0.60); got != 0.0 {
		t.Errorf("BackfireChance(0.60) = %v, want 0", got)
	}
	if got := BackfireChance(0.08); got != 0.30 {
		t.Errorf("BackfireChance(0.08) = %v, want 0.30", got)
	}
}

func TestEcologyPurityFactor(t *testing.T) {
	cases := []struct {
		purity, want float64
	}{
		{1.00, 1.00},
		{0.70, 1.00},
		{0.50, 0.75},
		{0.30, 0.50},
		{0.10, 0.25},
		{0.00, 0.25},
	}
	for _, tt := range cases {
		if got := EcologyPurityFactor(tt.purity); got != tt.want {
			t.Errorf("EcologyPurityFactor(%v) = %v, want %v", tt.purity, got, tt.want)
		}
	}
}

func TestIsChaoticTribulation(t *testing.T) {
	// Pure single path: never chaotic.
	if IsChaoticTribulation(map[string]int64{"fire": 100000}, nil) {
		t.Error("pure path should not be chaotic")
	}
	// Two irreconcilable paths with low purity: chaotic (P=0.5, N_eff=1.9 < 2.5).
	two := map[string]int64{"fire": 50000, "water": 50000}
	if IsChaoticTribulation(two, nil) {
		t.Error("dual path (N_eff 1.9) should not reach the 2.5 chaotic threshold")
	}
	// Many conflicting paths: chaotic.
	many := map[string]int64{
		"fire": 10000, "water": 10000, "light": 10000, "dark": 10000, "wood": 10000, "poison": 10000,
	}
	if !IsChaoticTribulation(many, nil) {
		t.Errorf("expected chaotic for tạp tu (P=%.2f, N_eff=%.2f)", 10000.0/60000.0, EffectivePathCount(many, nil))
	}
	if got := BuildDaoCombatProfile(many, nil, false, nil).DominantPath; got != "dark" {
		t.Errorf("equal Dao Mark totals should choose the lexicographically first path, got %q", got)
	}
	// Empty marks: not chaotic.
	if IsChaoticTribulation(nil, nil) {
		t.Error("empty marks should not be chaotic")
	}
}

func BenchmarkBuildDaoCombatProfile(b *testing.B) {
	daoMarks := map[string]int64{
		"fire":     50000,
		"water":    20000,
		"wood":     10000,
		"strength": 5000,
		"wisdom":   2000,
	}
	masteries := map[string]int{
		"fire":  1000000,
		"water": 100000,
		"wood":  50000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = BuildDaoCombatProfile(daoMarks, masteries, false, nil)
	}
}
