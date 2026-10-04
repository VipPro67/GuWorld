package gamefx

import (
	"math"
	"strconv"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestBaseRankPower(t *testing.T) {
	if BaseRankPower(1) != 10.0 {
		t.Errorf("expected 10.0 for rank 1, got %f", BaseRankPower(1))
	}
	if BaseRankPower(6) != 100.0 {
		t.Errorf("expected 100.0 for rank 6, got %f", BaseRankPower(6))
	}
	if BaseRankPower(9) != 500.0 {
		t.Errorf("expected 500.0 for rank 9, got %f", BaseRankPower(9))
	}
}

func TestGetPathAttainment(t *testing.T) {
	masteries := models.PathMasteriesMap{
		"wisdom":      15000,  // Level 1: Master
		"information": 120000, // Level 2: Grandmaster
	}

	base, mult := GetPathAttainment(masteries, "wisdom")
	if mult != ScoutingAttainMult[1] || base != ScoutingAttainBase[1] {
		t.Errorf("wisdom master: got base=%f mult=%f, want %f/%f",
			base, mult, ScoutingAttainBase[1], ScoutingAttainMult[1])
	}

	// Highest attained level among the candidates wins.
	baseMax, multMax := GetPathAttainment(masteries, "wisdom", "information")
	if multMax != ScoutingAttainMult[2] || baseMax != ScoutingAttainBase[2] {
		t.Errorf("expected level 2 values, got base=%f mult=%f", baseMax, multMax)
	}

	// The flat base must strictly increase: Cảnh giới is the dominant pillar.
	for lvl := 1; lvl < len(ScoutingAttainBase); lvl++ {
		if ScoutingAttainBase[lvl] <= ScoutingAttainBase[lvl-1] {
			t.Errorf("attain base not increasing at level %d: %f <= %f",
				lvl, ScoutingAttainBase[lvl], ScoutingAttainBase[lvl-1])
		}
	}
}

// TestMaxPathLevelExactMatch guards the substring-matching bug: probe
// "information" must not match "formation", and vice versa.
func TestMaxPathLevelExactMatch(t *testing.T) {
	masteries := models.PathMasteriesMap{
		"formation": 10000000, // Supreme formation mastery
	}
	if lvl := maxPathLevel(masteries, "information"); lvl != 0 {
		t.Errorf("information must not inherit formation mastery, got level %d", lvl)
	}
	if lvl := maxPathLevel(masteries, "transformation"); lvl != 0 {
		t.Errorf("transformation must not inherit formation mastery, got level %d", lvl)
	}
	// Exact hit still works, including the "X Path" spelling from CSV.
	mixed := models.PathMasteriesMap{"Wisdom Path": 120000}
	if lvl := maxPathLevel(mixed, "wisdom"); lvl != 2 {
		t.Errorf("expected level 2 for 'Wisdom Path', got %d", lvl)
	}
}

func TestEvaluateDivinationOutcomes(t *testing.T) {
	ScoutingRand.Seed(7)
	// 1. Full Success: High Rank + Wisdom Tông Sư + Correct Clues vs Weak Target
	viewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"wisdom": 150000}},
	}
	target := &models.Player{
		Account:          models.Account{Username: "TargetMortal"},
		CultivationState: models.CultivationState{Rank: 2},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	viewerDao := models.DaoMarksMap{"wisdom": 1000} // +100%
	clues := models.DivinationRequest{
		TargetUsername:  "TargetMortal",
		SuspectedRegion: "hub_nam_cuong",
	}

	status, delta, _, myScore := EvaluateDivination(viewer, target, nil, nil, viewerDao, nil, clues, 20.0, false, false)
	if status != "full" || delta < 30.0 {
		t.Errorf("expected full success with delta >= 30, got status=%s, delta=%f", status, delta)
	}
	if myScore.Stat <= 0 || myScore.Total == 0 {
		t.Errorf("expected populated myScore breakdown, got %+v", myScore)
	}

	// 2. Backfire: Low Rank Mortal divining Immortal with High Dark Dao
	ScoutingRand.Seed(7)
	weakViewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 1},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	supremeTarget := &models.Player{
		Account:          models.Account{Username: "ImmortalVenerable"},
		CultivationState: models.CultivationState{Rank: 8},
		WorldState:       models.WorldState{CurrentRegion: "hub_trung_chau"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"dark": 1500000}}, // Great Grandmaster (2.0)
	}
	targetDao := models.DaoMarksMap{"dark": 2000}
	blindClues := models.DivinationRequest{
		TargetUsername: "ImmortalVenerable",
	}

	weakStatus, weakDelta, backfireDamage, _ := EvaluateDivination(weakViewer, supremeTarget, nil, nil, nil, targetDao, blindClues, 0.0, false, false)
	if weakStatus != "backfire" || weakDelta >= -30.0 {
		t.Errorf("expected backfire with delta < -30, got status=%s, delta=%f", weakStatus, weakDelta)
	}
	if backfireDamage <= 0 {
		t.Errorf("expected positive backfire damage, got %f", backfireDamage)
	}
}

