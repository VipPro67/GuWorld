package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartSocietySimulationWorker starts the background worker that simulates
// society dynamics for variant humans: food consumption, population growth,
// prosperity, and rebellion checks.
// Tick interval: 5 minutes.
func StartSocietySimulationWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	RunEvery(ctx, interval, "Society Simulation Worker", func(ctx context.Context) {
		processSocietyTick(ctx, db)
	})
}

type societyVariant struct {
	id                string
	apertureID        string
	race              string
	population        int
	happiness         float64
	oppression        float64
	rebellionRate     float64
	laborIntensity    string
	prosperity        float64
	foodStored        float64
	foodPerCapita     float64
	populationGrowth  float64
	mortalityRate     float64
	autoEssenceRelief bool
}

var raceHabitatRequirement = map[string]struct {
	path      string
	threshold int64
	reproMult float64
}{
	"HAIRY_MAN":   {path: "refinement", threshold: 1000, reproMult: 1.2},
	"ROCKMAN":     {path: "earth", threshold: 1000, reproMult: 1.0},
	"FEATHERMAN":  {path: "wind", threshold: 800, reproMult: 1.0},
	"SNOWMAN":     {path: "ice", threshold: 1200, reproMult: 0.7},
	"INKMAN":      {path: "wisdom", threshold: 1000, reproMult: 0.7},
	"MINIMAN":     {path: "wood", threshold: 800, reproMult: 1.5},
	"MERMAN":      {path: "water", threshold: 1000, reproMult: 1.0},
	"MUSHROOMMAN": {path: "poison", threshold: 800, reproMult: 1.2},
	"DRAGONMAN":   {path: "transformation", threshold: 1200, reproMult: 0.7},
}

func processSocietyTick(ctx context.Context, db *sql.DB) {
	// Load all active variant humans.
	rows, err := db.QueryContext(ctx, `
		SELECT id, aperture_id, race, population, happiness, oppression, rebellion_rate,
		       labor_intensity, prosperity, food_stored, food_per_capita,
		       population_growth_rate, mortality_rate, COALESCE(auto_essence_relief, false)
		FROM aperture_variant_humans
		WHERE population > 0
		ORDER BY aperture_id, race
	`)
	if err != nil {
		logger.S.Error("SocietySim: failed to query variant humans: ", err)
		return
	}
	defer rows.Close()

	var variants []societyVariant
	for rows.Next() {
		var v societyVariant
		if err := rows.Scan(&v.id, &v.apertureID, &v.race, &v.population,
			&v.happiness, &v.oppression, &v.rebellionRate,
			&v.laborIntensity, &v.prosperity, &v.foodStored,
			&v.foodPerCapita, &v.populationGrowth, &v.mortalityRate, &v.autoEssenceRelief); err != nil {
			continue
		}
		variants = append(variants, v)
	}
	if err := rows.Err(); err != nil {
		logger.S.Error("SocietySim: rows error: ", err)
		return
	}
	if len(variants) == 0 {
		return
	}

	// Group by aperture.
	byAperture := make(map[string][]societyVariant)
	for _, v := range variants {
		byAperture[v.apertureID] = append(byAperture[v.apertureID], v)
	}

	apertureIDs := make([]string, 0, len(byAperture))
	for id := range byAperture {
		apertureIDs = append(apertureIDs, id)
	}

	essenceMap, daoMarksMap, err := loadApertureSocietyContext(ctx, db, apertureIDs)
	if err != nil {
		logger.S.Error("SocietySim: load aperture context: ", err)
		return
	}

	capacityMap, err := loadSettlementCapacities(ctx, db, apertureIDs)
	if err != nil {
		logger.S.Error("SocietySim: load capacities: ", err)
		return
	}

	stewardMap, immortalStewardMap := loadStewardCounts(ctx, db, apertureIDs)
	activeEventMap := loadUnresolvedEventRaces(ctx, db, apertureIDs)

	// Process each aperture's society.
	var updates []societyUpdate
	essenceDeductions := make(map[string]int64)

	for apID, apVariants := range byAperture {
		essence := essenceMap[apID]
		apDaoMarks := daoMarksMap[apID]

		for _, v := range apVariants {
			cap := 1000
			if apCaps, ok := capacityMap[apID]; ok {
				if raceCap, ok := apCaps[v.race]; ok && raceCap > 0 {
					cap = raceCap
				}
			}
			habitatOK := checkRaceHabitat(v.race, apDaoMarks)
			stewards := stewardMap[apID][v.race]
			immStewards := immortalStewardMap[apID]
			u, essenceUsed := simulateRace(v, essence, cap, habitatOK, stewards)
			u.id = v.id
			u.apertureID = apID
			u.race = v.race
			if essenceUsed > 0 {
				essence -= essenceUsed
				essenceDeductions[apID] += essenceUsed
			}
			updates = append(updates, u)

			// Trigger or auto-resolve rebellion events.
			hasActive := activeEventMap[apID][v.race]
			if hasActive && immStewards > 0 {
				// Loyal Immortal Steward auto-pacifies minor STRIKE events.
				_, _ = db.ExecContext(ctx, `
					UPDATE aperture_variant_events
					SET resolved = true, resolved_at = NOW()
					WHERE aperture_id = $1 AND race = $2 AND event_type = 'STRIKE' AND resolved = false
				`, apID, v.race)
			} else if !hasActive {
				triggerRebellionEventIfNeeded(ctx, db, apID, v.race, u.rebellionRate)
			}
		}
	}

	// Deduct actually consumed immortal_essence from relief.
	for apID, spent := range essenceDeductions {
		if spent > 0 {
			_, _ = db.ExecContext(ctx, `
				UPDATE immortal_apertures
				SET immortal_essence = GREATEST(0, immortal_essence - $1), updated_at = NOW()
				WHERE id = $2
			`, spent, apID)
		}
	}

	// Batch update.
	if len(updates) > 0 {
		batchUpdateSociety(ctx, db, updates)
	}
}

