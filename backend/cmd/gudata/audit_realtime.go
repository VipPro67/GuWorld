package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"

	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
)

// cmdAuditRealtime classifies every Gu effect block for the realtime zone
// runtime and optionally writes the generated coverage CSV/JSON. It never
// mutates Gu.csv.
func cmdAuditRealtime(args []string) error {
	fs := flag.NewFlagSet("audit-realtime", flag.ExitOnError)
	data := fs.String("data", "data", "data directory")
	csvPath := fs.String("csv", "", "write human-readable coverage CSV to this path")
	reportPath := fs.String("report", "", "write machine-readable JSON report to this path")
	strict := fs.Bool("strict", false, "fail when an operation is unclassified or an MVP entry is missing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	entries, err := loadRealtimeGuEntries(*data + "/Gu.csv")
	if err != nil {
		return err
	}
	report := realtimeeffects.Audit(entries, realtimeeffects.DefaultMVP())

	if *csvPath != "" {
		file, err := os.Create(*csvPath)
		if err != nil {
			return fmt.Errorf("create %s: %w", *csvPath, err)
		}
		writeErr := realtimeeffects.WriteCSV(file, report)
		closeErr := file.Close()
		if writeErr != nil {
			return fmt.Errorf("write %s: %w", *csvPath, writeErr)
		}
		if closeErr != nil {
			return closeErr
		}
	}
	if *reportPath != "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(*reportPath, append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	if *csvPath == "" && *reportPath == "" {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	} else {
		fmt.Printf("realtime gu coverage: entries=%d supported=%d needs_reauth=%d unsupported=%d not_active=%d blocks=%d operations=%d mvp=%d\n",
			report.Summary.Entries, report.Summary.Supported, report.Summary.NeedsReauthor,
			report.Summary.Unsupported, report.Summary.NotActive, report.Summary.Blocks,
			report.Summary.Operations, len(report.MVP)-len(report.MissingMVP))
	}
	if *strict && (report.Summary.Unclassified > 0 || len(report.MissingMVP) > 0) {
		return fmt.Errorf("realtime audit is not clean: unclassified=%d missing_mvp=%d", report.Summary.Unclassified, len(report.MissingMVP))
	}
	return nil
}

func loadRealtimeGuEntries(path string) ([]realtimeeffects.GuEntry, error) {
	t, err := readCSV(path)
	if err != nil {
		return nil, err
	}
	entries := make([]realtimeeffects.GuEntry, 0, len(t.rows))
	for _, row := range t.rows {
		rank, _ := strconv.Atoi(t.get(row, "rank"))
		mana, _ := strconv.ParseFloat(t.get(row, "base_mana_cost"), 64)
		entries = append(entries, realtimeeffects.GuEntry{
			ID: t.get(row, "id"), Name: t.get(row, "name"), Rank: rank,
			Type: t.get(row, "gu_type"), PrimaryRole: t.get(row, "primary_role"),
			ManaCost: mana, Effects: json.RawMessage(t.get(row, "effects")),
		})
	}
	return entries, nil
}
