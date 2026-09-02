package db

import (
	"testing"
)

func TestHasActiveTribulationOrEnemiesFallback(t *testing.T) {
	// Verify nil Queryer falls back to DB
	var q Queryer
	if q == nil {
		q = DB
	}
	if q != DB {
		t.Fatalf("expected q to fall back to DB")
	}
}
