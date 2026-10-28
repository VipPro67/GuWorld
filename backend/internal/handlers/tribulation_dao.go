package handlers

import (
	"database/sql"
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const heavenDaoPath = "heaven"

// eligibleOrdinaryDaoMarkPaths returns a fresh reward pool for ordinary
// ascension and tribulation rewards. Heaven Path is intentionally reserved for
// a separate acquisition mechanic.
func eligibleOrdinaryDaoMarkPaths(paths []string) []string {
	eligible := make([]string, 0, len(paths))
	for _, path := range paths {
		normalized := strings.TrimSpace(strings.ToLower(path))
		normalized = strings.TrimSpace(strings.TrimSuffix(normalized, " path"))
		if normalized != "" && normalized != heavenDaoPath {
			eligible = append(eligible, normalized)
		}
	}
	return eligible
}

func randomOrdinaryDaoMarkPaths(paths []string, count int) []string {
	eligible := eligibleOrdinaryDaoMarkPaths(paths)
	rand.Shuffle(len(eligible), func(i, j int) {
		eligible[i], eligible[j] = eligible[j], eligible[i]
	})
	if count < 0 {
		count = 0
	}
	if count > len(eligible) {
		count = len(eligible)
	}
	return eligible[:count]
}

func daoMarkGainMultiplierTx(tx *sql.Tx, playerID string) float64 {
	var constitution string
	if err := tx.QueryRow(`SELECT COALESCE(constitution, '') FROM players WHERE id = $1`, playerID).Scan(&constitution); err != nil {
		return 1.0
	}
	return 1.0 + gamefx.Get(&models.Player{CharacterState: models.CharacterState{Constitution: constitution}}).DaoMarkGainBonus
}

func addDaoMarksTx(tx *sql.Tx, playerID string, path string, amount int64) error {
	amount = int64(float64(amount) * daoMarkGainMultiplierTx(tx, playerID))
	var daoJSON []byte
	err := tx.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1 FOR UPDATE`, playerID).Scan(&daoJSON)
	if err != nil {
		return err
	}

	daoMap := make(models.DaoMarksMap)
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

func addMultipleDaoMarksTx(tx *sql.Tx, playerID string, rewards map[string]int64) (models.DaoMarksMap, error) {
	mult := daoMarkGainMultiplierTx(tx, playerID)
	for path, amount := range rewards {
		rewards[path] = int64(float64(amount) * mult)
	}
	var daoJSON []byte
	err := tx.QueryRow(`SELECT COALESCE(dao_marks, '{}'::jsonb) FROM immortal_apertures WHERE player_id = $1 FOR UPDATE`, playerID).Scan(&daoJSON)
	if err != nil {
		return nil, err
	}

	daoMap := make(models.DaoMarksMap)
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

func distributeDaoMarks(mainPath string, totalMarks int64) map[string]int64 {
	// Chaotic/Hybrid tribulations store their constituent elements as a
	// comma-joined path list and spread rewards evenly (spec §5.3).
	if strings.Contains(mainPath, ",") {
		return distributeEvenDaoMarks(strings.Split(mainPath, ","), totalMarks)
	}
	return distributeDaoMarksFromPaths(mainPath, totalMarks, models.AllGuPaths())
}

// distributeEvenDaoMarks splits totalMarks equally across the given paths,
// skipping ineligible paths (e.g. heaven dao). No dominant-path bias.
func distributeEvenDaoMarks(paths []string, totalMarks int64) map[string]int64 {
	rewards := make(map[string]int64)
	if totalMarks <= 0 {
		return rewards
	}
	eligible := eligibleOrdinaryDaoMarkPaths(paths)
	if len(eligible) == 0 {
		return rewards
	}
	share := totalMarks / int64(len(eligible))
	remainder := totalMarks - share*int64(len(eligible))
	for i, p := range eligible {
		amount := share
		if i == 0 {
			amount += remainder
		}
		rewards[p] = amount
	}
	return rewards
}

func distributeDaoMarksFromPaths(mainPath string, totalMarks int64, paths []string) map[string]int64 {
	rewards := make(map[string]int64)
	if totalMarks <= 0 {
		return rewards
	}
	eligiblePaths := eligibleOrdinaryDaoMarkPaths(paths)
	if len(eligiblePaths) == 0 {
		return rewards
	}
	mainPath = strings.TrimSpace(strings.ToLower(mainPath))
	mainPath = strings.TrimSpace(strings.TrimSuffix(mainPath, " path"))
	if mainPath == heavenDaoPath || mainPath == "" {
		mainPath = eligiblePaths[rand.Intn(len(eligiblePaths))]
	}
	rewards[mainPath] = totalMarks
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
	tribulationPenaltyAfter = 24 * time.Hour  // offline/pending hold beyond this causes immediate tribulation failure
)
