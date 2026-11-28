package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const (
	// maxTournamentMatchRetries bounds consecutive simulation errors on a single match
	// so a corrupt/missing combatant cannot stall an entire bracket indefinitely.
	maxTournamentMatchRetries = 3

	// MartialRoundDuration is the inter-round pause for Martial Arena (independent of Death Arena's RoundDuration).
	MartialRoundDuration = 2 * time.Minute
)

// arenaClock abstracts wall-clock time and cancellable sleeps for testability.
type arenaClock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) bool
}

type realArenaClock struct{}

func (realArenaClock) Now() time.Time { return time.Now() }

func (realArenaClock) Sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type bracketSlot struct {
	Position int
	SideAIdx int
	SideBIdx int // -1 means BYE
	IsBye    bool
}

// buildBracketPairs computes round-1 bracket slots for n participants.
func buildBracketPairs(n int) []bracketSlot {
	if n < 2 {
		return nil
	}
	numSlots := nextPowerOf2(n)
	byes := numSlots - n
	slots := make([]bracketSlot, 0, numSlots/2)

	pos := 0
	for i := 0; i < byes; i++ {
		pos++
		slots = append(slots, bracketSlot{Position: pos, SideAIdx: i, SideBIdx: -1, IsBye: true})
	}
	for i := byes; i < n; i += 2 {
		pos++
		if i+1 < n {
			slots = append(slots, bracketSlot{Position: pos, SideAIdx: i, SideBIdx: i + 1, IsBye: false})
		} else {
			slots = append(slots, bracketSlot{Position: pos, SideAIdx: i, SideBIdx: -1, IsBye: true})
		}
	}
	return slots
}

// tournamentAdapter parameterizes the shared tournament progression engine for
// Death Arena (6v6 teams) and Martial Arena (1v1 combatants).
type tournamentAdapter struct {
	Name               string
	DisplayTitle       string
	TournamentTable    string
	MatchTable         string
	ParticipantTable   string
	SideAColumn        string
	SideBColumn        string
	WinnerColumn       string
	ChampionColumn     string // champion_team_id or champion_player_id
	RewardColumn       string
	RoundDuration      time.Duration
	RoundReward        func(round int) int
	ResolveMatch       func(tournamentID, matchID, sideAID string, sideBID *string) error
	ResolveChampionID  func(tx *sql.Tx, sideID string) (string, error) // maps winning side to champion column value
	SettleRewards      func(tx *sql.Tx, tournamentID, championSideID string, finalRound int) ([]string, error)
	InvalidateCache    func(tournamentID string)
	BroadcastEvent     func(eventType, tournamentID string, data interface{})
	TryStartProcessing func(tournamentID string) bool
	StopProcessing     func(tournamentID string)
}

