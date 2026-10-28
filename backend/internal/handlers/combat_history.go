package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

type CombatHistoryEntry struct {
	ID         string          `json:"id"`
	PlayerID   string          `json:"player_id"`
	Winner     string          `json:"winner"`
	Logs       json.RawMessage `json:"logs"`
	Summary    json.RawMessage `json:"summary"`
	Combatants json.RawMessage `json:"combatants"`
	EnemyName  string          `json:"enemy_name"`
	EnemyType  string          `json:"enemy_type"`
	Region     string          `json:"region"`
	CreatedAt  time.Time       `json:"created_at"`
}

// CombatHistoryListEntry is the list-view projection: the full logs and
// combatants JSONB blobs are only fetched per entry via GET /combat/history/:id.
type CombatHistoryListEntry struct {
	ID        string          `json:"id"`
	PlayerID  string          `json:"player_id"`
	Winner    string          `json:"winner"`
	Summary   json.RawMessage `json:"summary"`
	EnemyName string          `json:"enemy_name"`
	EnemyType string          `json:"enemy_type"`
	Region    string          `json:"region"`
	LogsCount int             `json:"logs_count"`
	CreatedAt time.Time       `json:"created_at"`
}

// ListCombatHistory returns the player's last 20 combat history entries.
func ListCombatHistory(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(),
		`SELECT id, player_id, winner, summary, enemy_name, enemy_type, region,
		        COALESCE(jsonb_array_length(logs), 0), created_at
		 FROM combat_history WHERE player_id = $1 ORDER BY created_at DESC LIMIT 20`,
		authID,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch combat history"})
		return
	}
	defer rows.Close()

	entries := []CombatHistoryListEntry{}
	for rows.Next() {
		var e CombatHistoryListEntry
		if err := rows.Scan(&e.ID, &e.PlayerID, &e.Winner, &e.Summary, &e.EnemyName, &e.EnemyType, &e.Region, &e.LogsCount, &e.CreatedAt); err != nil {
			continue
		}
		entries = append(entries, e)
	}

	c.JSON(http.StatusOK, gin.H{"entries": entries})
}

// GetCombatHistory returns a single combat history entry by ID.
func GetCombatHistory(c *gin.Context) {
	authID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing combat history ID"})
		return
	}

	var e CombatHistoryEntry
	err := db.DB.QueryRowContext(c.Request.Context(),
		`SELECT id, player_id, winner, logs, summary, combatants, enemy_name, enemy_type, region, created_at
		 FROM combat_history WHERE id = $1 AND player_id = $2`,
		id, authID,
	).Scan(&e.ID, &e.PlayerID, &e.Winner, &e.Logs, &e.Summary, &e.Combatants, &e.EnemyName, &e.EnemyType, &e.Region, &e.CreatedAt)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Combat history not found"})
		return
	}

	c.JSON(http.StatusOK, e)
}
