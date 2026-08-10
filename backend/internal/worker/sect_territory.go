package worker

import (
	"context"
	"database/sql"
	"encoding/json"
	"math"
	"time"

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
				processSectTerritoryTick(db)
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

func processSectTerritoryTick(db *sql.DB) {
	rows, err := db.Query(`
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

	for _, mr := range mines {
		if err := processTerritoryMine(db, mr); err != nil {
			logger.S.Errorf("Sect Territory Worker: mine %d error: %v", mr.id, err)
		}
	}
}

func processTerritoryMine(db *sql.DB, mr territoryMineRow) error {
	elapsedMin := time.Since(mr.lastClaimed).Minutes()
	if elapsedMin <= 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var defCount int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM sect_mine_defenders WHERE mine_id = $1`, mr.id).Scan(&defCount)

	var memberCount int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM sect_members WHERE sect_id = $1`, mr.sectID).Scan(&memberCount)

	defMult := 1 + 0.2*float64(defCount)
	fundsRate := sectMineBaseRate(mr.rarity) * float64(mr.sectLevel) * defMult
	matRate := sectMineBaseRate(mr.rarity) * 0.2 * defMult

	var newFunds, newMats float64
	err = tx.QueryRow(`
		UPDATE sect_mines
		SET pending_funds = pending_funds + $1,
		    pending_materials = pending_materials + $2,
		    last_claimed_at = now()
		WHERE id = $3
		RETURNING pending_funds, pending_materials
	`, fundsRate*elapsedMin, matRate*elapsedMin, mr.id).Scan(&newFunds, &newMats)
	if err != nil {
		return err
	}

	fundsCredit := int(math.Floor(newFunds))
	if fundsCredit > 0 {
		if _, err = tx.Exec(`UPDATE sects SET funds = funds + $1 WHERE id = $2`, fundsCredit, mr.sectID); err != nil {
			return err
		}
		if _, err = tx.Exec(`UPDATE sect_mines SET pending_funds = pending_funds - $1 WHERE id = $2`, float64(fundsCredit), mr.id); err != nil {
			return err
		}
	}

	matCredit := int(math.Floor(newMats))
	if matCredit > 0 {
		matID := ""
		_ = tx.QueryRow(`
			SELECT id FROM item_dictionary
			WHERE item_type = 'Material' AND rank >= 1 AND rank <= $1
			ORDER BY RANDOM() LIMIT 1
		`, mr.sectLevel).Scan(&matID)
		if matID != "" {
			var vaultMatsRaw []byte
			_ = tx.QueryRow(`SELECT vault_materials FROM sects WHERE id = $1`, mr.sectID).Scan(&vaultMatsRaw)
			vaultMats := map[string]int{}
			if len(vaultMatsRaw) > 0 {
				_ = json.Unmarshal(vaultMatsRaw, &vaultMats)
			}
			vaultMats[matID] += matCredit
			newRaw, _ := json.Marshal(vaultMats)
			if _, err = tx.Exec(`UPDATE sects SET vault_materials = $1 WHERE id = $2`, newRaw, mr.sectID); err != nil {
				return err
			}
		}
		if _, err = tx.Exec(`UPDATE sect_mines SET pending_materials = pending_materials - $1 WHERE id = $2`, float64(matCredit), mr.id); err != nil {
			return err
		}
	}

	// Contribution: defenders get the defender rate; other members get the member rate.
	defenderIDs := map[string]bool{}
	rows, err := tx.Query(`SELECT player_id::text FROM sect_mine_defenders WHERE mine_id = $1 AND defender_type = 'member'`, mr.id)
	if err == nil {
		for rows.Next() {
			var pid string
			if rows.Scan(&pid) == nil {
				defenderIDs[pid] = true
			}
		}
		rows.Close()
	}

	// Collect all member ids first (must close rows before issuing more queries
	// on the same tx — lib/pq cannot interleave a second query mid-resultset).
	var memberIDs []string
	memberRows, err := tx.Query(`SELECT player_id::text FROM sect_members WHERE sect_id = $1`, mr.sectID)
	if err == nil {
		for memberRows.Next() {
			var pid string
			if memberRows.Scan(&pid) == nil {
				memberIDs = append(memberIDs, pid)
			}
		}
		memberRows.Close()
	}

	for _, pid := range memberIDs {
		rate := territoryMemberRate
		if defenderIDs[pid] {
			rate = territoryDefenderRate
		}
		creditTerritoryContributionWorker(tx, pid, rate, elapsedMin)
	}

	return tx.Commit()
}

// creditTerritoryContributionWorker accrues pending contribution and credits
// whole points while respecting the daily cap.
func creditTerritoryContributionWorker(tx *sql.Tx, playerID string, ratePerMin, elapsedMin float64) {
	today := time.Now().Format("2006-01-02")
	_, _ = tx.Exec(`
		UPDATE sect_members SET territory_credited_at = $1::date, territory_credited_today = 0
		WHERE player_id = $2 AND (territory_credited_at IS NULL OR territory_credited_at IS DISTINCT FROM $1::date)
	`, today, playerID)

	var creditedToday int
	_ = tx.QueryRow(`SELECT territory_credited_today FROM sect_members WHERE player_id = $1`, playerID).Scan(&creditedToday)
	if creditedToday >= territoryDailyCap {
		return
	}

	var pending float64
	_ = tx.QueryRow(`SELECT territory_pending FROM sect_members WHERE player_id = $1`, playerID).Scan(&pending)
	pending += ratePerMin * elapsedMin

	credit := int(math.Floor(pending))
	remaining := territoryDailyCap - creditedToday
	if credit > remaining {
		credit = remaining
	}
	if credit <= 0 {
		_, _ = tx.Exec(`UPDATE sect_members SET territory_pending = $1 WHERE player_id = $2`, pending, playerID)
		return
	}

	_, err := tx.Exec(`
		UPDATE sect_members SET contribution_points = contribution_points + $1,
		       territory_pending = $2, territory_credited_today = $3
		WHERE player_id = $4
	`, credit, pending-float64(credit), creditedToday+credit, playerID)
	if err != nil {
		logger.S.Errorf("creditTerritoryContributionWorker error for %s: %v", playerID, err)
	}
}
