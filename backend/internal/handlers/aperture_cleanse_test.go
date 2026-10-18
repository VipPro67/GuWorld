package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestCleanseRefundQty(t *testing.T) {
	cases := []struct {
		cleansed int64
		want     int64
	}{
		{0, 0},
		{999, 0},
		{1000, 0}, // 30% = 300 < 1000
		{3333, 0}, // 30% = 999
		{3400, 1}, // 30% = 1020
		{5000, 1},
		{10000, 3},
	}
	for _, tc := range cases {
		if got := cleanseRefundQty(tc.cleansed); got != tc.want {
			t.Errorf("cleanseRefundQty(%d) = %d, want %d", tc.cleansed, got, tc.want)
		}
	}
}

func TestDaoMarksItemPath(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{`["Fire Path"]`, "fire"},
		{`["Killing Path"]`, "killing"},
		{`["Hỏa Đạo"]`, "hỏa đạo"},
		{`[]`, ""},
		{``, ""},
		{`not-json`, ""},
	}
	for _, tc := range cases {
		got := DaoMarksItemPath(models.ItemSpec{Path: tc.path})
		if got != tc.want {
			t.Errorf("DaoMarksItemPath(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

// TestDaoMaterialItemEffectsParse bám đúng hiệu ứng ghi trong Item.csv của item
// Đạo Tài: block ADD_DAO_MARKS phải parse được và key phải nằm trong whitelist
// CompileItemEffects (nếu không, LoadItemRegistry sẽ fail toàn bộ).
func TestDaoMaterialItemEffectsParse(t *testing.T) {
	raw := `[{"id": "ADD_DAO_MARKS", "target": {"selector": "SELF"}, "trigger": "ON_CONSUME", ` +
		`"operations": [{"op": "UNIQUE", "code": "IMMORTAL_ONLY", "params": {"value": 1}}, ` +
		`{"op": "UNIQUE", "code": "ADD_DAO_MARKS", "params": {"value": 1000}}]}]`

	var effects models.ItemEffects
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("Scan effects: %v", err)
	}
	if effects["add_dao_marks"] != 1000 {
		t.Fatalf("add_dao_marks = %v, want 1000 (effects=%v)", effects["add_dao_marks"], effects)
	}
	if effects["immortal_only"] != 1 {
		t.Fatalf("immortal_only = %v, want 1", effects["immortal_only"])
	}
	item := models.ItemSpec{ID: "dao_material_fire", Name: "Đạo Tài Hỏa Đạo", Type: "Consumable", Path: `["Fire Path"]`, Effects: effects}
	commands := item.EffectCommands()
	if len(commands) != 2 {
		t.Fatalf("EffectCommands = %v, want 2 commands", commands)
	}
	if _, ok := ApplyItemEffects["add_dao_marks"]; !ok {
		t.Fatal("ApplyItemEffects thiếu handler add_dao_marks")
	}
	if got := DaoMarksItemPath(item); got != "fire" {
		t.Fatalf("DaoMarksItemPath(item) = %q, want fire", got)
	}
}
