// Package handlers implements HTTP handlers and gameplay logic for GuWorld.
// This file consolidates the Combat Lifecycle domain: session execution,
// state transitions, speed/damage calculation helpers, and authoritative settlement.
package handlers

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

// requiresRegionCombatPolicy determines whether an encounter requires standard
// world region combat verification (e.g., forbidding combat in safe zones).
// Virtual practice and Infinite Tower bypass this requirement.
func requiresRegionCombatPolicy(combat *models.CombatState) bool {
	if combat == nil {
		return true
	}
	typeName := strings.ToUpper(strings.TrimSpace(combat.EnemyType))
	return typeName != "PRACTICE" && typeName != "TOWER"
}

// CombatContext tracks combatants and battle calculations for an action exchange.
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

// applyDamageWithLogs calculates virtual shield absorption and damage reduction for
// legacy damage evaluation, returning the descriptive combat log.
func applyDamageWithLogs(initiatorName, targetName string, target *models.Combatant, damage float64) string {
	if (target.Kind == "Player" || target.Kind == "Ally") && target.Rank >= 6 {
		apertureDmg := damage * 0.5
		target.ApertureDmg += apertureDmg
		damage = damage / 200.0
	}

	if target.VirtualShield > 0 {
		if target.VirtualShield >= damage {
			target.VirtualShield -= damage
			return fmt.Sprintf("-> %s gây %.1f sát thương lên %s (Bị hấp thụ hoàn toàn bởi Khiên ảo, còn %.1f Khiên).",
				initiatorName, damage, targetName, target.VirtualShield)
		} else {
			shieldDmg := target.VirtualShield
			hpDmg := damage - shieldDmg
			target.VirtualShield = 0
			target.HP -= hpDmg
			if target.HP < 0 {
				target.HP = 0
			}
			return fmt.Sprintf("-> %s gây %.1f sát thương lên %s (Lớp Khiên ảo bị đánh vỡ, %.1f sát thương trừ thẳng vào HP, còn %.1f HP).",
				initiatorName, damage, targetName, hpDmg, target.HP)
		}
	} else {
		target.HP -= damage
		if target.HP < 0 {
			target.HP = 0
		}
		return fmt.Sprintf("-> %s gây %.1f sát thương lên %s (%.1f sát thương trừ thẳng vào HP, còn %.1f HP).",
			initiatorName, damage, targetName, damage, target.HP)
	}
}

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

// SettleCombatSession settles an already-completed interactive combat session using
// its terminal state and logs, bypassing redundant re-simulation while preserving
// deterministic settlement (consumption, loot, defeat, history, notifications).
func SettleCombatSession(tx *sql.Tx, player *models.Player, loadout *simulator.CombatLoadout, initialState *simulator.CombatState, terminalState *simulator.CombatState, terminalLogs []simulator.CombatLog) (*CombatResolution, error) {
	var guIDs []string
	if loadout != nil {
		guIDs = loadout.GuIds
	}
	return resolveCombatWithTerminalState(tx, player, loadout, guIDs, initialState, terminalState, terminalLogs)
}
