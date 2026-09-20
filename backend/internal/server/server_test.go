package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type memorySyncer struct {
	buf bytes.Buffer
}

func (m *memorySyncer) Write(p []byte) (n int, err error) {
	return m.buf.Write(p)
}

func (m *memorySyncer) Sync() error {
	return nil
}

func TestRequestLogging_SuccessInfoLevel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mem := &memorySyncer{}
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), mem, zapcore.InfoLevel)
	origLogger := logger.L
	logger.L = zap.New(core)
	defer func() {
		logger.L = origLogger
	}()

	r := gin.New()
	r.Use(requestLogging())
	r.Use(monitor.Middleware())

	r.GET("/test-success", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest(http.MethodGet, "/test-success", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	lines := strings.Split(strings.TrimSpace(mem.buf.String()), "\n")
	var completedLog map[string]interface{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			continue
		}
		if parsed["msg"] == "http_request_completed" {
			completedLog = parsed
		}
	}

	if completedLog == nil {
		t.Fatal("expected http_request_completed log at InfoLevel for status 200")
	}
	if completedLog["level"] != "info" {
		t.Errorf("expected level info, got %v", completedLog["level"])
	}
}

func TestRequestLogging_ErrorResponseBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mem := &memorySyncer{}
	encCfg := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		MessageKey:     "msg",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), mem, zapcore.DebugLevel)
	origLogger := logger.L
	logger.L = zap.New(core)
	defer func() {
		logger.L = origLogger
	}()

	r := gin.New()
	r.Use(requestLogging())
	r.Use(monitor.Middleware())

	r.POST("/test-error", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failure"})
	})

	body := bytes.NewBufferString(`{"x":5003,"y":5004}`)
	req, _ := http.NewRequest(http.MethodPost, "/test-error", body)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}

	lines := strings.Split(strings.TrimSpace(mem.buf.String()), "\n")
	var errorLog, completedLog map[string]interface{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(line), &parsed); err != nil {
			continue
		}
		if parsed["msg"] == "http_request_error" {
			errorLog = parsed
		}
		if parsed["msg"] == "http_request_completed" {
			completedLog = parsed
		}
	}

	if errorLog == nil {
		t.Fatal("expected http_request_error log")
	}
	if completedLog == nil {
		t.Fatal("expected http_request_completed log")
	}

	// Verify http_request_error
	if errorLog["trace_id"] == nil || errorLog["trace_id"] == "" {
		t.Errorf("expected non-empty trace_id in http_request_error, got %v", errorLog["trace_id"])
	}
	if errorLog["error"] != "Transaction commit failure" {
		t.Errorf("expected error field 'Transaction commit failure', got %v", errorLog["error"])
	}
	if errorLog["response_body"] == nil {
		t.Error("expected response_body in http_request_error")
	}
	if errorLog["request_body"] == nil {
		t.Error("expected request_body in http_request_error")
	}

	// Verify http_request_completed
	if completedLog["trace_id"] == nil || completedLog["trace_id"] == "" {
		t.Errorf("expected non-empty trace_id in http_request_completed, got %v", completedLog["trace_id"])
	}
	if completedLog["error"] != "Transaction commit failure" {
		t.Errorf("expected error field in http_request_completed, got %v", completedLog["error"])
	}
	if completedLog["response_body"] == nil {
		t.Error("expected response_body in http_request_completed")
	}
	if completedLog["request_body"] == nil {
		t.Error("expected request_body in http_request_completed")
	}
}
