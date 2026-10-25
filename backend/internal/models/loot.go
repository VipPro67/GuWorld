package models

import (
	"database/sql"
	"encoding/json"
	"strings"
	"sync"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// RegionLootRule represents a region-specific drop rule from region_loot_tables.
type RegionLootRule struct {
	RegionID    string  `json:"region_id"`
	LootTableID string  `json:"loot_table_id"`
	MonsterType string  `json:"monster_type"`
	MinRank     int     `json:"min_rank"`
	MaxRank     int     `json:"max_rank"`
	Chance      float64 `json:"chance"`
	IsActive    bool    `json:"is_active"`
}

var (
	lootMu             sync.RWMutex
	LootTableRegistry  = make(map[string]LootTable)
	RegionLootRegistry = make(map[string][]RegionLootRule) // key: lower(region_id)
	regionAncestors    = make(map[string][]string)         // key: lower(region_id) -> list of [region_id, parent_id, ...]
)

// LoadLootRegistry loads loot_tables and region_loot_tables into memory cache.
func LoadLootRegistry(db *sql.DB) error {
	if db == nil {
		return nil
	}

	// 1. Load region hierarchy for ancestry resolution
	ancestorMap := make(map[string][]string)
	parentMap := make(map[string]string)
	rRows, err := db.Query(`SELECT id, COALESCE(parent_id, '') FROM regions`)
	if err == nil {
		defer rRows.Close()
		for rRows.Next() {
			var id, parentID string
			if err := rRows.Scan(&id, &parentID); err == nil {
				parentMap[strings.ToLower(id)] = strings.ToLower(parentID)
			}
		}
		for id := range parentMap {
			scope := []string{id}
			curr := id
			for i := 0; i < 8; i++ {
				p, hasP := parentMap[curr]
				if !hasP || p == "" || p == curr {
					break
				}
				scope = append(scope, p)
				curr = p
			}
			ancestorMap[id] = scope
		}
	} else {
		logger.S.Warnf("LoadLootRegistry: regions query error (skipping ancestors): %v", err)
	}

	// 2. Load loot_tables
	newTables := make(map[string]LootTable)
	ltRows, err := db.Query(`SELECT id, name, COALESCE(entries, '[]'::jsonb) FROM loot_tables`)
	if err != nil {
		return err
	}
	defer ltRows.Close()

	for ltRows.Next() {
		var lt LootTable
		var entriesRaw []byte
		if err := ltRows.Scan(&lt.ID, &lt.Name, &entriesRaw); err != nil {
			logger.S.Warnf("LoadLootRegistry: scan error on loot_tables: %v", err)
			continue
		}
		if len(entriesRaw) > 0 {
			_ = json.Unmarshal(entriesRaw, &lt.Entries)
		} else {
			lt.Entries = []LootEntry{}
		}
		newTables[lt.ID] = lt
	}
	if err := ltRows.Err(); err != nil {
		return err
	}

	// 3. Load region_loot_tables
	newRegionLoot := make(map[string][]RegionLootRule)
	rlRows, err := db.Query(`
		SELECT region_id, loot_table_id, monster_type, min_rank, max_rank, chance, is_active
		FROM region_loot_tables
		WHERE is_active = TRUE AND chance > 0
		ORDER BY region_id, monster_type, loot_table_id`)
	if err != nil {
		logger.S.Warnf("LoadLootRegistry: region_loot_tables query error: %v", err)
	} else {
		defer rlRows.Close()
		for rlRows.Next() {
			var rule RegionLootRule
			if err := rlRows.Scan(&rule.RegionID, &rule.LootTableID, &rule.MonsterType, &rule.MinRank, &rule.MaxRank, &rule.Chance, &rule.IsActive); err != nil {
				continue
			}
			key := strings.ToLower(rule.RegionID)
			newRegionLoot[key] = append(newRegionLoot[key], rule)
		}
	}

	lootMu.Lock()
	LootTableRegistry = newTables
	RegionLootRegistry = newRegionLoot
	regionAncestors = ancestorMap
	lootMu.Unlock()

	logger.S.Infof("LoadLootRegistry: loaded %d loot tables and %d region loot rules into memory", len(newTables), len(newRegionLoot))
	return nil
}

// GetLootTable returns a loot table from the in-memory registry.
func GetLootTable(id string) (LootTable, bool) {
	lootMu.RLock()
	defer lootMu.RUnlock()
	lt, ok := LootTableRegistry[id]
	return lt, ok
}

// AllLootTables returns a slice of all cached loot tables.
func AllLootTables() []LootTable {
	lootMu.RLock()
	defer lootMu.RUnlock()
	tables := make([]LootTable, 0, len(LootTableRegistry))
	for _, lt := range LootTableRegistry {
		tables = append(tables, lt)
	}
	return tables
}

// GetRegionLootRules returns all matching region loot rules for a given region (including ancestor regions),
// monster type, and monster rank.
func GetRegionLootRules(regionID, monsterType string, rank int) []RegionLootRule {
	if regionID == "" || monsterType == "" {
		return nil
	}

	lootMu.RLock()
	defer lootMu.RUnlock()

	lowRegion := strings.ToLower(regionID)
	scope, hasScope := regionAncestors[lowRegion]
	if !hasScope || len(scope) == 0 {
		scope = []string{lowRegion}
	}

	var matched []RegionLootRule
	for _, rID := range scope {
		rules := RegionLootRegistry[rID]
		for _, rule := range rules {
			if !rule.IsActive || rule.Chance <= 0 {
				continue
			}
			if !strings.EqualFold(rule.MonsterType, monsterType) {
				continue
			}
			if rank < rule.MinRank || rank > rule.MaxRank {
				continue
			}
			matched = append(matched, rule)
		}
	}
	return matched
}
