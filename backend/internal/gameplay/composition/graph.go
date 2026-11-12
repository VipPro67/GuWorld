// Package composition resolves product-specific Gu composition graphs. It
// validates provenance and context without applying effect commands; each
// product adapter still owns command compilation and execution.
package composition

import (
	"fmt"
	"sort"
	"strings"
)

const GraphSchemaVersion = 1

type ContributionKind string

const (
	ContributionUnlock    ContributionKind = "UNLOCK"
	ContributionBridge    ContributionKind = "BRIDGE"
	ContributionAmplify   ContributionKind = "AMPLIFY"
	ContributionStabilize ContributionKind = "STABILIZE"
	ContributionCombine   ContributionKind = "COMBINE"
)

type NodeRole string

const (
	NodeOutput  NodeRole = "OUTPUT"
	NodeSupport NodeRole = "SUPPORT"
)

type Node struct {
	EffectKey      string   `json:"effect_key"`
	BlockID        string   `json:"block_id,omitempty"`
	SourceBlockID  string   `json:"source_block_id,omitempty"`
	SourceRevision string   `json:"source_revision,omitempty"`
	OperationIndex *int     `json:"operation_index,omitempty"`
	Role           NodeRole `json:"role"`
	GuInstanceID   string   `json:"gu_instance_id"`
	GuID           string   `json:"gu_id"`
	Rank           int      `json:"rank"`
	Trigger        string   `json:"trigger"`
	Context        string   `json:"context"`
	Executor       string   `json:"executor"`
}

type Edge struct {
	From   string           `json:"from"`
	To     string           `json:"to"`
	Kind   ContributionKind `json:"kind"`
	RuleID string           `json:"rule_id,omitempty"`
}

type Graph struct {
	SchemaVersion    int    `json:"schema_version"`
	CatalogDraftID   string `json:"catalog_draft_id,omitempty"`
	Product          string `json:"product"`
	ProductProfile   string `json:"product_profile"`
	Context          string `json:"context"`
	Executor         string `json:"executor"`
	RuleSetRevision  string `json:"rule_set_revision,omitempty"`
	CoreGuInstanceID string `json:"core_gu_instance_id"`
	OutputEffectKey  string `json:"output_effect_key"`
	Nodes            []Node `json:"nodes"`
	Edges            []Edge `json:"edges"`
}

type Profile struct {
	ID                     string
	Product                string
	Context                string
	Executor               string
	MinGuComponents        int
	MaxNodes               int
	MaxGuComponents        int
	MaxAuxGuCount          int
	MaxSaved               int
	MaxEquipped            int
	AllowedKinds           map[ContributionKind]bool
	ContextByTrigger       map[string]string
	ExecutorByContext      map[string]string
	GuPolicy               GuPolicy
	PowerBudgetBase        float64
	PowerBudgetPerCoreRank float64
	PowerBudgetPerSupport  float64
	SupportBudgetExponent  float64
	MinimumEffectScale     float64
}

type GuPolicy string

const (
	GuPolicyReference     GuPolicy = "REFERENCE"
	GuPolicyReserveActive GuPolicy = "RESERVE_ACTIVE"
	GuPolicyCatalyst      GuPolicy = "CATALYST"
	GuPolicyConsume       GuPolicy = "CONSUME"
)

