package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/composition"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/httperr"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/auditlog"
)

var guHouseProductProfile = composition.GuHouseProfile()

type guHouseDraftRequest struct {
	Name                      string                       `json:"name"`
	CoreGuInstanceIDs         []string                     `json:"core_gu_instance_ids"`
	AuxGuCount                int                          `json:"aux_gu_count"`
	ActionEffectKeys          map[string]string            `json:"action_effect_keys,omitempty"`
	ActionCompositionDraftIDs map[string]string            `json:"action_composition_draft_ids,omitempty"`
	ActionCompositionGraphs   map[string]composition.Graph `json:"action_composition_graphs,omitempty"`
	PreviewFingerprint        string                       `json:"preview_fingerprint,omitempty"`
}

func bindGuHouseCompositionDrafts(c *gin.Context, tx *sql.Tx, playerID uuid.UUID, req *guHouseDraftRequest) error {
	if len(req.ActionCompositionDraftIDs) == 0 {
		for key, graph := range req.ActionCompositionGraphs {
			graph.CatalogDraftID = ""
			req.ActionCompositionGraphs[key] = graph
		}
		return nil
	}
	if req.ActionCompositionGraphs == nil {
		req.ActionCompositionGraphs = make(map[string]composition.Graph)
	}
	for key, graph := range req.ActionCompositionGraphs {
		graph.CatalogDraftID = ""
		req.ActionCompositionGraphs[key] = graph
	}
	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		return err
	}
	selectedCoreSet := make(map[string]bool, len(req.CoreGuInstanceIDs))
	for _, instanceID := range req.CoreGuInstanceIDs {
		selectedCoreSet[instanceID] = true
	}
	resolvedByDraft := make(map[string]resolvedAdminComposition)
	for coreID, draftIDRaw := range req.ActionCompositionDraftIDs {
		if !selectedCoreSet[coreID] {
			return fmt.Errorf("Composition draft action phải gắn với Cổ hạch tâm đã chọn")
		}
		id, err := uuid.Parse(draftIDRaw)
		if err != nil {
			return fmt.Errorf("Composition draft ID của Tiên Cổ Ốc không hợp lệ")
		}
		var raw []byte
		if err := tx.QueryRowContext(c.Request.Context(), `SELECT definition FROM admin_effect_composition_drafts WHERE id=$1 AND status='DRAFT' AND product_profile=$2`, id, composition.GuHouseActionProfile().ID).Scan(&raw); err != nil {
			return fmt.Errorf("Không tìm thấy composition draft Trận Kích đang hoạt động")
		}
		var draft adminCompositionDraft
		if err := json.Unmarshal(raw, &draft); err != nil {
			return fmt.Errorf("Composition draft bị hỏng: %w", err)
		}
		resolved, cached := resolvedByDraft[id.String()]
		if !cached {
			resolved, err = resolveAdminComposition(c, draft)
			if err != nil {
				return fmt.Errorf("Composition draft không còn hợp lệ; Admin cần validate và lưu lại: %w", err)
			}
			resolvedByDraft[id.String()] = resolved
		}
		core, ok := owned[coreID]
		outputIndex := -1
		for i := range resolved.Draft.Sources {
			if resolved.Draft.Sources[i].Key == resolved.Draft.OutputEffectKey && resolved.Draft.Sources[i].Role == string(composition.NodeOutput) {
				outputIndex = i
				break
			}
		}
		if !ok || outputIndex < 0 || core.GuID != resolved.Draft.Sources[outputIndex].GuID || core.Rank != resolved.Draft.Sources[outputIndex].Rank {
			return fmt.Errorf("Cổ hạch tâm %s không khớp output của draft Trận Kích", coreID)
		}
		instanceBySource := map[string]string{resolved.Draft.OutputEffectKey: coreID}
		instanceByGuRank := map[string]string{}
		selectedSet := map[string]bool{coreID: true}
		for _, source := range resolved.Draft.Sources {
			if source.Role != string(composition.NodeSupport) {
				continue
			}
			group := fmt.Sprintf("%s:%d", source.GuID, source.Rank)
			if instanceID := instanceByGuRank[group]; instanceID != "" {
				instanceBySource[source.Key] = instanceID
				continue
			}
			for _, instanceID := range req.CoreGuInstanceIDs {
				gu := owned[instanceID]
				if !selectedSet[instanceID] && gu.GuID == source.GuID && gu.Rank == source.Rank && strings.EqualFold(gu.State, "ACTIVE") && gu.Location == "INVENTORY" && !gu.IsLocked {
					instanceBySource[source.Key] = instanceID
					instanceByGuRank[group] = instanceID
					selectedSet[instanceID] = true
					break
				}
			}
			if instanceBySource[source.Key] == "" {
				return fmt.Errorf("Tiên Cổ Ốc cần Cổ hỗ trợ %s rank %d trong các hạch tâm đã chọn", source.GuID, source.Rank)
			}
		}
		bindings := make(map[string]composition.SourceBinding, len(resolved.Draft.Sources))
		for _, source := range resolved.Draft.Sources {
			options, err := getKillerMoveEffectOptions(owned[instanceBySource[source.Key]])
			if err != nil {
				return err
			}
			for _, option := range options {
				if option.SourceBlockID == source.BlockID && option.OperationIndex == source.OperationIndex && option.SourceRevision == source.SourceRevision && option.CombatSupported && guHouseActionOperationAllowed(option.Operation, option.Target) {
					profile := composition.GuHouseActionProfile()
					bindings[source.Key] = composition.SourceBinding{
						EffectKey: option.Key, GuInstanceID: option.SourceID, GuID: option.GuID, Rank: option.Rank,
						SourceBlockID: option.SourceBlockID, SourceRevision: option.SourceRevision,
						OperationIndex: option.OperationIndex, Trigger: option.Trigger,
						Context: profile.ContextByTrigger[option.Trigger], Executor: profile.ExecutorByContext[profile.ContextByTrigger[option.Trigger]],
					}
					break
				}
			}
			if _, found := bindings[source.Key]; !found {
				return fmt.Errorf("Effect source %s không còn được hỗ trợ bởi Tiên Cổ Ốc", source.Key)
			}
		}
		graph, err := composition.BindCatalogGraph(resolved.Graph, id.String(), bindings)
		if err != nil {
			return err
		}
		req.ActionCompositionGraphs[coreID] = graph
	}
	return nil
}

