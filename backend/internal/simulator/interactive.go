package simulator

import (
	"errors"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/gamefx"
)

func combatActionAvailable(actor *CombatantAuto, action *SimAction, target *CombatantAuto) bool {
	if action == nil || actor == nil || !checkActionCondition(action, actor, target) {
		return false
	}
	if (actor.IsSilenced || actor.IsFeared) && action.ActionType != "BASIC_ATTACK" {
		return false
	}
	if actor.HasGuSeal && (action.ActionType == "GU_SKILL" || action.ActionType == "KILLER_MOVE") {
		return false
	}
	if (actor.Kind == "Player" || actor.Kind == "VariantNPC") && actor.Mana < action.ManaCost*gamefx.ManaTaxMultiplier(actionConflictMult(actor, action.Path)) {
		return false
	}
	if action.ActionType == "GU_SKILL" && actor.GuHunger != nil {
		key := action.InstanceID
		if key == "" {
			key = action.SourceID
		}
		if hunger, present := actor.GuHunger[key]; present && hunger <= 0 {
			return false
		}
	}
	return true
}

// InteractiveActionTargetIDs returns the alive targets that the client may
// select for an action, using the same formation and stealth rules as command
// validation.
func InteractiveActionTargetIDs(actor *CombatantAuto, enemies []*CombatantAuto, action *SimAction) []string {
	if actor == nil || action == nil {
		return nil
	}
	targets := enemies
	if strings.EqualFold(action.TargetType, "SELF") {
		targets = []*CombatantAuto{actor}
	}
	result := make([]string, 0, len(targets))
	for _, target := range targets {
		if CanSelectCombatTarget(actor, enemies, action, target) {
			result = append(result, target.ID)
		}
	}
	return result
}

// CanSelectCombatTarget is shared by the interactive action list and command
// validation so the client never offers a target rejected by formation rules.
func CanSelectCombatTarget(actor *CombatantAuto, enemies []*CombatantAuto, action *SimAction, target *CombatantAuto) bool {
	if actor == nil || action == nil || target == nil {
		return false
	}
	if strings.EqualFold(action.TargetType, "SELF") {
		return target.ID == actor.ID
	}

	isAliveEnemy := false
	for _, enemy := range enemies {
		if enemy != nil && enemy.ID == target.ID && enemy.HP > 0 && !enemy.IsDead {
			isAliveEnemy = true
			break
		}
	}
	if !isAliveEnemy {
		return false
	}
	if action.TargetType == "" || strings.EqualFold(action.TargetType, "TARGET") || strings.EqualFold(action.TargetType, "SINGLE") {
		copyAction := *action
		copyAction.TargetX, copyAction.TargetY = target.GridX, target.GridY
		resolved := resolveTargets(enemies, &copyAction)
		return len(resolved) > 0 && resolved[0].ID == target.ID
	}
	return true
}

func selectCombatCommand(actor *CombatantAuto, enemies []*CombatantAuto, command CombatCommand) (*SimAction, *CombatantAuto, error) {
	if command.ActionType == "FLEE" && command.SourceID == "" && command.InstanceID == "" {
		return &SimAction{ActionType: "FLEE", Name: "Bỏ chạy"}, nil, nil
	}
	var selected *SimAction
	for i := range actor.Actions {
		action := &actor.Actions[i]
		if action.ActionType != command.ActionType || action.SourceID != command.SourceID || action.InstanceID != command.InstanceID {
			continue
		}
		selected = action
		break
	}
	if selected == nil && command.ActionType == "BASIC_ATTACK" && command.SourceID == "" && command.InstanceID == "" {
		selected = &SimAction{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: actor.BaseDmg, DamageMult: 1, TargetType: "SINGLE", Path: []string{"strength"}}
	}
	if selected == nil {
		return nil, nil, errors.New("action is not in the prepared loadout")
	}
	var target *CombatantAuto
	if strings.EqualFold(selected.TargetType, "SELF") {
		if command.TargetID != "" && command.TargetID != actor.ID {
			return nil, nil, errors.New("self action must target the acting combatant")
		}
		target = actor
	} else if command.TargetID != "" {
		for _, candidate := range enemies {
			if candidate.ID == command.TargetID && candidate.HP > 0 && !candidate.IsDead {
				target = candidate
				break
			}
		}
		if target == nil {
			return nil, nil, errors.New("target is not alive or is not on the target side")
		}
	}
	if target == nil {
		target = getTarget(enemies)
	}
	if !combatActionAvailable(actor, selected, target) {
		return nil, nil, errors.New("action cannot be used in the current combat state")
	}
	if target != nil && !CanSelectCombatTarget(actor, enemies, selected, target) {
		return nil, nil, errors.New("target is protected by formation or stealth")
	}
	return selected, target, nil
}

