package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestPracticeCombatDoesNotAwardArenaCoins(t *testing.T) {
	for _, alliesAlive := range []bool{true, false} {
		allyHP, enemyHP := 0.0, 0.0
		if alliesAlive {
			allyHP = 10
		} else {
			enemyHP = 10
		}
		state := &simulator.CombatState{
			Allies:  []*simulator.CombatantAuto{{ID: "player", HP: allyHP}},
			Enemies: []*simulator.CombatantAuto{{ID: "dummy", HP: enemyHP}},
		}
		outcome := EvaluateCombatOutcome(CombatOutcomeInput{
			SimState: state, PlayerID: "player", ActiveCombat: &models.CombatState{
				EnemyType: "PRACTICE", CombatMode: "virtual", EnemyName: "Hình nhân",
			}, IsVirtual: true,
		}, nil)
		if outcome.ArenaCoinsAwarded != 0 {
			t.Fatalf("practice alliesAlive=%v awarded %d arena coins", alliesAlive, outcome.ArenaCoinsAwarded)
		}
		if _, exists := outcome.Summary["reward"]; exists {
			t.Fatalf("practice alliesAlive=%v has arena reward summary: %#v", alliesAlive, outcome.Summary)
		}
	}
}

func TestWorldMonsterCombatUsesOnlySpawnedEnemies(t *testing.T) {
	worldEncounter := &models.CombatState{WorldMonsterEncounterID: "zone:event", EnemyHP: 100, EnemyTemplateID: "wolf"}
	if shouldSpawnCombatMinions(worldEncounter) {
		t.Fatal("world monster encounter should not synthesize random guards")
	}
	legacyEncounter := &models.CombatState{EnemyHP: 100, EnemyTemplateID: "wolf"}
	if !shouldSpawnCombatMinions(legacyEncounter) {
		t.Fatal("legacy exploration combat should preserve its configured guard behavior")
	}
}

func TestCombatOwnerAllowsOnlyInitiatingPlayer(t *testing.T) {
	state := &models.CombatState{CombatOwnerID: "leader"}
	if !combatOwnedByPlayer(state, "leader") {
		t.Fatal("combat owner should be allowed to resolve the encounter")
	}
	if combatOwnedByPlayer(state, "ally") {
		t.Fatal("party ally must not resolve the leader-owned encounter")
	}
	if !combatOwnedByPlayer(&models.CombatState{}, "legacy") {
		t.Fatal("legacy combat without owner metadata should remain compatible")
	}
}