type guHouseRecord struct {
	ID                string                      `json:"id"`
	PlayerID          string                      `json:"player_id"`
	Name              string                      `json:"name"`
	Rank              int                         `json:"rank"`
	CoreGuInstances   []string                    `json:"core_gu_instances"`
	AuxGuCount        int                         `json:"aux_gu_count"`
	HouseProfile      models.GuHouseCombatProfile `json:"house_profile"`
	CurrentDurability float64                     `json:"current_durability"`
	IsEquipped        bool                        `json:"is_equipped"`
	CreatedAt         string                      `json:"created_at"`
}

func calculateGuHousePreview(tx *sql.Tx, playerID uuid.UUID, req guHouseDraftRequest) (models.GuHouseCombatProfile, int, error) {
	if guHouseProductProfile.ID == "" || guHouseProductProfile.GuPolicy == "" {
		return models.GuHouseCombatProfile{}, 0, fmt.Errorf("Tiên Cổ Ốc chưa có product profile và chính sách Cổ thành phần")
	}
	req.Name = strings.TrimSpace(req.Name)
	if len(req.CoreGuInstanceIDs) < guHouseProductProfile.MinGuComponents || len(req.CoreGuInstanceIDs) > guHouseProductProfile.MaxGuComponents {
		return models.GuHouseCombatProfile{}, 0, fmt.Errorf("Tiên Cổ Ốc cần từ %d đến %d Tiên Cổ hạch tâm", guHouseProductProfile.MinGuComponents, guHouseProductProfile.MaxGuComponents)
	}
	if req.AuxGuCount < 0 {
		req.AuxGuCount = 0
	}
	if req.AuxGuCount > guHouseProductProfile.MaxAuxGuCount {
		req.AuxGuCount = guHouseProductProfile.MaxAuxGuCount
	}

	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		return models.GuHouseCombatProfile{}, 0, err
	}

	highestRank := 6
	coreGuIDs := make([]string, 0, len(req.CoreGuInstanceIDs))
	seen := map[string]bool{}

	actions := make([]models.KillerMoveCombatProfile, 0, len(req.CoreGuInstanceIDs))

	for _, id := range req.CoreGuInstanceIDs {
		if seen[id] {
			return models.GuHouseCombatProfile{}, 0, fmt.Errorf("Một Cổ hạch tâm không thể được chọn nhiều lần")
		}
		seen[id] = true
		gu, ok := owned[id]
		if !ok || !strings.EqualFold(gu.State, "ACTIVE") || gu.Location != "INVENTORY" || gu.IsLocked {
			return models.GuHouseCombatProfile{}, 0, fmt.Errorf("Cổ hạch tâm không hợp lệ hoặc đã kiệt sức")
		}
		if gu.Rank < 6 {
			return models.GuHouseCombatProfile{}, 0, fmt.Errorf("Hạch tâm Tiên Cổ Ốc phải là Tiên Cổ (Rank 6 trở lên)")
		}
		if gu.Rank > highestRank {
			highestRank = gu.Rank
		}
		coreGuIDs = append(coreGuIDs, gu.GuID)

		spec, found := models.GetGu(gu.GuID, gu.Rank)
		if found {
			actionPath := "strength"
			if len(spec.Path) > 0 {
				actionPath = gamefx.CleanPath(spec.Path[0])
			}
			if graph, hasGraph := req.ActionCompositionGraphs[id]; hasGraph {
				action, compileErr := compileGuHouseComposition(gu, spec, actionPath, graph, req.CoreGuInstanceIDs, owned)
				if compileErr != nil {
					return models.GuHouseCombatProfile{}, 0, compileErr
				}
				actions = append(actions, action)
				continue
			}
			if req.ActionEffectKeys != nil {
				key := req.ActionEffectKeys[id]
				if key == "" {
					continue
				}
				options, optionErr := getKillerMoveEffectOptions(gu)
				if optionErr != nil {
					return models.GuHouseCombatProfile{}, 0, optionErr
				}
				var selected *killerMoveEffectOption
				for optionIndex := range options {
					if options[optionIndex].Key == key && options[optionIndex].CombatSupported {
						selected = &options[optionIndex]
						break
					}
				}
				if selected == nil {
					return models.GuHouseCombatProfile{}, 0, fmt.Errorf("Trận Kích đã chọn không hợp lệ hoặc chưa có executor combat theo lượt")
				}
				houseActionProfile := composition.GuHouseActionProfile()
				sourceBlockID, operationIndex := selected.SourceBlockID, selected.OperationIndex
				legacyGraph := composition.Graph{
					SchemaVersion: composition.GraphSchemaVersion,
					Product:       houseActionProfile.Product, ProductProfile: houseActionProfile.ID,
					Context: houseActionProfile.Context, Executor: houseActionProfile.Executor, RuleSetRevision: composition.GuHouseActionRuleSet().Revision,
					CoreGuInstanceID: gu.InstanceID, OutputEffectKey: selected.Key,
					Nodes: []composition.Node{{
						EffectKey: selected.Key, Role: composition.NodeOutput, GuInstanceID: gu.InstanceID,
						GuID: gu.GuID, Rank: gu.Rank, Trigger: selected.Trigger,
						Context: houseActionProfile.Context, Executor: houseActionProfile.Executor, SourceBlockID: sourceBlockID, SourceRevision: selected.SourceRevision, OperationIndex: &operationIndex,
					}},
				}
				action, compileErr := compileGuHouseComposition(gu, spec, actionPath, legacyGraph, req.CoreGuInstanceIDs, owned)
				if compileErr != nil {
					return models.GuHouseCombatProfile{}, 0, compileErr
				}
				actions = append(actions, action)
				continue
			}
			// Products cannot invent combat effects from Gu rank alone. Without
			// an authored effect selection this core contributes no Trận Kích.
		}
	}

	baseHP := float64(highestRank*500 + len(coreGuIDs)*250 + req.AuxGuCount*35)
	baseDef := float64(highestRank*40 + len(coreGuIDs)*25 + req.AuxGuCount*4)
	reduction := math.Min(0.35, 0.10+float64(len(coreGuIDs))*0.05)

	profile := models.GuHouseCombatProfile{
		ProfileVersion:  2,
		ProductProfile:  guHouseProductProfile.ID,
		Context:         guHouseProductProfile.Context,
		Executor:        guHouseProductProfile.Executor,
		Name:            req.Name,
		Rank:            highestRank,
		CoreGuIDs:       coreGuIDs,
		AuxGuCount:      req.AuxGuCount,
		MaxHouseHP:      baseHP,
		CurrentHouseHP:  baseHP,
		HouseDefense:    baseDef,
		DamageReduction: reduction,
		ImmuneDebuffs:   true,
		ZeroBackfire:    true,
		Actions:         actions,
	}
	profile.PreviewFingerprint, err = guHousePreviewHash(profile)
	if err != nil {
		return models.GuHouseCombatProfile{}, 0, fmt.Errorf("không thể chuẩn hóa Block v2 của Trận Kích: %w", err)
	}

	return profile, highestRank, nil
}

