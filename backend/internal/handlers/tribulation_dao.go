package handlers

import (
	"database/sql"
	"encoding/json"
	"math"
	"math/rand"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func daoMarkGainMultiplierTx(tx *sql.Tx, playerID string) float64 {
	var constitution string
	if err := tx.QueryRow(`SELECT COALESCE(constitution, '') FROM players WHERE id = $1`, playerID).Scan(&constitution); err != nil {
		return 1.0
	}
	return 1.0 + gamefx.Get(&models.Player{CharacterState: models.CharacterState{Constitution: constitution}}).DaoMarkGainBonus
}

func addDaoMarksTx(tx *sql.Tx, playerID string, path string, amount int) error {
	amount = int(float64(amount) * daoMarkGainMultiplierTx(tx, playerID))
	var daoJSON []byte
	err := tx.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1 FOR UPDATE`, playerID).Scan(&daoJSON)
	if err != nil {
		return err
	}

	daoMap := make(map[string]int)
	if len(daoJSON) > 0 {
		if err := json.Unmarshal(daoJSON, &daoMap); err != nil {
			return err
		}
	}
	daoMap[path] += amount

	daoJSON, err = json.Marshal(daoMap)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE immortal_apertures SET dao_marks = $1 WHERE player_id = $2`, daoJSON, playerID)
	return err
}

func addMultipleDaoMarksTx(tx *sql.Tx, playerID string, rewards map[string]int) (map[string]int, error) {
	mult := daoMarkGainMultiplierTx(tx, playerID)
	for path, amount := range rewards {
		rewards[path] = int(float64(amount) * mult)
	}
	var daoJSON []byte
	err := tx.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1 FOR UPDATE`, playerID).Scan(&daoJSON)
	if err != nil {
		return nil, err
	}

	daoMap := make(map[string]int)
	if len(daoJSON) > 0 {
		if err := json.Unmarshal(daoJSON, &daoMap); err != nil {
			return nil, err
		}
	}
	for path, amount := range rewards {
		daoMap[path] += amount
	}

	daoJSON, err = json.Marshal(daoMap)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`UPDATE immortal_apertures SET dao_marks = $1 WHERE player_id = $2`, daoJSON, playerID)
	if err != nil {
		return nil, err
	}
	return daoMap, nil
}

func distributeDaoMarks(mainPath string, totalMarks int) map[string]int {
	rewards := make(map[string]int)
	if totalMarks <= 0 {
		return rewards
	}
	// Main path gets 70%
	mainAmount := int(float64(totalMarks) * 0.7)
	if mainAmount < 1 {
		mainAmount = totalMarks
	}
	rewards[mainPath] = mainAmount

	remaining := totalMarks - mainAmount
	if remaining > 0 {
		allPaths := models.AllGuPaths()
		otherPaths := []string{}
		for _, p := range allPaths {
			if p != mainPath {
				otherPaths = append(otherPaths, p)
			}
		}
		rand.Shuffle(len(otherPaths), func(i, j int) {
			otherPaths[i], otherPaths[j] = otherPaths[j], otherPaths[i]
		})

		numExtra := rand.Intn(2) + 1 // 1 or 2 extra paths
		if numExtra > len(otherPaths) {
			numExtra = len(otherPaths)
		}

		if numExtra == 1 {
			rewards[otherPaths[0]] = remaining
		} else {
			part1 := remaining / 2
			part2 := remaining - part1
			rewards[otherPaths[0]] = part1
			rewards[otherPaths[1]] = part2
		}
	}
	return rewards
}

// Background Worker Tick: Advance time and trigger tribulations
// guardNextTribulationYear ensures the next scheduled tribulation never lands
// behind the current internal year (e.g. after a rank-up, where the new rank's
// schedule is computed from intervals smaller than the already-passed year).
// A milestone within the current 10-year window (e.g. overdue by a few moments)
// is left as-is so that due events trigger cleanly.
// If nextYear is more than 10 years in the past (e.g. corrupted or stale data),
// it is pushed forward to the next valid milestone strictly greater than currentYears.
// Also guards against math.MaxFloat64 (infinity) returned when no valid tribulation
// type exists for the current rank (e.g., rank < 6 or rank > 9).
func guardNextTribulationYear(nextYear, currentYears float64) float64 {
	if math.IsInf(nextYear, 0) || nextYear == math.MaxFloat64 {
		return math.Ceil(currentYears) + 10
	}
	if currentYears-nextYear > 10.0 {
		return math.Floor(currentYears/10)*10 + 10
	}
	return math.Round(nextYear)
}

// Offline-protection tuning for tribulations. While the player is offline the
// tribulation is held (no spawn / no damage); the years still flow so players
// cannot game the calendar. A grace period after spawn/return lets them prepare.
const (
	tribulationOnlineWindow = 5 * time.Minute // how fresh last_activity must be to count as "online"
	tribulationGracePeriod  = 5 * time.Minute // enemies idle this long after spawn/return
	tribulationPenaltyAfter = 24 * time.Hour  // offline hold beyond this halves the Dao Ngân reward
)
