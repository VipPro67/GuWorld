package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func setupPvPTestRegistry(t *testing.T) {
	t.Helper()
	models.GuRegistry = map[string]map[int]models.GuDictionaryItem{
		"GU_SHIELD": {
			1: {
				ID:   "GU_SHIELD",
				Name: "Khiên Cổ",
				Rank: 1,
				Type: "Defense",
				Effects: models.GuEffects{
					Active: &models.ActiveEffect{ShieldFlat: 50.0},
				},
				BaseManaCost: 10,
			},
		},
		"GU_ATTACK": {
			1: {
				ID:   "GU_ATTACK",
				Name: "Công Cổ",
				Rank: 1,
				Type: "Attack",
				Effects: models.GuEffects{
					Active: &models.ActiveEffect{DamageFlat: 30.0, DamageMult: 1.0},
				},
				BaseManaCost: 15,
			},
		},
		"GU_HEAL": {
			1: {
				ID:   "GU_HEAL",
				Name: "Trị Liệu Cổ",
				Rank: 1,
				Type: "Defense",
				Effects: models.GuEffects{
					Active: &models.ActiveEffect{HealFlat: 40.0},
				},
				BaseManaCost: 12,
			},
		},
	}
	t.Cleanup(func() { models.GuRegistry = map[string]map[int]models.GuDictionaryItem{} })
}

func TestSelectActionsByPlayerTacticsConditions(t *testing.T) {
	combatSettings := &models.AutoCombatConfig{
		GuPriorities: []models.GuTactic{
			{GuID: "GU_HEAL", Priority: 1, UseCondition: "hp_below", ConditionValue: 30},
			{GuID: "GU_SHIELD", Priority: 2, UseCondition: "always", ConditionValue: 0},
			{GuID: "GU_ATTACK", Priority: 3, UseCondition: "always", ConditionValue: 0},
		},
		CategoryPriorities: []string{"shield", "heal", "damage", "other"},
	}

	potentialActions := []Action{
		{Name: "Trị Liệu Cổ", GuID: "GU_HEAL", Def: 40, IsUtility: true, ManaCost: 10, Priority: 1, UseCondition: "hp_below", ConditionValue: 30},
		{Name: "Khiên Cổ", GuID: "GU_SHIELD", Def: 50, ManaCost: 10, Priority: 2, UseCondition: "always"},
		{Name: "Công Cổ", GuID: "GU_ATTACK", Dmg: 30, ManaCost: 15, Priority: 3, UseCondition: "always"},
	}

	// Case 1: Enemy has 100% HP -> Heal (Priority 1) should be skipped because HP >= 30%.
	// Shield (Priority 2) and Attack (Priority 3) should be selected.
	enemyHighHP := &models.Combatant{
		Name:           "Defender",
		HP:             100,
		MaxHP:          100,
		CurrentMana:    100,
		MaxMana:        100,
		CombatSettings: combatSettings,
	}

	selected := SelectActionsByPlayerTactics(nil, enemyHighHP, potentialActions, 100)
	if len(selected) != 2 {
		t.Fatalf("expected 2 actions selected (Shield + Attack), got %d: %+v", len(selected), selected)
	}
	if selected[0].GuID != "GU_SHIELD" {
		t.Errorf("expected first action to be GU_SHIELD, got %s", selected[0].GuID)
	}
	if selected[1].GuID != "GU_ATTACK" {
		t.Errorf("expected second action to be GU_ATTACK, got %s", selected[1].GuID)
	}

	// Case 2: Enemy has 20% HP -> Heal (Priority 1) condition is met (HP < 30%).
	// Heal should be selected first, followed by Attack!
	enemyLowHP := &models.Combatant{
		Name:           "Defender",
		HP:             20,
		MaxHP:          100,
		CurrentMana:    100,
		MaxMana:        100,
		CombatSettings: combatSettings,
	}

	selectedLow := SelectActionsByPlayerTactics(nil, enemyLowHP, potentialActions, 100)
	if len(selectedLow) != 2 {
		t.Fatalf("expected 2 actions selected (Heal + Attack), got %d: %+v", len(selectedLow), selectedLow)
	}
	if selectedLow[0].GuID != "GU_HEAL" {
		t.Errorf("expected first action to be GU_HEAL when HP < 30%%, got %s", selectedLow[0].GuID)
	}
	if selectedLow[1].GuID != "GU_ATTACK" {
		t.Errorf("expected second action to be GU_ATTACK, got %s", selectedLow[1].GuID)
	}
}