func TestDivinationClueValidation(t *testing.T) {
	viewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	target := &models.Player{
		Account:          models.Account{Username: "Prey"},
		CultivationState: models.CultivationState{Rank: 5},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	base := models.DivinationRequest{TargetUsername: "Prey"}

	// Item clue without ownership must grant nothing (blind base avoids the +40 cap).
	noItem := DeductionBreakdown(viewer, nil, nil, models.DivinationRequest{ItemClueID: "xuan_bing_gu"}, target, 0, false, false, 0)
	withItem := DeductionBreakdown(viewer, nil, nil, models.DivinationRequest{ItemClueID: "xuan_bing_gu"}, target, 0, true, false, 0)
	if withItem.Total-noItem.Total != 30.0 {
		t.Errorf("expected validated item clue worth +30, got diff=%f", withItem.Total-noItem.Total)
	}

	// Wrong sect grants nothing; correct sect grants +10.
	wrongSect := DeductionBreakdown(viewer, nil, nil, models.DivinationRequest{TargetUsername: "Prey", SuspectedSect: "Thiên Kiếm", TargetSectName: "Huyết Hải"}, target, 0, false, false, 0)
	rightSect := DeductionBreakdown(viewer, nil, nil, models.DivinationRequest{TargetUsername: "Prey", SuspectedSect: "Huyết Hải", TargetSectName: "Huyết Hải"}, target, 0, false, false, 0)
	if rightSect.Total-wrongSect.Total != 10.0 {
		t.Errorf("expected validated sect clue worth +10, got diff=%f", rightSect.Total-wrongSect.Total)
	}
	_ = base

	// Valid suspected Gu grants +10.
	noGu := DeductionBreakdown(viewer, nil, nil, models.DivinationRequest{TargetUsername: "Prey", SuspectedGu: "moon_gu"}, target, 0, false, false, 0)
	withGu := DeductionBreakdown(viewer, nil, nil, models.DivinationRequest{TargetUsername: "Prey", SuspectedGu: "moon_gu"}, target, 0, false, true, 0)
	if withGu.Total-noGu.Total != 10.0 {
		t.Errorf("expected validated gu clue worth +10, got diff=%f", withGu.Total-noGu.Total)
	}

	// Clue pillar is hard-capped at +40.
	stacked := DeductionBreakdown(viewer, nil, nil, models.DivinationRequest{
		TargetUsername: "Prey", SuspectedSect: "Huyết Hải", TargetSectName: "Huyết Hải",
		SuspectedRegion: "hub_nam_cuong", SuspectedGu: "moon_gu", ItemClueID: "moon_gu",
	}, target, 0, true, true, 0)
	if stacked.Clue > 40.0 {
		t.Errorf("expected clue pillar capped at 40, got %f", stacked.Clue)
	}
}

func TestFaithClueAmplifyAndSever(t *testing.T) {
	plainViewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	faithViewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"information": 150000}}, // level 2
	}
	target := &models.Player{
		Account:          models.Account{Username: "Prey"},
		CultivationState: models.CultivationState{Rank: 5},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	clues := models.DivinationRequest{TargetUsername: "Prey", SuspectedRegion: "hub_nam_cuong"}

	plain := DeductionBreakdown(plainViewer, nil, nil, clues, target, 0, false, false, 0)
	faith := DeductionBreakdown(faithViewer, nil, nil, clues, target, 0, false, false, 0)
	// 35 clue * 1.3 = 45.5 -> capped at 40.
	if faith.Clue != 40.0 || plain.Clue != 35.0 {
		t.Errorf("expected faith-amplified clue 40 vs plain 35, got %f vs %f", faith.Clue, plain.Clue)
	}

	// Defender Tín đạo severs the same clue: level 2 -> *0.76.
	warded := &models.Player{
		Account:          models.Account{Username: "Prey"},
		CultivationState: models.CultivationState{Rank: 5},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"information": 150000}},
	}
	severed := DeductionBreakdown(plainViewer, nil, nil, clues, warded, 0, false, false, 0)
	if math.Abs(severed.Clue-35.0*0.76) > 1e-9 {
		t.Errorf("expected severed clue %f, got %f", 35.0*0.76, severed.Clue)
	}
}

