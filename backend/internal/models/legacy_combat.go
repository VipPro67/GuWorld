package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// StatusEffect is a combat-only, turn-counted effect snapshot. Persistent
// player statuses continue to use StatusInstance.
type StatusEffect struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
	Turns int     `json:"turns"`
}

type TitleEffectSnapshot struct {
	AchievementID string  `json:"achievement_id"`
	MonsterType   string  `json:"monster_type"`
	DamagePct     float64 `json:"damage_pct"`
}

// CombatState is the durable encounter snapshot consumed by the historical
// deterministic turn-based simulator. It is retained as a domain type while
// encounter persistence is moved out of the hot players row.
type CombatState struct {
	CombatID                   string               `json:"combat_id,omitempty"`
	CombatOwnerID              string               `json:"combat_owner_id,omitempty"`
	TitleEffectSnapshot        *TitleEffectSnapshot `json:"title_effect_snapshot,omitempty"`
	EnemyTemplateID            string               `json:"enemy_template_id,omitempty"`
	WorldMonsterRegionID       string               `json:"world_monster_region_id,omitempty"`
	WorldMonsterID             string               `json:"world_monster_id,omitempty"`
	WorldMonsterEncounterID    string               `json:"world_monster_encounter_id,omitempty"`
	WorldBossCellRegionID      string               `json:"world_boss_cell_region_id,omitempty"`
	WorldBossCellX             int                  `json:"world_boss_cell_x,omitempty"`
	WorldBossCellY             int                  `json:"world_boss_cell_y,omitempty"`
	EnemyName                  string               `json:"enemy_name"`
	EnemyAvatarURL             string               `json:"enemy_avatar_url,omitempty"`
	EnemyType                  string               `json:"enemy_type,omitempty"`
	CombatMode                 string               `json:"combat_mode,omitempty"`
	EnemyHP                    float64              `json:"enemy_hp"`
	EnemyMaxHP                 float64              `json:"enemy_max_hp"`
	EnemyMana                  float64              `json:"enemy_mana,omitempty"`
	EnemyMaxMana               float64              `json:"enemy_max_mana,omitempty"`
	EnemyDamage                float64              `json:"enemy_damage"`
	EnemyDefense               float64              `json:"enemy_defense"`
	EnemySpeed                 float64              `json:"enemy_speed,omitempty"`
	EnemyRank                  int                  `json:"enemy_rank,omitempty"`
	EnemySubRank               int                  `json:"enemy_sub_rank,omitempty"`
	EnemyGuInventory           GuInventory          `json:"enemy_gu_inventory,omitempty"`
	EnemyDaoMarks              DaoMarksMap          `json:"enemy_dao_marks,omitempty"`
	EnemyPathMasteries         map[string]int       `json:"enemy_path_masteries,omitempty"`
	AIProfile                  string               `json:"ai_profile,omitempty"`
	LootTableID                string               `json:"loot_table_id,omitempty"`
	ChargePoints               int                  `json:"charge_points"`
	TargetPlayerID             string               `json:"target_player_id,omitempty"`
	TargetSectID               string               `json:"target_sect_id,omitempty"`
	EnemyStatusEffects         []StatusEffect       `json:"enemy_status_effects,omitempty"`
	PlayerStatusEffects        []StatusEffect       `json:"player_status_effects,omitempty"`
	EnemyPathResistance        map[string]float64   `json:"enemy_path_resistance,omitempty"`
	EnemyMentalResistance      float64              `json:"enemy_mental_resistance,omitempty"`
	EnemyTenacity              float64              `json:"enemy_tenacity,omitempty"`
	EnemyCCResist              float64              `json:"enemy_cc_resist,omitempty"`
	EnemyDaoBonus              map[string]float64   `json:"enemy_dao_bonus,omitempty"`
	EnemyPathDamageBonus       map[string]float64   `json:"enemy_path_damage_bonus,omitempty"`
	EnemyHealingReceived       float64              `json:"enemy_healing_received,omitempty"`
	EnemySoulDefenseBonus      float64              `json:"enemy_soul_defense_bonus,omitempty"`
	EnemyLuckBonus             float64              `json:"enemy_luck_bonus,omitempty"`
	EnemyCombatSettings        *AutoCombatConfig    `json:"enemy_combat_settings,omitempty"`
	SkillCooldowns             map[string]int       `json:"skill_cooldowns,omitempty"`
	ItemsUsedCount             int                  `json:"items_used_count"`
	ItemCooldowns              map[string]int       `json:"item_cooldowns,omitempty"`
	PathMasteryGained          int                  `json:"path_mastery_gained,omitempty"`
	Allies                     []Combatant          `json:"allies,omitempty"`
	PetIDs                     []string             `json:"pet_ids,omitempty"`
	Enemies                    []Combatant          `json:"enemies,omitempty"`
	PetSoulCharged             bool                 `json:"pet_soul_charged,omitempty"`
	TribulationEventID         string               `json:"tribulation_event_id,omitempty"`
	ApertureHP                 float64              `json:"aperture_hp,omitempty"`
	ApertureMaxHP              float64              `json:"aperture_max_hp,omitempty"`
	TempStolenGuInstanceID     string               `json:"temp_stolen_gu_instance_id,omitempty"`
	TempStolenGuID             string               `json:"temp_stolen_gu_id,omitempty"`
	TempStolenGuRank           int                  `json:"temp_stolen_gu_rank,omitempty"`
	TempStolenGuName           string               `json:"temp_stolen_gu_name,omitempty"`
	PermStolenGuInstanceID     string               `json:"perm_stolen_gu_instance_id,omitempty"`
	PermStolenGuID             string               `json:"perm_stolen_gu_id,omitempty"`
	PermStolenGuRank           int                  `json:"perm_stolen_gu_rank,omitempty"`
	PermStolenGuName           string               `json:"perm_stolen_gu_name,omitempty"`
	StealLogMessage            string               `json:"steal_log_message,omitempty"`
	StealMissLogs              []string             `json:"steal_miss_logs,omitempty"`
}

