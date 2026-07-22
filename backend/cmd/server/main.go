package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/server"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/internal/worker"
)

func main() {
	// Try loading .env from different possible locations depending on where the server is run from
	var envLoaded bool
	for _, envPath := range []string{".env", "backend/.env", "../.env", "../../.env"} {
		if err := godotenv.Load(envPath); err == nil {
			log.Printf("Loaded environment variables from %s", envPath)
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		log.Println("Warning: No .env file loaded, using default system/environment variables")
	}

	// 1. Connect to Database
	err := db.Connect()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.DB.Close()

	// 2. Initialize Database Schema (tables only, no CSV seed)
	err = db.InitSchema()
	if err != nil {
		log.Fatalf("Database schema initialization failed: %v", err)
	}

	// Initialize Redis Cache
	if err := cache.Connect(); err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v. Proceeding without caching.", err)
	} else {
		// Clear dictionary cache to ensure latest database values are loaded on startup
		_ = cache.Delete("game_dictionary_unified")
	}

	// 2.2. Load Gu registry cache from database (may be empty until admin imports data)
	err = models.LoadGuRegistry(db.DB)
	if err != nil {
		log.Printf("Warning: Failed to load Gu registry: %v (data may need import)", err)
	}

	// 2.3. Load Item registry cache from database
	err = models.LoadItemRegistry(db.DB)
	if err != nil {
		log.Printf("Warning: Failed to load Item registry: %v (data may need import)", err)
	}

	// Load Monster registry cache from database
	err = models.LoadMonsterRegistry(db.DB)
	if err != nil {
		log.Printf("Warning: Failed to load Monster registry: %v (data may need import)", err)
	}

	// 2.4. Load Quest registry from DB (or CSV fallback)
	err = handlers.LoadQuestRegistry()
	if err != nil {
		log.Printf("Warning: Failed to load Quest registry: %v (data may need import)", err)
	}

	// 2.5. Initialize services
	playerSvc := services.NewPlayerService(db.DB)
	handlers.PlayerSvc = playerSvc
	handlers.RecipeSvc = services.NewRecipeService(db.DB)
	handlers.MapService = services.NewMapService(db.DB)
	handlers.CombatSessionSvc = services.NewCombatSessionService(db.DB)
	handlers.DaoSvc = services.NewDaoService(db.DB)
	handlers.InventorySvc = services.NewInventoryService(db.DB)
	handlers.QuestSvc = services.NewQuestService(db.DB)
	handlers.AchievementSvc = services.NewAchievementService(db.DB)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 2.5. Start Hunger Decay Background Worker (3-hour intervals for slower, balanced survival gameplay)
	worker.StartHungerWorker(ctx, db.DB, 3*time.Hour)

	// 2.6. Start Tribulation background worker
	handlers.StartTribulationWorker(ctx)

	// 2.7. Start Mine Monster Worker
	handlers.StartMineMonsterWorker(ctx)

	// 2.8. Start Midnight Mine Worker (daily 00:00 reset)
	handlers.StartMidnightMineWorker(ctx)

	// 2.9. Start Arena Settlement Worker (daily 22:00)
	handlers.StartArenaSettlementWorker(ctx)

	// 3. Setup router & run
	r := server.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		log.Printf("Cultivation Backend listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly")
}
