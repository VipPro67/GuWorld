package handlers

import (
	"database/sql"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type LeaderboardEntry struct {
	Position        int     `json:"position"`
	PlayerID        string  `json:"player_id"`
	Username        string  `json:"username"`
	AvatarURL       string  `json:"avatar_url"`
	CultivationRank int     `json:"cultivation_rank"`
	SubRank         int     `json:"sub_rank"`
	Value           float64 `json:"value"`
	ValueLabel      string  `json:"value_label"`
}

type LeaderboardResponse struct {
	Category      string             `json:"category"`
	Entries       []LeaderboardEntry `json:"entries"`
	MyRank        *int               `json:"my_rank"`
	MyValue       float64            `json:"my_value"`
	TotalPlayers  int                `json:"total_players"`
	LastUpdated   string             `json:"last_updated"`
}

type LeaderboardSettings struct {
	ShowOnLeaderboard bool `json:"show_on_leaderboard"`
}

const cacheTTL = 24 * time.Hour

type leaderboardCacheEntry struct {
	entries      []LeaderboardEntry
	totalPlayers int
	cachedAt     time.Time
}

var (
	leaderboardCache   = make(map[string]*leaderboardCacheEntry)
	leaderboardCacheMu sync.RWMutex
	leaderboardLastUpdated time.Time
)

func leaderboardCacheLoad(category string) (*leaderboardCacheEntry, bool) {
	leaderboardCacheMu.RLock()
	defer leaderboardCacheMu.RUnlock()
	entry, ok := leaderboardCache[category]
	if !ok {
		return nil, false
	}
	if time.Since(entry.cachedAt) >= cacheTTL {
		return nil, false
	}
	return entry, true
}

func leaderboardCacheStore(category string, entries []LeaderboardEntry, totalPlayers int) {
	leaderboardCacheMu.Lock()
	defer leaderboardCacheMu.Unlock()
	leaderboardCache[category] = &leaderboardCacheEntry{
		entries:      entries,
		totalPlayers: totalPlayers,
		cachedAt:     time.Now(),
	}
	leaderboardLastUpdated = time.Now()
}

func ensureLeaderboardSetting(playerID uuid.UUID) {
	_, err := db.DB.Exec(`
		INSERT INTO leaderboard_settings (player_id, show_on_leaderboard)
		VALUES ($1, TRUE)
		ON CONFLICT (player_id) DO NOTHING
	`, playerID)
	if err != nil {
		// Non-fatal; log if needed
	}
}

func GetLeaderboard(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ensureLeaderboardSetting(authID)

	category := c.Param("category")
	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing category"})
		return
	}

	validCategories := map[string]bool{
		"rank":            true,
		"primeval_stones": true,
		"gu_strongest":    true,
		"immortal_stones": true,
	}
	if !validCategories[category] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category: " + category})
		return
	}

	lastUpdatedStr := ""
	if !leaderboardLastUpdated.IsZero() {
		lastUpdatedStr = leaderboardLastUpdated.Format("2006-01-02T15:04:05Z07:00")
	}

	// Try cache
	if cached, ok := leaderboardCacheLoad(category); ok {
		var myRank *int
		var myValue float64

		rankQuery := buildRankQuery(category)
		if rankQuery != "" {
			var pos int
			err := db.DB.QueryRow(rankQuery, authID).Scan(&pos, &myValue)
			if err == nil {
				myRank = &pos
			}
		}

		c.JSON(http.StatusOK, LeaderboardResponse{
			Category:     category,
			Entries:      cached.entries,
			MyRank:       myRank,
			MyValue:      myValue,
			TotalPlayers: cached.totalPlayers,
			LastUpdated:  lastUpdatedStr,
		})
		return
	}

	// Compute fresh
	var query string
	switch category {
	case "rank":
		query = `
			SELECT p.id::text, p.username, COALESCE(p.avatar_url, ''), p.rank, p.sub_rank,
				(p.rank * 10 + p.sub_rank)::float8 as val,
				p.rank::text || '.' || p.sub_rank::text as label
			FROM players p
			JOIN leaderboard_settings ls ON p.id = ls.player_id
			WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			ORDER BY p.rank DESC, p.sub_rank DESC
			LIMIT 50`
	case "primeval_stones":
		query = `
			SELECT p.id::text, p.username, COALESCE(p.avatar_url, ''), p.rank, p.sub_rank,
				p.primeval_stones::float8 as val,
				p.primeval_stones::text as label
			FROM players p
			JOIN leaderboard_settings ls ON p.id = ls.player_id
			WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			ORDER BY p.primeval_stones DESC
			LIMIT 50`
	case "gu_strongest":
		query = `
			WITH gu_max AS (
				SELECT p.id,
					MAX(COALESCE(gd.rank, 0)) as max_gu_rank
				FROM players p
				LEFT JOIN LATERAL jsonb_array_elements(
					CASE WHEN jsonb_typeof(p.gu_inventory) = 'array' THEN p.gu_inventory ELSE '[]'::jsonb END
				) AS gu_elems ON true
				LEFT JOIN gu_dictionary gd ON gd.id = gu_elems->>'gu_id'
				GROUP BY p.id
			)
			SELECT p.id::text, p.username, COALESCE(p.avatar_url, ''), p.rank, p.sub_rank,
				COALESCE(gm.max_gu_rank, 0)::float8 as val,
				COALESCE(gm.max_gu_rank::text, '0') as label
			FROM players p
			JOIN leaderboard_settings ls ON p.id = ls.player_id
			LEFT JOIN gu_max gm ON gm.id = p.id
			WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			ORDER BY val DESC
			LIMIT 50`
	case "immortal_stones":
		query = `
			SELECT p.id::text, p.username, COALESCE(p.avatar_url, ''), p.rank, p.sub_rank,
				p.immortal_stones as val,
				p.immortal_stones::text as label
			FROM players p
			JOIN leaderboard_settings ls ON p.id = ls.player_id
			WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			ORDER BY p.immortal_stones DESC
			LIMIT 50`
	}

	rows, err := db.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
		return
	}
	defer rows.Close()

	entries := []LeaderboardEntry{}
	pos := 1
	for rows.Next() {
		var entry LeaderboardEntry
		var label string
		if err := rows.Scan(&entry.PlayerID, &entry.Username, &entry.AvatarURL, &entry.CultivationRank, &entry.SubRank, &entry.Value, &label); err == nil {
			entry.Position = pos
			entry.ValueLabel = label
			entries = append(entries, entry)
			pos++
		}
	}

	// Count total visible players
	var totalPlayers int
	_ = db.DB.QueryRow(`
		SELECT COUNT(*) FROM players p
		JOIN leaderboard_settings ls ON p.id = ls.player_id
		WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
	`).Scan(&totalPlayers)

	// Cache the result
	leaderboardCacheStore(category, entries, totalPlayers)
	lastUpdatedStr = time.Now().Format("2006-01-02T15:04:05Z07:00")

	// Get caller's rank & value
	var myRank *int
	var myValue float64

	rankQuery := buildRankQuery(category)
	if rankQuery != "" {
		var pos int
		err := db.DB.QueryRow(rankQuery, authID).Scan(&pos, &myValue)
		if err == nil {
			myRank = &pos
		}
	}

	c.JSON(http.StatusOK, LeaderboardResponse{
		Category:     category,
		Entries:      entries,
		MyRank:       myRank,
		MyValue:      myValue,
		TotalPlayers: totalPlayers,
		LastUpdated:  lastUpdatedStr,
	})
}