// FleeChance uses current effective speed and is deterministic when rolled
// through the simulator's battle-seeded random source.
func FleeChance(actor *CombatantAuto, enemies []*CombatantAuto) float64 {
	if actor == nil {
		return 0
	}
	actorSpeed := actor.EffectiveSpeed
	if actorSpeed <= 0 {
		actorSpeed = actor.Speed
	}
	fastestEnemy := 0.0
	for _, enemy := range enemies {
		if enemy == nil || enemy.HP <= 0 || enemy.IsDead {
			continue
		}
		speed := enemy.EffectiveSpeed
		if speed <= 0 {
			speed = enemy.Speed
		}
		if speed > fastestEnemy {
			fastestEnemy = speed
		}
	}
	chance := 0.5 + (actorSpeed-fastestEnemy)*0.02
	return max(0.15, min(0.85, chance))
}

func chooseAutoCombatAction(actor *CombatantAuto, enemies []*CombatantAuto) (*SimAction, *CombatantAuto) {
	target := getTarget(enemies)
	profile := strings.ToLower(strings.TrimSpace(actor.AIProfile))
	if profile == "boss_burst" || profile == "boss_aggressive" || profile == "beast_aggressive" || profile == "beast_elite" || profile == "defensive" || profile == "boss_balanced" {
		var best *SimAction
		bestScore := -1.0
		lowHealth := actor.MaxHP > 0 && actor.HP/actor.MaxHP <= 0.4
		for i := range actor.Actions {
			action := &actor.Actions[i]
			checkTarget := target
			if strings.EqualFold(action.TargetType, "SELF") {
				checkTarget = actor
			}
			if !combatActionAvailable(actor, action, checkTarget) {
				continue
			}
			damageScore := action.Damage * action.DamageMult
			if action.DamageHPPercent > 0 && target != nil {
				damageScore += target.MaxHP * action.DamageHPPercent
			}
			shieldValue := action.ShieldMult
			if shieldValue <= 0 {
				shieldValue = 1
			}
			supportScore := action.Healing + action.Defense*shieldValue
			score := 0.0
			switch profile {
			case "boss_burst", "boss_aggressive", "beast_aggressive", "beast_elite":
				score = damageScore
			case "defensive":
				if lowHealth {
					score = supportScore
				} else {
					score = damageScore
				}
			case "boss_balanced":
				if lowHealth {
					score = supportScore
				} else {
					score = damageScore
				}
			}
			if score > bestScore {
				best, bestScore = action, score
			}
		}
		if best != nil {
			if strings.EqualFold(best.TargetType, "SELF") {
				return best, actor
			}
			return best, target
		}
	}
	lastIndex := -1
	for i := range actor.Actions {
		if autoActionKey(&actor.Actions[i]) == actor.LastAutoActionKey {
			lastIndex = i
			break
		}
	}
	for offset := 1; offset <= len(actor.Actions); offset++ {
		index := offset - 1
		if lastIndex >= 0 {
			index = (lastIndex + offset) % len(actor.Actions)
		}
		action := &actor.Actions[index]
		checkTarget := target
		if strings.EqualFold(action.TargetType, "SELF") {
			checkTarget = actor
		}
		if combatActionAvailable(actor, action, checkTarget) {
			return rememberAutoAction(actor, action, target)
		}
	}
	basic := &SimAction{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: actor.BaseDmg, DamageMult: 1, TargetType: "SINGLE", Path: []string{"strength"}}
	return rememberAutoAction(actor, basic, target)
}

func autoActionKey(action *SimAction) string {
	if action == nil {
		return ""
	}
	return action.ActionType + "|" + action.SourceID + "|" + action.InstanceID + "|" + action.Name
}

func rememberAutoAction(actor *CombatantAuto, action *SimAction, target *CombatantAuto) (*SimAction, *CombatantAuto) {
	actor.LastAutoActionKey = autoActionKey(action)
	if strings.EqualFold(action.TargetType, "SELF") {
		return action, actor
	}
	return action, target
}
