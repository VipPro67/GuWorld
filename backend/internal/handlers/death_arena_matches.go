package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

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
		"team":              team,
		"members":           members,
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
