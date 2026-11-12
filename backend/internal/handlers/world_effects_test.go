package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

func TestWorldUseAdapterAppliesSelfHPAndManaCommands(t *testing.T) {
	player := models.Player{}
	player.HP = 60
	player.BaseHP = 100
	player.CurrentMana = 80
	player.MaxMana = 100
	blocks := []effects.Block{{
		ID: "USE_RECOVERY", Trigger: effects.TriggerOnUse, Target: effects.Target{Selector: effects.TargetSelf},
		Operations: []effects.Operation{
			{Op: effects.OpHeal, SubType: "PERCENT", Value: 25},
			{Op: effects.OpHeal, SubType: "RESOURCE", Resource: "MANA", Value: 40},
		},
	}}
	compiled, err := (effects.Compiler{}).Compile(blocks)
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(effects.Context{}, effects.Event{Trigger: effects.TriggerOnUse})
	if err != nil {
		t.Fatal(err)
	}
	healed, restoredMana, _, _, _, err := applyWorldUseCommands(&player, result.Commands, 6, "test")
	if err != nil {
		t.Fatal(err)
	}
	if healed != 25 || player.HP != 85 {
		t.Fatalf("HP result: healed=%v HP=%v, want 25 and 85", healed, player.HP)
	}
	if restoredMana != 20 || player.CurrentMana != 100 {
		t.Fatalf("mana result: restored=%v mana=%v, want capped at 100", restoredMana, player.CurrentMana)
	}
}

func TestWorldUseAdapterRejectsUnsupportedCommand(t *testing.T) {
	player := models.Player{}
	_, _, _, _, _, err := applyWorldUseCommands(&player, []effects.Command{{
		Domain: effects.DomainCombat, Operation: effects.OpDamage, Target: effects.TargetSelf, Value: 10,
	}}, 6, "test")
	if err == nil {
		t.Fatal("unsupported ON_USE damage command accepted")
	}
}

