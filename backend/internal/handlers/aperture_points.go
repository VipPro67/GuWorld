package handlers

// Resource points (Điểm Tài Nguyên) own the Immortal Aperture production
// economy, canon-style (blessed-land resource points). Points are generated
// from the aperture's Dao-mark composition instead of being placed cell by
// cell; they accumulate stock lazily, drain vitality per harvest and can be
// traded. Defense structures remain on aperture_resource_nodes.

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
)

const (
	MaxPointLevel            = 5
	aperturePointPointGenCap = 8
)

// ResourcePointTemplate mirrors one row of data/ResourcePointTemplate.csv.
type ResourcePointTemplate struct {
	PointType            string
	Name                 string
	Path                 string  // canonical aperture path label, e.g. "Wood Path"
	ProduceKey           string  // item family prefix, e.g. "wood_crystal"
	ProduceIntervalHours int     // internal hours between production ticks
	BaseYield            float64 // base units per interval, before multipliers
	VitalityDrain        int     // vitality lost per harvest
	VitalityRegenHours   int     // internal hours for a full vitality refill
	IsFood               bool
	WorkerRace           string  // variant-human race tending the point
	RareRate             float64 // chance a produced interval is one tier higher
	GuRate               float64 // chance each harvest rolls a wild Gu worm (rank 1-6 by difficulty)
	PointClass           string  // NORMAL | FAMOUS (canon famous mountains / heaven-earth secret realms)
}

// ResourcePointTemplates holds resource point definitions loaded from
// data/ResourcePointTemplate.csv at startup. Edit the CSV and restart to
// rebalance points without a code change.
var ResourcePointTemplates = loadResourcePointTemplates()

func GetResourcePointTemplate(pointType string) (ResourcePointTemplate, bool) {
	pointType = strings.TrimSpace(pointType)
	if len(ResourcePointTemplates) == 0 {
		ResourcePointTemplates = loadResourcePointTemplates()
	}
	t, ok := ResourcePointTemplates[pointType]
	return t, ok
}

func loadResourcePointTemplates() map[string]ResourcePointTemplate {
	f, err := os.Open(csvPath("ResourcePointTemplate.csv"))
	if err != nil {
		if logger.S != nil {
			logger.S.Warnf("resource point templates not found: %v", err)
		}
		return map[string]ResourcePointTemplate{}
	}
	defer f.Close()
	return parseResourcePointTemplates(f)
}

// legacyWorkerRaces maps pre-registry template spellings onto the current
// Race.csv keys. The labor/population multiplier matches races exactly, so a
// stale spelling silently costs the player every worker bonus.
var legacyWorkerRaces = map[string]string{
	"hairy_man":  "fur_people",
	"featherman": "feathered",
	"rock_man":   "rockman",
}

// normalizeWorkerRace lower-cases a template/DB race key and resolves legacy
// aliases so it lines up with the Race.csv registry keys.
func normalizeWorkerRace(race string) string {
	key := strings.ToLower(strings.TrimSpace(race))
	if mapped, ok := legacyWorkerRaces[key]; ok {
		return mapped
	}
	return key
}

func parseResourcePointTemplates(r io.Reader) map[string]ResourcePointTemplate {
	templates := make(map[string]ResourcePointTemplate)
	rows, err := csv.NewReader(r).ReadAll()
	if err != nil || len(rows) < 2 {
		return templates
	}
	header := make(map[string]int, len(rows[0]))
	for i, h := range rows[0] {
		header[strings.TrimSpace(h)] = i
	}
	atof := func(s string) float64 {
		v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
		return v
	}
	atoi := func(s string) int {
		n, _ := strconv.Atoi(strings.TrimSpace(s))
		return n
	}
	for _, row := range rows[1:] {
		at := func(key string) string {
			if i, ok := header[key]; ok && i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}
		t := ResourcePointTemplate{
			PointType:            at("point_type"),
			Name:                 at("name"),
			Path:                 at("path"),
			ProduceKey:           at("produce_key"),
			ProduceIntervalHours: atoi(at("produce_interval_hours")),
			BaseYield:            atof(at("base_yield")),
			VitalityDrain:        atoi(at("vitality_drain")),
			VitalityRegenHours:   atoi(at("vitality_regen_hours")),
			IsFood:               at("is_food") == "true",
			WorkerRace:           normalizeWorkerRace(at("worker_race")),
			RareRate:             atof(at("rare_rate")),
			GuRate:               atof(at("gu_rate")),
			PointClass:           strings.ToUpper(strings.TrimSpace(at("point_class"))),
		}
		if t.PointType != "" {
			templates[t.PointType] = t
		}
	}
	return templates
}

// ---- item helpers -------------------------------------------------------

// pointProduceItem returns the item id produced by a point for one interval.
// The produced tier rises with the point level and instance grade; the rare
// chance can bump one interval to the next tier.
func pointProduceItem(t ResourcePointTemplate, pt models.ApertureResourcePoint, rare bool) string {
	rank := pointProduceRank(pt.Level, pt.Grade)
	if rare {
		rank = min(9, rank+1)
	}
	return fmt.Sprintf("%s_rank%d", t.ProduceKey, rank)
}

func pointProduceRank(level int, grade string) int {
	gradeBonus := map[string]int{"LOW": 0, "MID": 2, "TOP": 4}[grade]
	rank := level + gradeBonus
	if rank < 1 {
		rank = 1
	}
	if rank > 9 {
		rank = 9
	}
	return rank
}

// pointStockCap bounds the accumulated units: a small multiple of one
// interval's yield so idling beyond it wastes produce (anti-idle canon:
// nobody leaves a blessed land unharvested forever).
func pointStockCap(t ResourcePointTemplate, pt models.ApertureResourcePoint, perInterval float64) int64 {
	cap := int64(math.Ceil(perInterval * 3.0))
	if cap < int64(t.BaseYield) {
		cap = int64(t.BaseYield)
	}
	return cap
}

// aperturePointVitalityMultiplier scales yield down as vitality drops.
func aperturePointVitalityMultiplier(vitality int) float64 {
	switch {
	case vitality <= 0:
		return 0
	case vitality < 30:
		return 0.5
	case vitality < 50:
		return 0.8
	default:
		return 1
	}
}

// ---- generation ---------------------------------------------------------

