package handlers

import (
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// applyPathMasteryGain adds Chân Ý points to a path. For mortal sources (sourceRank < 6),
// it respects the per-battle cap (gamefx.PathMasteryBattleGainLimit(rank)).
// For immortal sources (Tiên Cổ / Sát chiêu Tiên đạo, sourceRank >= 6), gains are uncapped.
func applyPathMasteryGain(player *models.Player, path string, gain int, sourceRank int) int {
	if player.PathMasteries == nil {
		player.PathMasteries = make(models.PathMasteriesMap)
	}
	if gain <= 0 {
		return 0
	}
	if player.ActiveCombat == nil || sourceRank >= 6 {
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
