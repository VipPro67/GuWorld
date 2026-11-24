package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStatusListUnmarshalLegacyStrings(t *testing.T) {
	var sl StatusList
	if err := json.Unmarshal([]byte(`["severely_wounded"]`), &sl); err != nil {
		t.Fatalf("legacy unmarshal: %v", err)
	}
	if len(sl) != 1 || sl[0].ID != "severely_wounded" {
		t.Fatalf("unexpected result: %+v", sl)
	}
	if sl[0].ExpiresAt != nil {
		t.Fatal("legacy status should have no expiry")
	}
}

func TestStatusListUnmarshalObjects(t *testing.T) {
	var sl StatusList
	raw := `[{"id":"poisoned","expires_at":"2099-01-01T00:00:00Z","source":"combat","value":2},{"id":"blessed"}]`
	if err := json.Unmarshal([]byte(raw), &sl); err != nil {
		t.Fatalf("object unmarshal: %v", err)
	}
	if len(sl) != 2 || sl[0].ID != "poisoned" || sl[0].Source != "combat" || sl[0].Value != 2 {
		t.Fatalf("unexpected result: %+v", sl)
	}
	if sl[1].ID != "blessed" || sl[1].ExpiresAt != nil {
		t.Fatalf("unexpected second entry: %+v", sl[1])
	}
}

func TestStatusListRoundTrip(t *testing.T) {
	sl := StatusList{{ID: "HASTE", Source: "Tiên Cổ Ốc", SourceID: "gu-instance-1", EffectID: "haste:0:0", Context: "world", StackPolicy: "STACK", Value: 15}}
	raw, err := json.Marshal(sl)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back StatusList
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal own output: %v", err)
	}
	if len(back) != 1 || back[0].ID != "HASTE" || back[0].SourceID != "gu-instance-1" || back[0].EffectID != "haste:0:0" || back[0].Context != "world" || back[0].StackPolicy != "STACK" || back[0].Source != "Tiên Cổ Ốc" {
		t.Fatalf("round trip mismatch: %+v", back)
	}
}

func TestStatusListRoundTripsStackingPolicy(t *testing.T) {
	var statuses StatusList
	statuses.AddStacking("sinh_khi_tan", "GU_KILL_ENERGY", 12*time.Hour, -0.4, StatusStackingMetadata{
		CurrentTier: 2, MaxStacks: 2, DurationHours: 12, DispelRule: "CLEAR_ALL",
		TierScaling: []StatusStackTier{{Tier: 1, StatPercent: -15}, {Tier: 2, StatPercent: -40}},
	})
	raw, err := json.Marshal(statuses)
	if err != nil {
		t.Fatal(err)
	}
	var restored StatusList
	if err := json.Unmarshal(raw, &restored); err != nil {
		t.Fatal(err)
	}
	if len(restored) != 1 || restored[0].Stacking == nil {
		t.Fatalf("stacking metadata was lost during persistence round trip: %+v", restored)
	}
	config := restored[0].Stacking
	if config.CurrentTier != 2 || config.MaxStacks != 2 || config.DurationHours != 12 || config.DispelRule != "CLEAR_ALL" || len(config.TierScaling) != 2 || config.TierScaling[1].StatPercent != -40 {
		t.Fatalf("stacking metadata mismatch after round trip: %+v", config)
	}
}

func TestStatusListAddDedupe(t *testing.T) {
	sl := StatusList{}
	sl.Add("poisoned", "combat", 30*time.Minute, 0)
	sl.Add("poisoned", "admin", 0, 5)
	if len(sl) != 1 {
		t.Fatalf("expected dedupe, got %d entries", len(sl))
	}
	if sl[0].Source != "admin" || sl[0].Value != 5 {
		t.Fatalf("refresh failed: %+v", sl[0])
	}
	if sl[0].ExpiresAt != nil {
		t.Fatal("duration 0 should be permanent")
	}
}

