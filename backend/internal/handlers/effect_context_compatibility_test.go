package handlers

import (
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

func TestCompositionSourceConditionsMatchProductExecutor(t *testing.T) {
	mastery := []gameplayeffects.Condition{{Field: "MASTERY", Operator: "GTE", Value: 2}}
	if !adminCompositionOperationAllowed("killer_move.v1", gameplayeffects.TriggerOnAction, mastery, gameplayeffects.Operation{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 10}, gameplayeffects.Target{Selector: gameplayeffects.TargetSingle}) {
		t.Fatal("turn-based killer move should allow a condition supplied by its simulator context")
	}
	if adminCompositionOperationAllowed("teleport_formation.v1", gameplayeffects.TriggerOnUse, mastery, gameplayeffects.Operation{Op: gameplayeffects.OpHeal, SubType: "FLAT", Value: 10}, gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}) {
		t.Fatal("teleport adapter must reject MASTERY because its arrival context does not supply it")
	}
	stacks := []gameplayeffects.Condition{{Field: "STACKS", Operator: "GTE", Value: 2}}
	if adminCompositionOperationAllowed("killer_move.v1", gameplayeffects.TriggerOnAction, stacks, gameplayeffects.Operation{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 10}, gameplayeffects.Target{Selector: gameplayeffects.TargetSingle}) {
		t.Fatal("composition source must reject STACKS because the simulator action context does not supply it")
	}
}

func TestRaceConditionsAreAvailableOnlyInContextsWithRaceValues(t *testing.T) {
	race := []gameplayeffects.Condition{{Field: "TARGET_RACE", Operator: "EQ", Value: "merman"}}
	if reason := unsupportedConditionReason("TURN_BASED_COMBAT", race); reason != "" {
		t.Fatalf("unsupportedConditionReason(TURN_BASED_COMBAT, race) = %q, want supported", reason)
	}
	if reason := unsupportedConditionReason("TELEPORT_ARRAY_ARRIVAL", race); reason != "" {
		t.Fatalf("unsupportedConditionReason(TELEPORT_ARRAY_ARRIVAL, race) = %q, want supported", reason)
	}
	if reason := unsupportedConditionReason("UNKNOWN_CONTEXT", race); reason == "" {
		t.Fatal("unsupportedConditionReason(UNKNOWN_CONTEXT, race) succeeded, want unsupported context")
	}
}

func TestGuConsumeCompatibilitySupportsTypedWorldPlacementAndConditions(t *testing.T) {
	block := gameplayeffects.Block{
		ID: "PLANT_SPRING", Trigger: gameplayeffects.TriggerOnConsume,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSelf},
		Conditions: []gameplayeffects.Condition{{Field: "RANK", Operator: "GTE", Value: 5}},
	}
	op := gameplayeffects.Operation{Op: gameplayeffects.OpPlaceWorldObject, SubType: "PRIMEVAL_SPRING"}
	status, reason := checkEffectCompatibility("GU_CONSUME_EVENT", block, op)
	if status != "conditional" || reason == "" {
		t.Fatalf("consume placement compatibility = (%q, %q), want conditional with explanation", status, reason)
	}

	block.Trigger = gameplayeffects.TriggerOnUse
	if status, _ = checkEffectCompatibility("GU_CONSUME_EVENT", block, op); status != "unsupported" {
		t.Fatalf("ON_USE placement status = %q, want unsupported", status)
	}
}

func TestGuConsumeConditionsRejectUnavailableTurnValue(t *testing.T) {
	conditions := []gameplayeffects.Condition{{Field: "TURN", Operator: "GTE", Value: 1}}
	if reason := unsupportedConditionReason("GU_CONSUME_EVENT", conditions); reason == "" {
		t.Fatal("Gu consume compatibility accepted TURN without an event turn")
	}
}

func TestRefinementCompatibilityRecognizesDeductionChancePrimitive(t *testing.T) {
	block := gameplayeffects.Block{ID: "DEDUCTION_FINISH_CHANCE", Trigger: gameplayeffects.TriggerAlwaysOn, Target: gameplayeffects.Target{Selector: gameplayeffects.TargetSelf}}
	status, reason := checkEffectCompatibility("REFINEMENT", block, gameplayeffects.Operation{Op: gameplayeffects.OpDeductionFinishChance, Params: []byte(`{"base_chance":0.2}`)})
	if status != "supported" || reason != "" {
		t.Fatalf("deduction chance compatibility = (%q, %q), want supported", status, reason)
	}
	block.Conditions = []gameplayeffects.Condition{{Field: "RANK", Operator: "GTE", Value: 2}}
	if status, _ := checkEffectCompatibility("REFINEMENT", block, gameplayeffects.Operation{Op: gameplayeffects.OpDeductionFinishChance}); status != "unsupported" {
		t.Fatalf("conditional deduction chance status = %q, want unsupported", status)
	}
}
