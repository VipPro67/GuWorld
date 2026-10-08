package handlers

import (
	"database/sql"
	"testing"
	"time"
)

func TestBanIsActive(t *testing.T) {
	past := sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true}
	future := sql.NullTime{Time: time.Now().Add(time.Hour), Valid: true}
	permanent := sql.NullTime{}

	cases := []struct {
		name     string
		isBanned bool
		until    sql.NullTime
		want     bool
	}{
		{"not banned", false, permanent, false},
		{"permanent ban", true, permanent, true},
		{"expired ban", true, past, false},
		{"active temporary ban", true, future, true},
	}
	for _, tc := range cases {
		if got := banIsActive(tc.isBanned, tc.until); got != tc.want {
			t.Errorf("%s: banIsActive=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestBanUntilFromRequest(t *testing.T) {
	if until, ok := banUntilFromRequest(0, true); !ok || until.Valid {
		t.Fatalf("permanent ban should be valid with no deadline, got ok=%v valid=%v", ok, until.Valid)
	}
	if _, ok := banUntilFromRequest(0, false); ok {
		t.Fatal("missing duration and not permanent must be rejected")
	}
	until, ok := banUntilFromRequest(2, false)
	if !ok || !until.Valid {
		t.Fatalf("2h ban should be valid, got ok=%v valid=%v", ok, until.Valid)
	}
	if delta := time.Until(until.Time); delta < time.Hour || delta > 3*time.Hour {
		t.Fatalf("2h ban deadline out of range: %v", delta)
	}
}
