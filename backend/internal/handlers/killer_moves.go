package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/composition"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

// validateKillerMoveCompositionSnapshot revalidates persisted provenance at
// activation/use time. Output-only legacy graphs remain readable; any graph
// with composition edges must carry the current explicit rule revision.
func validateKillerMoveCompositionSnapshot(graph composition.Graph, refs []killerMoveEffectRef, components []killerMoveComponent, blocks []gameplayeffects.Block) error {
	return validateKillerMoveCompositionSnapshotMode(graph, refs, components, blocks, true)
}

func validateKillerMoveCompositionSnapshotMode(graph composition.Graph, refs []killerMoveEffectRef, components []killerMoveComponent, blocks []gameplayeffects.Block, requireCurrentSource bool) error {
	if len(graph.Nodes) == 0 || len(graph.Nodes) != len(refs) || len(graph.Nodes) != len(blocks) {
		return fmt.Errorf("graph, selected effects and Block v2 snapshot do not have matching node counts")
	}
	refByKey := make(map[string]killerMoveEffectRef, len(refs))
	for _, ref := range refs {
		if ref.Key == "" || refByKey[ref.Key].Key != "" {
			return fmt.Errorf("selected effect provenance is missing or duplicated")
		}
		refByKey[ref.Key] = ref
	}
	componentsByID := make(map[string]killerMoveComponent, len(components))
	selected := make([]string, 0, len(components))
	for _, component := range components {
		if component.InstanceID == "" || componentsByID[component.InstanceID].InstanceID != "" {
			return fmt.Errorf("component provenance is missing or duplicated")
		}
		componentsByID[component.InstanceID] = component
		selected = append(selected, component.InstanceID)
	}
	capabilitiesByEffect := make(map[string]map[string]bool, len(graph.Nodes))
	blocksByID := make(map[string]gameplayeffects.Block, len(blocks))
	for _, block := range blocks {
		if block.ID == "" || blocksByID[block.ID].ID != "" {
			return fmt.Errorf("compiled Block v2 IDs are missing or duplicated")
		}
		blocksByID[block.ID] = block
	}
	legacyOutputOnly := graph.RuleSetRevision == "" && len(graph.Edges) == 0 && len(graph.Nodes) == 1
	for nodeIndex := range graph.Nodes {
		node := &graph.Nodes[nodeIndex]
		ref, exists := refByKey[node.EffectKey]
		if !exists || node.EffectKey != ref.Key || node.GuInstanceID != ref.InstanceID || node.GuID != ref.GuID || node.Rank != ref.Rank || node.Trigger != ref.Trigger {
			return fmt.Errorf("graph node %q does not match its selected effect provenance", node.EffectKey)
		}
		if node.SourceBlockID == "" && legacyOutputOnly {
			node.SourceBlockID = ref.SourceBlockID
		}
		if node.OperationIndex == nil && legacyOutputOnly {
			opIndex := ref.OpIndex
			node.OperationIndex = &opIndex
		}
		if node.SourceBlockID != ref.SourceBlockID || node.OperationIndex == nil || *node.OperationIndex != ref.OpIndex {
			return fmt.Errorf("graph node %q has stale Block/op provenance", node.EffectKey)
		}
		if ref.SourceRevision != "" && node.SourceRevision != ref.SourceRevision {
			return fmt.Errorf("graph node %q does not match its pinned source revision", node.EffectKey)
		}
		component, ownsComponent := componentsByID[node.GuInstanceID]
		if !ownsComponent || component.GuID != node.GuID || component.Rank != node.Rank {
			return fmt.Errorf("graph node %q is not backed by its declared Gu component", node.EffectKey)
		}
		if requireCurrentSource {
			instance := ownedKillerMoveGu{InstanceID: ref.InstanceID, GuID: ref.GuID, Rank: ref.Rank}
			options, err := getKillerMoveEffectOptions(instance)
			if err != nil {
				return err
			}
			var option *killerMoveEffectOption
			for index := range options {
				if options[index].Key == ref.Key {
					option = &options[index]
					break
				}
			}
			if option == nil || option.SourceBlockID != ref.SourceBlockID || option.OperationIndex != ref.OpIndex || option.Trigger != ref.Trigger || (ref.SourceRevision != "" && option.SourceRevision != ref.SourceRevision) {
				return fmt.Errorf("selected effect %q no longer matches its authored Block/op revision", ref.Key)
			}
			if len(option.CompositionCapabilities) > 0 {
				capabilitiesByEffect[node.EffectKey] = make(map[string]bool, len(option.CompositionCapabilities))
				for _, capability := range option.CompositionCapabilities {
					capabilitiesByEffect[node.EffectKey][capability] = true
				}
			}
			snapshot, found := blocksByID[node.BlockID]
			if !found || snapshot.Trigger != gameplayeffects.Trigger(option.Trigger) || snapshot.Target != option.Target || !reflect.DeepEqual(snapshot.Conditions, option.Conditions) || len(snapshot.Operations) != 1 || !composition.SameOperationSemantics(snapshot.Operations[0], option.Operation) {
				return fmt.Errorf("Block v2 snapshot for %q does not match its authored operation", node.EffectKey)
			}
		} else if node.BlockID == "" || blocksByID[node.BlockID].ID == "" {
			return fmt.Errorf("active Block v2 snapshot for %q is missing its materialized Block", node.EffectKey)
		}
		snapshot, exists := blocksByID[node.BlockID]
		if !exists {
			return fmt.Errorf("Block v2 snapshot for %q is missing", node.EffectKey)
		}
		conditionContext := "KILLER_MOVE_COMBAT"
		if node.Trigger == string(gameplayeffects.TriggerOnUse) {
			conditionContext = "KILLER_MOVE_ON_USE"
		}
		if reason := unsupportedConditionReason(conditionContext, snapshot.Conditions); reason != "" {
			return fmt.Errorf("Block v2 snapshot for %q has unsupported conditions: %s", node.EffectKey, reason)
		}
	}
	profile := composition.KillerMoveProfile()
	if graph.RuleSetRevision == "" {
		if len(graph.Edges) > 0 || !legacyOutputOnly {
			return fmt.Errorf("composed Sát Chiêu graph is missing its rule catalog revision")
		}
		_, err := composition.Resolve(graph, profile, selected)
		return err
	}
	if requireCurrentSource {
		_, err := composition.ResolveWithRules(graph, profile, selected, composition.KillerMoveRuleSet(), capabilitiesByEffect)
		return err
	}
	if len(graph.Edges) > 0 && graph.RuleSetRevision == "" {
		return fmt.Errorf("active composed Sát Chiêu is missing its pinned rule revision")
	}
	for _, edge := range graph.Edges {
		if edge.RuleID == "" {
			return fmt.Errorf("active composition edge is missing its pinned rule provenance")
		}
	}
	_, err := composition.Resolve(graph, profile, selected)
	return err
}

