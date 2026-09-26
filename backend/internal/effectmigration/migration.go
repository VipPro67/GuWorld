// Package effectmigration converts legacy authoring formats into auditable
// block-v2 candidates. It is deliberately separate from runtime adapters:
// conversion must never silently invent semantics or mutate source data.
package effectmigration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type Format string

const (
	FormatGu       Format = "gu"
	FormatItem     Format = "item"
	FormatGameFX   Format = "gamefx"
	FormatKiller   Format = "killer_move"
	FormatAperture Format = "aperture_variant_event"
)

type Finding struct {
	Format           Format          `json:"format"`
	SourceID         string          `json:"source_id,omitempty"`
	Namespace        string          `json:"namespace,omitempty"`
	Key              string          `json:"key,omitempty"`
	Reason           string          `json:"reason,omitempty"`
	SuggestedMapping json.RawMessage `json:"suggested_mapping,omitempty"`
}

type Report struct {
	SchemaVersion                int       `json:"schema_version"`
	Phase                        string    `json:"phase"`
	Total                        int       `json:"total"`
	Converted                    []Finding `json:"converted"`
	ExactParity                  []Finding `json:"exact_parity"`
	Unsupported                  []Finding `json:"unsupported"`
	Ambiguous                    []Finding `json:"ambiguous"`
	Duplicate                    []Finding `json:"duplicate"`
	DivergentExplicitBlockLegacy []Finding `json:"divergent_explicit_block_legacy"`
	SuggestedMapping             []Finding `json:"suggested_mapping"`
	ShadowMismatchCount          int       `json:"shadow_mismatch_count"`
	AuditCoveragePercent         float64   `json:"audit_coverage_percent"`
	MigrationGate                bool      `json:"migration_gate"`
}

type Result struct {
	Envelope json.RawMessage `json:"envelope,omitempty"`
	Report   Report          `json:"report"`
}

type block struct {
	ID         string      `json:"id"`
	Trigger    string      `json:"trigger"`
	Target     target      `json:"target"`
	Operations []operation `json:"operations"`
}
type target struct {
	Selector        string `json:"selector"`
	BypassFrontline bool   `json:"bypass_frontline,omitempty"`
}
type operation struct {
	Op       string          `json:"op"`
	SubType  string          `json:"sub_type,omitempty"`
	Value    any             `json:"value,omitempty"`
	Stat     string          `json:"stat,omitempty"`
	Resource string          `json:"resource,omitempty"`
	StatusID string          `json:"status_id,omitempty"`
	Turns    any             `json:"turns,omitempty"`
	Chance   any             `json:"chance,omitempty"`
	Path     string          `json:"path,omitempty"`
	Code     string          `json:"code,omitempty"`
	Params   json.RawMessage `json:"params,omitempty"`
}

func Convert(format Format, sourceID string, raw []byte) Result {
	switch format {
	case FormatGu:
		return ConvertGu(sourceID, raw)
	case FormatItem:
		return ConvertItem(sourceID, raw)
	case FormatGameFX:
		return ConvertGameFX(sourceID, raw)
	case FormatKiller:
		return ConvertKillerMove(sourceID, raw)
	case FormatAperture:
		return ConvertApertureEvent(sourceID, raw)
	default:
		return finalize(nil, []Finding{{Format: format, SourceID: sourceID, Reason: "unknown format"}}, nil, nil, nil, nil)
	}
}

