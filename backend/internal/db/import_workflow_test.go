package db

import (
	"database/sql"
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
	if rec.argLens[0] != 500*20 || rec.argLens[1] != 20 {
		t.Fatalf("arg counts = %v, want [%d %d]", rec.argLens, 500*20, 20)
	}
	if !strings.Contains(rec.queries[0], "($1,") || !strings.Contains(rec.queries[0], "$10000") {
		t.Fatalf("first batch placeholders malformed: %q", rec.queries[0][:80])
	}
	if strings.Contains(rec.queries[0], "$10001") {
		t.Fatalf("first batch has placeholder beyond its 10000 params")
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
