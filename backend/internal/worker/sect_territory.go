package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"time"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func StartSectTerritoryWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	RunEvery(ctx, interval, "Sect Territory Worker", func(ctx context.Context) {
		processSectTerritoryTick(ctx, db)
	})
}

type sectTerritoryRow struct {
	id        string
	level     int
	buildings []byte
	lastTick  time.Time
}

func getBuildingLevel(buildingsJSON []byte, buildingName string) int {
	var buildings map[string]int
	if err := json.Unmarshal(buildingsJSON, &buildings); err == nil {
		if lvl, ok := buildings[buildingName]; ok {
			return lvl
		}
	}
	return 1 // default level 1
}

// sectVeinCoverage is the share of the daily level + member maintenance fee the
// spirit vein pays out, scaling with its building level: 60% + 5% per level,
// capped at 105%. The vault fee is never offset — it stays a sink that has to
// be covered by donations, missions and wars.
func sectVeinCoverage(spiritVeinLevel int) float64 {
	coverage := 0.60 + 0.05*float64(spiritVeinLevel)
	if coverage > 1.05 {
		coverage = 1.05
	}
	return coverage
}

// sectVeinDailyCredit is the Quỹ the vein credits to a sect over one day: the
// level + member upkeep scaled by the vein coverage and the +10%/defender guard
// bonus. At coverage 1.0 (vein level 8) it exactly cancels the level + member
// fee, which is the intended break-even point.
func sectVeinDailyCredit(sectLevel, memberCount, spiritVeinLevel, defenderCount int) float64 {
	upkeep := float64(sectLevel)*dailyLevelFee + float64(memberCount)*dailyMemberFee
	return upkeep * sectVeinCoverage(spiritVeinLevel) * (1.0 + 0.1*float64(defenderCount))
}

func processSectTerritoryTick(ctx context.Context, db *sql.DB) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, level, buildings, last_territory_tick_at
		FROM sects
	`)
	if err != nil {
		logger.S.Errorf("Sect Territory Worker: query sects failed: %v", err)
		return
	}

	var sects []sectTerritoryRow
	for rows.Next() {
		var sr sectTerritoryRow
		if err := rows.Scan(&sr.id, &sr.level, &sr.buildings, &sr.lastTick); err == nil {
			sects = append(sects, sr)
		}
	}
	rows.Close()

	if len(sects) == 0 {
		return
	}

	var activeSects []sectTerritoryRow
	for _, sr := range sects {
		if time.Since(sr.lastTick).Minutes() >= 1.0 {
			activeSects = append(activeSects, sr)
		}
	}
	if len(activeSects) == 0 {
		return
	}
	sort.Slice(activeSects, func(i, j int) bool {
		return activeSects[i].id < activeSects[j].id
	})

	sectIDs := make([]string, 0, len(activeSects))
	for _, sr := range activeSects {
		sectIDs = append(sectIDs, sr.id)
	}
	sort.Strings(sectIDs)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.S.Errorf("Sect Territory Worker: begin tx failed: %v", err)
		return
	}
	defer tx.Rollback()
	_, _ = tx.ExecContext(ctx, `SELECT set_config('lock_timeout', '3000', true), set_config('statement_timeout', '10000', true)`)

	// 1. Defender counts per sect
	defenderCountMap := map[string]int{}
	if dRows, err := tx.QueryContext(ctx, `SELECT sect_id, COUNT(*) FROM sect_defenders WHERE sect_id = ANY($1::text[]) GROUP BY sect_id`, pq.Array(sectIDs)); err != nil {
		logger.S.Errorf("Sect Territory Worker: defender count query failed: %v", err)
		return
	} else {
		for dRows.Next() {
			var sid string
			var cnt int
			if dRows.Scan(&sid, &cnt) == nil {
				defenderCountMap[sid] = cnt
			}
		}
		dRows.Close()
	}

	now := time.Now()

	// 2. Member counts per sect — the member term of the upkeep the vein offsets.
	memberCountMap := map[string]int{}
	if mRows, err := tx.QueryContext(ctx, `SELECT sect_id, COUNT(*) FROM sect_members WHERE sect_id = ANY($1::text[]) GROUP BY sect_id`, pq.Array(sectIDs)); err != nil {
		logger.S.Errorf("Sect Territory Worker: member count query failed: %v", err)
		return
	} else {
		for mRows.Next() {
			var sid string
			var cnt int
			if mRows.Scan(&sid, &cnt) == nil {
				memberCountMap[sid] = cnt
			}
		}
		mRows.Close()
	}

	sectFundCreditIDs := []string{}
	sectFundCredits := []float64{}

	for _, sr := range activeSects {
		elapsedMin := now.Sub(sr.lastTick).Minutes()
		defCount := defenderCountMap[sr.id]

		spiritVeinLvl := getBuildingLevel(sr.buildings, "spirit_vein")

		// The vein offsets the level + member maintenance fee; the vault fee is
		// left for members to cover by other means so it stays a real sink.
		fundsRate := sectVeinDailyCredit(sr.level, memberCountMap[sr.id], spiritVeinLvl, defCount) / 1440.0
		fundsCredit := fundsRate * elapsedMin

		if fundsCredit > 0 {
			sectFundCreditIDs = append(sectFundCreditIDs, sr.id)
			sectFundCredits = append(sectFundCredits, fundsCredit)
		}
	}

	// Batch UPDATE sects funds and last_territory_tick_at
	if len(sectFundCreditIDs) > 0 {
		if _, err := tx.Exec(`
			UPDATE sects s SET 
			    funds = s.funds + v.credit,
			    last_territory_tick_at = now()
			FROM (SELECT unnest($1::text[]) AS id, unnest($2::float8[]) AS credit) v
			WHERE s.id = v.id
		`, pq.Array(sectFundCreditIDs), pq.Array(sectFundCredits)); err != nil {
			logger.S.Errorf("Sect Territory Worker: batch update sect funds failed: %v", err)
			return
		}
	} else {
		// Just update the tick time
		if _, err := tx.Exec(`UPDATE sects SET last_territory_tick_at = now() WHERE id = ANY($1::text[])`, pq.Array(sectIDs)); err != nil {
			logger.S.Errorf("Sect Territory Worker: batch update tick time failed: %v", err)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Sect Territory Worker: commit failed: %v", err)
	}
}
