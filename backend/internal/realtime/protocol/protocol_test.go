package protocol

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func TestSnapshotSupportsBaselineDeltaAndRemovedEntities(t *testing.T) {
	want := Snapshot{
		Tick: 12, AckSeq: 8, BaselineID: 2, BaseTick: 10,
		Entities: []EntityState{{EntityID: "player-1", Revision: 3, X: 4, Y: 5, HP: 90}},
		Removed:  []string{"monster-1"},
	}
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got Snapshot
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if got.Tick != want.Tick || got.AckSeq != want.AckSeq || got.BaselineID != want.BaselineID || got.BaseTick != want.BaseTick || len(got.Entities) != 1 || len(got.Removed) != 1 || got.Removed[0] != "monster-1" {
		t.Fatalf("snapshot delta round trip mismatch: got=%+v want=%+v", got, want)
	}
}

func TestDecodeWorldInput(t *testing.T) {
	envelope, err := DecodeEnvelope([]byte(`{"v":1,"type":"world.input","request_id":"c-17","payload":{"seq":17,"move":{"x":0.6,"y":0.8}}}`))
	if err != nil {
		t.Fatal(err)
	}
	input, err := DecodePayload[Input](envelope)
	if err != nil || input.Validate() != nil || input.Seq != 17 {
		t.Fatalf("input=%+v, decode=%v", input, err)
	}
}

func TestDecodeBasicAttackRequiresTargetOrAimAndSequence(t *testing.T) {
	_, decoded, err := DecodeClientMessage([]byte(`{"v":1,"type":"world.attack","payload":{"seq":1,"target_id":"m:1"}}`))
	attack, ok := decoded.(Attack)
	if err != nil || !ok || attack.Seq != 1 || attack.TargetID != "m:1" {
		t.Fatalf("attack=%+v decoded=%T err=%v", attack, decoded, err)
	}
	_, decoded, err = DecodeClientMessage([]byte(`{"v":1,"type":"world.attack","payload":{"seq":2,"aim_x":1,"aim_y":0}}`))
	aimed, ok := decoded.(Attack)
	if err != nil || !ok || aimed.Seq != 2 || aimed.TargetID != "" || aimed.AimX != 1 {
		t.Fatalf("aimed attack=%+v decoded=%T err=%v", aimed, decoded, err)
	}
	for _, raw := range []string{
		`{"v":1,"type":"world.attack","payload":{"seq":0,"target_id":"m:1"}}`,
		`{"v":1,"type":"world.attack","payload":{"seq":1}}`,
		`{"v":1,"type":"world.attack","payload":{"seq":1,"target_id":""}}`,
		`{"v":1,"type":"world.attack","payload":{"seq":1,"aim_x":1000001}}`,
	} {
		if _, _, err := DecodeClientMessage([]byte(raw)); err == nil {
			t.Fatalf("accepted malformed attack: %s", raw)
		}
	}
}

func TestRejectMalformedWorldMessages(t *testing.T) {
	for _, raw := range []string{
		`{"v":2,"type":"world.input","payload":{}}`,
		`{"v":1,"type":"world.invalid","payload":{}}`,
		`{"v":1,"type":"world.input","payload":null}`,
		`{"v":1,"type":"world.input","payload":{},"player_id":"spoof"}`,
		strings.Repeat("x", MaxMessageSize+1),
	} {
		if _, err := DecodeEnvelope([]byte(raw)); err == nil {
			t.Fatalf("accepted: %.80s", raw)
		}
	}
	envelope, err := DecodeEnvelope([]byte(`{"v":1,"type":"world.input","payload":{"seq":1,"move":{"x":0,"y":1},"hp":999}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodePayload[Input](envelope); err == nil {
		t.Fatal("accepted unauthorized payload field")
	}
	if err := (Input{Seq: 1, Move: &Vector{X: math.Inf(1)}}).Validate(); err == nil {
		t.Fatal("accepted infinite vector")
	}
	if err := (Cast{Seq: 1, Slot: 7}).Validate(); err == nil {
		t.Fatal("accepted invalid slot")
	}
}

func TestClientBoundaryRejectsIncompleteJoinAndServerMessages(t *testing.T) {
	for _, raw := range []string{
		`{"v":1,"type":"world.join","payload":{}}`,
		`{"v":1,"type":"world.join","payload":{"zone_id":"demo","map_version":"m1","content_version":"c1","player_id":"spoof"}}`,
		`{"v":1,"type":"world.welcome","payload":{}}`,
		`{"v":1,"type":"world.input","payload":123}`,
	} {
		if _, _, err := DecodeClientMessage([]byte(raw)); err == nil {
			t.Fatalf("accepted client command: %s", raw)
		}
	}
	if _, _, err := DecodeClientMessage([]byte(`{"v":1,"type":"world.join","payload":{"zone_id":"demo","map_version":"m1","content_version":"c1"}}`)); err != nil {
		t.Fatal(err)
	}
}
