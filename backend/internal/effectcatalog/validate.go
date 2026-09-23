// Package effectcatalog validates and normalizes the authoring format used by
// Gu effects. It deliberately does not execute effects: callers can reject
// structurally unsafe data while retaining legacy fields that are not wired to
// the runtime yet.
package effectcatalog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/statusfx"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Issue struct {
	Severity Severity `json:"severity"`
	Code     string   `json:"code"`
	Path     string   `json:"path"`
	Message  string   `json:"message"`
}

type Report struct {
	Issues []Issue `json:"issues"`
}

func (r Report) HasErrors() bool {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (r Report) Warnings() []Issue {
	result := make([]Issue, 0)
	for _, issue := range r.Issues {
		if issue.Severity == SeverityWarning {
			result = append(result, issue)
		}
	}
	return result
}

func (r Report) Error() string {
	for _, issue := range r.Issues {
		if issue.Severity == SeverityError {
			return fmt.Sprintf("%s: %s", issue.Path, issue.Message)
		}
	}
	return ""
}

var Sections = []string{"blocks", "active", "support", "passive", "house", "explore", "refinement", "growth", "other"}

var sectionSet = func() map[string]bool {
	m := make(map[string]bool, len(Sections))
	for _, section := range Sections {
		m[section] = true
	}
	return m
}()

// runtimeFields are fields represented by the current typed Go runtime.
// Legacy catalog fields outside this set are retained but reported, never
// silently presented as supported.
var runtimeFields = map[string]map[string]bool{
	"active": set(
		"damage_flat", "heal_flat", "shield_flat", "apply_status", "status_tick_damage", "status_duration_ticks", "target_selector",
		"range", "curve_attack", "mana_regen", "wood_transformation", "blood_drain", "replicate", "lifesteal_rate", "refine_mana_tier",
		"damage_mult", "shield_mult", "cleansing_power", "summon_phantom", "generate_gu", "capture_bonus", "portal_create", "random_portal",
		"summon_growth_phantoms", "summon_count_per_tick", "phantom_damage_scale", "ignore_defense", "ignore_defense_pct", "crit_rate_boost",
		"all_stats_buff_percent", "speed_boost_flat", "freeze", "slow", "slow1", "weak", "stun", "silence", "bleed", "poison",
		"thorns", "regen", "atk_up", "def_up", "taunt",
		"damage_hp_percent", "shield_flat_reduce_percent", "disable_gu_next_turn", "forget_reality_chance",
		"self_evasion_boost_percent", "target_damage_reduce_percent", "damage_path", "stat_base_attack",
	),
	"passive": set(
		"stat_base_attack", "stat_base_defense", "stat_base_hp", "stat_base_speed", "path_conflict_reduction", "damage_multiplier_on_hit",
		"soul_refinement", "storage_capacity", "movement", "investigation", "hearing", "evasion", "refine_mana_tier", "cleansing_power",
		"luck", "success_rate_bonus", "time_speed_multiplier", "accuracy", "tenacity", "cc_resist",
		"anti_heaven_damage_reduction_percent", "anti_earth_damage_reduction_percent", "anti_dream_damage_reduction_percent",
		"immune_dream_status", "forget_heaven_protection", "forget_earth_protection", "disguise_level",
	),
	"growth": set("max_attack", "max_defense", "max_hp", "max_speed", "hours_to_max"),
}

var numberFields = set(
	"damage_flat", "heal_flat", "shield_flat", "status_tick_damage", "range", "mana_regen", "lifesteal_rate", "refine_mana_tier",
	"damage_mult", "shield_mult", "cleansing_power", "capture_bonus", "summon_count_per_tick", "phantom_damage_scale", "ignore_defense", "ignore_defense_pct",
	"crit_rate_boost", "all_stats_buff_percent", "speed_boost_flat", "freeze", "slow", "slow1", "weak", "stun", "silence", "bleed",
	"poison", "thorns", "regen", "atk_up", "def_up", 	"taunt", "tenacity", "cc_resist", "stat_base_attack", "stat_base_defense", "stat_base_hp",
	"stat_base_speed", "path_conflict_reduction", "damage_multiplier_on_hit", "soul_refinement", "storage_capacity", "movement",
	"investigation", "hearing", "evasion", "luck", "success_rate_bonus", "time_speed_multiplier", "accuracy", "max_attack", "max_defense",
	"max_hp", "max_speed", "hours_to_max",
	"damage_hp_percent", "shield_flat_reduce_percent", "forget_reality_chance", "self_evasion_boost_percent", "target_damage_reduce_percent",
	"anti_heaven_damage_reduction_percent", "anti_earth_damage_reduction_percent", "anti_dream_damage_reduction_percent",
	"disguise_level",
	"absorb_damage_flat", "convert_damage_to_heal_percent", "decoy_count", "invulnerable_ticks", "evasion", "stat_base_luck",
	"dao_mark_multiplier", "kill_path_damage_buff", "luck_flat", "storage_slots", "kill_path_dao_marks_mult",
)
var integerFields = set("status_duration_ticks", "decoy_count", "invulnerable_ticks", "storage_slots")
var booleanFields = set(
	"curve_attack", "wood_transformation", "blood_drain", "replicate", "summon_phantom", "generate_gu", "portal_create", "random_portal",
	"summon_growth_phantoms", "reveal_all_stealth", "reveal_weakness",
	"disable_gu_next_turn", "immune_dream_status", "forget_heaven_protection", "forget_earth_protection",
)
var stringFields = set("target_selector", "summon_monster_id", "damage_path")
	var ratioFields = set("lifesteal_rate", "ignore_defense_pct", "crit_rate_boost", "all_stats_buff_percent", "convert_damage_to_heal_percent", "kill_path_damage_buff", "tenacity", "cc_resist",
		"damage_hp_percent", "shield_flat_reduce_percent", "forget_reality_chance", "self_evasion_boost_percent", "target_damage_reduce_percent",
		"anti_heaven_damage_reduction_percent", "anti_earth_damage_reduction_percent", "anti_dream_damage_reduction_percent")
var signedNumberFields = set("luck", "success_rate_bonus")
var targetSelectors = set("SINGLE", "ROW", "COL", "COLUMN", "CROSS", "ALL")

func set(values ...string) map[string]bool {
	m := make(map[string]bool, len(values))
	for _, value := range values {
		m[value] = true
	}
	return m
}

// NormalizeAndValidate preserves every author-provided field, adds missing
// envelope sections, and returns errors for malformed data. A flat legacy
// object is treated as the active section.
func NormalizeAndValidate(raw []byte) ([]byte, Report, error) {
	report := Report{}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		trimmed = []byte(`{}`)
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &root); err != nil {
		report.add(SeverityError, "invalid_json", "$", err.Error())
		return nil, report, report
	}
	if root == nil || trimmed[0] != '{' {
		report.add(SeverityError, "invalid_root", "$", "effect phải là JSON object")
		return nil, report, report
	}

	hasSection := false
	for key := range root {
		if sectionSet[key] {
			hasSection = true
			break
		}
	}
	if !hasSection {
		for key := range root {
			if _, ok := DefinitionFor("active", key); !ok {
				report.add(SeverityError, "unknown_section", "$."+key, "namespace effect không được hỗ trợ")
			}
		}
		if report.HasErrors() {
			report.sort()
			return nil, report, report
		}
		legacy := root
		root = map[string]json.RawMessage{}
		encoded, _ := json.Marshal(legacy)
		root["active"] = encoded
	}

	for key := range root {
		if !sectionSet[key] {
			report.add(SeverityError, "unknown_section", "$."+key, "namespace effect không được hỗ trợ")
		}
	}

	for _, section := range Sections {
		value, ok := root[section]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			if section == "other" || section == "blocks" {
				root[section] = json.RawMessage(`[]`)
			} else {
				root[section] = json.RawMessage(`{}`)
			}
			continue
		}
		if section == "other" {
			validateOther(value, &report)
			continue
		}
		if section == "blocks" {
			validateBlocks(value, &report)
			continue
		}
		validateObjectSection(section, value, &report)
	}

	if report.HasErrors() {
		report.sort()
		return nil, report, report
	}
	report.sort()
	normalized, err := marshalOrdered(root)
	if err != nil {
		report.add(SeverityError, "encode_failed", "$", err.Error())
		return nil, report, report
	}
	return normalized, report, nil
}

