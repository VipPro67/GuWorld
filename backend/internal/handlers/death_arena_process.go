package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

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
