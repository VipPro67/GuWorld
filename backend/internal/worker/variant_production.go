package worker

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartVariantProductionWorker starts the background worker that processes
// auto-production for variant humans (mining, farming, refinement).
// Tick interval: 10 minutes (processes aperture internal years in batches).
func StartVariantProductionWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	RunEvery(ctx, interval, "Variant Production Worker", func(ctx context.Context) {
		processVariantProductionTick(ctx, db)
	})
}

// --- package-level types (used by processApertureProduction) ---

type apertureTickData struct {
	apertureID  string
	grade       string
	timeRate    float64
	yearsPassed float64
	variants    []variantRow
	nodes       []nodeWorkerRow
	namedNPCs   []namedNPCWorker
}

type namedNPCWorker struct {
	role            string
	rank            int
	talent, loyalty float64
}

type variantRow struct {
	id             string
	race           string
	population     int
	laborIntensity string
	happiness      float64
}

type nodeWorkerRow struct {
	nodeID             string
	nodeType           string
	productionKey      string
	variantHumanID     string
	race               string
	assignedPopulation int
	nodeLevel          int
	productionAmount   int64
}

// --- main tick ---

func processVariantProductionTick(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, `
		SELECT a.id, a.grade, a.time_rate, a.internal_years_passed
		FROM immortal_apertures a
		WHERE a.state != 'DESTROYED'
		ORDER BY a.id
	`)
	if err != nil {
		logger.S.Error("VariantProduction: failed to query apertures: ", err)
		return
	}
	defer rows.Close()

	var apertures []apertureTickData
	for rows.Next() {
		var d apertureTickData
		if err := rows.Scan(&d.apertureID, &d.grade, &d.timeRate, &d.yearsPassed); err != nil {
			logger.S.Error("VariantProduction: scan aperture: ", err)
			continue
		}
		apertures = append(apertures, d)
	}
	if err := rows.Err(); err != nil {
		logger.S.Error("VariantProduction: rows error: ", err)
		return
	}
	if len(apertures) == 0 {
		return
	}

	apertureIDs := make([]string, len(apertures))
	for i, a := range apertures {
		apertureIDs[i] = a.apertureID
	}

	variantMap, err := loadVariantHumansBatch(ctx, db, apertureIDs)
	if err != nil {
		logger.S.Error("VariantProduction: load variants: ", err)
		return
	}

	workerMap, err := loadNodeWorkersBatch(ctx, db, apertureIDs)
	if err != nil {
		logger.S.Error("VariantProduction: load workers: ", err)
		return
	}
	namedNPCMap, err := loadNamedNPCWorkersBatch(ctx, db, apertureIDs)
	if err != nil {
		logger.S.Error("VariantProduction: load named NPCs: ", err)
		return
	}

	for i := range apertures {
		apertures[i].variants = variantMap[apertures[i].apertureID]
		apertures[i].nodes = workerMap[apertures[i].apertureID]
		apertures[i].namedNPCs = namedNPCMap[apertures[i].apertureID]
		processApertureProduction(ctx, db, &apertures[i])
		processApertureRefinement(ctx, db, apertures[i].apertureID)
	}
}

func loadNamedNPCWorkersBatch(ctx context.Context, db *sql.DB, apertureIDs []string) (map[string][]namedNPCWorker, error) {
	rows, err := db.QueryContext(ctx, `SELECT aperture_id,role,rank,talent,loyalty FROM non_player_actors
		WHERE aperture_id=ANY($1::uuid[]) AND actor_kind='VARIANT_NPC' AND lifecycle_state='ACTIVE' AND execution_state='WORKING'`, pq.Array(apertureIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string][]namedNPCWorker)
	for rows.Next() {
		var id string
		var n namedNPCWorker
		if err := rows.Scan(&id, &n.role, &n.rank, &n.talent, &n.loyalty); err != nil {
			return nil, err
		}
		result[id] = append(result[id], n)
	}
	return result, rows.Err()
}

