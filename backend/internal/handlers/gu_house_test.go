package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/gameplay/composition"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestGetGuTargetType(t *testing.T) {
	singleGu := models.Gu{
		ID:   "single_gu",
		Rank: 1,
	}
	if tt := getGuTargetType(singleGu); tt != "SINGLE" {
		t.Fatalf("expected SINGLE, got %s", tt)
	}

	aoeGu := models.Gu{
		ID:      "aoe_gu",
		Rank:    3,
		Effects: models.GuEffects{Blocks: []effects.Block{{Target: effects.Target{Selector: effects.TargetAllEnemies}}}},
	}
	if tt := getGuTargetType(aoeGu); tt != "ALL_ENEMIES" {
		t.Fatalf("expected ALL_ENEMIES, got %s", tt)
	}

	lineGu := models.Gu{
		ID:      "line_gu",
		Rank:    2,
		Effects: models.GuEffects{Blocks: []effects.Block{{Target: effects.Target{Selector: effects.TargetRow}}}},
	}
	if tt := getGuTargetType(lineGu); tt != "ROW" {
		t.Fatalf("expected ROW, got %s", tt)
	}
}

func TestCompileGuHouseActionUsesSharedCompilerAndPreservesEffectProvenance(t *testing.T) {
	gu := ownedKillerMoveGu{InstanceID: "core-instance", GuID: "core_gu", Rank: 6}
	spec := models.Gu{ID: "core_gu", Name: "Core Gu", Rank: 6}
	selected := killerMoveEffectOption{
		Key: "core-instance:0:0", SourceID: "core-instance", GuID: "core_gu", Label: "Gây sát thương", Trigger: string(effects.TriggerOnAction),
		Operation: effects.Operation{Op: effects.OpDamage, SubType: "FLAT", Value: 40},
		Target:    effects.Target{Selector: effects.TargetSingle}, CombatSupported: true,
	}
	compiled, err := compileGuHouseAction(gu, spec, selected, "strength")
	if err != nil {
		t.Fatal(err)
	}
	if compiled.EffectKey != selected.Key || compiled.EffectContext != "TURN_BASED_COMBAT" || compiled.EffectExecutor != "GU_HOUSE_ACTION_ADAPTER" || compiled.BaseDamage != 40 || compiled.TargetType != "SINGLE" {
		t.Fatalf("compiled action lost its contract or provenance: %+v", compiled)
	}
	profile := composition.GuHouseProfile()
	simAction, ok := guHouseActionToSimAction("house-id", profile, map[string]bool{"core-instance": true}, map[string]bool{"core_gu": true}, compiled)
	if !ok || simAction.ActionType != "KILLER_MOVE" || simAction.Damage != 40 || !simAction.IsFromGuHouse || len(simAction.ComponentGuIDs) != 1 {
		t.Fatalf("executor action = %+v, accepted=%v", simAction, ok)
	}
	if _, ok := guHouseActionToSimAction("house-id", profile, nil, map[string]bool{"core_gu": true}, compiled); ok {
		t.Fatal("action from a Gu outside the House core set was accepted")
	}

	selected.Conditions = []effects.Condition{{Field: "target_hp_pct", Operator: "LT", Value: 50}}
	if _, err := compileGuHouseAction(gu, spec, selected, "strength"); err == nil {
		t.Fatal("conditional action accepted without an executor projection")
	}
}

