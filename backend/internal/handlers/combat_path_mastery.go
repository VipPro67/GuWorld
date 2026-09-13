package handlers

import (
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// applyPathMasteryGain adds Chân Ý points to a path while respecting the
// per-battle cap (gamefx.PathMasteryBattleGainLimit(rank) * battle). The amount
// earned in the current battle is tracked on the active combat state; once the
// cap is reached further gains are suppressed. Returns the points actually
// granted (may be less than gain, or 0 when capped).
func applyPathMasteryGain(player *models.Player, path string, gain int) int {
	if player.PathMasteries == nil {
		player.PathMasteries = make(models.PathMasteriesMap)
	}
	if gain <= 0 {
		return 0
	}
	if player.ActiveCombat == nil {
		player.PathMasteries[path] += gain
		return gain
	}
	capLimit := gamefx.PathMasteryBattleGainLimit(player.Rank)
	remaining := capLimit - player.ActiveCombat.PathMasteryGained
	if remaining <= 0 {
		return 0
	}
	if gain > remaining {
		gain = remaining
	}
	player.ActiveCombat.PathMasteryGained += gain
	player.PathMasteries[path] += gain
	return gain
}
