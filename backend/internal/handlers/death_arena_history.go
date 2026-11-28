package handlers

import (
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// GetTournamentHistory returns past tournaments
func GetTournamentHistory(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT id, status, scheduled_at, registration_started_at, registration_closed_at,
		       started_at, finished_at, total_players, total_teams, current_round, total_rounds,
		       champion_team_id, created_at
		FROM death_arena_tournaments
		WHERE status = 'FINISHED'
		ORDER BY finished_at DESC
		LIMIT 20
	`)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to fetch history", err))
		return
	}
	defer rows.Close()

	var tournaments []Tournament
	for rows.Next() {
		var t Tournament
		if errScan := rows.Scan(&t.ID, &t.Status, &t.ScheduledAt, &t.RegistrationStartedAt,
			&t.RegistrationClosedAt, &t.StartedAt, &t.FinishedAt, &t.TotalPlayers, &t.TotalTeams,
			&t.CurrentRound, &t.TotalRounds, &t.ChampionTeamID, &t.CreatedAt); errScan == nil {
			tournaments = append(tournaments, t)
		}
	}

	c.JSON(http.StatusOK, gin.H{"tournaments": tournaments})
}

// GetMyRewards returns unclaimed rewards for the player
func GetMyRewards(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT dar.id, dar.tournament_id, dar.player_id, p.username, dar.placement,
		       dar.team_id, dt.team_name, dar.title, dar.reward_stones, dar.reward_immortal_stones,
		       dar.reward_gu_id, dar.reward_gu_rank, dar.reward_items, dar.claimed, dar.created_at
		FROM death_arena_champion_rewards dar
		JOIN players p ON dar.player_id = p.id
		JOIN death_arena_teams dt ON dar.team_id = dt.id
		WHERE dar.player_id = $1
		ORDER BY dar.created_at DESC
	`, authID)

	if err != nil {
		httperr.Respond(c, httperr.Internal("Failed to fetch rewards", err))
		return
	}
	defer rows.Close()

	var rewards []ChampionReward
	for rows.Next() {
		var r ChampionReward
		if errScan := rows.Scan(&r.ID, &r.TournamentID, &r.PlayerID, &r.Username, &r.Placement,
			&r.TeamID, &r.TeamName, &r.Title, &r.RewardStones, &r.RewardImmortalStones,
			&r.RewardGuID, &r.RewardGuRank, &r.RewardItems, &r.Claimed, &r.CreatedAt); errScan == nil {
			rewards = append(rewards, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{"rewards": rewards})
}

// ClaimReward claims a champion reward
func ClaimReward(c *gin.Context) {
	claimTournamentReward(c, deathArenaRewardClaimSpec)
}

// generateTeamName generates a random team name
func generateTeamName() string {
	prefixes := []string{"Thiên", "Địa", "Nhân", "Phong", "Lôi", "Hỏa", "Thủy", "Mộc", "Kim", "Sơn", "Hà", "Tinh"}
	suffixes := []string{"Sư", "Tử", "Vương", "Hầu", "Tướng", "Sĩ", "Giả", "Đạo", "Cung", "Môn", "Phái", "Bang"}
	return prefixes[rand.Intn(len(prefixes))] + suffixes[rand.Intn(len(suffixes))]
}
