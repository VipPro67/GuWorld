package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestEvaluateCombatOutcome_PveVictory(t *testing.T) {
	sim := &simulator.CombatState{
		Turn:    3,
		Allies:  []*simulator.CombatantAuto{{ID: "player-1", Name: "Player", HP: 85, Mana: 40}},
		Enemies: []*simulator.CombatantAuto{{ID: "enemy-1", Name: "Wild Wolf", HP: 0}},
	}
	active := &models.CombatState{
		EnemyName:       "Wild Wolf",
		EnemyType:       "Normal",
		EnemyRank:       2,
		EnemyTemplateID: "wolf_1",
	}
	input := CombatOutcomeInput{
		SimState:      sim,
		PlayerID:      "player-1",
		PlayerX:       120,
		PlayerY:       250,
		CurrentRegion: "Southern_Border",
		ActiveCombat:  active,
		IsVirtual:     false,
		IsTribulation: false,
	}

	outcome := EvaluateCombatOutcome(input, nil)

	if outcome.Winner != "Phe Ta" {
		t.Fatalf("expected Winner 'Phe Ta', got %q", outcome.Winner)
	}
	if !outcome.AlliesAlive {
		t.Fatal("expected AlliesAlive to be true")
	}
	if outcome.FinalPlayerHP != 85 {
		t.Fatalf("expected FinalPlayerHP 85, got %v", outcome.FinalPlayerHP)
	}
	if outcome.FinalPlayerMana != 40 {
		t.Fatalf("expected FinalPlayerMana 40, got %v", outcome.FinalPlayerMana)
	}
	if !outcome.ShouldClearCell {
		t.Fatal("expected ShouldClearCell to be true for PVE win")
	}
	if outcome.ClearedCellKey != "120,250" {
		t.Fatalf("expected ClearedCellKey '120,250', got %q", outcome.ClearedCellKey)
	}
	expectedQi := combatHumanQiReward(2)
	if outcome.HumanQiReward != expectedQi {
		t.Fatalf("expected HumanQiReward %v, got %v", expectedQi, outcome.HumanQiReward)
	}
}

func TestEvaluateCombatOutcome_TribulationDoesNotClearCell(t *testing.T) {
	sim := &simulator.CombatState{
		Turn:    5,
		Allies:  []*simulator.CombatantAuto{{ID: "player-1", Name: "Immortal", HP: 500, Mana: 200}},
		Enemies: []*simulator.CombatantAuto{{ID: "trib-1", Name: "Lightning", HP: 0}},
	}
	active := &models.CombatState{
		EnemyName:          "Heavenly Tribulation",
		EnemyType:          "TRIBULATION",
		TribulationEventID: "event-123",
	}
	input := CombatOutcomeInput{
		SimState:      sim,
		PlayerID:      "player-1",
		PlayerX:       10,
		PlayerY:       10,
		ActiveCombat:  active,
		IsVirtual:     false,
		IsTribulation: true,
	}

	outcome := EvaluateCombatOutcome(input, nil)

	if outcome.Winner != "Phe Ta" {
		t.Fatalf("expected Winner 'Phe Ta', got %q", outcome.Winner)
	}
	if outcome.ShouldClearCell {
		t.Fatal("tribulation victory must never clear map cells")
	}
}

func TestEvaluateCombatOutcome_ArenaVirtualModes(t *testing.T) {
	t.Run("arena victory", func(t *testing.T) {
		sim := &simulator.CombatState{
			Turn:    4,
			Allies:  []*simulator.CombatantAuto{{ID: "p1", Name: "Player 1", HP: 100, Mana: 50}},
			Enemies: []*simulator.CombatantAuto{{ID: "p2", Name: "Opponent", HP: 0}},
		}
		active := &models.CombatState{
			EnemyName:      "Opponent",
			EnemyType:      "PVP",
			CombatMode:     "virtual",
			TargetPlayerID: "target-uuid-999",
		}
		input := CombatOutcomeInput{
			SimState:     sim,
			PlayerID:     "p1",
			ActiveCombat: active,
			IsVirtual:    true,
		}

		outcome := EvaluateCombatOutcome(input, nil)

		if outcome.Winner != "Phe Ta" {
			t.Fatalf("expected Winner 'Phe Ta', got %q", outcome.Winner)
		}
		if outcome.ArenaCoinsAwarded != 50 {
			t.Fatalf("expected 50 arena coins on win, got %d", outcome.ArenaCoinsAwarded)
		}
		if !outcome.ArenaSwapEligible {
			t.Fatal("expected ArenaSwapEligible to be true")
		}
		if outcome.ArenaSwapTargetID != "target-uuid-999" {
			t.Fatalf("expected swap target 'target-uuid-999', got %q", outcome.ArenaSwapTargetID)
		}
		if outcome.ShouldClearCell {
			t.Fatal("virtual arena must not clear map cells")
		}
	})

	t.Run("arena defeat", func(t *testing.T) {
		sim := &simulator.CombatState{
			Turn:    2,
			Allies:  []*simulator.CombatantAuto{{ID: "p1", Name: "Player 1", HP: 0, Mana: 10}},
			Enemies: []*simulator.CombatantAuto{{ID: "p2", Name: "Opponent", HP: 150}},
		}
		active := &models.CombatState{
			EnemyName:      "Opponent",
			EnemyType:      "PVP",
			CombatMode:     "virtual",
			TargetPlayerID: "target-uuid-999",
		}
		input := CombatOutcomeInput{
			SimState:     sim,
			PlayerID:     "p1",
			ActiveCombat: active,
			IsVirtual:    true,
		}

		outcome := EvaluateCombatOutcome(input, nil)

		if outcome.Winner != "Phe Địch" {
			t.Fatalf("expected Winner 'Phe Địch', got %q", outcome.Winner)
		}
		if outcome.AlliesAlive {
			t.Fatal("expected AlliesAlive to be false")
		}
		if outcome.ArenaCoinsAwarded != 1 {
			t.Fatalf("expected 1 consolation arena coin on defeat, got %d", outcome.ArenaCoinsAwarded)
		}
		if outcome.ArenaSwapEligible {
			t.Fatal("defeat must not trigger arena rank swap")
		}
	})
}

func TestBuildFormationAndCombatTypeLogs(t *testing.T) {
	allies := []*simulator.CombatantAuto{
		{Name: "Fang Yuan", Kind: "Player"},
		{Name: "White Bear", Kind: "Pet"},
	}
	formationLogs := BuildFormationLogs(allies)
	if len(formationLogs) != 1 || formationLogs[0].Action != "FORMATION" {
		t.Fatalf("expected 1 FORMATION log, got %#v", formationLogs)
	}

	active := &models.CombatState{EnemyType: "BOSS"}
	typeLog := BuildCombatTypeLog(active)
	if typeLog.Action != "COMBAT_TYPE" {
		t.Fatalf("expected COMBAT_TYPE action, got %q", typeLog.Action)
	}
}
