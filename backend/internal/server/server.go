package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// SetupRouter initializes the Gin router with middlewares and routes
func SetupRouter() *gin.Engine {
	gin.DefaultWriter = logger.GinWriter()
	r := gin.Default()

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

	// Serve uploaded images
	r.Static("/images", "./storage/images")

	// API routes
	api := r.Group("/api")
	{
		// Public Dictionary and Recipes
		api.GET("/dictionary", handlers.GetDictionary)
		api.GET("/recipes", handlers.GetRecipes)

		// Public Auth Endpoints
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/send-otp", middleware.RateLimit(5, time.Minute), handlers.SendOTP)
			authGroup.POST("/register", middleware.RateLimit(5, time.Minute), handlers.RegisterPlayer)
			authGroup.POST("/login", middleware.RateLimit(5, time.Minute), handlers.LoginPlayer)
			authGroup.POST("/logout", handlers.LogoutPlayer)
			authGroup.GET("/me", middleware.AuthRequired(), handlers.GetCurrentPlayer)

			// Google OAuth2
			authGroup.GET("/google/login", middleware.RateLimit(5, time.Minute), handlers.GoogleLogin)
			authGroup.GET("/google/callback", middleware.RateLimit(10, time.Minute), handlers.GoogleCallback)
		}

		api.GET("/topup/packages", handlers.GetTopupPackages)
		api.POST("/webhooks/payos", handlers.PayOSWebhook)
		api.GET("/admin/topup-orders", middleware.AuthRequired(), handlers.AdminListTopupOrders)
		api.GET("/admin/wallet-transactions", middleware.AuthRequired(), handlers.AdminListWalletTransactions)
		api.GET("/admin/players/online", middleware.AuthRequired(), handlers.AdminListOnlinePlayers)
		api.GET("/admin/players", middleware.AuthRequired(), handlers.AdminListPlayers)
		api.GET("/admin/players/:id", middleware.AuthRequired(), handlers.AdminGetPlayerDetail)
		api.GET("/admin/players/:id/items", middleware.AuthRequired(), handlers.AdminListPlayerItems)
		api.GET("/admin/players/:id/gus", middleware.AuthRequired(), handlers.AdminListPlayerGus)
		api.PUT("/admin/players/:id", middleware.AuthRequired(), handlers.AdminUpdatePlayer)
		api.GET("/admin/gus/search", middleware.AuthRequired(), handlers.AdminSearchGus)
		api.POST("/admin/players/:id/items", middleware.AuthRequired(), handlers.AdminAddPlayerItems)
		api.DELETE("/admin/players/:id/items", middleware.AuthRequired(), handlers.AdminRemovePlayerItems)
		api.POST("/admin/players/:id/gus", middleware.AuthRequired(), handlers.AdminAddPlayerGus)
		api.DELETE("/admin/players/:id/gus/:instance_id", middleware.AuthRequired(), handlers.AdminRemovePlayerGu)
		api.PUT("/admin/players/:id/stones", middleware.AuthRequired(), handlers.AdminChangePlayerStones)
		api.GET("/admin/sects", middleware.AuthRequired(), handlers.AdminListSects)
		api.GET("/admin/sects/:id", middleware.AuthRequired(), handlers.AdminGetSectDetail)
		api.PUT("/admin/sects/:id", middleware.AuthRequired(), handlers.AdminUpdateSect)
		api.POST("/admin/sects/:id/members/:memberId/leader", middleware.AuthRequired(), handlers.AdminSetSectLeader)
		api.DELETE("/admin/sects/:id/members/:memberId", middleware.AuthRequired(), handlers.AdminKickSectMember)
		api.POST("/admin/sects/:id/members", middleware.AuthRequired(), handlers.AdminAddToSect)
		api.DELETE("/admin/sects/:id", middleware.AuthRequired(), handlers.AdminDeleteSect)
		api.GET("/admin/sects/:id/missions", middleware.AuthRequired(), handlers.AdminListSectMissions)
		api.POST("/admin/sects/:id/missions", middleware.AuthRequired(), handlers.AdminCreateSectMission)
		api.DELETE("/admin/sects/:id/missions/:missionId", middleware.AuthRequired(), handlers.AdminDeleteSectMission)
		api.POST("/admin/sects/:id/vault/stock", middleware.AuthRequired(), handlers.AdminStockSectVault)
		api.DELETE("/admin/sects/:id/vault/stock", middleware.AuthRequired(), handlers.AdminRemoveSectVaultItem)
		api.POST("/admin/data/import", middleware.AuthRequired(), handlers.AdminImportData)
		api.POST("/admin/data/export", middleware.AuthRequired(), handlers.AdminExportData)
		api.POST("/admin/clear-cache", middleware.AuthRequired(), handlers.AdminClearCache)
		api.GET("/admin/audit-logs", middleware.AuthRequired(), handlers.AdminGetAuditLogs)
		api.GET("/admin/shop", middleware.AuthRequired(), handlers.AdminListShopItems)
		api.POST("/admin/shop", middleware.AuthRequired(), handlers.AdminCreateShopItem)
		api.PUT("/admin/shop/:id", middleware.AuthRequired(), handlers.AdminUpdateShopItem)
		api.DELETE("/admin/shop/:id", middleware.AuthRequired(), handlers.AdminDeleteShopItem)
		api.GET("/admin/arena-shop", middleware.AuthRequired(), handlers.AdminListArenaShopItems)
		api.POST("/admin/arena-shop", middleware.AuthRequired(), handlers.AdminCreateArenaShopItem)
		api.PUT("/admin/arena-shop/:id", middleware.AuthRequired(), handlers.AdminUpdateArenaShopItem)
		api.DELETE("/admin/arena-shop/:id", middleware.AuthRequired(), handlers.AdminDeleteArenaShopItem)
		api.GET("/admin/mine-slots", middleware.AuthRequired(), handlers.AdminListMineSlots)
		api.PUT("/admin/mine-slots/:id", middleware.AuthRequired(), handlers.AdminUpdateMineSlot)
		api.POST("/admin/mine-slots/:id/reset", middleware.AuthRequired(), handlers.AdminResetMineSlot)
		api.GET("/admin/gu-dictionary", middleware.AuthRequired(), handlers.AdminListGuItems)
		api.POST("/admin/gu-dictionary", middleware.AuthRequired(), handlers.AdminCreateGuItem)
		api.PUT("/admin/gu-dictionary/:id", middleware.AuthRequired(), handlers.AdminUpdateGuItem)
		api.DELETE("/admin/gu-dictionary/:id", middleware.AuthRequired(), handlers.AdminDeleteGuItem)
		api.GET("/admin/item-dictionary", middleware.AuthRequired(), handlers.AdminListItemItems)
		api.POST("/admin/item-dictionary", middleware.AuthRequired(), handlers.AdminCreateItemItem)
		api.PUT("/admin/item-dictionary/:id", middleware.AuthRequired(), handlers.AdminUpdateItemItem)
		api.DELETE("/admin/item-dictionary/:id", middleware.AuthRequired(), handlers.AdminDeleteItemItem)
		api.GET("/admin/recipes", middleware.AuthRequired(), handlers.AdminListRecipes)
		api.POST("/admin/recipes", middleware.AuthRequired(), handlers.AdminCreateRecipe)
		api.PUT("/admin/recipes/:id", middleware.AuthRequired(), handlers.AdminUpdateRecipe)
		api.DELETE("/admin/recipes/:id", middleware.AuthRequired(), handlers.AdminDeleteRecipe)
		api.POST("/admin/gift-codes", middleware.AuthRequired(), handlers.AdminCreateGiftCode)
		api.GET("/admin/gift-codes", middleware.AuthRequired(), handlers.AdminListGiftCodes)
		api.POST("/admin/upload-image", middleware.AuthRequired(), handlers.AdminUploadImage)
		api.GET("/wallet", middleware.AuthRequired(), handlers.GetWallet)
		api.POST("/topup/orders", middleware.AuthRequired(), middleware.RateLimit(10, time.Minute), handlers.CreateTopupOrder)
		api.GET("/topup/orders/:orderCode", middleware.AuthRequired(), handlers.GetTopupOrder)

		// Protected Cultivation Endpoints
		player := api.Group("/player")
		player.Use(middleware.AuthRequired())
		{
			player.GET("/events", handlers.StreamPlayerEvents)
			player.GET("/status/:id", handlers.GetPlayerStatus)
			player.GET("/stats", handlers.GetPlayerStatsHandler)
			player.POST("/explore/:id", handlers.Explore)
			player.POST("/use-item", handlers.UseItem)
			player.POST("/recharge", handlers.RechargeWithStones)
			player.POST("/clear-stress", handlers.ClearStress)
			player.POST("/sacrifice", handlers.SacrificeNPC)
			player.GET("/profile/:username", handlers.GetPlayerProfile)
			player.PUT("/profile/settings", handlers.UpdateProfileSettings)
			player.GET("/referrals", handlers.GetReferralStats)
			player.POST("/simulate-deposit", handlers.SimulateDeposit)
			player.POST("/avatar", handlers.UploadAvatar)
			player.POST("/tick-regen", handlers.TickRegen)
			player.POST("/redeem-gift-code", handlers.RedeemGiftCode)
			player.POST("/complete-onboarding", handlers.CompleteOnboarding)
			player.GET("/combat-settings", handlers.GetCombatSettings)
			player.PUT("/combat-settings", handlers.UpdateCombatSettings)
			player.GET("/audit-logs", handlers.GetMyAuditLogs)
		}

		mine := api.Group("/mine")
		mine.Use(middleware.AuthRequired())
		{
			mine.GET("/slots", handlers.GetMineSlots)
			mine.POST("/occupy", handlers.OccupyMineSlot)
			mine.POST("/claim", handlers.ClaimMineRewards)
			mine.POST("/abandon", handlers.AbandonMineSlot)
			mine.POST("/attack", handlers.AttackMineSlot)
		}

		cultivate := api.Group("/cultivate")
		cultivate.Use(middleware.AuthRequired())
		{
			cultivate.POST("/break-wall", handlers.BreakWall)
			cultivate.POST("/use-relic-gu", handlers.UseRelicGu)
			cultivate.POST("/ascension", handlers.Ascension)
			cultivate.POST("/tribulation", handlers.Tribulation)
		}

		quest := api.Group("/quest")
		quest.Use(middleware.AuthRequired())
		{
			quest.GET("", handlers.ListQuests)
			quest.POST("/complete", handlers.CompleteQuest)
		}

		gacha := api.Group("/gacha")
		gacha.Use(middleware.AuthRequired())
		{
			gacha.GET("/pools", handlers.GetGachaPools)
			gacha.POST("/open", handlers.OpenGacha)
			gacha.POST("/open10", handlers.OpenGacha10)
		}

		achievement := api.Group("/achievements")
		achievement.Use(middleware.AuthRequired())
		{
			achievement.GET("", handlers.GetAchievements)
			achievement.POST("/claim", handlers.ClaimAchievement)
			achievement.POST("/select", handlers.SelectTitleFrame)
		}

		// Protected Inventory Endpoints
		party := api.Group("/party")
		party.Use(middleware.AuthRequired())
		{
			party.POST("/create", handlers.CreateParty)
			party.GET("/status", handlers.GetPartyStatus)
			party.POST("/invite", handlers.InvitePlayer)
			party.POST("/leave", handlers.LeaveParty)
		}

		inventory := api.Group("/inventory")
		inventory.Use(middleware.AuthRequired())
		{
			inventory.POST("/feed", handlers.FeedGu)
			inventory.POST("/feed-group", handlers.FeedGroup)
			inventory.POST("/equip", handlers.EquipGu)
			inventory.POST("/status", handlers.UpdateGuStatus)
			inventory.POST("/set-vital", handlers.SetVitalGu)
			inventory.POST("/use-gu", handlers.UseConsumableGu)
			inventory.POST("/use-support", handlers.UseSupportGu)
			inventory.POST("/drop", handlers.DropGu)
		}

		// Protected Refining Endpoints
		refining := api.Group("/refine")
		refining.Use(middleware.AuthRequired())
		{
			refining.POST("", handlers.RefineGu)
			refining.POST("/deduce", handlers.DeduceRecipe)
		}

		// Protected Killer Move Endpoints
		killerMoves := api.Group("/killer-moves")
		killerMoves.Use(middleware.AuthRequired())
		{
			killerMoves.POST("/calculate", handlers.CalculateKillerMove)
			killerMoves.POST("", handlers.CreateKillerMove)
			killerMoves.GET("", handlers.ListKillerMoves)
			killerMoves.GET("/:id", handlers.GetKillerMove)
			killerMoves.DELETE("/:id", handlers.DeleteKillerMove)
		}

		// Protected Combat & Explore Node Endpoints
		combat := api.Group("/combat")
		combat.Use(middleware.AuthRequired())
		{
			combat.POST("/start", handlers.AutoCombatStart)
			combat.POST("/attack", handlers.CombatAttack)
			combat.POST("/cast-killer-move", handlers.CastKillerMove)
			combat.GET("/history", handlers.ListCombatHistory)
			combat.GET("/history/:id", handlers.GetCombatHistory)
		}

		// Protected Immortal Aperture Endpoints
		aperture := api.Group("/immortal-aperture")
		aperture.Use(middleware.AuthRequired())
		{
			aperture.GET("", handlers.GetImmortalAperture)
			aperture.GET("/nodes", handlers.GetApertureNodes)
			aperture.POST("/nodes", handlers.PlaceApertureNode)
			aperture.POST("/nodes/:id/collect", handlers.CollectApertureResource)
			aperture.POST("/nodes/:id/uninstall", handlers.UninstallApertureNode)
			aperture.DELETE("/nodes/:id", handlers.DeleteApertureNode)
			aperture.POST("/time-rate", handlers.SetApertureTimeRate)
			aperture.POST("/labor", handlers.SetApertureLabor)
			aperture.POST("/nodes/:id/repair", handlers.RepairApertureNode)
			aperture.GET("/variant-humans", handlers.GetApertureVariantHumans)
		}

		// Protected Dao Marks Endpoints
		daoMarks := api.Group("/dao-marks")
		daoMarks.Use(middleware.AuthRequired())
		{
			daoMarks.GET("", handlers.GetDaoMarks)
		}

		// Protected Tribulation Endpoints
		tribulations := api.Group("/tribulations")
		tribulations.Use(middleware.AuthRequired())
		{
			tribulations.GET("/active", handlers.GetActiveTribulation)
			tribulations.POST("/:eventId/attack", handlers.AttackTribulationEnemy)
			tribulations.POST("/:eventId/resolve", handlers.ResolveTribulation)
		}

		// Arena Endpoints
		arena := api.Group("/arena")
		arena.Use(middleware.AuthRequired())
		{
			arena.GET("/leaderboard", handlers.GetArenaLeaderboard)
			arena.POST("/challenge", handlers.ChallengeArena)
			arena.GET("/shop", handlers.GetArenaShop)
			arena.POST("/shop/buy", handlers.BuyArenaShopItem)
		}

		// Leaderboard Endpoints
		leaderboard := api.Group("/leaderboard")
		leaderboard.Use(middleware.AuthRequired())
		{
			leaderboard.GET("/:category", handlers.GetLeaderboard)
			leaderboard.POST("/toggle", handlers.ToggleLeaderboardVisibility)
			leaderboard.GET("/settings", handlers.GetLeaderboardSettings)
		}

		// Chat / Mail Endpoints
		chat := api.Group("/chat")
		chat.Use(middleware.AuthRequired())
		{
			chat.POST("/mail/send", handlers.SendPrivateMail)
			chat.GET("/mail/inbox", handlers.GetInbox)
			chat.POST("/bao-hoang/send", handlers.SendBaoHoangChat)
			chat.GET("/bao-hoang/messages", handlers.GetBaoHoangMessages)
			chat.GET("/bao-hoang/events", handlers.SubscribeBaoHoangMessages)
		}

		api.POST("/explore", middleware.AuthRequired(), handlers.ExploreNode)
		api.POST("/explore/retreat/:id", middleware.AuthRequired(), handlers.Retreat)
		api.POST("/explore/flee/:id", middleware.AuthRequired(), handlers.Flee)
		api.GET("/explore/grid", middleware.AuthRequired(), handlers.GetGrid)
		api.GET("/explore/regions", middleware.AuthRequired(), handlers.GetRegions)
		api.POST("/explore/move", middleware.AuthRequired(), handlers.MovePlayer)

		// Shop Endpoints
		api.GET("/shop/packages", middleware.AuthRequired(), handlers.GetPremiumShopPackages)
		api.GET("/shop/list", handlers.GetShopList)
		shop := api.Group("/shop")
		shop.Use(middleware.AuthRequired())
		{
			shop.POST("/buy-package", middleware.RateLimit(20, time.Minute), handlers.BuyPremiumPackage)
			shop.POST("/buy", handlers.BuyShopItem)
			shop.POST("/sell", handlers.SellShopItem)
			shop.GET("/bao-hoang/listings", handlers.ListListings)
			shop.POST("/bao-hoang/listings", handlers.CreateListing)
			shop.POST("/bao-hoang/buy", handlers.BuyListing)
			shop.POST("/bao-hoang/bid", handlers.PlaceBid)
			shop.POST("/bao-hoang/cancel", handlers.CancelListing)
			shop.POST("/bao-hoang/settle", handlers.SettleExpiredAuctions)
		}

		// Direct P2P Trade Endpoints
		trade := api.Group("/trade")
		trade.Use(middleware.AuthRequired())
		{
			trade.POST("/initiate", handlers.InitiateTrade)
			trade.GET("/active", handlers.GetActiveTrade)
			trade.GET("/events", handlers.SubscribeTradeEvents)
			trade.POST("/accept/:id", handlers.AcceptTrade)
			trade.POST("/decline/:id", handlers.DeclineTrade)
			trade.POST("/cancel/:id", handlers.CancelTrade)
			trade.POST("/offer/:id", handlers.UpdateTradeOffer)
			trade.POST("/lock/:id", handlers.LockTrade)
			trade.POST("/confirm/:id", handlers.ConfirmTrade)
		}

		// Protected Soul Endpoints
		soul := api.Group("/soul")
		soul.Use(middleware.AuthRequired())
		{
			soul.POST("/use-guts", handlers.UseGutsGu)
			soul.POST("/refine", handlers.RefineSoul)

		}

		// Protected Instance Endpoints
		instance := api.Group("/instance")
		instance.Use(middleware.AuthRequired())
		{
			instance.GET("/active", handlers.GetActiveInstance)
			instance.POST("/enter", handlers.EnterInstance)
			instance.POST("/action", handlers.ActionInstance)
		}

		// Protected Sect Endpoints
		sect := api.Group("/sects")
		sect.Use(middleware.AuthRequired())
		{
			sect.GET("", handlers.GetSects)
			sect.POST("/:id/join", handlers.JoinSect)
			sect.POST("/leave", handlers.LeaveSect)
			sect.GET("/me", handlers.GetMySectStatus)
			sect.POST("/promote", handlers.PromoteSectRank)
			sect.POST("/donate/stones", handlers.DonateStones)
			sect.POST("/donate/materials", handlers.DonateMaterials)
			sect.POST("/donate/gu", handlers.DonateGu)
			sect.POST("/vault/buy", handlers.BuyFromSectVault)
			sect.GET("/missions", handlers.GetSectMissions)
			sect.POST("/missions/:missionId/accept", handlers.AcceptSectMission)
			sect.POST("/missions/:missionId/complete", handlers.CompleteSectMission)
			sect.POST("/create", handlers.CreateSect)
			sect.GET("/me/members", handlers.GetSectMembers)
			sect.POST("/members/:memberId/rank", handlers.SetMemberRank)
			sect.POST("/members/:memberId/kick", handlers.KickMember)
			sect.POST("/missions/create", handlers.CreateCustomMission)
			sect.POST("/vault/stock", handlers.StockSectVault)
			sect.POST("/upgrade", handlers.UpgradeSect)
			sect.POST("/toggle-approval", handlers.ToggleSectApproval)
			sect.GET("/join-requests", handlers.GetJoinRequests)
			sect.POST("/join-requests/:requestId/action", handlers.ActionJoinRequest)
			sect.GET("/promotion-requests", handlers.GetPromotionRequests)
			sect.POST("/promotion-requests/:requestId/action", handlers.ActionPromotionRequest)
			sect.GET("/mines", handlers.GetSectMines)
			sect.POST("/mines/claim", handlers.ClaimSectMineRewards)
			sect.POST("/mines/attack", handlers.AttackSectMine)
		}

		// Feedback Endpoints
		api.GET("/feedback", handlers.GetFeedbacks)
		api.GET("/feedback/:id/comments", handlers.GetFeedbackComments)
		feedback := api.Group("/feedback")
		feedback.Use(middleware.AuthRequired())
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
