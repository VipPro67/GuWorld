package handlers

import (
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	gameplayeffects "github.com/vippro67/GuWorld/backend/internal/gameplay/effects"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

type effectCompatibilityOperation struct {
	BlockID        string `json:"block_id,omitempty"`
	OperationIndex int    `json:"operation_index"`
	Operation      string `json:"operation"`
	Status         string `json:"status"`
	Reason         string `json:"reason,omitempty"`
}

type effectCompatibilityContext struct {
	ID         string                         `json:"id"`
	Label      string                         `json:"label"`
	Status     string                         `json:"status"`
	Operations []effectCompatibilityOperation `json:"operations"`
}

type effectCompatibilityPolicy struct {
	id, label string
}

var effectCompatibilityPolicies = []effectCompatibilityPolicy{
	{"TURN_BASED_COMBAT", "Combat theo lượt · Cổ"},
	{"KILLER_MOVE_COMBAT", "Combat theo lượt · Sát Chiêu"},
	{"GU_HOUSE_COMBAT", "Combat theo lượt · Tiên Cổ Ốc"},
	{"WORLD_ON_USE", "World ON_USE · dùng trực tiếp"},
	{"KILLER_MOVE_ON_USE", "World ON_USE · Sát Chiêu"},
	{"GU_HOUSE_ON_USE", "World ON_USE · Tiên Cổ Ốc"},
	{"TELEPORT_ARRAY_ARRIVAL", "World ON_USE · Trận Pháp dịch chuyển"},
	{"ITEM_CONSUME", "Vật phẩm · ON_CONSUME"},
	{"GU_CONSUME_EVENT", "Gu event · ON_CONSUME / ON_USE"},
	{"PASSIVE_DERIVED", "ALWAYS_ON · passive/derived"},
	{"ON_TICK_GROWTH", "ON_TICK · worker/resource growth"},
	{"REFINEMENT", "Refinement/deduction"},
	{"COMPOSITION_CAPABILITY", "Composition capability"},
}

func analyzeEffectCompatibility(blocks []gameplayeffects.Block) []effectCompatibilityContext {
	results := make([]effectCompatibilityContext, 0, len(effectCompatibilityPolicies))
	for _, policy := range effectCompatibilityPolicies {
		result := effectCompatibilityContext{ID: policy.id, Label: policy.label, Operations: make([]effectCompatibilityOperation, 0)}
		for _, block := range blocks {
			for operationIndex, operation := range block.Operations {
				compatibility := effectCompatibilityOperation{
					BlockID: block.ID, OperationIndex: operationIndex, Operation: string(operation.Op),
				}
				compatibility.Status, compatibility.Reason = checkEffectCompatibility(policy.id, block, operation)
				result.Operations = append(result.Operations, compatibility)
			}
		}
		result.Status = summarizeEffectCompatibility(result.Operations)
		results = append(results, result)
	}
	return results
}

func summarizeEffectCompatibility(operations []effectCompatibilityOperation) string {
	if len(operations) == 0 {
		return "unsupported"
	}
	supported, conditional, unsupported := 0, 0, 0
	for _, operation := range operations {
		switch operation.Status {
		case "supported":
			supported++
		case "conditional":
			conditional++
		default:
			unsupported++
		}
	}
	if unsupported == 0 && conditional == 0 {
		return "supported"
	}
	if supported == 0 && conditional == 0 {
		return "unsupported"
	}
	if supported == 0 {
		return "conditional"
	}
	return "partial"
}

func unsupportedEffectOperations(contexts []effectCompatibilityContext, contextID string) []effectCompatibilityOperation {
	for _, context := range contexts {
		if context.ID != contextID {
			continue
		}
		unsupported := make([]effectCompatibilityOperation, 0)
		for _, operation := range context.Operations {
			if operation.Status == "unsupported" {
				unsupported = append(unsupported, operation)
			}
		}
		return unsupported
	}
	return []effectCompatibilityOperation{}
}

func checkEffectCompatibility(contextID string, block gameplayeffects.Block, operation gameplayeffects.Operation) (string, string) {
	if reason := unsupportedConditionReason(contextID, block.Conditions); reason != "" {
		return "unsupported", reason
	}
	target := block.Target
	if operation.Target != nil {
		target = *operation.Target
	}

	switch contextID {
	case "TURN_BASED_COMBAT", "KILLER_MOVE_COMBAT":
		if block.Trigger != gameplayeffects.TriggerOnAction {
			if operation.Op == gameplayeffects.OpUnique && isNativeHookFor(operation.Code, string(block.Trigger), "combat") {
				if contextID == "KILLER_MOVE_COMBAT" && !killerMoveOperationAllowed(operation, target) {
					return "unsupported", "Native hook chưa thuộc allowlist Sát Chiêu turn-based."
				}
				return "conditional", "Native hook được chuyển tới simulator; semantics phụ thuộc handler riêng của action code."
			}
			return "unsupported", "Context này chỉ chuyển operation tổng quát của trigger ON_ACTION; trigger khác cần native combat hook cụ thể."
		}
		if operation.Op == gameplayeffects.OpUnique {
			if contextID == "KILLER_MOVE_COMBAT" && !killerMoveOperationAllowed(operation, target) {
				return "unsupported", "Native hook chưa thuộc allowlist Sát Chiêu turn-based."
			}
			if !isNativeHookFor(operation.Code, string(block.Trigger), "combat") {
				return "unsupported", "Native hook không có domain combat và trigger tương ứng."
			}
			if len(block.Conditions) > 0 {
				return "unsupported", "Combat native hook chưa nhận conditions từ Block trong đường dispatch hiện tại."
			}
			return "conditional", "Native hook được chuyển tới simulator; semantics phụ thuộc handler riêng của action code."
		}
		compiled, err := (gameplayeffects.Compiler{}).Compile([]gameplayeffects.Block{{
			ID: block.ID, Trigger: block.Trigger, Target: block.Target, Conditions: block.Conditions,
			Operations: []gameplayeffects.Operation{operation},
		}})
		if err != nil {
			return "unsupported", "Không compile được operation: " + err.Error()
		}
		compiledOperations := compiled.Operations(gameplayeffects.TriggerOnAction)
		if len(compiledOperations) != 1 {
			return "unsupported", "Không tạo được đúng một compiled operation."
		}
		actor := &models.Combatant{}
		spec := &models.GuDictionaryItem{ID: "EFFECT_PREVIEW", Name: "Effect preview", Rank: 1}
		if _, err := adaptBlockCombatOperations(actor, spec, 1, compiledOperations); err != nil {
			return "unsupported", err.Error()
		}
		if contextID == "KILLER_MOVE_COMBAT" && !killerMoveOperationAllowed(operation, target) {
			return "unsupported", "Operation chưa thuộc allowlist Sát Chiêu turn-based."
		}
		return "supported", ""

	case "GU_HOUSE_COMBAT":
		if block.Trigger != gameplayeffects.TriggerOnAction || len(block.Conditions) > 0 {
			return "unsupported", "Trận Kích cần ON_ACTION và không có condition."
		}
		if len(block.Operations) != 1 || !guHouseActionOperationAllowed(operation, target) {
			return "unsupported", "Trận Kích cần đúng một DAMAGE/HEAL/SHIELD/BARRIER FLAT có target SELF hoặc TARGET."
		}
		return "supported", ""

	case "WORLD_ON_USE", "KILLER_MOVE_ON_USE", "GU_HOUSE_ON_USE", "TELEPORT_ARRAY_ARRIVAL":
		if block.Trigger != gameplayeffects.TriggerOnUse {
			return "unsupported", "World-use executor chỉ nhận trigger ON_USE."
		}
		if !worldUseOperationAllowed(operation, target) {
			return "unsupported", "Executor hiện chỉ nhận self HEAL HP/MANA và STATUS FLYING/HASTE/STEALTH theo world limits."
		}
		if contextID == "TELEPORT_ARRAY_ARRIVAL" && len(block.Operations) != 1 {
			return "unsupported", "Teleport Array yêu cầu mỗi Block snapshot có đúng một operation."
		}
		if contextID == "KILLER_MOVE_ON_USE" && strings.EqualFold(operation.SubType, "RESOURCE") && !strings.EqualFold(operation.Resource, "MANA") {
			return "unsupported", "Sát Chiêu world executor chỉ hỗ trợ khôi phục MANA."
		}
		if contextID == "WORLD_ON_USE" {
			return "conditional", "World-use adapter dùng chung cho product producer; producer hiện hành phải chọn effect rõ ràng."
		}
		return "supported", ""

	case "ITEM_CONSUME":
		if block.Trigger != gameplayeffects.TriggerOnConsume {
			return "unsupported", "Item consume adapter chỉ phát ON_CONSUME."
		}
		if target.Selector != gameplayeffects.TargetSelf {
			return "unsupported", "Item consume adapter chỉ hỗ trợ target SELF."
		}
		if operation.Op != gameplayeffects.OpUnique || !isNativeHookFor(operation.Code, string(block.Trigger), "item_use") {
			return "unsupported", "Item adapter chỉ chạy native item-use hooks đã đăng ký."
		}
		if len(block.Conditions) > 0 {
			return "unsupported", "Item adapter hiện không evaluate Block conditions."
		}
		return "conditional", "Native item-use handler kiểm tra precondition và transaction state khi chạy."

	case "GU_CONSUME_EVENT":
		if block.Trigger != gameplayeffects.TriggerOnConsume && block.Trigger != gameplayeffects.TriggerOnUse {
			return "unsupported", "Gu event dispatcher chỉ phát ON_CONSUME hoặc ON_USE."
		}
		if operation.Op != gameplayeffects.OpUnique || !isNativeHookFor(operation.Code, string(block.Trigger), "item_use", "world") {
			return "unsupported", "Event dispatcher chỉ chạy native hook đã đăng ký cho trigger này; không Evaluate operation tổng quát."
		}
		if len(block.Conditions) > 0 {
			return "unsupported", "Event dispatcher hiện không đánh giá Block conditions cho native hook."
		}
		return "conditional", "Native action handler kiểm tra precondition và transaction state khi chạy."

	case "PASSIVE_DERIVED":
		if block.Trigger != gameplayeffects.TriggerAlwaysOn {
			return "unsupported", "Derived/passive projection hiện đọc trigger ALWAYS_ON cụ thể."
		}
		if target.Selector != gameplayeffects.TargetSelf {
			return "unsupported", "Derived/passive projection chỉ hỗ trợ target SELF."
		}
		if operation.Op == gameplayeffects.OpStatMod {
			if !gamefx.PassiveStatModSupported(operation.Stat, operation.SubType, operation.Unit, operation.Path) {
				return "unsupported", "Stat/subtype/unit chưa có passive projection adapter."
			}
			if len(block.Conditions) > 0 {
				return "unsupported", "Derived stat projections hiện không evaluate conditions."
			}
			return "conditional", "Các consumer chỉ đọc stat/subtype projection mà họ yêu cầu."
		}
		if operation.Op == gameplayeffects.OpUnique && isNativeHookFor(operation.Code, string(block.Trigger), "gamefx") {
			return "conditional", "Native derived hook chỉ có hiệu lực ở consumer chuyên biệt của code này."
		}
		return "unsupported", "Operation không có projection derived/passive hiện hành."

	case "ON_TICK_GROWTH":
		if block.Trigger != gameplayeffects.TriggerOnTick || operation.Op != gameplayeffects.OpUnique {
			return "unsupported", "Worker/projection hiện chỉ đọc native UNIQUE hook ON_TICK cụ thể."
		}
		if len(block.Conditions) > 0 {
			return "unsupported", "ON_TICK worker/projection hiện không evaluate Block conditions."
		}
		switch strings.ToUpper(operation.Code) {
		case "MANA_REGEN_TICK", "THOUGHT_REGEN_TICK", "SOUL_REGEN_TICK", "GENERATE_STONES_TICK", "GENERATE_IMMORTAL_STONES_TICK":
			return "supported", ""
		default:
			return "unsupported", "Không có consumer ON_TICK cho native code này."
		}

	case "REFINEMENT":
		if block.Trigger == gameplayeffects.TriggerAlwaysOn && operation.Op == gameplayeffects.OpStatMod && (strings.EqualFold(operation.Stat, "REFINE_SUCCESS") || strings.EqualFold(operation.Stat, "REFINE_MANA_TIER")) {
			if len(block.Conditions) > 0 {
				return "unsupported", "Refinement stat projection hiện không evaluate Block conditions."
			}
			return "conditional", "Refinement consumer chỉ đọc các stat projection được yêu cầu."
		}
		if block.Trigger == gameplayeffects.TriggerAlwaysOn && operation.Op == gameplayeffects.OpUnique && strings.EqualFold(operation.Code, "DEDUCTION_INSTANT_FINISH") {
			if len(block.Conditions) > 0 {
				return "unsupported", "Deduction consumer hiện không evaluate Block conditions."
			}
			return "conditional", "Deduction handler tự đọc params và áp xác suất theo rule hiện hành."
		}
		return "unsupported", "Không có generic REFINE trigger/executor cho operation này."

	case "COMPOSITION_CAPABILITY":
		if block.Trigger == gameplayeffects.TriggerCapability && operation.Op == gameplayeffects.OpUnique {
			if len(block.Conditions) > 0 {
				return "unsupported", "Composition capability projection hiện không evaluate conditions."
			}
			return "supported", ""
		}
		return "unsupported", "Composition rules chỉ đọc UNIQUE CAPABILITY grants đã khai báo."
	default:
		return "unsupported", "Execution context không được đăng ký."
	}
}

func unsupportedConditionReason(contextID string, conditions []gameplayeffects.Condition) string {
	if len(conditions) == 0 {
		return ""
	}
	values := gameplayeffects.ConditionValues{}
	switch contextID {
	case "TURN_BASED_COMBAT", "KILLER_MOVE_COMBAT":
		values["RANK"] = 1
		values["TURN"] = 1
		values["MASTERY"] = 0
		values["HP_PERCENT"] = 0
		values["ACTOR_HP_PERCENT"] = 0
		values["MANA_PERCENT"] = 0
		values["TARGET_HP_PERCENT"] = 0
		if contextID == "KILLER_MOVE_COMBAT" {
			values["MASTERY"] = 0
		}
	case "WORLD_ON_USE", "KILLER_MOVE_ON_USE", "GU_HOUSE_ON_USE", "TELEPORT_ARRAY_ARRIVAL":
		values["RANK"] = 1
		values["HP_PERCENT"] = 0
		values["ACTOR_HP_PERCENT"] = 0
		values["MANA_PERCENT"] = 0
		values["TARGET_HP_PERCENT"] = 0
		if contextID == "KILLER_MOVE_ON_USE" {
			values["MASTERY"] = 0
		}
	default:
		return "Conditions không được evaluate trong context này."
	}
	if _, err := gameplayeffects.ConditionsMatch(conditions, values); err != nil {
		return fmt.Sprintf("Condition không khả dụng trong context: %v", err)
	}
	return ""
}

func isNativeHookFor(code, trigger string, domains ...string) bool {
	hook, ok := effectcatalog.NativeHooks[strings.ToUpper(strings.TrimSpace(code))]
	if !ok || !hook.Triggers[strings.ToUpper(strings.TrimSpace(trigger))] {
		return false
	}
	for _, domain := range domains {
		if strings.EqualFold(hook.Domain, domain) {
			return true
		}
	}
	return false
}

func guHouseActionOperationAllowed(operation gameplayeffects.Operation, target gameplayeffects.Target) bool {
	if target.Selector != gameplayeffects.TargetSelf && target.Selector != gameplayeffects.TargetSingle {
		return false
	}
	if operation.SubType != "" && !strings.EqualFold(operation.SubType, "FLAT") {
		return false
	}
	if operation.Value <= 0 {
		return false
	}
	switch operation.Op {
	case gameplayeffects.OpDamage:
		return target.Selector == gameplayeffects.TargetSingle
	case gameplayeffects.OpHeal, gameplayeffects.OpShield, gameplayeffects.OpBarrier:
		return target.Selector == gameplayeffects.TargetSelf
	default:
		return false
	}
}
