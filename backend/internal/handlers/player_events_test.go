package handlers

import (
	"testing"
)

func TestDecodeActiveCombatTreatsDatabaseNullAsNoCombat(t *testing.T) {
	for _, raw := range [][]byte{nil, {}, []byte("null"), []byte("{}")} {
		combat, err := decodeActiveCombat(raw)
		if err != nil {
			t.Fatalf("decodeActiveCombat(%q) returned error: %v", raw, err)
		}
		if combat != nil {
			t.Fatalf("decodeActiveCombat(%q) = %#v, want nil", raw, combat)
		}
	}
}

func TestDecodeActiveCombatRejectsInvalidJSON(t *testing.T) {
	if _, err := decodeActiveCombat([]byte("{")); err == nil {
		t.Fatal("decodeActiveCombat accepted invalid JSON")
	}
}
