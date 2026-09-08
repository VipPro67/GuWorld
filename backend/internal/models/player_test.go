package models

import (
	"encoding/json"
	"testing"
)

func TestCombatState_IsEmpty(t *testing.T) {
	var nilState *CombatState
	if !nilState.IsEmpty() {
		t.Errorf("Expected nil CombatState to be empty")
	}

	emptyState := &CombatState{}
	if !emptyState.IsEmpty() {
		t.Errorf("Expected &CombatState{} to be empty")
	}

	activeState := &CombatState{
		EnemyName: "Bọ Cạp Quái",
	}
	if activeState.IsEmpty() {
		t.Errorf("Expected CombatState with EnemyName to not be empty")
	}
}

func TestCombatState_Value(t *testing.T) {
	var nilState *CombatState
	val, err := nilState.Value()
	if err != nil || val != nil {
		t.Errorf("Expected nil Value for nil CombatState, got %v, err: %v", val, err)
	}

	emptyState := &CombatState{}
	val, err = emptyState.Value()
	if err != nil || val != nil {
		t.Errorf("Expected nil Value for empty CombatState, got %v, err: %v", val, err)
	}

	activeState := &CombatState{
		EnemyName: "Wild Beast",
		EnemyHP:   100,
	}
	val, err = activeState.Value()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	strVal, ok := val.(string)
	if !ok {
		t.Fatalf("Expected string driver.Value, got %T", val)
	}
	var check CombatState
	if err := json.Unmarshal([]byte(strVal), &check); err != nil {
		t.Fatalf("Failed to unmarshal string Value: %v", err)
	}
	if check.EnemyName != "Wild Beast" {
		t.Errorf("Expected EnemyName Wild Beast, got %s", check.EnemyName)
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
}
