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
