package handlers

import (
	"encoding/csv"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestApplyTitleYieldBonus(t *testing.T) {
	cases := []struct {
		amount int64
		pct    float64
		want   int64
	}{
		{amount: 100, pct: 5, want: 105},
		{amount: 10, pct: 5, want: 10},
		{amount: 7, pct: 5, want: 7},
		{amount: 100, pct: 0, want: 100},
		{amount: 0, pct: 5, want: 0},
	}
	for _, c := range cases {
		if got := applyTitleYieldBonus(c.amount, c.pct); got != c.want {
			t.Fatalf("applyTitleYieldBonus(%d, %.1f) = %d, want %d", c.amount, c.pct, got, c.want)
		}
	}
}

func TestWorldNormalPointCapScalesWithArea(t *testing.T) {
	cases := []struct {
		minX, minY, maxX, maxY int
		want                   int
	}{
		{0, 0, 999, 999, 10},    // 1000×1000 → 10 mines
		{0, 0, 499, 499, 2},     // 500×500 → floor 2
		{0, 0, 4999, 4999, 16},  // 5000×5000 → capped 16
		{500, 500, 599, 599, 2}, // tiny map → floor 2
	}
	for _, c := range cases {
		got := worldNormalPointCap(worldRegionSeed{MinX: c.minX, MinY: c.minY, MaxX: c.maxX, MaxY: c.maxY})
		if got != c.want {
			t.Fatalf("worldNormalPointCap(%d,%d,%d,%d) = %d, want %d", c.minX, c.minY, c.maxX, c.maxY, got, c.want)
		}
	}
}

func TestWorldRegionSweepNeedsTopUp(t *testing.T) {
	full := worldRegionSweepRow{
		Seed:        worldRegionSeed{MinX: 0, MinY: 0, MaxX: 999, MaxY: 999, MaxRank: 5},
		MonsterRank: 5,
		HasBoss:     true,
		Normal:      10,
		Famous:      1,
	}
	if full.needsTopUp() {
		t.Fatal("a full region must not be topped up")
	}
	missingFamous := full
	missingFamous.Famous = 0
	if !missingFamous.needsTopUp() {
		t.Fatal("a rank>=4 region with a boss but no famous mine must be topped up")
	}
	fullNormalMissingFamous := worldRegionSweepRow{
		Seed:    worldRegionSeed{MinX: 0, MinY: 0, MaxX: 999, MaxY: 999, MaxRank: 2},
		HasBoss: false,
		Normal:  10,
		Famous:  0,
	}
	if fullNormalMissingFamous.needsTopUp() {
		t.Fatal("a boss-less low-rank region must not be re-scanned forever")
	}
}

func TestWorldPointGradeForRank(t *testing.T) {
	cases := []struct {
		rank int
		want string
	}{
		{rank: 0, want: "LOW"},
		{rank: 3, want: "LOW"},
		{rank: 4, want: "MID"},
		{rank: 6, want: "MID"},
		{rank: 7, want: "TOP"},
		{rank: 9, want: "TOP"},
	}
	for _, c := range cases {
		if got := worldPointGradeForRank(c.rank); got != c.want {
			t.Fatalf("worldPointGradeForRank(%d) = %s, want %s", c.rank, got, c.want)
		}
	}
}

func TestBuildApertureYieldPreview(t *testing.T) {
	y := gamefx.ComputeApertureYield(gamefx.ApertureYieldInputs{
		Level:        2,
		AreaSize:     3_000_000,
		DaoMult:      1.0,
		ConflictMult: 1.0,
		EcologyMult:  1.0,
		LaborMult:    1.0,
		PopBonus:     1.0,
	})
	// levelM 1.5 × areaM 1.3 = 1.95 → 19.5 per 12 internal hours.
	if y.Count != 19.5 {
		t.Fatalf("unexpected yield %.2f", y.Count)
	}
	p := buildApertureYieldPreview(y, 12, 30)
	if p.Base != gamefx.ApertureYieldBase {
		t.Fatalf("base = %.1f, want %.1f", p.Base, gamefx.ApertureYieldBase)
	}
	if p.Whole != 19 {
		t.Fatalf("whole = %d, want 19", p.Whole)
	}
	if p.Remainder < 0.49 || p.Remainder > 0.51 {
		t.Fatalf("remainder = %.2f, want 0.5", p.Remainder)
	}
	// 19.5 per 12 internal hours at 30x → 48.75 per real hour.
	if p.PerHour < 48.74 || p.PerHour > 48.76 {
		t.Fatalf("per_hour = %.2f, want 48.75", p.PerHour)
	}
	if len(p.Factors) != 7 {
		t.Fatalf("factors = %d, want 7", len(p.Factors))
	}
	if p.Factors[0].Key != "level" || p.Factors[6].Key != "pop" {
		t.Fatalf("unexpected factor order: %+v", p.Factors)
	}
}

func TestDecorateAperturePointViewFillsTemplateFields(t *testing.T) {
	// Built inline so the test does not depend on the CSV being reachable from
	// the package working directory.
	template := ResourcePointTemplate{
		PointType:            "HERB_VALLEY",
		Name:                 "Thung Lũng Linh Thảo",
		Path:                 "Wood Path",
		ProduceKey:           "wood_crystal",
		ProduceIntervalHours: 12,
		BaseYield:            10,
		WorkerRace:           "HAIRY_MAN",
	}
	pt := &models.ApertureResourcePoint{PointType: template.PointType, Grade: "LOW", Level: 1}
	y := gamefx.ComputeApertureYield(gamefx.ApertureYieldInputs{Level: 1, AreaSize: 3_000_000, DaoMult: 1, ConflictMult: 1, EcologyMult: 1, LaborMult: 1, PopBonus: 1})
	decorateAperturePointView(pt, template, y, 10, 120)
	if pt.ProduceKey != template.ProduceKey {
		t.Fatalf("produce_key = %s, want %s", pt.ProduceKey, template.ProduceKey)
	}
	if pt.NextItemID != "wood_crystal_rank1" {
		t.Fatalf("next_item_id = %s, want wood_crystal_rank1", pt.NextItemID)
	}
	if pt.ProductionIntervalHours != template.ProduceIntervalHours {
		t.Fatalf("interval = %d, want %d", pt.ProductionIntervalHours, template.ProduceIntervalHours)
	}
	if pt.AssignedWorkers != 120 {
		t.Fatalf("assigned_workers = %d, want 120", pt.AssignedWorkers)
	}
	if pt.YieldPreview == nil {
		t.Fatal("yield preview must be filled")
	}
}

func TestWorldTransplantManaCost(t *testing.T) {
	cases := []struct {
		grade string
		rank  int
		want  float64
	}{
		{grade: "LOW", rank: 6, want: 20},
		{grade: "MID", rank: 6, want: 30},
		{grade: "TOP", rank: 6, want: 50},
		// Rank 9 is three Chuyển above the base: ×2.5.
		{grade: "TOP", rank: 9, want: 125},
		{grade: "MID", rank: 7, want: 45},
		// Sub-rank-6 callers clamp onto the baseline instead of discounting.
		{grade: "LOW", rank: 1, want: 20},
		// Unknown grades fall back to the LOW table.
		{grade: "bogus", rank: 6, want: 20},
	}
	for _, c := range cases {
		if got := worldTransplantManaCost(c.grade, c.rank); got != c.want {
			t.Fatalf("worldTransplantManaCost(%s, %d) = %.0f, want %.0f", c.grade, c.rank, got, c.want)
		}
	}
}

func TestWorldTransplantRitualSeconds(t *testing.T) {
	cases := []struct {
		grade string
		rank  int
		want  int
	}{
		{grade: "LOW", rank: 6, want: 180},
		{grade: "MID", rank: 6, want: 300},
		{grade: "TOP", rank: 6, want: 480},
		// −12% per Chuyển above six: rank 9 → 64% of the base.
		{grade: "TOP", rank: 9, want: 307},
		// The rank speed-up is floored at 35% of the base duration.
		{grade: "LOW", rank: 40, want: 63},
	}
	for _, c := range cases {
		if got := worldTransplantRitualSeconds(c.grade, c.rank); got != c.want {
			t.Fatalf("worldTransplantRitualSeconds(%s, %d) = %d, want %d", c.grade, c.rank, got, c.want)
		}
	}
	if got := worldTransplantDuration("MID", 6); got != 5*time.Minute {
		t.Fatalf("worldTransplantDuration(MID, 6) = %s, want 5m", got)
	}
}

func TestWorldPointInReach(t *testing.T) {
	cases := []struct {
		name                 string
		playerRegion, region string
		playerX, playerY     float64
		pointX, pointY       int
		want                 bool
	}{
		{name: "on the mine", playerRegion: "R", region: "R", playerX: 10, playerY: 10, pointX: 10, pointY: 10, want: true},
		{name: "edge of range", playerRegion: "R", region: "R", playerX: 16, playerY: 4, pointX: 10, pointY: 10, want: true},
		{name: "out of range", playerRegion: "R", region: "R", playerX: 17, playerY: 10, pointX: 10, pointY: 10, want: false},
		{name: "wrong region", playerRegion: "R2", region: "R", playerX: 10, playerY: 10, pointX: 10, pointY: 10, want: false},
		{name: "empty region", playerRegion: "", region: "R", playerX: 10, playerY: 10, pointX: 10, pointY: 10, want: false},
	}
	for _, c := range cases {
		if got := worldPointInReach(c.playerRegion, c.region, c.playerX, c.playerY, c.pointX, c.pointY); got != c.want {
			t.Fatalf("%s: worldPointInReach = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestFormatTransplantCountdown(t *testing.T) {
	cases := []struct {
		in   time.Duration
		want string
	}{
		{in: -5 * time.Second, want: "0 giây"},
		{in: 45 * time.Second, want: "45 giây"},
		{in: 90 * time.Second, want: "1 phút 30 giây"},
		{in: 2 * time.Minute, want: "2 phút"},
	}
	for _, c := range cases {
		if got := formatTransplantCountdown(c.in); got != c.want {
			t.Fatalf("formatTransplantCountdown(%s) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNormalizeWorkerRace(t *testing.T) {
	cases := map[string]string{
		"HAIRY_MAN":  "fur_people",
		"FEATHERMAN": "feathered",
		"ROCKMAN":    "rockman",
		" rockman ":  "rockman",
		"ghostman":   "ghostman",
	}
	for in, want := range cases {
		if got := normalizeWorkerRace(in); got != want {
			t.Fatalf("normalizeWorkerRace(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestResourcePointTemplatesReferenceRealData keeps ResourcePointTemplate.csv
// honest: every produce_key must resolve to an item family and every worker
// race must exist in Race.csv, otherwise a point silently produces nothing or
// its worker bonus never applies.
func TestResourcePointTemplatesReferenceRealData(t *testing.T) {
	itemFile, err := os.Open(csvPath("Item.csv"))
	if err != nil {
		t.Skipf("Item.csv unavailable: %v", err)
	}
	defer itemFile.Close()
	families := map[string]bool{}
	itemRows, err := csv.NewReader(itemFile).ReadAll()
	if err != nil || len(itemRows) < 2 {
		t.Fatalf("read Item.csv: %v", err)
	}
	idIdx := 0
	for i, h := range itemRows[0] {
		if strings.TrimSpace(h) == "id" {
			idIdx = i
			break
		}
	}
	for _, row := range itemRows[1:] {
		if idIdx >= len(row) {
			continue
		}
		id := strings.TrimSpace(row[idIdx])
		if strings.HasSuffix(id, "_rank1") {
			families[strings.TrimSuffix(id, "_rank1")] = true
		}
	}

	raceFile, err := os.Open(csvPath("Race.csv"))
	if err != nil {
		t.Skipf("Race.csv unavailable: %v", err)
	}
	defer raceFile.Close()
	races := map[string]bool{}
	raceRows, err := csv.NewReader(raceFile).ReadAll()
	if err != nil || len(raceRows) < 2 {
		t.Fatalf("read Race.csv: %v", err)
	}
	keyIdx := 0
	for i, h := range raceRows[0] {
		if strings.TrimSpace(h) == "key" {
			keyIdx = i
			break
		}
	}
	for _, row := range raceRows[1:] {
		if keyIdx < len(row) {
			races[strings.TrimSpace(row[keyIdx])] = true
		}
	}

	if len(ResourcePointTemplates) == 0 {
		t.Fatal("no resource point templates loaded")
	}
	for pointType, tpl := range ResourcePointTemplates {
		if !families[tpl.ProduceKey] {
			t.Fatalf("%s: produce_key %q has no %s_rank1 item in Item.csv", pointType, tpl.ProduceKey, tpl.ProduceKey)
		}
		if !races[tpl.WorkerRace] {
			t.Fatalf("%s: worker_race %q is not a Race.csv key", pointType, tpl.WorkerRace)
		}
	}
}