func buildRankQuery(category string) string {
	switch category {
	case "rank":
		return `
			SELECT position, val FROM (
				SELECT p.id, (p.rank * 10 + p.sub_rank)::float8 as val,
					ROW_NUMBER() OVER (ORDER BY p.rank DESC, p.sub_rank DESC) as position
				FROM players p
				JOIN leaderboard_settings ls ON p.id = ls.player_id
				WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			) sub WHERE sub.id = $1`
	case "primeval_stones":
		return `
			SELECT position, val FROM (
				SELECT p.id, p.primeval_stones::float8 as val,
					ROW_NUMBER() OVER (ORDER BY p.primeval_stones DESC) as position
				FROM players p
				JOIN leaderboard_settings ls ON p.id = ls.player_id
				WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			) sub WHERE sub.id = $1`
	case "gu_strongest":
		return `
			SELECT position, val FROM (
				SELECT p.id, COALESCE(gm.max_gu_rank, 0)::float8 as val,
					ROW_NUMBER() OVER (ORDER BY COALESCE(gm.max_gu_rank, 0) DESC) as position
				FROM players p
				JOIN leaderboard_settings ls ON p.id = ls.player_id
				LEFT JOIN LATERAL (
					SELECT MAX(COALESCE(gd.rank, 0)) as max_gu_rank
					FROM jsonb_array_elements(
						CASE WHEN jsonb_typeof(p.gu_inventory) = 'array' THEN p.gu_inventory ELSE '[]'::jsonb END
					) AS gu_elems
					LEFT JOIN gu_dictionary gd ON gd.id = gu_elems->>'gu_id'
				) gm ON true
				WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			) sub WHERE sub.id = $1`
	case "immortal_stones":
		return `
			SELECT position, val FROM (
				SELECT p.id, p.immortal_stones as val,
					ROW_NUMBER() OVER (ORDER BY p.immortal_stones DESC) as position
				FROM players p
				JOIN leaderboard_settings ls ON p.id = ls.player_id
				WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			) sub WHERE sub.id = $1`
	}
	return ""
}

func ToggleLeaderboardVisibility(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ensureLeaderboardSetting(authID)

	_, err := db.DB.Exec(`
		UPDATE leaderboard_settings
		SET show_on_leaderboard = NOT show_on_leaderboard
		WHERE player_id = $1
	`, authID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle visibility"})
		return
	}

	var show bool
	_ = db.DB.QueryRow(`SELECT show_on_leaderboard FROM leaderboard_settings WHERE player_id = $1`, authID).Scan(&show)

	c.JSON(http.StatusOK, gin.H{"show_on_leaderboard": show})
}

func GetLeaderboardSettings(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	ensureLeaderboardSetting(authID)

	var show bool
	err := db.DB.QueryRow(`SELECT show_on_leaderboard FROM leaderboard_settings WHERE player_id = $1`, authID).Scan(&show)
	if err != nil {
		if err == sql.ErrNoRows {
			show = true
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"show_on_leaderboard": show})
}