func clampNPCPercent(v float64) float64 { return math.Max(0, math.Min(100, v)) / 100 }
func namedNPCBonus(npcs []namedNPCWorker, role string, cap float64) float64 {
	bonus := 0.0
	for _, n := range npcs {
		if n.role == role {
			bonus += .005 * float64(n.rank) * clampNPCPercent(n.talent) * clampNPCPercent(n.loyalty)
		}
	}
	return math.Min(cap, bonus)
}
func namedNPCNodeRole(nodeType, productionKey string) string {
	v := nodeType + " " + productionKey
	if strings.Contains(strings.ToUpper(v), "FARM") || strings.Contains(strings.ToUpper(v), "BREED") || strings.Contains(strings.ToUpper(v), "FOOD") {
		return "FARMER"
	}
	return "MINER"
}

// --- batch loaders ---

func loadVariantHumansBatch(ctx context.Context, db *sql.DB, apertureIDs []string) (map[string][]variantRow, error) {
	if len(apertureIDs) == 0 {
		return nil, nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, aperture_id, race, population, labor_intensity, happiness
		FROM aperture_variant_humans
		WHERE aperture_id = ANY($1::uuid[])
	`, pq.Array(apertureIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]variantRow)
	for rows.Next() {
		var r variantRow
		var apID string
		if err := rows.Scan(&r.id, &apID, &r.race, &r.population, &r.laborIntensity, &r.happiness); err != nil {
			continue
		}
		result[apID] = append(result[apID], r)
	}
	return result, rows.Err()
}

func loadNodeWorkersBatch(ctx context.Context, db *sql.DB, apertureIDs []string) (map[string][]nodeWorkerRow, error) {
	if len(apertureIDs) == 0 {
		return nil, nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT n.aperture_id, n.id, n.node_type, COALESCE(n.production_key, ''),
		       aw.variant_human_id, vh.race, aw.assigned_population,
		       n.level, n.production_amount
		FROM aperture_resource_nodes n
		JOIN aperture_node_workers aw ON aw.node_id = n.id
		JOIN aperture_variant_humans vh ON vh.id = aw.variant_human_id
		WHERE n.aperture_id = ANY($1::uuid[])
		  AND n.is_destroyed = false
		  AND n.node_category = 'RESOURCE'
	`, pq.Array(apertureIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]nodeWorkerRow)
	for rows.Next() {
		var r nodeWorkerRow
		var apID string
		if err := rows.Scan(&apID, &r.nodeID, &r.nodeType, &r.productionKey,
			&r.variantHumanID, &r.race, &r.assignedPopulation,
			&r.nodeLevel, &r.productionAmount); err != nil {
			continue
		}
		result[apID] = append(result[apID], r)
	}
	return result, rows.Err()
}

// --- per-aperture production ---

func processApertureProduction(ctx context.Context, db *sql.DB, ap *apertureTickData) {
	if len(ap.nodes) == 0 {
		return
	}

	variantByRace := make(map[string]*variantRow)
	for i := range ap.variants {
		variantByRace[ap.variants[i].race] = &ap.variants[i]
	}

	for _, nw := range ap.nodes {
		vh, ok := variantByRace[nw.race]
		if !ok || vh.population == 0 || nw.assignedPopulation <= 0 {
			continue
		}

		laborMod := laborIntensityModifier(vh.laborIntensity)
		baseOutput := float64(nw.productionAmount)
		popBonus := 1.0 + float64(vh.population)/1000.0
		if popBonus > 5.0 {
			popBonus = 5.0
		}
		levelBonus := 1.0 + float64(nw.nodeLevel-1)*0.1
		npcBonus := namedNPCBonus(ap.namedNPCs, namedNPCNodeRole(nw.nodeType, nw.productionKey), .50)
		output := int64(baseOutput * popBonus * levelBonus * laborMod * (1 + npcBonus))
		if output <= 0 {
			continue
		}

		_, err := db.ExecContext(ctx, `
			UPDATE immortal_apertures
			SET immortal_essence = immortal_essence + $2, updated_at = NOW()
			WHERE id = $1
		`, ap.apertureID, output)
		if err != nil {
			logger.S.Errorf("VariantProduction: update aperture %s: %v", ap.apertureID, err)
		}
	}
}

func laborIntensityModifier(intensity string) float64 {
	switch intensity {
	case "LOW":
		return 0.7
	case "HIGH":
		return 1.5
	case "EXPLOIT":
		return 2.0
	default:
		return 1.0
	}
}
