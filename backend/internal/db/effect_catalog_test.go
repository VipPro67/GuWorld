package db

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

func TestGuCSVEffectCatalogCoverage(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	f, reader, header, err := openCSV("Gu.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	idIdx, rankIdx, effectsIdx := colIndex(header, "id"), colIndex(header, "rank"), colIndex(header, "effects")
	if idIdx < 0 || rankIdx < 0 || effectsIdx < 0 {
		t.Fatal("Gu.csv missing id/rank/effects")
	}

	// Match production import semantics: the final duplicate (id, rank) wins.
	entries := map[string]effectcatalog.GuEntry{}
	line := 1
	for {
		line++
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			t.Fatalf("Gu.csv line %d: %v", line, readErr)
		}
		if len(row) <= effectsIdx || row[idIdx] == "" {
			continue
		}
		rank := 0
		if _, err := fmt.Sscanf(row[rankIdx], "%d", &rank); err != nil {
			t.Fatalf("Gu.csv line %d invalid rank: %v", line, err)
		}
		key := fmt.Sprintf("%s\x00%d", row[idIdx], rank)
		entries[key] = effectcatalog.GuEntry{ID: row[idIdx], Rank: rank, Effects: json.RawMessage(row[effectsIdx])}
	}

	list := make([]effectcatalog.GuEntry, 0, len(entries))
	for _, entry := range entries {
		list = append(list, entry)
	}
	report := effectcatalog.ValidateGuCatalog(list)
	if report.HasErrors() || report.Summary.Unclassified != 0 {
		t.Fatalf("Gu.csv effect coverage failed: summary=%+v first issues=%+v", report.Summary, firstCatalogIssues(report.Issues, 10))
	}

	wantKeys := map[string]int{"active": 36, "passive": 34, "growth": 5, "status": 22, "other": 45} // support section deleted; healing lives in active.heal_flat
	gotKeys := map[string]int{}
	for _, coverage := range report.Coverage {
		gotKeys[coverage.Section]++
		if coverage.Owner == "" || !coverage.Disposition.Valid() {
			t.Errorf("unowned coverage: %+v", coverage)
		}
	}
	if len(gotKeys) == 0 {
		t.Skip("Gu.csv has been fully migrated to pure Block v2 format; legacy section coverage skipped.")
		return
	}
	for section, want := range wantKeys {
		if gotKeys[section] != want {
			t.Errorf("%s keys = %d, want %d; classify new data in effectcatalog manifest", section, gotKeys[section], want)
		}
	}
}

// nativeUniqueAllowlist pins the effect codes that are intentionally still
// authored as UNIQUE native hooks in the shipped CSVs. The migration target is
// an empty list (zero UNIQUE operations remaining across all datasets).
var nativeUniqueAllowlist = map[string]int{}

func TestCSVNativeUniqueAllowlist(t *testing.T) {
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWD)

	gu := collectUniqueOps(t, "Gu.csv")
	if len(gu) != 0 {
		t.Fatalf("Gu.csv UNIQUE codes = %v, want empty (zero UNIQUE operations remaining)", gu)
	}
	for _, filename := range []string{"Item.csv", "Race.csv", "Constitution.csv", "Status.csv"} {
		if got := collectUniqueOps(t, filename); len(got) != 0 {
			t.Errorf("%s still authors UNIQUE operations: %v", filename, got)
		}
	}
}

func collectUniqueOps(t *testing.T, filename string) map[string]int {
	t.Helper()
	f, reader, header, err := openCSV(filename)
	if err != nil {
		t.Fatalf("open %s: %v", filename, err)
	}
	defer f.Close()
	effectsIdx := colIndex(header, "effects")
	if effectsIdx < 0 {
		t.Fatalf("%s missing effects column", filename)
	}

	type operation struct {
		Op   string `json:"op"`
		Code string `json:"code"`
	}
	type block struct {
		Operations []operation `json:"operations"`
	}

	counts := map[string]int{}
	line := 1
	for {
		line++
		row, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			t.Fatalf("%s line %d: %v", filename, line, readErr)
		}
		if len(row) <= effectsIdx {
			continue
		}
		raw := strings.TrimSpace(row[effectsIdx])
		if raw == "" || raw == "null" {
			continue
		}

		var blocks []block
		if strings.HasPrefix(raw, "[") {
			if err := json.Unmarshal([]byte(raw), &blocks); err != nil {
				t.Fatalf("%s line %d: %v", filename, line, err)
			}
		} else {
			var envelope struct {
				Blocks []block `json:"blocks"`
			}
			if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
				t.Fatalf("%s line %d: %v", filename, line, err)
			}
			blocks = envelope.Blocks
		}

		for _, b := range blocks {
			for _, op := range b.Operations {
				if op.Op == "UNIQUE" {
					counts[op.Code]++
				}
			}
		}
	}
	return counts
}

func firstCatalogIssues(issues []effectcatalog.CatalogIssue, limit int) []effectcatalog.CatalogIssue {
	if len(issues) <= limit {
		return issues
	}
	return issues[:limit]
}
