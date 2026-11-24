package effectcatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
)

const BlockSchemaVersion = 2

type Location struct {
	Source string `json:"source,omitempty"`
	Row    int    `json:"row,omitempty"`
	Key    string `json:"key,omitempty"`
}

type ValidationIssue struct {
	Source         string `json:"source,omitempty"`
	Row            int    `json:"row,omitempty"`
	Key            string `json:"key,omitempty"`
	BlockID        string `json:"block_id,omitempty"`
	OperationIndex *int   `json:"operation_index,omitempty"`
	Path           string `json:"path"`
	Code           string `json:"error_code"`
	Message        string `json:"message"`
}

type ValidationError struct {
	Issues []ValidationIssue `json:"issues"`
}

// ValidatePersistentStatusKeys validates APPLY_STACKING_STATUS references
// against the separate registry used by player-persisted statuses.
func ValidatePersistentStatusKeys(raw []byte, registered map[string]bool, loc Location) []ValidationIssue {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || registered == nil {
		return nil
	}
	type operation struct {
		Op        string `json:"op"`
		StatusKey string `json:"status_key"`
	}
	type block struct {
		ID         string      `json:"id"`
		Operations []operation `json:"operations"`
	}
	var blocks []block
	if trimmed[0] == '[' {
		if json.Unmarshal(trimmed, &blocks) != nil {
			return nil
		}
	} else if trimmed[0] == '{' {
		var envelope struct {
			Blocks []block `json:"blocks"`
		}
		if json.Unmarshal(trimmed, &envelope) != nil {
			return nil
		}
		blocks = envelope.Blocks
	} else {
		return nil
	}
	var issues []ValidationIssue
	for blockIndex, authoredBlock := range blocks {
		for operationIndex, op := range authoredBlock.Operations {
			if strings.ToUpper(strings.TrimSpace(op.Op)) != "APPLY_STACKING_STATUS" || op.StatusKey == "" || registered[op.StatusKey] {
				continue
			}
			index := operationIndex
			issues = append(issues, ValidationIssue{
				Source: loc.Source, Row: loc.Row, Key: loc.Key, BlockID: authoredBlock.ID, OperationIndex: &index,
				Path: fmt.Sprintf("$.blocks[%d].operations[%d].status_key", blockIndex, operationIndex),
				Code: "unknown_status_key", Message: fmt.Sprintf("player-persistent status %q is not registered", op.StatusKey),
			})
		}
	}
	return issues
}

func (e *ValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "invalid effect definition"
	}
	return e.Issues[0].Path + ": " + e.Issues[0].Message
}

var canonicalID = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// ValidateBlockV2 validates the block-only authoring contract. It never
// normalizes legacy input: accepting it here would let an author believe an
// effect is executable when the block compiler cannot see it.
func ValidateBlockV2(raw []byte, loc Location) []ValidationIssue {
	v := blockValidator{loc: loc}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return v.issues
	}

	var blocks []map[string]json.RawMessage

	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &blocks); err != nil {
			v.add("$", "invalid_type", "blocks must be an array: "+err.Error(), "", nil)
			return v.issues
		}
	} else if trimmed[0] == '{' {
		var root map[string]json.RawMessage
		if err := decodeObject(raw, &root); err != nil {
			v.add("$", "invalid_json", err.Error(), "", nil)
			return v.issues
		}
		v.fields(root, "$", "schema_version", "blocks")
		if b, ok := root["blocks"]; !ok {
			v.add("$.blocks", "required_field", "blocks is required", "", nil)
		} else if json.Unmarshal(b, &blocks) != nil || firstNonSpace(b) != '[' {
			v.add("$.blocks", "invalid_type", "blocks must be an array", "", nil)
		}
	} else {
		v.add("$", "invalid_json", "expected JSON array or object", "", nil)
		return v.issues
	}

	seen := map[string]bool{}
	for i, block := range blocks {
		v.block(block, i, seen)
	}
	sort.SliceStable(v.issues, func(i, j int) bool { return v.issues[i].Path < v.issues[j].Path })
	return v.issues
}

func ValidateBlockV2Definition(raw []byte, loc Location) error {
	issues := ValidateBlockV2(raw, loc)
	if len(issues) > 0 {
		return &ValidationError{Issues: issues}
	}
	return nil
}

// CanonicalBlockV2Envelope validates a Block v2 array or envelope and returns
// the canonical versioned envelope. Flat arrays remain accepted as a legacy
// storage shape at read/import seams, but new writes always carry the schema
// version. Legacy namespace objects are deliberately rejected here.
func CanonicalBlockV2Envelope(raw []byte, loc Location) ([]byte, []ValidationIssue) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		trimmed = []byte("[]")
	}

	var blocks json.RawMessage
	switch trimmed[0] {
	case '[':
		blocks = append(json.RawMessage(nil), trimmed...)
	case '{':
		var root map[string]json.RawMessage
		if err := decodeObject(trimmed, &root); err != nil {
			return nil, []ValidationIssue{{Source: loc.Source, Row: loc.Row, Key: loc.Key, Path: "$", Code: "invalid_json", Message: err.Error()}}
		}
		for key := range root {
			if key != "schema_version" && key != "blocks" {
				return nil, []ValidationIssue{{Source: loc.Source, Row: loc.Row, Key: loc.Key, Path: "$." + key, Code: "unknown_field", Message: "Block v2 envelope cannot contain sibling field " + key}}
			}
		}
		var version int
		versionRaw, ok := root["schema_version"]
		if ok && (json.Unmarshal(versionRaw, &version) != nil || version != BlockSchemaVersion) {
			return nil, []ValidationIssue{{Source: loc.Source, Row: loc.Row, Key: loc.Key, Path: "$.schema_version", Code: "unsupported_schema_version", Message: fmt.Sprintf("schema_version must be %d", BlockSchemaVersion)}}
		}
		if value, ok := root["blocks"]; ok {
			blocks = append(json.RawMessage(nil), value...)
		}
	default:
		return nil, []ValidationIssue{{Source: loc.Source, Row: loc.Row, Key: loc.Key, Path: "$", Code: "invalid_root", Message: "expected a Block v2 array or envelope"}}
	}

	if len(blocks) == 0 {
		return nil, []ValidationIssue{{Source: loc.Source, Row: loc.Row, Key: loc.Key, Path: "$.blocks", Code: "required_field", Message: "blocks is required"}}
	}
	candidate, err := json.Marshal(struct {
		SchemaVersion int             `json:"schema_version"`
		Blocks        json.RawMessage `json:"blocks"`
	}{SchemaVersion: BlockSchemaVersion, Blocks: blocks})
	if err != nil {
		return nil, []ValidationIssue{{Source: loc.Source, Row: loc.Row, Key: loc.Key, Path: "$", Code: "encode_failed", Message: err.Error()}}
	}
	if issues := ValidateBlockV2(candidate, loc); len(issues) > 0 {
		return nil, issues
	}
	return candidate, nil
}

