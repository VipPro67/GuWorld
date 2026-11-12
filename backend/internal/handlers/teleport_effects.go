package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/composition"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type teleportArrayEffectEnvelope struct {
	SchemaVersion int                     `json:"schema_version"`
	Blocks        []gameplayeffects.Block `json:"blocks"`
}

type teleportArrayEffectProfile struct {
	ProfileVersion       int                         `json:"profile_version"`
	ProductProfile       string                      `json:"product_profile"`
	Context              string                      `json:"context"`
	Executor             string                      `json:"executor"`
	GuRank               int                         `json:"gu_rank"`
	CompositionGraph     composition.Graph           `json:"composition_graph"`
	ComponentGuInstances []string                    `json:"component_gu_instances"`
	Effects              teleportArrayEffectEnvelope `json:"effects"`
}

type teleportArrayEffectPreview struct {
	EffectEnabled        bool                        `json:"effect_enabled"`
	ProductProfile       string                      `json:"product_profile"`
	Context              string                      `json:"context"`
	Executor             string                      `json:"executor"`
	GuRank               int                         `json:"gu_rank"`
	CompositionGraph     composition.Graph           `json:"composition_graph"`
	ComponentGuInstances []string                    `json:"component_gu_instances"`
	Effects              teleportArrayEffectEnvelope `json:"effects"`
	PreviewFingerprint   string                      `json:"preview_fingerprint"`
}