func CombatantFromPlayer(player *Player, stats map[string]float64) Combatant {
	maxHP := player.BaseHP + stats["hp"]
	if maxHP <= 0 {
		maxHP = player.BaseHP
	}
	return Combatant{
		ID: player.ID.String(), Name: player.Username, Kind: "Player",
		Rank: player.Rank, SubRank: player.SubRank, HP: player.HP, MaxHP: maxHP,
		CurrentMana: player.CurrentMana, MaxMana: player.MaxMana,
		BaseAttack: stats["attack"], BaseDefense: stats["defense"], Speed: stats["speed"],
		GuInventory: player.GuInventory, DaoMarks: player.DaoMarks,
		PathMasteries: player.PathMasteries, BodyMods: player.BodyMods,
		HasNonConflictingDao: player.BodyMods.Has(BodyModSovereignImmortal),
		SoulHP:               float64(player.SoulHP), MaxSoulHP: float64(player.MaxSoul),
		DaoMarkDamage: player.DaoMarkDamage, ApertureDmg: player.ApertureDamage,
		AvatarURL: player.AvatarURL, FullyGrown: player.GrowthData.FullyGrown,
		CombatSettings: &player.CombatSettings,
	}
}

func CombatantFromActiveCombat(cs *CombatState) Combatant {
	return Combatant{
		ID: cs.EnemyTemplateID, Name: cs.EnemyName, Kind: cs.EnemyType,
		Rank: cs.EnemyRank, SubRank: cs.EnemySubRank, HP: cs.EnemyHP, MaxHP: cs.EnemyMaxHP,
		CurrentMana: cs.EnemyMana, MaxMana: cs.EnemyMaxMana,
		BaseAttack: cs.EnemyDamage, BaseDefense: cs.EnemyDefense, Speed: cs.EnemySpeed,
		GuInventory: cs.EnemyGuInventory, DaoMarks: cs.EnemyDaoMarks,
		PathMasteries: cs.EnemyPathMasteries, AIProfile: cs.AIProfile, LootTableID: cs.LootTableID,
		StatusEffects: cs.EnemyStatusEffects, AvatarURL: cs.EnemyAvatarURL,
		CombatSettings: cs.EnemyCombatSettings,
	}
}

// Scan and Value keep nullable JSONB encounter snapshots compatible with the
// historical players.active_combat column.
func (state *CombatState) Scan(value any) error {
	if state == nil {
		return errors.New("nil CombatState receiver")
	}
	if value == nil {
		*state = CombatState{}
		return nil
	}
	var raw []byte
	switch typed := value.(type) {
	case []byte:
		raw = typed
	case string:
		raw = []byte(typed)
	default:
		return errors.New("unsupported active_combat database value")
	}
	if len(raw) == 0 || string(raw) == "null" {
		*state = CombatState{}
		return nil
	}
	return json.Unmarshal(raw, state)
}

func (state *CombatState) Value() (driver.Value, error) {
	if state == nil {
		return nil, nil
	}
	return json.Marshal(state)
}