func TestWorldUseAdapterAppliesWorldFlightStatus(t *testing.T) {
	player := models.Player{}
	compiled, err := (effects.Compiler{}).Compile([]effects.Block{{
		ID: "WORLD_FLIGHT", Trigger: effects.TriggerOnUse, Target: effects.Target{Selector: effects.TargetSelf},
		Operations: []effects.Operation{{Op: effects.OpStatus, StatusID: statusfx.StatusFlying, Turns: 6, Chance: 1}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(effects.Context{}, effects.Event{Trigger: effects.TriggerOnUse})
	if err != nil {
		t.Fatal(err)
	}
	_, _, durationMinutes, _, _, err := applyWorldUseCommands(&player, result.Commands, 6, "world-flight")
	if err != nil {
		t.Fatal(err)
	}
	if durationMinutes != 6 || !player.Statuses.Has(StatusFlying) || !PlayerIgnoresMonsters(&player) {
		t.Fatalf("flight adapter result: duration=%d statuses=%+v; expected 6 minute world flight", durationMinutes, player.Statuses)
	}
}

func TestWorldUseAdapterAppliesWorldStealthAsConcealment(t *testing.T) {
	player := models.Player{}
	player.Rank = 5
	block := effects.Block{
		ID: "WORLD_STEALTH", Trigger: effects.TriggerOnUse, Target: effects.Target{Selector: effects.TargetSelf},
		Operations: []effects.Operation{{Op: effects.OpStatus, StatusID: statusfx.StatusStealth, Turns: 12, Value: 20, Chance: 1}},
	}
	if !worldUseOperationAllowed(block.Operations[0], block.Target) {
		t.Fatal("self STEALTH should be available to the ON_USE world executor")
	}
	compiled, err := (effects.Compiler{}).Compile([]effects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(effects.Context{}, effects.Event{Trigger: effects.TriggerOnUse})
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, concealment, _, err := applyWorldUseCommands(&player, result.Commands, 5, "world-stealth")
	if err != nil {
		t.Fatal(err)
	}
	if concealment != 20 || !player.Statuses.Has(statusfx.StatusStealth) || PlayerDisguiseLevel(&player) != 25 {
		t.Fatalf("stealth result: concealment=%d disguise=%v statuses=%+v", concealment, PlayerDisguiseLevel(&player), player.Statuses)
	}
	block.Operations[0].Turns = 24*60 + 1
	if worldUseOperationAllowed(block.Operations[0], block.Target) {
		t.Fatal("world status duration beyond 24 hours was accepted")
	}
}

func TestWorldUseAdapterAppliesWorldHasteToPlayerSpeed(t *testing.T) {
	player := models.Player{}
	player.Rank = 1
	player.SubRank = 1
	block := effects.Block{
		ID: "WORLD_HASTE", Trigger: effects.TriggerOnUse, Target: effects.Target{Selector: effects.TargetSelf},
		Operations: []effects.Operation{{Op: effects.OpStatus, StatusID: statusfx.StatusHaste, Turns: 9, Value: 12, Chance: 1}},
	}
	if !worldUseOperationAllowed(block.Operations[0], block.Target) {
		t.Fatal("self HASTE should be available to the ON_USE world executor")
	}
	compiled, err := (effects.Compiler{}).Compile([]effects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(effects.Context{}, effects.Event{Trigger: effects.TriggerOnUse})
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, _, durationMinutes, err := applyWorldUseCommands(&player, result.Commands, 5, "world-haste")
	if err != nil {
		t.Fatal(err)
	}
	if durationMinutes != 9 || !player.Statuses.Has(statusfx.StatusHaste) {
		t.Fatalf("haste duration=%d statuses=%+v; expected a 9-minute HASTE", durationMinutes, player.Statuses)
	}
	if got := CalculatePlayerStats(&player)["speed"]; got != 11.2 {
		t.Fatalf("speed after HASTE=%v, want base 10 increased by 12%%", got)
	}
}

func TestWorldUseEvaluationHonorsConditions(t *testing.T) {
	compiled, err := (effects.Compiler{}).Compile([]effects.Block{{
		ID: "CONDITIONAL_USE", Trigger: effects.TriggerOnUse, Target: effects.Target{Selector: effects.TargetSelf},
		Conditions: []effects.Condition{{Field: "HP_PERCENT", Operator: "LT", Value: 0.4}},
		Operations: []effects.Operation{{Op: effects.OpHeal, SubType: "FLAT", Value: 20}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiled.Evaluate(effects.Context{Actor: effects.Snapshot{HP: 50, MaxHP: 100}}, effects.Event{Trigger: effects.TriggerOnUse})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Commands) != 0 {
		t.Fatalf("condition should suppress ON_USE command, got %+v", result.Commands)
	}
}

func TestWorldUseAdapterPreflightsBeforeMutating(t *testing.T) {
	player := models.Player{}
	player.HP = 50
	player.BaseHP = 100
	commands := []effects.Command{
		{Domain: effects.DomainCombat, Operation: effects.OpHeal, SubType: "FLAT", Target: effects.TargetSelf, Value: 20},
		{Domain: effects.DomainCombat, Operation: effects.OpDamage, Target: effects.TargetSelf, Value: 10},
	}
	if _, _, _, _, _, err := applyWorldUseCommands(&player, commands, 5, "test"); err == nil {
		t.Fatal("unsupported command should reject the whole effect result")
	}
	if player.HP != 50 {
		t.Fatalf("valid first command mutated HP before later command failed: %.0f", player.HP)
	}
}

func TestWorldUseOperationPolicyMatchesRuntimeChanceSupport(t *testing.T) {
	op := effects.Operation{Op: effects.OpStatus, StatusID: statusfx.StatusHaste, Turns: 5, Value: 10, Chance: 0.5}
	if worldUseOperationAllowed(op, effects.Target{Selector: effects.TargetSelf}) {
		t.Fatal("fractional status chance passed the builder policy but is unsupported by the runtime adapter")
	}
}

func TestWorldUseOperationPolicyRejectsZeroValueRecovery(t *testing.T) {
	target := effects.Target{Selector: effects.TargetSelf}
	for _, subtype := range []string{"FLAT", "PERCENT", "RESOURCE"} {
		op := effects.Operation{Op: effects.OpHeal, SubType: subtype, Resource: map[string]string{"RESOURCE": "MANA"}[subtype], Value: 0}
		if worldUseOperationAllowed(op, target) {
			t.Fatalf("zero-value %s recovery passed builder policy", subtype)
		}
		op.Value = 1
		if !worldUseOperationAllowed(op, target) {
			t.Fatalf("positive %s recovery rejected by builder policy", subtype)
		}
	}
}

func TestWorldUsePolicyAndRuntimeRejectUnsupportedStatusStackPolicy(t *testing.T) {
	target := effects.Target{Selector: effects.TargetSelf}
	op := effects.Operation{Op: effects.OpStatus, StatusID: statusfx.StatusHaste, Turns: 5, Value: 10, Chance: 1, StackPolicy: "STACK"}
	if worldUseOperationAllowed(op, target) {
		t.Fatal("unsupported STATUS stack policy passed builder policy")
	}
	player := models.Player{}
	commands := []effects.Command{
		{Domain: effects.DomainCombat, Operation: effects.OpHeal, SubType: "FLAT", Target: effects.TargetSelf, Value: 20},
		{Domain: effects.DomainCombat, Operation: effects.OpStatus, Target: effects.TargetSelf, StackPolicy: "STACK", Status: &statusfx.StatusOp{ID: statusfx.StatusHaste, Turns: 5, Value: 10, Chance: 1}},
	}
	if _, _, _, _, _, err := applyWorldUseCommands(&player, commands, 5, "test"); err == nil {
		t.Fatal("runtime adapter accepted an unsupported status stack policy")
	}
	if player.HP != 0 || player.Statuses.Has(statusfx.StatusHaste) {
		t.Fatalf("preflight partially applied unsupported status: HP=%.0f statuses=%+v", player.HP, player.Statuses)
	}
}
