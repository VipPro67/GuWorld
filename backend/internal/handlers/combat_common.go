package handlers

import "github.com/vippro67/GuWorld/backend/internal/models"

type CombatContext struct {
	Attacker    *models.Combatant
	Defender    *models.Combatant
	DamageDealt float64
	ShieldAdded float64
	DamageBoost float64
	Logs        []string
}

// effectiveCombatSpeed returns a combatant's speed after slow-type debuffs.
// Used to order actions and to decide who opens an interleaved exchange.
func effectiveCombatSpeed(combatState *models.CombatState, id string) float64 {
	ref, _, _ := findCombatant(combatState, id)
	if ref == nil {
		return 0
	}
	speed := ref.Speed
	for _, eff := range ref.StatusEffects {
		if (eff.Type == "slow" || eff.Type == "slow_damage" || eff.Type == "slow1") && eff.Turns > 0 {
			speed *= (1.0 - eff.Value/100.0)
		}
	}
	return speed
}
