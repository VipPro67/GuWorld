package worker

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// StartGuMasterSpawningWorker starts the background worker that checks
// variant human populations and spawns Gu Master NPCs when thresholds are met.
// Tick interval: 30 minutes.
func StartGuMasterSpawningWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Info("Starting Gu Master Spawning Worker with tick interval of ", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Gu Master Spawning Worker stopped")
				return
			case <-ticker.C:
				processGuMasterSpawning(ctx, db)
			}
		}
	}()
}

// Spawning thresholds per race.
const (
	spawnThresholdPop     = 10000  // minimum population to start spawning
	spawnChanceBase       = 0.001  // 0.1% base chance per tick per 10k population
	maxGuMastersPerRace   = 100    // cap per aperture per race
)

// processGuMasterSpawning checks all apertures and spawns Gu Masters as needed.
func processGuMasterSpawning(ctx context.Context, db *sql.DB) {
	type racePop struct {
		apertureID string
		race       string
		pop        int
		guMasters  int
		prosperity float64
	}

	rows, err := db.QueryContext(ctx, `
		SELECT vh.aperture_id, vh.race, vh.population, vh.gu_master_count, vh.prosperity
		FROM aperture_variant_humans vh
		JOIN immortal_apertures a ON a.id = vh.aperture_id
		WHERE vh.population >= $1 AND a.state != 'DESTROYED'
		ORDER BY vh.aperture_id, vh.race
	`, spawnThresholdPop)
	if err != nil {
		logger.S.Error("GuMasterSpawn: failed to query: ", err)
		return
	}
	defer rows.Close()

	var races []racePop
	for rows.Next() {
		var r racePop
		if err := rows.Scan(&r.apertureID, &r.race, &r.pop, &r.guMasters, &r.prosperity); err != nil {
			continue
		}
		races = append(races, r)
	}
	if err := rows.Err(); err != nil {
		logger.S.Error("GuMasterSpawn: rows error: ", err)
		return
	}
	if len(races) == 0 {
		return
	}

	// Collect aperture IDs for NPC name generation.
	type spawnCandidate struct {
		apertureID string
		race       string
		pop        int
		prosperity float64
		guMasters  int
	}

	var candidates []spawnCandidate
	for _, r := range races {
		if r.guMasters >= maxGuMastersPerRace {
			continue
		}

		// Calculate spawn chance based on population and prosperity.
		popFactor := float64(r.pop) / 10000.0
		prosperityMod := 1.0 + (r.prosperity-50.0)/200.0 // prosperity 100 → 1.25x
		chance := spawnChanceBase * popFactor * prosperityMod

		// Labor intensity modifier: EXPLOIT reduces happiness which reduces chance.
		if rand.Float64() < chance {
			candidates = append(candidates, spawnCandidate{
				apertureID: r.apertureID,
				race:       r.race,
				pop:        r.pop,
				prosperity: r.prosperity,
				guMasters:  r.guMasters,
			})
		}
	}

	if len(candidates) == 0 {
		return
	}

	// Spawn NPCs.
	for _, c := range candidates {
		npcName := generateNPCName(c.race)
		npcRank := rollGuMasterRank()
		npcTalent := 20.0 + rand.Float64()*60.0 // 20-80
		npcLoyalty := 50.0 + rand.Float64()*50.0 // 50-100

		_, err := db.ExecContext(ctx, `
			INSERT INTO aperture_variant_npcs (aperture_id, race, name, rank, sub_rank, path, role, loyalty, talent, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'IDLE')
		`, c.apertureID, c.race, npcName, npcRank, 0, "REFINEMENT", "NONE", npcLoyalty, npcTalent)
		if err != nil {
			logger.S.Errorf("GuMasterSpawn: insert NPC for %s: %v", c.apertureID, err)
			continue
		}

		// Update gu_master_count.
		_, err = db.ExecContext(ctx, `
			UPDATE aperture_variant_humans
			SET gu_master_count = gu_master_count + 1, updated_at = NOW()
			WHERE aperture_id = $1 AND race = $2
		`, c.apertureID, c.race)
		if err != nil {
			logger.S.Errorf("GuMasterSpawn: update count for %s %s: %v", c.apertureID, c.race, err)
		}
	}

	logger.S.Infof("GuMasterSpawn: spawned %d new Gu Masters", len(candidates))
}

// rollGuMasterRank determines the rank of a newly spawned Gu Master.
// Distribution: Rank 1: 70%, Rank 2: 20%, Rank 3: 8%, Rank 4: 2%
func rollGuMasterRank() int {
	r := rand.Float64()
	if r < 0.70 {
		return 1
	} else if r < 0.90 {
		return 2
	} else if r < 0.98 {
		return 3
	}
	return 4
}

// generateNPCName creates a culturally appropriate name for a variant human NPC.
func generateNPCName(race string) string {
	rand.Seed(time.Now().UnixNano())

	prefixes := map[string][]string{
		"HAIRY_MAN": {"Mao", "Hou", "Yuan", "Teng", "Guan", "Dai", "Shen", "Wu"},
		"ROCKMAN":   {"Shi", "Yan", "Kuang", "Tu", "Feng", "Lei", "Qiao", "Chui"},
		"FEATHERMAN": {"Yu", "Ling", "Feng", "Yun", "Kong", "Ying", "Chi", "Xiang"},
	}

	suffixes := []string{"Shi", "Er", "San", "Si", "Wu", "Liu", "Qi", "Ba",
		"Tian", "Di", "Ren", "He", "Ming", "Guang", "Da", "Chang"}

	p := prefixes[race]
	if len(p) == 0 {
		p = prefixes["HAIRY_MAN"]
	}

	return fmt.Sprintf("%s%s", p[rand.Intn(len(p))], suffixes[rand.Intn(len(suffixes))])
}

// --- NPC API handlers are in handlers/variant_economy.go ---
