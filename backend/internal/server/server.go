package server

import (
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/aurowora/compress"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/monitor"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/internal/ws"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

const slowRequestThreshold = 5 * time.Second

// requestLogging logs the start and end of every request so that hung
// handlers are visible (Gin's own logger only prints after completion).
// It also flags requests that exceed slowRequestThreshold.
func requestLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		if method != http.MethodOptions {
			logger.S.Debugf("REQ START %s %s from %s", method, path, clientIP)
		}

		c.Next()

		duration := time.Since(start)
		if method == http.MethodOptions {
			return
		}

		status := c.Writer.Status()
		switch {
		case duration >= slowRequestThreshold:
			logger.S.Warnf("REQ SLOW %s %s status=%d duration=%s from %s", method, path, status, duration, clientIP)
		case duration >= 500*time.Millisecond:
			logger.S.Infof("REQ END %s %s status=%d duration=%s from %s", method, path, status, duration, clientIP)
		default:
			logger.S.Debugf("REQ END %s %s status=%d duration=%s from %s", method, path, status, duration, clientIP)
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
	worker.PlayerEventPublisher = handlers.PublishPlayerRealtimeEvent
	gin.DefaultWriter = logger.GinWriter()
	r := gin.Default()

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

	// Debug endpoints: goroutine/pprof dumps and DB pool stats
	registerDebugEndpoints(r)

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
		api.GET("/dictionary", middleware.RateLimitByIP(40, time.Minute), handlers.GetDictionary)
		api.GET("/dictionary/gus", middleware.RateLimitByIP(40, time.Minute), handlers.GetGuDictionary)
		api.GET("/dictionary/items", middleware.RateLimitByIP(40, time.Minute), handlers.GetItemDictionary)
		api.GET("/dictionary/monsters", middleware.RateLimitByIP(40, time.Minute), handlers.GetMonsterDictionary)
		api.GET("/dictionary/constitutions", middleware.RateLimitByIP(40, time.Minute), handlers.GetConstitutionDictionary)
		api.GET("/dictionary/races", middleware.RateLimitByIP(40, time.Minute), handlers.GetRaceDictionary)
		api.GET("/dictionary/loot-tables", middleware.RateLimitByIP(40, time.Minute), handlers.GetLootTables)
		api.GET("/dictionary/regions", middleware.RateLimitByIP(40, time.Minute), handlers.GetDictionaryRegions)
		api.GET("/dictionary/monster-spawns", middleware.RateLimitByIP(40, time.Minute), handlers.GetMonsterSpawns)
		api.GET("/recipes", middleware.RateLimitByIP(40, time.Minute), handlers.GetRecipes)

		api.GET("/debug/monsters", func(c *gin.Context) {
			var totalCount, activeCount int
			err1 := db.DB.QueryRow("SELECT COUNT(*) FROM monster_templates").Scan(&totalCount)
			err2 := db.DB.QueryRow("SELECT COUNT(*) FROM monster_templates WHERE is_active = true OR is_active IS NULL").Scan(&activeCount)

			// Test the exact query that LoadMonsterRegistry runs
			rows, err3 := db.DB.Query(`
				SELECT id, name, type, COALESCE(max_hp, 100), COALESCE(base_attack, 10), COALESCE(base_defense, 10), COALESCE(gu_inventory, '[]'::jsonb), COALESCE(gu_pool, '[]'::jsonb), COALESCE(item_inventory, '{}'::jsonb), COALESCE(dao_marks, '{}'::jsonb), COALESCE(path_masteries, '{}'::jsonb), COALESCE(ai_profile, 'beast_basic'), COALESCE(loot_table_id, ''), COALESCE(description, ''),
					   COALESCE(is_capturable, false),
					   COALESCE(capture_chance, 0.20),
					   COALESCE(pet_hunger_per_hour, 2.0)
				FROM monster_templates
				WHERE is_active = true OR is_active IS NULL
			`)

			var queryErr string
			var scanErrs []string
			var parsedCount int
			if err3 != nil {
				queryErr = err3.Error()
			} else {
				defer rows.Close()
				for rows.Next() {
					var mon models.MonsterTemplate
					var guInvRaw, guPoolRaw, itemInvRaw, daoMarksRaw, masteriesRaw []byte
					err := rows.Scan(
						&mon.ID, &mon.Name, &mon.Type,
						&mon.MaxHP, &mon.BaseAttack, &mon.BaseDefense,
						&guInvRaw, &guPoolRaw, &itemInvRaw, &daoMarksRaw, &masteriesRaw, &mon.AIProfile, &mon.LootTableID, &mon.Description,
						&mon.IsCapturable, &mon.CaptureChance, &mon.PetHungerPerHour,
					)
					if err != nil {
						scanErrs = append(scanErrs, err.Error())
						if len(scanErrs) > 10 {
							break
						}
					} else {
						parsedCount++
					}
				}
				if err := rows.Err(); err != nil {
					queryErr = "rows.Err: " + err.Error()
				}
			}

			c.JSON(200, gin.H{
				"total_count":  totalCount,
				"active_count": activeCount,
				"parsed_count": parsedCount,
				"query_err":    queryErr,
				"scan_errs":    scanErrs,
				"err1": func() string {
					if err1 != nil {
						return err1.Error()
					}
					return ""
				}(),
				"err2": func() string {
					if err2 != nil {
						return err2.Error()
					}
					return ""
				}(),
			})
		})

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

		// Admin endpoints — group: 80/min per IP
		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired())
		admin.Use(middleware.RateLimitByGroup("admin", 80, time.Minute))
		{
			// Application Monitoring & Tracing endpoints
			adminMonitor := admin.Group("/monitor")
			{
				adminMonitor.GET("/system", handlers.AdminGetSystemStats)
				adminMonitor.GET("/requests/summary", handlers.AdminGetRequestSummary)
				adminMonitor.GET("/requests/traces", handlers.AdminGetRequestTraces)
				adminMonitor.GET("/requests/traces/:id", handlers.AdminGetRequestTraceDetail)
				adminMonitor.POST("/requests/reset", handlers.AdminResetRequestMetrics)
			}

			admin.GET("/topup-orders", handlers.AdminListTopupOrders)
			admin.GET("/wallet-transactions", handlers.AdminListWalletTransactions)
			admin.GET("/players/online", handlers.AdminListOnlinePlayers)
			admin.GET("/players", handlers.AdminListPlayers)
			admin.GET("/players/:id", handlers.AdminGetPlayerDetail)
			admin.GET("/players/:id/items", handlers.AdminListPlayerItems)
			admin.GET("/players/:id/gus", handlers.AdminListPlayerGus)
			admin.PUT("/players/:id", handlers.AdminUpdatePlayer)
			admin.GET("/apertures", handlers.AdminListApertures)
			admin.GET("/apertures/:id", handlers.AdminGetApertureDetail)
			admin.PUT("/apertures/:id", handlers.AdminUpdateAperture)
			admin.GET("/gus/search", handlers.AdminSearchGus)
			admin.POST("/players/:id/items", handlers.AdminAddPlayerItems)
			admin.DELETE("/players/:id/items", handlers.AdminRemovePlayerItems)
			admin.POST("/players/:id/gus", handlers.AdminAddPlayerGus)
			admin.DELETE("/players/:id/gus/:instance_id", handlers.AdminRemovePlayerGu)
			admin.GET("/players/:id/pets", handlers.AdminListPlayerPets)
			admin.DELETE("/players/:id/pets/:pet_id", handlers.AdminDeletePlayerPet)
			admin.PUT("/players/:id/stones", handlers.AdminChangePlayerStones)
			admin.GET("/sects", handlers.AdminListSects)
			admin.GET("/sects/:id", handlers.AdminGetSectDetail)
			admin.PUT("/sects/:id", handlers.AdminUpdateSect)
			admin.POST("/sects/:id/members/:memberId/leader", handlers.AdminSetSectLeader)
			admin.DELETE("/sects/:id/members/:memberId", handlers.AdminKickSectMember)
			admin.POST("/sects/:id/members", handlers.AdminAddToSect)
			admin.DELETE("/sects/:id", handlers.AdminDeleteSect)
			admin.GET("/sects/:id/missions", handlers.AdminListSectMissions)
			admin.POST("/sects/:id/missions", handlers.AdminCreateSectMission)
			admin.DELETE("/sects/:id/missions/:missionId", handlers.AdminDeleteSectMission)
			admin.POST("/sects/:id/vault/stock", handlers.AdminStockSectVault)
			admin.DELETE("/sects/:id/vault/stock", handlers.AdminRemoveSectVaultItem)
			admin.POST("/data/import", handlers.AdminImportData)
			admin.POST("/data/export", handlers.AdminExportData)
			admin.GET("/data/tables", handlers.AdminListTables)
			admin.POST("/data/import/:table", handlers.AdminImportTable)
			admin.POST("/data/export/:table", handlers.AdminExportTable)
			admin.POST("/clear-cache", handlers.AdminClearCache)
			admin.POST("/clear-redis-cache", handlers.AdminClearRedisCache)
			admin.GET("/audit-logs", handlers.AdminGetAuditLogs)
			admin.GET("/regions", handlers.AdminListRegions)
			admin.POST("/regions", handlers.AdminCreateRegion)
			admin.PUT("/regions/:id", handlers.AdminUpdateRegion)
			admin.DELETE("/regions/:id", handlers.AdminDeleteRegion)
			admin.GET("/map-cells", handlers.AdminListMapCells)
			admin.POST("/map-cells", handlers.AdminCreateMapCell)
			admin.PUT("/map-cells/:id", handlers.AdminUpdateMapCell)
			admin.DELETE("/map-cells/:id", handlers.AdminDeleteMapCell)
			admin.GET("/shop", handlers.AdminListShopItems)
			admin.POST("/shop", handlers.AdminCreateShopItem)
			admin.PUT("/shop/:id/:rank", handlers.AdminUpdateShopItem)
			admin.DELETE("/shop/:id/:rank", handlers.AdminDeleteShopItem)
			admin.GET("/arena-shop", handlers.AdminListArenaShopItems)
			admin.POST("/arena-shop", handlers.AdminCreateArenaShopItem)
			admin.PUT("/arena-shop/:id", handlers.AdminUpdateArenaShopItem)
			admin.DELETE("/arena-shop/:id", handlers.AdminDeleteArenaShopItem)
			admin.GET("/mine-slots", handlers.AdminListMineSlots)
			admin.PUT("/mine-slots/:id", handlers.AdminUpdateMineSlot)
			admin.POST("/mine-slots/:id/reset", handlers.AdminResetMineSlot)
			admin.GET("/gu-dictionary", handlers.AdminListGuItems)
			admin.POST("/gu-dictionary", handlers.AdminCreateGuItem)
			admin.PUT("/gu-dictionary/:id", handlers.AdminUpdateGuItem)
			admin.DELETE("/gu-dictionary/:id", handlers.AdminDeleteGuItem)
			admin.GET("/item-dictionary", handlers.AdminListItemItems)
			admin.POST("/item-dictionary", handlers.AdminCreateItemItem)
			admin.PUT("/item-dictionary/:id", handlers.AdminUpdateItemItem)
			admin.DELETE("/item-dictionary/:id", handlers.AdminDeleteItemItem)
			admin.GET("/recipes", handlers.AdminListRecipes)
			admin.POST("/recipes", handlers.AdminCreateRecipe)
			admin.PUT("/recipes/:id", handlers.AdminUpdateRecipe)
			admin.DELETE("/recipes/:id", handlers.AdminDeleteRecipe)
			admin.GET("/monsters", handlers.AdminListMonsters)
			admin.POST("/monsters", handlers.AdminCreateMonster)
			admin.PUT("/monsters/:id", handlers.AdminUpdateMonster)
			admin.DELETE("/monsters/:id", handlers.AdminDeleteMonster)
			admin.GET("/loot-tables", handlers.AdminListLootTables)
			admin.POST("/loot-tables", handlers.AdminCreateLootTable)
			admin.PUT("/loot-tables/:id", handlers.AdminUpdateLootTable)
			admin.DELETE("/loot-tables/:id", handlers.AdminDeleteLootTable)
			admin.POST("/gift-codes", handlers.AdminCreateGiftCode)
			admin.GET("/gift-codes", handlers.AdminListGiftCodes)
			admin.POST("/upload-image", handlers.AdminUploadImage)

			// New Admin Endpoints for complete DB management
			admin.PUT("/players/:id/gus/:instance_id", handlers.AdminUpdatePlayerGu)
			admin.POST("/players/:id/pets", handlers.AdminCreatePlayerPet)
			admin.PUT("/players/:id/pets/:pet_id", handlers.AdminUpdatePlayerPet)

			admin.GET("/constitution-dictionary", handlers.AdminListConstitutions)
			admin.POST("/constitution-dictionary", handlers.AdminCreateConstitution)
			admin.PUT("/constitution-dictionary/:key", handlers.AdminUpdateConstitution)
			admin.DELETE("/constitution-dictionary/:key", handlers.AdminDeleteConstitution)

			admin.GET("/race-dictionary", handlers.AdminListRaces)
			admin.POST("/race-dictionary", handlers.AdminCreateRace)
			admin.PUT("/race-dictionary/:key", handlers.AdminUpdateRace)
			admin.DELETE("/race-dictionary/:key", handlers.AdminDeleteRace)

			admin.GET("/batch-metrics", handlers.GetBatchMetrics)

			admin.GET("/status-dictionary", handlers.AdminListStatuses)
			admin.POST("/status-dictionary", handlers.AdminCreateStatus)
			admin.PUT("/status-dictionary/:key", handlers.AdminUpdateStatus)
			admin.DELETE("/status-dictionary/:key", handlers.AdminDeleteStatus)

			admin.GET("/monster-spawns", handlers.AdminListMonsterSpawns)
			admin.POST("/monster-spawns", handlers.AdminCreateMonsterSpawn)
			admin.PUT("/monster-spawns/:id", handlers.AdminUpdateMonsterSpawn)
			admin.DELETE("/monster-spawns/:id", handlers.AdminDeleteMonsterSpawn)

			admin.GET("/quests", handlers.AdminListQuests)
			admin.POST("/quests", handlers.AdminCreateQuest)
			admin.PUT("/quests/:id", handlers.AdminUpdateQuest)
			admin.DELETE("/quests/:id", handlers.AdminDeleteQuest)

			admin.GET("/dungeons", handlers.AdminListDungeons)
			admin.POST("/dungeons", handlers.AdminCreateDungeon)
			admin.PUT("/dungeons/:id", handlers.AdminUpdateDungeon)
			admin.DELETE("/dungeons/:id", handlers.AdminDeleteDungeon)

			admin.GET("/sect-mines", handlers.AdminListSectMines)
			admin.PUT("/sect-mines/:id", handlers.AdminUpdateSectMine)

			admin.GET("/apertures/:id/nodes", handlers.AdminListApertureNodes)
			admin.POST("/apertures/:id/nodes", handlers.AdminCreateApertureNode)
			admin.DELETE("/apertures/:id/nodes/:node_id", handlers.AdminDeleteApertureNode)

			admin.GET("/apertures/:id/variant-humans", handlers.AdminListApertureVariantHumans)
			admin.POST("/apertures/:id/variant-humans", handlers.AdminCreateApertureVariantHuman)
			admin.DELETE("/apertures/:id/variant-humans/:vh_id", handlers.AdminDeleteApertureVariantHuman)
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
			player.POST("/simulate-deposit", handlers.SimulateDeposit)
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
		gacha.Use(middleware.RateLimitByPlayer(5, time.Minute))
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
			combat.POST("/attack", handlers.CombatAttack)
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
		api.GET("/explore/cells", middleware.AuthRequired(), middleware.RateLimitByPlayer(120, time.Minute), handlers.GetNearbyCells)
		api.GET("/explore/cells/rect", middleware.AuthRequired(), middleware.RateLimitByPlayer(240, time.Minute), handlers.GetCellsRect)
		api.GET("/explore/players", middleware.AuthRequired(), middleware.RateLimitByPlayer(120, time.Minute), handlers.GetNearbyPlayers)
		api.POST("/explore/move", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.MovePlayer)
		api.POST("/explore/move-path", middleware.AuthRequired(), middleware.RateLimitByPlayer(120, time.Minute), handlers.MovePlayerPath)
		api.POST("/explore/teleport", middleware.AuthRequired(), middleware.RateLimitByPlayer(60, time.Minute), handlers.TeleportToRegion)
		api.POST("/explore/use_gu", middleware.AuthRequired(), middleware.RateLimitByPlayer(30, time.Minute), handlers.UseGuOnMap)
		api.POST("/explore/use_killer_move", middleware.AuthRequired(), middleware.RateLimitByPlayer(30, time.Minute), handlers.UseKillerMoveOnMap)

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
			sect.GET("/mines", handlers.GetSectMines)
			sect.POST("/mines/defend", handlers.DefendSectMine)
			sect.POST("/mines/undefend", handlers.UndefendSectMine)
			sect.POST("/mines/attack", handlers.AttackSectMine)
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

// registerDebugEndpoints exposes diagnostics used to investigate hung requests:
// goroutine dumps (net/http/pprof) and live database connection pool stats.
func registerDebugEndpoints(r *gin.Engine) {
	debug := r.Group("/debug")
	{
		// Standard net/http/pprof endpoints for go tool pprof & browser inspection
		debug.GET("/pprof", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "/debug/pprof/")
		})
		debug.GET("/pprof/", gin.WrapF(pprof.Index))
		debug.GET("/pprof/cmdline", gin.WrapF(pprof.Cmdline))
		debug.GET("/pprof/profile", gin.WrapF(pprof.Profile))
		debug.GET("/pprof/symbol", gin.WrapF(pprof.Symbol))
		debug.POST("/pprof/symbol", gin.WrapF(pprof.Symbol))
		debug.GET("/pprof/trace", gin.WrapF(pprof.Trace))

		// Individual profile handlers
		debug.GET("/pprof/goroutine", gin.WrapH(pprof.Handler("goroutine")))
		debug.GET("/pprof/heap", gin.WrapH(pprof.Handler("heap")))
		debug.GET("/pprof/allocs", gin.WrapH(pprof.Handler("allocs")))
		debug.GET("/pprof/block", gin.WrapH(pprof.Handler("block")))
		debug.GET("/pprof/mutex", gin.WrapH(pprof.Handler("mutex")))
		debug.GET("/pprof/threadcreate", gin.WrapH(pprof.Handler("threadcreate")))
		debug.GET("/pprof/goroutine/stack", func(c *gin.Context) {
			buf := make([]byte, 8<<20)
			n := runtime.Stack(buf, true)
			c.Data(http.StatusOK, "text/plain", buf[:n])
		})

		// DB connection pool stats
		debug.GET("/db-pool", func(c *gin.Context) {
			if db.DB == nil {
				c.JSON(http.StatusOK, gin.H{"error": "database not connected"})
				return
			}
			stats := db.DB.Stats()
			c.JSON(http.StatusOK, gin.H{
				"max_open_connections": stats.MaxOpenConnections,
				"open_connections":     stats.OpenConnections,
				"in_use":               stats.InUse,
				"idle":                 stats.Idle,
				"wait_count":           stats.WaitCount,
				"wait_duration":        stats.WaitDuration.String(),
				"max_idle_closed":      stats.MaxIdleClosed,
				"max_idle_time_closed": stats.MaxIdleTimeClosed,
				"max_lifetime_closed":  stats.MaxLifetimeClosed,
			})
		})

		// Public diagnostic endpoints
		debug.GET("/system", handlers.DebugSystemSummary)
		debug.GET("/requests/summary", handlers.DebugRequestSummary)
	}
}
