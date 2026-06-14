package server

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
)

// SetupRouter initializes the Gin router with middlewares and routes
func SetupRouter() *gin.Engine {
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
		for _, o := range allowedOrigins {
			if o == origin || o == "*" {
				isAllowed = true
				break
			}
		}

		if isAllowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		} else if allowedOriginsStr == "*" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		}

		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// API routes
	api := r.Group("/api")
	{
		// Public Dictionary and Recipes
		api.GET("/dictionary", handlers.GetDictionary)
		api.GET("/recipes", handlers.GetRecipes)

		// Public Auth Endpoints
		authGroup := api.Group("/auth")
		{
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
			player.GET("/profile/:username", handlers.GetPlayerProfile)
			player.PUT("/profile/settings", handlers.UpdateProfileSettings)
			player.GET("/referrals", handlers.GetReferralStats)
			player.POST("/simulate-deposit", handlers.SimulateDeposit)
			player.POST("/avatar", handlers.UploadAvatar)
			player.POST("/tick-regen", handlers.TickRegen)
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
			quest.POST("/complete", handlers.CompleteQuest)
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
			combat.POST("/attack", handlers.CombatAttack)
			combat.POST("/cast-killer-move", handlers.CastKillerMove)
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
			soul.POST("/calm", handlers.CalmSoul)
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
		feedback := api.Group("/feedback")
		feedback.Use(middleware.AuthRequired())
		{
			feedback.POST("", handlers.SubmitFeedback)
			feedback.POST("/:id/vote", handlers.VoteFeedback)
			feedback.PUT("/:id/status", handlers.UpdateFeedbackStatus)
		}
	}

	return r
}
