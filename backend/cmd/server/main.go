package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/handlers"
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
		logger.S.Warnf("Realtime world config is invalid; admission is unavailable: %v", err)
	} else if realtimeConfig.Enabled {
		logger.S.Infof("Realtime world enabled for signed-in players (all shipped public maps plus private event maps, max_players_per_zone=%d)", realtimeConfig.MaxPlayersPerZone)
	} else {
		realtimeStartupError = "realtime world configuration is disabled"
		logger.S.Info("Realtime world configuration is disabled")
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
	var tribulationMapLoadErr error
	if realtimeConfig.Enabled {
		instances := realtimeConfig.InstancesPerZone
		if instances < 1 {
			instances = 1
		}
		configuredZoneIDs := expandRealtimeZoneIDs(realtimeConfig.ZoneIDs, realtimeConfig.MapDir, realtimeConfig.MapVersion, realtimeConfig.ContentVersion)
		zoneIDs := make(map[string]struct{}, len(configuredZoneIDs)+1)
		for mapID := range configuredZoneIDs {
			zoneIDs[mapID] = struct{}{}
		}
		// aperture_tribulation is loaded as a private per-event map. The preview artifact also
		// backs isolated practice and arena combat sessions.
		zoneIDs["aperture_tribulation"] = struct{}{}
		zoneIDs["realtime_preview"] = struct{}{}
		var spatialRealmMapErr error
		spatialRows, spatialErr := db.DB.QueryContext(ctx, `
			SELECT DISTINCT CASE
				WHEN r.zone_type IN ('KHU_VUC','SAFE_ZONE','TRIBULATION') THEN r.id
				ELSE COALESCE(NULLIF(w.world_map_id,''),'world_map')
			END
			FROM instance_templates t
			JOIN regions r ON r.id=t.spatial_region_id AND r.is_active=TRUE
			LEFT JOIN region_world_maps w ON w.region_id=r.id
			WHERE t.type IN ('SPATIAL_REALM','SPATIAL_INHERITANCE') AND t.spatial_region_id IS NOT NULL
		`)
		if spatialErr != nil {
			spatialRealmMapErr = fmt.Errorf("load realtime spatial realm maps: %w", spatialErr)
		} else {
			for spatialRows.Next() {
				var mapID string
				if err := spatialRows.Scan(&mapID); err != nil {
					spatialRealmMapErr = fmt.Errorf("scan realtime spatial realm map: %w", err)
					break
				}
				if mapID != "" {
					zoneIDs[mapID] = struct{}{}
				}
			}
			if err := spatialRows.Err(); err != nil && spatialRealmMapErr == nil {
				spatialRealmMapErr = fmt.Errorf("read realtime spatial realm maps: %w", err)
			}
			spatialRows.Close()
		}
		logger.S.Infof("Realtime world maps enabled: %d (phaser_client=%v, map_version=%q, content_version=%q)",
			len(zoneIDs), realtimeConfig.PhaserClientEnabled, realtimeConfig.MapVersion, realtimeConfig.ContentVersion)
		zoneKeys := make([]sim.ZoneKey, 0, len(zoneIDs)*instances)
		for mapID := range zoneIDs {
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
		mapArtifacts := make(map[string]realtimeworld.ZoneMap, len(zoneIDs))
		mapLoadErr := spatialRealmMapErr
		for _, key := range zoneKeys {
			// Instances of the same map share one immutable collision/spawn
			// artifact; only their world state and runners differ.
			artifact, loaded := mapArtifacts[key.MapID]
			if !loaded {
				mapPath := resolveRealtimeMapPath(realtimeConfig.MapDir, key.MapID)
				loadedMap, mapErr := nav.LoadFile(mapPath)
				if mapErr != nil {
					if key.MapID == "aperture_tribulation" {
						tribulationMapLoadErr = fmt.Errorf("private tribulation map %q could not be loaded from %s: %w", key.MapID, mapPath, mapErr)
						continue
					}
					if errors.Is(mapErr, os.ErrNotExist) {
						mapLoadErr = fmt.Errorf("configured realtime map %q not found at %s", key.MapID, mapPath)
					} else {
						mapLoadErr = fmt.Errorf("configured realtime map %q is invalid at %s: %w", key.MapID, mapPath, mapErr)
					}
					break
				}
				expectedMapVersion, expectedContentVersion := realtimeconfig.VersionsForMap(key.MapID)
				if loadedMap.Version != expectedMapVersion {
					if key.MapID == "aperture_tribulation" {
						tribulationMapLoadErr = fmt.Errorf("private tribulation map %q has version %q; shipped release requires %q", key.MapID, loadedMap.Version, expectedMapVersion)
						continue
					}
					mapLoadErr = fmt.Errorf("configured realtime map %q has version %q; shipped artifact requires %q", key.MapID, loadedMap.Version, expectedMapVersion)
					break
				}
				if loadedMap.ContentVersion != expectedContentVersion {
					if key.MapID == "aperture_tribulation" {
						tribulationMapLoadErr = fmt.Errorf("private tribulation map %q has content version %q; shipped artifact requires %q", key.MapID, loadedMap.ContentVersion, expectedContentVersion)
						continue
					}
					mapLoadErr = fmt.Errorf("configured realtime map %q has content version %q; shipped artifact requires %q", key.MapID, loadedMap.ContentVersion, expectedContentVersion)
					break
				}
				spawns := make([]sim.Vec2, 0, len(loadedMap.Spawns))
				monsters := make([]realtimeworld.MonsterSpawn, 0)
				for _, spawn := range loadedMap.Spawns {
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
							TemplateID: template.ID, Rank: spawn.Rank, Name: template.Name, X: spawn.X, Y: spawn.Y,
							MaxHP: template.MaxHP, AttackDamage: template.BaseAttack, Defense: template.BaseDefense,
							Speed: stats.Speed, Radius: stats.Radius,
							AggroRadius: stats.AggroRadius, LeashRadius: stats.LeashRadius,
							AttackRange: stats.AttackRange, PatrolRadius: stats.PatrolRadius,
							AttackCooldownTicks: stats.AttackCooldownTicks, RespawnTicks: stats.RespawnTicks,
							SplashRadius: stats.SplashRadius, SplashDamage: stats.SplashDamage,
							Passive: loadedMap.Ruleset == "safe",
						})
					}
				}
				portals := make([]realtimeworld.Portal, 0, len(loadedMap.Portals))
				for _, portal := range loadedMap.Portals {
					side, onEdge := edgeSide(portal.X, portal.Y, loadedMap.Width, loadedMap.Height)
					if !onEdge {
						continue
					}
					portals = append(portals, realtimeworld.Portal{
						Side:         side,
						TargetZoneID: portal.TargetZoneID,
						TargetX:      portal.TargetX,
						TargetY:      portal.TargetY,
					})
				}
				artifact = realtimeworld.ZoneMap{
					Collision: loadedMap.Collision(), Spawns: spawns, Monsters: monsters, Portals: portals,
					PvP: loadedMap.Ruleset == "pvp", Safe: loadedMap.Ruleset == "safe",
					Hash: loadedMap.Hash, SafeSpawn: loadedMap.SafeSpawn,
					LegacyOriginX: loadedMap.LegacyOriginX, LegacyOriginY: loadedMap.LegacyOriginY, LegacyScale: loadedMap.LegacyScale,
					Width: loadedMap.Width, Height: loadedMap.Height,
				}
				mapArtifacts[key.MapID] = artifact
				logger.S.Infof("Realtime map %q loaded (%dx%d, %d monsters, hash %s, %d instance(s))", key.MapID, loadedMap.Width, loadedMap.Height, len(monsters), loadedMap.Hash, instances)
			}
			zoneMaps[key] = artifact
		}
		realtimeStateStore = persistence.NewService(db.DB)
		var manager *realtimeworld.Manager
		var managerErr error
		if mapLoadErr != nil {
			managerErr = mapLoadErr
		} else {
			manager, managerErr = realtimeworld.NewLazyManagerWithMaps(zoneKeys, sim.DefaultQueueSize, realtimeConfig.MaxPlayersPerZone, zoneMaps)
		}
		if managerErr == nil {
			manager.SetSnapshotObserver(func(snapshotCtx context.Context, snapshot sim.Snapshot) {
				persistCtx, cancel := context.WithTimeout(snapshotCtx, 2*time.Second)
				defer cancel()
				if err := handlers.PersistRealtimeCombatEvents(persistCtx, snapshot); err != nil {
					logger.S.Warnf("Realtime combat history persistence failed for %v: %v", snapshot.ZoneKey, err)
				}
			})
			inventorySvc := handlers.InventorySvc
			if inventorySvc == nil {
				inventorySvc = services.NewInventoryService(db.DB)
			}
			lootSvc := realloot.NewService(db.DB, inventorySvc)
			manager.SetRetryingDeathHandler(func(callCtx context.Context, death realtimeworld.DeathEvent) error {
				if handled, combatErr := handlers.HandleRealtimeCombatInstanceDeath(callCtx, death); combatErr != nil {
					logger.S.Errorf("Private realtime combat settlement failed for %s: %v", death.TargetID, combatErr)
					return combatErr
				} else if handled {
					return nil
				}
				if death.TargetKind == string(sim.EntityPlayer) {
					if death.ZoneKey.MapID == "aperture_tribulation" {
						if err := handlers.HandleRealtimeTribulationPlayerDeath(callCtx, death); err != nil {
							logger.S.Errorf("Realtime tribulation player death handling failed for %s: %v", death.TargetID, err)
							return err
						}
					} else {
						if err := handlers.HandleRealtimePlayerDeath(callCtx, death); err != nil {
							logger.S.Errorf("Realtime player death handling failed for %s: %v", death.TargetID, err)
							return err
						}
					}
					return nil
				}
				if death.ZoneKey.MapID == "aperture_tribulation" {
					if err := handlers.HandleRealtimeTribulationDeath(callCtx, death); err != nil {
						logger.S.Errorf("Realtime tribulation death handling failed for %s: %v", death.TargetID, err)
						return err
					}
				}
				if strings.HasPrefix(death.PersistentID, "ascension:") {
					rawID := strings.TrimPrefix(death.PersistentID, "ascension:")
					if pID, err := uuid.Parse(rawID); err == nil {
						if err := handlers.CompleteAscensionCalamityForPlayer(callCtx, pID); err != nil {
							logger.S.Errorf("Complete ascension calamity failed for player %s: %v", pID, err)
							return err
						}
					}
				}
				if death.TemplateID == "practice_dummy" {
					return nil
				}
				if death.IsNode {
					return nil
				}
				recipient, parseErr := uuid.Parse(string(death.SourceID))
				if parseErr != nil {
					return nil
				}
				result, grantErr := lootSvc.GrantKill(callCtx, realloot.GrantKey{
					ZoneID: death.ZoneKey.MapID, DeathID: fmt.Sprintf("%s:%d:%s", death.Generation, death.EventID, death.TargetID),
					RecipientID: recipient, TemplateID: death.TemplateID, MonsterRank: death.MonsterRank, RewardVersion: realloot.RewardVersion,
				})
				if grantErr != nil {
					logger.S.Errorf("Realtime loot grant failed for %s: %v", death.TargetID, grantErr)
					return grantErr
				}
				monsterName := death.TemplateID
				if template, ok := models.MonsterRegistry[death.TemplateID]; ok && template.Name != "" {
					monsterName = template.Name
				}
				if historyErr := handlers.RecordRealtimeMonsterKill(callCtx, recipient, death, monsterName); historyErr != nil {
					logger.S.Errorf("Realtime monster combat history persistence failed for %s: %v", death.TargetID, historyErr)
					return historyErr
				}
				if !result.AlreadyGranted {
					logger.S.Infof("Realtime loot granted: recipient=%s template=%s rewards=%d", recipient, death.TemplateID, len(result.Rewards))
				}
				if !result.AlreadyGranted && len(result.Rewards) > 0 {
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
				return nil
			})
			managerErr = manager.Start(ctx)
		}
		if managerErr != nil {
			realtimeStartupError = managerErr.Error()
			logger.S.Errorf("Realtime world manager not started; admission remains disabled: %v", managerErr)
		} else {
			realtimeStartupError = ""
			realtimeWorldManager = manager
			logger.S.Infof("Realtime world manager started with %d public/private zones across %d checked maps (lazy worlds, 5m idle eviction)", len(zoneKeys), len(mapArtifacts))
			if tribulationMapLoadErr != nil {
				logger.S.Errorf("Realtime tribulation event map unavailable: %v", tribulationMapLoadErr)
			}
		}
	}
	handlers.SetRealtimeRuntimeError(realtimeStartupError)
	if realtimeWorldManager != nil {
		middleware.SetRealtimeOwnershipChecker(func(playerID uuid.UUID) bool {
			return realtimeWorldManager.OwnsCharacter(sim.CharacterID(playerID.String()))
		})
		middleware.SetRealtimeVitalsBridge(
			func(playerID uuid.UUID) (hp, maxHP float64, ok bool) {
				return realtimeWorldManager.PlayerVitals(sim.CharacterID(playerID.String()))
			},
			func(playerID uuid.UUID, delta float64) bool {
				return realtimeWorldManager.ApplyHPDelta(sim.CharacterID(playerID.String()), delta) == nil
			},
		)
		handlers.SetAscensionWorldSpawner(func(ctx context.Context, playerID uuid.UUID, zoneID string, x, y float64, hp, atk, def float64) error {
			key := sim.ZoneKey{MapID: zoneID}
			if liveKey, livePos, ok := realtimeWorldManager.PlayerPosition(sim.CharacterID(playerID.String())); ok {
				key = liveKey
				x = livePos.X
				y = livePos.Y
			}
			spawnPos, _ := realtimeWorldManager.SafeSpawn(key, sim.Vec2{X: x + 1.5, Y: y + 1.5}, 0.8)
			spawn := realtimeworld.MonsterSpawn{
				TemplateID:          "monster_earth_calamity_boss",
				Rank:                5,
				Name:                "Địa Tai Bản Yếu",
				X:                   spawnPos.X,
				Y:                   spawnPos.Y,
				MaxHP:               hp,
				CurrentHP:           hp,
				AttackDamage:        atk,
				Defense:             def,
				Speed:               2.2,
				Radius:              0.8,
				AggroRadius:         30.0,
				LeashRadius:         60.0,
				AttackRange:         2.2,
				PatrolRadius:        4.0,
				AttackCooldownTicks: 20,
				PersistentID:        fmt.Sprintf("ascension:%s", playerID.String()),
				FixedTarget:         playerID.String(),
				NoRespawn:           true,
			}
			_, err := realtimeWorldManager.SpawnDynamicMonster(key, spawn)
			return err
		})
	} else {
		middleware.SetRealtimeOwnershipChecker(nil)
		middleware.SetRealtimeVitalsBridge(nil, nil)
		handlers.SetAscensionWorldSpawner(nil)
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
		realtimeAdmission.SetTribulationZoneLoader(func(loadCtx context.Context, playerID uuid.UUID) (sim.ZoneKey, realtimeworld.ZoneMap, bool, error) {
			if tribulationMapLoadErr != nil {
				return sim.ZoneKey{}, realtimeworld.ZoneMap{}, false, tribulationMapLoadErr
			}
			base, ok := realtimeWorldManager.BaseZoneMap("aperture_tribulation")
			if !ok {
				return sim.ZoneKey{}, realtimeworld.ZoneMap{}, false, fmt.Errorf("aperture_tribulation map is not configured")
			}
			eventID, spawns, active, err := handlers.LoadRealtimeTribulationSpawns(loadCtx, playerID)
			if err != nil || !active {
				return sim.ZoneKey{}, realtimeworld.ZoneMap{}, active, err
			}
			base.Monsters = spawns
			// The tribulation encounter is a private, single-player map. Send
			// every monster and node to the client (and every checkpoint) so a
			// kill can never be hidden outside the public AOI radius.
			base.ViewRadius = math.Hypot(float64(base.Width), float64(base.Height)) + 4
			return sim.ZoneKey{MapID: "aperture_tribulation", InstanceID: eventID, RulesetVersion: "v1"}, base, true, nil
		})
		realtimeAdmission.SetCombatInstanceZoneLoader(func(loadCtx context.Context, playerID uuid.UUID, state handlers.RealtimeJoinState) (realtimegateway.CombatInstanceZone, bool, error) {
			const realmPrefix = "realm:"
			if strings.HasPrefix(state.WorldInstanceID, realmPrefix) {
				routeSuffix := strings.TrimPrefix(state.WorldInstanceID, realmPrefix)
				instanceToken, _, _ := strings.Cut(routeSuffix, ":")
				instanceID, parseErr := uuid.Parse(instanceToken)
				if parseErr != nil || state.MapInstanceID != instanceID.String() {
					return realtimegateway.CombatInstanceZone{}, false, nil
				}
				var zoneID, pvpMode string
				var rawRules []byte
				var duration int
				loadErr := db.DB.QueryRowContext(loadCtx, `
					SELECT s.zone_id,COALESCE(t.rules,'{}'::jsonb),t.duration_minutes
					FROM realtime_combat_sessions s
					JOIN realtime_combat_session_participants p ON p.session_id=s.id AND p.player_id=$2 AND p.status='active'
					JOIN shared_spatial_instances i ON i.id=s.id AND i.status='ACTIVE' AND i.expires_at>NOW()
					JOIN shared_spatial_instance_members m ON m.instance_id=i.id AND m.player_id=p.player_id AND m.exited_at IS NULL
					JOIN instance_templates t ON t.id=i.template_id
					WHERE s.id=$1 AND s.mode='secret_realm' AND s.status='active'
				`, instanceID, playerID).Scan(&zoneID, &rawRules, &duration)
				if errors.Is(loadErr, sql.ErrNoRows) {
					return realtimegateway.CombatInstanceZone{}, false, nil
				}
				if loadErr != nil {
					return realtimegateway.CombatInstanceZone{}, false, loadErr
				}
				var rawRuleSet struct {
					PvPMode string `json:"pvp_mode"`
				}
				if err := json.Unmarshal(rawRules, &rawRuleSet); err != nil {
					return realtimegateway.CombatInstanceZone{}, false, err
				}
				pvpMode = strings.ToUpper(strings.TrimSpace(rawRuleSet.PvPMode))
				base, found := realtimeWorldManager.BaseZoneMap(zoneID)
				if !found {
					return realtimegateway.CombatInstanceZone{}, false, fmt.Errorf("spatial realm map %q is not configured", zoneID)
				}
				base.Safe = pvpMode == "" || pvpMode == "SAFE"
				base.PvP = pvpMode == "FREE_PK" || pvpMode == "FACTION_PK"
				// Realm exits are explicit, session-aware actions. Authored world
				// portals would bypass membership cleanup and leak the player out.
				base.Portals = nil
				partyIDOverride := ""
				if pvpMode == "FACTION_PK" {
					var sectID string
					if err := db.DB.QueryRowContext(loadCtx, `SELECT COALESCE(sect_id,'') FROM sect_members WHERE player_id=$1 LIMIT 1`, playerID).Scan(&sectID); err != nil && !errors.Is(err, sql.ErrNoRows) {
						return realtimegateway.CombatInstanceZone{}, false, err
					}
					if sectID != "" {
						partyIDOverride = "sect:" + sectID
					} else {
						partyIDOverride = "unaffiliated:" + playerID.String()
					}
				}
				return realtimegateway.CombatInstanceZone{
					Key:             sim.ZoneKey{MapID: zoneID, InstanceID: instanceID.String(), RulesetVersion: "v1"},
					Map:             base,
					PartyIDOverride: partyIDOverride,
				}, true, nil
			}
			const prefix = "combat:"
			if !strings.HasPrefix(state.WorldInstanceID, prefix) {
				return realtimegateway.CombatInstanceZone{}, false, nil
			}
			instanceID, parseErr := uuid.Parse(strings.TrimPrefix(state.WorldInstanceID, prefix))
			if parseErr != nil {
				return realtimegateway.CombatInstanceZone{}, false, nil
			}
			var mode, zoneID string
			var payload, sessionPayload []byte
			loadErr := db.DB.QueryRowContext(loadCtx, `
				SELECT mode, zone_id, payload FROM realtime_combat_instances
				WHERE id=$1 AND player_id=$2 AND mode='training' AND status='active'
			`, instanceID, playerID).Scan(&mode, &zoneID, &payload)
			if errors.Is(loadErr, sql.ErrNoRows) {
				loadErr = db.DB.QueryRowContext(loadCtx, `
					SELECT s.mode, s.zone_id, p.payload, s.payload
					FROM realtime_combat_sessions s
					JOIN realtime_combat_session_participants p ON p.session_id=s.id
					WHERE s.id=$1 AND p.player_id=$2 AND s.status='active' AND p.status='active'
				`, instanceID, playerID).Scan(&mode, &zoneID, &payload, &sessionPayload)
			}
			if loadErr != nil {
				if errors.Is(loadErr, sql.ErrNoRows) {
					return realtimegateway.CombatInstanceZone{}, false, nil
				}
				return realtimegateway.CombatInstanceZone{}, false, loadErr
			}
			base, found := realtimeWorldManager.BaseZoneMap(zoneID)
			if !found {
				return realtimegateway.CombatInstanceZone{}, false, fmt.Errorf("combat map %q is not configured", zoneID)
			}
			if mode == "training" {
				var config handlers.PracticeRealtimeConfig
				if err := json.Unmarshal(payload, &config); err != nil {
					return realtimegateway.CombatInstanceZone{}, false, fmt.Errorf("decode practice config: %w", err)
				}
				base.Monsters = append(base.Monsters, realtimeworld.MonsterSpawn{
					TemplateID: "practice_dummy", Name: "Hình Nhân Gỗ", X: float64(base.Width) * 0.55, Y: float64(base.Height) * 0.5,
					MaxHP: config.DummyHP, CurrentHP: config.DummyHP, Shield: config.DummyShield, AttackDamage: config.DummyAtk, Defense: config.DummyDef,
					Speed: config.DummySpeed, Radius: 0.55, AggroRadius: 24, LeashRadius: 30, AttackRange: 1.5,
					AttackCooldownTicks: 20, NoRespawn: true,
				})
			} else if mode == "arena" || mode == "tournament" || mode == "sect_war" {
				var vitals handlers.RealtimeCombatInstanceVitals
				if err := json.Unmarshal(payload, &vitals); err != nil {
					return realtimegateway.CombatInstanceZone{}, false, fmt.Errorf("decode combat instance vitals: %w", err)
				}
				if math.IsNaN(vitals.CurrentHP) || math.IsInf(vitals.CurrentHP, 0) || vitals.CurrentHP <= 0 || vitals.CurrentHP > state.BaseHP ||
					math.IsNaN(vitals.CurrentMana) || math.IsInf(vitals.CurrentMana, 0) || vitals.CurrentMana < 0 || vitals.CurrentMana > state.MaxMana ||
					math.IsNaN(vitals.X) || math.IsInf(vitals.X, 0) || math.IsNaN(vitals.Y) || math.IsInf(vitals.Y, 0) {
					return realtimegateway.CombatInstanceZone{}, false, fmt.Errorf("invalid combat instance participant state")
				}
				state.HP = vitals.CurrentHP
				state.CurrentMana = vitals.CurrentMana
				if vitals.CombatTeamID != "" {
					state.PartyID = vitals.CombatTeamID
				}
				base.Monsters = nil
				base.Portals = nil
				base.Safe = false
				base.PvP = true
				if mode == "sect_war" {
					var config struct {
						GuardianTemplateID string  `json:"guardian_template_id"`
						GuardianName       string  `json:"guardian_name"`
						GuardianRank       int     `json:"guardian_rank"`
						GuardianHP         float64 `json:"guardian_hp"`
						GuardianAttack     float64 `json:"guardian_attack"`
						GuardianDefense    float64 `json:"guardian_defense"`
						GuardianDead       bool    `json:"guardian_dead"`
					}
					if err := json.Unmarshal(sessionPayload, &config); err != nil || config.GuardianTemplateID == "" || config.GuardianHP <= 0 {
						return realtimegateway.CombatInstanceZone{}, false, fmt.Errorf("invalid sect war guardian configuration")
					}
					if !config.GuardianDead {
						stats := realtimeai.DefaultStats("BOSS")
						base.Monsters = append(base.Monsters, realtimeworld.MonsterSpawn{
							TemplateID: config.GuardianTemplateID, Name: config.GuardianName,
							X: float64(base.Width) * 0.70, Y: float64(base.Height) * 0.5,
							MaxHP: config.GuardianHP, CurrentHP: config.GuardianHP,
							AttackDamage: config.GuardianAttack, Defense: config.GuardianDefense,
							Speed: stats.Speed, Radius: stats.Radius, AggroRadius: 30,
							LeashRadius: 36, AttackRange: stats.AttackRange,
							AttackCooldownTicks: stats.AttackCooldownTicks, NoRespawn: true,
						})
					}
				}
				spawn := sim.Vec2{X: vitals.X, Y: vitals.Y}
				return realtimegateway.CombatInstanceZone{
					Key: sim.ZoneKey{MapID: zoneID, InstanceID: instanceID.String(), RulesetVersion: "v1"},
					Map: base, OverrideState: &state, SpawnPosition: &spawn,
					Isolated: true, PersistCombatState: true,
				}, true, nil
			} else if mode == "ascension" {
				var config struct {
					TemplateID   string  `json:"template_id"`
					Name         string  `json:"name"`
					MaxHP        float64 `json:"max_hp"`
					AttackDamage float64 `json:"attack_damage"`
					Defense      float64 `json:"defense"`
					CurrentHP    float64 `json:"current_hp"`
					CurrentMana  float64 `json:"current_mana"`
					X            float64 `json:"x"`
					Y            float64 `json:"y"`
				}
				if err := json.Unmarshal(payload, &config); err != nil || config.TemplateID == "" || config.MaxHP <= 0 || config.CurrentHP <= 0 {
					return realtimegateway.CombatInstanceZone{}, false, fmt.Errorf("invalid ascension encounter config")
				}
				if math.IsNaN(config.X) || math.IsInf(config.X, 0) || math.IsNaN(config.Y) || math.IsInf(config.Y, 0) ||
					config.X < 0 || config.Y < 0 || config.X > float64(base.Width) || config.Y > float64(base.Height) {
					return realtimegateway.CombatInstanceZone{}, false, fmt.Errorf("invalid ascension participant position")
				}
				state.HP = math.Min(state.BaseHP, config.CurrentHP)
				state.CurrentMana = math.Max(0, math.Min(state.MaxMana, config.CurrentMana))
				base.Monsters = nil
				base.Portals = nil
				base.Safe = false
				base.PvP = false
				template := models.MonsterRegistry[config.TemplateID]
				stats := realtimeai.DefaultStats(template.Type)
				base.Monsters = append(base.Monsters, realtimeworld.MonsterSpawn{
					TemplateID: config.TemplateID, Name: config.Name,
					X: float64(base.Width) * 0.65, Y: float64(base.Height) * 0.5,
					MaxHP: config.MaxHP, CurrentHP: config.MaxHP,
					AttackDamage: config.AttackDamage, Defense: config.Defense,
					Speed: stats.Speed, Radius: stats.Radius, AggroRadius: 28,
					LeashRadius: 32, AttackRange: stats.AttackRange,
					AttackCooldownTicks: stats.AttackCooldownTicks, NoRespawn: true,
				})
				spawn := sim.Vec2{X: config.X, Y: config.Y}
				return realtimegateway.CombatInstanceZone{
					Key: sim.ZoneKey{MapID: zoneID, InstanceID: instanceID.String(), RulesetVersion: "v1"},
					Map: base, OverrideState: &state, SpawnPosition: &spawn, Isolated: true, PersistCombatState: true,
				}, true, nil
			} else {
				return realtimegateway.CombatInstanceZone{}, false, fmt.Errorf("unsupported realtime combat mode %q", mode)
			}
			return realtimegateway.CombatInstanceZone{
				Key: sim.ZoneKey{MapID: zoneID, InstanceID: instanceID.String(), RulesetVersion: "v1"},
				Map: base, Isolated: true,
			}, true, nil
		})
		realtimeAdmission.SetApertureStateSink(func(sinkCtx context.Context, playerID uuid.UUID, snapshot sim.Snapshot) error {
			return handlers.PersistRealtimeTribulationSnapshot(sinkCtx, playerID, snapshot)
		})
		realtimeAdmission.SetCombatInstanceStateSink(func(sinkCtx context.Context, playerID uuid.UUID, key sim.ZoneKey, entity sim.Entity) error {
			sessionID, parseErr := uuid.Parse(key.InstanceID)
			if parseErr != nil {
				return fmt.Errorf("parse combat instance session id: %w", parseErr)
			}
			result, saveErr := db.DB.ExecContext(sinkCtx, `
				UPDATE realtime_combat_session_participants
				SET payload = payload || jsonb_build_object('current_hp',$1,'current_mana',$2,'x',$3,'y',$4)
				WHERE session_id=$5 AND player_id=$6 AND status='active'
			`, entity.Vitals.HP, entity.Combat.Mana, entity.Transform.Position.X, entity.Transform.Position.Y, sessionID, playerID)
			if saveErr != nil {
				return fmt.Errorf("checkpoint combat instance participant: %w", saveErr)
			}
			if changed, rowsErr := result.RowsAffected(); rowsErr != nil || changed != 1 {
				if rowsErr == nil && changed == 0 {
					var status string
					statusErr := db.DB.QueryRowContext(sinkCtx, `
						SELECT status FROM realtime_combat_session_participants
						WHERE session_id=$1 AND player_id=$2
					`, sessionID, playerID).Scan(&status)
					if statusErr == nil && status != "active" {
						return nil // Completed virtual sessions intentionally discard later socket flushes.
					}
				}
				return fmt.Errorf("combat instance participant checkpoint matched %d rows: %v", changed, rowsErr)
			}
			return nil
		})
		handlers.SetRealtimeStatsProvider(func() any { return realtimeWorldManager.Stats() })
		worker.RunEvery(ctx, 10*time.Second, "realtime-checkpoint", func(context.Context) {
			realtimeAdmission.CheckpointActive()
		})
		worker.RunEvery(ctx, 5*time.Second, "sect-war-disconnect-settlement", handlers.SweepDisconnectedSectWars)
		worker.RunEvery(ctx, 5*time.Second, "arena-disconnect-settlement", handlers.SweepDisconnectedArenas)
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

// edgeSide classifies an authored portal position onto one map border. Region
// map art always places side portals 2 units from the outer wall.
func edgeSide(x, y float64, width, height int) (realtimeworld.EdgeSide, bool) {
	switch {
	case x <= 2:
		return realtimeworld.EdgeWest, true
	case x >= float64(width)-2:
		return realtimeworld.EdgeEast, true
	case y <= 2:
		return realtimeworld.EdgeNorth, true
	case y >= float64(height)-2:
		return realtimeworld.EdgeSouth, true
	default:
		return "", false
	}
}

// expandRealtimeZoneIDs turns the public map wildcard into the shipped map
// catalogue. Region maps are one JSON file per KHU_VUC/SAFE_ZONE; loading the
// whole directory keeps every authored edge target available.
func expandRealtimeZoneIDs(configured map[string]struct{}, mapDir, mapVersion, contentVersion string) map[string]struct{} {
	_, wildcardStar := configured["*"]
	_, wildcardAll := configured["all"]
	if !wildcardStar && !wildcardAll {
		return configured
	}
	directory := resolveRealtimeMapDir(mapDir)
	entries, err := os.ReadDir(directory)
	if err != nil {
		logger.S.Errorf("Could not read public map directory %s: %v", directory, err)
		return configured
	}
	expanded := make(map[string]struct{}, len(entries))
	ids := make([]string, 0, len(entries))
	skipped := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		// Preview/Alpha/legacy artifacts share the directory but not the
		// rollout version. Loading one of them would fail startup, so `all`
		// expands only the maps matching the configured versions.
		header, err := readRealtimeMapHeader(filepath.Join(directory, entry.Name()))
		if err != nil {
			logger.S.Warnf("Public map catalogue skipped %s: %v", entry.Name(), err)
			skipped++
			continue
		}
		if (mapVersion != "" && header.Version != mapVersion) || (contentVersion != "" && header.ContentVersion != contentVersion) {
			skipped++
			continue
		}
		expanded[id] = struct{}{}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	logger.S.Infof("Public map catalogue loaded %d maps from %s (%d version-mismatched skipped, version=%q content=%q)",
		len(ids), directory, skipped, mapVersion, contentVersion)
	return expanded
}

type realtimeMapHeader struct {
	Version        string `json:"map_version"`
	ContentVersion string `json:"content_version"`
}

func readRealtimeMapHeader(path string) (realtimeMapHeader, error) {
	var header realtimeMapHeader
	raw, err := os.ReadFile(path)
	if err != nil {
		return header, err
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return header, err
	}
	if header.Version == "" {
		return header, errors.New("map_version is empty")
	}
	return header, nil
}

// resolveRealtimeMapDir mirrors resolveRealtimeMapPath for directory scans.
func resolveRealtimeMapDir(mapDir string) string {
	if filepath.IsAbs(mapDir) {
		return mapDir
	}
	if cwd, err := os.Getwd(); err == nil {
		for current := cwd; ; current = filepath.Dir(current) {
			candidate := filepath.Join(current, mapDir)
			if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
				return candidate
			}
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
		}
	}
	return mapDir
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
