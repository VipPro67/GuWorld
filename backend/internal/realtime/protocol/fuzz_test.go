package protocol

import (
	"encoding/json"
	"testing"
)

func FuzzDecodeClientMessage(f *testing.F) {
	f.Add([]byte(`{"v":1,"type":"world.input","request_id":"a","payload":{"seq":1,"move":{"x":0.5,"y":0.5}}}`))
	f.Add([]byte(`{"v":1,"type":"world.join","payload":{"zone_id":"z","map_version":"m","content_version":"c"}}`))
	f.Add([]byte(`{"v":1,"type":"world.cast","payload":{"seq":2,"slot":2,"mode":"DIRECTION","aim_x":1,"aim_y":0}}`))
	f.Add([]byte(`{"v":1,"type":"world.input","payload":{"seq":0}}`))
	f.Add([]byte(`not-json`))
	f.Add([]byte(`{"v":1,"type":"world.input","payload":{"seq":1,"move":{"x":2,"y":0}}}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		envelope, message, err := DecodeClientMessage(data)
		if err != nil {
			return
		}
		if envelope.Version != Version {
			t.Fatalf("accepted unsupported version: %+v", envelope)
		}
		if message == nil {
			t.Fatal("accepted message without payload")
		}
	})
}

func TestStrictJoinRejectsIdentityAndUnknownFields(t *testing.T) {
	raw := []byte(`{"v":1,"type":"world.join","payload":{"zone_id":"z","map_version":"m","content_version":"c","player_id":"victim"}}`)
	if _, _, err := DecodeClientMessage(raw); err == nil {
		t.Fatal("join payload with client-supplied player_id was accepted")
	}
}

func TestRejectsOversizedAndNonFiniteLikeInput(t *testing.T) {
	huge := make([]byte, MaxMessageSize+1)
	for index := range huge {
		huge[index] = 'x'
	}
	if _, _, err := DecodeClientMessage(huge); err == nil {
		t.Fatal("oversized message accepted")
	}
	bad := []byte(`{"v":1,"type":"world.input","payload":{"seq":1,"move":{"x":2,"y":0}}}`)
	if _, _, err := DecodeClientMessage(bad); err == nil {
		t.Fatal("oversized movement vector accepted")
	}
}

func TestEnvelopeSizeDoesNotLeakUnknownPayload(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{"zone_id": "z", "map_version": "m", "content_version": "c"})
	raw, _ := json.Marshal(Envelope{Version: 1, Type: TypeJoin, Payload: payload})
	if _, _, err := DecodeClientMessage(raw); err != nil {
		t.Fatal(err)
	}
}
