package effects

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBlockSetWritesVersionedEnvelopeAndReadsLegacyProjection(t *testing.T) {
	blocks := BlockSet{{
		ID: "EFFECT_01", Trigger: TriggerOnAction, Target: Target{Selector: TargetSingle},
		Operations: []Operation{{Op: OpDamage, SubType: "FLAT", Value: 10}},
	}}
	raw, err := json.Marshal(blocks)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"schema_version":2`) || !strings.Contains(string(raw), `"blocks":[`) {
		t.Fatalf("BlockSet did not marshal as a v2 envelope: %s", raw)
	}
	var roundTrip BlockSet
	if err := json.Unmarshal(raw, &roundTrip); err != nil || len(roundTrip) != 1 || roundTrip[0].ID != "EFFECT_01" {
		t.Fatalf("v2 round trip = %+v, err=%v", roundTrip, err)
	}
	legacy := []byte(`[{"id":"legacy-key:0:0","trigger":"ON_ACTION","target":{"selector":"TARGET"},"operations":[{"op":"DAMAGE","value":3}]}]`)
	if err := json.Unmarshal(legacy, &roundTrip); err != nil || len(roundTrip) != 1 || roundTrip[0].ID != "legacy-key:0:0" {
		t.Fatalf("legacy projection read = %+v, err=%v", roundTrip, err)
	}
}

func TestBlockSetRejectsUnsupportedEnvelopeVersion(t *testing.T) {
	var blocks BlockSet
	if err := json.Unmarshal([]byte(`{"schema_version":3,"blocks":[]}`), &blocks); err == nil {
		t.Fatal("unsupported Block v2 envelope version was accepted")
	}
}
