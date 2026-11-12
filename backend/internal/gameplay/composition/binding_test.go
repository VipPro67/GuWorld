package composition

import "testing"

func TestBindCatalogGraphMaterializesSourcesAndPreservesInput(t *testing.T) {
	outputIndex, supportIndex := 0, 1
	catalog := Graph{
		Product: "KILLER_MOVE", ProductProfile: "killer_move.v1", Context: "KILLER_MOVE", Executor: "KILLER_MOVE_ROUTER",
		RuleSetRevision: "rules.v1", CoreGuInstanceID: "catalog:core:6", OutputEffectKey: "core:6:BLOCK_OUT:0",
		Nodes: []Node{
			{EffectKey: "core:6:BLOCK_OUT:0", SourceBlockID: "BLOCK_OUT", SourceRevision: "rev-core", OperationIndex: &outputIndex, Role: NodeOutput, GuInstanceID: "catalog:core:6", GuID: "core", Rank: 6, Trigger: "ON_ACTION", Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER"},
			{EffectKey: "support:6:BLOCK_HELP:1", SourceBlockID: "BLOCK_HELP", SourceRevision: "rev-support", OperationIndex: &supportIndex, Role: NodeSupport, GuInstanceID: "catalog:support:6", GuID: "support", Rank: 6, Trigger: "ON_ACTION", Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER"},
		},
		Edges: []Edge{{From: "support:6:BLOCK_HELP:1", To: "core:6:BLOCK_OUT:0", Kind: ContributionCombine, RuleID: "combine"}},
	}
	bindings := map[string]SourceBinding{
		"core:6:BLOCK_OUT:0": {EffectKey: "player-core:2:0", GuInstanceID: "player-core", GuID: "core", Rank: 6, SourceBlockID: "BLOCK_OUT", SourceRevision: "rev-core", OperationIndex: 0, Trigger: "ON_ACTION", Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER"},
		"support:6:BLOCK_HELP:1": {EffectKey: "player-support:4:1", GuInstanceID: "player-support", GuID: "support", Rank: 6, SourceBlockID: "BLOCK_HELP", SourceRevision: "rev-support", OperationIndex: 1, Trigger: "ON_ACTION", Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER"},
	}

	got, err := BindCatalogGraph(catalog, "draft-123", bindings)
	if err != nil {
		t.Fatalf("BindCatalogGraph() error = %v", err)
	}
	if got.CatalogDraftID != "draft-123" || got.CoreGuInstanceID != "player-core" || got.OutputEffectKey != "player-core:2:0" {
		t.Fatalf("materialized graph header = draft %q, core %q, output %q", got.CatalogDraftID, got.CoreGuInstanceID, got.OutputEffectKey)
	}
	if got.Edges[0].From != "player-support:4:1" || got.Edges[0].To != "player-core:2:0" {
		t.Fatalf("materialized edge = %+v", got.Edges[0])
	}
	if catalog.CatalogDraftID != "" || catalog.Nodes[0].EffectKey != "core:6:BLOCK_OUT:0" || catalog.Edges[0].From != "support:6:BLOCK_HELP:1" {
		t.Fatal("BindCatalogGraph mutated its input graph")
	}
}

func TestBindCatalogGraphRejectsStaleSourceRevision(t *testing.T) {
	index := 0
	graph := Graph{
		OutputEffectKey: "source", CoreGuInstanceID: "catalog:core:6",
		Nodes: []Node{{EffectKey: "source", SourceBlockID: "BLOCK", SourceRevision: "old", OperationIndex: &index, Role: NodeOutput, GuInstanceID: "catalog:core:6", GuID: "core", Rank: 6, Trigger: "ON_ACTION"}},
	}
	bindings := map[string]SourceBinding{
		"source": {EffectKey: "player-key", GuInstanceID: "player-core", GuID: "core", Rank: 6, SourceBlockID: "BLOCK", SourceRevision: "new", OperationIndex: 0, Trigger: "ON_ACTION", Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER"},
	}
	if _, err := BindCatalogGraph(graph, "draft-123", bindings); err == nil {
		t.Fatal("BindCatalogGraph() accepted a stale source revision")
	}
}
