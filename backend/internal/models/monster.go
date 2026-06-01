package models

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
}

func BuildCombatStateFromMonster(mon MonsterTemplate) CombatState {
	return CombatState{
		EnemyTemplateID:    mon.ID,
		EnemyName:          mon.Name,
		EnemyType:          mon.Type,
		EnemyHP:            mon.MaxHP,
		EnemyMaxHP:         mon.MaxHP,
		EnemyMana:          mon.CurrentMana,
		EnemyMaxMana:       mon.MaxMana,
		EnemyDamage:        mon.BaseAttack,
		EnemyDefense:       mon.BaseDefense,
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
