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

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

type Format string

const (
	FormatGu       Format = "gu"
	FormatItem     Format = "item"
	FormatGameFX   Format = "gamefx"
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
	Validated                    bool      `json:"validated"`
	GoldenParity                 []Finding `json:"golden_parity"`
	DifferentialParity           []Finding `json:"differential_parity"`
	ExactParity                  []Finding `json:"exact_parity"`
	Unsupported                  []Finding `json:"unsupported"`
	Ambiguous                    []Finding `json:"ambiguous"`
	Duplicate                    []Finding `json:"duplicate"`
	DivergentExplicitBlockLegacy []Finding `json:"divergent_explicit_block_legacy"`
	SuggestedMapping             []Finding `json:"suggested_mapping"`
	ShadowCompared               bool      `json:"shadow_compared"`
	ShadowMismatchCount          int       `json:"shadow_mismatch_count"`
	ShadowParity                 bool      `json:"shadow_parity"`
	ShadowSnapshotID             string    `json:"shadow_snapshot_id,omitempty"`
	ShadowCatalogRevision        uint64    `json:"shadow_catalog_revision,omitempty"`
	AuditCoveragePercent         float64   `json:"audit_coverage_percent"`
	MigrationGate                bool      `json:"migration_gate"`
	ErrorCode                    string    `json:"error_code,omitempty"`
	ErrorReason                  string    `json:"error_reason,omitempty"`
}

// GateEvaluation is the stable decision shared by file and database migrations.
type GateEvaluation struct {
	MigrationGate bool   `json:"migration_gate"`
	ErrorCode     string `json:"error_code,omitempty"`
	Reason        string `json:"reason,omitempty"`
}

// EvaluateMigrationGate evaluates evidence, rather than treating successful
// serialization as semantic parity. GoldenParity and DifferentialParity must
// be populated by the corresponding fixture/differential runner.
func EvaluateMigrationGate(report Report) GateEvaluation {
	fail := func(code, reason string) GateEvaluation { return GateEvaluation{ErrorCode: code, Reason: reason} }
	switch {
	case len(report.Unsupported) > 0:
		return fail("unsupported_effect", "legacy payload contains unsupported effects")
	case len(report.Ambiguous) > 0:
		return fail("ambiguous_effect", "legacy payload contains ambiguous effects")
	case len(report.Duplicate) > 0:
		return fail("duplicate_effect", "legacy payload contains duplicate effects")
	case len(report.DivergentExplicitBlockLegacy) > 0:
		return fail("differential_divergence", "explicit blocks diverge from the legacy conversion")
	case !report.Validated:
		return fail("candidate_validation_failed", "candidate Block v2 envelope is invalid")
	case report.ShadowMismatchCount < 0:
		return fail("invalid_shadow_count", "shadow mismatch count cannot be negative")
	case report.ShadowMismatchCount > 0:
		return fail("shadow_mismatch", "shadow execution differs from the legacy runtime")
	}
	// A validated empty envelope has no semantics for a parity runner to compare.
	if report.Total == 0 {
		return GateEvaluation{MigrationGate: true}
	}
	if parityCoverage(report.Converted, report.ExactParity) < len(report.Converted) {
		return fail("missing_parity", "converted effects lack golden or differential parity evidence")
	}
	if !report.ShadowCompared {
		return fail("shadow_not_compared", "shadow execution has not been recorded")
	}
	if strings.TrimSpace(report.ShadowSnapshotID) == "" || report.ShadowCatalogRevision == 0 {
		return fail("shadow_evidence_unbound", "shadow evidence must identify the selected snapshot and catalog revision")
	}
	return GateEvaluation{MigrationGate: true}
}

