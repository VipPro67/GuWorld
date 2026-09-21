package db

import (
	"testing"

	"github.com/lib/pq"
)

// LogGuHistory chỉ log Tiên Cổ rank>=6, bỏ qua rank thấp ngay cả khi có DB.
func TestLogGuHistoryDropsLowRank(t *testing.T) {
	if err := LogGuHistory(nil, "inst-1", "GU_WOLF", 5, "CREATED_LOOT", nil, nil, "x", nil); err != nil {
		t.Fatalf("low rank should be dropped without error, got %v", err)
	}
	if err := LogGuHistory(nil, "", "GU_WOLF", 9, "CREATED_LOOT", nil, nil, "x", nil); err != nil {
		t.Fatalf("empty instance should be dropped without error, got %v", err)
	}
}

func TestIsImmortalConflict(t *testing.T) {
	immortal := &pq.Error{Code: "23505", Constraint: "uq_player_gu_immortal"}
	if !IsImmortalConflict(immortal) {
		t.Errorf("expected uq_player_gu_immortal to be immortal conflict")
	}
	vault := &pq.Error{Code: "23505", Constraint: "uq_sect_vault_gu_immortal"}
	if !IsImmortalConflict(vault) {
		t.Errorf("expected uq_sect_vault_gu_immortal to be immortal conflict")
	}
	other := &pq.Error{Code: "23505", Constraint: "player_gu_pkey"}
	if IsImmortalConflict(other) {
		t.Errorf("pkey violation should not be immortal conflict")
	}
	if IsImmortalConflict(nil) {
		t.Errorf("nil should not be immortal conflict")
	}
}
