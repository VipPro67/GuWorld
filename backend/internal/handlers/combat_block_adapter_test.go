package handlers

import (
	"testing"

	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestAdaptBlockCombatOperationsPreservesTargetsAndConditions(t *testing.T) {
	actor := &models.Combatant{MaxHP: 120, BaseAttack: 45, GridX: 2, GridY: 3}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 3, Path: []string{"Fire Path"}}
	block := gameplayeffects.Block{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle, BypassFrontline: true},
		Conditions: []gameplayeffects.Condition{{Field: "TARGET_HP_PERCENT", Operator: "LTE", Value: 0.5}},
		Operations: []gameplayeffects.Operation{
			{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 24},
			{Op: gameplayeffects.OpStatus, StatusID: "STUN", Turns: 2},
		},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1.25, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 {
		t.Fatalf("expected same-block operations to share one simulator action, got %d", len(effects))
	}
	got := effects[0]
	if got.Compiled.BlockID != "ACTION" || got.SourceRank != 3 || got.Action == nil {
		t.Fatalf("operation provenance was lost: %+v", got)
	}
	if got.Action.Damage != 24 || got.Action.DamageMult != 1.25 || got.Action.TargetType != "TARGET" || !got.Action.Bypass {
		t.Fatalf("combat projection mismatch: %+v", got.Action)
	}
	if got.Action.TargetX != 2 || got.Action.TargetY != 3 || len(got.Action.OperationConditions) != 1 {
		t.Fatalf("target position or conditions were lost: %+v", got.Action)
	}
	if len(got.Action.StatusOps) != 1 || got.Action.StatusOps[0].ID != "STUN" || got.Action.StatusOps[0].Turns != 2 {
		t.Fatalf("status operation was not compiled: %+v", got.Action.StatusOps)
	}
}

func TestBuildSimActionFromGuRoutesBlockV2ThroughOperationAdapter(t *testing.T) {
	actor := &models.Combatant{Rank: 3, HP: 120, MaxHP: 120, CurrentMana: 100, GridX: 2, GridY: 3}
	spec := &models.GuDictionaryItem{
		ID: "GU_TEST", Name: "Cổ thử", Rank: 3, Type: "Active", Path: []string{"Fire Path"}, BaseManaCost: 1,
		Effects: models.GuEffects{Blocks: []gameplayeffects.Block{{
			ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
			Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
			Conditions: []gameplayeffects.Condition{{Field: "TARGET_HP_PERCENT", Operator: "LTE", Value: 0.5}},
			Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpDamage, SubType: "FLAT", Value: 24}},
		}}},
	}
	action, err := buildSimActionFromGu(actor, spec, "GU_SKILL")
	if err != nil {
		t.Fatal(err)
	}
	if len(action.OperationEffects) != 1 || action.OperationEffects[0].Action == nil {
		t.Fatalf("Block v2 action was not routed through operation adapter: %+v", action)
	}
	if action.ManaCost <= 0 || action.Damage != 24 || action.OperationEffects[0].Action.Damage != 24 || len(action.OperationEffects[0].Action.OperationConditions) != 1 {
		t.Fatalf("parent cost, damage projection, or conditions are wrong: %+v", action)
	}
}

func TestAdaptBlockCombatOperationsFailsClosedOnUnsupportedStatModifier(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpStatMod, Stat: "ATTACK", Value: 5}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction)); err == nil {
		t.Fatal("expected turn-based STAT_MOD to fail closed until it has stat semantics")
	}
}

func TestAdaptBlockCombatOperationsFailsClosedOnHealAgainstEnemy(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpHeal, SubType: "FLAT", Value: 10}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction)); err == nil {
		t.Fatal("expected healing an enemy to fail closed")
	}
}

func TestAdaptBlockCombatOperationKeepsGuaranteedHitAsMechanic(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpUnique, Code: "GUARANTEED_HIT"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || effects[0].Action.Bypass || len(effects[0].Action.Mechanics) != 1 || effects[0].Action.Mechanics[0].ActionCode != "GUARANTEED_HIT" {
		t.Fatalf("guaranteed hit mechanic was not preserved as its own behavior: %+v", effects)
	}
}

func TestAdaptBlockCombatOperationPassesThroughUniqueMechanics(t *testing.T) {
	actor := &models.Combatant{MaxHP: 100}
	spec := &models.GuDictionaryItem{ID: "GU_TEST", Name: "Cổ thử", Rank: 1}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
		ID: "ACTION", Trigger: gameplayeffects.TriggerOnAction,
		Target:     gameplayeffects.Target{Selector: gameplayeffects.TargetSingle},
		Operations: []gameplayeffects.Operation{{Op: gameplayeffects.OpUnique, Code: "CALCULATED_STRIKE"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	effects, err := adaptBlockCombatOperations(actor, spec, 1, compiled.Operations(gameplayeffects.TriggerOnAction))
	if err != nil {
		t.Fatal(err)
	}
	if len(effects) != 1 || effects[0].Action == nil || len(effects[0].Action.Mechanics) != 1 || effects[0].Action.Mechanics[0].ActionCode != "CALCULATED_STRIKE" {
		t.Fatalf("unique mechanic was not preserved on operation child action: %+v", effects)
	}
}
