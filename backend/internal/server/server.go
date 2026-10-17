package server

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aurowora/compress"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/authz"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/internal/ws"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const slowRequestThreshold = 5 * time.Second

type adminRouteSpec struct {
	method                   string
	path                     string
	permission               string
	handler                  gin.HandlerFunc
	auditAction              string
	auditTargetType          string
	reasonRequired           bool
	confirmationTarget       string
	reauthenticationRequired bool
	idempotencyKeyRequired   bool
	destructive              bool
}

// adminRouteSpecs is the permission inventory for every /api/admin endpoint.
// Keeping registration and authorization metadata together makes missing permissions impossible.
func adminRouteSpecs() []adminRouteSpec {
	return []adminRouteSpec{
		{http.MethodGet, "/me", authz.AdminAccessPermission, handlers.AdminGetMe, "", "", false, "", false, false, false},
		{http.MethodGet, "/resources", authz.AdminAccessPermission, handlers.AdminGetResources, "", "", false, "", false, false, false},
		// monitoring
		{http.MethodGet, "/monitor/system", "monitoring.read", handlers.AdminGetSystemStats, "", "", false, "", false, false, false},
		{http.MethodGet, "/monitor/requests/summary", "monitoring.read", handlers.AdminGetRequestSummary, "", "", false, "", false, false, false},
		{http.MethodGet, "/monitor/admin-operations", "monitoring.read", handlers.AdminGetOperationMetrics, "", "", false, "", false, false, false},
		{http.MethodGet, "/monitor/requests/traces", "monitoring.read", handlers.AdminGetRequestTraces, "", "", false, "", false, false, false},
		{http.MethodGet, "/monitor/requests/traces/:id", "monitoring.read", handlers.AdminGetRequestTraceDetail, "", "", false, "", false, false, false},
		{http.MethodPost, "/monitor/requests/reset", "monitoring.reset", handlers.AdminResetRequestMetrics, "admin.reset_request_metrics", "request_metrics", false, "", false, false, false},
		// billing
		{http.MethodGet, "/topup-orders", "billing.read", handlers.AdminListTopupOrders, "", "", false, "", false, false, false},
		{http.MethodGet, "/wallet-transactions", "billing.read", handlers.AdminListWalletTransactions, "", "", false, "", false, false, false},
		// combat history is append-only; admins can inspect but not rewrite or delete it.
		{http.MethodGet, "/combat-history", "combat_history.read", handlers.AdminListCombatHistory, "", "", false, "", false, false, false},
		{http.MethodGet, "/combat-history/:id", "combat_history.read", handlers.AdminGetCombatHistory, "", "", false, "", false, false, false},
		{http.MethodGet, "/arena-rankings", "arena_rankings.read", handlers.AdminListArenaRankings, "", "", false, "", false, false, false},
		{http.MethodPut, "/arena-rankings/:player_id", "arena_rankings.update", handlers.AdminUpdateArenaRanking, "admin.update_arena_ranking", "arena_rankings", false, "", false, true, false},
		// players
		{http.MethodGet, "/players/online", "players.read", handlers.AdminListOnlinePlayers, "", "", false, "", false, false, false},
		{http.MethodGet, "/players", "players.read", handlers.AdminListPlayers, "", "", false, "", false, false, false},
		{http.MethodGet, "/players/:id", "players.read", handlers.AdminGetPlayerDetail, "", "", false, "", false, false, false},
		{http.MethodGet, "/players/:id/items", "players.read", handlers.AdminListPlayerItems, "", "", false, "", false, false, false},
		{http.MethodPut, "/players/:id/items/:item_id/:category", "players.update", handlers.AdminUpdatePlayerInventoryItem, "admin.update_player_inventory_item", "player_inventory_items", false, "path", false, false, false},
		{http.MethodGet, "/players/:id/gus", "players.read", handlers.AdminListPlayerGus, "", "", false, "", false, false, false},
		{http.MethodGet, "/players/:id/quests", "players.read", handlers.AdminListPlayerQuests, "", "", false, "", false, false, false},
		{http.MethodGet, "/players/:id/instance-sessions", "players.read", handlers.AdminListPlayerInstanceSessions, "", "", false, "", false, false, false},
		{http.MethodGet, "/players/:id/sect-missions", "players.read", handlers.AdminListPlayerSectMissions, "", "", false, "", false, false, false},
		{http.MethodGet, "/players/:id/tracking-marks", "players.read", handlers.AdminListPlayerTrackingMarks, "", "", false, "", false, false, false},
		{http.MethodGet, "/players/:id/achievements", "players.read", handlers.AdminListPlayerAchievements, "", "", false, "", false, false, false},
		{http.MethodPut, "/players/:id", "players.update", handlers.AdminUpdatePlayer, "admin.update_player", "player", false, "", false, false, false},
		{http.MethodPut, "/players/:id/ban", "players.ban", handlers.AdminBanPlayer, "admin.ban_player", "player", true, "id", false, false, true},
		{http.MethodPut, "/players/:id/unban", "players.ban", handlers.AdminUnbanPlayer, "admin.unban_player", "player", true, "id", false, false, true},
		{http.MethodPut, "/players/:id/chat-mute", "players.ban", handlers.AdminMutePlayer, "admin.mute_player", "player", true, "id", false, false, true},
		{http.MethodPut, "/players/:id/chat-unmute", "players.ban", handlers.AdminUnmutePlayer, "admin.unmute_player", "player", true, "id", false, false, true},
		// apertures
		{http.MethodGet, "/apertures", "apertures.read", handlers.AdminListApertures, "", "", false, "", false, false, false},
		{http.MethodGet, "/apertures/:id", "apertures.read", handlers.AdminGetApertureDetail, "", "", false, "", false, false, false},
		{http.MethodPut, "/apertures/:id", "apertures.update", handlers.AdminUpdateAperture, "admin.update_aperture", "aperture", false, "", false, false, false},
		// gus
		{http.MethodGet, "/gus/search", "gus.read", handlers.AdminSearchGus, "", "", false, "", false, false, false},
		{http.MethodGet, "/gu-history", "gus.read", handlers.AdminListGuHistory, "", "", false, "", false, false, false},
		// players
		{http.MethodPost, "/players/:id/items", "players.inventory.grant", handlers.AdminAddPlayerItems, "admin.add_player_items", "player", false, "", false, false, false},
		{http.MethodDelete, "/players/:id/items", "players.delete", handlers.AdminRemovePlayerItems, "admin.remove_player_items", "player", true, "path", false, false, true},
		{http.MethodPut, "/players/:id/recipes/:recipe_id", "players.update", handlers.AdminUpdatePlayerRecipe, "admin.update_recipe_progress", "player_recipe_progress", false, "path", false, false, false},
		{http.MethodPut, "/players/:id/quests/:quest_id", "players.update", handlers.AdminUpdatePlayerQuest, "admin.update_quest_progress", "player_quest_progress", false, "path", false, false, false},
		{http.MethodDelete, "/players/:id/quests/:quest_id", "players.delete", handlers.AdminDeletePlayerQuest, "admin.delete_quest_progress", "player_quest_progress", true, "path", false, false, true},
		{http.MethodPut, "/players/:id/instance-sessions/:session_id", "players.update", handlers.AdminUpdatePlayerInstanceSession, "admin.update_instance_session", "player_instance_sessions", false, "path", false, false, false},
		{http.MethodDelete, "/players/:id/instance-sessions/:session_id", "players.delete", handlers.AdminDeletePlayerInstanceSession, "admin.delete_instance_session", "player_instance_sessions", true, "path", false, false, true},
		{http.MethodPut, "/players/:id/sect-missions/:mission_row_id", "players.update", handlers.AdminUpdatePlayerSectMission, "admin.update_player_sect_mission", "player_sect_missions", false, "path", false, false, false},
		{http.MethodPut, "/players/:id/tracking-marks/:mark_id", "players.update", handlers.AdminUpdatePlayerTrackingMark, "admin.update_player_tracking_mark", "player_tracking_marks", false, "path", false, false, false},
		{http.MethodDelete, "/players/:id/tracking-marks/:mark_id", "players.delete", handlers.AdminDeletePlayerTrackingMark, "admin.delete_player_tracking_mark", "player_tracking_marks", true, "path", false, false, true},
		{http.MethodPost, "/players/:id/achievements", "players.update", handlers.AdminAddPlayerAchievement, "admin.add_player_achievement", "player_achievements", false, "path", false, false, false},
		{http.MethodPut, "/players/:id/achievements/:achievement_id", "players.update", handlers.AdminUpdatePlayerAchievement, "admin.update_player_achievement", "player_achievements", false, "path", false, false, false},
		{http.MethodDelete, "/players/:id/achievements/:achievement_id", "players.delete", handlers.AdminDeletePlayerAchievement, "admin.delete_player_achievement", "player_achievements", true, "path", false, false, true},
		{http.MethodPost, "/players/:id/gus", "players.inventory.grant", handlers.AdminAddPlayerGus, "admin.add_player_gus", "player", false, "", false, false, false},
		{http.MethodDelete, "/players/:id/gus/:instance_id", "players.delete", handlers.AdminRemovePlayerGu, "admin.remove_player_gu", "player", true, "path", false, false, true},
		{http.MethodGet, "/players/:id/pets", "players.read", handlers.AdminListPlayerPets, "", "", false, "", false, false, false},
		{http.MethodDelete, "/players/:id/pets/:pet_id", "players.delete", handlers.AdminDeletePlayerPet, "admin.delete_player_pet", "player", true, "path", false, false, true},
		{http.MethodPut, "/players/:id/stones", "players.resources.adjust", handlers.AdminChangePlayerStones, "admin.change_player_stones", "player", true, "id", false, true, true},
		{http.MethodGet, "/roles", "admin_roles.read", handlers.AdminListRoles, "", "", false, "", false, false, false},
		{http.MethodGet, "/players/:id/roles", "admin_roles.read", handlers.AdminGetPlayerRoles, "", "", false, "", false, false, false},
		{http.MethodPut, "/players/:id/roles", "admin_roles.assign", handlers.AdminUpdatePlayerRoles, "admin.update_player_roles", "player", true, "id", true, true, true},
		// sects
		{http.MethodGet, "/sects", "sects.read", handlers.AdminListSects, "", "", false, "", false, false, false},
		{http.MethodGet, "/sects/:id", "sects.read", handlers.AdminGetSectDetail, "", "", false, "", false, false, false},
		{http.MethodPut, "/sects/:id", "sects.update", handlers.AdminUpdateSect, "admin.update_sect", "sect", false, "", false, false, false},
		{http.MethodPost, "/sects/:id/members/:memberId/leader", "sects.update", handlers.AdminSetSectLeader, "admin.set_sect_leader", "sect", false, "", false, false, false},
		{http.MethodDelete, "/sects/:id/members/:memberId", "sects.delete", handlers.AdminKickSectMember, "admin.kick_sect_member", "sect", true, "path", false, false, true},
		{http.MethodPost, "/sects/:id/members", "sects.update", handlers.AdminAddToSect, "admin.add_to_sect", "sect", false, "", false, false, false},
		{http.MethodGet, "/sects/:id/defenders", "sects.read", handlers.AdminListSectDefenders, "", "", false, "", false, false, false},
		{http.MethodDelete, "/sects/:id/defenders/:defender_id", "sects.delete", handlers.AdminRemoveSectDefender, "admin.remove_sect_defender", "sect", true, "path", false, false, true},
		{http.MethodGet, "/sects/:id/requests", "sects.read", handlers.AdminListSectRequests, "", "", false, "", false, false, false},
		{http.MethodDelete, "/sects/:id/requests/:request_type/:request_id", "sects.delete", handlers.AdminDeleteSectRequest, "admin.delete_sect_request", "sect", true, "path", false, false, true},
		{http.MethodDelete, "/sects/:id", "sects.delete", handlers.AdminDeleteSect, "admin.delete_sect", "sect", true, "path", false, false, true},
		{http.MethodGet, "/sects/:id/missions", "sects.read", handlers.AdminListSectMissions, "", "", false, "", false, false, false},
		{http.MethodPost, "/sects/:id/missions", "sects.update", handlers.AdminCreateSectMission, "admin.create_sect_mission", "sect", false, "", false, false, false},
		{http.MethodDelete, "/sects/:id/missions/:missionId", "sects.delete", handlers.AdminDeleteSectMission, "admin.delete_sect_mission", "sect", true, "path", false, false, true},
		{http.MethodPost, "/sects/:id/vault/stock", "sects.update", handlers.AdminStockSectVault, "admin.stock_sect_vault", "sect", false, "", false, false, false},
		{http.MethodDelete, "/sects/:id/vault/stock", "sects.delete", handlers.AdminRemoveSectVaultItem, "admin.remove_sect_vault_item", "sect", true, "path", false, false, true},
		// game_data
		{http.MethodPost, "/data/import", "game_data.import_all", handlers.AdminImportData, "admin.import_data", "game_data", false, "", false, false, false},
		{http.MethodPost, "/data/import/commit", "game_data.import_all", handlers.AdminCommitImportData, "admin.commit_import_data", "game_data", true, "all", false, true, true},
		{http.MethodPost, "/data/export", "game_data.export", handlers.AdminExportData, "admin.export_data", "game_data", false, "", false, false, false},
		{http.MethodGet, "/data/tables", "game_data.read", handlers.AdminListTables, "", "", false, "", false, false, false},
		{http.MethodPost, "/data/import/:table", "game_data.import", handlers.AdminImportTable, "admin.import_table", "game_data", false, "", false, false, false},
		{http.MethodPost, "/data/import/:table/commit", "game_data.import", handlers.AdminCommitImportTable, "admin.commit_import_table", "game_data", true, "table", false, true, true},
		{http.MethodPost, "/data/export/:table", "game_data.export", handlers.AdminExportTable, "admin.export_table", "game_data", false, "", false, false, false},
		// cache
		{http.MethodPost, "/clear-cache", "cache.reload", handlers.AdminClearCache, "admin.clear_cache", "cache", false, "", false, false, false},
		{http.MethodPost, "/clear-redis-cache", "cache.flush", handlers.AdminClearRedisCache, "admin.clear_redis_cache", "redis_cache", true, "Redis", true, true, true},
		// audit
		{http.MethodGet, "/audit-logs", "audit.read", handlers.AdminGetAuditLogs, "", "", false, "", false, false, false},
		// regions
		{http.MethodGet, "/regions", "regions.read", handlers.AdminListRegions, "", "", false, "", false, false, false},
		{http.MethodPost, "/regions", "regions.update", handlers.AdminCreateRegion, "admin.create_region", "region", false, "", false, false, false},
		{http.MethodPut, "/regions/:id", "regions.update", handlers.AdminUpdateRegion, "admin.update_region", "region", false, "", false, false, false},
		{http.MethodDelete, "/regions/:id", "regions.delete", handlers.AdminDeleteRegion, "admin.delete_region", "region", true, "path", false, false, true},
		// map_cells
		{http.MethodGet, "/map-cells", "map_cells.read", handlers.AdminListMapCells, "", "", false, "", false, false, false},
		{http.MethodGet, "/residual-traces", "regions.read", handlers.AdminListResidualTraces, "", "", false, "", false, false, false},
		{http.MethodPut, "/residual-traces/:id", "regions.update", handlers.AdminUpdateResidualTrace, "admin.update_residual_trace", "world_residual_traces", false, "path", false, false, false},
		{http.MethodDelete, "/residual-traces/:id", "regions.delete", handlers.AdminDeleteResidualTrace, "admin.delete_residual_trace", "world_residual_traces", true, "path", false, false, true},
		// map_ground_drops
		{http.MethodGet, "/map-drops", "map_drops.read", handlers.AdminListMapGroundDrops, "", "", false, "", false, false, false},
		{http.MethodPut, "/map-drops/:id", "map_drops.update", handlers.AdminUpdateMapGroundDrop, "admin.update_map_drop", "map_ground_drops", false, "", false, false, false},
		{http.MethodDelete, "/map-drops/:id", "map_drops.delete", handlers.AdminDeleteMapGroundDrop, "admin.delete_map_drop", "map_ground_drops", true, "path", false, false, true},
		{http.MethodPost, "/map-cells", "map_cells.update", handlers.AdminCreateMapCell, "admin.create_map_cell", "map_cell", false, "", false, false, false},
		{http.MethodPut, "/map-cells/:id", "map_cells.update", handlers.AdminUpdateMapCell, "admin.update_map_cell", "map_cell", false, "", false, false, false},
		{http.MethodDelete, "/map-cells/:id", "map_cells.delete", handlers.AdminDeleteMapCell, "admin.delete_map_cell", "map_cell", true, "path", false, false, true},
		// shop
		{http.MethodGet, "/shop", "shop.read", handlers.AdminListShopItems, "", "", false, "", false, false, false},
		{http.MethodPost, "/shop", "shop.update", handlers.AdminCreateShopItem, "admin.create_shop_item", "shop_item", false, "", false, false, false},
		{http.MethodPut, "/shop/:id/:rank", "shop.update", handlers.AdminUpdateShopItem, "admin.update_shop_item", "shop_item", false, "", false, false, false},
		{http.MethodDelete, "/shop/:id/:rank", "shop.delete", handlers.AdminDeleteShopItem, "admin.delete_shop_item", "shop_item", true, "path", false, false, true},
		// arena_shop
		{http.MethodGet, "/arena-shop", "arena_shop.read", handlers.AdminListArenaShopItems, "", "", false, "", false, false, false},
		{http.MethodPost, "/arena-shop", "arena_shop.update", handlers.AdminCreateArenaShopItem, "admin.create_arena_shop_item", "arena_shop_item", false, "", false, false, false},
		{http.MethodPut, "/arena-shop/:id", "arena_shop.update", handlers.AdminUpdateArenaShopItem, "admin.update_arena_shop_item", "arena_shop_item", false, "", false, false, false},
		{http.MethodDelete, "/arena-shop/:id", "arena_shop.delete", handlers.AdminDeleteArenaShopItem, "admin.delete_arena_shop_item", "arena_shop_item", true, "path", false, false, true},
		// mine_slots
		{http.MethodGet, "/mine-slots", "mine_slots.read", handlers.AdminListMineSlots, "", "", false, "", false, false, false},
		{http.MethodPut, "/mine-slots/:id", "mine_slots.update", handlers.AdminUpdateMineSlot, "admin.update_mine_slot", "mine_slot", false, "", false, false, false},
		{http.MethodPost, "/mine-slots/:id/reset", "mine_slots.update", handlers.AdminResetMineSlot, "admin.reset_mine_slot", "mine_slot", false, "", false, false, false},
		// gu_dictionary
		{http.MethodGet, "/gu-dictionary", "gu_dictionary.read", handlers.AdminListGuItems, "", "", false, "", false, false, false},
		{http.MethodPost, "/gu-effects/validate", "gu_dictionary.update", handlers.AdminValidateGuEffects, "admin.validate_gu_effects", "gu_effect", false, "", false, false, false},
		{http.MethodGet, "/gu-effects/audit", "gu_dictionary.read", handlers.AdminAuditGuEffects, "", "", false, "", false, false, false},
		{http.MethodPost, "/effect-catalog/validate", "gu_dictionary.update", handlers.AdminValidateGuEffects, "admin.validate_gu_effects", "effect_catalog", false, "", false, false, false},
		{http.MethodGet, "/effect-catalog/metadata", "gu_dictionary.read", handlers.AdminEffectCatalogMetadata, "", "", false, "", false, false, false},
		{http.MethodGet, "/effect-catalog/audit", "gu_dictionary.read", handlers.AdminAuditEffectCatalog, "", "", false, "", false, false, false},
		{http.MethodPost, "/gu-dictionary", "gu_dictionary.update", handlers.AdminCreateGuItem, "admin.create_gu_item", "gu_dictionary", false, "", false, false, false},
		{http.MethodPut, "/gu-dictionary/:id", "gu_dictionary.update", handlers.AdminUpdateGuItem, "admin.update_gu_item", "gu_dictionary", false, "", false, false, false},
		{http.MethodDelete, "/gu-dictionary/:id", "gu_dictionary.delete", handlers.AdminDeleteGuItem, "admin.delete_gu_item", "gu_dictionary", true, "path", false, false, true},
		// item_dictionary
		{http.MethodGet, "/item-dictionary", "item_dictionary.read", handlers.AdminListItemItems, "", "", false, "", false, false, false},
		{http.MethodPost, "/item-dictionary", "item_dictionary.update", handlers.AdminCreateItemItem, "admin.create_item_item", "item_dictionary", false, "", false, false, false},
		{http.MethodPut, "/item-dictionary/:id", "item_dictionary.update", handlers.AdminUpdateItemItem, "admin.update_item_item", "item_dictionary", false, "", false, false, false},
		{http.MethodDelete, "/item-dictionary/:id", "item_dictionary.delete", handlers.AdminDeleteItemItem, "admin.delete_item_item", "item_dictionary", true, "path", false, false, true},
		// recipes
		{http.MethodGet, "/recipes", "recipes.read", handlers.AdminListRecipes, "", "", false, "", false, false, false},
		{http.MethodPost, "/recipes", "recipes.update", handlers.AdminCreateRecipe, "admin.create_recipe", "recipe", false, "", false, false, false},
		{http.MethodPut, "/recipes/:id", "recipes.update", handlers.AdminUpdateRecipe, "admin.update_recipe", "recipe", false, "", false, false, false},
		{http.MethodDelete, "/recipes/:id", "recipes.delete", handlers.AdminDeleteRecipe, "admin.delete_recipe", "recipe", true, "path", false, false, true},
		// monsters
		{http.MethodGet, "/monsters", "monsters.read", handlers.AdminListMonsters, "", "", false, "", false, false, false},
		{http.MethodPost, "/monsters", "monsters.update", handlers.AdminCreateMonster, "admin.create_monster", "monster", false, "", false, false, false},
		{http.MethodPut, "/monsters/:id", "monsters.update", handlers.AdminUpdateMonster, "admin.update_monster", "monster", false, "", false, false, false},
		{http.MethodDelete, "/monsters/:id", "monsters.delete", handlers.AdminDeleteMonster, "admin.delete_monster", "monster", true, "path", false, false, true},
		// loot_tables
		{http.MethodGet, "/loot-tables", "loot_tables.read", handlers.AdminListLootTables, "", "", false, "", false, false, false},
		{http.MethodPost, "/loot-tables", "loot_tables.update", handlers.AdminCreateLootTable, "admin.create_loot_table", "loot_table", false, "", false, false, false},
		{http.MethodPut, "/loot-tables/:id", "loot_tables.update", handlers.AdminUpdateLootTable, "admin.update_loot_table", "loot_table", false, "", false, false, false},
		{http.MethodDelete, "/loot-tables/:id", "loot_tables.delete", handlers.AdminDeleteLootTable, "admin.delete_loot_table", "loot_table", true, "path", false, false, true},
		// gift_codes
		{http.MethodPost, "/gift-codes", "gift_codes.update", handlers.AdminCreateGiftCode, "admin.create_gift_code", "gift_code", false, "", false, false, false},
		{http.MethodGet, "/gift-codes", "gift_codes.read", handlers.AdminListGiftCodes, "", "", false, "", false, false, false},
		// images
		{http.MethodPost, "/upload-image", "images.update", handlers.AdminUploadImage, "admin.upload_image", "image", false, "", false, false, false},
		// players
		{http.MethodPut, "/players/:id/gus/:instance_id", "players.update", handlers.AdminUpdatePlayerGu, "admin.update_player_gu", "player", false, "", false, false, false},
		{http.MethodPost, "/players/:id/pets", "players.update", handlers.AdminCreatePlayerPet, "admin.create_player_pet", "player", false, "", false, false, false},
		{http.MethodPut, "/players/:id/pets/:pet_id", "players.update", handlers.AdminUpdatePlayerPet, "admin.update_player_pet", "player", false, "", false, false, false},
		// constitution_dictionary
		{http.MethodGet, "/constitution-dictionary", "constitution_dictionary.read", handlers.AdminListConstitutions, "", "", false, "", false, false, false},
		{http.MethodPost, "/constitution-dictionary", "constitution_dictionary.update", handlers.AdminCreateConstitution, "admin.create_constitution", "constitution", false, "", false, false, false},
		{http.MethodPut, "/constitution-dictionary/:key", "constitution_dictionary.update", handlers.AdminUpdateConstitution, "admin.update_constitution", "constitution", false, "", false, false, false},
		{http.MethodDelete, "/constitution-dictionary/:key", "constitution_dictionary.delete", handlers.AdminDeleteConstitution, "admin.delete_constitution", "constitution", true, "path", false, false, true},
		// race_dictionary
		{http.MethodGet, "/race-dictionary", "race_dictionary.read", handlers.AdminListRaces, "", "", false, "", false, false, false},
		{http.MethodPost, "/race-dictionary", "race_dictionary.update", handlers.AdminCreateRace, "admin.create_race", "race", false, "", false, false, false},
		{http.MethodPut, "/race-dictionary/:key", "race_dictionary.update", handlers.AdminUpdateRace, "admin.update_race", "race", false, "", false, false, false},
		{http.MethodDelete, "/race-dictionary/:key", "race_dictionary.delete", handlers.AdminDeleteRace, "admin.delete_race", "race", true, "path", false, false, true},
		// monitoring
		{http.MethodGet, "/batch-metrics", "monitoring.read", handlers.GetBatchMetrics, "", "", false, "", false, false, false},
		// status_dictionary
		{http.MethodGet, "/status-dictionary", "status_dictionary.read", handlers.AdminListStatuses, "", "", false, "", false, false, false},
		{http.MethodPost, "/status-dictionary", "status_dictionary.update", handlers.AdminCreateStatus, "admin.create_status", "status", false, "", false, false, false},
		{http.MethodPut, "/status-dictionary/:key", "status_dictionary.update", handlers.AdminUpdateStatus, "admin.update_status", "status", false, "", false, false, false},
		{http.MethodDelete, "/status-dictionary/:key", "status_dictionary.delete", handlers.AdminDeleteStatus, "admin.delete_status", "status", true, "path", false, false, true},
		// monster_spawns
		{http.MethodGet, "/monster-spawns", "monster_spawns.read", handlers.AdminListMonsterSpawns, "", "", false, "", false, false, false},
		{http.MethodPost, "/monster-spawns", "monster_spawns.update", handlers.AdminCreateMonsterSpawn, "admin.create_monster_spawn", "monster_spawn", false, "", false, false, false},
		{http.MethodPut, "/monster-spawns/:id", "monster_spawns.update", handlers.AdminUpdateMonsterSpawn, "admin.update_monster_spawn", "monster_spawn", false, "", false, false, false},
		{http.MethodDelete, "/monster-spawns/:id", "monster_spawns.delete", handlers.AdminDeleteMonsterSpawn, "admin.delete_monster_spawn", "monster_spawn", true, "path", false, false, true},
		// quests
		{http.MethodGet, "/quests", "quests.read", handlers.AdminListQuests, "", "", false, "", false, false, false},
		{http.MethodPost, "/quests", "quests.update", handlers.AdminCreateQuest, "admin.create_quest", "quest", false, "", false, false, false},
		{http.MethodPut, "/quests/:id", "quests.update", handlers.AdminUpdateQuest, "admin.update_quest", "quest", false, "", false, false, false},
		{http.MethodDelete, "/quests/:id", "quests.delete", handlers.AdminDeleteQuest, "admin.delete_quest", "quest", true, "path", false, false, true},
		// dungeons
		{http.MethodGet, "/dungeons", "dungeons.read", handlers.AdminListDungeons, "", "", false, "", false, false, false},
		{http.MethodPost, "/dungeons", "dungeons.update", handlers.AdminCreateDungeon, "admin.create_dungeon", "dungeon", false, "", false, false, false},
		{http.MethodPut, "/dungeons/:id", "dungeons.update", handlers.AdminUpdateDungeon, "admin.update_dungeon", "dungeon", false, "", false, false, false},
		{http.MethodDelete, "/dungeons/:id", "dungeons.delete", handlers.AdminDeleteDungeon, "admin.delete_dungeon", "dungeon", true, "path", false, false, true},
		// sect_mines
		{http.MethodGet, "/sect-mines", "sect_mines.read", handlers.AdminListSectMines, "", "", false, "", false, false, false},
		{http.MethodPut, "/sect-mines/:id", "sect_mines.update", handlers.AdminUpdateSectMine, "admin.update_sect_mine", "sect_mine", false, "", false, false, false},
		// apertures
		{http.MethodGet, "/apertures/:id/nodes", "apertures.read", handlers.AdminListApertureNodes, "", "", false, "", false, false, false},
		{http.MethodGet, "/apertures/:id/tribulations", "apertures.read", handlers.AdminListApertureTribulations, "", "", false, "", false, false, false},
		{http.MethodPost, "/apertures/:id/nodes", "apertures.update", handlers.AdminCreateApertureNode, "admin.create_aperture_node", "aperture", false, "", false, false, false},
		{http.MethodPut, "/apertures/:id/nodes/:node_id", "apertures.update", handlers.AdminUpdateApertureNode, "admin.update_aperture_node", "aperture", false, "", false, false, false},
		{http.MethodDelete, "/apertures/:id/nodes/:node_id", "apertures.delete", handlers.AdminDeleteApertureNode, "admin.delete_aperture_node", "aperture", true, "path", false, false, true},
		{http.MethodGet, "/apertures/:id/nodes/:node_id/workers", "apertures.read", handlers.AdminListApertureNodeWorkers, "", "", false, "", false, false, false},
		{http.MethodPost, "/apertures/:id/nodes/:node_id/workers", "apertures.update", handlers.AdminUpsertApertureNodeWorker, "admin.upsert_aperture_node_worker", "aperture", false, "", false, false, false},
		{http.MethodDelete, "/apertures/:id/nodes/:node_id/workers/:worker_id", "apertures.delete", handlers.AdminDeleteApertureNodeWorker, "admin.delete_aperture_node_worker", "aperture", true, "path", false, false, true},
		{http.MethodGet, "/apertures/:id/refinement-queue", "apertures.read", handlers.AdminListApertureRefinementJobs, "", "", false, "", false, false, false},
		{http.MethodPost, "/apertures/:id/refinement-queue", "apertures.update", handlers.AdminCreateApertureRefinementJob, "admin.create_aperture_refinement_job", "aperture", false, "", false, false, false},
		{http.MethodPut, "/apertures/:id/refinement-queue/:job_id", "apertures.update", handlers.AdminUpdateApertureRefinementJob, "admin.update_aperture_refinement_job", "aperture", false, "", false, false, false},
		{http.MethodDelete, "/apertures/:id/refinement-queue/:job_id", "apertures.delete", handlers.AdminDeleteApertureRefinementJob, "admin.delete_aperture_refinement_job", "aperture", true, "path", false, false, true},
		{http.MethodGet, "/apertures/:id/settlements", "apertures.read", handlers.AdminListApertureSettlements, "", "", false, "", false, false, false},
		{http.MethodPost, "/apertures/:id/settlements", "apertures.update", handlers.AdminCreateApertureSettlement, "admin.create_aperture_settlement", "aperture", false, "", false, false, false},
		{http.MethodPut, "/apertures/:id/settlements/:settlement_id", "apertures.update", handlers.AdminUpdateApertureSettlement, "admin.update_aperture_settlement", "aperture", false, "", false, false, false},
		{http.MethodDelete, "/apertures/:id/settlements/:settlement_id", "apertures.delete", handlers.AdminDeleteApertureSettlement, "admin.delete_aperture_settlement", "aperture", true, "path", false, false, true},
		{http.MethodGet, "/apertures/:id/events", "apertures.read", handlers.AdminListApertureEvents, "", "", false, "", false, false, false},
		{http.MethodPost, "/apertures/:id/events", "apertures.update", handlers.AdminCreateApertureEvent, "admin.create_aperture_event", "aperture", false, "", false, false, false},
		{http.MethodPut, "/apertures/:id/events/:event_id", "apertures.update", handlers.AdminUpdateApertureEvent, "admin.update_aperture_event", "aperture", false, "", false, false, false},
		{http.MethodDelete, "/apertures/:id/events/:event_id", "apertures.delete", handlers.AdminDeleteApertureEvent, "admin.delete_aperture_event", "aperture", true, "path", false, false, true},
		{http.MethodGet, "/apertures/:id/variant-npcs", "apertures.read", handlers.AdminListVariantNPCs, "", "", false, "", false, false, false},
		{http.MethodPost, "/apertures/:id/variant-npcs", "apertures.update", handlers.AdminCreateVariantNPC, "admin.create_variant_npc", "aperture", false, "", false, false, false},
		{http.MethodPut, "/apertures/:id/variant-npcs/:npc_id", "apertures.update", handlers.AdminUpdateVariantNPC, "admin.update_variant_npc", "aperture", false, "", false, false, false},
		{http.MethodDelete, "/apertures/:id/variant-npcs/:npc_id", "apertures.delete", handlers.AdminDismissVariantNPC, "admin.dismiss_variant_npc", "aperture", true, "path", false, false, true},
		{http.MethodGet, "/apertures/:id/variant-humans", "apertures.read", handlers.AdminListApertureVariantHumans, "", "", false, "", false, false, false},
		{http.MethodPost, "/apertures/:id/variant-humans", "apertures.update", handlers.AdminCreateApertureVariantHuman, "admin.create_aperture_variant_human", "aperture", false, "", false, false, false},
		{http.MethodPut, "/apertures/:id/variant-humans/:vh_id", "apertures.update", handlers.AdminUpdateApertureVariantHuman, "admin.update_aperture_variant_human", "aperture", false, "", false, false, false},
		{http.MethodDelete, "/apertures/:id/variant-humans/:vh_id", "apertures.delete", handlers.AdminDeleteApertureVariantHuman, "admin.delete_aperture_variant_human", "aperture", true, "path", false, false, true},
	}
}

