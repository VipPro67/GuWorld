package handlers

import (
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// statusStatFactor returns the multiplicative factor the player's active
// statuses apply to one stat ("attack"/"defense"/...): the product of
// (1 + stat_mod) over every active status. Used to scale Gu skill / killer
// move action output so suppression (e.g. Áp Chế Giới Vực −93% tấn công)
// visibly weakens skills instead of being diluted inside the additive
// (DamageFlat + attack) skill formula. Clamped at 0 (never negative).
func statusStatFactor(player *models.Player, stat string) float64 {
	if player == nil || len(player.Statuses) == 0 {
		return 1
	}
	factor := 1.0
	for _, st := range player.Statuses.Active() {
		for _, def := range gamefx.AllStatuses() {
			if def.Key != st.ID {
				continue
			}
			if pct := def.EffectsForValue(st.Value).StatMods[stat]; pct != 0 {
				factor *= (1.0 + pct)
			}
			break
		}
	}
	if factor < 0 {
		return 0
	}
	return factor
}
