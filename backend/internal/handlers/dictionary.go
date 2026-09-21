package handlers

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vippro67/GuWorld/backend/internal/cache"
	"github.com/vippro67/GuWorld/backend/internal/db"
	"github.com/vippro67/GuWorld/backend/internal/gamefx"
	"github.com/vippro67/GuWorld/backend/internal/models"
	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// WikiRegion is a lightweight region view for the public dictionary/wiki.
// It omits internal geometry (min_x, max_x, …) and timestamps, keeping only
// fields that wiki consumers need: identity, rank range, hierarchy, and the
// pool_weights that tell which resources / wild Gu / monsters appear there.
type WikiRegion struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	MinRank     int              `json:"min_rank"`
	MaxRank     int              `json:"max_rank"`
	ParentID    string           `json:"parent_id,omitempty"`
	ZoneType    string           `json:"zone_type"`
	MinX        int              `json:"min_x,omitempty"`
	MinY        int              `json:"min_y,omitempty"`
	MaxX        int              `json:"max_x,omitempty"`
	MaxY        int              `json:"max_y,omitempty"`
	PoolWeights models.ZonePools `json:"pool_weights"`
}

// WikiMonsterSpawn is a lightweight spawn rule for the public dictionary/wiki.
// It maps a monster template to a region with spawn chance and rank range.
type WikiMonsterSpawn struct {
	RegionID    string  `json:"region_id"`
	MonsterID   string  `json:"monster_id"`
	Chance      float64 `json:"chance"`
	MinRank     int     `json:"min_rank"`
	MaxRank     int     `json:"max_rank"`
	IsBossSpawn bool    `json:"is_boss_spawn"`
}

// DictionaryResponse is the unified catalog the game client hydrates its
// IndexedDB from. Wiki-only sections (loot_tables, regions, monster_spawns)
// are served by their dedicated /api/dictionary/* part endpoints instead.
type DictionaryResponse struct {
	Gu            interface{}              `json:"gu"`
	Items         interface{}              `json:"items"`
	Monsters      map[string]PublicMonsterView `json:"monsters"`
	Constitutions []gamefx.ConstitutionDef `json:"constitutions"`
	Races         []gamefx.RaceDef         `json:"races"`
	Statuses      []gamefx.StatusDef       `json:"statuses"`
}

// PublicMonsterView is the client-facing projection of a MonsterTemplate.
// Internal combat/loot data (gu_inventory, gu_pool, item_inventory, dao_marks,
// path_masteries, ai_profile) stays server-side; the admin panel uses
// /api/admin/monsters for full details.
type PublicMonsterView struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Type             string  `json:"type"`
	MaxHP            float64 `json:"max_hp"`
	BaseAttack       float64 `json:"base_attack"`
	BaseDefense      float64 `json:"base_defense"`
	LootTableID      string  `json:"loot_table_id"`
	Description      string  `json:"description"`
	IsCapturable     bool    `json:"is_capturable"`
	CaptureChance    float64 `json:"capture_chance"`
	PetHungerPerHour float64 `json:"pet_hunger_per_hour"`
}

// publicMonsterDictionary projects the full monster registry down to the
// fields the game client and wiki actually read.
func publicMonsterDictionary() map[string]PublicMonsterView {
	out := make(map[string]PublicMonsterView, len(models.MonsterRegistry))
	for id, m := range models.MonsterRegistry {
		out[id] = PublicMonsterView{
			ID:               m.ID,
			Name:             m.Name,
			Type:             m.Type,
			MaxHP:            m.MaxHP,
			BaseAttack:       m.BaseAttack,
			BaseDefense:      m.BaseDefense,
			LootTableID:      m.LootTableID,
			Description:      m.Description,
			IsCapturable:     m.IsCapturable,
			CaptureChance:    m.CaptureChance,
			PetHungerPerHour: m.PetHungerPerHour,
		}
	}
	return out
}

