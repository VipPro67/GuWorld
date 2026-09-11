package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

const (
	// MaxTournamentTeams is the maximum number of teams allowed in a tournament (16 teams = 4 rounds)
	MaxTournamentTeams = 16
	// MaxTournamentPlayers is the maximum number of participants allowed (16 teams * 6 members = 96)
	MaxTournamentPlayers = MaxTournamentTeams * 6
)

// Tournament represents a death arena tournament
type Tournament struct {
	ID                    string     `json:"id"`
	Status                string     `json:"status"`
	ScheduledAt           time.Time  `json:"scheduled_at"`
	RegistrationStartedAt time.Time  `json:"registration_started_at"`
	RegistrationClosedAt  time.Time  `json:"registration_closed_at"`
	StartedAt             *time.Time `json:"started_at,omitempty"`
	FinishedAt            *time.Time `json:"finished_at,omitempty"`
	TotalPlayers          int        `json:"total_players"`
	TotalTeams            int        `json:"total_teams"`
	CurrentRound          int        `json:"current_round"`
	TotalRounds           int        `json:"total_rounds"`
	ChampionTeamID        *string    `json:"champion_team_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
}

// Team represents a team in the tournament
type Team struct {
	ID               string     `json:"id"`
	TournamentID     string     `json:"tournament_id"`
	TeamName         string     `json:"team_name"`
	SeedNumber       int        `json:"seed_number"`
	Eliminated       bool       `json:"eliminated"`
	EliminatedAtRound *int      `json:"eliminated_at_round,omitempty"`
	FinalPlacement   *int       `json:"final_placement,omitempty"`
	MemberCount      int        `json:"member_count"`
	AvgHP            float64    `json:"avg_hp"`
	CreatedAt        time.Time  `json:"created_at"`
}

// TeamMember represents a team member
type TeamMember struct {
	ID              string    `json:"id"`
	TeamID          string    `json:"team_id"`
	PlayerID        string    `json:"player_id"`
	Username        string    `json:"username"`
	AvatarURL       string    `json:"avatar_url"`
	Rank            int       `json:"rank"`
	CurrentHP       float64   `json:"current_hp"`
	MaxHP           float64   `json:"max_hp"`
	CurrentMana     float64   `json:"current_mana"`
	MaxMana         float64   `json:"max_mana"`
	Eliminated      bool      `json:"eliminated"`
	EliminatedAtRound *int    `json:"eliminated_at_round,omitempty"`
	JoinedAt        time.Time `json:"joined_at"`
}

// Match represents a tournament match
type Match struct {
	ID                string     `json:"id"`
	TournamentID      string     `json:"tournament_id"`
	RoundNumber       int        `json:"round_number"`
	BracketPosition   int        `json:"bracket_position"`
	TeamAID           string     `json:"team_a_id"`
	TeamAName         string     `json:"team_a_name"`
	TeamBID           *string    `json:"team_b_id,omitempty"`
	TeamBName         *string    `json:"team_b_name,omitempty"`
	WinnerTeamID      *string    `json:"winner_team_id,omitempty"`
	Status            string     `json:"status"`
	RewardStonesPerMember int    `json:"reward_stones_per_member"`
	CreatedAt         time.Time  `json:"created_at"`
	FinishedAt        *time.Time `json:"finished_at,omitempty"`
}

// ChampionReward represents a champion reward
type ChampionReward struct {
	ID                   string     `json:"id"`
	TournamentID         string     `json:"tournament_id"`
	PlayerID             string     `json:"player_id"`
	Username             string     `json:"username"`
	Placement            int        `json:"placement"`
	TeamID               string     `json:"team_id"`
	TeamName             string     `json:"team_name"`
	Title                string     `json:"title"`
	RewardStones         int        `json:"reward_stones"`
	RewardImmortalStones int        `json:"reward_immortal_stones"`
	RewardGuID           *string    `json:"reward_gu_id,omitempty"`
	RewardGuRank         *int       `json:"reward_gu_rank,omitempty"`
	RewardItems          []byte     `json:"reward_items"`
	Claimed              bool       `json:"claimed"`
	CreatedAt            time.Time  `json:"created_at"`
}

// GetCurrentTournament returns the current active tournament
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
		"message":    "Đăng ký thành công!",
		"team_id":    teamID,
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

// BracketResponse represents the JSON payload returned by GetBracket
type BracketResponse struct {
	Tournament *Tournament `json:"tournament"`
	Matches    []Match     `json:"matches"`
}

// GetBracket returns the tournament bracket
func GetBracket(c *gin.Context) {
	var cached BracketResponse
	if err := cache.GetJSON("death_arena:bracket:current", &cached); err == nil && cached.Tournament != nil {
		c.JSON(http.StatusOK, gin.H{
			"tournament": cached.Tournament,
			"matches":    cached.Matches,
		})
		return
	}

	// Find current/latest tournament
	var t Tournament
	err := db.DB.QueryRow(`
		SELECT id, status, scheduled_at, registration_started_at, registration_closed_at,
		       started_at, finished_at, total_players, total_teams, current_round, total_rounds,
		       champion_team_id, created_at
		FROM death_arena_tournaments
		WHERE status != 'REGISTRATION'
		ORDER BY scheduled_at DESC
		LIMIT 1
	`).Scan(&t.ID, &t.Status, &t.ScheduledAt, &t.RegistrationStartedAt, &t.RegistrationClosedAt,
		&t.StartedAt, &t.FinishedAt, &t.TotalPlayers, &t.TotalTeams, &t.CurrentRound, &t.TotalRounds,
		&t.ChampionTeamID, &t.CreatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"tournament": nil, "matches": []Match{}})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tournament"})
		return
	}

	// Get all matches
	rows, err := db.DB.Query(`
		SELECT m.id, m.tournament_id, m.round_number, m.bracket_position,
		       m.team_a_id, ta.team_name,
		       m.team_b_id, tb.team_name,
		       m.winner_team_id, m.status, m.reward_stones_per_member,
		       m.created_at, m.finished_at
		FROM death_arena_matches m
		JOIN death_arena_teams ta ON m.team_a_id = ta.id
		LEFT JOIN death_arena_teams tb ON m.team_b_id = tb.id
		WHERE m.tournament_id = $1
		ORDER BY m.round_number, m.bracket_position
	`, t.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches"})
		return
	}
	defer rows.Close()

	matches := []Match{}
	for rows.Next() {
		var m Match
		if errScan := rows.Scan(&m.ID, &m.TournamentID, &m.RoundNumber, &m.BracketPosition,
			&m.TeamAID, &m.TeamAName, &m.TeamBID, &m.TeamBName, &m.WinnerTeamID,
			&m.Status, &m.RewardStonesPerMember, &m.CreatedAt, &m.FinishedAt); errScan == nil {
			matches = append(matches, m)
		}
	}

	ttl := 3 * time.Second
	if t.Status == "FINISHED" {
		ttl = 10 * time.Minute
	}
	_ = cache.SetJSON("death_arena:bracket:current", BracketResponse{
		Tournament: &t,
		Matches:    matches,
	}, ttl)

	c.JSON(http.StatusOK, gin.H{
		"tournament": t,
		"matches":    matches,
	})
}

// GetMyTeam returns the player's team with member details
func GetMyTeam(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Find latest tournament
	var tournamentID, tournamentStatus string
	err := db.DB.QueryRow(`
		SELECT id, status FROM death_arena_tournaments
		ORDER BY scheduled_at DESC LIMIT 1
	`).Scan(&tournamentID, &tournamentStatus)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"team": nil, "message": "Không có giải đấu nào"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Find player's team ID
	var teamID string
	err = db.DB.QueryRow(`
		SELECT dtm.team_id
		FROM death_arena_team_members dtm
		JOIN death_arena_teams dt ON dtm.team_id = dt.id
		WHERE dt.tournament_id = $1 AND dtm.player_id = $2
		LIMIT 1
	`, tournamentID, authID).Scan(&teamID)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"team": nil, "tournament_status": tournamentStatus})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Find player's team details
	var team Team
	err = db.DB.QueryRow(`
		SELECT dt.id, dt.tournament_id, dt.team_name, dt.seed_number, dt.eliminated,
		       dt.eliminated_at_round, dt.final_placement, dt.created_at,
		       COUNT(dtm.id) as member_count,
		       COALESCE(AVG(dtm.current_hp / NULLIF(dtm.max_hp, 0) * 100), 0) as avg_hp
		FROM death_arena_teams dt
		LEFT JOIN death_arena_team_members dtm ON dt.id = dtm.team_id
		WHERE dt.id = $1
		GROUP BY dt.id, dt.tournament_id, dt.team_name, dt.seed_number, dt.eliminated,
		         dt.eliminated_at_round, dt.final_placement, dt.created_at
	`, teamID).Scan(&team.ID, &team.TournamentID, &team.TeamName, &team.SeedNumber,
		&team.Eliminated, &team.EliminatedAtRound, &team.FinalPlacement, &team.CreatedAt,
		&team.MemberCount, &team.AvgHP)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get team members
	rows, err := db.DB.Query(`
		SELECT dtm.id, dtm.team_id, dtm.player_id, p.username, COALESCE(p.avatar_url, ''), p.rank,
		       dtm.current_hp, dtm.max_hp, dtm.current_mana, dtm.max_mana,
		       dtm.eliminated, dtm.eliminated_at_round, dtm.joined_at
		FROM death_arena_team_members dtm
		JOIN players p ON dtm.player_id = p.id
		WHERE dtm.team_id = $1
		ORDER BY dtm.joined_at
	`, team.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch team members"})
		return
	}
	defer rows.Close()

	var members []TeamMember
	for rows.Next() {
		var m TeamMember
		if errScan := rows.Scan(&m.ID, &m.TeamID, &m.PlayerID, &m.Username, &m.AvatarURL,
			&m.Rank, &m.CurrentHP, &m.MaxHP, &m.CurrentMana, &m.MaxMana,
			&m.Eliminated, &m.EliminatedAtRound, &m.JoinedAt); errScan == nil {
			members = append(members, m)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"team":             team,
		"members":          members,
		"tournament_status": tournamentStatus,
	})
}

// GetMyMatches returns matches involving the player's team
func GetMyMatches(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Find player's team in latest tournament
	var teamID string
	err := db.DB.QueryRow(`
		SELECT dtm.team_id FROM death_arena_team_members dtm
		JOIN death_arena_teams dt ON dtm.team_id = dt.id
		JOIN death_arena_tournaments t ON dt.tournament_id = t.id
		WHERE dtm.player_id = $1
		ORDER BY t.scheduled_at DESC
		LIMIT 1
	`, authID).Scan(&teamID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"matches": []Match{}})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Get matches for this team
	rows, err := db.DB.Query(`
		SELECT m.id, m.tournament_id, m.round_number, m.bracket_position,
		       m.team_a_id, ta.team_name,
		       m.team_b_id, tb.team_name,
		       m.winner_team_id, m.status, m.reward_stones_per_member,
		       m.created_at, m.finished_at
		FROM death_arena_matches m
		JOIN death_arena_teams ta ON m.team_a_id = ta.id
		LEFT JOIN death_arena_teams tb ON m.team_b_id = tb.id
		WHERE m.team_a_id = $1 OR m.team_b_id = $1
		ORDER BY m.round_number, m.bracket_position
	`, teamID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches"})
		return
	}
	defer rows.Close()

	var matches []Match
	for rows.Next() {
		var m Match
		if errScan := rows.Scan(&m.ID, &m.TournamentID, &m.RoundNumber, &m.BracketPosition,
			&m.TeamAID, &m.TeamAName, &m.TeamBID, &m.TeamBName, &m.WinnerTeamID,
			&m.Status, &m.RewardStonesPerMember, &m.CreatedAt, &m.FinishedAt); errScan == nil {
			matches = append(matches, m)
		}
	}

	c.JSON(http.StatusOK, gin.H{"matches": matches})
}

// MatchLogResponse represents the payload returned for a match combat replay
type MatchLogResponse struct {
	Match        Match         `json:"match"`
	TeamAMembers []TeamMember  `json:"team_a_members"`
	TeamBMembers []TeamMember  `json:"team_b_members"`
	CombatLog    []interface{} `json:"combat_log"`
}

// GetMatchCombatLog returns detailed match information and combat replay logs
func GetMatchCombatLog(c *gin.Context) {
	matchID := c.Param("id")
	if matchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã trận đấu"})
		return
	}

	cacheKey := fmt.Sprintf("death_arena:match_log:%s", matchID)
	var cached MatchLogResponse
	if err := cache.GetJSON(cacheKey, &cached); err == nil && cached.Match.ID != "" {
		c.JSON(http.StatusOK, cached)
		return
	}

	var m Match
	var teamBName sql.NullString
	var teamBID sql.NullString
	var winnerID sql.NullString
	var combatLogRaw []byte

	err := db.DB.QueryRow(`
		SELECT m.id, m.tournament_id, m.round_number, m.bracket_position,
		       m.team_a_id, ta.team_name,
		       m.team_b_id, COALESCE(tb.team_name, ''),
		       m.winner_team_id, m.status, m.reward_stones_per_member,
		       m.created_at, m.finished_at,
		       COALESCE(m.combat_log, '[]'::jsonb)
		FROM death_arena_matches m
		JOIN death_arena_teams ta ON m.team_a_id = ta.id
		LEFT JOIN death_arena_teams tb ON m.team_b_id = tb.id
		WHERE m.id = $1
	`, matchID).Scan(
		&m.ID, &m.TournamentID, &m.RoundNumber, &m.BracketPosition,
		&m.TeamAID, &m.TeamAName,
		&teamBID, &teamBName,
		&winnerID, &m.Status, &m.RewardStonesPerMember,
		&m.CreatedAt, &m.FinishedAt,
		&combatLogRaw,
	)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy trận đấu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn trận đấu"})
		return
	}

	if teamBID.Valid && teamBID.String != "" {
		m.TeamBID = &teamBID.String
	}
	if teamBName.Valid && teamBName.String != "" {
		m.TeamBName = &teamBName.String
	}
	if winnerID.Valid && winnerID.String != "" {
		m.WinnerTeamID = &winnerID.String
	}

	var combatLogs []interface{}
	if len(combatLogRaw) > 0 {
		_ = json.Unmarshal(combatLogRaw, &combatLogs)
	}
	if combatLogs == nil {
		combatLogs = []interface{}{}
	}

	loadMembers := func(teamID string) []TeamMember {
		if teamID == "" {
			return []TeamMember{}
		}
		rows, err := db.DB.Query(`
			SELECT dtm.id, dtm.team_id, dtm.player_id, p.username, COALESCE(p.avatar_url, ''), p.rank,
			       dtm.current_hp, dtm.max_hp, dtm.current_mana, dtm.max_mana,
			       dtm.eliminated, dtm.eliminated_at_round, dtm.joined_at
			FROM death_arena_team_members dtm
			JOIN players p ON dtm.player_id = p.id
			WHERE dtm.team_id = $1
			ORDER BY p.rank DESC, dtm.joined_at
		`, teamID)
		if err != nil {
			return []TeamMember{}
		}
		defer rows.Close()

		list := []TeamMember{}
		for rows.Next() {
			var tm TeamMember
			if err := rows.Scan(&tm.ID, &tm.TeamID, &tm.PlayerID, &tm.Username, &tm.AvatarURL,
				&tm.Rank, &tm.CurrentHP, &tm.MaxHP, &tm.CurrentMana, &tm.MaxMana,
				&tm.Eliminated, &tm.EliminatedAtRound, &tm.JoinedAt); err == nil {
				list = append(list, tm)
			}
		}
		return list
	}

	teamAMembers := loadMembers(m.TeamAID)
	teamBMembers := []TeamMember{}
	if m.TeamBID != nil && *m.TeamBID != "" {
		teamBMembers = loadMembers(*m.TeamBID)
	}

	resp := MatchLogResponse{
		Match:        m,
		TeamAMembers: teamAMembers,
		TeamBMembers: teamBMembers,
		CombatLog:    combatLogs,
	}

	if m.Status == "FINISHED" || m.Status == "BYE" {
		_ = cache.SetJSON(cacheKey, resp, 24*time.Hour)
	}

	c.JSON(http.StatusOK, resp)
}

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
		"message":    "Nhận phần thưởng thành công!",
		"placement":  reward.Placement,
		"title":      reward.Title,
		"stones":     reward.RewardStones,
		"immortal_stones": reward.RewardImmortalStones,
	})
}

// generateTeamName generates a random team name
func generateTeamName() string {
	prefixes := []string{"Thiên", "Địa", "Nhân", "Phong", "Lôi", "Hỏa", "Thủy", "Mộc", "Kim", "Sơn", "Hà", "Tinh"}
	suffixes := []string{"Sư", "Tử", "Vương", "Hầu", "Tướng", "Sĩ", "Giả", "Đạo", "Cung", "Môn", "Phái", "Bang"}
	return prefixes[rand.Intn(len(prefixes))] + suffixes[rand.Intn(len(suffixes))]
}