func runTournamentMatchLoop(ctx context.Context, a *tournamentAdapter, tournamentID string, clk arenaClock) {
	if !a.TryStartProcessing(tournamentID) {
		return
	}
	defer a.StopProcessing(tournamentID)

	if clk == nil {
		clk = realArenaClock{}
	}

	logger.S.Infof("%s: processing matches for tournament %s", a.Name, tournamentID)

	// Reopen in-progress matches on restart.
	_, _ = db.DB.Exec(
		fmt.Sprintf(`UPDATE %s SET status='PENDING' WHERE tournament_id=$1 AND status='IN_PROGRESS'`, a.MatchTable),
		tournamentID,
	)

	roundStartTime := clk.Now()
	failCounts := make(map[string]int)

	for {
		if ctx.Err() != nil {
			return
		}

		var status string
		if err := db.DB.QueryRow(
			fmt.Sprintf(`SELECT status FROM %s WHERE id = $1`, a.TournamentTable),
			tournamentID,
		).Scan(&status); err != nil || status != "IN_PROGRESS" {
			logger.S.Infof("%s: tournament %s no longer in progress (status: %s)", a.Name, tournamentID, status)
			return
		}

		var matchID, sideAID string
		var sideBID sql.NullString
		err := db.DB.QueryRow(fmt.Sprintf(`
			SELECT id, %s, %s FROM %s
			WHERE tournament_id = $1 AND status = 'PENDING'
			ORDER BY round_number, bracket_position
			LIMIT 1
		`, a.SideAColumn, a.SideBColumn, a.MatchTable), tournamentID).Scan(&matchID, &sideAID, &sideBID)

		if err == sql.ErrNoRows {
			var activeCount int
			_ = db.DB.QueryRow(
				fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE tournament_id=$1 AND status='IN_PROGRESS'`, a.MatchTable),
				tournamentID,
			).Scan(&activeCount)
			if activeCount > 0 {
				if !clk.Sleep(ctx, time.Second) {
					return
				}
				continue
			}

			var pendingCount, finishedCount int
			_ = db.DB.QueryRow(fmt.Sprintf(`
				SELECT
					COUNT(*) FILTER (WHERE status = 'PENDING'),
					COUNT(*) FILTER (WHERE status IN ('FINISHED', 'BYE'))
				FROM %s
				WHERE tournament_id = $1 AND round_number = (SELECT current_round FROM %s WHERE id = $1)
			`, a.MatchTable, a.TournamentTable), tournamentID).Scan(&pendingCount, &finishedCount)

			if pendingCount == 0 && finishedCount > 0 {
				advanceTournamentRound(a, tournamentID)

				var currentStatus string
				var currentRound, totalRounds int
				_ = db.DB.QueryRow(
					fmt.Sprintf(`SELECT status, current_round, total_rounds FROM %s WHERE id = $1`, a.TournamentTable),
					tournamentID,
				).Scan(&currentStatus, &currentRound, &totalRounds)

				if currentStatus == "IN_PROGRESS" {
					elapsed := clk.Now().Sub(roundStartTime)
					if elapsed < a.RoundDuration {
						intermission := a.RoundDuration - elapsed
						intermissionSec := int(intermission.Seconds())
						logger.S.Infof("%s: round finished. Intermission %v before round %d", a.Name, intermission.Round(time.Second), currentRound)

						a.BroadcastEvent("round_intermission", tournamentID, map[string]interface{}{
							"current_round":     currentRound,
							"total_rounds":      totalRounds,
							"countdown_seconds": intermissionSec,
						})

						mins := intermissionSec / 60
						secs := intermissionSec % 60
						timeMsg := fmt.Sprintf("%d giây", secs)
						if mins > 0 {
							timeMsg = fmt.Sprintf("%d phút %02d giây", mins, secs)
						}
						_, _ = db.DB.Exec(`
							INSERT INTO messages (sender_id, content, channel)
							VALUES ('00000000-0000-0000-0000-000000000000', $1, 'world')
						`, fmt.Sprintf("[%s] Vòng %d đã kết thúc! Vòng %d sẽ chính thức khai chiến sau %s...", a.DisplayTitle, currentRound-1, currentRound, timeMsg))

						if !clk.Sleep(ctx, intermission) {
							return
						}
					}
					roundStartTime = clk.Now()
					continue
				}
			} else if pendingCount == 0 && finishedCount == 0 {
				var remainingPending int
				_ = db.DB.QueryRow(
					fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE tournament_id = $1 AND status = 'PENDING'`, a.MatchTable),
					tournamentID,
				).Scan(&remainingPending)
				if remainingPending == 0 {
					finishTournamentFallback(a, tournamentID)
					return
				}
			}
			return
		} else if err != nil {
			logger.S.Errorf("%s: error finding pending match for %s: %v", a.Name, tournamentID, err)
			return
		}

		claimed, claimErr := db.DB.Exec(
			fmt.Sprintf(`UPDATE %s SET status='IN_PROGRESS' WHERE id=$1 AND status='PENDING'`, a.MatchTable),
			matchID,
		)
		if claimErr != nil {
			logger.S.Errorf("%s: failed to claim match %s: %v", a.Name, matchID, claimErr)
			return
		}
		if affected, _ := claimed.RowsAffected(); affected != 1 {
			continue
		}

		var bPtr *string
		if sideBID.Valid && sideBID.String != "" {
			bPtr = &sideBID.String
		}

		if err := a.ResolveMatch(tournamentID, matchID, sideAID, bPtr); err != nil {
			failCounts[matchID]++
			logger.S.Errorf("%s: error resolving match %s (attempt %d/%d): %v", a.Name, matchID, failCounts[matchID], maxTournamentMatchRetries, err)

			if failCounts[matchID] >= maxTournamentMatchRetries {
				forfeitFailedTournamentMatch(a, tournamentID, matchID, sideAID, bPtr)
				delete(failCounts, matchID)
			} else {
				_, _ = db.DB.Exec(
					fmt.Sprintf(`UPDATE %s SET status='PENDING' WHERE id=$1 AND status='IN_PROGRESS'`, a.MatchTable),
					matchID,
				)
			}
			if !clk.Sleep(ctx, 2*time.Second) {
				return
			}
			continue
		}

		delete(failCounts, matchID)
		if !clk.Sleep(ctx, 2*time.Second) {
			return
		}
	}
}

