package models

import (
	"strings"
)

type MonsterTemplate struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Type          string             `json:"type"`
	Rank          int                `json:"rank"`
	SubRank       int                `json:"sub_rank"`
	Path          string             `json:"path"`
	MaxHP         float64            `json:"max_hp"`
	BaseAttack    float64            `json:"base_attack"`
	BaseDefense   float64            `json:"base_defense"`
	MaxMana       float64            `json:"max_mana"`
	CurrentMana   float64            `json:"current_mana"`
	GuInventory   GuInventory        `json:"gu_inventory"`
	ItemInventory MaterialsInventory `json:"item_inventory"`
	DaoMarks      map[string]int     `json:"dao_marks"`
	PathMasteries map[string]string  `json:"path_masteries"`
	AIProfile     string             `json:"ai_profile"`
	LootTableID   string             `json:"loot_table_id"`
	Description   string             `json:"description"`
}

type LootEntry struct {
	ItemID string  `json:"item_id,omitempty"`
	GuID   string  `json:"gu_id,omitempty"`
	MinQty int     `json:"min_qty,omitempty"`
	MaxQty int     `json:"max_qty,omitempty"`
	Chance float64 `json:"chance"`
}

type LootTable struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Entries []LootEntry `json:"entries"`
}

type MonsterSpawn struct {
	ID             string  `json:"id"`
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

type Combatant struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Kind          string            `json:"kind"` // "Player", "Normal", "Boss", etc.
	Rank          int               `json:"rank"`
	SubRank       int               `json:"sub_rank"`
	HP            float64           `json:"hp"`
	MaxHP         float64           `json:"max_hp"`
	CurrentMana   float64           `json:"current_mana"`
	MaxMana       float64           `json:"max_mana"`
	BaseAttack    float64           `json:"base_attack"`
	BaseDefense   float64           `json:"base_defense"`
	GuInventory   GuInventory       `json:"gu_inventory"`
	DaoMarks      map[string]int    `json:"dao_marks"`
	PathMasteries map[string]string `json:"path_masteries"`
	AIProfile     string            `json:"ai_profile"`
	LootTableID   string            `json:"loot_table_id"`
	StatusEffects []StatusEffect    `json:"status_effects"`
	GridX         int               `json:"grid_x"` // 1 (Front), 2 (Mid), 3 (Back)
	GridY         int               `json:"grid_y"` // 1 (Top), 2 (Mid), 3 (Bottom)
	Speed         float64           `json:"speed"`
	VirtualShield float64           `json:"virtual_shield"`
	SoulHP        float64           `json:"soul_hp"`
	MaxSoulHP     float64           `json:"max_soul_hp"`
	DaoMarkDamage float64           `json:"dao_mark_damage"`
	ApertureDmg   float64           `json:"aperture_dmg"`
}

func ScaleMonsterStats(baseMaxHP, baseAttack, baseDefense float64, rank int, monsterType string) (maxHP, attack, defense float64) {
	rankMultiplier := 1.0
	switch rank {
	case 1:
		rankMultiplier = 1.0
	case 2:
		rankMultiplier = 2.0
	case 3:
		rankMultiplier = 4.0
	case 4:
		rankMultiplier = 8.0
	case 5:
		rankMultiplier = 16.0
	case 6:
		rankMultiplier = 1000.0
	case 7:
		rankMultiplier = 3000.0
	case 8:
		rankMultiplier = 10000.0
	case 9:
		rankMultiplier = 50000.0
	default:
		if rank > 9 {
			rankMultiplier = 50000.0
		}
	}

	typeMultiplier := 1.0
	switch strings.ToUpper(monsterType) {
	case "NORMAL", "MONSTER":
		typeMultiplier = 1.0
	case "ELITE":
		typeMultiplier = 2.0
	case "BOSS":
		typeMultiplier = 5.0
	case "WORLDBOSS", "WORLD BOSS":
		typeMultiplier = 25.0
	case "TRIBULATION_MONSTER":
		typeMultiplier = 3.0
	case "TRIBULATION_BOSS":
		typeMultiplier = 10.0
	}

	maxHP = baseMaxHP * rankMultiplier * typeMultiplier

	// Scale attack and defense by flater multipliers for balance
	atkMultiplier := 1.0
	switch rank {
	case 1:
		atkMultiplier = 1.0
	case 2:
		atkMultiplier = 1.5
	case 3:
		atkMultiplier = 2.5
	case 4:
		atkMultiplier = 4.0
	case 5:
		atkMultiplier = 6.0
	case 6:
		atkMultiplier = 50.0
	case 7:
		atkMultiplier = 150.0
	case 8:
		atkMultiplier = 500.0
	case 9:
		atkMultiplier = 2000.0
	default:
		if rank > 9 {
			atkMultiplier = 2000.0
		}
	}

	attack = baseAttack * atkMultiplier * typeMultiplier
	defense = baseDefense * atkMultiplier * typeMultiplier

	if defense < 0 {
		defense = 0
	}
	return
}