type blockValidator struct {
	loc    Location
	issues []ValidationIssue
}

func (v *blockValidator) add(path, code, msg, block string, op *int) {
	v.issues = append(v.issues, ValidationIssue{Source: v.loc.Source, Row: v.loc.Row, Key: v.loc.Key, BlockID: block, OperationIndex: op, Path: path, Code: code, Message: msg})
}
func (v *blockValidator) fields(m map[string]json.RawMessage, path string, allowed ...string) {
	a := map[string]bool{}
	for _, s := range allowed {
		a[s] = true
	}
	for k := range m {
		if !a[k] {
			v.add(path+"."+k, "unknown_field", "unknown field "+k, "", nil)
		}
	}
}

func (v *blockValidator) block(b map[string]json.RawMessage, i int, seen map[string]bool) {
	base := fmt.Sprintf("$.blocks[%d]", i)
	v.fields(b, base, "id", "trigger", "target", "conditions", "operations")
	id := v.requiredCanonical(b, "id", base, "", nil)
	if id != "" {
		if seen[id] {
			v.add(base+".id", "duplicate_block_id", "block id must be unique", id, nil)
		}
		seen[id] = true
	}
	trigger := v.requiredCanonical(b, "trigger", base, id, nil)
	if trigger != "" && !TriggerCatalog[trigger] {
		v.add(base+".trigger", "unknown_trigger", "trigger is not registered", id, nil)
	}
	target := v.target(b["target"], base+".target", id, nil, true)
	if trigger != "" && target != "" && !triggerAllowsTarget(trigger, target) {
		v.add(base+".target.selector", "incompatible_trigger_target", "target is incompatible with trigger", id, nil)
	}
	hasConditions := false
	if raw, ok := b["conditions"]; ok {
		var cs []map[string]json.RawMessage
		if json.Unmarshal(raw, &cs) != nil {
			v.add(base+".conditions", "invalid_type", "conditions must be an array", id, nil)
		} else {
			hasConditions = len(cs) > 0
			for n, c := range cs {
				v.condition(c, fmt.Sprintf("%s.conditions[%d]", base, n), id)
			}
		}
	}
	var ops []map[string]json.RawMessage
	raw, ok := b["operations"]
	if !ok {
		v.add(base+".operations", "required_field", "operations is required", id, nil)
		return
	}
	if json.Unmarshal(raw, &ops) != nil || firstNonSpace(raw) != '[' {
		v.add(base+".operations", "invalid_type", "operations must be an array", id, nil)
		return
	}
	if len(ops) == 0 {
		v.add(base+".operations", "empty_operations", "at least one operation is required", id, nil)
	}
	for n, op := range ops {
		v.operation(op, fmt.Sprintf("%s.operations[%d]", base, n), id, n, trigger, target, hasConditions)
	}
}

func (v *blockValidator) condition(c map[string]json.RawMessage, path, id string) {
	v.fields(c, path, "field", "operator", "value")
	f := v.requiredCanonical(c, "field", path, id, nil)
	o := v.requiredCanonical(c, "operator", path, id, nil)
	if !ConditionFields[f] {
		v.add(path+".field", "unknown_condition_field", "condition field is not registered", id, nil)
	}
	if !ConditionOperators[o] {
		v.add(path+".operator", "unknown_condition_operator", "condition operator is not registered", id, nil)
	}
	if r, ok := c["value"]; !ok {
		v.add(path+".value", "required_field", "value is required", id, nil)
	} else {
		var value any
		if json.Unmarshal(r, &value) != nil {
			v.add(path+".value", "invalid_type", "value must be a number or string", id, nil)
			return
		}
		if _, err := NormalizeConditionValue(f, o, value); err != nil {
			v.add(path+".value", "invalid_condition_value", err.Error(), id, nil)
		}
	}
}

