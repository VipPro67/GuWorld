package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestValidateTradeSource(t *testing.T) {
	sender := models.Player{WorldState: models.WorldState{CurrentRegion: "region-a", X: 10, Y: 10}}
	receiver := models.Player{WorldState: models.WorldState{CurrentRegion: "region-a", X: 20, Y: 10}}

	if err := validateTradeSource(sender, receiver, "map"); err != nil {
		t.Fatalf("nearby map trade should not require communication Gus: %v", err)
	}

	receiver.X = 21
	if err := validateTradeSource(sender, receiver, "map"); err == nil {
		t.Fatal("map trade beyond 10 tiles should fail")
	}

	receiver.X = 10
	receiver.CurrentRegion = "region-b"
	if err := validateTradeSource(sender, receiver, "map"); err == nil {
		t.Fatal("map trade across regions should fail")
	}

	receiver.CurrentRegion = sender.CurrentRegion
	if err := validateTradeSource(sender, receiver, "chat"); err == nil {
		t.Fatal("chat trade without communication Gus should fail even when nearby")
	}
}
