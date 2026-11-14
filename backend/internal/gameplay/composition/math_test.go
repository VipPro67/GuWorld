package composition

import (
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

func TestEffectLoadCalculation(t *testing.T) {
	op := gameplayeffects.Operation{
		Op:    gameplayeffects.OpDamage,
		Value: 50,
	}
	targetSingle := gameplayeffects.Target{Selector: gameplayeffects.TargetSingle}
	targetAll := gameplayeffects.Target{Selector: gameplayeffects.TargetAll}

	singleLoad := EffectLoad(op, targetSingle)
	allLoad := EffectLoad(op, targetAll)

	if singleLoad <= 0 {
		t.Fatalf("expected positive load, got %f", singleLoad)
	}
	if allLoad <= singleLoad {
		t.Fatalf("expected all-target load (%f) > single-target load (%f)", allLoad, singleLoad)
	}
}

func TestScaleOperation(t *testing.T) {
	op := gameplayeffects.Operation{
		Op:    gameplayeffects.OpDamage,
		Value: 100,
	}
	ScaleOperation(&op, 0.8)
	if op.Value != 80 {
		t.Fatalf("expected scaled value 80, got %f", op.Value)
	}
}

func TestSameOperationSemantics(t *testing.T) {
	op1 := gameplayeffects.Operation{Op: gameplayeffects.OpDamage, SubType: "FLAT"}
	op2 := gameplayeffects.Operation{Op: gameplayeffects.OpDamage, SubType: "FLAT"}
	op3 := gameplayeffects.Operation{Op: gameplayeffects.OpDamage, SubType: "PERCENT"}

	if !SameOperationSemantics(op1, op2) {
		t.Fatal("expected op1 and op2 to have same semantics")
	}
	if SameOperationSemantics(op1, op3) {
		t.Fatal("expected op1 and op3 to differ in subtype")
	}
}

func TestAdjustedSuccessChanceClamping(t *testing.T) {
	// Base high should clamp to 0.95
	high := AdjustedSuccessChance(1.2, 1, 5, 5000)
	if high != 0.95 {
		t.Fatalf("expected clamped high 0.95, got %f", high)
	}

	// Base low should clamp to 0.05
	low := AdjustedSuccessChance(-0.5, 5, 1, 0)
	if low != 0.05 {
		t.Fatalf("expected clamped low 0.05, got %f", low)
	}
}
