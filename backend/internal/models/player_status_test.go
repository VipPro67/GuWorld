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
	sl := StatusList{{ID: "severely_wounded"}}
	raw, err := json.Marshal(sl)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var back StatusList
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("unmarshal own output: %v", err)
	}
	if len(back) != 1 || back[0].ID != "severely_wounded" {
		t.Fatalf("round trip mismatch: %+v", back)
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
