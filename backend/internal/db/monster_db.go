package db

import (
	"database/sql"
	"encoding/json"
	"math/rand"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// LoadMonsterTemplate loads a monster template from database by ID
func LoadMonsterTemplate(tx *sql.Tx, id string) (models.MonsterTemplate, error) {
	var mon models.MonsterTemplate
	var guInvRaw, itemInvRaw, daoMarksRaw, pathMasteriesRaw []byte

	query := `
		SELECT id, name, type, max_hp, base_attack, base_defense,
		       gu_inventory, item_inventory, dao_marks,
		       path_masteries, ai_profile, loot_table_id, description
		FROM monster_templates
		WHERE id = $1 AND is_active = TRUE
	`
	err := tx.QueryRow(query, id).Scan(
		&mon.ID, &mon.Name, &mon.Type,
		&mon.MaxHP, &mon.BaseAttack, &mon.BaseDefense,
		&guInvRaw, &itemInvRaw, &daoMarksRaw, &pathMasteriesRaw, &mon.AIProfile,
		&mon.LootTableID, &mon.Description,
	)
	if err != nil {
		return mon, err
	}

	if len(guInvRaw) > 0 {
		_ = json.Unmarshal(guInvRaw, &mon.GuInventory)
	} else {
		mon.GuInventory = models.GuInventory{}
	}

	if len(itemInvRaw) > 0 {
		_ = json.Unmarshal(itemInvRaw, &mon.ItemInventory)
	} else {
		mon.ItemInventory = models.MaterialsInventory{}
	}

	if len(daoMarksRaw) > 0 {
		_ = json.Unmarshal(daoMarksRaw, &mon.DaoMarks)
	} else {
		mon.DaoMarks = make(models.DaoMarksMap)
	}

	if len(pathMasteriesRaw) > 0 {
		_ = json.Unmarshal(pathMasteriesRaw, &mon.PathMasteries)
	} else {
		mon.PathMasteries = make(map[string]int)
	}

	return mon, nil
}

// FindFixedSpawn checks if there is a fixed spawn for this region at (x, y)
func FindFixedSpawn(tx *sql.Tx, regionID string, x, y int) (models.MonsterSpawn, bool, error) {
	var spawn models.MonsterSpawn
	query := `
		SELECT region_id, monster_id, x, y, spawn_mode, chance, respawn_seconds,
		       min_rank, max_rank, is_boss_spawn, is_active
		FROM monster_spawns
		WHERE region_id = $1 AND x = $2 AND y = $3 AND is_active = TRUE
		LIMIT 1
	`
	err := tx.QueryRow(query, regionID, x, y).Scan(
		&spawn.RegionID, &spawn.MonsterID, &spawn.X, &spawn.Y,
		&spawn.SpawnMode, &spawn.Chance, &spawn.RespawnSeconds, &spawn.MinRank,
		&spawn.MaxRank, &spawn.IsBossSpawn, &spawn.IsActive,
	)
	if err == sql.ErrNoRows {
		return spawn, false, nil
	}
	if err != nil {
		return spawn, false, err
	}
	return spawn, true, nil
}

// PickRandomMonsterFromRegion picks a random active monster template mapped to a region
// Returns monsterID, minRank, maxRank, error
func PickRandomMonsterFromRegion(tx *sql.Tx, regionID string) (string, int, int, error) {
	var monsterID string
	var minRank, maxRank int

	query := `
		SELECT monster_id, min_rank, max_rank
		FROM monster_spawns
		WHERE region_id = $1 AND spawn_mode IN ('Roaming', 'Random') AND is_active = TRUE
		ORDER BY RANDOM()
		LIMIT 1
	`
	err := tx.QueryRow(query, regionID).Scan(&monsterID, &minRank, &maxRank)
	if err == sql.ErrNoRows {
		// Fallback to random Normal monster template if no random spawn configured
		err = tx.QueryRow(`
			SELECT id FROM monster_templates 
			WHERE type = 'Normal' AND is_active = TRUE 
			ORDER BY RANDOM() LIMIT 1
		`).Scan(&monsterID)
		if err != nil {
			return "", 1, 1, err
		}

		// Fetch region min/max rank for fallback
		err = tx.QueryRow(`
			SELECT min_rank, max_rank FROM regions WHERE id = $1
		`, regionID).Scan(&minRank, &maxRank)
		if err != nil {
			minRank = 1
			maxRank = 1
		}

		return monsterID, minRank, maxRank, nil
	}
	if err != nil {
		return "", 1, 1, err
	}
	return monsterID, minRank, maxRank, nil
}

// LoadLootTable loads a loot table by ID, preferring the in-memory cache
// and falling back to a DB query when the cache is cold or in uninitialized tests.
func LoadLootTable(tx *sql.Tx, id string) (models.LootTable, error) {
	if lt, ok := models.GetLootTable(id); ok {
		return lt, nil
	}

	var lt models.LootTable
	var entriesRaw []byte

	query := `SELECT id, name, entries FROM loot_tables WHERE id = $1`
	err := tx.QueryRow(query, id).Scan(&lt.ID, &lt.Name, &entriesRaw)
	if err != nil {
		return lt, err
	}

	if len(entriesRaw) > 0 {
		_ = json.Unmarshal(entriesRaw, &lt.Entries)
	} else {
		lt.Entries = []models.LootEntry{}
	}

	return lt, nil
}

// FindSpecialCell checks map_cells for any special location at (x, y)
func FindSpecialCell(tx *sql.Tx, regionID string, x, y int) (models.Cell, bool, error) {
	var cell models.Cell
	cell.X = x
	cell.Y = y

	var metadataBytes []byte
	query := `
		SELECT cell_type, content_id, metadata
		FROM map_cells
		WHERE region_id = $1 AND x = $2 AND y = $3 AND is_active = TRUE
		LIMIT 1
	`
	err := tx.QueryRow(query, regionID, x, y).Scan(&cell.Type, &cell.Content, &metadataBytes)
	if err == sql.ErrNoRows {
		return cell, false, nil
	}
	if err != nil {
		return cell, false, err
	}
	if len(metadataBytes) > 0 {
		var meta struct {
			Rank int `json:"rank"`
		}
		if err := json.Unmarshal(metadataBytes, &meta); err == nil {
			cell.Rank = meta.Rank
		}
	}
	return cell, true, nil
}

// PickDeterministicMonsterFromRegion picks a monster deterministically using a seed
func PickDeterministicMonsterFromRegion(tx *sql.Tx, regionID string, seed int64) (string, int, int, error) {
	var monsterID string
	var minRank, maxRank int

	rng := rand.New(rand.NewSource(seed))

	// Get all possible active random/roaming spawns for this region
	rows, err := tx.Query(`
		SELECT monster_id, min_rank, max_rank
		FROM monster_spawns
		WHERE region_id = $1 AND spawn_mode IN ('Roaming', 'Random') AND is_active = TRUE
		ORDER BY monster_id ASC
	`, regionID)

	if err == nil {
		defer rows.Close()
		type spawn struct {
			monsterID string
			minRank   int
			maxRank   int
		}
		var spawns []spawn
		for rows.Next() {
			var s spawn
			if errScan := rows.Scan(&s.monsterID, &s.minRank, &s.maxRank); errScan == nil {
				spawns = append(spawns, s)
			}
		}

		if len(spawns) > 0 {
			picked := spawns[rng.Intn(len(spawns))]
			return picked.monsterID, picked.minRank, picked.maxRank, nil
		}
	}

	// Fallback to random Normal monster template if no random spawn configured
	rows, err = tx.Query(`
		SELECT id FROM monster_templates 
		WHERE type = 'Normal' AND is_active = TRUE 
		ORDER BY id ASC
	`)
	if err != nil {
		return "", 1, 1, err
	}
	defer rows.Close()

	var templates []string
	for rows.Next() {
		var id string
		if errScan := rows.Scan(&id); errScan == nil {
			templates = append(templates, id)
		}
	}

	if len(templates) == 0 {
		return "", 1, 1, sql.ErrNoRows
	}

	monsterID = templates[rng.Intn(len(templates))]

	// Fetch region min/max rank for fallback
	err = tx.QueryRow(`
		SELECT min_rank, max_rank FROM regions WHERE id = $1
	`, regionID).Scan(&minRank, &maxRank)
	if err != nil {
		minRank = 1
		maxRank = 1
	}

	return monsterID, minRank, maxRank, nil
}
