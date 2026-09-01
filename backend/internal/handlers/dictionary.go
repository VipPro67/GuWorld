package handlers

import (
	"encoding/json"
	"net/http"
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

type DictionaryResponse struct {
	Gu            interface{}              `json:"gu"`
	Items         interface{}              `json:"items"`
	Monsters      interface{}              `json:"monsters"`
	Constitutions []gamefx.ConstitutionDef `json:"constitutions"`
	Races         []gamefx.RaceDef         `json:"races"`
	LootTables    []models.LootTable       `json:"loot_tables"`
	Regions       []WikiRegion             `json:"regions"`
	MonsterSpawns []WikiMonsterSpawn       `json:"monster_spawns"`
}

// GetDictionary returns the unified Gu and Item master catalogs, cached in Redis
func GetDictionary(c *gin.Context) {
	var resp DictionaryResponse
	cacheKey := "game_dictionary_unified"

	if err := cache.GetJSON(cacheKey, &resp); err == nil && resp.Gu != nil && resp.Items != nil && resp.Monsters != nil && resp.LootTables != nil && resp.Regions != nil && resp.MonsterSpawns != nil {
		c.JSON(http.StatusOK, resp)
		return
	}

	flatGu := make([]models.GuDictionaryItem, 0)
	for _, ranks := range models.GuRegistry {
		for _, item := range ranks {
			flatGu = append(flatGu, item)
		}
	}

	resp = DictionaryResponse{
		Gu:            flatGu,
		Items:         models.ItemRegistry,
		Monsters:      models.MonsterRegistry,
		Constitutions: gamefx.AllConstitutions(),
		Races:         gamefx.AllRaces(),
		LootTables:    loadAllLootTables(c),
		Regions:       buildWikiRegions(),
		MonsterSpawns: buildWikiMonsterSpawns(),
	}

	logger.S.Infof("GetDictionary: returning %d gu, %d items, %d monsters", len(flatGu), len(models.ItemRegistry), len(models.MonsterRegistry))

	_ = cache.SetJSON(cacheKey, resp, 24*time.Hour)
	c.JSON(http.StatusOK, resp)
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
