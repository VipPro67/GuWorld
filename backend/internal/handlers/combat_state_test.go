package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestPartyJoinsCombat(t *testing.T) {
	cases := []struct {
		name string
		cs   *models.CombatState
		want bool
	}{
		{"arena virtual", &models.CombatState{EnemyType: "PVP", CombatMode: "virtual"}, false},
		{"open-world pvp", &models.CombatState{EnemyType: "PVP", CombatMode: "real"}, true},
		{"pve normal", &models.CombatState{EnemyType: "Elite", CombatMode: "real"}, true},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		if got := partyJoinsCombat(tc.cs); got != tc.want {
			t.Fatalf("%s: partyJoinsCombat = %v, want %v", tc.name, got, tc.want)
		}
	}
}
