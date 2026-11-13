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

func TestInfiniteTowerDoesNotAwardArenaCoinsThroughArenaSettlement(t *testing.T) {
	for _, alliesAlive := range []bool{true, false} {
		state := &simulator.CombatState{Allies: []*simulator.CombatantAuto{{ID: "player", HP: 10, MaxHP: 10}}}
		if !alliesAlive {
			state.Allies[0].HP = 0
		}
		outcome := EvaluateCombatOutcome(CombatOutcomeInput{
			SimState: state, PlayerID: "player", IsVirtual: true,
			ActiveCombat: &models.CombatState{EnemyType: "TOWER"},
		}, nil)
		if outcome.ArenaCoinsAwarded != 0 {
			t.Fatalf("tower alliesAlive=%v awarded %d arena coins through arena settlement", alliesAlive, outcome.ArenaCoinsAwarded)
		}
		if outcome.HumanQiReward != 0 {
			t.Fatalf("tower alliesAlive=%v awarded %.1f Human Qi", alliesAlive, outcome.HumanQiReward)
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

func TestCombatOutcomeAwardsWinToOpponentWhenActorFlees(t *testing.T) {
	t.Run("player flee", func(t *testing.T) {
		state := &simulator.CombatState{
			Allies:  []*simulator.CombatantAuto{{ID: "player", HP: 40}},
			Enemies: []*simulator.CombatantAuto{{ID: "enemy", HP: 60}},
			FledBy:  "player",
		}
		outcome := EvaluateCombatOutcome(CombatOutcomeInput{SimState: state, PlayerID: "player"}, nil)
		if outcome.Winner != "Phe Địch" || outcome.AlliesAlive || outcome.FinalPlayerHP != 40 {
			t.Fatalf("player flee outcome = %+v", outcome)
		}
	})

	t.Run("opponent flee", func(t *testing.T) {
		state := &simulator.CombatState{
			Allies:  []*simulator.CombatantAuto{{ID: "player", HP: 40}},
			Enemies: []*simulator.CombatantAuto{{ID: "enemy", HP: 60}},
			FledBy:  "enemy",
		}
		outcome := EvaluateCombatOutcome(CombatOutcomeInput{SimState: state, PlayerID: "player"}, nil)
		if outcome.Winner != "Phe Ta" || !outcome.AlliesAlive {
			t.Fatalf("opponent flee outcome = %+v", outcome)
		}
	})
}

func TestLivePvpWinDoesNotClearWorldCell(t *testing.T) {
	state := &simulator.CombatState{
		Allies:  []*simulator.CombatantAuto{{ID: "player", HP: 100}},
		Enemies: []*simulator.CombatantAuto{{ID: "opponent", HP: 0}},
	}
	outcome := EvaluateCombatOutcome(CombatOutcomeInput{
		SimState: state, PlayerID: "player",
		ActiveCombat: &models.CombatState{EnemyType: "PVP", CombatMode: "real"},
		PlayerX:      4, PlayerY: 7,
	}, nil)
	if !outcome.AlliesAlive || outcome.ShouldClearCell {
		t.Fatalf("PvP victory should preserve map cell state, got %+v", outcome)
	}
}