func flatGuDictionary() []models.GuDictionaryItem {
	flatGu := make([]models.GuDictionaryItem, 0)
	for _, ranks := range models.GuRegistry {
		for _, item := range ranks {
			flatGu = append(flatGu, item)
		}
	}
	return flatGu
}

type dictCacheEntry struct {
	rawJSON []byte
	etag    string
}

var (
	dictCacheMu sync.RWMutex
	dictCache   = make(map[string]dictCacheEntry)
)

// InvalidateDictionaryMemoryCache clears the local in-memory cache when registries change.
func InvalidateDictionaryMemoryCache() {
	dictCacheMu.Lock()
	dictCache = make(map[string]dictCacheEntry)
	dictCacheMu.Unlock()
}

func getCachedDictionaryPart(c *gin.Context, key string, target interface{}, build func() interface{}) {
	dictCacheMu.RLock()
	entry, found := dictCache[key]
	dictCacheMu.RUnlock()

	if found {
		if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" && ifNoneMatch == entry.etag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Header("ETag", entry.etag)
		c.Header("Cache-Control", "public, max-age=3600, stale-while-revalidate=86400")
		c.Data(http.StatusOK, "application/json; charset=utf-8", entry.rawJSON)
		return
	}

	var raw []byte
	if err := cache.GetJSON(key, target); err == nil {
		raw, _ = json.Marshal(target)
	} else {
		val := build()
		_ = cache.SetJSON(key, val, 24*time.Hour)
		raw, _ = json.Marshal(val)
	}

	hash := md5.Sum(raw)
	etag := fmt.Sprintf(`"%x"`, hash)

	dictCacheMu.Lock()
	dictCache[key] = dictCacheEntry{
		rawJSON: raw,
		etag:    etag,
	}
	dictCacheMu.Unlock()

	if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" && ifNoneMatch == etag {
		c.Status(http.StatusNotModified)
		return
	}

	c.Header("ETag", etag)
	c.Header("Cache-Control", "public, max-age=3600, stale-while-revalidate=86400")
	c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
}

// GetGuDictionary returns only the Gu catalog used by gameplay screens.
func GetGuDictionary(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_gus", &[]models.GuDictionaryItem{}, func() interface{} {
		return flatGuDictionary()
	})
}

// GetItemDictionary returns only the item catalog used by gameplay screens.
func GetItemDictionary(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_items", &map[string]models.ItemSpec{}, func() interface{} {
		return models.ItemRegistry
	})
}

// GetMonsterDictionary returns only the monster catalog used by gameplay screens
// (projected to client-facing fields — no internal combat/loot loadouts).
func GetMonsterDictionary(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_monsters", &map[string]PublicMonsterView{}, func() interface{} {
		return publicMonsterDictionary()
	})
}

// GetConstitutionDictionary returns only the constitution catalog.
func GetConstitutionDictionary(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_constitutions", &[]gamefx.ConstitutionDef{}, func() interface{} {
		return gamefx.AllConstitutions()
	})
}

// GetRaceDictionary returns only the race catalog.
func GetRaceDictionary(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_races", &[]gamefx.RaceDef{}, func() interface{} {
		return gamefx.AllRaces()
	})
}

// GetStatusDictionary returns only the status catalog.
func GetStatusDictionary(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_statuses", &[]gamefx.StatusDef{}, func() interface{} {
		return gamefx.AllStatuses()
	})
}

// GetLootTables returns only the loot table catalog used by the public wiki.
func GetLootTables(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_loot_tables", &[]models.LootTable{}, func() interface{} {
		return loadAllLootTables(c)
	})
}

// GetDictionaryRegions returns only the public wiki region catalog.
func GetDictionaryRegions(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_regions", &[]WikiRegion{}, func() interface{} {
		return buildWikiRegions()
	})
}