func buildTeleportArrayEffectProfile(tx *sql.Tx, playerID uuid.UUID, req models.BuildTeleportArrayReq) (teleportArrayEffectProfile, string, error) {
	profile := composition.TeleportFormationProfile()
	graph := req.CompositionGraph
	if len(graph.Nodes) == 0 {
		return teleportArrayEffectProfile{}, "", fmt.Errorf("cần chọn Cổ hạch tâm và effect đầu ra cho effect Trận Pháp")
	}
	selected := make([]string, 0, len(graph.Nodes))
	seen := make(map[string]bool, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if node.GuInstanceID == "" || seen[node.GuInstanceID] {
			continue
		}
		seen[node.GuInstanceID] = true
		selected = append(selected, node.GuInstanceID)
	}
	sort.Strings(selected)
	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		return teleportArrayEffectProfile{}, "", err
	}
	ruleSet := composition.TeleportFormationRuleSet()
	capabilitiesByEffect := make(map[string]map[string]bool, len(graph.Nodes))
	selectedOptions := make(map[string]killerMoveEffectOption, len(graph.Nodes))
	for _, node := range graph.Nodes {
		gu, ok := owned[node.GuInstanceID]
		if !ok {
			return teleportArrayEffectProfile{}, "", fmt.Errorf("Cổ %q không thuộc hành trang hiện tại", node.GuInstanceID)
		}
		options, optionErr := getKillerMoveEffectOptions(gu)
		if optionErr != nil {
			return teleportArrayEffectProfile{}, "", optionErr
		}
		for _, option := range options {
			if option.Key != node.EffectKey || option.SourceBlockID != node.SourceBlockID || node.OperationIndex == nil || option.OperationIndex != *node.OperationIndex || option.Trigger != node.Trigger || !option.TravelSupported {
				continue
			}
			selectedOptions[node.EffectKey] = option
			for _, capability := range option.CompositionCapabilities {
				if capabilitiesByEffect[node.EffectKey] == nil {
					capabilitiesByEffect[node.EffectKey] = map[string]bool{}
				}
				capabilitiesByEffect[node.EffectKey][capability] = true
			}
			break
		}
	}
	resolved, err := composition.ResolveWithRules(graph, profile, selected, ruleSet, capabilitiesByEffect)
	if err != nil {
		return teleportArrayEffectProfile{}, "", err
	}
	blocks := make([]gameplayeffects.Block, 0, len(resolved.Nodes))
	highestRank := 1
	for index, node := range resolved.Nodes {
		gu, ok := owned[node.GuInstanceID]
		if !ok || gu.GuID != node.GuID || gu.Rank != node.Rank || !strings.EqualFold(gu.State, "ACTIVE") || gu.Location != "INVENTORY" || gu.IsLocked {
			return teleportArrayEffectProfile{}, "", fmt.Errorf("Cổ %q không còn đúng instance/rank hoặc không thể dùng làm thành phần Trận Pháp", node.GuInstanceID)
		}
		if gu.Rank > highestRank {
			highestRank = gu.Rank
		}
		selectedEffect, exists := selectedOptions[node.EffectKey]
		if !exists {
			return teleportArrayEffectProfile{}, "", fmt.Errorf("effect %q không có executor arrival Trận Pháp hoặc provenance đã thay đổi", node.EffectKey)
		}
		operation := selectedEffect.Operation
		operation.Target = nil
		if operation.Op == gameplayeffects.OpStatus {
			if operation.Turns <= 0 {
				operation.Turns = node.Rank
			}
			if operation.Value <= 0 {
				operation.Value = float64(node.Rank)
			}
		}
		blocks = append(blocks, gameplayeffects.Block{
			ID: fmt.Sprintf("ARRIVAL_%02d", index+1), Trigger: gameplayeffects.TriggerOnUse,
			Target: selectedEffect.Target, Conditions: append([]gameplayeffects.Condition(nil), selectedEffect.Conditions...),
			Operations: []gameplayeffects.Operation{operation},
		})
		resolved.Nodes[index].BlockID = fmt.Sprintf("ARRIVAL_%02d", index+1)
	}
	if _, err := (gameplayeffects.Compiler{}).Compile(blocks); err != nil {
		return teleportArrayEffectProfile{}, "", fmt.Errorf("effect Trận Pháp không compile được: %w", err)
	}
	rawBlocks, err := json.Marshal(blocks)
	if err != nil {
		return teleportArrayEffectProfile{}, "", err
	}
	canonicalBlocks, issues := effectcatalog.CanonicalBlockV2Envelope(rawBlocks, effectcatalog.Location{Source: "teleport_array", Key: "arrival_effect"})
	if len(issues) > 0 {
		return teleportArrayEffectProfile{}, "", &effectcatalog.ValidationError{Issues: issues}
	}
	var effectEnvelope teleportArrayEffectEnvelope
	if err := json.Unmarshal(canonicalBlocks, &effectEnvelope); err != nil {
		return teleportArrayEffectProfile{}, "", err
	}
	profileSnapshot := teleportArrayEffectProfile{
		ProfileVersion: 1, ProductProfile: profile.ID, Context: profile.Context, Executor: profile.Executor,
		GuRank: highestRank, CompositionGraph: resolved, ComponentGuInstances: selected,
		Effects: effectEnvelope,
	}
	canonicalReq := req
	canonicalReq.PreviewFingerprint = ""
	fingerprintInput, err := json.Marshal(struct {
		Tier    string                     `json:"tier"`
		Graph   composition.Graph          `json:"graph"`
		Profile teleportArrayEffectProfile `json:"profile"`
	}{Tier: canonicalReq.Tier, Graph: resolved, Profile: profileSnapshot})
	if err != nil {
		return teleportArrayEffectProfile{}, "", err
	}
	digest := sha256.Sum256(fingerprintInput)
	return profileSnapshot, hex.EncodeToString(digest[:]), nil
}

func teleportArrayRequestHasEffect(req models.BuildTeleportArrayReq) bool {
	return req.CompositionGraph.Product != "" || req.CompositionGraph.ProductProfile != "" ||
		req.CompositionGraph.OutputEffectKey != "" || len(req.CompositionGraph.Nodes) > 0 || len(req.CompositionGraph.Edges) > 0
}

