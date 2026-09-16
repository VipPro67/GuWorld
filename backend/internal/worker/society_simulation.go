package worker

import (
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartSocietySimulationWorker starts the background worker that simulates
// society dynamics for variant humans: food consumption, population growth,
// prosperity, and rebellion checks.
// Tick interval: 5 minutes.
func StartSocietySimulationWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Info("Starting Society Simulation Worker with tick interval of ", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Society Simulation Worker stopped")
				return
			case <-ticker.C:
				processSocietyTick(ctx, db)
			}
		}
	}()
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
}

func processSocietyTick(ctx context.Context, db *sql.DB) {
	// Load all active variant humans.
	rows, err := db.QueryContext(ctx, `
		SELECT id, aperture_id, race, population, happiness, oppression, rebellion_rate,
		       labor_intensity, prosperity, food_stored, food_per_capita,
		       population_growth_rate, mortality_rate
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
			&v.foodPerCapita, &v.populationGrowth, &v.mortalityRate); err != nil {
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

	// Batch-load aperture inventory (immortal_essence as proxy for food for now).
	essenceMap, err := loadApertureEssence(ctx, db, apertureIDs)
	if err != nil {
		logger.S.Error("SocietySim: load essence: ", err)
		return
	}

	capacityMap, err := loadSettlementCapacities(ctx, db, apertureIDs)
	if err != nil {
		logger.S.Error("SocietySim: load capacities: ", err)
		return
	}

	// Process each aperture's society.
	var updates []societyUpdate

	for apID, apVariants := range byAperture {
		essence := essenceMap[apID]

		for _, v := range apVariants {
			cap := 1000
			if apCaps, ok := capacityMap[apID]; ok {
				if raceCap, ok := apCaps[v.race]; ok && raceCap > 0 {
					cap = raceCap
				}
			}
			u := simulateRace(v, essence, cap)
			u.id = v.id
			updates = append(updates, u)
		}
	}

	// Batch update.
	if len(updates) > 0 {
		batchUpdateSociety(ctx, db, updates)
	}
}

func loadApertureEssence(ctx context.Context, db *sql.DB, apertureIDs []string) (map[string]int64, error) {
	if len(apertureIDs) == 0 {
		return nil, nil
	}
	rows, err := db.QueryContext(ctx, `
		SELECT id, immortal_essence FROM immortal_apertures WHERE id = ANY($1::uuid[])
	`, pq.Array(apertureIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var id string
		var essence int64
		if err := rows.Scan(&id, &essence); err != nil {
			continue
		}
		result[id] = essence
	}
	return result, rows.Err()
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
// Returns the new state to update.
func simulateRace(v societyVariant, apertureEssence int64, capacity int) societyUpdate {
	pop := float64(v.population)
	if pop <= 0 {
		return societyUpdate{}
	}

	// --- Food consumption ---
	foodNeeded := pop * v.foodPerCapita
	foodAvailable := v.foodStored
	if foodAvailable < foodNeeded {
		// Food shortage: consume from aperture essence.
		shortage := foodNeeded - foodAvailable
		if float64(apertureEssence) >= shortage {
			foodAvailable = 0
		} else {
			foodAvailable += float64(apertureEssence)
			shortage -= float64(apertureEssence)
		}
	}

	newFoodStored := foodAvailable - foodNeeded
	if newFoodStored < 0 {
		newFoodStored = 0
	}

	// --- Happiness ---
	newHappiness := v.happiness

	// Base: drift toward 50.
	newHappiness += (50.0 - newHappiness) * 0.01

	// Food effect.
	if newFoodStored <= 0 {
		newHappiness -= 3.0 // food shortage
	} else if newFoodStored > pop*v.foodPerCapita*10 {
		newHappiness += 0.5 // surplus bonus
	}

	// Labor intensity effect.
	switch v.laborIntensity {
	case "LOW":
		newHappiness += 1.0
	case "HIGH":
		newHappiness -= 2.0
		newHappiness = math.Max(newHappiness, 0)
	case "EXPLOIT":
		newHappiness -= 5.0
		newHappiness = math.Max(newHappiness, 0)
	}

	// Oppression effect.
	newHappiness -= v.oppression * 0.1

	// Clamp happiness [0, 100].
	newHappiness = math.Max(0, math.Min(100, newHappiness))

	// --- Oppression ---
	newOppression := v.oppression
	// Slow natural decay.
	newOppression *= 0.99
	// EXPLOIT increases oppression.
	if v.laborIntensity == "EXPLOIT" {
		newOppression += 1.0
	}
	newOppression = math.Max(0, math.Min(100, newOppression))

	// --- Rebellion rate ---
	newRebellion := v.rebellionRate
	newRebellion *= 0.98 // natural decay
	if newHappiness < 30 {
		newRebellion += 2.0
	}
	if newOppression > 50 {
		newRebellion += 1.0
	}
	if v.laborIntensity == "EXPLOIT" {
		newRebellion += 1.5
	}
	newRebellion = math.Max(0, math.Min(100, newRebellion))

	// --- Population growth / mortality ---
	growthRate := v.populationGrowth
	mortality := v.mortalityRate

	// Prosperity modifier.
	growthMod := 1.0 + (v.prosperity-50.0)/200.0 // prosperity 100 → 1.25x, 0 → 0.75x
	growthRate *= growthMod

	// Food shortage increases mortality.
	if newFoodStored <= 0 {
		mortality *= 3.0
	}

	// Low happiness increases mortality.
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
	newProsperity := v.prosperity
	// Drift toward 50.
	newProsperity += (50.0 - newProsperity) * 0.005
	// Food surplus increases prosperity.
	if newFoodStored > pop*v.foodPerCapita*5 {
		newProsperity += 0.5
	}
	// Low happiness decreases prosperity.
	if newHappiness < 30 {
		newProsperity -= 1.0
	}
	// High rebellion decreases prosperity.
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
	}
}

type societyUpdate struct {
	id            string
	population    int
	happiness     float64
	oppression    float64
	rebellionRate float64
	prosperity    float64
	foodStored    float64
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

	for i, u := range updates {
		ids[i] = u.id
		pops[i] = u.population
		happiness[i] = u.happiness
		oppression[i] = u.oppression
		rebellion[i] = u.rebellionRate
		prosperity[i] = u.prosperity
		foodStored[i] = u.foodStored
	}

	_, err := db.ExecContext(ctx, `
		UPDATE aperture_variant_humans AS v SET
			population = x.population,
			happiness = x.happiness,
			oppression = x.oppression,
			rebellion_rate = x.rebellion_rate,
			prosperity = x.prosperity,
			food_stored = x.food_stored,
			updated_at = NOW()
		FROM (
			SELECT
				unnest($1::uuid[]) AS id,
				unnest($2::int[]) AS population,
				unnest($3::double precision[]) AS happiness,
				unnest($4::double precision[]) AS oppression,
				unnest($5::double precision[]) AS rebellion_rate,
				unnest($6::double precision[]) AS prosperity,
				unnest($7::double precision[]) AS food_stored
		) x
		WHERE v.id = x.id
	`, pq.Array(ids), pq.Array(pops), pq.Array(happiness),
		pq.Array(oppression), pq.Array(rebellion), pq.Array(prosperity), pq.Array(foodStored))
	if err != nil {
		logger.S.Error("SocietySim: batch update failed: ", err)
	}
}
