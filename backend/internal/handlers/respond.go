package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
)

// responseWithPlayer sends a JSON response with the player object, ensuring MaterialsInventory and CompletedQuests are populated.
// Use only for GET/session endpoints (auth/me, player/status) where full player initialization is required.
// For action endpoints, prefer respondActionWithInventory, respondActionGuOnly, or respondActionOnly.
func responseWithPlayer(c *gin.Context, status int, player models.Player, extra ...gin.H) {
	player.MaxMana = gamefx.EffectiveMaxMana(&player)
	player.BaseHP = gamefx.EffectiveMaxHP(&player)
	player.TimeSpeedMultiplier += models.EquippedTimeSpeedBonus(player.GuInventory)
	player.MaterialsInventory = loadMaterialsInventory(player.ID.String())
	player.CompletedQuests = loadCompletedQuests(player.ID.String())
	player.UnlockedRecipes = loadUnlockedRecipes(player.ID.String())
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated)
	resp := gin.H{"player": player}
	for _, e := range extra {
		for k, v := range e {
			resp[k] = v
		}
	}
	c.JSON(status, resp)
}

// respondActionWithInventory replaces responseWithPlayer for action endpoints that modify
// gu_inventory and/or materials_inventory. It publishes the WS snapshot (for hp/mana/stones/combat
// realtime updates) and returns a slim payload with only inventory data + extra fields.
// This eliminates the completed_quests and unlocked_recipes DB queries from every action response.
func respondActionWithInventory(c *gin.Context, status int, player models.Player, extra ...gin.H) {
	player.MaxMana = gamefx.EffectiveMaxMana(&player)
	player.BaseHP = gamefx.EffectiveMaxHP(&player)
	player.TimeSpeedMultiplier += models.EquippedTimeSpeedBonus(player.GuInventory)
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated)
	resp := gin.H{
		"success":             true,
		"gu_inventory":        player.GuInventory,
		"materials_inventory": loadMaterialsInventory(player.ID.String()),
		"primeval_stones":     player.PrimevalStones,
		"immortal_stones":     player.ImmortalStones,
		"human_qi":            player.HumanQi,
	}
	for _, e := range extra {
		for k, v := range e {
			resp[k] = v
		}
	}
	c.JSON(status, resp)
}

// respondActionGuOnly replaces responseWithPlayer for action endpoints that only modify
// gu_inventory (feed, equip, drop, status changes). No materials DB query is made.
func respondActionGuOnly(c *gin.Context, status int, player models.Player, extra ...gin.H) {
	player.MaxMana = gamefx.EffectiveMaxMana(&player)
	player.BaseHP = gamefx.EffectiveMaxHP(&player)
	player.TimeSpeedMultiplier += models.EquippedTimeSpeedBonus(player.GuInventory)
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated)
	resp := gin.H{
		"success":         true,
		"gu_inventory":    player.GuInventory,
		"primeval_stones": player.PrimevalStones,
		"immortal_stones": player.ImmortalStones,
		"human_qi":        player.HumanQi,
	}
	for _, e := range extra {
		for k, v := range e {
			resp[k] = v
		}
	}
	c.JSON(status, resp)
}

// respondActionOnly replaces responseWithPlayer for action endpoints whose player state changes
// are fully covered by the WS snapshot (hp, mana, stones, position, combat, rank, wall_progress).
// No inventory data is returned — the frontend relies on the WS player.updated event.
func respondActionOnly(c *gin.Context, status int, player models.Player, extra ...gin.H) {
	player.MaxMana = gamefx.EffectiveMaxMana(&player)
	player.BaseHP = gamefx.EffectiveMaxHP(&player)
	player.TimeSpeedMultiplier += models.EquippedTimeSpeedBonus(player.GuInventory)
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated)
	resp := gin.H{"success": true}
	for _, e := range extra {
		for k, v := range e {
			resp[k] = v
		}
	}
	c.JSON(status, resp)
}

// responseWithPlayerData sends a JSON response with pre-loaded player data,
// avoiding redundant DB queries for MaterialsInventory and CompletedQuests.
func responseWithPlayerData(c *gin.Context, status int, player models.Player, materials models.MaterialsInventory, quests models.CompletedQuestsList, extra ...gin.H) {
	player.MaxMana = gamefx.EffectiveMaxMana(&player)
	player.BaseHP = gamefx.EffectiveMaxHP(&player)
	player.TimeSpeedMultiplier += models.EquippedTimeSpeedBonus(player.GuInventory)
	player.MaterialsInventory = materials
	player.CompletedQuests = quests
	player.UnlockedRecipes = loadUnlockedRecipes(player.ID.String())
	resp := gin.H{"player": player}
	for _, e := range extra {
		for k, v := range e {
			resp[k] = v
		}
	}
	c.JSON(status, resp)
}

// loadUnlockedRecipes returns the player's unlocked recipe progress for UI.
func loadUnlockedRecipes(playerID string) models.UnlockedRecipesList {
	if RecipeSvc == nil {
		return models.UnlockedRecipesList{}
	}
	progress, err := RecipeSvc.ListByPlayer(playerID)
	if err != nil {
		return models.UnlockedRecipesList{}
	}
	out := make(models.UnlockedRecipesList, 0, len(progress))
	for _, p := range progress {
		out = append(out, models.UnlockedRecipe{
			RecipeID:       p.RecipeID,
			CompletionRate: p.CompletionRate,
		})
	}
	return out
}

// loadMaterialsInventory fetches the player's materials inventory as a flat id->quantity map.
func loadMaterialsInventory(playerID string) models.MaterialsInventory {
	if InventorySvc == nil {
		return nil
	}
	items, err := InventorySvc.ListByPlayer(playerID)
	if err != nil {
		return nil
	}
	mi := make(models.MaterialsInventory)
	for _, item := range items {
		mi[item.ItemID] += item.Quantity
	}
	return mi
}

// loadCompletedQuests returns the list of quest IDs the player has completed.
func loadCompletedQuests(playerID string) models.CompletedQuestsList {
	if QuestSvc == nil {
		return nil
	}
	progress, err := QuestSvc.ListCompleted(playerID)
	if err != nil {
		return nil
	}
	result := make(models.CompletedQuestsList, 0, len(progress))
	for _, p := range progress {
		result = append(result, p.QuestID)
	}
	return result
}
