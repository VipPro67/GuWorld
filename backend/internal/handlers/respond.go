package handlers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/playerstate"
)

func refreshActionPlayerState(player *models.Player) {
	if player == nil || player.ID == uuid.Nil || db.DB == nil {
		return
	}
	// Read the response fields and their revision from one MVCC snapshot. Never
	// attach a newer revision to values retained in a pre-commit Go model.
	_ = playerstate.RefreshResponse(db.DB, player)
}

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
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain)
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
	respondActionWithInventoryDomains(c, status, player, nil, extra...)
}

// respondActionWithInventoryDomains is respondActionWithInventory plus extra
// realtime invalidation domains. Use it when the action mutated a collection
// that is intentionally not embedded in the slim response (for example
// RecipesDomain for player_recipe_progress); connected clients then refresh
// that collection instead of keeping a stale cached list.
func respondActionWithInventoryDomains(c *gin.Context, status int, player models.Player, extraDomains []PlayerEventDomain, extra ...gin.H) {
	refreshActionPlayerState(&player)
	if player.GuInventory == nil {
		if player.ID != uuid.Nil && db.DB != nil {
			player.GuInventory, _ = db.LoadPlayerGus(db.DB, player.ID.String(), false)
		}
		if player.GuInventory == nil {
			player.GuInventory = models.GuInventory{}
		}
	}
	player.MaxMana = gamefx.EffectiveMaxMana(&player)
	player.BaseHP = gamefx.EffectiveMaxHP(&player)
	player.TimeSpeedMultiplier += models.EquippedTimeSpeedBonus(player.GuInventory)
	ClearPendingPlayerQueues(player.ID.String())
	domains := []PlayerEventDomain{PlayerDomain, GuInventoryDomain, ItemInventoryDomain, CombatStatsDomain}
	domains = append(domains, extraDomains...)
	PublishPlayerSnapshot(&player, PlayerEventUpdated, domains...)
	resp := gin.H{
		"success":             true,
		"gu_inventory":        player.GuInventory,
		"materials_inventory": loadMaterialsInventory(player.ID.String()),
		"primeval_stones":     player.PrimevalStones,
		"immortal_stones":     player.ImmortalStones,
		"human_qi":            player.HumanQi,
		"hp":                  player.HP,
		"base_hp":             player.BaseHP,
		"current_mana":        player.CurrentMana,
		"max_mana":            player.MaxMana,
		"current_thought":     player.CurrentThought,
		"soul_hp":             player.SoulHP,
		"lifespan":            player.Lifespan,
		"aptitude_percent":    player.AptitudePercent,
		"rank":                player.Rank,
		"sub_rank":            player.SubRank,
		"wall_progress":       player.WallProgress,
		"state_revision":      player.StateRevision,
	}
	for _, e := range extra {
		for k, v := range e {
			if k == "player" {
				resp[k] = player
			} else {
				resp[k] = v
			}
		}
	}
	c.JSON(status, resp)
}

// respondActionGuOnly replaces responseWithPlayer for action endpoints that only modify
// gu_inventory (feed, equip, drop, status changes). No materials DB query is made.
func respondActionGuOnly(c *gin.Context, status int, player models.Player, extra ...gin.H) {
	refreshActionPlayerState(&player)
	if player.GuInventory == nil {
		if player.ID != uuid.Nil && db.DB != nil {
			player.GuInventory, _ = db.LoadPlayerGus(db.DB, player.ID.String(), false)
		}
		if player.GuInventory == nil {
			player.GuInventory = models.GuInventory{}
		}
	}
	player.MaxMana = gamefx.EffectiveMaxMana(&player)
	player.BaseHP = gamefx.EffectiveMaxHP(&player)
	player.TimeSpeedMultiplier += models.EquippedTimeSpeedBonus(player.GuInventory)
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain, GuInventoryDomain, CombatStatsDomain)
	resp := gin.H{
		"success":          true,
		"gu_inventory":     player.GuInventory,
		"primeval_stones":  player.PrimevalStones,
		"immortal_stones":  player.ImmortalStones,
		"human_qi":         player.HumanQi,
		"hp":               player.HP,
		"base_hp":          player.BaseHP,
		"current_mana":     player.CurrentMana,
		"max_mana":         player.MaxMana,
		"current_thought":  player.CurrentThought,
		"soul_hp":          player.SoulHP,
		"lifespan":         player.Lifespan,
		"aptitude_percent": player.AptitudePercent,
		"rank":             player.Rank,
		"sub_rank":         player.SubRank,
		"wall_progress":    player.WallProgress,
		"state_revision":   player.StateRevision,
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
	refreshActionPlayerState(&player)
	player.MaxMana = gamefx.EffectiveMaxMana(&player)
	player.BaseHP = gamefx.EffectiveMaxHP(&player)
	player.TimeSpeedMultiplier += models.EquippedTimeSpeedBonus(player.GuInventory)
	ClearPendingPlayerQueues(player.ID.String())
	PublishPlayerSnapshot(&player, PlayerEventUpdated, PlayerDomain)
	resp := gin.H{"success": true, "state_revision": player.StateRevision}
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

// loadSessionProgress combines the two small progress reads returned by
// /auth/me. It falls back to the existing service readers if either service
// is unavailable or the combined query fails, preserving their error behavior.
func loadSessionProgress(ctx context.Context, playerID string) (models.CompletedQuestsList, models.UnlockedRecipesList) {
	fallback := func() (models.CompletedQuestsList, models.UnlockedRecipesList) {
		return loadCompletedQuests(playerID), loadUnlockedRecipes(playerID)
	}
	if db.DB == nil || QuestSvc == nil || RecipeSvc == nil {
		return fallback()
	}

	rows, err := db.DB.QueryContext(ctx, `
		SELECT progress_kind, progress_id, completion_rate
		FROM (
			SELECT 1 AS progress_kind, quest_id AS progress_id, 0::double precision AS completion_rate
			FROM player_quest_progress
			WHERE player_id = $1 AND status = 'COMPLETED'
			UNION ALL
			SELECT 2 AS progress_kind, recipe_id AS progress_id, completion_rate::double precision
			FROM player_recipe_progress
			WHERE player_id = $1
		) AS session_progress
		ORDER BY progress_kind, progress_id
	`, playerID)
	if err != nil {
		return fallback()
	}
	defer rows.Close()

	quests := make(models.CompletedQuestsList, 0)
	recipes := make(models.UnlockedRecipesList, 0)
	for rows.Next() {
		var kind int
		var id string
		var completionRate float64
		if err := rows.Scan(&kind, &id, &completionRate); err != nil {
			_ = rows.Close()
			return fallback()
		}
		switch kind {
		case 1:
			quests = append(quests, id)
		case 2:
			recipes = append(recipes, models.UnlockedRecipe{RecipeID: id, CompletionRate: completionRate})
		default:
			_ = rows.Close()
			return fallback()
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fallback()
	}
	return quests, recipes
}
