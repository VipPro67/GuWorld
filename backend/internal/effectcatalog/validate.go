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

var Sections = []string{"active", "support", "passive", "house", "explore", "refinement", "growth", "other"}

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
	),
	"passive": set(
		"stat_base_attack", "stat_base_defense", "stat_base_hp", "stat_base_speed", "path_conflict_reduction", "damage_multiplier_on_hit",
		"soul_refinement", "storage_capacity", "movement", "investigation", "hearing", "evasion", "refine_mana_tier", "cleansing_power",
		"luck", "success_rate_bonus", "time_speed_multiplier", "accuracy",
	),
	"growth": set("max_attack", "max_defense", "max_hp", "max_speed", "hours_to_max"),
}

var numberFields = set(
	"damage_flat", "heal_flat", "shield_flat", "status_tick_damage", "range", "mana_regen", "lifesteal_rate", "refine_mana_tier",
	"damage_mult", "shield_mult", "cleansing_power", "capture_bonus", "summon_count_per_tick", "phantom_damage_scale", "ignore_defense", "ignore_defense_pct",
	"crit_rate_boost", "all_stats_buff_percent", "speed_boost_flat", "freeze", "slow", "slow1", "weak", "stun", "silence", "bleed",
	"poison", "thorns", "regen", "atk_up", "def_up", "taunt", "stat_base_attack", "stat_base_defense", "stat_base_hp",
	"stat_base_speed", "path_conflict_reduction", "damage_multiplier_on_hit", "soul_refinement", "storage_capacity", "movement",
	"investigation", "hearing", "evasion", "luck", "success_rate_bonus", "time_speed_multiplier", "accuracy", "max_attack", "max_defense",
	"max_hp", "max_speed", "hours_to_max",
	"absorb_damage_flat", "convert_damage_to_heal_percent", "decoy_count", "invulnerable_ticks", "evasion", "stat_base_luck",
	"dao_mark_multiplier", "kill_path_damage_buff", "luck_flat", "storage_slots", "kill_path_dao_marks_mult",
)
var integerFields = set("status_duration_ticks", "decoy_count", "invulnerable_ticks", "storage_slots")
var booleanFields = set(
	"curve_attack", "wood_transformation", "blood_drain", "replicate", "summon_phantom", "generate_gu", "portal_create", "random_portal",
	"summon_growth_phantoms", "reveal_all_stealth", "reveal_weakness",
)
var stringFields = set("apply_status", "target_selector", "summon_monster_id")
var ratioFields = set("lifesteal_rate", "ignore_defense_pct", "crit_rate_boost", "all_stats_buff_percent", "convert_damage_to_heal_percent", "kill_path_damage_buff")
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
			if section == "other" {
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
			var status string
			if err := json.Unmarshal(value, &status); err != nil || strings.TrimSpace(status) == "" {
				report.add(SeverityError, "invalid_status", path, "apply_status phải là chuỗi không rỗng")
			} else {
				upper := strings.ToUpper(strings.TrimSpace(status))
				if status != upper {
					report.add(SeverityError, "noncanonical_status", path, "status phải viết hoa canonical")
				}
				definition, ok := StatusDefinitionFor(upper)
				if !ok {
					report.add(SeverityError, "unknown_status", path, "status chưa có owner và disposition trong manifest")
				} else if definition.Disposition != DispositionSupported {
					report.add(SeverityWarning, "status_"+string(definition.Disposition), path, "owner="+definition.Owner)
				} else if _, ok := statusfx.Spec(status); !ok {
					report.add(SeverityError, "status_owner_mismatch", path, "status được đánh dấu supported nhưng statusfx không nhận diện")
				}
			}
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
	switch action {
	case "REBORN":
		requireRange("chance", 0, 1)
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
