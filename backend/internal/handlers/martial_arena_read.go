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

// currentMartialEventIDs returns the IDs of the brackets belonging to the current
// event: active (LOCKED/IN_PROGRESS) brackets if any, otherwise the most recent
// finished event.
func currentMartialEventIDs() []string {
	rows, err := db.DB.Query(`SELECT id FROM martial_tournaments WHERE status IN ('LOCKED','IN_PROGRESS') ORDER BY rank`)
	if err == nil {
		var ids []string
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				ids = append(ids, id)
			}
		}
		rows.Close()
		if len(ids) > 0 {
			return ids
		}
	}

	rows2, err := db.DB.Query(`
		SELECT id FROM martial_tournaments
		WHERE status = 'FINISHED' AND scheduled_at::date = (SELECT MAX(scheduled_at::date) FROM martial_tournaments WHERE status = 'FINISHED')
		ORDER BY rank
	`)
	if err != nil {
		return nil
	}
	defer rows2.Close()
	var ids []string
	for rows2.Next() {
		var id string
		if rows2.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func scanMartialTournament(row *sql.Row) (MartialTournament, error) {
	var t MartialTournament
	err := row.Scan(&t.ID, &t.Status, &t.Rank, &t.ScheduledAt, &t.StartedAt, &t.FinishedAt,
		&t.TotalPlayers, &t.CurrentRound, &t.TotalRounds, &t.ChampionPlayerID, &t.CreatedAt)
	return t, err
}

func loadMartialTournament(id string) (MartialTournament, error) {
	return scanMartialTournament(db.DB.QueryRow(`
		SELECT id, status, rank, scheduled_at, started_at, finished_at,
		       total_players, current_round, total_rounds, champion_player_id, created_at
		FROM martial_tournaments WHERE id = $1
	`, id))
}

func myMartialCombatant(tournamentID string, playerID interface{}) *MartialCombatant {
	var c MartialCombatant
	err := db.DB.QueryRow(`
		SELECT id, tournament_id, player_id, seed_number, rank, sub_rank, eliminated, eliminated_at_round, final_placement, created_at
		FROM martial_combatants WHERE tournament_id = $1 AND player_id = $2
	`, tournamentID, playerID).Scan(&c.ID, &c.TournamentID, &c.PlayerID, &c.SeedNumber, &c.Rank, &c.SubRank,
		&c.Eliminated, &c.EliminatedAtRound, &c.FinalPlacement, &c.CreatedAt)
	if err != nil {
		return nil
	}
	return &c
}

// GetCurrentMartialTournament lists the brackets of the current event and the
// caller's participation in each.
func GetCurrentMartialTournament(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ids := currentMartialEventIDs()
	type entry struct {
		Rank        int               `json:"rank"`
		Tournament  MartialTournament `json:"tournament"`
		MyCombatant *MartialCombatant `json:"my_combatant,omitempty"`
	}
	entries := []entry{}
	for _, id := range ids {
		t, err := loadMartialTournament(id)
		if err != nil {
			continue
		}
		e := entry{Rank: t.Rank, Tournament: t}
		if mc := myMartialCombatant(id, authID); mc != nil {
			e.MyCombatant = mc
		}
		entries = append(entries, e)
	}

	c.JSON(http.StatusOK, gin.H{"tournaments": entries})
}

// GetMartialBracket returns each bracket of the current event with its matches.
func GetMartialBracket(c *gin.Context) {
	var cached struct {
		Tournaments []struct {
			Rank       int               `json:"rank"`
			Tournament MartialTournament `json:"tournament"`
			Matches    []MartialMatch    `json:"matches"`
		} `json:"tournaments"`
	}
	if err := cache.GetJSON("martial_arena:bracket:current", &cached); err == nil && len(cached.Tournaments) > 0 {
		c.JSON(http.StatusOK, gin.H{"tournaments": cached.Tournaments})
		return
	}

	ids := currentMartialEventIDs()
	type entry struct {
		Rank       int               `json:"rank"`
		Tournament MartialTournament `json:"tournament"`
		Matches    []MartialMatch    `json:"matches"`
	}
	entries := []entry{}
	for _, id := range ids {
		t, err := loadMartialTournament(id)
		if err != nil {
			continue
		}
		matches, _ := loadMartialMatches(id)
		entries = append(entries, entry{Rank: t.Rank, Tournament: t, Matches: matches})
	}

	ttl := 3 * time.Second
	_ = cache.SetJSON("martial_arena:bracket:current", gin.H{"tournaments": entries}, ttl)
	c.JSON(http.StatusOK, gin.H{"tournaments": entries})
}

func loadMartialMatches(tournamentID string) ([]MartialMatch, error) {
	rows, err := db.DB.Query(`
		SELECT m.id, m.tournament_id, m.round_number, m.bracket_position,
		       m.combatant_a_id, pa.username,
		       m.combatant_b_id, pb.username,
		       m.winner_combatant_id, m.status, m.reward_stones,
		       m.created_at, m.finished_at
		FROM martial_matches m
		JOIN martial_combatants ca ON m.combatant_a_id = ca.id
		JOIN players pa ON ca.player_id = pa.id
		LEFT JOIN martial_combatants cb ON m.combatant_b_id = cb.id
		LEFT JOIN players pb ON cb.player_id = pb.id
		WHERE m.tournament_id = $1
		ORDER BY m.round_number, m.bracket_position
	`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	matches := []MartialMatch{}
	for rows.Next() {
		var m MartialMatch
		var bID, bName, wID sql.NullString
		if errScan := rows.Scan(&m.ID, &m.TournamentID, &m.RoundNumber, &m.BracketPosition,
			&m.CombatantAID, &m.CombatantAName, &bID, &bName, &wID, &m.Status, &m.RewardStones,
			&m.CreatedAt, &m.FinishedAt); errScan == nil {
			if bID.Valid {
				s := bID.String
				m.CombatantBID = &s
			}
			if bName.Valid {
				s := bName.String
				m.CombatantBName = &s
			}
			if wID.Valid {
				s := wID.String
				m.WinnerCombatantID = &s
			}
			matches = append(matches, m)
		}
	}
	return matches, nil
}

// GetMyMartialMatches returns matches involving the caller's combatant in the
// current event.
func GetMyMartialMatches(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var combatantID string
	err := db.DB.QueryRow(`
		SELECT mc.id FROM martial_combatants mc
		WHERE mc.player_id = $1
		ORDER BY mc.created_at DESC LIMIT 1
	`, authID).Scan(&combatantID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"matches": []MartialMatch{}})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	rows, err := db.DB.Query(`
		SELECT m.id, m.tournament_id, m.round_number, m.bracket_position,
		       m.combatant_a_id, pa.username,
		       m.combatant_b_id, pb.username,
		       m.winner_combatant_id, m.status, m.reward_stones,
		       m.created_at, m.finished_at
		FROM martial_matches m
		JOIN martial_combatants ca ON m.combatant_a_id = ca.id
		JOIN players pa ON ca.player_id = pa.id
		LEFT JOIN martial_combatants cb ON m.combatant_b_id = cb.id
		LEFT JOIN players pb ON cb.player_id = pb.id
		WHERE m.combatant_a_id = $1 OR m.combatant_b_id = $1
		ORDER BY m.round_number, m.bracket_position
	`, combatantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches"})
		return
	}
	defer rows.Close()

	matches := []MartialMatch{}
	for rows.Next() {
		var m MartialMatch
		var bID, bName, wID sql.NullString
		if errScan := rows.Scan(&m.ID, &m.TournamentID, &m.RoundNumber, &m.BracketPosition,
			&m.CombatantAID, &m.CombatantAName, &bID, &bName, &wID, &m.Status, &m.RewardStones,
			&m.CreatedAt, &m.FinishedAt); errScan == nil {
			if bID.Valid {
				s := bID.String
				m.CombatantBID = &s
			}
			if bName.Valid {
				s := bName.String
				m.CombatantBName = &s
			}
			if wID.Valid {
				s := wID.String
				m.WinnerCombatantID = &s
			}
			matches = append(matches, m)
		}
	}

	c.JSON(http.StatusOK, gin.H{"matches": matches})
}

// GetMartialMatchCombatLog returns a match's combat replay log.
func GetMartialMatchCombatLog(c *gin.Context) {
	matchID := c.Param("id")
	if matchID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã trận đấu"})
		return
	}

	cacheKey := fmt.Sprintf("martial_arena:match_log:%s", matchID)
	var cached struct {
		Match     MartialMatch  `json:"match"`
		CombatLog []interface{} `json:"combat_log"`
	}
	if err := cache.GetJSON(cacheKey, &cached); err == nil && cached.Match.ID != "" {
		c.JSON(http.StatusOK, cached)
		return
	}

	var m MartialMatch
	var bID, bName, wID sql.NullString
	var combatLogRaw []byte
	err := db.DB.QueryRow(`
		SELECT m.id, m.tournament_id, m.round_number, m.bracket_position,
		       m.combatant_a_id, pa.username,
		       m.combatant_b_id, COALESCE(pb.username, ''),
		       m.winner_combatant_id, m.status, m.reward_stones,
		       m.created_at, m.finished_at,
		       COALESCE(m.combat_log, '[]'::jsonb)
		FROM martial_matches m
		JOIN martial_combatants ca ON m.combatant_a_id = ca.id
		JOIN players pa ON ca.player_id = pa.id
		LEFT JOIN martial_combatants cb ON m.combatant_b_id = cb.id
		LEFT JOIN players pb ON cb.player_id = pb.id
		WHERE m.id = $1
	`, matchID).Scan(&m.ID, &m.TournamentID, &m.RoundNumber, &m.BracketPosition,
		&m.CombatantAID, &m.CombatantAName, &bID, &bName, &wID, &m.Status, &m.RewardStones,
		&m.CreatedAt, &m.FinishedAt, &combatLogRaw)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy trận đấu"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi truy vấn trận đấu"})
		return
	}

	if bID.Valid {
		s := bID.String
		m.CombatantBID = &s
	}
	if bName.Valid {
		s := bName.String
		m.CombatantBName = &s
	}
	if wID.Valid {
		s := wID.String
		m.WinnerCombatantID = &s
	}

	var combatLogs []interface{}
	if len(combatLogRaw) > 0 {
		_ = json.Unmarshal(combatLogRaw, &combatLogs)
	}
	if combatLogs == nil {
		combatLogs = []interface{}{}
	}

	resp := gin.H{"match": m, "combat_log": combatLogs}
	if m.Status == "FINISHED" || m.Status == "BYE" {
		_ = cache.SetJSON(cacheKey, resp, 24*time.Hour)
	}
	c.JSON(http.StatusOK, resp)
}

// GetMartialHistory returns past (finished) brackets.
func GetMartialHistory(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT id, status, rank, scheduled_at, started_at, finished_at,
		       total_players, current_round, total_rounds, champion_player_id, created_at
		FROM martial_tournaments
		WHERE status = 'FINISHED'
		ORDER BY finished_at DESC
		LIMIT 40
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	defer rows.Close()

	var tournaments []MartialTournament
	for rows.Next() {
		var t MartialTournament
		if errScan := rows.Scan(&t.ID, &t.Status, &t.Rank, &t.ScheduledAt, &t.StartedAt, &t.FinishedAt,
			&t.TotalPlayers, &t.CurrentRound, &t.TotalRounds, &t.ChampionPlayerID, &t.CreatedAt); errScan == nil {
			tournaments = append(tournaments, t)
		}
	}

	c.JSON(http.StatusOK, gin.H{"tournaments": tournaments})
}
