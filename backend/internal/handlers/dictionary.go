package handlers

import (
	"crypto/md5"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
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

// WikiRegionLoot is a lightweight representation of region-specific loot rules for the public wiki.
type WikiRegionLoot struct {
	RegionID    string  `json:"region_id"`
	LootTableID string  `json:"loot_table_id"`
	MonsterType string  `json:"monster_type"`
	MinRank     int     `json:"min_rank"`
	MaxRank     int     `json:"max_rank"`
	Chance      float64 `json:"chance"`
}

// DictionaryResponse is the unified catalog the game client hydrates its
// IndexedDB from. Wiki-only sections (loot_tables, regions, monster_spawns, region_loot)
// are served by their dedicated /api/dictionary/* part endpoints instead.
type DictionaryResponse struct {
	Gu            interface{}                  `json:"gu"`
	Items         interface{}                  `json:"items"`
	Monsters      map[string]PublicMonsterView `json:"monsters"`
	Constitutions []gamefx.ConstitutionDef     `json:"constitutions"`
	Races         []gamefx.RaceDef             `json:"races"`
	Statuses      []gamefx.StatusDef           `json:"statuses"`
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
	getCachedDictionaryPartWithError(c, key, target, func() (interface{}, error) {
		return build(), nil
	}, nil)
}

// getCachedDictionaryPartWithError validates cache values before serving them
// and only caches successful builds. This is used for catalogs where an empty
// result usually means the backing query failed or the cache was poisoned.
func getCachedDictionaryPartWithError(c *gin.Context, key string, target interface{}, build func() (interface{}, error), valid func(interface{}) bool) {
	dictCacheMu.RLock()
	entry, found := dictCache[key]
	dictCacheMu.RUnlock()

	if found {
		if err := json.Unmarshal(entry.rawJSON, target); err == nil && (valid == nil || valid(target)) {
			if ifNoneMatch := c.GetHeader("If-None-Match"); ifNoneMatch != "" && ifNoneMatch == entry.etag {
				c.Status(http.StatusNotModified)
				return
			}
			c.Header("ETag", entry.etag)
			c.Header("Cache-Control", "public, max-age=3600, stale-while-revalidate=86400")
			c.Data(http.StatusOK, "application/json; charset=utf-8", entry.rawJSON)
			return
		}
		dictCacheMu.Lock()
		delete(dictCache, key)
		dictCacheMu.Unlock()
		_ = cache.Delete(key)
	}

	var raw []byte
	loaded := false
	if err := cache.GetJSON(key, target); err == nil {
		if valid == nil || valid(target) {
			raw, _ = json.Marshal(target)
			loaded = true
		} else {
			_ = cache.Delete(key)
		}
	}

	if !loaded {
		val, err := build()
		if err != nil {
			logger.S.Errorf("dictionary cache %s rebuild failed: %v", key, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tải danh sách dữ liệu"})
			return
		}
		if valid != nil && !valid(val) {
			logger.S.Errorf("dictionary cache %s rebuild returned invalid data", key)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Dữ liệu hiện chưa sẵn sàng"})
			return
		}
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

func invalidateCachedDictionaryPart(key string) {
	dictCacheMu.Lock()
	delete(dictCache, key)
	dictCacheMu.Unlock()
	_ = cache.Delete(key)
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

// GetRegionLoot returns only the public wiki region loot table rules.
func GetRegionLoot(c *gin.Context) {
	getCachedDictionaryPartWithError(c, "game_dictionary_region_loot", &[]WikiRegionLoot{}, func() (interface{}, error) {
		res := loadAllRegionLoot(c)
		return res, nil
	}, func(val interface{}) bool {
		if list, ok := val.(*[]WikiRegionLoot); ok && list != nil {
			return len(*list) > 0
		}
		if list, ok := val.([]WikiRegionLoot); ok {
			return len(list) > 0
		}
		return false
	})
}

func buildUnifiedDictionary() DictionaryResponse {
	return DictionaryResponse{
		Gu:            flatGuDictionary(),
		Items:         models.ItemRegistry,
		Monsters:      publicMonsterDictionary(),
		Constitutions: gamefx.AllConstitutions(),
		Races:         gamefx.AllRaces(),
		Statuses:      gamefx.AllStatuses(),
	}
}

// WarmDictionaryCache builds the unified catalog once at process start so the
// first player request does not pay build + marshal cost on a cold cache.
func WarmDictionaryCache() {
	val := buildUnifiedDictionary()
	raw, err := json.Marshal(val)
	if err != nil {
		return
	}
	hash := md5.Sum(raw)
	dictCacheMu.Lock()
	dictCache["game_dictionary_unified"] = dictCacheEntry{rawJSON: raw, etag: fmt.Sprintf(`"%x"`, hash)}
	dictCacheMu.Unlock()
}

// GetDictionary returns the unified Gu and Item master catalogs, cached in memory/Redis with ETag support.
func GetDictionary(c *gin.Context) {
	getCachedDictionaryPart(c, "game_dictionary_unified", &DictionaryResponse{}, func() interface{} {
		return buildUnifiedDictionary()
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

// loadRegionLootFromCSV reads RegionLootTables.csv as a fallback when database is empty or unavailable.
func loadRegionLootFromCSV() []WikiRegionLoot {
	path := csvPath("RegionLootTables.csv")
	f, err := os.Open(path)
	if err != nil {
		logger.S.Warnf("dictionary: failed to open RegionLootTables.csv fallback: %v", err)
		return []WikiRegionLoot{}
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return []WikiRegionLoot{}
	}

	col := func(name string) int {
		for i, h := range header {
			if strings.EqualFold(strings.TrimSpace(h), name) {
				return i
			}
		}
		return -1
	}

	regionIdx, lootIdx, typeIdx := col("region_id"), col("loot_table_id"), col("monster_type")
	minRankIdx, maxRankIdx, chanceIdx, activeIdx := col("min_rank"), col("max_rank"), col("chance"), col("is_active")
	if regionIdx < 0 || lootIdx < 0 || typeIdx < 0 {
		return []WikiRegionLoot{}
	}

	get := func(row []string, idx int) string {
		if idx < 0 || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	var list []WikiRegionLoot
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(row) == 0 {
			continue
		}

		active := true
		if activeIdx >= 0 {
			av := strings.ToLower(get(row, activeIdx))
			if av == "false" || av == "0" || av == "no" {
				active = false
			}
		}
		if !active {
			continue
		}

		regionID, lootID, monsterType := get(row, regionIdx), get(row, lootIdx), get(row, typeIdx)
		if regionID == "" || lootID == "" || monsterType == "" {
			continue
		}

		minRank, maxRank, chance := 1, 9, 1.0
		if minRankIdx >= 0 {
			if v := get(row, minRankIdx); v != "" {
				fmt.Sscanf(v, "%d", &minRank)
			}
		}
		if maxRankIdx >= 0 {
			if v := get(row, maxRankIdx); v != "" {
				fmt.Sscanf(v, "%d", &maxRank)
			}
		}
		if chanceIdx >= 0 {
			if v := get(row, chanceIdx); v != "" {
				fmt.Sscanf(v, "%f", &chance)
			}
		}

		list = append(list, WikiRegionLoot{
			RegionID:    regionID,
			LootTableID: lootID,
			MonsterType: monsterType,
			MinRank:     minRank,
			MaxRank:     maxRank,
			Chance:      chance,
		})
	}
	return list
}

// loadAllRegionLoot queries every active region loot rule from the database,
// falling back to RegionLootTables.csv if the database table is empty or unpopulated.
func loadAllRegionLoot(c *gin.Context) []WikiRegionLoot {
	var list []WikiRegionLoot
	if db.DB != nil {
		rows, err := db.DB.QueryContext(c.Request.Context(),
			`SELECT region_id, loot_table_id, monster_type, min_rank, max_rank, chance
			 FROM region_loot_tables
			 WHERE is_active = TRUE
			 ORDER BY region_id, monster_type, loot_table_id`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var r WikiRegionLoot
				if err := rows.Scan(&r.RegionID, &r.LootTableID, &r.MonsterType, &r.MinRank, &r.MaxRank, &r.Chance); err != nil {
					logger.S.Warnf("dictionary: failed to scan region loot row: %v", err)
					continue
				}
				list = append(list, r)
			}
			if err := rows.Err(); err != nil {
				logger.S.Warnf("dictionary: region loot row iteration error: %v", err)
			}
		} else {
			logger.S.Warnf("dictionary: failed to load region loot: %v", err)
		}
	}

	if len(list) == 0 {
		list = loadRegionLootFromCSV()
	}
	if list == nil {
		list = []WikiRegionLoot{}
	}
	return list
}

