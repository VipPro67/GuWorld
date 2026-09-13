package handlers

import (
	"database/sql"
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rewards"})
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
	var req struct {
		RewardID string `json:"reward_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reward ID is required"})
		return
	}

	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rewardUUID, err := uuid.Parse(req.RewardID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid reward ID"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Verify ownership and not claimed
	var reward ChampionReward
	err = tx.QueryRow(`
		SELECT id, player_id, placement, reward_stones, reward_immortal_stones, reward_gu_id, reward_gu_rank, claimed
		FROM death_arena_champion_rewards
		WHERE id = $1 AND player_id = $2
	`, rewardUUID, authID).Scan(&reward.ID, &reward.PlayerID, &reward.Placement,
		&reward.RewardStones, &reward.RewardImmortalStones, &reward.RewardGuID,
		&reward.RewardGuRank, &reward.Claimed)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy phần thưởng"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if reward.Claimed {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phần thưởng đã được nhận rồi"})
		return
	}

	// Award stones
	if reward.RewardStones > 0 {
		_, err = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones + $1 WHERE id = $2`,
			reward.RewardStones, authID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to award stones"})
			return
		}
	}

	if reward.RewardImmortalStones > 0 {
		_, err = tx.Exec(`UPDATE players SET immortal_stones = immortal_stones + $1 WHERE id = $2`,
			reward.RewardImmortalStones, authID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to award immortal stones"})
			return
		}
	}

	// Award title
	if reward.Title != "" {
		_, err = tx.Exec(`UPDATE players SET death_arena_title = $1 WHERE id = $2`,
			reward.Title, authID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to award title"})
			return
		}
	}

	// Mark as claimed
	_, err = tx.Exec(`UPDATE death_arena_champion_rewards SET claimed = true, claimed_at = NOW() WHERE id = $1`,
		rewardUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as claimed"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Nhận phần thưởng thành công!",
		"placement":       reward.Placement,
		"title":           reward.Title,
		"stones":          reward.RewardStones,
		"immortal_stones": reward.RewardImmortalStones,
	})
}

// generateTeamName generates a random team name
func generateTeamName() string {
	prefixes := []string{"Thiên", "Địa", "Nhân", "Phong", "Lôi", "Hỏa", "Thủy", "Mộc", "Kim", "Sơn", "Hà", "Tinh"}
	suffixes := []string{"Sư", "Tử", "Vương", "Hầu", "Tướng", "Sĩ", "Giả", "Đạo", "Cung", "Môn", "Phái", "Bang"}
	return prefixes[rand.Intn(len(prefixes))] + suffixes[rand.Intn(len(suffixes))]
}