func bindKillerMoveCompositionDraft(c *gin.Context, tx *sql.Tx, playerID uuid.UUID, req *killerMoveDraftRequest) error {
	if req.CompositionDraftID == "" {
		return nil
	}
	id, err := uuid.Parse(req.CompositionDraftID)
	if err != nil {
		return fmt.Errorf("Composition draft ID không hợp lệ")
	}
	var raw []byte
	if err := tx.QueryRowContext(c.Request.Context(), `SELECT definition FROM admin_effect_composition_drafts WHERE id=$1 AND status='DRAFT' AND product_profile=$2`, id, composition.KillerMoveProfile().ID).Scan(&raw); err != nil {
		return fmt.Errorf("Không tìm thấy composition draft Sát Chiêu đang hoạt động")
	}
	var draft adminCompositionDraft
	if err := json.Unmarshal(raw, &draft); err != nil {
		return fmt.Errorf("Composition draft bị hỏng: %w", err)
	}
	resolved, err := resolveAdminComposition(c, draft)
	if err != nil {
		return fmt.Errorf("Composition draft không còn hợp lệ; Admin cần validate và lưu lại: %w", err)
	}
	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		return err
	}
	output := resolved.Draft.OutputEffectKey
	var outputSource *adminCompositionSource
	for i := range resolved.Draft.Sources {
		if resolved.Draft.Sources[i].Key == output {
			outputSource = &resolved.Draft.Sources[i]
			break
		}
	}
	if outputSource == nil {
		return fmt.Errorf("Composition draft thiếu Gu output")
	}
	main, ok := owned[req.MainGuInstanceID]
	if !ok || main.GuID != outputSource.GuID || main.Rank != outputSource.Rank {
		return fmt.Errorf("Cổ chính phải khớp Gu/rank output của composition draft (%s R%d)", outputSource.GuID, outputSource.Rank)
	}
	eligible := func(gu ownedKillerMoveGu) bool {
		return !strings.EqualFold(gu.State, "DEAD") && !gu.IsLocked && gu.Location == "INVENTORY"
	}
	preferred := make(map[string]bool, len(req.ComponentIDs))
	for _, instanceID := range req.ComponentIDs {
		preferred[instanceID] = true
	}
	ownedIDs := make([]string, 0, len(owned))
	for instanceID := range owned {
		ownedIDs = append(ownedIDs, instanceID)
	}
	sort.Strings(ownedIDs)
	assignedBySource := map[string]string{output: main.InstanceID}
	assignedComponents := make([]string, 0)
	usedInstances := map[string]bool{main.InstanceID: true}
	assignedByGuRank := make(map[string]string)
	for _, source := range resolved.Draft.Sources {
		if source.Role != string(composition.NodeSupport) {
			continue
		}
		groupKey := fmt.Sprintf("%s:%d", source.GuID, source.Rank)
		if existing := assignedByGuRank[groupKey]; existing != "" {
			assignedBySource[source.Key] = existing
			continue
		}
		instanceID := ""
		for pass := 0; pass < 2 && instanceID == ""; pass++ {
			for _, candidateID := range ownedIDs {
				candidate := owned[candidateID]
				if candidateID == main.InstanceID || usedInstances[candidateID] || candidate.GuID != source.GuID || candidate.Rank != source.Rank || !eligible(candidate) || (pass == 0) != preferred[candidateID] {
					continue
				}
				instanceID = candidateID
				break
			}
		}
		if instanceID == "" {
			return fmt.Errorf("Cần Cổ hỗ trợ %s rank %d còn sống, mở khóa và trong hành trang", source.GuID, source.Rank)
		}
		assignedBySource[source.Key] = instanceID
		assignedByGuRank[groupKey] = instanceID
		usedInstances[instanceID] = true
		assignedComponents = append(assignedComponents, instanceID)
	}
	bindings := make(map[string]composition.SourceBinding, len(resolved.Draft.Sources))
	for _, source := range resolved.Draft.Sources {
		instanceID := assignedBySource[source.Key]
		instance := owned[instanceID]
		options, optionErr := getKillerMoveEffectOptions(instance)
		if optionErr != nil {
			return optionErr
		}
		for _, option := range options {
			if option.SourceBlockID == source.BlockID && option.OperationIndex == source.OperationIndex && option.GuID == source.GuID && option.Rank == source.Rank && option.SourceRevision == source.SourceRevision {
				contextID := composition.KillerMoveProfile().ContextByTrigger[option.Trigger]
				bindings[source.Key] = composition.SourceBinding{
					EffectKey: option.Key, GuInstanceID: option.SourceID, GuID: option.GuID, Rank: option.Rank,
					SourceBlockID: option.SourceBlockID, SourceRevision: option.SourceRevision,
					OperationIndex: option.OperationIndex, Trigger: option.Trigger, Context: contextID,
					Executor: composition.KillerMoveProfile().ExecutorByContext[contextID],
				}
				break
			}
		}
		if _, found := bindings[source.Key]; !found {
			return fmt.Errorf("Effect source %s không được hỗ trợ bởi Sát Chiêu hiện tại", source.Key)
		}
	}
	graph, err := composition.BindCatalogGraph(resolved.Graph, id.String(), bindings)
	if err != nil {
		return err
	}
	req.ComponentIDs = assignedComponents
	req.SelectedEffectKeys = make([]string, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		req.SelectedEffectKeys = append(req.SelectedEffectKeys, node.EffectKey)
	}
	req.OutputEffectKey = graph.OutputEffectKey
	req.RuleSetRevision = graph.RuleSetRevision
	req.CompositionEdges = graph.Edges
	return nil
}

func GetKillerMoveOptions(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải Cổ thành phần"})
		return
	}
	instances := make([]ownedKillerMoveGu, 0, len(owned))
	for _, gu := range owned {
		if strings.EqualFold(gu.State, "DEAD") || gu.IsLocked || gu.Location != "INVENTORY" {
			continue
		}
		instances = append(instances, gu)
	}
	sort.Slice(instances, func(i, j int) bool {
		if instances[i].GuID == instances[j].GuID {
			return instances[i].Rank < instances[j].Rank
		}
		return instances[i].GuID < instances[j].GuID
	})
	options := make([]killerMoveEffectOption, 0)
	for _, inst := range instances {
		items, optionErr := getKillerMoveEffectOptions(inst)
		if optionErr == nil {
			options = append(options, items...)
		}
	}
	profile := composition.KillerMoveProfile()
	templates, err := listAdminCompositionTemplates(c.Request.Context(), profile.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải composition templates"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"gus": instances, "effect_options": options, "composition_templates": templates, "composition_rule_set": composition.KillerMoveRuleSet(), "max_saved": profile.MaxSaved, "max_equipped": profile.MaxEquipped, "max_components": profile.MaxGuComponents, "max_effects": profile.MaxNodes})
}