func compileGuHouseComposition(core ownedKillerMoveGu, spec models.Gu, actionPath string, graph composition.Graph, houseGuIDs []string, owned map[string]ownedKillerMoveGu) (models.KillerMoveCombatProfile, error) {
	profile := composition.GuHouseActionProfile()
	if graph.CoreGuInstanceID != core.InstanceID {
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Trận Kích phải dùng đúng Cổ hạch tâm đã chọn")
	}
	selected := make([]string, 0, len(graph.Nodes))
	seen := map[string]bool{}
	for _, node := range graph.Nodes {
		if !containsString(houseGuIDs, node.GuInstanceID) {
			return models.KillerMoveCombatProfile{}, fmt.Errorf("Cổ hỗ trợ của Trận Kích chưa được chọn làm thành phần Cổ Ốc")
		}
		if !seen[node.GuInstanceID] {
			selected = append(selected, node.GuInstanceID)
			seen[node.GuInstanceID] = true
		}
	}
	capabilitiesByEffect := make(map[string]map[string]bool, len(graph.Nodes))
	for _, node := range graph.Nodes {
		gu := owned[node.GuInstanceID]
		options, optionErr := getKillerMoveEffectOptions(gu)
		if optionErr != nil {
			return models.KillerMoveCombatProfile{}, optionErr
		}
		for _, option := range options {
			if option.Key != node.EffectKey || option.SourceBlockID != node.SourceBlockID || option.OperationIndex != valueOrZero(node.OperationIndex) {
				continue
			}
			capabilitiesByEffect[node.EffectKey] = make(map[string]bool, len(option.CompositionCapabilities))
			for _, capability := range option.CompositionCapabilities {
				capabilitiesByEffect[node.EffectKey][capability] = true
			}
			break
		}
	}
	ruleSet := composition.GuHouseActionRuleSet()
	resolved, err := composition.ResolveWithRules(graph, profile, selected, ruleSet, capabilitiesByEffect)
	if err != nil {
		return models.KillerMoveCombatProfile{}, err
	}
	var output models.KillerMoveCombatProfile
	blocks := make([]gameplayeffects.Block, 0, len(resolved.Nodes))
	components := make([]string, 0, len(selected))
	for _, instanceID := range selected {
		components = append(components, instanceID)
	}
	for nodeIndex := range resolved.Nodes {
		node := &resolved.Nodes[nodeIndex]
		gu, ok := owned[node.GuInstanceID]
		if !ok || gu.GuID != node.GuID || gu.Rank != node.Rank {
			return models.KillerMoveCombatProfile{}, fmt.Errorf("provenance Cổ của effect %q đã thay đổi", node.EffectKey)
		}
		options, err := getKillerMoveEffectOptions(gu)
		if err != nil {
			return models.KillerMoveCombatProfile{}, err
		}
		var selectedEffect *killerMoveEffectOption
		for i := range options {
			if options[i].Key == node.EffectKey && options[i].CombatSupported {
				selectedEffect = &options[i]
				break
			}
		}
		if selectedEffect == nil || selectedEffect.Trigger != node.Trigger || selectedEffect.SourceID != node.GuInstanceID || selectedEffect.GuID != node.GuID {
			return models.KillerMoveCombatProfile{}, fmt.Errorf("effect %q không khớp provenance hoặc chưa được executor hỗ trợ", node.EffectKey)
		}
		node.SourceRevision = selectedEffect.SourceRevision
		action, err := compileGuHouseAction(gu, spec, *selectedEffect, actionPath)
		if err != nil {
			return models.KillerMoveCombatProfile{}, err
		}
		node.BlockID = fmt.Sprintf("EFFECT_%02d", nodeIndex+1)
		operation := selectedEffect.Operation
		operation.Target = nil
		block := gameplayeffects.Block{ID: node.BlockID, Trigger: gameplayeffects.TriggerOnAction, Target: selectedEffect.Target, Conditions: selectedEffect.Conditions, Operations: []gameplayeffects.Operation{operation}}
		blocks = append(blocks, block)
		if node.Role == composition.NodeOutput {
			output = action
			continue
		}
		output.BaseDamage += action.BaseDamage
		output.BaseHealing += action.BaseHealing
		output.BaseShield += action.BaseShield
		output.ManaCost += action.ManaCost
	}
	if output.EffectKey == "" {
		return models.KillerMoveCombatProfile{}, fmt.Errorf("composition thiếu effect đầu ra")
	}
	if output.TargetType == "SELF" && output.BaseDamage > 0 {
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Trận Kích SELF không thể ghép effect gây sát thương lên mục tiêu khác")
	}
	output.CompositionGraph = &resolved
	output.EffectBlocks = gameplayeffects.BlockSet(blocks)
	output.ComponentGuInstances = components
	return output, nil
}