func forfeitFailedTournamentMatch(a *tournamentAdapter, tournamentID, matchID, sideAID string, sideBID *string) {
	tx, err := db.DB.Begin()
	if err != nil {
		logger.S.Errorf("%s: failed to begin forfeit tx for match %s: %v", a.Name, matchID, err)
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		fmt.Sprintf(`UPDATE %s SET status='FINISHED', %s=$1, finished_at=NOW() WHERE id=$2 AND status='IN_PROGRESS'`, a.MatchTable, a.WinnerColumn),
		sideAID, matchID,
	); err != nil {
		logger.S.Errorf("%s: failed to forfeit match %s: %v", a.Name, matchID, err)
		return
	}
	if sideBID != nil && *sideBID != "" {
		_, _ = tx.Exec(fmt.Sprintf(`
			UPDATE %s SET eliminated = true, eliminated_at_round = (
				SELECT current_round FROM %s WHERE id = $1
			) WHERE id = $2
		`, a.ParticipantTable, a.TournamentTable), tournamentID, *sideBID)
	}
	if err := tx.Commit(); err != nil {
		logger.S.Errorf("%s: failed to commit forfeit match %s: %v", a.Name, matchID, err)
		return
	}
	a.InvalidateCache(tournamentID)
	a.BroadcastEvent("match_updated", tournamentID, map[string]interface{}{
		"match_id":     matchID,
		a.WinnerColumn: sideAID,
		"status":       "FINISHED",
	})
	logger.S.Warnf("%s: match %s exceeded %d retries; awarded forfeit win to %s", a.Name, matchID, maxTournamentMatchRetries, sideAID)
}

