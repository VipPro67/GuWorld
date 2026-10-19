package gamefx

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
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

// fallbackRaces loads race definitions from Race.csv when the in-memory
// registry is unpopulated (e.g. during standalone unit tests).
func fallbackRaces() map[string]RaceDef {
	fallback := make(map[string]RaceDef)
	path := csvPath("Race.csv")
	f, err := os.Open(path)
	if err != nil {
		return fallback
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return fallback
	}
	keyIdx, nameIdx, effectsIdx := -1, -1, -1
	for i, col := range header {
		switch strings.TrimSpace(strings.ToLower(col)) {
		case "key":
			keyIdx = i
		case "name":
			nameIdx = i
		case "effects":
			effectsIdx = i
		}
	}
	if keyIdx == -1 || nameIdx == -1 {
		return fallback
	}

	for {
		rec, err := r.Read()
		if err != nil {
			break
		}
		if len(rec) <= keyIdx {
			continue
		}
		key := strings.TrimSpace(rec[keyIdx])
		if key == "" {
			continue
		}
		name := ""
		if nameIdx >= 0 && nameIdx < len(rec) {
			name = strings.TrimSpace(rec[nameIdx])
		}
		var eff Effects
		if effectsIdx >= 0 && effectsIdx < len(rec) {
			raw := strings.TrimSpace(rec[effectsIdx])
			if raw != "" {
				_ = json.Unmarshal([]byte(raw), &eff)
			}
		}
		fallback[key] = RaceDef{
			Key:     key,
			Name:    name,
			Effects: eff,
		}
	}
	return fallback
}

// effectiveRaces returns the registry, falling back to the CSV dataset
// when it has not been loaded from the database yet.
func effectiveRaces() map[string]RaceDef {
	if len(RaceRegistry) > 0 {
		return RaceRegistry
	}
	return fallbackRaces()
}

// LoadRaceRegistry queries the database and populates the in-memory cache.
// When the table is empty the CSV dataset is used as fallback.
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
		newRegistry = fallbackRaces()
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