// Resolve verifies a composition graph against a product profile and the
// selected Gu instances. It returns a stable graph suitable for persistence
// and for preview fingerprints. Every selected Gu must lead to the output:
// the core owns the output node, and every support has exactly one typed edge.
func Resolve(graph Graph, profile Profile, selectedGuInstances []string) (Graph, error) {
	if graph.SchemaVersion != 0 && graph.SchemaVersion != GraphSchemaVersion {
		return Graph{}, fmt.Errorf("unsupported composition graph version %d", graph.SchemaVersion)
	}
	if graph.Product != profile.Product || graph.ProductProfile != profile.ID || graph.Context != profile.Context || graph.Executor != profile.Executor {
		return Graph{}, fmt.Errorf("composition graph does not match product profile context or executor")
	}
	if strings.TrimSpace(graph.CoreGuInstanceID) == "" || strings.TrimSpace(graph.OutputEffectKey) == "" {
		return Graph{}, fmt.Errorf("composition graph requires a core Gu and output effect")
	}
	if len(graph.Nodes) == 0 || len(graph.Nodes) > profile.MaxNodes {
		return Graph{}, fmt.Errorf("composition node count must be between 1 and %d", profile.MaxNodes)
	}
	minGuComponents := profile.MinGuComponents
	if minGuComponents < 1 {
		minGuComponents = 1
	}
	if len(selectedGuInstances) < minGuComponents || len(selectedGuInstances) > profile.MaxGuComponents {
		return Graph{}, fmt.Errorf("selected Gu count must be between %d and %d", minGuComponents, profile.MaxGuComponents)
	}
	selected := make(map[string]bool, len(selectedGuInstances))
	for _, id := range selectedGuInstances {
		id = strings.TrimSpace(id)
		if id == "" {
			return Graph{}, fmt.Errorf("selected Gu instance id cannot be empty")
		}
		if selected[id] {
			return Graph{}, fmt.Errorf("selected Gu instance %q appears more than once", id)
		}
		selected[id] = true
	}
	if !selected[graph.CoreGuInstanceID] {
		return Graph{}, fmt.Errorf("composition core Gu is not selected")
	}
	nodes := make(map[string]Node, len(graph.Nodes))
	outputCount := 0
	usedGus := map[string]bool{}
	for _, node := range graph.Nodes {
		if node.EffectKey == "" || node.GuInstanceID == "" || node.GuID == "" || node.Rank < 1 {
			return Graph{}, fmt.Errorf("composition node has incomplete effect provenance")
		}
		if profile.ContextByTrigger[node.Trigger] == "" || node.Context != profile.ContextByTrigger[node.Trigger] || node.Executor != profile.ExecutorByContext[node.Context] {
			return Graph{}, fmt.Errorf("effect %q is not supported by this product context and executor", node.EffectKey)
		}
		if _, exists := nodes[node.EffectKey]; exists {
			return Graph{}, fmt.Errorf("composition effect %q appears more than once", node.EffectKey)
		}
		if !selected[node.GuInstanceID] {
			return Graph{}, fmt.Errorf("composition effect %q comes from an unselected Gu", node.EffectKey)
		}
		switch node.Role {
		case NodeOutput:
			outputCount++
			if node.EffectKey != graph.OutputEffectKey || node.GuInstanceID != graph.CoreGuInstanceID {
				return Graph{}, fmt.Errorf("output effect must be authored by the selected core Gu")
			}
		case NodeSupport:
			if node.EffectKey == graph.OutputEffectKey || node.GuInstanceID == graph.CoreGuInstanceID {
				return Graph{}, fmt.Errorf("support node cannot be the output or come from the core Gu")
			}
		default:
			return Graph{}, fmt.Errorf("unsupported composition node role %q", node.Role)
		}
		nodes[node.EffectKey] = node
		usedGus[node.GuInstanceID] = true
	}
	if outputCount != 1 {
		return Graph{}, fmt.Errorf("composition graph must contain exactly one output effect")
	}
	for _, id := range selectedGuInstances {
		if !usedGus[id] {
			return Graph{}, fmt.Errorf("selected Gu %q has no provenance to the output", id)
		}
	}
	if len(graph.Edges) != len(nodes)-1 {
		return Graph{}, fmt.Errorf("every support effect must have exactly one typed rule edge")
	}
	seenFrom := map[string]bool{}
	for _, edge := range graph.Edges {
		from, fromOK := nodes[edge.From]
		to, toOK := nodes[edge.To]
		if !fromOK || !toOK || from.Role != NodeSupport || to.Role != NodeOutput || edge.To != graph.OutputEffectKey {
			return Graph{}, fmt.Errorf("composition edge must connect a support effect to the output")
		}
		if seenFrom[edge.From] {
			return Graph{}, fmt.Errorf("support effect %q has more than one provenance edge", edge.From)
		}
		if !profile.AllowedKinds[edge.Kind] {
			return Graph{}, fmt.Errorf("rule kind %q is not allowed by product profile %q", edge.Kind, profile.ID)
		}
		if from.Context != to.Context || from.Executor != to.Executor {
			return Graph{}, fmt.Errorf("support and output must use the same context executor")
		}
		seenFrom[edge.From] = true
	}
	for _, node := range graph.Nodes {
		if node.Role == NodeSupport && !seenFrom[node.EffectKey] {
			return Graph{}, fmt.Errorf("support effect %q has no provenance edge", node.EffectKey)
		}
	}
	graph.SchemaVersion = GraphSchemaVersion
	graph.Nodes = append([]Node(nil), graph.Nodes...)
	graph.Edges = append([]Edge(nil), graph.Edges...)
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].EffectKey < graph.Nodes[j].EffectKey })
	sort.Slice(graph.Edges, func(i, j int) bool { return graph.Edges[i].From < graph.Edges[j].From })
	return graph, nil
}

