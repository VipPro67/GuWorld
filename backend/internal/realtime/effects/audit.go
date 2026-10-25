package effects

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

type rawBlock struct {
	ID         string            `json:"id"`
	Trigger    string            `json:"trigger"`
	Target     rawTarget         `json:"target"`
	Conditions []json.RawMessage `json:"conditions"`
	Operations []rawOperation    `json:"operations"`
}

type rawTarget struct {
	Selector string `json:"selector"`
}

type rawOperation struct {
	Op          string     `json:"op"`
	SubType     string     `json:"sub_type"`
	Stat        string     `json:"stat"`
	Resource    string     `json:"resource"`
	StatusID    string     `json:"status_id"`
	Code        string     `json:"code"`
	StackPolicy string     `json:"stack_policy"`
	Turns       int        `json:"turns"`
	Chance      float64    `json:"chance"`
	Target      *rawTarget `json:"target"`
}

var domainTriggers = map[string]bool{
	"CAPABILITY": true,
	"ON_CONSUME": true,
	"ON_USE":     true,
}

var reauthorTriggers = map[string]string{
	"ON_TURN_START":       "Replace turn start with ON_INTERVAL/ON_ENTER_COMBAT and explicit intervalMs.",
	"ON_BATTLE_START":     "Map to ON_ENTER_COMBAT with hysteresis; do not fire on every combat-boundary crossing.",
	"BEFORE_ENEMY_ATTACK": "Map to a pre-hit/telegraph event with deterministic priority.",
	"ON_DEATH":            "Resolve as ON_PRE_DEATH/ON_DEATH with an explicit priority order.",
	"BEFORE_HIT":          "Map to the hit-resolution pipeline; document before/after shield and crit.",
	"ON_TICK":             "Author an explicit intervalMs/periodMs; never infer a timer from turns.",
}

var supportedTriggers = map[string]bool{
	"ON_ACTION":      true,
	"ON_CAST":        true,
	"ON_HIT":         true,
	"ALWAYS_ON":      true,
	"PASSIVE":        true,
	"COMBAT_PASSIVE": true,
}

var knownOperations = map[string]bool{
	"DAMAGE": true, "SHIELD": true, "HEAL": true, "DIRECT_HP": true,
	"STAT_MOD": true, "STATUS": true, "BARRIER": true, "CLEANSE": true, "UNIQUE": true,
}

// Audit classifies every Gu entry and block. It is pure and deterministic; the
// caller supplies the entries and MVP selection.
func Audit(entries []GuEntry, mvp []EntryRef) Report {
	report := Report{SchemaVersion: SchemaVersion, MVP: append([]EntryRef(nil), mvp...)}
	mvpSet := make(map[string]bool, len(mvp))
	for _, ref := range mvp {
		mvpSet[refKey(ref.ID, ref.Rank)] = true
	}
	seenMVP := make(map[string]bool, len(mvp))
	triggers := map[string]int{}
	operations := map[string]int{}
	ordered := append([]GuEntry(nil), entries...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].ID != ordered[j].ID {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].Rank < ordered[j].Rank
	})

	for _, entry := range ordered {
		audit := auditEntry(entry)
		audit.MVP = mvpSet[refKey(entry.ID, entry.Rank)]
		if audit.MVP {
			seenMVP[refKey(entry.ID, entry.Rank)] = true
		}
		report.Summary.Entries++
		switch audit.Status {
		case StatusSupported:
			report.Summary.Supported++
		case StatusNeedsReauthor:
			report.Summary.NeedsReauthor++
		case StatusUnsupported:
			report.Summary.Unsupported++
		case StatusNotActive:
			report.Summary.NotActive++
		}
		for _, block := range audit.Blocks {
			report.Summary.Blocks++
			switch block.Status {
			case StatusSupported:
				report.Summary.BlocksSupported++
			case StatusNeedsReauthor:
				report.Summary.BlocksNeedsReauth++
			case StatusUnsupported:
				report.Summary.BlocksUnsupported++
			case StatusNotActive:
				report.Summary.BlocksNotActive++
			}
			triggers[block.Trigger]++
			for _, operation := range block.Operations {
				report.Summary.Operations++
				operations[operation.Op]++
			}
		}
		report.Entries = append(report.Entries, audit)
	}
	for _, ref := range mvp {
		if !seenMVP[refKey(ref.ID, ref.Rank)] {
			report.MissingMVP = append(report.MissingMVP, ref)
		}
	}
	if len(report.MissingMVP) > 0 {
		report.Warnings = append(report.Warnings, "MVP references missing Gu entries")
	}
	report.Summary.Triggers = counts(triggers)
	report.Summary.OperationKinds = counts(operations)
	if !hasHiddenPassiveCandidate(ordered) {
		report.Warnings = append(report.Warnings,
			"no Gu block in Gu.csv uses conditions/stack counters; a hidden passive must be authored before PR-19 can ship one")
	}
	return report
}

