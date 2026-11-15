package worker

import (
	"reflect"
	"testing"
)

func TestPlanRefinementMaterialDeductions(t *testing.T) {
	ids := []string{"stone", "wood"}
	materials := map[string]int64{"stone": 3, "wood": 6}
	inventory := map[string][]refinementInventoryEntry{
		"stone": {{materialID: "stone", category: "materials", quantity: 8}},
		"wood": {
			{materialID: "wood", category: "materials", quantity: 5},
			{materialID: "wood", category: "treasures", quantity: 2},
		},
	}
	want := []refinementMaterialDeduction{
		{materialID: "stone", category: "materials", quantity: 3},
		{materialID: "wood", category: "materials", quantity: 5},
		{materialID: "wood", category: "treasures", quantity: 1},
	}

	got, sufficient := planRefinementMaterialDeductions(ids, materials, inventory)
	if !sufficient {
		t.Fatal("expected sufficient inventory")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("deductions = %#v, want %#v", got, want)
	}
}

func TestPlanRefinementMaterialDeductionsRejectsShortInventoryAtomically(t *testing.T) {
	ids := []string{"stone", "wood"}
	materials := map[string]int64{"stone": 3, "wood": 6}
	inventory := map[string][]refinementInventoryEntry{
		"stone": {{materialID: "stone", category: "materials", quantity: 8}},
		"wood":  {{materialID: "wood", category: "materials", quantity: 5}},
	}

	got, sufficient := planRefinementMaterialDeductions(ids, materials, inventory)
	if sufficient {
		t.Fatal("expected insufficient inventory")
	}
	if len(got) != 0 {
		t.Fatalf("returned partial deductions on insufficiency: %#v", got)
	}
}
