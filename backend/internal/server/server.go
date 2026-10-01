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
	method     string
	path       string
	permission string
	handler    gin.HandlerFunc
}

// adminRouteSpecs is the permission inventory for every /api/admin endpoint.
// Keeping registration and authorization metadata together makes missing permissions impossible.
func adminRouteSpecs() []adminRouteSpec {
	return []adminRouteSpec{
		{http.MethodGet, "/me", authz.AdminAccessPermission, handlers.AdminGetMe},
		{http.MethodGet, "/resources", authz.AdminAccessPermission, handlers.AdminGetResources},
		// monitoring
		{http.MethodGet, "/monitor/system", "monitoring.read", handlers.AdminGetSystemStats},
		{http.MethodGet, "/monitor/requests/summary", "monitoring.read", handlers.AdminGetRequestSummary},
		{http.MethodGet, "/monitor/requests/traces", "monitoring.read", handlers.AdminGetRequestTraces},
		{http.MethodGet, "/monitor/requests/traces/:id", "monitoring.read", handlers.AdminGetRequestTraceDetail},
		{http.MethodPost, "/monitor/requests/reset", "monitoring.reset", handlers.AdminResetRequestMetrics},
		// billing
		{http.MethodGet, "/topup-orders", "billing.read", handlers.AdminListTopupOrders},
		{http.MethodGet, "/wallet-transactions", "billing.read", handlers.AdminListWalletTransactions},
		// players
		{http.MethodGet, "/players/online", "players.read", handlers.AdminListOnlinePlayers},
		{http.MethodGet, "/players", "players.read", handlers.AdminListPlayers},
		{http.MethodGet, "/players/:id", "players.read", handlers.AdminGetPlayerDetail},
		{http.MethodGet, "/players/:id/items", "players.read", handlers.AdminListPlayerItems},
		{http.MethodGet, "/players/:id/gus", "players.read", handlers.AdminListPlayerGus},
		{http.MethodPut, "/players/:id", "players.update", handlers.AdminUpdatePlayer},
		// apertures
		{http.MethodGet, "/apertures", "apertures.read", handlers.AdminListApertures},
		{http.MethodGet, "/apertures/:id", "apertures.read", handlers.AdminGetApertureDetail},
		{http.MethodPut, "/apertures/:id", "apertures.update", handlers.AdminUpdateAperture},
		// gus
		{http.MethodGet, "/gus/search", "gus.read", handlers.AdminSearchGus},
		{http.MethodGet, "/gu-history", "gus.read", handlers.AdminListGuHistory},
		// players
		{http.MethodPost, "/players/:id/items", "players.inventory.grant", handlers.AdminAddPlayerItems},
		{http.MethodDelete, "/players/:id/items", "players.update", handlers.AdminRemovePlayerItems},
		{http.MethodPost, "/players/:id/gus", "players.inventory.grant", handlers.AdminAddPlayerGus},
		{http.MethodDelete, "/players/:id/gus/:instance_id", "players.update", handlers.AdminRemovePlayerGu},
		{http.MethodGet, "/players/:id/pets", "players.read", handlers.AdminListPlayerPets},
		{http.MethodDelete, "/players/:id/pets/:pet_id", "players.update", handlers.AdminDeletePlayerPet},
		{http.MethodPut, "/players/:id/stones", "players.update", handlers.AdminChangePlayerStones},
		// sects
		{http.MethodGet, "/sects", "sects.read", handlers.AdminListSects},
		{http.MethodGet, "/sects/:id", "sects.read", handlers.AdminGetSectDetail},
		{http.MethodPut, "/sects/:id", "sects.update", handlers.AdminUpdateSect},
		{http.MethodPost, "/sects/:id/members/:memberId/leader", "sects.update", handlers.AdminSetSectLeader},
		{http.MethodDelete, "/sects/:id/members/:memberId", "sects.update", handlers.AdminKickSectMember},
		{http.MethodPost, "/sects/:id/members", "sects.update", handlers.AdminAddToSect},
		{http.MethodDelete, "/sects/:id", "sects.update", handlers.AdminDeleteSect},
		{http.MethodGet, "/sects/:id/missions", "sects.read", handlers.AdminListSectMissions},
		{http.MethodPost, "/sects/:id/missions", "sects.update", handlers.AdminCreateSectMission},
		{http.MethodDelete, "/sects/:id/missions/:missionId", "sects.update", handlers.AdminDeleteSectMission},
		{http.MethodPost, "/sects/:id/vault/stock", "sects.update", handlers.AdminStockSectVault},
		{http.MethodDelete, "/sects/:id/vault/stock", "sects.update", handlers.AdminRemoveSectVaultItem},
		// game_data
		{http.MethodPost, "/data/import", "game_data.import_all", handlers.AdminImportData},
		{http.MethodPost, "/data/import/commit", "game_data.import_all", handlers.AdminCommitImportData},
		{http.MethodPost, "/data/export", "game_data.export", handlers.AdminExportData},
		{http.MethodGet, "/data/tables", "game_data.read", handlers.AdminListTables},
		{http.MethodPost, "/data/import/:table", "game_data.import", handlers.AdminImportTable},
		{http.MethodPost, "/data/import/:table/commit", "game_data.import", handlers.AdminCommitImportTable},
		{http.MethodPost, "/data/export/:table", "game_data.export", handlers.AdminExportTable},
		// cache
		{http.MethodPost, "/clear-cache", "cache.reload", handlers.AdminClearCache},
		{http.MethodPost, "/clear-redis-cache", "cache.flush", handlers.AdminClearRedisCache},
		// audit
		{http.MethodGet, "/audit-logs", "audit.read", handlers.AdminGetAuditLogs},
		// regions
		{http.MethodGet, "/regions", "regions.read", handlers.AdminListRegions},
		{http.MethodPost, "/regions", "regions.update", handlers.AdminCreateRegion},
		{http.MethodPut, "/regions/:id", "regions.update", handlers.AdminUpdateRegion},
		{http.MethodDelete, "/regions/:id", "regions.update", handlers.AdminDeleteRegion},
		// map_cells
		{http.MethodGet, "/map-cells", "map_cells.read", handlers.AdminListMapCells},
		{http.MethodPost, "/map-cells", "map_cells.update", handlers.AdminCreateMapCell},
		{http.MethodPut, "/map-cells/:id", "map_cells.update", handlers.AdminUpdateMapCell},
		{http.MethodDelete, "/map-cells/:id", "map_cells.update", handlers.AdminDeleteMapCell},
		// shop
		{http.MethodGet, "/shop", "shop.read", handlers.AdminListShopItems},
		{http.MethodPost, "/shop", "shop.update", handlers.AdminCreateShopItem},
		{http.MethodPut, "/shop/:id/:rank", "shop.update", handlers.AdminUpdateShopItem},
		{http.MethodDelete, "/shop/:id/:rank", "shop.update", handlers.AdminDeleteShopItem},
		// arena_shop
		{http.MethodGet, "/arena-shop", "arena_shop.read", handlers.AdminListArenaShopItems},
		{http.MethodPost, "/arena-shop", "arena_shop.update", handlers.AdminCreateArenaShopItem},
		{http.MethodPut, "/arena-shop/:id", "arena_shop.update", handlers.AdminUpdateArenaShopItem},
		{http.MethodDelete, "/arena-shop/:id", "arena_shop.update", handlers.AdminDeleteArenaShopItem},
		// mine_slots
		{http.MethodGet, "/mine-slots", "mine_slots.read", handlers.AdminListMineSlots},
		{http.MethodPut, "/mine-slots/:id", "mine_slots.update", handlers.AdminUpdateMineSlot},
		{http.MethodPost, "/mine-slots/:id/reset", "mine_slots.update", handlers.AdminResetMineSlot},
		// gu_dictionary
		{http.MethodGet, "/gu-dictionary", "gu_dictionary.read", handlers.AdminListGuItems},
		{http.MethodPost, "/gu-effects/validate", "gu_dictionary.update", handlers.AdminValidateGuEffects},
		{http.MethodGet, "/gu-effects/audit", "gu_dictionary.read", handlers.AdminAuditGuEffects},
		{http.MethodPost, "/effect-catalog/validate", "gu_dictionary.update", handlers.AdminValidateGuEffects},
		{http.MethodGet, "/effect-catalog/metadata", "gu_dictionary.read", handlers.AdminEffectCatalogMetadata},
		{http.MethodGet, "/effect-catalog/audit", "gu_dictionary.read", handlers.AdminAuditEffectCatalog},
		{http.MethodPost, "/gu-dictionary", "gu_dictionary.update", handlers.AdminCreateGuItem},
		{http.MethodPut, "/gu-dictionary/:id", "gu_dictionary.update", handlers.AdminUpdateGuItem},
		{http.MethodDelete, "/gu-dictionary/:id", "gu_dictionary.update", handlers.AdminDeleteGuItem},
		// item_dictionary
		{http.MethodGet, "/item-dictionary", "item_dictionary.read", handlers.AdminListItemItems},
		{http.MethodPost, "/item-dictionary", "item_dictionary.update", handlers.AdminCreateItemItem},
		{http.MethodPut, "/item-dictionary/:id", "item_dictionary.update", handlers.AdminUpdateItemItem},
		{http.MethodDelete, "/item-dictionary/:id", "item_dictionary.update", handlers.AdminDeleteItemItem},
		// recipes
		{http.MethodGet, "/recipes", "recipes.read", handlers.AdminListRecipes},
		{http.MethodPost, "/recipes", "recipes.update", handlers.AdminCreateRecipe},
		{http.MethodPut, "/recipes/:id", "recipes.update", handlers.AdminUpdateRecipe},
		{http.MethodDelete, "/recipes/:id", "recipes.update", handlers.AdminDeleteRecipe},
		// monsters
		{http.MethodGet, "/monsters", "monsters.read", handlers.AdminListMonsters},
		{http.MethodPost, "/monsters", "monsters.update", handlers.AdminCreateMonster},
		{http.MethodPut, "/monsters/:id", "monsters.update", handlers.AdminUpdateMonster},
		{http.MethodDelete, "/monsters/:id", "monsters.update", handlers.AdminDeleteMonster},
		// loot_tables
		{http.MethodGet, "/loot-tables", "loot_tables.read", handlers.AdminListLootTables},
		{http.MethodPost, "/loot-tables", "loot_tables.update", handlers.AdminCreateLootTable},
		{http.MethodPut, "/loot-tables/:id", "loot_tables.update", handlers.AdminUpdateLootTable},
		{http.MethodDelete, "/loot-tables/:id", "loot_tables.update", handlers.AdminDeleteLootTable},
		// gift_codes
		{http.MethodPost, "/gift-codes", "gift_codes.update", handlers.AdminCreateGiftCode},
		{http.MethodGet, "/gift-codes", "gift_codes.read", handlers.AdminListGiftCodes},
		// images
		{http.MethodPost, "/upload-image", "images.update", handlers.AdminUploadImage},
		// players
		{http.MethodPut, "/players/:id/gus/:instance_id", "players.update", handlers.AdminUpdatePlayerGu},
		{http.MethodPost, "/players/:id/pets", "players.update", handlers.AdminCreatePlayerPet},
		{http.MethodPut, "/players/:id/pets/:pet_id", "players.update", handlers.AdminUpdatePlayerPet},
		// constitution_dictionary
		{http.MethodGet, "/constitution-dictionary", "constitution_dictionary.read", handlers.AdminListConstitutions},
		{http.MethodPost, "/constitution-dictionary", "constitution_dictionary.update", handlers.AdminCreateConstitution},
		{http.MethodPut, "/constitution-dictionary/:key", "constitution_dictionary.update", handlers.AdminUpdateConstitution},
		{http.MethodDelete, "/constitution-dictionary/:key", "constitution_dictionary.update", handlers.AdminDeleteConstitution},
		// race_dictionary
		{http.MethodGet, "/race-dictionary", "race_dictionary.read", handlers.AdminListRaces},
		{http.MethodPost, "/race-dictionary", "race_dictionary.update", handlers.AdminCreateRace},
		{http.MethodPut, "/race-dictionary/:key", "race_dictionary.update", handlers.AdminUpdateRace},
		{http.MethodDelete, "/race-dictionary/:key", "race_dictionary.update", handlers.AdminDeleteRace},
		// monitoring
		{http.MethodGet, "/batch-metrics", "monitoring.read", handlers.GetBatchMetrics},
		// status_dictionary
		{http.MethodGet, "/status-dictionary", "status_dictionary.read", handlers.AdminListStatuses},
		{http.MethodPost, "/status-dictionary", "status_dictionary.update", handlers.AdminCreateStatus},
		{http.MethodPut, "/status-dictionary/:key", "status_dictionary.update", handlers.AdminUpdateStatus},
		{http.MethodDelete, "/status-dictionary/:key", "status_dictionary.update", handlers.AdminDeleteStatus},
		// monster_spawns
		{http.MethodGet, "/monster-spawns", "monster_spawns.read", handlers.AdminListMonsterSpawns},
		{http.MethodPost, "/monster-spawns", "monster_spawns.update", handlers.AdminCreateMonsterSpawn},
		{http.MethodPut, "/monster-spawns/:id", "monster_spawns.update", handlers.AdminUpdateMonsterSpawn},
		{http.MethodDelete, "/monster-spawns/:id", "monster_spawns.update", handlers.AdminDeleteMonsterSpawn},
		// quests
		{http.MethodGet, "/quests", "quests.read", handlers.AdminListQuests},
		{http.MethodPost, "/quests", "quests.update", handlers.AdminCreateQuest},
		{http.MethodPut, "/quests/:id", "quests.update", handlers.AdminUpdateQuest},
		{http.MethodDelete, "/quests/:id", "quests.update", handlers.AdminDeleteQuest},
		// dungeons
		{http.MethodGet, "/dungeons", "dungeons.read", handlers.AdminListDungeons},
		{http.MethodPost, "/dungeons", "dungeons.update", handlers.AdminCreateDungeon},
		{http.MethodPut, "/dungeons/:id", "dungeons.update", handlers.AdminUpdateDungeon},
		{http.MethodDelete, "/dungeons/:id", "dungeons.update", handlers.AdminDeleteDungeon},
		// sect_mines
		{http.MethodGet, "/sect-mines", "sect_mines.read", handlers.AdminListSectMines},
		{http.MethodPut, "/sect-mines/:id", "sect_mines.update", handlers.AdminUpdateSectMine},
		// apertures
		{http.MethodGet, "/apertures/:id/nodes", "apertures.read", handlers.AdminListApertureNodes},
		{http.MethodPost, "/apertures/:id/nodes", "apertures.update", handlers.AdminCreateApertureNode},
		{http.MethodDelete, "/apertures/:id/nodes/:node_id", "apertures.update", handlers.AdminDeleteApertureNode},
		{http.MethodGet, "/apertures/:id/variant-humans", "apertures.read", handlers.AdminListApertureVariantHumans},
		{http.MethodPost, "/apertures/:id/variant-humans", "apertures.update", handlers.AdminCreateApertureVariantHuman},
		{http.MethodDelete, "/apertures/:id/variant-humans/:vh_id", "apertures.update", handlers.AdminDeleteApertureVariantHuman},
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
		errMsgRaw, _ := c.Get("error_msg")
		var responseBody, errMsg string
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
					respBodyField, zap.String("error", errMsg),
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
				fields = append(fields, respBodyField, zap.String("error", errMsg))
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
			shub.BroadcastToTargets(ws.ChannelName(channel), payload, targets)
		}
		handlers.WorldCellBroadcaster = func(channel string, payload []byte, targets map[uuid.UUID]struct{}) {
			shub.BroadcastToTargets(ws.ChannelName(channel), payload, targets)
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
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

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
			handlers := []gin.HandlerFunc{middleware.PermissionRequired(route.permission)}
			if (route.method == http.MethodPost && strings.HasSuffix(route.path, "/commit")) || route.permission == "cache.flush" {
				handlers = append(handlers, middleware.AdminReasonRequired())
			}
			handlers = append(handlers, route.handler)
			admin.Handle(route.method, route.path, handlers...)
		}

		api.GET("/wallet", middleware.AuthRequired(), handlers.GetWallet)
		api.POST("/topup/orders", middleware.AuthRequired(), middleware.RateLimitByPlayer(10, time.Minute), handlers.CreateTopupOrder)
		api.GET("/topup/orders/:orderCode", middleware.AuthRequired(), handlers.GetTopupOrder)

		// Player endpoints — group: 60/min, per player: 30/min
		player := api.Group("/player")
		player.Use(middleware.AuthRequired())
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
			player.POST("/sacrifice", handlers.SacrificeNPC)
			player.GET("/profile/:username", handlers.GetPlayerProfile)
			player.PUT("/profile/settings", handlers.UpdateProfileSettings)
			player.GET("/referrals", handlers.GetReferralStats)
			player.POST("/referrals/claim", handlers.ClaimReferralReward)
			player.POST("/referrals/bind", handlers.BindReferralCode)
			player.POST("/simulate-deposit", middleware.RequirePermission(authz.AdminAccessPermission), handlers.SimulateDeposit)
			player.POST("/avatar", handlers.UploadAvatar)
			player.POST("/redeem-gift-code", handlers.RedeemGiftCode)
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
		quest.Use(middleware.RateLimitByPlayer(25, time.Minute))
		{
			quest.GET("", handlers.ListQuests)
			quest.POST("/complete", handlers.CompleteQuest)
		}

		// Gacha — group: 50/min, per player: 15/min
		gacha := api.Group("/gacha")
		gacha.Use(middleware.AuthRequired())
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
		achievement.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			achievement.GET("", handlers.GetAchievements)
			achievement.POST("/claim", handlers.ClaimAchievement)
			achievement.POST("/select", handlers.SelectTitleFrame)
		}

		// Party — group: 30/min, per player: 20/min
		party := api.Group("/party")
		party.Use(middleware.AuthRequired())
		party.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			party.POST("/create", handlers.CreateParty)
			party.GET("/status", handlers.GetPartyStatus)
			party.POST("/invite", handlers.InvitePlayer)
			party.POST("/leave", handlers.LeaveParty)
			party.GET("/invitations", handlers.GetPendingInvitations)
			party.POST("/invitations/:id/respond", handlers.RespondToInvitation)
		}

		// Inventory — group: 30/min, per player: 20/min
		inventory := api.Group("/inventory")
		inventory.Use(middleware.AuthRequired())
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
		refining.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			refining.POST("", handlers.RefineGu)
			refining.POST("/deduce", handlers.DeduceRecipe)
		}

		// Scouting & Divination — group: 60/min, per player: 30/min
		scouting := api.Group("/scouting")
		scouting.Use(middleware.AuthRequired())
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
		pets.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			pets.GET("", handlers.ListPlayerPets)
			pets.POST("/feed", handlers.FeedPet)
			pets.POST("/formation", handlers.SetPetFormation)
			pets.POST("/sleep", handlers.SetPetSleep)
			pets.POST("/release", handlers.ReleasePet)
			pets.POST("/merge", handlers.MergePets)
		}

		// Killer Moves — group: 30/min, per player: 20/min
		killerMoves := api.Group("/killer-moves")
		killerMoves.Use(middleware.AuthRequired())
		killerMoves.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			killerMoves.POST("/calculate", handlers.CalculateKillerMove)
			killerMoves.POST("", handlers.CreateKillerMove)
			killerMoves.GET("", handlers.ListKillerMoves)
			killerMoves.GET("/:id", handlers.GetKillerMove)
			killerMoves.DELETE("/:id", handlers.DeleteKillerMove)
		}

		// Combat — group: 60/min, per player: 60/min
		combat := api.Group("/combat")
		combat.Use(middleware.AuthRequired())
		combat.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			combat.POST("/start", handlers.AutoCombatStart)
			combat.POST("/practice", handlers.PracticeCombatStart)
			combat.POST("/cast-killer-move", handlers.CastKillerMove)
			combat.POST("/capture", handlers.CaptureMonster)
			combat.GET("/history", handlers.ListCombatHistory)
			combat.GET("/history/:id", handlers.GetCombatHistory)
		}

		// Immortal Aperture — group: 30/min, per player: 20/min
		aperture := api.Group("/immortal-aperture")
		aperture.Use(middleware.AuthRequired())
		aperture.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			aperture.GET("", handlers.GetImmortalAperture)
			aperture.GET("/nodes", handlers.GetApertureNodes)
			aperture.POST("/nodes", handlers.PlaceApertureNode)
			aperture.POST("/nodes/:id/collect", handlers.CollectApertureResource)
			aperture.POST("/nodes/collect-all", handlers.CollectAllApertureResources)
			aperture.POST("/nodes/repair-all", handlers.RepairAllApertureNodes)
			aperture.POST("/nodes/:id/uninstall", handlers.UninstallApertureNode)
			aperture.DELETE("/nodes/:id", handlers.DeleteApertureNode)
			aperture.POST("/time-rate", handlers.SetApertureTimeRate)
			aperture.POST("/labor", handlers.SetApertureLabor)
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
			aperture.DELETE("/npcs/:npc_id", handlers.DismissNPC)
		}

		// Dao Marks — group: 20/min, per player: 15/min
		daoMarks := api.Group("/dao-marks")
		daoMarks.Use(middleware.AuthRequired())
		daoMarks.Use(middleware.RateLimitByPlayer(15, time.Minute))
		{
			daoMarks.GET("", handlers.GetDaoMarks)
			daoMarks.GET("/dao-ngan", handlers.GetDaoNganHandler)
		}

		// Tribulations — group: 30/min, per player: 20/min
		tribulations := api.Group("/tribulations")
		tribulations.Use(middleware.AuthRequired())
		tribulations.Use(middleware.RateLimitByPlayer(60, time.Minute))
		{
			tribulations.GET("/active", handlers.GetActiveTribulation)
			tribulations.POST("/:eventId/engage", handlers.EngageTribulation)
			tribulations.POST("/:eventId/resolve", handlers.ResolveTribulation)
		}

		// Arena — group: 40/min, per player: 25/min
		arena := api.Group("/arena")
		arena.Use(middleware.AuthRequired())
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

		// Leaderboard — group: 30/min, per player: 20/min
		leaderboard := api.Group("/leaderboard")
		leaderboard.Use(middleware.AuthRequired())
		leaderboard.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			leaderboard.GET("/:category", handlers.GetLeaderboard)
			leaderboard.POST("/toggle", handlers.ToggleLeaderboardVisibility)
			leaderboard.GET("/settings", handlers.GetLeaderboardSettings)
		}

		// Chat — no group limit, per-route IP limit
		chat := api.Group("/chat")
		chat.Use(middleware.AuthRequired())
		{
			chat.POST("/mail/send", handlers.SendPrivateMail)
			chat.GET("/mail/inbox", handlers.GetInbox)
			chat.POST("/bao-hoang/send", middleware.RateLimitByIP(30, time.Minute), handlers.SendBaoHoangChat)
			chat.GET("/bao-hoang/messages", handlers.GetBaoHoangMessages)
		}

		// Explore — per player: 25/min
		api.POST("/explore", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.ExploreNode)
		api.POST("/explore/retreat/:id", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.Retreat)
		api.POST("/explore/flee/:id", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.Flee)
		api.GET("/explore/grid", middleware.AuthRequired(), handlers.GetGrid)
		api.GET("/explore/regions", middleware.AuthRequired(), handlers.GetRegions)
		api.GET("/explore/world", middleware.AuthRequired(), handlers.GetWorld)
		api.GET("/explore/objects", middleware.AuthRequired(), middleware.RateLimitByPlayer(30, time.Minute), handlers.GetExploreObjects)
		api.GET("/explore/cells/rect", middleware.AuthRequired(), middleware.RateLimitByPlayer(240, time.Minute), handlers.GetCellsRect)
		api.GET("/explore/players", middleware.AuthRequired(), middleware.RateLimitByPlayer(120, time.Minute), handlers.GetNearbyPlayers)
		api.POST("/explore/move", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.MovePlayer)
		api.POST("/explore/move-path", middleware.AuthRequired(), middleware.RateLimitByPlayer(120, time.Minute), handlers.MovePlayerPath)
		api.POST("/explore/teleport", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.TeleportToRegion)
		api.POST("/explore/use_gu", middleware.AuthRequired(), middleware.RateLimitByPlayer(30, time.Minute), handlers.UseGuOnMap)
		api.POST("/explore/use_killer_move", middleware.AuthRequired(), middleware.RateLimitByPlayer(30, time.Minute), handlers.UseKillerMoveOnMap)

		// Ground drops / Chiến lợi phẩm trên bản đồ
		api.GET("/world/drops", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.GetTileGroundDrops)
		api.POST("/world/drops/:id/loot", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.LootGroundDrop)

		// Teleportation Arrays / Truyền Tống Trận
		teleport := api.Group("/teleport")
		teleport.Use(middleware.AuthRequired())
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

		// Trade — group: 25/min, per player: 8/min
		trade := api.Group("/trade")
		trade.Use(middleware.AuthRequired())
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
		soul.Use(middleware.RateLimitByPlayer(15, time.Minute))
		{
			soul.POST("/refine", handlers.RefineSoul)
		}

		// Instance — group: 30/min, per player: 20/min
		instance := api.Group("/instance")
		instance.Use(middleware.AuthRequired())
		instance.Use(middleware.RateLimitByPlayer(20, time.Minute))
		{
			instance.GET("/active", handlers.GetActiveInstance)
			instance.GET("/list", handlers.ListInstances)
			instance.POST("/enter", handlers.EnterInstance)
			instance.POST("/action", handlers.ActionInstance)
		}

		// Sect — group: 40/min, per player: 25/min
		sect := api.Group("/sects")
		sect.Use(middleware.AuthRequired())
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
