package handlers

import (
	"encoding/json"
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
