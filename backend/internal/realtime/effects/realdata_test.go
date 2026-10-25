package effects

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestRealGuCatalogHasExplicitRealtimeStatus(t *testing.T) {
	entries, err := loadGuCSV("../../../data/Gu.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 1000 {
		t.Fatalf("unexpected Gu catalog size: %d", len(entries))
	}
	report := Audit(entries, DefaultMVP())
	if len(report.MissingMVP) != 0 {
		t.Fatalf("MVP references missing entries: %+v", report.MissingMVP)
	}
	if report.Summary.Entries != len(entries) {
		t.Fatalf("summary entries=%d catalog=%d", report.Summary.Entries, len(entries))
	}
	if report.Summary.Supported+report.Summary.NeedsReauthor+report.Summary.Unsupported+report.Summary.NotActive != len(entries) {
		t.Fatalf("entry status counts do not add up: %+v", report.Summary)
	}
	for _, entry := range report.Entries {
		if !entry.Status.Valid() {
			t.Fatalf("%s has invalid status %q", entry.ID, entry.Status)
		}
		for _, block := range entry.Blocks {
			if !block.Status.Valid() || block.Trigger == "" {
				t.Fatalf("%s block %q has invalid audit: %+v", entry.ID, block.ID, block)
			}
		}
	}
	if len(report.Warnings) == 0 {
		t.Fatal("expected a warning that no hidden passive is authored")
	}
	byID := map[string]EntryAudit{}
	for _, entry := range report.Entries {
		byID[entry.ID] = entry
	}
	if byID["GU_WATER_ARROW"].Status != StatusNeedsReauthor {
		t.Fatalf("GU_WATER_ARROW should need a formation→direction reauthor, got %s", byID["GU_WATER_ARROW"].Status)
	}
	if byID["GU_STRENGTH"].Status != StatusSupported {
		t.Fatalf("GU_STRENGTH passive stats should be supported, got %s", byID["GU_STRENGTH"].Status)
	}
	if byID["GU_WATER_CLEANSING"].Status != StatusSupported {
		t.Fatalf("GU_WATER_CLEANSING heal should be supported, got %s", byID["GU_WATER_CLEANSING"].Status)
	}
	if report.Summary.Entries != 4427 || report.Summary.Supported != 3374 ||
		report.Summary.NeedsReauthor != 871 || report.Summary.Unsupported != 0 ||
		report.Summary.NotActive != 182 || report.Summary.Blocks != 5021 ||
		report.Summary.Operations != 6380 {
		t.Fatalf("realtime coverage summary drifted from the committed baseline: %+v", report.Summary)
	}
	var csvBuffer bytes.Buffer
	if err := WriteCSV(&csvBuffer, report); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(csvBuffer.Bytes())
	if got := hex.EncodeToString(digest[:]); got != "549c9dd30b79e70d2a3e24edbb57ed18078c84f66a507461f6b3fd7005fae2d9" {
		t.Fatalf("gu-coverage.csv drifted; regenerate docs/mmo/gu-coverage.csv: %s", got)
	}
}

func loadGuCSV(path string) ([]GuEntry, error) {
	return LoadGuCSV(path)
}