func validateObjectSection(section string, raw json.RawMessage, report *Report) {
	var values map[string]json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil || firstNonSpace(raw) != '{' {
		report.add(SeverityError, "invalid_section_type", "$."+section, "namespace phải là JSON object")
		return
	}
	for key, value := range values {
		path := "$." + section + "." + key
		definition, classified := DefinitionFor(section, key)
		if !classified {
			report.add(SeverityError, "unclassified_effect_key", path, "effect key chưa có owner và disposition trong manifest")
		} else if definition.Disposition != DispositionSupported && definition.Disposition != DispositionMetadata {
			report.add(SeverityWarning, "effect_"+string(definition.Disposition), path, "owner="+definition.Owner)
		}
		validateFieldValue(section, key, value, path, report)
		if section == "active" && key == "apply_status" {
			validateApplyStatus(value, path, report)
		}
		if (section == "active" && key == "status_duration_ticks") || (section == "growth" && key == "hours_to_max") {
			var number float64
			if err := json.Unmarshal(value, &number); err != nil || number < 0 {
				report.add(SeverityError, "invalid_range", path, "giá trị phải là số không âm")
			}
		}
	}
}

func validateFieldValue(section, key string, raw json.RawMessage, path string, report *Report) {
	if key == "description" {
		var value string
		if json.Unmarshal(raw, &value) != nil {
			report.add(SeverityError, "invalid_field_type", path, "giá trị phải là string")
		}
		return
	}
	if integerFields[key] {
		var value int
		if json.Unmarshal(raw, &value) != nil {
			report.add(SeverityError, "invalid_field_type", path, "giá trị phải là integer")
		}
		return
	}
	if numberFields[key] {
		var value float64
		if json.Unmarshal(raw, &value) != nil {
			report.add(SeverityError, "invalid_field_type", path, "giá trị phải là number")
			return
		}
		if ratioFields[key] && (value < 0 || value > 1) {
			report.add(SeverityError, "invalid_range", path, "tỉ lệ phải nằm trong khoảng 0..1")
		} else if value < 0 && !signedNumberFields[key] {
			report.add(SeverityError, "invalid_range", path, "giá trị phải là số không âm")
		}
		return
	}
	if booleanFields[key] {
		var value bool
		if json.Unmarshal(raw, &value) != nil {
			report.add(SeverityError, "invalid_field_type", path, "giá trị phải là boolean")
		}
		return
	}
	if stringFields[key] {
		var value string
		if json.Unmarshal(raw, &value) != nil {
			report.add(SeverityError, "invalid_field_type", path, "giá trị phải là string")
		} else if key == "target_selector" {
			canonical := strings.ToUpper(strings.TrimSpace(value))
			if !targetSelectors[canonical] {
				report.add(SeverityError, "invalid_target_selector", path, "target_selector không được hỗ trợ")
			} else if value != canonical {
				report.add(SeverityError, "noncanonical_target_selector", path, "target_selector phải viết hoa canonical")
			}
		}
	}
	_ = section
}

