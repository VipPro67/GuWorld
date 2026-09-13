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

// StartSectTerritoryWorker periodically accrues territory resources (Quỹ,
// Linh tài, Điểm cống hiến) for every mine currently held by a sect.
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

type territoryMineRow struct {
	id           int
	rarity       string
	sectID       string
	pendingFunds float64
	pendingMats  float64
	lastClaimed  time.Time
	sectLevel    int
}

func sectMineBaseRate(rarity string) float64 {
	switch rarity {
	case "Siêu Cấp":
		return 200.0
	case "Truyền Thuyết":
		return 100.0
	case "Sử Thi":
		return 50.0
	case "Hiếm":
		return 25.0
	default:
		return 10.0
	}
}

func processSectTerritoryTick(ctx context.Context, db *sql.DB) {
	// ── Phase 0: Fetch all occupied mines ──
	rows, err := db.QueryContext(ctx, `
		SELECT m.id, m.rarity, m.occupier_sect_id, m.pending_funds, m.pending_materials,
		       COALESCE(m.last_claimed_at, m.occupied_at, now()), COALESCE(s.level, 1)
		FROM sect_mines m
		JOIN sects s ON s.id = m.occupier_sect_id
		WHERE m.occupier_sect_id IS NOT NULL
	`)
	if err != nil {
		logger.S.Errorf("Sect Territory Worker: query mines failed: %v", err)
		return
	}

	var mines []territoryMineRow
	for rows.Next() {
		var mr territoryMineRow
		if err := rows.Scan(&mr.id, &mr.rarity, &mr.sectID, &mr.pendingFunds, &mr.pendingMats, &mr.lastClaimed, &mr.sectLevel); err == nil {
			mines = append(mines, mr)
		}
	}
	rows.Close()

	if len(mines) == 0 {
		return
	}

	// Filter to mines with elapsed time >= 1 minute
	var activeMines []territoryMineRow
	for _, mr := range mines {
		if time.Since(mr.lastClaimed).Minutes() >= 1.0 {
			activeMines = append(activeMines, mr)
		}
	}
	if len(activeMines) == 0 {
		return
	}

	// Collect unique mine IDs and sect IDs
	mineIDs := make([]int, 0, len(activeMines))
	sectIDSet := map[string]bool{}
	for _, mr := range activeMines {
		mineIDs = append(mineIDs, mr.id)
		sectIDSet[mr.sectID] = true
	}
	sectIDs := make([]string, 0, len(sectIDSet))
	for sid := range sectIDSet {
		sectIDs = append(sectIDs, sid)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.S.Errorf("Sect Territory Worker: begin tx failed: %v", err)
		return
	}
	defer tx.Rollback()

	// ── Phase 1: Batch reads ──

	// 1a. Defender counts per mine
	defenderCountMap := map[int]int{}
	if dRows, err := tx.QueryContext(ctx, `SELECT mine_id, COUNT(*) FROM sect_mine_defenders WHERE mine_id = ANY($1::int[]) GROUP BY mine_id`, pq.Array(mineIDs)); err == nil {
		for dRows.Next() {
			var mid, cnt int
			if dRows.Scan(&mid, &cnt) == nil {
				defenderCountMap[mid] = cnt
			}
		}
		dRows.Close()
	}

	// 1b. Defender player IDs per mine
	defenderIDsByMine := map[int]map[string]bool{}
	if dRows, err := tx.QueryContext(ctx, `SELECT mine_id, player_id::text FROM sect_mine_defenders WHERE mine_id = ANY($1::int[]) AND defender_type = 'member'`, pq.Array(mineIDs)); err == nil {
		for dRows.Next() {
			var mid int
			var pid string
			if dRows.Scan(&mid, &pid) == nil {
				if defenderIDsByMine[mid] == nil {
					defenderIDsByMine[mid] = map[string]bool{}
				}
				defenderIDsByMine[mid][pid] = true
			}
		}
		dRows.Close()
	}

	// 1c. Member player IDs and counts per sect
	memberIDsBySect := map[string][]string{}
	memberCountMap := map[string]int{}
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
	for sid, members := range memberIDsBySect {
		memberCountMap[sid] = len(members)
	}

	// All unique member player IDs for contribution credit
	allMemberIDs := make([]string, 0, len(allMemberIDSet))
	for pid := range allMemberIDSet {
		allMemberIDs = append(allMemberIDs, pid)
	}

	// ── Phase 2: Compute per-mine (pure Go) ──
	now := time.Now()
	today := now.Format("2006-01-02")

	// Arrays for batch UPDATE sect_mines
	muMineIDs := make([]int, 0, len(activeMines))
	muFundRates := make([]float64, 0, len(activeMines))
	muMatRates := make([]float64, 0, len(activeMines))
	muFundCredits := make([]float64, 0, len(activeMines))
	muMatCredits := make([]float64, 0, len(activeMines))

	// Arrays for batch UPDATE sects funds
	sectFundCreditIDs := []string{}
	sectFundCredits := []float64{}

	// Material handling: accumulate per sect
	type matCredit struct {
		matID string
		qty   int
	}
	sectMatCredits := map[string][]matCredit{} // sectID -> []matCredit
	sectMatMineIDs := []int{}                  // mine IDs that had matCredit
	sectMatCreditsFlat := []float64{}

	// Max sect level for material dictionary lookup
	maxSectLevel := 1
	for _, mr := range activeMines {
		if mr.sectLevel > maxSectLevel {
			maxSectLevel = mr.sectLevel
		}
	}

	for _, mr := range activeMines {
		elapsedMin := now.Sub(mr.lastClaimed).Minutes()
		defCount := defenderCountMap[mr.id]
		defMult := 1.0 + 0.2*float64(defCount)

		fundsRate := sectMineBaseRate(mr.rarity) * float64(mr.sectLevel) * defMult
		matRate := sectMineBaseRate(mr.rarity) * 0.2 * defMult

		newFunds := mr.pendingFunds + fundsRate*elapsedMin
		newMats := mr.pendingMats + matRate*elapsedMin

		fundsCredit := math.Floor(newFunds)
		matCreditAmt := math.Floor(newMats)

		muMineIDs = append(muMineIDs, mr.id)
		muFundRates = append(muFundRates, fundsRate*elapsedMin)
		muMatRates = append(muMatRates, matRate*elapsedMin)
		muFundCredits = append(muFundCredits, fundsCredit)
		muMatCredits = append(muMatCredits, matCreditAmt)

		if fundsCredit > 0 {
			// Deduplicate sect fund credits (multiple mines can share sect)
			found := false
			for i, sid := range sectFundCreditIDs {
				if sid == mr.sectID {
					sectFundCredits[i] += fundsCredit
					found = true
					break
				}
			}
			if !found {
				sectFundCreditIDs = append(sectFundCreditIDs, mr.sectID)
				sectFundCredits = append(sectFundCredits, fundsCredit)
			}
		}

		if matCreditAmt > 0 {
			sectMatMineIDs = append(sectMatMineIDs, mr.id)
			sectMatCreditsFlat = append(sectMatCreditsFlat, matCreditAmt)
			// matID will be resolved after batch read
			sectMatCredits[mr.sectID] = append(sectMatCredits[mr.sectID], matCredit{
				qty: int(matCreditAmt),
			})
		}
	}

	// ── Phase 3: Batch writes ──

	// 3a. UPDATE sect_mines pending_funds + pending_materials + last_claimed_at
	if _, err := tx.Exec(`
		UPDATE sect_mines m SET
			pending_funds = m.pending_funds - v.fund_credit + v.fund_rate,
			pending_materials = m.pending_materials - v.mat_credit + v.mat_rate,
			last_claimed_at = now()
		FROM (SELECT unnest($1::int[]) AS id,
		             unnest($2::float8[]) AS fund_rate,
		             unnest($3::float8[]) AS mat_rate,
		             unnest($4::float8[]) AS fund_credit,
		             unnest($5::float8[]) AS mat_credit) v
		WHERE m.id = v.id
	`, pq.Array(muMineIDs), pq.Array(muFundRates), pq.Array(muMatRates),
		pq.Array(muFundCredits), pq.Array(muMatCredits)); err != nil {
		logger.S.Errorf("Sect Territory Worker: batch update mines failed: %v", err)
		return
	}

	// 3b. Batch UPDATE sects funds
	if len(sectFundCreditIDs) > 0 {
		if _, err := tx.Exec(`
			UPDATE sects s SET funds = s.funds + v.credit
			FROM (SELECT unnest($1::text[]) AS id, unnest($2::float8[]) AS credit) v
			WHERE s.id = v.id
		`, pq.Array(sectFundCreditIDs), pq.Array(sectFundCredits)); err != nil {
			logger.S.Errorf("Sect Territory Worker: batch update sect funds failed: %v", err)
			return
		}
	}

	// 3c. Material handling
	if len(sectMatMineIDs) > 0 {
		// Pre-fetch random materials per rank (one per rank up to maxSectLevel)
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

		// Assign material IDs to each mine's credit
		// For each mine with matCredit, find a random material based on sect level
		sectIDsNeedingVault := map[string]bool{}
		for i, mid := range sectMatMineIDs {
			// Find the mine's sect level
			for _, mr := range activeMines {
				if mr.id == mid {
					matID := matByRank[mr.sectLevel]
					if matID == "" {
						// Fallback to rank 1
						matID = matByRank[1]
					}
					if matID != "" {
						// Find the correct index in sectMatCredits to assign matID
						credits := sectMatCredits[mr.sectID]
						for j := range credits {
							if credits[j].matID == "" && credits[j].qty == int(sectMatCreditsFlat[i]) {
								credits[j].matID = matID
								break
							}
						}
						sectIDsNeedingVault[mr.sectID] = true
					}
					break
				}
			}
		}

		if len(sectIDsNeedingVault) > 0 {
			vaultIDs := make([]string, 0, len(sectIDsNeedingVault))
			for sid := range sectIDsNeedingVault {
				vaultIDs = append(vaultIDs, sid)
			}

			// Batch read vaults
			vaultMap := map[string]map[string]int{}
			if vRows, err := tx.Query(`SELECT id, vault_materials FROM sects WHERE id = ANY($1::text[])`, pq.Array(vaultIDs)); err == nil {
				for vRows.Next() {
					var sid string
					var raw []byte
					if vRows.Scan(&sid, &raw) == nil {
						vault := map[string]int{}
						if len(raw) > 0 {
							_ = json.Unmarshal(raw, &vault)
						}
						vaultMap[sid] = vault
					}
				}
				vRows.Close()
			}

			// Accumulate materials in vaults
			for sid, credits := range sectMatCredits {
				vault := vaultMap[sid]
				if vault == nil {
					vault = map[string]int{}
					vaultMap[sid] = vault
				}
				for _, mc := range credits {
					if mc.matID != "" {
						vault[mc.matID] += mc.qty
					}
				}
			}

			// Batch write vaults
			writeIDs := make([]string, 0, len(vaultMap))
			writeRaw := make([]string, 0, len(vaultMap))
			for sid, vault := range vaultMap {
				raw, _ := json.Marshal(vault)
				writeIDs = append(writeIDs, sid)
				writeRaw = append(writeRaw, string(raw))
			}
			if len(writeIDs) > 0 {
				if _, err := tx.Exec(`
					UPDATE sects s SET vault_materials = v.vault
					FROM (SELECT unnest($1::text[]) AS id, unnest($2::jsonb[]) AS vault) v
					WHERE s.id = v.id
				`, pq.Array(writeIDs), pq.Array(writeRaw)); err != nil {
					logger.S.Errorf("Sect Territory Worker: batch update vaults failed: %v", err)
					return
				}
			}
		}
	}

	// 3d. Batch contribution credit
	// Original logic: for each mine, for each member of the mine's sect,
	// add rate * elapsedMin to territory_pending, then credit whole points.
	// A player can appear in multiple mines (if their sect owns >1 mine).
	if len(allMemberIDs) > 0 {
		// Read all member states in 1 query
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

		// Accumulate pending per player across all mines
		// pendingAccum[pid] = extra pending to add this tick
		pendingAccum := map[string]float64{}
		for _, mr := range activeMines {
			elapsedMin := now.Sub(mr.lastClaimed).Minutes()
			if elapsedMin <= 0 {
				continue
			}
			defenderIDs := defenderIDsByMine[mr.id]
			for _, pid := range memberIDsBySect[mr.sectID] {
				rate := territoryMemberRate
				if defenderIDs[pid] {
					rate = territoryDefenderRate
				}
				pendingAccum[pid] += rate * elapsedMin
			}
		}

		// Compute per-player contribution credit
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

			// Date rollover
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
				playerID:         pid,
				contributionDelta: credit,
				newPending:       newPending,
				newCreditedToday: newCreditedToday,
				newCreditedAt:    credDate,
			})
		}

		// Batch write contribution credits
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
				UPDATE sm SET
					contribution_points = sm.contribution_points + v.delta,
					territory_pending = v.pending,
					territory_credited_today = v.credited_today,
					territory_credited_at = v.credited_at::date
				FROM sect_members sm
				JOIN (SELECT unnest($1::uuid[]) AS player_id,
				             unnest($2::int[]) AS delta,
				             unnest($3::float8[]) AS pending,
				             unnest($4::int[]) AS credited_today,
				             unnest($5::text[]) AS credited_at) v ON sm.player_id = v.player_id
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
