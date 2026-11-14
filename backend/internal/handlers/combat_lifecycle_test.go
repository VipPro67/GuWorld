package handlers

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestCombatLifecycleRegionPolicy(t *testing.T) {
	if requiresRegionCombatPolicy(&models.CombatState{EnemyType: "PRACTICE"}) {
		t.Error("PRACTICE should bypass region combat policy")
	}
	if requiresRegionCombatPolicy(&models.CombatState{EnemyType: "TOWER"}) {
		t.Error("TOWER should bypass region combat policy")
	}
	if !requiresRegionCombatPolicy(&models.CombatState{EnemyType: "MONSTER"}) {
		t.Error("MONSTER must require region combat policy")
	}
	if !requiresRegionCombatPolicy(&models.CombatState{EnemyType: "PVP"}) {
		t.Error("PVP must require region combat policy")
	}
}

func TestCombatLifecycleStatusStatFactor(t *testing.T) {
	player := &models.Player{}
	if factor := statusStatFactor(player, "attack"); factor != 1.0 {
		t.Errorf("expected empty status factor 1.0, got %f", factor)
	}
	if factor := statusStatFactor(nil, "attack"); factor != 1.0 {
		t.Errorf("expected nil player factor 1.0, got %f", factor)
	}
}

func TestCombatLifecycleEffectiveCombatSpeed(t *testing.T) {
	state := &models.CombatState{
		Allies: []models.Combatant{
			{
				ID:    "ally-1",
				Speed: 100,
				StatusEffects: []models.StatusEffect{
					{Type: "slow", Value: 20, Turns: 2},
				},
			},
		},
	}
	speed := effectiveCombatSpeed(state, "ally-1")
	if speed != 80 {
		t.Fatalf("expected speed 80 under 20%% slow, got %f", speed)
	}
}

func TestCombatLifecycleApplyDamageWithLogs(t *testing.T) {
	target := &models.Combatant{
		Kind:          "Player",
		Rank:          3,
		HP:            100,
		VirtualShield: 30,
	}
	log := applyDamageWithLogs("Attacker", "Defender", target, 50)
	if target.VirtualShield != 0 {
		t.Fatalf("expected shield 0, got %f", target.VirtualShield)
	}
	if target.HP != 80 {
		t.Fatalf("expected HP 80, got %f", target.HP)
	}
	if log == "" {
		t.Fatal("expected descriptive combat log")
	}
}

func TestCombatLifecycleSettleRequiresTransaction(t *testing.T) {
	player := &models.Player{
		ActiveCombat: &models.CombatState{
			CombatMode: "virtual",
			EnemyType:  "PRACTICE",
		},
	}
	player.ID = uuid.New()
	_, err := SettleCombatSession(nil, player, nil, nil, nil, nil)
	if err == nil || err.Error() != "transaction is required for combat settlement" {
		t.Fatalf("expected 'transaction is required for combat settlement' error, got %v", err)
	}
}

func TestCombatLifecycleSettleRequiresActiveCombat(t *testing.T) {
	mockTx := &sql.Tx{}
	player := &models.Player{}
	_, err := SettleCombatSession(mockTx, player, nil, nil, nil, nil)
	if err == nil || err.Error() != "no active combat" {
		t.Fatalf("expected 'no active combat' error, got %v", err)
	}
}

func TestCombatLifecycleSettleEnforcesPartyOwner(t *testing.T) {
	mockTx := &sql.Tx{}
	player := &models.Player{
		ActiveCombat: &models.CombatState{
			CombatOwnerID: "other-leader-id",
		},
	}
	player.ID = uuid.New()
	_, err := SettleCombatSession(mockTx, player, nil, nil, nil, nil)
	if err == nil || err.Error() != "party combat is controlled by its initiating player" {
		t.Fatalf("expected party combat owner error, got %v", err)
	}
}

