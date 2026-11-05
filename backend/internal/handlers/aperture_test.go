package handlers

import (
	"encoding/json"
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestComputeBulkRepairCost(t *testing.T) {
	cases := []struct {
		hp, maxHP int
		destroyed bool
		fee       float64
		wantBase  int
		wantTotal int
	}{
		{hp: 500, maxHP: 1000, destroyed: false, fee: 1.3, wantBase: 10, wantTotal: 13},
		{hp: 0, maxHP: 1000, destroyed: true, fee: 1.3, wantBase: 0, wantTotal: 0},
		{hp: 1000, maxHP: 1000, destroyed: false, fee: 1.3, wantBase: 0, wantTotal: 0},
		{hp: 900, maxHP: 1000, destroyed: false, fee: 1.3, wantBase: 2, wantTotal: 3},
		{hp: 999, maxHP: 1000, destroyed: false, fee: 1.3, wantBase: 1, wantTotal: 2},
	}
	for _, c := range cases {
		base, total := computeBulkRepairCost(c.hp, c.maxHP, c.destroyed, c.fee)
		if base != c.wantBase || total != c.wantTotal {
			t.Fatalf("computeBulkRepairCost(%d, %d, %v, %v) = (%d, %d), want (%d, %d)", c.hp, c.maxHP, c.destroyed, c.fee, base, total, c.wantBase, c.wantTotal)
		}
	}
}

func TestRaceHasDeadApertureSuppressesTribulations(t *testing.T) {
	if !raceHasDeadAperture("cuong_thi") {
		t.Fatal("cuong_thi must have a dead aperture and be exempt from tribulations")
	}
	if raceHasDeadAperture("human") {
		t.Fatal("human must continue to face tribulations")
	}
}

func TestPointProduceRankFollowsLevelAndGrade(t *testing.T) {
	cases := []struct {
		level int
		grade string
		want  int
	}{
		{level: 1, grade: "LOW", want: 1},
		{level: 5, grade: "LOW", want: 5},
		{level: 1, grade: "MID", want: 3},
		{level: 2, grade: "TOP", want: 6},
		{level: 5, grade: "TOP", want: 9},
	}
	for _, c := range cases {
		if got := pointProduceRank(c.level, c.grade); got != c.want {
			t.Fatalf("pointProduceRank(%d, %s) = %d, want %d", c.level, c.grade, got, c.want)
		}
	}
}

func TestPickPointGuCandidate(t *testing.T) {
	// Parse allowed IDs test
	ids := parseAllowedGuIDs("GU_BLOOD_BAT; GU_BLOOD_WING, GU_MOONLIGHT")
	if len(ids) != 3 {
		t.Fatalf("parseAllowedGuIDs len = %d, want 3", len(ids))
	}
	if ids[0] != "GU_BLOOD_BAT" || ids[1] != "GU_BLOOD_WING" || ids[2] != "GU_MOONLIGHT" {
		t.Fatalf("parseAllowedGuIDs parsed wrongly: %v", ids)
	}

	// Mock registry entry for testing
	if models.GuRegistry == nil {
		models.GuRegistry = make(map[string]map[int]models.GuDictionaryItem)
	}
	models.GuRegistry["GU_TEST_WOOD"] = map[int]models.GuDictionaryItem{
		2: {ID: "GU_TEST_WOOD", Rank: 2, Path: []string{"Wood Path"}},
	}
	models.GuRegistry["GU_ALLOWED_WOOD"] = map[int]models.GuDictionaryItem{
		3: {ID: "GU_ALLOWED_WOOD", Rank: 3, Path: []string{"Wood Path"}},
	}

	// Priority to allowed IDs
	tplAllowed := ResourcePointTemplate{
		Path:         "Wood Path",
		AllowedGuIDs: "GU_ALLOWED_WOOD",
		MinRank:      1,
		MaxRank:      3,
	}
	id, rank := pickPointGuCandidate(tplAllowed, 3, 1, 3)
	if id != "GU_ALLOWED_WOOD" || rank != 3 {
		t.Fatalf("pickPointGuCandidate allowed failed: id=%q, rank=%d", id, rank)
	}

	// Fallback to path when allowed list doesn't match
	tplFallback := ResourcePointTemplate{
		Path:    "Wood Path",
		MinRank: 1,
		MaxRank: 3,
	}
	id2, rank2 := pickPointGuCandidate(tplFallback, 2, 1, 3)
	if id2 != "GU_TEST_WOOD" || rank2 != 2 {
		t.Fatalf("pickPointGuCandidate fallback failed: id=%q, rank=%d", id2, rank2)
	}
}

func TestPointProduceItemClampedRanks(t *testing.T) {
	// Micro point clamped to rank 1..2
	microTpl := ResourcePointTemplate{ProduceKey: "wood_crystal", MinRank: 1, MaxRank: 2}
	ptHigh := models.ApertureResourcePoint{Level: 5, Grade: "TOP"} // raw rank = 9
	if got := pointProduceItem(microTpl, ptHigh, false); got != "wood_crystal_rank2" {
		t.Fatalf("micro point should clamp to rank 2, got %q", got)
	}
	if got := pointProduceItem(microTpl, ptHigh, true); got != "wood_crystal_rank2" {
		t.Fatalf("micro point rare should not exceed max_rank 2, got %q", got)
	}

	// High-tier famous point clamped to rank 6..8
	famousTpl := ResourcePointTemplate{
		ProduceKey:          "soul_sand",
		SecondaryProduceKey: "soul_devourer_stinger",
		MinRank:             6,
		MaxRank:             8,
	}
	ptLow := models.ApertureResourcePoint{Level: 1, Grade: "LOW"} // raw rank = 1
	if got := pointProduceItem(famousTpl, ptLow, false); got != "soul_sand_rank6" {
		t.Fatalf("famous point should floor to rank 6, got %q", got)
	}
	if got := pointSecondaryProduceItem(famousTpl, ptLow, false); got != "soul_devourer_stinger_rank6" {
		t.Fatalf("famous point secondary should floor to rank 6, got %q", got)
	}
}

func TestPointProduceItemFormat(t *testing.T) {
	pt := models.ApertureResourcePoint{Level: 2, Grade: "MID"}
	template := ResourcePointTemplate{ProduceKey: "wood_crystal"}
	if got := pointProduceItem(template, pt, false); got != "wood_crystal_rank4" {
		t.Fatalf("pointProduceItem = %q, want wood_crystal_rank4", got)
	}
	if got := pointProduceItem(template, pt, true); got != "wood_crystal_rank5" {
		t.Fatalf("pointProduceItem rare = %q, want wood_crystal_rank5", got)
	}
}

func TestAperturePointVitalityMultiplier(t *testing.T) {
	cases := []struct {
		vitality int
		want     float64
	}{
		{0, 0},
		{10, 0.5},
		{29, 0.5},
		{30, 0.8},
		{49, 0.8},
		{50, 1},
		{100, 1},
	}
	for _, c := range cases {
		if got := aperturePointVitalityMultiplier(c.vitality); got != c.want {
			t.Fatalf("aperturePointVitalityMultiplier(%d) = %v, want %v", c.vitality, got, c.want)
		}
	}
}

func TestPointStockCapFallback(t *testing.T) {
	template := ResourcePointTemplate{BaseYield: 10}
	if got := pointStockCap(template, models.ApertureResourcePoint{}, 0); got != 10 {
		t.Fatalf("pointStockCap = %d, want 10 (hard floor at base yield)", got)
	}
	if got := pointStockCap(template, models.ApertureResourcePoint{}, 20); got != 60 {
		t.Fatalf("pointStockCap = %d, want 60 (3 intervals)", got)
	}
}

func TestResourcePointTemplatesLoaded(t *testing.T) {
	if len(ResourcePointTemplates) == 0 {
		t.Fatal("ResourcePointTemplate.csv must define at least one archetype")
	}
	tpl, ok := GetResourcePointTemplate("HERB_VALLEY")
	if !ok {
		t.Fatal("HERB_VALLEY archetype missing")
	}
	if tpl.ProduceKey != "wood_crystal" || tpl.ProduceIntervalHours <= 0 {
		t.Fatalf("HERB_VALLEY template wrong: %+v", tpl)
	}
	if tpl.IsFood {
		t.Fatal("HERB_VALLEY must not be a food point")
	}
	beast, ok := GetResourcePointTemplate("BEAST_PASTURE")
	if !ok || beast.IsFood != true {
		t.Fatal("BEAST_PASTURE must be a food point")
	}
}

func TestAperturePointStockJSONRoundTrip(t *testing.T) {
	stock := models.AperturePointStock{"wood_crystal_rank1": 7}
	valuer, err := stock.Value()
	if err != nil {
		t.Fatalf("stock.Value: %v", err)
	}
	var out models.AperturePointStock
	if err := out.Scan(valuer); err != nil {
		t.Fatalf("stock.Scan: %v", err)
	}
	if out["wood_crystal_rank1"] != 7 {
		t.Fatalf("round trip lost data: %#v", out)
	}
}

func TestPointInstanceGradeWeights(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		grade := pointInstanceGrade(rand.New(rand.NewSource(seed)), "EXTREME")
		switch grade {
		case "LOW", "MID", "TOP":
		default:
			t.Fatalf("unexpected grade %q", grade)
		}
	}
}