func reserveTeleportArrayGuComponents(tx *sql.Tx, playerID, arrayID uuid.UUID, profile teleportArrayEffectProfile) error {
	profileGu := make(map[string]composition.Node)
	for _, node := range profile.CompositionGraph.Nodes {
		profileGu[node.GuInstanceID] = node
	}
	for _, instanceID := range profile.ComponentGuInstances {
		node, exists := profileGu[instanceID]
		if !exists {
			return fmt.Errorf("Cổ giữ chỗ %q không có provenance trong composition graph", instanceID)
		}
		var guID, state, location string
		var rank int
		var locked bool
		if err := tx.QueryRow(`SELECT gu_id,gu_rank,state,location,is_locked FROM player_gu WHERE id=$1 AND player_id=$2 FOR UPDATE`, instanceID, playerID).Scan(&guID, &rank, &state, &location, &locked); err != nil {
			return fmt.Errorf("không thể khóa Cổ thành phần %q: %w", instanceID, err)
		}
		if guID != node.GuID || rank != node.Rank || !strings.EqualFold(state, "ACTIVE") || location != "INVENTORY" || locked {
			return fmt.Errorf("Cổ thành phần %q đã đổi trạng thái hoặc không còn khả dụng", instanceID)
		}
		reserved, err := guHasActiveProductReservation(tx, instanceID)
		if err != nil {
			return fmt.Errorf("không thể kiểm tra reservation của Cổ %q: %w", instanceID, err)
		}
		if reserved {
			return fmt.Errorf("Cổ thành phần %q đã được sản phẩm active khác giữ chỗ", instanceID)
		}
		if _, err := tx.Exec(`INSERT INTO teleport_array_gu_reservations(gu_instance_id,array_id,owner_id,gu_id,gu_rank,gu_location,gu_is_locked)
			VALUES($1,$2,$3,$4,$5,$6,$7)`, instanceID, arrayID, playerID, guID, rank, location, locked); err != nil {
			return fmt.Errorf("không thể giữ chỗ Cổ %q cho Trận Pháp: %w", instanceID, err)
		}
	}
	return nil
}

func TeleportArrayEffectOptions(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở danh sách Cổ Trận Pháp"})
		return
	}
	defer tx.Rollback()
	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc Cổ thành phần Trận Pháp"})
		return
	}
	ids := make([]string, 0, len(owned))
	for id := range owned {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	gus := make([]ownedKillerMoveGu, 0, len(ids))
	options := make([]killerMoveEffectOption, 0)
	for _, id := range ids {
		gu := owned[id]
		gus = append(gus, gu)
		if !strings.EqualFold(gu.State, "ACTIVE") || gu.Location != "INVENTORY" || gu.IsLocked {
			continue
		}
		guOptions, err := getKillerMoveEffectOptions(gu)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc effect Cổ cho Trận Pháp"})
			return
		}
		for _, option := range guOptions {
			if option.TravelSupported {
				options = append(options, option)
			}
		}
	}
	productProfile := composition.TeleportFormationProfile()
	c.JSON(http.StatusOK, gin.H{
		"gus": gus, "effect_options": options,
		"product_profile": productProfile.ID, "max_components": productProfile.MaxGuComponents,
		"max_effects": productProfile.MaxNodes, "composition_rule_set": composition.TeleportFormationRuleSet(),
	})
}

func PreviewTeleportArray(c *gin.Context) {
	playerID, exists := middleware.GetPlayerID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req models.BuildTeleportArrayReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ: " + err.Error()})
		return
	}
	if _, ok := teleportTemplates[req.Tier]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Phẩm cấp Trận Pháp không tồn tại"})
		return
	}
	tx, err := db.DB.BeginTx(c.Request.Context(), &sql.TxOptions{ReadOnly: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể mở preview Trận Pháp"})
		return
	}
	defer tx.Rollback()
	effectProfile, fingerprint, err := buildTeleportArrayEffectProfile(tx, playerID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, teleportArrayEffectPreview{
		EffectEnabled: true, ProductProfile: effectProfile.ProductProfile,
		Context: effectProfile.Context, Executor: effectProfile.Executor, GuRank: effectProfile.GuRank,
		CompositionGraph: effectProfile.CompositionGraph, ComponentGuInstances: effectProfile.ComponentGuInstances,
		Effects: effectProfile.Effects, PreviewFingerprint: fingerprint,
	})
}

