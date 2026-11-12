package db

import (
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordingCSVImportExecutor struct {
	queries []string
	argLens []int
}

func (r *recordingCSVImportExecutor) Exec(q string, args ...interface{}) (sql.Result, error) {
	r.queries = append(r.queries, q)
	r.argLens = append(r.argLens, len(args))
	return nil, nil
}

func (r *recordingCSVImportExecutor) Query(string, ...interface{}) (*sql.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (r *recordingCSVImportExecutor) QueryRow(string, ...interface{}) *sql.Row { return nil }

func TestSeedGusFromCSVBatchesInserts(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "Gu.csv")
	var content strings.Builder
	content.WriteString("id,name,rank,effects\n")
	for i := 0; i < 501; i++ {
		fmt.Fprintf(&content, "gu_%d,Gu %d,1,[]\n", i, i)
	}
	if err := os.WriteFile(filename, []byte(content.String()), 0o600); err != nil {
		t.Fatal(err)
	}

	rec := &recordingCSVImportExecutor{}
	csvImportState.Lock()
	defer csvImportState.Unlock()
	prevExecutor, prevOverride := csvImportState.executor, csvImportState.override
	csvImportState.executor = rec
	csvImportState.override = nil
	defer func() { csvImportState.executor = prevExecutor; csvImportState.override = prevOverride }()

	if err := seedGusFromCSV(filename); err != nil {
		t.Fatalf("seedGusFromCSV() error = %v", err)
	}
	if len(rec.queries) != 2 {
		t.Fatalf("exec count = %d, want 2 batched inserts", len(rec.queries))
	}
	colsPerRow := 21
	if rec.argLens[0] != 500*colsPerRow || rec.argLens[1] != colsPerRow {
		t.Fatalf("arg counts = %v, want [%d %d]", rec.argLens, 500*colsPerRow, colsPerRow)
	}
	if !strings.Contains(rec.queries[0], "($1,") || !strings.Contains(rec.queries[0], fmt.Sprintf("$%d", 500*colsPerRow)) {
		t.Fatalf("first batch placeholders malformed: %q", rec.queries[0][:80])
	}
	if strings.Contains(rec.queries[0], fmt.Sprintf("$%d", 500*colsPerRow+1)) {
		t.Fatalf("first batch has placeholder beyond its %d params", 500*colsPerRow)
	}
	if !strings.Contains(rec.queries[1], "($1,") {
		t.Fatalf("second batch placeholders must restart at $1")
	}
}

type failingCSVImportExecutor struct{ err error }

func (e failingCSVImportExecutor) Exec(string, ...interface{}) (sql.Result, error) {
	return nil, e.err
}

func (failingCSVImportExecutor) Query(string, ...interface{}) (*sql.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (failingCSVImportExecutor) QueryRow(string, ...interface{}) *sql.Row { return nil }

func TestSeedGusFromCSVReturnsUpsertError(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "Gu.csv")
	if err := os.WriteFile(filename, []byte("id,name,rank,effects\ngu_test,Test Gu,1,[]\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("database constraint rejected Gu")
	csvImportState.Lock()
	defer csvImportState.Unlock()
	previousExecutor := csvImportState.executor
	previousOverride := csvImportState.override
	csvImportState.executor = failingCSVImportExecutor{err: wantErr}
	csvImportState.override = nil
	defer func() {
		csvImportState.executor = previousExecutor
		csvImportState.override = previousOverride
	}()

	err := seedGusFromCSV(filename)
	if !errors.Is(err, wantErr) {
		t.Fatalf("seedGusFromCSV() error = %v, want wrapped upsert error %v", err, wantErr)
	}
}

func TestValidateImportCSVRejectsDuplicateAndBlankKeys(t *testing.T) {
	dataset, ok := findImportDataset("item")
	if !ok {
		t.Fatal("item dataset is not registered")
	}
	rows, issues := validateImportCSV(dataset, []byte("id,name,item_type\nitem_a,A,MATERIAL\nitem_a,B,MATERIAL\n,Blank,MATERIAL\n"))
	if rows != 2 {
		t.Fatalf("rows = %d, want 2 valid keyed rows", rows)
	}
	if len(issues) != 2 {
		t.Fatalf("issues = %#v, want duplicate and blank-key issues", issues)
	}
	if issues[0].Severity != "error" || issues[1].Severity != "error" {
		t.Fatalf("issues must block commit: %#v", issues)
	}
}

func TestValidateImportCSVRejectsMissingRequiredHeader(t *testing.T) {
	dataset, _ := findImportDataset("gu")
	_, issues := validateImportCSV(dataset, []byte("id,name\ngu_a,A\n"))
	if len(issues) != 1 || issues[0].Code != "missing_header" || issues[0].Field != "rank" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestValidateImportCSVDreamRealmPathIsNotJSON(t *testing.T) {
	dataset, ok := findImportDataset("dream_realm")
	if !ok {
		t.Fatal("dream_realm dataset is not registered")
	}
	content, err := os.ReadFile(filepath.Join("..", "..", "data", dataset.filename))
	if err != nil {
		t.Fatal(err)
	}

	_, issues := validateImportCSV(dataset, content)
	for _, issue := range issues {
		if issue.Code == "invalid_json" && issue.Field == "path" {
			t.Fatalf("DreamRealm path is a scalar, but validation rejected row %d: %#v", issue.Row, issue)
		}
	}
}

func TestValidateImportCSVStillRequiresGuPathJSON(t *testing.T) {
	dataset, ok := findImportDataset("gu")
	if !ok {
		t.Fatal("gu dataset is not registered")
	}

	_, issues := validateImportCSV(dataset, []byte("id,rank,path\ngu_test,1,not-json\n"))
	if len(issues) != 1 || issues[0].Code != "invalid_json" || issues[0].Field != "path" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestRegionCSVKeepsApertureInteriorInactiveAndLootRegionsPresent(t *testing.T) {
	readRows := func(filename string) ([]string, [][]string) {
		t.Helper()
		file, err := os.Open(filepath.Join("..", "..", "data", filename))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		reader := csv.NewReader(file)
		header, err := reader.Read()
		if err != nil {
			t.Fatal(err)
		}
		rows, err := reader.ReadAll()
		if err != nil {
			t.Fatal(err)
		}
		return header, rows
	}

	regionHeader, regionRows := readRows("Region.csv")
	regionIDColumn := colIndex(regionHeader, "id")
	regionActiveColumn := colIndex(regionHeader, "is_active")
	if regionIDColumn < 0 || regionActiveColumn < 0 {
		t.Fatal("Region.csv must contain id and is_active columns")
	}
	regions := make(map[string]string, len(regionRows))
	for _, row := range regionRows {
		regions[row[regionIDColumn]] = row[regionActiveColumn]
	}
	active, ok := regions["aperture_interior"]
	if !ok {
		t.Fatal("Region.csv must contain aperture_interior")
	}
	if !strings.EqualFold(active, "false") {
		t.Fatal("Region.csv must keep aperture_interior inactive until its layer is opened")
	}
	if _, ok := regions["am_hon_sa_vuc"]; !ok {
		t.Fatal("Region.csv must contain am_hon_sa_vuc before importing its loot tables")
	}

	lootHeader, lootRows := readRows("RegionLootTables.csv")
	lootRegionColumn := colIndex(lootHeader, "region_id")
	if lootRegionColumn < 0 {
		t.Fatal("RegionLootTables.csv must contain region_id")
	}
	for _, row := range lootRows {
		if _, ok := regions[row[lootRegionColumn]]; !ok {
			t.Errorf("RegionLootTables.csv references unknown region %q", row[lootRegionColumn])
		}
	}
}

func TestImportDatasetCatalogMatchesAvailableTables(t *testing.T) {
	want := strings.Join(ImportDatasetKeys(), ",")
	got := strings.Join(AvailableTables(), ",")
	if got != want {
		t.Fatalf("AvailableTables = %q, want canonical catalog %q", got, want)
	}
	for _, key := range ImportDatasetKeys() {
		if _, ok := findImportDataset(key); !ok {
			t.Fatalf("missing dataset definition for %s", key)
		}
	}
}