var guFlat = map[string]operation{
	"active.damage_flat": {Op: "DAMAGE", SubType: "FLAT"}, "active.damage_mult": {Op: "DAMAGE", SubType: "MULTIPLIER"},
	"active.heal_flat": {Op: "HEAL", SubType: "FLAT"}, "active.shield_flat": {Op: "SHIELD", SubType: "FLAT"}, "active.shield_mult": {Op: "SHIELD", SubType: "MULTIPLIER"},
	"active.damage_hp_percent": {Op: "DIRECT_HP", SubType: "MAX_HP_PERCENT"}, "active.speed_boost_flat": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "SPEED"},
	"active.crit_rate_boost": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "CRIT_RATE"}, "active.capture_bonus": {Op: "UNIQUE", Code: "CAPTURE_BONUS"},
	"active.portal_create": {Op: "UNIQUE", Code: "PORTAL_CREATE"}, "active.random_portal": {Op: "UNIQUE", Code: "RANDOM_PORTAL"},
	"passive.stat_base_attack": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "ATTACK"}, "passive.stat_base_defense": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "DEFENSE"},
	"passive.stat_base_hp": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "HP"}, "passive.stat_base_speed": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "SPEED"},
	"passive.movement": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "MOVEMENT"}, "passive.investigation": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "INVESTIGATION"},
	"passive.hearing": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "HEARING"}, "passive.disguise_level": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "DISGUISE"},
	"passive.time_speed_multiplier": {Op: "STAT_MOD", SubType: "MULTIPLY_PERCENT", Stat: "TIME_SPEED"}, "passive.success_rate_bonus": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "REFINE_SUCCESS"},
	"passive.refine_mana_tier": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "REFINE_MANA_TIER"}, "passive.storage_capacity": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "STORAGE"},
	"passive.tenacity": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "TENACITY"}, "passive.cc_resist": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "CC_RESIST"},
	"passive.dao_mark_multiplier": {Op: "STAT_MOD", SubType: "MULTIPLY_PERCENT", Stat: "DAO_MARK"}, "passive.path_conflict_reduction": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "PATH_CONFLICT_REDUCTION"},
	"growth.max_attack": {Op: "STAT_MOD", SubType: "GROWTH_CAP", Stat: "ATTACK"}, "growth.max_defense": {Op: "STAT_MOD", SubType: "GROWTH_CAP", Stat: "DEFENSE"},
	"growth.max_hp": {Op: "STAT_MOD", SubType: "GROWTH_CAP", Stat: "HP"}, "growth.max_speed": {Op: "STAT_MOD", SubType: "GROWTH_CAP", Stat: "SPEED"},
	"growth.hours_to_max": {Op: "UNIQUE", Code: "GROWTH_DURATION_HOURS"},
}

var ambiguousGu = map[string]string{
	"active.convert_damage_to_heal_percent": "requires damage-result ordering", "active.sleep": "status tier/chance is not explicit", "active.summon_monster_id": "summon ownership and lifetime are not explicit",
	"active.decoy_count": "decoy targeting semantics are not explicit", "active.reveal_weakness": "weakness payload is not explicit", "passive.damage_mult": "legacy consumer and unit differ",
	"passive.kill_path_damage_buff": "path and unit require an authored mapping", "passive.kill_path_dao_marks_mult": "legacy multiplier unit is ambiguous",
}

func ConvertGu(id string, raw []byte) Result {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return finalize(nil, []Finding{{Format: FormatGu, SourceID: id, Reason: err.Error()}}, nil, nil, nil, nil)
	}
	var blocks []block
	var converted, unsupported, ambiguous, duplicate, divergent []Finding
	seen := map[string]bool{}
	for _, ns := range []string{"active", "passive", "growth", "explore", "support", "refinement", "house"} {
		var fields map[string]json.RawMessage
		if len(root[ns]) == 0 || bytes.Equal(bytes.TrimSpace(root[ns]), []byte("null")) {
			continue
		}
		if json.Unmarshal(root[ns], &fields) != nil {
			unsupported = append(unsupported, Finding{Format: FormatGu, SourceID: id, Namespace: ns, Reason: "namespace is not an object"})
			continue
		}
		keys := sortedKeys(fields)
		for _, key := range keys {
			if emptyJSON(fields[key]) || key == "description" || key == "target_selector" {
				continue
			}
			fq := ns + "." + key
			f := Finding{Format: FormatGu, SourceID: id, Namespace: ns, Key: key}
			if reason, ok := ambiguousGu[fq]; ok {
				f.Reason = reason
				f.SuggestedMapping = suggest("manual", fq)
				ambiguous = append(ambiguous, f)
				continue
			}
			tmpl, ok := guFlat[fq]
			if !ok {
				f.Reason = "no exact semantic mapping"
				f.SuggestedMapping = suggest("native", strings.ToUpper(key))
				unsupported = append(unsupported, f)
				continue
			}
			bid := strings.ToUpper(ns + "_" + key)
			if seen[bid] {
				duplicate = append(duplicate, f)
				continue
			}
			seen[bid] = true
			tmpl.Value = decodeAny(fields[key])
			trigger := "ALWAYS_ON"
			if ns == "active" {
				trigger = "ON_ACTION"
			}
			if ns == "growth" {
				trigger = "ON_TICK"
			}
			blocks = append(blocks, block{ID: bid, Trigger: trigger, Target: target{Selector: "SELF"}, Operations: []operation{tmpl}})
			converted = append(converted, f)
		}
	}
	var mechanics []struct {
		EventTrigger string          `json:"event_trigger"`
		ActionCode   string          `json:"action_code"`
		Params       json.RawMessage `json:"params"`
	}
	_ = json.Unmarshal(root["other"], &mechanics)
	for _, m := range mechanics {
		f := Finding{Format: FormatGu, SourceID: id, Namespace: "other", Key: m.ActionCode}
		key := m.EventTrigger + "/" + m.ActionCode
		if seen[key] {
			duplicate = append(duplicate, f)
			continue
		}
		seen[key] = true
		blocks = append(blocks, block{ID: m.ActionCode, Trigger: m.EventTrigger, Target: target{Selector: "SELF"}, Operations: []operation{{Op: "UNIQUE", Code: m.ActionCode, Params: m.Params}}})
		converted = append(converted, f)
	}
	envelope, _ := json.Marshal(struct {
		SchemaVersion int     `json:"schema_version"`
		Blocks        []block `json:"blocks"`
	}{2, blocks})
	if explicit := root["blocks"]; len(explicit) > 0 && !emptyJSON(explicit) {
		var got []block
		if json.Unmarshal(explicit, &got) == nil {
			a, _ := json.Marshal(got)
			b, _ := json.Marshal(blocks)
			if !bytes.Equal(a, b) {
				divergent = append(divergent, Finding{Format: FormatGu, SourceID: id, Namespace: "blocks", Reason: "explicit blocks differ from deterministic legacy conversion"})
			}
		}
	}
	return finalize(envelope, unsupported, ambiguous, duplicate, divergent, converted)
}

