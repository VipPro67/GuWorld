package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"time"

	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Territory daily contribution cap per player.
const territoryDailyCap = 200

// Territory contribution rates per minute.
const (
	territoryDefenderRate = 20.0 / 60.0 // defenders
	territoryMemberRate   = 2.0 / 60.0  // regular members
)

func StartSectTerritoryWorker(ctx context.Context, db *sql.DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		logger.S.Infof("Starting background Sect Territory Worker with tick interval of %v", interval)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				logger.S.Info("Sect Territory Worker stopped")
				return
			case <-ticker.C:
				processSectTerritoryTick(ctx, db)
			}
		}
	}()
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

func sectBaseRate(spiritVeinLevel int) float64 {
	// e.g. Level 1: 50, Level 2: 100, Level 3: 200 funds/hour...
	// per minute rate
	return 10.0 + float64(spiritVeinLevel)*10.0
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

	sectIDs := make([]string, 0, len(activeSects))
	for _, sr := range activeSects {
		sectIDs = append(sectIDs, sr.id)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.S.Errorf("Sect Territory Worker: begin tx failed: %v", err)
		return
	}
	defer tx.Rollback()

	// 1a. Defender counts per sect
	defenderCountMap := map[string]int{}
	if dRows, err := tx.QueryContext(ctx, `SELECT sect_id, COUNT(*) FROM sect_defenders WHERE sect_id = ANY($1::text[]) GROUP BY sect_id`, pq.Array(sectIDs)); err == nil {
		for dRows.Next() {
			var sid string
			var cnt int
			if dRows.Scan(&sid, &cnt) == nil {
				defenderCountMap[sid] = cnt
			}
		}
		dRows.Close()
	}

	// 1b. Defender player IDs per sect
	defenderIDsBySect := map[string]map[string]bool{}
	if dRows, err := tx.QueryContext(ctx, `SELECT sect_id, player_id::text FROM sect_defenders WHERE sect_id = ANY($1::text[]) AND defender_type = 'member'`, pq.Array(sectIDs)); err == nil {
		for dRows.Next() {
			var sid string
			var pid string
			if dRows.Scan(&sid, &pid) == nil {
				if defenderIDsBySect[sid] == nil {
					defenderIDsBySect[sid] = map[string]bool{}
				}
				defenderIDsBySect[sid][pid] = true
			}
		}
		dRows.Close()
	}

	// 1c. Member player IDs per sect
	memberIDsBySect := map[string][]string{}
	allMemberIDSet := map[string]bool{}
	if mRows, err := tx.QueryContext(ctx, `SELECT sect_id, player_id::text FROM sect_members WHERE sect_id = ANY($1::text[])`, pq.Array(sectIDs)); err == nil {
		for mRows.Next() {
			var sid, pid string
			if mRows.Scan(&sid, &pid) == nil {
				memberIDsBySect[sid] = append(memberIDsBySect[sid], pid)
				allMemberIDSet[pid] = true
			}
		}
		mRows.Close()
	}

	allMemberIDs := make([]string, 0, len(allMemberIDSet))
	for pid := range allMemberIDSet {
		allMemberIDs = append(allMemberIDs, pid)
	}

	now := time.Now()
	today := now.Format("2006-01-02")

	sectFundCreditIDs := []string{}
	sectFundCredits := []float64{}
	sectMatCredits := map[string]int{}

	maxSectLevel := 1
	for _, sr := range activeSects {
		if sr.level > maxSectLevel {
			maxSectLevel = sr.level
		}
	}

	for _, sr := range activeSects {
		elapsedMin := now.Sub(sr.lastTick).Minutes()
		defCount := defenderCountMap[sr.id]
		defMult := 1.0 + 0.1*float64(defCount)

		spiritVeinLvl := getBuildingLevel(sr.buildings, "spirit_vein")

		fundsRate := sectBaseRate(spiritVeinLvl) * defMult
		matRate := sectBaseRate(spiritVeinLvl) * 0.2 * defMult

		fundsCredit := fundsRate * elapsedMin
		matCreditAmt := matRate * elapsedMin

		if fundsCredit > 0 {
			sectFundCreditIDs = append(sectFundCreditIDs, sr.id)
			sectFundCredits = append(sectFundCredits, fundsCredit)
		}

		if matCreditAmt > 1.0 { // only credit if >= 1.0 to avoid fractional spam (though we can floor it)
			sectMatCredits[sr.id] = int(math.Floor(matCreditAmt))
		}
	}

	// 3a. Batch UPDATE sects funds and last_territory_tick_at
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
		}
	}

	// 3b. Material handling
	if len(sectMatCredits) > 0 {
		matByRank := map[int]string{}
		if matRows, err := tx.Query(`
			SELECT DISTINCT ON (rank) rank, id FROM item_dictionary
			WHERE item_type = 'Material' AND rank >= 1 AND rank <= $1
			ORDER BY rank, RANDOM()
		`, maxSectLevel); err == nil {
			for matRows.Next() {
				var rank int
				var id string
				if matRows.Scan(&rank, &id) == nil {
					matByRank[rank] = id
				}
			}
			matRows.Close()
		}

		vaultIDs := make([]string, 0, len(sectMatCredits))
		for sid := range sectMatCredits {
			vaultIDs = append(vaultIDs, sid)
		}

		vaultMap := map[string]map[string]int{}
		if vRows, err := tx.Query(`SELECT sect_id, item_id, quantity FROM sect_vault_materials WHERE sect_id = ANY($1::text[])`, pq.Array(vaultIDs)); err == nil {
			for vRows.Next() {
				var sid string
				var itemID string
				var qty int
				if vRows.Scan(&sid, &itemID, &qty) == nil {
					if vaultMap[sid] == nil {
						vaultMap[sid] = map[string]int{}
					}
					vaultMap[sid][itemID] += qty
				}
			}
			vRows.Close()
		}

		for _, sr := range activeSects {
			qty := sectMatCredits[sr.id]
			if qty > 0 {
				matID := matByRank[sr.level]
				if matID == "" {
					matID = matByRank[1]
				}
				if matID != "" {
					vault := vaultMap[sr.id]
					if vault == nil {
						vault = map[string]int{}
						vaultMap[sr.id] = vault
					}
					vault[matID] += qty
				}
			}
		}

		// Update sect_vault_materials using upsert
		for sid, vault := range vaultMap {
			for matID, qty := range vault {
				_, err := tx.Exec(`
					INSERT INTO sect_vault_materials (sect_id, item_id, quantity)
					VALUES ($1, $2, $3)
					ON CONFLICT (sect_id, item_id) DO UPDATE SET
						quantity = sect_vault_materials.quantity + EXCLUDED.quantity,
						updated_at = NOW()
				`, sid, matID, qty)
				if err != nil {
					logger.S.Errorf("Sect Territory Worker: failed to update vault material %s for sect %s: %v", matID, sid, err)
				}
			}
		}
	}

	// 3c. Batch contribution credit
	if len(allMemberIDs) > 0 {
		type memberState struct {
			territoryPending    float64
			creditedToday       int
			territoryCreditedAt string
		}
		memberStates := map[string]memberState{}
		if cRows, err := tx.Query(`
			SELECT player_id, territory_pending, territory_credited_today, COALESCE(territory_credited_at::text, '')
			FROM sect_members WHERE player_id = ANY($1::uuid[])
		`, pq.Array(allMemberIDs)); err == nil {
			for cRows.Next() {
				var pid string
				var ms memberState
				if cRows.Scan(&pid, &ms.territoryPending, &ms.creditedToday, &ms.territoryCreditedAt) == nil {
					memberStates[pid] = ms
				}
			}
			cRows.Close()
		}

		pendingAccum := map[string]float64{}
		for _, sr := range activeSects {
			elapsedMin := now.Sub(sr.lastTick).Minutes()
			if elapsedMin <= 0 {
				continue
			}

			trainingLvl := getBuildingLevel(sr.buildings, "training_ground")
			trainingMult := 1.0 + float64(trainingLvl)*0.1

			defenderIDs := defenderIDsBySect[sr.id]
			for _, pid := range memberIDsBySect[sr.id] {
				rate := territoryMemberRate
				if defenderIDs != nil && defenderIDs[pid] {
					rate = territoryDefenderRate
				}
				pendingAccum[pid] += rate * elapsedMin * trainingMult
			}
		}

		type memberCreditResult struct {
			playerID          string
			contributionDelta int
			newPending        float64
			newCreditedToday  int
			newCreditedAt     string
		}
		var creditResults []memberCreditResult

		for _, pid := range allMemberIDs {
			ms := memberStates[pid]
			creditedToday := ms.creditedToday
			credDate := ms.territoryCreditedAt
			if credDate == "" || credDate != today {
				creditedToday = 0
				credDate = today
			}

			pending := ms.territoryPending + pendingAccum[pid]

			if creditedToday >= territoryDailyCap {
				creditResults = append(creditResults, memberCreditResult{
					playerID:         pid,
					newPending:       pending,
					newCreditedToday: creditedToday,
					newCreditedAt:    credDate,
				})
				continue
			}

			credit := int(math.Floor(pending))
			remaining := territoryDailyCap - creditedToday
			if credit > remaining {
				credit = remaining
			}
			newPending := pending
			newCreditedToday := creditedToday
			if credit > 0 {
				newPending = pending - float64(credit)
				newCreditedToday = creditedToday + credit
			}

			creditResults = append(creditResults, memberCreditResult{
				playerID:          pid,
				contributionDelta: credit,
				newPending:        newPending,
				newCreditedToday:  newCreditedToday,
				newCreditedAt:     credDate,
			})
		}

		if len(creditResults) > 0 {
			crIDs := make([]string, 0, len(creditResults))
			crDeltas := make([]int, 0, len(creditResults))
			crPendings := make([]float64, 0, len(creditResults))
			crCreditedTodays := make([]int, 0, len(creditResults))
			crCreditedAts := make([]string, 0, len(creditResults))

			for _, cr := range creditResults {
				crIDs = append(crIDs, cr.playerID)
				crDeltas = append(crDeltas, cr.contributionDelta)
				crPendings = append(crPendings, cr.newPending)
				crCreditedTodays = append(crCreditedTodays, cr.newCreditedToday)
				crCreditedAts = append(crCreditedAts, cr.newCreditedAt)
			}

			if _, err := tx.ExecContext(ctx, `
				UPDATE sect_members AS sm SET
					contribution_points = sm.contribution_points + v.delta,
					territory_pending = v.pending,
					territory_credited_today = v.credited_today,
					territory_credited_at = v.credited_at::date
				FROM (SELECT unnest($1::uuid[]) AS player_id,
				             unnest($2::int[]) AS delta,
				             unnest($3::float8[]) AS pending,
				             unnest($4::int[]) AS credited_today,
				             unnest($5::text[]) AS credited_at) v
				WHERE sm.player_id = v.player_id
			`, pq.Array(crIDs), pq.Array(crDeltas), pq.Array(crPendings),
				pq.Array(crCreditedTodays), pq.Array(crCreditedAts)); err != nil {
				logger.S.Errorf("Sect Territory Worker: batch update contributions failed: %v", err)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Sect Territory Worker: commit failed: %v", err)
	}
}