func applyMigrationGate(report *Report) {
	// ExactParity has provenance: only actual golden/differential successes feed it.
	report.ExactParity = mergeParityEvidence(report.Converted, report.GoldenParity, report.DifferentialParity)
	report.ShadowParity = report.ShadowCompared && report.ShadowMismatchCount == 0 && strings.TrimSpace(report.ShadowSnapshotID) != "" && report.ShadowCatalogRevision > 0
	report.AuditCoveragePercent = 100
	if report.Total > 0 {
		report.AuditCoveragePercent = 100 * float64(len(report.ExactParity)) / float64(report.Total)
		if report.AuditCoveragePercent > 100 {
			report.AuditCoveragePercent = 100
		}
	}
	decision := EvaluateMigrationGate(*report)
	report.MigrationGate, report.ErrorCode, report.ErrorReason = decision.MigrationGate, decision.ErrorCode, decision.Reason
	if decision.MigrationGate {
		report.Phase = "migration"
	}
}

func findingIdentity(f Finding) string {
	return string(f.Format) + "\x00" + f.SourceID + "\x00" + f.Namespace + "\x00" + f.Key
}

func parityCoverage(converted, evidence []Finding) int {
	wanted := make(map[string]bool, len(converted))
	for _, finding := range converted {
		wanted[findingIdentity(finding)] = true
	}
	covered := map[string]bool{}
	for _, finding := range evidence {
		id := findingIdentity(finding)
		if wanted[id] {
			covered[id] = true
		}
	}
	return len(covered)
}

func mergeParityEvidence(converted []Finding, groups ...[]Finding) []Finding {
	wanted := make(map[string]bool, len(converted))
	for _, finding := range converted {
		wanted[findingIdentity(finding)] = true
	}
	seen := map[string]bool{}
	out := []Finding{}
	for _, group := range groups {
		for _, finding := range group {
			id := findingIdentity(finding)
			if wanted[id] && !seen[id] {
				seen[id] = true
				out = append(out, finding)
			}
		}
	}
	return out
}

// RecordGoldenParity records successful fixture comparisons and re-evaluates the gate.
func RecordGoldenParity(report *Report, findings []Finding) {
	report.GoldenParity = nz(findings)
	applyMigrationGate(report)
}

// RecordDifferentialParity records successful legacy-vs-candidate comparisons.
func RecordDifferentialParity(report *Report, findings []Finding) {
	report.DifferentialParity = nz(findings)
	applyMigrationGate(report)
}

// RecordShadowParity records that a shadow run completed and how many comparisons differed.
func RecordShadowParity(report *Report, mismatchCount int) {
	report.ShadowCompared = true
	report.ShadowMismatchCount = mismatchCount
	applyMigrationGate(report)
}

// RecordShadowParityFor binds runtime evidence to the immutable player/world
// snapshot and catalog revision selected for the migration decision.
func RecordShadowParityFor(report *Report, snapshotID string, catalogRevision uint64, mismatchCount int) {
	report.ShadowCompared = true
	report.ShadowMismatchCount = mismatchCount
	report.ShadowSnapshotID = strings.TrimSpace(snapshotID)
	report.ShadowCatalogRevision = catalogRevision
	applyMigrationGate(report)
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
	Op              string          `json:"op"`
	SubType         string          `json:"sub_type,omitempty"`
	Enabled         *bool           `json:"enabled,omitempty"`
	Target          *target         `json:"target,omitempty"`
	ApplicationMode string          `json:"application_mode,omitempty"`
	Value           any             `json:"value,omitempty"`
	Stat            string          `json:"stat,omitempty"`
	Resource        string          `json:"resource,omitempty"`
	StatusID        string          `json:"status_id,omitempty"`
	Turns           any             `json:"turns,omitempty"`
	Chance          any             `json:"chance,omitempty"`
	Path            string          `json:"path,omitempty"`
	Code            string          `json:"code,omitempty"`
	Unit            string          `json:"unit,omitempty"`
	Params          json.RawMessage `json:"params,omitempty"`
}

