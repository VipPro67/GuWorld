package handlers

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
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

func TestHandleWorldWSCommandRejectsMalformedChatSend(t *testing.T) {
	status, body := HandleWorldWSCommand(uuid.New(), "chat.send", json.RawMessage(`{"content":`))
	if status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", status, http.StatusBadRequest)
	}
	if !json.Valid(body) {
		t.Fatalf("invalid JSON response: %s", body)
	}
}

