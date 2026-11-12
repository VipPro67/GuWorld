package composition

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const RuleSetSchemaVersion = 1

var compositionRuleID = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// CompositionRule is an authored compatibility rule for a specific product
// profile. It grants one typed support-to-output relation only when both
// source effects carry the declared capabilities.
type CompositionRule struct {
	ID                string           `json:"id"`
	ProductProfile    string           `json:"product_profile"`
	Context           string           `json:"context"`
	Executor          string           `json:"executor"`
	Kind              ContributionKind `json:"kind"`
	CoreCapability    string           `json:"core_capability"`
	SupportCapability string           `json:"support_capability"`
	OutputTemplate    OutputTemplate   `json:"output_template"`
}

// OutputTemplate declares how a permitted support becomes product output.
// APPEND_SOURCE_OPERATION_BLOCKS preserves each authored source command as a
// separate Block v2 command while making its inclusion an explicit rule result.
type OutputTemplate struct {
	Mode          string `json:"mode"`
	BlockIDPrefix string `json:"block_id_prefix"`
}

type RuleSet struct {
	SchemaVersion int               `json:"schema_version"`
	Revision      string            `json:"revision"`
	Rules         []CompositionRule `json:"rules"`
}

// KillerMoveRuleSet is the first curated runtime catalog. Keep this revision
// stable once graphs using it are persisted; edits require a new revision.
func KillerMoveRuleSet() RuleSet {
	return RuleSet{
		SchemaVersion: RuleSetSchemaVersion,
		Revision:      "killer_move.rules.v2",
		Rules: []CompositionRule{{
			ID: "combine-flat-single-target-damage", ProductProfile: "killer_move.v1",
			Context: "TURN_BASED_COMBAT", Executor: "TURN_BASED_COMBAT_ADAPTER", Kind: ContributionCombine,
			CoreCapability: "COMBAT.DAMAGE.FLAT.SINGLE_TARGET", SupportCapability: "COMBAT.DAMAGE.FLAT.SINGLE_TARGET",
			OutputTemplate: OutputTemplate{Mode: "APPEND_SOURCE_OPERATION_BLOCKS", BlockIDPrefix: "EFFECT"},
		}},
	}
}

// GuHouseActionRuleSet has an independent revision because Gu House actions
// have their own executor and persisted lifecycle.
func GuHouseActionRuleSet() RuleSet {
	return RuleSet{
		SchemaVersion: RuleSetSchemaVersion,
		Revision:      "gu_house_action.rules.v1",
		Rules: []CompositionRule{{
			ID: "combine-flat-single-target-damage", ProductProfile: "gu_house_action.v1",
			Context: "TURN_BASED_COMBAT", Executor: "GU_HOUSE_ACTION_ADAPTER", Kind: ContributionCombine,
			CoreCapability: capabilityFlatSingleTargetDamage, SupportCapability: capabilityFlatSingleTargetDamage,
			OutputTemplate: OutputTemplate{Mode: "APPEND_SOURCE_OPERATION_BLOCKS", BlockIDPrefix: "EFFECT"},
		}},
	}
}

// TeleportFormationRuleSet owns effect composition for placed travel arrays.
func TeleportFormationRuleSet() RuleSet {
	return RuleSet{
		SchemaVersion: RuleSetSchemaVersion,
		Revision:      "teleport_formation.rules.v1",
		Rules: []CompositionRule{{
			ID: "combine-arrival-self-effects", ProductProfile: "teleport_formation.v1",
			Context: "WORLD_TRAVEL_ARRIVAL", Executor: "TELEPORT_ARRAY_TRAVEL_ADAPTER", Kind: ContributionCombine,
			CoreCapability: capabilityTravelArrivalSelfEffect, SupportCapability: capabilityTravelArrivalSelfEffect,
			OutputTemplate: OutputTemplate{Mode: "APPEND_SOURCE_OPERATION_BLOCKS", BlockIDPrefix: "ARRIVAL"},
		}},
	}
}

