package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// RoundDuration is the duration allocated to each tournament round (default 5 minutes)
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
		ID       string
		SeedNum  int
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

// processPendingMatches processes all pending matches in the tournament
func processPendingMatches(tournamentID string) {
	matchProcessingMutex.Lock()
	if isProcessingMatches {
		matchProcessingMutex.Unlock()
		return
	}
	isProcessingMatches = true
	matchProcessingMutex.Unlock()

	defer func() {
		matchProcessingMutex.Lock()
		isProcessingMatches = false
		matchProcessingMutex.Unlock()
	}()

	logger.S.Infof("Death Arena: Starting match processing for tournament %s", tournamentID)

	// Recover any match stuck IN_PROGRESS from previous crashes/restarts back to PENDING
	_, _ = db.DB.Exec(`UPDATE death_arena_matches SET status = 'PENDING' WHERE tournament_id = $1 AND status = 'IN_PROGRESS'`, tournamentID)

	roundStartTime := time.Now()

	for {
		// Check if tournament is still IN_PROGRESS
		var status string
		err := db.DB.QueryRow(`SELECT status FROM death_arena_tournaments WHERE id = $1`, tournamentID).Scan(&status)
		if err != nil || status != "IN_PROGRESS" {
			logger.S.Infof("Death Arena: Tournament %s no longer in progress (status: %s)", tournamentID, status)
			return
		}

		// Find and process next pending match
		var matchID string
		var teamAID string
		var teamBID sql.NullString
		err = db.DB.QueryRow(`
			SELECT id, team_a_id, team_b_id FROM death_arena_matches
			WHERE tournament_id = $1 AND status = 'PENDING'
			ORDER BY round_number, bracket_position
			LIMIT 1
		`, tournamentID).Scan(&matchID, &teamAID, &teamBID)

		if err == sql.ErrNoRows {
			logger.S.Infof("Death Arena: No more pending matches for tournament %s", tournamentID)
			// Check if all matches in current round are finished
			var pendingCount, finishedCount int
			_ = db.DB.QueryRow(`
				SELECT 
					COUNT(*) FILTER (WHERE status = 'PENDING') as pending,
					COUNT(*) FILTER (WHERE status IN ('FINISHED', 'BYE')) as finished
				FROM death_arena_matches WHERE tournament_id = $1 AND round_number = (
					SELECT current_round FROM death_arena_tournaments WHERE id = $1
				)
			`, tournamentID).Scan(&pendingCount, &finishedCount)

			if pendingCount == 0 && finishedCount > 0 {
				// All matches in current round finished, advance to next round
				advanceToNextRound(tournamentID)

				// Check if tournament has advanced to next round or finished
				var currentStatus string
				var currentRound, totalRounds int
				_ = db.DB.QueryRow(`SELECT status, current_round, total_rounds FROM death_arena_tournaments WHERE id = $1`, tournamentID).Scan(&currentStatus, &currentRound, &totalRounds)
				if currentStatus == "IN_PROGRESS" {
					// Round pacing: wait for RoundDuration before starting the next round
					elapsed := time.Since(roundStartTime)
					if elapsed < RoundDuration {
						intermission := RoundDuration - elapsed
						intermissionSec := int(intermission.Seconds())
						logger.S.Infof("Death Arena: Round finished. Intermission of %v before round %d begins", intermission.Round(time.Second), currentRound)

						// Broadcast intermission event to WebSocket
						BroadcastDeathArenaEvent("round_intermission", tournamentID, map[string]interface{}{
							"current_round":     currentRound,
							"total_rounds":      totalRounds,
							"countdown_seconds": intermissionSec,
						})

						// Announce to world chat
						mins := intermissionSec / 60
						secs := intermissionSec % 60
						var timeMsg string
						if mins > 0 {
							timeMsg = fmt.Sprintf("%d phút %02d giây", mins, secs)
						} else {
							timeMsg = fmt.Sprintf("%d giây", secs)
						}
						_, _ = db.DB.Exec(`
							INSERT INTO messages (sender_id, content, channel)
							VALUES ('00000000-0000-0000-0000-000000000000', $1, 'world')
						`, fmt.Sprintf("[Đấu Trường Sinh Tử] Vòng %d đã kết thúc! Vòng %d sẽ chính thức khai chiến sau %s...", currentRound-1, currentRound, timeMsg))

						time.Sleep(intermission)
					}
					roundStartTime = time.Now()
					continue
				}
			} else if pendingCount == 0 && finishedCount == 0 {
				// No matches in current round! Check if tournament has any pending matches at all
				var remainingPending int
				_ = db.DB.QueryRow(`SELECT COUNT(*) FROM death_arena_matches WHERE tournament_id = $1 AND status = 'PENDING'`, tournamentID).Scan(&remainingPending)
				if remainingPending == 0 {
					logger.S.Warnf("Death Arena: Tournament %s has no pending or finished matches in current round, auto-finishing", tournamentID)
					_, _ = db.DB.Exec(`UPDATE death_arena_tournaments SET status = 'FINISHED', finished_at = NOW() WHERE id = $1`, tournamentID)
					invalidateTournamentCache(tournamentID)
					BroadcastDeathArenaEvent("tournament_finished", tournamentID, nil)
					return
				}
			}
			return
		} else if err != nil {
			logger.S.Errorf("Death Arena: Error finding pending match: %v", err)
			return
		}

		// Mark as in progress
		_, _ = db.DB.Exec(`UPDATE death_arena_matches SET status = 'IN_PROGRESS' WHERE id = $1`, matchID)

		var teamBIDPtr *string
		if teamBID.Valid && teamBID.String != "" {
			teamBIDPtr = &teamBID.String
		}

		// Resolve the match
		err = resolveTeamMatch(tournamentID, matchID, teamAID, teamBIDPtr)
		if err != nil {
			logger.S.Errorf("Death Arena: Error resolving match %s: %v", matchID, err)
			// Mark as finished with fallback winner so bracket is never stuck
			_, _ = db.DB.Exec(`UPDATE death_arena_matches SET status = 'FINISHED', winner_team_id = $1, finished_at = NOW() WHERE id = $2`, teamAID, matchID)
			if teamBIDPtr != nil {
				_, _ = db.DB.Exec(`UPDATE death_arena_teams SET eliminated = true WHERE id = $1`, *teamBIDPtr)
			}
		}

		// Small delay between matches
		time.Sleep(2 * time.Second)
	}
}

