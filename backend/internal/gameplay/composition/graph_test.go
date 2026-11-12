package composition

import "testing"

func validGraph() (Graph, Profile, []string) {
	profile := KillerMoveProfile()
	graph := Graph{
		Product: profile.Product, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
		CoreGuInstanceID: "core-instance", OutputEffectKey: "core-instance:0:0",
		Nodes: []Node{
			{EffectKey: "core-instance:0:0", Role: NodeOutput, GuInstanceID: "core-instance", GuID: "GU_CORE", Rank: 6, Trigger: "ON_ACTION", Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER"},
			{EffectKey: "support-instance:1:0", Role: NodeSupport, GuInstanceID: "support-instance", GuID: "GU_SUPPORT", Rank: 5, Trigger: "ON_ACTION", Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER"},
		},
		Edges: []Edge{{From: "support-instance:1:0", To: "core-instance:0:0", Kind: ContributionAmplify}},
	}
	return graph, profile, []string{"core-instance", "support-instance"}
}

func TestResolveKeepsTypedProvenanceAndCanonicalOrder(t *testing.T) {
	graph, profile, selected := validGraph()
	resolved, err := Resolve(graph, profile, selected)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.SchemaVersion != GraphSchemaVersion || resolved.Nodes[0].EffectKey != "core-instance:0:0" || resolved.Edges[0].Kind != ContributionAmplify {
		t.Fatalf("resolved graph = %+v", resolved)
	}
}

func TestResolveFailsClosedForMissingProvenanceOrWrongProfile(t *testing.T) {
	graph, profile, selected := validGraph()
	graph.Edges = nil
	if _, err := Resolve(graph, profile, selected); err == nil {
		t.Fatal("support without provenance edge accepted")
	}
	graph, profile, selected = validGraph()
	graph.Executor = "WORLD_ADAPTER"
	if _, err := Resolve(graph, profile, selected); err == nil {
		t.Fatal("graph for another executor accepted")
	}
}

func TestResolveRequiresEverySelectedGuToContribute(t *testing.T) {
	graph, profile, selected := validGraph()
	selected = append(selected, "unused-instance")
	if _, err := Resolve(graph, profile, selected); err == nil {
		t.Fatal("unused selected Gu accepted")
	}
}

func TestResolveRejectsDuplicateOrEmptySelectedGuIDs(t *testing.T) {
	graph, profile, _ := validGraph()
	for _, tc := range []struct {
		name string
		gus  []string
	}{
		{name: "duplicate", gus: []string{"core-instance", "support-instance", "support-instance"}},
		{name: "empty", gus: []string{"core-instance", "support-instance", ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Resolve(graph, profile, tc.gus); err == nil {
				t.Fatalf("Resolve(%v) unexpectedly succeeded", tc.gus)
			}
		})
	}
}

func TestKillerMovePolicyReservesComponentsOnlyWhileActive(t *testing.T) {
	profile := KillerMoveProfile()
	if profile.GuPolicy != GuPolicyReserveActive {
		t.Fatalf("Gu policy = %q, want %q", profile.GuPolicy, GuPolicyReserveActive)
	}
}

func TestGuHouseProfileOwnsComponentCapsAndActiveReservationPolicy(t *testing.T) {
	profile := GuHouseProfile()
	if profile.ID != "gu_house.v1" || profile.MinGuComponents != 2 || profile.MaxGuComponents != 5 || profile.MaxSaved != 4 || profile.MaxEquipped != 1 {
		t.Fatalf("unexpected Gu House profile: %+v", profile)
	}
	if profile.GuPolicy != GuPolicyReserveActive || profile.Executor != "GU_HOUSE_ACTION_ADAPTER" {
		t.Fatalf("Gu House policy/executor = %q/%q", profile.GuPolicy, profile.Executor)
	}
}

func TestTeleportFormationProfileOwnsTravelContextAndReservationPolicy(t *testing.T) {
	profile := TeleportFormationProfile()
	if profile.ID != "teleport_formation.v1" || profile.Product != "TELEPORT_ARRAY" || profile.GuPolicy != GuPolicyReserveActive {
		t.Fatalf("unexpected teleport formation profile: %+v", profile)
	}
	graph := Graph{
		Product: profile.Product, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
		CoreGuInstanceID: "core", OutputEffectKey: "core:0:0",
		Nodes: []Node{{EffectKey: "core:0:0", Role: NodeOutput, GuInstanceID: "core", GuID: "GU_CORE", Rank: 6,
			Trigger: "ON_USE", Context: "WORLD_TRAVEL_ARRIVAL", Executor: "TELEPORT_ARRAY_TRAVEL_ADAPTER"}},
	}
	if _, err := Resolve(graph, profile, []string{"core"}); err != nil {
		t.Fatalf("valid travel arrival graph rejected: %v", err)
	}
	graph.Nodes[0].Executor = "TURN_BASED_COMBAT_ADAPTER"
	if _, err := Resolve(graph, profile, []string{"core"}); err == nil {
		t.Fatal("formation graph from another executor was accepted")
	}
}
