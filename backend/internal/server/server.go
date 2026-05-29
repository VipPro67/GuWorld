package server

import (
	"net/http"
	"os"
	"strings"

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
		// Public Dictionary Endpoint
		api.GET("/dictionary", handlers.GetDictionary)
		api.GET("/recipes", handlers.GetRecipes)

		// Public Auth Endpoints
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", handlers.RegisterPlayer)
			authGroup.POST("/login", handlers.LoginPlayer)
			authGroup.POST("/logout", handlers.LogoutPlayer)
			authGroup.GET("/me", middleware.AuthRequired(), handlers.GetCurrentPlayer)
		}

		// Protected Cultivation Endpoints
		player := api.Group("/player")
		player.Use(middleware.AuthRequired())
		{
			player.GET("/status/:id", handlers.GetPlayerStatus)
			player.GET("/stats", handlers.GetPlayerStatsHandler)
			player.POST("/explore/:id", handlers.Explore)
			player.POST("/use-item", handlers.UseItem)
		}

		// Protected Inventory Endpoints
		inventory := api.Group("/inventory")
		inventory.Use(middleware.AuthRequired())
		{
			inventory.POST("/feed", handlers.FeedGu)
			inventory.POST("/equip", handlers.EquipGu)
			inventory.POST("/status", handlers.UpdateGuStatus)
			inventory.POST("/set-vital", handlers.SetVitalGu)
			inventory.POST("/use-gu", handlers.UseConsumableGu)
			inventory.POST("/drop", handlers.DropGu)
		}

		// Protected Refining Endpoints
		refining := api.Group("/refine")
		refining.Use(middleware.AuthRequired())
		{
			refining.POST("", handlers.RefineGu)
		}

		// Protected Combat & Explore Node Endpoints
		combat := api.Group("/combat")
		combat.Use(middleware.AuthRequired())
		{
			combat.POST("/attack", handlers.CombatAttack)
		}

		api.POST("/explore", middleware.AuthRequired(), handlers.ExploreNode)
		api.POST("/explore/retreat/:id", middleware.AuthRequired(), handlers.Retreat)
		api.GET("/explore/grid", middleware.AuthRequired(), handlers.GetGrid)
		api.POST("/explore/move", middleware.AuthRequired(), handlers.MovePlayer)

		// Shop Endpoints
		api.GET("/shop/list", handlers.GetShopList)
		shop := api.Group("/shop")
		shop.Use(middleware.AuthRequired())
		{
			shop.POST("/buy", handlers.BuyShopItem)
			shop.POST("/sell", handlers.SellShopItem)
			shop.GET("/bao-hoang/listings", handlers.ListListings)
			shop.POST("/bao-hoang/listings", handlers.CreateListing)
			shop.POST("/bao-hoang/bid", handlers.PlaceBid)
			shop.POST("/bao-hoang/cancel", handlers.CancelListing)
			shop.POST("/bao-hoang/settle", handlers.SettleExpiredAuctions)
		}
	}

	return r
}