// resolveTeamMatch resolves a single team match
func resolveTeamMatch(tournamentID, matchID, teamAID string, teamBID *string) error {
	// Check if this is a BYE
	if teamBID == nil || *teamBID == "" {
		_, err := db.DB.Exec(`
			UPDATE death_arena_matches SET status = 'BYE', winner_team_id = $1, finished_at = NOW()
			WHERE id = $2
		`, teamAID, matchID)
		invalidateTournamentCache(tournamentID)
		BroadcastDeathArenaEvent("match_updated", tournamentID, map[string]interface{}{
			"match_id":       matchID,
			"winner_team_id": teamAID,
			"status":         "BYE",
		})
		return err
	}

	// Load team members (alive only) without holding a transaction
	teamAMembers, err := loadTeamMembersForCombat(db.DB, teamAID)
	if err != nil {
		return err
	}

	teamBMembers, err := loadTeamMembersForCombat(db.DB, *teamBID)
	if err != nil {
		return err
	}

	// If either team has no members, the other wins by default
	if len(teamAMembers) == 0 {
		tx, err := db.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		_, err = tx.Exec(`
			UPDATE death_arena_matches SET status = 'FINISHED', winner_team_id = $1, finished_at = NOW()
			WHERE id = $2
		`, *teamBID, matchID)
		if err != nil {
			return err
		}
		_, _ = tx.Exec(`UPDATE death_arena_teams SET eliminated = true, eliminated_at_round = (
			SELECT current_round FROM death_arena_tournaments WHERE id = $1
		) WHERE id = $2`, tournamentID, teamAID)
		invalidateTournamentCache(tournamentID)
		if err := tx.Commit(); err != nil {
			return err
		}
		BroadcastDeathArenaEvent("match_updated", tournamentID, map[string]interface{}{
			"match_id":       matchID,
			"winner_team_id": *teamBID,
			"status":         "FINISHED",
		})
		return nil
	}

	if len(teamBMembers) == 0 {
		tx, err := db.DB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()
		_, err = tx.Exec(`
			UPDATE death_arena_matches SET status = 'FINISHED', winner_team_id = $1, finished_at = NOW()
			WHERE id = $2
		`, teamAID, matchID)
		if err != nil {
			return err
		}
		_, _ = tx.Exec(`UPDATE death_arena_teams SET eliminated = true, eliminated_at_round = (
			SELECT current_round FROM death_arena_tournaments WHERE id = $1
		) WHERE id = $2`, tournamentID, *teamBID)
		invalidateTournamentCache(tournamentID)
		if err := tx.Commit(); err != nil {
			return err
		}
		BroadcastDeathArenaEvent("match_updated", tournamentID, map[string]interface{}{
			"match_id":       matchID,
			"winner_team_id": teamAID,
			"status":         "FINISHED",
		})
		return nil
	}

	// Build combat state for 6v6 purely in-memory
	simState := &simulator.CombatState{
		Turn:        0,
		Seed:        time.Now().UnixNano(),
		VirtualMode: true, // No resource consumption
	}

	// Build team A combatants
	allySlots := [][2]int{
		{2, 3}, {2, 1}, {1, 1}, {3, 1}, {2, 2}, {1, 2}, {3, 2}, {1, 3}, {3, 3},
	}

	for i, member := range teamAMembers {
		if i >= 9 {
			break
		}
		simAlly, err := buildTeamCombatant(member)
		if err != nil {
			logger.S.Warnf("Failed to build combatant for team A member %s: %v", member.PlayerID, err)
			continue
		}
		simAlly.Kind = "Player"
		if i < len(allySlots) {
			simAlly.GridX = allySlots[i][0]
			simAlly.GridY = allySlots[i][1]
		}
		simState.Allies = append(simState.Allies, simAlly)
	}

	// Build team B combatants
	for i, member := range teamBMembers {
		if i >= 9 {
			break
		}
		simEnemy, err := buildTeamCombatant(member)
		if err != nil {
			logger.S.Warnf("Failed to build combatant for team B member %s: %v", member.PlayerID, err)
			continue
		}
		simEnemy.Kind = "Enemy"
		if i < len(allySlots) {
			simEnemy.GridX = allySlots[i][0]
			simEnemy.GridY = allySlots[i][1]
		}
		simState.Enemies = append(simState.Enemies, simEnemy)
	}

	// Run simulation purely in-memory (zero DB transaction lock)
	logs := simulator.Simulate(simState)

	// Determine winner
	var winnerTeamID string
	teamAAlive := false
	teamBAlive := false

	for _, ally := range simState.Allies {
		if ally.HP > 0 {
			teamAAlive = true
			break
		}
	}
	for _, enemy := range simState.Enemies {
		if enemy.HP > 0 {
			teamBAlive = true
			break
		}
	}

	if teamAAlive && !teamBAlive {
		winnerTeamID = teamAID
	} else if teamBAlive && !teamAAlive {
		winnerTeamID = *teamBID
	} else if teamAAlive && teamBAlive {
		// Both alive - team with more total HP percentage wins
		hpA := 0.0
		for _, ally := range simState.Allies {
			if ally.MaxHP > 0 {
				hpA += ally.HP / ally.MaxHP
			}
		}
		hpB := 0.0
		for _, enemy := range simState.Enemies {
			if enemy.MaxHP > 0 {
				hpB += enemy.HP / enemy.MaxHP
			}
		}
		if hpA >= hpB {
			winnerTeamID = teamAID
		} else {
			winnerTeamID = *teamBID
		}
	} else {
		// Both dead - draw, team A wins by default
		winnerTeamID = teamAID
	}

	// Store combat log JSON
	combatLogJSON, _ := json.Marshal(logs)

	// Fast write transaction (< 5ms)
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update match result
	_, err = tx.Exec(`
		UPDATE death_arena_matches 
		SET status = 'FINISHED', winner_team_id = $1, combat_log = $2, finished_at = NOW()
		WHERE id = $3
	`, winnerTeamID, combatLogJSON, matchID)
	if err != nil {
		return err
	}

	// Update HP for all members and mark eliminated ones
	for _, ally := range simState.Allies {
		newHP := ally.HP
		if newHP < 0 {
			newHP = 0
		}
		_, _ = tx.Exec(`
			UPDATE death_arena_team_members 
			SET current_hp = $1, current_mana = $2
			WHERE player_id = $3 AND team_id = $4
		`, newHP, ally.Mana, ally.ID, teamAID)

		// Mark eliminated if HP <= 0
		if ally.HP <= 0 {
			_, _ = tx.Exec(`
				UPDATE death_arena_team_members 
				SET eliminated = true, eliminated_at_round = (
					SELECT current_round FROM death_arena_tournaments WHERE id = $1
				)
				WHERE player_id = $2 AND team_id = $3
			`, tournamentID, ally.ID, teamAID)
		}
	}

	for _, enemy := range simState.Enemies {
		newHP := enemy.HP
		if newHP < 0 {
			newHP = 0
		}
		_, _ = tx.Exec(`
			UPDATE death_arena_team_members 
			SET current_hp = $1, current_mana = $2
			WHERE player_id = $3 AND team_id = $4
		`, newHP, enemy.Mana, enemy.ID, *teamBID)

		// Mark eliminated if HP <= 0
		if enemy.HP <= 0 {
			_, _ = tx.Exec(`
				UPDATE death_arena_team_members 
				SET eliminated = true, eliminated_at_round = (
					SELECT current_round FROM death_arena_tournaments WHERE id = $1
				)
				WHERE player_id = $2 AND team_id = $3
			`, tournamentID, enemy.ID, *teamBID)
		}
	}

	// Eliminate losing team
	loserTeamID := *teamBID
	if winnerTeamID == *teamBID {
		loserTeamID = teamAID
	}

	_, err = tx.Exec(`
		UPDATE death_arena_teams 
		SET eliminated = true, eliminated_at_round = (
			SELECT current_round FROM death_arena_tournaments WHERE id = $1
		)
		WHERE id = $2
	`, tournamentID, loserTeamID)
	if err != nil {
		return err
	}

	// Award round rewards to winning team members
	reward := roundRewards[1]
	_ = tx.QueryRow(`SELECT reward_stones_per_member FROM death_arena_matches WHERE id = $1`, matchID).Scan(&reward)

	for _, member := range getTeamMembers(tx, winnerTeamID) {
		_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`, reward, member)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	invalidateTournamentCache(tournamentID)
	BroadcastDeathArenaEvent("match_updated", tournamentID, map[string]interface{}{
		"match_id":       matchID,
		"winner_team_id": winnerTeamID,
		"status":         "FINISHED",
	})
	return nil
}

// advanceToNextRound advances the tournament to the next round
func advanceToNextRound(tournamentID string) {
	tx, err := db.DB.Begin()
	if err != nil {
		logger.S.Errorf("Death Arena: Failed to begin tx for round advance: %v", err)
		return
	}
	defer tx.Rollback()

	// Get current round
	var currentRound, totalRounds int
	err = tx.QueryRow(`
		SELECT current_round, total_rounds FROM death_arena_tournaments WHERE id = $1
	`, tournamentID).Scan(&currentRound, &totalRounds)
	if err != nil {
		logger.S.Errorf("Death Arena: Failed to get current round: %v", err)
		return
	}

	// If this was the final round, tournament is finished
	if currentRound >= totalRounds {
		// Find champion team (prefer battle winner, fallback to bye team)
		var championTeamID string
		err = tx.QueryRow(`
			SELECT COALESCE(winner_team_id, team_a_id) FROM death_arena_matches
			WHERE tournament_id = $1 AND round_number = $2 AND status IN ('FINISHED', 'BYE')
			ORDER BY CASE WHEN status = 'FINISHED' THEN 1 ELSE 2 END
			LIMIT 1
		`, tournamentID, currentRound).Scan(&championTeamID)
		if err != nil {
			logger.S.Warnf("Death Arena: Failed to find champion in final round: %v, marking FINISHED", err)
			_, _ = tx.Exec(`UPDATE death_arena_tournaments SET status = 'FINISHED', finished_at = NOW() WHERE id = $1`, tournamentID)
			_ = tx.Commit()
			invalidateTournamentCache(tournamentID)
			BroadcastDeathArenaEvent("tournament_finished", tournamentID, nil)
			return
		}

		// Update tournament
		_, _ = tx.Exec(`
			UPDATE death_arena_tournaments 
			SET status = 'FINISHED', champion_team_id = $1, finished_at = NOW()
			WHERE id = $2
		`, championTeamID, tournamentID)

		// Award champion rewards
		if err := awardChampionRewards(tx, tournamentID, championTeamID, 1); err != nil {
			logger.S.Errorf("Death Arena: Failed to award champion rewards: %v", err)
		}

		// Send notification
		_, _ = tx.Exec(`
			INSERT INTO messages (sender_id, content, channel)
			VALUES ('00000000-0000-0000-0000-000000000000', $1, 'world')
		`, fmt.Sprintf("[Đấu Trường Sinh Tử] Giải đấu đã kết thúc! Chúc mừng vô địch!"))

		_ = tx.Commit()
		invalidateTournamentCache(tournamentID)
		BroadcastDeathArenaEvent("tournament_finished", tournamentID, map[string]interface{}{
			"champion_team_id": championTeamID,
		})
		logger.S.Infof("Death Arena: Tournament %s finished with champion %s", tournamentID, championTeamID)
		return
	}

	// Get winners from current round
	rows, err := tx.Query(`
		SELECT winner_team_id FROM death_arena_matches
		WHERE tournament_id = $1 AND round_number = $2 AND status = 'FINISHED' AND winner_team_id IS NOT NULL
		ORDER BY bracket_position
	`, tournamentID, currentRound)
	if err != nil {
		logger.S.Errorf("Death Arena: Failed to get round winners: %v", err)
		return
	}
	defer rows.Close()

	var winners []string
	for rows.Next() {
		var winnerID string
		if errScan := rows.Scan(&winnerID); errScan == nil && winnerID != "" {
			winners = append(winners, winnerID)
		}
	}

	// Also get BYE winners from current round
	rows2, err := tx.Query(`
		SELECT COALESCE(winner_team_id, team_a_id) FROM death_arena_matches
		WHERE tournament_id = $1 AND round_number = $2 AND status = 'BYE'
		ORDER BY bracket_position
	`, tournamentID, currentRound)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var teamID string
			if errScan := rows2.Scan(&teamID); errScan == nil && teamID != "" {
				winners = append(winners, teamID)
			}
		}
	}

	if len(winners) <= 1 {
		// Only 1 or 0 winners left - crown champion or finish immediately
		if len(winners) == 1 {
			championTeamID := winners[0]
			_, _ = tx.Exec(`
				UPDATE death_arena_tournaments 
				SET status = 'FINISHED', champion_team_id = $1, finished_at = NOW()
				WHERE id = $2
			`, championTeamID, tournamentID)

			if err := awardChampionRewards(tx, tournamentID, championTeamID, 1); err != nil {
				logger.S.Errorf("Death Arena: Failed to award champion rewards: %v", err)
			}

			_, _ = tx.Exec(`
				INSERT INTO messages (sender_id, content, channel)
				VALUES ('00000000-0000-0000-0000-000000000000', $1, 'world')
			`, fmt.Sprintf("[Đấu Trường Sinh Tử] Giải đấu đã kết thúc! Chúc mừng vô địch!"))

			_ = tx.Commit()
			invalidateTournamentCache(tournamentID)
			BroadcastDeathArenaEvent("tournament_finished", tournamentID, map[string]interface{}{
				"champion_team_id": championTeamID,
			})
			logger.S.Infof("Death Arena: Tournament %s finished early with champion %s", tournamentID, championTeamID)
			return
		}
		_, _ = tx.Exec(`UPDATE death_arena_tournaments SET status = 'FINISHED', finished_at = NOW() WHERE id = $1`, tournamentID)
		_ = tx.Commit()
		invalidateTournamentCache(tournamentID)
		BroadcastDeathArenaEvent("tournament_finished", tournamentID, nil)
		return
	}

	// Create next round matches
	nextRound := currentRound + 1
	positions := []int{}
	for i := 0; i < len(winners)/2; i++ {
		positions = append(positions, i+1)
	}

	for i := 0; i < len(winners); i += 2 {
		if i+1 < len(winners) {
			reward := roundRewards[nextRound]
			if reward == 0 {
				reward = roundRewards[len(roundRewards)]
			}
			_, err = tx.Exec(`
				INSERT INTO death_arena_matches (tournament_id, round_number, bracket_position, team_a_id, team_b_id, status, reward_stones_per_member)
				VALUES ($1, $2, $3, $4, $5, 'PENDING', $6)
			`, tournamentID, nextRound, (i/2)+1, winners[i], winners[i+1], reward)
		} else {
			// Odd winner - BYE to next round
			_, err = tx.Exec(`
				INSERT INTO death_arena_matches (tournament_id, round_number, bracket_position, team_a_id, winner_team_id, status, reward_stones_per_member)
				VALUES ($1, $2, $3, $4, $4, 'BYE', 0)
			`, tournamentID, nextRound, (i/2)+1, winners[i])
		}
		if err != nil {
			logger.S.Errorf("Death Arena: Failed to create next round match: %v", err)
			return
		}
	}

	// Update current round
	_, err = tx.Exec(`
		UPDATE death_arena_tournaments SET current_round = $1 WHERE id = $2
	`, nextRound, tournamentID)
	if err != nil {
		logger.S.Errorf("Death Arena: Failed to update current round: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Death Arena: Failed to commit round advance: %v", err)
		return
	}
	invalidateTournamentCache(tournamentID)
	BroadcastDeathArenaEvent("round_advanced", tournamentID, map[string]interface{}{
		"current_round": nextRound,
	})

	// Process next round matches if not already running in current worker loop
	if !IsProcessingMatches() {
		go processPendingMatches(tournamentID)
	}
}

// loadTeamMembersForCombat loads team members for combat
func loadTeamMembersForCombat(q db.Queryer, teamID string) ([]*teamMemberCombatant, error) {
	rows, err := q.Query(`
		SELECT dtm.player_id, dtm.current_hp, dtm.max_hp, dtm.current_mana, dtm.max_mana,
		       p.username, p.rank, p.sub_rank, p.hp, p.base_hp, p.current_mana, p.max_mana,
		       COALESCE(p.race, ''), COALESCE(p.constitution, ''),
		       `+db.GuInventoryAggregateEquipped("p.id")+`,
		       COALESCE(ia.dao_marks, '{}'::jsonb),
		       COALESCE(p.path_masteries, '{}'::jsonb),
		       COALESCE(p.combat_settings, '{}'::jsonb)
		FROM death_arena_team_members dtm
		JOIN players p ON dtm.player_id = p.id
		LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
		WHERE dtm.team_id = $1 AND dtm.eliminated = false AND dtm.current_hp > 0
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*teamMemberCombatant
	for rows.Next() {
		var m teamMemberCombatant
		if errScan := rows.Scan(
			&m.PlayerID, &m.CurrentHP, &m.MaxHP, &m.CurrentMana, &m.MaxMana,
			&m.Username, &m.Rank, &m.SubRank, &m.PlayerHP, &m.BaseHP, &m.PlayerMana, &m.MaxManaPlayer,
			&m.Race, &m.Constitution,
			&m.GuInventory, &m.DaoMarks, &m.PathMasteries, &m.CombatSettings,
		); errScan != nil {
			logger.S.Errorf("Death Arena: Failed to scan team member %s: %v", m.PlayerID, errScan)
			return nil, errScan
		}
		members = append(members, &m)
	}

	return members, nil
}

type teamMemberCombatant struct {
	PlayerID      string
	CurrentHP     float64
	MaxHP         float64
	CurrentMana   float64
	MaxMana       float64
	Username      string
	Rank          int
	SubRank       int
	PlayerHP      float64
	BaseHP        float64
	PlayerMana    float64
	MaxManaPlayer float64
	Race          string
	Constitution  string
	GuInventory   models.GuInventory
	DaoMarks      models.DaoMarksMap
	PathMasteries models.PathMasteriesMap
	CombatSettings models.AutoCombatConfig
}

// buildTeamCombatant builds a combatant from team member data
func buildTeamCombatant(member *teamMemberCombatant) (*simulator.CombatantAuto, error) {
	// Calculate stats
	player := &models.Player{}
	player.ID = uuid.MustParse(member.PlayerID)
	player.Username = member.Username
	player.Rank = member.Rank
	player.SubRank = member.SubRank
	player.HP = member.CurrentHP
	player.BaseHP = member.BaseHP
	player.CurrentMana = member.CurrentMana
	player.MaxMana = member.MaxMana
	player.Race = member.Race
	player.Constitution = member.Constitution
	player.GuInventory = member.GuInventory
	player.DaoMarks = member.DaoMarks
	player.PathMasteries = member.PathMasteries
	player.CombatSettings = member.CombatSettings

	playerStats := CalculatePlayerStats(player)
	actor := models.CombatantFromPlayer(player, playerStats)

	// Build actions from equipped Gus
	var allActions []simulator.SimAction
	for _, gu := range player.GuInventory {
		if gu.Equipped && gu.Status == "ACTIVE" && gu.Hunger > 0 {
			if spec, ok := models.GetGu(gu.GuID, gu.Rank); ok {
				if hasCombatActiveEffects(&spec) {
					act, err := buildSimActionFromGu(&actor, &spec, "GU_SKILL")
					if err != nil {
						continue
					}
					act.SourceID = gu.GuID
					allActions = append(allActions, act)
				}
			}
		}
	}

	// Basic attack fallback
	allActions = append(allActions, simulator.SimAction{
		ActionType: "BASIC_ATTACK",
		Name:       "Đánh thường",
		Damage:     playerStats["attack"],
		ManaCost:   0,
	})

	// Build DaoMarks
	daoMarks := make(simulator.DaoMarks)
	for k, v := range member.DaoMarks {
		daoMarks[k] = v
	}

	// Calculate max HP
	baseHP := member.BaseHP + playerStats["hp"]
	maxHP := baseHP
	if maxHP <= 0 {
		maxHP = 100
	}

	// Use current HP from tournament state
	simHP := member.CurrentHP
	if simHP > maxHP {
		simHP = maxHP
	}
	if simHP <= 0 {
		simHP = maxHP
	}

	simMana := member.CurrentMana
	if simMana > member.MaxMana {
		simMana = member.MaxMana
	}

	charFX := gamefx.Get(player)

	simCombatant := &simulator.CombatantAuto{
		ID:                   member.PlayerID,
		Name:                 member.Username,
		Kind:                 "Player",
		Rank:                 member.Rank,
		SubRank:              member.SubRank,
		MaxHP:                maxHP,
		HP:                   simHP,
		MaxMana:              member.MaxMana,
		Mana:                 simMana,
		BaseDmg:              playerStats["attack"],
		BaseDef:              playerStats["defense"],
		Speed:                playerStats["speed"],
		DaoMarks:             daoMarks,
		PathMasteries:        member.PathMasteries,
		PathMasteryGainLimit: gamefx.PathMasteryBattleGainLimit(member.Rank),
		DaoBonus:             charFX.DaoNganBonus,
		PathDamageBonus:      charFX.PathDamageBonus,
		PathResistance:       charFX.PathResistanceBonus,
		HealingReceived:      charFX.HealingReceivedBonus,
		MentalResistance:     charFX.MentalResistanceBonus,
		Actions:              allActions,
		AutoHealHP:           false,
		HealThreshold:        0.3,
		AutoRestoreMana:      true,
		ManaThreshold:        0.2,
		HealCharges:          0,
		HealPotency:          0,
		PrimevalStones:       999999,
		ImmortalStones:       999999,
		RefineManaTier:       gamefx.RefineManaTier(member.Rank, member.GuInventory),
		Loadout: &simulator.CombatLoadout{
			GuIds: func() []string {
				var ids []string
				for _, gu := range player.GuInventory {
					if gu.Equipped && gu.Status == "ACTIVE" && gu.Hunger > 0 {
						ids = append(ids, gu.GuID)
					}
				}
				return ids
			}(),
		},
	}

	return simCombatant, nil
}

// awardChampionRewards awards rewards to top 3 teams
func awardChampionRewards(tx *sql.Tx, tournamentID, championTeamID string, placement int) error {
	// Get all teams ordered by placement
	teams := getTeamPlacements(tx, tournamentID)

	placementTitles := map[int]string{
		1: "Quán Quân Đấu Trường Sinh Tử",
		2: "Á Quân Đấu Trường Sinh Tử",
		3: "Quý Quân Đấu Trường Sinh Tử",
	}

	placementStones := map[int]int{
		1: 5000,
		2: 3000,
		3: 1500,
	}

	placementImmortalStones := map[int]int{
		1: 500,
		2: 300,
		3: 150,
	}

	// Gu rewards for top 3 (guID, rank)
	type guReward struct {
		guID string
		rank int
	}
	placementGuRewards := map[int]guReward{
		1: {guID: "GU_LIFESPAN_500", rank: 5},
		2: {guID: "GU_LIFESPAN_100", rank: 5},
		3: {guID: "GU_LIFESPAN_50", rank: 4},
	}

	for i, teamID := range teams {
		if i >= 3 {
			break
		}
		place := i + 1
		title := placementTitles[place]
		stones := placementStones[place]
		immortalStones := placementImmortalStones[place]
		gu := placementGuRewards[place]

		// Get team members
		members := getTeamMembers(tx, teamID)
		for _, memberID := range members {
			_, err := tx.Exec(`
				INSERT INTO death_arena_champion_rewards 
				(tournament_id, player_id, placement, team_id, title, reward_stones, reward_immortal_stones, reward_gu_id, reward_gu_rank)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`, tournamentID, memberID, place, teamID, title, stones, immortalStones, gu.guID, gu.rank)
			if err != nil {
				logger.S.Errorf("Failed to insert champion reward: %v", err)
			}

			// Award Gu directly to player
			newGu := models.GuInstance{
				InstanceID: fmt.Sprintf("death_arena_%s_%d_%s", tournamentID, place, memberID[:8]),
				GuID:       gu.guID,
				Rank:       gu.rank,
				Hunger:     100,
				Status:     "ACTIVE",
				Equipped:   false,
				IsVital:    false,
			}
			if err := db.AddPlayerGu(tx, memberID, newGu); err != nil {
				logger.S.Errorf("Failed to award Gu %s rank %d to player %s: %v", gu.guID, gu.rank, memberID, err)
			}
		}

		// Update team final placement
		_, _ = tx.Exec(`UPDATE death_arena_teams SET final_placement = $1 WHERE id = $2`, place, teamID)
	}

	return nil
}

// getTeamPlacements returns team IDs ordered by placement
func getTeamPlacements(tx *sql.Tx, tournamentID string) []string {
	// First get champion
	var championID string
	err := tx.QueryRow(`
		SELECT champion_team_id FROM death_arena_tournaments WHERE id = $1
	`, tournamentID).Scan(&championID)
	if err != nil || championID == "" {
		return nil
	}

	result := []string{championID}

	// Get runner-up from final match (the loser of the final match)
	var runnerUp *string
	err = tx.QueryRow(`
		SELECT CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END
		FROM death_arena_matches
		WHERE tournament_id = $1 AND round_number = (
			SELECT total_rounds FROM death_arena_tournaments WHERE id = $1
		) AND status = 'FINISHED'
		LIMIT 1
	`, tournamentID).Scan(&runnerUp)
	if err == nil && runnerUp != nil && *runnerUp != "" {
		result = append(result, *runnerUp)
	}

	// Get 3rd place from semi-final losers
	runnerUpID := ""
	if runnerUp != nil {
		runnerUpID = *runnerUp
	}
	rows, err := tx.Query(`
		SELECT DISTINCT 
			CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END AS loser_id
		FROM death_arena_matches
		WHERE tournament_id = $1 
		AND round_number = (SELECT total_rounds - 1 FROM death_arena_tournaments WHERE id = $1)
		AND status = 'FINISHED'
		AND (CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END) IS NOT NULL
		AND (CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END) != $2
		AND (CASE WHEN winner_team_id = team_a_id THEN team_b_id ELSE team_a_id END) != $3
		LIMIT 1
	`, tournamentID, championID, runnerUpID)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			var third string
			if errScan := rows.Scan(&third); errScan == nil && third != "" {
				result = append(result, third)
			}
		}
	}

	return result
}

// getTeamMembers returns player IDs for a team
func getTeamMembers(tx *sql.Tx, teamID string) []string {
	rows, err := tx.Query(`SELECT player_id FROM death_arena_team_members WHERE team_id = $1`, teamID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var members []string
	for rows.Next() {
		var id string
		if errScan := rows.Scan(&id); errScan == nil {
			members = append(members, id)
		}
	}
	return members
}