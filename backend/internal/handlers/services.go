package handlers

import "github.com/vippro67/GuWorld/backend/internal/services"

var (
	RecipeSvc       *services.RecipeService
	MapService      *services.MapService
	CombatSessionSvc *services.CombatSessionService
	DaoSvc          *services.DaoService
	InventorySvc    *services.InventoryService
	QuestSvc        *services.QuestService
	AchievementSvc  *services.AchievementService
)