func (v *blockValidator) operation(op map[string]json.RawMessage, path, id string, index int, trigger, blockTarget string, hasConditions bool) {
	p := index
	typ := v.requiredCanonical(op, "op", path, id, &p)
	def, ok := OperationCatalog[typ]
	if !ok {
		if typ != "" {
			v.add(path+".op", "unknown_operation", "operation is not registered", id, &p)
		}
		return
	}
	for field := range op {
		if !def.Allowed[field] {
			v.add(path+"."+field, "unknown_field", "field is not allowed for "+typ, id, &p)
		}
	}
	if !def.Triggers[trigger] {
		v.add(path+".op", "incompatible_trigger_operation", "operation is incompatible with trigger", id, &p)
	}
	effectiveTarget := blockTarget
	if raw, exists := op["target"]; exists {
		sel := v.target(raw, path+".target", id, &p, true)
		if sel != "" {
			effectiveTarget = sel
		}
		if sel != "" && !def.Targets[sel] {
			v.add(path+".target.selector", "invalid_operation_target", "operation target override is not allowed", id, &p)
		}
	} else if blockTarget != "" && !def.Targets[blockTarget] {
		v.add(path+".target", "invalid_operation_target", "block target is not allowed for operation", id, &p)
	}
	sub := optionalCanonical(op, "sub_type")
	if raw, exists := op["sub_type"]; exists {
		var authored string
		if json.Unmarshal(raw, &authored) == nil && authored != sub {
			v.add(path+".sub_type", "noncanonical_value", "sub_type must use canonical casing", id, &p)
		}
	}
	if len(def.SubTypes) > 0 {
		if sub == "" {
			v.add(path+".sub_type", "required_field", "sub_type is required", id, &p)
		} else if !def.SubTypes[sub] {
			v.add(path+".sub_type", "unknown_sub_type", "sub_type is not registered for operation", id, &p)
		}
	}
	if err := ValidateOperationCombination(typ, sub, trigger, effectiveTarget); err != nil {
		v.add(path+".op", "invalid_operation_combination", err.Error(), id, &p)
	}
	if typ == "STAT_MOD" && sub == "ADD_COUNTER_DELTA" {
		var stat, key string
		var perStack float64
		if json.Unmarshal(op["stat"], &stat) != nil || json.Unmarshal(op["key"], &key) != nil || json.Unmarshal(op["per_stack"], &perStack) != nil {
			v.add(path, "invalid_counter_stat_modifier", "STAT_MOD ADD_COUNTER_DELTA requires stat, key, and numeric per_stack", id, &p)
		} else if err := ValidateCounterStatModifier(stat, key, perStack); err != nil {
			v.add(path, "invalid_counter_stat_modifier", err.Error(), id, &p)
		}
	}
	if typ == "THRESHOLD_RECOVERY" {
		var hpThreshold, healPercent float64
		var maxUses int
		var cleanseDebuffs bool
		var logAction, message string
		if json.Unmarshal(op["hp_threshold"], &hpThreshold) != nil || json.Unmarshal(op["heal_percent"], &healPercent) != nil ||
			json.Unmarshal(op["max_uses"], &maxUses) != nil || json.Unmarshal(op["cleanse_debuffs"], &cleanseDebuffs) != nil ||
			json.Unmarshal(op["log_action"], &logAction) != nil || json.Unmarshal(op["message"], &message) != nil {
			v.add(path, "invalid_threshold_recovery", "THRESHOLD_RECOVERY requires numeric thresholds, max_uses, cleanse_debuffs, and log fields", id, &p)
		} else if err := ValidateThresholdRecovery(hpThreshold, healPercent, maxUses, cleanseDebuffs, logAction, message); err != nil {
			v.add(path, "invalid_threshold_recovery", err.Error(), id, &p)
		}
	}
	if ((typ == "DAMAGE_MODIFIER" && sub == "MINION_COUNT") || typ == "MITIGATE_DAMAGE" || typ == "DETONATE_STATUS" || typ == "HP_COST" || typ == "DEDUCTION_FINISH_CHANCE" || typ == "APPLY_STACKING_STATUS" || typ == "COUNTER_ADD" || typ == "THRESHOLD_CHECK" || typ == "REVOKE_STATUS" || typ == "SUMMON" || typ == "SLOT_INTERACTION" || (typ == "DISPLACE" && trigger == "ON_BATTLE_START") || (typ == "TRANSFER_RESOURCE" && sub == "PERCENT_OF_MAX_HP")) && hasConditions {
		v.add(path, "unsupported_conditions", typ+" does not support block conditions", id, &p)
	}
	if typ == "DAMAGE_MODIFIER" && trigger == "COMBAT_PASSIVE" && hasConditions {
		if _, hasPhase := op["phase"]; !hasPhase {
			v.add(path+".phase", "required_passive_phase", "conditional COMBAT_PASSIVE DAMAGE_MODIFIER requires phase LATE_RIDER", id, &p)
		}
	}
	if typ == "CLEANSE" && sub == "STATUS" && trigger == "COMBAT_PASSIVE" {
		if _, hasPhase := op["phase"]; !hasPhase {
			v.add(path+".phase", "required_passive_phase", "COMBAT_PASSIVE CLEANSE STATUS requires phase LATE_RIDER", id, &p)
		}
	}
	if typ == "HIT_RULE" {
		params, _ := op["params"]
		if err := ValidateHitRuleParams(sub, params); err != nil {
			v.add(path+".params", "invalid_hit_rule_params", err.Error(), id, &p)
		}
	}
	if typ == "COUNTER_ADD" {
		var key string
		var delta, minimum, maximum int
		_ = json.Unmarshal(op["key"], &key)
		_ = json.Unmarshal(op["delta"], &delta)
		_ = json.Unmarshal(op["min"], &minimum)
		_ = json.Unmarshal(op["max"], &maximum)
		if err := ValidateCounterAdd(key, delta, minimum, maximum); err != nil {
			v.add(path, "invalid_counter_add", err.Error(), id, &p)
		}
	}
	if typ == "APPLY_STACKING_STATUS" {
		var statusKey, dispelRule string
		var maxStacks int
		var durationHours float64
		var pvpOnly bool
		var tiers []StackingStatusTierValue
		typed := true
		for field, dst := range map[string]any{
			"status_key": &statusKey, "max_stacks": &maxStacks, "duration_hours": &durationHours,
			"dispel_rule": &dispelRule, "pvp_only": &pvpOnly, "tier_scaling": &tiers,
		} {
			raw, exists := op[field]
			if !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, dst) != nil {
				typed = false
				v.add(path+"."+field, "required_field", field+" is required with the correct type for APPLY_STACKING_STATUS", id, &p)
			}
		}
		if typed {
			if err := ValidateStackingStatus(statusKey, maxStacks, durationHours, dispelRule, pvpOnly, tiers); err != nil {
				v.add(path, "invalid_stacking_status", err.Error(), id, &p)
			}
		}
	}
	if typ == "MODIFY_APTITUDE" {
		var value float64
		if json.Unmarshal(op["value"], &value) != nil {
			v.add(path+".value", "invalid_aptitude_value", "value must be an integer between 1 and 99", id, &p)
		} else if err := ValidateModifyAptitude(sub, value); err != nil {
			v.add(path+".value", "invalid_aptitude_value", err.Error(), id, &p)
		}
	}
	if raw, exists := op["rank_suppression"]; exists {
		var enabled bool
		if json.Unmarshal(raw, &enabled) != nil {
			v.add(path+".rank_suppression", "invalid_type", "rank_suppression must be a boolean", id, &p)
		} else if typ != "DIRECT_HP" || sub != "SHIELD_STRIP" {
			v.add(path+".rank_suppression", "invalid_rank_suppression", "rank_suppression is only valid for DIRECT_HP SHIELD_STRIP", id, &p)
		}
	}
	if typ == "TURN_ORDER_RULE" {
		params, _ := op["params"]
		if err := ValidateTurnOrderRuleParams(sub, params); err != nil {
			v.add(path+".params", "invalid_turn_order_rule_params", err.Error(), id, &p)
		}
	}
	if typ == "DEDUCTION_FINISH_CHANCE" {
		if params, exists := op["params"]; exists {
			if err := ValidateDeductionFinishChanceParams(params); err != nil {
				v.add(path+".params", "invalid_deduction_finish_chance", err.Error(), id, &p)
			}
		}
	}
	if typ == "DAMAGE_MODIFIER" {
		var base, slope, perTargetDebuff, perMinion, perStack float64
		var counterKey string
		baseOK := json.Unmarshal(op["value"], &base) == nil
		coefficientField := "per_missing_hp"
		if sub == "TARGET_DEBUFF_COUNT" {
			coefficientField = "per_target_debuff"
		} else if sub == "MINION_COUNT" {
			coefficientField = "per_minion"
		} else if sub == "COUNTER_COUNT" {
			coefficientField = "per_stack"
		}
		slopeRaw, hasSlope := op[coefficientField]
		slopeOK := sub == "TARGET_HAS_STATUS" || hasSlope && json.Unmarshal(slopeRaw, &slope) == nil
		if sub == "TARGET_DEBUFF_COUNT" {
			perTargetDebuff = slope
			slope = 0
		} else if sub == "MINION_COUNT" {
			perMinion = slope
			slope = 0
		} else if sub == "COUNTER_COUNT" {
			perStack = slope
			slope = 0
		}
		var statusID, logAction, message string
		_ = json.Unmarshal(op["key"], &counterKey)
		_ = json.Unmarshal(op["status_id"], &statusID)
		_ = json.Unmarshal(op["log_action"], &logAction)
		_ = json.Unmarshal(op["message"], &message)
		if !baseOK || !slopeOK {
			v.add(path, "invalid_damage_modifier", "value and subtype coefficient must be numbers", id, &p)
		} else if err := ValidateDamageModifier(sub, base, slope, perTargetDebuff, perMinion, perStack, counterKey, statusID, logAction, message); err != nil {
			v.add(path, "invalid_damage_modifier", err.Error(), id, &p)
		}
	}
	if typ == "HEAL" && sub == "COUNTER_PERCENT" {
		var key string
		var perCounter float64
		valueOK := json.Unmarshal(op["value"], &perCounter) == nil
		_ = json.Unmarshal(op["key"], &key)
		if !valueOK {
			v.add(path+".value", "invalid_counter_heal", "HEAL COUNTER_PERCENT value must be numeric", id, &p)
		} else if err := ValidateCounterHeal(key, perCounter); err != nil {
			v.add(path, "invalid_counter_heal", err.Error(), id, &p)
		}
		var phase string
		_ = json.Unmarshal(op["phase"], &phase)
		if phase != "FINAL_RIDER" || trigger != "COMBAT_PASSIVE" {
			v.add(path+".phase", "invalid_counter_heal_phase", "HEAL COUNTER_PERCENT requires phase FINAL_RIDER on COMBAT_PASSIVE", id, &p)
		}
	}
	if typ == "DAMAGE_MODIFIER" && sub == "COUNTER_COUNT" {
		if trigger != "COMBAT_PASSIVE" {
			v.add(path+".trigger", "invalid_counter_modifier_trigger", "DAMAGE_MODIFIER COUNTER_COUNT requires COMBAT_PASSIVE", id, &p)
		}
		var phase string
		_ = json.Unmarshal(op["phase"], &phase)
		if phase != "FINAL_RIDER" {
			v.add(path+".phase", "invalid_counter_modifier_phase", "DAMAGE_MODIFIER COUNTER_COUNT requires FINAL_RIDER", id, &p)
		}
	}
	if typ == "TARGET_DAMAGE_MODIFIER" {
		var reduction, minimum float64
		var logAction, message string
		reductionOK := json.Unmarshal(op["value"], &reduction) == nil
		minimumOK := json.Unmarshal(op["min_multiplier"], &minimum) == nil
		_ = json.Unmarshal(op["log_action"], &logAction)
		_ = json.Unmarshal(op["message"], &message)
		if !reductionOK || !minimumOK {
			v.add(path, "invalid_target_damage_modifier", "value and min_multiplier must be numbers", id, &p)
		} else if err := ValidateTargetDamageModifier(sub, reduction, minimum, logAction, message); err != nil {
			v.add(path, "invalid_target_damage_modifier", err.Error(), id, &p)
		}
	}
	if typ == "BONUS_STRIKE" {
		var chance, value, minDamage float64
		var logAction, message string
		_ = json.Unmarshal(op["log_action"], &logAction)
		_ = json.Unmarshal(op["message"], &message)
		if json.Unmarshal(op["chance"], &chance) != nil || json.Unmarshal(op["value"], &value) != nil || json.Unmarshal(op["min_damage"], &minDamage) != nil {
			v.add(path, "invalid_bonus_strike", "BONUS_STRIKE requires numeric chance, value, and min_damage", id, &p)
		} else if err := ValidateBonusStrike(chance, value, minDamage, logAction, message); err != nil {
			v.add(path, "invalid_bonus_strike", err.Error(), id, &p)
		}
	}
	if typ == "CHAIN_DAMAGE" {
		var value, minDamage float64
		var targetCount int
		var logAction, message string
		_ = json.Unmarshal(op["target_count"], &targetCount)
		_ = json.Unmarshal(op["log_action"], &logAction)
		_ = json.Unmarshal(op["message"], &message)
		if json.Unmarshal(op["value"], &value) != nil || json.Unmarshal(op["min_damage"], &minDamage) != nil {
			v.add(path, "invalid_chain_damage", "CHAIN_DAMAGE requires numeric value and min_damage", id, &p)
		} else if err := ValidateChainDamage(value, minDamage, targetCount, logAction, message); err != nil {
			v.add(path, "invalid_chain_damage", err.Error(), id, &p)
		}
	}
	if typ == "DEFENSE_PENETRATION" {
		var value float64
		if json.Unmarshal(op["value"], &value) != nil {
			v.add(path+".value", "invalid_defense_penetration", "value must be a number", id, &p)
		} else if err := ValidateDefensePenetration(value); err != nil {
			v.add(path+".value", "invalid_defense_penetration", err.Error(), id, &p)
		}
	}
	if typ == "MITIGATE_DAMAGE" {
		var value float64
		var logAction, message string
		valueOK := json.Unmarshal(op["value"], &value) == nil
		_ = json.Unmarshal(op["log_action"], &logAction)
		_ = json.Unmarshal(op["message"], &message)
		if !valueOK {
			v.add(path, "invalid_mitigate_damage", "value must be a number", id, &p)
		} else if err := ValidateMitigateDamage(sub, value, logAction, message); err != nil {
			v.add(path, "invalid_mitigate_damage", err.Error(), id, &p)
		}
	}
	if typ == "DETONATE_STATUS" {
		var statusID, logAction, message string
		var perStack, perStrength, maxHPPercent, minDamage float64
		statusOK := json.Unmarshal(op["status_id"], &statusID) == nil
		perStackOK := json.Unmarshal(op["per_stack"], &perStack) == nil
		perStrengthOK := json.Unmarshal(op["per_strength"], &perStrength) == nil
		maxHPOK := json.Unmarshal(op["max_hp_percent"], &maxHPPercent) == nil
		minDamageOK := json.Unmarshal(op["min_damage"], &minDamage) == nil
		var phase string
		phaseOK := json.Unmarshal(op["phase"], &phase) == nil
		_ = json.Unmarshal(op["log_action"], &logAction)
		_ = json.Unmarshal(op["message"], &message)
		if !statusOK || !perStackOK || !perStrengthOK || !maxHPOK || !minDamageOK || !phaseOK {
			v.add(path, "invalid_status_detonation", "status_id, formula values and phase must have valid types", id, &p)
		} else if err := ValidateStatusDetonation(statusID, perStack, perStrength, maxHPPercent, minDamage, phase); err != nil {
			v.add(path, "invalid_status_detonation", err.Error(), id, &p)
		} else if err := ValidateCombatLogAction(logAction); err != nil {
			v.add(path+".log_action", "invalid_combat_log_action", err.Error(), id, &p)
		} else if err := ValidateNotifyMessage(message); err != nil {
			v.add(path+".message", "invalid_message", err.Error(), id, &p)
		}
	}
	if typ == "HP_COST" {
		var subType, logAction, message string
		var percent, minCost, minRemaining float64
		subTypeOK := json.Unmarshal(op["sub_type"], &subType) == nil
		percentOK := json.Unmarshal(op["value"], &percent) == nil
		minCostOK := json.Unmarshal(op["min_cost"], &minCost) == nil
		minRemainingOK := json.Unmarshal(op["min_remaining"], &minRemaining) == nil
		_ = json.Unmarshal(op["log_action"], &logAction)
		_ = json.Unmarshal(op["message"], &message)
		if !subTypeOK || !percentOK || !minCostOK || !minRemainingOK {
			v.add(path, "invalid_hp_cost", "HP_COST subtype, value, min_cost and min_remaining must have valid types", id, &p)
		} else if err := ValidateHPCost(strings.ToUpper(subType), percent, minCost, minRemaining, logAction, message); err != nil {
			v.add(path, "invalid_hp_cost", err.Error(), id, &p)
		}
	}
	if typ == "DISPEL" {
		var maxCount float64
		var logAction, message string
		_ = json.Unmarshal(op["log_action"], &logAction)
		_ = json.Unmarshal(op["message"], &message)
		rawCount, hasCount := op["value"]
		if hasCount && json.Unmarshal(rawCount, &maxCount) != nil {
			v.add(path+".value", "invalid_dispel_count", "value must be an integer between 1 and 10", id, &p)
		} else if !hasCount && sub == "BUFFS" {
			v.add(path+".value", "missing_dispel_count", "DISPEL BUFFS requires a value between 1 and 10", id, &p)
		} else if err := ValidateDispel(sub, maxCount, logAction, message); err != nil {
			v.add(path, "invalid_dispel", err.Error(), id, &p)
		}
	}
	if raw, exists := op["application_mode"]; exists {
		var mode string
		if json.Unmarshal(raw, &mode) != nil {
			v.add(path+".application_mode", "invalid_type", "application_mode must be a string", id, &p)
		} else if typ != "STATUS" || mode != "DIRECT" {
			v.add(path+".application_mode", "invalid_status_application_mode", "only STATUS application_mode DIRECT is supported", id, &p)
		}
	}
	if raw, exists := op["phase"]; exists {
		var phase string
		if json.Unmarshal(raw, &phase) != nil {
			v.add(path+".phase", "invalid_type", "phase must be a string", id, &p)
		} else if typ == "DAMAGE" && (phase != "PRE_CHAIN" || trigger != "ON_ACTION" || sub != "MULTIPLIER") {
			v.add(path+".phase", "invalid_damage_phase", "DAMAGE phase PRE_CHAIN requires ON_ACTION MULTIPLIER", id, &p)
		} else if typ == "STATUS" && phase != "PRE_DAMAGE" && phase != "PRE_CHAIN" {
			v.add(path+".phase", "invalid_status_phase", "STATUS phase must be PRE_DAMAGE or PRE_CHAIN", id, &p)
		} else if typ == "DETONATE_STATUS" && phase != "EARLY_RIDER" && phase != "LATE_RIDER" {
			v.add(path+".phase", "invalid_detonation_phase", "DETONATE_STATUS phase must be EARLY_RIDER or LATE_RIDER", id, &p)
		} else if typ == "DAMAGE_MODIFIER" && ((phase != "EARLY_RIDER" && phase != "LATE_RIDER" && !(phase == "FINAL_RIDER" && sub == "COUNTER_COUNT")) || trigger != "COMBAT_PASSIVE") {
			v.add(path+".phase", "invalid_passive_modifier_phase", "DAMAGE_MODIFIER phase must be EARLY_RIDER or LATE_RIDER, or FINAL_RIDER for COUNTER_COUNT, on COMBAT_PASSIVE", id, &p)
		} else if typ == "HEAL" && (phase != "FINAL_RIDER" || trigger != "COMBAT_PASSIVE" || sub != "COUNTER_PERCENT") {
			v.add(path+".phase", "invalid_counter_heal_phase", "HEAL phase FINAL_RIDER requires COMBAT_PASSIVE COUNTER_PERCENT", id, &p)
		} else if typ == "CLEANSE" && (phase != "LATE_RIDER" || trigger != "COMBAT_PASSIVE" || sub != "STATUS") {
			v.add(path+".phase", "invalid_late_rider_phase", "CLEANSE phase LATE_RIDER requires COMBAT_PASSIVE STATUS", id, &p)
		} else if typ == "TRANSFER_RESOURCE" && phase != "LATE_RIDER" && !(phase == "EARLY_RIDER" && sub == "PERCENT_OF_TARGET_SHIELD" && trigger == "ON_ACTION") {
			v.add(path+".phase", "invalid_resource_transfer_phase", "TRANSFER_RESOURCE supports LATE_RIDER, or EARLY_RIDER for PERCENT_OF_TARGET_SHIELD", id, &p)
		} else if typ == "HP_COST" && ((phase != "EARLY_RIDER" && phase != "PRE_RIDER") || trigger != "ON_ACTION") {
			v.add(path+".phase", "invalid_hp_cost_phase", "HP_COST phase PRE_RIDER or EARLY_RIDER requires ON_ACTION", id, &p)
		} else if typ == "NOTIFY" && !((phase == "FINAL_RIDER" && trigger == "COMBAT_PASSIVE") || (phase == "PRE_CHAIN" && trigger == "ON_ACTION") || (phase == "BEFORE_HIT" && trigger == "BEFORE_HIT")) {
			v.add(path+".phase", "invalid_notify_phase", "NOTIFY supports COMBAT_PASSIVE FINAL_RIDER, ON_ACTION PRE_CHAIN, or BEFORE_HIT BEFORE_HIT", id, &p)
		} else if typ != "STATUS" && typ != "DETONATE_STATUS" && typ != "TRANSFER_RESOURCE" && typ != "DAMAGE_MODIFIER" && typ != "CLEANSE" && typ != "HP_COST" && typ != "HEAL" && typ != "NOTIFY" && typ != "DAMAGE" {
			v.add(path+".phase", "invalid_phase", "phase is not supported by this operation", id, &p)
		}
	}
	if typ == "CLEANSE" && sub == "STATUS" {
		var statusID string
		if json.Unmarshal(op["status_id"], &statusID) != nil {
			v.add(path+".status_id", "invalid_cleanse_status", "CLEANSE STATUS requires a known status_id", id, &p)
		} else if err := ValidateCombatStatusID(statusID); err != nil {
			v.add(path+".status_id", "invalid_cleanse_status", err.Error(), id, &p)
		}
	}
	for _, field := range def.Required {
		if _, exists := op[field]; !exists {
			if typ == "STATUS" && field == "chance" && optionalCanonical(op, "application_mode") == "DIRECT" {
				continue
			}
			v.add(path+"."+field, "required_field", field+" is required", id, &p)
		}
	}
	if typ == "NOTIFY" {
		var message, phase, key string
		raw, exists := op["message"]
		if exists && (json.Unmarshal(raw, &message) != nil || ValidateNotifyMessage(message) != nil) {
			v.add(path+".message", "invalid_message", "message must contain 1 to 500 characters", id, &p)
		}
		for field, dst := range map[string]*string{"phase": &phase, "key": &key} {
			if raw, exists := op[field]; exists && (bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, dst) != nil) {
				v.add(path+"."+field, "invalid_type", field+" must be a string", id, &p)
			}
		}
		if exists && ValidateNotifyMessage(message) == nil {
			if err := ValidateNotifyOperation(trigger, effectiveTarget, strings.ToUpper(phase), key, message); err != nil {
				v.add(path, "invalid_notify_configuration", err.Error(), id, &p)
			}
		}
		if raw, exists := op["log_action"]; exists {
			var action string
			if json.Unmarshal(raw, &action) != nil || ValidateCombatLogAction(action) != nil {
				v.add(path+".log_action", "invalid_log_action", "log_action must be an uppercase identifier", id, &p)
			}
		}
	}
	if typ == "STATUS" {
		if raw, exists := op["message"]; exists {
			var message string
			if json.Unmarshal(raw, &message) != nil || ValidateNotifyMessage(message) != nil {
				v.add(path+".message", "invalid_message", "message must contain 1 to 500 characters", id, &p)
			}
		}
		if raw, exists := op["log_action"]; exists {
			var action string
			if json.Unmarshal(raw, &action) != nil || ValidateCombatLogAction(action) != nil {
				v.add(path+".log_action", "invalid_log_action", "log_action must be an uppercase identifier", id, &p)
			}
		}
		if raw, exists := op["suppress_log"]; exists {
			var suppress bool
			if json.Unmarshal(raw, &suppress) != nil {
				v.add(path+".suppress_log", "invalid_type", "suppress_log must be a boolean", id, &p)
			} else if suppress && optionalCanonical(op, "application_mode") != "DIRECT" {
				v.add(path+".application_mode", "invalid_status_log_override", "suppress_log requires application_mode DIRECT", id, &p)
			}
		}
		if (op["message"] != nil || op["log_action"] != nil) && optionalCanonical(op, "application_mode") != "DIRECT" {
			v.add(path+".application_mode", "invalid_status_log_override", "message and log_action require application_mode DIRECT", id, &p)
		}
	}
	if typ == "SET_RULE" {
		var enabled bool
		raw, exists := op["enabled"]
		if exists && json.Unmarshal(raw, &enabled) != nil {
			v.add(path+".enabled", "invalid_type", "enabled must be a boolean", id, &p)
		}
	}
	if typ == "TRANSFER_RESOURCE" {
		var subType, resource, phase, logAction, message string
		var fraction, minAmount float64
		typed := true
		if raw, exists := op["sub_type"]; !exists || json.Unmarshal(raw, &subType) != nil {
			typed = false
		}
		if raw, exists := op["resource"]; !exists || json.Unmarshal(raw, &resource) != nil {
			typed = false
		}
		if raw, exists := op["value"]; !exists || json.Unmarshal(raw, &fraction) != nil {
			typed = false
		}
		if raw, exists := op["min_amount"]; exists && (bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, &minAmount) != nil) {
			typed = false
		}
		for field, dst := range map[string]*string{"phase": &phase, "log_action": &logAction, "message": &message} {
			if raw, exists := op[field]; exists && (bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, dst) != nil) {
				typed = false
			}
		}
		if !typed {
			v.add(path, "invalid_resource_transfer", "TRANSFER_RESOURCE fields have invalid types", id, &p)
		} else if err := ValidateResourceTransfer(strings.ToUpper(subType), strings.ToUpper(resource), fraction, minAmount, strings.ToUpper(phase), logAction, message); err != nil {
			v.add(path, "invalid_resource_transfer", err.Error(), id, &p)
		}
	}
	if typ == "RESOURCE_CONVERSION" {
		if strings.EqualFold(sub, "CURRENT_HP_TO_SHIELD") {
			var subType, logAction, message string
			var fraction, minRemaining float64
			typed := true
			for field, dst := range map[string]*string{"sub_type": &subType, "log_action": &logAction, "message": &message} {
				if raw, exists := op[field]; exists && (bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, dst) != nil) {
					typed = false
				}
			}
			for field, dst := range map[string]*float64{"value": &fraction, "min_remaining": &minRemaining} {
				if raw, exists := op[field]; !exists || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) || json.Unmarshal(raw, dst) != nil {
					typed = false
				}
			}
			if !typed {
				v.add(path, "invalid_resource_conversion", "RESOURCE_CONVERSION fields have invalid types", id, &p)
			} else if err := ValidateResourceConversion(strings.ToUpper(subType), fraction, minRemaining, logAction, message); err != nil {
				v.add(path, "invalid_resource_conversion", err.Error(), id, &p)
			}
		} else if strings.EqualFold(sub, "DISPEL_TO_RESOURCES") {
			if err := ValidateDispelToResourcesParams(op["params"]); err != nil {
				v.add(path+".params", "invalid_dispel_to_resources_params", err.Error(), id, &p)
			}
		}
	}
	if raw, exists := op["value"]; exists {
		var x float64
		if json.Unmarshal(raw, &x) != nil {
			v.add(path+".value", "invalid_type", "value must be a number", id, &p)
		} else if (def.NonNegative && x < 0) || (def.Ratio && (x < 0 || x > 1)) {
			v.add(path+".value", "invalid_range", "value is outside the allowed range", id, &p)
		} else if typ == "DISPLACE" {
			if err := ValidateDisplacement(sub, x); err != nil {
				v.add(path+".value", "invalid_displacement_value", err.Error(), id, &p)
			}
		}
	} else if typ == "DISPLACE" && sub == "SET_ROW" {
		v.add(path+".value", "required_field", "DISPLACE SET_ROW requires an explicit row", id, &p)
	}
	for field, registry := range map[string]map[string]bool{"stat": StatCatalog, "resource": ResourceCatalog, "unit": UnitCatalog, "stack_policy": StackPolicyCatalog} {
		if raw, exists := op[field]; exists {
			var s string
			if json.Unmarshal(raw, &s) != nil || s == "" {
				v.add(path+"."+field, "invalid_type", field+" must be a string", id, &p)
			} else if s != strings.ToUpper(strings.TrimSpace(s)) {
				v.add(path+"."+field, "noncanonical_value", field+" must use canonical casing", id, &p)
			} else if !registry[s] {
				v.add(path+"."+field, "unknown_"+field, field+" is not registered", id, &p)
			}
		}
	}
	if raw, exists := op["status_id"]; exists && typ == "REVOKE_STATUS" {
		var statusID string
		if json.Unmarshal(raw, &statusID) != nil {
			v.add(path+".status_id", "invalid_type", "status_id must be a string", id, &p)
		} else if err := ValidatePersistentStatusID(statusID); err != nil {
			v.add(path+".status_id", "invalid_status_id", err.Error(), id, &p)
		}
	} else if raw, exists := op["status_id"]; exists {
		var statusID string
		if json.Unmarshal(raw, &statusID) != nil || statusID == "" {
			v.add(path+".status_id", "invalid_type", "status_id must be a string", id, &p)
		} else if statusID != strings.ToUpper(strings.TrimSpace(statusID)) {
			v.add(path+".status_id", "noncanonical_value", "status_id must use canonical casing", id, &p)
		} else if !StatusCatalog[statusID] {
			v.add(path+".status_id", "unknown_status_id", "status_id is not registered", id, &p)
		}
	}
	if raw, exists := op["turns"]; exists {
		var n int
		if json.Unmarshal(raw, &n) != nil || n < 1 {
			v.add(path+".turns", "invalid_turns", "turns must be a positive integer", id, &p)
		}
	}
	if raw, exists := op["chance"]; exists {
		var x float64
		if json.Unmarshal(raw, &x) != nil || x < 0 || x > 1 {
			v.add(path+".chance", "invalid_chance", "chance must be between 0 and 1", id, &p)
		}
	}
	if typ == "UNIQUE" {
		code := v.requiredCanonical(op, "code", path, id, &p)
		hook, exists := NativeHooks[code]
		if code != "" && !exists {
			v.add(path+".code", "unknown_native_hook", "UNIQUE code is not registered", id, &p)
		} else if exists {
			if !hook.Triggers[trigger] {
				v.add(path+".code", "incompatible_native_trigger", "native hook is not registered for trigger", id, &p)
			}
			v.params(op["params"], path+".params", id, &p, hook)
		}
	}
	if typ == "GRANT_CAPABILITY" {
		code := sub
		hook, exists := NativeHooks[code]
		if !exists || hook.Domain != "capability" {
			v.add(path+".sub_type", "unknown_capability", "capability is not registered", id, &p)
		} else if !hook.Triggers[trigger] {
			v.add(path+".sub_type", "incompatible_capability_trigger", "capability is not registered for trigger", id, &p)
		} else if err := ValidateNativeParams(code, op["params"]); err != nil {
			v.add(path+".params", "invalid_capability_params", err.Error(), id, &p)
		}
	}
	if typ == "REVIVE" {
		if hasConditions {
			v.add(path, "unsupported_conditions", "REVIVE does not support block conditions", id, &p)
		}
		if err := ValidateReviveParams(sub, op["params"]); err != nil {
			v.add(path+".params", "invalid_revive_params", err.Error(), id, &p)
		}
	}
	if typ == "EFFECT_IMMUNITY" && hasConditions {
		v.add(path, "unsupported_conditions", "EFFECT_IMMUNITY does not support block conditions", id, &p)
	}
	if typ == "RANK_SCALED_DEBUFF" {
		if hasConditions {
			v.add(path, "unsupported_conditions", "RANK_SCALED_DEBUFF does not support block conditions", id, &p)
		}
		if err := ValidateRankScaledDebuffParams(sub, op["params"]); err != nil {
			v.add(path+".params", "invalid_rank_scaled_debuff_params", err.Error(), id, &p)
		}
	}
	if typ == "GU_SEAL" {
		if hasConditions {
			v.add(path, "unsupported_conditions", "GU_SEAL does not support block conditions", id, &p)
		}
		if err := ValidateGuSealParams(op["params"]); err != nil {
			v.add(path+".params", "invalid_gu_seal_params", err.Error(), id, &p)
		}
	}
	if typ == "TARGET_REDIRECTION" {
		if hasConditions {
			v.add(path, "unsupported_conditions", "TARGET_REDIRECTION does not support block conditions", id, &p)
		}
		if err := ValidateTargetRedirectionParams(sub, op["params"]); err != nil {
			v.add(path+".params", "invalid_target_redirection_params", err.Error(), id, &p)
		}
	}
	if typ == "OUTCOME_ROULETTE" {
		if hasConditions {
			v.add(path, "unsupported_conditions", "OUTCOME_ROULETTE does not support block conditions", id, &p)
		}
		if err := ValidateOutcomeRouletteParams(sub, op["params"]); err != nil {
			v.add(path+".params", "invalid_outcome_roulette_params", err.Error(), id, &p)
		}
	}
	if typ == "PERIODIC_RESOURCE" {
		var resource string
		_ = json.Unmarshal(op["resource"], &resource)
		if err := ValidatePeriodicResource(sub, resource); err != nil {
			v.add(path+".resource", "invalid_periodic_resource", err.Error(), id, &p)
		}
	}
	if typ == "SUMMON" {
		if err := ValidateSummonParams(op["params"]); err != nil {
			v.add(path+".params", "invalid_summon_params", err.Error(), id, &p)
		}
	}
	if typ == "SLOT_INTERACTION" {
		if err := ValidateSlotInteractionParams(sub, op["params"]); err != nil {
			v.add(path+".params", "invalid_slot_interaction", err.Error(), id, &p)
		}
	}
}

