package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ErrOptimisticLockConflict is returned when a player row update fails due to a version mismatch.
var ErrOptimisticLockConflict = errors.New("concurrent update detected")

// GuInstance represents an individual Gu bug owned by a player.
type GuInstance struct {
	InstanceID         string     `json:"instance_id"`
	GuID               string     `json:"gu_id"`
	Rank               int        `json:"rank"`
	Hunger             int        `json:"hunger"`
	RefinementProgress float64    `json:"refinement_progress"`
	Durability         float64    `json:"durability"`
	Status             string     `json:"status"` // "ACTIVE", "SLEEPING", "DEAD", "DAMAGED"
	Equipped           bool       `json:"equipped"`
	IsVital            bool       `json:"is_vital"`
	IntegrationTime    float64    `json:"integration_time,omitempty"`
	EquippedAt         *time.Time `json:"equipped_at,omitempty"`
	GrowthHours        float64    `json:"growth_hours,omitempty"`
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

const BodyModSovereignImmortal = "sovereign_immortal_body"

// Has reports whether the player has a permanent body modification.
func (b BodyModsList) Has(id string) bool {
	for _, mod := range b {
		if mod == id {
			return true
		}
	}
	return false
}

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

// AutoCombatConfig stores realtime combat controls and client auto-farm policy.
type AutoCombatConfig struct {
	PvPEnabled               bool    `json:"pvp_enabled"`
	RealtimeLeftClickSlot    int     `json:"realtime_left_click_slot"`
	RealtimeRightClickAction string  `json:"realtime_right_click_action"`
	RealtimeRightClickSlot   int     `json:"realtime_right_click_slot"`
	RealtimeAutoFarm         bool    `json:"realtime_auto_farm"`
	RealtimeFarmRange        float64 `json:"realtime_farm_range"`
	RealtimeFarmStopHP       float64 `json:"realtime_farm_stop_hp_percent"`
	RealtimeFarmUseSkills    bool    `json:"realtime_farm_use_skills"`
}

func (c *AutoCombatConfig) NormalizeRealtimeSettings() {
	if c.RealtimeLeftClickSlot < 0 || c.RealtimeLeftClickSlot > 6 {
		c.RealtimeLeftClickSlot = 0
	}
	if c.RealtimeRightClickAction != "attack" && c.RealtimeRightClickAction != "skill" {
		c.RealtimeRightClickAction = "attack"
	}
	if c.RealtimeRightClickSlot < 1 || c.RealtimeRightClickSlot > 6 {
		c.RealtimeRightClickSlot = 0
		if c.RealtimeRightClickAction == "skill" {
			c.RealtimeRightClickAction = "attack"
		}
	}
	if math.IsNaN(c.RealtimeFarmRange) || math.IsInf(c.RealtimeFarmRange, 0) || c.RealtimeFarmRange <= 0 {
		c.RealtimeFarmRange = 14
	}
	if c.RealtimeFarmRange < 4 {
		c.RealtimeFarmRange = 4
	}
	if c.RealtimeFarmRange > 32 {
		c.RealtimeFarmRange = 32
	}
	if math.IsNaN(c.RealtimeFarmStopHP) || math.IsInf(c.RealtimeFarmStopHP, 0) || c.RealtimeFarmStopHP <= 0 {
		c.RealtimeFarmStopHP = 25
	}
	if c.RealtimeFarmStopHP > 90 {
		c.RealtimeFarmStopHP = 90
	}
}

func scanBytes(value interface{}) ([]byte, bool) {
	if value == nil {
		return nil, true
	}
	switch v := value.(type) {
	case []byte:
		return v, true
	case string:
		return []byte(v), true
	default:
		return nil, false
	}
}

// Scan implements the sql.Scanner interface for AutoCombatConfig.
func (c *AutoCombatConfig) Scan(value interface{}) error {
	if value == nil {
		*c = DefaultAutoCombatConfig()
		return nil
	}
	bytes, ok := scanBytes(value)
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
	if err == nil {
		c.NormalizeRealtimeSettings()
	}
	return err
}

// Value implements the driver.Valuer interface for AutoCombatConfig.
func (c AutoCombatConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

// DefaultAutoCombatConfig returns sensible defaults.
func DefaultAutoCombatConfig() AutoCombatConfig {
	return AutoCombatConfig{
		RealtimeRightClickAction: "attack",
		RealtimeFarmRange:        14,
		RealtimeFarmStopHP:       25,
	}
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
	b, ok := scanBytes(value)
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
	b, err := json.Marshal(s)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

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
	GrowthData         GrowthData          `json:"growth_data"`
	MaterialsInventory MaterialsInventory  `json:"materials_inventory,omitempty"`
	CompletedQuests    CompletedQuestsList `json:"completed_quests,omitempty"`
	UnlockedRecipes    UnlockedRecipesList `json:"unlocked_recipes,omitempty"`

	// UI Transient fields
	ManaRegenRate       float64 `json:"mana_regen_rate"`
	HPRegenRate         float64 `json:"hp_regen_rate"`
	SoulRegenRate       float64 `json:"soul_regen_rate"`
	IsSparseGuInventory bool    `json:"is_sparse_gu_inventory,omitempty"`
	CanAccessAdmin      bool    `json:"can_access_admin,omitempty"`
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
	b, ok := scanBytes(value)
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
	b, err := json.Marshal(g)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

type EquippedGusList []string
type ClearedCellsMap map[string]string

// RealtimeHotbarList stores the player's explicit realtime hotbar assignment.
// Index 0..5 maps to hotbar slot 1..6; an empty string means the slot is empty
// (kept empty on purpose) or reserved by a temporarily invalid Gu.
type RealtimeHotbarList []string

// Scan implements the sql.Scanner interface for RealtimeHotbarList
func (h *RealtimeHotbarList) Scan(value interface{}) error {
	if value == nil {
		*h = RealtimeHotbarList{}
		return nil
	}
	bytes, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
	}
	return json.Unmarshal(bytes, h)
}

// Value implements the driver.Valuer interface for RealtimeHotbarList
func (h RealtimeHotbarList) Value() (driver.Value, error) {
	if h == nil {
		return "[]", nil
	}
	b, err := json.Marshal(h)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for EquippedGusList
func (e *EquippedGusList) Scan(value interface{}) error {
	if value == nil {
		*e = EquippedGusList{}
		return nil
	}
	bytes, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
	}
	return json.Unmarshal(bytes, e)
}

// Value implements the driver.Valuer interface for EquippedGusList
func (e EquippedGusList) Value() (driver.Value, error) {
	if e == nil {
		return "[]", nil
	}
	b, err := json.Marshal(e)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for ClearedCellsMap
func (c *ClearedCellsMap) Scan(value interface{}) error {
	if value == nil {
		*c = ClearedCellsMap{}
		return nil
	}
	bytes, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	b, err := json.Marshal(c)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for GuInventory
func (g *GuInventory) Scan(value interface{}) error {
	if value == nil {
		*g = GuInventory{}
		return nil
	}
	b, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	b, err := json.Marshal(g)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
}

// UnmarshalJSON handles both integer and float numbers from JSON to prevent unmarshal errors.
func (m *MaterialsInventory) UnmarshalJSON(b []byte) error {
	var raw map[string]float64
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	res := make(MaterialsInventory, len(raw))
	for k, v := range raw {
		res[k] = int64(v)
	}
	*m = res
	return nil
}

// Scan implements the sql.Scanner interface for MaterialsInventory
func (m *MaterialsInventory) Scan(value interface{}) error {
	if value == nil {
		*m = MaterialsInventory{}
		return nil
	}
	b, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	b, err := json.Marshal(m)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

// UnmarshalJSON handles both integer and float numbers from JSON to prevent unmarshal errors.
func (d *DaoMarksMap) UnmarshalJSON(b []byte) error {
	var raw map[string]float64
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	res := make(DaoMarksMap, len(raw))
	for k, v := range raw {
		res[k] = int64(v)
	}
	*d = res
	return nil
}

// Scan implements the sql.Scanner interface for DaoMarksMap
func (d *DaoMarksMap) Scan(value interface{}) error {
	if value == nil {
		*d = DaoMarksMap{}
		return nil
	}
	b, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	b, err := json.Marshal(d)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

// UnmarshalJSON handles both integer and float numbers from JSON to prevent unmarshal errors.
func (p *PathMasteriesMap) UnmarshalJSON(b []byte) error {
	var raw map[string]float64
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	res := make(PathMasteriesMap, len(raw))
	for k, v := range raw {
		res[k] = int(v)
	}
	*p = res
	return nil
}

// Scan implements the sql.Scanner interface for PathMasteriesMap
func (p *PathMasteriesMap) Scan(value interface{}) error {
	if value == nil {
		*p = PathMasteriesMap{}
		return nil
	}
	b, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	b, err := json.Marshal(p)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for BodyModsList
func (b *BodyModsList) Scan(value interface{}) error {
	if value == nil {
		*b = BodyModsList{}
		return nil
	}
	bytes, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	bJSON, err := json.Marshal(b)
	if err != nil {
		return "[]", err
	}
	return string(bJSON), nil
}

// Scan implements the sql.Scanner interface for StatusList
func (s *StatusList) Scan(value interface{}) error {
	if value == nil {
		*s = StatusList{}
		return nil
	}
	bytes, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	b, err := json.Marshal(s)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
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
	bytes, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	b, err := json.Marshal(v)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
}

type CompletedQuestsList []string

// Scan implements the sql.Scanner interface for CompletedQuestsList
func (q *CompletedQuestsList) Scan(value interface{}) error {
	if value == nil {
		*q = CompletedQuestsList{}
		return nil
	}
	bytes, ok := scanBytes(value)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
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
	b, err := json.Marshal(q)
	if err != nil {
		return "[]", err
	}
	return string(b), nil
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