func calculateKillerMoveDraft(tx *sql.Tx, playerID uuid.UUID, req killerMoveDraftRequest, masteryXP int) (killerMovePreview, ownedKillerMoveGu, error) {
	productProfile := composition.KillerMoveProfile()
	ruleSet := composition.KillerMoveRuleSet()
	if req.RuleSetRevision != ruleSet.Revision {
		return killerMovePreview{}, ownedKillerMoveGu{}, fmt.Errorf("Bộ quy tắc composition đã thay đổi; hãy tải lại builder và xem trước lại")
	}
	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		return killerMovePreview{}, ownedKillerMoveGu{}, err
	}
	main, exists := owned[req.MainGuInstanceID]
	if !exists || strings.EqualFold(main.State, "DEAD") {
		return killerMovePreview{}, main, fmt.Errorf("Cổ chính không còn thuộc sở hữu của bạn")
	}
	if main.IsLocked || main.Location != "INVENTORY" {
		return killerMovePreview{}, main, fmt.Errorf("Cổ chính đang bị khóa hoặc không ở trong hành trang")
	}
	ids := []string{main.InstanceID}
	seen := map[string]bool{main.InstanceID: true}
	for _, id := range req.ComponentIDs {
		if id == "" || seen[id] {
			continue
		}
		if _, exists := owned[id]; !exists {
			return killerMovePreview{}, main, fmt.Errorf("Cổ thành phần không thuộc sở hữu của bạn")
		}
		if strings.EqualFold(owned[id].State, "DEAD") || owned[id].IsLocked || owned[id].Location != "INVENTORY" {
			return killerMovePreview{}, main, fmt.Errorf("Cổ đã chết, đang bị khóa hoặc không ở trong hành trang")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) > productProfile.MaxGuComponents {
		return killerMovePreview{}, main, fmt.Errorf("Mỗi sát chiêu tối đa %d Cổ thành phần", productProfile.MaxGuComponents)
	}
	if len(req.SelectedEffectKeys) == 0 || len(req.SelectedEffectKeys) > productProfile.MaxNodes {
		return killerMovePreview{}, main, fmt.Errorf("Chọn từ 1 đến %d node effect", productProfile.MaxNodes)
	}
	allowed := make(map[string]killerMoveEffectOption)
	components := make([]killerMoveComponent, 0, len(ids))
	paths := make(map[string]bool)
	pathByGu := make(map[string]string)
	manaCost := 0.0
	for _, id := range ids {
		inst := owned[id]
		spec, ok := models.GetGu(inst.GuID, inst.Rank)
		if !ok {
			return killerMovePreview{}, main, fmt.Errorf("Thiếu dữ liệu Cổ %s rank %d", inst.GuID, inst.Rank)
		}
		components = append(components, killerMoveComponent{InstanceID: id, GuID: inst.GuID, Rank: inst.Rank})
		for _, p := range spec.Path {
			clean := gamefx.CleanPath(p)
			paths[clean] = true
			pathByGu[id] = clean
		}
		if spec.BaseManaCost > 0 {
			manaCost += spec.BaseManaCost
		}
		options, optionErr := getKillerMoveEffectOptions(inst)
		if optionErr == nil {
			for _, option := range options {
				allowed[option.Key] = option
			}
		}
	}
	selected := make([]killerMoveEffectRef, 0, len(req.SelectedEffectKeys))
	blocks := make([]gameplayeffects.Block, 0, len(req.SelectedEffectKeys))
	graphNodes := make([]composition.Node, 0, len(req.SelectedEffectKeys))
	capabilitiesByEffect := make(map[string]map[string]bool, len(req.SelectedEffectKeys))
	load := 0.0
	unscalableLoad := 0.0
	selectedKeys := make(map[string]bool, len(req.SelectedEffectKeys))
	for _, key := range req.SelectedEffectKeys {
		if selectedKeys[key] {
			return killerMovePreview{}, main, fmt.Errorf("Không thể chọn trùng một hiệu ứng")
		}
		selectedKeys[key] = true
		option, ok := allowed[key]
		if !ok {
			return killerMovePreview{}, main, fmt.Errorf("Hiệu ứng không thuộc các Cổ đã chọn hoặc không được hỗ trợ")
		}
		bi, oi := option.BlockIndex, option.OperationIndex
		coefficient := option.PlayerATKCoefficient
		capabilitiesByEffect[key] = make(map[string]bool, len(option.CompositionCapabilities))
		for _, capability := range option.CompositionCapabilities {
			capabilitiesByEffect[key][capability] = true
		}
		selected = append(selected, killerMoveEffectRef{Key: key, SourceBlockID: option.SourceBlockID, SourceRevision: option.SourceRevision, Trigger: option.Trigger, InstanceID: option.SourceID, GuID: option.GuID, Rank: option.Rank, BlockIndex: bi, OpIndex: oi, PlayerATKCoefficient: &coefficient})
		role := composition.NodeSupport
		if key == req.OutputEffectKey {
			role = composition.NodeOutput
		}
		executionContext := productProfile.ContextByTrigger[option.Trigger]
		opIndex := oi
		graphNodes = append(graphNodes, composition.Node{EffectKey: key, SourceBlockID: option.SourceBlockID, SourceRevision: option.SourceRevision, OperationIndex: &opIndex, Role: role, GuInstanceID: option.SourceID, GuID: option.GuID, Rank: option.Rank, Trigger: option.Trigger, Context: executionContext, Executor: productProfile.ExecutorByContext[executionContext]})
		blockTarget := option.Target
		operation := option.Operation
		// Block v2 stores target selection on the block, never on the operation.
		operation.Target = nil
		blocks = append(blocks, gameplayeffects.Block{ID: fmt.Sprintf("EFFECT_%02d", len(blocks)+1), Trigger: gameplayeffects.Trigger(option.Trigger), Target: blockTarget, Conditions: append([]gameplayeffects.Condition(nil), option.Conditions...), Operations: []gameplayeffects.Operation{operation}})
		effectLoad := composition.EffectLoad(option.Operation, blockTarget)
		load += effectLoad
		if option.Operation.Op == gameplayeffects.OpCleanse {
			unscalableLoad += effectLoad
		}
	}
	graph, err := composition.ResolveWithRules(composition.Graph{
		CatalogDraftID: req.CompositionDraftID, Product: productProfile.Product, ProductProfile: productProfile.ID, Context: productProfile.Context, Executor: productProfile.Executor,
		RuleSetRevision:  ruleSet.Revision,
		CoreGuInstanceID: main.InstanceID, OutputEffectKey: req.OutputEffectKey,
		Nodes: graphNodes, Edges: req.CompositionEdges,
	}, productProfile, ids, ruleSet, capabilitiesByEffect)
	if err != nil {
		return killerMovePreview{}, main, err
	}
	selectedIndexByKey := make(map[string]int, len(selected))
	for index, ref := range selected {
		selectedIndexByKey[ref.Key] = index
	}
	ruleByID := make(map[string]composition.CompositionRule, len(ruleSet.Rules))
	for _, rule := range ruleSet.Rules {
		ruleByID[rule.ID] = rule
	}
	for nodeIndex := range graph.Nodes {
		node := &graph.Nodes[nodeIndex]
		blockIndex, ok := selectedIndexByKey[node.EffectKey]
		if !ok {
			return killerMovePreview{}, main, fmt.Errorf("composition output template lost effect provenance")
		}
		prefix := "EFFECT"
		if node.Role == composition.NodeSupport {
			ruleID := ""
			for _, edge := range graph.Edges {
				if edge.From == node.EffectKey {
					ruleID = edge.RuleID
					break
				}
			}
			rule, exists := ruleByID[ruleID]
			if !exists || rule.OutputTemplate.Mode != "APPEND_SOURCE_OPERATION_BLOCKS" {
				return killerMovePreview{}, main, fmt.Errorf("composition rule has no supported Block v2 output template")
			}
			prefix = strings.ToUpper(rule.OutputTemplate.BlockIDPrefix)
		}
		blocks[blockIndex].ID = fmt.Sprintf("%s_%02d", prefix, blockIndex+1)
		node.BlockID = blocks[blockIndex].ID
	}
	if len(paths) == 0 {
		paths[gamefx.CleanPath("strength")] = true
	}
	mainSpec, ok := models.GetGu(main.GuID, main.Rank)
	if !ok {
		return killerMovePreview{}, main, fmt.Errorf("Không tìm thấy Cổ chính trong từ điển")
	}
	path := "strength"
	if len(mainSpec.Path) > 0 {
		path = mainSpec.Path[0]
	}
	pathMasteryLevel := 0
	var rawMasteries []byte
	if err := tx.QueryRow(`SELECT COALESCE(path_masteries,'{}'::jsonb) FROM players WHERE id=$1`, playerID).Scan(&rawMasteries); err == nil {
		masteries := models.PathMasteriesMap{}
		_ = json.Unmarshal(rawMasteries, &masteries)
		pathMasteryLevel = gamefx.PathMasteryLevel(masteries[gamefx.CleanPath(path)])
	}
	componentCount := len(components)
	budget := productProfile.PowerBudgetBase + float64(main.Rank)*productProfile.PowerBudgetPerCoreRank
	for i := 1; i < componentCount; i++ {
		budget += productProfile.PowerBudgetPerSupport / math.Pow(float64(i), productProfile.SupportBudgetExponent)
	}
	if budget <= 0 {
		budget = 30
	}
	complexity := 1 + 0.14*float64(componentCount-1) + 0.12*float64(len(selected)-1) + 0.10*float64(max(0, len(paths)-1))
	harmony := 1.0
	if componentCount > 1 {
		matching := 0
		pairs := componentCount * (componentCount - 1) / 2
		for i := 0; i < len(components); i++ {
			for j := i + 1; j < len(components); j++ {
				if pathByGu[components[i].InstanceID] != "" && pathByGu[components[i].InstanceID] == pathByGu[components[j].InstanceID] {
					matching++
				}
			}
		}
		harmony = 0.6 + 0.4*float64(matching)/float64(pairs)
	}
	scale := 1.0
	if load > budget {
		scalableLoad := load - unscalableLoad
		if unscalableLoad >= budget || scalableLoad <= 0 {
			return killerMovePreview{}, main, fmt.Errorf("Các hiệu ứng không thể tự cân đã vượt ngân sách rank; hãy bỏ bớt effect hoặc chọn Cổ chính rank cao hơn")
		}
		scale = (budget - unscalableLoad) / scalableLoad
	}
	if scale < productProfile.MinimumEffectScale {
		return killerMovePreview{}, main, fmt.Errorf("Công thức vượt ngân sách sức mạnh quá xa; hãy giảm số hiệu ứng hoặc chọn Cổ rank cao hơn")
	}
	for i := range selected {
		if selected[i].PlayerATKCoefficient != nil {
			*selected[i].PlayerATKCoefficient *= scale
		}
	}
	for bi := range blocks {
		for oi := range blocks[bi].Operations {
			composition.ScaleOperation(&blocks[bi].Operations[oi], scale)
		}
	}
	masteryBonus := math.Min(0.15, float64(masteryXP)/1000*0.15)
	success := 0.55 + float64(pathMasteryLevel)*0.035 + (harmony-0.6)*0.18 + masteryBonus - (complexity-1)*0.15
	if success < 0.10 {
		success = 0.10
	}
	if success > 0.95 {
		success = 0.95
	}
	warnings := []string{}
	if scale < 0.999 {
		warnings = append(warnings, fmt.Sprintf("Hiệu ứng được giảm còn %.0f%% để vừa ngân sách rank %d", scale*100, main.Rank))
	}
	if harmony < 0.8 {
		warnings = append(warnings, "Các lưu phái phối hợp chưa hòa hợp; tỉ lệ thành công giảm")
	}
	profile := compileKillerMoveCombatProfile(mainSpec, main, components, selected, blocks, path, pathMasteryLevel, harmony, complexity, success, manaCost)
	preview := killerMovePreview{Components: components, CompositionGraph: graph, SelectedEffects: selected, CompiledEffects: blocks, Rank: main.Rank, Path: path, PathMasteryLevel: pathMasteryLevel, PowerBudget: budget, PowerLoad: load, EffectScale: scale, Harmony: harmony, Complexity: complexity, SuccessChance: success, ManaCost: manaCost, CombatProfile: profile, Warnings: warnings}
	preview.PreviewFingerprint = killerMovePreviewHash(preview)
	return preview, main, nil
}

