package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestCombatHungerDeductionByInstanceID(t *testing.T) {
	setupGuRegistry(t)

	// Player has 2 equipped ACTIVE Gu with the same GuID "GU_A" but different InstanceIDs
	p := models.Player{}
	p.ID = uuid.New()
	p.Username = "Tester"
	p.Rank = 1
	p.BaseHP = 500
	p.HP = 500
	p.MaxMana = 100
	p.CurrentMana = 100
	p.GuInventory = models.GuInventory{
		{InstanceID: "inst_a_1", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 50},
		{InstanceID: "inst_a_2", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 80},
	}

	// 1. Test buildSimCombatant selecting only inst_a_1
	loadout := &simulator.CombatLoadout{
		GuIds: []string{"inst_a_1"},
	}
	simPlayer, err := buildSimCombatant(nil, &p, false, true, loadout)
	if err != nil {
		t.Fatalf("buildSimCombatant failed: %v", err)
	}

	// Verify SimAction has InstanceID = "inst_a_1" and SourceID = "GU_A"
	var foundGuAction bool
	for _, act := range simPlayer.Actions {
		if act.ActionType == "GU_SKILL" {
			foundGuAction = true
			if act.InstanceID != "inst_a_1" {
				t.Fatalf("expected Action.InstanceID to be 'inst_a_1', got '%s'", act.InstanceID)
			}
			if act.SourceID != "GU_A" {
				t.Fatalf("expected Action.SourceID to be 'GU_A', got '%s'", act.SourceID)
			}
		}
	}
	if !foundGuAction {
		t.Fatalf("expected to find GU_SKILL action for inst_a_1")
	}

	// Verify GuHunger map is keyed by instance_id
	if simPlayer.GuHunger["inst_a_1"] != 50 {
		t.Fatalf("expected GuHunger[inst_a_1] = 50, got %d", simPlayer.GuHunger["inst_a_1"])
	}
	if simPlayer.GuHunger["inst_a_2"] != 80 {
		t.Fatalf("expected GuHunger[inst_a_2] = 80, got %d", simPlayer.GuHunger["inst_a_2"])
	}

	// Simulate battle where inst_a_1 was used 3 times
	simPlayer.GuUsedCounts = map[string]int{
		"inst_a_1": 3,
	}

	// 2. Test applySimConsumption deducts hunger from inst_a_1 ONLY
	applySimConsumption(nil, &p, simPlayer, false, 5)

	if p.GuInventory[0].Hunger != 47 {
		t.Fatalf("expected inst_a_1 hunger to be 47 (50 - 3), got %d", p.GuInventory[0].Hunger)
	}
	if p.GuInventory[1].Hunger != 80 {
		t.Fatalf("expected inst_a_2 hunger to remain 80, got %d", p.GuInventory[1].Hunger)
	}
}

func TestCombatHungerDeductionFromCombatSettings(t *testing.T) {
	setupGuRegistry(t)

	// Player has 2 instances of "GU_A" (inst_a_1 rank 2, inst_a_2 rank 1) and 1 instance of "GU_B" (inst_b_1)
	p := models.Player{}
	p.ID = uuid.New()
	p.Username = "Tester"
	p.Rank = 2
	p.BaseHP = 500
	p.HP = 500
	p.MaxMana = 100
	p.CurrentMana = 100
	p.GuInventory = models.GuInventory{
		{InstanceID: "inst_a_1", GuID: "GU_A", Rank: 2, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "inst_a_2", GuID: "GU_A", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
		{InstanceID: "inst_b_1", GuID: "GU_B", Rank: 1, Status: "ACTIVE", Equipped: true, Hunger: 100},
	}
	p.CombatSettings = models.AutoCombatConfig{
		GuPriorities: []models.GuTactic{
			{GuID: "GU_A", Priority: 1, UseCondition: "always", ConditionValue: 30},
			{GuID: "GU_B", Priority: 2, UseCondition: "always", ConditionValue: 30},
		},
	}

	// Case: Player does NOT send loadout (reqLoadout == nil) -> loaded from CombatSettings.GuPriorities
	simPlayer, err := buildSimCombatant(nil, &p, false, true, nil)
	if err != nil {
		t.Fatalf("buildSimCombatant failed: %v", err)
	}

	// Verify that GU_A resolved to highest rank equipped instance "inst_a_1"
	var guAAction *simulator.SimAction
	var guBAction *simulator.SimAction
	for i := range simPlayer.Actions {
		if simPlayer.Actions[i].SourceID == "GU_A" {
			guAAction = &simPlayer.Actions[i]
		}
		if simPlayer.Actions[i].SourceID == "GU_B" {
			guBAction = &simPlayer.Actions[i]
		}
	}

	if guAAction == nil || guAAction.InstanceID != "inst_a_1" {
		t.Fatalf("expected GU_A action to have InstanceID = 'inst_a_1', got %+v", guAAction)
	}
	if guBAction == nil || guBAction.InstanceID != "inst_b_1" {
		t.Fatalf("expected GU_B action to have InstanceID = 'inst_b_1', got %+v", guBAction)
	}

	// Verify tactics priorities applied correctly
	if guAAction.Priority != 1 || guAAction.UseCondition != "always" {
		t.Fatalf("expected GU_A priority=1 always, got priority=%d condition=%s", guAAction.Priority, guAAction.UseCondition)
	}
	if guBAction.Priority != 2 || guBAction.UseCondition != "always" {
		t.Fatalf("expected GU_B priority=2 always, got priority=%d condition=%s", guBAction.Priority, guBAction.UseCondition)
	}

	// Simulate combat usage: inst_a_1 used 4 times, inst_b_1 used 2 times
	simPlayer.GuUsedCounts = map[string]int{
		"inst_a_1": 4,
		"inst_b_1": 2,
	}

	applySimConsumption(nil, &p, simPlayer, false, 5)

	// inst_a_1 should be 96 (100 - 4)
	if p.GuInventory[0].Hunger != 96 {
		t.Fatalf("expected inst_a_1 hunger 96, got %d", p.GuInventory[0].Hunger)
	}
	// inst_a_2 (not selected into battle) should remain 100
	if p.GuInventory[1].Hunger != 100 {
		t.Fatalf("expected inst_a_2 hunger to remain 100, got %d", p.GuInventory[1].Hunger)
	}
	// inst_b_1 should be 98 (100 - 2)
	if p.GuInventory[2].Hunger != 98 {
		t.Fatalf("expected inst_b_1 hunger 98, got %d", p.GuInventory[2].Hunger)
	}
}