func KillerMoveProfile() Profile {
	return Profile{
		ID: "killer_move.v1", Product: "KILLER_MOVE", Context: "KILLER_MOVE", Executor: "KILLER_MOVE_ROUTER",
		MinGuComponents: 1, MaxNodes: 8, MaxGuComponents: 6, MaxSaved: 12, MaxEquipped: 4,
		GuPolicy:        GuPolicyReserveActive,
		PowerBudgetBase: 20, PowerBudgetPerCoreRank: 26, PowerBudgetPerSupport: 13, SupportBudgetExponent: 0.5, MinimumEffectScale: 0.15,
		AllowedKinds: map[ContributionKind]bool{
			ContributionUnlock: true, ContributionBridge: true,
			ContributionAmplify: true, ContributionStabilize: true, ContributionCombine: true,
		},
		ContextByTrigger:  map[string]string{"ON_ACTION": "TURN_BASED_COMBAT", "ON_USE": "OUT_OF_COMBAT_USE"},
		ExecutorByContext: map[string]string{"TURN_BASED_COMBAT": "TURN_BASED_COMBAT_ADAPTER", "OUT_OF_COMBAT_USE": "KILLER_MOVE_USE_ADAPTER"},
	}
}

// GuHouseProfile owns the Gu House's component and activation policy. Mounted
// actions use GuHouseActionProfile for their nested effect composition graphs.
func GuHouseProfile() Profile {
	return Profile{
		ID: "gu_house.v1", Product: "GU_HOUSE", Context: "TURN_BASED_COMBAT", Executor: "GU_HOUSE_ACTION_ADAPTER",
		MinGuComponents: 2, MaxGuComponents: 5, MaxAuxGuCount: 100000, MaxSaved: 4, MaxEquipped: 1,
		GuPolicy:          GuPolicyReserveActive,
		ContextByTrigger:  map[string]string{"ON_ACTION": "TURN_BASED_COMBAT"},
		ExecutorByContext: map[string]string{"TURN_BASED_COMBAT": "GU_HOUSE_ACTION_ADAPTER"},
	}
}

// GuHouseActionProfile validates one mounted Trận Kích. Its component limits
// apply to the action graph, while the parent Gu House owns inventory policy.
func GuHouseActionProfile() Profile {
	profile := GuHouseProfile()
	profile.ID = "gu_house_action.v1"
	profile.Product = "GU_HOUSE_ACTION"
	profile.MinGuComponents = 1
	profile.MaxNodes = 5
	profile.MaxGuComponents = 5
	profile.AllowedKinds = map[ContributionKind]bool{
		ContributionUnlock: true, ContributionBridge: true,
		ContributionAmplify: true, ContributionStabilize: true, ContributionCombine: true,
	}
	return profile
}

// TeleportFormationProfile describes the optional effect installed in a
// placed teleport array. The traveler activates it on arrival; components are
// reserved for as long as that profile remains installed on the array.
func TeleportFormationProfile() Profile {
	return Profile{
		ID: "teleport_formation.v1", Product: "TELEPORT_ARRAY", Context: "TELEPORT_ARRAY", Executor: "TELEPORT_ARRAY_ROUTER",
		MinGuComponents: 1, MaxNodes: 8, MaxGuComponents: 6, MaxSaved: 0, MaxEquipped: 0,
		GuPolicy: GuPolicyReserveActive,
		AllowedKinds: map[ContributionKind]bool{
			ContributionCombine: true,
		},
		ContextByTrigger:  map[string]string{"ON_USE": "WORLD_TRAVEL_ARRIVAL"},
		ExecutorByContext: map[string]string{"WORLD_TRAVEL_ARRIVAL": "TELEPORT_ARRAY_TRAVEL_ADAPTER"},
	}
}
