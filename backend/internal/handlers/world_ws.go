package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// HandleWorldWSCommand dispatches authenticated WebSocket commands directly to domain logic
// without fabricating HTTP requests or contexts.
func HandleWorldWSCommand(playerID uuid.UUID, command string, payload json.RawMessage) (int, json.RawMessage) {
	var limit int
	switch command {
	case "world.move_path":
		limit = 120
	case "world.cells_rect":
		limit = 240
	case "world.objects":
		limit = 30
	case "chat.send":
		limit = 30
	default:
		return http.StatusNotFound, json.RawMessage(`{"error":"Unknown world command"}`)
	}

	if !middleware.AllowPlayer(playerID, limit, time.Minute) {
		return http.StatusTooManyRequests, json.RawMessage(`{"error":"Đạo hữu thao tác quá nhanh. Vui lòng đợi rồi thử lại!"}`)
	}

	var status int
	var result any

	switch command {
	case "world.move_path":
		var req MoveRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return http.StatusBadRequest, json.RawMessage(`{"error":"Invalid move request"}`)
		}
		status, result = ExecuteMovePlayerPath(playerID, req)

	case "chat.send":
		var req BaoHoangRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			return http.StatusBadRequest, json.RawMessage(`{"error":"Content is required"}`)
		}
		status, result = ExecuteSendBaoHoangChat(context.Background(), playerID, req)

	case "world.cells_rect":
		var req struct {
			MinX int `json:"min_x"`
			MinY int `json:"min_y"`
			MaxX int `json:"max_x"`
			MaxY int `json:"max_y"`
		}
		if err := json.Unmarshal(payload, &req); err != nil {
			return http.StatusBadRequest, json.RawMessage(`{"error":"Invalid cells rectangle"}`)
		}
		status, result = ExecuteGetCellsRect(playerID, req.MinX, req.MinY, req.MaxX, req.MaxY)

	case "world.objects":
		status, result = ExecuteGetExploreObjects()
	}

	data, err := json.Marshal(result)
	if err != nil {
		return http.StatusInternalServerError, json.RawMessage(`{"error":"Failed to marshal response"}`)
	}
	return status, json.RawMessage(data)
}
