package httperr

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func init() {
	gin.SetMode(gin.TestMode)
	_ = logger.Init()
}

func TestAppErrorUnwrap(t *testing.T) {
	root := errors.New("connection failed")
	appErr := Internal("Lỗi kết nối", root)

	if !errors.Is(appErr, root) {
		t.Fatalf("Expected errors.Is to match root error")
	}
	if appErr.Error() != "Lỗi kết nối: connection failed" {
		t.Fatalf("Unexpected Error() message: %s", appErr.Error())
	}
}

func TestRespondWithAppError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	c.Request = req
	c.Set("trace_id", "trace-test-123")

	err := BadRequest("Tham số không hợp lệ", errors.New("field missing"))
	Respond(c, err)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if body["error"] != "Tham số không hợp lệ" {
		t.Fatalf("Expected error message 'Tham số không hợp lệ', got '%v'", body["error"])
	}
	if body["trace_id"] != "trace-test-123" {
		t.Fatalf("Expected trace_id 'trace-test-123', got '%v'", body["trace_id"])
	}
	if len(c.Errors) == 0 {
		t.Fatalf("Expected c.Errors to contain the error")
	}
}

func TestRespondWithGenericErrorProduction(t *testing.T) {
	os.Setenv("PROD_ENV", "production")
	os.Setenv("DEBUG_ERRORS", "false")
	defer os.Unsetenv("PROD_ENV")
	defer os.Unsetenv("DEBUG_ERRORS")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	c.Request = req
	c.Set("trace_id", "trace-prod-999")

	rawErr := errors.New("pq: syntax error at or near SELECT")
	Respond(c, rawErr)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("Expected status 500, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	// In production, raw SQL must be masked
	if body["error"] != "Lỗi xử lý hệ thống" {
		t.Fatalf("Expected sanitized message 'Lỗi xử lý hệ thống', got '%v'", body["error"])
	}
	if body["trace_id"] != "trace-prod-999" {
		t.Fatalf("Expected trace-prod-999, got '%v'", body["trace_id"])
	}
	if _, exists := body["debug"]; exists {
		t.Fatalf("Expected debug field to be omitted in production")
	}
}

func TestRespondWithGenericErrorDebug(t *testing.T) {
	os.Setenv("DEBUG_ERRORS", "true")
	defer os.Unsetenv("DEBUG_ERRORS")

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
	c.Request = req
	c.Set("trace_id", "trace-dbg-456")

	rawErr := errors.New("pq: relation does not exist")
	Respond(c, rawErr)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("Failed to parse response body: %v", err)
	}

	if body["debug"] != "pq: relation does not exist" {
		t.Fatalf("Expected debug field with raw error, got '%v'", body["debug"])
	}
}
