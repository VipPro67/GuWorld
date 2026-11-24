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
	status, ok := GuPersistentHitStatus(spec)
	if !ok {
		t.Fatal("expected persistent hit status from PERSIST_STATUS_ON_HIT hook")
	}
	if status.StatusKey != "sinh_khi_tan" {
		t.Fatalf("status key = %q, want sinh_khi_tan", status.StatusKey)
	}
	if !status.PvPOnly {
		t.Fatal("expected pvp_only=true")
	}
}

func TestGuPersistentHitStatusFromTypedOp(t *testing.T) {
	var effects GuEffects
	raw := `{"schema_version": 2, "blocks": [{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10}]},{"id":"SINH_KHI_TAN_HIT","trigger":"ON_HIT","target":{"selector":"TARGET"},"operations":[{"op":"APPLY_STACKING_STATUS","status_key":"sinh_khi_tan","max_stacks":2,"duration_hours":12,"dispel_rule":"CLEAR_ALL","pvp_only":true,"tier_scaling":[{"tier":1,"stat_percent":-15},{"tier":2,"stat_percent":-40}]}]}]}`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}
	spec := GuDictionaryItem{ID: "GU_KILL_ENERGY", Rank: 9, Effects: effects}
	status, ok := GuPersistentHitStatus(spec)
	if !ok {
		t.Fatal("expected persistent hit status from APPLY_STACKING_STATUS operation")
	}
	if status.StatusKey != "sinh_khi_tan" {
		t.Fatalf("status key = %q, want sinh_khi_tan", status.StatusKey)
	}
	if !status.PvPOnly {
		t.Fatal("expected pvp_only=true")
	}
	if !status.Configured || status.MaxStacks != 2 || status.DurationHours != 12 || status.DispelRule != "CLEAR_ALL" || len(status.TierScaling) != 2 || status.TierScaling[1].StatPercent != -40 {
		t.Fatalf("typed stacking config not preserved: %+v", status)
	}
}

func TestGuPersistentHitStatusAbsent(t *testing.T) {
	var effects GuEffects
	raw := `[{"id":"ACTIVE_SKILL","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","sub_type":"FLAT","value":10}]}]`
	if err := effects.Scan(raw); err != nil {
		t.Fatalf("scan block effects: %v", err)
	}
	spec := GuDictionaryItem{ID: "GU_PLAIN", Rank: 5, Effects: effects}
	if _, ok := GuPersistentHitStatus(spec); ok {
		t.Fatal("Gu without hook must report no persistent status")
	}
}