var itemMappings = map[string]string{"restore_hp": "RESTORE_HP", "increase_human_qi": "INCREASE_HUMAN_QI", "heal_aperture": "HEAL_APERTURE", "random_constitution": "RANDOM_CONSTITUTION", "random_race": "RANDOM_RACE", "open_mine_chest": "OPEN_MINE_CHEST", "hunger": "RESTORE_HUNGER"}

func ConvertItem(id string, raw []byte) Result {
	return convertFlatNative(FormatItem, id, raw, "ON_CONSUME", itemMappings)
}

var gameFXMappings = map[string]string{"stat_mods": "STAT_MODS", "refine": "REFINE_SUCCESS", "dao_mark": "DAO_MARK_MODIFIER", "path_damage": "PATH_DAMAGE_MODIFIER", "path_resistance": "PATH_RESISTANCE_MODIFIER", "luck": "LUCK", "max_mana": "MAX_MANA", "mana_regen": "MANA_REGEN", "cultivation_efficiency": "CULTIVATION_EFFICIENCY", "dao_mark_gain": "DAO_MARK_GAIN", "healing_received": "HEALING_RECEIVED", "hp_regen": "HP_REGEN", "mental_resistance": "MENTAL_RESISTANCE", "soul_defense": "SOUL_DEFENSE", "soul_regen": "SOUL_REGEN", "tenacity": "TENACITY", "cc_resist": "CC_RESIST", "aperture_stress_bonus": "APERTURE_STRESS", "non_conflicting_dao_marks": "NON_CONFLICTING_DAO_MARKS", "immortal": "IMMORTAL", "disable_hp_regen": "DISABLE_HP_REGEN", "disable_mana_regen": "DISABLE_MANA_REGEN", "disable_resource_regen": "DISABLE_RESOURCE_REGEN", "disable_soul_regen": "DISABLE_SOUL_REGEN", "blocked_actions": "BLOCKED_ACTIONS"}

func ConvertGameFX(id string, raw []byte) Result {
	return convertFlatNative(FormatGameFX, id, raw, "ALWAYS_ON", gameFXMappings)
}
func ConvertKillerMove(id string, raw []byte) Result {
	return convertFlatNative(FormatKiller, id, raw, "ON_ACTION", map[string]string{"damage": "DAMAGE", "defense": "DEFENSE", "heal": "HEAL", "shield": "SHIELD", "lifesteal": "LIFESTEAL", "life_steal": "LIFESTEAL", "ignore_defense": "IGNORE_DEFENSE", "target_selector": "TARGET_SELECTOR", "curve_attack": "BYPASS_FRONTLINE"})
}
func ConvertApertureEvent(id string, raw []byte) Result {
	var x any
	if json.Unmarshal(raw, &x) != nil {
		return finalize(nil, []Finding{{Format: FormatAperture, SourceID: id, Reason: "invalid JSON"}}, nil, nil, nil, nil)
	}
	return finalize(nil, nil, []Finding{{Format: FormatAperture, SourceID: id, Reason: "no runtime semantics exist; manual mapping required", SuggestedMapping: suggest("manual", "author event contract")}}, nil, nil, nil)
}