func Convert(format Format, sourceID string, raw []byte) Result {
	switch format {
	case FormatGu:
		return ConvertGu(sourceID, raw)
	case FormatItem:
		return ConvertItem(sourceID, raw)
	case FormatGameFX:
		return ConvertGameFX(sourceID, raw)
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
	"active.crit_rate_boost": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "CRIT_RATE"}, "active.capture_bonus": {Op: "GRANT_CAPABILITY", SubType: "CAPTURE_BONUS"},
	"active.portal_create": {Op: "GRANT_CAPABILITY", SubType: "PORTAL_CREATE"}, "active.random_portal": {Op: "GRANT_CAPABILITY", SubType: "RANDOM_PORTAL"},
	"passive.stat_base_attack": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "ATTACK"}, "passive.stat_base_defense": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "DEFENSE"},
	"passive.stat_base_hp": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "HP"}, "passive.stat_base_speed": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "SPEED"},
	"passive.movement": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "MOVEMENT"}, "passive.investigation": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "INVESTIGATION"},
	"passive.hearing": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "HEARING"}, "passive.disguise_level": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "DISGUISE"},
	"passive.time_speed_multiplier": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "TIME_SPEED", Unit: "RATIO"}, "passive.success_rate_bonus": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "REFINE_SUCCESS", Unit: "PERCENT"},
	"passive.refine_mana_tier": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "REFINE_MANA_TIER"}, "passive.storage_capacity": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "STORAGE"},
	"passive.luck":     {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "LUCK"},
	"passive.tenacity": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "TENACITY"}, "passive.cc_resist": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "CC_RESIST"},
	"passive.dao_mark_multiplier": {Op: "STAT_MOD", SubType: "MULTIPLY_PERCENT", Stat: "DAO_MARK", Unit: "RATIO"}, "passive.path_conflict_reduction": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "PATH_CONFLICT_REDUCTION", Unit: "PERCENT"},
	"passive.anti_heaven_damage_reduction_percent": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "PATH_RESISTANCE", Path: "heaven", Unit: "PERCENT"}, "passive.anti_earth_damage_reduction_percent": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "PATH_RESISTANCE", Path: "earth", Unit: "PERCENT"}, "passive.anti_dream_damage_reduction_percent": {Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "PATH_RESISTANCE", Path: "dream", Unit: "PERCENT"}, "passive.immune_dream_status": {Op: "SET_RULE", SubType: "IMMUNE_DREAM_STATUS"},
	"growth.max_attack": {Op: "STAT_MOD", SubType: "GROWTH_CAP", Stat: "ATTACK"}, "growth.max_defense": {Op: "STAT_MOD", SubType: "GROWTH_CAP", Stat: "DEFENSE"},
	"growth.max_hp": {Op: "STAT_MOD", SubType: "GROWTH_CAP", Stat: "HP"}, "growth.max_speed": {Op: "STAT_MOD", SubType: "GROWTH_CAP", Stat: "SPEED"},
	"growth.hours_to_max": {Op: "GROWTH_DURATION"},
}

var ambiguousGu = map[string]string{
	"active.convert_damage_to_heal_percent": "requires damage-result ordering", "active.sleep": "status tier/chance is not explicit", "active.summon_monster_id": "summon ownership and lifetime are not explicit",
	"active.decoy_count": "decoy targeting semantics are not explicit", "active.reveal_weakness": "weakness payload is not explicit", "passive.damage_mult": "legacy consumer and unit differ",
	"passive.kill_path_damage_buff": "path and unit require an authored mapping", "passive.kill_path_dao_marks_mult": "legacy multiplier unit is ambiguous",
}

