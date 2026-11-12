package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

const (
	killerMoveBalanceVersion = 2
)

type killerMoveComponent struct {
	InstanceID string `json:"instance_id"`
	GuID       string `json:"gu_id"`
	Rank       int    `json:"rank"`
}

func killerMoveComponentMatches(component killerMoveComponent, gu models.GuInstance) bool {
	return gu.InstanceID == component.InstanceID && gu.GuID == component.GuID && gu.Rank == component.Rank && !strings.EqualFold(gu.Status, "DEAD")
}

func evaluateKillerMoveComponents(components []killerMoveComponent, ownedSet map[string]ownedKillerMoveGu) (bool, bool, []string) {
	available := true
	hasStarvingGu := false
	var missingGu []string
	for _, component := range components {
		gu, exists := ownedSet[component.InstanceID]
		if !exists || strings.EqualFold(gu.State, "DEAD") || gu.IsLocked || gu.Location != "INVENTORY" || gu.GuID != component.GuID || gu.Rank != component.Rank {
			available = false
			missingGu = append(missingGu, component.InstanceID)
			continue
		}
		if gu.Hunger < 1 {
			hasStarvingGu = true
		}
	}
	if len(missingGu) > 0 {
		available = false
	}
	return available, hasStarvingGu, missingGu
}

type killerMoveEffectRef struct {
	Key                  string   `json:"key"`
	SourceBlockID        string   `json:"source_block_id"`
	Trigger              string   `json:"trigger"`
	InstanceID           string   `json:"instance_id"`
	GuID                 string   `json:"gu_id"`
	Rank                 int      `json:"rank"`
	BlockIndex           int      `json:"block_index"`
	OpIndex              int      `json:"operation_index"`
	PlayerATKCoefficient *float64 `json:"player_atk_coefficient,omitempty"`
}

type killerMoveEffectOption struct {
	Key                     string                      `json:"key"`
	SourceBlockID           string                      `json:"source_block_id"`
	BlockIndex              int                         `json:"block_index"`
	OperationIndex          int                         `json:"operation_index"`
	SourceID                string                      `json:"source_instance_id"`
	SourceName              string                      `json:"source_name"`
	GuID                    string                      `json:"gu_id"`
	Rank                    int                         `json:"rank"`
	Label                   string                      `json:"label"`
	Trigger                 string                      `json:"trigger"`
	Operation               gameplayeffects.Operation   `json:"operation"`
	Target                  gameplayeffects.Target      `json:"target"`
	Conditions              []gameplayeffects.Condition `json:"conditions,omitempty"`
	PlayerATKCoefficient    float64                     `json:"player_atk_coefficient"`
	CompositionCapabilities []string                    `json:"composition_capabilities,omitempty"`
	CombatSupported         bool                        `json:"combat_supported"`
	TravelSupported         bool                        `json:"travel_supported,omitempty"`
	UnavailableReason       string                      `json:"unavailable_reason,omitempty"`
}

type killerMoveDraftRequest struct {
	Name               string             `json:"name"`
	MainGuInstanceID   string             `json:"main_gu_instance_id" binding:"required"`
	ComponentIDs       []string           `json:"component_instance_ids"`
	SelectedEffectKeys []string           `json:"selected_effect_keys"`
	OutputEffectKey    string             `json:"output_effect_key"`
	RuleSetRevision    string             `json:"rule_set_revision"`
	CompositionEdges   []composition.Edge `json:"composition_edges"`
	PreviewFingerprint string             `json:"preview_fingerprint,omitempty"`
	AcceptAdjustments  bool               `json:"accept_adjustments,omitempty"`
}

type killerMovePreview struct {
	Components         []killerMoveComponent          `json:"components"`
	CompositionGraph   composition.Graph              `json:"composition_graph"`
	SelectedEffects    []killerMoveEffectRef          `json:"selected_effects"`
	CompiledEffects    []gameplayeffects.Block        `json:"compiled_effects"`
	Rank               int                            `json:"rank"`
	Path               string                         `json:"path"`
	PathMasteryLevel   int                            `json:"path_mastery_level"`
	PowerBudget        float64                        `json:"power_budget"`
	PowerLoad          float64                        `json:"power_load"`
	EffectScale        float64                        `json:"effect_scale"`
	Harmony            float64                        `json:"harmony"`
	Complexity         float64                        `json:"complexity"`
	SuccessChance      float64                        `json:"success_chance"`
	ManaCost           float64                        `json:"mana_cost"`
	CombatProfile      models.KillerMoveCombatProfile `json:"combat_profile"`
	Warnings           []string                       `json:"warnings"`
	PreviewFingerprint string                         `json:"preview_fingerprint"`
}

