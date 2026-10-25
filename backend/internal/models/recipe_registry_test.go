package models

import (
	"testing"
)

func TestRecipeRegistry(t *testing.T) {
	recipeMu.Lock()
	origRegistry := RecipeRegistry
	origByTarget := RecipeByTarget
	origList := recipeList

	dummyRecipe := RefiningRecipe{
		ID:                 "recipe_GU_SPRING_EGG_1_1",
		TargetGuID:         "GU_SPRING_EGG",
		TargetRank:         1,
		Variant:            1,
		SuccessRate:        0.9,
		PrimevalStonesCost: 100,
	}

	RecipeRegistry = map[string]RefiningRecipe{
		dummyRecipe.ID: dummyRecipe,
	}
	RecipeByTarget = map[string]map[int][]RefiningRecipe{
		"GU_SPRING_EGG": {
			1: {dummyRecipe},
		},
	}
	recipeList = []RefiningRecipe{dummyRecipe}
	recipeMu.Unlock()

	defer func() {
		recipeMu.Lock()
		RecipeRegistry = origRegistry
		RecipeByTarget = origByTarget
		recipeList = origList
		recipeMu.Unlock()
	}()

	// 1. GetRecipe by ID
	rec, ok := GetRecipe("recipe_GU_SPRING_EGG_1_1")
	if !ok || rec.TargetGuID != "GU_SPRING_EGG" {
		t.Fatalf("expected to find recipe_GU_SPRING_EGG_1_1, got ok=%v, rec=%+v", ok, rec)
	}
	if _, ok := GetRecipe("non_existent"); ok {
		t.Fatalf("expected non_existent recipe to fail")
	}

	// 2. GetRecipeByTarget
	recTarget, ok := GetRecipeByTarget("GU_SPRING_EGG", 1, 1)
	if !ok || recTarget.ID != dummyRecipe.ID {
		t.Fatalf("expected recipe by target, got ok=%v, rec=%+v", ok, recTarget)
	}
	if _, ok := GetRecipeByTarget("GU_SPRING_EGG", 2, 1); ok {
		t.Fatalf("expected rank 2 to fail")
	}

	// 3. GetAllRecipes
	all := GetAllRecipes()
	if len(all) != 1 || all[0].ID != dummyRecipe.ID {
		t.Fatalf("unexpected all recipes: %+v", all)
	}
}
