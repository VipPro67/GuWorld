package handlers

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/gameplay/composition"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/statusfx"
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
	SourceRevision       string   `json:"source_revision,omitempty"`
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
	SourceRevision          string                      `json:"source_revision,omitempty"`
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
	CompositionDraftID string             `json:"composition_draft_id,omitempty"`
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
	sourceRevision, err := gameplayeffects.SourceRevision(spec.Effects.GetBlocks())
	if err != nil {
		return nil, fmt.Errorf("Không thể tính revision effect nguồn của Cổ %s rank %d: %w", inst.GuID, inst.Rank, err)
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
				Key: key, SourceBlockID: block.ID, SourceRevision: sourceRevision, BlockIndex: bi, OperationIndex: oi, SourceID: inst.InstanceID, SourceName: spec.Name,
				GuID: inst.GuID, Rank: inst.Rank, Label: killerMoveOperationLabel(op), Trigger: string(block.Trigger),
				Operation: op, Target: target, Conditions: append([]gameplayeffects.Condition(nil), block.Conditions...),
				PlayerATKCoefficient: spec.PlayerATKCoefficient, CombatSupported: combatSupported,
				CompositionCapabilities: capabilities,
				TravelSupported:         outsideCombatSupported,
				UnavailableReason:       map[bool]string{true: "", false: "Chỉ dùng ngoài combat"}[combatSupported],
			})
		}
	}
	return result, nil
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
			return false
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

func killerMoveEffectKey(instanceID string, blockIndex, operationIndex int) string {
	return fmt.Sprintf("%s:%d:%d", instanceID, blockIndex, operationIndex)
}

func killerMovePreviewHash(preview killerMovePreview) string {
	preview.PreviewFingerprint = ""
	raw, _ := json.Marshal(preview)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