func checkRaceHabitat(race string, daoMarks map[string]int64) bool {
	req, ok := raceHabitatRequirement[race]
	if !ok {
		return true
	}
	for k, v := range daoMarks {
		if strings.EqualFold(strings.TrimSuffix(strings.ToLower(k), " path"), req.path) || strings.EqualFold(k, req.path) {
			if v >= req.threshold {
				return true
			}
		}
	}
	return false
}

func loadApertureSocietyContext(ctx context.Context, db *sql.DB, apertureIDs []string) (map[string]int64, map[string]map[string]int64, error) {
	if len(apertureIDs) == 0 {
		return nil, nil, nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, immortal_essence, COALESCE(dao_marks, '{}'::jsonb)
		FROM immortal_apertures WHERE id = ANY($1::uuid[])
	`, pq.Array(apertureIDs))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	essenceMap := make(map[string]int64)
	daoMap := make(map[string]map[string]int64)
	for rows.Next() {
		var id string
		var essence int64
		var daoRaw []byte
		if err := rows.Scan(&id, &essence, &daoRaw); err != nil {
			continue
		}
		essenceMap[id] = essence
		dm := map[string]int64{}
		_ = json.Unmarshal(daoRaw, &dm)
		daoMap[id] = dm
	}
	return essenceMap, daoMap, rows.Err()
}

func loadStewardCounts(ctx context.Context, db *sql.DB, apertureIDs []string) (map[string]map[string]int, map[string]int) {
	perRace := make(map[string]map[string]int)
	immortals := make(map[string]int)
	rows, err := db.QueryContext(ctx, `
		SELECT aperture_id, COALESCE(race, ''), rank
		FROM non_player_actors
		WHERE aperture_id = ANY($1::uuid[]) AND actor_kind = 'VARIANT_NPC'
		  AND lifecycle_state = 'ACTIVE' AND role = 'STEWARD' AND loyalty >= 50
	`, pq.Array(apertureIDs))
	if err != nil {
		return perRace, immortals
	}
	defer rows.Close()
	for rows.Next() {
		var apID, race string
		var rank int
		if rows.Scan(&apID, &race, &rank) == nil {
			if perRace[apID] == nil {
				perRace[apID] = make(map[string]int)
			}
			perRace[apID][race]++
			if rank >= 6 {
				immortals[apID]++
			}
		}
	}
	return perRace, immortals
}

func loadUnresolvedEventRaces(ctx context.Context, db *sql.DB, apertureIDs []string) map[string]map[string]bool {
	res := make(map[string]map[string]bool)
	rows, err := db.QueryContext(ctx, `
		SELECT aperture_id, COALESCE(race, '')
		FROM aperture_variant_events
		WHERE aperture_id = ANY($1::uuid[]) AND resolved = false
	`, pq.Array(apertureIDs))
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var apID, race string
		if rows.Scan(&apID, &race) == nil {
			if res[apID] == nil {
				res[apID] = make(map[string]bool)
			}
			res[apID][race] = true
		}
	}
	return res
}

func triggerRebellionEventIfNeeded(ctx context.Context, db *sql.DB, apertureID, race string, rebellionRate float64) {
	var eventType, severity, desc string
	switch {
	case rebellionRate >= 85:
		eventType = "REBELLION"
		severity = "CRITICAL"
		desc = fmt.Sprintf("Bộ tộc %s bạo loạn dữ dội do bất ổn kéo dài! Tiên Khiếu chịu tổn hại và dân số suy giảm.", race)
		_, _ = db.ExecContext(ctx, `
			UPDATE immortal_apertures
			SET aperture_hp = GREATEST(0, aperture_hp - max_aperture_hp * 0.03), updated_at = NOW()
			WHERE id = $1
		`, apertureID)
		_, _ = db.ExecContext(ctx, `
			UPDATE aperture_variant_humans
			SET population = GREATEST(0, FLOOR(population * 0.9)::int)
			WHERE aperture_id = $1 AND race = $2
		`, apertureID, race)
	case rebellionRate >= 65:
		eventType = "SABOTAGE"
		severity = "HIGH"
		desc = fmt.Sprintf("Bộ tộc %s phá hoại điểm tài nguyên trong Tiên Khiếu, làm tụt sinh khí mỏ!", race)
		_, _ = db.ExecContext(ctx, `
			UPDATE aperture_resource_points
			SET vitality = GREATEST(0, vitality / 2),
			    state = CASE WHEN vitality / 2 <= 0 THEN 'EXHAUSTED' ELSE state END,
			    updated_at = NOW()
			WHERE id IN (
				SELECT id FROM aperture_resource_points
				WHERE aperture_id = $1 AND vitality > 0
				ORDER BY random() LIMIT 1
			)
		`, apertureID)
	case rebellionRate >= 40:
		eventType = "STRIKE"
		severity = "MEDIUM"
		desc = fmt.Sprintf("Bộ tộc %s đình công phản đối điều kiện lao động! Sản lượng điểm tài nguyên và luyện cổ bị đình trệ.", race)
	default:
		return
	}

	_, _ = db.ExecContext(ctx, `
		INSERT INTO aperture_variant_events (aperture_id, event_type, race, severity, description, resolved)
		VALUES ($1, $2, $3, $4, $5, false)
	`, apertureID, eventType, race, severity, desc)
}

func loadSettlementCapacities(ctx context.Context, db *sql.DB, apertureIDs []string) (map[string]map[string]int, error) {
	if len(apertureIDs) == 0 {
		return nil, nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT aperture_id, race, SUM(population_capacity)
		FROM aperture_variant_settlements
		WHERE aperture_id = ANY($1::uuid[])
		GROUP BY aperture_id, race
	`, pq.Array(apertureIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]map[string]int)
	for rows.Next() {
		var apID, race string
		var cap int
		if err := rows.Scan(&apID, &race, &cap); err != nil {
			continue
		}
		if result[apID] == nil {
			result[apID] = make(map[string]int)
		}
		result[apID][race] = cap
	}
	return result, rows.Err()
}