func (v *blockValidator) params(raw json.RawMessage, path, id string, op *int, s NativeHook) {
	var m map[string]json.RawMessage
	if len(raw) == 0 {
		m = map[string]json.RawMessage{}
	} else if decodeObject(raw, &m) != nil {
		v.add(path, "invalid_type", "params must be an object", id, op)
		return
	}
	for k := range m {
		if _, ok := s.Params[k]; !ok {
			v.add(path+"."+k, "unknown_param", "native hook parameter is not declared", id, op)
		}
	}
	for k, p := range s.Params {
		r, ok := m[k]
		if p.Required && !ok {
			v.add(path+"."+k, "required_param", "native hook parameter is required", id, op)
			continue
		}
		if !ok {
			continue
		}
		if !p.validate(r) {
			v.add(path+"."+k, "invalid_param", "native hook parameter has invalid type or range", id, op)
		}
	}
}
func (v *blockValidator) requiredCanonical(m map[string]json.RawMessage, key, path, id string, op *int) string {
	raw, ok := m[key]
	var s string
	if !ok || json.Unmarshal(raw, &s) != nil || strings.TrimSpace(s) == "" {
		v.add(path+"."+key, "required_field", key+" must be a non-empty string", id, op)
		return ""
	}
	c := strings.ToUpper(strings.TrimSpace(s))
	if s != c || !canonicalID.MatchString(c) {
		v.add(path+"."+key, "noncanonical_value", key+" must use UPPER_SNAKE_CASE", id, op)
	}
	return c
}
func optionalCanonical(m map[string]json.RawMessage, key string) string {
	var s string
	_ = json.Unmarshal(m[key], &s)
	return strings.ToUpper(strings.TrimSpace(s))
}
func (v *blockValidator) target(raw json.RawMessage, path, id string, op *int, required bool) string {
	var m map[string]json.RawMessage
	if len(raw) == 0 || decodeObject(raw, &m) != nil {
		if required {
			v.add(path, "required_field", "target must be an object", id, op)
		}
		return ""
	}
	allowed := map[string]bool{"selector": true, "bypass_frontline": true}
	for k := range m {
		if !allowed[k] {
			v.add(path+"."+k, "unknown_field", "unknown target field "+k, id, op)
		}
	}
	sel := v.requiredCanonical(m, "selector", path, id, op)
	if !TargetCatalog[sel] {
		v.add(path+".selector", "unknown_target", "target selector is not registered", id, op)
	}
	if r, ok := m["bypass_frontline"]; ok {
		var b bool
		if json.Unmarshal(r, &b) != nil {
			v.add(path+".bypass_frontline", "invalid_type", "bypass_frontline must be boolean", id, op)
		}
	}
	return sel
}
func decodeObject(raw []byte, out *map[string]json.RawMessage) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(out); err != nil {
		return err
	}
	if *out == nil || firstNonSpace(raw) != '{' {
		return fmt.Errorf("value must be an object")
	}
	var trailing any
	if err := d.Decode(&trailing); err == nil {
		return fmt.Errorf("unexpected trailing JSON value")
	} else if err != io.EOF {
		return err
	}
	return nil
}
func triggerAllowsTarget(trigger, target string) bool {
	if trigger == "ALWAYS_ON" || trigger == "ON_TICK" || trigger == "CAPABILITY" || trigger == "ON_CONSUME" || trigger == "ON_USE" {
		return target == "SELF"
	}
	return true
}
