package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/vippro67/GuWorld/backend/internal/db"
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

func martialRoundReward(round int) int {
	if r := roundRewards[round]; r > 0 {
		return r
	}
	return roundRewards[len(roundRewards)]
}

func resolveMartialChampionPlayerID(tx *sql.Tx, combatantID string) (string, error) {
	var playerID string
	if err := tx.QueryRow(`SELECT player_id FROM martial_combatants WHERE id = $1`, combatantID).Scan(&playerID); err != nil {
		return "", fmt.Errorf("load martial arena champion player %s: %w", combatantID, err)
	}
	return playerID, nil
}

func martialArenaAdapter() *tournamentAdapter {
	return &tournamentAdapter{
		Name:              "Martial Arena",
		DisplayTitle:      "Đại Hội Võ Thuật",
		TournamentTable:   "martial_tournaments",
		MatchTable:        "martial_matches",
		ParticipantTable:  "martial_combatants",
		SideAColumn:       "combatant_a_id",
		SideBColumn:       "combatant_b_id",
		WinnerColumn:      "winner_combatant_id",
		ChampionColumn:    "champion_player_id",
		RewardColumn:      "reward_stones",
		RoundDuration:     MartialRoundDuration,
		RoundReward:       martialRoundReward,
		ResolveMatch:      resolveMartialMatch,
		ResolveChampionID: resolveMartialChampionPlayerID,
		SettleRewards: func(tx *sql.Tx, tournamentID, _ string, finalRound int) ([]string, error) {
			return awardMartialChampionRewards(tx, tournamentID, finalRound)
		},
		InvalidateCache:    invalidateMartialCache,
		BroadcastEvent:     BroadcastMartialArenaEvent,
		TryStartProcessing: tryStartMartialProcessing,
		StopProcessing:     stopMartialProcessing,
	}
}

// processPendingMartialMatches resolves every pending match in a bracket, advancing
// rounds until the tournament finishes. Each rank bracket runs in its own goroutine.
func processPendingMartialMatches(tournamentID string) {
	runTournamentMatchLoop(context.Background(), martialArenaAdapter(), tournamentID, realArenaClock{})
}

// resolveMartialMatch simulates one 1v1 turn-based match and records the result.
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

	return resolveMartialArenaTurnBasedMatch(tournamentID, matchID, combatantAID, *combatantBID)
}

// advanceMartialRound creates next-round matches from the current winners, or crowns
// the champion when the final round is complete.
func advanceMartialRound(tournamentID string) {
	advanceTournamentRound(martialArenaAdapter(), tournamentID)
}