func ConvertGu(id string, raw []byte) Result {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && (trimmed[0] == '[' || trimmed[0] == '{') {
		var root map[string]json.RawMessage
		isBlockV2 := trimmed[0] == '['
		if trimmed[0] == '{' && json.Unmarshal(trimmed, &root) == nil {
			_, hasVersion := root["schema_version"]
			_, hasBlocks := root["blocks"]
			legacyNamespace := false
			for _, namespace := range []string{"active", "passive", "growth", "explore", "support", "refinement", "house", "other"} {
				if _, ok := root[namespace]; ok {
					legacyNamespace = true
					break
				}
			}
			isBlockV2 = hasVersion || (hasBlocks && !legacyNamespace)
		}
		if isBlockV2 {
			canonical, issues := effectcatalog.CanonicalBlockV2Envelope(trimmed, effectcatalog.Location{Source: "gu", Key: id})
			if len(issues) > 0 {
				findings := make([]Finding, 0, len(issues))
				for _, issue := range issues {
					findings = append(findings, Finding{Format: FormatGu, SourceID: id, Namespace: "blocks", Key: issue.Path, Reason: issue.Code + ": " + issue.Message})
				}
				return finalize(nil, findings, nil, nil, nil, nil)
			}
			result := finalize(canonical, nil, nil, nil, nil, nil)
			result.Report.Total = 0 // already-authored v2 is validation, not legacy conversion
			result.Report.Validated = true
			applyMigrationGate(&result.Report)
			return result
		}
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return finalize(nil, []Finding{{Format: FormatGu, SourceID: id, Reason: err.Error()}}, nil, nil, nil, nil)
	}
	blocks := []block{}
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
			value := decodeAny(fields[key])
			if tmpl.Op == "UNIQUE" || tmpl.Op == "GRANT_CAPABILITY" {
				if (tmpl.SubType == "PORTAL_CREATE" || tmpl.SubType == "RANDOM_PORTAL" || tmpl.Code == "IMMUNE_DREAM_STATUS") && value == false {
					continue
				}
				if tmpl.SubType == "CAPTURE_BONUS" {
					tmpl.Params, _ = json.Marshal(map[string]any{"value": value})
				}
			} else if tmpl.Op == "SET_RULE" {
				enabled, ok := value.(bool)
				if !ok {
					f.Reason = "expected a boolean value"
					f.SuggestedMapping = suggest("manual", fq)
					unsupported = append(unsupported, f)
					continue
				}
				if !enabled {
					continue
				}
				tmpl.Enabled = &enabled
			} else {
				tmpl.Value = value
			}
			trigger := "ALWAYS_ON"
			if ns == "active" {
				trigger = "ON_ACTION"
			}
			if fq == "active.capture_bonus" || fq == "active.portal_create" || fq == "active.random_portal" {
				trigger = "CAPABILITY"
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
		if strings.EqualFold(m.EventTrigger, "ALWAYS_ON") && strings.EqualFold(strings.TrimSpace(m.ActionCode), "PATH_DAMAGE_MODIFIER") {
			var params struct {
				Value map[string]float64 `json:"value"`
			}
			if err := json.Unmarshal(m.Params, &params); err != nil || len(params.Value) == 0 {
				unsupported = append(unsupported, f)
				continue
			}
			paths := make([]string, 0, len(params.Value))
			for path := range params.Value {
				paths = append(paths, path)
			}
			sort.Strings(paths)
			operations := make([]operation, 0, len(paths))
			for _, path := range paths {
				operations = append(operations, operation{Op: "STAT_MOD", SubType: "ADD_FLAT", Stat: "PATH_DAMAGE", Path: strings.ToLower(path), Value: params.Value[path]})
			}
			blocks = append(blocks, block{ID: "PATH_DAMAGE_MODIFIER", Trigger: m.EventTrigger, Target: target{Selector: "SELF"}, Operations: operations})
			converted = append(converted, f)
			continue
		}
		if strings.EqualFold(m.EventTrigger, "ON_ACTION") && strings.EqualFold(strings.TrimSpace(m.ActionCode), "STAR_THOUGHT_RESTORE") {
			self := target{Selector: "SELF"}
			blocks = append(blocks, block{
				ID: m.ActionCode, Trigger: m.EventTrigger, Target: target{Selector: "TARGET"},
				Operations: []operation{{Op: "HEAL", SubType: "RESOURCE", Resource: "MANA", Unit: "PERCENT", Value: 30, Target: &self}},
			})
			converted = append(converted, f)
			continue
		}
		if statusID, value, turns, ok := legacyStatusOperation(m.ActionCode); ok &&
			(strings.EqualFold(m.EventTrigger, "ON_ACTION") || strings.EqualFold(m.EventTrigger, "ON_HIT")) {
			blocks = append(blocks, block{
				ID: m.ActionCode, Trigger: m.EventTrigger, Target: target{Selector: "TARGET"},
				Operations: []operation{{Op: "STATUS", StatusID: statusID, Value: value, Turns: turns, Chance: float64(0), ApplicationMode: "DIRECT"}},
			})
			converted = append(converted, f)
			continue
		}
		op := operation{Op: "UNIQUE", Code: m.ActionCode, Params: m.Params}
		resource, subtype, defaultValue := "", "", 1.0
		resourceOperation := ""
		if (strings.EqualFold(m.EventTrigger, "ON_BATTLE_START") || strings.EqualFold(m.EventTrigger, "ON_CAST")) && strings.EqualFold(strings.TrimSpace(m.ActionCode), "SUMMON_MONSTER") {
			op = operation{Op: "SUMMON", Params: m.Params}
		}
		if strings.EqualFold(m.EventTrigger, "ON_TICK") {
			switch strings.ToUpper(strings.TrimSpace(m.ActionCode)) {
			case "MANA_REGEN_TICK":
				resource, subtype, defaultValue = "MANA", "REGEN_PER_HOUR", 0.5
			case "THOUGHT_REGEN_TICK":
				resource, subtype = "THOUGHT", "REGEN_PER_HOUR"
			case "SOUL_REGEN_TICK":
				resource, subtype = "SOUL", "REGEN_PER_HOUR"
			case "GENERATE_STONES_TICK":
				resource, subtype = "PRIMEVAL_STONE", "GENERATION_WORKER_TICK"
			case "GENERATE_IMMORTAL_STONES_TICK":
				resource, subtype = "IMMORTAL_STONE", "GENERATION_WORKER_TICK"
			}
		}
		if strings.EqualFold(m.EventTrigger, "ON_CONSUME") {
			switch strings.ToUpper(strings.TrimSpace(m.ActionCode)) {
			case "MAN_BEAST_BURIAL":
				op = operation{Op: "ADVANCE_MAJOR_REALM"}
			case "RANK_PROGRESS":
				op = operation{Op: "ADVANCE_PROGRESSION"}
			case "STONE_APERTURE", "APERTURE_PETRIFICATION":
				op = operation{Op: "PETRIFY_APERTURE"}
			case "REDUCE_STRESS", "INCREASE_TIME_SPEED", "DECREASE_TIME_SPEED", "THIRD_WATCH", "TIME_ACCELERATION":
				value := 0.1
				var params map[string]float64
				if json.Unmarshal(m.Params, &params) == nil && params["value"] > 0 {
					value = params["value"]
				} else if strings.EqualFold(m.ActionCode, "REDUCE_STRESS") {
					value = 5
				} else if strings.EqualFold(m.ActionCode, "THIRD_WATCH") || strings.EqualFold(m.ActionCode, "TIME_ACCELERATION") {
					value = 3
				}
				subType := map[string]string{
					"REDUCE_STRESS":       "STRESS_REDUCE",
					"INCREASE_TIME_SPEED": "TIME_RATE_ADD",
					"DECREASE_TIME_SPEED": "TIME_RATE_SUBTRACT",
					"THIRD_WATCH":         "TIME_RATE_MULTIPLY",
					"TIME_ACCELERATION":   "TIME_RATE_MULTIPLY",
				}[strings.ToUpper(strings.TrimSpace(m.ActionCode))]
				op = operation{Op: "MODIFY_APERTURE", SubType: subType, Value: value}
			case "ADD_LIFESPAN_YEARS":
				resource, resourceOperation = "LIFESPAN", "GRANT_RESOURCE"
			case "ADD_SOUL_MAX":
				resource, resourceOperation = "SOUL", "MODIFY_MAX_RESOURCE"
			case "ADD_THOUGHT_MAX":
				resource, resourceOperation = "THOUGHT", "MODIFY_MAX_RESOURCE"
			case "RESTORE_SOUL_HP":
				resource, resourceOperation = "SOUL", "RESTORE_RESOURCE"
			case "RESTORE_MENTAL":
				resource, resourceOperation = "THOUGHT", "RESTORE_RESOURCE"
			}
		}
		if resource != "" {
			value := defaultValue
			var params map[string]float64
			if json.Unmarshal(m.Params, &params) == nil && params["value"] > 0 {
				value = params["value"]
			} else if resourceOperation != "" {
				resource = ""
			}
			if resource != "" && resourceOperation != "" {
				op = operation{Op: resourceOperation, Resource: resource, Value: value}
			} else if resource != "" {
				op = operation{Op: "PERIODIC_RESOURCE", SubType: subtype, Resource: resource, Value: value}
			}
		}
		blocks = append(blocks, block{ID: m.ActionCode, Trigger: m.EventTrigger, Target: target{Selector: "SELF"}, Operations: []operation{op}})
		converted = append(converted, f)
	}
	envelope, _ := json.Marshal(struct {
		SchemaVersion int     `json:"schema_version"`
		Blocks        []block `json:"blocks"`
	}{2, blocks})
	differentialProven := false
	if explicit := root["blocks"]; len(explicit) > 0 && !emptyJSON(explicit) {
		var got []block
		if err := json.Unmarshal(explicit, &got); err != nil {
			unsupported = append(unsupported, Finding{Format: FormatGu, SourceID: id, Namespace: "blocks", Reason: "explicit blocks must be an array"})
		} else {
			a, _ := json.Marshal(got)
			b, _ := json.Marshal(blocks)
			if !bytes.Equal(a, b) {
				divergent = append(divergent, Finding{Format: FormatGu, SourceID: id, Namespace: "blocks", Reason: "explicit blocks differ from deterministic legacy conversion"})
			} else {
				differentialProven = true
			}
		}
	}
	result := finalize(envelope, unsupported, ambiguous, duplicate, divergent, converted)
	if differentialProven {
		RecordDifferentialParity(&result.Report, converted)
	}
	return result
}

