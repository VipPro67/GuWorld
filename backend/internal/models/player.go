package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// GuInstance represents an individual Gu bug owned by a player.
type GuInstance struct {
	InstanceID      string     `json:"instance_id"`
	GuID            string     `json:"gu_id"`
	Rank            int        `json:"rank"`
	Hunger          int        `json:"hunger"`
	Status          string     `json:"status"` // "ACTIVE", "SLEEPING", "DEAD"
	Equipped        bool       `json:"equipped"`
	IsVital         bool       `json:"is_vital"`
	IntegrationTime float64    `json:"integration_time,omitempty"`
	EquippedAt      *time.Time `json:"equipped_at,omitempty"`
	GrowthHours     float64    `json:"growth_hours,omitempty"`
}

// GuInventory is a collection of GuInstances owned by a player.
type GuInventory []GuInstance

// MaterialsInventory maps material keys to their quantities.
type MaterialsInventory map[string]int

// ItemCategory returns the sub-inventory bucket name for a given item type.
func ItemCategory(itemType string) string {
	switch itemType {
	case "Consumable", "SoulResource":
		return "consumables"
	case "Treasure":
		return "treasures"
	case "ImmortalMaterial":
		return "immortal_materials"
	default:
		return "materials"
	}
}

// InventoryForCategory is a deprecated helper kept for backward compat.
// Inventory is now managed via InventoryService.
func (p *Player) InventoryForCategory(category string) *MaterialsInventory {
	return nil
}

// DaoMarksMap maps Dao mark types to their counts.
type DaoMarksMap map[string]int

// PathMasteriesMap maps cultivation path names to their mastery levels.
type PathMasteriesMap map[string]string

// BodyModsList represents passive body modifications
type BodyModsList []string

// ExploredCellsList represents coordinates visited by the player
type ExploredCellsList []string

type StatusEffect struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
	Turns int     `json:"turns"`
}

// GuTactic defines auto-combat behavior for a specific Gu.
type GuTactic struct {
	GuID           string  `json:"gu_id"`
	Priority       int     `json:"priority"`
	UseCondition   string  `json:"use_condition"`   // "always","hp_below","mana_above","enemy_hp_below"
	ConditionValue float64 `json:"condition_value"`
}

// KillerMoveTactic defines auto-combat behavior for a specific killer move.
type KillerMoveTactic struct {
	KillerMoveID   string  `json:"killer_move_id"`
	Priority       int     `json:"priority"`
	UseCondition   string  `json:"use_condition"`
	ConditionValue float64 `json:"condition_value"`
}

// AutoCombatConfig holds a player's persistent auto-combat settings.
type AutoCombatConfig struct {
	AutoHealHP      bool               `json:"auto_heal_hp"`
	HealThreshold   float64            `json:"heal_threshold"`
	AutoRestoreMana bool               `json:"auto_restore_mana"`
	ManaThreshold   float64            `json:"mana_threshold"`
	GuPriorities    []GuTactic         `json:"gu_priorities"`
	KMPriorities    []KillerMoveTactic `json:"km_priorities"`
}