type storedKillerMove struct {
	ID                       string                         `json:"id"`
	Name                     string                         `json:"name"`
	MainGuInstance           string                         `json:"main_gu_instance_id"`
	MainGuID                 string                         `json:"main_gu_id"`
	MainGuRank               int                            `json:"main_gu_rank"`
	Components               []killerMoveComponent          `json:"components"`
	CompositionGraph         composition.Graph              `json:"composition_graph"`
	SelectedEffects          []killerMoveEffectRef          `json:"selected_effects"`
	CompiledEffects          []gameplayeffects.Block        `json:"compiled_effects"`
	Path                     string                         `json:"path"`
	CreationPathMasteryLevel int                            `json:"creation_path_mastery_level"`
	PowerBudget              float64                        `json:"power_budget"`
	PowerLoad                float64                        `json:"power_load"`
	Harmony                  float64                        `json:"harmony"`
	Complexity               float64                        `json:"complexity"`
	SuccessChance            float64                        `json:"success_chance"`
	CombatProfile            models.KillerMoveCombatProfile `json:"combat_profile"`
	MasteryXP                int                            `json:"mastery_xp"`
	MasteryLevel             int                            `json:"mastery_level"`
	Equipped                 bool                           `json:"is_equipped"`
	Available                bool                           `json:"available"`
	CanRepair                bool                           `json:"can_repair"`
	UnavailableReason        string                         `json:"unavailable_reason,omitempty"`
	HasStarvingGu            bool                           `json:"has_starving_gu,omitempty"`
	MissingGu                []string                       `json:"missing_gu,omitempty"`
	BalanceVersion           int                            `json:"balance_version"`
}

type ownedKillerMoveGu struct {
	InstanceID string  `json:"instance_id"`
	GuID       string  `json:"gu_id"`
	Rank       int     `json:"rank"`
	Hunger     float64 `json:"hunger"`
	State      string  `json:"state"`
	Location   string  `json:"location"`
	IsLocked   bool    `json:"is_locked"`
}

func killerMoveOwner(c *gin.Context) (uuid.UUID, bool) {
	id, ok := middleware.GetPlayerID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return uuid.Nil, false
	}
	return id, true
}