func validateTeleportArrayEffectProfile(raw []byte) (teleportArrayEffectProfile, error) {
	var profile teleportArrayEffectProfile
	if err := json.Unmarshal(raw, &profile); err != nil {
		return teleportArrayEffectProfile{}, fmt.Errorf("effect profile không hợp lệ: %w", err)
	}
	expected := composition.TeleportFormationProfile()
	if profile.ProfileVersion != 1 || profile.ProductProfile != expected.ID || profile.Context != expected.Context || profile.Executor != expected.Executor {
		return teleportArrayEffectProfile{}, fmt.Errorf("effect profile không khớp Trận Pháp/executor")
	}
	graph := profile.CompositionGraph
	var resolved composition.Graph
	var err error
	if graph.RuleSetRevision != "" {
		ruleSet := composition.TeleportFormationRuleSet()
		capabilitiesByEffect := make(map[string]map[string]bool, len(graph.Nodes))
		for _, node := range graph.Nodes {
			spec, found := models.GetGu(node.GuID, node.Rank)
			if !found || node.SourceBlockID == "" || node.OperationIndex == nil || !teleportSnapshotMatchesSource(node, spec.Effects.GetBlocks(), profile.Effects.Blocks) {
				return teleportArrayEffectProfile{}, fmt.Errorf("provenance Block v2 của Trận Pháp không khớp Cổ nguồn")
			}
			for _, grant := range spec.CompositionContributions.Grants {
				if grant.EffectBlockID == node.SourceBlockID && grant.OperationIndex == *node.OperationIndex {
					if capabilitiesByEffect[node.EffectKey] == nil {
						capabilitiesByEffect[node.EffectKey] = map[string]bool{}
					}
					capabilitiesByEffect[node.EffectKey][grant.Capability] = true
				}
			}
		}
		resolved, err = composition.ResolveWithRules(graph, expected, profile.ComponentGuInstances, ruleSet, capabilitiesByEffect)
	} else {
		if len(graph.Edges) > 0 {
			return teleportArrayEffectProfile{}, fmt.Errorf("graph ghép Trận Pháp cũ thiếu rule revision; cần tháo và tạo lại effect qua preview")
		}
		resolved, err = composition.Resolve(graph, expected, profile.ComponentGuInstances)
	}
	if err != nil {
		return teleportArrayEffectProfile{}, err
	}
	if len(profile.Effects.Blocks) != len(resolved.Nodes) || profile.Effects.SchemaVersion != effectcatalog.BlockSchemaVersion {
		return teleportArrayEffectProfile{}, fmt.Errorf("Block v2 snapshot không khớp composition graph")
	}
	rawEnvelope, err := json.Marshal(profile.Effects)
	if err != nil {
		return teleportArrayEffectProfile{}, err
	}
	canonical, issues := effectcatalog.CanonicalBlockV2Envelope(rawEnvelope, effectcatalog.Location{Source: "teleport_array", Key: "arrival_effect"})
	if len(issues) > 0 {
		return teleportArrayEffectProfile{}, &effectcatalog.ValidationError{Issues: issues}
	}
	var canonicalEnvelope teleportArrayEffectEnvelope
	if err := json.Unmarshal(canonical, &canonicalEnvelope); err != nil {
		return teleportArrayEffectProfile{}, err
	}
	for _, block := range profile.Effects.Blocks {
		if block.Trigger != gameplayeffects.TriggerOnUse || len(block.Operations) != 1 || !worldUseOperationAllowed(block.Operations[0], block.Target) {
			return teleportArrayEffectProfile{}, fmt.Errorf("Block v2 snapshot chứa operation ngoài executor Trận Pháp")
		}
	}
	if _, err := (gameplayeffects.Compiler{}).Compile(profile.Effects.Blocks); err != nil {
		return teleportArrayEffectProfile{}, err
	}
	profile.CompositionGraph = resolved
	profile.Effects = canonicalEnvelope
	return profile, nil
}

