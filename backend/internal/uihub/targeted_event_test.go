package uihub

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestTargetedEventPreservesPayloadAndRecipients(t *testing.T) {
	playerID := uuid.MustParse("6d17a711-195d-4ae7-8c0b-d57bd6fdc0b3")
	payload := json.RawMessage(`{"id":"` + playerID.String() + `","x":7,"y":9}`)
	wire, err := json.Marshal(TargetedEvent{Targets: []uuid.UUID{playerID}, Payload: payload})
	if err != nil {
		t.Fatalf("marshal targeted event: %v", err)
	}
	var decoded TargetedEvent
	if err := json.Unmarshal(wire, &decoded); err != nil {
		t.Fatalf("unmarshal targeted event: %v", err)
	}
	if len(decoded.Targets) != 1 || decoded.Targets[0] != playerID {
		t.Fatalf("targets = %v, want %s", decoded.Targets, playerID)
	}
	if string(decoded.Payload) != string(payload) {
		t.Fatalf("payload = %s, want %s", decoded.Payload, payload)
	}
}