func getGuTargetType(spec models.Gu) string {
	for _, b := range spec.Effects.GetBlocks() {
		sel := strings.ToUpper(string(b.Target.Selector))
		if strings.Contains(sel, "ALL") {
			return "ALL_ENEMIES"
		}
		if strings.Contains(sel, "ROW") || strings.Contains(sel, "COL") {
			return "ROW"
		}
	}
	return "SINGLE"
}

func compileKillerMoveCombatProfile(
	mainSpec models.Gu,
	main ownedKillerMoveGu,
	components []killerMoveComponent,
	selected []killerMoveEffectRef,
	blocks []gameplayeffects.Block,
	path string,
	pathMasteryLevel int,
	harmony, complexity, success, manaCost float64,
) models.KillerMoveCombatProfile {
	targetType := "SINGLE"
	bypass := false
	allPaths := []string{path}
	seenPaths := map[string]bool{path: true}

	for _, comp := range components {
		if comp.InstanceID == main.InstanceID {
			continue
		}
		spec, ok := models.GetGu(comp.GuID, comp.Rank)
		if !ok {
			continue
		}
		for _, p := range spec.Path {
			clean := gamefx.CleanPath(p)
			if !seenPaths[clean] {
				seenPaths[clean] = true
				allPaths = append(allPaths, clean)
			}
		}
		tgt := getGuTargetType(spec)
		if tgt == "ALL" || tgt == "ALL_ENEMIES" {
			targetType = "ALL_ENEMIES"
		} else if (tgt == "ROW" || tgt == "COL" || tgt == "CROSS") && targetType != "ALL_ENEMIES" {
			targetType = "ROW"
		}
		for _, tag := range spec.CombatTags {
			tagLower := strings.ToLower(tag)
			if strings.Contains(tagLower, "aoe") || strings.Contains(tagLower, "range") {
				if targetType != "ALL_ENEMIES" {
					targetType = "ROW"
				}
			}
			if strings.Contains(tagLower, "pierce") || strings.Contains(tagLower, "penetrat") || strings.Contains(tagLower, "xuyen") {
				bypass = true
			}
		}
	}

	hasTargetEffect, hasSelfEffect := false, false
	for _, block := range blocks {
		for _, operation := range block.Operations {
			target := block.Target
			if operation.Target != nil {
				target = *operation.Target
			}
			if target.Selector == gameplayeffects.TargetSingle {
				hasTargetEffect = true
			}
			if target.Selector == gameplayeffects.TargetSelf {
				hasSelfEffect = true
			}
		}
	}
	if hasTargetEffect || hasSelfEffect {
		targetType = "SELF"
		if hasTargetEffect && hasSelfEffect {
			targetType = "SINGLE+SELF"
		} else if hasTargetEffect {
			targetType = "SINGLE"
		}
	}

	backfirePct := math.Max(0.08, 0.28-float64(pathMasteryLevel)*0.035-(harmony-0.6)*0.20)
	// Backfire currently applies damage only; do not create an unsupported
	// status effect that would make an otherwise valid profile unusable.
	backfireStatus := ""

	return models.KillerMoveCombatProfile{
		ProfileVersion:    2,
		Rank:              main.Rank,
		MainPath:          path,
		AllPaths:          allPaths,
		TargetType:        targetType,
		Bypass:            bypass,
		Priority:          1,
		ManaCost:          manaCost,
		HungerCost:        float64(len(components)),
		SuccessChance:     success,
		BackfireHPPercent: backfirePct,
		BackfireStatus:    backfireStatus,
	}
}

func PreviewKillerMove(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	var req killerMoveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu công thức không hợp lệ"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	if err := bindKillerMoveCompositionDraft(c, tx, playerID, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	preview, _, err := calculateKillerMoveDraft(tx, playerID, req, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, preview)
}

func CreateKillerMove(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	var req killerMoveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu sát chiêu không hợp lệ"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên sát chiêu phải có từ 1 đến 80 ký tự"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	var lockedPlayer string
	if err = tx.QueryRow(`SELECT id::text FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&lockedPlayer); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhân vật"})
		return
	}
	if err := bindKillerMoveCompositionDraft(c, tx, playerID, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	productProfile := composition.KillerMoveProfile()
	var count int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM player_killer_moves WHERE player_id=$1`, playerID).Scan(&count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra giới hạn sát chiêu"})
		return
	}
	if count >= productProfile.MaxSaved {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Mỗi người chơi lưu tối đa %d sát chiêu", productProfile.MaxSaved)})
		return
	}
	preview, main, err := calculateKillerMoveDraft(tx, playerID, req, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.PreviewFingerprint == "" || req.PreviewFingerprint != preview.PreviewFingerprint {
		c.JSON(http.StatusConflict, gin.H{"error": "Bản preview đã thay đổi; hãy xem lại profile mới trước khi lưu"})
		return
	}
	if preview.EffectScale < 0.999 && !req.AcceptAdjustments {
		c.JSON(http.StatusConflict, gin.H{"error": "Hãy xác nhận các effect đã được tự cân bằng trước khi lưu"})
		return
	}
	components, _ := json.Marshal(preview.Components)
	compositionGraph, _ := json.Marshal(preview.CompositionGraph)
	selected, _ := json.Marshal(preview.SelectedEffects)
	compiled, err := json.Marshal(gameplayeffects.BlockSet(preview.CompiledEffects))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể chuẩn hóa Block v2 của sát chiêu"})
		return
	}
	profBytes, _ := json.Marshal(preview.CombatProfile)
	var id string
	err = tx.QueryRow(`INSERT INTO player_killer_moves(player_id,name,main_gu_instance_id,main_gu_id,main_gu_rank,components,composition_graph,selected_effects,compiled_effects,path,creation_path_mastery_level,power_budget,power_load,harmony,complexity,success_chance,balance_version,combat_profile)
	VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8::jsonb,$9::jsonb,$10,$11,$12,$13,$14,$15,$16,$17,$18::jsonb) RETURNING id`, playerID, name, main.InstanceID, main.GuID, main.Rank, components, compositionGraph, selected, compiled, preview.Path, preview.PathMasteryLevel, preview.PowerBudget, preview.PowerLoad, preview.Harmony, preview.Complexity, preview.SuccessChance, killerMoveBalanceVersion, profBytes).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu sát chiêu"})
		return
	}
	auditlog.Log(tx, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "killer_move.create",
		EntityType:     "killer_move",
		EntityID:       id,
		RequestSummary: fmt.Sprintf("Sáng tạo sát chiêu [%s] (%s, tổng lực %.2f/%.2f, hài hòa %.2f, tỉ lệ %.0f%%)", name, preview.Path, preview.PowerLoad, preview.PowerBudget, preview.Harmony, preview.SuccessChance*100),
		Metadata: map[string]interface{}{
			"killer_move_id":  id,
			"name":            name,
			"path":            preview.Path,
			"main_gu_id":      main.GuID,
			"main_gu_rank":    main.Rank,
			"components":      len(preview.Components),
			"power_load":      preview.PowerLoad,
			"power_budget":    preview.PowerBudget,
			"harmony":         preview.Harmony,
			"complexity":      preview.Complexity,
			"success_chance":  preview.SuccessChance,
			"balance_version": killerMoveBalanceVersion,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu sát chiêu"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Đã sáng tạo sát chiêu"})
}