func teleportSnapshotMatchesSource(node composition.Node, sourceBlocks []gameplayeffects.Block, snapshots []gameplayeffects.Block) bool {
	if node.OperationIndex == nil || *node.OperationIndex < 0 {
		return false
	}
	var source *gameplayeffects.Block
	for index := range sourceBlocks {
		if sourceBlocks[index].ID == node.SourceBlockID {
			source = &sourceBlocks[index]
			break
		}
	}
	if source == nil || *node.OperationIndex >= len(source.Operations) {
		return false
	}
	var snapshot *gameplayeffects.Block
	for index := range snapshots {
		if snapshots[index].ID == node.BlockID {
			snapshot = &snapshots[index]
			break
		}
	}
	if snapshot == nil || len(snapshot.Operations) != 1 {
		return false
	}
	expectedOperation := source.Operations[*node.OperationIndex]
	expectedOperation.Target = nil
	if expectedOperation.Op == gameplayeffects.OpStatus {
		if expectedOperation.Turns <= 0 {
			expectedOperation.Turns = node.Rank
		}
		if expectedOperation.Value <= 0 {
			expectedOperation.Value = float64(node.Rank)
		}
	}
	expectedTarget := source.Target
	if source.Operations[*node.OperationIndex].Target != nil {
		expectedTarget = *source.Operations[*node.OperationIndex].Target
	}
	return snapshot.Trigger == source.Trigger && reflect.DeepEqual(snapshot.Target, expectedTarget) && reflect.DeepEqual(snapshot.Conditions, source.Conditions) && reflect.DeepEqual(snapshot.Operations[0], expectedOperation)
}

func applyTeleportArrayArrivalEffect(player *models.Player, raw []byte) (string, error) {
	profile, err := validateTeleportArrayEffectProfile(raw)
	if err != nil {
		return "", err
	}
	compiled, err := (gameplayeffects.Compiler{}).Compile(profile.Effects.Blocks)
	if err != nil {
		return "", err
	}
	actor := gameplayeffects.Snapshot{
		ID: player.ID.String(), HP: player.HP, MaxHP: gamefx.EffectiveMaxHP(player),
		Mana: player.CurrentMana, MaxMana: gamefx.EffectiveMaxMana(player),
	}
	result, err := compiled.Evaluate(gameplayeffects.Context{Actor: actor, Target: actor, Rank: profile.GuRank}, gameplayeffects.Event{Trigger: gameplayeffects.TriggerOnUse})
	if err != nil {
		return "", err
	}
	if len(result.Commands) == 0 {
		return "", nil
	}
	healed, restoredMana, flightMinutes, concealment, hasteMinutes, err := applyWorldUseCommands(player, result.Commands, profile.GuRank, "Trận Pháp")
	if err != nil {
		return "", err
	}
	var applied []string
	if healed > 0 {
		applied = append(applied, fmt.Sprintf("hồi %.0f HP", healed))
	}
	if restoredMana > 0 {
		applied = append(applied, fmt.Sprintf("hoàn %.0f Chân Nguyên", restoredMana))
	}
	if flightMinutes > 0 {
		applied = append(applied, fmt.Sprintf("Phi Hành %d phút", flightMinutes))
	}
	if concealment > 0 {
		applied = append(applied, fmt.Sprintf("Ẩn Nấp +%d", concealment))
	}
	if hasteMinutes > 0 {
		applied = append(applied, fmt.Sprintf("Tăng Tốc %d phút", hasteMinutes))
	}
	return strings.Join(applied, ", "), nil
}
