package effects

import (
	"testing"
)

func TestValidateContextSupportItemUse(t *testing.T) {
	validItemBlock := []Block{
		{
			ID:      "HEAL_POTION",
			Trigger: TriggerOnConsume,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{
				{Op: OpUnique, Code: "RESTORE_HP", Value: 50},
			},
		},
	}
	if err := ValidateContextSupport(DomainItemUse, validItemBlock); err != nil {
		t.Fatalf("expected valid item use block to pass, got: %v", err)
	}

	// Reject Combat trigger in ItemUse
	combatTriggerBlock := []Block{
		{
			ID:      "INVALID_TRIGGER",
			Trigger: TriggerOnAction,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{
				{Op: OpUnique, Code: "RESTORE_HP", Value: 50},
			},
		},
	}
	if err := ValidateContextSupport(DomainItemUse, combatTriggerBlock); err == nil {
		t.Fatal("expected ON_ACTION to be rejected in DomainItemUse")
	}

	// Reject TargetSingle in ItemUse
	targetSingleBlock := []Block{
		{
			ID:      "INVALID_TARGET",
			Trigger: TriggerOnConsume,
			Target:  Target{Selector: TargetSingle},
			Operations: []Operation{
				{Op: OpUnique, Code: "RESTORE_HP", Value: 50},
			},
		},
	}
	if err := ValidateContextSupport(DomainItemUse, targetSingleBlock); err == nil {
		t.Fatal("expected TargetSingle to be rejected in DomainItemUse")
	}

	// Reject OpDamage in ItemUse
	damageOpBlock := []Block{
		{
			ID:      "INVALID_OP",
			Trigger: TriggerOnConsume,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{
				{Op: OpDamage, SubType: "FLAT", Value: 50},
			},
		},
	}
	if err := ValidateContextSupport(DomainItemUse, damageOpBlock); err == nil {
		t.Fatal("expected OpDamage to be rejected in DomainItemUse")
	}

	// Reject Conditions in ItemUse (unconditional execution)
	conditionBlock := []Block{
		{
			ID:         "INVALID_CONDITION",
			Trigger:    TriggerOnConsume,
			Target:     Target{Selector: TargetSelf},
			Conditions: []Condition{{Field: "RANK", Operator: "GTE", Value: 2}},
			Operations: []Operation{
				{Op: OpUnique, Code: "RESTORE_HP", Value: 50},
			},
		},
	}
	if err := ValidateContextSupport(DomainItemUse, conditionBlock); err == nil {
		t.Fatal("expected conditions to be rejected in DomainItemUse")
	}
}

func TestValidateContextSupportDerived(t *testing.T) {
	validDerivedBlock := []Block{
		{
			ID:      "PASSIVE_STAT",
			Trigger: TriggerAlwaysOn,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{
				{Op: OpStatMod, SubType: "ADD_FLAT", Stat: "ATTACK", Value: 10},
			},
		},
	}
	if err := ValidateContextSupport(DomainDerived, validDerivedBlock); err != nil {
		t.Fatalf("expected valid derived block to pass, got: %v", err)
	}

	// Reject ON_ACTION in Derived
	invalidTriggerBlock := []Block{
		{
			ID:      "BAD_TRIGGER",
			Trigger: TriggerOnAction,
			Target:  Target{Selector: TargetSelf},
			Operations: []Operation{
				{Op: OpStatMod, SubType: "ADD_FLAT", Stat: "ATTACK", Value: 10},
			},
		},
	}
	if err := ValidateContextSupport(DomainDerived, invalidTriggerBlock); err == nil {
		t.Fatal("expected ON_ACTION to be rejected in DomainDerived")
	}
}
