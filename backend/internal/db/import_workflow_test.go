package db

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
