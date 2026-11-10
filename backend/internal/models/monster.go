package models

import (
	"database/sql"
	"encoding/json"
	"os"
	"strconv"
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

func BuildCombatStateFromMonster(mon MonsterTemplate, rank, subRank int) CombatState {
	return BuildCombatStateFromMonsterWithRegion(mon, rank, subRank, nil)
}

func BuildCombatStateFromMonsterWithRegion(mon MonsterTemplate, rank, subRank int, _ []GuPoolEntry) CombatState {
	hp, attack, defense := ScaleMonsterStats(mon.MaxHP, mon.BaseAttack, mon.BaseDefense, rank, subRank, mon.Type)
	return CombatState{
		EnemyTemplateID: mon.ID, EnemyName: mon.Name, EnemyType: mon.Type,
		EnemyHP: hp, EnemyMaxHP: hp, EnemyMana: 100, EnemyMaxMana: 100,
		EnemyDamage: attack, EnemyDefense: defense, EnemyRank: rank, EnemySubRank: subRank,
		EnemyGuInventory: append(GuInventory(nil), mon.GuInventory...),
		EnemyDaoMarks:    mon.DaoMarks, EnemyPathMasteries: mon.PathMasteries,
		AIProfile: mon.AIProfile, LootTableID: mon.LootTableID,
	}
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

// Combatant is the historical turn-based battle projection of a player,
// monster, pet, or other combat actor.
type Combatant struct {
	ID                      string            `json:"id"`
	Name                    string            `json:"name"`
	Kind                    string            `json:"kind"`
	Rank                    int               `json:"rank"`
	SubRank                 int               `json:"sub_rank"`
	HP                      float64           `json:"hp"`
	MaxHP                   float64           `json:"max_hp"`
	CurrentMana             float64           `json:"current_mana"`
	MaxMana                 float64           `json:"max_mana"`
	BaseAttack              float64           `json:"base_attack"`
	BaseDefense             float64           `json:"base_defense"`
	GuInventory             GuInventory       `json:"gu_inventory"`
	DaoMarks                DaoMarksMap       `json:"dao_marks"`
	PathMasteries           map[string]int    `json:"path_masteries"`
	BodyMods                BodyModsList      `json:"body_mods,omitempty"`
	AIProfile               string            `json:"ai_profile"`
	LootTableID             string            `json:"loot_table_id"`
	StatusEffects           []StatusEffect    `json:"status_effects"`
	GridX                   int               `json:"grid_x"`
	GridY                   int               `json:"grid_y"`
	Speed                   float64           `json:"speed"`
	VirtualShield           float64           `json:"virtual_shield"`
	SoulHP                  float64           `json:"soul_hp"`
	MaxSoulHP               float64           `json:"max_soul_hp"`
	DaoMarkDamage           float64           `json:"dao_mark_damage"`
	ApertureDmg             float64           `json:"aperture_dmg"`
	AvatarURL               string            `json:"avatar_url,omitempty"`
	FullyGrown              []FullyGrownInfo  `json:"fully_grown,omitempty"`
	CombatSettings          *AutoCombatConfig `json:"combat_settings,omitempty"`
	HasNonConflictingDao    bool              `json:"has_non_conflicting_dao,omitempty"`
	DaoConflictReductionPct float64           `json:"dao_conflict_reduction_pct,omitempty"`
}

// MonsterRankScalingEnabled reports whether monster and boss stats are
// multiplied by the shared RankScale(rank) curve. Defaults to enabled; set
// MONSTER_RANK_SCALING_ENABLED to false/0/off/no to drop the rank multiplier
// while keeping the type and sub-rank multipliers.
func MonsterRankScalingEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MONSTER_RANK_SCALING_ENABLED"))) {
	case "false", "0", "off", "no":
		return false
	}
	return true
}

// MonsterStatMultiplier is the global multiplier applied to every monster/boss
// stat (HP/ATK/DEF) after rank, type and sub-rank scaling. Defaults to 1.0;
// set MONSTER_STAT_MULTIPLIER to a positive float (e.g. 0.5) for a temporary
// global nerf/buff. Missing, invalid or non-positive values fall back to 1.0.
func MonsterStatMultiplier() float64 {
	raw := strings.TrimSpace(os.Getenv("MONSTER_STAT_MULTIPLIER"))
	if raw == "" {
		return 1.0
	}
	mult, err := strconv.ParseFloat(raw, 64)
	if err != nil || mult <= 0 {
		return 1.0
	}
	return mult
}

func ScaleMonsterStats(baseMaxHP, baseAttack, baseDefense float64, rank int, subRank int, monsterType string) (maxHP, attack, defense float64) {
	// Single shared scale table (models.RankScaleTable). Monster base stats are
	// authored rank-independent in Monster.csv, so the rank curve is the only
	// thing that grows with rank — matching Gu values (ratio * RankScale).
	// MONSTER_RANK_SCALING_ENABLED=false keeps only the type/sub-rank multipliers.
	rankMultiplier := 1.0
	if MonsterRankScalingEnabled() {
		rankMultiplier = RankScale(rank)
		if rank > 9 {
			// Chaos/Hybrid tribulation spawns at rank 10 (there is no rank-10 row in
			// the table). Preserve its historical order-of-magnitude bump over rank 9
			// (10^10 -> 10^13) instead of clamping it.
			rankMultiplier = 10000000000000.0
		}
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

	// Global tuning knob applied last so every spawn path (world monsters,
	// region preview, sect beasts, ascension/tribulation bosses) stays in sync.
	if mult := MonsterStatMultiplier(); mult != 1.0 {
		maxHP *= mult
		attack *= mult
		defense *= mult
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
