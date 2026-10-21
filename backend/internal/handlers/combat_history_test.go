package handlers

import "testing"

func TestShouldRecordCombatHistory(t *testing.T) {
	cases := []struct {
		enemyType         string
		winner            string
		immortalGuDropped bool
		want              bool
	}{
		{"PVP", "Phe Ta", false, true},
		{"PVP", "Phe Địch", false, true},
		{"MONSTER", "Phe Địch", false, true},
		{"BOSS", "Phe Địch", false, true},
		// Boss fights are recorded even on victory (world boss / tribulation).
		{"BOSS", "Phe Ta", false, true},
		{"Boss", "Phe Ta", false, true},
		{"World_Boss", "Phe Ta", false, true},
		{"TRIBULATION_BOSS", "Phe Ta", false, true},
		{"boss", "Phe Ta", false, true},
		{"MONSTER", "Phe Ta", false, false},
		{"ELITE", "Phe Ta", false, false},
		{"TRIBULATION_MONSTER", "Phe Ta", false, false},
		// A PVE victory that drops a Tiên Cổ is recorded so the run can be
		// replayed, including normal and elite monsters.
		{"Normal", "Phe Ta", true, true},
		{"Elite", "Phe Ta", true, true},
		{"MONSTER", "Phe Ta", true, true},
		{"ELITE", "Phe Ta", true, true},
		{"TRIBULATION_MONSTER", "Phe Ta", true, true},
		// Sect wars are recorded on both outcomes, for attacker and defenders.
		{"SECT_WAR", "Phe Ta", false, true},
		{"SECT_WAR", "Phe Địch", false, true},
	}
	for _, tc := range cases {
		if got := shouldRecordCombatHistory(tc.enemyType, tc.winner, tc.immortalGuDropped); got != tc.want {
			t.Errorf("shouldRecordCombatHistory(%q, %q, %v) = %v, want %v", tc.enemyType, tc.winner, tc.immortalGuDropped, got, tc.want)
		}
	}
}