func valueOrZero(value *int) int {
	if value == nil {
		return -1
	}
	return *value
}

func containsString(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func compileGuHouseAction(gu ownedKillerMoveGu, spec models.Gu, selected killerMoveEffectOption, actionPath string) (models.KillerMoveCombatProfile, error) {
	if selected.Trigger != string(gameplayeffects.TriggerOnAction) || !selected.CombatSupported || selected.SourceID != gu.InstanceID || selected.GuID != gu.GuID {
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Trận Kích không có provenance/context/executor hợp lệ")
	}
	operation := selected.Operation
	operation.Target = nil
	block := gameplayeffects.Block{
		ID: selected.Key, Trigger: gameplayeffects.TriggerOnAction, Target: selected.Target,
		Conditions: selected.Conditions, Operations: []gameplayeffects.Operation{operation},
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{block})
	if err != nil {
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Trận Kích %q không compile được: %w", selected.Label, err)
	}
	command := compiled.CompileCombatAction()
	if len(command.Operations) != 1 {
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Trận Kích %q phải tạo đúng một effect command", selected.Label)
	}
	if len(command.Operations[0].Conditions) > 0 {
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Executor Trận Kích chưa hỗ trợ effect có điều kiện")
	}
	if command.Operations[0].Value <= 0 {
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Trận Kích cần giá trị effect lớn hơn 0")
	}
	op := selected.Operation
	target := selected.Target
	if op.Target != nil {
		target = *op.Target
	}
	targetType := ""
	switch target.Selector {
	case gameplayeffects.TargetSingle:
		targetType = "SINGLE"
	case gameplayeffects.TargetSelf:
		targetType = "SELF"
	default:
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Executor Trận Kích chưa hỗ trợ mục tiêu %q", target.Selector)
	}
	if op.SubType != "" && !strings.EqualFold(op.SubType, "FLAT") {
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Executor Trận Kích chưa hỗ trợ subtype %q", op.SubType)
	}
	action := models.KillerMoveCombatProfile{
		ProfileVersion: 2, EffectLabel: selected.Label, EffectKey: selected.Key,
		EffectContext: guHouseProductProfile.Context, EffectExecutor: guHouseProductProfile.Executor, EffectGuID: gu.GuID,
		EffectOperation: &op, EffectTarget: &target, EffectConditions: append([]gameplayeffects.Condition(nil), selected.Conditions...),
		ID: gu.InstanceID, Name: fmt.Sprintf("Trận Kích: %s", spec.Name), Rank: gu.Rank,
		MainPath: actionPath, AllPaths: []string{actionPath}, TargetType: targetType,
		Priority: 1, ManaCost: spec.BaseManaCost * 1.5, SuccessChance: 0.95,
	}
	switch command.Operations[0].Type {
	case gameplayeffects.OpDamage:
		if targetType != "SINGLE" {
			return models.KillerMoveCombatProfile{}, fmt.Errorf("Trận Kích damage chỉ hỗ trợ mục tiêu đơn hiện tại")
		}
		action.BaseDamage = command.Operations[0].Value
		action.DamageMult = 1
	case gameplayeffects.OpHeal:
		if targetType != "SELF" {
			return models.KillerMoveCombatProfile{}, fmt.Errorf("Trận Kích heal chỉ hỗ trợ mục tiêu bản thân hiện tại")
		}
		action.BaseHealing = command.Operations[0].Value
	case gameplayeffects.OpShield, gameplayeffects.OpBarrier:
		if targetType != "SELF" {
			return models.KillerMoveCombatProfile{}, fmt.Errorf("Trận Kích shield chỉ hỗ trợ mục tiêu bản thân hiện tại")
		}
		action.BaseShield = command.Operations[0].Value
	default:
		return models.KillerMoveCombatProfile{}, fmt.Errorf("Executor Trận Kích chưa hỗ trợ operation %q", command.Operations[0].Type)
	}
	return action, nil
}

func guHousePreviewHash(profile models.GuHouseCombatProfile) (string, error) {
	profile.PreviewFingerprint = ""
	raw, err := json.Marshal(profile)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func ListGuHouses(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}

	rows, err := db.DB.QueryContext(c.Request.Context(), `
		SELECT id::text, player_id::text, name, rank, core_gu_instances, aux_gu_count, house_profile, current_durability, is_equipped, to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM player_gu_houses
		WHERE player_id = $1
		ORDER BY created_at DESC
	`, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải danh sách Tiên Cổ Ốc", err))
		return
	}
	defer rows.Close()

	list := make([]guHouseRecord, 0)
	for rows.Next() {
		var r guHouseRecord
		var coreBytes, profBytes []byte
		if err := rows.Scan(&r.ID, &r.PlayerID, &r.Name, &r.Rank, &coreBytes, &r.AuxGuCount, &profBytes, &r.CurrentDurability, &r.IsEquipped, &r.CreatedAt); err != nil {
			httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu Tiên Cổ Ốc", err))
			return
		}
		_ = json.Unmarshal(coreBytes, &r.CoreGuInstances)
		if err := json.Unmarshal(profBytes, &r.HouseProfile); err != nil {
			httperr.Respond(c, httperr.Internal("Profile Block v2 của Tiên Cổ Ốc không hợp lệ", err))
			return
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi đọc dữ liệu Tiên Cổ Ốc", err))
		return
	}
	if err := rows.Close(); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể đóng danh sách Tiên Cổ Ốc", err))
		return
	}

	templates, err := listAdminCompositionTemplates(c.Request.Context(), composition.GuHouseActionProfile().ID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể tải composition templates Tiên Cổ Ốc", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"gu_houses":             list,
		"composition_templates": templates,
		"product_profile":       guHouseProductProfile.ID,
		"composition_rule_set":  composition.GuHouseActionRuleSet(),
		"max_saved":             guHouseProductProfile.MaxSaved,
		"max_core_gus":          guHouseProductProfile.MaxGuComponents,
		"min_core_gus":          guHouseProductProfile.MinGuComponents,
	})
}

