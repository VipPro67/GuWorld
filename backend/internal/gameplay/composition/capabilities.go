package composition

import (
	"fmt"
	"math"
	"strings"

	effects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

const capabilityFlatSingleTargetDamage = "COMBAT.DAMAGE.FLAT.SINGLE_TARGET"
const capabilityTravelArrivalSelfEffect = "WORLD_TRAVEL.ARRIVAL.SELF_EFFECT"

func knownCapability(id string) bool {
	return id == capabilityFlatSingleTargetDamage || id == capabilityTravelArrivalSelfEffect
}

// validateCapabilityGrant checks that the authored capability describes the
// operation it is attached to. Unknown capability IDs fail closed until their
// semantics are added here.
func validateCapabilityGrant(id string, block effects.Block, operation effects.Operation) error {
	switch id {
	case capabilityFlatSingleTargetDamage:
		target := block.Target
		if operation.Target != nil {
			target = *operation.Target
		}
		if block.Trigger != effects.TriggerOnAction || operation.Op != effects.OpDamage || !strings.EqualFold(operation.SubType, "FLAT") || target.Selector != effects.TargetSingle {
			return fmt.Errorf("capability %q requires ON_ACTION DAMAGE FLAT targeting one opponent", id)
		}
		return nil
	case capabilityTravelArrivalSelfEffect:
		target := block.Target
		if operation.Target != nil {
			target = *operation.Target
		}
		if block.Trigger != effects.TriggerOnUse || target.Selector != effects.TargetSelf {
			return fmt.Errorf("capability %q requires ON_USE targeting SELF", id)
		}
		switch operation.Op {
		case effects.OpHeal:
			subType := strings.ToUpper(strings.TrimSpace(operation.SubType))
			if operation.Resource != "" && !(subType == "RESOURCE" && strings.EqualFold(operation.Resource, "MANA")) {
				return fmt.Errorf("capability %q only allows self healing or mana recovery", id)
			}
			if subType != "" && subType != "FLAT" && subType != "PERCENT" && subType != "RESOURCE" {
				return fmt.Errorf("capability %q has unsupported healing subtype %q", id, subType)
			}
			if subType == "RESOURCE" && !strings.EqualFold(operation.Resource, "MANA") {
				return fmt.Errorf("capability %q only allows mana resource recovery", id)
			}
			return nil
		case effects.OpStatus:
			statusID := strings.ToUpper(strings.TrimSpace(operation.StatusID))
			if statusID != statusfx.StatusFlying && statusID != statusfx.StatusStealth && statusID != statusfx.StatusHaste {
				return fmt.Errorf("capability %q does not allow status %q", id, statusID)
			}
			if operation.Turns < 0 || operation.Turns > 24*60 || operation.Value < 0 || operation.Value > 100 || operation.Chance < 0 || operation.Chance > 1 || (operation.Chance > 0 && operation.Chance < 1) || math.IsNaN(operation.Value) || math.IsInf(operation.Value, 0) || math.IsNaN(operation.Chance) || math.IsInf(operation.Chance, 0) {
				return fmt.Errorf("capability %q has invalid status bounds", id)
			}
			return nil
		default:
			return fmt.Errorf("capability %q does not allow operation %q", id, operation.Op)
		}
	default:
		return fmt.Errorf("unknown composition capability %q", id)
	}
}
