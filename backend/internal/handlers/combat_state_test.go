package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestPvPPreparationOpponentIDResolvesAttackerAndDefender(t *testing.T) {
	attackerID := uuid.MustParse("10000000-0000-0000-0000-000000000001")
	defenderID := uuid.MustParse("20000000-0000-0000-0000-000000000002")
	tests := []struct {
		name   string
		state  models.CombatState
		viewer uuid.UUID
		want   uuid.UUID
	}{
		{name: "attacker mirror", state: models.CombatState{EnemyType: "PVP", CombatMode: "real", CombatOwnerID: attackerID.String(), TargetPlayerID: defenderID.String()}, viewer: attackerID, want: defenderID},
		{name: "defender mirror", state: models.CombatState{EnemyType: "PVP", CombatMode: "real", CombatOwnerID: attackerID.String(), TargetPlayerID: defenderID.String()}, viewer: defenderID, want: attackerID},
		{name: "legacy defender mirror", state: models.CombatState{EnemyType: "PVP", CombatMode: "real", CombatOwnerID: attackerID.String(), TargetPlayerID: attackerID.String()}, viewer: defenderID, want: attackerID},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := pvpPreparationOpponentID(&test.state, test.viewer)
			if !ok || got != test.want {
				t.Fatalf("pvpPreparationOpponentID() = (%s, %v), want (%s, true)", got, ok, test.want)
			}
		})
	}
}
