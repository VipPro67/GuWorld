package handlers

import (
	"testing"

	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/simulator"
)

func TestLoadCombatKillerMoves_NilAndEmpty(t *testing.T) {
	player := &models.Player{
		Account: models.Account{ID: uuid.New()},
	}

	// 1. tx == nil returns nil, nil
	moves, err := loadCombatKillerMoves(nil, player, nil)
	if err != nil || moves != nil {
		t.Fatalf("expected nil, nil for nil tx, got %v, %v", moves, err)
	}

	// 2. explicit loadout with empty KillerMoveIds returns nil, nil (no killer moves used)
	emptyLoadout := &simulator.CombatLoadout{
		KillerMoveIds: []string{},
	}
	moves, err = loadCombatKillerMoves(nil, player, emptyLoadout)
	if err != nil || moves != nil {
		t.Fatalf("expected nil, nil for empty KillerMoveIds, got %v, %v", moves, err)
	}

	// 3. no loadout, empty CombatSettings.KillerMovePriorities returns nil, nil
	player.CombatSettings = models.AutoCombatConfig{}
	moves, err = loadCombatKillerMoves(nil, player, nil)
	if err != nil || moves != nil {
		t.Fatalf("expected nil, nil for unconfigured CombatSettings, got %v, %v", moves, err)
	}
}

func TestAutoCombatConfig_KillerMovePrioritiesScan(t *testing.T) {
	// Verify that Scan and unmarshal correctly populates KillerMovePriorities
	rawJSON := `{"auto_heal_hp":true,"heal_threshold":40,"killer_move_priorities":[{"gu_id":"km_123","priority":1,"use_condition":"hp_below","condition_value":50}]}`
	var cfg models.AutoCombatConfig
	if err := cfg.Scan([]byte(rawJSON)); err != nil {
		t.Fatalf("unexpected Scan error: %v", err)
	}
	if len(cfg.KillerMovePriorities) != 1 {
		t.Fatalf("expected 1 KillerMovePriority, got %d", len(cfg.KillerMovePriorities))
	}
	if cfg.KillerMovePriorities[0].GuID != "km_123" || cfg.KillerMovePriorities[0].UseCondition != "hp_below" {
		t.Fatalf("unexpected tactic values: %+v", cfg.KillerMovePriorities[0])
	}
	// Verify KmPriorities alias fallback
	aliasJSON := `{"km_priorities":[{"gu_id":"km_alias","priority":2,"use_condition":"always","condition_value":0}]}`
	var aliasCfg models.AutoCombatConfig
	if err := aliasCfg.Scan([]byte(aliasJSON)); err != nil {
		t.Fatalf("unexpected Scan error: %v", err)
	}
	if len(aliasCfg.KillerMovePriorities) != 1 || aliasCfg.KillerMovePriorities[0].GuID != "km_alias" {
		t.Fatalf("expected alias fallback to KillerMovePriorities, got %+v", aliasCfg.KillerMovePriorities)
	}
}
