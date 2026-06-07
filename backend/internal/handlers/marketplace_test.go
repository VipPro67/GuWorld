package handlers

import (
	"encoding/json"
	"testing"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

func TestAddAuctionItemToPlayer_Material(t *testing.T) {
	player := &models.Player{
		MaterialsInventory: models.MaterialsInventory{
			"iron_ore": 5,
		},
	}

	err := addAuctionItemToPlayer(player, "Material", "iron_ore", 3)
	if err != nil {
		t.Fatalf("unexpected error adding material: %v", err)
	}

	if qty := player.MaterialsInventory["iron_ore"]; qty != 8 {
		t.Errorf("expected 8 iron_ore, got %d", qty)
	}

	err = addAuctionItemToPlayer(player, "Material", "copper_ore", 2)
	if err != nil {
		t.Fatalf("unexpected error adding new material: %v", err)
	}

	if qty := player.MaterialsInventory["copper_ore"]; qty != 2 {
		t.Errorf("expected 2 copper_ore, got %d", qty)
	}
}

func TestAddAuctionItemToPlayer_Gu(t *testing.T) {
	player := &models.Player{
		GuInventory: models.GuInventory{},
	}

	err := addAuctionItemToPlayer(player, "Gu", "GU_SPRING_CICADA", 1)
	if err != nil {
		t.Fatalf("unexpected error adding Gu: %v", err)
	}

	if len(player.GuInventory) != 1 {
		t.Fatalf("expected 1 Gu in inventory, got %d", len(player.GuInventory))
	}

	addedGu := player.GuInventory[0]
	if addedGu.GuID != "GU_SPRING_CICADA" {
		t.Errorf("expected GuID GU_SPRING_CICADA, got %s", addedGu.GuID)
	}
	if addedGu.Hunger != 100 || addedGu.Status != "SLEEPING" || addedGu.IsVital {
		t.Errorf("Gu instance properties not correctly initialized: %+v", addedGu)
	}
}

func TestAddAuctionItemToPlayer_Recipe(t *testing.T) {
	player := &models.Player{
		UnlockedRecipes: models.UnlockedRecipesList{
			{RecipeID: "recipe_spring_cicada", CompletionRate: 40.0},
		},
	}

	// 1. Add existing recipe - should merge completion rates (40% + 30% = 70%)
	recipeData1 := models.UnlockedRecipe{RecipeID: "recipe_spring_cicada", CompletionRate: 30.0}
	jsonData1, _ := json.Marshal(recipeData1)

	err := addAuctionItemToPlayer(player, "Recipe", string(jsonData1), 1)
	if err != nil {
		t.Fatalf("unexpected error adding recipe: %v", err)
	}

	if len(player.UnlockedRecipes) != 1 {
		t.Fatalf("expected 1 recipe, got %d", len(player.UnlockedRecipes))
	}
	if rate := player.UnlockedRecipes[0].CompletionRate; rate != 70.0 {
		t.Errorf("expected 70.0 completion rate, got %f", rate)
	}

	// 2. Add existing recipe capping at 100% (70% + 50% = 120% -> 100%)
	recipeData2 := models.UnlockedRecipe{RecipeID: "recipe_spring_cicada", CompletionRate: 50.0}
	jsonData2, _ := json.Marshal(recipeData2)

	err = addAuctionItemToPlayer(player, "Recipe", string(jsonData2), 1)
	if err != nil {
		t.Fatalf("unexpected error adding recipe: %v", err)
	}

	if rate := player.UnlockedRecipes[0].CompletionRate; rate != 100.0 {
		t.Errorf("expected capped at 100.0, got %f", rate)
	}

	// 3. Add a completely new recipe - should append it to the list
	recipeData3 := models.UnlockedRecipe{RecipeID: "recipe_liquor_worm", CompletionRate: 15.0}
	jsonData3, _ := json.Marshal(recipeData3)

	err = addAuctionItemToPlayer(player, "Recipe", string(jsonData3), 1)
	if err != nil {
		t.Fatalf("unexpected error adding new recipe: %v", err)
	}

	if len(player.UnlockedRecipes) != 2 {
		t.Fatalf("expected 2 recipes, got %d", len(player.UnlockedRecipes))
	}
	if player.UnlockedRecipes[1].RecipeID != "recipe_liquor_worm" || player.UnlockedRecipes[1].CompletionRate != 15.0 {
		t.Errorf("new recipe not added correctly: %+v", player.UnlockedRecipes[1])
	}
}