func TestHeavenVeilAndPierce(t *testing.T) {
	ScoutingRand.Seed(11)
	mortal := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"wisdom": 150000}},
	}
	veiled := &models.Player{
		Account:          models.Account{Username: "Veiled"},
		CultivationState: models.CultivationState{Rank: 6},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"heaven": 150000}}, // level 2
	}
	veiledDao := models.DaoMarksMap{"heaven": 1000} // veil = 40 + 2 = 42
	clues := models.DivinationRequest{TargetUsername: "Veiled"}

	ScoutingRand.Seed(11)
	_, blind, _, _ := EvaluateDivination(mortal, veiled, nil, nil, nil, veiledDao, clues, 0, false, false)

	piercer := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		WorldState:       models.WorldState{CurrentRegion: "hub_nam_cuong"},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"wisdom": 150000, "heaven": 150000}},
	}
	piercerDao := models.DaoMarksMap{"heaven": 1000} // pierce = 20 + 2 = 22
	ScoutingRand.Seed(11)
	_, pierced, _, _ := EvaluateDivination(piercer, veiled, nil, nil, piercerDao, veiledDao, clues, 0, false, false)

	// Deltas now pass the rank normaliser; pierce is worth pierce × norm.
	want := 22.0 * scoutingRankNorm(veiled.Rank)
	if math.Abs((pierced-blind)-want) > 1e-9 {
		t.Errorf("expected heaven pierce worth +%f delta, got %f", want, pierced-blind)
	}
}

func TestScoutingDaoMultSaturates(t *testing.T) {
	if scoutingDaoMult() != 1.0 || scoutingDaoMult(0) != 1.0 {
		t.Errorf("expected 1.0 for no marks, got %f / %f", scoutingDaoMult(), scoutingDaoMult(0))
	}
	m1k := scoutingDaoMult(1000)
	if m1k < 1.04 || m1k > 1.06 {
		t.Errorf("expected ~1.05 for 1000 marks, got %f", m1k)
	}
	// Whale marks must asymptote at the low ceiling (+20%), never explode.
	if w := scoutingDaoMult(1000000); w > 1.0+ScoutingDaoCeiling || w < 1.19 {
		t.Errorf("expected ~1.20 cap for 1M marks, got %f", w)
	}
	// Đạo ngân is the weakest pillar: its ceiling must stay below a single
	// Cảnh giới level's flat base, so dao marks can never outrank attainment.
	if ScoutingDaoCeiling*100 > ScoutingAttainBase[1] {
		t.Errorf("dao ceiling adds %.0f, more than one cảnh giới level's %.0f flat base",
			ScoutingDaoCeiling*100, ScoutingAttainBase[1])
	}
	// Whale breakdown stays proportional and bounded.
	whale := &models.Player{
		CultivationState: models.CultivationState{Rank: 9},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"wisdom": 20000000}},
	}
	bd := DeductionBreakdown(whale, nil, models.DaoMarksMap{"wisdom": 100000}, models.DivinationRequest{}, nil, 0, false, false, 0)
	if bd.RealmDao > 2.0*bd.Stat {
		t.Errorf("whale realm_dao unbounded: stat=%f realm=%f", bd.Stat, bd.RealmDao)
	}
}