func TestGuHouseLegacyCompositionCannotExecuteWithoutRuleSet(t *testing.T) {
	graph := &composition.Graph{
		SchemaVersion: composition.GraphSchemaVersion, Product: "GU_HOUSE_ACTION", ProductProfile: "gu_house_action.v1",
		Context: "TURN_BASED_COMBAT", Executor: "GU_HOUSE_ACTION_ADAPTER", CoreGuInstanceID: "core-instance", OutputEffectKey: "damage",
		Nodes: []composition.Node{
			{EffectKey: "damage", BlockID: "EFFECT_01", Role: composition.NodeOutput, GuInstanceID: "core-instance", GuID: "core_gu", Rank: 6, Trigger: "ON_ACTION", Context: "TURN_BASED_COMBAT", Executor: "GU_HOUSE_ACTION_ADAPTER"},
			{EffectKey: "heal", BlockID: "EFFECT_02", Role: composition.NodeSupport, GuInstanceID: "support-instance", GuID: "support_gu", Rank: 6, Trigger: "ON_ACTION", Context: "TURN_BASED_COMBAT", Executor: "GU_HOUSE_ACTION_ADAPTER"},
		},
		Edges: []composition.Edge{{From: "heal", To: "damage", Kind: composition.ContributionBridge}},
	}
	action := models.KillerMoveCombatProfile{
		EffectKey: "damage", TargetType: "SINGLE", BaseDamage: 40, BaseHealing: 8, SuccessChance: 1,
		CompositionGraph: graph, ComponentGuInstances: []string{"core-instance", "support-instance"},
		EffectBlocks: effects.BlockSet{
			{ID: "EFFECT_01", Trigger: effects.TriggerOnAction, Target: effects.Target{Selector: effects.TargetSingle}, Operations: []effects.Operation{{Op: effects.OpDamage, SubType: "FLAT", Value: 40}}},
			{ID: "EFFECT_02", Trigger: effects.TriggerOnAction, Target: effects.Target{Selector: effects.TargetSelf}, Operations: []effects.Operation{{Op: effects.OpHeal, SubType: "FLAT", Value: 8}}},
		},
	}
	if _, err := composition.Resolve(*graph, composition.GuHouseActionProfile(), action.ComponentGuInstances); err != nil {
		t.Fatalf("composition graph did not resolve: %v", err)
	}
	if _, ok := guHouseComposedActionToSimAction("house", map[string]bool{"core-instance": true, "support-instance": true}, map[string]bool{"core_gu": true, "support_gu": true}, action); ok {
		t.Fatal("legacy composition edge executed without a versioned rule catalog")
	}

	// Output-only legacy actions remain safe because they do not compose effects.
	outputOnly := action
	graphCopy := *graph
	graphCopy.Nodes = graphCopy.Nodes[:1]
	graphCopy.Edges = nil
	outputOnly.CompositionGraph = &graphCopy
	outputOnly.ComponentGuInstances = []string{"core-instance"}
	outputOnly.EffectBlocks = outputOnly.EffectBlocks[:1]
	outputOnly.BaseHealing = 0
	got, ok := guHouseComposedActionToSimAction("house", map[string]bool{"core-instance": true}, map[string]bool{"core_gu": true}, outputOnly)
	if !ok || got.Damage != 40 || got.TargetType != "SINGLE" || len(got.ComponentGuIDs) != 1 {
		t.Fatalf("legacy output-only action = %+v, accepted=%v", got, ok)
	}
	outputOnly.TargetType = "SELF"
	if _, ok := guHouseComposedActionToSimAction("house", map[string]bool{"core-instance": true}, map[string]bool{"core_gu": true}, outputOnly); ok {
		t.Fatal("enemy damage was accepted under a SELF-target action")
	}
}

func TestGuHouseActivationRequiresRuleRevisionForComposedActions(t *testing.T) {
	legacy := []byte(`{"actions":[{"composition_graph":{"edges":[{"from":"support","to":"output","kind":"BRIDGE"}]}}]}`)
	if err := validateGuHouseActivationProfile(legacy); err == nil {
		t.Fatal("legacy composition profile activated without a rule revision")
	}
	versioned := []byte(`{"actions":[{"composition_graph":{"rule_set_revision":"gu_house_action.rules.v1","edges":[{"from":"support","to":"output","kind":"COMBINE","rule_id":"combine-flat-single-target-damage"}]}}]}`)
	if err := validateGuHouseActivationProfile(versioned); err != nil {
		t.Fatalf("versioned composition profile rejected: %v", err)
	}
	staleOutputOnly := []byte(`{"actions":[{"composition_graph":{"rule_set_revision":"gu_house_action.rules.v0","edges":[]}}]}`)
	if err := validateGuHouseActivationProfile(staleOutputOnly); err == nil {
		t.Fatal("output-only graph with stale non-empty rule revision activated")
	}
}

