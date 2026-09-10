package handlers

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func BuildEnemyActions(ctx *CombatContext, enemy *models.Combatant) []Action {
	var potentialActions []Action
	for _, eg := range enemy.GuInventory {
		if eg.Status == "ACTIVE" {
			spec, ok := models.GetGu(eg.GuID, eg.Rank)
			if ok {
				action, err := BuildCombatAction(ctx, spec)
				if err == nil {
					action.GuID = eg.GuID
					if enemy.CombatSettings != nil {
						for _, t := range enemy.CombatSettings.GuPriorities {
							if t.GuID == eg.GuID {
								action.Priority = t.Priority
								action.UseCondition = t.UseCondition
								action.ConditionValue = t.ConditionValue
								break
							}
						}
					}
					potentialActions = append(potentialActions, action)
				}
			}
		}
	}

	virtualMana := enemy.CurrentMana
	if enemy.Kind != "Player" && enemy.Kind != "Ally" && !strings.Contains(enemy.Kind, "PVP") {
		virtualMana = 99999.0
	}

	var selected []Action
	if enemy.Kind == "Player" || strings.Contains(enemy.Kind, "PVP") || enemy.CombatSettings != nil {
		selected = SelectActionsByPlayerTactics(ctx, enemy, potentialActions, virtualMana)
	} else {
		selected = SelectActionsByAI(enemy.AIProfile, potentialActions, virtualMana)
	}

	if len(selected) == 0 {
		return BuildBasicEnemyAction(enemy)
	}

	return selected
}

func getActionCategoryIndex(act Action, priorities []string) int {
	cat := "other"
	if act.Def > 0 && act.Dmg == 0 {
		cat = "shield"
	} else if act.IsUtility && act.Dmg == 0 {
		cat = "heal"
	} else if act.Dmg > 0 {
		cat = "damage"
	}
	for idx, p := range priorities {
		if strings.ToLower(p) == cat {
			return idx
		}
	}
	return len(priorities)
}

func SelectActionsByPlayerTactics(ctx *CombatContext, enemy *models.Combatant, potentialActions []Action, currentMana float64) []Action {
	if len(potentialActions) == 0 {
		return nil
	}

	cs := enemy.CombatSettings
	actions := make([]Action, len(potentialActions))
	copy(actions, potentialActions)

	// Sort actions by Priority (lower number = higher priority).
	sort.SliceStable(actions, func(i, j int) bool {
		pI := actions[i].Priority
		pJ := actions[j].Priority
		if pI == 0 {
			pI = 500
		}
		if pJ == 0 {
			pJ = 500
		}
		if pI != pJ {
			return pI < pJ
		}
		// Tie-break with CategoryPriorities if set
		if cs != nil && len(cs.CategoryPriorities) > 0 {
			catI := getActionCategoryIndex(actions[i], cs.CategoryPriorities)
			catJ := getActionCategoryIndex(actions[j], cs.CategoryPriorities)
			if catI != catJ {
				return catI < catJ
			}
		}
		return false
	})

	var selected []Action
	remainingMana := currentMana

	// Check if an action's condition is met
	checkCondition := func(act *Action) bool {
		if act.UseCondition == "" || act.UseCondition == "always" {
			return true
		}
		condVal := act.ConditionValue
		if condVal <= 0 {
			condVal = 30.0
		}
		switch act.UseCondition {
		case "hp_below":
			if enemy.MaxHP <= 0 {
				return false
			}
			return (enemy.HP / enemy.MaxHP * 100.0) <= condVal
		case "mana_above":
			if enemy.MaxMana <= 0 {
				return false
			}
			return (enemy.CurrentMana / enemy.MaxMana * 100.0) >= condVal
		case "enemy_hp_below":
			if ctx != nil && ctx.Defender != nil && ctx.Defender.MaxHP > 0 {
				return (ctx.Defender.HP / ctx.Defender.MaxHP * 100.0) <= condVal
			}
			return false
		}
		return true
	}

	// 1. Support/Defense pass: if the highest priority support/heal meets condition and has mana, pick it
	for _, act := range actions {
		isSupport := act.Def > 0 || act.IsUtility
		if isSupport && checkCondition(&act) && act.ManaCost <= remainingMana {
			selected = append(selected, act)
			remainingMana -= act.ManaCost
			break // Only 1 support action per turn
		}
	}

	// 2. Attack pass: pick highest priority damaging action that meets condition and has mana
	for _, act := range actions {
		alreadyPicked := false
		for _, s := range selected {
			if s.Name == act.Name {
				alreadyPicked = true
				break
			}
		}
		if alreadyPicked {
			continue
		}

		if act.Dmg > 0 && checkCondition(&act) && act.ManaCost <= remainingMana {
			selected = append(selected, act)
			remainingMana -= act.ManaCost
			break
		}
	}

	// Fallback: If no action was selected, pick the first affordable action from the priority list whose condition is "always" or general attack
	if len(selected) == 0 {
		for _, act := range actions {
			if (act.UseCondition == "" || act.UseCondition == "always") && act.ManaCost <= remainingMana {
				selected = append(selected, act)
				remainingMana -= act.ManaCost
				break
			}
		}
	}

	return selected
}

