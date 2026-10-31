package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// Per-tournament processing guards: several rank brackets run concurrently, so
// unlike death_arena's single global mutex we track each tournament separately.
var (
	martialProcessingMu sync.Mutex
	martialProcessing   = map[string]bool{}
)

func tryStartMartialProcessing(tournamentID string) bool {
	martialProcessingMu.Lock()
	defer martialProcessingMu.Unlock()
	if martialProcessing[tournamentID] {
		return false
	}
	martialProcessing[tournamentID] = true
	return true
}

func stopMartialProcessing(tournamentID string) {
	martialProcessingMu.Lock()
	delete(martialProcessing, tournamentID)
	martialProcessingMu.Unlock()
}

// processPendingMartialMatches resolves every pending match in a bracket, advancing
// rounds until the tournament finishes. Each rank bracket runs in its own goroutine.
func processPendingMartialMatches(tournamentID string) {
	if !tryStartMartialProcessing(tournamentID) {
		return
	}
	defer stopMartialProcessing(tournamentID)

	logger.S.Infof("Martial Arena: processing matches for tournament %s", tournamentID)

	// Reopen only matches whose realtime session was lost during a restart.
	_, _ = db.DB.Exec(`UPDATE martial_matches m SET status='PENDING'
		WHERE m.tournament_id=$1 AND m.status='IN_PROGRESS' AND NOT EXISTS (
			SELECT 1 FROM realtime_combat_sessions s WHERE s.mode='tournament' AND s.status='active'
			AND s.payload->>'tournament_id'=$1 AND s.payload->>'match_id'=m.id::text)`, tournamentID)

	for {
		var status string
		if err := db.DB.QueryRow(`SELECT status FROM martial_tournaments WHERE id = $1`, tournamentID).Scan(&status); err != nil || status != "IN_PROGRESS" {
			return
		}

		var matchID, combatantAID string
		var combatantBID sql.NullString
		err := db.DB.QueryRow(`
			SELECT id, combatant_a_id, combatant_b_id FROM martial_matches
			WHERE tournament_id = $1 AND status = 'PENDING'
			ORDER BY round_number, bracket_position
			LIMIT 1
		`, tournamentID).Scan(&matchID, &combatantAID, &combatantBID)

		if err == sql.ErrNoRows {
			var activeCount int
			_ = db.DB.QueryRow(`SELECT COUNT(*) FROM martial_matches WHERE tournament_id=$1 AND status='IN_PROGRESS'`, tournamentID).Scan(&activeCount)
			if activeCount > 0 {
				settleExpiredTournamentDisconnects(context.Background(), tournamentID, "martial_arena")
				time.Sleep(time.Second)
				continue
			}
			var pending, finished int
			_ = db.DB.QueryRow(`
				SELECT COUNT(*) FILTER (WHERE status = 'PENDING'),
				       COUNT(*) FILTER (WHERE status IN ('FINISHED','BYE'))
				FROM martial_matches
				WHERE tournament_id = $1 AND round_number = (SELECT current_round FROM martial_tournaments WHERE id = $1)
			`, tournamentID).Scan(&pending, &finished)

			if pending == 0 && finished > 0 {
				advanceMartialRound(tournamentID)
				var st string
				_ = db.DB.QueryRow(`SELECT status FROM martial_tournaments WHERE id = $1`, tournamentID).Scan(&st)
				if st == "IN_PROGRESS" {
					BroadcastMartialArenaEvent("round_intermission", tournamentID, nil)
					time.Sleep(RoundDuration)
					continue
				}
			}
			return
		} else if err != nil {
			logger.S.Errorf("Martial Arena: error finding pending match for %s: %v", tournamentID, err)
			return
		}

		_, _ = db.DB.Exec(`UPDATE martial_matches SET status = 'IN_PROGRESS' WHERE id = $1`, matchID)

		var bPtr *string
		if combatantBID.Valid && combatantBID.String != "" {
			bPtr = &combatantBID.String
		}

		if err := resolveMartialMatch(tournamentID, matchID, combatantAID, bPtr); err != nil {
			logger.S.Errorf("Martial Arena: error resolving match %s: %v", matchID, err)
			_, _ = db.DB.Exec(`UPDATE martial_matches SET status='PENDING' WHERE id=$1 AND status='IN_PROGRESS'`, matchID)
			time.Sleep(2 * time.Second)
			continue
		}

		time.Sleep(2 * time.Second)
	}
}

// resolveMartialMatch starts one 1v1 realtime arena match and records the result.
func resolveMartialMatch(tournamentID, matchID, combatantAID string, combatantBID *string) error {
	if combatantBID == nil || *combatantBID == "" {
		_, err := db.DB.Exec(`UPDATE martial_matches SET status = 'BYE', winner_combatant_id = $1, finished_at = NOW() WHERE id = $2`, combatantAID, matchID)
		invalidateMartialCache(tournamentID)
		BroadcastMartialArenaEvent("match_updated", tournamentID, map[string]interface{}{
			"match_id":            matchID,
			"winner_combatant_id": combatantAID,
			"status":              "BYE",
		})
		return err
	}

	memberA, err := loadMartialCombatant(db.DB, combatantAID)
	if err != nil {
		return err
	}
	memberB, err := loadMartialCombatant(db.DB, *combatantBID)
	if err != nil {
		return err
	}

	return startRealtimeMartialArenaMatch(tournamentID, matchID, combatantAID, *combatantBID, memberA, memberB)
}

