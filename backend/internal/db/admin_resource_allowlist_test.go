package db

import (
	"strings"
	"testing"
)

func TestTableImportExportRejectClientDatabaseIdentifiers(t *testing.T) {
	for _, key := range []string{"players", "information_schema.tables", "gu; DROP TABLE players", `"gu_dictionary"`} {
		if err := TableImport(key); err == nil || !strings.Contains(err.Error(), "unknown table") {
			t.Errorf("TableImport(%q) error = %v, want unknown-table rejection", key, err)
		}
		if err := TableExport(key); err == nil || !strings.Contains(err.Error(), "unknown table") {
			t.Errorf("TableExport(%q) error = %v, want unknown-table rejection", key, err)
		}
	}
}