// requestLogging logs the start and end of every request so that hung
// handlers are visible (Gin's own logger only prints after completion).
// It also flags requests that exceed slowRequestThreshold.
func requestLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		traceID, _ := c.Get("trace_id")

		if method != http.MethodOptions {
			logger.L.Debug("http_request_started", zap.Any("trace_id", traceID), zap.String("method", method), zap.String("path", path), zap.String("client_ip", clientIP))
		}

		c.Next()

		duration := time.Since(start)
		if method == http.MethodOptions {
			return
		}

		status := c.Writer.Status()
		level := zapcore.InfoLevel
		switch {
		case status >= 500:
			level = zapcore.ErrorLevel
		case duration >= slowRequestThreshold:
			level = zapcore.WarnLevel
		case status >= 400:
			level = zapcore.WarnLevel
		}

		traceID, _ = c.Get("trace_id")
		playerID, _ := c.Get("player_id")
		username, _ := c.Get("username")
		requestBody, _ := c.Get("request_body")
		bodyTruncated, _ := c.Get("request_body_truncated")
		responseBodyRaw, _ := c.Get("response_body")
		responseBodyTruncatedRaw, _ := c.Get("response_body_truncated")
		errMsgRaw, _ := c.Get("error_msg")
		var responseBody, errMsg string
		responseBodyTruncated, _ := responseBodyTruncatedRaw.(bool)
		if v, ok := responseBodyRaw.(string); ok {
			responseBody = v
		}
		if v, ok := errMsgRaw.(string); ok {
			errMsg = v
		}

		isTruncated, _ := bodyTruncated.(bool)

		var respBodyField zapcore.Field
		if responseBody != "" {
			var parsedJSON interface{}
			if err := json.Unmarshal([]byte(responseBody), &parsedJSON); err == nil {
				respBodyField = zap.Any("response_body", parsedJSON)
			} else {
				respBodyField = zap.String("response_body", responseBody)
			}
		} else {
			respBodyField = zap.String("response_body", "")
		}

		if status >= 400 {
			if checked := logger.L.Check(level, "http_request_error"); checked != nil {
				checked.Write(
					zap.Any("trace_id", traceID), zap.String("method", method), zap.String("path", path),
					zap.String("route", c.FullPath()), zap.Int("status", status), zap.Float64("duration_ms", float64(duration.Microseconds())/1000),
					zap.String("client_ip", clientIP), zap.Any("player_id", playerID), zap.Any("username", username),
					zap.Any("request_body", requestBody), zap.Bool("body_truncated", isTruncated),
					respBodyField, zap.Bool("response_body_truncated", responseBodyTruncated), zap.String("error", errMsg),
					zap.String("user_agent", c.Request.UserAgent()),
				)
			}
		}

		if checked := logger.L.Check(level, "http_request_completed"); checked != nil {
			fields := []zapcore.Field{
				zap.Any("trace_id", traceID), zap.String("method", method), zap.String("path", path),
				zap.String("route", c.FullPath()), zap.Int("status", status), zap.Float64("duration_ms", float64(duration.Microseconds())/1000),
				zap.String("client_ip", clientIP), zap.Any("player_id", playerID), zap.Any("username", username),
				zap.Any("request_body", requestBody), zap.Bool("body_truncated", isTruncated),
			}
			if status >= 400 {
				fields = append(fields, respBodyField, zap.Bool("response_body_truncated", responseBodyTruncated), zap.String("error", errMsg))
			}
			fields = append(fields, zap.String("user_agent", c.Request.UserAgent()))
			checked.Write(fields...)
		}
	}
}

