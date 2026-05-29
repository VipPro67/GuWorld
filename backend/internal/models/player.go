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
	InstanceID string `json:"instance_id"`
	GuID       string `json:"gu_id"`
	Hunger     int    `json:"hunger"`
	Status     string `json:"status"` // "ACTIVE", "SLEEPING", "DEAD"
	Equipped   bool   `json:"equipped"`
	IsVital    bool   `json:"is_vital"`
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

// CombatState represents the PVE combat status with a wild beast.
type CombatState struct {
	EnemyName      string  `json:"enemy_name"`
	EnemyHP        float64 `json:"enemy_hp"`
	EnemyMaxHP     float64 `json:"enemy_max_hp"`
	EnemyDamage    float64 `json:"enemy_damage"`
	EnemyDefense   float64 `json:"enemy_defense"`
	ChargePoints   int     `json:"charge_points"`
	TargetPlayerID string  `json:"target_player_id,omitempty"`
}

// Player represents a cultivator in the Gu World.
type Player struct {
	ID                  uuid.UUID          `json:"id"`
	Username            string             `json:"username"`
	PasswordHash        string             `json:"-"`
	Rank                int                `json:"rank"`             // 1 to 9 (Nhất chuyển to Cửu chuyển)
	SubRank             int                `json:"sub_rank"`         // 1: Sơ giai, 2: Trung giai, 3: Cao giai, 4: Đỉnh phong
	AptitudeType        string             `json:"aptitude_type"`    // 'Giáp', 'Ất', 'Bính', 'Đinh'
	AptitudePercent     int                `json:"aptitude_percent"` // 20 to 99
	CurrentMana         float64            `json:"current_mana"`     // Current Primeval Essence
	MaxMana             float64            `json:"max_mana"`         // Max Primeval Essence (equals AptitudePercent)
	PrimevalStones      int                `json:"primeval_stones"`  // Game currency
	ImmortalStones      float64            `json:"immortal_stones"`
	CultivationProgress float64            `json:"cultivation_progress"`
	GuInventory         GuInventory        `json:"gu_inventory"`
	MaterialsInventory  MaterialsInventory `json:"materials_inventory"`
	HP                  float64            `json:"hp"`
	BaseHP              float64            `json:"base_hp"`
	BodyMods            BodyModsList       `json:"body_mods"`
	X                   int                `json:"x"`
	Y                   int                `json:"y"`
	ExploredCells       ExploredCellsList  `json:"explored_cells"`
	EquippedGus         EquippedGusList    `json:"equipped_gus"`
	ClearedCells        ClearedCellsMap    `json:"cleared_cells"`
	CurrentRegion       string             `json:"current_region"`
	DaoMarks            DaoMarksMap        `json:"dao_marks"`
	PathMasteries       PathMasteriesMap   `json:"path_masteries"`
	ActiveCombat        *CombatState       `json:"active_combat"`
	LastManaUpdate      time.Time          `json:"last_mana_update"` // Time of last mana recalculation
	CreatedAt           time.Time          `json:"created_at"`
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
