package simulator

import "testing"

func TestCombatAreaCells(t *testing.T) {
	tests := []struct {
		name       string
		targetType string
		x, y       int
		want       int
	}{
		{name: "row", targetType: "ROW", x: 3, y: 3, want: 3},
		{name: "column", targetType: "COL", x: 3, y: 3, want: 3},
		{name: "cross", targetType: "CROSS", x: 3, y: 3, want: 3},
		{name: "all", targetType: "ALL", x: 3, y: 3, want: 9},
		{name: "all at corner", targetType: "ALL", x: 1, y: 1, want: 9},
		{name: "cross at corner", targetType: "CROSS", x: 1, y: 5, want: 3},
		{name: "cross near top edge", targetType: "CROSS", x: 3, y: 1, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cells := combatAreaCells(tt.targetType, tt.x, tt.y)
			if len(cells) != tt.want {
				t.Fatalf("combatAreaCells(%q, %d, %d) returned %d cells, want %d", tt.targetType, tt.x, tt.y, len(cells), tt.want)
			}
			if _, ok := cells[combatGridCell{x: tt.x, y: tt.y}]; !ok {
				t.Fatalf("combatAreaCells(%q, %d, %d) omitted the selected cell", tt.targetType, tt.x, tt.y)
			}
		})
	}
}

func TestResolveTargetsUsesFiveByFiveAreas(t *testing.T) {
	enemies := make([]*CombatantAuto, 0, 25)
	for x := 1; x <= 5; x++ {
		for y := 1; y <= 5; y++ {
			enemies = append(enemies, &CombatantAuto{ID: "target", HP: 100, GridX: x, GridY: y})
		}
	}

	for _, tt := range []struct {
		targetType string
		want       int
	}{{"ROW", 3}, {"COL", 3}, {"CROSS", 3}, {"ALL", 9}} {
		got := resolveTargets(enemies, &SimAction{TargetType: tt.targetType, TargetX: 3, TargetY: 3})
		if len(got) != tt.want {
			t.Errorf("%s selected %d combatants, want %d", tt.targetType, len(got), tt.want)
		}
	}
}