// simulateRace runs one society tick for a single race in one aperture.
// Returns the new state to update and any immortal_essence actually consumed for relief.
func simulateRace(v societyVariant, apertureEssence int64, capacity int, habitatOK bool, stewards int) (societyUpdate, int64) {
	pop := float64(v.population)
	if pop <= 0 {
		return societyUpdate{}, 0
	}

	foodIntensityMult := 1.0
	if v.laborIntensity == "HIGH" {
		foodIntensityMult = 1.15
	} else if v.laborIntensity == "EXPLOIT" {
		foodIntensityMult = 1.30
	}

	// --- Food consumption ---
	foodNeeded := pop * v.foodPerCapita * foodIntensityMult
	newFoodStored := v.foodStored - foodNeeded
	starving := false
	var essenceSpent int64
	if newFoodStored < 0 {
		shortage := -newFoodStored
		newFoodStored = 0
		if v.autoEssenceRelief && apertureEssence > 0 {
			neededEssence := int64(math.Ceil(shortage / 100.0))
			if neededEssence < 1 {
				neededEssence = 1
			}
			if apertureEssence >= neededEssence {
				essenceSpent = neededEssence
			} else {
				essenceSpent = apertureEssence
				starving = true
			}
		} else {
			starving = true
		}
	}

	// --- Happiness ---
	newHappiness := v.happiness
	newHappiness += (50.0 - newHappiness) * 0.01

	if starving {
		newHappiness -= 4.0
	} else if newFoodStored > pop*v.foodPerCapita*10 {
		newHappiness += 0.5
	}
	if !habitatOK {
		newHappiness -= 0.5
	}

	switch v.laborIntensity {
	case "LOW":
		newHappiness += 1.0
	case "HIGH":
		loss := 2.0
		if v.race == "ROCKMAN" {
			loss *= 0.7 // Rockman endurance
		}
		newHappiness = math.Max(newHappiness-loss, 0)
	case "EXPLOIT":
		newHappiness = math.Max(newHappiness-5.0, 0)
	}

	newHappiness -= v.oppression * 0.1
	newHappiness = math.Max(0, math.Min(100, newHappiness))

	// --- Oppression ---
	newOppression := v.oppression * 0.99
	if v.laborIntensity == "EXPLOIT" {
		newOppression += 1.5
	}
	if stewards > 0 {
		newOppression -= 0.8 * float64(stewards)
	}
	newOppression = math.Max(0, math.Min(100, newOppression))

	// --- Rebellion rate ---
	newRebellion := v.rebellionRate * 0.98
	if starving {
		newRebellion += 3.0
	}
	if newHappiness < 30 {
		newRebellion += 2.0
	}
	if newOppression > 50 {
		newRebellion += 1.0
	}
	if v.laborIntensity == "EXPLOIT" {
		newRebellion += 1.5
	}
	if stewards > 0 {
		newRebellion -= 1.0 * float64(stewards)
	}
	newRebellion = math.Max(0, math.Min(100, newRebellion))

	// --- Population growth / mortality ---
	growthRate := v.populationGrowth
	if req, ok := raceHabitatRequirement[v.race]; ok {
		growthRate *= req.reproMult
	}
	if !habitatOK {
		// Missing Dao Mark habitat freezes natural reproduction.
		growthRate = 0
	} else {
		growthRate *= 1.25
	}
	mortality := v.mortalityRate

	growthMod := 1.0 + (v.prosperity-50.0)/200.0
	growthRate *= growthMod

	if starving {
		mortality *= 3.0
	}
	if newHappiness < 20 {
		mortality *= 2.0
	}

	births := int(math.Round(pop * growthRate))
	deaths := int(math.Round(pop * mortality))
	newPop := v.population + births - deaths
	if capacity > 0 && newPop > capacity {
		newPop = capacity
	}
	if newPop < 0 {
		newPop = 0
	}

	// --- Prosperity ---
	newProsperity := v.prosperity + (50.0-v.prosperity)*0.005
	if !starving && newFoodStored > pop*v.foodPerCapita*5 {
		newProsperity += 0.5
	}
	if newHappiness < 30 {
		newProsperity -= 1.0
	}
	if newRebellion > 50 {
		newProsperity -= 1.5
	}
	newProsperity = math.Max(0, math.Min(100, newProsperity))

	return societyUpdate{
		population:    newPop,
		happiness:     newHappiness,
		oppression:    newOppression,
		rebellionRate: newRebellion,
		prosperity:    newProsperity,
		foodStored:    newFoodStored,
		starving:      starving,
	}, essenceSpent
}

