package sim

import (
	"path/filepath"
	"testing"
)

func TestLoadTransformationForms(t *testing.T) {
	csvPath := filepath.Join("..", "..", "..", "data", "TransformationForm.csv")
	err := LoadTransformationForms(csvPath)
	if err != nil {
		t.Fatalf("LoadTransformationForms failed: %v", err)
	}

	count := CountTransformationForms()
	if count < 5 {
		t.Fatalf("expected at least 5 forms, got %d", count)
	}

	// Verify Flame Bear
	bear, ok := GetTransformationForm("FORM_FLAME_BEAR")
	if !ok {
		t.Fatalf("FORM_FLAME_BEAR not found in registry")
	}
	if bear.FormKind != TransformationFull {
		t.Errorf("expected FULL form kind, got %s", bear.FormKind)
	}
	if bear.MaxFormHP != 180 {
		t.Errorf("expected 180 MaxFormHP, got %f", bear.MaxFormHP)
	}

	// Verify Six Arm Demon
	demon, ok := GetTransformationForm("FORM_SIX_ARM_ICE_DEMON")
	if !ok {
		t.Fatalf("FORM_SIX_ARM_ICE_DEMON not found")
	}
	if demon.CollisionRadiusMult != 1.6 {
		t.Errorf("expected 1.6 radius mult, got %f", demon.CollisionRadiusMult)
	}

	// Verify Partial Wolf
	wolf, ok := GetTransformationForm("FORM_WOLF_PARTIAL")
	if !ok {
		t.Fatalf("FORM_WOLF_PARTIAL not found")
	}
	if wolf.FormKind != TransformationPartial {
		t.Errorf("expected PARTIAL, got %s", wolf.FormKind)
	}
}
