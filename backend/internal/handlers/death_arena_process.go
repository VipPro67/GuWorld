package handlers

import (
	"context"
	"database/sql"

	"github.com/vippro67/GuWorld/backend/internal/db"
)

func tryStartDeathArenaProcessing(_ string) bool {
	matchProcessingMutex.Lock()
	defer matchProcessingMutex.Unlock()
	if isProcessingMatches {
		return false
	}
	isProcessingMatches = true
	return true
}

func stopDeathArenaProcessing(_ string) {
	matchProcessingMutex.Lock()
	isProcessingMatches = false
	matchProcessingMutex.Unlock()
}

func deathArenaRoundReward(round int) int {
	if r := roundRewards[round]; r > 0 {
		return r
	}
	return roundRewards[len(roundRewards)]
}

func deathArenaAdapter() *tournamentAdapter {
	return &tournamentAdapter{
		Name:               "Death Arena",
		DisplayTitle:       "Đấu Trường Sinh Tử",
		TournamentTable:    "death_arena_tournaments",
		MatchTable:         "death_arena_matches",
		ParticipantTable:   "death_arena_teams",
		SideAColumn:        "team_a_id",
		SideBColumn:        "team_b_id",
		WinnerColumn:       "winner_team_id",
		ChampionColumn:     "champion_team_id",
		RewardColumn:       "reward_stones_per_member",
		RoundDuration:      RoundDuration,
		RoundReward:        deathArenaRoundReward,
		ResolveMatch:       resolveTeamMatch,
		SettleRewards:      awardChampionRewards,
		InvalidateCache:    invalidateTournamentCache,
		BroadcastEvent:     BroadcastDeathArenaEvent,
		TryStartProcessing: tryStartDeathArenaProcessing,
		StopProcessing:     stopDeathArenaProcessing,
	}
}

// processPendingMatches processes all pending matches in the tournament.
func processPendingMatches(tournamentID string) {
	runTournamentMatchLoop(context.Background(), deathArenaAdapter(), tournamentID, realArenaClock{})
}

// advanceToNextRound advances the tournament to the next round or settles the champion.
func advanceToNextRound(tournamentID string) {
	advanceTournamentRound(deathArenaAdapter(), tournamentID)
}

// resolveTeamMatch resolves a single team match.
func resolveTeamMatch(tournamentID, matchID, teamAID string, teamBID *string) error {
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

	teamAMembers, err := loadTeamMembersForCombat(db.DB, teamAID)
	if err != nil {
		return err
	}

	teamBMembers, err := loadTeamMembersForCombat(db.DB, *teamBID)
	if err != nil {
		return err
	}

	if len(teamAMembers) == 0 {
		return recordWalkoverTeamMatch(tournamentID, matchID, *teamBID, teamAID)
	}
	if len(teamBMembers) == 0 {
		return recordWalkoverTeamMatch(tournamentID, matchID, teamAID, *teamBID)
	}

	return resolveDeathArenaTurnBasedMatch(tournamentID, matchID, teamAID, *teamBID)
}

func recordWalkoverTeamMatch(tournamentID, matchID, winnerTeamID, loserTeamID string) error {
	tx, err := db.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE death_arena_matches SET status = 'FINISHED', winner_team_id = $1, finished_at = NOW()
		WHERE id = $2
	`, winnerTeamID, matchID); err != nil {
		return err
	}
	_, _ = tx.Exec(`UPDATE death_arena_teams SET eliminated = true, eliminated_at_round = (
		SELECT current_round FROM death_arena_tournaments WHERE id = $1
	) WHERE id = $2`, tournamentID, loserTeamID)
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

// Keep sql import used by tournamentAdapter signature.
var _ *sql.Tx
