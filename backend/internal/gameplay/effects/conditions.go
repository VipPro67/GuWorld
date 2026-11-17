package effects

import (
	"fmt"
	"math"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

// ConditionValues contains only the values available in the current execution
// context. A missing key is different from a value of zero: evaluation must
// fail closed instead of treating unavailable context as zero.
type ConditionValues map[string]any

// ConditionsMatch applies the shared Block v2 condition semantics. Callers
// provide the values their context owns; unavailable fields return an error.
func ConditionsMatch(conditions []Condition, values ConditionValues) (bool, error) {
	for _, condition := range conditions {
		field := strings.ToUpper(strings.TrimSpace(condition.Field))
		actual, ok := values[field]
		if !ok {
			if field == "ACTOR_RACE" || field == "TARGET_RACE" {
				return false, nil
			}
			return false, fmt.Errorf("condition field %q is unavailable in this execution context", field)
		}
		operator := strings.ToUpper(strings.TrimSpace(condition.Operator))
		actual, err := effectcatalog.NormalizeConditionValue(field, operator, actual)
		if err != nil {
			return false, fmt.Errorf("condition field %q has invalid context value: %w", field, err)
		}
		expected, err := effectcatalog.NormalizeConditionValue(field, operator, condition.Value)
		if err != nil {
			return false, err
		}

		switch actual := actual.(type) {
		case string:
			value, ok := expected.(string)
			if !ok {
				return false, fmt.Errorf("condition field %q requires a string context value", field)
			}
			switch operator {
			case "EQ":
				if actual != value {
					return false, nil
				}
			case "NE":
				if actual == value {
					return false, nil
				}
			}
		case float64:
			value, ok := expected.(float64)
			if !ok {
				return false, fmt.Errorf("condition field %q requires a numeric context value", field)
			}
			if math.IsNaN(actual) || math.IsInf(actual, 0) || math.IsNaN(value) || math.IsInf(value, 0) {
				return false, fmt.Errorf("condition field %q contains a non-finite value", field)
			}
			switch operator {
			case "LT":
				if !(actual < value) {
					return false, nil
				}
			case "LTE":
				if !(actual <= value) {
					return false, nil
				}
			case "GT":
				if !(actual > value) {
					return false, nil
				}
			case "GTE":
				if !(actual >= value) {
					return false, nil
				}
			case "EQ":
				if actual != value {
					return false, nil
				}
			case "NE":
				if actual == value {
					return false, nil
				}
			}
		default:
			return false, fmt.Errorf("condition field %q has unsupported context value type %T", field, actual)
		}
	}
	return true, nil
}
