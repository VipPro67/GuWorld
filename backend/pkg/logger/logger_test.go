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
