package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestVirtualPracticeAndTowerIgnoreWorldCombatRegionPolicy(t *testing.T) {
	for _, mode := range []string{"PRACTICE", "TOWER"} {
		if requiresRegionCombatPolicy(&models.CombatState{EnemyType: mode}) {
			t.Errorf("%s should be playable independently of the player's world region", mode)
		}
	}
}

func TestWorldCombatAndVirtualArenaKeepRegionPolicy(t *testing.T) {
	for _, mode := range []string{"MONSTER", "PVP"} {
		if !requiresRegionCombatPolicy(&models.CombatState{EnemyType: mode}) {
			t.Errorf("%s should keep world-region combat policy", mode)
		}
	}
}

func TestPvPCombatDoesNotJoinInitiatingParty(t *testing.T) {
	if partyJoinsCombat(&models.CombatState{EnemyType: "PVP", CombatMode: "real"}) {
		t.Fatal("world PvP must stay one player plus that player's pets/NPCs")
	}
	if !partyJoinsCombat(&models.CombatState{EnemyType: "MONSTER", CombatMode: "real"}) {
		t.Fatal("real world PvE should continue to include eligible party members")
	}
}

func TestCombatSessionWinnerTracksWhichSideFled(t *testing.T) {
	state := &simulator.CombatState{
		Allies:  []*simulator.CombatantAuto{{ID: "attacker", HP: 80, MaxHP: 100}},
		Enemies: []*simulator.CombatantAuto{{ID: "defender", HP: 100, MaxHP: 100}},
	}
	state.FledBy = "attacker"
	if got := combatSessionWinner(state); got != "Phe Địch" {
		t.Fatalf("attacker fled: winner = %q, want Phe Địch", got)
	}
	state.FledBy = "defender"
	if got := combatSessionWinner(state); got != "Phe Ta" {
		t.Fatalf("defender fled: winner = %q, want Phe Ta", got)
	}
}