// GetMonsterSpawns returns only the public wiki monster spawn catalog.
func GetMonsterSpawns(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_monster_spawns", &[]WikiMonsterSpawn{}, func() interface{} {
		return buildWikiMonsterSpawns()
	})
}

// GetDictionary returns the unified Gu and Item master catalogs, cached in memory/Redis with ETag support.
func GetDictionary(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_unified", &DictionaryResponse{}, func() interface{} {
		flatGu := flatGuDictionary()
		logger.S.Infof("GetDictionary: returning %d gu, %d items, %d monsters", len(flatGu), len(models.ItemRegistry), len(models.MonsterRegistry))
		return DictionaryResponse{
			Gu:            flatGu,
			Items:         models.ItemRegistry,
			Monsters:      publicMonsterDictionary(),
			Constitutions: gamefx.AllConstitutions(),
			Races:         gamefx.AllRaces(),
			Statuses:      gamefx.AllStatuses(),
		}
	})
}

// loadAllLootTables queries every loot table from the database and returns
// them as parsed models.LootTable (with entries unmarshalled).
func loadAllLootTables(c *gin.Context) []models.LootTable {
	rows, err := db.DB.QueryContext(c.Request.Context(),
		`SELECT id, name, COALESCE(entries,'[]'::jsonb) FROM loot_tables ORDER BY name`)
	if err != nil {
		logger.S.Warnf("dictionary: failed to load loot tables: %v", err)
		return []models.LootTable{}
	}
	defer rows.Close()

	var tables []models.LootTable
	for rows.Next() {
		var lt models.LootTable
		var entriesRaw json.RawMessage
		if err := rows.Scan(&lt.ID, &lt.Name, &entriesRaw); err != nil {
			logger.S.Warnf("dictionary: failed to scan loot table row: %v", err)
			continue
		}
		if len(entriesRaw) > 0 {
			_ = json.Unmarshal(entriesRaw, &lt.Entries)
		}
		if lt.Entries == nil {
			lt.Entries = []models.LootEntry{}
		}
		tables = append(tables, lt)
	}
	if err := rows.Err(); err != nil {
		logger.S.Warnf("dictionary: loot tables row iteration error: %v", err)
	}
	if tables == nil {
		tables = []models.LootTable{}
	}
	return tables
}

// buildWikiRegions reads from the in-memory zone registry (loaded at startup)
// and returns lightweight WikiRegion structs suitable for the public wiki.
func buildWikiRegions() []WikiRegion {
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()

	regions := make([]WikiRegion, 0, len(zoneReg.zones))
	for _, z := range zoneReg.zones {
		regions = append(regions, WikiRegion{
			ID:          z.ID,
			Name:        z.Name,
			Description: z.Description,
			MinRank:     z.MinRank,
			MaxRank:     z.MaxRank,
			ParentID:    z.ParentID,
			ZoneType:    z.ZoneType,
			MinX:        z.MinX,
			MinY:        z.MinY,
			MaxX:        z.MaxX,
			MaxY:        z.MaxY,
			PoolWeights: z.PoolWeights,
		})
	}
	return regions
}

// buildWikiMonsterSpawns reads from the in-memory zone registry spawn pools
// and returns lightweight WikiMonsterSpawn structs for the public wiki.
func buildWikiMonsterSpawns() []WikiMonsterSpawn {
	ensureZonesLoaded()
	zoneReg.mu.RLock()
	defer zoneReg.mu.RUnlock()

	var spawns []WikiMonsterSpawn
	for _, pool := range zoneReg.spawnPools {
		for _, s := range pool {
			spawns = append(spawns, WikiMonsterSpawn{
				RegionID:    s.RegionID,
				MonsterID:   s.MonsterID,
				Chance:      s.Chance,
				MinRank:     s.MinRank,
				MaxRank:     s.MaxRank,
				IsBossSpawn: s.IsBossSpawn,
			})
		}
	}
	if spawns == nil {
		spawns = []WikiMonsterSpawn{}
	}
	return spawns
}
