package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/server"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

func init() {
	// Set the global default timezone for Go (time.Local)
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		// Fallback to fixed UTC+7 if system tzdata fails to load
		loc = time.FixedZone("ICT", 7*3600)
	}
	time.Local = loc
	os.Setenv("TZ", "Asia/Ho_Chi_Minh")
}

func main() {
	if err := logger.Init(); err != nil {
		os.Stderr.WriteString("Failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger.S.Infof("App running in default local timezone: %s", time.Now().Location())

	var envLoaded bool
	for _, envPath := range []string{".env", "backend/.env", "../.env", "../../.env"} {
		if err := godotenv.Load(envPath); err == nil {
			logger.S.Infof("Loaded environment variables from %s", envPath)
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		logger.S.Warn("No .env file loaded, using default system/environment variables")
	}

	err := db.Connect()
	if err != nil {
		logger.S.Fatalf("Database connection failed: %v", err)
	}
	defer db.DB.Close()

	err = db.InitSchema()
	if err != nil {
		logger.S.Fatalf("Database schema initialization failed: %v", err)
	}

	if err := cache.Connect(); err != nil {
		logger.S.Warnf("Failed to connect to Redis: %v. Proceeding without caching.", err)
	} else {
		_ = cache.Delete("game_dictionary_unified")
	}

	// Initialize rate limiter store (Redis primary, in-memory fallback)
	rateLimitStore := middleware.NewFallbackStore()
	middleware.SetDefaultStore(rateLimitStore)
	rateLimitStore.StartRedisWatchdog(30 * time.Second)

	err = models.LoadGuRegistry(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load Gu registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Gu registry loaded successfully")
	}

	err = models.LoadItemRegistry(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load Item registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Item registry loaded successfully")
	}

	err = gamefx.LoadConstitutionRegistry(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load Constitution registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Constitution registry loaded successfully")
	}

	err = gamefx.LoadRaceRegistry(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load Race registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Race registry loaded successfully")
	}

	err = gamefx.LoadStatusRegistry(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load Status registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Status registry loaded successfully")
	}

	err = models.LoadMonsterRegistry(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load Monster registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Monster registry loaded successfully")
	}

	err = handlers.LoadZoneRegistry(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load Zone registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Zone registry loaded successfully")
	}

	err = handlers.LoadQuestRegistry()
	if err != nil {
		logger.S.Warnf("Failed to load Quest registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Quest registry loaded successfully")
	}

	err = handlers.InitInstanceCache()
	if err != nil {
		logger.S.Warnf("Failed to load Instance registry: %v", err)
	} else {
		logger.S.Info("Instance registry loaded successfully")
	}

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

	worker.StartHungerWorker(ctx, db.DB, 3*time.Hour)
	worker.StartStatusExpiryWorker(ctx, db.DB, 1*time.Minute)
	worker.StartPetHungerWorker(ctx, db.DB, 1*time.Hour)
	worker.StartGrowthWorker(ctx, db.DB, 60*time.Second)
	handlers.StartTribulationWorker(ctx)
	handlers.StartWorldIndexSyncWorker(ctx)
	handlers.StartMineMonsterWorker(ctx)
	handlers.StartMidnightMineWorker(ctx)
	handlers.StartArenaSettlementWorker(ctx)
	handlers.StartAuctionSettlementWorker(ctx)
	go handlers.StartBeastBettingWorker(ctx)
	worker.StartPoolWatchdog(ctx, db.DB, 60*time.Second)
	worker.StartSectMissionWorker(ctx, db.DB, 60*time.Second)
	worker.StartSectTerritoryWorker(ctx, db.DB, 60*time.Second)
	worker.StartSectMaintenanceWorker(ctx, db.DB, 24*time.Hour)

	r := server.SetupRouter()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.S.Infof("Cultivation Backend listening on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.S.Fatalf("Server failed to start: %v", err)
		}
	}()

	<-ctx.Done()
	logger.S.Info("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.S.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.S.Info("Server exited cleanly")
}