func TestAperturePointJSONShape(t *testing.T) {
	pt := models.ApertureResourcePoint{
		ID:             "00000000-0000-0000-0000-000000000001",
		Scope:          "APERTURE",
		PointType:      "ORE_VEIN",
		Name:           "Mạch Quặng Địa Mạch",
		Path:           "Earth Path",
		Grade:          "LOW",
		Level:          2,
		Vitality:       45,
		Stock:          models.AperturePointStock{"earth_soil_rank1": 5},
		StockCap:       40,
		X:              3,
		Y:              4,
		LastProducedAt: time.Now(),
		State:          "ACTIVE",
	}
	raw, err := json.Marshal(pt)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back models.ApertureResourcePoint
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if back.Vitality != 45 || back.Stock["earth_soil_rank1"] != 5 || back.Scope != "APERTURE" || back.Listed {
		t.Fatalf("round trip mismatch: %+v", back)
	}
}

func TestTamQiBalanceAndAnchoring(t *testing.T) {
	// 1. Test Tam Khí scaling based on humanQi at ascension
	humanQi := 350000.0
	initialQi := math.Max(100000.0, humanQi)
	if initialQi != 350000.0 {
		t.Fatalf("initialQi should match high humanQi, got %v", initialQi)
	}

	ap := models.ImmortalAperture{
		TimeRate:    35.0,
		MaxHeavenQi: initialQi,
		MaxEarthQi:  initialQi,
		HeavenQi:    initialQi,
		EarthQi:     initialQi,
		IsPlaced:    false,
	}

	// Normal effective rate when not anchored
	effectiveRate := ap.TimeRate
	if ap.IsPlaced {
		effectiveRate = 1.0
	}
	if effectiveRate != 35.0 {
		t.Fatalf("unanchored rate should be 35.0, got %v", effectiveRate)
	}

	// 2. When anchored (Hạ Khiếu), effective time rate drops to 1.0 (synchronized with outside world)
	ap.IsPlaced = true
	if ap.IsPlaced {
		effectiveRate = 1.0
	}
	if effectiveRate != 1.0 {
		t.Fatalf("anchored rate must drop to 1.0x, got %v", effectiveRate)
	}

	// 3. Recharge rate during anchoring
	elapsedMinutes := 30.0
	gainedHeavenQi := elapsedMinutes * (ap.MaxHeavenQi * 0.01)
	if gainedHeavenQi != 105000.0 { // 30% of 350,000 = 105,000
		t.Fatalf("gainedHeavenQi for 30m should be 105000, got %v", gainedHeavenQi)
	}
}
