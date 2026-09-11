package worker

import (
	"encoding/json"
	"testing"
)

func TestLegacyPlayerMovementPayloadCarriesVersionFence(t *testing.T) {
	payload, err := json.Marshal(PlayerMovementPayload{
		ID:            "player-1",
		X:             10,
		Y:             20,
		CurrentRegion: "region-1",
		Version:       7,
	})
	if err != nil {
		t.Fatalf("marshal movement payload: %v", err)
	}

	var decoded PlayerMovementPayload
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal movement payload: %v", err)
	}
	if decoded.Version != 7 {
		t.Fatalf("version fence lost during serialization: got %d", decoded.Version)
	}
}
