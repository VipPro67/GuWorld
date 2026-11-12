package effects

import (
	"fmt"
	"math"
	"strings"
)

// ConditionValues contains only the values available in the current execution
// context. A missing key is different from a value of zero: evaluation must
// fail closed instead of treating unavailable context as zero.
type ConditionValues map[string]float64

// ConditionsMatch applies the shared Block v2 condition semantics. Callers
// provide the values their context owns; unavailable fields return an error.
func ConditionsMatch(conditions []Condition, values ConditionValues) (bool, error) {
	for _, condition := range conditions {
		field := strings.ToUpper(strings.TrimSpace(condition.Field))
		actual, ok := values[field]
		if !ok {
			return false, fmt.Errorf("condition field %q is unavailable in this execution context", field)
		}
		if math.IsNaN(actual) || math.IsInf(actual, 0) || math.IsNaN(condition.Value) || math.IsInf(condition.Value, 0) {
			return false, fmt.Errorf("condition field %q contains a non-finite value", field)
		}

		switch strings.ToUpper(strings.TrimSpace(condition.Operator)) {
		case "LT":
			if !(actual < condition.Value) {
				return false, nil
			}
		case "LTE":
			if !(actual <= condition.Value) {
				return false, nil
			}
		case "GT":
			if !(actual > condition.Value) {
				return false, nil
			}
		case "GTE":
			if !(actual >= condition.Value) {
				return false, nil
			}
		case "EQ":
			if actual != condition.Value {
				return false, nil
			}
		case "NE":
			if actual == condition.Value {
				return false, nil
			}
		default:
			return false, fmt.Errorf("condition operator %q is unsupported", condition.Operator)
		}
	}
	return true, nil
}
