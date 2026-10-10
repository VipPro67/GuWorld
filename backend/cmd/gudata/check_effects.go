package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/vippro67/GuWorld/backend/internal/effectcatalog"
)

// cmdCheckEffects validates the effect definitions in Gu.csv (via the catalog,
// which also runs Block-v2 validation and unclassified-key detection) and
// Item.csv (structural Block-v2 validation). It is read-only.
func cmdCheckEffects(args []string) error {
	fs := flag.NewFlagSet("check-effects", flag.ExitOnError)
	data := fs.String("data", "data", "data directory")
	report := fs.String("report", "", "write JSON report to this file instead of stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	dir := *data

	entries, err := loadGuEntries(dir + "/Gu.csv")
	if err != nil {
		return err
	}
	guReport := effectcatalog.ValidateGuCatalog(entries)

	itemIssues, err := validateItemBlocks(dir + "/Item.csv")
	if err != nil {
		return err
	}

	out := struct {
		Gu         effectcatalog.CatalogReport     `json:"gu"`
		ItemIssues []effectcatalog.ValidationIssue `json:"item_issues"`
	}{
		Gu:         guReport,
		ItemIssues: itemIssues,
	}

	if *report != "" {
		b, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(*report, b, 0o644); err != nil {
			return err
		}
	} else if err := printJSON(out); err != nil {
		return err
	}

	if guReport.HasErrors() || len(itemIssues) > 0 {
		return fmt.Errorf("found %d Gu error(s) and %d Item issue(s)", guReport.Summary.Errors, len(itemIssues))
	}
	return nil
}

func loadGuEntries(path string) ([]effectcatalog.GuEntry, error) {
	t, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	entries := make([]effectcatalog.GuEntry, 0, len(t.rows))
	for _, row := range t.rows {
		id := t.get(row, "id")
		rank, _ := strconv.Atoi(t.get(row, "rank"))
		entries = append(entries, effectcatalog.GuEntry{
			ID:      id,
			Rank:    rank,
			Effects: json.RawMessage(t.get(row, "effects")),
		})
	}
	return entries, nil
}

func validateItemBlocks(path string) ([]effectcatalog.ValidationIssue, error) {
	t, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	var issues []effectcatalog.ValidationIssue
	for i, row := range t.rows {
		id := t.get(row, "id")
		raw := t.get(row, "effects")
		rowIssues := effectcatalog.ValidateBlockV2([]byte(raw), effectcatalog.Location{Source: "item", Row: i + 1, Key: id})
		issues = append(issues, rowIssues...)
	}
	return issues, nil
}
