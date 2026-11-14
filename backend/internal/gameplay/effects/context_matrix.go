package effects

import (
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

// ContextRule defines the constraints for blocks evaluated within a domain context.
type ContextRule struct {
	AllowedTriggers   map[Trigger]bool
	AllowedSelectors  map[Selector]bool
	AllowedOperations map[OperationType]bool
	AllowConditions   bool
	ValidateOperation func(op Operation) error
}

// ContextRules associates each domain with its allowed execution parameters.
var ContextRules = map[Domain]ContextRule{
	DomainItemUse: {
		AllowedTriggers: map[Trigger]bool{
			TriggerOnConsume: true,
		},
		AllowedSelectors: map[Selector]bool{
			TargetSelf: true,
		},
		AllowedOperations: map[OperationType]bool{
			OpUnique:  true,
			OpHeal:    true,
			OpStatMod: true,
		},
		AllowConditions: false,
		ValidateOperation: func(op Operation) error {
			if op.Op == OpUnique {
				code := strings.ToUpper(strings.TrimSpace(op.Code))
				hook, ok := effectcatalog.NativeHooks[code]
				if !ok {
					return fmt.Errorf("item action %q is not registered", op.Code)
				}
				if hook.Domain != string(DomainItemUse) && !hook.Triggers[string(TriggerOnConsume)] {
					return fmt.Errorf("item action %q is not supported in item_use domain", op.Code)
				}
			}
			return nil
		},
	},
	DomainDerived: {
		AllowedTriggers: map[Trigger]bool{
			TriggerAlwaysOn: true,
			TriggerPassive:  true,
		},
		AllowedSelectors: map[Selector]bool{
			TargetSelf: true,
		},
		AllowedOperations: map[OperationType]bool{
			OpStatMod: true,
			OpHeal:    true,
			OpUnique:  true,
		},
		AllowConditions: false,
		ValidateOperation: func(op Operation) error {
			if op.Op == OpUnique {
				code := strings.ToUpper(strings.TrimSpace(op.Code))
				hook, ok := effectcatalog.NativeHooks[code]
				if !ok {
					return fmt.Errorf("derived action %q is not registered", op.Code)
				}
				if hook.Domain != "gamefx" && hook.Domain != string(DomainDerived) && !hook.Triggers[string(TriggerAlwaysOn)] {
					return fmt.Errorf("action %q is not supported in derived domain", op.Code)
				}
			}
			return nil
		},
	},
	DomainWorld: {
		AllowedTriggers: map[Trigger]bool{
			TriggerOnUse:    true,
			TriggerAlwaysOn: true,
		},
		AllowedSelectors: map[Selector]bool{
			TargetSelf: true,
		},
		AllowedOperations: map[OperationType]bool{
			OpStatus:  true,
			OpHeal:    true,
			OpUnique:  true,
			OpStatMod: true,
		},
		AllowConditions: true,
	},
	DomainGrowth: {
		AllowedTriggers: map[Trigger]bool{
			TriggerOnTick:   true,
			TriggerAlwaysOn: true,
		},
		AllowedSelectors: map[Selector]bool{
			TargetSelf: true,
		},
		AllowedOperations: map[OperationType]bool{
			OpUnique:  true,
			OpStatMod: true,
		},
		AllowConditions: false,
	},
	DomainCombat: {
		AllowedTriggers: map[Trigger]bool{
			TriggerOnAction:          true,
			TriggerOnHit:             true,
			TriggerBeforeHit:         true,
			TriggerOnTurnStart:       true,
			TriggerOnDeath:           true,
			TriggerCombatPassive:     true,
			TriggerPassive:           true,
			TriggerOnBattleStart:     true,
			TriggerBeforeEnemyAttack: true,
			TriggerOnCast:            true,
		},
		AllowedSelectors: map[Selector]bool{
			TargetSelf:       true,
			TargetSingle:     true,
			TargetRow:        true,
			TargetCol:        true,
			TargetAllEnemies: true,
			TargetAllAllies:  true,
			TargetAll:        true,
		},
		AllowedOperations: map[OperationType]bool{
			OpDamage:   true,
			OpShield:   true,
			OpHeal:     true,
			OpDirectHP: true,
			OpStatMod:  true,
			OpStatus:   true,
			OpBarrier:  true,
			OpCleanse:  true,
			OpUnique:   true,
		},
		AllowConditions: true,
	},
}

// ValidateContextSupport checks whether the provided blocks conform to the context's rules.
// It fails-closed on any unsupported trigger, selector, operation, or condition.
func ValidateContextSupport(domain Domain, blocks []Block) error {
	rule, exists := ContextRules[domain]
	if !exists {
		// Domain without specific constraints passes through standard compiler checks
		return nil
	}

	for _, block := range blocks {
		if rule.AllowedTriggers != nil && !rule.AllowedTriggers[block.Trigger] {
			return fmt.Errorf("domain %s does not support trigger %q in block %q", domain, block.Trigger, block.ID)
		}
		if rule.AllowedSelectors != nil && !rule.AllowedSelectors[block.Target.Selector] {
			return fmt.Errorf("domain %s does not support target selector %q in block %q", domain, block.Target.Selector, block.ID)
		}
		if !rule.AllowConditions && len(block.Conditions) > 0 {
			return fmt.Errorf("domain %s requires unconditional execution in block %q", domain, block.ID)
		}

		for _, op := range block.Operations {
			if rule.AllowedOperations != nil && !rule.AllowedOperations[op.Op] {
				return fmt.Errorf("domain %s does not support operation %q in block %q", domain, op.Op, block.ID)
			}
			if op.Target != nil && rule.AllowedSelectors != nil && !rule.AllowedSelectors[op.Target.Selector] {
				return fmt.Errorf("domain %s does not support operation target selector %q in block %q", domain, op.Target.Selector, block.ID)
			}
			if rule.ValidateOperation != nil {
				if err := rule.ValidateOperation(op); err != nil {
					return fmt.Errorf("domain %s block %q: %w", domain, block.ID, err)
				}
			}
		}
	}
	return nil
}
