package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

func GetCurrentTournament(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Find current tournament (cached for 5s to avoid DB load)
	var t Tournament
	var found bool
	if err := cache.GetJSON("death_arena:current_tournament_meta", &t); err == nil && t.ID != "" {
		found = true
	} else {
		err := db.DB.QueryRow(`
			SELECT id, status, scheduled_at, registration_started_at, registration_closed_at,
			       started_at, finished_at, total_players, total_teams, current_round, total_rounds,
			       champion_team_id, created_at
			FROM death_arena_tournaments
			WHERE status != 'FINISHED'
			ORDER BY scheduled_at DESC
			LIMIT 1
		`).Scan(&t.ID, &t.Status, &t.ScheduledAt, &t.RegistrationStartedAt, &t.RegistrationClosedAt,
			&t.StartedAt, &t.FinishedAt, &t.TotalPlayers, &t.TotalTeams, &t.CurrentRound, &t.TotalRounds,
			&t.ChampionTeamID, &t.CreatedAt)

		if err == sql.ErrNoRows {
			// No active tournament, check for most recent finished one
			err = db.DB.QueryRow(`
				SELECT id, status, scheduled_at, registration_started_at, registration_closed_at,
				       started_at, finished_at, total_players, total_teams, current_round, total_rounds,
				       champion_team_id, created_at
				FROM death_arena_tournaments
				WHERE status = 'FINISHED'
				ORDER BY finished_at DESC
				LIMIT 1
			`).Scan(&t.ID, &t.Status, &t.ScheduledAt, &t.RegistrationStartedAt, &t.RegistrationClosedAt,
				&t.StartedAt, &t.FinishedAt, &t.TotalPlayers, &t.TotalTeams, &t.CurrentRound, &t.TotalRounds,
				&t.ChampionTeamID, &t.CreatedAt)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"tournament": nil, "message": "Không có giải đấu nào"})
				return
			}
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tournament"})
			return
		}
		found = true
		_ = cache.SetJSON("death_arena:current_tournament_meta", t, 5*time.Second)
	}
	if !found {
		c.JSON(http.StatusOK, gin.H{"tournament": nil, "message": "Không có giải đấu nào"})
		return
	}

	// Check if player is registered
	var isRegistered bool
	var teamID *string
	_ = db.DB.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM death_arena_team_members dtm
			JOIN death_arena_teams dt ON dtm.team_id = dt.id
			WHERE dtm.player_id = $1 AND dt.tournament_id = $2
		), (
			SELECT dtm.team_id FROM death_arena_team_members dtm
			JOIN death_arena_teams dt ON dtm.team_id = dt.id
			WHERE dtm.player_id = $1 AND dt.tournament_id = $2
			LIMIT 1
		)
	`, authID, t.ID).Scan(&isRegistered, &teamID)

	// Get team and members if registered
	var members []TeamMember
	var team *Team
	if isRegistered && teamID != nil {
		var tm Team
		errTeam := db.DB.QueryRow(`
			SELECT dt.id, dt.tournament_id, dt.team_name, dt.seed_number, dt.eliminated,
			       dt.eliminated_at_round, dt.final_placement, dt.created_at,
			       COUNT(dtm.id) as member_count,
			       COALESCE(AVG(dtm.current_hp / NULLIF(dtm.max_hp, 0) * 100), 0) as avg_hp
			FROM death_arena_teams dt
			LEFT JOIN death_arena_team_members dtm ON dt.id = dtm.team_id
			WHERE dt.id = $1
			GROUP BY dt.id, dt.tournament_id, dt.team_name, dt.seed_number, dt.eliminated,
			         dt.eliminated_at_round, dt.final_placement, dt.created_at
		`, *teamID).Scan(&tm.ID, &tm.TournamentID, &tm.TeamName, &tm.SeedNumber,
			&tm.Eliminated, &tm.EliminatedAtRound, &tm.FinalPlacement, &tm.CreatedAt,
			&tm.MemberCount, &tm.AvgHP)
		if errTeam == nil {
			team = &tm
		}

		rows, err := db.DB.Query(`
			SELECT dtm.id, dtm.team_id, dtm.player_id, p.username, COALESCE(p.avatar_url, ''), p.rank,
			       dtm.current_hp, dtm.max_hp, dtm.current_mana, dtm.max_mana,
			       dtm.eliminated, dtm.eliminated_at_round, dtm.joined_at
			FROM death_arena_team_members dtm
			JOIN players p ON dtm.player_id = p.id
			WHERE dtm.team_id = $1
			ORDER BY dtm.joined_at
		`, *teamID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var m TeamMember
				if errScan := rows.Scan(&m.ID, &m.TeamID, &m.PlayerID, &m.Username, &m.AvatarURL,
					&m.Rank, &m.CurrentHP, &m.MaxHP, &m.CurrentMana, &m.MaxMana,
					&m.Eliminated, &m.EliminatedAtRound, &m.JoinedAt); errScan == nil {
					members = append(members, m)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"tournament":    t,
		"is_registered": isRegistered,
		"team_id":       teamID,
		"team":          team,
		"members":       members,
	})
}

// RegisterForTournament registers the player for the current tournament
func RegisterForTournament(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Find current REGISTRATION tournament
	var tournamentID string
	err = tx.QueryRow(`
		SELECT id FROM death_arena_tournaments WHERE status = 'REGISTRATION' 
		ORDER BY scheduled_at DESC LIMIT 1
	`).Scan(&tournamentID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có giải đấu nào đang mở đăng ký"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Check if already registered
	var exists2 bool
	_ = tx.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM death_arena_team_members dtm
			JOIN death_arena_teams dt ON dtm.team_id = dt.id
			WHERE dtm.player_id = $1 AND dt.tournament_id = $2
		)
	`, authID, tournamentID).Scan(&exists2)
	if exists2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu đã đăng ký giải đấu này rồi"})
		return
	}

	// Check max total players limit
	var currentTotalPlayers int
	_ = tx.QueryRow(`SELECT total_players FROM death_arena_tournaments WHERE id = $1`, tournamentID).Scan(&currentTotalPlayers)
	if currentTotalPlayers >= MaxTournamentPlayers {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Giải đấu đã đủ giới hạn tối đa %d người chơi (%d đội)!", MaxTournamentPlayers, MaxTournamentTeams),
		})
		return
	}

	// Load player to get HP/Mana
	var playerHP, playerMaxHP, playerMana, playerMaxMana float64
	err = tx.QueryRow(`
		SELECT hp, base_hp, current_mana, max_mana FROM players WHERE id = $1
	`, authID).Scan(&playerHP, &playerMaxHP, &playerMana, &playerMaxMana)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải thông tin nhân vật"})
		return
	}

	// Check if there's an unassigned team (not full yet, < 6 members)
	var teamID string
	var teamMemberCount int
	err = tx.QueryRow(`
		SELECT dt.id, COUNT(dtm.id)
		FROM death_arena_teams dt
		LEFT JOIN death_arena_team_members dtm ON dt.id = dtm.team_id
		WHERE dt.tournament_id = $1
		GROUP BY dt.id
		HAVING COUNT(dtm.id) < 6
		ORDER BY COUNT(dtm.id) DESC
		LIMIT 1
	`, tournamentID).Scan(&teamID, &teamMemberCount)

	if err == sql.ErrNoRows {
		// Check if we reached the max teams limit before creating a new team
		var currentTeams int
		_ = tx.QueryRow(`SELECT COUNT(*) FROM death_arena_teams WHERE tournament_id = $1`, tournamentID).Scan(&currentTeams)
		if currentTeams >= MaxTournamentTeams {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Giải đấu đã đạt giới hạn tối đa %d đội, các đội hiện tại đã kín chỗ!", MaxTournamentTeams),
			})
			return
		}

		// No team available, create new team
		teamName := generateTeamName()
		var seedNum int
		_ = tx.QueryRow(`
			SELECT COALESCE(MAX(seed_number), 0) + 1 FROM death_arena_teams WHERE tournament_id = $1
		`, tournamentID).Scan(&seedNum)

		err = tx.QueryRow(`
			INSERT INTO death_arena_teams (tournament_id, team_name, seed_number)
			VALUES ($1, $2, $3) RETURNING id
		`, tournamentID, teamName, seedNum).Scan(&teamID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo đội mới"})
			return
		}
		teamMemberCount = 0
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Add player to team
	_, err = tx.Exec(`
		INSERT INTO death_arena_team_members (team_id, player_id, current_hp, max_hp, current_mana, max_mana)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, teamID, authID, playerHP, playerMaxHP, playerMana, playerMaxMana)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể thêm vào đội"})
		return
	}

	// Update tournament total players
	_, err = tx.Exec(`
		UPDATE death_arena_tournaments 
		SET total_players = total_players + 1,
		    total_teams = (SELECT COUNT(DISTINCT team_id) FROM death_arena_team_members WHERE team_id IN 
		                   (SELECT id FROM death_arena_teams WHERE tournament_id = $1))
		WHERE id = $1
	`, tournamentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật giải đấu"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Đăng ký thành công!",
		"team_id":           teamID,
		"team_member_count": teamMemberCount + 1,
	})
}

// UnregisterFromTournament removes player from tournament
func UnregisterFromTournament(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	tx, err := db.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Find current REGISTRATION tournament
	var tournamentID string
	err = tx.QueryRow(`
		SELECT id FROM death_arena_tournaments WHERE status = 'REGISTRATION' 
		ORDER BY scheduled_at DESC LIMIT 1
	`).Scan(&tournamentID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Không có giải đấu nào đang mở đăng ký"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Find player's team
	var teamID string
	err = tx.QueryRow(`
		SELECT dtm.team_id FROM death_arena_team_members dtm
		JOIN death_arena_teams dt ON dtm.team_id = dt.id
		WHERE dtm.player_id = $1 AND dt.tournament_id = $2
	`, authID, tournamentID).Scan(&teamID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đạo hữu chưa đăng ký giải đấu này"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Remove player from team
	_, err = tx.Exec(`
		DELETE FROM death_arena_team_members WHERE player_id = $1 AND team_id = $2
	`, authID, teamID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hủy đăng ký"})
		return
	}

	// If team is now empty, delete the team
	var memberCount int
	_ = tx.QueryRow(`SELECT COUNT(*) FROM death_arena_team_members WHERE team_id = $1`, teamID).Scan(&memberCount)
	if memberCount == 0 {
		_, _ = tx.Exec(`DELETE FROM death_arena_teams WHERE id = $1`, teamID)
	}

	// Update tournament stats
	_, _ = tx.Exec(`
		UPDATE death_arena_tournaments 
		SET total_players = (SELECT COUNT(*) FROM death_arena_team_members dtm 
		                     JOIN death_arena_teams dt ON dtm.team_id = dt.id 
		                     WHERE dt.tournament_id = $1),
		    total_teams = (SELECT COUNT(DISTINCT team_id) FROM death_arena_team_members dtm
		                   JOIN death_arena_teams dt ON dtm.team_id = dt.id
		                   WHERE dt.tournament_id = $1)
		WHERE id = $1
	`, tournamentID)

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Hủy đăng ký thành công"})
}
