package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIsPlayerOnline_WSProvider(t *testing.T) {
	playerID := uuid.New()
	wsConnected := false

	// Inject custom WS provider
	SetWSConnectedProvider(func(id uuid.UUID) bool {
		if id == playerID {
			return wsConnected
		}
		return false
	})
	defer SetWSConnectedProvider(nil)

	// Initially offline
	if isPlayerOnline(playerID.String()) {
		t.Fatal("expected player to be offline")
	}

	// Player connects via WebSocket
	wsConnected = true
	if !isPlayerOnline(playerID.String()) {
		t.Fatal("expected player to be online via WebSocket connection")
	}

	// Player disconnects from WebSocket
	wsConnected = false
	if isPlayerOnline(playerID.String()) {
		t.Fatal("expected player to be offline after WS disconnect and no activity")
	}
}

func TestIsPlayerOnline_RecentActivity(t *testing.T) {
	playerID := uuid.New()

	// Ensure WS provider returns false
	SetWSConnectedProvider(func(id uuid.UUID) bool {
		return false
	})
	defer SetWSConnectedProvider(nil)

	// Initially offline
	if isPlayerOnline(playerID.String()) {
		t.Fatal("expected player to be offline initially")
	}

	// Player sends a ping/message -> touches activity
	TouchPlayerActivity(playerID)

	if !IsPlayerActiveRecently(playerID, 5*time.Minute) {
		t.Fatal("expected player to be active recently in memory")
	}

	// Now isPlayerOnline should return true via Tier 2 (recent activity)
	if !isPlayerOnline(playerID.String()) {
		t.Fatal("expected player to be online due to recent in-memory activity")
	}
}
