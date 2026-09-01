package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
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
	DaoMarks         map[string]int    `json:"dao_marks"`
	PathMasteries    map[string]int    `json:"path_masteries"`
	AIProfile        string             `json:"ai_profile"`
	LootTableID      string             `json:"loot_table_id"`
	Description      string             `json:"description"`
	IsCapturable     bool               `json:"is_capturable"`
	CaptureChance    float64            `json:"capture_chance"`
	PetHungerPerHour float64            `json:"pet_hunger_per_hour"`
}

type GuPoolEntry struct {
	GuID         string  `json:"gu_id"`
	Rank         int     `json:"rank"`
	Weight       float64 `json:"weight"`
	MinCount     int     `json:"min_count"`
	MaxCount     int     `json:"max_count"`
	IsWorldBoss  bool    `json:"is_world_boss"`
	IsImmortal   bool    `json:"is_immortal"`
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
	PathMasteries map[string]int    `json:"path_masteries"`
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
	AvatarURL     string            `json:"avatar_url,omitempty"`
	FullyGrown    []FullyGrownInfo  `json:"fully_grown,omitempty"`
}

func ScaleMonsterStats(baseMaxHP, baseAttack, baseDefense float64, rank int, subRank int, monsterType string) (maxHP, attack, defense float64) {
	rankMultiplier := 1.0
	switch rank {
	case 1:
		rankMultiplier = 1.0
	case 2:
		rankMultiplier = 2.0
	case 3:
		rankMultiplier = 5.0
	case 4:
		rankMultiplier = 20.0
	case 5:
		rankMultiplier = 100.0
	case 6:
		rankMultiplier = 1000.0
	case 7:
		rankMultiplier = 10000.0
	case 8:
		rankMultiplier = 100000.0
	case 9:
		rankMultiplier = 10000000.0
	default:
		if rank > 9 {
			rankMultiplier = 10000000.0
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

func BuildCombatStateFromMonster(mon MonsterTemplate, rank int, subRank int) CombatState {
	hp, atk, def := ScaleMonsterStats(mon.MaxHP, mon.BaseAttack, mon.BaseDefense, rank, subRank, mon.Type)
	guInventory := buildMonsterGuInventory(mon, rank)
	return CombatState{
		EnemyTemplateID:    mon.ID,
		EnemyName:          mon.Name,
		EnemyType:          mon.Type,
		EnemyHP:            hp,
		EnemyMaxHP:         hp,
		EnemyMana:          100.0,
		EnemyMaxMana:       100.0,
		EnemyDamage:        atk,
		EnemyDefense:       def,
		EnemyRank:          rank,
		EnemySubRank:       subRank,
		EnemyGuInventory:   guInventory,
		EnemyDaoMarks:      mon.DaoMarks,
		EnemyPathMasteries: mon.PathMasteries,
		AIProfile:          mon.AIProfile,
		LootTableID:        mon.LootTableID,
	}
}

func buildMonsterGuInventory(mon MonsterTemplate, monsterRank int) GuInventory {
	var result GuInventory

	if len(mon.GuInventory) > 0 {
		result = append(result, mon.GuInventory...)
	}

	if len(mon.GuPool) == 0 {
		return result
	}

	pool := make([]GuPoolEntry, 0, len(mon.GuPool))
	for _, entry := range mon.GuPool {
		if entry.IsWorldBoss {
			continue
		}
		if entry.Rank > monsterRank {
			continue
		}
		pool = append(pool, entry)
	}

	if len(pool) == 0 {
		return result
	}

	totalWeight := 0.0
	for _, e := range pool {
		totalWeight += e.Weight
	}
	if totalWeight <= 0 {
		return result
	}

	for _, entry := range pool {
		count := entry.MinCount
		if entry.MaxCount > entry.MinCount {
			count += rand.Intn(entry.MaxCount - entry.MinCount + 1)
		}
		for i := 0; i < count; i++ {
			roll := rand.Float64() * totalWeight
			acc := 0.0
			var selected *GuPoolEntry
			for j := range pool {
				acc += pool[j].Weight
				if roll <= acc {
					selected = &pool[j]
					break
				}
			}
			if selected == nil {
				selected = &pool[0]
			}
			inst := GuInstance{
				InstanceID: fmt.Sprintf("%s_gu_%s_%d", mon.ID, selected.GuID, rand.Int63()),
				GuID:       selected.GuID,
				Rank:       selected.Rank,
				Hunger:     0,
				Status:     "ACTIVE",
				Equipped:   true,
				IsVital:    false,
			}
			result = append(result, inst)
		}
	}

	return result
}

func GetWorldBossGuPool(mon MonsterTemplate, monsterRank int) []GuPoolEntry {
	var pool []GuPoolEntry
	for _, entry := range mon.GuPool {
		if !entry.IsWorldBoss {
			continue
		}
		if entry.Rank > monsterRank {
			continue
		}
		pool = append(pool, entry)
	}
	return pool
}

func RollWorldBossGuDrop(pool []GuPoolEntry, playerRank int) *GuPoolEntry {
	if len(pool) == 0 {
		return nil
	}

	filtered := make([]GuPoolEntry, 0, len(pool))
	for _, e := range pool {
		if e.Rank <= playerRank+1 {
			filtered = append(filtered, e)
		}
	}
	if len(filtered) == 0 {
		filtered = pool
	}

	totalWeight := 0.0
	for _, e := range filtered {
		totalWeight += e.Weight
	}
	if totalWeight <= 0 {
		return nil
	}

	roll := rand.Float64() * totalWeight
	acc := 0.0
	for i := range filtered {
		acc += filtered[i].Weight
		if roll <= acc {
			return &filtered[i]
		}
	}
	return nil
}

func CombatantFromPlayer(player *Player, stats map[string]float64) Combatant {
	hpBonus := stats["hp"]
	effectiveBaseHP := player.BaseHP + hpBonus
	return Combatant{
		ID:            player.ID.String(),
		Name:          player.Username,
		Kind:          "Player",
		Rank:          player.Rank,
		SubRank:       player.SubRank,
		HP:            player.HP,
		MaxHP:         effectiveBaseHP,
		CurrentMana:   player.CurrentMana,
		MaxMana:       player.MaxMana,
		BaseAttack:    stats["attack"],
		BaseDefense:   stats["defense"],
		Speed:         stats["speed"],
		GuInventory:   player.GuInventory,
		DaoMarks:      player.DaoMarks,
		PathMasteries: player.PathMasteries,
		StatusEffects: []StatusEffect{}, // Will be populated in combat loop
		SoulHP:        float64(player.SoulHP),
		MaxSoulHP:     float64(player.MaxSoul),
		DaoMarkDamage: player.DaoMarkDamage,
		ApertureDmg:   player.ApertureDamage,
		AvatarURL:     player.AvatarURL,
		FullyGrown:    player.GrowthData.FullyGrown,
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
		Speed:         cs.EnemySpeed,
		GuInventory:   cs.EnemyGuInventory,
		DaoMarks:      cs.EnemyDaoMarks,
		PathMasteries: cs.EnemyPathMasteries,
		AIProfile:     cs.AIProfile,
		LootTableID:   cs.LootTableID,
		StatusEffects: cs.EnemyStatusEffects,
		AvatarURL:     cs.EnemyAvatarURL,
	}
}

var MonsterRegistry = make(map[string]MonsterTemplate)

func LoadMonsterRegistry(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, name, type, max_hp, base_attack, base_defense, gu_inventory, gu_pool, item_inventory, dao_marks, path_masteries, ai_profile, loot_table_id, description,
		       COALESCE(is_capturable, false),
		       COALESCE(capture_chance, 0.20),
		       COALESCE(pet_hunger_per_hour, 2.0)
		FROM monster_templates
		WHERE is_active = true
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

	MonsterRegistry = newRegistry
	logger.S.Infof("Loaded %d monster templates from database registry into memory cache", count)
	return nil
}