func advanceTournamentRound(a *tournamentAdapter, tournamentID string) {
	tx, err := db.DB.Begin()
	if err != nil {
		logger.S.Errorf("%s: failed to begin tx for round advance: %v", a.Name, err)
		return
	}
	defer tx.Rollback()

	var currentRound, totalRounds int
	if err := tx.QueryRow(
		fmt.Sprintf(`SELECT current_round, total_rounds FROM %s WHERE id = $1`, a.TournamentTable),
		tournamentID,
	).Scan(&currentRound, &totalRounds); err != nil {
		logger.S.Errorf("%s: failed to get round state: %v", a.Name, err)
		return
	}

	if currentRound >= totalRounds {
		championSideID, err := findRoundChampionSide(tx, a, tournamentID, currentRound)
		if err != nil || championSideID == "" {
			logger.S.Warnf("%s: failed to find champion in final round %d for %s: %v", a.Name, currentRound, tournamentID, err)
			_, _ = tx.Exec(fmt.Sprintf(`UPDATE %s SET status='FINISHED', finished_at=NOW() WHERE id=$1`, a.TournamentTable), tournamentID)
			if commitErr := tx.Commit(); commitErr == nil {
				a.InvalidateCache(tournamentID)
				a.BroadcastEvent("tournament_finished", tournamentID, nil)
			}
			return
		}
		commitTournamentFinish(tx, a, tournamentID, championSideID, currentRound)
		return
	}

	winners, err := collectRoundWinners(tx, a, tournamentID, currentRound)
	if err != nil {
		logger.S.Errorf("%s: failed to collect round %d winners for %s: %v", a.Name, currentRound, tournamentID, err)
		return
	}

	if len(winners) <= 1 {
		if len(winners) == 1 {
			commitTournamentFinish(tx, a, tournamentID, winners[0], currentRound)
			return
		}
		_, _ = tx.Exec(fmt.Sprintf(`UPDATE %s SET status='FINISHED', finished_at=NOW() WHERE id=$1`, a.TournamentTable), tournamentID)
		if commitErr := tx.Commit(); commitErr == nil {
			a.InvalidateCache(tournamentID)
			a.BroadcastEvent("tournament_finished", tournamentID, nil)
		}
		return
	}

	nextRound := currentRound + 1
	reward := a.RoundReward(nextRound)
	for i := 0; i < len(winners); i += 2 {
		if i+1 < len(winners) {
			_, err = tx.Exec(fmt.Sprintf(`
				INSERT INTO %s (tournament_id, round_number, bracket_position, %s, %s, status, %s)
				VALUES ($1, $2, $3, $4, $5, 'PENDING', $6)
			`, a.MatchTable, a.SideAColumn, a.SideBColumn, a.RewardColumn),
				tournamentID, nextRound, (i/2)+1, winners[i], winners[i+1], reward)
		} else {
			_, err = tx.Exec(fmt.Sprintf(`
				INSERT INTO %s (tournament_id, round_number, bracket_position, %s, %s, status, %s)
				VALUES ($1, $2, $3, $4, $4, 'BYE', 0)
			`, a.MatchTable, a.SideAColumn, a.WinnerColumn, a.RewardColumn),
				tournamentID, nextRound, (i/2)+1, winners[i])
		}
		if err != nil {
			logger.S.Errorf("%s: failed to create next round match: %v", a.Name, err)
			return
		}
	}

	if _, err = tx.Exec(
		fmt.Sprintf(`UPDATE %s SET current_round = $1 WHERE id = $2`, a.TournamentTable),
		nextRound, tournamentID,
	); err != nil {
		logger.S.Errorf("%s: failed to update current round: %v", a.Name, err)
		return
	}

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("%s: failed to commit round advance: %v", a.Name, err)
		return
	}
	a.InvalidateCache(tournamentID)
	a.BroadcastEvent("round_advanced", tournamentID, map[string]interface{}{
		"current_round": nextRound,
	})
}

func findRoundChampionSide(tx *sql.Tx, a *tournamentAdapter, tournamentID string, round int) (string, error) {
	var championSideID string
	err := tx.QueryRow(fmt.Sprintf(`
		SELECT COALESCE(%s, %s) FROM %s
		WHERE tournament_id = $1 AND round_number = $2 AND status IN ('FINISHED', 'BYE')
		ORDER BY CASE WHEN status = 'FINISHED' THEN 1 ELSE 2 END
		LIMIT 1
	`, a.WinnerColumn, a.SideAColumn, a.MatchTable), tournamentID, round).Scan(&championSideID)
	return championSideID, err
}