func auditEntry(entry GuEntry) EntryAudit {
	audit := EntryAudit{
		ID: entry.ID, Rank: entry.Rank, Type: entry.Type, PrimaryRole: entry.PrimaryRole,
		Status: StatusNotActive,
	}
	blocks, err := decodeBlocks(entry.Effects)
	if err != nil {
		audit.Status = StatusUnsupported
		audit.Reasons = []string{err.Error()}
		return audit
	}
	if len(blocks) == 0 {
		audit.Reasons = []string{"no_effect_blocks"}
		return audit
	}
	for index, raw := range blocks {
		block := auditBlock(entry, index, raw)
		audit.Blocks = append(audit.Blocks, block)
		audit.Status = Worst(audit.Status, block.Status)
		audit.Reasons = appendUnique(audit.Reasons, block.Reasons...)
	}
	if audit.Status == StatusNotActive {
		audit.Reasons = appendUnique(audit.Reasons, "all_blocks_are_domain_or_world_only")
	}
	sort.Strings(audit.Reasons)
	return audit
}

func auditBlock(entry GuEntry, index int, raw json.RawMessage) BlockAudit {
	var block rawBlock
	blockAudit := BlockAudit{Status: StatusSupported}
	if err := json.Unmarshal(raw, &block); err != nil {
		blockAudit.Status = StatusUnsupported
		blockAudit.Reasons = []string{"invalid_block_json"}
		return blockAudit
	}
	blockAudit.ID = block.ID
	blockAudit.Trigger = strings.ToUpper(strings.TrimSpace(block.Trigger))
	blockAudit.Target = strings.ToUpper(strings.TrimSpace(block.Target.Selector))

	if issues := effectcatalog.ValidateBlockV2(singleBlockArray(raw), effectcatalog.Location{Source: "gu", Key: entry.ID}); len(issues) > 0 {
		blockAudit.Status = StatusUnsupported
		for _, issue := range issues {
			blockAudit.Reasons = append(blockAudit.Reasons, issue.Code+":"+issue.Path)
		}
		sort.Strings(blockAudit.Reasons)
		return blockAudit
	}

	switch {
	case domainTriggers[blockAudit.Trigger]:
		blockAudit.Status = StatusNotActive
		blockAudit.Reasons = append(blockAudit.Reasons, "domain_trigger_"+blockAudit.Trigger)
		blockAudit.Proposal = "Keep as domain/world action; it is not part of zone combat."
	case reauthorTriggers[blockAudit.Trigger] != "":
		blockAudit.Status = Worst(blockAudit.Status, StatusNeedsReauthor)
		blockAudit.Reasons = append(blockAudit.Reasons, "trigger_"+blockAudit.Trigger)
		blockAudit.Proposal = reauthorTriggers[blockAudit.Trigger]
	case supportedTriggers[blockAudit.Trigger]:
		// supported base
	default:
		blockAudit.Status = StatusUnsupported
		blockAudit.Reasons = append(blockAudit.Reasons, "unknown_trigger_"+blockAudit.Trigger)
	}

	if blockAudit.Status != StatusNotActive {
		switch blockAudit.Target {
		case "ROW":
			blockAudit.Status = Worst(blockAudit.Status, StatusNeedsReauthor)
			blockAudit.Reasons = append(blockAudit.Reasons, "formation_target_ROW")
			blockAudit.Proposal = "Replace ROW formation targeting with a bounded GROUND line/AoE and max-target cap."
		case "COL":
			blockAudit.Status = Worst(blockAudit.Status, StatusNeedsReauthor)
			blockAudit.Reasons = append(blockAudit.Reasons, "formation_target_COL")
			blockAudit.Proposal = "Replace COL formation targeting with DIRECTION/cone or a bound TARGET; never trust a client path."
		case "ALL", "ALL_ENEMIES", "ALL_ALLIES":
			if blockAudit.Proposal == "" {
				blockAudit.Proposal = "Resolve as GROUND/AoE with radius in world units and a deterministic target cap."
			}
		}
		if len(block.Conditions) > 0 {
			blockAudit.Status = Worst(blockAudit.Status, StatusNeedsReauthor)
			blockAudit.Reasons = append(blockAudit.Reasons, "conditions_need_realtime_authoring")
			blockAudit.Proposal = appendProposal(blockAudit.Proposal, "Author condition fields against tick/HP/stack counters with an internal cooldown.")
		}
	}

	for operationIndex, operation := range block.Operations {
		blockAudit.Operations = append(blockAudit.Operations, auditOperation(operationIndex, operation))
	}
	for _, operation := range blockAudit.Operations {
		blockAudit.Reasons = append(blockAudit.Reasons, operation.Reasons...)
		if blockAudit.Status == StatusNotActive {
			continue
		}
		blockAudit.Status = Worst(blockAudit.Status, operation.Status)
		blockAudit.Proposal = appendProposal(blockAudit.Proposal, operation.Proposal)
	}
	blockAudit.Reasons = dedupeSorted(blockAudit.Reasons)
	return blockAudit
}

