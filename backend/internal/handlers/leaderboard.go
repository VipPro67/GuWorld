package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/cache"
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
	X               *int    `json:"x,omitempty"`
	Y               *int    `json:"y,omitempty"`
	Region          string  `json:"region,omitempty"`
	IsOnline        bool    `json:"is_online"`
}

type LeaderboardResponse struct {
	Category     string             `json:"category"`
	Entries      []LeaderboardEntry `json:"entries"`
	MyRank       *int               `json:"my_rank"`
	MyValue      float64            `json:"my_value"`
	TotalPlayers int                `json:"total_players"`
	LastUpdated  string             `json:"last_updated"`
}

type LeaderboardSettings struct {
	ShowOnLeaderboard bool `json:"show_on_leaderboard"`
}

const cacheTTL = 24 * time.Hour

// playerRankCacheTTL is how long we cache a player's own leaderboard rank.
// The leaderboard list itself is cached for 24h; per-player rank being ~5min
// stale is acceptable and avoids a full ROW_NUMBER() scan per request.
const playerRankCacheTTL = 5 * time.Minute

type leaderboardCacheEntry struct {
	entries      []LeaderboardEntry
	totalPlayers int
	cachedAt     time.Time
}

var (
	leaderboardCache       = make(map[string]*leaderboardCacheEntry)
	leaderboardCacheMu     sync.RWMutex
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

type cachedRankResult struct {
	Rank  int     `json:"rank"`
	Value float64 `json:"value"`
	Found bool    `json:"found"`
}

// cachedGetMyRank returns the calling player's position on the given leaderboard
// category, using a Redis cache to avoid a full ROW_NUMBER() window-function scan
// on every request when the top-50 list is already served from the in-memory cache.
func cachedGetMyRank(category string, authID uuid.UUID) (*int, float64) {
	cacheKey := fmt.Sprintf("lb_rank:%s:%s", category, authID.String())
	var cached cachedRankResult
	if err := cache.GetJSON(cacheKey, &cached); err == nil {
		if !cached.Found {
			return nil, 0
		}
		return &cached.Rank, cached.Value
	}

	rankQuery := buildRankQuery(category)
	if rankQuery == "" {
		return nil, 0
	}

	var pos int
	var myValue float64
	err := db.DB.QueryRow(rankQuery, authID).Scan(&pos, &myValue)
	result := cachedRankResult{Found: err == nil}
	if err == nil {
		result.Rank = pos
		result.Value = myValue
	}
	_ = cache.SetJSON(cacheKey, result, playerRankCacheTTL)
	if !result.Found {
		return nil, 0
	}
	return &result.Rank, result.Value
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
		"sat_khi":         true,
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
		myRank, myValue := cachedGetMyRank(category, authID)

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
			SELECT p.id::text, p.username, COALESCE(p.avatar_url, ''), p.rank,
				CASE WHEN p.rank >= 6 THEN (COALESCE(ia.earth_calamity_count, 0) + COALESCE(ia.heavenly_tribulation_count, 0) + COALESCE(ia.grand_tribulation_count, 0) + COALESCE(ia.myriad_tribulation_count, 0) + COALESCE(ia.chaos_calamity_count, 0)) ELSE p.sub_rank END as sub_rank,
				CASE WHEN p.rank >= 6 THEN (p.rank * 10000 + COALESCE(ia.earth_calamity_count, 0) + COALESCE(ia.heavenly_tribulation_count, 0) + COALESCE(ia.grand_tribulation_count, 0) + COALESCE(ia.myriad_tribulation_count, 0) + COALESCE(ia.chaos_calamity_count, 0))::float8 ELSE (p.rank * 10 + p.sub_rank)::float8 END as val,
				CASE WHEN p.rank >= 6 THEN p.rank::text || '.' || (COALESCE(ia.earth_calamity_count, 0) + COALESCE(ia.heavenly_tribulation_count, 0) + COALESCE(ia.grand_tribulation_count, 0) + COALESCE(ia.myriad_tribulation_count, 0) + COALESCE(ia.chaos_calamity_count, 0))::text ELSE p.rank::text || '.' || p.sub_rank::text END as label,
				p.x, p.y, COALESCE(r.name, p.current_region, ''),
				(p.last_activity >= NOW() - INTERVAL '5 minutes' OR p.last_mana_update >= NOW() - INTERVAL '5 minutes') as is_online
			FROM players p
			JOIN leaderboard_settings ls ON p.id = ls.player_id
			LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
			LEFT JOIN regions r ON p.current_region = r.id
			WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			ORDER BY p.rank DESC, 
				CASE WHEN p.rank >= 6 THEN (COALESCE(ia.earth_calamity_count, 0) + COALESCE(ia.heavenly_tribulation_count, 0) + COALESCE(ia.grand_tribulation_count, 0) + COALESCE(ia.myriad_tribulation_count, 0) + COALESCE(ia.chaos_calamity_count, 0)) ELSE p.sub_rank END DESC
			LIMIT 50`
	case "primeval_stones":
		query = `
			SELECT p.id::text, p.username, COALESCE(p.avatar_url, ''), p.rank, p.sub_rank,
				p.primeval_stones::float8 as val,
				p.primeval_stones::text as label,
				p.x, p.y, COALESCE(r.name, p.current_region, ''),
				(p.last_activity >= NOW() - INTERVAL '5 minutes' OR p.last_mana_update >= NOW() - INTERVAL '5 minutes') as is_online
			FROM players p
			JOIN leaderboard_settings ls ON p.id = ls.player_id
			LEFT JOIN regions r ON p.current_region = r.id
			WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			ORDER BY p.primeval_stones DESC
			LIMIT 50`
	case "gu_strongest":
		query = `
			WITH gu_max AS (
				SELECT pg.player_id as id,
					MAX(pg.gu_rank) as max_gu_rank
				FROM player_gu pg
				GROUP BY pg.player_id
			)
			SELECT p.id::text, p.username, COALESCE(p.avatar_url, ''), p.rank, p.sub_rank,
				COALESCE(gm.max_gu_rank, 0)::float8 as val,
				COALESCE(gm.max_gu_rank::text, '0') as label,
				p.x, p.y, COALESCE(r.name, p.current_region, ''),
				(p.last_activity >= NOW() - INTERVAL '5 minutes' OR p.last_mana_update >= NOW() - INTERVAL '5 minutes') as is_online
			FROM players p
			JOIN leaderboard_settings ls ON p.id = ls.player_id
			LEFT JOIN gu_max gm ON gm.id = p.id
			LEFT JOIN regions r ON p.current_region = r.id
			WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			ORDER BY val DESC
			LIMIT 50`
	case "immortal_stones":
		query = `
			SELECT p.id::text, p.username, COALESCE(p.avatar_url, ''), p.rank, p.sub_rank,
				p.immortal_stones as val,
				p.immortal_stones::text as label,
				p.x, p.y, COALESCE(r.name, p.current_region, ''),
				(p.last_activity >= NOW() - INTERVAL '5 minutes' OR p.last_mana_update >= NOW() - INTERVAL '5 minutes') as is_online
			FROM players p
			JOIN leaderboard_settings ls ON p.id = ls.player_id
			LEFT JOIN regions r ON p.current_region = r.id
			WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
			ORDER BY p.immortal_stones DESC
			LIMIT 50`
	case "sat_khi":
		query = `
			SELECT p.id::text, p.username, COALESCE(p.avatar_url, ''), p.rank, p.sub_rank,
				COALESCE(sk.level, 0)::float8 as val,
				'Tầng ' || COALESCE(sk.level, 0)::text as label,
				p.x, p.y, COALESCE(r.name, p.current_region, ''),
				(p.last_activity >= NOW() - INTERVAL '5 minutes' OR p.last_mana_update >= NOW() - INTERVAL '5 minutes') as is_online
			FROM players p
			LEFT JOIN leaderboard_settings ls ON p.id = ls.player_id
			LEFT JOIN regions r ON p.current_region = r.id
			JOIN LATERAL (
				SELECT (elem->>'value')::int as level
				FROM jsonb_array_elements(COALESCE(p.statuses, '[]'::jsonb)) elem
				WHERE elem->>'id' = 'sat_khi'
				  AND (elem->>'expires_at' IS NULL OR (elem->>'expires_at')::timestamptz > NOW())
				LIMIT 1
			) sk ON true
			WHERE (p.is_admin IS NULL OR p.is_admin = false)
			  AND sk.level > 0
			ORDER BY sk.level DESC, p.last_activity DESC
			LIMIT 50`
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), query)
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
		var x, y int
		var region string
		var isOnline bool
		if err := rows.Scan(&entry.PlayerID, &entry.Username, &entry.AvatarURL, &entry.CultivationRank, &entry.SubRank, &entry.Value, &label, &x, &y, &region, &isOnline); err == nil {
			entry.Position = pos
			entry.ValueLabel = label
			entry.X = &x
			entry.Y = &y
			entry.Region = region
			entry.IsOnline = isOnline || isPlayerOnline(entry.PlayerID)
			entries = append(entries, entry)
			pos++
		}
	}

	// Count total visible players
	var totalPlayers int
	if category == "sat_khi" {
		_ = db.DB.QueryRowContext(c.Request.Context(), `
			SELECT COUNT(*) FROM players p
			JOIN LATERAL (
				SELECT (elem->>'value')::int as level
				FROM jsonb_array_elements(COALESCE(p.statuses, '[]'::jsonb)) elem
				WHERE elem->>'id' = 'sat_khi'
				  AND (elem->>'expires_at' IS NULL OR (elem->>'expires_at')::timestamptz > NOW())
				LIMIT 1
			) sk ON true
			WHERE (p.is_admin IS NULL OR p.is_admin = false) AND sk.level > 0
		`).Scan(&totalPlayers)
	} else {
		_ = db.DB.QueryRowContext(c.Request.Context(), `
			SELECT COUNT(*) FROM players p
			JOIN leaderboard_settings ls ON p.id = ls.player_id
			WHERE ls.show_on_leaderboard = true AND (p.is_admin IS NULL OR p.is_admin = false)
		`).Scan(&totalPlayers)
	}

	// Cache the result
	leaderboardCacheStore(category, entries, totalPlayers)
	lastUpdatedStr = time.Now().Format("2006-01-02T15:04:05Z07:00")

	// Get caller's rank & value
	myRank, myValue := cachedGetMyRank(category, authID)

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
				SELECT p.id,
					CASE WHEN p.rank >= 6 THEN (p.rank * 10000 + COALESCE(ia.earth_calamity_count, 0) + COALESCE(ia.heavenly_tribulation_count, 0) + COALESCE(ia.grand_tribulation_count, 0) + COALESCE(ia.myriad_tribulation_count, 0) + COALESCE(ia.chaos_calamity_count, 0))::float8 ELSE (p.rank * 10 + p.sub_rank)::float8 END as val,
					ROW_NUMBER() OVER (
						ORDER BY p.rank DESC, 
						CASE WHEN p.rank >= 6 THEN (COALESCE(ia.earth_calamity_count, 0) + COALESCE(ia.heavenly_tribulation_count, 0) + COALESCE(ia.grand_tribulation_count, 0) + COALESCE(ia.myriad_tribulation_count, 0) + COALESCE(ia.chaos_calamity_count, 0)) ELSE p.sub_rank END DESC
					) as position
				FROM players p
				JOIN leaderboard_settings ls ON p.id = ls.player_id
				LEFT JOIN immortal_apertures ia ON p.id = ia.player_id
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
				LEFT JOIN (
					SELECT pg.player_id, MAX(pg.gu_rank) as max_gu_rank
					FROM player_gu pg
					GROUP BY pg.player_id
				) gm ON gm.player_id = p.id
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
	case "sat_khi":
		return `
			SELECT position, val FROM (
				SELECT p.id, COALESCE(sk.level, 0)::float8 as val,
					ROW_NUMBER() OVER (ORDER BY COALESCE(sk.level, 0) DESC) as position
				FROM players p
				LEFT JOIN leaderboard_settings ls ON p.id = ls.player_id
				LEFT JOIN LATERAL (
					SELECT (elem->>'value')::int as level
					FROM jsonb_array_elements(COALESCE(p.statuses, '[]'::jsonb)) elem
					WHERE elem->>'id' = 'sat_khi'
					  AND (elem->>'expires_at' IS NULL OR (elem->>'expires_at')::timestamptz > NOW())
					LIMIT 1
				) sk ON true
				WHERE (p.is_admin IS NULL OR p.is_admin = false) AND sk.level > 0
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
	err := db.DB.QueryRowContext(c.Request.Context(), `SELECT show_on_leaderboard FROM leaderboard_settings WHERE player_id = $1`, authID).Scan(&show)
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