type societyUpdate struct {
	id            string
	apertureID    string
	race          string
	population    int
	happiness     float64
	oppression    float64
	rebellionRate float64
	prosperity    float64
	foodStored    float64
	starving      bool
}

func batchUpdateSociety(ctx context.Context, db *sql.DB, updates []societyUpdate) {
	if len(updates) == 0 {
		return
	}

	ids := make([]string, len(updates))
	pops := make([]int, len(updates))
	happiness := make([]float64, len(updates))
	oppression := make([]float64, len(updates))
	rebellion := make([]float64, len(updates))
	prosperity := make([]float64, len(updates))
	foodStored := make([]float64, len(updates))
	starving := make([]bool, len(updates))

	for i, u := range updates {
		ids[i] = u.id
		pops[i] = u.population
		happiness[i] = u.happiness
		oppression[i] = u.oppression
		rebellion[i] = u.rebellionRate
		prosperity[i] = u.prosperity
		foodStored[i] = u.foodStored
		starving[i] = u.starving
	}

	_, err := db.ExecContext(ctx, `
		UPDATE aperture_variant_humans AS v SET
			population = x.population,
			happiness = x.happiness,
			oppression = x.oppression,
			rebellion_rate = x.rebellion_rate,
			prosperity = x.prosperity,
			food_stored = x.food_stored,
			starving = x.starving,
			updated_at = NOW()
		FROM (
			SELECT
				unnest($1::uuid[]) AS id,
				unnest($2::int[]) AS population,
				unnest($3::double precision[]) AS happiness,
				unnest($4::double precision[]) AS oppression,
				unnest($5::double precision[]) AS rebellion_rate,
				unnest($6::double precision[]) AS prosperity,
				unnest($7::double precision[]) AS food_stored,
				unnest($8::boolean[]) AS starving
		) x
		WHERE v.id = x.id
	`, pq.Array(ids), pq.Array(pops), pq.Array(happiness),
		pq.Array(oppression), pq.Array(rebellion), pq.Array(prosperity), pq.Array(foodStored), pq.Array(starving))
	if err != nil {
		logger.S.Error("SocietySim: batch update failed: ", err)
	}
}
