package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

func TestHandleWorldWSCommandRejectsUnknownCommand(t *testing.T) {
	status, body := HandleWorldWSCommand(uuid.New(), "world.not_real", json.RawMessage(`{}`))
	if status != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", status, http.StatusNotFound)
	}
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if result["error"] == "" {
		t.Fatal("expected an error message")
	}
}

func TestHandleWorldWSCommandRejectsMalformedRectangle(t *testing.T) {
	status, body := HandleWorldWSCommand(uuid.New(), "world.cells_rect", json.RawMessage(`{"min_x":`))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if !json.Valid(body) {
		t.Fatalf("invalid JSON response: %s", body)
	}
}

func TestHandleWorldWSCommandRejectsMalformedMovePath(t *testing.T) {
	status, body := HandleWorldWSCommand(uuid.New(), "world.move_path", json.RawMessage(`{"x": "invalid"}`))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if !json.Valid(body) {
		t.Fatalf("invalid JSON response: %s", body)
	}
}

func TestHandleWorldWSCommandBlocksLegacyMovementForRealtimeOwner(t *testing.T) {
	playerID := uuid.New()
	middleware.SetRealtimeOwnershipChecker(func(id uuid.UUID) bool { return id == playerID })
	t.Cleanup(func() { middleware.SetRealtimeOwnershipChecker(nil) })
	status, body := HandleWorldWSCommand(playerID, "world.move_path", json.RawMessage(`{"x":10,"y":10}`))
	if status != http.StatusConflict {
		t.Fatalf("status = %d, want %d", status, http.StatusConflict)
	}
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if result["code"] != "realtime_world_owns_character" {
		t.Fatalf("unexpected rejection code: %q", result["code"])
	}
}

func TestHandleWorldWSCommandRejectsMalformedChatSend(t *testing.T) {
	status, body := HandleWorldWSCommand(uuid.New(), "chat.send", json.RawMessage(`{"content":`))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if !json.Valid(body) {
		t.Fatalf("invalid JSON response: %s", body)
	}
}