func PreviewGuHouse(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	var req guHouseDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()
	if err := bindGuHouseCompositionDrafts(c, tx, playerID, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile, _, err := calculateGuHousePreview(tx, playerID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func CreateGuHouse(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	var req guHouseDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || len([]rune(name)) > 80 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tên Tiên Cổ Ốc phải từ 1 đến 80 ký tự"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()
	if err := bindGuHouseCompositionDrafts(c, tx, playerID, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM player_gu_houses WHERE player_id = $1`, playerID).Scan(&count); err != nil {
		httperr.Respond(c, httperr.Internal("Không thể kiểm tra giới hạn", err))
		return
	}
	if count >= guHouseProductProfile.MaxSaved {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Mỗi người chơi tối đa %d Tiên Cổ Ốc", guHouseProductProfile.MaxSaved)})
		return
	}

	profile, highestRank, err := calculateGuHousePreview(tx, playerID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.PreviewFingerprint == "" || req.PreviewFingerprint != profile.PreviewFingerprint {
		c.JSON(http.StatusConflict, gin.H{"error": "Bản preview Cổ Ốc đã thay đổi; hãy xem lại profile trước khi lưu"})
		return
	}
	if req.ActionEffectKeys != nil && len(profile.Actions) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chọn ít nhất một Trận Kích được hỗ trợ trong combat theo lượt trước khi đúc kết Tiên Cổ Ốc"})
		return
	}

	coreBytes, _ := json.Marshal(req.CoreGuInstanceIDs)
	profile.PreviewFingerprint = ""
	profBytes, err := json.Marshal(profile)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể chuẩn hóa Block v2 của Trận Kích", err))
		return
	}

	var houseID string
	err = tx.QueryRow(`
		INSERT INTO player_gu_houses (player_id, name, rank, core_gu_instances, aux_gu_count, house_profile, current_durability, is_equipped)
		VALUES ($1, $2, $3, $4::jsonb, $5, $6::jsonb, 100, false)
		RETURNING id::text
	`, playerID, name, highestRank, coreBytes, req.AuxGuCount, profBytes).Scan(&houseID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể lưu Tiên Cổ Ốc", err))
		return
	}

	auditlog.Log(tx, auditlog.Entry{
		PlayerID:   &playerID,
		Action:     "gu_house.create",
		EntityType: "gu_house",
		EntityID:   houseID,
		Metadata: map[string]interface{}{
			"name":         name,
			"rank":         highestRank,
			"core_count":   len(req.CoreGuInstanceIDs),
			"aux_gu_count": req.AuxGuCount,
		},
		IPAddress: c.ClientIP(),
		Success:   true,
	})

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi xác nhận giao dịch", err))
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": houseID, "message": "Đã chế tạo thành công Tiên Cổ Ốc!"})
}

func EquipGuHouse(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	houseID := c.Param("id")
	if houseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã Tiên Cổ Ốc"})
		return
	}

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()
	var lockedPlayer string
	if err = tx.QueryRow(`SELECT id::text FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&lockedPlayer); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhân vật"})
		return
	}

	var isEquipped bool
	var coreBytes []byte
	var profileBytes []byte
	var durability float64
	err = tx.QueryRow(`SELECT is_equipped,core_gu_instances,house_profile,current_durability FROM player_gu_houses WHERE id=$1 AND player_id=$2 FOR UPDATE`, houseID, playerID).Scan(&isEquipped, &coreBytes, &profileBytes, &durability)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Tiên Cổ Ốc"})
		return
	}

	newState := !isEquipped
	if newState {
		if err := validateGuHouseActivationProfile(profileBytes); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if durability <= 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Tiên Cổ Ốc đã hư hỏng hoàn toàn; hãy tu sửa trước khi trang bị"})
			return
		}
		var coreIDs []string
		if err = json.Unmarshal(coreBytes, &coreIDs); err != nil || len(coreIDs) < guHouseProductProfile.MinGuComponents || len(coreIDs) > guHouseProductProfile.MaxGuComponents {
			c.JSON(http.StatusConflict, gin.H{"error": "Danh sách Cổ hạch tâm không hợp lệ; hãy đúc lại Tiên Cổ Ốc"})
			return
		}
		seenCoreIDs := make(map[string]struct{}, len(coreIDs))
		for _, coreID := range coreIDs {
			if _, duplicate := seenCoreIDs[coreID]; duplicate {
				c.JSON(http.StatusConflict, gin.H{"error": "Danh sách Cổ hạch tâm có instance trùng lặp"})
				return
			}
			seenCoreIDs[coreID] = struct{}{}
		}
		if guHouseProductProfile.GuPolicy == composition.GuPolicyReserveActive {
			// Release the outgoing house first. Any later error rolls the entire
			// transaction back, and this permits two saved houses to share a core.
			if _, err = tx.Exec(`UPDATE player_gu_houses SET is_equipped=false,updated_at=NOW() WHERE player_id=$1 AND is_equipped=true AND id<>$2`, playerID, houseID); err != nil {
				httperr.Respond(c, httperr.Internal("Không thể tháo Tiên Cổ Ốc đang hoạt động", err))
				return
			}
			if _, err = tx.Exec(`DELETE FROM player_gu_house_core_reservations WHERE player_id=$1 AND gu_house_id IN (SELECT id FROM player_gu_houses WHERE player_id=$1 AND is_equipped=false) AND gu_house_id<>$2`, playerID, houseID); err != nil {
				httperr.Respond(c, httperr.Internal("Không thể giải phóng Cổ hạch tâm cũ", err))
				return
			}
		}
		var houseProf models.GuHouseCombatProfile
		_ = json.Unmarshal(profileBytes, &houseProf)
		relinkedHouse := false
		for idx, coreID := range coreIDs {
			var guID, state, location string
			var rank int
			var isLocked bool
			err = tx.QueryRow(`SELECT gu_id,gu_rank,state,location,is_locked FROM player_gu WHERE id=$1 AND player_id=$2 FOR UPDATE`, coreID, playerID).Scan(&guID, &rank, &state, &location, &isLocked)
			if err != nil || rank < 6 || !strings.EqualFold(state, "ACTIVE") || location != "INVENTORY" || isLocked {
				// Try auto-relink if instance changed or upgraded
				var targetGuID string
				if idx < len(houseProf.CoreGuIDs) {
					targetGuID = houseProf.CoreGuIDs[idx]
				}
				if targetGuID != "" {
					var newID string
					var newRank int
					findErr := tx.QueryRow(`
						SELECT id::text, gu_rank FROM player_gu 
						WHERE player_id=$1 AND gu_id=$2 AND gu_rank >= 6 AND state='ACTIVE' AND location='INVENTORY' AND is_locked=false 
						ORDER BY gu_rank DESC LIMIT 1
					`, playerID, targetGuID).Scan(&newID, &newRank)

					if findErr == nil && newID != "" {
						coreIDs[idx] = newID
						coreID = newID
						relinkedHouse = true
						err = nil
						rank = newRank
						state = "ACTIVE"
						location = "INVENTORY"
						isLocked = false
					}
				}
			}
			if err != nil || rank < 6 || !strings.EqualFold(state, "ACTIVE") || location != "INVENTORY" || isLocked {
				c.JSON(http.StatusConflict, gin.H{"error": "Tiên Cổ Ốc thiếu đúng Cổ hạch tâm còn sống trong hành trang"})
				return
			}
			var heldByArray string
			reservationErr := tx.QueryRow(`SELECT array_id::text FROM teleport_array_gu_reservations WHERE gu_instance_id=$1`, coreID).Scan(&heldByArray)
			if reservationErr == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Cổ hạch tâm đang được Trận Pháp trên bản đồ giữ chỗ"})
				return
			}
			if reservationErr != sql.ErrNoRows {
				httperr.Respond(c, httperr.Internal("Không thể kiểm tra giữ chỗ Trận Pháp", reservationErr))
				return
			}
			var heldByMove string
			reservationErr = tx.QueryRow(`SELECT killer_move_id::text FROM player_killer_move_gu_reservations WHERE gu_instance_id=$1`, coreID).Scan(&heldByMove)
			if reservationErr == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "Cổ hạch tâm đang được Sát Chiêu active giữ chỗ"})
				return
			}
			if reservationErr != sql.ErrNoRows {
				httperr.Respond(c, httperr.Internal("Không thể kiểm tra giữ chỗ Cổ hạch tâm", reservationErr))
				return
			}
			var heldByHouse string
			reservationErr = tx.QueryRow(`SELECT gu_house_id::text FROM player_gu_house_core_reservations WHERE gu_instance_id=$1`, coreID).Scan(&heldByHouse)
			if reservationErr == nil && heldByHouse != houseID {
				c.JSON(http.StatusConflict, gin.H{"error": "Cổ hạch tâm đang được Tiên Cổ Ốc active khác giữ chỗ"})
				return
			}
			if reservationErr != nil && reservationErr != sql.ErrNoRows {
				httperr.Respond(c, httperr.Internal("Không thể kiểm tra giữ chỗ Cổ hạch tâm", reservationErr))
				return
			}
			var legacyMoveUse bool
			if err = tx.QueryRow(`SELECT EXISTS (
				SELECT 1 FROM player_killer_moves AS move, jsonb_array_elements(move.components) AS component
				WHERE move.player_id=$1 AND move.is_equipped=true AND component->>'instance_id'=$2
			)`, playerID, coreID).Scan(&legacyMoveUse); err != nil {
				httperr.Respond(c, httperr.Internal("Không thể kiểm tra Sát Chiêu đang active", err))
				return
			}
			if legacyMoveUse {
				c.JSON(http.StatusConflict, gin.H{"error": "Cổ hạch tâm đã thuộc Sát Chiêu active"})
				return
			}
			if guHouseProductProfile.GuPolicy == composition.GuPolicyReserveActive {
				if _, err = tx.Exec(`INSERT INTO player_gu_house_core_reservations(gu_instance_id,gu_house_id,player_id,gu_id,gu_rank,gu_location,gu_is_locked,gu_state)
				VALUES($1,$2,$3,$4,$5,'INVENTORY',false,$6)
				ON CONFLICT(gu_instance_id) DO UPDATE SET created_at=player_gu_house_core_reservations.created_at
				WHERE player_gu_house_core_reservations.gu_house_id=EXCLUDED.gu_house_id`, coreID, houseID, playerID, guID, rank, state); err != nil {
					c.JSON(http.StatusConflict, gin.H{"error": "Không thể giữ chỗ Cổ hạch tâm cho Tiên Cổ Ốc"})
					return
				}
			}
		}
		if relinkedHouse {
			newCoreBytes, _ := json.Marshal(coreIDs)
			if _, err = tx.Exec(`UPDATE player_gu_houses SET core_gu_instances=$1 WHERE id=$2`, newCoreBytes, houseID); err != nil {
				httperr.Respond(c, httperr.Internal("Không thể cập nhật Cổ hạch tâm đã tái liên kết", err))
				return
			}
		}
	}

	_, err = tx.Exec(`UPDATE player_gu_houses SET is_equipped=$1, updated_at=NOW() WHERE id=$2 AND player_id=$3`, newState, houseID, playerID)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Không thể cập nhật trạng thái trang bị", err))
		return
	}
	if !newState && guHouseProductProfile.GuPolicy == composition.GuPolicyReserveActive {
		if _, err = tx.Exec(`DELETE FROM player_gu_house_core_reservations WHERE gu_house_id=$1 AND player_id=$2`, houseID, playerID); err != nil {
			httperr.Respond(c, httperr.Internal("Không thể giải phóng Cổ hạch tâm", err))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi xác nhận", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_equipped": newState, "message": "Cập nhật trang bị thành công"})
}

