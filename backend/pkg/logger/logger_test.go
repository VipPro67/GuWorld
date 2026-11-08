package logger

import (
	"os"
	"testing"
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
