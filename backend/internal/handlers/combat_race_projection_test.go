package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestBuildSimVariantNPCPreservesRace(t *testing.T) {
	combatant := &models.Combatant{ID: "npc-1", Kind: "VariantNPC", Race: "merman"}

	simulated := buildSimVariantNPC(combatant)
	if simulated.Race != "merman" {
		t.Fatalf("buildSimVariantNPC() race = %q, want merman", simulated.Race)
	}
}

func TestBuildSimEnemyCombatantPreservesRace(t *testing.T) {
	combatant := &models.Combatant{ID: "enemy-1", Kind: "Normal", Race: "rockman"}

	simulated := buildSimEnemyCombatant(nil, combatant, nil)
	if simulated.Race != "rockman" {
		t.Fatalf("buildSimEnemyCombatant() race = %q, want rockman", simulated.Race)
	}
}