func BuildCombatStateFromMonster(mon MonsterTemplate) CombatState {
	hp, atk, def := ScaleMonsterStats(mon.MaxHP, mon.BaseAttack, mon.BaseDefense, mon.Rank, mon.Type)
	return CombatState{
		EnemyTemplateID:    mon.ID,
		EnemyName:          mon.Name,
		EnemyType:          mon.Type,
		EnemyHP:            hp,
		EnemyMaxHP:         hp,
		EnemyMana:          mon.CurrentMana,
		EnemyMaxMana:       mon.MaxMana,
		EnemyDamage:        atk,
		EnemyDefense:       def,
		EnemyRank:          mon.Rank,
		EnemySubRank:       mon.SubRank,
		EnemyGuInventory:   mon.GuInventory,
		EnemyDaoMarks:      mon.DaoMarks,
		EnemyPathMasteries: mon.PathMasteries,
		AIProfile:          mon.AIProfile,
		LootTableID:        mon.LootTableID,
	}
}

func CombatantFromPlayer(player *Player, stats map[string]float64) Combatant {
	return Combatant{
		ID:            player.ID.String(),
		Name:          player.Username,
		Kind:          "Player",
		Rank:          player.Rank,
		SubRank:       player.SubRank,
		HP:            player.HP,
		MaxHP:         player.BaseHP,
		CurrentMana:   player.CurrentMana,
		MaxMana:       player.MaxMana,
		BaseAttack:    stats["attack"],
		BaseDefense:   stats["defense"],
		GuInventory:   player.GuInventory,
		DaoMarks:      player.DaoMarks,
		PathMasteries: player.PathMasteries,
		StatusEffects: []StatusEffect{}, // Will be populated in combat loop
		SoulHP:        float64(player.SoulHP),
		MaxSoulHP:     float64(player.MaxSoul),
		DaoMarkDamage: player.DaoMarkDamage,
		ApertureDmg:   player.ApertureDamage,
	}
}

func CombatantFromActiveCombat(cs *CombatState) Combatant {
	return Combatant{
		ID:            cs.EnemyTemplateID,
		Name:          cs.EnemyName,
		Kind:          cs.EnemyType,
		Rank:          cs.EnemyRank,
		SubRank:       cs.EnemySubRank,
		HP:            cs.EnemyHP,
		MaxHP:         cs.EnemyMaxHP,
		CurrentMana:   cs.EnemyMana,
		MaxMana:       cs.EnemyMaxMana,
		BaseAttack:    cs.EnemyDamage,
		BaseDefense:   cs.EnemyDefense,
		GuInventory:   cs.EnemyGuInventory,
		DaoMarks:      cs.EnemyDaoMarks,
		PathMasteries: cs.EnemyPathMasteries,
		AIProfile:     cs.AIProfile,
		LootTableID:   cs.LootTableID,
		StatusEffects: cs.EnemyStatusEffects,
	}
}