func ListKillerMoves(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id::text,name,main_gu_instance_id,main_gu_id,main_gu_rank,components,composition_graph,selected_effects,compiled_effects,path,creation_path_mastery_level,power_budget,power_load,harmony,complexity,success_chance,mastery_xp,is_equipped,balance_version,COALESCE(combat_profile,'{}'::jsonb) FROM player_killer_moves WHERE player_id=$1 ORDER BY created_at DESC`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải sát chiêu"})
		return
	}
	defer rows.Close()
	owned, err := db.DB.QueryContext(c.Request.Context(), `SELECT id,gu_id,gu_rank,hunger,state,location,is_locked FROM player_gu WHERE player_id=$1`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh Cổ thành phần"})
		return
	}
	ownedSet := map[string]ownedKillerMoveGu{}
	for owned.Next() {
		var gu ownedKillerMoveGu
		if err := owned.Scan(&gu.InstanceID, &gu.GuID, &gu.Rank, &gu.Hunger, &gu.State, &gu.Location, &gu.IsLocked); err == nil {
			ownedSet[gu.InstanceID] = gu
		}
	}
	owned.Close()
	var rawMasteries []byte
	_ = db.DB.QueryRowContext(c.Request.Context(), `SELECT COALESCE(path_masteries,'{}'::jsonb) FROM players WHERE id=$1`, playerID).Scan(&rawMasteries)
	masteries := models.PathMasteriesMap{}
	_ = json.Unmarshal(rawMasteries, &masteries)
	result := make([]storedKillerMove, 0)
	for rows.Next() {
		var m storedKillerMove
		var components, compositionGraph, selected, compiled, combatProf []byte
		if err := rows.Scan(&m.ID, &m.Name, &m.MainGuInstance, &m.MainGuID, &m.MainGuRank, &components, &compositionGraph, &selected, &compiled, &m.Path, &m.CreationPathMasteryLevel, &m.PowerBudget, &m.PowerLoad, &m.Harmony, &m.Complexity, &m.SuccessChance, &m.MasteryXP, &m.Equipped, &m.BalanceVersion, &combatProf); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc sát chiêu"})
			return
		}
		var snapshotErr error
		if err := json.Unmarshal(components, &m.Components); err != nil {
			snapshotErr = fmt.Errorf("provenance Cổ thành phần không hợp lệ")
		}
		if err := json.Unmarshal(compositionGraph, &m.CompositionGraph); err != nil && snapshotErr == nil {
			snapshotErr = fmt.Errorf("composition graph không hợp lệ")
		}
		if err := json.Unmarshal(selected, &m.SelectedEffects); err != nil && snapshotErr == nil {
			snapshotErr = fmt.Errorf("provenance effect không hợp lệ")
		}
		blocks, err := gameplayeffects.DecodeBlockSet(compiled)
		if err != nil {
			if snapshotErr == nil {
				snapshotErr = fmt.Errorf("Block v2 đã lưu không hợp lệ")
			}
		} else {
			m.CompiledEffects = []gameplayeffects.Block(blocks)
		}
		_ = json.Unmarshal(combatProf, &m.CombatProfile)
		m.MasteryLevel = m.MasteryXP / 100
		currentLevel := gamefx.PathMasteryLevel(masteries[gamefx.CleanPath(m.Path)])
		m.SuccessChance = composition.AdjustedSuccessChance(m.SuccessChance, m.CreationPathMasteryLevel, currentLevel, m.MasteryXP)
		available, hasStarvingGu, missingGu := evaluateKillerMoveComponents(m.Components, ownedSet)
		compositionErr := snapshotErr
		if compositionErr == nil {
			compositionErr = validateKillerMoveCompositionSnapshotMode(m.CompositionGraph, m.SelectedEffects, m.Components, m.CompiledEffects, !m.Equipped)
		}
		m.Available = available && compositionErr == nil
		m.HasStarvingGu = hasStarvingGu
		m.MissingGu = missingGu
		if compositionErr != nil {
			m.UnavailableReason = "Composition graph hoặc Block v2 không còn khớp catalog hiện hành: " + compositionErr.Error()
		}
		if !m.Available {
			canRepair := true
			for _, comp := range m.Components {
				found := false
				for _, g := range ownedSet {
					if g.GuID == comp.GuID && !strings.EqualFold(g.State, "DEAD") && !g.IsLocked && g.Location == "INVENTORY" {
						found = true
						break
					}
				}
				if !found {
					canRepair = false
					break
				}
			}
			m.CanRepair = canRepair
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc sát chiêu"})
		return
	}
	productProfile := composition.KillerMoveProfile()
	c.JSON(http.StatusOK, gin.H{"killer_moves": result, "max_saved": productProfile.MaxSaved, "max_equipped": productProfile.MaxEquipped})
}

func EquipKillerMove(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	moveID := c.Param("id")
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	var lockedPlayer string
	if err = tx.QueryRow(`SELECT id::text FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&lockedPlayer); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhân vật"})
		return
	}
	var equipped bool
	var raw, graphRaw, selectedRaw, compiledRaw []byte
	err = tx.QueryRow(`SELECT is_equipped,components,composition_graph,selected_effects,compiled_effects FROM player_killer_moves WHERE id=$1 AND player_id=$2 FOR UPDATE`, moveID, playerID).Scan(&equipped, &raw, &graphRaw, &selectedRaw, &compiledRaw)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy sát chiêu"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc sát chiêu"})
		return
	}
	var components []killerMoveComponent
	productProfile := composition.KillerMoveProfile()
	if !equipped {
		var graph composition.Graph
		var selected []killerMoveEffectRef
		if err := json.Unmarshal(raw, &components); err != nil || json.Unmarshal(graphRaw, &graph) != nil || json.Unmarshal(selectedRaw, &selected) != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Provenance composition của sát chiêu không hợp lệ; hãy tạo lại sát chiêu"})
			return
		}
		compiledBlocks, err := gameplayeffects.DecodeBlockSet(compiledRaw)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Block v2 của sát chiêu không hợp lệ; hãy tạo lại sát chiêu"})
			return
		}
		needsReconcile := false
		if err := validateKillerMoveCompositionSnapshot(graph, selected, components, []gameplayeffects.Block(compiledBlocks)); err != nil {
			needsReconcile = true
		} else {
			for _, component := range components {
				var guID, state, location string
				var rank int
				var isLocked bool
				err = tx.QueryRow(`SELECT gu_id,gu_rank,state,location,is_locked FROM player_gu WHERE id=$1 AND player_id=$2 FOR UPDATE`, component.InstanceID, playerID).Scan(&guID, &rank, &state, &location, &isLocked)
				if err != nil || strings.EqualFold(state, "DEAD") || location != "INVENTORY" || isLocked || guID != component.GuID || rank != component.Rank {
					needsReconcile = true
					break
				}
			}
		}
		if needsReconcile {
			reconciled, rErr := reconcileKillerMoveTx(tx, playerID, moveID)
			if rErr != nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Không thể tự động đồng bộ Cổ thành phần cho sát chiêu: " + rErr.Error()})
				return
			}
			components = reconciled.Components
		}
	}
	if !equipped {
		if productProfile.GuPolicy == composition.GuPolicyReserveActive {
			for _, component := range components {
				var reservedByArray string
				arrayReservationErr := tx.QueryRow(`SELECT array_id::text FROM teleport_array_gu_reservations WHERE gu_instance_id=$1`, component.InstanceID).Scan(&reservedByArray)
				if arrayReservationErr == nil {
					c.JSON(http.StatusConflict, gin.H{"error": "Cổ thành phần đang được Trận Pháp trên bản đồ giữ chỗ"})
					return
				}
				if arrayReservationErr != sql.ErrNoRows {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra giữ chỗ Trận Pháp"})
					return
				}
				var reservedByHouse string
				houseReservationErr := tx.QueryRow(`SELECT gu_house_id::text FROM player_gu_house_core_reservations WHERE gu_instance_id=$1`, component.InstanceID).Scan(&reservedByHouse)
				if houseReservationErr == nil {
					c.JSON(http.StatusConflict, gin.H{"error": "Cổ thành phần đang được Tiên Cổ Ốc active giữ chỗ"})
					return
				}
				if houseReservationErr != sql.ErrNoRows {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra giữ chỗ Tiên Cổ Ốc"})
					return
				}
				var reservedBy string
				reservationErr := tx.QueryRow(`SELECT killer_move_id::text FROM player_killer_move_gu_reservations WHERE gu_instance_id=$1`, component.InstanceID).Scan(&reservedBy)
				if reservationErr == nil && reservedBy != moveID {
					c.JSON(http.StatusConflict, gin.H{"error": "Cổ thành phần đang được một Sát Chiêu đã trang bị giữ chỗ"})
					return
				}
				if reservationErr != nil && reservationErr != sql.ErrNoRows {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra giữ chỗ Cổ thành phần"})
					return
				}
				var inLegacyEquippedMove bool
				if err = tx.QueryRow(`SELECT EXISTS (
					SELECT 1 FROM player_killer_moves AS other_move,
					jsonb_array_elements(other_move.components) AS component
					WHERE other_move.player_id=$1 AND other_move.id<>$2 AND other_move.is_equipped=true
					AND component->>'instance_id'=$3
				)`, playerID, moveID, component.InstanceID).Scan(&inLegacyEquippedMove); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra Sát Chiêu đang hoạt động"})
					return
				}
				if inLegacyEquippedMove {
					c.JSON(http.StatusConflict, gin.H{"error": "Cổ thành phần đã thuộc Sát Chiêu khác đang trang bị"})
					return
				}
				var heldBy string
				err = tx.QueryRow(`INSERT INTO player_killer_move_gu_reservations(gu_instance_id,killer_move_id,player_id,gu_id,gu_rank,gu_location,gu_is_locked)
					VALUES($1,$2,$3,$4,$5,$6,false)
					ON CONFLICT (gu_instance_id) DO UPDATE SET created_at=player_killer_move_gu_reservations.created_at
					WHERE player_killer_move_gu_reservations.killer_move_id=EXCLUDED.killer_move_id
					RETURNING killer_move_id::text`, component.InstanceID, moveID, playerID, component.GuID, component.Rank, "INVENTORY").Scan(&heldBy)
				if err != nil || heldBy != moveID {
					c.JSON(http.StatusConflict, gin.H{"error": "Không thể giữ chỗ Cổ thành phần cho Sát Chiêu"})
					return
				}
			}
		}
		var count int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM player_killer_moves WHERE player_id=$1 AND is_equipped=true`, playerID).Scan(&count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra ô trang bị"})
			return
		}
		if count >= productProfile.MaxEquipped {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Chỉ trang bị tối đa %d sát chiêu", productProfile.MaxEquipped)})
			return
		}
	}
	if _, err = tx.Exec(`UPDATE player_killer_moves SET is_equipped=$1,updated_at=NOW() WHERE id=$2 AND player_id=$3`, !equipped, moveID, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trang bị"})
		return
	}
	if equipped && productProfile.GuPolicy == composition.GuPolicyReserveActive {
		if _, err = tx.Exec(`DELETE FROM player_killer_move_gu_reservations WHERE killer_move_id=$1 AND player_id=$2`, moveID, playerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể giải phóng Cổ thành phần"})
			return
		}
	}
	equipAction := "trang bị"
	if equipped {
		equipAction = "tháo trang bị"
	}
	auditlog.Log(tx, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "killer_move.equip",
		EntityType:     "killer_move",
		EntityID:       moveID,
		RequestSummary: fmt.Sprintf("%s sát chiêu %s (%d Cổ thành phần)", equipAction, moveID, len(components)),
		Metadata: map[string]interface{}{
			"killer_move_id": moveID,
			"equipped":       !equipped,
			"components":     len(components),
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})
	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trang bị"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"is_equipped": !equipped})
}

func DeleteKillerMove(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	moveID := c.Param("id")
	res, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM player_killer_moves WHERE id=$1 AND player_id=$2`, moveID, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa sát chiêu"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy sát chiêu"})
		return
	}
	// ponytail: handler has no transaction; audit is written via db.DB after the delete.
	auditlog.Log(db.DB, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "killer_move.delete",
		EntityType:     "killer_move",
		EntityID:       moveID,
		RequestSummary: fmt.Sprintf("Xóa sát chiêu %s", moveID),
		Metadata: map[string]interface{}{
			"killer_move_id": moveID,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa sát chiêu"})
}

func RepairKillerMove(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	moveID := c.Param("id")
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var lockedPlayer string
	if err = tx.QueryRow(`SELECT id::text FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&lockedPlayer); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhân vật"})
		return
	}

	reconciled, err := reconcileKillerMoveTx(tx, playerID, moveID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi xác nhận"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Đã tu sửa và đồng bộ sát chiêu thành công!",
		"killer_move": reconciled,
	})
}

func reconcileKillerMoveTx(tx *sql.Tx, playerID uuid.UUID, moveID string) (storedKillerMove, error) {
	var name, path string
	var mainGuInstanceID, mainGuID string
	var mainGuRank, masteryXP, balanceVersion int
	var isEquipped bool
	var rawComponents, rawGraph, rawSelected, rawCompiled, rawCombatProf []byte

	err := tx.QueryRow(`
		SELECT name, main_gu_instance_id, main_gu_id, main_gu_rank, components, composition_graph, selected_effects, compiled_effects, path, mastery_xp, is_equipped, balance_version, COALESCE(combat_profile,'{}'::jsonb)
		FROM player_killer_moves
		WHERE id=$1 AND player_id=$2 FOR UPDATE
	`, moveID, playerID).Scan(&name, &mainGuInstanceID, &mainGuID, &mainGuRank, &rawComponents, &rawGraph, &rawSelected, &rawCompiled, &path, &masteryXP, &isEquipped, &balanceVersion, &rawCombatProf)
	if err != nil {
		return storedKillerMove{}, fmt.Errorf("không tìm thấy sát chiêu")
	}
	if isEquipped {
		return storedKillerMove{}, fmt.Errorf("Sát chiêu active đang pin snapshot effect; hãy tháo trang bị trước khi đồng bộ draft theo catalog mới")
	}

	var components []killerMoveComponent
	var graph composition.Graph
	var selected []killerMoveEffectRef

	if err := json.Unmarshal(rawComponents, &components); err != nil {
		return storedKillerMove{}, fmt.Errorf("lỗi đọc danh sách Cổ thành phần")
	}
	if err := json.Unmarshal(rawGraph, &graph); err != nil {
		return storedKillerMove{}, fmt.Errorf("lỗi đọc composition graph")
	}
	if err := json.Unmarshal(rawSelected, &selected); err != nil {
		return storedKillerMove{}, fmt.Errorf("lỗi đọc danh sách effect")
	}

	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		return storedKillerMove{}, fmt.Errorf("không thể tải Cổ trong hành trang: %w", err)
	}

	instMap := make(map[string]string)
	usedInstances := make(map[string]bool)

	// 1. Reconcile Main Gu
	if g, ok := owned[mainGuInstanceID]; ok && g.GuID == mainGuID && strings.EqualFold(g.State, "ACTIVE") && g.Location == "INVENTORY" && !g.IsLocked {
		instMap[mainGuInstanceID] = mainGuInstanceID
		usedInstances[mainGuInstanceID] = true
	} else {
		var bestCandidate *ownedKillerMoveGu
		for _, g := range owned {
			if g.GuID == mainGuID && strings.EqualFold(g.State, "ACTIVE") && g.Location == "INVENTORY" && !g.IsLocked && !usedInstances[g.InstanceID] {
				if bestCandidate == nil || g.Rank > bestCandidate.Rank {
					candidate := g
					bestCandidate = &candidate
				}
			}
		}
		if bestCandidate == nil {
			return storedKillerMove{}, fmt.Errorf("không tìm thấy Cổ chính [%s] còn sống trong hành trang", mainGuID)
		}
		instMap[mainGuInstanceID] = bestCandidate.InstanceID
		usedInstances[bestCandidate.InstanceID] = true
	}

	// 2. Reconcile each component
	for _, comp := range components {
		oldID := comp.InstanceID
		if instMap[oldID] != "" {
			continue
		}
		if g, ok := owned[oldID]; ok && g.GuID == comp.GuID && strings.EqualFold(g.State, "ACTIVE") && g.Location == "INVENTORY" && !g.IsLocked && !usedInstances[oldID] {
			instMap[oldID] = oldID
			usedInstances[oldID] = true
		} else {
			var bestCandidate *ownedKillerMoveGu
			for _, g := range owned {
				if g.GuID == comp.GuID && strings.EqualFold(g.State, "ACTIVE") && g.Location == "INVENTORY" && !g.IsLocked && !usedInstances[g.InstanceID] {
					if bestCandidate == nil || g.Rank > bestCandidate.Rank {
						candidate := g
						bestCandidate = &candidate
					}
				}
			}
			if bestCandidate == nil {
				return storedKillerMove{}, fmt.Errorf("không tìm thấy Cổ thành phần [%s] trong hành trang để đồng bộ", comp.GuID)
			}
			instMap[oldID] = bestCandidate.InstanceID
			usedInstances[bestCandidate.InstanceID] = true
		}
	}

	// 3. Map effect keys (instance:block:op)
	keyMap := make(map[string]string)
	newSelectedEffectKeys := make([]string, 0, len(selected))
	for _, ref := range selected {
		newInstID := instMap[ref.InstanceID]
		if newInstID == "" {
			newInstID = ref.InstanceID
		}
		newKey := killerMoveEffectKey(newInstID, ref.BlockIndex, ref.OpIndex)
		keyMap[ref.Key] = newKey
		newSelectedEffectKeys = append(newSelectedEffectKeys, newKey)
	}

	// 4. Map composition edges
	newEdges := make([]composition.Edge, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		newFrom := keyMap[edge.From]
		newTo := keyMap[edge.To]
		if newFrom != "" && newTo != "" {
			newEdges = append(newEdges, composition.Edge{From: newFrom, To: newTo, RuleID: edge.RuleID})
		}
	}

	// 5. Output key & Component IDs
	newOutputKey := keyMap[graph.OutputEffectKey]
	if newOutputKey == "" && len(newSelectedEffectKeys) > 0 {
		newOutputKey = newSelectedEffectKeys[0]
	}

	newMainInstanceID := instMap[mainGuInstanceID]
	newComponentIDs := make([]string, 0, len(components))
	for _, comp := range components {
		newID := instMap[comp.InstanceID]
		if newID != "" && newID != newMainInstanceID {
			newComponentIDs = append(newComponentIDs, newID)
		}
	}

	req := killerMoveDraftRequest{
		Name:               name,
		MainGuInstanceID:   newMainInstanceID,
		ComponentIDs:       newComponentIDs,
		SelectedEffectKeys: newSelectedEffectKeys,
		OutputEffectKey:    newOutputKey,
		RuleSetRevision:    composition.KillerMoveRuleSet().Revision,
		CompositionEdges:   newEdges,
		AcceptAdjustments:  true,
	}

	preview, mainGu, err := calculateKillerMoveDraft(tx, playerID, req, masteryXP)
	if err != nil {
		return storedKillerMove{}, fmt.Errorf("lỗi tính toán lại sát chiêu: %w", err)
	}

	componentsBytes, _ := json.Marshal(preview.Components)
	compositionGraphBytes, _ := json.Marshal(preview.CompositionGraph)
	selectedBytes, _ := json.Marshal(preview.SelectedEffects)
	compiledBytes, _ := json.Marshal(gameplayeffects.BlockSet(preview.CompiledEffects))
	profBytes, _ := json.Marshal(preview.CombatProfile)

	_, err = tx.Exec(`
		UPDATE player_killer_moves
		SET main_gu_instance_id = $1,
			main_gu_id = $2,
			main_gu_rank = $3,
			components = $4::jsonb,
			composition_graph = $5::jsonb,
			selected_effects = $6::jsonb,
			compiled_effects = $7::jsonb,
			path = $8,
			power_budget = $9,
			power_load = $10,
			harmony = $11,
			complexity = $12,
			success_chance = $13,
			combat_profile = $14::jsonb,
			balance_version = $15,
			updated_at = NOW()
		WHERE id = $16 AND player_id = $17
	`, mainGu.InstanceID, mainGu.GuID, mainGu.Rank, componentsBytes, compositionGraphBytes, selectedBytes, compiledBytes, preview.Path, preview.PowerBudget, preview.PowerLoad, preview.Harmony, preview.Complexity, preview.SuccessChance, profBytes, killerMoveBalanceVersion, moveID, playerID)
	if err != nil {
		return storedKillerMove{}, fmt.Errorf("không thể cập nhật sát chiêu vào cơ sở dữ liệu: %w", err)
	}

	if isEquipped && composition.KillerMoveProfile().GuPolicy == composition.GuPolicyReserveActive {
		_, _ = tx.Exec(`DELETE FROM player_killer_move_gu_reservations WHERE killer_move_id = $1 AND player_id = $2`, moveID, playerID)
		for _, comp := range preview.Components {
			_, _ = tx.Exec(`
				INSERT INTO player_killer_move_gu_reservations(gu_instance_id, killer_move_id, player_id, gu_id, gu_rank, gu_location, gu_is_locked)
				VALUES($1, $2, $3, $4, $5, 'INVENTORY', false)
				ON CONFLICT (gu_instance_id) DO UPDATE SET updated_at = NOW()
			`, comp.InstanceID, moveID, playerID, comp.GuID, comp.Rank)
		}
	}

	return storedKillerMove{
		ID:                       moveID,
		Name:                     name,
		MainGuInstance:           mainGu.InstanceID,
		MainGuID:                 mainGu.GuID,
		MainGuRank:               mainGu.Rank,
		Components:               preview.Components,
		CompositionGraph:         preview.CompositionGraph,
		SelectedEffects:          preview.SelectedEffects,
		CompiledEffects:          preview.CompiledEffects,
		Path:                     preview.Path,
		CreationPathMasteryLevel: preview.PathMasteryLevel,
		PowerBudget:              preview.PowerBudget,
		PowerLoad:                preview.PowerLoad,
		Harmony:                  preview.Harmony,
		Complexity:               preview.Complexity,
		SuccessChance:            preview.SuccessChance,
		CombatProfile:            preview.CombatProfile,
		MasteryXP:                masteryXP,
		MasteryLevel:             masteryXP / 100,
		Equipped:                 isEquipped,
		Available:                true,
		BalanceVersion:           killerMoveBalanceVersion,
	}, nil
}

// UseKillerMove executes the intentionally narrow out-of-combat subset:
// self-directed HP/mana recovery and the existing world-map flight status.
// Other selected effects remain unavailable until a context owner defines
// their world semantics.
func UseKillerMove(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()
	player, err := evaluateManaTx(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải nhân vật"})
		return
	}
	var move storedKillerMove
	var componentsRaw, graphRaw, selectedRaw, effectsRaw []byte
	err = tx.QueryRow(`SELECT id::text,name,main_gu_instance_id,main_gu_id,main_gu_rank,components,composition_graph,selected_effects,compiled_effects,path,creation_path_mastery_level,success_chance,mastery_xp FROM player_killer_moves WHERE id=$1 AND player_id=$2 AND is_equipped=true FOR UPDATE`, c.Param("id"), playerID).Scan(&move.ID, &move.Name, &move.MainGuInstance, &move.MainGuID, &move.MainGuRank, &componentsRaw, &graphRaw, &selectedRaw, &effectsRaw, &move.Path, &move.CreationPathMasteryLevel, &move.SuccessChance, &move.MasteryXP)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sát chiêu chưa được trang bị hoặc không tồn tại"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải sát chiêu"})
		return
	}
	if err := json.Unmarshal(componentsRaw, &move.Components); err != nil || json.Unmarshal(graphRaw, &move.CompositionGraph) != nil || json.Unmarshal(selectedRaw, &move.SelectedEffects) != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Provenance composition của sát chiêu không hợp lệ"})
		return
	}
	blocks, err := gameplayeffects.DecodeBlockSet(effectsRaw)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Block v2 của sát chiêu đã lưu không hợp lệ"})
		return
	}
	move.CompiledEffects = []gameplayeffects.Block(blocks)
	if err := validateKillerMoveCompositionSnapshotMode(move.CompositionGraph, move.SelectedEffects, move.Components, move.CompiledEffects, false); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Composition graph của sát chiêu không còn hợp lệ; hãy tạo lại từ preview"})
		return
	}
	owned := make(map[string]*models.GuInstance, len(player.GuInventory))
	for i := range player.GuInventory {
		owned[player.GuInventory[i].InstanceID] = &player.GuInventory[i]
	}
	for _, component := range move.Components {
		gu := owned[component.InstanceID]
		if gu == nil || !killerMoveComponentMatches(component, *gu) {
			c.JSON(http.StatusConflict, gin.H{"error": "Sát chiêu thiếu đúng Cổ thành phần còn sống"})
			return
		}
		if gu.Hunger < 1 {
			c.JSON(http.StatusConflict, gin.H{"error": "Cổ thành phần đã kiệt sức, cần cho ăn trước"})
			return
		}
	}
	if err := checkRegionAction(player.CurrentRegion, "gu_use"); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	mainSpec, found := models.GetGu(move.MainGuID, move.MainGuRank)
	if !found {
		c.JSON(http.StatusConflict, gin.H{"error": "Không tìm thấy Cổ chính trong từ điển"})
		return
	}
	mainSpec.BaseManaCost = 0
	for _, component := range move.Components {
		spec, exists := models.GetGu(component.GuID, component.Rank)
		if !exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Không tìm thấy dữ liệu Cổ thành phần"})
			return
		}
		mainSpec.BaseManaCost += spec.BaseManaCost
		if err := checkRegionGu(player.CurrentRegion, spec.ID, spec.Rank, spec.Path); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
	}
	compiledEffects, err := (gameplayeffects.Compiler{}).Compile(move.CompiledEffects)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Effect sát chiêu không còn hợp lệ"})
		return
	}
	canUse := false
	for _, block := range move.CompiledEffects {
		if block.Trigger != gameplayeffects.TriggerOnUse {
			continue
		}
		for _, op := range block.Operations {
			target := block.Target
			if op.Target != nil {
				target = *op.Target
			}
			if worldUseOperationAllowed(op, target) {
				canUse = true
			}
		}
	}
	if !canUse {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sát chiêu không có effect ON_USE được hỗ trợ ngoài combat"})
		return
	}
	componentIDs := make([]string, len(move.Components))
	for i, comp := range move.Components {
		componentIDs[i] = comp.InstanceID
	}
	cost := gamefx.EnergyCost(&player, &mainSpec)
	if player.CurrentMana < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}
	cleanPath := gamefx.CleanPath(move.Path)
	if cleanPath == "" {
		cleanPath = "strength"
	}
	currentLevel := gamefx.PathMasteryLevel(player.PathMasteries[cleanPath])
	chance := composition.AdjustedSuccessChance(move.SuccessChance, move.CreationPathMasteryLevel, currentLevel, move.MasteryXP)
	roll := rand.Float64()
	success := roll < chance
	message := ""
	pathGain := 0
	pathPoints := player.PathMasteries[cleanPath]
	if success {
		evalCtx := gameplayeffects.Context{
			Rank: move.MainGuRank, Mastery: currentLevel, HasMastery: true,
		}
		outcome, execErr := executeWorldUseEffect(tx, &player, compiledEffects, evalCtx, move.MainGuRank, cost, move.Name, componentIDs)
		if execErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": execErr.Error()})
			return
		}
		message = fmt.Sprintf("Sát chiêu [%s] thành công: %s.", move.Name, outcome.EffectsSummary)
		if _, err := tx.Exec(`UPDATE player_killer_moves SET mastery_xp=LEAST(5000,mastery_xp+1),updated_at=NOW() WHERE id=$1 AND player_id=$2`, move.ID, playerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi nhận thành thạo sát chiêu"})
			return
		}
		// Sát chiêu thành công cộng Chân Ý cho path hạch tâm, cùng bảng rank
		// với cast Cổ. Tính theo cảnh giới hiện tại để giữ hiệu suất.
		pathGain = gamefx.CalculatePathMasteryGain(move.MainGuRank, currentLevel, gamefx.PathMasteryGuUseBaseGain(move.MainGuRank))
		if pathGain > 0 {
			if player.PathMasteries == nil {
				player.PathMasteries = make(models.PathMasteriesMap)
			}
			player.PathMasteries[cleanPath] += pathGain
			pathPoints = player.PathMasteries[cleanPath]
			message += fmt.Sprintf(" Ngộ Đạo tăng tiến! +%d điểm Chân Ý lưu phái %s (Cảnh giới: %s).", pathGain, strings.Title(cleanPath), gamefx.PathMasteryLevelName(gamefx.PathMasteryLevel(pathPoints)))
		}
	} else {
		player.CurrentMana -= cost
		if err := deductGuHungerTx(tx, playerID, &player, componentIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể giảm độ no Cổ thành phần"})
			return
		}
		message = fmt.Sprintf("Thôi động [%s] thất bại (%.0f%% / tỉ lệ %.0f%%). Các Cổ thành phần vẫn hao độ no.", move.Name, roll*100, chance*100)
	}
	if _, err := tx.Exec(`UPDATE players SET hp=$1,current_mana=$2,path_masteries=$3,statuses=$4,version=version+1,state_revision=state_revision+1 WHERE id=$5`, player.HP, player.CurrentMana, player.PathMasteries, player.Statuses, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu trạng thái nhân vật"})
		return
	}
	auditlog.Log(tx, auditlog.Entry{
		PlayerID:       &playerID,
		Action:         "killer_move.use",
		EntityType:     "killer_move",
		EntityID:       move.ID,
		RequestSummary: fmt.Sprintf("Thôi động sát chiêu [%s]: %s (mana -%.0f, tỉ lệ %.0f%%, roll %.0f%%)", move.Name, message, cost, chance*100, roll*100),
		Metadata: map[string]interface{}{
			"killer_move_id": move.ID,
			"name":           move.Name,
			"path":           move.Path,
			"success":        success,
			"success_chance": chance,
			"roll":           roll,
			"mana_cost":      cost,
			"hp_after":       player.HP,
			"mana_after":     player.CurrentMana,
			"components":     len(move.Components),
			"path_gain":      pathGain,
			"path_points":    pathPoints,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất sát chiêu"})
		return
	}
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": message, "killer_move_id": move.ID, "success": success, "success_chance": chance, "path": cleanPath, "path_gain": pathGain, "path_masteries": player.PathMasteries})
}