// statusEntry mirrors one apply_status entry in any of its authoring forms.
type statusEntry struct {
	ID     string          `json:"id"`
	Value  json.RawMessage `json:"value"`
	Turns  json.RawMessage `json:"turns"`
	Chance json.RawMessage `json:"chance"`
}

// validateApplyStatus accepts the three authoring forms — a legacy status
// string, a single {id,value,turns,chance} object, or an array of such objects —
// and validates every referenced status against the manifest + statusfx.
func validateApplyStatus(raw json.RawMessage, path string, report *Report) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return
	}
	var entries []statusEntry
	switch trimmed[0] {
	case '"':
		var id string
		if err := json.Unmarshal(trimmed, &id); err != nil {
			report.add(SeverityError, "invalid_status", path, "apply_status phải là chuỗi, object {id,value,turns,chance} hoặc mảng")
			return
		}
		entries = []statusEntry{{ID: id}}
	case '{':
		var one statusEntry
		if err := json.Unmarshal(trimmed, &one); err != nil {
			report.add(SeverityError, "invalid_status", path, "apply_status object phải có dạng {id,value,turns,chance}")
			return
		}
		entries = []statusEntry{one}
	case '[':
		if err := json.Unmarshal(trimmed, &entries); err != nil {
			report.add(SeverityError, "invalid_status", path, "apply_status mảng phải chứa các object {id,value,turns,chance}")
			return
		}
	default:
		report.add(SeverityError, "invalid_status", path, "apply_status phải là chuỗi, object {id,value,turns,chance} hoặc mảng")
		return
	}
	if len(entries) == 0 {
		report.add(SeverityError, "invalid_status", path, "apply_status không được rỗng")
		return
	}
	for i, entry := range entries {
		entryPath := fmt.Sprintf("%s[%d]", path, i)
		if len(entries) == 1 && trimmed[0] == '{' {
			entryPath = path
		}
		id := strings.ToUpper(strings.TrimSpace(entry.ID))
		if entry.ID == "" || id == "" {
			report.add(SeverityError, "invalid_status", entryPath+".id", "id phải là chuỗi không rỗng")
			continue
		}
		if entry.ID != id {
			report.add(SeverityError, "noncanonical_status", entryPath+".id", "status phải viết hoa canonical")
		}
		definition, ok := StatusDefinitionFor(id)
		if !ok {
			report.add(SeverityError, "unknown_status", entryPath+".id", "status chưa có owner và disposition trong manifest")
		} else if definition.Disposition != DispositionSupported {
			report.add(SeverityWarning, "status_"+string(definition.Disposition), entryPath+".id", "owner="+definition.Owner)
		} else if _, ok := statusfx.Spec(id); !ok {
			report.add(SeverityError, "status_owner_mismatch", entryPath+".id", "status được đánh dấu supported nhưng statusfx không nhận diện")
		}
		for field, rawValue := range map[string]json.RawMessage{"value": entry.Value, "turns": entry.Turns} {
			if len(rawValue) == 0 {
				continue
			}
			var number float64
			if err := json.Unmarshal(rawValue, &number); err != nil || number < 0 {
				report.add(SeverityError, "invalid_status_field", entryPath+"."+field, field+" phải là số không âm")
			}
		}
		if len(entry.Chance) > 0 {
			var chance float64
			if err := json.Unmarshal(entry.Chance, &chance); err != nil || chance < 0 || chance > 1 {
				report.add(SeverityError, "invalid_status_field", entryPath+".chance", "chance phải trong khoảng 0..1")
			}
		}
	}
}

