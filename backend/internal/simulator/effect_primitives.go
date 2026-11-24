package simulator

import (
	"strings"
)

// addBoundedCounter updates a combatant's integer effect state and returns the
// resulting value. A non-positive maximum means the counter is unbounded.
func addBoundedCounter(combatant *CombatantAuto, key string, delta, minimum, maximum int) int {
	if combatant == nil || delta == 0 {
		if combatant == nil || combatant.CustomState == nil {
			return minimum
		}
		return combatant.CustomState[key]
	}
	if combatant.CustomState == nil {
		combatant.CustomState = make(map[string]int)
	}

	value := combatant.CustomState[key] + delta
	if value < minimum {
		value = minimum
	}
	if maximum > 0 && value > maximum {
		value = maximum
	}
	combatant.CustomState[key] = value
	return value
}

// ApplyCounter is the public helper for the COUNTER_ADD primitive.
func ApplyCounter(combatant *CombatantAuto, key string, delta, minimum, maximum int) int {
	return addBoundedCounter(combatant, key, delta, minimum, maximum)
}

func applyTurnStartCounterAdds(combatant *CombatantAuto) bool {
	if combatant == nil {
		return false
	}
	poemCounter := false
	for _, counter := range combatant.TurnStartCounterAdds {
		ApplyCounter(combatant, counter.Key, counter.Delta, counter.Min, counter.Max)
		if counter.Key == "thi_y" {
			poemCounter = true
		}
	}
	return poemCounter
}

// CheckThreshold evaluates whether a combatant's counter has met or exceeded
// the specified threshold. When met and autoReset is true, the counter is reset to 0.
func CheckThreshold(combatant *CombatantAuto, key string, threshold int, autoReset bool) bool {
	if combatant == nil || combatant.CustomState == nil || threshold <= 0 {
		return false
	}
	val := combatant.CustomState[key]
	if val >= threshold {
		if autoReset {
			combatant.CustomState[key] = 0
		}
		return true
	}
	return false
}

// StackingDispelRule defines how an APPLY_STACKING_STATUS effect is cleansed.
type StackingDispelRule string

const (
	DispelDecrementOne StackingDispelRule = "DECREMENT_ONE"
	DispelClearAll     StackingDispelRule = "CLEAR_ALL"
)

// StackingStatusTier configures the scaling percentage for a specific stack tier.
type StackingStatusTier struct {
	Tier        int     `json:"tier"`
	StatPercent float64 `json:"stat_percent"`
}

// StackingStatusState tracks active stacks on a combatant for a stacking status.
type StackingStatusState struct {
	Key           string
	CurrentStacks int
	MaxStacks     int
	TurnsLeft     int
	DispelRule    StackingDispelRule
	Tiers         []StackingStatusTier
}

func stackingStatusStateKey(key string) string {
	return "stacking_status_" + strings.ToLower(strings.TrimSpace(key))
}

// ApplyStackingStatus applies or increments stacks on a combatant.
func ApplyStackingStatus(combatant *CombatantAuto, key string, deltaStacks, maxStacks, durationTurns int, dispelRule StackingDispelRule, tiers []StackingStatusTier) int {
	if combatant == nil || key == "" || deltaStacks == 0 {
		return 0
	}
	stateKey := stackingStatusStateKey(key)
	cur := addBoundedCounter(combatant, stateKey, deltaStacks, 0, maxStacks)
	if durationTurns > 0 {
		durKey := stateKey + "_turns"
		combatant.CustomState[durKey] = durationTurns
	}
	return cur
}

// DispelStackingStatus handles dispel for a stacking status based on its rule.
func DispelStackingStatus(combatant *CombatantAuto, key string, rule StackingDispelRule) (removed int, remaining int) {
	if combatant == nil || combatant.CustomState == nil || key == "" {
		return 0, 0
	}
	stateKey := stackingStatusStateKey(key)
	cur := combatant.CustomState[stateKey]
	if cur <= 0 {
		return 0, 0
	}
	if rule == DispelClearAll {
		combatant.CustomState[stateKey] = 0
		delete(combatant.CustomState, stateKey+"_turns")
		return cur, 0
	}
	// Default: DECREMENT_ONE
	combatant.CustomState[stateKey] = cur - 1
	if combatant.CustomState[stateKey] <= 0 {
		delete(combatant.CustomState, stateKey+"_turns")
	}
	return 1, combatant.CustomState[stateKey]
}

// GetStackingStatusPercent returns the active scaling percentage for the combatant's stack tier.
func GetStackingStatusPercent(combatant *CombatantAuto, key string, tiers []StackingStatusTier) float64 {
	if combatant == nil || combatant.CustomState == nil || key == "" {
		return 0
	}
	stateKey := stackingStatusStateKey(key)
	cur := combatant.CustomState[stateKey]
	if cur <= 0 {
		return 0
	}
	for _, tier := range tiers {
		if tier.Tier == cur {
			return tier.StatPercent
		}
	}
	// Fallback to highest tier if stacks exceed max declared tier
	var highestTier int
	var highestPct float64
	for _, tier := range tiers {
		if tier.Tier > highestTier && cur >= tier.Tier {
			highestTier = tier.Tier
			highestPct = tier.StatPercent
		}
	}
	return highestPct
}
