package handlers

import "testing"

func TestShouldRecordCombatHistory(t *testing.T) {
	cases := []struct {
		enemyType string
		winner    string
		want      bool
	}{
		{"PVP", "Phe Ta", true},
		{"PVP", "Phe Địch", true},
		{"MONSTER", "Phe Địch", true},
		{"BOSS", "Phe Địch", true},
		// Boss fights are recorded even on victory (world boss / tribulation).
		{"BOSS", "Phe Ta", true},
		{"Boss", "Phe Ta", true},
		{"World_Boss", "Phe Ta", true},
		{"TRIBULATION_BOSS", "Phe Ta", true},
		{"boss", "Phe Ta", true},
		{"MONSTER", "Phe Ta", false},
		{"ELITE", "Phe Ta", false},
		{"TRIBULATION_MONSTER", "Phe Ta", false},
		// Sect wars are recorded on both outcomes, for attacker and defenders.
		{"SECT_WAR", "Phe Ta", true},
		{"SECT_WAR", "Phe Địch", true},
	}
	for _, tc := range cases {
		if got := shouldRecordCombatHistory(tc.enemyType, tc.winner); got != tc.want {
			t.Errorf("shouldRecordCombatHistory(%q, %q) = %v, want %v", tc.enemyType, tc.winner, got, tc.want)
		}
	}
}
