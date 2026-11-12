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
