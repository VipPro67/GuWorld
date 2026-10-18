package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/match"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// matchInputRateLimit is per minute. The client sends an input frame ~30x/s, so
// the ceiling is set well above that to absorb jitter without allowing spam.
//
// ponytail: shared player rate limiter. Give the match channel its own budget if
// the world-map budget ever becomes contended.
const matchInputRateLimit = 3000

// ExecuteMatchInput accepts one realtime input frame. The client only sends
// intent; every cast and movement is re-validated on the next tick.
func ExecuteMatchInput(playerID uuid.UUID, payload json.RawMessage) (int, any) {
	if !middleware.AllowPlayer(playerID, matchInputRateLimit, time.Minute) {
		return http.StatusTooManyRequests, json.RawMessage(`{"error":"Đạo hữu thao tác quá nhanh."}`)
	}

	active, ok := match.Global.ForPlayer(playerID)
	if !ok {
		return http.StatusConflict, json.RawMessage(`{"error":"Không có trận đấu nào đang diễn ra."}`)
	}

	input, err := match.ParseInput(payload)
	if err != nil {
		return http.StatusBadRequest, json.RawMessage(`{"error":"Dữ liệu điều khiển không hợp lệ."}`)
	}
	if !active.SubmitInput(playerID, input) {
		return http.StatusConflict, json.RawMessage(`{"error":"Trận đấu đã kết thúc."}`)
	}
	return http.StatusOK, json.RawMessage(`{"ok":true}`)
}