func TestStatusListApplyEffectLifecycle(t *testing.T) {
	var statuses StatusList
	statuses.ApplyEffect("HASTE", "first", "gu-a", "block:0:0", "world", "", time.Minute, 10)
	firstExpiry := *statuses[0].ExpiresAt
	statuses.ApplyEffect("HASTE", "first", "gu-a", "block:0:0", "world", "", 2*time.Minute, 10)
	if len(statuses) != 1 || !statuses[0].ExpiresAt.After(firstExpiry) {
		t.Fatalf("same source did not refresh: %+v", statuses)
	}
	statuses.ApplyEffect("HASTE", "second", "gu-b", "block:0:0", "world", "STACK", time.Minute, 20)
	if len(statuses) != 2 {
		t.Fatalf("different sources should coexist: %+v", statuses)
	}
	statuses.ApplyEffect("HASTE", "strongest", "gu-c", "block:0:0", "world", "MAX", time.Minute, 30)
	if len(statuses) != 1 || statuses[0].SourceID != "gu-c" || statuses[0].Value != 30 {
		t.Fatalf("MAX winner lost provenance: %+v", statuses)
	}
	statuses.ApplyEffect("HASTE", "ignored", "gu-d", "block:0:0", "world", "IGNORE", time.Minute, 40)
	if len(statuses) != 1 {
		t.Fatalf("IGNORE replaced active status: %+v", statuses)
	}
	statuses.ApplyEffect("HASTE", "replacement", "gu-e", "block:0:0", "world", "REPLACE", time.Minute, 5)
	if len(statuses) != 1 || statuses[0].SourceID != "gu-e" {
		t.Fatalf("REPLACE did not replace same-id statuses: %+v", statuses)
	}
	statuses.ApplyEffect("HASTE", "minimum", "gu-f", "block:0:0", "world", "MIN", time.Minute, 3)
	if statuses[0].SourceID != "gu-f" || statuses[0].Value != 3 {
		t.Fatalf("MIN winner lost provenance: %+v", statuses)
	}
}

func TestStatusListWinnerPolicyIgnoresExpiredCandidates(t *testing.T) {
	for _, policy := range []string{"MAX", "MIN"} {
		t.Run(policy, func(t *testing.T) {
			past := time.Now().Add(-time.Minute)
			statuses := StatusList{{
				ID: "HASTE", ExpiresAt: &past, SourceID: "expired", EffectID: "effect", Context: "world",
				Value: map[string]float64{"MAX": 100, "MIN": -100}[policy], StackPolicy: policy,
			}}
			statuses.ApplyEffect("HASTE", "new", "new-source", "effect", "world", policy, time.Minute, 10)
			if len(statuses) != 1 || statuses[0].SourceID != "new-source" || statuses[0].Value != 10 {
				t.Fatalf("expired %s candidate blocked new status: %+v", policy, statuses)
			}
		})
	}
}

func TestStatusListHasRemoveExpire(t *testing.T) {
	sl := StatusList{}
	sl.Add("poisoned", "combat", time.Hour, 0)
	past := time.Now().Add(-time.Minute)
	sl = append(sl, StatusInstance{ID: "blessed", ExpiresAt: &past, Source: "admin"})

	if !sl.Has("poisoned") || !sl.Has("blessed") {
		t.Fatal("Has should find both entries before expiry cleanup")
	}

	active := sl.Active()
	if len(active) != 1 || active[0].ID != "poisoned" {
		t.Fatalf("Active should filter expired: %+v", active)
	}

	if !sl.Expire(time.Now()) {
		t.Fatal("Expire should report change")
	}
	if sl.Has("blessed") || !sl.Has("poisoned") {
		t.Fatalf("Expire removed wrong entry: %+v", sl)
	}
	if sl.Expire(time.Now()) {
		t.Fatal("second Expire should be a no-op")
	}

	sl.Remove("poisoned")
	if len(sl) != 0 {
		t.Fatalf("Remove failed: %+v", sl)
	}
}

func TestStatusListScanValue(t *testing.T) {
	sl := StatusList{{ID: "severely_wounded"}}
	v, err := sl.Value()
	if err != nil {
		t.Fatalf("Value: %v", err)
	}
	var back StatusList
	if err := back.Scan(v); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(back) != 1 || back[0].ID != "severely_wounded" {
		t.Fatalf("scan/value mismatch: %+v", back)
	}
	if err := back.Scan(nil); err != nil {
		t.Fatalf("Scan nil should not error: %v", err)
	}
	if len(back) != 0 {
		t.Fatalf("Scan nil should reset: %+v", back)
	}
}
