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
	var root map[string]json.RawMessage
	if err := decodeObject(raw, &root); err != nil {
		v.add("$", "invalid_json", err.Error(), "", nil)
		return v.issues
	}
	v.fields(root, "$", "schema_version", "blocks")
	var version int
	if b, ok := root["schema_version"]; !ok {
		v.add("$.schema_version", "required_field", "schema_version is required", "", nil)
	} else if json.Unmarshal(b, &version) != nil || version != BlockSchemaVersion {
		v.add("$.schema_version", "unsupported_schema_version", "schema_version must be 2", "", nil)
	}
	var blocks []map[string]json.RawMessage
	if b, ok := root["blocks"]; !ok {
		v.add("$.blocks", "required_field", "blocks is required", "", nil)
	} else if json.Unmarshal(b, &blocks) != nil || firstNonSpace(b) != '[' {
		v.add("$.blocks", "invalid_type", "blocks must be an array", "", nil)
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
	if raw, ok := b["conditions"]; ok {
		var cs []map[string]json.RawMessage
		if json.Unmarshal(raw, &cs) != nil {
			v.add(base+".conditions", "invalid_type", "conditions must be an array", id, nil)
		} else {
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
		v.operation(op, fmt.Sprintf("%s.operations[%d]", base, n), id, n, trigger, target)
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
		var x float64
		if json.Unmarshal(r, &x) != nil {
			v.add(path+".value", "invalid_type", "value must be a number", id, nil)
		}
	}
}

func (v *blockValidator) operation(op map[string]json.RawMessage, path, id string, index int, trigger, blockTarget string) {
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
	if raw, exists := op["target"]; exists {
		sel := v.target(raw, path+".target", id, &p, true)
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
	for _, field := range def.Required {
		if _, exists := op[field]; !exists {
			v.add(path+"."+field, "required_field", field+" is required", id, &p)
		}
	}
	if raw, exists := op["value"]; exists {
		var x float64
		if json.Unmarshal(raw, &x) != nil {
			v.add(path+".value", "invalid_type", "value must be a number", id, &p)
		} else if (def.NonNegative && x < 0) || (def.Ratio && (x < 0 || x > 1)) {
			v.add(path+".value", "invalid_range", "value is outside the allowed range", id, &p)
		}
	}
	for field, registry := range map[string]map[string]bool{"stat": StatCatalog, "resource": ResourceCatalog, "status_id": StatusCatalog, "unit": UnitCatalog, "stack_policy": StackPolicyCatalog} {
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
