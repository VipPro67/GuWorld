package effectcatalog

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestManifestDefinitionsHaveOwnersAndDispositions(t *testing.T) {
	for _, definition := range Definitions() {
		if definition.Owner == "" {
			t.Errorf("%s.%s has no owner", definition.Section, definition.Key)
		}
		if !definition.Disposition.Valid() {
			t.Errorf("%s.%s has invalid disposition %q", definition.Section, definition.Key, definition.Disposition)
		}
	}
	for _, definition := range MechanicDefinitions() {
		if definition.Owner == "" || !definition.Disposition.Valid() {
			t.Errorf("%s/%s has invalid ownership: %+v", definition.Trigger, definition.ActionCode, definition)
		}
	}
	for _, definition := range StatusDefinitions() {
		if definition.Owner == "" || !definition.Disposition.Valid() || definition.Key == "" {
			t.Errorf("status has invalid ownership: %+v", definition)
		}
	}
	for section, fields := range runtimeFields {
		for key := range fields {
			if _, ok := DefinitionFor(section, key); !ok {
				t.Errorf("typed field %s.%s is missing from manifest", section, key)
			}
		}
	}
}

func TestValidateRejectsUnclassifiedEffectKey(t *testing.T) {
	_, report, err := NormalizeAndValidate([]byte(`{"active":{"brand_new_typo":1}}`))
	if err == nil || !report.HasErrors() {
		t.Fatalf("unclassified key accepted: %+v", report)
	}
	if report.Issues[0].Code != "unclassified_effect_key" {
		t.Fatalf("issue = %+v", report.Issues[0])
	}
}

func TestValidateGuCatalogAggregatesCoverageAndIssues(t *testing.T) {
	entries := []GuEntry{
		{ID: "GU_OK", Rank: 1, Effects: json.RawMessage(`{"active":{"damage_flat":10}}`)},
		{ID: "GU_MIGRATE", Rank: 2, Effects: json.RawMessage(`{"support":{"heal_flat":5}}`)},
		{ID: "GU_BAD", Rank: 3, Effects: json.RawMessage(`{"active":{"typo":1}}`)},
	}

	report := ValidateGuCatalog(entries)
	if report.Summary.Entries != 3 || report.Summary.WithErrors != 1 || report.Summary.WithWarnings != 1 {
		t.Fatalf("summary = %+v", report.Summary)
	}
	if report.Summary.Unclassified != 1 {
		t.Fatalf("unclassified = %d, want 1", report.Summary.Unclassified)
	}
	if len(report.Coverage) != 3 {
		t.Fatalf("coverage = %+v", report.Coverage)
	}
	if report.Coverage[0].Section != "active" || report.Coverage[0].Key != "damage_flat" || report.Coverage[0].Count != 1 {
		t.Fatalf("coverage not deterministic: %+v", report.Coverage)
	}
}

func TestValidateGuCatalogReportsDuplicateIdentity(t *testing.T) {
	entry := GuEntry{ID: "GU_DUP", Rank: 1, Effects: json.RawMessage(`{}`)}
	report := ValidateGuCatalog([]GuEntry{entry, entry})
	if !report.HasErrors() || report.Summary.WithErrors != 1 {
		t.Fatalf("duplicate was not rejected: %+v", report)
	}
}

func TestValidateGuCatalogCapsExpandedIssues(t *testing.T) {
	entries := make([]GuEntry, maxCatalogIssues+1)
	for i := range entries {
		entries[i] = GuEntry{ID: fmt.Sprintf("GU_BAD_%05d", i), Rank: 1, Effects: json.RawMessage(`{"active":{"typo":1}}`)}
	}
	report := ValidateGuCatalog(entries)
	if !report.IssuesTruncated || len(report.Issues) != maxCatalogIssues {
		t.Fatalf("issue cap not applied: len=%d truncated=%v", len(report.Issues), report.IssuesTruncated)
	}
	if report.Summary.Errors != maxCatalogIssues+1 {
		t.Fatalf("summary errors = %d, want %d", report.Summary.Errors, maxCatalogIssues+1)
	}
}
