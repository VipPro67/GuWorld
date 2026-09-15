package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ErrOptimisticLockConflict is returned when a player row update fails due to a version mismatch.
var ErrOptimisticLockConflict = errors.New("concurrent update detected")


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
type MaterialsInventory map[string]int64

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
// Amount is int64 to support large quantities.
type DaoMarksMap map[string]int64

// PathMasteriesMap maps cultivation path names to their accumulated mastery
// points (cảnh giới lưu phái). The attained realm is derived from point
// milestones via gamefx.PathMasteryLevel (stored like Dao Marks).
type PathMasteriesMap map[string]int

// BodyModsList represents passive body modifications
type BodyModsList []string

// StatusInstance is a single active player status (trạng thái). ExpiresAt is
// nil for permanent statuses; Source records where it came from ("admin",
// "combat", "tribulation", ...); Value carries an optional intensity.
type StatusInstance struct {
	ID        string     `json:"id"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	Source    string     `json:"source,omitempty"`
	Value     float64    `json:"value,omitempty"`
}

// StatusList represents the player's active statuses (trạng thái), e.g.
// "severely_wounded". Persisted as a JSONB array of objects; legacy rows that
// stored bare strings are normalized to {id: ...} on unmarshal.
type StatusList []StatusInstance

// Has reports whether a status id is currently active (expired entries still
// count until they are cleaned; use ActiveIDs/Expire for time filtering).
func (s StatusList) Has(id string) bool {
	for _, st := range s {
		if st.ID == id {
			return true
		}
	}
	return false
}

// Add inserts or refreshes a status, deduping by id. d <= 0 means permanent.
func (s *StatusList) Add(id, source string, d time.Duration, value float64) {
	var expiry *time.Time
	if d > 0 {
		t := time.Now().Add(d)
		expiry = &t
	}
	for i := range *s {
		if (*s)[i].ID == id {
			(*s)[i].ExpiresAt = expiry
			(*s)[i].Source = source
			(*s)[i].Value = value
			return
		}
	}
	*s = append(*s, StatusInstance{ID: id, ExpiresAt: expiry, Source: source, Value: value})
}

// Active returns the non-expired statuses (permanent ones always included),
// without mutating the receiver.
func (s StatusList) Active() StatusList {
	out := make(StatusList, 0, len(s))
	now := time.Now()
	for _, st := range s {
		if st.ExpiresAt != nil && !st.ExpiresAt.After(now) {
			continue
		}
		out = append(out, st)
	}
	return out
}

// Remove drops every instance of a status id.
func (s *StatusList) Remove(id string) {
	out := (*s)[:0]
	for _, st := range *s {
		if st.ID != id {
			out = append(out, st)
		}
	}
	*s = out
}

// Expire filters out statuses whose ExpiresAt has passed and reports whether
// the list changed. Permanent statuses (nil ExpiresAt) are kept.
func (s *StatusList) Expire(now time.Time) bool {
	out := (*s)[:0]
	changed := false
	for _, st := range *s {
		if st.ExpiresAt != nil && !st.ExpiresAt.After(now) {
			changed = true
			continue
		}
		out = append(out, st)
	}
	*s = out
	return changed
}

// VisitedSubZonesList tracks the small-zone keys the player has visited in the
// shared world map (bounded set, used for exploration achievements).
type VisitedSubZonesList []string

type StatusEffect struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
	Turns int     `json:"turns"`
}

// GuTactic defines auto-combat behavior for a specific Gu.
type GuTactic struct {
	GuID           string  `json:"gu_id"`
	Priority       int     `json:"priority"`
	UseCondition   string  `json:"use_condition"` // "always","hp_below","mana_above","enemy_hp_below"
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
	AutoHealHP         bool               `json:"auto_heal_hp"`
	HealThreshold      float64            `json:"heal_threshold"`
	AutoRestoreMana    bool               `json:"auto_restore_mana"`
	ManaThreshold      float64            `json:"mana_threshold"`
	CategoryPriorities []string           `json:"category_priorities"`
	GuPriorities       []GuTactic         `json:"gu_priorities"`
	KMPriorities       []KillerMoveTactic `json:"km_priorities"`
	SkipBattle         bool               `json:"skip_battle"`
}

// Scan implements the sql.Scanner interface for AutoCombatConfig.
func (c *AutoCombatConfig) Scan(value interface{}) error {
	if value == nil {
		*c = DefaultAutoCombatConfig()
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		*c = DefaultAutoCombatConfig()
		logger.S.Warnf("AutoCombatConfig.Scan: unexpected type %T", value)
		return nil
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*c = DefaultAutoCombatConfig()
		return nil
	}
	err := json.Unmarshal(bytes, c)
	if err == nil && len(c.CategoryPriorities) == 0 {
		c.CategoryPriorities = []string{"other", "shield", "heal", "damage"}
	}
	return err
}

// Value implements the driver.Valuer interface for AutoCombatConfig.
func (c AutoCombatConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// DefaultAutoCombatConfig returns sensible defaults.
func DefaultAutoCombatConfig() AutoCombatConfig {
	return AutoCombatConfig{
		AutoHealHP:         false,
		HealThreshold:      30,
		AutoRestoreMana:    false,
		ManaThreshold:      30,
		CategoryPriorities: []string{"other", "shield", "heal", "damage"},
		GuPriorities:       []GuTactic{},
		KMPriorities:       []KillerMoveTactic{},
		SkipBattle:         false,
	}
}

// CombatState represents the PVE combat status with a wild beast.
type CombatState struct {
	EnemyTemplateID     string         `json:"enemy_template_id,omitempty"`
	EnemyName           string         `json:"enemy_name"`
	EnemyAvatarURL      string         `json:"enemy_avatar_url,omitempty"`
	EnemyType           string         `json:"enemy_type,omitempty"`
	CombatMode          string         `json:"combat_mode,omitempty"` // "virtual" = illusion (no real cost), "real" = actual resource consumption, "" = legacy default
	EnemyHP             float64        `json:"enemy_hp"`
	EnemyMaxHP          float64        `json:"enemy_max_hp"`
	EnemyMana           float64        `json:"enemy_mana,omitempty"`
	EnemyMaxMana        float64        `json:"enemy_max_mana,omitempty"`
	EnemyDamage         float64        `json:"enemy_damage"`
	EnemyDefense        float64        `json:"enemy_defense"`
	EnemySpeed          float64        `json:"enemy_speed,omitempty"`
	EnemyRank           int            `json:"enemy_rank,omitempty"`
	EnemySubRank        int            `json:"enemy_sub_rank,omitempty"`
	EnemyGuInventory    GuInventory    `json:"enemy_gu_inventory,omitempty"`
	EnemyDaoMarks       DaoMarksMap    `json:"enemy_dao_marks,omitempty"`
	EnemyPathMasteries  map[string]int `json:"enemy_path_masteries,omitempty"`
	AIProfile           string         `json:"ai_profile,omitempty"`
	LootTableID         string         `json:"loot_table_id,omitempty"`
	ChargePoints        int            `json:"charge_points"`
	TargetPlayerID      string         `json:"target_player_id,omitempty"`
	TargetSectID        string         `json:"target_sect_id,omitempty"`
	MineSlotID          int            `json:"mine_slot_id,omitempty"`
	EnemyStatusEffects  []StatusEffect `json:"enemy_status_effects,omitempty"`
	PlayerStatusEffects []StatusEffect `json:"player_status_effects,omitempty"`
	// Defender-side character effects for PVP enemies (race/constitution/status).
	EnemyPathResistance   map[string]float64 `json:"enemy_path_resistance,omitempty"`
	EnemyMentalResistance float64            `json:"enemy_mental_resistance,omitempty"`
	EnemyCombatSettings   *AutoCombatConfig  `json:"enemy_combat_settings,omitempty"`
	SkillCooldowns        map[string]int     `json:"skill_cooldowns,omitempty"`
	ItemsUsedCount        int                `json:"items_used_count"`
	ItemCooldowns         map[string]int     `json:"item_cooldowns,omitempty"`
	// PathMasteryGained tracks Chân Ý points already earned in this battle, so
	// combat gains can be capped at PathMasteryBattleGainLimit(rank) per battle.
	PathMasteryGained int         `json:"path_mastery_gained,omitempty"`
	Allies            []Combatant `json:"allies,omitempty"`
	PetIDs            []string    `json:"pet_ids,omitempty"`
	Enemies           []Combatant `json:"enemies,omitempty"`
	// PetSoulCharged marks that pet deployment soul cost was already paid for
	// this combat, preventing a double charge when InitializeCombatState is
	// re-invoked (e.g. combat setup when Allies is empty).
	PetSoulCharged bool `json:"pet_soul_charged,omitempty"`
	// TribulationEventID links this combat to a tribulation (tai kiếp) event.
	// When non-empty, enemies overflow damage spills into the Immortal Aperture.
	TribulationEventID string  `json:"tribulation_event_id,omitempty"`
	ApertureHP         float64 `json:"aperture_hp,omitempty"`
	ApertureMaxHP      float64 `json:"aperture_max_hp,omitempty"`
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
	CharacterState
	SoulState
	ThoughtPool
	DaoState
	WorldState
	SocialState
	CombatConfig

	ImmortalState      ImmortalState       `json:"immortal_state"`
	GuInventory        GuInventory         `json:"gu_inventory"`
	ActiveCombat       *CombatState        `json:"active_combat"`
	GrowthData         GrowthData          `json:"growth_data"`
	MaterialsInventory MaterialsInventory `json:"materials_inventory,omitempty"`
	CompletedQuests    CompletedQuestsList `json:"completed_quests,omitempty"`
	UnlockedRecipes    UnlockedRecipesList `json:"unlocked_recipes,omitempty"`

	// UI Transient fields
	ManaRegenRate       float64 `json:"mana_regen_rate"`
	HPRegenRate         float64 `json:"hp_regen_rate"`
	SoulRegenRate       float64 `json:"soul_regen_rate"`
	IsSparseGuInventory bool    `json:"is_sparse_gu_inventory,omitempty"`
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
	if err := json.Unmarshal(b, g); err != nil {
		return err
	}
	// Normalize so fully_grown is always an array, never null.
	if g.FullyGrown == nil {
		g.FullyGrown = []FullyGrownInfo{}
	}
	return nil
}

// MarshalJSON ensures fully_grown serializes as [] instead of null when empty.
func (g GrowthData) MarshalJSON() ([]byte, error) {
	type alias GrowthData
	if g.FullyGrown == nil {
		g.FullyGrown = []FullyGrownInfo{}
	}
	return json.Marshal(alias(g))
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
	if len(bytes) == 0 || string(bytes) == "null" {
		*c = ClearedCellsMap{}
		return nil
	}
	bytes = truncateJSON(bytes)
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
	if len(b) == 0 || string(b) == "null" {
		*g = GuInventory{}
		return nil
	}
	b = truncateJSON(b)
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
	if len(b) == 0 || string(b) == "null" {
		*m = MaterialsInventory{}
		return nil
	}
	b = truncateJSON(b)
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
	if len(b) == 0 || string(b) == "null" {
		*d = DaoMarksMap{}
		return nil
	}
	b = truncateJSON(b)
	return json.Unmarshal(b, d)
}

// Value implements the driver.Valuer interface for DaoMarksMap
func (d DaoMarksMap) Value() (driver.Value, error) {
	if d == nil {
		return "{}", nil
	}
	return json.Marshal(d)
}

// IsEmpty returns true if the CombatState has no active combat content.
func (c *CombatState) IsEmpty() bool {
	if c == nil {
		return true
	}
	return c.EnemyName == "" && c.EnemyTemplateID == "" && len(c.Enemies) == 0 && c.TargetPlayerID == "" && c.TribulationEventID == "" && c.MineSlotID == 0
}

// Scan implements the sql.Scanner interface for CombatState pointer
func (c *CombatState) Scan(value interface{}) error {
	if c == nil {
		return nil
	}
	if value == nil {
		*c = CombatState{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		if s, ok := value.(string); ok {
			b = []byte(s)
		} else {
			return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
		}
	}
	if len(b) == 0 || string(b) == "null" || string(b) == "{}" {
		*c = CombatState{}
		return nil
	}
	b = truncateJSON(b)
	return json.Unmarshal(b, c)
}

// Value implements the driver.Valuer interface for CombatState
func (c *CombatState) Value() (driver.Value, error) {
	if c == nil || c.IsEmpty() {
		return nil, nil
	}
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	return string(b), nil
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
	if len(b) == 0 || string(b) == "null" {
		*p = PathMasteriesMap{}
		return nil
	}
	b = truncateJSON(b)
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
	if len(bytes) == 0 || string(bytes) == "null" {
		*b = BodyModsList{}
		return nil
	}
	bytes = truncateJSON(bytes)
	return json.Unmarshal(bytes, b)
}

// Value implements the driver.Valuer interface for BodyModsList
func (b BodyModsList) Value() (driver.Value, error) {
	if b == nil {
		return "[]", nil
	}
	return json.Marshal(b)
}

// Scan implements the sql.Scanner interface for StatusList
func (s *StatusList) Scan(value interface{}) error {
	if value == nil {
		*s = StatusList{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*s = StatusList{}
		return nil
	}
	bytes = truncateJSON(bytes)
	return json.Unmarshal(bytes, s)
}

// Value implements the driver.Valuer interface for StatusList
func (s StatusList) Value() (driver.Value, error) {
	if s == nil {
		return "[]", nil
	}
	return json.Marshal(s)
}

// UnmarshalJSON accepts both the current array-of-objects form and the legacy
// array-of-strings form (["severely_wounded"]), normalizing strings into
// StatusInstance entries with no expiry.
func (s *StatusList) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make(StatusList, 0, len(raw))
	for _, r := range raw {
		var inst StatusInstance
		if len(r) > 0 && r[0] == '"' {
			var id string
			if err := json.Unmarshal(r, &id); err != nil {
				return err
			}
			inst = StatusInstance{ID: id}
		} else if err := json.Unmarshal(r, &inst); err != nil {
			return err
		}
		out = append(out, inst)
	}
	*s = out
	return nil
}

// Scan implements the sql.Scanner interface for VisitedSubZonesList
func (v *VisitedSubZonesList) Scan(value interface{}) error {
	if value == nil {
		*v = VisitedSubZonesList{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	if len(bytes) == 0 || string(bytes) == "null" {
		*v = VisitedSubZonesList{}
		return nil
	}
	bytes = truncateJSON(bytes)
	return json.Unmarshal(bytes, v)
}

// Value implements the driver.Valuer interface for VisitedSubZonesList
func (v VisitedSubZonesList) Value() (driver.Value, error) {
	if v == nil {
		return "[]", nil
	}
	return json.Marshal(v)
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
	if len(bytes) == 0 || string(bytes) == "null" {
		*q = CompletedQuestsList{}
		return nil
	}
	bytes = truncateJSON(bytes)
	return json.Unmarshal(bytes, q)
}

// Value implements the driver.Valuer interface for CompletedQuestsList
func (q CompletedQuestsList) Value() (driver.Value, error) {
	if q == nil {
		return "[]", nil
	}
	return json.Marshal(q)
}

// truncateJSON finds the end of the top-level JSON value in b and returns a
// byte slice containing only that value. This handles corrupted DB columns
// that have trailing non-whitespace bytes after valid JSON.
func truncateJSON(b []byte) []byte {
	if len(b) == 0 {
		return b
	}
	// Skip leading whitespace
	start := 0
	for start < len(b) && (b[start] == ' ' || b[start] == '\t' || b[start] == '\n' || b[start] == '\r') {
		start++
	}
	if start >= len(b) {
		return b[:0]
	}

	depth := 0
	inString := false
	escape := false

	for i := start; i < len(b); i++ {
		ch := b[i]

		if inString {
			if escape {
				escape = false
			} else if ch == '\\' {
				escape = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{', '[':
			if depth == 0 {
				depth = 1
			} else {
				depth++
			}
		case '}', ']':
			depth--
			if depth == 0 {
				return b[start : i+1]
			}
		case ',', ':', '=', ';':
			if depth == 0 {
				return b[start:i]
			}
		default:
			if depth == 0 && (ch < 0x20 || ch == '#' || ch == '-') {
				// Start of trailing garbage (comment, negative sign outside value, etc.)
				return b[start:i]
			}
		}
	}

	// If we reach here without finding a clean boundary, return as-is
	// (json.Unmarshal will error, but we won't panic)
	return b
}