func TestCompileKillerMoveCombatProfile(t *testing.T) {
	mainSpec := models.Gu{
		ID:   "main_gu",
		Name: "Hạch Tâm Cổ",
		Rank: 3,
		Path: []string{"strength"},
	}
	main := ownedKillerMoveGu{
		InstanceID: "inst_main",
		GuID:       "main_gu",
		Rank:       3,
		Location:   "INVENTORY",
	}
	components := []killerMoveComponent{
		{InstanceID: "inst_main", GuID: "main_gu", Rank: 3},
		{InstanceID: "inst_sub", GuID: "sub_gu", Rank: 3},
	}

	prof := compileKillerMoveCombatProfile(
		mainSpec,
		main,
		components,
		nil,
		nil,
		"strength",
		1,
		0.85,
		1.1,
		0.80,
		50,
	)

	if prof.Rank != 3 {
		t.Fatalf("expected rank 3, got %d", prof.Rank)
	}
	if prof.TargetType != "SINGLE" {
		t.Fatalf("expected target SINGLE, got %s", prof.TargetType)
	}
}

func TestKillerMoveStarvingComponentAvailability(t *testing.T) {
	ownedSet := map[string]ownedKillerMoveGu{
		"inst_1": {InstanceID: "inst_1", GuID: "gu_a", Rank: 1, Hunger: 100, State: "ACTIVE", Location: "INVENTORY"},
		"inst_2": {InstanceID: "inst_2", GuID: "gu_b", Rank: 1, Hunger: 0, State: "ACTIVE", Location: "INVENTORY"},
	}

	components := []killerMoveComponent{
		{InstanceID: "inst_1", GuID: "gu_a", Rank: 1},
		{InstanceID: "inst_2", GuID: "gu_b", Rank: 1},
	}

	available, hasStarvingGu, missingGu := evaluateKillerMoveComponents(components, ownedSet)
	if !available {
		t.Fatalf("expected available=true when all components are owned and alive, got available=false")
	}
	if !hasStarvingGu {
		t.Fatalf("expected has_starving_gu=true when a component has hunger=0, got false")
	}
	if len(missingGu) > 0 {
		t.Fatalf("expected 0 missing gu, got %d", len(missingGu))
	}

	// Missing component test
	delete(ownedSet, "inst_2")
	available2, _, missingGu2 := evaluateKillerMoveComponents(components, ownedSet)
	if available2 {
		t.Fatalf("expected available=false when component is missing, got true")
	}
	if len(missingGu2) != 1 || missingGu2[0] != "inst_2" {
		t.Fatalf("expected missing gu inst_2, got %v", missingGu2)
	}
}

func TestKillerMoveComponentMustBeUnlockedInInventory(t *testing.T) {
	components := []killerMoveComponent{{InstanceID: "inst_1", GuID: "gu_a", Rank: 1}}
	for name, gu := range map[string]ownedKillerMoveGu{
		"locked": {InstanceID: "inst_1", GuID: "gu_a", Rank: 1, State: "ACTIVE", Location: "INVENTORY", IsLocked: true},
		"escrow": {InstanceID: "inst_1", GuID: "gu_a", Rank: 1, State: "ACTIVE", Location: "ACTOR"},
	} {
		t.Run(name, func(t *testing.T) {
			available, _, _ := evaluateKillerMoveComponents(components, map[string]ownedKillerMoveGu{"inst_1": gu})
			if available {
				t.Fatal("locked or non-inventory Gu was considered available")
			}
		})
	}
}