// seedScoutingGu installs a minimal Gu registry so tests can exercise the Gu
// pillars (INVESTIGATION / DISGUISE) without a live database.
func seedScoutingGu(t *testing.T, investigation, disguise float64) {
	t.Helper()
	previous := models.GuRegistry
	t.Cleanup(func() { models.GuRegistry = previous })

	mk := func(stat string, value float64) string {
		if value == 0 {
			return `[]`
		}
		return `[{"id":"PASSIVE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},` +
			`"operations":[{"op":"STAT_MOD","stat":"` + stat + `","sub_type":"ADD_FLAT","value":` +
			formatFloat(value) + `}]}]`
	}
	_ = mk

	raw := `[{"id":"PASSIVE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[` +
		`{"op":"STAT_MOD","stat":"INVESTIGATION","sub_type":"ADD_FLAT","value":` + formatFloat(investigation) + `},` +
		`{"op":"STAT_MOD","stat":"DISGUISE","sub_type":"ADD_FLAT","value":` + formatFloat(disguise) + `}]}]`

	var effects models.GuEffects
	if err := effects.Scan([]byte(raw)); err != nil {
		t.Fatalf("scan scouting test Gu effects: %v", err)
	}
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_SCOUT": {
			9: {ID: "GU_SCOUT", Rank: 9, Effects: effects},
		},
	}
}

func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// TestScoutingPriorityOrder locks the intended power hierarchy for divination:
// Cảnh giới lưu phái > Cổ trùng > Đạo ngân.
func TestScoutingPriorityOrder(t *testing.T) {
	seedScoutingGu(t, 9, 9)

	base := &models.Player{CultivationState: models.CultivationState{Rank: 7}}
	guMax := &models.Player{CultivationState: models.CultivationState{Rank: 7}}
	for i := 0; i < 8; i++ {
		guMax.GuInventory = append(guMax.GuInventory, models.GuInstance{
			GuID: "GU_SCOUT", Rank: 9, Equipped: true, Status: "ACTIVE",
		})
	}
	wisdomL4 := &models.Player{
		CultivationState: models.CultivationState{Rank: 7},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"wisdom": 10000000}},
	}

	total := func(p *models.Player, marks models.DaoMarksMap) float64 {
		bd := DeductionBreakdown(p, models.FilterEquippedActive(p.GuInventory), marks,
			models.DivinationRequest{TargetUsername: "Prey"}, nil, 0, false, false, 0)
		return bd.Total
	}

	// Compare each pillar across its full investment range, since a player
	// who maxes one should out-invest someone who maxes the next.
	attainGain := total(wisdomL4, nil) - total(base, nil)
	guGain := total(guMax, nil) - total(base, nil)
	daoGain := total(base, models.DaoMarksMap{"wisdom": 50000}) - total(base, nil)

	if attainGain <= 0 || guGain <= 0 || daoGain <= 0 {
		t.Fatalf("a pillar gives no gain: cảnh giới %+.1f, cổ trùng %+.1f, đạo ngân %+.1f",
			attainGain, guGain, daoGain)
	}
	if !(attainGain > guGain && guGain > daoGain) {
		t.Errorf("priority order broken: cảnh giới %+.1f, cổ trùng %+.1f, đạo ngân %+.1f",
			attainGain, guGain, daoGain)
	}
}

func TestLuckRollBounds(t *testing.T) {
	ScoutingRand.Seed(3)
	viewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	target := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	// No luck attainment/stats: width 10, no bias.
	for i := 0; i < 100; i++ {
		r := RollLuckDelta(viewer, nil, target, nil)
		if r < -10.0 || r >= 10.0 {
			t.Fatalf("luck roll out of bounds [-10,10): %f", r)
		}
	}
}

func TestCalculateSeverCluesPower(t *testing.T) {
	player := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"sword": 120000}}, // Grandmaster 1.5
	}
	power := CalculateSeverCluesPower(player, nil)
	want := 50.0 * ScoutingAttainMult[2]
	if math.Abs(power-want) > 1e-9 {
		t.Errorf("expected %f, got %f", want, power)
	}
}

