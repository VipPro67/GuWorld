package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
)

// GuEffects represents flexible stats map stored in JSONB
type GuEffects map[string]float64

// GuDictionaryItem defines the static attributes for a type of Gu bug.
type GuDictionaryItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Rank         int       `json:"rank"`
	Path         string    `json:"path"`    // e.g. "Strength Path", "Moon Path"
	Type         string    `json:"gu_type"` // e.g. "Active", "Passive", "Support", "Consumable"
	BaseManaCost float64   `json:"base_mana_cost"`
	Effects      GuEffects `json:"effects"`
	FoodTypes    []string  `json:"food_types"`  // Supports multiple food types
	HungerRate   int       `json:"hunger_rate"` // Hunger lost per tick
	MaxHunger    int       `json:"max_hunger"`
	Description  string    `json:"description"`
	TargetType   string    `json:"target_type"` // e.g. "SINGLE", "ROW", "COL", "ALL", "CROSS"
}

// Scan implements the sql.Scanner interface for GuEffects
func (g *GuEffects) Scan(value interface{}) error {
	if value == nil {
		*g = GuEffects{}
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}

	var rawEffects map[string]json.RawMessage
	if err := json.Unmarshal(b, &rawEffects); err != nil {
		return err
	}

	effects := make(GuEffects, len(rawEffects))
	for key, raw := range rawEffects {
		var number float64
		if err := json.Unmarshal(raw, &number); err == nil {
			effects[key] = number
			continue
		}

		// Older dictionary rows used JSON booleans for feature flags.
		var flag bool
		if err := json.Unmarshal(raw, &flag); err == nil {
			if flag {
				effects[key] = 1
			} else {
				effects[key] = 0
			}
			continue
		}

		return fmt.Errorf("effect %q must be a number or boolean", key)
	}

	*g = effects
	return nil
}

// Value implements the driver.Valuer interface for GuEffects
func (g GuEffects) Value() (driver.Value, error) {
	if g == nil {
		return "{}", nil
	}
	return json.Marshal(g)
}

// GuRegistry holds the memory map cache of all Gu specifications.
var GuRegistry = make(map[string]GuDictionaryItem)

// LoadGuRegistry queries the database and populates the in-memory cache
func LoadGuRegistry(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, name, rank, path, gu_type, base_mana_cost, effects, food_types, hunger_rate, max_hunger, description, target_type
		FROM gu_dictionary
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Clear and re-populate
	newRegistry := make(map[string]GuDictionaryItem)
	count := 0

	for rows.Next() {
		var item GuDictionaryItem
		var foodTypesRaw []byte
		err = rows.Scan(
			&item.ID, &item.Name, &item.Rank, &item.Path,
			&item.Type, &item.BaseManaCost,
			&item.Effects, &foodTypesRaw, &item.HungerRate,
			&item.MaxHunger, &item.Description, &item.TargetType,
		)
		if err != nil {
			log.Printf("Warning: failed to scan Gu spec row: %v", err)
			continue
		}

		if len(foodTypesRaw) > 0 {
			_ = json.Unmarshal(foodTypesRaw, &item.FoodTypes)
		}

		newRegistry[item.ID] = item
		count++
	}

	GuRegistry = newRegistry
	log.Printf("Loaded %d Gu bugs from database registry into memory cache", count)
	return nil
}
