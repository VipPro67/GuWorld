package handlers

import "github.com/vippro67/GuWorld/backend/internal/services"

var (
	PlayerSvc        *services.PlayerService
	RecipeSvc        *services.RecipeService
	CombatSessionSvc *services.CombatSessionService
	InventorySvc     *services.InventoryService
	QuestSvc         *services.QuestService
	AchievementSvc   *services.AchievementService
)