func TestEvaluateGuDivination(t *testing.T) {
	// 1. High rank player divining mortal Gu -> Full success
	ScoutingRand.Seed(7)
	viewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"wisdom": 150000}},
	}
	mortalGu := models.GuDictionaryItem{
		ID:   "moonlight_gu",
		Name: "Nguyệt Quang Cổ",
		Rank: 1,
		Path: []string{"moon", "light"},
	}
	status, delta, _, _, _ := EvaluateGuDivination(viewer, nil, nil, mortalGu, 1, 10.0, 10.0)
	if status != "full" || delta < 20.0 {
		t.Errorf("expected full success for rank 5 divining rank 1 Gu, got status=%s, delta=%f", status, delta)
	}

	// 2. Rank 1 mortal trying to divine Rank 6 Immortal Gu -> Backfire
	ScoutingRand.Seed(7)
	weakViewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 1},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{}},
	}
	immortalGu := models.GuDictionaryItem{
		ID:   "chun_qiu_chan",
		Name: "Xuân Thu Thiền",
		Rank: 6,
		Path: []string{"time"},
	}
	weakStatus, weakDelta, dmg, soulDmg, _ := EvaluateGuDivination(weakViewer, nil, nil, immortalGu, 1, 150.0, 0.0)
	if weakStatus != "backfire" || weakDelta >= -40.0 {
		t.Errorf("expected backfire for rank 1 divining immortal Gu, got status=%s, delta=%f", weakStatus, weakDelta)
	}
	if dmg <= 0 || soulDmg <= 0 {
		t.Errorf("expected positive backfire damages, got dmg=%f, soulDmg=%f", dmg, soulDmg)
	}

	// 3. Ownership resonance: when player holds the Gu, they get direct perception resonance
	ScoutingRand.Seed(7)
	equippedGu := []models.GuInstance{
		{GuID: "chun_qiu_chan", Rank: 6, Equipped: true, Status: "ACTIVE"},
	}
	ownerViewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 6},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"time": 150000}},
	}
	ownerStatus, ownerDelta, _, _, _ := EvaluateGuDivination(ownerViewer, equippedGu, nil, immortalGu, 1, 50.0, 10.0)
	if ownerStatus != "full" || ownerDelta < 20.0 {
		t.Errorf("expected full success with ownership resonance, got status=%s, delta=%f", ownerStatus, ownerDelta)
	}
}

func TestGuRankDivisor(t *testing.T) {
	want := map[int]float64{0: 1, 1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 6: 10, 7: 18, 8: 30, 9: 50}
	for rank, d := range want {
		if got := GuRankDivisor(rank); got != d {
			t.Errorf("GuRankDivisor(%d) = %f, want %f", rank, got, d)
		}
	}
}

func TestGuDivinationRankScaling(t *testing.T) {
	viewer := &models.Player{
		CultivationState: models.CultivationState{Rank: 5},
		DaoState:         models.DaoState{PathMasteries: models.PathMasteriesMap{"wisdom": 150000}},
	}
	rank1Gu := models.GuDictionaryItem{ID: "g1", Name: "G1", Rank: 1, Path: []string{"moon"}}

	// Same viewer fulls an unowned rank-1 Gu.
	ScoutingRand.Seed(21)
	st1, _, _, _, s1 := EvaluateGuDivination(viewer, nil, nil, rank1Gu, 0, 0, 0)
	if st1 != "full" {
		t.Errorf("expected full on unowned rank-1 Gu, got %s (delta=%f)", st1, s1.Total-15.0)
	}
	if s1.RankDivisor != 1.0 {
		t.Errorf("expected divisor 1 for rank 1, got %f", s1.RankDivisor)
	}

	// Same viewer cannot touch a rank-6 Gu hidden by a strong holder.
	rank6Gu := models.GuDictionaryItem{ID: "g6", Name: "G6", Rank: 6, Path: []string{"time"}}
	ScoutingRand.Seed(21)
	st6, d6, _, _, s6 := EvaluateGuDivination(viewer, nil, nil, rank6Gu, 1, 150.0, 0)
	if st6 == "full" || st6 == "partial" {
		t.Errorf("expected fogged/backfire on holder-hidden rank-6 Gu, got %s delta=%f", st6, d6)
	}
	if s6.RankDivisor != 10.0 {
		t.Errorf("expected divisor 10 for rank 6, got %f", s6.RankDivisor)
	}
}
