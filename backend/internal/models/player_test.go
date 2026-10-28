package models

import (
	"testing"
)

func TestAutoCombatConfigNormalizesRealtimeControls(t *testing.T) {
	settings := AutoCombatConfig{
		RealtimeLeftClickSlot:    8,
		RealtimeRightClickAction: "unknown",
		RealtimeRightClickSlot:   9,
		RealtimeFarmRange:        100,
		RealtimeFarmStopHP:       120,
	}
	settings.NormalizeRealtimeSettings()
	if settings.RealtimeLeftClickSlot != 0 || settings.RealtimeRightClickAction != "attack" || settings.RealtimeRightClickSlot != 0 {
		t.Fatalf("invalid click bindings were not normalized: %+v", settings)
	}
	if settings.RealtimeFarmRange != 32 || settings.RealtimeFarmStopHP != 90 {
		t.Fatalf("farm limits were not clamped: range=%v stopHP=%v", settings.RealtimeFarmRange, settings.RealtimeFarmStopHP)
	}
}

func TestTruncateJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid object", `{"a":1}`, `{"a":1}`},
		{"valid nested", `{"a":{"b":1}}`, `{"a":{"b":1}}`},
		{"trailing garbage", `{"a":1} garbage`, `{"a":1}`},
		{"trailing newline and garbage", `{"a":1}\ngarbage`, `{"a":1}`},
		{"array", `[1,2,3]`, `[1,2,3]`},
		{"array with trailing", `[1,2] xxx`, `[1,2]`},
		{"empty braces", `{}`, `{}`},
		{"with leading whitespace", `  {"a":1}xx`, `{"a":1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := string(truncateJSON([]byte(tt.input)))
			if result != tt.expected {
				t.Errorf("truncateJSON(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDaoMarksMap_ScanCorrupted(t *testing.T) {
	// Simulate corrupted dao_marks data (trailing non-whitespace bytes)
	corrupted := []byte(`{"strength":500}CORRUPTED_GARBAGE`)
	var dm DaoMarksMap
	if err := dm.Scan(corrupted); err != nil {
		t.Errorf("Expected graceful handling of corrupted data, got error: %v", err)
	}
	if dm["strength"] != 500 {
		t.Errorf("Expected strength=500, got %d", dm["strength"])
	}

	// Valid data should still work
	valid := []byte(`{"fire":1000,"water":500}`)
	var dm2 DaoMarksMap
	if err := dm2.Scan(valid); err != nil {
		t.Errorf("Unexpected error for valid data: %v", err)
	}
	if dm2["fire"] != 1000 || dm2["water"] != 500 {
		t.Errorf("Expected fire=1000, water=500, got %v", dm2)
	}

	// Floats and scientific notation in JSON should be gracefully parsed to int64
	floatJSON := []byte(`{"lightning": 1234567.89, "heaven": 1e10, "earth": 9999999999}`)
	var dm3 DaoMarksMap
	if err := dm3.Scan(floatJSON); err != nil {
		t.Errorf("Unexpected error parsing float JSON: %v", err)
	}
	if dm3["lightning"] != 1234567 || dm3["heaven"] != 10000000000 || dm3["earth"] != 9999999999 {
		t.Errorf("Expected lightning=1234567, heaven=10000000000, earth=9999999999, got %v", dm3)
	}
}
