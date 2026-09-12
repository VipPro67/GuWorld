package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// HandleWorldWSCommand adapts the existing exploration handlers to the
// authenticated WebSocket command transport. Keeping one handler path ensures
// HTTP fallback and WebSocket commands have identical validation and results.
func HandleWorldWSCommand(playerID uuid.UUID, command string, payload json.RawMessage) (int, json.RawMessage) {
	method := http.MethodGet
	target := "/"
	var body []byte

	switch command {
	case "world.move_path":
		method = http.MethodPost
		target = "/api/explore/move-path"
		body = payload
	case "chat.send":
		method = http.MethodPost
		target = "/api/chat/bao-hoang/send"
		body = payload
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
		q := url.Values{}
		q.Set("min_x", fmt.Sprint(req.MinX))
		q.Set("min_y", fmt.Sprint(req.MinY))
		q.Set("max_x", fmt.Sprint(req.MaxX))
		q.Set("max_y", fmt.Sprint(req.MaxY))
		target = "/api/explore/cells/rect?" + q.Encode()
	case "world.objects":
		target = "/api/explore/objects"
	default:
		return http.StatusNotFound, json.RawMessage(`{"error":"Unknown world command"}`)
	}

	req := httptest.NewRequest(method, target, bytes.NewReader(body))
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = req
	c.Set("player_id", playerID)
	var limit gin.HandlerFunc
	switch command {
	case "world.move_path":
		limit = middleware.RateLimitByPlayer(120, time.Minute)
	case "world.cells_rect":
		limit = middleware.RateLimitByPlayer(240, time.Minute)
	case "world.objects":
		limit = middleware.RateLimitByPlayer(30, time.Minute)
	case "chat.send":
		limit = middleware.RateLimitByPlayer(30, time.Minute)
	}
	if limit != nil {
		limit(c)
		if c.IsAborted() {
			return recorder.Code, json.RawMessage(append([]byte(nil), bytes.TrimSpace(recorder.Body.Bytes())...))
		}
	}

	switch command {
	case "world.move_path":
		MovePlayerPath(c)
	case "chat.send":
		SendBaoHoangChat(c)
	case "world.cells_rect":
		GetCellsRect(c)
	case "world.objects":
		GetExploreObjects(c)
	}

	result := bytes.TrimSpace(recorder.Body.Bytes())
	if len(result) == 0 {
		result = []byte(`null`)
	}
	return recorder.Code, json.RawMessage(append([]byte(nil), result...))
}