// GenerateApertureResourcePoints creates `count` resource points for the
// aperture. The point-type weights follow the aperture's Dao-mark composition
// (dominant path first, minor paths share the rest), mirroring how the novel
// moulds blessed-land terrain out of dao marks. Positions are deterministic
// per aperture so snapshot replays stay stable.
func GenerateApertureResourcePoints(tx *sql.Tx, apertureID string, ownerPlayerID string, daoMarks models.DaoMarksMap, apertureGrade string, count int) error {
	if count <= 0 {
		return nil
	}
	var existing int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM aperture_resource_points WHERE aperture_id = $1`, apertureID).Scan(&existing); err != nil {
		return err
	}
	count = min(count, aperturePointPointGenCap-existing)
	if count <= 0 {
		return nil
	}

	// Dao-mark weighted archetype pool.
	pool := make([]aperturePointTemplateEntry, 0, len(ResourcePointTemplates))
	for _, t := range ResourcePointTemplates {
		if t.PointType == "" {
			continue
		}
		pool = append(pool, aperturePointTemplateEntry{t: t, weight: 1})
	}
	var totalMarks float64
	pathWeights := map[string]float64{}
	for path, marks := range daoMarks {
		if marks > 0 {
			totalMarks += float64(marks)
			pathWeights[normalizePathKey(path)] = float64(marks)
		}
	}
	if totalMarks > 0 {
		dominant := 0.0
		dominantKey := ""
		for key, w := range pathWeights {
			if w > dominant {
				dominant = w
				dominantKey = key
			}
		}
		poolPathWeight := make([]float64, len(pool))
		poolTotal := 0.0
		for i, e := range pool {
			key := normalizePathKey(e.t.Path)
			// Dao-following: the dominant path gets 70% share, minor paths
			// split the remaining 30% proportionally (same shape as the
			// tribulation reward split).
			w := 0.3 * (e.t.BaseYield)
			if key == dominantKey {
				w += 0.7 * totalMarks
			} else if marks := pathWeights[key]; marks > 0 {
				w += 0.3 * marks
			}
			poolPathWeight[i] = w
			poolTotal += w
		}
		if poolTotal > 0 {
			for i := range pool {
				pool[i].weight = poolPathWeight[i]
			}
		}
	}

	seed := fnv.New64a()
	_, _ = seed.Write([]byte(apertureID))
	rng := rand.New(rand.NewSource(int64(seed.Sum64())))

	usedCells := map[string]bool{}
	{
		rows, err := tx.Query(`SELECT x, y FROM aperture_resource_points WHERE aperture_id = $1`, apertureID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var x, y int
				if rows.Scan(&x, &y) == nil {
					usedCells[fmt.Sprintf("%d,%d", x, y)] = true
				}
			}
		}
	}

	instanceGrade := pointInstanceGrade(rng, apertureGrade)
	for i := 0; i < count; i++ {
		pick := weightedPointPick(rng, pool, instanceGrade)
		if pick.PointType == "" {
			break
		}

		x, y, ok := pickFreeApertureCell(rng, usedCells)
		if !ok {
			return nil
		}

		if _, err := tx.Exec(`
			INSERT INTO aperture_resource_points (aperture_id, point_type, name, path, grade, level, vitality, stock, stock_cap, x, y, last_produced_at, state)
			VALUES ($1, $2, $3, $4, $5, 1, 100, '{}'::jsonb, 0, $6, $7, NOW(), 'ACTIVE')
		`, apertureID, pick.PointType, pick.Name, pick.Path, instanceGrade, x, y); err != nil {
			return err
		}
	}
	_ = ownerPlayerID
	return nil
}

// errGiftCodeNeedsAperture marks gift-code resource-point rewards that cannot be
// applied because the player has not opened an aperture yet.
var errGiftCodeNeedsAperture = errors.New("đạo hữu cần mở Tiên Khiếu (Lục Chuyển) trước khi nhận Điểm Tài Nguyên")

// apertureUsedCells collects the occupied 20×20 grid cells of an aperture.
func apertureUsedCells(tx *sql.Tx, apertureID string) map[string]bool {
	used := map[string]bool{}
	rows, err := tx.Query(`SELECT x, y FROM aperture_resource_points WHERE aperture_id = $1`, apertureID)
	if err != nil {
		return used
	}
	defer rows.Close()
	for rows.Next() {
		var x, y int
		if rows.Scan(&x, &y) == nil {
			used[fmt.Sprintf("%d,%d", x, y)] = true
		}
	}
	return used
}

// pickFreeApertureCell returns a free cell on the 20×20 aperture grid. Random
// probing keeps layouts varied; the deterministic scan guarantees the remaining
// cells are still found when the grid is nearly full.
func pickFreeApertureCell(rng *rand.Rand, used map[string]bool) (int, int, bool) {
	for attempt := 0; attempt < 64; attempt++ {
		x, y := rng.Intn(20), rng.Intn(20)
		key := fmt.Sprintf("%d,%d", x, y)
		if used[key] {
			continue
		}
		used[key] = true
		return x, y, true
	}
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			key := fmt.Sprintf("%d,%d", x, y)
			if used[key] {
				continue
			}
			used[key] = true
			return x, y, true
		}
	}
	return 0, 0, false
}

// grantApertureResourcePointsFromReward seeds up to `count` resource points into
// the player's aperture for a gift-code reward. An empty pointType keeps the
// natural Dao-mark weighted selection; a non-empty one forces that template.
// The 8-point cap is authoritative: when the aperture is full this returns 0.
func grantApertureResourcePointsFromReward(tx *sql.Tx, playerID uuid.UUID, pointType string, count int) (int, error) {
	if tx == nil || count <= 0 {
		return 0, nil
	}
	var apertureID, grade string
	var daoJSON []byte
	err := tx.QueryRow(`
		SELECT id, grade, COALESCE(dao_marks, '{}'::jsonb)
		FROM immortal_apertures WHERE player_id = $1 FOR UPDATE
	`, playerID).Scan(&apertureID, &grade, &daoJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, errGiftCodeNeedsAperture
	}
	if err != nil {
		return 0, err
	}

	pointType = strings.ToUpper(strings.TrimSpace(pointType))
	if pointType != "" {
		if _, ok := GetResourcePointTemplate(pointType); !ok {
			return 0, fmt.Errorf("loại điểm tài nguyên không tồn tại: %s", pointType)
		}
	}

	if pointType == "" {
		var dao models.DaoMarksMap
		_ = json.Unmarshal(daoJSON, &dao)
		var before, after int
		if err := tx.QueryRow(`SELECT COUNT(1) FROM aperture_resource_points WHERE aperture_id = $1`, apertureID).Scan(&before); err != nil {
			return 0, err
		}
		if before >= aperturePointPointGenCap {
			return 0, nil
		}
		if err := GenerateApertureResourcePoints(tx, apertureID, playerID.String(), dao, grade, count); err != nil {
			return 0, err
		}
		if err := tx.QueryRow(`SELECT COUNT(1) FROM aperture_resource_points WHERE aperture_id = $1`, apertureID).Scan(&after); err != nil {
			return 0, err
		}
		return after - before, nil
	}

	template, _ := GetResourcePointTemplate(pointType)
	used := apertureUsedCells(tx, apertureID)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	added := 0
	for i := 0; i < count; i++ {
		var existing int
		if err := tx.QueryRow(`SELECT COUNT(1) FROM aperture_resource_points WHERE aperture_id = $1`, apertureID).Scan(&existing); err != nil {
			return added, err
		}
		if existing >= aperturePointPointGenCap {
			break
		}
		x, y, ok := pickFreeApertureCell(rng, used)
		if !ok {
			break
		}
		instanceGrade := pointInstanceGrade(rng, grade)
		if _, err := tx.Exec(`
			INSERT INTO aperture_resource_points
				(aperture_id, point_type, name, path, grade, level, vitality, stock, stock_cap, x, y, last_produced_at, state)
			VALUES ($1, $2, $3, $4, $5, 1, 100, '{}'::jsonb, 0, $6, $7, NOW(), 'ACTIVE')
		`, apertureID, template.PointType, template.Name, template.Path, instanceGrade, x, y); err != nil {
			return added, err
		}
		added++
	}
	return added, nil
}

// pointInstanceGrade rents the aperture grade down to the point grade: higher
// blessed lands roll better mixtures (ch. 609 — grade decides essence beads
// and wealth).
func pointInstanceGrade(rng *rand.Rand, apertureGrade string) string {
	weights := map[string]float64{"LOW": 0.6, "MID": 0.3, "TOP": 0.1}
	switch apertureGrade {
	case "MID":
		weights = map[string]float64{"LOW": 0.45, "MID": 0.4, "TOP": 0.15}
	case "HIGH":
		weights = map[string]float64{"LOW": 0.25, "MID": 0.45, "TOP": 0.3}
	case "EXTREME":
		weights = map[string]float64{"LOW": 0.1, "MID": 0.4, "TOP": 0.5}
	}
	roll := rng.Float64()
	t := 0.0
	for _, grade := range []string{"LOW", "MID", "TOP"} {
		t += weights[grade]
		if roll < t {
			return grade
		}
	}
	return "LOW"
}

type aperturePointTemplateEntry struct {
	t      ResourcePointTemplate
	weight float64
}

func weightedPointPick(rng *rand.Rand, pool []aperturePointTemplateEntry, grade string) ResourcePointTemplate {
	total := 0.0
	for _, e := range pool {
		total += e.weight
	}
	if total <= 0 {
		return ResourcePointTemplate{}
	}
	roll := rng.Float64() * total
	acc := 0.0
	for _, e := range pool {
		acc += e.weight
		if roll < acc {
			return e.t
		}
	}
	return pool[len(pool)-1].t
}

// ---- accrual ------------------------------------------------------------

// accrueAperturePoint applies the lazy projection to one locked point row:
// vitality regenerates across elapsed internal time and produced intervals
// fill the stock up to its cap (overflow is lost — canon anti-idle).
// It runs inside the caller's transaction and writes back when anything moved.
func accrueAperturePoint(tx *sql.Tx, pt *models.ApertureResourcePoint, template ResourcePointTemplate, effectiveTimeRate float64, areaSize int64, ctx *aperturePointRewardContext, workerMult float64) error {
	now := time.Now()
	elapsed := now.Sub(pt.LastProducedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	elapsedInternalHours := elapsed.Hours() * effectiveTimeRate

	changed := false

	// 1. Vitality regen: full refill takes VitalityRegenHours internal hours.
	if template.VitalityRegenHours > 0 && pt.Vitality < 100 {
		regen := math.Floor(elapsedInternalHours * 100.0 / float64(template.VitalityRegenHours))
		if regen > 0 {
			pt.Vitality = int(math.Min(100, float64(pt.Vitality)+regen))
			changed = true
		}
	}

	// 2. Production: whole intervals only; partial intervals stay pending.
	intervalHours := template.ProduceIntervalHours
	if intervalHours <= 0 {
		intervalHours = 12
	}
	if intervals := math.Floor(elapsedInternalHours / float64(intervalHours)); intervals > 0 {
		y := aperturePointYield(ctx, *pt, template, areaSize, workerMult)
		rareBump := 0.0
		if template.RareRate > 0 {
			// Expected extra-tier output folds the rare chance in smoothly.
			rareBump = y.Count * template.RareRate
		}
		perInterval := y.Count + rareBump
		pt.StockCap = pointStockCap(template, *pt, perInterval)
		mult := aperturePointVitalityMultiplier(pt.Vitality)
		if pt.State == "EXHAUSTED" {
			mult = 0
		}
		produced := perInterval * mult * intervals
		perItemTemplate := template
		rareItem := pointProduceItem(perItemTemplate, *pt, false)
		rareItemNext := pointProduceItem(perItemTemplate, *pt, true)
		if pt.Stock == nil {
			pt.Stock = models.AperturePointStock{}
		}
		current := pt.Stock[rareItem]
		freeCap := float64(pt.StockCap) - float64(current)
		if freeCap > 0 {
			added := math.Floor(math.Min(produced, freeCap))
			if produced > freeCap {
				changed = true
			}
			if added > 0 {
				pt.Stock[rareItem] = current + int64(added)
				// Rare bump lands on the next-tier item.
				if rareBump > 0 {
					nextFree := float64(pt.StockCap) - float64(pt.Stock[rareItemNext])
					if nextFree > 0 {
						rareUnits := rareBump * mult * intervals
						pt.Stock[rareItemNext] += int64(math.Floor(math.Min(rareUnits, nextFree)))
					}
				}
				changed = true
			}
		}
		pt.LastProducedAt = now
		changed = true
	}

	if !changed {
		return nil
	}
	_, err := tx.Exec(`
		UPDATE aperture_resource_points
		SET vitality = $1, stock = $2::jsonb, last_produced_at = $3, updated_at = NOW()
		WHERE id = $4
	`, pt.Vitality, models.AperturePointStock(pt.Stock), pt.LastProducedAt, pt.ID)
	return err
}

// ---- reward context -----------------------------------------------------

// aperturePointRewardContext is the point-shape cousin of the resource-node
// context: dao marks/conflict profile plus variant-human labour data.
type aperturePointRewardContext struct {
	daoMarks      map[string]int64
	daoProfile    *gamefx.DaoCombatProfile
	variantHumans map[string]struct {
		population     int
		laborIntensity string
	}
	assignedWorkers map[string]int // pointID -> assigned population
}

func loadAperturePointRewardContext(q interface {
	QueryRow(query string, args ...any) *sql.Row
	Query(query string, args ...any) (*sql.Rows, error)
}, playerID uuid.UUID, apertureID string) (*aperturePointRewardContext, error) {
	ctx := &aperturePointRewardContext{
		daoMarks:        map[string]int64{},
		assignedWorkers: map[string]int{},
	}
	var daoJSON []byte
	if err := q.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&daoJSON); err == nil && len(daoJSON) > 0 {
		_ = json.Unmarshal(daoJSON, &ctx.daoMarks)
	}
	ctx.daoProfile, _ = loadApertureDaoCombatProfile(q, playerID.String(), ctx.daoMarks)

	vhRows, err := q.Query(`SELECT race, population, labor_intensity FROM aperture_variant_humans WHERE aperture_id = $1`, apertureID)
	if err == nil {
		defer vhRows.Close()
		for vhRows.Next() {
			var race, intensity string
			var pop int
			if vhRows.Scan(&race, &pop, &intensity) == nil {
				ctx.variantHumans[normalizeWorkerRace(race)] = struct {
					population     int
					laborIntensity string
				}{population: pop, laborIntensity: intensity}
			}
		}
	}

	wRows, err := q.Query(`
		SELECT w.point_id, SUM(w.assigned_population)
		FROM aperture_point_workers w
		JOIN aperture_resource_points p ON p.id = w.point_id
		WHERE p.aperture_id = $1
		GROUP BY w.point_id
	`, apertureID)
	if err == nil {
		defer wRows.Close()
		for wRows.Next() {
			var pointID string
			var pop int
			if wRows.Scan(&pointID, &pop) == nil {
				ctx.assignedWorkers[pointID] = pop
			}
		}
	}
	return ctx, nil
}

// aperturePointYield assembles the shared yield formula for one point.
// WORLD-scope points (public terrain on the map) run WITHOUT the owner's Dao
// amplification: production is plain, grade-tempered (canon: world mining is
// slow; only inside a blessed land do time-rate and dao marks accelerate it).
func aperturePointYield(ctx *aperturePointRewardContext, pt models.ApertureResourcePoint, template ResourcePointTemplate, areaSize int64, workerMult float64) gamefx.ApertureYield {
	if pt.Scope == "WORLD" {
		gradeBoost := map[string]int{"LOW": 0, "MID": 2, "TOP": 4}[pt.Grade]
		return gamefx.ComputeApertureYield(gamefx.ApertureYieldInputs{
			Level:        pt.Level,
			DaoMult:      1 + 0.02*float64(gradeBoost),
			ConflictMult: 1,
			EcologyMult:  1,
			LaborMult:    1,
			PopBonus:     max(1, max(workerMult, 1)),
		})
	}

	daoMult := 1.0
	if item, exists := models.ItemRegistry[fmt.Sprintf("%s_rank1", template.ProduceKey)]; exists && item.Path != "" {
		daoMult = getPathDaoMultiplier(item.Path, ctx.daoMarks)
	}
	if daoMult <= 1.0 {
		daoMult = getPathDaoMultiplier(pt.Path, ctx.daoMarks)
	}
	conflictMult := apertureDaoConflictMultiplierForProfile(ctx.daoProfile, pt.Path)

	ecologyMult := 1.0
	if p := nodeDaoPurity(ctx.daoMarks); p > 0 {
		purityMultiplier := gamefx.EcologyPurityFactor(p)
		reduction := 0.0
		if ctx.daoProfile != nil {
			reduction = ctx.daoProfile.ConflictReductionPct
		}
		ecologyPenalty := (1.0 - purityMultiplier) * (1.0 - reduction/100.0)
		ecologyMult = 1.0 - ecologyPenalty
	}

	laborMult := 1.0
	popBonus := 1.0
	if ctx != nil {
		if vh, ok := ctx.variantHumans[template.WorkerRace]; ok {
			if vh.population > 0 {
				popBonus = 1.0 + float64(vh.population)/1000.0
			}
			switch vh.laborIntensity {
			case "LOW":
				laborMult = 0.7
			case "HIGH":
				laborMult = 1.5
			case "EXPLOIT":
				laborMult = 2.0
			}
		}
	}
	popBonus *= workerMult

	return gamefx.ComputeApertureYield(gamefx.ApertureYieldInputs{
		Level:        pt.Level,
		AreaSize:     areaSize,
		DaoMult:      daoMult,
		ConflictMult: conflictMult,
		EcologyMult:  ecologyMult,
		LaborMult:    laborMult,
		PopBonus:     popBonus,
	})
}

// ---- view helpers -------------------------------------------------------

// decorateAperturePointView fills the view-only fields the client renders (they
// are not database columns): the produce family, production interval, the item
// the next harvest yields and the full multiplier breakdown.
func decorateAperturePointView(pt *models.ApertureResourcePoint, template ResourcePointTemplate, y gamefx.ApertureYield, effectiveRate float64, assignedWorkers int) {
	pt.ProduceKey = template.ProduceKey
	pt.WorkerRace = template.WorkerRace
	pt.AssignedWorkers = assignedWorkers
	pt.ProductionIntervalHours = template.ProduceIntervalHours
	if pt.ProductionIntervalHours <= 0 {
		pt.ProductionIntervalHours = 12
	}
	pt.NextItemID = pointProduceItem(template, *pt, false)
	pt.YieldPreview = buildApertureYieldPreview(y, pt.ProductionIntervalHours, effectiveRate)
}

// applyTitleYieldBonus scales a harvested amount by the equipped title's
// aperture-yield utility bonus (rounded down); 0% leaves it untouched.
func applyTitleYieldBonus(amount int64, bonusPct float64) int64 {
	if amount <= 0 || bonusPct <= 0 {
		return amount
	}
	return int64(math.Floor(float64(amount) * (1 + bonusPct/100)))
}

// buildApertureYieldPreview mirrors gamefx.ComputeApertureYield so the client
// never re-derives the harvest formula.
func buildApertureYieldPreview(y gamefx.ApertureYield, intervalHours int, effectiveRate float64) *models.ApertureYieldPreview {
	whole := int64(math.Floor(y.Count))
	factors := make([]models.ApertureYieldFactor, 0, len(y.Parts))
	for _, key := range []string{"level", "dao", "area", "conflict", "ecology", "labor", "pop"} {
		if value, ok := y.Parts[key]; ok {
			factors = append(factors, models.ApertureYieldFactor{Key: key, Value: value})
		}
	}
	hours := float64(intervalHours)
	if hours <= 0 {
		hours = 12
	}
	perHour := 0.0
	if effectiveRate > 0 {
		perHour = y.Count * effectiveRate / hours
	}
	return &models.ApertureYieldPreview{
		Base:      gamefx.ApertureYieldBase,
		Expected:  y.Count,
		Whole:     whole,
		Remainder: y.Count - float64(whole),
		Bonus:     y.Bonus,
		RawBonus:  y.RawBonus,
		Capped:    y.Capped,
		PerHour:   perHour,
		Factors:   factors,
	}
}

// ---- handlers -----------------------------------------------------------

func effectiveApertureTimeRate(tx *sql.Tx, playerID string) (areaSize int64, effectiveTimeRate float64) {
	var timeRate, tsm float64
	var race string
	if err := tx.QueryRow(`
		SELECT COALESCE(a.area_size, 0), COALESCE(a.time_rate, 1.0), p.time_speed_multiplier, COALESCE(p.race, '')
		FROM immortal_apertures a JOIN players p ON a.player_id = p.id
		WHERE a.player_id = $1
	`, playerID).Scan(&areaSize, &timeRate, &tsm, &race); err != nil {
		return 0, 1.0
	}
	if timeRate <= 0 {
		timeRate = 1.0
	}
	dummyP := models.Player{}
	dummyP.Race = race
	if !gamefx.Get(&dummyP).Immortal {
		if tsm <= 0 {
			tsm = 1.0
		}
		if bonus, err := db.EquippedTimeSpeedBonus(tx, playerID); err == nil {
			tsm += bonus
		}
	} else {
		tsm = 1.0
	}
	rate := timeRate * tsm
	if rate <= 0 {
		rate = 1.0
	}
	return areaSize, rate
}

// GetApertureResourcePoints returns every resource point of the aperture with
// lazily-accrued stock and a harvest preview.
func GetApertureResourcePoints(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var apertureID string
	err = tx.QueryRow(`SELECT id FROM immortal_apertures WHERE player_id = $1`, playerID).Scan(&apertureID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	}

	areaSize, effectiveTimeRate := effectiveApertureTimeRate(tx, playerID.String())
	ctx, err := loadAperturePointRewardContext(tx, playerID, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải bối cảnh sản lượng"})
		return
	}

	rows, err := tx.Query(`
		SELECT id, aperture_id, point_type, name, path, grade, COALESCE(point_class, 'NORMAL'), level, vitality, stock, stock_cap, x, y, last_produced_at, state
		FROM aperture_resource_points
		WHERE aperture_id = $1 AND state <> 'GONE'
		ORDER BY created_at ASC
	`, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải điểm tài nguyên"})
		return
	}
	defer rows.Close()

	points := []models.ApertureResourcePoint{}
	type lockedPoint struct {
		models.ApertureResourcePoint
		template ResourcePointTemplate
	}
	locked := []lockedPoint{}
	for rows.Next() {
		var p models.ApertureResourcePoint
		var aperturePtr *string
		var stockJSON []byte
		if err := rows.Scan(&p.ID, &aperturePtr, &p.PointType, &p.Name, &p.Path, &p.Grade, &p.PointClass, &p.Level, &p.Vitality, &stockJSON, &p.StockCap, &p.X, &p.Y, &p.LastProducedAt, &p.State); err == nil {
			p.ApertureID = aperturePtr
			_ = json.Unmarshal(stockJSON, &p.Stock)
			t, _ := GetResourcePointTemplate(p.PointType)
			locked = append(locked, lockedPoint{ApertureResourcePoint: p, template: t})
			points = append(points, p)
		}
	}
	rows.Close()

	for i := range locked {
		p := &locked[i]
		workerMult := 1.0
		if pop := ctx.assignedWorkers[p.ID]; pop > 0 {
			workerMult = 1 + 0.25*math.Min(1.0, float64(pop)/500.0)
		}
		if err := accrueAperturePoint(tx, &p.ApertureResourcePoint, p.template, effectiveTimeRate, areaSize, ctx, workerMult); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tích lũy sản lượng"})
			return
		}
		preview := aperturePointYield(ctx, p.ApertureResourcePoint, p.template, areaSize, workerMult)
		decorateAperturePointView(&p.ApertureResourcePoint, p.template, preview, effectiveTimeRate, ctx.assignedWorkers[p.ID])
		points[i] = p.ApertureResourcePoint
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu tích lũy"})
		return
	}
	c.JSON(http.StatusOK, points)
}

// ---- WORLD scope: public contested mining -------------------------------

// worldHarvestRange is the Chebyshev distance (cells) a player must stand
// within to harvest a public world resource point.
const worldHarvestRange = 6

// GetWorldResourcePoints lists public resource points around the player's
// current world position, lazily accruing their stock so previews are honest.
func GetWorldResourcePoints(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var regionID string
	var worldX, worldY float64
	err = tx.QueryRow(`SELECT COALESCE(current_region, ''), COALESCE(world_x, 0), COALESCE(world_y, 0) FROM players WHERE id = $1`, playerID).Scan(&regionID, &worldX, &worldY)
	if err != nil || regionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa đặt chân vào thế giới thực"})
		return
	}

	// Lazy top-up: a region must have mines the first time anyone looks at it,
	// before the periodic sweep (StartWorldResourcePointWorker) reaches it.
	if _, _, err := ensureWorldResourcePoints(c.Request.Context(), tx, regionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi sinh điểm tài nguyên thế giới"})
		return
	}

	rows, err := tx.Query(`
		SELECT id, COALESCE(scope, 'WORLD'), COALESCE(region_id, ''), point_type, name, path, grade, COALESCE(point_class, 'NORMAL'), level, vitality, stock, stock_cap, x, y, last_produced_at, state
		FROM aperture_resource_points
		WHERE scope = 'WORLD' AND region_id = $1
	`, regionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải điểm tài nguyên thế giới"})
		return
	}
	defer rows.Close()

	points := []models.ApertureResourcePoint{}
	for rows.Next() {
		var p models.ApertureResourcePoint
		var stockJSON []byte
		var pointRegion string
		if err := rows.Scan(&p.ID, &p.Scope, &pointRegion, &p.PointType, &p.Name, &p.Path, &p.Grade, &p.PointClass, &p.Level, &p.Vitality, &stockJSON, &p.StockCap, &p.X, &p.Y, &p.LastProducedAt, &p.State); err == nil {
			p.RegionID = &pointRegion
			_ = json.Unmarshal(stockJSON, &p.Stock)
			points = append(points, p)
		}
	}
	rows.Close()

	// Accrue (real-time pace: effectiveTimeRate = 1) without farming boosts.
	for i := range points {
		p := &points[i]
		template, ok := GetResourcePointTemplate(p.PointType)
		if !ok {
			continue
		}
		emptyCtx := &aperturePointRewardContext{daoMarks: map[string]int64{}}
		if err := accrueAperturePoint(tx, p, template, 1.0, 0, emptyCtx, 1.0); err != nil {
			continue
		}
		preview := aperturePointYield(emptyCtx, *p, template, 0, 1.0)
		decorateAperturePointView(p, template, preview, 1.0, 0)
	}

	// A transplant ritual only survives while its owner keeps standing at the
	// mine. The panel polls this listing during a ritual, so presence is
	// re-checked here as well as on the transplant call itself. Done after the
	// points are touched so every writer locks the point row before the ritual
	// row (the transplant handler does the same, avoiding a lock inversion).
	ritualAborted, err := abortPointTransplantWithoutPresence(tx, playerID, regionID, worldX, worldY)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra nghi thức di dời"})
		return
	}

	// Surface the in-flight ritual on its point so the client can resume the
	// countdown after reopening the panel.
	if ritual, ritualErr := loadPointTransplantRitual(tx, playerID); ritualErr == nil && ritual != nil {
		for i := range points {
			if points[i].ID == ritual.PointID {
				points[i].Transplant = ritual
			}
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu tích lũy"})
		return
	}
	if ritualAborted {
		// The listing also runs from the HUD without the panel open, so the
		// owner must still learn that the essence is gone.
		notify.Send(db.DB, playerID, "point_transplant", "Nghi Thức Tan",
			"Nghi thức di dời đã đứt vì đạo hữu rời khỏi mỏ — Tiên Nguyên đã tiêu không thể thu hồi.")
	}
	c.JSON(http.StatusOK, points)
}

// CollectWorldResourcePoint harvests a public world point in person: the
// player must stand close and the guarding cell must be cleared.
func CollectWorldResourcePoint(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pointID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã điểm tài nguyên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var p models.ApertureResourcePoint
	var apertureID *string
	var regionID string
	var stockJSON []byte
	err = tx.QueryRow(`
		SELECT p.id, COALESCE(p.aperture_id, NULL), COALESCE(p.scope, 'WORLD'), COALESCE(p.region_id, ''), p.x, p.y, p.vitality, p.stock, p.last_produced_at
		FROM aperture_resource_points p WHERE p.id = $1 FOR UPDATE OF p
	`, pointID).Scan(&p.ID, &apertureID, &p.Scope, &regionID, &p.X, &p.Y, &p.Vitality, &stockJSON, &p.LastProducedAt)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Điểm tài nguyên không tồn tại"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}
	if p.Scope != "WORLD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm này nằm trong Tiên Khiếu, thu hoạch theo luồng riêng."})
		return
	}
	_ = json.Unmarshal(stockJSON, &p.Stock)

	// 1. Presence: the player must be physically at the point.
	var playerRegion string
	var px, py float64
	if err := tx.QueryRow(`SELECT COALESCE(current_region, ''), COALESCE(world_x, 0), COALESCE(world_y, 0) FROM players WHERE id = $1`, playerID).Scan(&playerRegion, &px, &py); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi vị trí đạo hữu"})
		return
	}
	distance := math.Max(math.Abs(px-float64(p.X)), math.Abs(py-float64(p.Y)))
	if playerRegion != regionID || distance > float64(worldHarvestRange) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Phải đến nơi (cách %d ô) mới khai thác được điểm tài nguyên này.", int(distance))})
		return
	}

	// 2. Guard cleared: a persistent guardian (Boss/Monster map cell) on the
	// point's cell must be inside its respawn window. Cells without a
	// persistent guardian are open ground and can be mined directly.
	if isWorldMonsterSafeZone(regionID, p.X, p.Y) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vùng an toàn không thể khai thác."})
		return
	}
	if worldPointGuardStatus(c.Request.Context(), regionID, p.X, p.Y) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quái canh giữ điểm tài nguyên còn sống — hãy đánh bại bọn nó!"})
		return
	}

	// 3. Harvest on the shared flow (world pace: 1x, no owner boosts).
	message, granted, err := harvestPointRow(tx, playerID, &p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if granted {
		var rank int
		if err := tx.QueryRow(`SELECT rank FROM players WHERE id=$1`, playerID).Scan(&rank); err == nil {
			_ = TrackDailyMissionProgress(tx, playerID, rank, "world-point-collect:"+pointID.String(), "aperture_collect", "", 1)
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu thu hoạch"})
		return
	}
	if granted {
		notify.Send(db.DB, playerID, "world_point_harvest", "Khai Thác Thế Giới", message)
	}
	respondActionWithInventory(c, http.StatusOK, models.Player{Account: models.Account{ID: playerID}}, gin.H{"message": message})
}

// harvestPointRow drains a fully-loaded point row (shared by aperture/world).
func harvestPointRow(tx *sql.Tx, playerID uuid.UUID, p *models.ApertureResourcePoint) (string, bool, error) {
	ctx := &aperturePointRewardContext{daoMarks: map[string]int64{}, assignedWorkers: map[string]int{}}
	if p.Scope != "WORLD" && p.ApertureID != nil {
		full, err := loadAperturePointRewardContext(tx, playerID, *p.ApertureID)
		if err == nil {
			ctx = full
		}
	}
	template, ok := GetResourcePointTemplate(p.PointType)
	if !ok {
		return "", false, fmt.Errorf("Loại điểm tài nguyên không hợp lệ")
	}
	workerMult := 1.0
	if pop := ctx.assignedWorkers[p.ID]; pop > 0 {
		workerMult = 1 + 0.25*math.Min(1.0, float64(pop)/500.0)
	}
	areaSize := int64(0)
	rate := 1.0
	if p.Scope != "WORLD" {
		areaSize, rate = effectiveApertureTimeRate(tx, playerID.String())
		if areaSize == 0 {
			areaSize = 1
		}
	}
	if err := accrueAperturePoint(tx, p, template, rate, areaSize, ctx, workerMult); err != nil {
		return "", false, err
	}
	if p.State == "EXHAUSTED" || p.Vitality <= 0 {
		return fmt.Sprintf("%s đã cạn kiệt sinh khí, cần Bồi Dưỡng trước khi thu hoạch lại.", p.Name), false, nil
	}
	if len(p.Stock) == 0 {
		return fmt.Sprintf("%s chưa tích lũy đủ tài nguyên.", p.Name), false, nil
	}

	// Equipped achievement titles keep their aperture-yield utility bonus on
	// blessed-land harvests (parity with the retired node flow). Public world
	// mines run without the owner's bonuses, so they are excluded.
	titleBonusPct := 0.0
	if p.Scope != "WORLD" {
		titleBonusPct = equippedTitleUtilityBonus(tx, playerID.String())
	}

	grantedNames := make([]string, 0, len(p.Stock))
	var totalAmount int64
	for itemID, amount := range p.Stock {
		if amount <= 0 {
			continue
		}
		amount = applyTitleYieldBonus(amount, titleBonusPct)
		if amount > 10000000 {
			amount = 10000000
		}
		if err := InventorySvc.AddItem(tx, playerID.String(), itemID, "materials", amount); err != nil {
			return "", false, fmt.Errorf("Lỗi cộng vật phẩm thu hoạch")
		}
		translated := itemID
		if itemSpec, exists := models.ItemRegistry[itemID]; exists {
			translated = itemSpec.Name
		}
		grantedNames = append(grantedNames, fmt.Sprintf("%dx %s", amount, translated))
		totalAmount += amount
	}

	drain := template.VitalityDrain
	p.Vitality -= drain
	p.Stock = models.AperturePointStock{}
	if p.Vitality <= 0 {
		p.Vitality = 0
		p.State = "EXHAUSTED"
	}
	if _, err := tx.Exec(`
		UPDATE aperture_resource_points
		SET vitality = $1, state = $2, stock = '{}'::jsonb, last_produced_at = $3, updated_at = NOW()
		WHERE id = $4
	`, p.Vitality, p.State, time.Now(), p.ID); err != nil {
		return "", false, err
	}

	immortalChance := 0.0005
	if p.Scope != "WORLD" {
		immortalChance = 0.003
		if p.Grade == "TOP" {
			immortalChance = 0.006
		}
	}
	mintedGu := rollPointGuWorm(tx, playerID, template, *p, immortalChance)

	message := fmt.Sprintf("Thu hoạch %s: %s!", p.Name, strings.Join(grantedNames, ", "))
	if totalAmount > 0 {
		message += " Nhân Khí +2."
		addHumanQi(tx, playerID, 2)
	}
	if titleBonusPct > 0 && totalAmount > 0 {
		message += fmt.Sprintf(" Danh hiệu +%.0f%% sản lượng.", titleBonusPct)
	}
	if mintedGu != "" {
		message += " Sinh ra " + mintedGu + "!"
	}
	return message, totalAmount > 0, nil
}

// placeTransplantedPointOnGrid picks a free cell of the persistent 20×20
// aperture grid so a transplanted mountain participates in the grid/arena
// projections instead of keeping its world-map coordinates.
func placeTransplantedPointOnGrid(tx *sql.Tx, apertureID string) (int, int, bool) {
	occupied := map[[2]int]bool{}
	rows, err := tx.Query(`SELECT x, y FROM aperture_resource_points WHERE aperture_id = $1`, apertureID)
	if err != nil {
		return 0, 0, false
	}
	for rows.Next() {
		var x, y int
		if rows.Scan(&x, &y) == nil {
			occupied[[2]int{x, y}] = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, false
	}
	cells := make([][2]int, 0, 400)
	for y := 0; y < 20; y++ {
		for x := 0; x < 20; x++ {
			if !occupied[[2]int{x, y}] {
				cells = append(cells, [2]int{x, y})
			}
		}
	}
	if len(cells) == 0 {
		return 0, 0, false
	}
	pick := cells[rand.Intn(len(cells))]
	return pick[0], pick[1], true
}

// ---- Transplantation ritual ---------------------------------------------
//
// Canon "Immortal Aperture Transplantation" (arc 2121–2153) is a ritual, not a
// purchase: the mountain stays in the world while the cultivator burns Tiên
// Nguyên and keeps standing at the mine. The point only moves into the
// aperture once the ritual ends; walking away breaks it and the essence that
// was already burnt is lost.

const (
	// Base Tiên Nguyên burnt by the ritual at Rank 6, per mine grade. These are
	// deliberately token amounts: the real price of a transplantation is the
	// time spent standing at the mine, not the essence.
	worldTransplantManaLow = 20.0
	worldTransplantManaMid = 30.0
	worldTransplantManaTop = 50.0
	// Every Chuyển above Rank 6 raises the cost by 50% (LOW 20 → 50, TOP 50 →
	// 125 at Cửu Chuyển). The grade already tracks the region's rank, so both
	// the mine's grade and the player's rank affect the price.
	worldTransplantRankStep = 0.5
	// Base ritual length at Rank 6, per mine grade.
	worldTransplantRitualLowSecs = 180
	worldTransplantRitualMidSecs = 300
	worldTransplantRitualTopSecs = 480
	// Higher Chuyển speeds the ritual up: −12% per rank, never below 35% of the
	// base duration and never under a minute.
	worldTransplantRankSpeedup = 0.12
	worldTransplantMinSpeedup  = 0.35
	worldTransplantMinRitual   = 60
	// worldTransplantGrace bounds how long an overdue ritual keeps its mine
	// claimed. Past it the ritual is abandoned: the row is deleted (the burnt
	// Tiên Nguyên is gone) so one stale row can never lock a public mine for
	// everyone forever.
	worldTransplantGrace = 10 * time.Minute
	// worldTransplantSweepInterval is the abandoned-ritual cleanup cadence.
	worldTransplantSweepInterval = time.Minute
)

// errWorldPointNotMovable reports that the point left WORLD scope between the
// ritual start and its completion (another player moved it).
var errWorldPointNotMovable = errors.New("world point is no longer a public mine")

// normalizeTransplantGrade clamps unknown grades onto the LOW baseline so the
// cost tables stay total.
func normalizeTransplantGrade(grade string) string {
	switch strings.ToUpper(strings.TrimSpace(grade)) {
	case "MID":
		return "MID"
	case "TOP":
		return "TOP"
	default:
		return "LOW"
	}
}

// worldTransplantManaCost is the Tiên Nguyên the ritual burns.
func worldTransplantManaCost(grade string, rank int) float64 {
	cost := worldTransplantManaLow
	switch normalizeTransplantGrade(grade) {
	case "MID":
		cost = worldTransplantManaMid
	case "TOP":
		cost = worldTransplantManaTop
	}
	if rank > 6 {
		cost *= 1 + worldTransplantRankStep*float64(rank-6)
	}
	return math.Round(cost)
}

// worldTransplantRitualSeconds is how long the cultivator must stand at the
// mine before the mountain uproots.
func worldTransplantRitualSeconds(grade string, rank int) int {
	secs := worldTransplantRitualLowSecs
	switch normalizeTransplantGrade(grade) {
	case "MID":
		secs = worldTransplantRitualMidSecs
	case "TOP":
		secs = worldTransplantRitualTopSecs
	}
	if rank > 6 {
		speedup := 1 - worldTransplantRankSpeedup*float64(rank-6)
		if speedup < worldTransplantMinSpeedup {
			speedup = worldTransplantMinSpeedup
		}
		secs = int(math.Round(float64(secs) * speedup))
	}
	if secs < worldTransplantMinRitual {
		secs = worldTransplantMinRitual
	}
	return secs
}

func worldTransplantDuration(grade string, rank int) time.Duration {
	return time.Duration(worldTransplantRitualSeconds(grade, rank)) * time.Second
}

// formatTransplantCountdown renders a remaining ritual time for messages.
func formatTransplantCountdown(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	secs := int(math.Ceil(d.Seconds()))
	if secs < 60 {
		return fmt.Sprintf("%d giây", secs)
	}
	if secs%60 == 0 {
		return fmt.Sprintf("%d phút", secs/60)
	}
	return fmt.Sprintf("%d phút %d giây", secs/60, secs%60)
}

// formatTransplantEssence prints a Tiên Nguyên amount without decimals.
func formatTransplantEssence(amount float64) string {
	return strconv.FormatFloat(math.Round(amount), 'f', -1, 64)
}

// worldPointInReach reports whether the cultivator stands in the mine's region
// and within the harvesting Chebyshev range.
func worldPointInReach(playerRegion, pointRegion string, playerX, playerY float64, pointX, pointY int) bool {
	if playerRegion == "" || pointRegion == "" || playerRegion != pointRegion {
		return false
	}
	return math.Max(math.Abs(playerX-float64(pointX)), math.Abs(playerY-float64(pointY))) <= float64(worldHarvestRange)
}

// loadPointTransplantRitual returns the player's in-flight ritual, if any.
func loadPointTransplantRitual(tx *sql.Tx, playerID uuid.UUID) (*models.AperturePointTransplant, error) {
	var ritual models.AperturePointTransplant
	err := tx.QueryRow(`
		SELECT point_id::text, region_id, mana_spent, started_at, completes_at
		FROM aperture_point_transplants WHERE player_id = $1
	`, playerID).Scan(&ritual.PointID, &ritual.RegionID, &ritual.ManaSpent, &ritual.StartedAt, &ritual.CompletesAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ritual, nil
}

// clearPointTransplantRitual ends the ritual without moving the mine.
func clearPointTransplantRitual(tx *sql.Tx, playerID uuid.UUID) error {
	_, err := tx.Exec(`DELETE FROM aperture_point_transplants WHERE player_id = $1`, playerID)
	return err
}

// abortPointTransplantWithoutPresence drops a ritual whose owner no longer
// stands at the mine. Presence is the mechanic itself, so every world-point
// listing re-checks it; the burnt essence is never refunded.
func abortPointTransplantWithoutPresence(tx *sql.Tx, playerID uuid.UUID, playerRegion string, playerX, playerY float64) (bool, error) {
	ritual, err := loadPointTransplantRitual(tx, playerID)
	if err != nil || ritual == nil {
		return false, err
	}
	var pointX, pointY int
	scanErr := tx.QueryRow(`SELECT x, y FROM aperture_resource_points WHERE id = $1`, ritual.PointID).Scan(&pointX, &pointY)
	if scanErr != nil && scanErr != sql.ErrNoRows {
		return false, scanErr
	}
	if scanErr == nil && worldPointInReach(playerRegion, ritual.RegionID, playerX, playerY, pointX, pointY) {
		return false, nil
	}
	return true, clearPointTransplantRitual(tx, playerID)
}

// loadPlayerAperturePointSlot returns the player's aperture id and how many
// resource points already sit inside it.
func loadPlayerAperturePointSlot(tx *sql.Tx, playerID uuid.UUID) (string, int, error) {
	var apertureID string
	var pointCount int
	err := tx.QueryRow(`
		SELECT a.id::text, (SELECT COUNT(1) FROM aperture_resource_points WHERE aperture_id = a.id)
		FROM immortal_apertures a WHERE a.player_id = $1
	`, playerID).Scan(&apertureID, &pointCount)
	return apertureID, pointCount, err
}

// moveWorldPointIntoAperture parks a WORLD point on the persistent 20×20
// aperture grid and switches it to APERTURE scope. The mine may have been
// drained while it waited in the world, so its state is derived from the
// surviving vitality instead of being forced ACTIVE.
func moveWorldPointIntoAperture(tx *sql.Tx, apertureID, pointID string) error {
	gridX, gridY, hasGridCell := placeTransplantedPointOnGrid(tx, apertureID)
	setClause := `
		UPDATE aperture_resource_points
		SET scope = 'APERTURE', aperture_id = $1, region_id = NULL,
		    last_produced_at = NOW(), updated_at = NOW(),
		    state = CASE WHEN vitality > 0 THEN 'ACTIVE' ELSE 'EXHAUSTED' END`
	var res sql.Result
	var err error
	if hasGridCell {
		res, err = tx.Exec(setClause+`, x = $2, y = $3 WHERE id = $4 AND scope = 'WORLD'`, apertureID, gridX, gridY, pointID)
	} else {
		res, err = tx.Exec(setClause+` WHERE id = $2 AND scope = 'WORLD'`, apertureID, pointID)
	}
	if err != nil {
		return err
	}
	if affected, rowsErr := res.RowsAffected(); rowsErr == nil && affected == 0 {
		return errWorldPointNotMovable
	}
	return nil
}

// expirePointTransplants deletes rituals that stayed overdue longer than the
// grace period and returns the owners that lost their ritual. An empty pointID
// sweeps every mine; otherwise only that mine's abandoned ritual is released
// so a new claimant can take it.
func expirePointTransplants(tx *sql.Tx, pointID string) ([]uuid.UUID, error) {
	query := `
		DELETE FROM aperture_point_transplants
		WHERE completes_at < NOW() - ($1 * INTERVAL '1 second')`
	args := []any{worldTransplantGrace.Seconds()}
	if pointID != "" {
		query += ` AND point_id = $2`
		args = append(args, pointID)
	}
	query += ` RETURNING player_id`

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	owners := []uuid.UUID{}
	for rows.Next() {
		var owner uuid.UUID
		if err := rows.Scan(&owner); err == nil {
			owners = append(owners, owner)
		}
	}
	return owners, rows.Err()
}

// notifyAbandonedTransplants tells every owner whose abandoned ritual was
// cleaned up that the burnt Tiên Nguyên is gone for good.
func notifyAbandonedTransplants(owners []uuid.UUID) {
	for _, owner := range owners {
		notify.Send(db.DB, owner, "point_transplant", "Nghi Thức Tan",
			"Nghi thức di dời đã quá hạn và bị hủy — Tiên Nguyên đã tiêu không thể thu hồi. Hãy làm lại nghi thức nếu vẫn còn muốn dời núi.")
	}
}

// ExpireStalePointTransplants releases mines claimed by abandoned rituals. It
// is wired to a periodic worker so an owner who logs out mid-ritual can never
// lock a public mine for everyone else forever.
func ExpireStalePointTransplants(ctx context.Context) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		logger.S.Errorf("Point transplant sweep: begin: %v", err)
		return
	}
	defer func() { _ = tx.Rollback() }()

	owners, err := expirePointTransplants(tx, "")
	if err != nil {
		logger.S.Errorf("Point transplant sweep: delete: %v", err)
		return
	}
	if len(owners) == 0 {
		return
	}
	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Point transplant sweep: commit: %v", err)
		return
	}
	notifyAbandonedTransplants(owners)
	if len(owners) > 0 {
		logger.S.Infof("Point transplant sweep: released %d abandoned ritual(s)", len(owners))
	}
}

// TransplantWorldResourcePoint runs the canon "Immortal Aperture
// Transplantation" ritual. The first call starts it (202): Tiên Nguyên is
// burnt and the mine stays in the world while the cultivator stands at it.
// A later call after completes_at finishes it (200) and moves the mountain
// into the aperture. Leaving the mine breaks the ritual and wastes the essence.
func TransplantWorldResourcePoint(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pointID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã điểm tài nguyên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	// Read the payer without a row lock: the burn is an atomic conditional
	// UPDATE later, and locking the player before the point would invert the
	// point→player lock order used by CollectWorldResourcePoint (deadlock).
	var rank int
	var currentMana float64
	var playerRegion string
	var playerX, playerY float64
	if err := tx.QueryRow(`
		SELECT rank, COALESCE(current_mana, 0), COALESCE(current_region, ''), COALESCE(world_x, 0), COALESCE(world_y, 0)
		FROM players WHERE id = $1
	`, playerID).Scan(&rank, &currentMana, &playerRegion, &playerX, &playerY); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi vị trí đạo hữu"})
		return
	}

	now := time.Now()
	ritual, err := loadPointTransplantRitual(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra nghi thức di dời"})
		return
	}
	// breakRitual ends the ritual (the burnt Tiên Nguyên is never refunded) and
	// commits, telling the player why. Errors are reported instead of silently
	// leaving a row that would keep the mine claimed.
	breakRitual := func(message string, status int) {
		if err := clearPointTransplantRitual(tx, playerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hủy nghi thức di dời"})
			return
		}
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu trạng thái nghi thức"})
			return
		}
		notify.Send(db.DB, playerID, "point_transplant", "Nghi Thức Tan", message)
		c.JSON(status, gin.H{"error": message})
	}
	if ritual != nil {
		var pointX, pointY int
		scanErr := tx.QueryRow(`SELECT x, y FROM aperture_resource_points WHERE id = $1`, ritual.PointID).Scan(&pointX, &pointY)
		if scanErr != nil && scanErr != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
			return
		}
		// Presence is the whole mechanic: stepping away (or the mine vanishing)
		// breaks the ritual, and the burnt essence stays gone.
		if scanErr == sql.ErrNoRows || !worldPointInReach(playerRegion, ritual.RegionID, playerX, playerY, pointX, pointY) {
			breakRitual("Nghi thức đứt đoạn vì đạo hữu đã rời khỏi mỏ — Tiên Nguyên tiêu hao không thể thu hồi.", http.StatusBadRequest)
			return
		}
		if ritual.PointID != pointID.String() {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Đạo hữu đang di dời mỏ khác (còn %s).", formatTransplantCountdown(ritual.CompletesAt.Sub(now)))})
			return
		}
		if now.Before(ritual.CompletesAt) {
			c.JSON(http.StatusConflict, gin.H{
				"error":               fmt.Sprintf("Nghi thức chưa xong — còn %s, hãy đứng yên tại mỏ.", formatTransplantCountdown(ritual.CompletesAt.Sub(now))),
				"code":                "TRANSPLANT_NOT_READY",
				"retry_after_seconds": int(math.Ceil(ritual.CompletesAt.Sub(now).Seconds())),
			})
			return
		}

		// Ritual due with the cultivator still present: uproot the mountain.
		var pointName, pointScope, pointRegion string
		err = tx.QueryRow(`
			SELECT name, COALESCE(scope, 'WORLD'), COALESCE(region_id, ''), x, y
			FROM aperture_resource_points WHERE id = $1 FOR UPDATE
		`, pointID).Scan(&pointName, &pointScope, &pointRegion, &pointX, &pointY)
		if err == sql.ErrNoRows {
			breakRitual("Mỏ đã biến mất khỏi thế giới — nghi thức tan.", http.StatusNotFound)
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
			return
		}
		if pointScope != "WORLD" || pointRegion != ritual.RegionID || !worldPointInReach(playerRegion, pointRegion, playerX, playerY, pointX, pointY) {
			breakRitual("Mỏ đã bị di dời hoặc đạo hữu đã rời khỏi nơi — nghi thức tan và Tiên Nguyên đã mất.", http.StatusConflict)
			return
		}
		apertureID, pointCount, err := loadPlayerAperturePointSlot(tx, playerID)
		if err == sql.ErrNoRows {
			breakRitual("Đạo hữu chưa khai mở Tiên Khiếu — không có đâu để đặt núi!", http.StatusBadRequest)
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra Tiên Khiếu"})
			return
		}
		if pointCount >= aperturePointPointGenCap {
			// Keep the ritual pending so the player can free a slot and retry;
			// the abandonment sweeper releases the mine if they never do.
			c.JSON(http.StatusConflict, gin.H{
				"error": fmt.Sprintf("Tiên Khiếu đã đủ %d điểm tài nguyên — hãy giải phóng một điểm rồi hoàn tất nghi thức.", aperturePointPointGenCap),
				"code":  "APERTURE_POINTS_FULL",
			})
			return
		}
		if err := moveWorldPointIntoAperture(tx, apertureID, pointID.String()); errors.Is(err, errWorldPointNotMovable) {
			breakRitual("Mỏ đã bị người khác di dời trước — nghi thức tan và Tiên Nguyên đã mất.", http.StatusConflict)
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi di dời điểm tài nguyên"})
			return
		}
		if err := clearPointTransplantRitual(tx, playerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kết thúc nghi thức di dời"})
			return
		}
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu di dời"})
			return
		}
		notify.Send(db.DB, playerID, "point_transplant", "Di Dời Núi Linh",
			fmt.Sprintf("Nghi thức hoàn tất: %s đã rời %s về Tiên Khiếu của đạo hữu! Vùng %s sẽ mọc lại mạch núi khác theo thời gian.", pointName, pointRegion, pointRegion))
		c.JSON(http.StatusOK, gin.H{
			"message":   fmt.Sprintf("Di dời %s vào Tiên Khiếu thành công!", pointName),
			"completed": true,
		})
		return
	}

	// No ritual in flight: validate the mine and start one.
	var pointName, pointGrade, pointScope string
	var pointVitality, pointX, pointY int
	var regionID string
	err = tx.QueryRow(`
		SELECT name, COALESCE(grade, 'LOW'), COALESCE(scope, 'WORLD'), vitality, COALESCE(region_id, ''), x, y
		FROM aperture_resource_points WHERE id = $1 FOR UPDATE
	`, pointID).Scan(&pointName, &pointGrade, &pointScope, &pointVitality, &regionID, &pointX, &pointY)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Điểm tài nguyên không tồn tại"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}
	if rank < 6 {
		c.JSON(http.StatusForbidden, gin.H{"error": "Đạo hữu chưa thăng tiên (Lục Chuyển) — chưa thể mở nghi thức di dời vào Tiên Khiếu."})
		return
	}
	if pointScope != "WORLD" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm này đã thuộc một Tiên Khiếu."})
		return
	}
	if pointVitality < 70 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sinh khí điểm tài nguyên dưới 70% — mạch núi suy yếu, không thể di dời. Hãy để nó hồi phục hoặc Bồi Dưỡng."})
		return
	}
	if !worldPointInReach(playerRegion, regionID, playerX, playerY, pointX, pointY) {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Phải đứng cùng vùng và cách mỏ tối đa %d ô mới bắt đầu được nghi thức di dời.", worldHarvestRange)})
		return
	}
	_, pointCount, err := loadPlayerAperturePointSlot(tx, playerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu — không có đâu để đặt núi!"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra Tiên Khiếu"})
		return
	}
	if pointCount >= aperturePointPointGenCap {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Khiếu đã có tối đa %d điểm tài nguyên.", aperturePointPointGenCap)})
		return
	}
	// Abandoned rituals on this mine are released first so they cannot block a
	// new claimant forever (the owner is notified once this transaction is in).
	releasedOwners, err := expirePointTransplants(tx, pointID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi dọn nghi thức di dời cũ"})
		return
	}
	// A mine already under a live ritual cannot be claimed twice.
	var claimed bool
	if err := tx.QueryRow(`SELECT EXISTS (SELECT 1 FROM aperture_point_transplants WHERE point_id = $1)`, pointID).Scan(&claimed); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi kiểm tra nghi thức di dời"})
		return
	}
	if claimed {
		c.JSON(http.StatusConflict, gin.H{"error": "Mỏ này đang được người khác làm nghi thức di dời."})
		return
	}
	cost := worldTransplantManaCost(pointGrade, rank)
	if currentMana < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên cho nghi thức: cần %s, hiện có %s.", formatTransplantEssence(cost), formatTransplantEssence(currentMana))})
		return
	}
	duration := worldTransplantDuration(pointGrade, rank)
	completesAt := now.Add(duration)
	// Atomic burn: the conditional UPDATE re-checks the balance under the row
	// lock, so a concurrent spender can never push the pool below zero.
	burn, err := tx.Exec(`UPDATE players SET current_mana = current_mana - $1 WHERE id = $2 AND current_mana >= $1`, cost, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên"})
		return
	}
	if affected, rowsErr := burn.RowsAffected(); rowsErr == nil && affected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Tiên Nguyên cho nghi thức: cần %s.", formatTransplantEssence(cost))})
		return
	}
	if _, err := tx.Exec(`
		INSERT INTO aperture_point_transplants (player_id, point_id, region_id, mana_spent, started_at, completes_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (player_id) DO UPDATE
		SET point_id = EXCLUDED.point_id, region_id = EXCLUDED.region_id,
		    mana_spent = EXCLUDED.mana_spent, started_at = EXCLUDED.started_at,
		    completes_at = EXCLUDED.completes_at, created_at = NOW()
	`, playerID, pointID, regionID, cost, now, completesAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi bắt đầu nghi thức di dời"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu nghi thức di dời"})
		return
	}
	notifyAbandonedTransplants(releasedOwners)
	c.JSON(http.StatusAccepted, gin.H{
		"message": fmt.Sprintf("Bắt đầu nghi thức di dời %s: tiêu hao %s Tiên Nguyên, đứng tại mỏ %s để hoàn tất. Rời đi sẽ hỏng nghi thức.",
			pointName, formatTransplantEssence(cost), formatTransplantCountdown(duration)),
		"started": true,
		"cost":    cost,
		"ritual": models.AperturePointTransplant{
			PointID:     pointID.String(),
			RegionID:    regionID,
			ManaSpent:   cost,
			StartedAt:   now,
			CompletesAt: completesAt,
		},
	})
}

// strongestBossCell finds the highest-rank Boss cell of a region.
func strongestBossCell(tx *sql.Tx, regionID string) (int, int, int, bool) {
	row, err := tx.Query(`
		SELECT x, y, COALESCE((metadata->>'rank')::int, 4) AS rank
		FROM map_cells
		WHERE region_id = $1 AND cell_type = 'Boss' AND is_active = true
		ORDER BY 3 DESC, random()
		LIMIT 1
	`, regionID)
	if err != nil {
		return 0, 0, 0, false
	}
	defer row.Close()
	if row.Next() {
		var x, y, bossRank int
		if row.Scan(&x, &y, &bossRank) == nil {
			return x, y, bossRank, true
		}
	}
	return 0, 0, 0, false
}

func canonWorldTemplatePool(famous bool) []aperturePointTemplateEntry {
	pool := make([]aperturePointTemplateEntry, 0, len(ResourcePointTemplates))
	for _, t := range ResourcePointTemplates {
		if t.PointType == "" {
			continue
		}
		if famous != (t.PointClass == "FAMOUS") {
			continue
		}
		pool = append(pool, aperturePointTemplateEntry{t: t, weight: 1})
	}
	return pool
}

func CollectAperturePointStock(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pointID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã điểm tài nguyên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var apertureID string
	err = tx.QueryRow(`
		SELECT p.aperture_id FROM aperture_resource_points p
		JOIN immortal_apertures a ON a.id = p.aperture_id
		WHERE p.id = $1 AND a.player_id = $2
		FOR UPDATE OF p
	`, pointID, playerID).Scan(&apertureID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Điểm tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}

	message, granted, err := harvestAperturePoint(tx, playerID, apertureID, pointID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if granted {
		var rank int
		if err := tx.QueryRow(`SELECT rank FROM players WHERE id=$1`, playerID).Scan(&rank); err == nil {
			if err := TrackDailyMissionProgress(tx, playerID, rank, "point-collect:"+pointID.String(), "aperture_collect", "", 1); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật nhiệm vụ hằng ngày"})
				return
			}
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu thu hoạch"})
		return
	}
	if granted {
		notify.Send(db.DB, playerID, "aperture_harvest", "Thu Hoạch Tiên Khiếu", message)
	}
	respondActionWithInventory(c, http.StatusOK, models.Player{Account: models.Account{ID: playerID}}, gin.H{
		"message": message,
	})
}

// harvestAperturePoint drains a point's stock inside a caller transaction.
// Returns the reward message and whether anything was actually granted.
func harvestAperturePoint(tx *sql.Tx, playerID uuid.UUID, apertureID string, pointID uuid.UUID) (string, bool, error) {
	var p models.ApertureResourcePoint
	var aperturePtr *string
	var stockJSON []byte
	err := tx.QueryRow(`
		SELECT p.id, p.aperture_id, p.point_type, p.name, p.path, p.grade, COALESCE(p.point_class, 'NORMAL'), p.level, p.vitality, p.stock, p.stock_cap, p.x, p.y, p.last_produced_at, p.state, COALESCE(p.scope, 'APERTURE')
		FROM aperture_resource_points p WHERE p.id = $1
	`, pointID).Scan(&p.ID, &aperturePtr, &p.PointType, &p.Name, &p.Path, &p.Grade, &p.PointClass, &p.Level, &p.Vitality, &stockJSON, &p.StockCap, &p.X, &p.Y, &p.LastProducedAt, &p.State, &p.Scope)
	if err != nil {
		return "", false, fmt.Errorf("Lỗi tải điểm tài nguyên")
	}
	p.ApertureID = aperturePtr
	_ = json.Unmarshal(stockJSON, &p.Stock)
	if _, ok := GetResourcePointTemplate(p.PointType); !ok {
		return "", false, fmt.Errorf("Loại điểm tài nguyên không hợp lệ")
	}
	return harvestPointRow(tx, playerID, &p)
}

// rollPointGuWorm hatches at most one wild Gu worm per harvest. Weighted tier
// roll: 1–3 phổ biến, 4–5 hiếm dần, 6 (Tiên Cổ) theo `immortalChance` (cực
// hiếm; rank ≥ 7 không sinh tự nhiên).
func rollPointGuWorm(tx *sql.Tx, playerID uuid.UUID, template ResourcePointTemplate, pt models.ApertureResourcePoint, immortalChance float64) string {
	if template.GuRate <= 0 {
		return ""
	}
	roll := rand.Float64()
	guChance := template.GuRate * (1 + 0.25*float64(pt.Level-1))
	if pt.Grade == "TOP" {
		guChance *= 1.5
	}
	if roll > guChance {
		return ""
	}
	// Rank tier roll within the allowed ceiling (≤ 6).
	pick := rand.Float64()
	guRank := 1
	switch {
	case pick < 0.45:
		guRank = 1
	case pick < 0.72:
		guRank = 2
	case pick < 0.88:
		guRank = 3
	case pick < 0.96:
		guRank = 4
	case pick < 0.995:
		guRank = 5
	default:
		if rand.Float64() < immortalChance {
			guRank = 6
		} else {
			guRank = 5
		}
	}
	candidates := models.GuCandidatesForPath(template.Path, guRank)
	if len(candidates) == 0 {
		if guRank > 1 {
			candidates = models.GuCandidatesForPath(template.Path, 1)
		}
		if len(candidates) == 0 {
			return ""
		}
		guRank = 1
	}
	guID := candidates[rand.Intn(len(candidates))]
	spec, ok := models.GetGu(guID, guRank)
	if !ok {
		return ""
	}
	newGu := models.GuInstance{InstanceID: uuid.New().String(), GuID: guID, Rank: spec.Rank, Hunger: 100, Status: "SLEEPING", Equipped: false, IsVital: false}
	if err := db.AddPlayerGu(tx, playerID.String(), newGu); err != nil {
		return ""
	}
	return fmt.Sprintf("%s (rank %d)", spec.Name, guRank)
}

// addHumanQi credits a small Nhân Khí reward for harvesting.
func addHumanQi(tx *sql.Tx, playerID uuid.UUID, amount int) {
	_, _ = tx.Exec(`UPDATE players SET human_qi = human_qi + $1 WHERE id = $2`, float64(amount), playerID)
}

// CollectAllAperturePointStocks drains every stocked point in one request.
func CollectAllAperturePointStocks(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var apertureID string
	err = tx.QueryRow(`
		SELECT id FROM immortal_apertures WHERE player_id = $1 FOR UPDATE
	`, playerID).Scan(&apertureID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Đạo hữu chưa khai mở Tiên Khiếu"})
		return
	}

	pointIDs := []uuid.UUID{}
	rows, err := tx.Query(`SELECT id FROM aperture_resource_points WHERE aperture_id = $1 AND state <> 'GONE' FOR UPDATE`, apertureID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải điểm tài nguyên"})
		return
	}
	for rows.Next() {
		var id uuid.UUID
		if rows.Scan(&id) == nil {
			pointIDs = append(pointIDs, id)
		}
	}
	rows.Close()

	messages := []string{}
	collected := 0
	for _, id := range pointIDs {
		message, granted, err := harvestAperturePoint(tx, playerID, apertureID, id)
		if err != nil {
			continue
		}
		if granted {
			collected++
			messages = append(messages, message)
		}
	}

	if collected > 0 {
		var rank int
		if err := tx.QueryRow(`SELECT rank FROM players WHERE id=$1`, playerID).Scan(&rank); err == nil {
			_ = TrackDailyMissionProgress(tx, playerID, rank, "point-collect-all:"+time.Now().UTC().Format("2006-01-02T15"), "aperture_collect", "", int64(collected))
		}
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu thu hoạch"})
		return
	}

	message := "Không có điểm tài nguyên nào sẵn sàng thu hoạch."
	if collected > 0 {
		message = fmt.Sprintf("Thu hoạch %d điểm tài nguyên:\n%s", collected, strings.Join(messages, "\n"))
		notify.Send(db.DB, playerID, "aperture_harvest", "Thu Hoạch Tiên Khiếu", message)
	}
	respondActionWithInventory(c, http.StatusOK, models.Player{Account: models.Account{ID: playerID}}, gin.H{
		"message": message,
		"count":   collected,
	})
}

// RevitalizeAperturePoint invests matching-path materials to restore vitality
// (canon: reviving a depleted resource point is an investment).
func RevitalizeAperturePoint(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pointID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã điểm tài nguyên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var pt models.ApertureResourcePoint
	err = tx.QueryRow(`
		SELECT p.id, p.point_type, p.name, p.level, p.vitality, p.state
		FROM aperture_resource_points p
		JOIN immortal_apertures a ON a.id = p.aperture_id
		WHERE p.id = $1 AND a.player_id = $2
		FOR UPDATE OF p
	`, pointID, playerID).Scan(&pt.ID, &pt.PointType, &pt.Name, &pt.Level, &pt.Vitality, &pt.State)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Điểm tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}
	if pt.Vitality >= 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Điểm tài nguyên đang đầy sinh khí."})
		return
	}

	template, ok := GetResourcePointTemplate(pt.PointType)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại điểm tài nguyên không hợp lệ"})
		return
	}
	itemID := fmt.Sprintf("%s_rank%d", template.ProduceKey, pointProduceRank(pt.Level, pt.Grade))
	unitsNeeded := int64(math.Ceil(float64(100-pt.Vitality) * 6.0))

	have, err := InventorySvc.GetQuantity(tx, playerID.String(), itemID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn túi đồ"})
		return
	}
	if have <= 0 {
		itemName := itemID
		if itemSpec, exists := models.ItemRegistry[itemID]; exists {
			itemName = itemSpec.Name
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Cần %s để bồi dưỡng %s (đang có 0).", itemName, pt.Name)})
		return
	}
	unitsPaid := min(unitsNeeded, int64(have))
	if err := InventorySvc.RemoveItem(tx, playerID.String(), itemID, "materials", unitsPaid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tiêu hao vật liệu bồi dưỡng"})
		return
	}
	restored := int(math.Max(1, math.Round(float64(unitsPaid)/6.0)))
	newVitality := pt.Vitality + restored
	if newVitality > 100 {
		newVitality = 100
	}
	newState := "ACTIVE"
	if newVitality < 30 {
		newState = pt.State
	} else if pt.State == "EXHAUSTED" {
		newState = "ACTIVE"
	}
	if _, err := tx.Exec(`
		UPDATE aperture_resource_points SET vitality = $1, state = $2, updated_at = NOW() WHERE id = $3
	`, newVitality, newState, pt.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu bồi dưỡng"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu bồi dưỡng"})
		return
	}
	respondActionWithInventory(c, http.StatusOK, models.Player{Account: models.Account{ID: playerID}}, gin.H{
		"message":  fmt.Sprintf("Bồi dưỡng %s thành công: sinh khí %d%%!", pt.Name, newVitality),
		"vitality": newVitality,
		"state":    newState,
	})
}

// UpgradeAperturePoint raises a point's level, lifting yield and stock cap.
func UpgradeAperturePoint(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pointID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã điểm tài nguyên không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var pt models.ApertureResourcePoint
	err = tx.QueryRow(`
		SELECT p.id, p.name, p.level FROM aperture_resource_points p
		JOIN immortal_apertures a ON a.id = p.aperture_id
		WHERE p.id = $1 AND a.player_id = $2
		FOR UPDATE OF p
	`, pointID, playerID).Scan(&pt.ID, &pt.Name, &pt.Level)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Điểm tài nguyên không tồn tại hoặc không thuộc đạo hữu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn điểm tài nguyên"})
		return
	}
	if pt.Level >= MaxPointLevel {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Đã đạt cấp tối đa (%d). Không thể nâng cấp thêm.", MaxPointLevel)})
		return
	}

	upgradeCost := int(float64(templateUpgradeBase) * float64(pt.Level) * 1.5)
	var stones int
	if err := tx.QueryRow(`SELECT FLOOR(immortal_stones)::BIGINT FROM players WHERE id = $1`, playerID).Scan(&stones); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy số dư tài khoản"})
		return
	}
	if stones < upgradeCost {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Tiên Nguyên Thạch không đủ để nâng cấp (Yêu cầu: %d, Hiện có: %d).", upgradeCost, stones)})
		return
	}
	var newStones float64
	if err := tx.QueryRow(`UPDATE players SET immortal_stones = immortal_stones - $1 WHERE id = $2 RETURNING immortal_stones`, upgradeCost, playerID).Scan(&newStones); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khấu trừ Tiên Nguyên Thạch"})
		return
	}
	newLevel := pt.Level + 1
	if _, err := tx.Exec(`UPDATE aperture_resource_points SET level = $1, updated_at = NOW() WHERE id = $2`, newLevel, pt.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi nâng cấp điểm tài nguyên"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu nâng cấp"})
		return
	}
	notify.Send(db.DB, playerID, "aperture_upgrade", "Nâng Cấp Điểm Tài Nguyên",
		fmt.Sprintf("Nâng cấp %s lên cấp %d thành công! Tiêu tốn %d Tiên Nguyên Thạch.", pt.Name, newLevel, upgradeCost))
	respondActionWithInventory(c, http.StatusOK, models.Player{Account: models.Account{ID: playerID}}, gin.H{
		"message":         fmt.Sprintf("Nâng cấp %s lên cấp %d thành công! Tiêu tốn %d Tiên Nguyên Thạch.", pt.Name, newLevel, upgradeCost),
		"new_level":       newLevel,
		"immortal_stones": newStones,
	})
}

const templateUpgradeBase = 30

// ---- worker assignment --------------------------------------------------

type AssignPointWorkerRequest struct {
	VariantHumanID     string `json:"variant_human_id" binding:"required"`
	AssignedPopulation int    `json:"assigned_population"`
}

// GetAperturePointWorkers lists the labour assigned to one resource point.
func GetAperturePointWorkers(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pointID := c.Param("id")
	pointUUID, err := uuid.Parse(pointID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã điểm tài nguyên không hợp lệ"})
		return
	}
	rows, err := db.DB.Query(`
		SELECT w.id, w.point_id, w.variant_human_id, v.race, v.population, w.assigned_population
		FROM aperture_point_workers w
		JOIN aperture_resource_points p ON p.id = w.point_id
		JOIN immortal_apertures a ON a.id = p.aperture_id
		JOIN aperture_variant_humans v ON v.id = w.variant_human_id
		WHERE w.point_id = $1 AND a.player_id = $2
	`, pointUUID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi tải phân công lao động"})
		return
	}
	defer rows.Close()
	list := []models.AperturePointWorkerView{}
	for rows.Next() {
		var w models.AperturePointWorkerView
		if rows.Scan(&w.ID, &w.PointID, &w.VariantHumanID, &w.Race, &w.Population, &w.AssignedPopulation) == nil {
			list = append(list, w)
		}
	}
	c.JSON(http.StatusOK, list)
}

// AssignWorkerToPoint assigns variant-human labour to a resource point so the
// assignment actually feeds the yield formula (+0.25 yield per 500 workers).
func AssignWorkerToPoint(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pointID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mã điểm tài nguyên không hợp lệ"})
		return
	}
	var req AssignPointWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu yêu cầu không hợp lệ"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	defer tx.Rollback()

	var apertureID string
	err = tx.QueryRow(`
		SELECT p.aperture_id FROM aperture_resource_points p
		JOIN immortal_apertures a ON a.id = p.aperture_id
		WHERE p.id = $1 AND a.player_id = $2
	`, pointID, playerID).Scan(&apertureID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy điểm tài nguyên"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}

	var vhRace string
	var vhPop int
	err = tx.QueryRow(`
		SELECT race, population FROM aperture_variant_humans
		WHERE id = $1 AND aperture_id = $2 FOR UPDATE
	`, req.VariantHumanID, apertureID).Scan(&vhRace, &vhPop)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy dị nhân"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}

	var assignedElsewhere int
	if err := tx.QueryRow(`
		SELECT COALESCE(SUM(assigned_population), 0)
		FROM (
			SELECT assigned_population FROM aperture_point_workers
			WHERE variant_human_id = $1 AND point_id <> $2
		) AS w
	`, req.VariantHumanID, pointID).Scan(&assignedElsewhere); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	if assignedElsewhere+req.AssignedPopulation > vhPop {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Số lượng lao động vượt quá dân số"})
		return
	}

	if _, err := tx.Exec(`
		INSERT INTO aperture_point_workers (point_id, variant_human_id, assigned_population)
		VALUES ($1, $2, $3)
		ON CONFLICT (point_id, variant_human_id)
		DO UPDATE SET assigned_population = $3, updated_at = NOW()
	`, pointID, req.VariantHumanID, req.AssignedPopulation); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lưu phân công"})
		return
	}
	notify.Send(db.DB, playerID, "worker_assigned", "⛏️ Phân Công Lao Động Dị Nhân", fmt.Sprintf(
		"Đã phân công %d %s vào điểm tài nguyên [%s] (+%d%% sản lượng).",
		req.AssignedPopulation, vhRace, pointID, workerBonusPercent(req.AssignedPopulation)))
	c.JSON(http.StatusOK, gin.H{"message": "Phân công lao động thành công"})
}

func workerBonusPercent(population int) int {
	return int((1+0.25*math.Min(1.0, float64(population)/500.0))*100) - 100
}

// RemoveWorkerFromPoint removes one labour assignment from a point.
func RemoveWorkerFromPoint(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	pointID := c.Param("id")
	workerRowID := c.Param("worker_id")

	result, err := db.DB.Exec(`
		DELETE FROM aperture_point_workers w
		USING aperture_resource_points p
		JOIN immortal_apertures a ON a.id = p.aperture_id
		WHERE w.point_id = p.id AND w.id = $1 AND p.id = $2 AND a.player_id = $3
	`, workerRowID, pointID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cơ sở dữ liệu"})
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không tìm thấy phân công lao động"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa phân công lao động"})
}
