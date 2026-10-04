package server

import (
	"encoding/json"
	"testing"
)

func TestWorldTargetedEventUsesClientEnvelope(t *testing.T) {
	raw := []byte(`{"event_id":"move-1","event_type":"monster_moved"}`)
	wrapped := worldEventEnvelope("world:combat", raw)
	var event struct {
		Type    string          `json:"type"`
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(wrapped, &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != "event" || event.Channel != "world:combat" || string(event.Data) != string(raw) {
		t.Fatalf("unexpected targeted wire event: %s", wrapped)
	}
}
