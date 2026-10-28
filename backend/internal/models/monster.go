package models

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

type MonsterTemplate struct {
	ID               string             `json:"id"`
	Name             string             `json:"name"`
	Type             string             `json:"type"`
	MaxHP            float64            `json:"max_hp"`
	BaseAttack       float64            `json:"base_attack"`
	BaseDefense      float64            `json:"base_defense"`
	GuInventory      GuInventory        `json:"gu_inventory"`
	GuPool           []GuPoolEntry      `json:"gu_pool"`
	ItemInventory    MaterialsInventory `json:"item_inventory"`
	DaoMarks         DaoMarksMap        `json:"dao_marks"`
	PathMasteries    map[string]int     `json:"path_masteries"`
	AIProfile        string             `json:"ai_profile"`
	LootTableID      string             `json:"loot_table_id"`
	Description      string             `json:"description"`
	IsCapturable     bool               `json:"is_capturable"`
	CaptureChance    float64            `json:"capture_chance"`
	PetHungerPerHour float64            `json:"pet_hunger_per_hour"`
}

type GuPoolEntry struct {
	GuID        string  `json:"gu_id"`
	Rank        int     `json:"rank"`
	Weight      float64 `json:"weight"`
	MinCount    int     `json:"min_count"`
	MaxCount    int     `json:"max_count"`
	IsWorldBoss bool    `json:"is_world_boss"`
}

type LootEntry struct {
	ItemID  string  `json:"item_id,omitempty"`
	GuID    string  `json:"gu_id,omitempty"`
	MinQty  int     `json:"min_qty,omitempty"`
	MaxQty  int     `json:"max_qty,omitempty"`
	Chance  float64 `json:"chance"`
	Rank    int     `json:"rank,omitempty"`    // 0 = universal/theme (always rolls), N = only drops at enemy rank N
	Jackpot bool    `json:"jackpot,omitempty"` // always rolls regardless of rank (rare bonus)
}

type LootTable struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Entries []LootEntry `json:"entries"`
}

type MonsterSpawn struct {
	RegionID       string  `json:"region_id"`
	MonsterID      string  `json:"monster_id"`
	X              *int    `json:"x"`
	Y              *int    `json:"y"`
	SpawnMode      string  `json:"spawn_mode"`
	Chance         float64 `json:"chance"`
	RespawnSeconds int     `json:"respawn_seconds"`
	MinRank        int     `json:"min_rank"`
	MaxRank        int     `json:"max_rank"`
	IsBossSpawn    bool    `json:"is_boss_spawn"`
	IsActive       bool    `json:"is_active"`
}

func ScaleMonsterStats(baseMaxHP, baseAttack, baseDefense float64, rank int, subRank int, monsterType string) (maxHP, attack, defense float64) {
	// Single shared scale table (models.RankScaleTable). Monster base stats are
	// authored rank-independent in Monster.csv, so the rank curve is the only
	// thing that grows with rank — matching Gu values (ratio * RankScale).
	rankMultiplier := RankScale(rank)
	if rank > 9 {
		// Chaos/Hybrid tribulation spawns at rank 10 (there is no rank-10 row in
		// the table). Preserve its historical order-of-magnitude bump over rank 9
		// (10^10 -> 10^13) instead of clamping it.
		rankMultiplier = 10000000000000.0
	}

	typeMultiplier := 1.0
	switch strings.ToUpper(monsterType) {
	case "NORMAL", "MONSTER":
		typeMultiplier = 1.0
	case "ELITE":
		typeMultiplier = 1.5
	case "BOSS":
		typeMultiplier = 3.0
	case "WORLDBOSS", "WORLD BOSS":
		typeMultiplier = 25.0
	case "TRIBULATION_MONSTER":
		typeMultiplier = 3.0
	case "TRIBULATION_BOSS":
		typeMultiplier = 10.0
	}

	// Each subRank above 1 gives a 20% multiplier boost.
	subRankMultiplier := 1.0
	if subRank > 1 {
		subRankMultiplier = 1.0 + float64(subRank-1)*0.20
	}

	maxHP = baseMaxHP * rankMultiplier * typeMultiplier * subRankMultiplier

	attack = baseAttack * rankMultiplier * typeMultiplier * subRankMultiplier
	defense = baseDefense * rankMultiplier * typeMultiplier * subRankMultiplier

	// Scale down attack and defense for tribulation entities to balance against players
	if strings.ToUpper(monsterType) == "TRIBULATION_MONSTER" {
		attack *= 0.10
		defense *= 0.10
	} else if strings.ToUpper(monsterType) == "TRIBULATION_BOSS" {
		attack *= 0.05
		defense *= 0.02
	}

	if defense < 0 {
		defense = 0
	}
	return
}

var MonsterRegistry = make(map[string]MonsterTemplate)

func LoadMonsterRegistry(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, name, type, COALESCE(max_hp, 100), COALESCE(base_attack, 10), COALESCE(base_defense, 10), COALESCE(gu_inventory, '[]'::jsonb), COALESCE(gu_pool, '[]'::jsonb), COALESCE(item_inventory, '{}'::jsonb), COALESCE(dao_marks, '{}'::jsonb), COALESCE(path_masteries, '{}'::jsonb), COALESCE(ai_profile, 'beast_basic'), COALESCE(loot_table_id, ''), COALESCE(description, ''),
		       COALESCE(is_capturable, false),
		       COALESCE(capture_chance, 0.20),
		       COALESCE(pet_hunger_per_hour, 2.0)
		FROM monster_templates
		WHERE is_active = true OR is_active IS NULL
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	newRegistry := make(map[string]MonsterTemplate)
	count := 0

	for rows.Next() {
		var mon MonsterTemplate
		var guInvRaw, guPoolRaw, itemInvRaw, daoMarksRaw, masteriesRaw []byte
		err = rows.Scan(
			&mon.ID, &mon.Name, &mon.Type,
			&mon.MaxHP, &mon.BaseAttack, &mon.BaseDefense,
			&guInvRaw, &guPoolRaw, &itemInvRaw, &daoMarksRaw, &masteriesRaw, &mon.AIProfile, &mon.LootTableID, &mon.Description,
			&mon.IsCapturable, &mon.CaptureChance, &mon.PetHungerPerHour,
		)
		if err != nil {
			logger.S.Warnf("failed to scan monster template row: %v", err)
			continue
		}

		_ = json.Unmarshal(guInvRaw, &mon.GuInventory)
		_ = json.Unmarshal(guPoolRaw, &mon.GuPool)
		_ = json.Unmarshal(itemInvRaw, &mon.ItemInventory)
		_ = json.Unmarshal(daoMarksRaw, &mon.DaoMarks)
		_ = json.Unmarshal(masteriesRaw, &mon.PathMasteries)

		newRegistry[mon.ID] = mon
		count++
	}
	if err := rows.Err(); err != nil {
		logger.S.Warnf("LoadMonsterRegistry rows iteration error: %v", err)
	}

	MonsterRegistry = newRegistry
	if count == 0 {
		logger.S.Warn("LoadMonsterRegistry completed but loaded 0 monsters. Please check if the 'monster_templates' table has data and if 'is_active' is set to true.")
	} else {
		logger.S.Infof("Loaded %d monster templates from database registry into memory cache", count)
	}
	return nil
}
