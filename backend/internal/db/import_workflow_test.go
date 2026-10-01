package db

import (
	"strings"
	"testing"
)

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
