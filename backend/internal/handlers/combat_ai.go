package handlers

import (
	"fmt"
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
					potentialActions = append(potentialActions, action)
				}
			}
		}
	}

	virtualMana := enemy.CurrentMana
	if enemy.Kind != "Player" && enemy.Kind != "Ally" && !strings.Contains(enemy.Kind, "PVP") {
		virtualMana = 99999.0
	}
	selected := SelectActionsByAI(enemy.AIProfile, potentialActions, virtualMana)

	if len(selected) == 0 {
		return BuildBasicEnemyAction(enemy)
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
