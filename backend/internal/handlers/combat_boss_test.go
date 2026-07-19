package handlers

import (
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestSelectActionsByAI_BossBurst(t *testing.T) {
	potentialActions := []Action{
		{Name: "Action1", Dmg: 10, Def: 0, ManaCost: 20},
		{Name: "Action2", Dmg: 30, Def: 0, ManaCost: 40},
		{Name: "Action3", Dmg: 20, Def: 0, ManaCost: 30},
	}

	// Should select Action2 and Action3 (total cost 70, max 2 actions)
	selected := SelectActionsByAI("boss_burst", potentialActions, 80)
	if len(selected) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(selected))
	}
	if selected[0].Name != "Action2" || selected[1].Name != "Action3" {
		t.Errorf("incorrect actions selected: %+v", selected)
	}

	// Should select only Action3 (mana limit 30)
	selected2 := SelectActionsByAI("boss_burst", potentialActions, 30)
	if len(selected2) != 1 {
		t.Fatalf("expected 1 action, got %d", len(selected2))
	}
	if selected2[0].Name != "Action3" {
		t.Errorf("incorrect action selected: %+v", selected2[0])
	}
}

func TestSelectActionsByAI_Defensive(t *testing.T) {
	potentialActions := []Action{
		{Name: "Shield1", Dmg: 0, Def: 10, ManaCost: 15},
		{Name: "Shield2", Dmg: 0, Def: 25, ManaCost: 30},
		{Name: "Attack", Dmg: 50, Def: 0, ManaCost: 40},
	}

	// Should select Shield2 and Shield1 (total cost 45)
	selected := SelectActionsByAI("defensive", potentialActions, 50)
	if len(selected) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(selected))
	}
	if selected[0].Name != "Shield2" || selected[1].Name != "Shield1" {
		t.Errorf("incorrect actions selected: %+v", selected)
	}
}

func TestSelectActionsByAI_BossBalanced(t *testing.T) {
	potentialActions := []Action{
		{Name: "DmgGu", Dmg: 40, Def: 0, ManaCost: 20},
		{Name: "DefGu", Dmg: 0, Def: 30, ManaCost: 20},
	}

	// Should pick one of each if mana suffices
	selected := SelectActionsByAI("boss_balanced", potentialActions, 40)
	if len(selected) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(selected))
	}
	var hasDmg, hasDef bool
	for _, s := range selected {
		if s.Name == "DmgGu" {
			hasDmg = true
		}
		if s.Name == "DefGu" {
			hasDef = true
		}
	}
	if !hasDmg || !hasDef {
		t.Errorf("balanced AI did not select both damage and defense: %+v", selected)
	}
}

func TestRollLoot(t *testing.T) {
	lt := models.LootTable{
		ID:   "test_table",
		Name: "Test Table",
		Entries: []models.LootEntry{
			{ItemID: "gu_skin", Chance: 1.0, MinQty: 2, MaxQty: 5},
			{ItemID: "rare_herb", Chance: 0.0, MinQty: 1, MaxQty: 1},
			{GuID: "GU_TEST", Chance: 1.0, MinQty: 1, MaxQty: 1},
		},
	}

	loot := RollLoot(lt)
	if loot.Items["gu_skin"] < 2 || loot.Items["gu_skin"] > 5 {
		t.Errorf("expected gu_skin quantity between 2 and 5, got %d", loot.Items["gu_skin"])
	}
	if _, ok := loot.Items["rare_herb"]; ok {
		t.Errorf("did not expect rare_herb (chance 0.0)")
	}
	if len(loot.Gus) != 1 || loot.Gus[0] != "GU_TEST" {
		t.Errorf("expected 1 GU_TEST, got %+v", loot.Gus)
	}
}

func TestCalculateGuEffectForCombatant_Backlash(t *testing.T) {
	actor := &models.Combatant{
		Name:    "Test Actor",
		Rank:    1,
		SubRank: 1,
		HP:      200,
		MaxHP:   200,
		MaxMana: 1000,
	}
	highRankGu := &models.GuDictionaryItem{
		Name:         "Mặt Trời Gu",
		Rank:         3,
		BaseManaCost: 50,
	}

	dmg, def, cost, err := CalculateGuEffectForCombatant(actor, highRankGu)
	if err != nil {
		t.Fatalf("expected no error (backlash applied but action proceeds), got %v", err)
	}
	// Backlash is no longer applied by CalculateGuEffectForCombatant
	// (it is now applied via BacklashDamage on SimAction in the simulator engine)
	if actor.HP != 200 {
		t.Fatalf("expected actor HP unchanged by CalculateGuEffectForCombatant, got %f", actor.HP)
	}
	// Rank 1 using mortal Gu (rank 3): multiplier 1x, cost = 500
	// Rank difference penalty removed; mortal rank multiplier applies instead
	expectedCost := (float64(highRankGu.BaseManaCost) / 100.0) * actor.MaxMana // 500
	if cost < expectedCost*0.9 || cost > expectedCost*1.1 {
		t.Fatalf("expected cost around %.0f for rank 1 mortal with mortal gu, got %.0f", expectedCost, cost)
	}
	_ = dmg
	_ = def
}
