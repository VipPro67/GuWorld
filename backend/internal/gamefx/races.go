package gamefx

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
)

// RaceDef describes a single race (chủng tộc) definition.
// The in-memory registry is loaded from the race_dictionary table
// (seeded from data/Race.csv) at startup and after admin imports.
type RaceDef struct {
	Key     string  `json:"key"`
	Name    string  `json:"name"`
	Effects Effects `json:"effects"`
}

// RaceRegistry holds the in-memory cache of race definitions.
var RaceRegistry = make(map[string]RaceDef)

// defaultRaces returns the built-in races used as a fallback when the
// race_dictionary table is empty (e.g. in tests or before a CSV import).
// CSV/DB rows are the source of truth once loaded.
func defaultRaces() []RaceDef {
	return []RaceDef{
		{Key: "human", Name: "Nhân Tộc", Effects: Effects{}},
		{Key: "cuong_thi", Name: "Cương Thi", Effects: Effects{
			Immortal:             true,
			DisableHPRegen:       true,
			DisableManaRegen:     true,
			DisableResourceRegen: true,
			DisableSoulRegen:     true,
		}},
		{Key: "fur_people", Name: "Người Lông", Effects: Effects{RefineBonus: 0.10}},
		{Key: "merman", Name: "Nhân Ngư", Effects: Effects{DaoNganBonus: map[string]float64{"water": 0.10}, PathDamageBonus: map[string]float64{"water": 0.05}}},
		{Key: "feathered", Name: "Vũ Dân", Effects: Effects{StatMods: map[string]float64{"speed": 0.10}, DaoNganBonus: map[string]float64{"space": 0.05}}},
		{Key: "inkman", Name: "Mặc Nhân", Effects: Effects{DaoNganBonus: map[string]float64{"wisdom": 0.08, "soul": 0.05}}},
		{Key: "rockman", Name: "Thạch Nhân", Effects: Effects{StatMods: map[string]float64{"defense": 0.15, "speed": -0.10}}},
		{Key: "snowman", Name: "Tuyết Dân", Effects: Effects{DaoNganBonus: map[string]float64{"ice": 0.15}, PathResistanceBonus: map[string]float64{"fire": 0.10}}},
		{Key: "variant_beastman", Name: "Thú Nhân", Effects: Effects{StatMods: map[string]float64{"attack": 0.08, "defense": 0.05}, RefineBonus: -0.05}},
		{Key: "dragonman", Name: "Long Nhân", Effects: Effects{StatMods: map[string]float64{"attack": 0.10, "defense": 0.10}, RefineBonus: -0.05}},
		{Key: "giant_sun_descendant", Name: "Cự Dương Huyết Mạch", Effects: Effects{LuckBonus: 5, CultivationEfficiencyBonus: 0.05}},
		{Key: "eggman", Name: "Đản Dân", Effects: Effects{DaoNganBonus: map[string]float64{"cloud": 0.05, "wind": 0.05}, StatMods: map[string]float64{"defense": -0.10, "speed": 0.10}, HealingReceivedBonus: 0.15}},
		{Key: "miniman", Name: "Tiểu Nhân", Effects: Effects{DaoNganBonus: map[string]float64{"wood": 0.10}, RefineBonus: 0.05, StatMods: map[string]float64{"attack": -0.12}}},
		{Key: "mushroomman", Name: "Nấm Nhân", Effects: Effects{DaoNganBonus: map[string]float64{"earth": 0.10}, StatMods: map[string]float64{"defense": 0.10, "speed": -0.10}, HPRegenBonus: 15}},
		{Key: "yakshaman", Name: "Dạ Xoa Dân", Effects: Effects{StatMods: map[string]float64{"attack": 0.12}, DaoNganBonus: map[string]float64{"dark": 0.08}, RefineBonus: -0.08, MentalResistanceBonus: -0.10}},
		{Key: "ghostman", Name: "U Hồn Tộc", Effects: Effects{DaoNganBonus: map[string]float64{"soul": 0.12}, SoulDefenseBonus: 0.20, SoulRegenBonus: 0.20, StatMods: map[string]float64{"hp": -0.10}, HealingReceivedBonus: -0.15}},
		{Key: "ancient_human", Name: "Thái Cổ Nhân Tộc", Effects: Effects{CultivationEfficiencyBonus: 0.10, LuckBonus: 3, DaoNganBonus: map[string]float64{"human": 0.10}, RefineBonus: -0.05}},
	}
}

// effectiveRaces returns the registry, falling back to the built-in defaults
// when it has not been loaded from the database yet.
func effectiveRaces() map[string]RaceDef {
	if len(RaceRegistry) > 0 {
		return RaceRegistry
	}
	fallback := make(map[string]RaceDef, len(defaultRaces()))
	for _, def := range defaultRaces() {
		fallback[def.Key] = def
	}
	return fallback
}

// LoadRaceRegistry queries the database and populates the in-memory cache.
// When the table is empty the built-in defaults are used.
func LoadRaceRegistry(db *sql.DB) error {
	// Empty-key sentinel rows exist only to keep the players.race FK satisfiable
	// for the '' value; they must never surface in the registry.
	rows, err := db.Query(`SELECT key, name, COALESCE(effects::text,'{}') FROM race_dictionary WHERE key <> '' ORDER BY key`)
	if err != nil {
		return err
	}
	defer rows.Close()

	newRegistry := make(map[string]RaceDef)
	for rows.Next() {
		var def RaceDef
		var effectsRaw string
		if err := rows.Scan(&def.Key, &def.Name, &effectsRaw); err != nil {
			return fmt.Errorf("scan race row: %w", err)
		}
		if err := json.Unmarshal([]byte(effectsRaw), &def.Effects); err != nil {
			return fmt.Errorf("race %s effects: %w", def.Key, err)
		}
		newRegistry[def.Key] = def
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(newRegistry) == 0 {
		for _, def := range defaultRaces() {
			newRegistry[def.Key] = def
		}
	}
	RaceRegistry = newRegistry
	return nil
}

// AllRaces returns the full list of race definitions, ordered by key.
func AllRaces() []RaceDef {
	defs := effectiveRaces()
	out := make([]RaceDef, 0, len(defs))
	for _, def := range defs {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
