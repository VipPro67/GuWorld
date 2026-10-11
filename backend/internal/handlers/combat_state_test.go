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
		{"mine pve", &models.CombatState{MineSlotID: 5}, false},
		{"mine pvp", &models.CombatState{MineSlotID: 3, EnemyType: "PVP", CombatMode: "real"}, false},
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
