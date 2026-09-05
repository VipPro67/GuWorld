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