func validateOther(raw json.RawMessage, report *Report) {
	var mechanics []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &mechanics); err != nil || firstNonSpace(raw) != '[' {
		report.add(SeverityError, "invalid_section_type", "$.other", "namespace other phải là JSON array")
		return
	}
	for i, mechanic := range mechanics {
		base := fmt.Sprintf("$.other[%d]", i)
		for _, field := range []string{"event_trigger", "action_code"} {
			var value string
			rawValue, ok := mechanic[field]
			if !ok || json.Unmarshal(rawValue, &value) != nil || strings.TrimSpace(value) == "" {
				report.add(SeverityError, "missing_mechanic_field", base+"."+field, field+" phải là chuỗi không rỗng")
			}
		}
		var trigger, action string
		triggerOK := json.Unmarshal(mechanic["event_trigger"], &trigger) == nil && trigger != ""
		actionOK := json.Unmarshal(mechanic["action_code"], &action) == nil && action != ""
		if triggerOK && actionOK {
			canonicalTrigger := strings.ToUpper(strings.TrimSpace(trigger))
			canonicalAction := strings.ToUpper(strings.TrimSpace(action))
			if trigger != canonicalTrigger || action != canonicalAction {
				report.add(SeverityError, "noncanonical_mechanic", base, "event_trigger và action_code phải viết hoa canonical")
			}
			trigger, action = canonicalTrigger, canonicalAction
			definition, ok := MechanicDefinitionFor(trigger, action)
			if !ok {
				report.add(SeverityError, "unclassified_mechanic", base+".action_code", "cặp event_trigger/action_code chưa có owner và disposition")
			} else if definition.Disposition != DispositionSupported {
				report.add(SeverityWarning, "mechanic_"+string(definition.Disposition), base+".action_code", "owner="+definition.Owner)
			}
		}
		if params, ok := mechanic["params"]; ok && firstNonSpace(params) != '{' {
			report.add(SeverityError, "invalid_mechanic_params", base+".params", "params phải là JSON object")
		} else if ok && actionOK {
			validateMechanicParams(strings.ToUpper(strings.TrimSpace(action)), params, base+".params", report)
		}
		for field := range mechanic {
			if field != "event_trigger" && field != "action_code" && field != "params" {
				report.add(SeverityWarning, "unknown_mechanic_field", base+"."+field, "field mechanic không được runtime sử dụng")
			}
		}
	}
}

