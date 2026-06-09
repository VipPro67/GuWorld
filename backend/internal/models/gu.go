package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

type GuEffects map[string]map[string]float64

type GuDictionaryItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Rank         int       `json:"rank"`
	Path         string    `json:"path"`
	Type         string    `json:"gu_type"`
	BaseManaCost float64   `json:"base_mana_cost"`
	Effects      GuEffects `json:"effects"`
	FoodTypes    []string  `json:"food_types"`
	HungerRate   int       `json:"hunger_rate"`
	MaxHunger    int       `json:"max_hunger"`
	Description  string    `json:"description"`
	OriginRegion string    `json:"origin_region"`
	Rarity       string    `json:"rarity"`
	IsImmortal   bool      `json:"is_immortal"`
	PrimaryRole  string    `json:"primary_role"`
	CombatRoles  []string  `json:"combat_roles"`
	CombatTags   []string  `json:"combat_tags"`
	SupportTags  []string  `json:"support_tags"`
	TargetType   string    `json:"target_type"`
}

type Gu = GuDictionaryItem

type PlayerGu struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	PlayerID  string    `json:"player_id" gorm:"index;not null"`
	GuID      string    `json:"gu_id" gorm:"index;not null"`
	Rank      int       `json:"rank"`
	Level     int       `json:"level"`
	Hunger    float64   `json:"hunger"`
	State     string    `json:"state"`
	Location  string    `json:"location"`
	IsLocked  bool      `json:"is_locked"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (g *GuEffects) Scan(value interface{}) error {
	if value == nil {
		*g = defaultGuEffects()
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("unsupported Scan, expected []byte, got %T", value)
	}

	var nested map[string]map[string]json.RawMessage
	if err := json.Unmarshal(b, &nested); err == nil && hasGuEffectSection(nested) {
		effects := defaultGuEffects()
		for section, values := range nested {
			if !validGuEffectSection(section) {
				continue
			}
			for key, raw := range values {
				if number, ok := rawEffectNumber(raw); ok {
					effects[section][key] = number
				}
			}
		}
		*g = effects
		return nil
	}

	var flat map[string]json.RawMessage
	if err := json.Unmarshal(b, &flat); err != nil {
		return err
	}
	effects := defaultGuEffects()
	for key, raw := range flat {
		if number, ok := rawEffectNumber(raw); ok {
			effects["active"][key] = number
		}
	}
	*g = effects
	return nil
}

func (g GuEffects) Value() (driver.Value, error) {
	if g == nil {
		return json.Marshal(defaultGuEffects())
	}
	return json.Marshal(g)
}

func GetGuEffectSection(gu Gu, section string) map[string]float64 {
	if !validGuEffectSection(section) || gu.Effects == nil {
		return map[string]float64{}
	}
	values, ok := gu.Effects[section]
	if !ok || values == nil {
		return map[string]float64{}
	}
	result := make(map[string]float64, len(values))
	for k, v := range values {
		result[k] = v
	}
	return result
}

func defaultGuEffects() GuEffects {
	return GuEffects{"active": map[string]float64{}, "support": map[string]float64{}, "passive": map[string]float64{}, "house": map[string]float64{}}
}

func validGuEffectSection(section string) bool {
	switch section {
	case "active", "support", "passive", "house":
		return true
	default:
		return false
	}
}

func hasGuEffectSection(values map[string]map[string]json.RawMessage) bool {
	for section := range values {
		if validGuEffectSection(section) {
			return true
		}
	}
	return false
}

func rawEffectNumber(raw json.RawMessage) (float64, bool) {
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, true
	}
	var flag bool
	if err := json.Unmarshal(raw, &flag); err == nil {
		if flag {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

var GuRegistry = make(map[string]GuDictionaryItem)

func LoadGuRegistry(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT id, name, rank, path, gu_type, base_mana_cost, effects, food_types, hunger_rate, max_hunger, description, target_type, origin_region, rarity, is_immortal, primary_role, combat_roles, combat_tags, support_tags
		FROM gu_dictionary
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	newRegistry := make(map[string]GuDictionaryItem)
	count := 0

	for rows.Next() {
		var item GuDictionaryItem
		var foodTypesRaw, combatRolesRaw, combatTagsRaw, supportTagsRaw []byte
		err = rows.Scan(
			&item.ID, &item.Name, &item.Rank, &item.Path,
			&item.Type, &item.BaseManaCost,
			&item.Effects, &foodTypesRaw, &item.HungerRate,
			&item.MaxHunger, &item.Description, &item.TargetType,
			&item.OriginRegion, &item.Rarity, &item.IsImmortal, &item.PrimaryRole,
			&combatRolesRaw, &combatTagsRaw, &supportTagsRaw,
		)
		if err != nil {
			log.Printf("Warning: failed to scan Gu spec row: %v", err)
			continue
		}

		_ = json.Unmarshal(foodTypesRaw, &item.FoodTypes)
		_ = json.Unmarshal(combatRolesRaw, &item.CombatRoles)
		_ = json.Unmarshal(combatTagsRaw, &item.CombatTags)
		_ = json.Unmarshal(supportTagsRaw, &item.SupportTags)

		newRegistry[item.ID] = item
		count++
	}

	GuRegistry = newRegistry
	log.Printf("Loaded %d Gu bugs from database registry into memory cache", count)
	return nil
}
