package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/composition"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
)

type adminCompositionSource struct {
	Key            string `json:"key"`
	GuID           string `json:"gu_id"`
	Rank           int    `json:"rank"`
	BlockID        string `json:"block_id"`
	OperationIndex int    `json:"operation_index"`
	Role           string `json:"role"`
	SourceRevision string `json:"source_revision,omitempty"`
}

type adminCompositionDraft struct {
	SchemaVersion   int                      `json:"schema_version"`
	Name            string                   `json:"name"`
	ProductProfile  string                   `json:"product_profile"`
	OutputEffectKey string                   `json:"output_effect_key"`
	RuleRevision    string                   `json:"rule_set_revision"`
	Sources         []adminCompositionSource `json:"sources"`
	Edges           []composition.Edge       `json:"edges"`
}

type adminCompositionSourceOption struct {
	Key            string                      `json:"key"`
	GuID           string                      `json:"gu_id"`
	Rank           int                         `json:"rank"`
	BlockID        string                      `json:"block_id"`
	OperationIndex int                         `json:"operation_index"`
	SourceRevision string                      `json:"source_revision"`
	Trigger        string                      `json:"trigger"`
	Context        string                      `json:"context"`
	Executor       string                      `json:"executor"`
	Operation      gameplayeffects.Operation   `json:"operation"`
	Target         gameplayeffects.Target      `json:"target"`
	Conditions     []gameplayeffects.Condition `json:"conditions,omitempty"`
	Capabilities   []string                    `json:"capabilities,omitempty"`
}

type adminCompositionProfile struct {
	ID      string              `json:"id"`
	Label   string              `json:"label"`
	Profile composition.Profile `json:"profile"`
	RuleSet composition.RuleSet `json:"rule_set"`
}

type adminCompositionTemplate struct {
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Definition adminCompositionDraft `json:"definition"`
}

