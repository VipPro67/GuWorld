package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

const (
	killerMoveMaxSaved       = 12
	killerMoveMaxEquipped    = 4
	killerMoveMaxComponents  = 6
	killerMoveMaxEffects     = 8
	killerMoveBalanceVersion = 1
)

type killerMoveComponent struct {
	InstanceID string `json:"instance_id"`
	GuID       string `json:"gu_id"`
	Rank       int    `json:"rank"`
}

type killerMoveEffectRef struct {
	Key        string `json:"key"`
	Trigger    string `json:"trigger"`
	InstanceID string `json:"instance_id"`
	GuID       string `json:"gu_id"`
	Rank       int    `json:"rank"`
	BlockIndex int    `json:"block_index"`
	OpIndex    int    `json:"operation_index"`
}

type killerMoveEffectOption struct {
	Key        string                    `json:"key"`
	SourceID   string                    `json:"source_instance_id"`
	SourceName string                    `json:"source_name"`
	GuID       string                    `json:"gu_id"`
	Rank       int                       `json:"rank"`
	Label      string                    `json:"label"`
	Trigger    string                    `json:"trigger"`
	Operation  gameplayeffects.Operation `json:"operation"`
	Target     gameplayeffects.Target    `json:"target"`
}

type killerMoveDraftRequest struct {
	Name               string   `json:"name"`
	MainGuInstanceID   string   `json:"main_gu_instance_id" binding:"required"`
	ComponentIDs       []string `json:"component_instance_ids"`
	SelectedEffectKeys []string `json:"selected_effect_keys"`
}

type killerMovePreview struct {
	Components       []killerMoveComponent   `json:"components"`
	SelectedEffects  []killerMoveEffectRef   `json:"selected_effects"`
	CompiledEffects  []gameplayeffects.Block `json:"compiled_effects"`
	Rank             int                     `json:"rank"`
	Path             string                  `json:"path"`
	PathMasteryLevel int                     `json:"path_mastery_level"`
	PowerBudget      float64                 `json:"power_budget"`
	PowerLoad        float64                 `json:"power_load"`
	EffectScale      float64                 `json:"effect_scale"`
	Harmony          float64                 `json:"harmony"`
	Complexity       float64                 `json:"complexity"`
	SuccessChance    float64                 `json:"success_chance"`
	ManaCost         float64                 `json:"mana_cost"`
	Warnings         []string                `json:"warnings"`
}

type storedKillerMove struct {
	ID                       string                  `json:"id"`
	Name                     string                  `json:"name"`
	MainGuInstance           string                  `json:"main_gu_instance_id"`
	MainGuID                 string                  `json:"main_gu_id"`
	MainGuRank               int                     `json:"main_gu_rank"`
	Components               []killerMoveComponent   `json:"components"`
	SelectedEffects          []killerMoveEffectRef   `json:"selected_effects"`
	CompiledEffects          []gameplayeffects.Block `json:"compiled_effects"`
	Path                     string                  `json:"path"`
	CreationPathMasteryLevel int                     `json:"creation_path_mastery_level"`
	PowerBudget              float64                 `json:"power_budget"`
	PowerLoad                float64                 `json:"power_load"`
	Harmony                  float64                 `json:"harmony"`
	Complexity               float64                 `json:"complexity"`
	SuccessChance            float64                 `json:"success_chance"`
	MasteryXP                int                     `json:"mastery_xp"`
	MasteryLevel             int                     `json:"mastery_level"`
	Equipped                 bool                    `json:"is_equipped"`
	Available                bool                    `json:"available"`
	MissingGu                []string                `json:"missing_gu,omitempty"`
	BalanceVersion           int                     `json:"balance_version"`
}