// Scan implements the sql.Scanner interface for AutoCombatConfig.
func (c *AutoCombatConfig) Scan(value interface{}) error {
	if value == nil {
		*c = AutoCombatConfig{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		*c = AutoCombatConfig{}
		logger.S.Warnf("AutoCombatConfig.Scan: unexpected type %T", value)
		return nil
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*c = AutoCombatConfig{}
		return nil
	}
	return json.Unmarshal(bytes, c)
}

// Value implements the driver.Valuer interface for AutoCombatConfig.
func (c AutoCombatConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// DefaultAutoCombatConfig returns sensible defaults.
func DefaultAutoCombatConfig() AutoCombatConfig {
	return AutoCombatConfig{
		AutoHealHP:      false,
		HealThreshold:   30,
		AutoRestoreMana: false,
		ManaThreshold:   30,
		GuPriorities:    []GuTactic{},
		KMPriorities:    []KillerMoveTactic{},
	}
}

// CombatState represents the PVE combat status with a wild beast.
type CombatState struct {
	EnemyTemplateID     string            `json:"enemy_template_id,omitempty"`
	EnemyName           string            `json:"enemy_name"`
	EnemyAvatarURL      string            `json:"enemy_avatar_url,omitempty"`
	EnemyType           string            `json:"enemy_type,omitempty"`
	CombatMode          string            `json:"combat_mode,omitempty"` // "virtual" = illusion (no real cost), "real" = actual resource consumption, "" = legacy default
	EnemyHP             float64           `json:"enemy_hp"`
	EnemyMaxHP          float64           `json:"enemy_max_hp"`
	EnemyMana           float64           `json:"enemy_mana,omitempty"`
	EnemyMaxMana        float64           `json:"enemy_max_mana,omitempty"`
	EnemyDamage         float64           `json:"enemy_damage"`
	EnemyDefense        float64           `json:"enemy_defense"`
	EnemyRank           int               `json:"enemy_rank,omitempty"`
	EnemySubRank        int               `json:"enemy_sub_rank,omitempty"`
	EnemyGuInventory    GuInventory       `json:"enemy_gu_inventory,omitempty"`
	EnemyDaoMarks       map[string]int    `json:"enemy_dao_marks,omitempty"`
	EnemyPathMasteries  map[string]string `json:"enemy_path_masteries,omitempty"`
	AIProfile           string            `json:"ai_profile,omitempty"`
	LootTableID         string            `json:"loot_table_id,omitempty"`
	ChargePoints        int               `json:"charge_points"`
	TargetPlayerID      string            `json:"target_player_id,omitempty"`
	MineSlotID          int               `json:"mine_slot_id,omitempty"`
	EnemyStatusEffects  []StatusEffect    `json:"enemy_status_effects,omitempty"`
	PlayerStatusEffects []StatusEffect    `json:"player_status_effects,omitempty"`
	SkillCooldowns      map[string]int    `json:"skill_cooldowns,omitempty"`
	ItemsUsedCount      int               `json:"items_used_count"`
	ItemCooldowns       map[string]int    `json:"item_cooldowns,omitempty"`
	Allies              []Combatant       `json:"allies,omitempty"`
	PetIDs              []string          `json:"pet_ids,omitempty"`
	Enemies             []Combatant       `json:"enemies,omitempty"`
}

// ImmortalState groups immortal expansion data.
type ImmortalState struct {
	ImmortalEssence  int                    `json:"immortal_essence"`
	ApertureGrade    string                 `json:"aperture_grade,omitempty"`
	ApertureTimeRate float64                `json:"aperture_time_rate,omitempty"`
	BlessedLand      map[string]interface{} `json:"blessed_land,omitempty"`
	HeavenSpirit     map[string]interface{} `json:"heaven_spirit,omitempty"`
}

// Scan implements the sql.Scanner interface for ImmortalState.
func (s *ImmortalState) Scan(value interface{}) error {
	if value == nil {
		*s = ImmortalState{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		*s = ImmortalState{}
		return nil
	}
	if len(b) == 0 || string(b) == "null" {
		*s = ImmortalState{}
		return nil
	}
	return json.Unmarshal(b, s)
}

// Value implements the driver.Valuer interface for ImmortalState.
func (s ImmortalState) Value() (driver.Value, error) {
	return json.Marshal(s)
}

// Player represents a cultivator in the Gu World.
// The struct embeds domain sub-states (Account, Cultivation, Body, etc.)
// so field promotion preserves backward-compatible Go access and JSON output.
type Player struct {
	Account
	CultivationState
	ResourceState
	BodyState
	SoulState
	ThoughtPool
	DaoState
	WorldState
	SocialState
	CombatConfig

	ImmortalState      ImmortalState      `json:"immortal_state"`
	GuInventory        GuInventory        `json:"gu_inventory"`
	ActiveCombat       *CombatState       `json:"active_combat"`
	TempLoot           TempLoot           `json:"temp_loot"`
	GrowthData         GrowthData         `json:"growth_data"`
	MaterialsInventory map[string]int     `json:"materials_inventory,omitempty"`
	CompletedQuests    CompletedQuestsList `json:"completed_quests,omitempty"`

	// UI Transient fields
	ManaRegenRate float64 `json:"mana_regen_rate"`
	HPRegenRate   float64 `json:"hp_regen_rate"`
}

type TempLoot struct {
	GuInventory        GuInventory        `json:"gu_inventory"`
	MaterialsInventory MaterialsInventory `json:"materials_inventory"`
	PrimevalStones     float64            `json:"primeval_stones"`
}

// Scan implements the sql.Scanner interface for TempLoot
func (t *TempLoot) Scan(value interface{}) error {
	if value == nil {
		*t = TempLoot{
			GuInventory:        GuInventory{},
			MaterialsInventory: MaterialsInventory{},
			PrimevalStones:     0,
		}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, t)
}

// Value implements the driver.Valuer interface for TempLoot
func (t TempLoot) Value() (driver.Value, error) {
	return json.Marshal(t)
}

type GrowthBoosts struct {
	Attack  float64 `json:"attack,omitempty"`
	Defense float64 `json:"defense,omitempty"`
	HP      float64 `json:"hp,omitempty"`
	Speed   float64 `json:"speed,omitempty"`
}

type FullyGrownInfo struct {
	GuID    string  `json:"gu_id"`
	Rank    int     `json:"rank"`
	Attack  float64 `json:"attack,omitempty"`
	Defense float64 `json:"defense,omitempty"`
	HP      float64 `json:"hp,omitempty"`
	Speed   float64 `json:"speed,omitempty"`
}

type GrowthData struct {
	Boosts     GrowthBoosts     `json:"boosts"`
	FullyGrown []FullyGrownInfo `json:"fully_grown"`
}

func (g *GrowthData) Scan(value interface{}) error {
	if value == nil {
		*g = GrowthData{Boosts: GrowthBoosts{}, FullyGrown: []FullyGrownInfo{}}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		*g = GrowthData{Boosts: GrowthBoosts{}, FullyGrown: []FullyGrownInfo{}}
		return nil
	}
	if len(b) == 0 || string(b) == "null" {
		*g = GrowthData{Boosts: GrowthBoosts{}, FullyGrown: []FullyGrownInfo{}}
		return nil
	}
	return json.Unmarshal(b, g)
}

func (g GrowthData) Value() (driver.Value, error) {
	return json.Marshal(g)
}

type EquippedGusList []string
type ClearedCellsMap map[string]string

// Scan implements the sql.Scanner interface for EquippedGusList
func (e *EquippedGusList) Scan(value interface{}) error {
	if value == nil {
		*e = EquippedGusList{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, e)
}

// Value implements the driver.Valuer interface for EquippedGusList
func (e EquippedGusList) Value() (driver.Value, error) {
	if e == nil {
		return "[]", nil
	}
	return json.Marshal(e)
}

// Scan implements the sql.Scanner interface for ClearedCellsMap
func (c *ClearedCellsMap) Scan(value interface{}) error {
	if value == nil {
		*c = ClearedCellsMap{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, c)
}

// Value implements the driver.Valuer interface for ClearedCellsMap
func (c ClearedCellsMap) Value() (driver.Value, error) {
	if c == nil {
		return "{}", nil
	}
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface for GuInventory
func (g *GuInventory) Scan(value interface{}) error {
	if value == nil {
		*g = GuInventory{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, g)
}

// Value implements the driver.Valuer interface for GuInventory
func (g GuInventory) Value() (driver.Value, error) {
	if g == nil {
		return "[]", nil
	}
	return json.Marshal(g)
}

// Scan implements the sql.Scanner interface for MaterialsInventory
func (m *MaterialsInventory) Scan(value interface{}) error {
	if value == nil {
		*m = MaterialsInventory{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, m)
}

// Value implements the driver.Valuer interface for MaterialsInventory
func (m MaterialsInventory) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for DaoMarksMap
func (d *DaoMarksMap) Scan(value interface{}) error {
	if value == nil {
		*d = DaoMarksMap{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, d)
}

// Value implements the driver.Valuer interface for DaoMarksMap
func (d DaoMarksMap) Value() (driver.Value, error) {
	if d == nil {
		return "{}", nil
	}
	return json.Marshal(d)
}

// Scan implements the sql.Scanner interface for CombatState pointer
func (c *CombatState) Scan(value interface{}) error {
	if value == nil {
		*c = CombatState{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, c)
}

// Value implements the driver.Valuer interface for CombatState
func (c *CombatState) Value() (driver.Value, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface for PathMasteriesMap
func (p *PathMasteriesMap) Scan(value interface{}) error {
	if value == nil {
		*p = PathMasteriesMap{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, p)
}

// Value implements the driver.Valuer interface for PathMasteriesMap
func (p PathMasteriesMap) Value() (driver.Value, error) {
	if p == nil {
		return "{}", nil
	}
	return json.Marshal(p)
}

// Scan implements the sql.Scanner interface for BodyModsList
func (b *BodyModsList) Scan(value interface{}) error {
	if value == nil {
		*b = BodyModsList{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, b)
}

// Value implements the driver.Valuer interface for BodyModsList
func (b BodyModsList) Value() (driver.Value, error) {
	if b == nil {
		return "[]", nil
	}
	return json.Marshal(b)
}

// Scan implements the sql.Scanner interface for ExploredCellsList
func (e *ExploredCellsList) Scan(value interface{}) error {
	if value == nil {
		*e = ExploredCellsList{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, e)
}

// Value implements the driver.Valuer interface for ExploredCellsList
func (e ExploredCellsList) Value() (driver.Value, error) {
	if e == nil {
		return "[\"50,50\"]", nil
	}
	return json.Marshal(e)
}

type CompletedQuestsList []string

// Scan implements the sql.Scanner interface for CompletedQuestsList
func (q *CompletedQuestsList) Scan(value interface{}) error {
	if value == nil {
		*q = CompletedQuestsList{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(bytes, q)
}

// Value implements the driver.Valuer interface for CompletedQuestsList
func (q CompletedQuestsList) Value() (driver.Value, error) {
	if q == nil {
		return "[]", nil
	}
	return json.Marshal(q)
}