func validateGuHouseActivationProfile(raw []byte) error {
	return validateGuHouseActivationProfileMode(raw, true)
}

func validateGuHouseActivationProfileMode(raw []byte, revalidateCurrent bool) error {
	var profile models.GuHouseCombatProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return fmt.Errorf("Profile Tiên Cổ Ốc không hợp lệ; hãy tạo lại từ preview")
	}
	for _, action := range profile.Actions {
		if action.CompositionGraph == nil {
			continue
		}
		graph := *action.CompositionGraph
		if revalidateCurrent {
			if graph.RuleSetRevision != composition.GuHouseActionRuleSet().Revision {
				return fmt.Errorf("Trận Kích dùng catalog rule đã cũ; hãy tạo lại Tiên Cổ Ốc từ preview")
			}
			capabilitiesByEffect := make(map[string]map[string]bool, len(graph.Nodes))
			for _, node := range graph.Nodes {
				spec, found := models.GetGu(node.GuID, node.Rank)
				if !found || !guHouseSnapshotMatchesSource(node, spec.Effects.GetBlocks(), action.EffectBlocks) {
					return fmt.Errorf("Trận Kích không còn khớp effect nguồn mới nhất; hãy cập nhật draft")
				}
				revision, err := gameplayeffects.SourceRevision(spec.Effects.GetBlocks())
				if err != nil || (node.SourceRevision != "" && revision != node.SourceRevision) {
					return fmt.Errorf("Trận Kích không còn khớp revision effect nguồn; hãy cập nhật draft")
				}
				for _, grant := range spec.CompositionContributions.Grants {
					if grant.EffectBlockID == node.SourceBlockID && grant.OperationIndex == valueOrZero(node.OperationIndex) {
						if capabilitiesByEffect[node.EffectKey] == nil {
							capabilitiesByEffect[node.EffectKey] = make(map[string]bool)
						}
						capabilitiesByEffect[node.EffectKey][grant.Capability] = true
					}
				}
			}
			if len(graph.Nodes) > 0 {
				if _, err := composition.ResolveWithRules(graph, composition.GuHouseActionProfile(), action.ComponentGuInstances, composition.GuHouseActionRuleSet(), capabilitiesByEffect); err != nil {
					return fmt.Errorf("Trận Kích draft không còn hợp lệ theo rule/effect hiện hành: %w", err)
				}
			}
			continue
		}
		if len(graph.Edges) > 0 && graph.RuleSetRevision == "" {
			return fmt.Errorf("Trận Kích active thiếu revision rule đã pin")
		}
		for _, edge := range graph.Edges {
			if edge.RuleID == "" {
				return fmt.Errorf("Trận Kích active thiếu rule provenance đã pin")
			}
		}
		if len(graph.Nodes) > 0 {
			if _, err := composition.Resolve(graph, composition.GuHouseActionProfile(), action.ComponentGuInstances); err != nil {
				return fmt.Errorf("Snapshot Trận Kích active không hợp lệ: %w", err)
			}
		}
	}
	return nil
}

