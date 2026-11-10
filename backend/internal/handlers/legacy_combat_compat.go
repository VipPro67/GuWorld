package handlers

import (
	"database/sql"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func markBossCleared(tx *sql.Tx, player *models.Player) {
	if player == nil || player.ActiveCombat == nil {
		return
	}
	typeName := strings.ToUpper(player.ActiveCombat.EnemyType)
	if typeName != "BOSS" && typeName != "WORLDBOSS" && typeName != "WORLD BOSS" {
		return
	}
	region, x, y := player.CurrentRegion, player.X, player.Y
	if player.ActiveCombat.WorldBossCellRegionID != "" {
		region, x, y = player.ActiveCombat.WorldBossCellRegionID, player.ActiveCombat.WorldBossCellX, player.ActiveCombat.WorldBossCellY
	}
	_ = MarkPersistentCellCleared(tx, player.ID, region, x, y)
}

func shouldRecordCombatHistory(enemyType, winner string, immortalGuDropped bool) bool {
	typeName := strings.ToLower(strings.TrimSpace(enemyType))
	return isBossCombatType(typeName) || typeName == "pvp" || typeName == "sect_war" || winner == "Phe Địch" || immortalGuDropped
}

func isBossCombatType(enemyType string) bool {
	switch strings.ToLower(strings.TrimSpace(enemyType)) {
	case "boss", "world_boss", "tribulation_boss":
		return true
	default:
		return false
	}
}

func filterCombatGuInventory(inv models.GuInventory, selectedGuIDs []string, settings models.AutoCombatConfig) models.GuInventory {
	if len(inv) == 0 {
		return inv
	}
	selected := selectedGuIDs
	if len(selected) == 0 {
		for _, tactic := range settings.GuPriorities {
			selected = append(selected, tactic.GuID)
		}
	}
	allowed := make(map[string]bool, len(selected))
	for _, id := range selected {
		allowed[id] = true
	}
	var result models.GuInventory
	for _, gu := range inv {
		if !gu.Equipped || gu.Status != "ACTIVE" {
			continue
		}
		if len(allowed) == 0 || allowed[gu.InstanceID] || allowed[gu.GuID] {
			result = append(result, gu)
		}
	}
	return result
}
