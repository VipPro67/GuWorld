package composition

import "fmt"

// SourceBinding connects one catalog-authored effect source to a concrete Gu
// instance selected by a product adapter. It is the only product-specific
// input needed to materialize a shared catalog graph for an active product.
type SourceBinding struct {
	EffectKey      string
	GuInstanceID   string
	GuID           string
	Rank           int
	SourceBlockID  string
	SourceRevision string
	OperationIndex int
	Trigger        string
	Context        string
	Executor       string
}

// BindCatalogGraph maps every source in a validated Admin graph to its
// instance-bound runtime key and pins the draft ID. The caller owns candidate
// selection and must provide bindings from its current executor catalog.
func BindCatalogGraph(catalog Graph, draftID string, bindings map[string]SourceBinding) (Graph, error) {
	if draftID == "" || len(catalog.Nodes) == 0 || len(bindings) != len(catalog.Nodes) {
		return Graph{}, fmt.Errorf("catalog graph requires a draft ID and one binding per source")
	}
	graph := catalog
	graph.Nodes = append([]Node(nil), catalog.Nodes...)
	graph.Edges = append([]Edge(nil), catalog.Edges...)
	graph.CatalogDraftID = draftID
	keyBySource := make(map[string]string, len(catalog.Nodes))
	instanceBySource := make(map[string]string, len(catalog.Nodes))
	for index := range graph.Nodes {
		node := &graph.Nodes[index]
		binding, ok := bindings[node.EffectKey]
		if !ok || binding.EffectKey == "" || binding.GuInstanceID == "" || binding.GuID != node.GuID || binding.Rank != node.Rank || binding.SourceBlockID != node.SourceBlockID || binding.SourceRevision == "" || node.SourceRevision != binding.SourceRevision || node.OperationIndex == nil || *node.OperationIndex != binding.OperationIndex || node.Trigger != binding.Trigger || binding.Context == "" || binding.Executor == "" {
			return Graph{}, fmt.Errorf("catalog source %q does not match its runtime binding", node.EffectKey)
		}
		keyBySource[node.EffectKey] = binding.EffectKey
		instanceBySource[node.EffectKey] = binding.GuInstanceID
		node.EffectKey = binding.EffectKey
		node.GuInstanceID = binding.GuInstanceID
		node.SourceRevision = binding.SourceRevision
		node.Context = binding.Context
		node.Executor = binding.Executor
		node.BlockID = ""
	}
	outputKey, ok := keyBySource[catalog.OutputEffectKey]
	if !ok {
		return Graph{}, fmt.Errorf("catalog graph output has no runtime binding")
	}
	graph.OutputEffectKey = outputKey
	graph.CoreGuInstanceID = instanceBySource[catalog.OutputEffectKey]
	if graph.CoreGuInstanceID == "" {
		return Graph{}, fmt.Errorf("catalog graph core has no runtime binding")
	}
	for index := range graph.Edges {
		from, fromOK := keyBySource[graph.Edges[index].From]
		to, toOK := keyBySource[graph.Edges[index].To]
		if !fromOK || !toOK {
			return Graph{}, fmt.Errorf("catalog graph edge has an unbound source")
		}
		graph.Edges[index].From = from
		graph.Edges[index].To = to
	}
	return graph, nil
}
