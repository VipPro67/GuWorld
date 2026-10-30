package handlers

import (
	"errors"
	"testing"
)

// validateCombatLoadoutEntries is the pure seam guarding slots 9-12 writes:
// stale or unsupported pins must never persist.

func TestValidateCombatLoadoutEntriesAcceptsSanitizedOrder(t *testing.T) {
	candidates := []combatSkillCandidate{
		{Kind: "move", ID: "mv1", Supported: true},
		{Kind: "house", ID: "hk1", Supported: true},
	}
	clean, err := validateCombatLoadoutEntries(
		[]combatLoadoutEntry{{Kind: "house", ID: "hk1"}, {Kind: "move", ID: "mv1"}},
		candidates,
	)
	if err != nil {
		t.Fatalf("valid pins were rejected: %v", err)
	}
	if len(clean) != 2 || clean[0].ID != "hk1" || clean[1].ID != "mv1" {
		t.Fatalf("unexpected sanitized entries: %+v", clean)
	}
	if clean[0].Kind != "house" {
		t.Fatalf("entry kind must be preserved for storage")
	}
}

func TestValidateCombatLoadoutEntriesRejectsBadEntries(t *testing.T) {
	candidates := []combatSkillCandidate{
		{Kind: "move", ID: "mv1", Supported: true},
		{Kind: "move", ID: "mvBroken", Reason: "Chiêu chưa có executor realtime"},
		{Kind: "house", ID: "hk1", Supported: true},
	}
	cases := []struct {
		name    string
		entries []combatLoadoutEntry
		wantErr error
	}{
		{"empty id", []combatLoadoutEntry{{Kind: "move", ID: ""}}, errLoadoutInvalidEntry},
		{"unknown kind", []combatLoadoutEntry{{Kind: "weird", ID: "x"}}, errLoadoutInvalidEntry},
		{"duplicate", []combatLoadoutEntry{{Kind: "move", ID: "mv1"}, {Kind: "move", ID: "mv1"}}, errLoadoutDuplicateEntry},
		{"not equipped", []combatLoadoutEntry{{Kind: "move", ID: "ghost"}}, errLoadoutUnknownEntry},
		{"unsupported", []combatLoadoutEntry{{Kind: "move", ID: "mvBroken"}}, errLoadoutUnsupportedEntry},
	}
	for _, tc := range cases {
		_, err := validateCombatLoadoutEntries(tc.entries, candidates)
		if err == nil {
			t.Fatalf("%s: expected rejection", tc.name)
		}
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("%s: got %v, want %v", tc.name, err, tc.wantErr)
		}
	}
	// The unsupported entry carries the compiler reason for the HTTP message.
	_, err := validateCombatLoadoutEntries([]combatLoadoutEntry{{Kind: "move", ID: "mvBroken"}}, candidates)
	var entryErr *loadoutEntryError
	if !errors.As(err, &entryErr) || entryErr.Detail != "Chiêu chưa có executor realtime" {
		t.Fatalf("unsupported detail lost: %v", err)
	}
}

func TestValidateCombatLoadoutEntriesEmptyOrderIsStable(t *testing.T) {
	// An explicit empty order must remain valid so clearing a slot is stable.
	clean, err := validateCombatLoadoutEntries(nil, []combatSkillCandidate{{Kind: "move", ID: "mv1", Supported: true}})
	if err != nil {
		t.Fatalf("empty order rejected: %v", err)
	}
	if len(clean) != 0 {
		t.Fatalf("empty order should sanitize to empty: %+v", clean)
	}
}
