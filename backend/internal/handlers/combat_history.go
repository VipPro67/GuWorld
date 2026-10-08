package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
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

type CombatSummary struct {
	TotalDamageDealt float64 `json:"total_damage_dealt"`
	TotalDamageTaken float64 `json:"total_damage_taken"`
	ShieldsUsed      float64 `json:"shields_used"`
	FinalPlayerHP    float64 `json:"final_player_hp"`
	ItemsUsed        int     `json:"items_used"`
	StonesUsed       int64   `json:"stones_used"`
	LootCount        int     `json:"loot_count"`
}

// shouldRecordCombatHistory reports whether a finished fight is persisted to
// combat_history: every PVP match counts, PVE only when the player side lost.
func shouldRecordCombatHistory(enemyType, winner string) bool {
	return enemyType == "PVP" || winner == "Phe Địch"
}

// SaveCombatHistory persists a completed combat result to the combat_history table.
func SaveCombatHistory(playerID uuid.UUID, winner string, logs []interface{}, summary CombatSummary, combatants []interface{}, enemyName, enemyType, region string) {
	if db.DB == nil {
		return
	}

	logsJSON, _ := json.Marshal(logs)
	summaryJSON, _ := json.Marshal(summary)
	combatantsJSON, _ := json.Marshal(combatants)

	_, err := db.DB.Exec(`
		INSERT INTO combat_history (player_id, winner, logs, summary, combatants, enemy_name, enemy_type, region)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, playerID, winner, logsJSON, summaryJSON, combatantsJSON, enemyName, enemyType, region)
	if err != nil {
		fmt.Println("Error saving combat history:", err)
	}
}

// savePVPCombatHistory writes a perspective-correct copy for every player in
// each participating team. NPCs, pets, and other non-player combatants remain
// in the replay payload but do not receive player history rows.
func savePVPCombatHistory(attackerIDs, defenderIDs []uuid.UUID, attackerWinner string, logs []interface{}, summary CombatSummary, combatants []CombatantInfo, attackerEnemyName, defenderEnemyName, region string) {
	attackerCombatants := combatantInfoInterfaces(combatants, false)
	defenderCombatants := combatantInfoInterfaces(combatants, true)
	defenderWinner := "Phe Ta"
	if attackerWinner == "Phe Ta" {
		defenderWinner = "Phe Địch"
	}

	for _, playerID := range uniquePlayerIDs(attackerIDs) {
		SaveCombatHistory(playerID, attackerWinner, logs, summary, attackerCombatants, attackerEnemyName, "PVP", region)
	}
	for _, playerID := range uniquePlayerIDs(defenderIDs) {
		SaveCombatHistory(playerID, defenderWinner, logs, summary, defenderCombatants, defenderEnemyName, "PVP", region)
	}
}

func combatantInfoInterfaces(combatants []CombatantInfo, reverseSides bool) []interface{} {
	result := make([]interface{}, len(combatants))
	for i, combatant := range combatants {
		if reverseSides {
			if combatant.Kind == "ally" {
				combatant.Kind = "enemy"
			} else if combatant.Kind == "enemy" {
				combatant.Kind = "ally"
			}
		}
		result[i] = combatant
	}
	return result
}

func uniquePlayerIDs(playerIDs []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(playerIDs))
	unique := make([]uuid.UUID, 0, len(playerIDs))
	for _, playerID := range playerIDs {
		if playerID == uuid.Nil {
			continue
		}
		if _, exists := seen[playerID]; exists {
			continue
		}
		seen[playerID] = struct{}{}
		unique = append(unique, playerID)
	}
	return unique
}

func pvpHistoryPlayerIDs(attackerID uuid.UUID, simAllies []*simulator.CombatantAuto, defenderID string) (attackerIDs, defenderIDs []uuid.UUID) {
	attackerIDs = append(attackerIDs, attackerID)
	for _, ally := range simAllies {
		if ally == nil || (ally.Kind != "Player" && ally.Kind != "Ally") {
			continue
		}
		if id, err := uuid.Parse(ally.ID); err == nil {
			attackerIDs = append(attackerIDs, id)
		}
	}
	if id, err := uuid.Parse(defenderID); err == nil {
		defenderIDs = append(defenderIDs, id)
	}
	return uniquePlayerIDs(attackerIDs), uniquePlayerIDs(defenderIDs)
}

func toInterfaceSlice(logs []simulator.CombatLog) []interface{} {
	s := make([]interface{}, len(logs))
	for i, l := range logs {
		s[i] = l
	}
	return s
}

func buildCombatSummaryFromLogs(logs []simulator.CombatLog, finalHP float64, itemsUsed int, stonesSpent int64) CombatSummary {
	var totalDmgDealt, totalDmgTaken, shieldsUsed float64
	for _, log := range logs {
		if log.Damage > 0 {
			if log.TargetId != "" && log.Action != "SHIELD_ABSORB" {
				// Damage to non-player = dealt, to player = taken
				totalDmgDealt += log.Damage
			} else {
				totalDmgTaken += log.Damage
			}
		}
		if log.Action == "SHIELD_GAIN" {
			shieldsUsed += log.Damage
		}
		if log.Action == "SHIELD_ABSORB" {
			shieldsUsed += log.Damage
		}
	}
	return CombatSummary{
		TotalDamageDealt: totalDmgDealt,
		TotalDamageTaken: totalDmgTaken,
		ShieldsUsed:      shieldsUsed,
		FinalPlayerHP:    finalHP,
		ItemsUsed:        itemsUsed,
		StonesUsed:       stonesSpent,
	}
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