func RepairGuHouse(c *gin.Context) {
	playerID, ok := killerMoveOwner(c)
	if !ok {
		return
	}
	houseID := c.Param("id")

	tx, err := db.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		httperr.Respond(c, httperr.Internal("Database error", err))
		return
	}
	defer tx.Rollback()

	var currentDurability float64
	var rank int
	var isEquipped bool
	var coreBytes, profBytes []byte
	err = tx.QueryRow(`
		SELECT current_durability, rank, is_equipped, core_gu_instances, house_profile 
		FROM player_gu_houses 
		WHERE id=$1 AND player_id=$2 FOR UPDATE
	`, houseID, playerID).Scan(&currentDurability, &rank, &isEquipped, &coreBytes, &profBytes)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy Tiên Cổ Ốc"})
		return
	}

	var houseProf models.GuHouseCombatProfile
	_ = json.Unmarshal(profBytes, &houseProf)

	var coreIDs []string
	_ = json.Unmarshal(coreBytes, &coreIDs)

	relinked := false
	highestRank := rank

	// Auto-relink / reconcile each core Gu if instance changed or upgraded
	for idx, coreID := range coreIDs {
		var guID, state, location string
		var guRank int
		var isLocked bool
		qErr := tx.QueryRow(`
			SELECT gu_id, gu_rank, state, location, is_locked 
			FROM player_gu 
			WHERE id=$1 AND player_id=$2 FOR UPDATE
		`, coreID, playerID).Scan(&guID, &guRank, &state, &location, &isLocked)

		if qErr != nil || guRank < 6 || !strings.EqualFold(state, "ACTIVE") || location != "INVENTORY" || isLocked {
			var targetGuID string
			if idx < len(houseProf.CoreGuIDs) {
				targetGuID = houseProf.CoreGuIDs[idx]
			}
			if targetGuID != "" {
				var newID string
				var newRank int
				findErr := tx.QueryRow(`
					SELECT id::text, gu_rank FROM player_gu 
					WHERE player_id=$1 AND gu_id=$2 AND gu_rank >= 6 AND state='ACTIVE' AND location='INVENTORY' AND is_locked=false 
					ORDER BY gu_rank DESC LIMIT 1
				`, playerID, targetGuID).Scan(&newID, &newRank)

				if findErr == nil && newID != "" {
					coreIDs[idx] = newID
					coreID = newID
					relinked = true
					guRank = newRank
					guID = targetGuID
					state = "ACTIVE"
				}
			}
		}

		if guRank > highestRank {
			highestRank = guRank
		}
	}

	var cost int64
	if currentDurability < 100 {
		cost = int64(math.Ceil((100.0 - currentDurability) * float64(highestRank) * 10))

		var currentStones int64
		err = tx.QueryRow(`SELECT primeval_stones FROM players WHERE id=$1 FOR UPDATE`, playerID).Scan(&currentStones)
		if err != nil || currentStones < cost {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Không đủ Nguyên Thạch tu sửa (cần %d)", cost)})
			return
		}

		_, _ = tx.Exec(`UPDATE players SET primeval_stones = primeval_stones - $1 WHERE id=$2`, cost, playerID)
		currentDurability = 100.0
	}

	if relinked || highestRank != rank || currentDurability == 100.0 {
		newCoreBytes, _ := json.Marshal(coreIDs)
		houseProf.Rank = highestRank
		newProfBytes, _ := json.Marshal(houseProf)

		_, err = tx.Exec(`
			UPDATE player_gu_houses 
			SET current_durability = $1, rank = $2, core_gu_instances = $3::jsonb, house_profile = $4::jsonb, updated_at = NOW() 
			WHERE id = $5
		`, currentDurability, highestRank, newCoreBytes, newProfBytes, houseID)
		if err != nil {
			httperr.Respond(c, httperr.Internal("Không thể cập nhật trạng thái Tiên Cổ Ốc", err))
			return
		}

		// If currently equipped and core Gus relinked, update reservations
		if isEquipped && relinked && guHouseProductProfile.GuPolicy == composition.GuPolicyReserveActive {
			_, _ = tx.Exec(`DELETE FROM player_gu_house_core_reservations WHERE gu_house_id=$1`, houseID)
			for _, cid := range coreIDs {
				var gID, gState string
				var gRank int
				if err := tx.QueryRow(`SELECT gu_id, gu_rank, state FROM player_gu WHERE id=$1`, cid).Scan(&gID, &gRank, &gState); err == nil {
					_, _ = tx.Exec(`
						INSERT INTO player_gu_house_core_reservations(gu_instance_id, gu_house_id, player_id, gu_id, gu_rank, gu_location, gu_is_locked, gu_state)
						VALUES($1, $2, $3, $4, $5, 'INVENTORY', false, $6)
						ON CONFLICT (gu_instance_id) DO UPDATE SET updated_at=NOW()
					`, cid, houseID, playerID, gID, gRank, gState)
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		httperr.Respond(c, httperr.Internal("Lỗi xác nhận", err))
		return
	}

	msg := "Đã tu sửa và đồng bộ Tiên Cổ Ốc thành công!"
	if cost == 0 && !relinked {
		msg = "Tiên Cổ Ốc vẫn nguyên vẹn và Cổ hạch tâm đã đồng bộ hoàn hảo!"
	}

	c.JSON(http.StatusOK, gin.H{
		"current_durability": currentDurability,
		"rank":               highestRank,
		"cost":               cost,
		"relinked":           relinked,
		"message":            msg,
	})
}

// ApplyGuHouseDamage inflicts structural damage on a Gu House's durability and transmits
// shock damage to its core Gu instances (core_gu_instances in player_gu).
func ApplyGuHouseDamage(tx *sql.Tx, houseID string, damage float64) (float64, error) {
	if damage <= 0 {
		return 0, nil
	}
	var currentDur float64
	var coreBytes []byte
	var playerID uuid.UUID
	err := tx.QueryRow(`
		SELECT player_id, current_durability, core_gu_instances
		FROM player_gu_houses
		WHERE id = $1 FOR UPDATE
	`, houseID).Scan(&playerID, &currentDur, &coreBytes)
	if err != nil {
		return 0, err
	}
	newDur := math.Max(0, currentDur-damage)
	if _, err := tx.Exec(`UPDATE player_gu_houses SET current_durability = $1, updated_at = NOW() WHERE id = $2`, newDur, houseID); err != nil {
		return 0, err
	}
	// Shock damage transmitted to core Gu instances: 30% of house damage
	if len(coreBytes) > 0 {
		var coreIDs []string
		if err := json.Unmarshal(coreBytes, &coreIDs); err == nil && len(coreIDs) > 0 {
			guShockDamage := math.Max(1.0, damage*0.3)
			for _, guID := range coreIDs {
				_, _ = db.ApplyGuDurabilityDegradation(tx, playerID.String(), guID, guShockDamage)
			}
		}
	}
	return newDur, nil
}