func validateMechanicParams(action string, raw json.RawMessage, path string, report *Report) {
	var params map[string]json.RawMessage
	if json.Unmarshal(raw, &params) != nil {
		return
	}
	requireRange := func(key string, min, max float64) {
		valueRaw, ok := params[key]
		var value float64
		if !ok || json.Unmarshal(valueRaw, &value) != nil || value < min || value > max {
			report.add(SeverityError, "invalid_mechanic_param", path+"."+key, fmt.Sprintf("%s phải trong khoảng %g..%g", key, min, max))
		}
	}
	requireOptionalRange := func(key string, min, max float64) {
		valueRaw, ok := params[key]
		if !ok {
			return
		}
		var value float64
		if json.Unmarshal(valueRaw, &value) != nil || value < min || value > max {
			report.add(SeverityError, "invalid_mechanic_param", path+"."+key, fmt.Sprintf("%s phải trong khoảng %g..%g", key, min, max))
		}
	}
	switch action {
	case "REBORN":
		requireRange("chance", 0, 1)
		// mode: rewind (reset battle to turn 1) or in_place (stand up at current HP).
		if modeRaw, ok := params["mode"]; ok {
			var mode string
			if json.Unmarshal(modeRaw, &mode) != nil || strings.TrimSpace(mode) == "" {
				report.add(SeverityError, "invalid_mechanic_param", path+".mode", "mode phải là chuỗi không rỗng")
			} else if m := strings.ToLower(strings.TrimSpace(mode)); m != "rewind" && m != "in_place" {
				report.add(SeverityError, "invalid_mechanic_param", path+".mode", "mode phải là rewind hoặc in_place")
			}
		}
		requireOptionalRange("max_uses", 1, 10)
		requireOptionalRange("revive_hp_percent", 0, 1)
		requireOptionalRange("empower_attack_percent", 0, 5)
		requireOptionalRange("empower_speed_percent", 0, 5)
		requireOptionalRange("empower_shield_percent", 0, 5)
		requireOptionalRange("empower_turns", 1, 100)
	case "FORGET_INTENT_INTERRUPT":
		requireRange("chance", 0, 1)
		requireOptionalRange("disable_target_gu_turns", 0, 10)
		requireOptionalRange("max_successful_activations", 1, 100)
	case "FORGET_WILL_CONFUSE":
		requireOptionalRange("activation_chance", 0, 1)
		requireOptionalRange("hit_user_chance", 0, 1)
		requireOptionalRange("hit_self_chance", 0, 1)
		requireOptionalRange("miss_chance", 0, 1)
		requireOptionalRange("backfire_chance", 0, 1)
		requireOptionalRange("backfire_disable_turns", 0, 10)
	case "SACRIFICE_NPC":
		requireRange("aptitude_gain", 1, 99)
		requireRange("aptitude_cap", 1, 100)
	}
}

