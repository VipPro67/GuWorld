package simulator

import (
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

// daoPathForAction returns the Dao path used for damage interaction. Pathless
// combat actions follow the established Strength fallback.
func daoPathForAction(paths []string) string {
	if len(paths) == 0 {
		return "strength"
	}
	path := gamefx.CleanPath(paths[0])
	if path == "" {
		return "strength"
	}
	return path
}

// targetDaoFactors returns the target's same-path resonance and natural Dao
// resistance for an incoming hit path.
func targetDaoFactors(target *CombatantAuto, path string) (resistance, resonance float64) {
	if target == nil {
		return 0, 1
	}
	ensureDaoProfile(target)
	path = daoPathForAction([]string{path})
	return target.DaoProfile.GetDefensiveResistance(path), target.DaoProfile.GetTargetDaoResonance(path)
}

func applyForeignDaoWound(target *CombatantAuto) {
	if target == nil || target.HP <= 0 {
		return
	}
	for i := range target.ActiveEffects {
		if target.ActiveEffects[i].ID == statusfx.StatusForeignDaoWound {
			target.ActiveEffects[i].Duration = 2
			ComputeEffectMultipliers(target)
			return
		}
	}
	target.ActiveEffects = append(target.ActiveEffects, Effect{ID: statusfx.StatusForeignDaoWound, Type: "debuff", Value: 30, Duration: 2})
	ComputeEffectMultipliers(target)
}

func effectiveHealingReceived(c *CombatantAuto) float64 {
	if c == nil {
		return 1
	}
	mult := 1 + c.HealingReceived - c.HealingReceivedPenalty
	if mult < 0 {
		return 0
	}
	return mult
}