// ParseRuleSet validates and canonicalizes a versioned, explicit rule set.
func ParseRuleSet(raw []byte) (RuleSet, error) {
	var set RuleSet
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&set); err != nil {
		return RuleSet{}, fmt.Errorf("composition rule set must be a JSON envelope: %w", err)
	}
	if set.SchemaVersion != RuleSetSchemaVersion || strings.TrimSpace(set.Revision) == "" || set.Rules == nil {
		return RuleSet{}, fmt.Errorf("composition rule set requires schema_version %d, revision, and rules array", RuleSetSchemaVersion)
	}
	seen := make(map[string]bool, len(set.Rules))
	for _, rule := range set.Rules {
		if !compositionRuleID.MatchString(rule.ID) || seen[rule.ID] || strings.TrimSpace(rule.ProductProfile) == "" || strings.TrimSpace(rule.Context) == "" || strings.TrimSpace(rule.Executor) == "" || !knownCapability(rule.CoreCapability) || !knownCapability(rule.SupportCapability) {
			return RuleSet{}, fmt.Errorf("composition rules require unique IDs, profile/context/executor, and both capabilities")
		}
		if rule.Kind != ContributionUnlock && rule.Kind != ContributionBridge && rule.Kind != ContributionAmplify && rule.Kind != ContributionStabilize && rule.Kind != ContributionCombine {
			return RuleSet{}, fmt.Errorf("composition rule %q has unsupported relation kind %q", rule.ID, rule.Kind)
		}
		if rule.OutputTemplate.Mode != "APPEND_SOURCE_OPERATION_BLOCKS" || !compositionRuleID.MatchString(strings.ToLower(rule.OutputTemplate.BlockIDPrefix)) {
			return RuleSet{}, fmt.Errorf("composition rule %q has an unsupported output template", rule.ID)
		}
		seen[rule.ID] = true
	}
	sort.Slice(set.Rules, func(i, j int) bool { return set.Rules[i].ID < set.Rules[j].ID })
	return set, nil
}

// ResolveWithRules adds strict rule provenance to the structural graph
// resolver. capabilitiesByEffect maps each selected effect key to explicit
// authoring grants; role and tag fields are never consulted.
func ResolveWithRules(graph Graph, profile Profile, selectedGuInstances []string, set RuleSet, capabilitiesByEffect map[string]map[string]bool) (Graph, error) {
	rawRules, err := json.Marshal(set)
	if err != nil {
		return Graph{}, fmt.Errorf("encode composition rule set: %w", err)
	}
	set, err = ParseRuleSet(rawRules)
	if err != nil {
		return Graph{}, err
	}
	if graph.RuleSetRevision != set.Revision {
		return Graph{}, fmt.Errorf("composition graph rule set revision does not match the active catalog")
	}
	resolved, err := Resolve(graph, profile, selectedGuInstances)
	if err != nil {
		return Graph{}, err
	}
	rules := make(map[string]CompositionRule, len(set.Rules))
	for _, rule := range set.Rules {
		if rule.ID == "" || rules[rule.ID].ID != "" {
			return Graph{}, fmt.Errorf("composition rule IDs must be non-empty and unique")
		}
		rules[rule.ID] = rule
	}
	for _, edge := range resolved.Edges {
		rule, ok := rules[edge.RuleID]
		if !ok {
			return Graph{}, fmt.Errorf("composition edge references unknown rule %q", edge.RuleID)
		}
		from := resolvedNode(resolved, edge.From)
		to := resolvedNode(resolved, edge.To)
		if from.SourceBlockID == "" || to.SourceBlockID == "" || from.OperationIndex == nil || to.OperationIndex == nil || *from.OperationIndex < 0 || *to.OperationIndex < 0 {
			return Graph{}, fmt.Errorf("composition rule edges require source Block IDs and operation indexes")
		}
		if rule.ProductProfile != profile.ID || rule.Context != from.Context || rule.Context != to.Context || rule.Executor != from.Executor || rule.Executor != to.Executor || rule.Kind != edge.Kind {
			return Graph{}, fmt.Errorf("composition rule %q does not match this profile, executor, or relation kind", rule.ID)
		}
		if !capabilitiesByEffect[edge.To][rule.CoreCapability] {
			return Graph{}, fmt.Errorf("output effect %q lacks core capability %q required by rule %q", edge.To, rule.CoreCapability, rule.ID)
		}
		if !capabilitiesByEffect[edge.From][rule.SupportCapability] {
			return Graph{}, fmt.Errorf("support effect %q lacks capability %q required by rule %q", edge.From, rule.SupportCapability, rule.ID)
		}
	}
	return resolved, nil
}

func resolvedNode(graph Graph, effectKey string) Node {
	for _, node := range graph.Nodes {
		if node.EffectKey == effectKey {
			return node
		}
	}
	return Node{}
}
