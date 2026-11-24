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

func TestCombatContextRejectsThresholdCheckAndAcceptsPersistentStatus(t *testing.T) {
	op := Operation{Op: OpThresholdCheck, Key: "test", Threshold: 3}
	{
		block := []Block{{
			ID: "UNSUPPORTED_STATEFUL_PRIMITIVE", Trigger: TriggerOnAction,
			Target: Target{Selector: TargetSelf}, Operations: []Operation{op},
		}}
		if err := ValidateContextSupport(DomainCombat, block); err == nil {
			t.Errorf("combat context accepted %q without an executor", op.Op)
		}
	}

	stacking := []Block{{
		ID: "PERSISTENT_HIT_STATUS", Trigger: TriggerOnHit,
		Target:     Target{Selector: TargetSingle},
		Operations: []Operation{{Op: OpApplyStackingStatus, StatusKey: "test", MaxStacks: 5, PvPOnly: true}},
	}}
	if err := ValidateContextSupport(DomainCombat, stacking); err != nil {
		t.Fatalf("combat context rejected the post-combat persistent-status consumer: %v", err)
	}

	block := []Block{{
		ID: "COUNTER_ADD", Trigger: TriggerOnAction,
		Target:     Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpCounterAdd, Key: "test", Delta: 1, Min: 0, Max: 5}},
	}}
	if err := ValidateContextSupport(DomainCombat, block); err != nil {
		t.Fatalf("combat context rejected the implemented COUNTER_ADD primitive: %v", err)
	}
}

func TestCombatNotificationAndTargetedManaRestoreAreAccepted(t *testing.T) {
	blocks := []Block{
		{ID: "MANA", Trigger: TriggerOnAction, Target: Target{Selector: TargetAllAllies}, Operations: []Operation{{Op: OpHeal, SubType: "RESOURCE", Resource: "MANA", Unit: "PERCENT", Value: 20, SuppressLog: true}}},
		{ID: "BUFF", Trigger: TriggerOnAction, Target: Target{Selector: TargetAllAllies}, Operations: []Operation{{Op: OpStatus, StatusID: "SONG_OF_DESTINY", Value: 40, Turns: 1, ApplicationMode: "DIRECT", StackPolicy: "REPLACE", SuppressLog: true}}},
		{ID: "LOG", Trigger: TriggerOnAction, Target: Target{Selector: TargetSelf}, Operations: []Operation{{Op: OpNotify, Message: "battle effect", LogAction: "EFFECT_NOTIFY"}}},
	}
	if err := ValidateContextSupport(DomainCombat, blocks); err != nil {
		t.Fatalf("combat notification and party mana restore rejected: %v", err)
	}
	compiled, err := (Compiler{}).Compile(blocks)
	if err != nil {
		t.Fatalf("combat notification and party mana restore did not compile: %v", err)
	}
	status := compiled.Operations(TriggerOnAction)[1].Status
	if status == nil || !status.ApplyToTargets {
		t.Fatalf("ALL_ALLIES selector did not override self-target status behavior: %+v", status)
	}
}

func TestDomainRefinementAcceptsDeductionChancePrimitive(t *testing.T) {
	blocks := []Block{{
		ID: "DEDUCTION_CHANCE", Trigger: TriggerAlwaysOn, Target: Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpDeductionFinishChance, Params: []byte(`{"base_chance":0.25,"per_rank_step":0.05,"max_chance":0.95`)}},
	}}
	if err := ValidateContextSupport(DomainRefinement, blocks); err != nil {
		t.Fatalf("refinement chance block rejected: %v", err)
	}
	blocks[0].Operations[0].Op = OpDamage
	if err := ValidateContextSupport(DomainRefinement, blocks); err == nil {
		t.Fatal("combat damage operation accepted in refinement domain")
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

func TestValidateContextSupportWorldPlacementOnConsume(t *testing.T) {
	block := []Block{{
		ID:      "PLANT_SPRING",
		Trigger: TriggerOnConsume,
		Target:  Target{Selector: TargetSelf},
		Operations: []Operation{{
			Op:      OpPlaceWorldObject,
			SubType: "PRIMEVAL_SPRING",
		}},
	}}
	if err := ValidateContextSupport(DomainWorld, block); err != nil {
		t.Fatalf("expected typed world placement to pass in the world domain: %v", err)
	}
}

func TestValidateContextSupportGuUseNotify(t *testing.T) {
	block := []Block{{
		ID: "RECORD_MESSAGE", Trigger: TriggerOnUse, Target: Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpNotify, Message: "{gu} ghi lại."}},
	}}
	if err := ValidateContextSupport(DomainGuUse, block); err != nil {
		t.Fatalf("expected NOTIFY to pass in Gu-use context: %v", err)
	}
	if err := ValidateContextSupport(DomainWorld, block); err == nil {
		t.Fatal("NOTIFY must not be advertised as a generic world-use operation")
	}
}

func TestValidateContextSupportGuUseAptitudeModifier(t *testing.T) {
	valid := []Block{{
		ID: "APTITUDE", Trigger: TriggerOnConsume, Target: Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpModifyAptitude, SubType: "ADD_FLAT", Value: 5}},
	}}
	if err := ValidateContextSupport(DomainGuUse, valid); err != nil {
		t.Fatalf("valid Gu aptitude operation rejected: %v", err)
	}
	valid[0].Trigger = TriggerOnAction
	if err := ValidateContextSupport(DomainGuUse, valid); err == nil {
		t.Fatal("combat trigger accepted for Gu use aptitude operation")
	}
}

func TestValidateContextSupportGrowthDuration(t *testing.T) {
	block := []Block{{
		ID:         "GROWTH",
		Trigger:    TriggerAlwaysOn,
		Target:     Target{Selector: TargetSelf},
		Operations: []Operation{{Op: OpGrowthDuration, Value: 48}},
	}}
	if err := ValidateContextSupport(DomainGrowth, block); err != nil {
		t.Fatalf("expected typed growth duration to pass in the growth domain: %v", err)
	}
}