func collectRoundWinners(tx *sql.Tx, a *tournamentAdapter, tournamentID string, round int) ([]string, error) {
	var winners []string
	rows, err := tx.Query(fmt.Sprintf(`
		SELECT %s FROM %s
		WHERE tournament_id = $1 AND round_number = $2 AND status = 'FINISHED' AND %s IS NOT NULL
		ORDER BY bracket_position
	`, a.WinnerColumn, a.MatchTable, a.WinnerColumn), tournamentID, round)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id string
		if errScan := rows.Scan(&id); errScan == nil && id != "" {
			winners = append(winners, id)
		}
	}
	rows.Close()

	rows2, err := tx.Query(fmt.Sprintf(`
		SELECT COALESCE(%s, %s) FROM %s
		WHERE tournament_id = $1 AND round_number = $2 AND status = 'BYE'
		ORDER BY bracket_position
	`, a.WinnerColumn, a.SideAColumn, a.MatchTable), tournamentID, round)
	if err != nil {
		return nil, err
	}
	for rows2.Next() {
		var id string
		if errScan := rows2.Scan(&id); errScan == nil && id != "" {
			winners = append(winners, id)
		}
	}
	rows2.Close()

	return winners, nil
}

func commitTournamentFinish(tx *sql.Tx, a *tournamentAdapter, tournamentID, championSideID string, finalRound int) {
	championValue := championSideID
	if a.ResolveChampionID != nil {
		resolved, err := a.ResolveChampionID(tx, championSideID)
		if err != nil {
			logger.S.Errorf("%s: failed to resolve champion for %s: %v", a.Name, tournamentID, err)
			return
		}
		championValue = resolved
	}

	if _, err := tx.Exec(fmt.Sprintf(`
		UPDATE %s SET status = 'FINISHED', %s = $1, finished_at = NOW()
		WHERE id = $2
	`, a.TournamentTable, a.ChampionColumn), championValue, tournamentID); err != nil {
		logger.S.Errorf("%s: failed to mark tournament %s finished: %v", a.Name, tournamentID, err)
		return
	}

	rewardedPlayers, err := a.SettleRewards(tx, tournamentID, championSideID, finalRound)
	if err != nil {
		logger.S.Errorf("%s: failed to settle rewards for %s: %v", a.Name, tournamentID, err)
		return
	}

	_, _ = tx.Exec(`
		INSERT INTO messages (sender_id, content, channel)
		VALUES ('00000000-0000-0000-0000-000000000000', $1, 'world')
	`, fmt.Sprintf("[%s] Giải đấu đã kết thúc! Chúc mừng vô địch! Top 3 hãy vào mục Phần thưởng để nhận quà.", a.DisplayTitle))

	if err := tx.Commit(); err != nil {
		logger.S.Errorf("%s: failed to commit tournament finish %s: %v", a.Name, tournamentID, err)
		return
	}

	a.InvalidateCache(tournamentID)
	a.BroadcastEvent("tournament_finished", tournamentID, map[string]interface{}{
		a.ChampionColumn: championValue,
	})
	if len(rewardedPlayers) > 0 {
		a.BroadcastEvent("reward_available", tournamentID, map[string]interface{}{
			"player_ids": rewardedPlayers,
		})
	}
	logger.S.Infof("%s: tournament %s finished with champion %s (%d rewarded players)", a.Name, tournamentID, championValue, len(rewardedPlayers))
}

func finishTournamentFallback(a *tournamentAdapter, tournamentID string) {
	tx, err := db.DB.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()

	var currentRound int
	_ = tx.QueryRow(fmt.Sprintf(`SELECT current_round FROM %s WHERE id = $1`, a.TournamentTable), tournamentID).Scan(&currentRound)

	if currentRound > 0 {
		if champSide, err := findRoundChampionSide(tx, a, tournamentID, currentRound); err == nil && champSide != "" {
			commitTournamentFinish(tx, a, tournamentID, champSide, currentRound)
			return
		}
	}

	logger.S.Warnf("%s: tournament %s has no pending or finished matches in current round, auto-finishing", a.Name, tournamentID)
	_, _ = tx.Exec(fmt.Sprintf(`UPDATE %s SET status = 'FINISHED', finished_at = NOW() WHERE id = $1`, a.TournamentTable), tournamentID)
	if err := tx.Commit(); err == nil {
		a.InvalidateCache(tournamentID)
		a.BroadcastEvent("tournament_finished", tournamentID, nil)
	}
}
