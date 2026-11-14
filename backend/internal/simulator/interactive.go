package simulator

import (
	"errors"
	"sort"
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
	if (action.ActionType == "KILLER_MOVE" || action.IsFromGuHouse) && actor.GuHunger != nil && len(action.ComponentGuIDs) > 0 {
		cost := action.HungerCost
		if cost <= 0 {
			cost = 1
		}
		for _, guID := range action.ComponentGuIDs {
			if hunger, present := actor.GuHunger[guID]; present && hunger < cost {
				return false
			}
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
	if command.ActionType == "EXIT_GU_HOUSE" {
		if actor == nil || actor.GuHouse == nil || !actor.GuHouse.Active {
			return nil, nil, errors.New("cannot exit: not inside an active Gu House")
		}
		return &SimAction{ActionType: "EXIT_GU_HOUSE", Name: "Rời Cổ Phòng", TargetType: "SELF", IsFromGuHouse: true}, actor, nil
	}
	if command.ActionType == "ENTER_GU_HOUSE" {
		if actor == nil || actor.GuHouse == nil || actor.GuHouse.IsDestroyed || actor.GuHouse.Active {
			return nil, nil, errors.New("cannot enter: Gu House is unavailable or already active")
		}
		if actor.IsStunned || actor.HasGuSeal {
			return nil, nil, errors.New("cannot enter: stunned or Gu sealed")
		}
		return &SimAction{ActionType: "ENTER_GU_HOUSE", Name: "Dung Nhập Cổ Phòng", TargetType: "SELF"}, actor, nil
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

type selectedCombatAction struct {
	action *SimAction
	target *CombatantAuto
}

func combatantPaysMana(actor *CombatantAuto) bool {
	return actor != nil && (actor.Kind == "Player" || actor.Kind == "VariantNPC")
}

func projectedCombatManaCost(actor *CombatantAuto, action *SimAction) float64 {
	if action == nil {
		return 0
	}
	cost := action.ManaCost * gamefx.ManaTaxMultiplier(actionConflictMult(actor, action.Path))
	if actor != nil && actor.HasThoughtChaos && (action.ActionType == "GU_SKILL" || action.ActionType == "KILLER_MOVE") {
		cost *= 2
	}
	return cost
}

// SelectCombatPlan validates the complete submitted plan against the current
// snapshot. Resources are checked as a total before the simulator executes any
// part of the plan; execution still rechecks each action after prior effects.
func selectCombatPlan(actor *CombatantAuto, enemies []*CombatantAuto, plan []CombatActionSelection, maxGuActions int) ([]selectedCombatAction, error) {
	if actor == nil || len(plan) == 0 {
		return nil, errors.New("combat plan is empty")
	}
	if maxGuActions < 1 {
		maxGuActions = 3
	}
	guActions, basicActions, manaCost := 0, 0, 0.0
	usedInstances := make(map[string]bool)
	selected := make([]selectedCombatAction, 0, len(plan))
	for index, choice := range plan {
		if choice.ActionType == "FLEE" {
			if len(plan) != 1 {
				return nil, errors.New("flee must be the only action in a plan")
			}
			action, target, err := selectCombatCommand(actor, enemies, CombatCommand{ActionType: "FLEE"})
			if err != nil {
				return nil, err
			}
			selected = append(selected, selectedCombatAction{action, target})
			continue
		}
		if choice.ActionType == "EXIT_GU_HOUSE" || choice.ActionType == "ENTER_GU_HOUSE" {
			action, target, err := selectCombatCommand(actor, enemies, CombatCommand{ActionType: choice.ActionType})
			if err != nil {
				return nil, err
			}
			selected = append(selected, selectedCombatAction{action, target})
			continue
		}
		command := CombatCommand{ActionType: choice.ActionType, SourceID: choice.SourceID, InstanceID: choice.InstanceID, TargetID: choice.TargetID}
		action, target, err := selectCombatCommand(actor, enemies, command)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(action.TargetType, "SELF") && choice.TargetID == "" {
			return nil, errors.New("each targeted action must lock its own target")
		}
		if !strings.EqualFold(action.TargetType, "SELF") && target == nil {
			return nil, errors.New("combat plan has no living target for a targeted action")
		}
		if action.ActionType == "BASIC_ATTACK" {
			basicActions++
			if basicActions > 1 || index != len(plan)-1 {
				return nil, errors.New("basic attack may appear once as the final action")
			}
		} else {
			guActions++
			if guActions > maxGuActions {
				return nil, errors.New("combat plan exceeds the Gu action limit")
			}
			ids := action.ComponentGuIDs
			if action.ActionType == "GU_SKILL" {
				id := action.InstanceID
				if id == "" {
					id = action.SourceID
				}
				ids = append([]string{id}, ids...)
			}
			for _, id := range ids {
				if id == "" {
					continue
				}
				if usedInstances[id] {
					return nil, errors.New("a Gu instance may only be used once per turn")
				}
				usedInstances[id] = true
			}
		}
		if combatantPaysMana(actor) {
			manaCost += projectedCombatManaCost(actor, action)
		}
		selected = append(selected, selectedCombatAction{action, target})
	}
	if manaCost > actor.Mana {
		return nil, errors.New("combat plan exceeds current mana")
	}
	// Self actions resolve before actions aimed at other combatants; preserve
	// player priority within each group for deterministic tie handling.
	stablePartition(selected, func(item selectedCombatAction) bool {
		return item.target != nil && item.target.ID == actor.ID
	})
	return selected, nil
}

// ValidateCombatPlan checks a player's locked plan against the session's
// current pre-resolution snapshot without changing combat state.
func ValidateCombatPlan(state *CombatState, actorID string, plan []CombatActionSelection, maxGuActions int) error {
	if state == nil {
		return errors.New("combat state is missing")
	}
	for _, actor := range state.Allies {
		if actor != nil && actor.ID == actorID {
			_, err := selectCombatPlan(actor, state.Enemies, plan, maxGuActions)
			return err
		}
	}
	for _, actor := range state.Enemies {
		if actor != nil && actor.ID == actorID {
			_, err := selectCombatPlan(actor, state.Allies, plan, maxGuActions)
			return err
		}
	}
	return errors.New("combat actor is not part of this session")
}

func resolveLockedCombatPlan(actor *CombatantAuto, enemies []*CombatantAuto, plan []CombatActionSelection) ([]selectedCombatAction, error) {
	if actor == nil || len(plan) == 0 {
		return nil, errors.New("combat plan is empty")
	}
	selected := make([]selectedCombatAction, 0, len(plan))
	for _, choice := range plan {
		if choice.ActionType == "FLEE" {
			if len(plan) != 1 {
				return nil, errors.New("flee must be the only action in a plan")
			}
			selected = append(selected, selectedCombatAction{action: &SimAction{ActionType: "FLEE", Name: "Bỏ chạy"}})
			continue
		}
		if choice.ActionType == "EXIT_GU_HOUSE" || choice.ActionType == "ENTER_GU_HOUSE" {
			action, target, err := selectCombatCommand(actor, enemies, CombatCommand{ActionType: choice.ActionType})
			if err != nil {
				continue
			}
			selected = append(selected, selectedCombatAction{action, target})
			continue
		}
		var action *SimAction
		for i := range actor.Actions {
			candidate := &actor.Actions[i]
			if candidate.ActionType == choice.ActionType && candidate.SourceID == choice.SourceID && candidate.InstanceID == choice.InstanceID {
				action = candidate
				break
			}
		}
		if action == nil && choice.ActionType == "BASIC_ATTACK" {
			action = &SimAction{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: actor.BaseDmg, DamageMult: 1, TargetType: "SINGLE", Path: []string{"strength"}}
		}
		if action == nil {
			return nil, errors.New("locked action is no longer in the prepared loadout")
		}
		var target *CombatantAuto
		if strings.EqualFold(action.TargetType, "SELF") {
			target = actor
		} else {
			for _, candidate := range enemies {
				if candidate != nil && candidate.ID == choice.TargetID {
					target = candidate
					break
				}
			}
		}
		selected = append(selected, selectedCombatAction{action: action, target: target})
	}
	stablePartition(selected, func(item selectedCombatAction) bool { return item.target != nil && item.target.ID == actor.ID })
	return selected, nil
}

func stablePartition[T any](items []T, first func(T) bool) {
	ordered := make([]T, 0, len(items))
	for _, item := range items {
		if first(item) {
			ordered = append(ordered, item)
		}
	}
	for _, item := range items {
		if !first(item) {
			ordered = append(ordered, item)
		}
	}
	copy(items, ordered)
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
	if actor != nil && actor.GuHouse != nil && !actor.GuHouse.IsDestroyed && actor.GuHouse.Active {
		basic := &SimAction{ActionType: "KILLER_MOVE", Name: "Trận Kích Cơ Bản", Damage: actor.BaseDmg, DamageMult: 1, TargetType: "SINGLE", IsFromGuHouse: true}
		return rememberAutoAction(actor, basic, target)
	}
	basic := &SimAction{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: actor.BaseDmg, DamageMult: 1, TargetType: "SINGLE", Path: []string{"strength"}}
	return rememberAutoAction(actor, basic, target)
}

// BuildAutomaticCombatPlan selects a full, deterministic action set from the
// actor's already-loaded actions. Targets and conditions are evaluated against
// the state at plan creation and remain locked for later slots.
func BuildAutomaticCombatPlan(actor *CombatantAuto, enemies []*CombatantAuto, maxGuActions int) []selectedCombatAction {
	if actor == nil {
		return nil
	}
	if maxGuActions < 1 {
		maxGuActions = 3
	}
	target := getTarget(enemies)
	profile := strings.ToLower(strings.TrimSpace(actor.AIProfile))
	ordered := make([]*SimAction, 0, len(actor.Actions))
	for i := range actor.Actions {
		ordered = append(ordered, &actor.Actions[i])
	}
	// Existing priority and category ordering remains authoritative for player
	// tactics; AI profiles use their established score ordering where relevant.
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		priority := func(action *SimAction) int {
			if action.ActionType == "BASIC_ATTACK" && action.Priority == 0 {
				return 999
			}
			if action.Priority == 0 {
				return 500
			}
			return action.Priority
		}
		if priority(a) != priority(b) {
			return priority(a) < priority(b)
		}
		if len(actor.CategoryPriorities) > 0 {
			return getCategoryPriorityIndex(GetSimActionCategory(*a), actor.CategoryPriorities) < getCategoryPriorityIndex(GetSimActionCategory(*b), actor.CategoryPriorities)
		}
		return false
	})
	if profile == "boss_burst" || profile == "boss_aggressive" || profile == "beast_aggressive" || profile == "beast_elite" || profile == "defensive" || profile == "boss_balanced" {
		sort.SliceStable(ordered, func(i, j int) bool {
			score := func(a *SimAction) float64 {
				lowHealth := actor.MaxHP > 0 && actor.HP/actor.MaxHP <= 0.4
				supportFirst := (profile == "defensive" || profile == "boss_balanced") && lowHealth
				isSupport := strings.EqualFold(a.TargetType, "SELF") || a.Healing > 0 || a.Defense > 0
				if supportFirst && isSupport {
					shieldMult := a.ShieldMult
					if shieldMult <= 0 {
						shieldMult = 1
					}
					return a.Healing + a.Defense*shieldMult
				}
				if supportFirst && !isSupport {
					return -1
				}
				if profile == "defensive" || profile == "boss_balanced" {
					if isSupport {
						return 0
					}
				}
				return a.Damage*a.DamageMult + a.DamageHPPercent*maxTargetHP(enemies)
			}
			si, sj := score(ordered[i]), score(ordered[j])
			if si != sj {
				return si > sj
			}
			return ordered[i].Priority < ordered[j].Priority
		})
	}
	plan := make([]selectedCombatAction, 0, maxGuActions+1)
	seen := make(map[string]bool)
	spent := 0.0
	for _, action := range ordered {
		if action.ActionType == "BASIC_ATTACK" {
			continue
		}
		if len(plan) >= maxGuActions {
			break
		}
		checkTarget := target
		if strings.EqualFold(action.TargetType, "SELF") {
			checkTarget = actor
		}
		if !combatActionAvailable(actor, action, checkTarget) || !checkActionCondition(action, actor, checkTarget) {
			continue
		}
		ids := append([]string(nil), action.ComponentGuIDs...)
		if action.ActionType == "GU_SKILL" {
			id := action.InstanceID
			if id == "" {
				id = action.SourceID
			}
			ids = append(ids, id)
		}
		duplicate := false
		for _, id := range ids {
			if id != "" && seen[id] {
				duplicate = true
			}
		}
		if duplicate {
			continue
		}
		cost := 0.0
		if combatantPaysMana(actor) {
			cost = projectedCombatManaCost(actor, action)
		}
		if spent+cost > actor.Mana && combatantPaysMana(actor) {
			continue
		}
		if checkTarget != nil && !CanSelectCombatTarget(actor, enemies, action, checkTarget) {
			continue
		}
		spent += cost
		for _, id := range ids {
			if id != "" {
				seen[id] = true
			}
		}
		plan = append(plan, selectedCombatAction{action: action, target: checkTarget})
	}
	if len(plan) == 0 && target != nil {
		if actor != nil && actor.GuHouse != nil && !actor.GuHouse.IsDestroyed && actor.GuHouse.Active {
			basic := &SimAction{ActionType: "KILLER_MOVE", Name: "Trận Kích Cơ Bản", Damage: actor.BaseDmg, DamageMult: 1, TargetType: "SINGLE", IsFromGuHouse: true}
			plan = append(plan, selectedCombatAction{basic, target})
		} else {
			basic := &SimAction{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: actor.BaseDmg, DamageMult: 1, TargetType: "SINGLE", Path: []string{"strength"}}
			plan = append(plan, selectedCombatAction{basic, target})
		}
	} else if target != nil {
		if actor == nil || actor.GuHouse == nil || actor.GuHouse.IsDestroyed || !actor.GuHouse.Active {
			basic := &SimAction{ActionType: "BASIC_ATTACK", Name: "Đánh thường", Damage: actor.BaseDmg, DamageMult: 1, TargetType: "SINGLE", Path: []string{"strength"}}
			plan = append(plan, selectedCombatAction{basic, target})
		}
	}
	// AUTO owns only the choice policy. Re-enter the same plan validator used
	// for player-submitted plans so resource, target, limit, and Gu-use rules
	// cannot drift between the two control methods.
	choices := make([]CombatActionSelection, 0, len(plan))
	for _, item := range plan {
		choice := CombatActionSelection{ActionType: item.action.ActionType, SourceID: item.action.SourceID, InstanceID: item.action.InstanceID}
		if item.target != nil {
			choice.TargetID = item.target.ID
		}
		choices = append(choices, choice)
	}
	validated, err := selectCombatPlan(actor, enemies, choices, maxGuActions)
	if err != nil {
		return nil
	}
	return validated
}

func maxTargetHP(targets []*CombatantAuto) float64 {
	var maxHP float64
	for _, target := range targets {
		if target != nil && target.HP > 0 && target.MaxHP > maxHP {
			maxHP = target.MaxHP
		}
	}
	return maxHP
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