func legacyStatusOperation(code string) (statusID string, value float64, turns int, ok bool) {
	switch strings.ToUpper(strings.TrimSpace(code)) {
	case "WOOD_BIND":
		return "BIND", 0, 2, true
	case "ICE_DEEP_FREEZE":
		return "FREEZE", 0, 1, true
	case "LIGHT_BLINDING_FLASH":
		return "BLIND", 60, 2, true
	case "SOUND_DISRUPT_SILENCE":
		return "SILENCE", 0, 2, true
	case "SOUND_RESONANCE":
		return "SOUL_WOUND", 25, 3, true
	case "ICE_FROSTBITE":
		return "SLOW_DAMAGE", 35, 3, true
	case "CHARM_GHOST_COME":
		return "CHARM", 0, 2, true
	case "FIVE_RESTRICTION_DEEP_LIGHT":
		return "GU_SEAL", 0, 3, true
	default:
		return "", 0, 0, false
	}
}

var itemMappings = map[string]operation{
	"restore_hp":                  {Op: "HEAL", SubType: "FLAT", Resource: "HP"},
	"restore_mana":                {Op: "HEAL", SubType: "FLAT", Resource: "MANA"},
	"increase_human_qi":           {Op: "GRANT_RESOURCE", Resource: "HUMAN_QI"},
	"restore_earth_qi":            {Op: "RESTORE_RESOURCE", Resource: "EARTH_QI"},
	"restore_heaven_qi":           {Op: "RESTORE_RESOURCE", Resource: "HEAVEN_QI"},
	"heal_aperture":               {Op: "REPAIR_APERTURE"},
	"increase_aperture_area":      {Op: "MODIFY_APERTURE", SubType: "AREA_SIZE"},
	"increase_aperture_time_rate": {Op: "MODIFY_APERTURE", SubType: "TIME_RATE"},
	"random_constitution":         {Op: "RANDOMIZE_TRAIT", SubType: "CONSTITUTION"},
	"random_race":                 {Op: "RANDOMIZE_TRAIT", SubType: "RACE"},
	"open_mine_chest":             {Op: "OPEN_REWARD_CONTAINER", SubType: "MORTAL_MINE"},
	"open_immortal_mine_chest":    {Op: "OPEN_REWARD_CONTAINER", SubType: "IMMORTAL_MINE"},
	"spring_egg_hatch":            {Op: "PLACE_WORLD_OBJECT", SubType: "PRIMEVAL_SPRING"},
}

