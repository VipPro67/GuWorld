package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/internal/server"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/internal/uihub"
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
	var envLoaded bool
	var loadedEnvPath string
	for _, envPath := range []string{".env", "backend/.env", "../.env", "../../.env"} {
		if err := godotenv.Load(envPath); err == nil {
			envLoaded = true
			loadedEnvPath = envPath
			break
		}
	}

	if err := logger.Init(); err != nil {
		os.Stderr.WriteString("Failed to initialize logger: " + err.Error() + "\n")
		os.Exit(1)
	}
	defer logger.Sync()

	logger.S.Infof("App running in default local timezone: %s", time.Now().Location())
	if envLoaded {
		logger.S.Infof("Loaded environment variables from %s", loadedEnvPath)
	} else {
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
		// Redis is required for full functionality, but a transient startup
		// failure (e.g. Docker DNS not ready) must not crash-loop the server:
		// restarting replays InitSchema's heavy backfills/DDL and can take the
		// whole database down. Continue with in-memory fallbacks instead.
		logger.S.Errorf("Redis unavailable after retries, continuing with in-memory fallbacks: %v", err)
	} else {
		_ = cache.Delete("game_dictionary_unified", "game_dictionary_gus", "game_dictionary_items", "game_dictionary_monsters", "game_dictionary_constitutions", "game_dictionary_races", "game_dictionary_loot_tables", "game_dictionary_regions", "game_dictionary_monster_spawns")
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

	err = models.LoadLootRegistry(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load Loot registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Loot registry loaded successfully")
	}

	err = models.LoadRecipeRegistry(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load Recipe registry: %v (data may need import)", err)
	} else {
		logger.S.Info("Recipe registry loaded successfully")
	}

	err = handlers.InitInstanceCache()
	if err != nil {
		logger.S.Warnf("Failed to load Instance registry: %v", err)
	} else {
		logger.S.Info("Instance registry loaded successfully")
	}

	// Precompute the unified dictionary payload off the request path.
	go handlers.WarmDictionaryCache()

	playerSvc := services.NewPlayerService(db.DB)
	handlers.PlayerSvc = playerSvc
	handlers.RecipeSvc = services.NewRecipeService(db.DB)
	handlers.InventorySvc = services.NewInventoryService(db.DB)
	handlers.QuestSvc = services.NewQuestService(db.DB)
	handlers.AchievementSvc = services.NewAchievementService(db.DB)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	worker.StartHungerWorker(ctx, db.DB, 3*time.Hour)
	worker.StartPetHungerWorker(ctx, db.DB, 1*time.Hour)
	worker.StartGrowthWorker(ctx, db.DB, 1*time.Hour)
	handlers.StartTribulationWorker(ctx)
	handlers.StartRefineSessionWorker(ctx)
	handlers.StartCombatSessionWorker(ctx)
	handlers.StartWorldIndexSyncWorker(ctx)
	handlers.StartSpatialRealmExpiryWorker(ctx)
	handlers.StartPrimevalVeinWorker(ctx)
	handlers.StartWorldResourcePointWorker(ctx)
	handlers.StartArenaSettlementWorker(ctx)
	handlers.StartAuctionSettlementWorker(ctx)
	go handlers.StartBeastBettingWorker(ctx)
	go handlers.StartDeathArenaWorker(ctx)
	go handlers.StartMartialArenaWorker(ctx)
	worker.StartPoolWatchdog(ctx, db.DB, 60*time.Second)
	worker.StartSectMissionWorker(ctx, db.DB, 60*time.Second)
	worker.StartSectTerritoryWorker(ctx, db.DB, 1*time.Hour)
	worker.StartSectMaintenanceWorker(ctx, db.DB, 1*time.Hour)
	worker.StartCombatHistoryCleanup(ctx, db.DB, 24*time.Hour)
	worker.StartDailyMissionRolloverWorker(ctx, db.DB, 5*time.Minute)
	handlers.StartLastActivityFlusher(ctx)
	worker.StartBatchPersister(ctx, 5*time.Second)
	worker.StartStatusExpiryWorker(ctx, db.DB, 5*time.Minute)
	worker.StartSocietySimulationWorker(ctx, db.DB, 5*time.Minute)
	worker.StartGuMasterSpawningWorker(ctx, db.DB, 30*time.Minute)
	worker.StartApertureRefinementWorker(ctx, db.DB)
	handlers.StartRoamingMerchantWorker(ctx, db.DB, 2*time.Minute)
	if cache.Client != nil {
		worker.StartOrphanedKeyCleanup(ctx, 10*time.Minute)
	}

	// Initialize UI hub and Redis Pub/Sub
	shub := uihub.InitShardedHub()
	if cache.Client != nil {
		cache.InitCircuitBreaker()
		uihub.StartRedisSubscribers(shub)
	}

	uiServer := uihub.NewServer(shub, uihub.UIOptions{
		Command: func(playerID uuid.UUID, command string, payload json.RawMessage) (int, json.RawMessage) {
			switch command {
			case "world.cells_rect", "world.drops":
				return handlers.HandleWorldWSCommand(playerID, command, payload)
			default:
				return http.StatusNotFound, json.RawMessage(`{"error":"Unknown UI command"}`)
			}
		},
		SectLookup: func(playerID uuid.UUID) string {
			if db.DB == nil {
				return ""
			}
			var sectID string
			if err := db.DB.QueryRow("SELECT sect_id FROM sect_members WHERE player_id = $1", playerID).Scan(&sectID); err != nil {
				return ""
			}
			return sectID
		},
		RegionScope: func(playerID uuid.UUID) (string, error) {
			return handlers.CurrentRegionChatScope(playerID.String())
		},
		InitialEvents: func(playerID uuid.UUID) [][]byte {
			var events [][]byte
			if snapshot, err := handlers.LoadPlayerRealtimeSnapshot(playerID); err == nil {
				if msg, err := json.Marshal(struct {
					Type    string                          `json:"type"`
					Channel string                          `json:"channel"`
					Data    handlers.PlayerRealtimeSnapshot `json:"data"`
				}{Type: "event", Channel: "player", Data: snapshot}); err == nil {
					events = append(events, msg)
				}
			}
			if trade, err := handlers.LoadActiveTrade(playerID); err == nil && trade != nil {
				if msg, err := json.Marshal(struct {
					Type    string `json:"type"`
					Channel string `json:"channel"`
					Data    any    `json:"data"`
				}{Type: "event", Channel: "trade", Data: map[string]any{"trade": trade}}); err == nil {
					events = append(events, msg)
				}
			}
			return events
		},
		TouchActivity: handlers.TouchPlayerActivity,
	})

	r := server.SetupRouter(shub)

	wsFallbackEnabled := !strings.EqualFold(strings.TrimSpace(os.Getenv("WS_FALLBACK_ENABLED")), "false")
	if wsFallbackEnabled {
		r.GET("/api/ws", func(c *gin.Context) {
			if !realtimeOriginChecker()(c.Request) {
				c.Status(http.StatusForbidden)
				return
			}
			identity, status, message := middleware.AuthenticateRequest(c.Request)
			if identity == nil {
				c.String(status, message)
				return
			}
			tabID := strings.TrimSpace(c.Query("tab_id"))
			if len(tabID) > 100 {
				tabID = ""
			}
			if tabID != "" {
				middleware.ClaimActiveTab(identity.PlayerID, tabID)
			}
			_ = uiServer.HandleWebSocket(c.Writer, c.Request, identity.PlayerID, tabID, realtimeOriginChecker())
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	tlsEnabled := envBool("TLS_ENABLED")
	tlsAddr := strings.TrimSpace(os.Getenv("TLS_ADDR"))
	if tlsAddr == "" {
		tlsAddr = ":8443"
	}

	var httpServer *http.Server

	if tlsEnabled {
		certFile := strings.TrimSpace(os.Getenv("TLS_CERT_FILE"))
		keyFile := strings.TrimSpace(os.Getenv("TLS_KEY_FILE"))
		if certFile == "" || keyFile == "" {
			logger.S.Fatalf("TLS_ENABLED requires TLS_CERT_FILE and TLS_KEY_FILE")
		}
		cert, certErr := tls.LoadX509KeyPair(certFile, keyFile)
		if certErr != nil {
			logger.S.Fatalf("load certificate: %v", certErr)
		}
		tlsTCP := &tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: []string{"h2", "http/1.1"}}
		httpServer = &http.Server{
			Addr:              tlsAddr,
			Handler:           r,
			TLSConfig:         tlsTCP,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		go func() {
			logger.S.Infof("Cultivation Backend listening on %s (HTTPS)", tlsAddr)
			if err := httpServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				logger.S.Fatalf("Server failed to start: %v", err)
			}
		}()
	} else {
		httpServer = &http.Server{
			Addr:              ":" + port,
			Handler:           r,
			ReadHeaderTimeout: 10 * time.Second,
			IdleTimeout:       120 * time.Second,
		}
		go func() {
			logger.S.Infof("Cultivation Backend listening on port %s (plain HTTP)", port)
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.S.Fatalf("Server failed to start: %v", err)
			}
		}()
	}

	<-ctx.Done()
	logger.S.Info("Shutting down gracefully...")

	if uiServer != nil {
		if closed := uiServer.Drain(); closed > 0 {
			logger.S.Infof("Sent 4010 server_draining to %d active UI sessions", closed)
		}
		deadline := time.Now().Add(3 * time.Second)
		for uiServer.ActiveSessions() > 0 && time.Now().Before(deadline) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.S.Fatalf("Server forced to shutdown: %v", err)
	}

	shub.Stop(shutdownCtx)

	logger.S.Info("Server exited cleanly")
}

func envBool(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func realtimeOriginChecker() func(*http.Request) bool {
	raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("WS_ALLOWED_ORIGINS"))
	}
	allowed := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		for _, item := range allowed {
			if strings.EqualFold(item, origin) {
				return true
			}
		}
		return false
	}
}
