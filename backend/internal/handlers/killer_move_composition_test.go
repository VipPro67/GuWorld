package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gameplay/composition"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestKillerMoveStoredCompositionRevalidatesRuleGrantAndBlockSnapshot(t *testing.T) {
	previousRegistry := models.GuRegistry
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"core_gu":    {2: testDamageGu("core_gu")},
		"support_gu": {2: testDamageGu("support_gu")},
	}
	t.Cleanup(func() { models.GuRegistry = previousRegistry })

	core := ownedKillerMoveGu{InstanceID: "core-instance", GuID: "core_gu", Rank: 2}
	support := ownedKillerMoveGu{InstanceID: "support-instance", GuID: "support_gu", Rank: 2}
	coreOptions, err := getKillerMoveEffectOptions(core)
	if err != nil {
		t.Fatal(err)
	}
	supportOptions, err := getKillerMoveEffectOptions(support)
	if err != nil {
		t.Fatal(err)
	}
	coreOption, supportOption := coreOptions[0], supportOptions[0]
	profile := composition.KillerMoveProfile()
	coreOpIndex, supportOpIndex := coreOption.OperationIndex, supportOption.OperationIndex
	graph := composition.Graph{
		SchemaVersion: composition.GraphSchemaVersion, Product: profile.Product, ProductProfile: profile.ID,
		Context: profile.Context, Executor: profile.Executor, RuleSetRevision: composition.KillerMoveRuleSet().Revision,
		CoreGuInstanceID: core.InstanceID, OutputEffectKey: coreOption.Key,
		Nodes: []composition.Node{
			{EffectKey: coreOption.Key, BlockID: "EFFECT_01", SourceBlockID: coreOption.SourceBlockID, OperationIndex: &coreOpIndex, Role: composition.NodeOutput, GuInstanceID: core.InstanceID, GuID: core.GuID, Rank: core.Rank, Trigger: coreOption.Trigger, Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER"},
			{EffectKey: supportOption.Key, BlockID: "EFFECT_02", SourceBlockID: supportOption.SourceBlockID, OperationIndex: &supportOpIndex, Role: composition.NodeSupport, GuInstanceID: support.InstanceID, GuID: support.GuID, Rank: support.Rank, Trigger: supportOption.Trigger, Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER"},
		},
		Edges: []composition.Edge{{From: supportOption.Key, To: coreOption.Key, Kind: composition.ContributionCombine, RuleID: "combine-flat-single-target-damage"}},
	}
	refs := []killerMoveEffectRef{
		{Key: coreOption.Key, SourceBlockID: coreOption.SourceBlockID, Trigger: coreOption.Trigger, InstanceID: core.InstanceID, GuID: core.GuID, Rank: core.Rank, BlockIndex: coreOption.BlockIndex, OpIndex: coreOption.OperationIndex},
		{Key: supportOption.Key, SourceBlockID: supportOption.SourceBlockID, Trigger: supportOption.Trigger, InstanceID: support.InstanceID, GuID: support.GuID, Rank: support.Rank, BlockIndex: supportOption.BlockIndex, OpIndex: supportOption.OperationIndex},
	}
	blocks := []effects.Block{
		{ID: "EFFECT_01", Trigger: effects.TriggerOnAction, Target: coreOption.Target, Operations: []effects.Operation{{Op: effects.OpDamage, SubType: "FLAT", Value: 80}}},
		{ID: "EFFECT_02", Trigger: effects.TriggerOnAction, Target: supportOption.Target, Operations: []effects.Operation{{Op: effects.OpDamage, SubType: "FLAT", Value: 40}}},
	}
	components := []killerMoveComponent{{InstanceID: core.InstanceID, GuID: core.GuID, Rank: core.Rank}, {InstanceID: support.InstanceID, GuID: support.GuID, Rank: support.Rank}}
	if err := validateKillerMoveCompositionSnapshot(graph, refs, components, blocks); err != nil {
		t.Fatalf("valid stored composition rejected: %v", err)
	}

	graph.Edges[0].RuleID = "unknown-rule"
	if err := validateKillerMoveCompositionSnapshot(graph, refs, components, blocks); err == nil {
		t.Fatal("stored composition with an unknown rule was accepted")
	}
}

func testDamageGu(id string) models.GuDictionaryItem {
	return models.GuDictionaryItem{
		ID: id, Rank: 2,
		Effects: models.GuEffects{Blocks: []effects.Block{{
			ID: "ACTIVE_SKILL", Trigger: effects.TriggerOnAction, Target: effects.Target{Selector: effects.TargetSingle},
			Operations: []effects.Operation{{Op: effects.OpDamage, SubType: "FLAT", Value: 40}},
		}}},
		CompositionContributions: composition.GrantCatalog{SchemaVersion: 1, Grants: []composition.CapabilityGrant{{
			EffectBlockID: "ACTIVE_SKILL", OperationIndex: 0, Capability: "COMBAT.DAMAGE.FLAT.SINGLE_TARGET",
		}}},
	}
}
