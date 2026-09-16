package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
)

func TestTeleportToRegionDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/explore/teleport", handlers.TeleportToRegion)

	req, _ := http.NewRequest("POST", "/api/explore/teleport", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}

	expectedErr := "Cơ chế dịch chuyển bằng tiền (Linh Thạch / Nguyên Thạch) đã bị hủy bỏ. Hãy sử dụng Cổ trùng có khả năng dịch chuyển."
	if resp["error"] != expectedErr {
		t.Fatalf("expected error message %q, got %q", expectedErr, resp["error"])
	}
}
