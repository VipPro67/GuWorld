package handlers

import (
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func playerHasBattleStartMechanic(player *models.Player, code string) bool {
	if player == nil {
		return false
	}
	for _, gu := range player.GuInventory {
		if !gu.Equipped || gu.Status != "ACTIVE" || gu.Hunger <= 0 {
			continue
		}
		spec, ok := models.GetGu(gu.GuID, gu.Rank)
		if !ok {
			spec, ok = models.GetGuAny(gu.GuID)
		}
		if !ok {
			continue
		}
		for _, mechanic := range models.CompiledGuEffects(spec).NativeMechanics() {
			if mechanic.Trigger == gameplayeffects.TriggerOnBattleStart && mechanic.Code == code {
				return true
			}
		}
	}
	return false
}

func maybeCleanseRegionalWall(player *models.Player) {
	if player != nil && playerHasBattleStartMechanic(player, "CLEANSE_REGIONAL_WALL") {
		player.Statuses.Remove("regional_wall_suppression")
	}
}