func convertFlatNative(format Format, id string, raw []byte, trigger string, mappings map[string]string) Result {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return finalize(nil, []Finding{{Format: format, SourceID: id, Reason: err.Error()}}, nil, nil, nil, nil)
	}
	var blocks []block
	var converted, unsupported, ambiguous, duplicate []Finding
	seen := map[string]bool{}
	for _, key := range sortedKeys(fields) {
		if emptyJSON(fields[key]) {
			continue
		}
		f := Finding{Format: format, SourceID: id, Key: key}
		code, ok := mappings[key]
		if !ok {
			f.Reason = "no exact semantic mapping"
			f.SuggestedMapping = suggest("native", strings.ToUpper(key))
			unsupported = append(unsupported, f)
			continue
		}
		if seen[code] {
			f.Reason = "aliases resolve to the same mechanic"
			duplicate = append(duplicate, f)
			continue
		}
		seen[code] = true
		p, _ := json.Marshal(map[string]any{"value": decodeAny(fields[key])})
		blocks = append(blocks, block{ID: code, Trigger: trigger, Target: target{Selector: "SELF"}, Operations: []operation{{Op: "UNIQUE", Code: code, Params: p}}})
		converted = append(converted, f)
	}
	envelope, _ := json.Marshal(struct {
		SchemaVersion int     `json:"schema_version"`
		Blocks        []block `json:"blocks"`
	}{2, blocks})
	return finalize(envelope, unsupported, ambiguous, duplicate, nil, converted)
}

func finalize(envelope json.RawMessage, unsupported, ambiguous, duplicate, divergent, converted []Finding) Result {
	r := Report{SchemaVersion: 1, Phase: "audit", Unsupported: nz(unsupported), Ambiguous: nz(ambiguous), Duplicate: nz(duplicate), DivergentExplicitBlockLegacy: nz(divergent), Converted: nz(converted), ExactParity: []Finding{}, SuggestedMapping: []Finding{}}
	for _, group := range [][]Finding{unsupported, ambiguous} {
		for _, f := range group {
			if len(f.SuggestedMapping) > 0 {
				r.SuggestedMapping = append(r.SuggestedMapping, f)
			}
		}
	}
	r.Total = len(converted) + len(unsupported) + len(ambiguous) + len(duplicate)
	if r.Total > 0 {
		r.AuditCoveragePercent = 100 * float64(len(converted)+len(r.ExactParity)) / float64(r.Total)
	}
	r.MigrationGate = r.Total > 0 && len(r.ExactParity) == r.Total && len(unsupported) == 0 && len(ambiguous) == 0 && len(duplicate) == 0 && len(divergent) == 0 && r.ShadowMismatchCount == 0
	if r.MigrationGate {
		r.Phase = "migration"
	}
	return Result{Envelope: envelope, Report: r}
}
func sortedKeys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
func emptyJSON(v []byte) bool {
	s := strings.TrimSpace(string(v))
	return s == "" || s == "null" || s == `""`
}
func decodeAny(v []byte) any {
	var x any
	if json.Unmarshal(v, &x) != nil {
		return string(v)
	}
	return x
}
func suggest(kind, value string) json.RawMessage {
	b, _ := json.Marshal(map[string]string{"kind": kind, "value": value})
	return b
}
func nz[T any](v []T) []T {
	if v == nil {
		return []T{}
	}
	return v
}
func (f Format) Valid() bool {
	switch f {
	case FormatGu, FormatItem, FormatGameFX, FormatKiller, FormatAperture:
		return true
	}
	return false
}
func ParseFormat(v string) (Format, error) {
	f := Format(strings.ToLower(strings.TrimSpace(v)))
	if !f.Valid() {
		return "", fmt.Errorf("unsupported format %q", v)
	}
	return f, nil
}
