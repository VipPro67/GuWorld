package effects

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func entry(id string, rank int, effects string) GuEntry {
	return GuEntry{ID: id, Rank: rank, Type: "Active", PrimaryRole: "ATTACK", Effects: json.RawMessage(effects)}
}

func TestAuditClassifiesSupportedAndReauthoredMechanics(t *testing.T) {
	entries := []GuEntry{
		entry("GU_SIMPLE_DAMAGE", 1, `[{"id":"ACTIVE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10}]}]`),
		entry("GU_FORMATION", 1, `[{"id":"ACTIVE","trigger":"ON_ACTION","target":{"selector":"COL"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10}]}]`),
		entry("GU_FREEZE", 2, `[{"id":"ACTIVE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"STATUS","status_id":"FREEZE","turns":1,"chance":1}]}]`),
		entry("GU_NATIVE", 3, `[{"id":"PASSIVE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"GROWTH_DURATION_HOURS","params":{"value":30}}]}]`),
		entry("GU_WORLD", 4, `[{"id":"CAP","trigger":"CAPABILITY","target":{"selector":"SELF"},"operations":[{"op":"UNIQUE","code":"LOOT_PREVIEW","params":{}}]}]`),
		entry("GU_NO_BLOCKS", 1, `[]`),
	}
	report := Audit(entries, nil)
	want := map[string]Status{
		"GU_SIMPLE_DAMAGE": StatusSupported,
		"GU_FORMATION":     StatusNeedsReauthor,
		"GU_FREEZE":        StatusNeedsReauthor,
		"GU_NATIVE":        StatusNeedsReauthor,
		"GU_WORLD":         StatusNotActive,
		"GU_NO_BLOCKS":     StatusNotActive,
	}
	for _, audited := range report.Entries {
		if audited.Status != want[audited.ID] {
			t.Errorf("%s status=%s, want %s (reasons=%v)", audited.ID, audited.Status, want[audited.ID], audited.Reasons)
		}
	}
	if report.Summary.Supported != 1 || report.Summary.NeedsReauthor != 3 || report.Summary.NotActive != 2 || report.Summary.Unsupported != 0 {
		t.Fatalf("unexpected summary: %+v", report.Summary)
	}
	if report.Summary.Blocks != 5 || report.Summary.Operations != 5 {
		t.Fatalf("unexpected block/operation counts: %+v", report.Summary)
	}
}

func TestAuditMarksInvalidBlockUnsupported(t *testing.T) {
	entries := []GuEntry{
		entry("GU_BAD", 1, `[{"id":"ACTIVE","trigger":"NOPE","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":1}]}]`),
	}
	report := Audit(entries, nil)
	if report.Entries[0].Status != StatusUnsupported {
		t.Fatalf("invalid block status=%s", report.Entries[0].Status)
	}
	if report.Summary.Unsupported != 1 || report.Entries[0].Blocks[0].Status != StatusUnsupported {
		t.Fatalf("invalid block was not reported: %+v", report.Entries[0])
	}
}

func TestAuditEntryWorstStatusAndMVPMatching(t *testing.T) {
	mvp := []EntryRef{{ID: "GU_MIXED", Rank: 5}, {ID: "GU_MISSING", Rank: 1}}
	entries := []GuEntry{
		entry("GU_MIXED", 5, `[
			{"id":"PASSIVE","trigger":"ALWAYS_ON","target":{"selector":"SELF"},"operations":[{"op":"STAT_MOD","stat":"ATTACK","sub_type":"ADD_FLAT","value":1}]},
			{"id":"ACTIVE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"STATUS","status_id":"FREEZE","turns":1,"chance":1}]}
		]`),
	}
	report := Audit(entries, mvp)
	if report.Entries[0].Status != StatusNeedsReauthor || !report.Entries[0].MVP {
		t.Fatalf("mixed entry=%+v", report.Entries[0])
	}
	if len(report.MissingMVP) != 1 || report.MissingMVP[0].ID != "GU_MISSING" {
		t.Fatalf("missing MVP not reported: %+v", report.MissingMVP)
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected hidden-passive/MVP warnings")
	}
}

func TestWriteCSVIncludesEveryStatusAndOperation(t *testing.T) {
	report := Audit([]GuEntry{
		entry("GU_A", 1, `[{"id":"ACTIVE","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":5},{"op":"HEAL","sub_type":"FLAT","value":2}]}]`),
	}, []EntryRef{{ID: "GU_A", Rank: 1}})
	var buf bytes.Buffer
	if err := WriteCSV(&buf, report); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("csv lines=%d: %q", len(lines), buf.String())
	}
	if !strings.Contains(lines[0], "gu_id,rank") || !strings.Contains(lines[1], "DAMAGE|HEAL") || !strings.Contains(lines[1], "SUPPORTED") {
		t.Fatalf("unexpected csv: %q", buf.String())
	}
}
