package composition

import (
	"encoding/json"
	"strings"
	"testing"

	effects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

func TestParseGrantCatalogBindsExactBlockOperation(t *testing.T) {
	blocks := []effects.Block{{ID: "ACTIVE_SKILL", Trigger: effects.TriggerOnAction, Target: effects.Target{Selector: effects.TargetSingle}, Operations: []effects.Operation{{Op: effects.OpDamage, SubType: "FLAT"}}}}
	catalog, err := ParseGrantCatalog([]byte(`{"schema_version":1,"grants":[{"effect_block_id":"ACTIVE_SKILL","operation_index":0,"capability":"combat.damage.flat.single_target"}]}`), blocks)
	if err != nil {
		t.Fatal(err)
	}
	if got := catalog.Grants[0].Capability; got != capabilityFlatSingleTargetDamage {
		t.Fatalf("capability = %q", got)
	}
	if _, err := ParseGrantCatalog([]byte(`{"schema_version":1,"grants":[{"effect_block_id":"ACTIVE_SKILL","operation_index":1,"capability":"COMBAT.DAMAGE.FLAT.SINGLE_TARGET"}]}`), blocks); err == nil {
		t.Fatal("grant to a missing operation was accepted")
	}
	healingBlocks := []effects.Block{{ID: "ACTIVE_SKILL", Trigger: effects.TriggerOnAction, Target: effects.Target{Selector: effects.TargetSingle}, Operations: []effects.Operation{{Op: effects.OpHeal, SubType: "FLAT"}}}}
	if _, err := ParseGrantCatalog([]byte(`{"schema_version":1,"grants":[{"effect_block_id":"ACTIVE_SKILL","operation_index":0,"capability":"COMBAT.DAMAGE.FLAT.SINGLE_TARGET"}]}`), healingBlocks); err == nil {
		t.Fatal("damage capability attached to a heal operation was accepted")
	}
}

func TestParseGrantCatalogRejectsUnknownFieldsAndDuplicateSources(t *testing.T) {
	blocks := []effects.Block{{ID: "ACTIVE_SKILL", Operations: []effects.Operation{{Op: effects.OpDamage}}}}
	for _, raw := range []string{
		`{"schema_version":1,"grants":[],"inferred_from_roles":true}`,
		`{"schema_version":1,"grants":[{"effect_block_id":"ACTIVE_SKILL","operation_index":0,"capability":"A"},{"effect_block_id":"ACTIVE_SKILL","operation_index":0,"capability":"B"}]}`,
	} {
		if _, err := ParseGrantCatalog([]byte(raw), blocks); err == nil {
			t.Fatalf("invalid catalog accepted: %s", raw)
		}
	}
}

func TestGrantCatalogValueUsesVersionedEnvelope(t *testing.T) {
	b, err := (GrantCatalog{}).Value()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"schema_version":1`) || !strings.Contains(string(b), `"grants":[]`) {
		t.Fatalf("catalog envelope = %s", b)
	}
}

func TestKillerMoveRuleSetDeclaresVersionedOutputTemplate(t *testing.T) {
	raw, err := json.Marshal(KillerMoveRuleSet())
	if err != nil {
		t.Fatal(err)
	}
	set, err := ParseRuleSet(raw)
	if err != nil {
		t.Fatal(err)
	}
	if set.Revision != "killer_move.rules.v2" || len(set.Rules) != 1 || set.Rules[0].Kind != ContributionCombine || set.Rules[0].OutputTemplate.Mode != "APPEND_SOURCE_OPERATION_BLOCKS" {
		t.Fatalf("unexpected curated killer move rule set: %+v", set)
	}
}

func TestGuHouseActionRuleSetIsOwnedByItsExecutor(t *testing.T) {
	set := GuHouseActionRuleSet()
	raw, err := json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseRuleSet(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Revision != "gu_house_action.rules.v1" || len(parsed.Rules) != 1 || parsed.Rules[0].ProductProfile != "gu_house_action.v1" || parsed.Rules[0].Executor != "GU_HOUSE_ACTION_ADAPTER" || parsed.Rules[0].Kind != ContributionCombine {
		t.Fatalf("unexpected Gu House action catalog: %+v", parsed)
	}
}

func TestTeleportFormationRuleSetIsOwnedByArrivalExecutor(t *testing.T) {
	set := TeleportFormationRuleSet()
	raw, err := json.Marshal(set)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseRuleSet(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Revision != "teleport_formation.rules.v1" || len(parsed.Rules) != 1 || parsed.Rules[0].Context != "WORLD_TRAVEL_ARRIVAL" || parsed.Rules[0].Executor != "TELEPORT_ARRAY_TRAVEL_ADAPTER" || parsed.Rules[0].Kind != ContributionCombine {
		t.Fatalf("unexpected Formation rule catalog: %+v", parsed)
	}
}

func TestTravelCapabilityGrantIsRestrictedToSupportedSelfEffects(t *testing.T) {
	valid := effects.Block{ID: "ON_USE", Trigger: effects.TriggerOnUse, Target: effects.Target{Selector: effects.TargetSelf}, Operations: []effects.Operation{{Op: effects.OpStatus, StatusID: "HASTE", Value: 20, Turns: 5, Chance: 1}}}
	if _, err := ParseGrantCatalog([]byte(`{"schema_version":1,"grants":[{"effect_block_id":"ON_USE","operation_index":0,"capability":"WORLD_TRAVEL.ARRIVAL.SELF_EFFECT"}]}`), []effects.Block{valid}); err != nil {
		t.Fatalf("supported arrival status grant rejected: %v", err)
	}
	invalid := valid
	invalid.Target.Selector = effects.TargetSingle
	if _, err := ParseGrantCatalog([]byte(`{"schema_version":1,"grants":[{"effect_block_id":"ON_USE","operation_index":0,"capability":"WORLD_TRAVEL.ARRIVAL.SELF_EFFECT"}]}`), []effects.Block{invalid}); err == nil {
		t.Fatal("non-self arrival effect received travel capability")
	}
}

func TestResolveWithRulesRequiresRuleAndExactCapabilityGrants(t *testing.T) {
	graph, profile, selected := validGraph()
	graph.Edges[0].Kind = ContributionCombine
	coreOperationIndex, supportOperationIndex := 0, 0
	graph.Nodes[0].SourceBlockID, graph.Nodes[0].OperationIndex = "ACTIVE_SKILL", &coreOperationIndex
	graph.Nodes[1].SourceBlockID, graph.Nodes[1].OperationIndex = "ACTIVE_SKILL", &supportOperationIndex
	graph.RuleSetRevision = "killer-move-rules.1"
	graph.Edges[0].RuleID = "combine-flat-damage"
	set := RuleSet{SchemaVersion: RuleSetSchemaVersion, Revision: graph.RuleSetRevision, Rules: []CompositionRule{{
		ID: "combine-flat-damage", ProductProfile: profile.ID, Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER",
		Kind: ContributionCombine, CoreCapability: capabilityFlatSingleTargetDamage, SupportCapability: capabilityFlatSingleTargetDamage,
		OutputTemplate: OutputTemplate{Mode: "APPEND_SOURCE_OPERATION_BLOCKS", BlockIDPrefix: "EFFECT"},
	}}}
	grants := map[string]map[string]bool{
		graph.OutputEffectKey: {capabilityFlatSingleTargetDamage: true},
		graph.Edges[0].From:   {capabilityFlatSingleTargetDamage: true},
	}
	if _, err := ResolveWithRules(graph, profile, selected, set, grants); err != nil {
		t.Fatalf("authored matching rule rejected: %v", err)
	}
	delete(grants[graph.Edges[0].From], capabilityFlatSingleTargetDamage)
	if _, err := ResolveWithRules(graph, profile, selected, set, grants); err == nil {
		t.Fatal("support effect without the rule's explicit capability was accepted")
	}
}
