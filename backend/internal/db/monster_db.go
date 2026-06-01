package db

import (
	"database/sql"
	"encoding/json"

	"github.com/vippro67/GuWorld/backend/internal/models"
)

// LoadMonsterTemplate loads a monster template from database by ID
func LoadMonsterTemplate(tx *sql.Tx, id string) (models.MonsterTemplate, error) {
	var mon models.MonsterTemplate
	var guInvRaw, itemInvRaw, daoMarksRaw, pathMasteriesRaw []byte

	query := `
		SELECT id, name, type, rank, sub_rank, path, max_hp, base_attack, base_defense,
		       max_mana, current_mana, gu_inventory, item_inventory, dao_marks,
		       path_masteries, ai_profile, loot_table_id, description
		FROM monster_templates
		WHERE id = $1 AND is_active = TRUE
	`
	err := tx.QueryRow(query, id).Scan(
		&mon.ID, &mon.Name, &mon.Type, &mon.Rank, &mon.SubRank, &mon.Path,
		&mon.MaxHP, &mon.BaseAttack, &mon.BaseDefense, &mon.MaxMana, &mon.CurrentMana,
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
		mon.DaoMarks = make(map[string]int)
	}

	if len(pathMasteriesRaw) > 0 {
		_ = json.Unmarshal(pathMasteriesRaw, &mon.PathMasteries)
	} else {
		mon.PathMasteries = make(map[string]string)
	}

	return mon, nil
}

// FindFixedSpawn checks if there is a fixed spawn for this region at (x, y)
func FindFixedSpawn(tx *sql.Tx, regionID string, x, y int) (models.MonsterSpawn, bool, error) {
	var spawn models.MonsterSpawn
	query := `
		SELECT id, region_id, monster_id, x, y, spawn_mode, chance, respawn_seconds,
		       min_rank, max_rank, is_boss_spawn, is_active
		FROM monster_spawns
		WHERE region_id = $1 AND x = $2 AND y = $3 AND is_active = TRUE
		LIMIT 1
	`
	err := tx.QueryRow(query, regionID, x, y).Scan(
		&spawn.ID, &spawn.RegionID, &spawn.MonsterID, &spawn.X, &spawn.Y,
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
func PickRandomMonsterFromRegion(tx *sql.Tx, regionID string) (string, error) {
	var monsterID string
	query := `
		SELECT monster_id
		FROM monster_spawns
		WHERE region_id = $1 AND spawn_mode = 'Random' AND is_active = TRUE
		ORDER BY RANDOM()
		LIMIT 1
	`
	err := tx.QueryRow(query, regionID).Scan(&monsterID)
	if err == sql.ErrNoRows {
		// Fallback to random Normal monster template if no random spawn configured
		err = tx.QueryRow(`
			SELECT id FROM monster_templates 
			WHERE type = 'Normal' AND is_active = TRUE 
			ORDER BY RANDOM() LIMIT 1
		`).Scan(&monsterID)
		if err != nil {
			return "", err
		}
		return monsterID, nil
	}
	if err != nil {
		return "", err
	}
	return monsterID, nil
}

// LoadLootTable loads a loot table from DB by ID
func LoadLootTable(tx *sql.Tx, id string) (models.LootTable, error) {
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

	query := `
		SELECT cell_type, content_id
		FROM map_cells
		WHERE region_id = $1 AND x = $2 AND y = $3 AND is_active = TRUE
		LIMIT 1
	`
	err := tx.QueryRow(query, regionID, x, y).Scan(&cell.Type, &cell.Content)
	if err == sql.ErrNoRows {
		return cell, false, nil
	}
	if err != nil {
		return cell, false, err
	}
	return cell, true, nil
}