func listAdminCompositionTemplates(ctx context.Context, profileID string) ([]adminCompositionTemplate, error) {
	rows, err := db.DB.QueryContext(ctx, `SELECT id::text,name,definition FROM admin_effect_composition_drafts WHERE status='DRAFT' AND product_profile=$1 ORDER BY updated_at DESC`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	templates := make([]adminCompositionTemplate, 0)
	for rows.Next() {
		var template adminCompositionTemplate
		var raw []byte
		if err := rows.Scan(&template.ID, &template.Name, &raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &template.Definition); err != nil {
			continue
		}
		templates = append(templates, template)
	}
	return templates, rows.Err()
}

type resolvedAdminComposition struct {
	Draft         adminCompositionDraft        `json:"draft"`
	Graph         composition.Graph            `json:"graph"`
	Effects       gameplayeffects.BlockSet     `json:"effects"`
	Compatibility []effectCompatibilityContext `json:"compatibility"`
}

func adminCompositionProfiles() []adminCompositionProfile {
	return []adminCompositionProfile{
		{ID: "killer_move.v1", Label: "Sát Chiêu", Profile: composition.KillerMoveProfile(), RuleSet: composition.KillerMoveRuleSet()},
		{ID: "gu_house_action.v1", Label: "Trận Kích · Tiên Cổ Ốc", Profile: composition.GuHouseActionProfile(), RuleSet: composition.GuHouseActionRuleSet()},
		{ID: "teleport_formation.v1", Label: "Trận Pháp dịch chuyển", Profile: composition.TeleportFormationProfile(), RuleSet: composition.TeleportFormationRuleSet()},
	}
}

func findAdminCompositionProfile(id string) (adminCompositionProfile, bool) {
	for _, profile := range adminCompositionProfiles() {
		if profile.ID == id {
			return profile, true
		}
	}
	return adminCompositionProfile{}, false
}

func AdminEffectCompositionCatalog(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	profileID := strings.TrimSpace(c.Query("product_profile"))
	profiles := adminCompositionProfiles()
	var options []adminCompositionSourceOption
	if profileID != "" {
		profile, ok := findAdminCompositionProfile(profileID)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Product profile không tồn tại"})
			return
		}
		var err error
		options, err = listAdminCompositionSourceOptions(c, profile)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc effect source: " + err.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"profiles": profiles, "sources": options})
}

func listAdminCompositionSourceOptions(c *gin.Context, profile adminCompositionProfile) ([]adminCompositionSourceOption, error) {
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id,rank,effects,composition_contributions FROM gu_dictionary ORDER BY id,rank`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	options := make([]adminCompositionSourceOption, 0)
	for rows.Next() {
		var guID string
		var rank int
		var rawEffects, rawGrants []byte
		if err := rows.Scan(&guID, &rank, &rawEffects, &rawGrants); err != nil {
			return nil, err
		}
		blocks, err := gameplayeffects.DecodeBlockSet(rawEffects)
		if err != nil {
			continue
		}
		grants, err := composition.ParseGrantCatalog(rawGrants, []gameplayeffects.Block(blocks))
		if err != nil {
			continue
		}
		revision, err := gameplayeffects.SourceRevision([]gameplayeffects.Block(blocks))
		if err != nil {
			continue
		}
		grantByOperation := make(map[string][]string, len(grants.Grants))
		for _, grant := range grants.Grants {
			key := fmt.Sprintf("%s:%d", grant.EffectBlockID, grant.OperationIndex)
			grantByOperation[key] = append(grantByOperation[key], grant.Capability)
		}
		for _, block := range blocks {
			contextID := profile.Profile.ContextByTrigger[string(block.Trigger)]
			executor := profile.Profile.ExecutorByContext[contextID]
			if contextID == "" || executor == "" {
				continue
			}
			for index, operation := range block.Operations {
				target := block.Target
				if operation.Target != nil {
					target = *operation.Target
				}
				if !adminCompositionOperationAllowed(profile.ID, block.Trigger, block.Conditions, operation, target) {
					continue
				}
				key := fmt.Sprintf("%s:%d:%s:%d", guID, rank, block.ID, index)
				options = append(options, adminCompositionSourceOption{
					Key: key, GuID: guID, Rank: rank, BlockID: block.ID, OperationIndex: index,
					SourceRevision: revision, Trigger: string(block.Trigger), Context: contextID, Executor: executor,
					Operation: operation, Target: target, Conditions: append([]gameplayeffects.Condition(nil), block.Conditions...),
					Capabilities: grantByOperation[fmt.Sprintf("%s:%d", block.ID, index)],
				})
			}
		}
	}
	return options, rows.Err()
}

func adminCompositionOperationAllowed(profileID string, trigger gameplayeffects.Trigger, conditions []gameplayeffects.Condition, operation gameplayeffects.Operation, target gameplayeffects.Target) bool {
	conditionContext := ""
	switch profileID {
	case "killer_move.v1":
		if trigger == gameplayeffects.TriggerOnAction {
			conditionContext = "KILLER_MOVE_COMBAT"
		} else if trigger == gameplayeffects.TriggerOnUse {
			conditionContext = "KILLER_MOVE_ON_USE"
		}
	case "gu_house_action.v1":
		conditionContext = "GU_HOUSE_COMBAT"
	case "teleport_formation.v1":
		conditionContext = "TELEPORT_ARRAY_ARRIVAL"
	}
	if conditionContext == "" || unsupportedConditionReason(conditionContext, conditions) != "" {
		return false
	}
	if operation.Op == gameplayeffects.OpUnique && len(conditions) > 0 {
		return false
	}
	switch profileID {
	case "killer_move.v1":
		if trigger == gameplayeffects.TriggerOnAction {
			return killerMoveOperationAllowed(operation, target)
		}
		return trigger == gameplayeffects.TriggerOnUse && worldUseOperationAllowed(operation, target)
	case "gu_house_action.v1":
		return trigger == gameplayeffects.TriggerOnAction && len(conditions) == 0 && guHouseActionOperationAllowed(operation, target)
	case "teleport_formation.v1":
		return trigger == gameplayeffects.TriggerOnUse && worldUseOperationAllowed(operation, target)
	default:
		return false
	}
}

func AdminValidateEffectComposition(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	var draft adminCompositionDraft
	if err := c.ShouldBindJSON(&draft); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Composition draft không hợp lệ: " + err.Error()})
		return
	}
	resolved, err := resolveAdminComposition(c, draft)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"valid": true, "preview": resolved})
}

func resolveAdminComposition(c *gin.Context, draft adminCompositionDraft) (resolvedAdminComposition, error) {
	profile, ok := findAdminCompositionProfile(draft.ProductProfile)
	if !ok {
		return resolvedAdminComposition{}, fmt.Errorf("Product profile không tồn tại")
	}
	if len(profile.RuleSet.Rules) == 0 {
		return resolvedAdminComposition{}, fmt.Errorf("Product profile chưa khai báo rule/output template")
	}
	draft.SchemaVersion = composition.GraphSchemaVersion
	draft.Name = strings.TrimSpace(draft.Name)
	if draft.Name == "" || len(draft.Name) > 120 || len(draft.Sources) == 0 {
		return resolvedAdminComposition{}, fmt.Errorf("Tên draft và ít nhất một effect source là bắt buộc")
	}
	if draft.RuleRevision != "" && draft.RuleRevision != profile.RuleSet.Revision {
		return resolvedAdminComposition{}, fmt.Errorf("Rule revision đã đổi; hãy nạp catalog mới rồi dựng lại draft")
	}
	draft.RuleRevision = profile.RuleSet.Revision
	options, err := listAdminCompositionSourceOptions(c, profile)
	if err != nil {
		return resolvedAdminComposition{}, err
	}
	optionByKey := make(map[string]adminCompositionSourceOption, len(options))
	for _, option := range options {
		optionByKey[option.Key] = option
	}
	graphNodes := make([]composition.Node, 0, len(draft.Sources))
	selected := make([]string, 0, len(draft.Sources))
	capabilitiesByEffect := make(map[string]map[string]bool, len(draft.Sources))
	blocks := make([]gameplayeffects.Block, 0, len(draft.Sources))
	blockIDByEffectKey := make(map[string]string, len(draft.Sources))
	seenKeys := make(map[string]bool, len(draft.Sources))
	for index := range draft.Sources {
		source := &draft.Sources[index]
		option, found := optionByKey[source.Key]
		if !found || option.GuID != source.GuID || option.Rank != source.Rank || option.BlockID != source.BlockID || option.OperationIndex != source.OperationIndex {
			return resolvedAdminComposition{}, fmt.Errorf("Effect source %q không còn hợp lệ theo catalog hiện tại", source.Key)
		}
		if seenKeys[source.Key] || (source.Role != string(composition.NodeOutput) && source.Role != string(composition.NodeSupport)) {
			return resolvedAdminComposition{}, fmt.Errorf("Effect source trùng hoặc có role không hợp lệ")
		}
		seenKeys[source.Key] = true
		source.SourceRevision = option.SourceRevision
		syntheticInstance := fmt.Sprintf("catalog:%s:%d", option.GuID, option.Rank)
		selected = append(selected, syntheticInstance)
		contextID, executor := option.Context, option.Executor
		graphNodes = append(graphNodes, composition.Node{
			EffectKey: source.Key, SourceBlockID: source.BlockID, SourceRevision: option.SourceRevision,
			OperationIndex: intPointer(source.OperationIndex), Role: composition.NodeRole(source.Role),
			GuInstanceID: syntheticInstance, GuID: option.GuID, Rank: option.Rank,
			Trigger: option.Trigger, Context: contextID, Executor: executor,
		})
		for _, capability := range option.Capabilities {
			if capabilitiesByEffect[source.Key] == nil {
				capabilitiesByEffect[source.Key] = map[string]bool{}
			}
			capabilitiesByEffect[source.Key][capability] = true
		}
		operation := option.Operation
		operation.Target = nil
		blockID := fmt.Sprintf("%s_%02d", profile.RuleSet.Rules[0].OutputTemplate.BlockIDPrefix, index+1)
		blockIDByEffectKey[source.Key] = blockID
		blocks = append(blocks, gameplayeffects.Block{
			ID:      blockID,
			Trigger: gameplayeffects.Trigger(option.Trigger), Target: option.Target,
			Conditions: append([]gameplayeffects.Condition(nil), option.Conditions...), Operations: []gameplayeffects.Operation{operation},
		})
	}
	if len(selected) == 0 {
		return resolvedAdminComposition{}, fmt.Errorf("Composition draft không có source")
	}
	uniqueSelected := make([]string, 0, len(selected))
	seenInstances := make(map[string]bool)
	for _, id := range selected {
		if !seenInstances[id] {
			seenInstances[id] = true
			uniqueSelected = append(uniqueSelected, id)
		}
	}
	graph := composition.Graph{
		SchemaVersion: composition.GraphSchemaVersion, Product: profile.Profile.Product, ProductProfile: profile.ID,
		Context: profile.Profile.Context, Executor: profile.Profile.Executor, RuleSetRevision: draft.RuleRevision,
		CoreGuInstanceID: "", OutputEffectKey: draft.OutputEffectKey, Nodes: graphNodes, Edges: draft.Edges,
	}
	for _, node := range graphNodes {
		if node.Role == composition.NodeOutput {
			graph.CoreGuInstanceID = node.GuInstanceID
			break
		}
	}
	resolved, err := composition.ResolveWithRules(graph, profile.Profile, uniqueSelected, profile.RuleSet, capabilitiesByEffect)
	if err != nil {
		return resolvedAdminComposition{}, err
	}
	if len(draft.Sources) > profile.Profile.MaxNodes || len(uniqueSelected) > profile.Profile.MaxGuComponents {
		return resolvedAdminComposition{}, fmt.Errorf("Composition vượt giới hạn node/component của product profile")
	}
	for index := range resolved.Nodes {
		resolved.Nodes[index].BlockID = blockIDByEffectKey[resolved.Nodes[index].EffectKey]
	}
	if _, err := (gameplayeffects.Compiler{}).Compile(blocks); err != nil {
		return resolvedAdminComposition{}, err
	}
	draft.Sources = draft.Sources[:0]
	for _, node := range resolved.Nodes {
		draft.Sources = append(draft.Sources, adminCompositionSource{Key: node.EffectKey, GuID: node.GuID, Rank: node.Rank, BlockID: node.SourceBlockID, OperationIndex: valueOrZero(node.OperationIndex), Role: string(node.Role), SourceRevision: node.SourceRevision})
	}
	draft.Edges = append([]composition.Edge(nil), resolved.Edges...)
	return resolvedAdminComposition{Draft: draft, Graph: resolved, Effects: gameplayeffects.BlockSet(blocks), Compatibility: analyzeEffectCompatibility(blocks)}, nil
}

func intPointer(value int) *int { return &value }

func AdminListEffectCompositionDrafts(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id,name,product_profile,definition,created_at,updated_at FROM admin_effect_composition_drafts WHERE status='DRAFT' ORDER BY updated_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc composition draft"})
		return
	}
	defer rows.Close()
	type rowDTO struct {
		ID             string                `json:"id"`
		Name           string                `json:"name"`
		ProductProfile string                `json:"product_profile"`
		Definition     adminCompositionDraft `json:"definition"`
		CreatedAt      string                `json:"created_at"`
		UpdatedAt      string                `json:"updated_at"`
	}
	items := make([]rowDTO, 0)
	for rows.Next() {
		var item rowDTO
		var raw []byte
		var createdAt, updatedAt any
		if err := rows.Scan(&item.ID, &item.Name, &item.ProductProfile, &raw, &createdAt, &updatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := json.Unmarshal(raw, &item.Definition); err != nil {
			continue
		}
		item.CreatedAt, item.UpdatedAt = fmt.Sprint(createdAt), fmt.Sprint(updatedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"drafts": items})
}

func saveAdminEffectCompositionDraft(c *gin.Context, update bool) {
	adminID, ok := requireAdmin(c)
	if !ok {
		return
	}
	var request struct {
		ID    string                `json:"id"`
		Draft adminCompositionDraft `json:"draft"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resolved, err := resolveAdminComposition(c, request.Draft)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}
	definition, err := json.Marshal(resolved.Draft)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể serialize draft"})
		return
	}
	if update {
		id, parseErr := uuid.Parse(c.Param("id"))
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Draft ID không hợp lệ"})
			return
		}
		result, execErr := db.DB.ExecContext(c.Request.Context(), `UPDATE admin_effect_composition_drafts SET name=$1,product_profile=$2,definition=$3::jsonb,updated_by=$4,updated_at=NOW() WHERE id=$5 AND status='DRAFT'`, resolved.Draft.Name, resolved.Draft.ProductProfile, string(definition), adminID, id)
		if execErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": execErr.Error()})
			return
		}
		changed, _ := result.RowsAffected()
		if changed == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Draft không tồn tại hoặc đã archive"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id.String(), "preview": resolved})
		return
	}
	var id uuid.UUID
	err = db.DB.QueryRowContext(c.Request.Context(), `INSERT INTO admin_effect_composition_drafts(name,product_profile,definition,created_by,updated_by) VALUES($1,$2,$3::jsonb,$4,$4) RETURNING id`, resolved.Draft.Name, resolved.Draft.ProductProfile, string(definition), adminID).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id.String(), "preview": resolved})
}

func AdminCreateEffectCompositionDraft(c *gin.Context) { saveAdminEffectCompositionDraft(c, false) }
func AdminUpdateEffectCompositionDraft(c *gin.Context) { saveAdminEffectCompositionDraft(c, true) }

func AdminArchiveEffectCompositionDraft(c *gin.Context) {
	if _, ok := requireAdmin(c); !ok {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Draft ID không hợp lệ"})
		return
	}
	result, err := db.DB.ExecContext(c.Request.Context(), `UPDATE admin_effect_composition_drafts SET status='ARCHIVED',updated_at=NOW() WHERE id=$1 AND status='DRAFT'`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Draft không tồn tại"})
		return
	}
	c.Status(http.StatusNoContent)
}