func auditOperation(index int, operation rawOperation) OperationAudit {
	op := strings.ToUpper(strings.TrimSpace(operation.Op))
	audit := OperationAudit{Index: index, Op: op, SubType: strings.ToUpper(strings.TrimSpace(operation.SubType)), Status: StatusSupported}
	if !knownOperations[op] {
		audit.Status = StatusUnsupported
		audit.Reasons = []string{"unsupported_operation_" + op}
		return audit
	}
	if operation.Target != nil {
		switch strings.ToUpper(strings.TrimSpace(operation.Target.Selector)) {
		case "ROW", "COL":
			audit.Status = Worst(audit.Status, StatusNeedsReauthor)
			audit.Reasons = append(audit.Reasons, "formation_operation_target")
			audit.Proposal = "Replace formation target with TARGET/GROUND/DIRECTION at cast resolve."
		}
	}
	switch op {
	case "DAMAGE", "SHIELD", "HEAL", "DIRECT_HP", "CLEANSE":
		if op == "HEAL" && operation.Resource != "" && operation.Resource != "HP" && operation.Resource != "MANA" {
			audit.Status = Worst(audit.Status, StatusNeedsReauthor)
			audit.Reasons = append(audit.Reasons, "resource_"+operation.Resource)
			audit.Proposal = "Author realtime SOUL/THOUGHT/resource regeneration instead of a turn-based heal resource."
		}
	case "STAT_MOD":
		if operation.SubType == "GROWTH_CAP" {
			audit.Status = Worst(audit.Status, StatusNeedsReauthor)
			audit.Reasons = append(audit.Reasons, "growth_cap")
			audit.Proposal = "Apply long-term growth outside the zone tick; snapshot the resulting stats on enter."
		}
		if operation.Turns > 0 {
			audit.Status = Worst(audit.Status, StatusNeedsReauthor)
			audit.Reasons = append(audit.Reasons, "turns")
			audit.Proposal = "Author durationMs/stack policy; do not convert turns to a fixed millisecond default."
		}
	case "STATUS":
		audit.Status = Worst(audit.Status, StatusNeedsReauthor)
		audit.Reasons = append(audit.Reasons, "status_duration_stack")
		audit.Proposal = "Author durationMs, periodMs, max stacks, refresh/ICD and PvP diminishing returns for " + operation.StatusID + "."
	case "BARRIER":
		if operation.Turns > 0 || operation.StackPolicy != "" {
			audit.Status = Worst(audit.Status, StatusNeedsReauthor)
			audit.Reasons = append(audit.Reasons, "barrier_duration_stack")
			audit.Proposal = "Author barrier durationMs/stack policy and expiry ordering."
		}
	case "UNIQUE":
		audit.Status = Worst(audit.Status, StatusNeedsReauthor)
		audit.Reasons = append(audit.Reasons, "native_hook_"+operation.Code)
		audit.Proposal = "Implement a versioned Go native handler for " + operation.Code + " with deterministic tests; do not run scripts from data."
	}
	audit.Reasons = dedupeSorted(audit.Reasons)
	return audit
}

func decodeBlocks(raw json.RawMessage) ([]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	if trimmed[0] == '[' {
		var blocks []json.RawMessage
		if err := json.Unmarshal(trimmed, &blocks); err != nil {
			return nil, fmt.Errorf("invalid effect blocks: %w", err)
		}
		return blocks, nil
	}
	if trimmed[0] == '{' {
		var wrapper struct {
			Blocks []json.RawMessage `json:"blocks"`
		}
		if err := json.Unmarshal(trimmed, &wrapper); err != nil {
			return nil, fmt.Errorf("invalid effect wrapper: %w", err)
		}
		if wrapper.Blocks == nil {
			return nil, fmt.Errorf("legacy effect shape without blocks[] cannot be audited")
		}
		return wrapper.Blocks, nil
	}
	return nil, fmt.Errorf("invalid effect JSON shape")
}

func singleBlockArray(raw json.RawMessage) []byte {
	return append(append([]byte("["), raw...), ']')
}

func refKey(id string, rank int) string {
	return fmt.Sprintf("%s\x00%d", id, rank)
}

func counts(values map[string]int) []Count {
	result := make([]Count, 0, len(values))
	for key, count := range values {
		result = append(result, Count{Key: key, Count: count})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Key < result[j].Key })
	return result
}

func appendUnique(target []string, values ...string) []string {
	seen := make(map[string]bool, len(target))
	for _, value := range target {
		seen[value] = true
	}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		target = append(target, value)
		seen[value] = true
	}
	return target
}

func dedupeSorted(values []string) []string {
	values = appendUnique(nil, values...)
	sort.Strings(values)
	return values
}

func appendProposal(current, next string) string {
	if next == "" {
		return current
	}
	if current == "" {
		return next
	}
	if strings.Contains(current, next) {
		return current
	}
	return current + " " + next
}

func hasHiddenPassiveCandidate(entries []GuEntry) bool {
	for _, entry := range entries {
		blocks, err := decodeBlocks(entry.Effects)
		if err != nil {
			continue
		}
		for _, raw := range blocks {
			var block rawBlock
			if json.Unmarshal(raw, &block) != nil {
				continue
			}
			if len(block.Conditions) > 0 {
				return true
			}
		}
	}
	return false
}
