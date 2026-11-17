package logger

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestInitConsoleOnly(t *testing.T) {
	err := Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	if L == nil || S == nil {
		t.Fatalf("Expected L and S to be non-nil")
	}

	S.Info("Test console log entry")
	_ = Sync()
}

func TestInitJSONMode(t *testing.T) {
	os.Setenv("JSON_LOGS", "true")
	defer os.Unsetenv("JSON_LOGS")

	err := Init()
	if err != nil {
		t.Fatalf("Init JSON mode failed: %v", err)
	}

	S.Info("Test JSON log entry")
	_ = Sync()
}

func TestContextLogger(t *testing.T) {
	ctx := WithTraceID(nil, "trace-abc-123")
	if TraceIDFromContext(ctx) != "trace-abc-123" {
		t.Fatalf("Expected trace-abc-123 for nil parent fallback, got %s", TraceIDFromContext(ctx))
	}

	if TraceIDFromContext(nil) != "" {
		t.Fatalf("Expected empty trace ID for nil context")
	}

	ctx = WithTraceID(t.Context(), "trace-xyz-789")
	if TraceIDFromContext(ctx) != "trace-xyz-789" {
		t.Fatalf("Expected trace-xyz-789, got %s", TraceIDFromContext(ctx))
	}

	ctx = WithPlayerID(ctx, "player-456")
	if PlayerIDFromContext(ctx) != "player-456" {
		t.Fatalf("Expected player-456, got %s", PlayerIDFromContext(ctx))
	}

	clog := FromContext(ctx)
	if clog == nil {
		t.Fatalf("Expected non-nil SugaredLogger")
	}
	clog.Info("Context logger test message")

	llog := LFromContext(ctx)
	if llog == nil {
		t.Fatalf("Expected non-nil zap.Logger")
	}
	llog.Info("Context zap logger test message")
}

func TestErrorDetailsPreservesWrappedCauses(t *testing.T) {
	root := errors.New("database unavailable")
	wrapped := fmt.Errorf("save player: %w", fmt.Errorf("begin transaction: %w", root))
	chain, cause := ErrorDetails(wrapped)
	if len(chain) != 3 || chain[2] != "database unavailable" {
		t.Fatalf("error chain = %#v", chain)
	}
	if cause != "database unavailable" {
		t.Fatalf("root cause = %q, want database unavailable", cause)
	}
}

func TestErrorLoggingAddsStructuredRootCause(t *testing.T) {
	mem := &memorySyncer{}
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{LevelKey: "level", MessageKey: "msg", EncodeLevel: zapcore.LowercaseLevelEncoder})
	base := zap.New(errorDetailsCore{Core: zapcore.NewCore(enc, mem, zapcore.DebugLevel)})
	originalL, originalS := L, S
	L, S = base, newSugaredLogger(base)
	defer func() { L, S = originalL, originalS }()

	root := errors.New("deadlock detected")
	S.Errorf("save failed: %v", fmt.Errorf("retry operation: %w", root))
	var event map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(mem.Bytes()), &event); err != nil {
		t.Fatalf("decode log event: %v", err)
	}
	if event["root_cause"] != "deadlock detected" {
		t.Fatalf("root_cause = %v", event["root_cause"])
	}
	chain, ok := event["error_chain"].([]interface{})
	if !ok || len(chain) != 2 || !strings.Contains(chain[0].(string), "retry operation") {
		t.Fatalf("error_chain = %#v", event["error_chain"])
	}
}

func TestZapErrorFieldAddsStructuredRootCause(t *testing.T) {
	mem := &memorySyncer{}
	enc := zapcore.NewJSONEncoder(zapcore.EncoderConfig{LevelKey: "level", MessageKey: "msg", EncodeLevel: zapcore.LowercaseLevelEncoder})
	base := zap.New(errorDetailsCore{Core: zapcore.NewCore(enc, mem, zapcore.DebugLevel)})
	base.Error("operation failed", zap.Error(fmt.Errorf("query player: %w", errors.New("connection reset"))))
	var event map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(mem.Bytes()), &event); err != nil {
		t.Fatalf("decode log event: %v", err)
	}
	if event["root_cause"] != "connection reset" {
		t.Fatalf("root_cause = %v", event["root_cause"])
	}
}

type memorySyncer struct{ bytes.Buffer }

func (memorySyncer) Sync() error { return nil }