func ConvertItem(id string, raw []byte) Result {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return finalize(nil, []Finding{{Format: FormatItem, SourceID: id, Reason: err.Error()}}, nil, nil, nil, nil)
	}
	blocks := []block{}
	var converted, unsupported []Finding
	for _, key := range sortedKeys(fields) {
		if emptyJSON(fields[key]) {
			continue
		}
		finding := Finding{Format: FormatItem, SourceID: id, Key: key}
		template, ok := itemMappings[key]
		if !ok {
			finding.Reason = "no exact typed item operation mapping"
			finding.SuggestedMapping = suggest("manual", "typed item operation")
			unsupported = append(unsupported, finding)
			continue
		}
		op := template
		if op.Op == "HEAL" || op.Op == "GRANT_RESOURCE" || op.Op == "RESTORE_RESOURCE" || op.Op == "REPAIR_APERTURE" || op.Op == "MODIFY_APERTURE" {
			op.Value = decodeAny(fields[key])
		}
		blockID := strings.ToUpper(key)
		blocks = append(blocks, block{ID: blockID, Trigger: "ON_CONSUME", Target: target{Selector: "SELF"}, Operations: []operation{op}})
		converted = append(converted, finding)
	}
	envelope, _ := json.Marshal(struct {
		SchemaVersion int     `json:"schema_version"`
		Blocks        []block `json:"blocks"`
	}{2, blocks})
	return finalize(envelope, unsupported, nil, nil, nil, converted)
}

