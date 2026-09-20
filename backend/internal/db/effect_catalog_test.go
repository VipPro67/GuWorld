package db

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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

	wantKeys := map[string]int{"active": 36, "passive": 28, "growth": 5, "support": 2, "status": 16, "other": 42}
	gotKeys := map[string]int{}
	for _, coverage := range report.Coverage {
		gotKeys[coverage.Section]++
		if coverage.Owner == "" || !coverage.Disposition.Valid() {
			t.Errorf("unowned coverage: %+v", coverage)
		}
	}
	for section, want := range wantKeys {
		if gotKeys[section] != want {
			t.Errorf("%s keys = %d, want %d; classify new data in effectcatalog manifest", section, gotKeys[section], want)
		}
	}
}

func firstCatalogIssues(issues []effectcatalog.CatalogIssue, limit int) []effectcatalog.CatalogIssue {
	if len(issues) <= limit {
		return issues
	}
	return issues[:limit]
}
