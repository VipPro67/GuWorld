package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestItemEffectRestoreMana(t *testing.T) {
	player := &models.Player{
		CurrentMana: 10,
		MaxMana:     50,
	}
	item := models.ItemSpec{
		Name:    "Rượu Thanh Trúc",
		Effects: models.ItemEffects{"restore_mana": 20},
	}

	msg, err := restoreManaEffect(player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.CurrentMana != 30 {
		t.Errorf("expected mana 30, got %v", player.CurrentMana)
	}
	if msg == "" {
		t.Error("expected non-empty success message")
	}

	// Test cap
	_, _ = restoreManaEffect(player, item) // 30 + 20 = 50
	_, _ = restoreManaEffect(player, item) // 50 + 20 = 70 -> cap at 50
	if player.CurrentMana != 50 {
		t.Errorf("expected capped mana 50, got %v", player.CurrentMana)
	}
}

func TestItemEffectRestoreHP(t *testing.T) {
	player := &models.Player{
		HP:     50,
		BaseHP: 100,
	}
	item := models.ItemSpec{
		Name:    "Ngọc Thạch",
		Effects: models.ItemEffects{"restore_hp": 30},
	}

	msg, err := restoreHPEffect(player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.HP != 80 {
		t.Errorf("expected HP 80, got %v", player.HP)
	}
	if msg == "" {
		t.Error("expected non-empty success message")
	}

	// Test cap
	_, _ = restoreHPEffect(player, item) // 80 + 30 = 110 -> cap at 100
	if player.HP != 100 {
		t.Errorf("expected capped HP 100, got %v", player.HP)
	}
}

func TestItemEffectMortalOnly(t *testing.T) {
	mortal := &models.Player{Rank: 3}
	immortal := &models.Player{Rank: 6}
	item := models.ItemSpec{
		Name: "Rượu Thanh Trúc",
	}

	_, err := mortalOnlyEffect(mortal, item)
	if err != nil {
		t.Errorf("mortal should be allowed to use, got error: %v", err)
	}

	_, err = mortalOnlyEffect(immortal, item)
	if err == nil {
		t.Error("immortal should not be allowed to use mortal_only item, but got no error")
	}
}

func TestItemEffectImmortalOnly(t *testing.T) {
	mortal := &models.Player{Rank: 3}
	immortal := &models.Player{Rank: 6}
	item := models.ItemSpec{
		Name: "Tiên Nguyên Thạch",
	}

	_, err := immortalOnlyEffect(immortal, item)
	if err != nil {
		t.Errorf("immortal should be allowed to use, got error: %v", err)
	}

	_, err = immortalOnlyEffect(mortal, item)
	if err == nil {
		t.Error("mortal should not be allowed to use immortal_only item, but got no error")
	}
}

func TestItemEffectSoulBoost(t *testing.T) {
	player := &models.Player{
		DaoMarks: models.DaoMarksMap{"soul": 5},
	}
	item := models.ItemSpec{
		Effects: models.ItemEffects{"soul_boost": 10},
	}

	_, err := soulBoostEffect(player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.DaoMarks["soul"] != 15 {
		t.Errorf("expected soul dao marks 15, got %v", player.DaoMarks["soul"])
	}
}

func TestItemEffectDaoMarkGain(t *testing.T) {
	player := &models.Player{
		DaoMarks: models.DaoMarksMap{"strength": 10},
	}
	item := models.ItemSpec{
		Path:    "Strength Path",
		Effects: models.ItemEffects{"dao_mark_gain": 5},
	}

	_, err := daoMarkGainEffect(player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if player.DaoMarks["strength"] != 15 {
		t.Errorf("expected strength dao marks 15, got %v", player.DaoMarks["strength"])
	}
}

func TestItemEffectRemoveNegativeStatus(t *testing.T) {
	player := &models.Player{
		ActiveCombat: &models.CombatState{
			PlayerStatusEffects: []models.StatusEffect{
				{Type: "bleed", Turns: 3},
			},
		},
	}
	item := models.ItemSpec{
		Name: "Tịnh Hóa Đan",
	}

	_, err := removeNegativeStatusEffect(player, item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(player.ActiveCombat.PlayerStatusEffects) != 0 {
		t.Errorf("expected empty status effects, got %v", len(player.ActiveCombat.PlayerStatusEffects))
	}
}
