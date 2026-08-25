package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vippro67/GuWorld/backend/pkg/logger"
)

// ItemEffects represents flexible stats map stored in JSONB
type ItemEffects map[string]float64

// Scan implements the sql.Scanner interface for ItemEffects
func (e *ItemEffects) Scan(value interface{}) error {
	if value == nil {
		*e = ItemEffects{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}
	return json.Unmarshal(b, e)
}

// Value implements the driver.Valuer interface for ItemEffects
func (e ItemEffects) Value() (driver.Value, error) {
	if e == nil {
		return "{}", nil
	}
	return json.Marshal(e)
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
	BasePrice   int         `json:"base_price"`
	ImageURL    string      `json:"image_url"`
	Rarity      int         `json:"rarity"`
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
		SELECT id, name, item_type, rank, effects, description, COALESCE(base_price, 0), COALESCE(image_url, ''), COALESCE(rarity, 0)
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
			&item.ImageURL, &item.Rarity,
		)
		if err != nil {
			logger.S.Warnf("failed to scan Item spec row: %v", err)
			continue
		}

		item.ItemType = item.Type // align for backward compatibility
		newRegistry[item.ID] = item
		count++
	}

	ItemRegistry = newRegistry
	logger.S.Infof("Loaded %d items from database registry into memory cache", count)
	return nil
}
