package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
	"github.com/vippro67/GuWorld/backend/internal/match"
	"github.com/vippro67/GuWorld/backend/internal/middleware"
	"github.com/vippro67/GuWorld/backend/internal/models"
	realtimeai "github.com/vippro67/GuWorld/backend/internal/realtime/ai"
	realtimeconfig "github.com/vippro67/GuWorld/backend/internal/realtime/config"
	realtimeeffects "github.com/vippro67/GuWorld/backend/internal/realtime/effects"
	realtimegateway "github.com/vippro67/GuWorld/backend/internal/realtime/gateway"
	realloot "github.com/vippro67/GuWorld/backend/internal/realtime/loot"
	"github.com/vippro67/GuWorld/backend/internal/realtime/nav"
	"github.com/vippro67/GuWorld/backend/internal/realtime/persistence"
	"github.com/vippro67/GuWorld/backend/internal/realtime/sim"
	realtimeworld "github.com/vippro67/GuWorld/backend/internal/realtime/world"
	"github.com/vippro67/GuWorld/backend/internal/server"
	"github.com/vippro67/GuWorld/backend/internal/services"
	"github.com/vippro67/GuWorld/backend/internal/worker"
	"github.com/vippro67/GuWorld/backend/internal/ws"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
	"github.com/vippro67/GuWorld/backend/pkg/notify"
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

	var realtimeStartupError string
	realtimeConfig, err := realtimeconfig.Load()
	if err != nil {
		realtimeStartupError = err.Error()
		logger.S.Warnf("Realtime world rollout config is invalid; keeping admission disabled: %v", err)
	} else if realtimeConfig.Enabled {
		if len(realtimeConfig.ZoneIDs) == 0 {
			realtimeStartupError = "REALTIME_WORLD_ENABLED=true but MMO_ZONE_IDS is empty"
		}
		logger.S.Infof("Realtime world rollout configured (zones=%d, allowlisted_players=%d, max_players_per_zone=%d); join also requires enabled client flag and matching map/content versions", len(realtimeConfig.ZoneIDs), len(realtimeConfig.PlayerAllowlist), realtimeConfig.MaxPlayersPerZone)
	} else {
		realtimeStartupError = "REALTIME_WORLD_ENABLED is false"
		logger.S.Info("Realtime world rollout is disabled")
	}

	err = db.Connect()
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

	err = models.LoadMobaProfiles(db.DB)
	if err != nil {
		logger.S.Warnf("Failed to load MOBA Gu profiles: %v (realtime mode falls back to defaults)", err)
	} else {
		logger.S.Info("MOBA Gu profiles loaded successfully")
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

	var realtimeWorldManager *realtimeworld.Manager
	var compiledGuCatalog *realtimeeffects.CompiledCatalog
	var realtimeStateStore *persistence.Service
	if realtimeConfig.Enabled && len(realtimeConfig.ZoneIDs) > 0 {
		instances := realtimeConfig.InstancesPerZone
		if instances < 1 {
			instances = 1
		}
		zoneKeys := make([]sim.ZoneKey, 0, len(realtimeConfig.ZoneIDs)*instances)
		for mapID := range realtimeConfig.ZoneIDs {
			for index := 0; index < instances; index++ {
				instanceID := "realtime-preview"
				if index > 0 {
					instanceID = fmt.Sprintf("realtime-preview-%d", index+1)
				}
				zoneKeys = append(zoneKeys, sim.ZoneKey{
					MapID:          mapID,
					InstanceID:     instanceID,
					RulesetVersion: "v1",
				})
			}
		}
		var guEntries []realtimeeffects.GuEntry
		if csvPath := resolveGuCSVPath(); csvPath != "" {
			if loaded, err := realtimeeffects.LoadGuCSV(csvPath); err == nil && len(loaded) > 0 {
				guEntries = loaded
			}
		}
		if len(guEntries) == 0 {
			// Fallback to in-memory database registry if CSV is unavailable
			for id, ranks := range models.GuRegistry {
				for rank, item := range ranks {
					guEntries = append(guEntries, realtimeeffects.GuEntry{
						ID: id, Name: item.Name, Rank: rank, Type: item.Type, PrimaryRole: item.PrimaryRole,
						ManaCost: item.BaseManaCost,
					})
				}
			}
		}
		profilePaths := resolveRealtimeProfilePaths(realtimeConfig.GuProfiles)
		if profiles, profileErr := realtimeeffects.LoadProfileCatalogs(profilePaths); profileErr != nil {
			logger.S.Errorf("Realtime Gu profiles unavailable at %s: %v", profilePaths, profileErr)
		} else if compiled, compileErr := realtimeeffects.CompileCatalog(profiles, guEntries, realtimeeffects.DefaultMVP()); compileErr != nil {
			logger.S.Errorf("Realtime Gu profiles failed compilation; live player hotbars remain empty (fail-closed): %v", compileErr)
		} else {
			compiledGuCatalog = compiled
			logger.S.Infof("Realtime Gu profiles compiled: profiles=%d actives=%d hash=%s", len(compiled.Profiles), len(compiled.DefaultSkills), compiled.Hash)
		}
		zoneMaps := make(map[sim.ZoneKey]realtimeworld.ZoneMap, len(zoneKeys))
		var mapLoadErr error
		for _, key := range zoneKeys {
			mapPath := resolveRealtimeMapPath(realtimeConfig.MapDir, key.MapID)
			loaded, mapErr := nav.LoadFile(mapPath)
			if mapErr != nil {
				if errors.Is(mapErr, os.ErrNotExist) {
					mapLoadErr = fmt.Errorf("configured realtime map %q not found at %s", key.MapID, mapPath)
				} else {
					mapLoadErr = fmt.Errorf("configured realtime map %q is invalid at %s: %w", key.MapID, mapPath, mapErr)
				}
				break
			}
			if realtimeConfig.MapVersion != "" && loaded.Version != realtimeConfig.MapVersion {
				mapLoadErr = fmt.Errorf("configured realtime map %q has version %q; MMO_MAP_VERSION requires %q", key.MapID, loaded.Version, realtimeConfig.MapVersion)
				break
			}
			if realtimeConfig.ContentVersion != "" && loaded.ContentVersion != realtimeConfig.ContentVersion {
				mapLoadErr = fmt.Errorf("configured realtime map %q has content version %q; MMO_CONTENT_VERSION requires %q", key.MapID, loaded.ContentVersion, realtimeConfig.ContentVersion)
				break
			}
			spawns := make([]sim.Vec2, 0, len(loaded.Spawns))
			monsters := make([]realtimeworld.MonsterSpawn, 0)
			for _, spawn := range loaded.Spawns {
				switch spawn.Kind {
				case "player":
					spawns = append(spawns, sim.Vec2{X: spawn.X, Y: spawn.Y})
				case "monster":
					template, ok := models.MonsterRegistry[spawn.MonsterID]
					if !ok {
						logger.S.Warnf("Realtime map %q monster spawn %q references unknown monster_id %q", key.MapID, spawn.Name, spawn.MonsterID)
						continue
					}
					stats := realtimeai.DefaultStats(template.Type)
					monsters = append(monsters, realtimeworld.MonsterSpawn{
						TemplateID: template.ID, Name: template.Name, X: spawn.X, Y: spawn.Y,
						MaxHP: template.MaxHP, AttackDamage: template.BaseAttack, Defense: template.BaseDefense,
						Speed: stats.Speed, Radius: stats.Radius,
						AggroRadius: stats.AggroRadius, LeashRadius: stats.LeashRadius,
						AttackRange: stats.AttackRange, PatrolRadius: stats.PatrolRadius,
						AttackCooldownTicks: stats.AttackCooldownTicks, RespawnTicks: stats.RespawnTicks,
						SplashRadius: stats.SplashRadius, SplashDamage: stats.SplashDamage,
					})
				}
			}
			zoneMaps[key] = realtimeworld.ZoneMap{
				Collision: loaded.Collision(), Spawns: spawns, Monsters: monsters,
				PvP: loaded.Ruleset == "pvp", Hash: loaded.Hash, SafeSpawn: loaded.SafeSpawn,
				LegacyOriginX: loaded.LegacyOriginX, LegacyOriginY: loaded.LegacyOriginY, LegacyScale: loaded.LegacyScale,
				Width: loaded.Width, Height: loaded.Height,
			}
			logger.S.Infof("Realtime map %q loaded (%dx%d, %d monsters, hash %s)", key.MapID, loaded.Width, loaded.Height, len(monsters), loaded.Hash)
		}
		realtimeStateStore = persistence.NewService(db.DB)
		var manager *realtimeworld.Manager
		var managerErr error
		if mapLoadErr != nil {
			managerErr = mapLoadErr
		} else {
			manager, managerErr = realtimeworld.NewManagerWithMaps(zoneKeys, sim.DefaultQueueSize, realtimeConfig.MaxPlayersPerZone, zoneMaps)
		}
		if managerErr == nil {
			inventorySvc := handlers.InventorySvc
			if inventorySvc == nil {
				inventorySvc = services.NewInventoryService(db.DB)
			}
			lootSvc := realloot.NewService(db.DB, inventorySvc)
			manager.SetDeathHandler(func(callCtx context.Context, death realtimeworld.DeathEvent) {
				recipient, parseErr := uuid.Parse(string(death.SourceID))
				if parseErr != nil {
					return
				}
				result, grantErr := lootSvc.GrantKill(callCtx, realloot.GrantKey{
					ZoneID: death.ZoneKey.MapID, DeathID: fmt.Sprintf("%d:%s", death.EventID, death.TargetID),
					RecipientID: recipient, TemplateID: death.TemplateID, RewardVersion: realloot.RewardVersion,
				})
				if grantErr != nil {
					logger.S.Errorf("Realtime loot grant failed for %s: %v", death.TargetID, grantErr)
					return
				}
				if result.AlreadyGranted {
					return
				}
				logger.S.Infof("Realtime loot granted: recipient=%s template=%s rewards=%d", recipient, death.TemplateID, len(result.Rewards))
				if len(result.Rewards) > 0 {
					templateName := death.TemplateID
					if tmpl, ok := models.MonsterRegistry[death.TemplateID]; ok && tmpl.Name != "" {
						templateName = tmpl.Name
					}
					var rewardParts []string
					for _, r := range result.Rewards {
						if r.Kind == "gu" {
							guName := r.ID
							if guMap, ok := models.GuRegistry[r.ID]; ok {
								if guItem, exists := guMap[r.Rank]; exists && guItem.Name != "" {
									guName = guItem.Name
								}
							}
							rewardParts = append(rewardParts, fmt.Sprintf("%s (Chuyển %d)", guName, r.Rank))
						} else {
							itemName := r.ID
							if itemSpec, ok := models.ItemRegistry[r.ID]; ok && itemSpec.Name != "" {
								itemName = itemSpec.Name
							}
							rewardParts = append(rewardParts, fmt.Sprintf("%s x%d", itemName, r.Quantity))
						}
					}
					_ = notify.Send(db.DB, recipient, "combat_loot", fmt.Sprintf("Trảm sát %s!", templateName), fmt.Sprintf("Thu được: %s.", strings.Join(rewardParts, ", ")))
				}
			})
			managerErr = manager.Start(ctx)
		}
		if managerErr != nil {
			realtimeStartupError = managerErr.Error()
			logger.S.Errorf("Realtime world manager not started; admission remains disabled: %v", managerErr)
		} else {
			realtimeStartupError = ""
			realtimeWorldManager = manager
			logger.S.Infof("Realtime world manager started with %d configured zones; WebSocket join is guarded by allowlist and version gates", len(zoneKeys))
		}
	}
	handlers.SetRealtimeRuntimeError(realtimeStartupError)
	if realtimeWorldManager != nil {
		middleware.SetRealtimeOwnershipChecker(func(playerID uuid.UUID) bool {
			return realtimeWorldManager.OwnsCharacter(sim.CharacterID(playerID.String()))
		})
	} else {
		middleware.SetRealtimeOwnershipChecker(nil)
	}

	worker.StartHungerWorker(ctx, db.DB, 3*time.Hour)
	worker.StartPetHungerWorker(ctx, db.DB, 1*time.Hour)
	worker.StartGrowthWorker(ctx, db.DB, 1*time.Hour)
	handlers.StartTribulationWorker(ctx)
	handlers.StartWorldIndexSyncWorker(ctx)
	handlers.StartWorldMonsterWorker(ctx)
	handlers.StartSpatialRealmExpiryWorker(ctx)
	handlers.StartPrimevalVeinWorker(ctx)
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
	// Aperture node output is granted by the harvest endpoints (aperture_collect.go)
	// alone; the old per-tick variant-production worker was removed so node yield
	// has a single source of truth.
	worker.StartSocietySimulationWorker(ctx, db.DB, 5*time.Minute) // Phase 2: society simulation tick
	worker.StartGuMasterSpawningWorker(ctx, db.DB, 30*time.Minute) // Phase 3: Gu Master spawning
	handlers.StartRoamingMerchantWorker(ctx, db.DB, 2*time.Minute) // Roaming traders on the world map
	match.StartTickLoop(ctx)                                       // Realtime PvP match simulation (20Hz)
	if cache.Client != nil {
		worker.StartOrphanedKeyCleanup(ctx, 10*time.Minute)
	}

	// Initialize WebSocket Hub and Redis Pub/Sub
	shub := ws.InitShardedHub()
	ws.InitConnectionLimiter(3, 10) // Max 3 connections per player, 10 per IP
	if cache.Client != nil {
		cache.InitCircuitBreaker() // Initialize Redis circuit breaker
		ws.StartRedisSubscribers(shub)
	}

	realtimeAdmission := realtimegateway.NewAdmission(realtimeConfig, realtimeWorldManager, nil)
	realtimeAdmission.SetLoadoutCatalog(compiledGuCatalog)
	realtimeAdmission.SetStateStore(realtimeStateStore)
	if realtimeWorldManager != nil {
		handlers.SetRealtimeStatsProvider(func() any { return realtimeWorldManager.Stats() })
		worker.RunEvery(ctx, 10*time.Second, "realtime-checkpoint", func(context.Context) {
			realtimeAdmission.CheckpointActive()
		})
	}
	r := server.SetupRouter(shub, realtimeAdmission)

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
	if realtimeAdmission != nil {
		for realtimeAdmission.ActiveSessionCount() > 0 && shutdownCtx.Err() == nil {
			if drained := realtimeAdmission.Drain(); drained > 0 {
				logger.S.Infof("Drained %d realtime sessions during shutdown", drained)
			} else {
				select {
				case <-shutdownCtx.Done():
				case <-time.After(500 * time.Millisecond):
				}
			}
		}
		if pending := realtimeAdmission.ActiveSessionCount(); pending > 0 {
			logger.S.Errorf("%d realtime sessions could not be settled before shutdown", pending)
		}
	}
	if realtimeWorldManager != nil {
		if err := realtimeWorldManager.Stop(shutdownCtx); err != nil {
			logger.S.Errorf("Realtime world manager shutdown did not complete cleanly: %v", err)
		}
	}

	shub.Stop(shutdownCtx)

	logger.S.Info("Server exited cleanly")
}

