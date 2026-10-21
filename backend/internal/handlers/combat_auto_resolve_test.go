package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

// combatTypeLabel is a display label, not an enemy type. Feeding it to
// shouldRecordCombatHistory or SaveCombatHistory silently drops PVP from combat
// history, because only the raw EnemyType equals "PVP". ResolveCombat depends on
// keeping the two apart.
func TestCombatTypeLabelIsNotARawEnemyType(t *testing.T) {
	if got := combatTypeLabel(&models.CombatState{EnemyType: "PVP"}); got == "PVP" {
		t.Fatalf("combatTypeLabel returned the raw enemy type %q", got)
	}
	if !shouldRecordCombatHistory("PVP", "Phe Ta", false) {
		t.Fatal("PVP wins must be recorded; history decisions need the raw enemy type")
	}
}

// ResolveCombat relies on "alliesAlive implies the player won": that is what
// lets the sect-war branch assume the attacker always won and skip the
// defender-victory case (which applyAutoCombatDefeat handles instead). If
// combatOutcome ever stops implying that, this test fails and the assumption
// must be revisited.
func TestCombatOutcomeAlliesAliveMeansPlayerWon(t *testing.T) {
	sim := &simulator.CombatState{
		Allies:  []*simulator.CombatantAuto{{ID: "p1", HP: 10}},
		Enemies: []*simulator.CombatantAuto{{ID: "e1", HP: 0}},
	}

	enemiesAlive, alliesAlive, finalHP := combatOutcome(sim, "p1")
	if enemiesAlive {
		t.Fatal("expected no enemies alive")
	}
	if !alliesAlive {
		t.Fatal("expected alliesAlive when every enemy is down")
	}
	if finalHP != 10 {
		t.Fatalf("finalPlayerHP = %v, want 10", finalHP)
	}

	// A live enemy must force alliesAlive to false.
	sim.Enemies[0].HP = 1
	enemiesAlive, alliesAlive, _ = combatOutcome(sim, "p1")
	if !enemiesAlive || alliesAlive {
		t.Fatalf("with a live enemy: enemiesAlive=%v alliesAlive=%v, want true/false", enemiesAlive, alliesAlive)
	}
}