func BuildBasicEnemyAction(enemy *models.Combatant) []Action {
	dmg := enemy.BaseAttack
	if dmg <= 0 {
		dmg = 5.0
	}
	name := "Đấm Tay Không (Basic Attack)"
	if enemy.Kind != "Player" {
		name = fmt.Sprintf("%s tấn công (Basic Attack)", enemy.Name)
	}
	return []Action{
		{
			Name:     name,
			Dmg:      dmg,
			ManaCost: 0,
			Logs:     []string{fmt.Sprintf("-> %s tung đòn đánh thường!", enemy.Name)},
		},
	}
}

func SelectActionsByAI(profile string, potentialActions []Action, currentMana float64) []Action {
	if len(potentialActions) == 0 {
		return nil
	}

	var selected []Action
	remainingMana := currentMana

	switch profile {
	case "boss_burst":
		actions := make([]Action, len(potentialActions))
		copy(actions, potentialActions)
		for i := 0; i < len(actions)-1; i++ {
			for j := i + 1; j < len(actions); j++ {
				if actions[i].Dmg < actions[j].Dmg {
					actions[i], actions[j] = actions[j], actions[i]
				}
			}
		}
		count := 0
		for _, act := range actions {
			if count >= 2 {
				break
			}
			if act.ManaCost <= remainingMana {
				selected = append(selected, act)
				remainingMana -= act.ManaCost
				count++
			}
		}

	case "defensive":
		actions := make([]Action, len(potentialActions))
		copy(actions, potentialActions)
		for i := 0; i < len(actions)-1; i++ {
			for j := i + 1; j < len(actions); j++ {
				if actions[i].Def < actions[j].Def {
					actions[i], actions[j] = actions[j], actions[i]
				}
			}
		}
		count := 0
		for _, act := range actions {
			if count >= 2 {
				break
			}
			if act.ManaCost <= remainingMana {
				selected = append(selected, act)
				remainingMana -= act.ManaCost
				count++
			}
		}

	case "boss_balanced":
		var attackActions []Action
		var defenseActions []Action
		for _, act := range potentialActions {
			if act.Dmg > 0 {
				attackActions = append(attackActions, act)
			}
			if act.Def > 0 {
				defenseActions = append(defenseActions, act)
			}
		}

		for i := 0; i < len(attackActions)-1; i++ {
			for j := i + 1; j < len(attackActions); j++ {
				if attackActions[i].Dmg < attackActions[j].Dmg {
					attackActions[i], attackActions[j] = attackActions[j], attackActions[i]
				}
			}
		}
		for i := 0; i < len(defenseActions)-1; i++ {
			for j := i + 1; j < len(defenseActions); j++ {
				if defenseActions[i].Def < defenseActions[j].Def {
					defenseActions[i], defenseActions[j] = defenseActions[j], defenseActions[i]
				}
			}
		}

		if len(defenseActions) > 0 && defenseActions[0].ManaCost <= remainingMana {
			selected = append(selected, defenseActions[0])
			remainingMana -= defenseActions[0].ManaCost
		}
		if len(attackActions) > 0 && attackActions[0].ManaCost <= remainingMana {
			alreadyPicked := false
			for _, s := range selected {
				if s.Name == attackActions[0].Name {
					alreadyPicked = true
				}
			}
			if !alreadyPicked {
				selected = append(selected, attackActions[0])
				remainingMana -= attackActions[0].ManaCost
			}
		}
		if len(selected) == 0 {
			for _, act := range potentialActions {
				if act.ManaCost <= remainingMana {
					selected = append(selected, act)
					break
				}
			}
		}

	default:
		for _, act := range potentialActions {
			if act.ManaCost <= remainingMana {
				selected = append(selected, act)
				remainingMana -= act.ManaCost
				break
			}
		}
	}

	return selected
}
