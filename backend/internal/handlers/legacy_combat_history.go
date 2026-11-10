package handlers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

type CombatSummary struct {
	TotalDamageDealt float64 `json:"total_damage_dealt"`
	TotalDamageTaken float64 `json:"total_damage_taken"`
	ShieldsUsed      float64 `json:"shields_used"`
	FinalPlayerHP    float64 `json:"final_player_hp"`
	ItemsUsed        int     `json:"items_used"`
	StonesUsed       int64   `json:"stones_used"`
	LootCount        int     `json:"loot_count"`
}

// isBossCombatType reports whether a fight was against a boss. Boss fights are
// always recorded, win or lose, so the player and admins can review the run.
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
	attackers := uniquePlayerIDs(attackerIDs)
	defenders := uniquePlayerIDs(defenderIDs)
	defenderWinner := "Phe Ta"
	if attackerWinner == "Phe Ta" {
		defenderWinner = "Phe Địch"
	}

	if db.DB == nil || len(attackers)+len(defenders) == 0 {
		return
	}
	attackerIDStrings := make([]string, len(attackers))
	for i, playerID := range attackers {
		attackerIDStrings[i] = playerID.String()
	}
	defenderIDStrings := make([]string, len(defenders))
	for i, playerID := range defenders {
		defenderIDStrings[i] = playerID.String()
	}
	logsJSON, _ := json.Marshal(logs)
	summaryJSON, _ := json.Marshal(summary)
	attackerCombatantsJSON, _ := json.Marshal(attackerCombatants)
	defenderCombatantsJSON, _ := json.Marshal(defenderCombatants)
	_, err := db.DB.Exec(`
		INSERT INTO combat_history (player_id, winner, logs, summary, combatants, enemy_name, enemy_type, region)
		SELECT p.id, $5, $1::jsonb, $2::jsonb, $3::jsonb, $7, 'PVP', $9
		FROM unnest($10::uuid[]) AS p(id)
		UNION ALL
		SELECT p.id, $6, $1::jsonb, $2::jsonb, $4::jsonb, $8, 'PVP', $9
		FROM unnest($11::uuid[]) AS p(id)
	`, string(logsJSON), string(summaryJSON), string(attackerCombatantsJSON), string(defenderCombatantsJSON),
		attackerWinner, defenderWinner, attackerEnemyName, defenderEnemyName, region,
		pq.Array(attackerIDStrings), pq.Array(defenderIDStrings))
	if err == nil {
		return
	}

	fmt.Println("Error saving PVP combat history batch:", err)
	var pgErr *pq.Error
	if !errors.As(err, &pgErr) || pgErr.Code != "23503" {
		return
	}
	// Preserve per-player history when a participant disappeared before the
	// background write and caused the shared INSERT to fail its foreign key.
	for _, playerID := range attackers {
		SaveCombatHistory(playerID, attackerWinner, logs, summary, attackerCombatants, attackerEnemyName, "PVP", region)
	}
	for _, playerID := range defenders {
		SaveCombatHistory(playerID, defenderWinner, logs, summary, defenderCombatants, defenderEnemyName, "PVP", region)
	}
}

// saveSectWarDefenderHistory writes one perspective-correct replay row per
// defending player. A sect war is simulated on the attacker's request, so the
// defenders appear as enemies in that snapshot; sides and winner are reversed
// here. One batched INSERT keeps cost flat regardless of defender count.
func saveSectWarDefenderHistory(defenderIDs []uuid.UUID, attackerWinner string, logs []interface{}, summary CombatSummary, combatants []CombatantInfo, attackerName, region string) {
	defenders := uniquePlayerIDs(defenderIDs)
	if db.DB == nil || len(defenders) == 0 {
		return
	}
	defenderWinner := "Phe Ta"
	if attackerWinner == "Phe Ta" {
		defenderWinner = "Phe Địch"
	}

	defenderCombatants := combatantInfoInterfaces(combatants, true)
	logsJSON, _ := json.Marshal(logs)
	summaryJSON, _ := json.Marshal(summary)
	combatantsJSON, _ := json.Marshal(defenderCombatants)

	defenderIDStrings := make([]string, len(defenders))
	for i, playerID := range defenders {
		defenderIDStrings[i] = playerID.String()
	}

	_, err := db.DB.Exec(`
		INSERT INTO combat_history (player_id, winner, logs, summary, combatants, enemy_name, enemy_type, region)
		SELECT p.id, $5, $1::jsonb, $2::jsonb, $3::jsonb, $6, 'SECT_WAR', $7
		FROM unnest($4::uuid[]) AS p(id)
		JOIN players pl ON pl.id = p.id
	`, string(logsJSON), string(summaryJSON), string(combatantsJSON), pq.Array(defenderIDStrings),
		defenderWinner, attackerName, region)
	if err != nil {
		fmt.Println("Error saving sect war defender history:", err)
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