// resolveRealtimeMapPath supports both documented startup locations: running
// from backend/ and running from the repository root (or frontend/).
func resolveRealtimeMapPath(mapDir, mapID string) string {
	if filepath.IsAbs(mapDir) {
		return filepath.Join(mapDir, mapID+".json")
	}
	if cwd, err := os.Getwd(); err == nil {
		for current := cwd; ; current = filepath.Dir(current) {
			candidate := filepath.Join(current, mapDir, mapID+".json")
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}
	return filepath.Join(mapDir, mapID+".json")
}

// resolveRealtimeProfilePaths supports comma-separated catalogs when the
// server is launched from the repo root, backend/, or backend/cmd/server.
func resolveRealtimeProfilePaths(rawPaths string) string {
	paths := strings.Split(rawPaths, ",")
	resolved := make([]string, 0, len(paths))
	cwd, _ := os.Getwd()
	for _, raw := range paths {
		path := strings.TrimSpace(raw)
		if path == "" || filepath.IsAbs(path) {
			resolved = append(resolved, path)
			continue
		}
		found := ""
		for current := cwd; current != ""; {
			candidate := filepath.Join(current, path)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				found = candidate
				break
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
		if found == "" {
			if cwd != "" {
				found = filepath.Join(cwd, path)
			} else {
				found = path
			}
		}
		resolved = append(resolved, found)
	}
	return strings.Join(resolved, ",")
}

func resolveGuCSVPath() string {
	cwd, _ := os.Getwd()
	for current := cwd; current != ""; {
		for _, sub := range []string{"data/Gu.csv", "backend/data/Gu.csv"} {
			candidate := filepath.Join(current, sub)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return "data/Gu.csv"
}
