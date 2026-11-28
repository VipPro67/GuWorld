package handlers

import (
	"math"

	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// MartialOnlineWindowSQL is the shared "online" predicate used by snapshot queries.
const MartialOnlineWindowSQL = `(last_activity >= NOW() - INTERVAL '5 minutes' OR last_mana_update >= NOW() - INTERVAL '5 minutes')`

// martialBracketShape returns the slot count and bye count for n combatants.
// numSlots is the smallest power of two >= n; byes receive a free pass to round 2.
func martialBracketShape(n int) (numSlots, byes int) {
	numSlots = nextPowerOf2(n)
	byes = numSlots - n
	return numSlots, byes
}

// runMartialArenaEvent snapshots online players per cultivation rank and opens a
// solo 1v1 bracket for every rank with >= MinMartialPlayersPerRank players.
func runMartialArenaEvent() {
	// Skip if an event is already running.
	var active int
	if err := db.DB.QueryRow(`SELECT COUNT(*) FROM martial_tournaments WHERE status IN ('LOCKED','IN_PROGRESS')`).Scan(&active); err == nil && active > 0 {
		logger.S.Info("Martial Arena: active tournaments already exist, skipping event")
		return
	}

	rows, err := db.DB.Query(`
		SELECT rank FROM players
		WHERE `+MartialOnlineWindowSQL+`
		GROUP BY rank
		HAVING COUNT(*) >= $1
		ORDER BY rank
	`, MinMartialPlayersPerRank)
	if err != nil {
		logger.S.Errorf("Martial Arena: failed to list eligible ranks: %v", err)
		return
	}
	defer rows.Close()

	var ranks []int
	for rows.Next() {
		var r int
		if errScan := rows.Scan(&r); errScan == nil {
			ranks = append(ranks, r)
		}
	}

	for _, rank := range ranks {
		if err := createMartialTournament(rank); err != nil {
			logger.S.Errorf("Martial Arena: failed to create rank %d tournament: %v", rank, err)
		}
	}
}

// createMartialTournament snapshots up to MaxMartialPlayersPerRank online players
// of one rank (seeded by sub_rank DESC), then generates and starts the bracket.
func createMartialTournament(rank int) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT id FROM players
		WHERE rank = $1 AND `+MartialOnlineWindowSQL+`
		ORDER BY sub_rank DESC
		LIMIT $2
	`, rank, MaxMartialPlayersPerRank)
	if err != nil {
		return err
	}
	defer rows.Close()

	var playerIDs []string
	for rows.Next() {
		var id string
		if errScan := rows.Scan(&id); errScan == nil {
			playerIDs = append(playerIDs, id)
		}
	}
	if len(playerIDs) < MinMartialPlayersPerRank {
		return nil // not enough online players for this rank
	}

	var tournamentID string
	err = tx.QueryRow(`
		INSERT INTO martial_tournaments (status, rank, scheduled_at)
		VALUES ('LOCKED', $1, NOW()) RETURNING id
	`, rank).Scan(&tournamentID)
	if err != nil {
		return err
	}

	for i, pid := range playerIDs {
		if _, err = tx.Exec(`
			INSERT INTO martial_combatants (tournament_id, player_id, seed_number, rank, sub_rank)
			SELECT $1, p.id, $2, p.rank, p.sub_rank FROM players p WHERE p.id = $3
		`, tournamentID, i+1, pid); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := GenerateMartialBracket(tournamentID); err != nil {
		logger.S.Errorf("Martial Arena: failed to generate bracket %s: %v", tournamentID, err)
		return err
	}
	return StartMartialTournament(tournamentID)
}

// GenerateMartialBracket creates round-1 matches from the seeded combatants.
// Seeds are kept in sub_rank order (no shuffle) so nearest sub_rank players meet
// first; top seeds receive byes when the bracket is not a power of two.
func GenerateMartialBracket(tournamentID string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT id FROM martial_combatants
		WHERE tournament_id = $1
		ORDER BY seed_number
	`, tournamentID)
	if err != nil {
		return err
	}
	defer rows.Close()

	var combatants []string
	for rows.Next() {
		var id string
		if errScan := rows.Scan(&id); errScan == nil {
			combatants = append(combatants, id)
		}
	}

	n := len(combatants)
	if n < 2 {
		var champion interface{}
		var rewardedPlayers []string
		if n == 1 {
			var pid string
			_ = tx.QueryRow(`SELECT player_id FROM martial_combatants WHERE id = $1`, combatants[0]).Scan(&pid)
			champion = pid
		}
		_, _ = tx.Exec(`
			UPDATE martial_tournaments SET status = 'FINISHED', finished_at = NOW(), champion_player_id = $1, total_players = $2
			WHERE id = $3
		`, champion, n, tournamentID)
		if n == 1 {
			rewardedPlayers, err = awardMartialChampionRewards(tx, tournamentID, 1)
			if err != nil {
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		invalidateMartialCache(tournamentID)
		BroadcastMartialArenaEvent("tournament_finished", tournamentID, nil)
		if len(rewardedPlayers) > 0 {
			BroadcastMartialArenaEvent("reward_available", tournamentID, map[string]interface{}{
				"player_ids": rewardedPlayers,
			})
		}
		return nil
	}

	totalRounds := int(math.Ceil(math.Log2(float64(n))))
	slots := buildBracketPairs(n)
	round1Reward := martialRoundReward(1)

	for _, slot := range slots {
		if slot.IsBye {
			if _, err = tx.Exec(`
				INSERT INTO martial_matches (tournament_id, round_number, bracket_position, combatant_a_id, winner_combatant_id, status, reward_stones)
				VALUES ($1, 1, $2, $3, $3, 'BYE', 0)
			`, tournamentID, slot.Position, combatants[slot.SideAIdx]); err != nil {
				return err
			}
		} else {
			if _, err = tx.Exec(`
				INSERT INTO martial_matches (tournament_id, round_number, bracket_position, combatant_a_id, combatant_b_id, status, reward_stones)
				VALUES ($1, 1, $2, $3, $4, 'PENDING', $5)
			`, tournamentID, slot.Position, combatants[slot.SideAIdx], combatants[slot.SideBIdx], round1Reward); err != nil {
				return err
			}
		}
	}

	if _, err = tx.Exec(`
		UPDATE martial_tournaments
		SET status = 'LOCKED', total_rounds = $1, current_round = 1, total_players = $2, started_at = NOW()
		WHERE id = $3
	`, totalRounds, n, tournamentID); err != nil {
		return err
	}

	return tx.Commit()
}

// StartMartialTournament transitions a LOCKED bracket to IN_PROGRESS and kicks off
// its match worker.
func StartMartialTournament(tournamentID string) error {
	res, err := db.DB.Exec(`
		UPDATE martial_tournaments SET status = 'IN_PROGRESS'
		WHERE id = $1 AND status = 'LOCKED' AND total_players >= 2
	`, tournamentID)
	if err != nil {
		return err
	}
	if rowsAff, _ := res.RowsAffected(); rowsAff == 0 {
		return nil // not lockable (already finished / not enough players)
	}

	invalidateMartialCache(tournamentID)
	BroadcastMartialArenaEvent("tournament_started", tournamentID, nil)
	go processPendingMartialMatches(tournamentID)
	return nil
}

func invalidateMartialCache(tournamentID string) {
	_ = cache.Delete("martial_arena:current")
	_ = cache.Delete("martial_arena:bracket:" + tournamentID)
}
