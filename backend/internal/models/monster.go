package models

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
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
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Kind           string            `json:"kind"` // "Player", "Normal", "Boss", etc.
	Rank           int               `json:"rank"`
	SubRank        int               `json:"sub_rank"`
	HP             float64           `json:"hp"`
	MaxHP          float64           `json:"max_hp"`
	CurrentMana    float64           `json:"current_mana"`
	MaxMana        float64           `json:"max_mana"`
	BaseAttack     float64           `json:"base_attack"`
	BaseDefense    float64           `json:"base_defense"`
	GuInventory    GuInventory       `json:"gu_inventory"`
	DaoMarks       DaoMarksMap       `json:"dao_marks"`
	PathMasteries  map[string]int    `json:"path_masteries"`
	BodyMods       BodyModsList      `json:"body_mods,omitempty"`
	AIProfile      string            `json:"ai_profile"`
	LootTableID    string            `json:"loot_table_id"`
	StatusEffects  []StatusEffect    `json:"status_effects"`
	GridX          int               `json:"grid_x"` // 1 (Front), 2 (Mid), 3 (Back)
	GridY          int               `json:"grid_y"` // 1 (Top), 2 (Mid), 3 (Bottom)
	Speed          float64           `json:"speed"`
	VirtualShield  float64           `json:"virtual_shield"`
	SoulHP         float64           `json:"soul_hp"`
	MaxSoulHP      float64           `json:"max_soul_hp"`
	DaoMarkDamage  float64           `json:"dao_mark_damage"`
	ApertureDmg    float64           `json:"aperture_dmg"`
	AvatarURL      string            `json:"avatar_url,omitempty"`
	FullyGrown     []FullyGrownInfo  `json:"fully_grown,omitempty"`
	CombatSettings *AutoCombatConfig `json:"combat_settings,omitempty"`
	// HasNonConflictingDao mirrors a permanent non-conflicting body trait.
	// When true, offensive conflict penalty is removed and natural defensive
	// resistance is zeroed out in DaoCombatProfile.
	HasNonConflictingDao     bool    `json:"has_non_conflicting_dao,omitempty"`
	DaoConflictReductionPct float64 `json:"dao_conflict_reduction_pct,omitempty"`
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
		rankMultiplier = 10000.0
	case 7:
		rankMultiplier = 1000000.0
	case 8:
		rankMultiplier = 100000000.0
	case 9:
		rankMultiplier = 10000000000.0
	default:
		if rank > 9 {
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
	return
}

func ScaleAndRandomizeMonsterDaoMarks(mon MonsterTemplate, rank int, subRank int) DaoMarksMap {
	if rank < 6 {
		return DaoMarksMap{}
	}

	baseDaoMarks := make(DaoMarksMap)

	// If template specifies DaoMarks, use them
	if len(mon.DaoMarks) > 0 {
		for k, v := range mon.DaoMarks {
			if v > 0 {
				baseDaoMarks[strings.TrimSuffix(strings.ToLower(k), " path")] = v
			}
		}
	}

	// If no base DaoMarks specified in template, pick random path(s)
	if len(baseDaoMarks) == 0 {
		availablePaths := make([]string, 0)
		pathSet := make(map[string]bool)
		for _, ranks := range GuRegistry {
			for _, gu := range ranks {
				for _, p := range gu.Path {
					cleanP := strings.TrimSuffix(strings.ToLower(p), " path")
					if cleanP != "" && !pathSet[cleanP] {
						pathSet[cleanP] = true
						availablePaths = append(availablePaths, cleanP)
					}
				}
			}
		}

		if len(availablePaths) == 0 {
			availablePaths = []string{"fire", "water", "earth", "wind", "lightning", "strength", "soul", "qi", "sword", "poison"}
		}

		isBoss := strings.Contains(strings.ToUpper(mon.Type), "BOSS")
		numPaths := 1
		if isBoss && len(availablePaths) > 1 {
			numPaths = 2
		}

		randIndexes := rand.Perm(len(availablePaths))
		for i := 0; i < numPaths && i < len(randIndexes); i++ {
			p := availablePaths[randIndexes[i]]
			baseDaoMarks[p] = 1000
		}
	}

	// Range [minDao, maxDao] per Rank
	var minDao, maxDao float64
	switch rank {
	case 6:
		minDao, maxDao = 10000.0, 40000.0
	case 7:
		minDao, maxDao = 40000.0, 220000.0
	case 8:
		minDao, maxDao = 220000.0, 1400000.0
	case 9:
		// Rank 9 (Venerable): No static upper cap!
		// Exponential growth by subRank (SubRank 1 ~8.5M, SubRank 10 ~580M+)
		minDao = 8500000.0
		srFactor := math.Pow(1.6, float64(subRank-1))
		maxDao = minDao * srFactor
	default:
		if rank > 9 {
			minDao = 8500000.0
			srFactor := math.Pow(1.6, float64(subRank-1))
			maxDao = minDao * srFactor
		}
	}

	if subRank < 1 {
		subRank = 1
	}

	targetTotalDao := minDao
	if rank < 9 {
		t := float64(subRank-1) / 9.0
		if t > 1.0 {
			t = 1.0
		}
		targetTotalDao = minDao + (maxDao-minDao)*t
	} else {
		targetTotalDao = maxDao
	}

	// Add random variance (-15% to +15%)
	variance := (rand.Float64()*0.30 - 0.15)
	targetTotalDao = targetTotalDao * (1.0 + variance)

	totalBase := 0.0
	for _, amt := range baseDaoMarks {
		totalBase += float64(amt)
	}
	if totalBase <= 0 {
		totalBase = 1.0
	}

	finalDaoMarks := make(DaoMarksMap)
	for path, baseAmt := range baseDaoMarks {
		portion := float64(baseAmt) / totalBase
		calculated := int64(targetTotalDao * portion)
		if calculated > 0 {
			finalDaoMarks[path] = calculated
		}
	}

	return finalDaoMarks
}

