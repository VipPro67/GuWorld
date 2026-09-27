// Command migrate_gu_effects audits all legacy effect formats and emits block-v2
// candidates. Dry-run is the default and source files are never overwritten.
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vippro67/GuWorld/backend/internal/effectmigration"
)

type aggregate struct {
	SchemaVersion int                      `json:"schema_version"`
	DryRun        bool                     `json:"dry_run"`
	Source        string                   `json:"source"`
	Format        effectmigration.Format   `json:"format"`
	Results       []effectmigration.Result `json:"results"`
	MigrationGate bool                     `json:"migration_gate"`
	ErrorCode     string                   `json:"error_code,omitempty"`
	ErrorReason   string                   `json:"error_reason,omitempty"`
}

func main() {
	var formatName, source, output, reportPath string
	var write bool
	flag.StringVar(&formatName, "format", "gu", "gu, item, gamefx, killer_move, or aperture_variant_event")
	flag.StringVar(&source, "source", "data/Gu.csv", "source CSV (gu/item) or JSON file")
	flag.StringVar(&output, "output", "", "candidate output path (required with -write)")
	flag.StringVar(&reportPath, "report", "", "machine-readable audit report; defaults to stdout")
	flag.BoolVar(&write, "write", false, "write candidate output; never overwrites source")
	flag.Parse()

	format, err := effectmigration.ParseFormat(formatName)
	fatalIf(err)
	absSource, err := filepath.Abs(source)
	fatalIf(err)
	if write && output == "" {
		fatalIf(fmt.Errorf("-output is required with -write"))
	}
	for _, destination := range []string{output, reportPath} {
		if destination != "" {
			fatalIf(rejectSourceDestination(absSource, destination))
		}
	}
	data, err := os.ReadFile(source)
	fatalIf(err)
	var results []effectmigration.Result
	var candidate []byte
	if filepath.Ext(source) == ".csv" {
		results, candidate, err = convertCSV(format, data)
	} else {
		results = []effectmigration.Result{effectmigration.Convert(format, filepath.Base(source), data)}
		candidate = results[0].Envelope
	}
	fatalIf(err)
	gate := len(results) > 0
	var gateCode, gateReason string
	for _, result := range results {
		gate = gate && result.Report.MigrationGate
		if gateCode == "" && !result.Report.MigrationGate {
			decision := effectmigration.EvaluateMigrationGate(result.Report)
			gateCode, gateReason = decision.ErrorCode, decision.Reason
		}
	}
	if len(results) == 0 {
		gateCode, gateReason = "empty_input", "input contains no effect records"
	}
	audit := aggregate{SchemaVersion: 1, DryRun: !write, Source: source, Format: format, Results: results, MigrationGate: gate, ErrorCode: gateCode, ErrorReason: gateReason}
	encoded, err := json.MarshalIndent(audit, "", "  ")
	fatalIf(err)
	encoded = append(encoded, '\n')
	if reportPath == "" {
		_, err = os.Stdout.Write(encoded)
	} else {
		err = os.WriteFile(reportPath, encoded, 0o644)
	}
	fatalIf(err)
	if write {
		if !gate {
			fatalIf(fmt.Errorf("%s: %s", gateCode, gateReason))
		}
		fatalIf(os.WriteFile(output, candidate, 0o644))
	}
}

func convertCSV(format effectmigration.Format, data []byte) ([]effectmigration.Result, []byte, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.LazyQuotes = true
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("empty CSV")
	}
	idIndex, effectsIndex := -1, -1
	for i, name := range rows[0] {
		if name == "id" {
			idIndex = i
		}
		if name == "effects" {
			effectsIndex = i
		}
	}
	if idIndex < 0 || effectsIndex < 0 {
		return nil, nil, fmt.Errorf("CSV requires id and effects columns")
	}
	results := make([]effectmigration.Result, 0, len(rows)-1)
	for i := 1; i < len(rows); i++ {
		if len(rows[i]) <= effectsIndex || len(rows[i]) <= idIndex {
			continue
		}
		result := effectmigration.Convert(format, rows[i][idIndex], []byte(rows[i][effectsIndex]))
		results = append(results, result)
		if len(result.Envelope) > 0 {
			candidate, mergeErr := mergeCandidate(format, []byte(rows[i][effectsIndex]), result.Envelope)
			if mergeErr != nil {
				return nil, nil, mergeErr
			}
			rows[i][effectsIndex] = string(candidate)
		}
	}
	var out bytes.Buffer
	writer := csv.NewWriter(&out)
	writer.WriteAll(rows)
	if err := writer.Error(); err != nil {
		return nil, nil, err
	}
	return results, out.Bytes(), nil
}

func fatalIf(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func rejectSourceDestination(absSource, destination string) error {
	info, err := os.Lstat(destination)
	if err == nil && info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing symlink destination %s", destination)
	}
	absDestination, err := filepath.Abs(destination)
	if err != nil {
		return err
	}
	if absDestination == absSource {
		return fmt.Errorf("refusing to overwrite source file %s", destination)
	}
	sourceInfo, sourceErr := os.Stat(absSource)
	destinationInfo, destinationErr := os.Stat(absDestination)
	if sourceErr == nil && destinationErr == nil && os.SameFile(sourceInfo, destinationInfo) {
		return fmt.Errorf("refusing source-equivalent destination %s", destination)
	}
	return nil
}

func mergeCandidate(format effectmigration.Format, legacy, envelope []byte) ([]byte, error) {
	_ = format
	_ = legacy
	var candidate struct {
		SchemaVersion int             `json:"schema_version"`
		Blocks        json.RawMessage `json:"blocks"`
	}
	if err := json.Unmarshal(envelope, &candidate); err != nil {
		return nil, err
	}
	return json.Marshal(candidate)
}
