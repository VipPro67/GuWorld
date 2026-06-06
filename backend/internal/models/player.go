package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GuInstance represents an individual Gu bug owned by a player.
type GuInstance struct {
	InstanceID      string  `json:"instance_id"`
	GuID            string  `json:"gu_id"`
	Hunger          int     `json:"hunger"`
	Status          string  `json:"status"` // "ACTIVE", "SLEEPING", "DEAD"
	Equipped        bool    `json:"equipped"`
	IsVital         bool    `json:"is_vital"`
	IntegrationTime float64 `json:"integration_time,omitempty"`
}

// GuInventory is a collection of GuInstances owned by a player.
type GuInventory []GuInstance

// MaterialsInventory maps material keys to their quantities.
type MaterialsInventory map[string]int

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

// CombatState represents the PVE combat status with a wild beast.
type CombatState struct {
	EnemyTemplateID     string            `json:"enemy_template_id,omitempty"`
	EnemyName           string            `json:"enemy_name"`
	EnemyType           string            `json:"enemy_type,omitempty"`
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
	Allies              []Combatant       `json:"allies,omitempty"`
	Enemies             []Combatant       `json:"enemies,omitempty"`
}

// Player represents a cultivator in the Gu World.
type Player struct {
	ID                  uuid.UUID           `json:"id"`
	Username            string              `json:"username"`
	PasswordHash        string              `json:"-"`
	Rank                int                 `json:"rank"`             // 1 to 9 (Nhất chuyển to Cửu chuyển)
	SubRank             int                 `json:"sub_rank"`         // 1: Sơ giai, 2: Trung giai, 3: Cao giai, 4: Đỉnh phong
	AptitudeType        string              `json:"aptitude_type"`    // 'Giáp', 'Ất', 'Bính', 'Đinh'
	AptitudePercent     int                 `json:"aptitude_percent"` // 20 to 99
	AptitudeGrade       int                 `json:"aptitude_grade"`   // 1..4 mapping to A..D (optional duplicate of AptitudeType)
	CurrentMana         float64             `json:"current_mana"`     // Current Primeval Essence
	MaxMana             float64             `json:"max_mana"`         // Max Primeval Essence (equals AptitudePercent)
	PrimevalStones      int                 `json:"primeval_stones"`  // Game currency
	ImmortalStones      float64             `json:"immortal_stones"`
	GuInventory         GuInventory         `json:"gu_inventory"`
	MaterialsInventory  MaterialsInventory  `json:"materials_inventory"`
	WallProgress        float64             `json:"wall_progress"` // 0.0..1.0 progress to next sub-rank for mortals
	HumanQi             float64             `json:"human_qi"`      // Accumulated hidden stat for ascension
	HP                  float64             `json:"hp"`
	BaseHP              float64             `json:"base_hp"`
	BodyMods            BodyModsList        `json:"body_mods"`
	X                   int                 `json:"x"`
	Y                   int                 `json:"y"`
	ExploredCells       ExploredCellsList   `json:"explored_cells"`
	EquippedGus         EquippedGusList     `json:"equipped_gus"`
	ClearedCells        ClearedCellsMap     `json:"cleared_cells"`
	CurrentRegion       string              `json:"current_region"`
	DaoMarks            DaoMarksMap         `json:"dao_marks"`
	PathMasteries       PathMasteriesMap    `json:"path_masteries"`
	NextTribulationTime *time.Time          `json:"next_tribulation_time"`
	TribulationCount    int                 `json:"tribulation_count"`
	ActiveCombat        *CombatState        `json:"active_combat"`
	LastManaUpdate      time.Time           `json:"last_mana_update"` // Time of last mana recalculation
	CreatedAt           time.Time           `json:"created_at"`
	Lifespan            float64             `json:"lifespan"`
	MaxLifespan         float64             `json:"max_lifespan"`
	TimeSpeedMultiplier float64             `json:"time_speed_multiplier"`
	PartyID             *uuid.UUID          `json:"party_id"`
	ArenaRank           *int                `json:"arena_rank"`
	Email               string              `json:"email"`
	PhoneNumber         string              `json:"phone_number"`
	CompletedQuests     CompletedQuestsList `json:"completed_quests"`
	TempLoot            TempLoot            `json:"temp_loot"`
	ExploreCount        int                 `json:"explore_count"`
	LastExploreReset    time.Time           `json:"last_explore_reset"`
	ApertureStress      float64             `json:"aperture_stress"`
	LastCultivateTime   time.Time           `json:"last_cultivate_time"`
	SoulHP              int                 `json:"soul_hp"`
	MaxSoul             int                 `json:"max_soul"`
	SoulTier            string              `json:"soul_tier"`
	NeedsSoulRefine     bool                `json:"needs_soul_refine"`
	UnlockedRecipes     UnlockedRecipesList `json:"unlocked_recipes"`
}

type TempLoot struct {
	GuInventory        GuInventory        `json:"gu_inventory"`
	MaterialsInventory MaterialsInventory `json:"materials_inventory"`
	PrimevalStones     int                `json:"primeval_stones"`
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