// SetupRouter initializes the Gin router with middlewares and routes
func SetupRouter(shub *ws.ShardedHub) *gin.Engine {
	if shub != nil {
		handlers.SetWSStatsProvider(shub.GetStats)
		handlers.SetWSConnectedProvider(shub.IsConnected)
		handlers.WorldPosBroadcaster = func(channel string, payload []byte, targets map[uuid.UUID]struct{}) {
			shub.BroadcastToTargets(ws.ChannelName(channel), worldEventEnvelope(channel, payload), targets)
		}
		handlers.WorldCellBroadcaster = func(channel string, payload []byte, targets map[uuid.UUID]struct{}) {
			shub.BroadcastToTargets(ws.ChannelName(channel), worldEventEnvelope(channel, payload), targets)
		}
		handlers.WorldCombatBroadcaster = func(channel string, payload []byte, targets map[uuid.UUID]struct{}) {
			shub.BroadcastToTargets(ws.ChannelName(channel), worldEventEnvelope(channel, payload), targets)
		}
	}
	worker.PlayerEventPublisher = func(playerID uuid.UUID, eventType string, domains ...string) {
		typedDomains := make([]handlers.PlayerEventDomain, 0, len(domains))
		for _, domain := range domains {
			typedDomains = append(typedDomains, handlers.PlayerEventDomain(domain))
		}
		handlers.PublishPlayerRealtimeEvent(playerID, eventType, typedDomains...)
	}
	gin.DefaultWriter = logger.GinWriter()
	r := gin.New()
	r.Use(gin.Recovery())

	// Capture the response before requestLogging reads the values from the context.
	r.Use(requestLogging())
	r.Use(monitor.Middleware())

	// CORS Middleware
	r.Use(func(c *gin.Context) {
		allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
		if allowedOriginsStr == "" {
			allowedOriginsStr = "http://localhost:3000"
		}
		allowedOrigins := strings.Split(allowedOriginsStr, ",")

		origin := c.Request.Header.Get("Origin")
		isAllowed := false
		isWildcardAllowed := false
		for _, o := range allowedOrigins {
			o = strings.TrimSpace(o)
			if o == "*" {
				isWildcardAllowed = true
			} else if o == origin {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		} else if isWildcardAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if allowedOriginsStr == "*" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, X-Admin-Reason, X-Admin-Confirmation-Target, Idempotency-Key, Authorization, X-Player-Tab-ID, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Compress responses with Brotli/Gzip/Zstd based on client Accept-Encoding.
	// Skip websocket route.
	r.Use(compress.Compress(
		compress.WithExcludeFunc(func(c *gin.Context) bool {
			return c.Request.URL.Path == "/api/ws"
		}),
	))

	// Serve uploaded images
	r.Static("/images", "./storage/images")

	// API routes
	api := r.Group("/api")
	{
		// WebSocket route
		api.GET("/ws", middleware.AuthRequired(), func(c *gin.Context) {
			if shub == nil {
				c.JSON(503, gin.H{"error": "WebSocket hub not initialized"})
				return
			}
			playerID, exists := middleware.GetPlayerID(c)
			if !exists {
				c.JSON(401, gin.H{"error": "Unauthorized"})
				return
			}
			// Check connection rate limit
			ip := c.ClientIP()
			if !ws.IsConnectionAllowed(playerID, ip) {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "Quá nhiều kết nối WebSocket. Vui lòng đóng các tab khác và thử lại!"})
				return
			}
			ws.ServeWs(shub, c.Writer, c.Request, playerID, ip)
		})

		// Public endpoints — per IP
		api.GET("/dictionary", middleware.RateLimitByIP(120, time.Minute), handlers.GetDictionary)
		api.GET("/dictionary/gus", middleware.RateLimitByIP(120, time.Minute), handlers.GetGuDictionary)
		api.GET("/dictionary/items", middleware.RateLimitByIP(120, time.Minute), handlers.GetItemDictionary)
		api.GET("/dictionary/monsters", middleware.RateLimitByIP(120, time.Minute), handlers.GetMonsterDictionary)
		api.GET("/dictionary/constitutions", middleware.RateLimitByIP(120, time.Minute), handlers.GetConstitutionDictionary)
		api.GET("/dictionary/races", middleware.RateLimitByIP(120, time.Minute), handlers.GetRaceDictionary)
		api.GET("/dictionary/statuses", middleware.RateLimitByIP(120, time.Minute), handlers.GetStatusDictionary)
		api.GET("/dictionary/loot-tables", middleware.RateLimitByIP(120, time.Minute), handlers.GetLootTables)
		api.GET("/dictionary/regions", middleware.RateLimitByIP(120, time.Minute), handlers.GetDictionaryRegions)
		api.GET("/dictionary/monster-spawns", middleware.RateLimitByIP(120, time.Minute), handlers.GetMonsterSpawns)
		api.GET("/recipes", middleware.RateLimitByIP(60, time.Minute), handlers.GetRecipes)
		api.GET("/gus/:instance_id/history", middleware.RateLimitByIP(60, time.Minute), handlers.GetGuHistory)

		// Auth Endpoints
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/send-otp", middleware.RateLimitByIP(1, time.Minute), handlers.SendOTP)
			authGroup.POST("/register", middleware.RateLimitByIP(1, time.Minute), handlers.RegisterPlayer)
			authGroup.POST("/login", middleware.RateLimitByIP(9, time.Minute), handlers.LoginPlayer)
			authGroup.POST("/logout", handlers.LogoutPlayer)
			authGroup.POST("/claim-tab", middleware.AuthRequired(), handlers.ClaimGameplayTab)
			authGroup.GET("/me", middleware.AuthRequired(), handlers.GetCurrentPlayer)

			// Google OAuth2
			authGroup.GET("/google/login", middleware.RateLimitByIP(9, time.Minute), handlers.GoogleLogin)
			authGroup.GET("/google/callback", handlers.GoogleCallback)
		}

		api.GET("/topup/packages", handlers.GetTopupPackages)
		api.POST("/webhooks/payos", handlers.PayOSWebhook)

		// Admin endpoints — group: 80/min per IP. PermissionRequired is attached
		// per route from the domain-oriented inventory. Handler-level admin checks
		// remain as defense in depth during the RBAC transition.
		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired())
		admin.Use(middleware.RateLimitByGroup("admin", 80, time.Minute))
		for _, route := range adminRouteSpecs() {
			handlers := []gin.HandlerFunc{}
			if route.auditAction != "" {
				handlers = append(handlers, middleware.AdminMutationAudit(route.auditAction, route.auditTargetType, route.permission, route.destructive))
			}
			handlers = append(handlers, middleware.PermissionRequired(route.permission))
			if route.reasonRequired {
				handlers = append(handlers, middleware.AdminReasonRequired())
			}
			if route.confirmationTarget != "" {
				target := route.confirmationTarget
				handlers = append(handlers, middleware.AdminConfirmationRequired(func(c *gin.Context) string {
					switch target {
					case "path":
						return c.Request.URL.Path
					case "id", "table":
						return c.Param(target)
					default:
						return target
					}
				}))
			}
			if route.reauthenticationRequired {
				handlers = append(handlers, middleware.AdminPasswordReauthentication())
			}
			if route.idempotencyKeyRequired {
				handlers = append(handlers, middleware.AdminIdempotencyKeyRequired())
			}
			handlers = append(handlers, route.handler)
			admin.Handle(route.method, route.path, handlers...)
		}

		api.GET("/wallet", middleware.AuthRequired(), handlers.GetWallet)
		api.POST("/topup/orders", middleware.AuthRequired(), middleware.RequireOnboarded(), middleware.RateLimitByPlayer(10, time.Minute), handlers.CreateTopupOrder)
		api.GET("/topup/orders/:orderCode", middleware.AuthRequired(), handlers.GetTopupOrder)

		// Player endpoints — group: 60/min, per player: 30/min
		player := api.Group("/player")
		player.Use(middleware.AuthRequired())
		player.Use(middleware.RequireOnboarded())
		player.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			player.GET("/status/:id", handlers.GetPlayerStatus)
			player.GET("/inventory/gus", handlers.GetPlayerInventoryGus)
			player.GET("/stats", handlers.GetPlayerStatsHandler)
			player.GET("/passive-stats", handlers.GetPassiveStatsHandler)
			player.POST("/explore/:id", handlers.Explore)
			player.POST("/use-item", handlers.UseItem)
			player.POST("/use-items", handlers.UseItems)
			player.POST("/recharge", handlers.RechargeWithStones)
			player.POST("/clear-stress", handlers.ClearStress)
			player.GET("/profile/:username", handlers.GetPlayerProfile)
			player.PUT("/profile/settings", handlers.UpdateProfileSettings)
			player.GET("/referrals", handlers.GetReferralStats)
			player.POST("/referrals/claim", handlers.ClaimReferralReward)
			player.POST("/referrals/bind", handlers.BindReferralCode)
			player.POST("/simulate-deposit", middleware.RequirePermission(authz.AdminAccessPermission), handlers.SimulateDeposit)
			player.POST("/avatar", handlers.UploadAvatar)
			player.POST("/redeem-gift-code", handlers.RedeemGiftCode)
			player.POST("/complete-tutorial", handlers.CompleteTutorial)
			player.POST("/complete-onboarding", handlers.CompleteOnboarding)
			player.GET("/combat-settings", handlers.GetCombatSettings)
			player.PUT("/combat-settings", handlers.UpdateCombatSettings)
			player.GET("/audit-logs", handlers.GetMyAuditLogs)
			player.GET("/notifications", handlers.GetNotifications)
			player.GET("/notifications/unread-count", handlers.GetUnreadNotificationsCount)
			player.POST("/notifications/read", handlers.MarkNotificationsRead)
			player.POST("/notifications/read-all", handlers.MarkAllNotificationsRead)

			// Beast betting
			player.GET("/beast-betting/current", handlers.GetCurrentBettingRound)
			player.GET("/beast-betting/history", handlers.GetBettingHistory)
			player.POST("/beast-betting/bet", handlers.PlaceBeastBet)
		}

		// Mine — group: 40/min, per player: 25/min
		mine := api.Group("/mine")
		mine.Use(middleware.AuthRequired())
		mine.Use(middleware.RequireOnboarded())
		mine.Use(middleware.RateLimitByPlayer(25, time.Minute))
		{
			mine.GET("/slots", handlers.GetMineSlots)
			mine.POST("/occupy", handlers.OccupyMineSlot)
			mine.POST("/claim", handlers.ClaimMineRewards)
			mine.POST("/abandon", handlers.AbandonMineSlot)
			mine.POST("/attack", handlers.AttackMineSlot)
		}

		// Cultivate — group: 30/min, per player: 20/min
		cultivate := api.Group("/cultivate")
		cultivate.Use(middleware.AuthRequired())
		cultivate.Use(middleware.RequireOnboarded())
		cultivate.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			cultivate.POST("/break-wall", handlers.BreakWall)
			cultivate.POST("/use-relic-gu", handlers.UseRelicGu)
			cultivate.POST("/ascension", handlers.Ascension)
			cultivate.POST("/tribulation", handlers.Tribulation)
		}

		// Quest — group: 40/min, per player: 25/min
		quest := api.Group("/quest")
		quest.Use(middleware.AuthRequired())
		quest.Use(middleware.RequireOnboarded())
		quest.Use(middleware.RateLimitByPlayer(25, time.Minute))
		{
			quest.GET("", handlers.ListQuests)
			quest.POST("/complete", handlers.CompleteQuest)
		}

		// Gacha — group: 50/min, per player: 15/min
		gacha := api.Group("/gacha")
		gacha.Use(middleware.AuthRequired())
		gacha.Use(middleware.RequireOnboarded())
		gacha.Use(middleware.RateLimitByGroup("gacha", 200, time.Minute))
		gacha.GET("/pools", handlers.GetGachaPools)
		gacha.Use(middleware.RateLimitByPlayer(1, time.Minute))
		{
			gacha.POST("/open", handlers.OpenGacha)
			gacha.POST("/open10", handlers.OpenGacha10)
		}

		// Achievement — group: 30/min, per player: 20/min
		achievement := api.Group("/achievements")
		achievement.Use(middleware.AuthRequired())
		achievement.Use(middleware.RequireOnboarded())
		achievement.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			achievement.GET("", handlers.GetAchievements)
			achievement.POST("/claim", handlers.ClaimAchievement)
			achievement.POST("/select", handlers.SelectTitleFrame)
		}

		dailyMissions := api.Group("/daily-missions")
		dailyMissions.Use(middleware.AuthRequired())
		dailyMissions.Use(middleware.RequireOnboarded())
		dailyMissions.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			dailyMissions.GET("", handlers.GetDailyMissions)
			dailyMissions.POST("/claim", handlers.ClaimDailyMission)
		}

		// Party — group: 30/min, per player: 20/min
		party := api.Group("/party")
		party.Use(middleware.AuthRequired())
		party.Use(middleware.RequireOnboarded())
		party.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			party.POST("/create", handlers.CreateParty)
			party.GET("/status", handlers.GetPartyStatus)
			party.POST("/invite", handlers.InvitePlayer)
			party.POST("/leave", handlers.LeaveParty)
			party.GET("/invitations", handlers.GetPendingInvitations)
			party.POST("/invitations/:id/respond", handlers.RespondToInvitation)
			party.POST("/formation", handlers.SavePartyFormation)
		}

		// Inventory — group: 30/min, per player: 20/min
		inventory := api.Group("/inventory")
		inventory.Use(middleware.AuthRequired())
		inventory.Use(middleware.RequireOnboarded())
		inventory.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			inventory.POST("/feed", handlers.FeedGu)
			inventory.POST("/feed-group", handlers.FeedGroup)
			inventory.POST("/feed-all", handlers.FeedAll)
			inventory.POST("/equip", handlers.EquipGu)
			inventory.POST("/unequip-all", handlers.UnequipAllGu)
			inventory.POST("/status", handlers.UpdateGuStatus)
			inventory.POST("/set-vital", handlers.SetVitalGu)
			inventory.POST("/use-gu", handlers.UseConsumableGu)
			inventory.POST("/use-support", handlers.UseSupportGu)
			inventory.POST("/drop", handlers.DropGu)
			inventory.POST("/drop-batch", handlers.DropGus)
		}

		// Refining — group: 30/min, per player: 20/min
		refining := api.Group("/refine")
		refining.Use(middleware.AuthRequired())
		refining.Use(middleware.RequireOnboarded())
		refining.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			refining.POST("", handlers.RefineGu)
			refining.POST("/deduce", handlers.DeduceRecipe)
		}

		// Scouting & Divination — group: 60/min, per player: 30/min
		scouting := api.Group("/scouting")
		scouting.Use(middleware.AuthRequired())
		scouting.Use(middleware.RequireOnboarded())
		scouting.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			scouting.POST("/inspect", handlers.InspectLocalPlayer)
			scouting.POST("/divine", handlers.ExecuteDivination)
			scouting.POST("/divine-gu", handlers.ExecuteDivineGu)
			scouting.POST("/apply-mark", handlers.ApplyTrackingMark)
			scouting.GET("/tracked-targets", handlers.GetTrackedTargets)
			scouting.POST("/sever-clues", handlers.SeverClues)
			scouting.POST("/resonance-scan", handlers.ResonanceScan)
		}

		// Pets — group: 30/min, per player: 20/min
		pets := api.Group("/pets")
		pets.Use(middleware.AuthRequired())
		pets.Use(middleware.RequireOnboarded())
		pets.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			pets.GET("", handlers.ListPlayerPets)
			pets.POST("/feed", handlers.FeedPet)
			pets.POST("/formation", handlers.SetPetFormation)
			pets.POST("/sleep", handlers.SetPetSleep)
			pets.POST("/release", handlers.ReleasePet)
			pets.POST("/merge", handlers.MergePets)
		}

		// Canonical actor roster. Domain-specific mutations remain under the pet
		// and aperture routes so each command can enforce its own invariants.
		actors := api.Group("/actors")
		actors.Use(middleware.AuthRequired())
		actors.Use(middleware.RequireOnboarded())
		actors.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			actors.GET("", handlers.ListActors)
			actors.GET("/:id", handlers.GetActor)
		}

		// Combat — group: 60/min, per player: 60/min
		combat := api.Group("/combat")
		combat.Use(middleware.AuthRequired())
		combat.Use(middleware.RequireOnboarded())
		combat.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			combat.POST("/start", handlers.AutoCombatStart)
			combat.POST("/practice", handlers.PracticeCombatStart)
			combat.POST("/capture", handlers.CaptureMonster)
			combat.GET("/history", handlers.ListCombatHistory)
			combat.GET("/history/:id", handlers.GetCombatHistory)
		}

		// Player-created killer moves use Gu components from player_gu and are
		// validated again at preview, save, equip, and combat activation.
		killerMoves := api.Group("/killer-moves")
		killerMoves.Use(middleware.AuthRequired())
		killerMoves.Use(middleware.RequireOnboarded())
		killerMoves.Use(middleware.RateLimitByPlayer(30, time.Minute))
		{
			killerMoves.GET("/options", handlers.GetKillerMoveOptions)
			killerMoves.GET("", handlers.ListKillerMoves)
			killerMoves.POST("/preview", handlers.PreviewKillerMove)
			killerMoves.POST("", handlers.CreateKillerMove)
			killerMoves.POST("/:id/equip", handlers.EquipKillerMove)
			killerMoves.POST("/:id/use", handlers.UseKillerMove)
			killerMoves.DELETE("/:id", handlers.DeleteKillerMove)
		}

		// Immortal Aperture — group: 30/min, per player: 20/min
		aperture := api.Group("/immortal-aperture")
		aperture.Use(middleware.AuthRequired())
		aperture.Use(middleware.RequireOnboarded())
		aperture.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			aperture.GET("", handlers.GetImmortalAperture)
			aperture.GET("/nodes", handlers.GetApertureNodes)
			aperture.POST("/nodes", handlers.PlaceApertureNode)
			aperture.POST("/nodes/batch", handlers.ManageApertureNodesBatch)
			aperture.POST("/nodes/:id/collect", handlers.CollectApertureResource)
			aperture.POST("/nodes/collect-all", handlers.CollectAllApertureResources)
			aperture.POST("/nodes/repair-all", handlers.RepairAllApertureNodes)
			aperture.POST("/nodes/:id/uninstall", handlers.UninstallApertureNode)
			aperture.DELETE("/nodes/:id", handlers.DeleteApertureNode)
			aperture.POST("/time-rate", handlers.SetApertureTimeRate)
			aperture.POST("/labor", handlers.SetApertureLabor)
			aperture.POST("/cleanse-dao-marks", handlers.CleanseDaoMarks)
			aperture.POST("/nodes/:id/repair", handlers.RepairApertureNode)
			aperture.POST("/nodes/:id/upgrade", handlers.UpgradeApertureNode)
			aperture.GET("/variant-humans", handlers.GetApertureVariantHumans)

			// Phase 1: Variant Economy — node workers
			aperture.GET("/nodes/:id/workers", handlers.GetApertureNodeWorkers)
			aperture.POST("/nodes/:id/workers", handlers.AssignWorkerToNode)
			aperture.DELETE("/nodes/:id/workers/:worker_id", handlers.RemoveWorkerFromNode)

			// Phase 1: Variant Economy — refinement queue
			aperture.GET("/refinement-queue", handlers.ListRefinementQueue)
			aperture.POST("/refinement-queue", handlers.CreateRefinementJob)
			aperture.DELETE("/refinement-queue/:job_id", handlers.DeleteRefinementJob)

			// Phase 2: Society — settlements
			aperture.GET("/settlements", handlers.GetSettlements)
			aperture.POST("/settlements", handlers.BuildSettlement)
			aperture.POST("/settlements/:settlement_id/upgrade", handlers.UpgradeSettlement)

			// Phase 3: Variant NPCs
			aperture.GET("/npcs", handlers.GetApertureNPCs)
			aperture.POST("/npcs/:npc_id/role", handlers.AssignNPCRole)
			aperture.POST("/npcs/:npc_id/formation", handlers.SetNPCFormation)
			aperture.POST("/npcs/:npc_id/cultivate", handlers.CultivateNPC)
			aperture.POST("/npcs/:npc_id/gu/equip", handlers.EquipNPCGu)
			aperture.DELETE("/npcs/:npc_id/gu/:gu_id", handlers.UnequipNPCGu)
			aperture.DELETE("/npcs/:npc_id", handlers.DismissNPC)
		}

		// Dao Marks — group: 20/min, per player: 15/min
		daoMarks := api.Group("/dao-marks")
		daoMarks.Use(middleware.AuthRequired())
		daoMarks.Use(middleware.RequireOnboarded())
		daoMarks.Use(middleware.RateLimitByPlayer(15, time.Minute))
		{
			daoMarks.GET("", handlers.GetDaoMarks)
			daoMarks.GET("/dao-ngan", handlers.GetDaoNganHandler)
		}

		// Tribulations — group: 30/min, per player: 20/min
		tribulations := api.Group("/tribulations")
		tribulations.Use(middleware.AuthRequired())
		tribulations.Use(middleware.RequireOnboarded())
		tribulations.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			tribulations.GET("/active", handlers.GetActiveTribulation)
			tribulations.POST("/:eventId/engage", handlers.EngageTribulation)
			tribulations.POST("/:eventId/resolve", handlers.ResolveTribulation)
		}

		// Arena — group: 40/min, per player: 25/min
		arena := api.Group("/arena")
		arena.Use(middleware.AuthRequired())
		arena.Use(middleware.RequireOnboarded())
		arena.Use(middleware.RateLimitByPlayer(25, time.Minute))
		{
			arena.GET("/leaderboard", handlers.GetArenaLeaderboard)
			arena.POST("/challenge", handlers.ChallengeArena)
			arena.GET("/shop", handlers.GetArenaShop)
			arena.POST("/shop/buy", handlers.BuyArenaShopItem)
		}

		// Death Arena — group: 40/min, per player: 25/min
		deathArena := api.Group("/death-arena")
		deathArena.Use(middleware.AuthRequired())
		deathArena.Use(middleware.RequireOnboarded())
		deathArena.Use(middleware.RateLimitByPlayer(25, time.Minute))
		{
			deathArena.GET("/current", handlers.GetCurrentTournament)
			deathArena.POST("/register", handlers.RegisterForTournament)
			deathArena.DELETE("/register", handlers.UnregisterFromTournament)
			deathArena.GET("/bracket", handlers.GetBracket)
			deathArena.GET("/my-team", handlers.GetMyTeam)
			deathArena.GET("/matches", handlers.GetMyMatches)
			deathArena.GET("/matches/:id/log", handlers.GetMatchCombatLog)
			deathArena.GET("/history", handlers.GetTournamentHistory)
			deathArena.GET("/rewards", handlers.GetMyRewards)
			deathArena.POST("/claim-reward", handlers.ClaimReward)
		}

		martialArena := api.Group("/martial-arena")
		martialArena.Use(middleware.AuthRequired())
		martialArena.Use(middleware.RequireOnboarded())
		martialArena.Use(middleware.RateLimitByPlayer(25, time.Minute))
		{
			martialArena.GET("/current", handlers.GetCurrentMartialTournament)
			martialArena.GET("/bracket", handlers.GetMartialBracket)
			martialArena.GET("/matches", handlers.GetMyMartialMatches)
			martialArena.GET("/matches/:id/log", handlers.GetMartialMatchCombatLog)
			martialArena.GET("/history", handlers.GetMartialHistory)
			martialArena.GET("/rewards", handlers.GetMyMartialRewards)
			martialArena.POST("/claim-reward", handlers.ClaimMartialReward)
		}

		// Leaderboard — group: 30/min, per player: 20/min
		leaderboard := api.Group("/leaderboard")
		leaderboard.Use(middleware.AuthRequired())
		leaderboard.Use(middleware.RequireOnboarded())
		leaderboard.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			leaderboard.GET("/:category", handlers.GetLeaderboard)
			leaderboard.POST("/toggle", handlers.ToggleLeaderboardVisibility)
			leaderboard.GET("/settings", handlers.GetLeaderboardSettings)
		}

		// Chat — no group limit, per-route IP limit
		chat := api.Group("/chat")
		chat.Use(middleware.AuthRequired())
		chat.Use(middleware.RequireOnboarded())
		{
			chat.POST("/mail/send", middleware.RateLimitByPlayer(20, time.Minute), handlers.SendPrivateMail)
			chat.GET("/mail/inbox", middleware.RateLimitByPlayer(30, time.Minute), handlers.GetInbox)
			chat.GET("/mail/conversations", middleware.RateLimitByPlayer(60, time.Minute), handlers.GetPrivateConversations)
			chat.GET("/mail/thread", middleware.RateLimitByPlayer(60, time.Minute), handlers.GetPrivateThread)
			chat.POST("/bao-hoang/send", middleware.RateLimitByIP(30, time.Minute), handlers.SendBaoHoangChat)
			chat.GET("/bao-hoang/messages", middleware.RateLimitByPlayer(60, time.Minute), handlers.GetBaoHoangMessages)
		}

		// Explore — per player: 25/min
		api.POST("/explore", middleware.AuthRequired(), middleware.RequireOnboarded(), middleware.RateLimitByPlayer(60, time.Minute), handlers.ExploreNode)
		api.POST("/explore/retreat/:id", middleware.AuthRequired(), middleware.RequireOnboarded(), middleware.RateLimitByPlayer(60, time.Minute), handlers.Retreat)
		api.POST("/explore/flee/:id", middleware.AuthRequired(), middleware.RequireOnboarded(), middleware.RateLimitByPlayer(60, time.Minute), handlers.Flee)
		api.GET("/explore/grid", middleware.AuthRequired(), middleware.RateLimitByPlayer(120, time.Minute), handlers.GetGrid)
		api.GET("/explore/regions", middleware.AuthRequired(), handlers.GetRegions)
		api.GET("/explore/regions/:id/details", middleware.AuthRequired(), handlers.GetRegionDetails)
		api.GET("/explore/maps", middleware.AuthRequired(), handlers.GetWorldMaps)
		api.GET("/explore/world", middleware.AuthRequired(), handlers.GetWorld)
		api.GET("/explore/objects", middleware.AuthRequired(), middleware.RateLimitByPlayer(30, time.Minute), handlers.GetExploreObjects)
		api.GET("/explore/cells/rect", middleware.AuthRequired(), middleware.RateLimitByPlayer(240, time.Minute), handlers.GetCellsRect)
		api.GET("/explore/players", middleware.AuthRequired(), middleware.RateLimitByPlayer(120, time.Minute), handlers.GetNearbyPlayers)
		api.GET("/world/monsters", middleware.AuthRequired(), middleware.RateLimitByPlayer(120, time.Minute), handlers.GetWorldMonsters)
		api.POST("/explore/move", middleware.AuthRequired(), middleware.RequireOnboarded(), middleware.RateLimitByPlayer(60, time.Minute), handlers.MovePlayer)
		api.POST("/explore/move-path", middleware.AuthRequired(), middleware.RequireOnboarded(), middleware.RateLimitByPlayer(120, time.Minute), handlers.MovePlayerPath)
		api.POST("/explore/teleport", middleware.AuthRequired(), middleware.RequireOnboarded(), middleware.RateLimitByPlayer(60, time.Minute), handlers.TeleportToRegion)
		api.POST("/explore/use_gu", middleware.AuthRequired(), middleware.RequireOnboarded(), middleware.RateLimitByPlayer(30, time.Minute), handlers.UseGuOnMap)

		// Ground drops / Chiến lợi phẩm trên bản đồ
		api.GET("/world/drops", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.GetTileGroundDrops)
		api.POST("/world/drops/:id/loot", middleware.AuthRequired(), middleware.RequireOnboarded(), middleware.RateLimitByPlayer(60, time.Minute), handlers.LootGroundDrop)

		// Teleportation Arrays / Truyền Tống Trận
		teleport := api.Group("/teleport")
		teleport.Use(middleware.AuthRequired())
		teleport.Use(middleware.RequireOnboarded())
		teleport.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			teleport.GET("/templates", handlers.GetTeleportTemplates)
			teleport.GET("/mine", handlers.GetMyTeleportArrays)
			teleport.POST("/build", handlers.BuildTeleportArray)
			teleport.GET("/directory", handlers.GetTeleportDirectory)
			teleport.POST("/travel", handlers.TravelTeleport)
			teleport.POST("/manage", handlers.ManageTeleportArray)
			teleport.POST("/repair", handlers.RepairTeleportArray)
			teleport.POST("/withdraw", handlers.WithdrawTeleportVault)
			teleport.POST("/dismantle", handlers.DismantleTeleportArray)
		}

		// Shop — group: 30/min, per player: 10/min
		api.GET("/shop/packages", middleware.AuthRequired(), handlers.GetPremiumShopPackages)
		api.GET("/shop/list", handlers.GetShopList)
		shop := api.Group("/shop")
		shop.Use(middleware.AuthRequired())
		shop.Use(middleware.RequireOnboarded())
		shop.Use(middleware.RateLimitByPlayer(15, time.Minute))
		{
			shop.POST("/buy-package", handlers.BuyPremiumPackage)
			shop.POST("/exchange", handlers.ExchangeStones)
			shop.POST("/buy", handlers.BuyShopItem)
			shop.POST("/sell", handlers.SellShopItem)
			shop.POST("/sell-bulk", handlers.SellShopItemsBulk)
			shop.GET("/bao-hoang/listings", handlers.ListListings)
			shop.POST("/bao-hoang/listings", handlers.CreateListing)
			shop.POST("/bao-hoang/buy", handlers.BuyListing)
			shop.POST("/bao-hoang/bid", handlers.PlaceBid)
			shop.POST("/bao-hoang/cancel", handlers.CancelListing)
			shop.POST("/bao-hoang/settle", handlers.SettleExpiredAuctions)

			// Buyback Order Book
			shop.GET("/bao-hoang/buyback", handlers.GetBuybackOrders)
			shop.POST("/bao-hoang/buyback", handlers.CreateBuybackOrder)
			shop.POST("/bao-hoang/buyback/sell", handlers.SellToBuybackOrder)
			shop.POST("/bao-hoang/buyback/cancel", handlers.CancelBuybackOrder)

			// Async Trade Offers
			shop.GET("/bao-hoang/trade-offers", handlers.GetTradeOffers)
			shop.POST("/bao-hoang/trade-offers", handlers.CreateTradeOffer)
			shop.POST("/bao-hoang/trade-offers/accept", handlers.AcceptTradeOffer)
			shop.POST("/bao-hoang/trade-offers/cancel", handlers.CancelTradeOffer)
		}

		// Merchants — dynamic supply/demand buy + fixed stock sell
		api.GET("/merchant", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.GetMerchants)
		merchant := api.Group("/merchant")
		merchant.Use(middleware.AuthRequired())
		merchant.Use(middleware.RequireOnboarded())
		merchant.Use(middleware.RateLimitByPlayer(30, time.Minute))
		{
			merchant.GET("/map", handlers.GetMerchantMap)
			merchant.POST("/quote", handlers.QuoteMerchant)
			merchant.POST("/sell", handlers.SellToMerchant)
			merchant.POST("/buy", handlers.BuyFromMerchant)
		}

		// Trade — group: 25/min, per player: 8/min
		trade := api.Group("/trade")
		trade.Use(middleware.AuthRequired())
		trade.Use(middleware.RequireOnboarded())
		trade.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			trade.POST("/initiate", handlers.InitiateTrade)
			trade.GET("/active", handlers.GetActiveTrade)
			trade.POST("/accept/:id", handlers.AcceptTrade)
			trade.POST("/decline/:id", handlers.DeclineTrade)
			trade.POST("/cancel/:id", handlers.CancelTrade)
			trade.POST("/offer/:id", handlers.UpdateTradeOffer)
			trade.POST("/lock/:id", handlers.LockTrade)
			trade.POST("/unlock/:id", handlers.UnlockTrade)
			trade.POST("/confirm/:id", handlers.ConfirmTrade)
		}

		// Soul — group: 20/min, per player: 15/min
		soul := api.Group("/soul")
		soul.Use(middleware.AuthRequired())
		soul.Use(middleware.RequireOnboarded())
		soul.Use(middleware.RateLimitByPlayer(15, time.Minute))
		{
			soul.POST("/refine", handlers.RefineSoul)
		}

		// Instance — group: 30/min, per player: 20/min
		instance := api.Group("/instance")
		instance.Use(middleware.AuthRequired())
		instance.Use(middleware.RequireOnboarded())
		instance.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			instance.GET("/active", handlers.GetActiveInstance)
			instance.GET("/list", handlers.ListInstances)
			instance.POST("/enter", handlers.EnterInstance)
			instance.POST("/action", handlers.ActionInstance)
			instance.GET("/spatial", handlers.ListSpatialRealms)
			instance.GET("/spatial/active", handlers.GetActiveSpatialRealm)
			instance.POST("/spatial/enter", handlers.EnterSpatialRealm)
			instance.POST("/spatial/exit", handlers.ExitSpatialRealm)
		}

		// Sect — group: 40/min, per player: 25/min
		sect := api.Group("/sects")
		sect.Use(middleware.AuthRequired())
		sect.Use(middleware.RequireOnboarded())
		sect.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			sect.GET("", handlers.GetSects)
			sect.POST("/:id/join", handlers.JoinSect)
			sect.POST("/join-request/cancel", handlers.CancelSectJoinRequest)
			sect.POST("/leave", handlers.LeaveSect)
			sect.GET("/me", handlers.GetMySectStatus)
			sect.POST("/promote", handlers.PromoteSectRank)
			sect.POST("/donate/stones", handlers.DonateStones)
			sect.POST("/donate/materials", handlers.DonateMaterials)
			sect.POST("/donate/gu", handlers.DonateGu)
			sect.POST("/donate/recipe", handlers.DonateRecipe)
			sect.POST("/donate/bulk", handlers.DonateBulk)
			sect.POST("/vault/buy", handlers.BuyFromSectVault)
			sect.GET("/missions", handlers.GetSectMissions)
			sect.POST("/missions/:missionId/accept", handlers.AcceptSectMission)
			sect.POST("/missions/:missionId/complete", handlers.CompleteSectMission)
			sect.DELETE("/missions/:missionId", handlers.DeleteSectMission)
			sect.DELETE("/missions/:missionId/abandon", handlers.AbandonSectMission)
			sect.POST("/create", handlers.CreateSect)
			sect.GET("/me/members", handlers.GetSectMembers)
			sect.POST("/members/:memberId/rank", handlers.SetMemberRank)
			sect.POST("/members/:memberId/kick", handlers.KickMember)
			sect.POST("/missions/create", handlers.CreateCustomMission)
			sect.POST("/vault/stock", handlers.StockSectVault)
			sect.POST("/vault/stock/bulk", handlers.StockSectVaultBulk)
			sect.POST("/vault/destroy", handlers.DestroySectVaultItem)
			sect.POST("/upgrade", handlers.UpgradeSect)
			sect.POST("/toggle-approval", handlers.ToggleSectApproval)
			sect.POST("/toggle-promotion-approval", handlers.ToggleSectPromotionApproval)
			sect.POST("/pay-salary", handlers.PaySectSalary)
			sect.GET("/join-requests", handlers.GetJoinRequests)
			sect.POST("/join-requests/:requestId/action", handlers.ActionJoinRequest)
			sect.GET("/promotion-requests", handlers.GetPromotionRequests)
			sect.POST("/promotion-requests/:requestId/action", handlers.ActionPromotionRequest)
			// Sect Warfare & Territories
			sect.GET("/map", handlers.GetSectsMap)
			sect.POST("/defend", handlers.DefendSect)
			sect.POST("/undefend", handlers.UndefendSect)
			sect.POST("/toggle-defend", handlers.ToggleDefendSect)
			sect.POST("/attack", handlers.AttackSect)
			sect.POST("/building/upgrade", handlers.UpgradeSectBuilding)
			sect.GET("/relocate/estimate", handlers.GetSectRelocationEstimate)
			sect.POST("/relocate", handlers.RelocateSect)
			sect.POST("/claim-leadership", handlers.ClaimSectLeadership)
		}

		// Feedback — group: 30/min, per player: 20/min
		api.GET("/feedback", middleware.RateLimitByIP(40, time.Minute), handlers.GetFeedbacks)
		api.GET("/feedback/:id/comments", middleware.RateLimitByIP(40, time.Minute), handlers.GetFeedbackComments)
		feedback := api.Group("/feedback")
		feedback.Use(middleware.AuthRequired())
		feedback.Use(middleware.RequireOnboarded())
		feedback.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			feedback.POST("", handlers.SubmitFeedback)
			feedback.POST("/:id/vote", handlers.VoteFeedback)
			feedback.PUT("/:id/status", handlers.UpdateFeedbackStatus)
			feedback.POST("/:id/comments", handlers.AddFeedbackComment)
			feedback.DELETE("/:id/comments/:commentId", handlers.DeleteFeedbackComment)
		}
	}

	return r
}

func worldEventEnvelope(channel string, payload []byte) []byte {
	event := struct {
		Type    string          `json:"type"`
		Channel string          `json:"channel"`
		Data    json.RawMessage `json:"data"`
	}{Type: "event", Channel: channel, Data: payload}
	wrapped, err := json.Marshal(event)
	if err != nil {
		return payload
	}
	return wrapped
}