func loadOwnedKillerMoveGus(tx *sql.Tx, playerID uuid.UUID) (map[string]ownedKillerMoveGu, error) {
	rows, err := tx.Query(`SELECT id, gu_id, gu_rank, hunger, state, location, is_locked FROM player_gu WHERE player_id=$1`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	owned := make(map[string]ownedKillerMoveGu)
	for rows.Next() {
		var gu ownedKillerMoveGu
		if err := rows.Scan(&gu.InstanceID, &gu.GuID, &gu.Rank, &gu.Hunger, &gu.State, &gu.Location, &gu.IsLocked); err != nil {
			return nil, err
		}
		owned[gu.InstanceID] = gu
	}
	return owned, rows.Err()
}

func getKillerMoveEffectOptions(inst ownedKillerMoveGu) ([]killerMoveEffectOption, error) {
	spec, ok := models.GetGu(inst.GuID, inst.Rank)
	if !ok {
		return nil, fmt.Errorf("Không tìm thấy dữ liệu Cổ %s rank %d", inst.GuID, inst.Rank)
	}
	var result []killerMoveEffectOption
	for bi, block := range spec.Effects.GetBlocks() {
		if block.Trigger != gameplayeffects.TriggerOnAction && block.Trigger != gameplayeffects.TriggerOnUse {
			continue
		}
		for oi, op := range block.Operations {
			target := block.Target
			if op.Target != nil {
				target = *op.Target
			}
			combatSupported := block.Trigger == gameplayeffects.TriggerOnAction && killerMoveOperationAllowed(op, target)
			outsideCombatSupported := block.Trigger == gameplayeffects.TriggerOnUse && worldUseOperationAllowed(op, target)
			if !combatSupported && !outsideCombatSupported {
				continue
			}
			key := killerMoveEffectKey(inst.InstanceID, bi, oi)
			capabilities := make([]string, 0, 1)
			for _, grant := range spec.CompositionContributions.Grants {
				if grant.EffectBlockID == block.ID && grant.OperationIndex == oi {
					capabilities = append(capabilities, grant.Capability)
				}
			}
			result = append(result, killerMoveEffectOption{
				Key: key, SourceBlockID: block.ID, BlockIndex: bi, OperationIndex: oi, SourceID: inst.InstanceID, SourceName: spec.Name,
				GuID: inst.GuID, Rank: inst.Rank, Label: killerMoveOperationLabel(op), Trigger: string(block.Trigger),
				Operation: op, Target: target, Conditions: append([]gameplayeffects.Condition(nil), block.Conditions...),
				PlayerATKCoefficient: spec.PlayerATKCoefficient, CombatSupported: combatSupported,
				CompositionCapabilities: capabilities,
				TravelSupported:         outsideCombatSupported,
				UnavailableReason:       map[bool]string{true: "", false: "Chỉ dùng ngoài combat"}[combatSupported],
			})
		}
	}
	// Older dictionary effects can be compiled into blocks at load time. If a
	// definition has no authoring blocks, it is not offered for player mixing.
	return result, nil
}

// validateKillerMoveCompositionSnapshot revalidates persisted provenance at
// activation/use time. Output-only legacy graphs remain readable; any graph
// with composition edges must carry the current explicit rule revision.
func validateKillerMoveCompositionSnapshot(graph composition.Graph, refs []killerMoveEffectRef, components []killerMoveComponent, blocks []gameplayeffects.Block) error {
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
		component, ownsComponent := componentsByID[node.GuInstanceID]
		if !ownsComponent || component.GuID != node.GuID || component.Rank != node.Rank {
			return fmt.Errorf("graph node %q is not backed by its declared Gu component", node.EffectKey)
		}
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
		if option == nil || option.SourceBlockID != ref.SourceBlockID || option.OperationIndex != ref.OpIndex || option.Trigger != ref.Trigger {
			return fmt.Errorf("selected effect %q no longer matches its authored Block/op", ref.Key)
		}
		if len(option.CompositionCapabilities) > 0 {
			capabilitiesByEffect[node.EffectKey] = make(map[string]bool, len(option.CompositionCapabilities))
			for _, capability := range option.CompositionCapabilities {
				capabilitiesByEffect[node.EffectKey][capability] = true
			}
		}
		if node.BlockID == "" && legacyOutputOnly {
			node.BlockID = blocks[0].ID
		}
		snapshot, found := blocksByID[node.BlockID]
		if !found || snapshot.Trigger != gameplayeffects.Trigger(option.Trigger) || snapshot.Target != option.Target || !reflect.DeepEqual(snapshot.Conditions, option.Conditions) || len(snapshot.Operations) != 1 || !sameKillerMoveOperationSemantics(snapshot.Operations[0], option.Operation) {
			return fmt.Errorf("Block v2 snapshot for %q does not match its authored operation", node.EffectKey)
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
	_, err := composition.ResolveWithRules(graph, profile, selected, composition.KillerMoveRuleSet(), capabilitiesByEffect)
	return err
}

func sameKillerMoveOperationSemantics(snapshot, authored gameplayeffects.Operation) bool {
	if snapshot.Target != nil {
		return false
	}
	return snapshot.Op == authored.Op && snapshot.SubType == authored.SubType && snapshot.Path == authored.Path &&
		snapshot.StatusID == authored.StatusID && snapshot.Stat == authored.Stat && snapshot.Unit == authored.Unit &&
		snapshot.Resource == authored.Resource && snapshot.StackPolicy == authored.StackPolicy && snapshot.Code == authored.Code &&
		reflect.DeepEqual(snapshot.Params, authored.Params)
}

func killerMoveEffectKey(instanceID string, blockIndex, operationIndex int) string {
	return fmt.Sprintf("%s:%d:%d", instanceID, blockIndex, operationIndex)
}

func killerMoveOperationAllowed(op gameplayeffects.Operation, target gameplayeffects.Target) bool {
	selector := strings.ToUpper(string(target.Selector))
	isSelf := selector == string(gameplayeffects.TargetSelf)
	isEnemy := selector == string(gameplayeffects.TargetSingle) || selector == string(gameplayeffects.TargetRow) ||
		selector == string(gameplayeffects.TargetCol) || selector == "CROSS" || selector == string(gameplayeffects.TargetAll) ||
		selector == string(gameplayeffects.TargetAllEnemies)
	switch op.Op {
	case gameplayeffects.OpDamage:
		return isEnemy && (op.SubType == "" || strings.EqualFold(op.SubType, "FLAT") || strings.EqualFold(op.SubType, "MULTIPLIER") || strings.EqualFold(op.SubType, "MAX_HP_PERCENT"))
	case gameplayeffects.OpDirectHP:
		return isEnemy && (strings.EqualFold(op.SubType, "MAX_HP_PERCENT") || strings.EqualFold(op.SubType, "SHIELD_STRIP"))
	case gameplayeffects.OpShield, gameplayeffects.OpBarrier:
		return isSelf && (op.SubType == "" || strings.EqualFold(op.SubType, "FLAT") || strings.EqualFold(op.SubType, "MULTIPLIER") || strings.EqualFold(op.SubType, "PERCENT"))
	case gameplayeffects.OpHeal:
		if strings.EqualFold(op.SubType, "LIFESTEAL") {
			return false // lifesteal needs a single damage executor to preserve ordering
		}
		return isSelf && (op.SubType == "" || strings.EqualFold(op.SubType, "FLAT") || strings.EqualFold(op.SubType, "PERCENT") || strings.EqualFold(op.SubType, "MANA") || strings.EqualFold(op.SubType, "RESOURCE"))
	case gameplayeffects.OpCleanse:
		return isSelf
	case gameplayeffects.OpStatus:
		status, ok := statusfx.Spec(op.StatusID)
		if !ok {
			return false
		}
		return (status.SelfTarget && isSelf) || (!status.SelfTarget && isEnemy)
	default:
		return false
	}
}

func killerMoveOperationLabel(op gameplayeffects.Operation) string {
	switch op.Op {
	case gameplayeffects.OpDamage:
		return "Gây sát thương"
	case gameplayeffects.OpShield, gameplayeffects.OpBarrier:
		return "Tạo khiên"
	case gameplayeffects.OpHeal:
		return "Hồi phục"
	case gameplayeffects.OpDirectHP:
		return "Tổn thương trực tiếp"
	case gameplayeffects.OpStatus:
		return "Hiệu ứng: " + strings.ToLower(op.StatusID)
	case gameplayeffects.OpCleanse:
		return "Thanh tẩy"
	case gameplayeffects.OpStatMod:
		return "Điều chỉnh " + strings.ToLower(op.Stat)
	default:
		return string(op.Op)
	}
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
	c.JSON(http.StatusOK, gin.H{"gus": instances, "effect_options": options, "composition_rule_set": composition.KillerMoveRuleSet(), "max_saved": profile.MaxSaved, "max_equipped": profile.MaxEquipped, "max_components": profile.MaxGuComponents, "max_effects": profile.MaxNodes})
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
		selected = append(selected, killerMoveEffectRef{Key: key, SourceBlockID: option.SourceBlockID, Trigger: option.Trigger, InstanceID: option.SourceID, GuID: option.GuID, Rank: option.Rank, BlockIndex: bi, OpIndex: oi, PlayerATKCoefficient: &coefficient})
		role := composition.NodeSupport
		if key == req.OutputEffectKey {
			role = composition.NodeOutput
		}
		executionContext := productProfile.ContextByTrigger[option.Trigger]
		opIndex := oi
		graphNodes = append(graphNodes, composition.Node{EffectKey: key, SourceBlockID: option.SourceBlockID, OperationIndex: &opIndex, Role: role, GuInstanceID: option.SourceID, GuID: option.GuID, Rank: option.Rank, Trigger: option.Trigger, Context: executionContext, Executor: productProfile.ExecutorByContext[executionContext]})
		blockTarget := option.Target
		operation := option.Operation
		// Block v2 stores target selection on the block, never on the operation.
		operation.Target = nil
		blocks = append(blocks, gameplayeffects.Block{ID: fmt.Sprintf("EFFECT_%02d", len(blocks)+1), Trigger: gameplayeffects.Trigger(option.Trigger), Target: blockTarget, Conditions: append([]gameplayeffects.Condition(nil), option.Conditions...), Operations: []gameplayeffects.Operation{operation}})
		effectLoad := killerMoveEffectLoad(option.Operation, blockTarget)
		load += effectLoad
		if option.Operation.Op == gameplayeffects.OpCleanse {
			unscalableLoad += effectLoad
		}
	}
	graph, err := composition.ResolveWithRules(composition.Graph{
		Product: productProfile.Product, ProductProfile: productProfile.ID, Context: productProfile.Context, Executor: productProfile.Executor,
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
			scaleKillerMoveOperation(&blocks[bi].Operations[oi], scale)
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

func killerMovePreviewHash(preview killerMovePreview) string {
	preview.PreviewFingerprint = ""
	raw, _ := json.Marshal(preview)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
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

func killerMoveEffectLoad(op gameplayeffects.Operation, target gameplayeffects.Target) float64 {
	base := 16.0
	switch op.Op {
	case gameplayeffects.OpDamage, gameplayeffects.OpDirectHP:
		base = 24
	case gameplayeffects.OpStatus:
		base = 20
	case gameplayeffects.OpShield, gameplayeffects.OpBarrier:
		base = 18
	case gameplayeffects.OpHeal:
		base = 17
	case gameplayeffects.OpCleanse:
		base = 12
	case gameplayeffects.OpStatMod:
		base = 14
	}
	valueFactor := 1 + math.Min(2.5, math.Log10(1+math.Abs(op.Value))/5)
	scope := strings.ToUpper(string(target.Selector))
	switch scope {
	case "ALL", "ALL_ENEMIES", "ALL_ALLIES":
		valueFactor *= 1.8
	case "ROW", "COL":
		valueFactor *= 1.35
	}
	if op.Op == gameplayeffects.OpStatus {
		valueFactor *= 1 + math.Min(0.6, float64(max(0, op.Turns-1))*0.12)
		chance := op.Chance
		if chance <= 0 {
			chance = 1
		}
		valueFactor *= 0.75 + math.Min(1, chance)*0.5
	}
	return base * valueFactor
}

func scaleKillerMoveOperation(op *gameplayeffects.Operation, scale float64) {
	if op == nil || scale >= 1 {
		return
	}
	if op.Op == gameplayeffects.OpStatus {
		chance := op.Chance
		if chance <= 0 {
			chance = 1
		}
		op.Chance = math.Max(0.05, chance*scale)
		op.Value *= scale
		if op.Turns > 1 {
			op.Turns = max(1, int(math.Ceil(float64(op.Turns)*scale)))
		}
		return
	}
	if op.Op == gameplayeffects.OpCleanse {
		return
	}
	if (op.Op == gameplayeffects.OpDamage || op.Op == gameplayeffects.OpShield) && strings.EqualFold(op.SubType, "MULTIPLIER") {
		op.Value = 1 + (op.Value-1)*scale
		return
	}
	op.Value *= scale
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
		m.SuccessChance = killerMoveAdjustedChance(m.SuccessChance, m.CreationPathMasteryLevel, currentLevel, m.MasteryXP)
		available, hasStarvingGu, missingGu := evaluateKillerMoveComponents(m.Components, ownedSet)
		compositionErr := snapshotErr
		if compositionErr == nil {
			compositionErr = validateKillerMoveCompositionSnapshot(m.CompositionGraph, m.SelectedEffects, m.Components, m.CompiledEffects)
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
	if err := validateKillerMoveCompositionSnapshot(move.CompositionGraph, move.SelectedEffects, move.Components, move.CompiledEffects); err != nil {
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
	cost := gamefx.EnergyCost(&player, &mainSpec)
	if player.CurrentMana < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}
	useSnapshot := gameplayeffects.Snapshot{
		ID: player.ID.String(), HP: player.HP, MaxHP: gamefx.EffectiveMaxHP(&player),
		Mana: player.CurrentMana, MaxMana: gamefx.EffectiveMaxMana(&player),
	}
	player.CurrentMana -= cost
	cleanPath := gamefx.CleanPath(move.Path)
	if cleanPath == "" {
		cleanPath = "strength"
	}
	currentLevel := gamefx.PathMasteryLevel(player.PathMasteries[cleanPath])
	chance := killerMoveAdjustedChance(move.SuccessChance, move.CreationPathMasteryLevel, currentLevel, move.MasteryXP)
	roll := rand.Float64()
	success := roll < chance
	message := ""
	pathGain := 0
	pathPoints := player.PathMasteries[cleanPath]
	if success {
		result, evalErr := compiledEffects.Evaluate(gameplayeffects.Context{
			Actor: useSnapshot,
			Rank:  move.MainGuRank, Mastery: currentLevel,
		}, gameplayeffects.Event{Trigger: gameplayeffects.TriggerOnUse})
		if evalErr != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Không thể đánh giá effect ON_USE"})
			return
		}
		healed, restoredMana, flightDurationMinutes, stealthConcealment, hasteDurationMinutes, applyErr := applyWorldUseCommands(&player, result.Commands, move.MainGuRank, move.Name)
		if applyErr != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Executor ON_USE từ chối effect không hỗ trợ"})
			return
		}
		if healed <= 0 && restoredMana <= 0 && flightDurationMinutes == 0 && stealthConcealment == 0 && hasteDurationMinutes == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Điều kiện effect ON_USE không thỏa hoặc không tạo tác động được hỗ trợ"})
			return
		}
		effectsApplied := make([]string, 0, 5)
		if healed > 0 {
			effectsApplied = append(effectsApplied, fmt.Sprintf("hồi %.0f HP", healed))
		}
		if restoredMana > 0 {
			effectsApplied = append(effectsApplied, fmt.Sprintf("hoàn %.0f Chân Nguyên", restoredMana))
		}
		if flightDurationMinutes > 0 {
			effectsApplied = append(effectsApplied, fmt.Sprintf("Phi Hành %d phút", flightDurationMinutes))
		}
		if stealthConcealment > 0 {
			effectsApplied = append(effectsApplied, fmt.Sprintf("Ẩn Nấp +%d điểm trong thời gian trạng thái", stealthConcealment))
		}
		if hasteDurationMinutes > 0 {
			effectsApplied = append(effectsApplied, fmt.Sprintf("Tăng Tốc trong %d phút", hasteDurationMinutes))
		}
		message = fmt.Sprintf("Sát chiêu [%s] thành công: %s.", move.Name, strings.Join(effectsApplied, "; "))
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
		message = fmt.Sprintf("Thôi động [%s] thất bại (%.0f%% / tỉ lệ %.0f%%). Các Cổ thành phần vẫn hao độ no.", move.Name, roll*100, chance*100)
	}
	for _, component := range move.Components {
		gu := owned[component.InstanceID]
		gu.Hunger = max(0, gu.Hunger-1)
		if _, err := tx.Exec(`UPDATE player_gu SET hunger=$1,updated_at=NOW() WHERE id=$2 AND player_id=$3`, gu.Hunger, gu.InstanceID, playerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể giảm độ no Cổ thành phần"})
			return
		}
	}
	if _, err := tx.Exec(`UPDATE players SET hp=$1,current_mana=$2,path_masteries=$3,statuses=$4 WHERE id=$5`, player.HP, player.CurrentMana, player.PathMasteries, player.Statuses, playerID); err != nil {
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

func killerMoveAdjustedChance(base float64, creationLevel, currentLevel, xp int) float64 {
	chance := base + float64(currentLevel-creationLevel)*0.035 + math.Min(.15, float64(xp)/1000*.15)
	if chance > .95 {
		return .95
	}
	if chance < .05 {
		return .05
	}
	return chance
}
