package effectcatalog

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const CatalogReportSchemaVersion = 1

const (
	MaxEffectDefinitionBytes = 256 << 10
	maxCatalogIssues         = 10_000
)

type GuEntry struct {
	ID      string          `json:"id"`
	Rank    int             `json:"rank"`
	Effects json.RawMessage `json:"effects"`
}

type EntryRef struct {
	ID   string `json:"id"`
	Rank int    `json:"rank"`
}

type CatalogIssue struct {
	Entry EntryRef `json:"entry"`
	Issue Issue    `json:"issue"`
}

type Coverage struct {
	Section     string      `json:"section"`
	Key         string      `json:"key"`
	Disposition Disposition `json:"disposition"`
	Owner       string      `json:"owner"`
	Count       int         `json:"count"`
}

type CatalogSummary struct {
	Entries      int `json:"entries"`
	Valid        int `json:"valid"`
	WithErrors   int `json:"with_errors"`
	WithWarnings int `json:"with_warnings"`
	Errors       int `json:"errors"`
	Warnings     int `json:"warnings"`
	Unclassified int `json:"unclassified"`
}

type CatalogReport struct {
	SchemaVersion   int            `json:"schema_version"`
	Summary         CatalogSummary `json:"summary"`
	Issues          []CatalogIssue `json:"issues"`
	Coverage        []Coverage     `json:"coverage"`
	IssuesTruncated bool           `json:"issues_truncated"`
}

func (r CatalogReport) HasErrors() bool { return r.Summary.Errors > 0 }

// ValidateGuCatalog validates all definitions without failing fast and builds
// a machine-readable inventory proving that every observed key has an explicit
// owner and disposition.
func ValidateGuCatalog(entries []GuEntry) CatalogReport {
	report := CatalogReport{SchemaVersion: CatalogReportSchemaVersion}
	report.Summary.Entries = len(entries)
	coverage := map[string]*Coverage{}
	seen := map[string]bool{}
	orderedEntries := append([]GuEntry(nil), entries...)
	sort.SliceStable(orderedEntries, func(i, j int) bool {
		if orderedEntries[i].ID == orderedEntries[j].ID {
			return orderedEntries[i].Rank < orderedEntries[j].Rank
		}
		return orderedEntries[i].ID < orderedEntries[j].ID
	})

	for _, entry := range orderedEntries {
		ref := EntryRef{ID: entry.ID, Rank: entry.Rank}
		identity := fmt.Sprintf("%s\x00%d", entry.ID, entry.Rank)
		duplicate := seen[identity]
		seen[identity] = true

		var validation Report
		var err error
		if len(entry.Effects) > MaxEffectDefinitionBytes {
			validation.add(SeverityError, "effect_too_large", "$", "một effect definition vượt giới hạn 256 KiB")
			err = validation
		} else {
			_, validation, err = NormalizeAndValidate(entry.Effects)
		}
		if duplicate {
			validation.add(SeverityError, "duplicate_entry", "$", "trùng định danh Gu (id, rank)")
			validation.sort()
			err = validation
		}

		hasWarning := false
		for _, issue := range validation.Issues {
			if len(report.Issues) < maxCatalogIssues {
				report.Issues = append(report.Issues, CatalogIssue{Entry: ref, Issue: issue})
			} else {
				report.IssuesTruncated = true
			}
			if issue.Severity == SeverityError {
				report.Summary.Errors++
				if strings.HasPrefix(issue.Code, "unclassified_") {
					report.Summary.Unclassified++
				}
			} else {
				report.Summary.Warnings++
				hasWarning = true
			}
		}
		if err != nil {
			report.Summary.WithErrors++
		} else {
			report.Summary.Valid++
		}
		if hasWarning {
			report.Summary.WithWarnings++
		}
		collectCoverage(entry.Effects, coverage)
	}

	for _, item := range coverage {
		report.Coverage = append(report.Coverage, *item)
	}
	sort.Slice(report.Coverage, func(i, j int) bool {
		if report.Coverage[i].Section == report.Coverage[j].Section {
			return report.Coverage[i].Key < report.Coverage[j].Key
		}
		return report.Coverage[i].Section < report.Coverage[j].Section
	})
	sort.SliceStable(report.Issues, func(i, j int) bool {
		a, b := report.Issues[i], report.Issues[j]
		if a.Entry.ID != b.Entry.ID {
			return a.Entry.ID < b.Entry.ID
		}
		if a.Entry.Rank != b.Entry.Rank {
			return a.Entry.Rank < b.Entry.Rank
		}
		if a.Issue.Path != b.Issue.Path {
			return a.Issue.Path < b.Issue.Path
		}
		return a.Issue.Code < b.Issue.Code
	})
	return report
}

func collectCoverage(raw json.RawMessage, coverage map[string]*Coverage) {
	var root map[string]json.RawMessage
	if json.Unmarshal(raw, &root) != nil || root == nil {
		return
	}
	hasSection := false
	for section := range root {
		if sectionSet[section] {
			hasSection = true
			break
		}
	}
	if !hasSection {
		collectObjectCoverage("active", root, coverage)
		return
	}
	for section, sectionRaw := range root {
		if section == "other" {
			collectMechanicCoverage(sectionRaw, coverage)
			continue
		}
		var values map[string]json.RawMessage
		if json.Unmarshal(sectionRaw, &values) == nil {
			collectObjectCoverage(section, values, coverage)
		}
	}
}

func collectObjectCoverage(section string, values map[string]json.RawMessage, coverage map[string]*Coverage) {
	for key, raw := range values {
		definition, ok := DefinitionFor(section, key)
		item := Coverage{Section: section, Key: key}
		if ok {
			item.Disposition = definition.Disposition
			item.Owner = definition.Owner
		}
		incrementCoverage(coverage, item)
		if section == "active" && key == "apply_status" {
			var status string
			if json.Unmarshal(raw, &status) == nil {
				status = strings.ToUpper(strings.TrimSpace(status))
				statusItem := Coverage{Section: "status", Key: status}
				if definition, ok := StatusDefinitionFor(status); ok {
					statusItem.Disposition = definition.Disposition
					statusItem.Owner = definition.Owner
				}
				incrementCoverage(coverage, statusItem)
			}
		}
	}
}

func collectMechanicCoverage(raw json.RawMessage, coverage map[string]*Coverage) {
	var mechanics []map[string]json.RawMessage
	if json.Unmarshal(raw, &mechanics) != nil {
		return
	}
	for _, mechanic := range mechanics {
		var trigger, action string
		_ = json.Unmarshal(mechanic["event_trigger"], &trigger)
		_ = json.Unmarshal(mechanic["action_code"], &action)
		key := strings.ToUpper(strings.TrimSpace(trigger)) + ":" + strings.ToUpper(strings.TrimSpace(action))
		definition, ok := MechanicDefinitionFor(strings.ToUpper(strings.TrimSpace(trigger)), strings.ToUpper(strings.TrimSpace(action)))
		item := Coverage{Section: "other", Key: key}
		if ok {
			item.Disposition = definition.Disposition
			item.Owner = definition.Owner
		}
		incrementCoverage(coverage, item)
	}
}

func incrementCoverage(coverage map[string]*Coverage, item Coverage) {
	identity := item.Section + "\x00" + item.Key
	if current, ok := coverage[identity]; ok {
		current.Count++
		return
	}
	item.Count = 1
	coverage[identity] = &item
}
