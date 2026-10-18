package models

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ItemEffects represents flexible stats map stored in JSONB
type ItemEffects map[string]float64

var itemBlockCodeToKey = map[string]string{
	"RESTORE_HUNGER":              "hunger",
	"RESTORE_HP":                  "restore_hp",
	"RESTORE_MANA":                "restore_mana",
	"INCREASE_HUMAN_QI":           "increase_human_qi",
	"HEAL_APERTURE":               "heal_aperture",
	"INCREASE_APERTURE_AREA":      "increase_aperture_area",
	"INCREASE_APERTURE_TIME_RATE": "increase_aperture_time_rate",
	"RANDOM_CONSTITUTION":         "random_constitution",
	"RANDOM_RACE":                 "random_race",
	"OPEN_MINE_CHEST":             "open_mine_chest",
	"OPEN_IMMORTAL_MINE_CHEST":    "open_immortal_mine_chest",
	"CHANGE_RACE":                 "change_race",
	"LIFESPAN_BOOST":              "lifespan_boost",
	"SOUL_BOOST":                  "soul_boost",
	"EXIT_SPATIAL_REALM":          "exit_spatial_realm",
	"ADD_DAO_MARKS":               "add_dao_marks",
}

// Scan implements the sql.Scanner interface for ItemEffects
func (e *ItemEffects) Scan(value interface{}) error {
	if value == nil {
		*e = ItemEffects{}
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("unsupported Scan, expected []byte or string, got %T", value)
	}
	trimmed := bytes.TrimSpace(b)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		*e = ItemEffects{}
		return nil
	}
	if trimmed[0] == '[' {
		return e.scanBlocks(trimmed)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &root); err == nil {
		if blocksRaw, ok := root["blocks"]; ok {
			trimmedB := bytes.TrimSpace(blocksRaw)
			if len(trimmedB) > 0 && trimmedB[0] == '[' {
				return e.scanBlocks(trimmedB)
			}
		}
	}
	return json.Unmarshal(trimmed, e)
}

func (e *ItemEffects) scanBlocks(data []byte) error {
	*e = make(ItemEffects)
	type blockOp struct {
		Op     string          `json:"op"`
		Code   string          `json:"code"`
		Params json.RawMessage `json:"params"`
	}
	type blockItem struct {
		Operations []blockOp `json:"operations"`
	}
	var blocks []blockItem
	if err := json.Unmarshal(data, &blocks); err != nil {
		return err
	}
	for _, b := range blocks {
		for _, op := range b.Operations {
			code := strings.ToUpper(op.Code)
			key, ok := itemBlockCodeToKey[code]
			if !ok {
				key = strings.ToLower(op.Code)
			}
			var p struct {
				Value float64 `json:"value"`
			}
			if len(op.Params) > 0 {
				_ = json.Unmarshal(op.Params, &p)
			}
			(*e)[key] = p.Value
		}
	}
	return nil
}

// Value implements the driver.Valuer interface for ItemEffects
func (e ItemEffects) Value() (driver.Value, error) {
	if e == nil {
		return "{}", nil
	}
	b, err := json.Marshal(e)
	if err != nil {
		return "{}", err
	}
	return string(b), nil
}

// ItemSpec defines the static attributes for a type of item.
type ItemSpec struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Type        string      `json:"type"`      // E.g. "Currency", "Consumable", "Food", "Material"
	ItemType    string      `json:"item_type"` // Backward compatibility for frontend
	Rank        int         `json:"rank"`
	Effects     ItemEffects `json:"effects"`
	Description string      `json:"description"`
	BasePrice   float64     `json:"base_price"`
	ImageURL    string      `json:"image_url"`
	Rarity      int         `json:"rarity"`
	Path        string      `json:"path"`
}

type ItemEffectCommand struct {
	Code  string
	Value float64
}

var itemEffectCodes = map[string]bool{
	"mortal_only": true, "immortal_only": true, "min_rank": true, "max_rank": true,
	"restore_hp": true, "restore_mana": true, "soul_boost": true, "remove_negative_status": true,
	"lifespan_boost": true, "open_mine_chest": true, "open_immortal_mine_chest": true, "heal_aperture": true,
	"increase_aperture_area": true, "increase_aperture_time_rate": true,
	"increase_human_qi": true, "random_constitution": true, "random_race": true,
	"change_race": true, "hunger": true, "exit_spatial_realm": true,
	"add_dao_marks": true,
}

func CompileItemEffects(input ItemEffects) ([]ItemEffectCommand, error) {
	keys := make([]string, 0, len(input))
	for key := range input {
		if !itemEffectCodes[key] {
			return nil, fmt.Errorf("unknown item effect %q", key)
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	commands := make([]ItemEffectCommand, 0, len(keys))
	for _, key := range keys {
		commands = append(commands, ItemEffectCommand{Code: key, Value: input[key]})
	}
	return commands, nil
}

// EffectCommands is the typed, deterministic item-use projection. Callers do
// not inspect the persisted effects map directly.
func (item ItemSpec) EffectCommands() []ItemEffectCommand {
	commands, _ := CompileItemEffects(item.Effects)
	return commands
}

func (item ItemSpec) HasEffect(code string) bool {
	for _, command := range item.EffectCommands() {
		if command.Code == code {
			return true
		}
	}
	return false
}

// ItemRegistry holds the memory map cache of all Item specifications.
var ItemRegistry = make(map[string]ItemSpec)

// ItemIDsByName returns all item ids whose name or ID contains the given search term (case/accent-insensitive).
func ItemIDsByName(search string) []string {
	if search == "" {
		return nil
	}
	cleanSearch := RemoveAccents(search)
	var ids []string
	for id, spec := range ItemRegistry {
		if strings.Contains(RemoveAccents(spec.Name), cleanSearch) || strings.Contains(strings.ToLower(id), cleanSearch) {
			ids = append(ids, id)
		}
	}
	return ids
}

// LoadItemRegistry queries the database and populates the in-memory cache
func LoadItemRegistry(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, name, item_type, rank, effects, description, COALESCE(base_price, 0), COALESCE(image_url, ''), COALESCE(rarity, 0), COALESCE(path, '')
		FROM item_dictionary
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Clear and re-populate
	newRegistry := make(map[string]ItemSpec)
	count := 0

	for rows.Next() {
		var item ItemSpec
		err = rows.Scan(
			&item.ID, &item.Name, &item.Type, &item.Rank,
			&item.Effects, &item.Description, &item.BasePrice,
			&item.ImageURL, &item.Rarity, &item.Path,
		)
		if err != nil {
			logger.S.Warnf("failed to scan Item spec row: %v", err)
			continue
		}

		item.ItemType = item.Type // align for backward compatibility
		if _, compileErr := CompileItemEffects(item.Effects); compileErr != nil {
			return fmt.Errorf("compile item %s effects: %w", item.ID, compileErr)
		}
		newRegistry[item.ID] = item
		count++
	}

	ItemRegistry = newRegistry
	logger.S.Infof("Loaded %d items from database registry into memory cache", count)
	return nil
}