// advanceMartialRound creates next-round matches from the current winners, or crowns
// the champion when the final round is complete.
func advanceMartialRound(tournamentID string) {
	tx, err := db.DB.Begin()
	if err != nil {
		logger.S.Errorf("Martial Arena: failed to begin round advance: %v", err)
		return
	}
	defer tx.Rollback()

	var currentRound, totalRounds int
	if err = tx.QueryRow(`SELECT current_round, total_rounds FROM martial_tournaments WHERE id = $1`, tournamentID).Scan(&currentRound, &totalRounds); err != nil {
		logger.S.Errorf("Martial Arena: failed to get round state: %v", err)
		return
	}

	if currentRound >= totalRounds {
		finishMartialTournament(tx, tournamentID, currentRound)
		_ = tx.Commit()
		invalidateMartialCache(tournamentID)
		return
	}

	var winners []string
	rows, err := tx.Query(`
		SELECT winner_combatant_id FROM martial_matches
		WHERE tournament_id = $1 AND round_number = $2 AND status = 'FINISHED' AND winner_combatant_id IS NOT NULL
		ORDER BY bracket_position
	`, tournamentID, currentRound)
	if err == nil {
		for rows.Next() {
			var id string
			if errScan := rows.Scan(&id); errScan == nil && id != "" {
				winners = append(winners, id)
			}
		}
		rows.Close()
	}

	rows2, err := tx.Query(`
		SELECT COALESCE(winner_combatant_id, combatant_a_id) FROM martial_matches
		WHERE tournament_id = $1 AND round_number = $2 AND status = 'BYE'
		ORDER BY bracket_position
	`, tournamentID, currentRound)
	if err == nil {
		for rows2.Next() {
			var id string
			if errScan := rows2.Scan(&id); errScan == nil && id != "" {
				winners = append(winners, id)
			}
		}
		rows2.Close()
	}

	if len(winners) <= 1 {
		finishMartialTournament(tx, tournamentID, currentRound)
		_ = tx.Commit()
		invalidateMartialCache(tournamentID)
		return
	}

	nextRound := currentRound + 1
	for i := 0; i < len(winners); i += 2 {
		if i+1 < len(winners) {
			if _, err = tx.Exec(`
				INSERT INTO martial_matches (tournament_id, round_number, bracket_position, combatant_a_id, combatant_b_id, status)
				VALUES ($1, $2, $3, $4, $5, 'PENDING')
			`, tournamentID, nextRound, (i/2)+1, winners[i], winners[i+1]); err != nil {
				logger.S.Errorf("Martial Arena: failed to create next round match: %v", err)
				return
			}
		} else {
			if _, err = tx.Exec(`
				INSERT INTO martial_matches (tournament_id, round_number, bracket_position, combatant_a_id, winner_combatant_id, status)
				VALUES ($1, $2, $3, $4, $4, 'BYE')
			`, tournamentID, nextRound, (i/2)+1, winners[i]); err != nil {
				logger.S.Errorf("Martial Arena: failed to create BYE match: %v", err)
				return
			}
		}
	}

	if _, err = tx.Exec(`UPDATE martial_tournaments SET current_round = $1 WHERE id = $2`, nextRound, tournamentID); err != nil {
		logger.S.Errorf("Martial Arena: failed to update current round: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("Martial Arena: failed to commit round advance: %v", err)
		return
	}
	invalidateMartialCache(tournamentID)
	BroadcastMartialArenaEvent("round_advanced", tournamentID, map[string]interface{}{
		"current_round": nextRound,
	})
}

// finishMartialTournament crowns the champion, awards rewards, and marks FINISHED.
func finishMartialTournament(tx *sql.Tx, tournamentID string, finalRound int) {
	var championCombatantID string
	err := tx.QueryRow(`
		SELECT COALESCE(winner_combatant_id, combatant_a_id) FROM martial_matches
		WHERE tournament_id = $1 AND round_number = $2 AND status IN ('FINISHED','BYE')
		ORDER BY CASE WHEN status = 'FINISHED' THEN 1 ELSE 2 END
		LIMIT 1
	`, tournamentID, finalRound).Scan(&championCombatantID)

	if err != nil || championCombatantID == "" {
		_, _ = tx.Exec(`UPDATE martial_tournaments SET status = 'FINISHED', finished_at = NOW() WHERE id = $1`, tournamentID)
		BroadcastMartialArenaEvent("tournament_finished", tournamentID, nil)
		return
	}

	var championPlayerID string
	_ = tx.QueryRow(`SELECT player_id FROM martial_combatants WHERE id = $1`, championCombatantID).Scan(&championPlayerID)

	_, _ = tx.Exec(`
		UPDATE martial_tournaments SET status = 'FINISHED', champion_player_id = $1, finished_at = NOW()
		WHERE id = $2
	`, championPlayerID, tournamentID)

	if err := awardMartialChampionRewards(tx, tournamentID); err != nil {
		logger.S.Errorf("Martial Arena: failed to award rewards for %s: %v", tournamentID, err)
	}

	_, _ = tx.Exec(`
		INSERT INTO messages (sender_id, content, channel)
		VALUES ('00000000-0000-0000-0000-000000000000', $1, 'world')
	`, fmt.Sprintf("[Đại Hội Võ Thuật] Giải đấu rank %d đã kết thúc! Chúc mừng nhà vô địch!", rankOfMartialTournament(tx, tournamentID)))

	BroadcastMartialArenaEvent("tournament_finished", tournamentID, map[string]interface{}{
		"champion_player_id": championPlayerID,
	})
	logger.S.Infof("Martial Arena: tournament %s finished with champion %s", tournamentID, championPlayerID)
}

func rankOfMartialTournament(q db.Queryer, tournamentID string) int {
	var r int
	_ = q.QueryRow(`SELECT rank FROM martial_tournaments WHERE id = $1`, tournamentID).Scan(&r)
	return r
}