func BuildCombatStateFromMonster(mon MonsterTemplate, rank int, subRank int) CombatState {
	hp, atk, def := ScaleMonsterStats(mon.MaxHP, mon.BaseAttack, mon.BaseDefense, rank, subRank, mon.Type)
	finalDaoMarks := ScaleAndRandomizeMonsterDaoMarks(mon, rank, subRank)
	guInventory := buildMonsterGuInventory(mon, rank, finalDaoMarks)

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
		EnemyDaoMarks:      finalDaoMarks,
		EnemyPathMasteries: mon.PathMasteries,
		AIProfile:          mon.AIProfile,
		LootTableID:        mon.LootTableID,
	}
}

// ImmortalGuChecker is an optional global checker to verify if an Immortal Gu already exists in the world.
// When set, buildMonsterGuInventory will skip Immortal Gu that are already owned by any player, listed in auction, or stored in sect vaults.
var ImmortalGuChecker func(guID string, rank int) bool

func buildMonsterGuInventory(mon MonsterTemplate, monsterRank int, daoMarks DaoMarksMap) GuInventory {
	// Normal monsters have NO Gu
	monTypeUpper := strings.ToUpper(strings.TrimSpace(mon.Type))
	if monTypeUpper == "NORMAL" || monTypeUpper == "MONSTER" {
		return GuInventory{}
	}

	var result GuInventory

	if len(mon.GuInventory) > 0 {
		result = append(result, mon.GuInventory...)
	}

	isBoss := strings.Contains(monTypeUpper, "BOSS")

	// Match Gu from GuRegistry (for Rank 6+ using DaoMarks, or for Rank 1-5 using any valid Gu rank <= monsterRank)
	if len(GuRegistry) > 0 {
		var activeGus []GuDictionaryItem
		var passiveGus []GuDictionaryItem
		var allGus []GuDictionaryItem
		hasPaths := len(daoMarks) > 0

		for _, ranks := range GuRegistry {
			for guRank, gu := range ranks {
				if guRank > monsterRank || (guRank >= 6 && monsterRank < 6) {
					continue
				}

				// Check if this Immortal Gu already exists anywhere in the world
				isImmortalGu := guRank >= 6
				if isImmortalGu && ImmortalGuChecker != nil {
					if ImmortalGuChecker(gu.ID, guRank) {
						continue // Already exists in the world -> skip!
					}
				}

				// Check path match with monster dao marks if available
				isMatch := false
				if hasPaths {
					for _, guPath := range gu.Path {
						cleanPath := strings.TrimSuffix(strings.ToLower(guPath), " path")
						if _, exists := daoMarks[cleanPath]; exists {
							isMatch = true
							break
						}
					}
				} else {
					// Rank 1-5 Elite/Boss without DaoMarks: any Gu rank <= monsterRank is valid
					isMatch = true
				}

				if isMatch {
					allGus = append(allGus, gu)
					// Block v2 only: passive pool = Type PASSIVE. Legacy
					// Effects.Passive presence is never consulted.
					isPassive := strings.EqualFold(gu.Type, "PASSIVE")
					if isPassive {
						passiveGus = append(passiveGus, gu)
					} else {
						activeGus = append(activeGus, gu)
					}
				}
			}
		}

		if len(allGus) > 0 {
			numActive := 2
			numPassive := 1
			if isBoss {
				numActive = 3
				numPassive = 2
			}

			addedSet := make(map[string]bool)
			immortalCount := 0

			addFromPool := func(pool []GuDictionaryItem, count int) {
				if len(pool) == 0 {
					return
				}
				perm := rand.Perm(len(pool))
				added := 0
				for _, idx := range perm {
					if added >= count {
						break
					}
					gu := pool[idx]

					// Limit: At most 1 same-rank Immortal Gu per monster (guRank == monsterRank && Rank >= 6)
					isSameRankImmortal := gu.Rank >= 6 && gu.Rank == monsterRank
					if isSameRankImmortal && immortalCount >= 1 {
						continue
					}

					if !addedSet[gu.ID] {
						addedSet[gu.ID] = true
						if isSameRankImmortal {
							immortalCount++
						}
						inst := GuInstance{
							InstanceID: fmt.Sprintf("%s_gu_%s_%d", mon.ID, gu.ID, rand.Int63()),
							GuID:       gu.ID,
							Rank:       gu.Rank,
							Hunger:     0,
							Status:     "ACTIVE",
							Equipped:   true,
							IsVital:    false,
						}
						result = append(result, inst)
						added++
					}
				}
			}

			// 1. Add Active Combat Gu
			if len(activeGus) > 0 {
				addFromPool(activeGus, numActive)
			} else {
				addFromPool(allGus, numActive)
			}

			// 2. Add Passive Stat Gu (attack_base, hp_base, speed_base, etc.)
			if len(passiveGus) > 0 {
				addFromPool(passiveGus, numPassive)
			} else {
				addFromPool(allGus, numPassive)
			}
		}
	}

	// Fallback/additional Gu from mon.GuPool if configured
	if len(mon.GuPool) > 0 {
		pool := make([]GuPoolEntry, 0, len(mon.GuPool))
		for _, entry := range mon.GuPool {
			if entry.IsWorldBoss || entry.Rank > monsterRank {
				continue
			}
			pool = append(pool, entry)
		}

		if len(pool) > 0 {
			totalWeight := 0.0
			for _, e := range pool {
				totalWeight += e.Weight
			}
			if totalWeight > 0 {
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
			}
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
		ID:             player.ID.String(),
		Name:           player.Username,
		Kind:           "Player",
		Rank:           player.Rank,
		SubRank:        player.SubRank,
		HP:             player.HP,
		MaxHP:          effectiveBaseHP,
		CurrentMana:    player.CurrentMana,
		MaxMana:        player.MaxMana,
		BaseAttack:     stats["attack"],
		BaseDefense:    stats["defense"],
		Speed:          stats["speed"],
		GuInventory:    player.GuInventory,
		DaoMarks:       player.DaoMarks,
		PathMasteries:  player.PathMasteries,
		BodyMods:       player.BodyMods,
		HasNonConflictingDao: player.BodyMods.Has(BodyModSovereignImmortal),
		StatusEffects:  []StatusEffect{}, // Will be populated in combat loop
		SoulHP:         float64(player.SoulHP),
		MaxSoulHP:      float64(player.MaxSoul),
		DaoMarkDamage:  player.DaoMarkDamage,
		ApertureDmg:    player.ApertureDamage,
		AvatarURL:      player.AvatarURL,
		FullyGrown:     player.GrowthData.FullyGrown,
		CombatSettings: &player.CombatSettings,
	}
}

func CombatantFromActiveCombat(cs *CombatState) Combatant {
	return Combatant{
		ID:             cs.EnemyTemplateID,
		Name:           cs.EnemyName,
		Kind:           cs.EnemyType,
		Rank:           cs.EnemyRank,
		SubRank:        cs.EnemySubRank,
		HP:             cs.EnemyHP,
		MaxHP:          cs.EnemyMaxHP,
		CurrentMana:    cs.EnemyMana,
		MaxMana:        cs.EnemyMaxMana,
		BaseAttack:     cs.EnemyDamage,
		BaseDefense:    cs.EnemyDefense,
		Speed:          cs.EnemySpeed,
		GuInventory:    cs.EnemyGuInventory,
		DaoMarks:       cs.EnemyDaoMarks,
		PathMasteries:  cs.EnemyPathMasteries,
		AIProfile:      cs.AIProfile,
		LootTableID:    cs.LootTableID,
		StatusEffects:  cs.EnemyStatusEffects,
		AvatarURL:      cs.EnemyAvatarURL,
		CombatSettings: cs.EnemyCombatSettings,
	}
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
