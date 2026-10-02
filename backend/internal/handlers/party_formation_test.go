package handlers

import (
	"testing"
)

func TestNormalizePartyFormation(t *testing.T) {
	leader := "11111111-1111-1111-1111-111111111111"
	member := "22222222-2222-2222-2222-222222222222"
	pet := "33333333-3333-3333-3333-333333333333"

	// Leader entry is dropped, order preserved.
	got, err := normalizePartyFormation([]PartyFormationEntry{
		{Kind: "player", ID: leader},
		{Kind: "player", ID: member},
		{Kind: "pet", ID: pet},
	}, leader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].ID != member || got[1].ID != pet {
		t.Fatalf("unexpected normalized order: %+v", got)
	}

	// Duplicates collapse.
	got, err = normalizePartyFormation([]PartyFormationEntry{
		{Kind: "pet", ID: pet},
		{Kind: "pet", ID: pet},
	}, leader)
	if err != nil || len(got) != 1 {
		t.Fatalf("expected dedupe, got %+v err %v", got, err)
	}

	// Bad kind rejected.
	if _, err := normalizePartyFormation([]PartyFormationEntry{{Kind: "boss", ID: pet}}, leader); err == nil {
		t.Fatal("expected error for bad kind")
	}

	// Bad UUID rejected.
	if _, err := normalizePartyFormation([]PartyFormationEntry{{Kind: "pet", ID: "nope"}}, leader); err == nil {
		t.Fatal("expected error for bad UUID")
	}

	// Over cap rejected: 9 distinct followers > MaxPartyFollowers(8).
	full := []PartyFormationEntry{}
	base := []string{"a", "b", "c", "d", "e", "f", "0", "1", "2"}
	for _, h := range base {
		full = append(full, PartyFormationEntry{Kind: "player", ID: h + "0000000-0000-0000-0000-000000000000"})
	}
	if _, err := normalizePartyFormation(full, leader); err == nil {
		t.Fatal("expected error over cap")
	}

	// Explicit pos kept; dup pos and out-of-range rejected.
	got, err = normalizePartyFormation([]PartyFormationEntry{
		{Kind: "pet", ID: pet, Pos: 1},
		{Kind: "player", ID: member, Pos: 5},
	}, leader)
	if err != nil || got[0].Pos != 1 || got[1].Pos != 5 {
		t.Fatalf("expected pos preserved, got %+v err %v", got, err)
	}
	if _, err := normalizePartyFormation([]PartyFormationEntry{
		{Kind: "pet", ID: pet, Pos: 3},
		{Kind: "player", ID: member, Pos: 3},
	}, leader); err == nil {
		t.Fatal("expected error for dup pos")
	}
	if _, err := normalizePartyFormation([]PartyFormationEntry{{Kind: "pet", ID: pet, Pos: 9}}, leader); err == nil {
		t.Fatal("expected error for pos out of range")
	}
}