type ownedKillerMoveGu struct {
	InstanceID string  `json:"instance_id"`
	GuID       string  `json:"gu_id"`
	Rank       int     `json:"rank"`
	Hunger     float64 `json:"hunger"`
	State      string  `json:"state"`
	Location   string  `json:"location"`
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
	rows, err := tx.Query(`SELECT id, gu_id, gu_rank, hunger, state, location FROM player_gu WHERE player_id=$1`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	owned := make(map[string]ownedKillerMoveGu)
	for rows.Next() {
		var gu ownedKillerMoveGu
		if err := rows.Scan(&gu.InstanceID, &gu.GuID, &gu.Rank, &gu.Hunger, &gu.State, &gu.Location); err != nil {
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
			if !killerMoveOperationAllowed(op) || (block.Trigger == gameplayeffects.TriggerOnUse && !killerMoveOutsideCombatOperationAllowed(op, target)) {
				continue
			}
			key := killerMoveEffectKey(inst.InstanceID, bi, oi)
			result = append(result, killerMoveEffectOption{
				Key: key, SourceID: inst.InstanceID, SourceName: spec.Name,
				GuID: inst.GuID, Rank: inst.Rank, Label: killerMoveOperationLabel(op), Trigger: string(block.Trigger),
				Operation: op, Target: target,
			})
		}
	}
	// Older dictionary effects can be compiled into blocks at load time. If a
	// definition has no authoring blocks, it is not offered for player mixing.
	return result, nil
}

func killerMoveEffectKey(instanceID string, blockIndex, operationIndex int) string {
	return fmt.Sprintf("%s:%d:%d", instanceID, blockIndex, operationIndex)
}

func killerMoveOperationAllowed(op gameplayeffects.Operation) bool {
	switch op.Op {
	case gameplayeffects.OpDamage, gameplayeffects.OpShield, gameplayeffects.OpHeal,
		gameplayeffects.OpDirectHP, gameplayeffects.OpStatus, gameplayeffects.OpBarrier,
		gameplayeffects.OpCleanse:
		return true
	case gameplayeffects.OpStatMod:
		stat := strings.ToUpper(op.Stat)
		return stat == "ATTACK" || stat == "DEFENSE"
	default:
		return false
	}
}

func killerMoveOutsideCombatOperationAllowed(op gameplayeffects.Operation, target gameplayeffects.Target) bool {
	return op.Op == gameplayeffects.OpHeal &&
		(op.SubType == "" || strings.EqualFold(op.SubType, "FLAT") || strings.EqualFold(op.SubType, "PERCENT")) &&
		(target.Selector == gameplayeffects.TargetSelf || target.Selector == "") &&
		(op.Target == nil || op.Target.Selector == gameplayeffects.TargetSelf || op.Target.Selector == "")
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
		if strings.EqualFold(gu.State, "DEAD") {
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
	c.JSON(http.StatusOK, gin.H{"gus": instances, "effect_options": options, "max_saved": killerMoveMaxSaved, "max_equipped": killerMoveMaxEquipped, "max_components": killerMoveMaxComponents, "max_effects": killerMoveMaxEffects})
}

func calculateKillerMoveDraft(tx *sql.Tx, playerID uuid.UUID, req killerMoveDraftRequest, masteryXP int) (killerMovePreview, ownedKillerMoveGu, error) {
	owned, err := loadOwnedKillerMoveGus(tx, playerID)
	if err != nil {
		return killerMovePreview{}, ownedKillerMoveGu{}, err
	}
	main, exists := owned[req.MainGuInstanceID]
	if !exists || strings.EqualFold(main.State, "DEAD") {
		return killerMovePreview{}, main, fmt.Errorf("Cổ chính không còn thuộc sở hữu của bạn")
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
		if strings.EqualFold(owned[id].State, "DEAD") {
			return killerMovePreview{}, main, fmt.Errorf("Cổ đã chết không thể làm thành phần sát chiêu")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	if len(ids) > killerMoveMaxComponents {
		return killerMovePreview{}, main, fmt.Errorf("Mỗi sát chiêu tối đa %d Cổ thành phần", killerMoveMaxComponents)
	}
	if len(req.SelectedEffectKeys) == 0 || len(req.SelectedEffectKeys) > killerMoveMaxEffects {
		return killerMovePreview{}, main, fmt.Errorf("Chọn từ 1 đến %d hiệu ứng", killerMoveMaxEffects)
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
	load := 0.0
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
		parts := strings.Split(key, ":")
		if len(parts) != 3 {
			return killerMovePreview{}, main, fmt.Errorf("Mã hiệu ứng không hợp lệ")
		}
		var bi, oi int
		if _, err := fmt.Sscanf(parts[1]+":"+parts[2], "%d:%d", &bi, &oi); err != nil {
			return killerMovePreview{}, main, fmt.Errorf("Mã hiệu ứng không hợp lệ")
		}
		selected = append(selected, killerMoveEffectRef{Key: key, Trigger: option.Trigger, InstanceID: option.SourceID, GuID: option.GuID, Rank: option.Rank, BlockIndex: bi, OpIndex: oi})
		blockTarget := option.Target
		operation := option.Operation
		// Keep each selected operation's target when the compiler flattens
		// multiple one-operation blocks into a single combat action.
		operation.Target = &blockTarget
		blocks = append(blocks, gameplayeffects.Block{ID: "KM_" + strings.ReplaceAll(key, ":", "_"), Trigger: gameplayeffects.Trigger(option.Trigger), Target: blockTarget, Operations: []gameplayeffects.Operation{operation}})
		load += killerMoveEffectLoad(option.Operation, blockTarget)
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
	budget := float64(main.Rank*26 + 20)
	for i := 1; i < componentCount; i++ {
		budget += 13 / math.Sqrt(float64(i))
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
		scale = budget / load
	}
	if scale < 0.15 {
		return killerMovePreview{}, main, fmt.Errorf("Công thức vượt ngân sách sức mạnh quá xa; hãy giảm số hiệu ứng hoặc chọn Cổ rank cao hơn")
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
	return killerMovePreview{Components: components, SelectedEffects: selected, CompiledEffects: blocks, Rank: main.Rank, Path: path, PathMasteryLevel: pathMasteryLevel, PowerBudget: budget, PowerLoad: load, EffectScale: scale, Harmony: harmony, Complexity: complexity, SuccessChance: success, ManaCost: manaCost, Warnings: warnings}, main, nil
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
	var count int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM player_killer_moves WHERE player_id=$1`, playerID).Scan(&count); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra giới hạn sát chiêu"})
		return
	}
	if count >= killerMoveMaxSaved {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Mỗi người chơi lưu tối đa %d sát chiêu", killerMoveMaxSaved)})
		return
	}
	preview, main, err := calculateKillerMoveDraft(tx, playerID, req, 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	components, _ := json.Marshal(preview.Components)
	selected, _ := json.Marshal(preview.SelectedEffects)
	compiled, _ := json.Marshal(preview.CompiledEffects)
	var id string
	err = tx.QueryRow(`INSERT INTO player_killer_moves(player_id,name,main_gu_instance_id,main_gu_id,main_gu_rank,components,selected_effects,compiled_effects,path,creation_path_mastery_level,power_budget,power_load,harmony,complexity,success_chance,balance_version)
	VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8::jsonb,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id`, playerID, name, main.InstanceID, main.GuID, main.Rank, components, selected, compiled, preview.Path, preview.PathMasteryLevel, preview.PowerBudget, preview.PowerLoad, preview.Harmony, preview.Complexity, preview.SuccessChance, killerMoveBalanceVersion).Scan(&id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu sát chiêu"})
		return
	}
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
	rows, err := db.DB.QueryContext(c.Request.Context(), `SELECT id::text,name,main_gu_instance_id,main_gu_id,main_gu_rank,components,selected_effects,compiled_effects,path,creation_path_mastery_level,power_budget,power_load,harmony,complexity,success_chance,mastery_xp,is_equipped,balance_version FROM player_killer_moves WHERE player_id=$1 ORDER BY created_at DESC`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải sát chiêu"})
		return
	}
	defer rows.Close()
	owned, err := db.DB.QueryContext(c.Request.Context(), `SELECT id,gu_id,gu_rank,state FROM player_gu WHERE player_id=$1`, playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác minh Cổ thành phần"})
		return
	}
	ownedSet := map[string]ownedKillerMoveGu{}
	for owned.Next() {
		var gu ownedKillerMoveGu
		if owned.Scan(&gu.InstanceID, &gu.GuID, &gu.Rank, &gu.State) == nil {
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
		var components, selected, compiled []byte
		if err := rows.Scan(&m.ID, &m.Name, &m.MainGuInstance, &m.MainGuID, &m.MainGuRank, &components, &selected, &compiled, &m.Path, &m.CreationPathMasteryLevel, &m.PowerBudget, &m.PowerLoad, &m.Harmony, &m.Complexity, &m.SuccessChance, &m.MasteryXP, &m.Equipped, &m.BalanceVersion); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc sát chiêu"})
			return
		}
		_ = json.Unmarshal(components, &m.Components)
		_ = json.Unmarshal(selected, &m.SelectedEffects)
		_ = json.Unmarshal(compiled, &m.CompiledEffects)
		m.MasteryLevel = m.MasteryXP / 100
		currentLevel := gamefx.PathMasteryLevel(masteries[gamefx.CleanPath(m.Path)])
		m.SuccessChance = killerMoveAdjustedChance(m.SuccessChance, m.CreationPathMasteryLevel, currentLevel, m.MasteryXP)
		m.Available = true
		for _, component := range m.Components {
			gu, exists := ownedSet[component.InstanceID]
			if !exists || strings.EqualFold(gu.State, "DEAD") || gu.GuID != component.GuID || gu.Rank != component.Rank {
				m.Available = false
				m.MissingGu = append(m.MissingGu, component.InstanceID)
			}
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc sát chiêu"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"killer_moves": result, "max_saved": killerMoveMaxSaved, "max_equipped": killerMoveMaxEquipped})
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
	var raw []byte
	err = tx.QueryRow(`SELECT is_equipped,components FROM player_killer_moves WHERE id=$1 AND player_id=$2 FOR UPDATE`, moveID, playerID).Scan(&equipped, &raw)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy sát chiêu"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể đọc sát chiêu"})
		return
	}
	var components []killerMoveComponent
	_ = json.Unmarshal(raw, &components)
	for _, component := range components {
		var guID, state string
		var rank int
		err = tx.QueryRow(`SELECT gu_id,gu_rank,state FROM player_gu WHERE id=$1 AND player_id=$2`, component.InstanceID, playerID).Scan(&guID, &rank, &state)
		if err != nil || strings.EqualFold(state, "DEAD") || guID != component.GuID || rank != component.Rank {
			c.JSON(http.StatusConflict, gin.H{"error": "Sát chiêu thiếu đúng Cổ thành phần còn sống"})
			return
		}
	}
	if !equipped {
		var count int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM player_killer_moves WHERE player_id=$1 AND is_equipped=true`, playerID).Scan(&count); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể kiểm tra ô trang bị"})
			return
		}
		if count >= killerMoveMaxEquipped {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Chỉ trang bị tối đa %d sát chiêu", killerMoveMaxEquipped)})
			return
		}
	}
	if _, err = tx.Exec(`UPDATE player_killer_moves SET is_equipped=$1,updated_at=NOW() WHERE id=$2 AND player_id=$3`, !equipped, moveID, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật trang bị"})
		return
	}
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
	res, err := db.DB.ExecContext(c.Request.Context(), `DELETE FROM player_killer_moves WHERE id=$1 AND player_id=$2`, c.Param("id"), playerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xóa sát chiêu"})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy sát chiêu"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Đã xóa sát chiêu"})
}

// Load equipped moves for combat construction. Components are checked against
// this player's live inventory on every build; missing definitions disable the
// move instead of silently changing its formula.
func loadEquippedKillerMoves(tx *sql.Tx, player *models.Player) ([]storedKillerMove, error) {
	if tx == nil || player == nil {
		return nil, nil
	}
	rows, err := tx.Query(`SELECT id::text,name,main_gu_instance_id,main_gu_id,main_gu_rank,components,compiled_effects,path,creation_path_mastery_level,power_budget,power_load,harmony,complexity,success_chance,mastery_xp FROM player_killer_moves WHERE player_id=$1 AND is_equipped=true`, player.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	owned := make(map[string]models.GuInstance, len(player.GuInventory))
	for _, g := range player.GuInventory {
		owned[g.InstanceID] = g
	}
	var result []storedKillerMove
	for rows.Next() {
		var m storedKillerMove
		var comps, compiled []byte
		if err := rows.Scan(&m.ID, &m.Name, &m.MainGuInstance, &m.MainGuID, &m.MainGuRank, &comps, &compiled, &m.Path, &m.CreationPathMasteryLevel, &m.PowerBudget, &m.PowerLoad, &m.Harmony, &m.Complexity, &m.SuccessChance, &m.MasteryXP); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(comps, &m.Components)
		_ = json.Unmarshal(compiled, &m.CompiledEffects)
		valid := true
		for _, component := range m.Components {
			g, ok := owned[component.InstanceID]
			if !ok || !killerMoveComponentMatches(component, g) || g.Hunger <= 0 {
				valid = false
				break
			}
		}
		hasCombatEffect := false
		for _, block := range m.CompiledEffects {
			if block.Trigger == gameplayeffects.TriggerOnAction && len(block.Operations) > 0 {
				hasCombatEffect = true
				break
			}
		}
		if valid && hasCombatEffect {
			result = append(result, m)
		}
	}
	return result, rows.Err()
}

// ComponentInstanceIDs exposes the Gu IDs whose fullness is consumed by a
// killer move action.
func (m storedKillerMove) ComponentInstanceIDs() []string {
	result := make([]string, 0, len(m.Components))
	for _, c := range m.Components {
		result = append(result, c.InstanceID)
	}
	return result
}

func killerMoveComponentMatches(component killerMoveComponent, gu models.GuInstance) bool {
	return gu.InstanceID == component.InstanceID && gu.GuID == component.GuID && gu.Rank == component.Rank && !strings.EqualFold(gu.Status, "DEAD")
}

// UseKillerMove executes the intentionally narrow out-of-combat subset: ON_USE
// self-healing effects. Other selected effects remain combat-only.
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
	if player.ActiveCombat != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Không thể dùng sát chiêu ngoài combat khi đang chiến đấu"})
		return
	}
	var move storedKillerMove
	var componentsRaw, effectsRaw []byte
	err = tx.QueryRow(`SELECT id::text,name,main_gu_instance_id,main_gu_id,main_gu_rank,components,compiled_effects,path,creation_path_mastery_level,success_chance,mastery_xp FROM player_killer_moves WHERE id=$1 AND player_id=$2 AND is_equipped=true FOR UPDATE`, c.Param("id"), playerID).Scan(&move.ID, &move.Name, &move.MainGuInstance, &move.MainGuID, &move.MainGuRank, &componentsRaw, &effectsRaw, &move.Path, &move.CreationPathMasteryLevel, &move.SuccessChance, &move.MasteryXP)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sát chiêu chưa được trang bị hoặc không tồn tại"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải sát chiêu"})
		return
	}
	_ = json.Unmarshal(componentsRaw, &move.Components)
	_ = json.Unmarshal(effectsRaw, &move.CompiledEffects)
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
		if gu.Hunger <= 0 {
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
	canUse := false
	for _, block := range move.CompiledEffects {
		if block.Trigger == gameplayeffects.TriggerOnUse && len(block.Operations) > 0 {
			canUse = true
			break
		}
	}
	if !canUse {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sát chiêu này chỉ có thể dùng trong combat"})
		return
	}
	cost := gamefx.EnergyCost(&player, &mainSpec)
	if player.CurrentMana < cost {
		c.JSON(http.StatusBadRequest, gin.H{"error": insufficientEnergyMessage(&player)})
		return
	}
	player.CurrentMana -= cost
	currentLevel := gamefx.PathMasteryLevel(player.PathMasteries[gamefx.CleanPath(move.Path)])
	chance := killerMoveAdjustedChance(move.SuccessChance, move.CreationPathMasteryLevel, currentLevel, move.MasteryXP)
	roll := rand.Float64()
	success := roll < chance
	message := ""
	if success {
		healed := 0.0
		for _, block := range move.CompiledEffects {
			if block.Trigger != gameplayeffects.TriggerOnUse {
				continue
			}
			for _, op := range block.Operations {
				if op.Op != gameplayeffects.OpHeal {
					continue
				}
				amount := op.Value
				if strings.EqualFold(op.SubType, "PERCENT") {
					if amount > 1 {
						amount /= 100
					}
					amount *= gamefx.EffectiveMaxHP(&player)
				}
				if amount > 0 {
					before := player.HP
					player.HP = math.Min(gamefx.EffectiveMaxHP(&player), player.HP+amount)
					healed += player.HP - before
				}
			}
		}
		if healed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Sát chiêu không có effect hồi phục hợp lệ để dùng ngoài combat"})
			return
		}
		message = fmt.Sprintf("Sát chiêu [%s] thành công, hồi %.0f HP.", move.Name, healed)
		if _, err := tx.Exec(`UPDATE player_killer_moves SET mastery_xp=LEAST(5000,mastery_xp+1),updated_at=NOW() WHERE id=$1 AND player_id=$2`, move.ID, playerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể ghi nhận thành thạo sát chiêu"})
			return
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
	if _, err := tx.Exec(`UPDATE players SET hp=$1,current_mana=$2 WHERE id=$3`, player.HP, player.CurrentMana, playerID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu trạng thái nhân vật"})
		return
	}
	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể hoàn tất sát chiêu"})
		return
	}
	respondActionWithInventory(c, http.StatusOK, player, gin.H{"message": message, "killer_move_id": move.ID, "success": success, "success_chance": chance})
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