var gameFXMappings = map[string]string{"stat_mods": "STAT_MODS", "refine": "REFINE_SUCCESS", "dao_mark": "DAO_MARK_MODIFIER", "path_damage": "PATH_DAMAGE_MODIFIER", "path_resistance": "PATH_RESISTANCE_MODIFIER", "luck": "LUCK", "max_mana": "MAX_MANA", "mana_regen": "MANA_REGEN", "cultivation_efficiency": "CULTIVATION_EFFICIENCY", "dao_mark_gain": "DAO_MARK_GAIN", "healing_received": "HEALING_RECEIVED", "hp_regen": "HP_REGEN", "mental_resistance": "MENTAL_RESISTANCE", "soul_defense": "SOUL_DEFENSE", "soul_regen": "SOUL_REGEN", "tenacity": "TENACITY", "cc_resist": "CC_RESIST", "aperture_stress_bonus": "APERTURE_STRESS", "non_conflicting_dao_marks": "NON_CONFLICTING_DAO_MARKS", "immortal": "IMMORTAL", "disable_hp_regen": "DISABLE_HP_REGEN", "disable_mana_regen": "DISABLE_MANA_REGEN", "disable_resource_regen": "DISABLE_RESOURCE_REGEN", "disable_soul_regen": "DISABLE_SOUL_REGEN"}

func ConvertGameFX(id string, raw []byte) Result {
	return convertFlatNative(FormatGameFX, id, raw, "ALWAYS_ON", gameFXMappings)
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
	blocks := []block{}
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
	r := Report{SchemaVersion: 1, Phase: "audit", Unsupported: nz(unsupported), Ambiguous: nz(ambiguous), Duplicate: nz(duplicate), DivergentExplicitBlockLegacy: nz(divergent), Converted: nz(converted), GoldenParity: []Finding{}, DifferentialParity: []Finding{}, ExactParity: []Finding{}, SuggestedMapping: []Finding{}}
	for _, group := range [][]Finding{unsupported, ambiguous} {
		for _, f := range group {
			if len(f.SuggestedMapping) > 0 {
				r.SuggestedMapping = append(r.SuggestedMapping, f)
			}
		}
	}
	r.Total = len(converted) + len(unsupported) + len(ambiguous) + len(duplicate)
	r.Validated = len(envelope) > 0 && len(effectcatalog.ValidateBlockV2(envelope, effectcatalog.Location{Source: "migration_candidate"})) == 0
	applyMigrationGate(&r)
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
	case FormatGu, FormatItem, FormatGameFX, FormatAperture:
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
