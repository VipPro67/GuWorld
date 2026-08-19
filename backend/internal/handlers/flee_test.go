package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestFleeFreeRetreatNilCombat(t *testing.T) {
	if fleeFreeRetreat(nil) {
		t.Fatal("expected nil combat state to not allow free retreat")
	}
}

func TestFleeFreeRetreatMine(t *testing.T) {
	cs := &models.CombatState{EnemyType: "MINE_PVE", MineSlotID: 7}
	if !fleeFreeRetreat(cs) {
		t.Fatal("expected mine combat to allow free retreat")
	}
}

func TestFleeFreeRetreatPVP(t *testing.T) {
	cs := &models.CombatState{EnemyType: "PVP", TargetPlayerID: "some-player"}
	if !fleeFreeRetreat(cs) {
		t.Fatal("expected PvP combat to allow free retreat")
	}
}

func TestFleeFreeRetreatNormalPvE(t *testing.T) {
	for _, et := range []string{"", "BOSS", "TRIBULATION_BOSS", "TRIBULATION_MONSTER"} {
		cs := &models.CombatState{EnemyType: et}
		if fleeFreeRetreat(cs) {
			t.Fatalf("expected combat type %q to NOT allow free retreat", et)
		}
	}
}
