package models

import (
	"testing"
)

func TestGuPersistentHitStatusFromHook(t *testing.T) {
	var effects GuEffects
	raw := `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10}]},{"id":"SINH_KHI_TAN_HIT","trigger":"ON_HIT","target":{"selector":"TARGET"},"operations":[{"op":"UNIQUE","code":"PERSIST_STATUS_ON_HIT","params":{"status_key":"sinh_khi_tan","pvp_only":true}}]}]`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}
	spec := GuDictionaryItem{ID: "GU_KILL_ENERGY", Rank: 9, Effects: effects}
	key, pvpOnly, ok := GuPersistentHitStatus(spec)
	if !ok {
		t.Fatal("expected persistent hit status from PERSIST_STATUS_ON_HIT hook")
	}
	if key != "sinh_khi_tan" {
		t.Fatalf("status key = %q, want sinh_khi_tan", key)
	}
	if !pvpOnly {
		t.Fatal("expected pvp_only=true")
	}
}

func TestGuPersistentHitStatusAbsent(t *testing.T) {
	var effects GuEffects
	raw := `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10}]}]`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}
	spec := GuDictionaryItem{ID: "GU_PLAIN", Rank: 5, Effects: effects}
	if _, _, ok := GuPersistentHitStatus(spec); ok {
		t.Fatal("Gu without hook must report no persistent status")
	}
}
