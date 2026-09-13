package handlers

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

var RoundDuration = 5 * time.Minute

// Round rewards by round number (stones per member)
var roundRewards = map[int]int{
	1: 50,
	2: 100,
	3: 200,
	4: 400,
	5: 800,
	6: 1500,
}

// nextPowerOf2 returns the next power of 2 >= n
func nextPowerOf2(n int) int {
	if n <= 1 {
		return 1
	}
	result := 1
	for result < n {
		result *= 2
	}
	return result
}

// GenerateBracket creates the tournament bracket from registered teams
func GenerateBracket(tournamentID string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get all teams
	rows, err := tx.Query(`
		SELECT dt.id, dt.seed_number
		FROM death_arena_teams dt
		WHERE dt.tournament_id = $1
		ORDER BY dt.seed_number
	`, tournamentID)
	if err != nil {
		return err
	}
	defer rows.Close()

	type teamInfo struct {
		ID      string
		SeedNum int
	}
	var teams []teamInfo
	for rows.Next() {
		var t teamInfo
		if errScan := rows.Scan(&t.ID, &t.SeedNum); errScan == nil {
			teams = append(teams, t)
		}
	}

	if len(teams) < 2 {
		// Auto-win if only 1 team
		if len(teams) == 1 {
			_, _ = tx.Exec(`
				UPDATE death_arena_tournaments 
				SET status = 'FINISHED', champion_team_id = $1, finished_at = NOW(), total_teams = 1
				WHERE id = $2
			`, teams[0].ID, tournamentID)

			// Award champion rewards
			if err := awardChampionRewards(tx, tournamentID, teams[0].ID, 1); err != nil {
				logger.S.Errorf("Failed to award champion rewards: %v", err)
			}
		} else {
			// 0 teams registered - tournament cannot proceed
			_, _ = tx.Exec(`
				UPDATE death_arena_tournaments 
				SET status = 'FINISHED', finished_at = NOW(), total_teams = 0
				WHERE id = $1
			`, tournamentID)
		}
		return tx.Commit()
	}

	// Shuffle teams randomly
	rand.Shuffle(len(teams), func(i, j int) {
		teams[i], teams[j] = teams[j], teams[i]
	})

	// Reassign seed numbers after shuffle
	for i := range teams {
		teams[i].SeedNum = i + 1
		_, _ = tx.Exec(`UPDATE death_arena_teams SET seed_number = $1 WHERE id = $2`, i+1, teams[i].ID)
	}

	// Calculate total rounds
	totalRounds := int(math.Ceil(math.Log2(float64(len(teams)))))

	// Calculate bracket slots and byes
	numSlots := nextPowerOf2(len(teams))
	byes := numSlots - len(teams)

	// Create round 1 matches
	matchPosition := 0
	for i := 0; i < numSlots/2; i++ {
		matchPosition++

		var teamAID, teamBID string
		var status string

		if i < byes {
			// BYE - team advances automatically
			teamAID = teams[i].ID
			status = "BYE"

			// Mark match as BYE
			_, err = tx.Exec(`
				INSERT INTO death_arena_matches (tournament_id, round_number, bracket_position, team_a_id, winner_team_id, status, reward_stones_per_member)
				VALUES ($1, 1, $2, $3, $3, 'BYE', 0)
			`, tournamentID, matchPosition, teamAID)
			if err != nil {
				return err
			}
		} else {
			// Regular match
			idx1 := i
			idx2 := numSlots - 1 - i
			if idx1 < len(teams) {
				teamAID = teams[idx1].ID
			} else {
				// This shouldn't happen with proper bye calculation
				continue
			}
			if idx2 < len(teams) {
				teamBID = teams[idx2].ID
				status = "PENDING"
			} else {
				// Team B is a BYE (shouldn't happen with our calculation)
				status = "BYE"
			}

			if status == "BYE" {
				_, err = tx.Exec(`
					INSERT INTO death_arena_matches (tournament_id, round_number, bracket_position, team_a_id, winner_team_id, status, reward_stones_per_member)
					VALUES ($1, 1, $2, $3, $3, 'BYE', 0)
				`, tournamentID, matchPosition, teamAID)
			} else {
				reward := roundRewards[1]
				_, err = tx.Exec(`
					INSERT INTO death_arena_matches (tournament_id, round_number, bracket_position, team_a_id, team_b_id, status, reward_stones_per_member)
					VALUES ($1, 1, $2, $3, $4, 'PENDING', $5)
				`, tournamentID, matchPosition, teamAID, teamBID, reward)
			}
			if err != nil {
				return err
			}
		}
	}

	// Update tournament status
	_, err = tx.Exec(`
		UPDATE death_arena_tournaments 
		SET status = 'LOCKED', total_rounds = $1, current_round = 1, total_teams = $2, started_at = NOW()
		WHERE id = $3
	`, totalRounds, len(teams), tournamentID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// StartTournament transitions tournament from LOCKED to IN_PROGRESS and starts first matches
func StartTournament(tournamentID string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update status only if tournament is currently LOCKED and has >= 2 teams
	res, err := tx.Exec(`
		UPDATE death_arena_tournaments 
		SET status = 'IN_PROGRESS' 
		WHERE id = $1 AND status = 'LOCKED' AND total_teams >= 2
	`, tournamentID)
	if err != nil {
		return err
	}
	rowsAff, _ := res.RowsAffected()
	if rowsAff == 0 {
		return nil // Tournament not in LOCKED status or not enough teams (already finished)
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	invalidateTournamentCache(tournamentID)
	BroadcastDeathArenaEvent("tournament_started", tournamentID, nil)

	// Start processing pending matches
	go processPendingMatches(tournamentID)

	return nil
}

func invalidateTournamentCache(tournamentID string) {
	_ = cache.Delete("death_arena:bracket:current")
	_ = cache.Delete("death_arena:bracket:" + tournamentID)
	_ = cache.Delete("death_arena:current_tournament_meta")
}

var (
	matchProcessingMutex sync.Mutex
	isProcessingMatches  bool
)

// IsProcessingMatches returns whether a match processing worker is currently active
func IsProcessingMatches() bool {
	matchProcessingMutex.Lock()
	defer matchProcessingMutex.Unlock()
	return isProcessingMatches
}