func firstNonSpace(raw []byte) byte {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return 0
	}
	return trimmed[0]
}

func marshalOrdered(root map[string]json.RawMessage) ([]byte, error) {
	ordered := bytes.NewBufferString("{")
	for i, section := range Sections {
		if i > 0 {
			ordered.WriteByte(',')
		}
		key, _ := json.Marshal(section)
		ordered.Write(key)
		ordered.WriteByte(':')
		ordered.Write(root[section])
	}
	ordered.WriteByte('}')
	return ordered.Bytes(), nil
}

func (r *Report) add(severity Severity, code, path, message string) {
	r.Issues = append(r.Issues, Issue{Severity: severity, Code: code, Path: path, Message: message})
}

func (r *Report) sort() {
	sort.SliceStable(r.Issues, func(i, j int) bool {
		if r.Issues[i].Path == r.Issues[j].Path {
			return r.Issues[i].Code < r.Issues[j].Code
		}
		return r.Issues[i].Path < r.Issues[j].Path
	})
}

var allowedBlockTriggers = set(
	"ALWAYS_ON", "ON_ACTION", "ON_HIT", "BEFORE_HIT", "ON_TURN_START",
	"ON_DEATH", "ON_TICK", "CAPABILITY", "COMBAT_PASSIVE", "PASSIVE",
	"ON_BATTLE_START", "BEFORE_ENEMY_ATTACK", "ON_CAST", "ON_CONSUME", "ON_USE",
)

var allowedTargetSelectors = set(
	"SELF", "TARGET", "ROW", "COL", "COLUMN", "CROSS", "ALL_ENEMIES", "ALL_ALLIES", "ALL", "SINGLE",
)

var allowedAtomicOps = set(
	"DAMAGE", "SHIELD", "HEAL", "DIRECT_HP", "STAT_MOD", "STATUS", "BARRIER", "CLEANSE", "UNIQUE",
)

func validateBlocks(raw json.RawMessage, report *Report) {
	var blocks []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &blocks); err != nil || firstNonSpace(raw) != '[' {
		report.add(SeverityError, "invalid_section_type", "$.blocks", "namespace blocks phải là JSON array")
		return
	}
	for i, block := range blocks {
		base := fmt.Sprintf("$.blocks[%d]", i)
		var trigger string
		triggerRaw, hasTrigger := block["trigger"]
		if !hasTrigger || json.Unmarshal(triggerRaw, &trigger) != nil || strings.TrimSpace(trigger) == "" {
			report.add(SeverityError, "missing_block_trigger", base+".trigger", "trigger phải là chuỗi không rỗng")
		} else {
			canonicalTrigger := strings.ToUpper(strings.TrimSpace(trigger))
			if trigger != canonicalTrigger {
				report.add(SeverityError, "noncanonical_trigger", base+".trigger", "trigger phải viết hoa canonical")
			}
			if !allowedBlockTriggers[canonicalTrigger] {
				report.add(SeverityError, "invalid_block_trigger", base+".trigger", "trigger không được hỗ trợ: "+canonicalTrigger)
			}
		}

		targetRaw, hasTarget := block["target"]
		if !hasTarget || firstNonSpace(targetRaw) != '{' {
			report.add(SeverityError, "invalid_block_target", base+".target", "target phải là JSON object")
		} else {
			var target map[string]json.RawMessage
			if json.Unmarshal(targetRaw, &target) == nil {
				if selRaw, ok := target["selector"]; ok {
					var sel string
					if json.Unmarshal(selRaw, &sel) != nil {
						report.add(SeverityError, "invalid_target_selector", base+".target.selector", "selector phải là chuỗi")
					} else {
						cSel := strings.ToUpper(strings.TrimSpace(sel))
						if !allowedTargetSelectors[cSel] {
							report.add(SeverityError, "invalid_target_selector", base+".target.selector", "selector không được hỗ trợ: "+cSel)
						}
					}
				}
			}
		}

		opsRaw, hasOps := block["operations"]
		if !hasOps || firstNonSpace(opsRaw) != '[' {
			report.add(SeverityError, "missing_block_operations", base+".operations", "operations phải là JSON array")
			continue
		}
		var ops []map[string]json.RawMessage
		if err := json.Unmarshal(opsRaw, &ops); err != nil {
			report.add(SeverityError, "invalid_block_operations", base+".operations", "operations JSON không hợp lệ")
			continue
		}
		if len(ops) == 0 {
			report.add(SeverityError, "empty_block_operations", base+".operations", "operations không được rỗng")
			continue
		}
		for j, op := range ops {
			opBase := fmt.Sprintf("%s.operations[%d]", base, j)
			var opType string
			opTypeRaw, hasOpType := op["op"]
			if !hasOpType || json.Unmarshal(opTypeRaw, &opType) != nil || strings.TrimSpace(opType) == "" {
				report.add(SeverityError, "missing_op_type", opBase+".op", "op phải là chuỗi không rỗng")
				continue
			}
			canonicalOp := strings.ToUpper(strings.TrimSpace(opType))
			if opType != canonicalOp {
				report.add(SeverityError, "noncanonical_op", opBase+".op", "op phải viết hoa canonical")
			}
			if !allowedAtomicOps[canonicalOp] {
				report.add(SeverityError, "invalid_op_type", opBase+".op", "op type không được hỗ trợ: "+canonicalOp)
				continue
			}

			// Validate specific ops
			if canonicalOp == "STATUS" {
				var statusID string
				if sRaw, ok := op["status_id"]; !ok || json.Unmarshal(sRaw, &statusID) != nil || strings.TrimSpace(statusID) == "" {
					report.add(SeverityError, "missing_status_id", opBase+".status_id", "status_id phải là chuỗi không rỗng")
				} else {
					cID := strings.ToUpper(strings.TrimSpace(statusID))
					if statusID != cID {
						report.add(SeverityError, "noncanonical_status", opBase+".status_id", "status_id phải viết hoa canonical")
					}
					if _, ok := StatusDefinitionFor(cID); !ok {
						report.add(SeverityError, "unknown_status", opBase+".status_id", "status_id không có trong manifest: "+cID)
					}
				}
			}

			if valRaw, ok := op["value"]; ok && len(valRaw) > 0 {
				var val float64
				if json.Unmarshal(valRaw, &val) != nil || (val < 0 && canonicalOp != "STAT_MOD") {
					report.add(SeverityError, "invalid_op_value", opBase+".value", "value phải là số không âm")
				}
			}

			if turnsRaw, ok := op["turns"]; ok && len(turnsRaw) > 0 {
				var turns int
				if json.Unmarshal(turnsRaw, &turns) != nil || turns < 0 {
					report.add(SeverityError, "invalid_op_turns", opBase+".turns", "turns phải là số nguyên không âm")
				}
			}

			if chanceRaw, ok := op["chance"]; ok && len(chanceRaw) > 0 {
				var chance float64
				if json.Unmarshal(chanceRaw, &chance) != nil || chance < 0 || chance > 1 {
					report.add(SeverityError, "invalid_op_chance", opBase+".chance", "chance phải trong khoảng 0..1")
				}
			}
		}
	}
}
